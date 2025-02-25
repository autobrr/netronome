// Copyright (c) 2024-2025, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package server

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

func (h *AuthHandler) InitOIDCRoutes(r *gin.RouterGroup) {
	if h.oidc == nil {
		return
	}

	r.GET("/oidc/login", h.handleOIDCLogin)
	r.GET("/oidc/callback", h.handleOIDCCallback)
}

func (h *AuthHandler) handleOIDCLogin(c *gin.Context) {
	authURL := h.oidc.AuthURL()
	c.Redirect(http.StatusTemporaryRedirect, authURL)
}

func (h *AuthHandler) handleOIDCCallback(c *gin.Context) {
	baseURL := c.GetString("base_url")
	if baseURL == "" {
		baseURL = "/"
	}

	code := c.Query("code")
	if code == "" {
		log.Error().Msg("no code received in callback")
		c.Redirect(http.StatusTemporaryRedirect, baseURL+"login?error=invalid_code")
		return
	}

	// Exchange code for token using the exported OAuth2Config
	token, err := h.oidc.OAuth2Config.Exchange(c.Request.Context(), code)
	if err != nil {
		log.Error().Err(err).Msg("failed to exchange code for token")
		c.Redirect(http.StatusTemporaryRedirect, baseURL+"login?error=token_exchange")
		return
	}

	// Get the ID token
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		log.Error().Msg("no id_token in oauth2 token")
		c.Redirect(http.StatusTemporaryRedirect, baseURL+"login?error=missing_id_token")
		return
	}

	// Verify the token
	if err := h.oidc.VerifyToken(c.Request.Context(), rawIDToken); err != nil {
		log.Error().Err(err).Msg("invalid ID token")
		c.Redirect(http.StatusTemporaryRedirect, baseURL+"login?error=invalid_token")
		return
	}

	sessionToken, err := generateSecureToken(32)
	if err != nil {
		log.Error().Err(err).Msg("failed to generate session token")
		c.Redirect(http.StatusTemporaryRedirect, baseURL+"login?error=server_error")
		return
	}

	if err := h.storeSession(sessionToken); err != nil {
		log.Error().Err(err).Msg("failed to store session")
		c.Redirect(http.StatusTemporaryRedirect, baseURL+"login?error=server_error")
		return
	}

	// Set session cookie with the random token instead
	c.SetCookie(
		"session",
		sessionToken,
		int((30 * 24 * time.Hour).Seconds()), // 30 days
		baseURL,                              // Use baseURL for cookie path
		"",
		c.Request.URL.Scheme == "https",
		true,
	)

	c.Redirect(http.StatusTemporaryRedirect, baseURL)
}

func generateSecureToken(length int) (string, error) {
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
