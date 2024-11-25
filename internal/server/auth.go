// Copyright (c) 2024, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package server

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"

	"github.com/autobrr/netronome/internal/database"
	"github.com/autobrr/netronome/internal/utils"
)

type AuthHandler struct {
	db database.Service
}

func NewAuthHandler(db database.Service) *AuthHandler {
	return &AuthHandler{
		db: db,
	}
}

// checks if registration is allowed (no users exist)
func (h *AuthHandler) CheckRegistrationStatus(c *gin.Context) {
	var count int
	err := h.db.QueryRow(c.Request.Context(), "SELECT COUNT(*) FROM users").Scan(&count)
	if err != nil && !isTableNotExistsError(err) {
		log.Error().Err(err).Msg("failed to check existing users")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"registrationEnabled": count == 0,
		"hasUsers":            count > 0,
	})
}

func (h *AuthHandler) Register(c *gin.Context) {
	var count int
	err := h.db.QueryRow(c.Request.Context(), "SELECT COUNT(*) FROM users").Scan(&count)
	if err != nil && !isTableNotExistsError(err) {
		log.Error().Err(err).Msg("failed to check existing users")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	if err == nil && count > 0 {
		c.JSON(http.StatusForbidden, gin.H{"error": "Registration is disabled. A user already exists."})
		return
	}

	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data"})
		return
	}

	if err := utils.ValidatePassword(req.Password); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := h.db.CreateUser(c.Request.Context(), req.Username, req.Password)
	if err != nil {
		if err == database.ErrUserAlreadyExists {
			c.JSON(http.StatusConflict, gin.H{"error": "Username already exists"})
			return
		}
		log.Error().Err(err).Msg("failed to create user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	sessionToken, err := utils.GenerateSecureToken(32)
	if err != nil {
		log.Error().Err(err).Msg("failed to generate session token")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	var isSecure = c.GetHeader("X-Forwarded-Proto") == "https"

	c.SetCookie(
		"session",
		sessionToken,
		int((24 * time.Hour).Seconds()),
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data"})
		return
	}

	user, err := h.db.GetUserByUsername(context.Background(), req.Username)
	if err != nil {
		if err == database.ErrUserNotFound {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
			return
		}
		log.Error().Err(err).Msg("failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	if !h.db.ValidatePassword(user, req.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	sessionToken, err := utils.GenerateSecureToken(32)
	if err != nil {
		log.Error().Err(err).Msg("failed to generate session token")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	var isSecure = c.GetHeader("X-Forwarded-Proto") == "https"

	c.SetCookie(
		"session",
		sessionToken,
		int((24 * time.Hour).Seconds()),
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
	_, err := c.Cookie("session")
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "No session found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Token is valid",
	})
}

func (h *AuthHandler) Logout(c *gin.Context) {
	var isSecure = c.GetHeader("X-Forwarded-Proto") == "https"

	// Clear session cookie
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
	username := c.GetString("username")
	if username == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "No session found"})
		return
	}

	user, err := h.db.GetUserByUsername(c.Request.Context(), username)
	if err != nil {
		log.Error().Err(err).Msg("failed to get user info")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
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
