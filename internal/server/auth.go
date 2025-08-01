// Copyright (c) 2024-2025, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package server

import (
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

type AuthHandler struct {
	db            database.Service
	oidc          *auth.OIDCConfig
	sessionTokens map[string]bool   // Track valid memory sessions
	pkceVerifiers map[string]string // Track PKCE code verifiers by state
	sessionMutex  sync.RWMutex
	pkceMutex     sync.RWMutex
	sessionSecret string
	whitelist     []string
}

func NewAuthHandler(db database.Service, oidc *auth.OIDCConfig, sessionSecret string, whitelist []string) *AuthHandler {
	return &AuthHandler{
		db:            db,
		oidc:          oidc,
		sessionTokens: make(map[string]bool),
		pkceVerifiers: make(map[string]string),
		sessionSecret: sessionSecret,
		whitelist:     whitelist,
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
		"hasUsers":    count > 0,
		"oidcEnabled": h.oidc != nil,
	})
}

// isJWT checks if a token appears to be a JWT (has 3 base64-encoded parts separated by dots)
func isJWT(token string) bool {
	parts := strings.Split(token, ".")
	return len(parts) == 3
}

// refreshSession updates the session cookie with a new expiry time
func (h *AuthHandler) refreshSession(c *gin.Context, token string) {
	isSecure := c.GetHeader("X-Forwarded-Proto") == "https" || strings.HasPrefix(c.Request.Proto, "HTTPS")

	var signedToken string
	// For OIDC tokens (JWTs), always treat as raw tokens to be signed
	// Only check for our signed tokens if they have our memory prefix or match our signing pattern
	if strings.HasPrefix(token, auth.MemoryOnlyPrefix) || (strings.Contains(token, ".") && h.sessionSecret != "" && !isJWT(token)) {
		// Token is already signed, verify and resign it
		rawToken, err := auth.VerifyToken(token, h.sessionSecret)
		if err != nil {
			log.Debug().Err(err).Msg("Failed to verify existing signed token")
			return
		}
		signedToken = auth.SignToken(rawToken, h.sessionSecret)
	} else {
		// Token is not signed yet (including OIDC JWTs), sign it
		signedToken = auth.SignToken(token, h.sessionSecret)
	}

	// Track memory-only sessions
	if h.sessionSecret == "" {
		h.sessionMutex.Lock()
		h.sessionTokens[signedToken] = true
		h.sessionMutex.Unlock()
	}

	log.Trace().
		Str("token", token).
		Str("signed_token", signedToken).
		Bool("secure", isSecure).
		Bool("memory_only", h.sessionSecret == "").
		Msg("Setting session cookie")

	// Set domain to empty string to work with both localhost and IP addresses
	c.SetCookie(
		"session",
		signedToken,
		int((24 * time.Hour).Seconds()), // 24 hour expiry
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

	h.sessionMutex.RLock()
	valid := h.sessionTokens[token]
	h.sessionMutex.RUnlock()
	return valid
}

// storePKCEVerifier stores a PKCE code verifier for the given state
func (h *AuthHandler) storePKCEVerifier(state, codeVerifier string) {
	h.pkceMutex.Lock()
	h.pkceVerifiers[state] = codeVerifier
	h.pkceMutex.Unlock()
}

// getPKCEVerifier retrieves and removes the PKCE code verifier for the given state
func (h *AuthHandler) getPKCEVerifier(state string) (string, bool) {
	h.pkceMutex.Lock()
	defer h.pkceMutex.Unlock()

	verifier, exists := h.pkceVerifiers[state]
	if exists {
		delete(h.pkceVerifiers, state)
	}
	return verifier, exists
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

	h.refreshSession(c, sessionToken)

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

	h.refreshSession(c, sessionToken)

	log.Debug().
		Str("username", user.Username).
		Str("session_token", sessionToken).
		Msg("User logged in successfully")

	c.JSON(http.StatusOK, gin.H{
		"access_token": sessionToken,
		"token_type":   "Bearer",
		"expires_in":   int((24 * time.Hour).Seconds()),
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
				h.refreshSession(c, signedToken)
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

	h.refreshSession(c, signedToken)

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

		// check if it's a memory session
		if strings.HasPrefix(signedToken, auth.MemoryOnlyPrefix) {
			if !handler.isValidMemorySession(signedToken) {
				log.Debug().Msg("Invalid memory session")
				c.AbortWithStatus(http.StatusUnauthorized)
				return
			}
		} else {
			_, err := auth.VerifyToken(signedToken, sessionSecret)
			if err != nil {
				log.Debug().Err(err).Msg("Invalid session token")
				c.AbortWithStatus(http.StatusUnauthorized)
				return
			}
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
				if rawToken, err := auth.VerifyToken(signedToken, sessionSecret); err == nil {
					actualToken = rawToken
				}
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
