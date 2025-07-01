// Copyright (c) 2024-2025, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package server

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"golang.org/x/oauth2"
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
	log.Debug().Msg("exchanging authorization code for token")
	
	// Create a custom context with our logging HTTP client
	ctx := context.WithValue(c.Request.Context(), oauth2.HTTPClient, h.oidc.GetHTTPClient())
	token, err := h.oidc.OAuth2Config.Exchange(ctx, code)
	if err != nil {
		log.Error().Err(err).Msg("failed to exchange code for token")
		c.Redirect(http.StatusTemporaryRedirect, baseURL+"login?error=token_exchange")
		return
	}
	log.Debug().Msg("token exchange successful")

	// Get the ID token
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		log.Error().Msg("no id_token in oauth2 token")
		c.Redirect(http.StatusTemporaryRedirect, baseURL+"login?error=missing_id_token")
		return
	}
	log.Debug().Msg("ID token extracted successfully")

	// Verify the token
	if err := h.oidc.VerifyToken(c.Request.Context(), rawIDToken); err != nil {
		log.Error().Err(err).Msg("invalid ID token")
		c.Redirect(http.StatusTemporaryRedirect, baseURL+"login?error=invalid_token")
		return
	}

	// Get claims from the verified token
	claims, err := h.oidc.GetClaims(c.Request.Context(), rawIDToken)
	if err != nil {
		log.Error().Err(err).Msg("failed to extract claims from ID token")
		c.Redirect(http.StatusTemporaryRedirect, baseURL+"login?error=invalid_claims")
		return
	}

	// Use the username or subject for the session
	sessionToken := claims.Username
	if sessionToken == "" {
		sessionToken = claims.Subject
	}
	
	log.Debug().Str("username", claims.Username).Msg("user authenticated successfully")
	h.refreshSession(c, sessionToken)

	c.Redirect(http.StatusTemporaryRedirect, baseURL)
}
