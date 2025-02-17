// Copyright (c) 2024-2025, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package server

import (
	"fmt"
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
	sessionTokens map[string]bool // Track valid memory sessions
	sessionMutex  sync.RWMutex
	sessionSecret string
}

func NewAuthHandler(db database.Service, oidc *auth.OIDCConfig, sessionSecret string) *AuthHandler {
	return &AuthHandler{
		db:            db,
		oidc:          oidc,
		sessionTokens: make(map[string]bool),
		sessionSecret: sessionSecret,
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

// refreshSession updates the session cookie with a new expiry time
func (h *AuthHandler) refreshSession(c *gin.Context, token string) {
	isSecure := c.GetHeader("X-Forwarded-Proto") == "https" || strings.HasPrefix(c.Request.Proto, "HTTPS")

	var signedToken string
	if strings.Contains(token, ".") && h.sessionSecret != "" {
		// Token is already signed, verify and resign it
		rawToken, err := auth.VerifyToken(token, h.sessionSecret)
		if err != nil {
			log.Debug().Err(err).Msg("Failed to verify existing signed token")
			return
		}
		signedToken = auth.SignToken(rawToken, h.sessionSecret)
	} else {
		// Token is not signed yet, sign it
		signedToken = auth.SignToken(token, h.sessionSecret)
	}

	// Track memory-only sessions
	if h.sessionSecret == "" {
		h.sessionMutex.Lock()
		h.sessionTokens[signedToken] = true
		h.sessionMutex.Unlock()
	}

	log.Debug().
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
		_ = c.Error(fmt.Errorf("invalid request data: %w", err))
		return
	}

	user, err := h.db.GetUserByUsername(c.Request.Context(), req.Username)
	if err != nil {
		if err == database.ErrUserNotFound {
			_ = c.Error(fmt.Errorf("invalid credentials: %w", err))
			return
		}
		log.Error().Err(err).Msg("Failed to get user")
		_ = c.Error(fmt.Errorf("failed to get user: %w", err))
		return
	}

	if !h.db.ValidatePassword(user, req.Password) {
		_ = c.Error(fmt.Errorf("invalid credentials"))
		return
	}

	sessionToken, err := utils.GenerateSecureToken(32)
	if err != nil {
		log.Error().Err(err).Msg("Failed to generate session token")
		_ = c.Error(fmt.Errorf("failed to generate session token: %w", err))
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
	signedToken, err := c.Cookie("session")
	if err != nil {
		log.Debug().Err(err).Msg("No session cookie found")
		_ = c.Error(fmt.Errorf("no session found: %w", err))
		return
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

	if h.oidc != nil {
		if err := h.oidc.VerifyToken(c.Request.Context(), signedToken); err == nil {
			h.refreshSession(c, signedToken)
			c.JSON(http.StatusOK, gin.H{
				"message": "Token is valid",
				"type":    "oidc",
			})
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
	isSecure := c.GetHeader("X-Forwarded-Proto") == "https" || strings.HasPrefix(c.Request.Proto, "HTTPS")

	c.SetCookie(
		"session",
		"",
		-1,
		"/",
		"",       // empty domain for maximum compatibility
		isSecure, // secure flag only if HTTPS
		true,     // httpOnly for security
	)

	c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully"})
}

func (h *AuthHandler) GetUserInfo(c *gin.Context) {
	signedToken, err := c.Cookie("session")
	if err != nil {
		_ = c.Error(fmt.Errorf("no session found"))
		return
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

	// try oidc first
	if h.oidc != nil {
		claims, err := h.oidc.GetClaims(c.Request.Context(), signedToken)
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

func RequireAuth(db database.Service, oidc *auth.OIDCConfig, sessionSecret string, handler *AuthHandler) gin.HandlerFunc {
	return func(c *gin.Context) {
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
			if err := oidc.VerifyToken(c.Request.Context(), signedToken); err == nil {
				c.Next()
				return
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

		log.Debug().
			Str("username", username).
			Bool("memory_only", strings.HasPrefix(signedToken, auth.MemoryOnlyPrefix)).
			Msg("User authenticated successfully")

		c.Set("username", username)
		c.Next()
	}
}

func (h *AuthHandler) storeSession(sessionToken string) error {
	h.sessionMutex.Lock()
	defer h.sessionMutex.Unlock()
	h.sessionTokens[sessionToken] = true
	return nil
}
