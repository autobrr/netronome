// Copyright (c) 2024-2026, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"

	"github.com/autobrr/netronome/internal/auth"
	"github.com/autobrr/netronome/internal/database"
)

func TestIsWhitelisted(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name       string
		remoteAddr string
		whitelist  []string
		expected   bool
	}{
		{
			name:       "IP in whitelist",
			remoteAddr: "192.168.1.100",
			whitelist:  []string{"192.168.1.0/24"},
			expected:   true,
		},
		{
			name:       "IP not in whitelist",
			remoteAddr: "10.0.0.5",
			whitelist:  []string{"192.168.1.0/24"},
			expected:   false,
		},
		{
			name:       "Multiple whitelists",
			remoteAddr: "10.0.0.5",
			whitelist:  []string{"192.168.1.0/24", "10.0.0.0/8"},
			expected:   true,
		},
		{
			name:       "Empty whitelist",
			remoteAddr: "192.168.1.100",
			whitelist:  []string{},
			expected:   false,
		},
		{
			name:       "Invalid CIDR",
			remoteAddr: "192.168.1.100",
			whitelist:  []string{"invalid"},
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(recorder)
			c.Request = &http.Request{RemoteAddr: tt.remoteAddr + ":12345"}

			got := isWhitelisted(c, tt.whitelist)
			assert.Equal(t, tt.expected, got, "isWhitelisted() result mismatch")
		})
	}
}

type stubDB struct {
	database.Service
	queryRow func(ctx context.Context, query string, args ...interface{}) *sql.Row
}

func (s stubDB) QueryRow(ctx context.Context, query string, args ...interface{}) *sql.Row {
	return s.queryRow(ctx, query, args...)
}

func newAuthStatusTestDB(t *testing.T) database.Service {
	t.Helper()

	db, err := sql.Open("sqlite", "file::memory:?cache=shared")
	require.NoError(t, err)

	t.Cleanup(func() {
		require.NoError(t, db.Close())
	})

	_, err = db.Exec("CREATE TABLE users (id INTEGER PRIMARY KEY)")
	require.NoError(t, err)

	return stubDB{
		queryRow: func(ctx context.Context, query string, args ...interface{}) *sql.Row {
			return db.QueryRowContext(ctx, query, args...)
		},
	}
}

func TestCheckRegistrationStatusReportsOIDCState(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name                   string
		oidc                   *auth.OIDCConfig
		oidcConfigured         bool
		expectedOIDCConfigured bool
		expectedOIDCReady      bool
	}{
		{
			name:                   "configured but provider unavailable",
			oidc:                   nil,
			oidcConfigured:         true,
			expectedOIDCConfigured: true,
			expectedOIDCReady:      false,
		},
		{
			name:                   "configured and provider ready",
			oidc:                   &auth.OIDCConfig{},
			oidcConfigured:         true,
			expectedOIDCConfigured: true,
			expectedOIDCReady:      true,
		},
		{
			name:                   "not configured and provider unavailable",
			oidc:                   nil,
			oidcConfigured:         false,
			expectedOIDCConfigured: false,
			expectedOIDCReady:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewAuthHandler(newAuthStatusTestDB(t), tt.oidc, tt.oidcConfigured, "", nil)

			recorder := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(recorder)
			c.Request = httptest.NewRequest(http.MethodGet, "/api/auth/status", nil)

			handler.CheckRegistrationStatus(c)

			require.Equal(t, http.StatusOK, recorder.Code)

			var response struct {
				HasUsers       bool `json:"hasUsers"`
				OIDCConfigured bool `json:"oidcConfigured"`
				OIDCReady      bool `json:"oidcReady"`
			}
			require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
			assert.False(t, response.HasUsers)
			assert.Equal(t, tt.expectedOIDCConfigured, response.OIDCConfigured)
			assert.Equal(t, tt.expectedOIDCReady, response.OIDCReady)
		})
	}
}

func TestHandleOIDCLoginRedirectUsesLoginPathUnderSubpathBaseURL(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewAuthHandler(nil, nil, true, "", nil)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/auth/oidc/login", nil)
	c.Set("base_url", "/netronome")

	handler.handleOIDCLogin(c)

	require.Equal(t, http.StatusTemporaryRedirect, recorder.Code)
	assert.Equal(t, "/netronome/login?error=oidc_unavailable", recorder.Header().Get("Location"))
}

func TestHandleOIDCCallbackRedirectUsesLoginPathUnderSubpathBaseURL(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewAuthHandler(nil, nil, true, "", nil)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/auth/oidc/callback", nil)
	c.Set("base_url", "/netronome")

	handler.handleOIDCCallback(c)

	require.Equal(t, http.StatusTemporaryRedirect, recorder.Code)
	assert.Equal(t, "/netronome/login?error=oidc_unavailable", recorder.Header().Get("Location"))
}
