// Copyright (c) 2024-2026, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package server

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
)

const (
	sessionClaimsVersion = 1
	sessionTypeLocal     = "local"
	sessionTypeOIDC      = "oidc"
)

const refreshTokenAAD = "netronome:oidc:refresh" // #nosec G101 -- label for AES-GCM AAD, not a credential

type SessionClaims struct {
	Version      int    `json:"v"`
	Type         string `json:"type"`
	Username     string `json:"username,omitempty"`
	Subject      string `json:"sub,omitempty"`
	IDTokenExp   int64  `json:"id_exp,omitempty"`
	RefreshToken string `json:"rt,omitempty"`
	LastRefresh  int64  `json:"lrt,omitempty"`
}

func encodeSessionClaims(claims SessionClaims) (string, error) {
	if claims.Version == 0 {
		claims.Version = sessionClaimsVersion
	}
	if claims.Type == "" {
		return "", errors.New("session claim type required")
	}
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(payload), nil
}

func decodeSessionClaims(token string) (*SessionClaims, bool) {
	payload, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return nil, false
	}
	var claims SessionClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, false
	}
	if claims.Version != sessionClaimsVersion || claims.Type == "" {
		return nil, false
	}
	return &claims, true
}

func encryptRefreshToken(secret, token string) (string, error) {
	if secret == "" {
		return "", errors.New("session secret required")
	}
	if token == "" {
		return "", nil
	}

	key := sha256.Sum256([]byte(secret))
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ciphertext := gcm.Seal(nil, nonce, []byte(token), []byte(refreshTokenAAD))
	payload := make([]byte, 0, len(nonce)+len(ciphertext))
	payload = append(payload, nonce...)
	payload = append(payload, ciphertext...)
	return base64.RawURLEncoding.EncodeToString(payload), nil
}

func decryptRefreshToken(secret, token string) (string, error) {
	if secret == "" {
		return "", errors.New("session secret required")
	}
	if token == "" {
		return "", nil
	}

	payload, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return "", err
	}

	key := sha256.Sum256([]byte(secret))
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	if len(payload) < gcm.NonceSize() {
		return "", errors.New("invalid refresh token payload")
	}
	nonce := payload[:gcm.NonceSize()]
	ciphertext := payload[gcm.NonceSize():]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, []byte(refreshTokenAAD))
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}
