// Copyright (c) 2024-2026, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package server

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"

	"github.com/autobrr/netronome/internal/auth"
	"github.com/autobrr/netronome/internal/database"
	"github.com/autobrr/netronome/internal/utils"
)

const (
	// pkceVerifierTTL is the time a user has to complete a login at the identity provider.
	pkceVerifierTTL = 15 * time.Minute
	// pkceVerifierMax is the number of pending logins kept in memory.
	pkceVerifierMax = 1024
	// sessionTTL is the life of a session cookie, and of its sessionTokens entry.
	sessionTTL = 24 * time.Hour
)

// pkceEntry is a PKCE code verifier and the time it becomes invalid.
type pkceEntry struct {
	verifier string
	expires  time.Time
}

// memorySession is a session that exists only in memory, with the time it
// becomes invalid.
type memorySession struct {
	claims  SessionClaims
	expires time.Time
}

type AuthHandler struct {
	db             database.Service
	oidc           *auth.OIDCConfig
	oidcConfigured bool                     // true when OIDC issuer is set in config, independent of provider state
	sessionTokens  map[string]memorySession // Track valid memory sessions
	pkceVerifiers  map[string]pkceEntry     // Track PKCE code verifiers by state
	sessionMutex   sync.RWMutex
	pkceMutex      sync.RWMutex
	sessionSecret  string
	whitelist      []string
}

func NewAuthHandler(db database.Service, oidc *auth.OIDCConfig, oidcConfigured bool, sessionSecret string, whitelist []string) *AuthHandler {
	return &AuthHandler{
		db:             db,
		oidc:           oidc,
		oidcConfigured: oidcConfigured,
		sessionTokens:  make(map[string]memorySession),
		pkceVerifiers:  make(map[string]pkceEntry),
		sessionSecret:  sessionSecret,
		whitelist:      whitelist,
	}
}

func (h *AuthHandler) CheckRegistrationStatus(c *gin.Context) {
	var count int
	err := h.db.QueryRow(c.Request.Context(), "SELECT COUNT(*) FROM users").Scan(&count)
	if err != nil && !isTableNotExistsError(err) {
		log.Error().Err(err).Msg("Failed to check existing users")
		_ = c.Error(fmt.Errorf("failed to check registration status: %w", err))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"hasUsers":       count > 0,
		"oidcConfigured": h.oidcConfigured,
		"oidcReady":      h.oidc != nil,
	})
}

// isJWT checks if a token appears to be a JWT (has 3 base64-encoded parts separated by dots)
func isJWT(token string) bool {
	parts := strings.Split(token, ".")
	return len(parts) == 3
}

// refreshSession updates the session cookie with a new expiry time
func (h *AuthHandler) refreshSession(c *gin.Context, token string, claims *SessionClaims) {
	isSecure := c.GetHeader("X-Forwarded-Proto") == "https" || strings.HasPrefix(c.Request.Proto, "HTTPS")

	rawToken := token
	if claims != nil {
		encoded, err := encodeSessionClaims(*claims)
		if err != nil {
			log.Error().Err(err).Msg("Failed to encode session claims")
			return
		}
		rawToken = encoded
	}

	var signedToken string
	// For OIDC tokens (JWTs), always treat as raw tokens to be signed
	// Only check for our signed tokens if they have our memory prefix or match our signing pattern
	if claims == nil && (strings.HasPrefix(token, auth.MemoryOnlyPrefix) || (strings.Contains(token, ".") && h.sessionSecret != "" && !isJWT(token))) {
		// Token is already signed, verify and resign it
		rawToken, err := auth.VerifyToken(token, h.sessionSecret)
		if err != nil {
			log.Debug().Err(err).Msg("Failed to verify existing signed token")
			return
		}
		signedToken = auth.SignToken(rawToken, h.sessionSecret)
	} else {
		// Token is not signed yet (including OIDC JWTs), sign it
		signedToken = auth.SignToken(rawToken, h.sessionSecret)
	}

	// Track memory-only sessions
	if h.sessionSecret == "" {
		h.trackMemorySession(token, signedToken, claims)
	}

	logger := log.Trace().
		Bool("secure", isSecure).
		Bool("memory_only", h.sessionSecret == "")
	if claims == nil {
		logger = logger.Str("token", token).Str("signed_token", signedToken)
	} else {
		logger = logger.Str("token", "session_claims")
	}
	logger.Msg("Setting session cookie")

	// Set domain to empty string to work with both localhost and IP addresses
	c.SetCookie(
		"session",
		signedToken,
		int(sessionTTL.Seconds()),
		"/",
		"",       // empty domain for maximum compatibility
		isSecure, // secure flag only if HTTPS
		true,     // httpOnly for security
	)
}

func (h *AuthHandler) isValidMemorySession(token string) bool {
	if !strings.HasPrefix(token, auth.MemoryOnlyPrefix) {
		return false
	}

	_, ok := h.getMemorySessionClaims(token)
	return ok
}

func (h *AuthHandler) getMemorySessionClaims(token string) (SessionClaims, bool) {
	if !strings.HasPrefix(token, auth.MemoryOnlyPrefix) {
		return SessionClaims{}, false
	}

	h.sessionMutex.RLock()
	session, ok := h.sessionTokens[token]
	h.sessionMutex.RUnlock()

	if !ok || time.Now().After(session.expires) {
		return SessionClaims{}, false
	}
	return session.claims, true
}

// trackMemorySession records a memory-only session and extends its expiry. A
// new session first removes the sessions that expired, so that abandoned
// browser sessions do not stay in memory. A refresh of a known session keeps
// the request path free of the sweep.
func (h *AuthHandler) trackMemorySession(oldToken, signedToken string, claims *SessionClaims) {
	now := time.Now()

	h.sessionMutex.Lock()
	defer h.sessionMutex.Unlock()

	existing, known := h.sessionTokens[signedToken]
	if !known {
		for token, session := range h.sessionTokens {
			if now.After(session.expires) {
				delete(h.sessionTokens, token)
			}
		}
	}

	session := memorySession{expires: now.Add(sessionTTL)}
	switch {
	case claims != nil:
		if oldToken != "" && oldToken != signedToken {
			delete(h.sessionTokens, oldToken)
		}
		session.claims = *claims
	case known:
		session.claims = existing.claims
	default:
		session.claims = SessionClaims{
			Version: sessionClaimsVersion,
			Type:    sessionTypeLocal,
		}
	}
	h.sessionTokens[signedToken] = session
}

func (h *AuthHandler) getSessionClaims(signedToken, rawToken string) (*SessionClaims, bool) {
	if strings.HasPrefix(signedToken, auth.MemoryOnlyPrefix) {
		claims, ok := h.getMemorySessionClaims(signedToken)
		if !ok || claims.Type == "" {
			return nil, false
		}
		return &claims, true
	}

	if rawToken == "" {
		if h.sessionSecret == "" {
			return nil, false
		}
		var err error
		rawToken, err = auth.VerifyToken(signedToken, h.sessionSecret)
		if err != nil {
			return nil, false
		}
	}

	claims, ok := decodeSessionClaims(rawToken)
	if !ok {
		return nil, false
	}
	return claims, true
}

func (h *AuthHandler) maybeRefreshOIDCSession(c *gin.Context, signedToken string, claims *SessionClaims) {
	if h.oidc == nil {
		h.refreshSession(c, signedToken, claims)
		return
	}

	refreshToken, err := h.getRefreshToken(claims)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to decode refresh token")
		h.refreshSession(c, signedToken, claims)
		return
	}
	if refreshToken == "" {
		h.refreshSession(c, signedToken, claims)
		return
	}

	now := time.Now().Unix()
	if claims.IDTokenExp != 0 && now < claims.IDTokenExp-60 {
		h.refreshSession(c, signedToken, claims)
		return
	}
	if claims.LastRefresh != 0 && now-claims.LastRefresh < 60 {
		h.refreshSession(c, signedToken, claims)
		return
	}

	token, err := h.oidc.RefreshToken(c.Request.Context(), refreshToken)
	claims.LastRefresh = now
	if err != nil {
		log.Debug().Err(err).Msg("Failed to refresh OIDC token")
		h.refreshSession(c, signedToken, claims)
		return
	}

	if rawIDToken, ok := token.Extra("id_token").(string); ok && rawIDToken != "" {
		idClaims, err := h.oidc.VerifyTokenWithClaims(c.Request.Context(), rawIDToken)
		if err != nil {
			log.Debug().Err(err).Msg("Failed to verify refreshed OIDC token")
			h.refreshSession(c, signedToken, claims)
			return
		}
		claims.Subject = idClaims.Subject
		claims.Username = pickOIDCUsername(idClaims)
		claims.IDTokenExp = idClaims.Expiry
	}

	if token.RefreshToken != "" && token.RefreshToken != refreshToken {
		if err := h.setRefreshToken(claims, token.RefreshToken); err != nil {
			log.Debug().Err(err).Msg("Failed to store refreshed token")
		}
	}

	h.refreshSession(c, signedToken, claims)
}

func (h *AuthHandler) getRefreshToken(claims *SessionClaims) (string, error) {
	if claims.RefreshToken == "" {
		return "", nil
	}
	if h.sessionSecret == "" {
		return "", errors.New("session secret required")
	}
	return decryptRefreshToken(h.sessionSecret, claims.RefreshToken)
}

func (h *AuthHandler) setRefreshToken(claims *SessionClaims, refreshToken string) error {
	if refreshToken == "" {
		claims.RefreshToken = ""
		return nil
	}
	if h.sessionSecret == "" {
		return errors.New("session secret required")
	}
	encrypted, err := encryptRefreshToken(h.sessionSecret, refreshToken)
	if err != nil {
		return err
	}
	claims.RefreshToken = encrypted
	return nil
}

func pickOIDCUsername(claims *auth.IDTokenClaims) string {
	if claims.Username != "" {
		return claims.Username
	}
	if claims.Name != "" {
		return claims.Name
	}
	return claims.Subject
}

func sessionUsername(claims *SessionClaims) string {
	if claims == nil {
		return ""
	}
	if claims.Username != "" {
		return claims.Username
	}
	return claims.Subject
}

// storePKCEVerifier stores a PKCE code verifier for the given state. It first
// removes the verifiers of abandoned logins, then keeps the map below
// pkceVerifierMax.
//
// Use it before you redirect a user to the identity provider. The callback
// reads the verifier back with getPKCEVerifier.
func (h *AuthHandler) storePKCEVerifier(state, codeVerifier string) {
	now := time.Now()

	h.pkceMutex.Lock()
	defer h.pkceMutex.Unlock()

	for key, entry := range h.pkceVerifiers {
		if now.After(entry.expires) {
			delete(h.pkceVerifiers, key)
		}
	}

	// ponytail: random eviction under a flood of login requests. Rate limit
	// the login route if that becomes a problem.
	if len(h.pkceVerifiers) >= pkceVerifierMax {
		for key := range h.pkceVerifiers {
			delete(h.pkceVerifiers, key)
			break
		}
	}

	h.pkceVerifiers[state] = pkceEntry{
		verifier: codeVerifier,
		expires:  now.Add(pkceVerifierTTL),
	}
}

// getPKCEVerifier retrieves and removes the PKCE code verifier for the given
// state. An expired verifier is reported as missing.
func (h *AuthHandler) getPKCEVerifier(state string) (string, bool) {
	h.pkceMutex.Lock()
	defer h.pkceMutex.Unlock()

	entry, exists := h.pkceVerifiers[state]
	delete(h.pkceVerifiers, state)
	if !exists || time.Now().After(entry.expires) {
		return "", false
	}
	return entry.verifier, true
}

func (h *AuthHandler) Register(c *gin.Context) {
	var count int
	err := h.db.QueryRow(c.Request.Context(), "SELECT COUNT(*) FROM users").Scan(&count)
	if err != nil && !isTableNotExistsError(err) {
		log.Error().Err(err).Msg("Failed to check existing users")
		_ = c.Error(fmt.Errorf("failed to check existing users: %w", err))
		return
	}

	if err == nil && count > 0 {
		_ = c.Error(fmt.Errorf("registration disabled: user already exists"))
		return
	}

	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(fmt.Errorf("invalid request data: %w", err))
		return
	}

	//if err := auth.ValidatePassword(req.Password); err != nil {
	//	_ = c.Error(fmt.Errorf("invalid password: %w", err))
	//	return
	//}

	user, err := h.db.CreateUser(c.Request.Context(), req.Username, req.Password)
	if err != nil {
		if err == database.ErrUserAlreadyExists {
			_ = c.Error(fmt.Errorf("username already exists: %w", err))
			return
		}
		log.Error().Err(err).Msg("Failed to create user")
		_ = c.Error(fmt.Errorf("failed to create user: %w", err))
		return
	}

	sessionToken, err := utils.GenerateSecureToken(32)
	if err != nil {
		log.Error().Err(err).Msg("Failed to generate session token")
		_ = c.Error(fmt.Errorf("failed to generate session token: %w", err))
		return
	}

	h.refreshSession(c, sessionToken, nil)

	c.JSON(http.StatusCreated, gin.H{
		"message": "User registered successfully",
		"user": gin.H{
			"id":       user.ID,
			"username": user.Username,
		},
	})
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request data",
		})
		return
	}

	user, err := h.db.GetUserByUsername(c.Request.Context(), req.Username)
	if err != nil {
		if err == database.ErrUserNotFound {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid credentials",
			})
			return
		}
		log.Error().Err(err).Msg("Failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get user",
		})
		return
	}

	if !h.db.ValidatePassword(user, req.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Invalid credentials",
		})
		return
	}

	sessionToken, err := utils.GenerateSecureToken(32)
	if err != nil {
		log.Error().Err(err).Msg("Failed to generate session token")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to generate session token",
		})
		return
	}

	h.refreshSession(c, sessionToken, nil)

	log.Debug().
		Str("username", user.Username).
		Str("session_token", sessionToken).
		Msg("User logged in successfully")

	c.JSON(http.StatusOK, gin.H{
		"access_token": sessionToken,
		"token_type":   "Bearer",
		"expires_in":   int(sessionTTL.Seconds()),
		"user": gin.H{
			"id":       user.ID,
			"username": user.Username,
		},
	})
}

func (h *AuthHandler) Verify(c *gin.Context) {
	if isWhitelisted(c, h.whitelist) {
		c.JSON(http.StatusOK, gin.H{
			"message": "IP is whitelisted",
			"type":    "whitelist",
		})
		return
	}

	signedToken, err := c.Cookie("session")
	if err != nil {
		log.Debug().Err(err).Msg("No session cookie found")
		_ = c.Error(fmt.Errorf("no session found: %w", err))
		return
	}

	if claims, ok := h.getSessionClaims(signedToken, ""); ok && claims.Type == sessionTypeOIDC {
		h.maybeRefreshOIDCSession(c, signedToken, claims)
		c.JSON(http.StatusOK, gin.H{
			"message": "Token is valid",
			"type":    "oidc",
		})
		return
	}

	// If OIDC is configured, check if this is a valid OIDC token first
	if h.oidc != nil {
		// Extract the actual token regardless of how it's stored
		actualToken := signedToken

		// Handle memory-only sessions (no session_secret)
		if strings.HasPrefix(signedToken, auth.MemoryOnlyPrefix) {
			actualToken = strings.TrimPrefix(signedToken, auth.MemoryOnlyPrefix)
		} else if h.sessionSecret != "" {
			// Handle signed tokens (with session_secret)
			// Try to verify and extract the raw token
			if rawToken, err := auth.VerifyToken(signedToken, h.sessionSecret); err == nil {
				actualToken = rawToken
			}
		}

		// Check if it's a JWT and verify with OIDC
		if isJWT(actualToken) {
			if err := h.oidc.VerifyToken(c.Request.Context(), actualToken); err == nil {
				h.refreshSession(c, signedToken, nil)
				c.JSON(http.StatusOK, gin.H{
					"message": "Token is valid",
					"type":    "oidc",
				})
				return
			}
		}
	}

	// check if it's a memory session
	if strings.HasPrefix(signedToken, auth.MemoryOnlyPrefix) {
		if !h.isValidMemorySession(signedToken) {
			log.Debug().Msg("Invalid memory session")
			_ = c.Error(fmt.Errorf("invalid memory session"))
			return
		}
	} else {
		_, err := auth.VerifyToken(signedToken, h.sessionSecret)
		if err != nil {
			log.Debug().Err(err).Msg("Invalid session token")
			_ = c.Error(fmt.Errorf("invalid session: %w", err))
			return
		}
	}

	var username string
	err = h.db.QueryRow(c.Request.Context(), "SELECT username FROM users LIMIT 1").Scan(&username)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to verify session")
		_ = c.Error(fmt.Errorf("invalid session: %w", err))
		return
	}

	h.refreshSession(c, signedToken, nil)

	c.JSON(http.StatusOK, gin.H{
		"message": "Token is valid",
		"type":    "session",
	})
}

func (h *AuthHandler) Logout(c *gin.Context) {
	baseURL := c.GetString("base_url")
	if baseURL == "" {
		baseURL = "/"
	}

	isSecure := c.GetHeader("X-Forwarded-Proto") == "https" || strings.HasPrefix(c.Request.Proto, "HTTPS")

	// Remove the session cookie
	c.SetCookie(
		"session",
		"",
		-1,
		baseURL,
		"",
		isSecure,
		true,
	)

	h.sessionMutex.Lock()
	if sessionToken, err := c.Cookie("session"); err == nil {
		delete(h.sessionTokens, sessionToken)
	}
	h.sessionMutex.Unlock()

	c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully"})
}

func (h *AuthHandler) GetUserInfo(c *gin.Context) {
	if isWhitelisted(c, h.whitelist) {
		c.JSON(http.StatusOK, gin.H{
			"user": gin.H{
				"id":       0,
				"username": "whitelisted",
			},
		})
		return
	}

	signedToken, err := c.Cookie("session")
	if err != nil {
		_ = c.Error(fmt.Errorf("no session found"))
		return
	}

	if claims, ok := h.getSessionClaims(signedToken, ""); ok && claims.Type == sessionTypeOIDC {
		username := sessionUsername(claims)
		c.JSON(http.StatusOK, gin.H{
			"user": gin.H{
				"id":       0,
				"username": username,
			},
		})
		return
	}

	// If OIDC is configured, check if this is a valid OIDC token first
	if h.oidc != nil {
		// Extract the actual token regardless of how it's stored
		actualToken := signedToken

		// Handle memory-only sessions (no session_secret)
		if strings.HasPrefix(signedToken, auth.MemoryOnlyPrefix) {
			actualToken = strings.TrimPrefix(signedToken, auth.MemoryOnlyPrefix)
		} else if h.sessionSecret != "" {
			// Handle signed tokens (with session_secret)
			// Try to verify and extract the raw token
			if rawToken, err := auth.VerifyToken(signedToken, h.sessionSecret); err == nil {
				actualToken = rawToken
			}
		}

		// Check if it's a JWT and verify with OIDC
		if isJWT(actualToken) {
			claims, err := h.oidc.GetClaims(c.Request.Context(), actualToken)
			if err == nil {
				c.JSON(http.StatusOK, gin.H{
					"user": gin.H{
						"id":       0, // OIDC users don't have local IDs
						"username": claims.Subject,
					},
				})
				return
			}
		}
	}

	// check if it's a memory session
	if strings.HasPrefix(signedToken, auth.MemoryOnlyPrefix) {
		if !h.isValidMemorySession(signedToken) {
			log.Debug().Msg("Invalid memory session")
			_ = c.Error(fmt.Errorf("invalid memory session"))
			return
		}
	} else {
		_, err := auth.VerifyToken(signedToken, h.sessionSecret)
		if err != nil {
			log.Debug().Err(err).Msg("Invalid session token")
			_ = c.Error(fmt.Errorf("invalid session: %w", err))
			return
		}
	}

	// fall back to regular user lookup
	username := c.GetString("username")
	if username == "" {
		_ = c.Error(fmt.Errorf("no session found"))
		return
	}

	user, err := h.db.GetUserByUsername(c.Request.Context(), username)
	if err != nil {
		_ = c.Error(fmt.Errorf("failed to get user: %w", err))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user": gin.H{
			"id":       user.ID,
			"username": user.Username,
		},
	})
}

func isTableNotExistsError(err error) bool {
	return err != nil && err.Error() == "SQL logic error: no such table: users (1)"
}

func isWhitelisted(c *gin.Context, whitelist []string) bool {
	clientIP := c.ClientIP()
	ip := net.ParseIP(clientIP)
	if ip == nil {
		log.Warn().Str("ip", clientIP).Msg("Failed to parse client IP")
		return false
	}

	for _, network := range whitelist {
		_, ipNet, err := net.ParseCIDR(network)
		if err != nil {
			log.Warn().Str("network", network).Err(err).Msg("Failed to parse whitelist network")
			continue
		}
		if ipNet.Contains(ip) {
			//log.Debug().Str("ip", clientIP).Str("network", network).Msg("Client IP is in whitelisted network")
			return true
		}
	}

	return false
}

func RequireAuth(db database.Service, oidc *auth.OIDCConfig, sessionSecret string, handler *AuthHandler, whitelist []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if isWhitelisted(c, whitelist) {
			c.Next()
			return
		}

		signedToken, err := c.Cookie("session")
		if err != nil {
			log.Debug().Err(err).Msg("No session cookie found")
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		var rawToken string
		// check if it's a memory session
		if strings.HasPrefix(signedToken, auth.MemoryOnlyPrefix) {
			if !handler.isValidMemorySession(signedToken) {
				log.Debug().Msg("Invalid memory session")
				c.AbortWithStatus(http.StatusUnauthorized)
				return
			}
		} else {
			rawToken, err = auth.VerifyToken(signedToken, sessionSecret)
			if err != nil {
				log.Debug().Err(err).Msg("Invalid session token")
				c.AbortWithStatus(http.StatusUnauthorized)
				return
			}
		}

		if claims, ok := handler.getSessionClaims(signedToken, rawToken); ok && claims.Type == sessionTypeOIDC {
			handler.maybeRefreshOIDCSession(c, signedToken, claims)
			username := sessionUsername(claims)
			c.Set("username", username)
			c.Next()
			return
		}

		// if oidc is configured, try to verify the token
		if oidc != nil {
			// Extract the actual token regardless of how it's stored
			actualToken := signedToken

			// Handle memory-only sessions (no session_secret)
			if strings.HasPrefix(signedToken, auth.MemoryOnlyPrefix) {
				actualToken = strings.TrimPrefix(signedToken, auth.MemoryOnlyPrefix)
			} else if sessionSecret != "" {
				// Handle signed tokens (with session_secret)
				// Try to verify and extract the raw token
				actualToken = rawToken
			}

			// Check if it's a JWT and verify with OIDC
			if isJWT(actualToken) {
				if err := oidc.VerifyToken(c.Request.Context(), actualToken); err == nil {
					c.Next()
					return
				}
			}
		}

		// fall back to regular auth check
		var username string
		err = db.QueryRow(c.Request.Context(), "SELECT username FROM users LIMIT 1").Scan(&username)
		if err != nil {
			log.Debug().Err(err).Msg("Failed to find user")
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		//log.Debug().
		//	Str("username", username).
		//	Bool("memory_only", strings.HasPrefix(signedToken, auth.MemoryOnlyPrefix)).
		//	Msg("User authenticated successfully")

		c.Set("username", username)
		c.Next()
	}
}
