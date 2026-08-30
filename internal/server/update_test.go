// Copyright (c) 2024-2026, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package server

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/autobrr/netronome/internal/config"
	"github.com/autobrr/netronome/internal/update"
)

func TestLatestVersionRouteUsesBaseURLAndReturnsCachedRelease(t *testing.T) {
	gin.SetMode(gin.TestMode)
	releaseServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"tag_name":"v1.2.4","name":"Netronome 1.2.4","html_url":"https://example.test/releases/v1.2.4","published_at":"2026-07-30T14:00:00+02:00"}`)
	}))
	defer releaseServer.Close()

	checker := update.New(true, "1.2.3", serverClient(t, releaseServer.URL))
	require.NoError(t, checker.Check(context.Background()))

	cfg := config.New()
	cfg.Server.BaseURL = "/netronome"
	cfg.Auth.Whitelist = []string{"127.0.0.1/32"}
	server := NewServer(nil, nil, nil, cfg, nil, nil, nil, nil)
	server.SetUpdateChecker(checker)
	server.RegisterRoutes()

	response := httptest.NewRecorder()
	request := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/netronome/api/version/latest", nil)
	request.RemoteAddr = "127.0.0.1:12345"
	server.Router.ServeHTTP(response, request)

	require.Equal(t, http.StatusOK, response.Code)
	assert.JSONEq(t, `{"tag_name":"v1.2.4","name":"Netronome 1.2.4","html_url":"https://example.test/releases/v1.2.4","published_at":"2026-07-30T12:00:00Z"}`, response.Body.String())
}

func TestLatestVersionRouteReturnsNoContentWithoutCachedRelease(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := config.New()
	cfg.Auth.Whitelist = []string{"127.0.0.1/32"}
	server := NewServer(nil, nil, nil, cfg, nil, nil, nil, nil)
	server.SetUpdateChecker(update.New(false, "1.2.3", nil))
	server.RegisterRoutes()

	response := httptest.NewRecorder()
	request := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/version/latest", nil)
	request.RemoteAddr = "127.0.0.1:12345"
	server.Router.ServeHTTP(response, request)

	assert.Equal(t, http.StatusNoContent, response.Code)
	assert.Empty(t, response.Body.String())
}

func serverClient(t *testing.T, target string) *http.Client {
	t.Helper()
	targetURL, err := url.Parse(target)
	require.NoError(t, err)

	return &http.Client{Transport: serverRoundTripper(func(r *http.Request) (*http.Response, error) {
		clone := r.Clone(r.Context())
		clone.URL.Scheme = targetURL.Scheme
		clone.URL.Host = targetURL.Host
		return http.DefaultTransport.RoundTrip(clone)
	})}
}

type serverRoundTripper func(*http.Request) (*http.Response, error)

func (f serverRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}
