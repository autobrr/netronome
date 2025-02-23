// Copyright (c) 2024-2025, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package server

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
)

func (h *AuthHandler) handleOIDCLogin(w http.ResponseWriter, r *http.Request) {
	authURL := h.oidc.AuthURL()
	http.Redirect(w, r, authURL, http.StatusTemporaryRedirect)
}

func (h *AuthHandler) handleOIDCCallback(w http.ResponseWriter, r *http.Request) {
	//baseURL := c.GetString("base_url")
	//if baseURL == "" {
	//	baseURL = "/"
	//}
	code := chi.URLParam(r, "code")
	if code == "" {
		log.Error().Msg("no code received in callback")
		http.Redirect(w, r, h.baseUrl+"login?error=invalid_code", http.StatusTemporaryRedirect)
		return
	}

	// Exchange code for token using the exported OAuth2Config
	token, err := h.oidc.OAuth2Config.Exchange(r.Context(), code)
	if err != nil {
		log.Error().Err(err).Msg("failed to exchange code for token")
		//c.Redirect(http.StatusTemporaryRedirect, baseURL+"login?error=token_exchange")
		http.Redirect(w, r, h.baseUrl+"login?error=token_exchange", http.StatusTemporaryRedirect)
		return
	}

	// Get the ID token
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		log.Error().Msg("no id_token in oauth2 token")
		//c.Redirect(http.StatusTemporaryRedirect, baseURL+"login?error=missing_id_token")
		http.Redirect(w, r, h.baseUrl+"login?error=missing_id_token", http.StatusTemporaryRedirect)
		return
	}

	// Verify the token
	if err := h.oidc.VerifyToken(r.Context(), rawIDToken); err != nil {
		log.Error().Err(err).Msg("invalid ID token")
		//c.Redirect(http.StatusTemporaryRedirect, baseURL+"login?error=invalid_token")
		http.Redirect(w, r, h.baseUrl+"login?error=invalid_token", http.StatusTemporaryRedirect)
		return
	}

	sessionToken, err := generateSecureToken(32)
	if err != nil {
		log.Error().Err(err).Msg("failed to generate session token")
		//c.Redirect(http.StatusTemporaryRedirect, baseURL+"login?error=server_error")
		http.Redirect(w, r, h.baseUrl+"login?error=server_error", http.StatusTemporaryRedirect)
		return
	}

	if err := h.storeSession(sessionToken); err != nil {
		log.Error().Err(err).Msg("failed to store session")
		//c.Redirect(http.StatusTemporaryRedirect, baseURL+"login?error=server_error")
		http.Redirect(w, r, h.baseUrl+"login?error=server_error", http.StatusTemporaryRedirect)
		return
	}

	// TODO Set session cookie with the random token instead
	//c.SetCookie(
	//	"session",
	//	sessionToken,
	//	int((30 * 24 * time.Hour).Seconds()), // 30 days
	//	h.baseUrl,                              // Use baseURL for cookie path
	//	"",
	//	c.Request.URL.Scheme == "https",
	//	true,
	//)

	//c.Redirect(http.StatusTemporaryRedirect, baseURL)
	http.Redirect(w, r, h.baseUrl, http.StatusTemporaryRedirect)
}

func generateSecureToken(length int) (string, error) {
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
