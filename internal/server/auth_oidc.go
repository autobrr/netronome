// Copyright (c) 2024-2026, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package server

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"

	"github.com/autobrr/netronome/internal/auth"
	"github.com/autobrr/netronome/internal/utils"
)

func loginErrorRedirectURL(baseURL, errorCode string) string {
	baseURL = strings.TrimRight(baseURL, "/")
	return baseURL + "/login?error=" + url.QueryEscape(errorCode)
}

func (h *AuthHandler) handleOIDCLogin(c *gin.Context) {
	baseURL := c.GetString("base_url")
	if baseURL == "" {
		baseURL = "/"
	}

	if h.oidc == nil {
		log.Warn().Msg("OIDC login attempted but provider is not ready")
		c.Redirect(http.StatusTemporaryRedirect, loginErrorRedirectURL(baseURL, "oidc_unavailable"))
		return
	}

	// Generate state parameter
	state, err := utils.GenerateSecureToken(32)
	if err != nil {
		log.Error().Err(err).Msg("Failed to generate state parameter")
		c.Redirect(http.StatusTemporaryRedirect, loginErrorRedirectURL(baseURL, "state_generation_failed"))
		return
	}

	// Generate PKCE parameters
	pkceParams, err := auth.GeneratePKCEParams()
	if err != nil {
		log.Error().Err(err).Msg("Failed to generate PKCE parameters")
		c.Redirect(http.StatusTemporaryRedirect, loginErrorRedirectURL(baseURL, "pkce_generation_failed"))
		return
	}

	// Store PKCE verifier for later use in callback
	h.storePKCEVerifier(state, pkceParams.CodeVerifier)

	// Generate auth URL with PKCE and state
	authURL := h.oidc.AuthURLWithPKCE(state, pkceParams)

	log.Debug().
		Str("state", state).
		Str("auth_url", authURL).
		Msg("Redirecting to OIDC provider")

	c.Redirect(http.StatusTemporaryRedirect, authURL)
}

func (h *AuthHandler) handleOIDCCallback(c *gin.Context) {
	baseURL := c.GetString("base_url")
	if baseURL == "" {
		baseURL = "/"
	}

	if h.oidc == nil {
		log.Warn().Msg("OIDC callback received but provider is not ready")
		c.Redirect(http.StatusTemporaryRedirect, loginErrorRedirectURL(baseURL, "oidc_unavailable"))
		return
	}

	code := c.Query("code")
	if code == "" {
		log.Error().Msg("no code received in callback")
		c.Redirect(http.StatusTemporaryRedirect, loginErrorRedirectURL(baseURL, "invalid_code"))
		return
	}

	state := c.Query("state")
	if state == "" {
		log.Error().Msg("no state received in callback")
		c.Redirect(http.StatusTemporaryRedirect, loginErrorRedirectURL(baseURL, "invalid_state"))
		return
	}

	// Always try PKCE first for enhanced security
	// Retrieve PKCE code verifier
	codeVerifier, exists := h.getPKCEVerifier(state)
	if !exists {
		log.Error().Str("state", state).Msg("no PKCE verifier found for state")
		c.Redirect(http.StatusTemporaryRedirect, loginErrorRedirectURL(baseURL, "invalid_state"))
		return
	}

	// Exchange code for token using PKCE
	token, err := h.oidc.ExchangeCodeWithPKCE(c.Request.Context(), code, codeVerifier)
	if err != nil {
		log.Error().Err(err).Msg("failed to exchange code for token with PKCE")
		c.Redirect(http.StatusTemporaryRedirect, loginErrorRedirectURL(baseURL, "token_exchange"))
		return
	}

	log.Debug().
		Str("state", state).
		Msg("Token exchange successful")

	// Get the ID token
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		log.Error().Msg("no id_token in oauth2 token")
		c.Redirect(http.StatusTemporaryRedirect, loginErrorRedirectURL(baseURL, "missing_id_token"))
		return
	}

	// Verify the token
	idClaims, err := h.oidc.VerifyTokenWithClaims(c.Request.Context(), rawIDToken)
	if err != nil {
		log.Error().Err(err).Msg("invalid ID token")
		c.Redirect(http.StatusTemporaryRedirect, loginErrorRedirectURL(baseURL, "invalid_token"))
		return
	}

	sessionClaims := SessionClaims{
		Version:    sessionClaimsVersion,
		Type:       sessionTypeOIDC,
		Subject:    idClaims.Subject,
		Username:   pickOIDCUsername(idClaims),
		IDTokenExp: idClaims.Expiry,
	}

	if token.RefreshToken != "" && h.sessionSecret != "" {
		if err := h.setRefreshToken(&sessionClaims, token.RefreshToken); err != nil {
			log.Debug().Err(err).Msg("Failed to store refresh token")
		}
	}

	h.refreshSession(c, "", &sessionClaims)

	c.Redirect(http.StatusTemporaryRedirect, baseURL)
}
