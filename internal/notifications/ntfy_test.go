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
		wantErr  string
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
			name:     "access token as password",
			input:    "ntfy://:tk_token@ntfy/htpc",
			wantURL:  "https://ntfy/htpc",
			wantPass: "tk_token",
		},
		{
			name:    "with scheme override to http",
			input:   "ntfy://ntfy.local/test?scheme=http",
			wantURL: "http://ntfy.local/test",
		},
		{
			// Shoutrrr resolves query keys case-insensitively.
			name:    "scheme query key is case insensitive",
			input:   "ntfy://:tk_token@ntfy/htpc?Scheme=http",
			wantURL: "http://ntfy/htpc",

			wantPass: "tk_token",
		},
		{
			name:    "scheme value is case insensitive",
			input:   "ntfy://ntfy.local/test?scheme=HTTP",
			wantURL: "http://ntfy.local/test",
		},
		{
			name:    "explicit https scheme",
			input:   "ntfy://ntfy.example.com/alerts?scheme=https",
			wantURL: "https://ntfy.example.com/alerts",
		},
		{
			name:    "tailscale host",
			input:   "ntfy://ntfy.tail-net.ts.net/netronome",
			wantURL: "https://ntfy.tail-net.ts.net/netronome",
		},
		{
			name:    "missing topic",
			input:   "ntfy://ntfy.example.com",
			wantErr: "must include a topic",
		},
		{
			name:    "empty path",
			input:   "ntfy://ntfy.example.com/",
			wantErr: "must include a topic",
		},
		{
			name:    "missing host with topic",
			input:   "ntfy:///alerts",
			wantErr: "must include a host",
		},
		{
			name:    "invalid scheme",
			input:   "ntfy://ntfy.example.com/alerts?scheme=ftp",
			wantErr: "invalid ntfy scheme",
		},
		{
			name:    "invalid priority",
			input:   "ntfy://ntfy.example.com/alerts?priority=critical",
			wantErr: "invalid ntfy priority",
		},
		{
			name:    "invalid bool",
			input:   "ntfy://ntfy.example.com/alerts?cache=maybe",
			wantErr: "accepted values are 1, true, yes or 0, false, no",
		},
		{
			name:    "unknown query key",
			input:   "ntfy://ntfy.example.com/alerts?bogus=1",
			wantErr: "bogus is not a valid ntfy config key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := parseNtfyURL(tt.input)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantURL, cfg.apiURL)
			assert.Equal(t, tt.wantUser, cfg.username)
			assert.Equal(t, tt.wantPass, cfg.password)
		})
	}
}

func TestParseNtfyURLOptions(t *testing.T) {
	t.Run("parses message options", func(t *testing.T) {
		cfg, err := parseNtfyURL("ntfy://ntfy.example.com/alerts?title=Hi&priority=high&tags=warning,skull&click=https://example.com&attach=https://example.com/a.png&filename=a.png&email=me@example.com&icon=https://example.com/i.png&cache=no&firebase=no&delay=30min")
		require.NoError(t, err)

		assert.Equal(t, "Hi", cfg.title)
		assert.Equal(t, "high", cfg.priority)
		assert.Equal(t, []string{"warning", "skull"}, cfg.tags)
		assert.Equal(t, "https://example.com", cfg.click)
		assert.Equal(t, "https://example.com/a.png", cfg.attach)
		assert.Equal(t, "a.png", cfg.filename)
		assert.Equal(t, "me@example.com", cfg.email)
		assert.Equal(t, "https://example.com/i.png", cfg.icon)
		assert.False(t, cfg.cache)
		assert.False(t, cfg.firebase)
		assert.Equal(t, "30min", cfg.delay)
	})

	t.Run("cache and firebase default to enabled", func(t *testing.T) {
		cfg, err := parseNtfyURL("ntfy://ntfy.example.com/alerts")
		require.NoError(t, err)

		assert.True(t, cfg.cache)
		assert.True(t, cfg.firebase)
		assert.Empty(t, cfg.priority)
	})

	t.Run("normalizes numeric and alias priorities", func(t *testing.T) {
		for value, want := range map[string]string{
			"1": "min", "2": "low", "3": "default", "4": "high", "5": "max",
			"urgent": "max", "Max": "max", "MIN": "min",
		} {
			cfg, err := parseNtfyURL("ntfy://ntfy.example.com/alerts?priority=" + value)
			require.NoError(t, err, value)
			assert.Equal(t, want, cfg.priority, value)
		}
	})

	t.Run("delay accepts at and in aliases", func(t *testing.T) {
		for _, key := range []string{"delay", "at", "in", "At", "IN"} {
			cfg, err := parseNtfyURL("ntfy://ntfy.example.com/alerts?" + key + "=tomorrow")
			require.NoError(t, err, key)
			assert.Equal(t, "tomorrow", cfg.delay, key)
		}
	})

	t.Run("actions are separated by semicolons", func(t *testing.T) {
		cfg, err := parseNtfyURL("ntfy://ntfy.example.com/alerts?actions=view, Open, https://example.com; http, Ack, https://example.com/ack")
		require.NoError(t, err)
		assert.Equal(t, []string{"view, Open, https://example.com", "http, Ack, https://example.com/ack"}, cfg.actions)
	})
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
		err := sendNtfy(ntfyURL, "Netronome", "Hello from Netronome")

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
		err := sendNtfy(ntfyURL, "Netronome", "test message")

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
		err := sendNtfy(ntfyURL, "Netronome", "authenticated message")

		require.NoError(t, err)
		assert.Equal(t, "myuser", receivedUser)
		assert.Equal(t, "mypass", receivedPass)
	})

	t.Run("passes an access token as the basic auth password", func(t *testing.T) {
		var receivedUser string
		var receivedPass string

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			receivedUser, receivedPass, _ = r.BasicAuth()
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		ntfyURL := "ntfy://:tk_token@" + server.Listener.Addr().String() + "/alerts?Scheme=http"
		err := sendNtfy(ntfyURL, "Netronome", "token message")

		require.NoError(t, err)
		assert.Empty(t, receivedUser)
		assert.Equal(t, "tk_token", receivedPass)
	})

	t.Run("sends message option headers", func(t *testing.T) {
		var received http.Header

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			received = r.Header.Clone()
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		ntfyURL := "ntfy://" + server.Listener.Addr().String() + "/alerts?scheme=http&priority=5&tags=warning,skull&click=https://example.com&icon=https://example.com/i.png&filename=a.png&email=me@example.com&cache=no&firebase=no&delay=30min"
		err := sendNtfy(ntfyURL, "Netronome: Speedtest", "message")

		require.NoError(t, err)
		assert.Equal(t, "Netronome: Speedtest", received.Get("Title"))
		assert.Equal(t, "max", received.Get("Priority"))
		assert.Equal(t, "warning,skull", received.Get("Tags"))
		assert.Equal(t, "https://example.com", received.Get("Click"))
		assert.Equal(t, "https://example.com/i.png", received.Get("X-Icon"))
		assert.Equal(t, "a.png", received.Get("Filename"))
		assert.Equal(t, "me@example.com", received.Get("Email"))
		assert.Equal(t, "30min", received.Get("Delay"))
		assert.Equal(t, "no", received.Get("Cache"))
		assert.Equal(t, "no", received.Get("Firebase"))
	})

	t.Run("omits optional headers when unset", func(t *testing.T) {
		var received http.Header

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			received = r.Header.Clone()
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		ntfyURL := "ntfy://" + server.Listener.Addr().String() + "/alerts?scheme=http"
		err := sendNtfy(ntfyURL, "", "message")

		require.NoError(t, err)
		for _, key := range []string{"Title", "Priority", "Tags", "Delay", "Actions", "Click", "Attach", "X-Icon", "Filename", "Email", "Cache", "Firebase"} {
			assert.Empty(t, received.Get(key), key)
		}
	})

	t.Run("caller title overrides the url title", func(t *testing.T) {
		var received http.Header

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			received = r.Header.Clone()
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		ntfyURL := "ntfy://" + server.Listener.Addr().String() + "/alerts?scheme=http&title=FromURL"

		require.NoError(t, sendNtfy(ntfyURL, "FromCaller", "message"))
		assert.Equal(t, "FromCaller", received.Get("Title"))

		require.NoError(t, sendNtfy(ntfyURL, "", "message"))
		assert.Equal(t, "FromURL", received.Get("Title"))
	})
}
