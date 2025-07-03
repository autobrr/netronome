// Copyright (c) 2024-2025, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"

	"github.com/autobrr/netronome/internal/auth"
	"github.com/autobrr/netronome/internal/utils"
)

func (h *AuthHandler) InitOIDCRoutes(r *gin.RouterGroup) {
	if h.oidc == nil {
		return
	}

	r.GET("/oidc/login", h.handleOIDCLogin)
	r.GET("/oidc/callback", h.handleOIDCCallback)
}

func (h *AuthHandler) handleOIDCLogin(c *gin.Context) {
	// Generate PKCE parameters
	pkceParams, err := auth.GeneratePKCEParams()
	if err != nil {
		log.Error().Err(err).Msg("Failed to generate PKCE parameters")
		c.Redirect(http.StatusTemporaryRedirect, "/?error=pkce_generation_failed")
		return
	}

	// Generate state parameter
	state, err := utils.GenerateSecureToken(32)
	if err != nil {
		log.Error().Err(err).Msg("Failed to generate state parameter")
		c.Redirect(http.StatusTemporaryRedirect, "/?error=state_generation_failed")
		return
	}

	// Store PKCE verifier for later use in callback
	h.storePKCEVerifier(state, pkceParams.CodeVerifier)

	// Generate auth URL with PKCE and state
	authURL := h.oidc.AuthURLWithPKCE(state, pkceParams)

	log.Debug().
		Str("state", state).
		Str("code_challenge", pkceParams.CodeChallenge).
		Str("auth_url", authURL).
		Msg("Redirecting to OIDC provider with PKCE")

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

	state := c.Query("state")
	if state == "" {
		log.Error().Msg("no state received in callback")
		c.Redirect(http.StatusTemporaryRedirect, baseURL+"login?error=invalid_state")
		return
	}

	// Retrieve PKCE code verifier
	codeVerifier, exists := h.getPKCEVerifier(state)
	if !exists {
		log.Error().Str("state", state).Msg("no PKCE verifier found for state")
		c.Redirect(http.StatusTemporaryRedirect, baseURL+"login?error=invalid_state")
		return
	}

	// Exchange code for token using PKCE
	token, err := h.oidc.ExchangeCodeWithPKCE(c.Request.Context(), code, codeVerifier)
	if err != nil {
		log.Error().Err(err).Msg("failed to exchange code for token with PKCE")
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

	log.Debug().
		Str("state", state).
		Str("code_verifier", codeVerifier[:20]+"...").
		Msg("PKCE token exchange successful")

	h.refreshSession(c, rawIDToken)

	c.Redirect(http.StatusTemporaryRedirect, baseURL)
}
