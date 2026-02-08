// Copyright (c) 2024-2026, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package notifications

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseNtfyURL(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantURL  string
		wantUser string
		wantPass string
		wantErr  bool
	}{
		{
			name:    "simple host and topic",
			input:   "ntfy://ntfy.example.com/alerts",
			wantURL: "https://ntfy.example.com/alerts",
		},
		{
			name:    "host with port and topic",
			input:   "ntfy://ntfy.example.com:8080/mytopic",
			wantURL: "https://ntfy.example.com:8080/mytopic",
		},
		{
			name:     "with basic auth credentials",
			input:    "ntfy://user:pass@ntfy.example.com/alerts",
			wantURL:  "https://ntfy.example.com/alerts",
			wantUser: "user",
			wantPass: "pass",
		},
		{
			name:    "with scheme override to http",
			input:   "ntfy://ntfy.local/test?scheme=http",
			wantURL: "http://ntfy.local/test",
		},
		{
			name:    "tailscale host",
			input:   "ntfy://ntfy.tail-net.ts.net/netronome",
			wantURL: "https://ntfy.tail-net.ts.net/netronome",
		},
		{
			name:    "missing topic",
			input:   "ntfy://ntfy.example.com",
			wantErr: true,
		},
		{
			name:    "empty path",
			input:   "ntfy://ntfy.example.com/",
			wantErr: true,
		},
		{
			name:    "missing host with topic",
			input:   "ntfy:///alerts",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := parseNtfyURL(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantURL, cfg.apiURL)
			assert.Equal(t, tt.wantUser, cfg.username)
			assert.Equal(t, tt.wantPass, cfg.password)
		})
	}
}

func TestIsNtfyURL(t *testing.T) {
	assert.True(t, isNtfyURL("ntfy://ntfy.sh/topic"))
	assert.True(t, isNtfyURL("ntfy://user:pass@host/topic"))
	assert.False(t, isNtfyURL("discord://token@id"))
	assert.False(t, isNtfyURL("pushover://token@user"))
	assert.False(t, isNtfyURL(""))
}

func TestSendNtfy(t *testing.T) {
	t.Run("sends plain text with correct content type", func(t *testing.T) {
		var receivedBody string
		var receivedContentType string

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			receivedContentType = r.Header.Get("Content-Type")
			body, _ := io.ReadAll(r.Body)
			receivedBody = string(body)
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"id":"test"}`))
		}))
		defer server.Close()

		ntfyURL := "ntfy://" + server.Listener.Addr().String() + "/test-topic?scheme=http"
		err := sendNtfy(ntfyURL, "Hello from Netronome")

		require.NoError(t, err)
		assert.Equal(t, "text/plain", receivedContentType)
		assert.Equal(t, "Hello from Netronome", receivedBody)
	})

	t.Run("handles server error response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error":"invalid request","code":40024}`))
		}))
		defer server.Close()

		ntfyURL := "ntfy://" + server.Listener.Addr().String() + "/test-topic?scheme=http"
		err := sendNtfy(ntfyURL, "test message")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "400")
	})

	t.Run("passes basic auth credentials", func(t *testing.T) {
		var receivedUser string
		var receivedPass string

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			receivedUser, receivedPass, _ = r.BasicAuth()
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"id":"test"}`))
		}))
		defer server.Close()

		ntfyURL := "ntfy://myuser:mypass@" + server.Listener.Addr().String() + "/alerts?scheme=http"
		err := sendNtfy(ntfyURL, "authenticated message")

		require.NoError(t, err)
		assert.Equal(t, "myuser", receivedUser)
		assert.Equal(t, "mypass", receivedPass)
	})
}
