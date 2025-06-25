// Copyright (c) 2024-2025, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package server

import (
	"net/http"

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

	h.refreshSession(c, rawIDToken)

	c.Redirect(http.StatusTemporaryRedirect, baseURL)
}
