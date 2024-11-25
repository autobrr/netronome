// Copyright (c) 2024, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package server

import (
	"fmt"
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

func (h *AuthHandler) CheckRegistrationStatus(c *gin.Context) {
	var count int
	err := h.db.QueryRow(c.Request.Context(), "SELECT COUNT(*) FROM users").Scan(&count)
	if err != nil && !isTableNotExistsError(err) {
		log.Error().Err(err).Msg("Failed to check existing users")
		_ = c.Error(fmt.Errorf("failed to check registration status: %w", err))
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

	if err := utils.ValidatePassword(req.Password); err != nil {
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

	sessionToken, err := utils.GenerateSecureToken(32)
	if err != nil {
		log.Error().Err(err).Msg("Failed to generate session token")
		_ = c.Error(fmt.Errorf("failed to generate session token: %w", err))
		return
	}

	isSecure := c.GetHeader("X-Forwarded-Proto") == "https"

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

	isSecure := c.GetHeader("X-Forwarded-Proto") == "https"

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
		_ = c.Error(fmt.Errorf("no session found: %w", err))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Token is valid",
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
	username := c.GetString("username")
	if username == "" {
		_ = c.Error(fmt.Errorf("no session found"))
		return
	}

	user, err := h.db.GetUserByUsername(c.Request.Context(), username)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get user info")
		_ = c.Error(fmt.Errorf("failed to get user info: %w", err))
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
