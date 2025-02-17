// Copyright (c) 2024-2025, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package server

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"

	"github.com/autobrr/netronome/internal/auth"
	"github.com/autobrr/netronome/internal/database"
)

type AuthHandler struct {
	db            database.Service
	oidc          *auth.OIDCConfig
	sessionTokens map[string]string
	sessionMutex  sync.RWMutex
}

func NewAuthHandler(db database.Service, oidc *auth.OIDCConfig) *AuthHandler {
	return &AuthHandler{
		db:            db,
		oidc:          oidc,
		sessionTokens: make(map[string]string),
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

	if err := auth.ValidatePassword(req.Password); err != nil {
		_ = c.Error(fmt.Errorf("invalid password: %w", err))
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

	sessionToken, err := auth.GenerateSecureToken(32)
	if err != nil {
		log.Error().Err(err).Msg("Failed to generate session token")
		_ = c.Error(fmt.Errorf("failed to generate session token: %w", err))
		return
	}

	isSecure := c.GetHeader("X-Forwarded-Proto") == "https"

	c.SetCookie(
		"session",
		sessionToken,
		int((30 * 24 * time.Hour).Seconds()),
		"/",
		"",
		isSecure,
		true,
	)

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

	sessionToken, err := auth.GenerateSecureToken(32)
	if err != nil {
		log.Error().Err(err).Msg("Failed to generate session token")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to generate session token",
		})
		return
	}

	isSecure := c.GetHeader("X-Forwarded-Proto") == "https"

	c.SetCookie(
		"session",
		sessionToken,
		int((30 * 24 * time.Hour).Seconds()),
		"/",
		"",
		isSecure,
		true,
	)

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
	sessionToken, err := c.Cookie("session")
	if err != nil {
		log.Debug().Err(err).Msg("No session cookie found")
		_ = c.Error(fmt.Errorf("no session found: %w", err))
		return
	}

	// First try OIDC verification if enabled
	if h.oidc != nil {
		if err := h.oidc.VerifyToken(c.Request.Context(), sessionToken); err == nil {
			c.JSON(http.StatusOK, gin.H{
				"message": "Token is valid",
				"type":    "oidc",
			})
			return
		}
	}

	// Fall back to regular session verification
	var username string
	err = h.db.QueryRow(c.Request.Context(), "SELECT username FROM users LIMIT 1").Scan(&username)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to verify session")
		_ = c.Error(fmt.Errorf("invalid session: %w", err))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Token is valid",
		"type":    "session",
	})
}

func (h *AuthHandler) Logout(c *gin.Context) {
	isSecure := c.GetHeader("X-Forwarded-Proto") == "https"

	c.SetCookie(
		"session",
		"",
		-1,
		"/",
		"",
		isSecure,
		true,
	)

	c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully"})
}

func (h *AuthHandler) GetUserInfo(c *gin.Context) {
	sessionToken, err := c.Cookie("session")
	if err != nil {
		_ = c.Error(fmt.Errorf("no session found"))
		return
	}

	// Try OIDC first
	if h.oidc != nil {
		claims, err := h.oidc.GetClaims(c.Request.Context(), sessionToken)
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

	// Fall back to regular user lookup
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

func RequireAuth(db database.Service, oidc *auth.OIDCConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionToken, err := c.Cookie("session")
		if err != nil {
			log.Trace().Err(err).Msg("No session cookie found")
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		// If OIDC is configured, try to verify the token
		if oidc != nil {
			if err := oidc.VerifyToken(c.Request.Context(), sessionToken); err == nil {
				c.Next()
				return
			}
		}

		// Fall back to regular auth check
		var username string
		err = db.QueryRow(c.Request.Context(), "SELECT username FROM users LIMIT 1").Scan(&username)
		if err != nil {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		c.Set("username", username)
		c.Next()
	}
}

func (h *AuthHandler) storeSession(sessionToken, idToken string) error {
	h.sessionMutex.Lock()
	defer h.sessionMutex.Unlock()
	h.sessionTokens[sessionToken] = idToken
	return nil
}
