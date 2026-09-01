// Copyright (c) 2024-2026, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package monitor

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/autobrr/netronome/internal/types"
)

func TestAgentBaseURL(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"stream suffix", "http://host:8200/events?stream=live-data", "http://host:8200"},
		{"trailing slash", "http://host:8200/", "http://host:8200"},
		{"bare", "http://host:8200", "http://host:8200"},
		{"base path", "http://host:8200/agent/events?stream=live-data", "http://host:8200/agent"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := AgentBaseURL(tt.in); got != tt.want {
				t.Errorf("AgentBaseURL(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestAgentStreamURL(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"already suffixed", "http://host:8200/events?stream=live-data", "http://host:8200/events?stream=live-data"},
		{"trailing slash", "http://host:8200/", "http://host:8200/events?stream=live-data"},
		{"bare", "http://host:8200", "http://host:8200/events?stream=live-data"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := AgentStreamURL(tt.in); got != tt.want {
				t.Errorf("AgentStreamURL(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

// fakeAgent starts an httptest server that records the path and API key of the
// last request it received.
type fakeAgent struct {
	server *httptest.Server
	path   string
	apiKey string
}

func newFakeAgent(t *testing.T, body string) *fakeAgent {
	t.Helper()

	fa := &fakeAgent{}
	fa.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fa.path = r.URL.RequestURI()
		fa.apiKey = r.Header.Get("X-API-Key")
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, body)
	}))
	t.Cleanup(fa.server.Close)
	return fa
}

func (fa *fakeAgent) client(apiKey *string) *AgentClient {
	return NewAgentClient(&types.MonitorAgent{URL: fa.server.URL + "/events?stream=live-data", APIKey: apiKey})
}

func TestAgentClientEndpoints(t *testing.T) {
	key := "secret"
	empty := ""

	tests := []struct {
		name     string
		apiKey   *string
		call     func(*AgentClient) error
		wantPath string
		wantKey  string
	}{
		{
			name:     "system info",
			apiKey:   &key,
			call:     func(c *AgentClient) error { _, err := c.SystemInfo(context.Background()); return err },
			wantPath: "/system/info",
			wantKey:  key,
		},
		{
			name:     "hardware stats",
			apiKey:   &key,
			call:     func(c *AgentClient) error { _, err := c.HardwareStats(context.Background()); return err },
			wantPath: "/system/hardware",
			wantKey:  key,
		},
		{
			name:     "peak stats",
			apiKey:   &key,
			call:     func(c *AgentClient) error { _, err := c.PeakStats(context.Background()); return err },
			wantPath: "/stats/peaks",
			wantKey:  key,
		},
		{
			name:     "historical without interface",
			apiKey:   &key,
			call:     func(c *AgentClient) error { _, err := c.Historical(context.Background(), ""); return err },
			wantPath: "/export/historical",
			wantKey:  key,
		},
		{
			name:     "historical with interface",
			apiKey:   &key,
			call:     func(c *AgentClient) error { _, err := c.Historical(context.Background(), "eth 0"); return err },
			wantPath: "/export/historical?interface=eth+0",
			wantKey:  key,
		},
		{
			// The agent info endpoint is public, so the API key must not be sent.
			name:     "info sends no api key",
			apiKey:   &key,
			call:     func(c *AgentClient) error { _, err := c.Info(context.Background()); return err },
			wantPath: "/netronome/info",
			wantKey:  "",
		},
		{
			name:     "no api key configured",
			apiKey:   nil,
			call:     func(c *AgentClient) error { _, err := c.SystemInfo(context.Background()); return err },
			wantPath: "/system/info",
			wantKey:  "",
		},
		{
			name:     "empty api key configured",
			apiKey:   &empty,
			call:     func(c *AgentClient) error { _, err := c.SystemInfo(context.Background()); return err },
			wantPath: "/system/info",
			wantKey:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fa := newFakeAgent(t, `{"version":"1.2.3"}`)
			if err := tt.call(fa.client(tt.apiKey)); err != nil {
				t.Fatalf("call failed: %v", err)
			}
			if fa.path != tt.wantPath {
				t.Errorf("path = %q, want %q", fa.path, tt.wantPath)
			}
			if fa.apiKey != tt.wantKey {
				t.Errorf("X-API-Key = %q, want %q", fa.apiKey, tt.wantKey)
			}
		})
	}
}

func TestAgentClientInfoDecodes(t *testing.T) {
	fa := newFakeAgent(t, `{"version":"1.2.3"}`)

	info, err := fa.client(nil).Info(context.Background())
	if err != nil {
		t.Fatalf("Info failed: %v", err)
	}
	if info.Version != "1.2.3" {
		t.Errorf("version = %q, want %q", info.Version, "1.2.3")
	}
}

// trackingBody reports whether the response body was closed.
type trackingBody struct {
	io.Reader
	closed bool
}

func (b *trackingBody) Close() error {
	b.closed = true
	return nil
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestAgentClientClosesBodyOnErrorStatus(t *testing.T) {
	body := &trackingBody{Reader: strings.NewReader("nope")}
	client := NewAgentClient(&types.MonitorAgent{URL: "http://agent.invalid"})
	client.http = &http.Client{Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusInternalServerError, Body: body, Request: r}, nil
	})}

	_, err := client.SystemInfo(context.Background())

	var statusErr *StatusError
	if !errors.As(err, &statusErr) {
		t.Fatalf("err = %v, want *StatusError", err)
	}
	if statusErr.StatusCode != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", statusErr.StatusCode, http.StatusInternalServerError)
	}
	if !body.closed {
		t.Error("response body was not closed")
	}
}

// blockedServer returns an agent that answers no request until the test ends.
func blockedServer(t *testing.T) *httptest.Server {
	t.Helper()

	blocked := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-blocked
	}))
	t.Cleanup(func() {
		close(blocked)
		server.Close()
	})
	return server
}

func TestAgentClientTimeoutCeiling(t *testing.T) {
	client := NewAgentClient(&types.MonitorAgent{URL: blockedServer(t).URL})

	// The caller passes a context with no deadline; the ceiling must still end the request.
	start := time.Now()
	_, err := client.get(context.Background(), "/system/info", 50*time.Millisecond, withAPIKey)

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("err = %v, want context.DeadlineExceeded", err)
	}
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Errorf("request took %v, ceiling did not apply", elapsed)
	}
}

func TestAgentClientCallerCancellation(t *testing.T) {
	client := NewAgentClient(&types.MonitorAgent{URL: blockedServer(t).URL})

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()
	defer cancel()

	_, err := client.SystemInfo(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
}
