// Copyright (c) 2024-2026, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package update

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"runtime"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsNewer(t *testing.T) {
	tests := []struct {
		name    string
		current string
		latest  string
		want    bool
	}{
		{name: "newer stable release", current: "v1.2.3", latest: "v1.2.4", want: true},
		{name: "same release", current: "1.2.3", latest: "v1.2.3", want: false},
		{name: "older release", current: "1.2.4", latest: "v1.2.3", want: false},
		{name: "stable current ignores newer prerelease", current: "1.2.3", latest: "1.3.0-rc.1", want: false},
		{name: "prerelease current accepts newer stable", current: "1.3.0-rc.1", latest: "1.3.0", want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := IsNewer(tt.current, tt.latest)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCheckSkipsDevelopmentVersionsWithoutNetworkRequest(t *testing.T) {
	versions := []string{"", "dev", "develop", "main", "latest", "pr-123", "1.2.3-dev", "1.2.3-develop", "1.2.3 (devel)"}

	for _, current := range versions {
		t.Run(current, func(t *testing.T) {
			var requests atomic.Int32
			client := &http.Client{Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
				requests.Add(1)
				return nil, assert.AnError
			})}
			checker := New(true, current, client)

			require.NoError(t, checker.Check(context.Background()))
			assert.Zero(t, requests.Load())
			assert.Nil(t, checker.Latest())
		})
	}
}

func TestCheckCachesNewerReleaseAndSendsReleaseHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Accept") != "application/vnd.github.v3+json" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if got, want := r.UserAgent(), "netronome/1.2.3 ("+runtime.GOOS+" "+runtime.GOARCH+")"; got != want {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		_, _ = io.WriteString(w, `{"tag_name":"v1.2.4","name":"Netronome 1.2.4","html_url":"https://example.test/releases/v1.2.4","published_at":"2026-07-30T12:00:00Z","ignored":"value"}`)
	}))
	defer server.Close()

	checker := New(true, "1.2.3", rewriteClient(t, server.URL))
	require.NoError(t, checker.Check(context.Background()))

	assert.Equal(t, &Release{
		TagName:     "v1.2.4",
		Name:        "Netronome 1.2.4",
		HTMLURL:     "https://example.test/releases/v1.2.4",
		PublishedAt: time.Date(2026, 7, 30, 12, 0, 0, 0, time.UTC),
	}, checker.Latest())
}

func TestCheckRetainsCachedUpdateOnFailureAndClearsItWhenCurrent(t *testing.T) {
	var mode atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		switch mode.Load() {
		case 0:
			_, _ = io.WriteString(w, `{"tag_name":"v1.2.4","html_url":"https://example.test/releases/v1.2.4","published_at":"2026-07-30T12:00:00Z"}`)
		case 1:
			w.WriteHeader(http.StatusCreated)
			_, _ = io.WriteString(w, `{"tag_name":"v1.2.5","html_url":"https://example.test/releases/v1.2.5","published_at":"2026-07-30T12:00:00Z"}`)
		case 2:
			_, _ = io.WriteString(w, `{"tag_name":"v1.2.3","html_url":"https://example.test/releases/v1.2.3","published_at":"2026-07-30T12:00:00Z"}`)
		default:
			_, _ = io.WriteString(w, `{"tag_name":"v1.2.5","html_url":"javascript:alert(1)","published_at":"2026-07-30T12:00:00Z"}`)
		}
	}))
	defer server.Close()

	checker := New(true, "1.2.3", rewriteClient(t, server.URL))
	require.NoError(t, checker.Check(context.Background()))
	first := checker.Latest()
	require.NotNil(t, first)

	mode.Store(1)
	require.Error(t, checker.Check(context.Background()))
	assert.Equal(t, first, checker.Latest())

	mode.Store(3)
	require.Error(t, checker.Check(context.Background()))
	assert.Equal(t, first, checker.Latest())

	mode.Store(2)
	require.NoError(t, checker.Check(context.Background()))
	assert.Nil(t, checker.Latest())
}

func TestCheckCancellationRetainsCachedUpdate(t *testing.T) {
	checker := New(true, "1.2.3", &http.Client{Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		<-r.Context().Done()
		return nil, r.Context().Err()
	})})
	checker.mu.Lock()
	checker.latest = &Release{TagName: "v1.2.4"}
	checker.mu.Unlock()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	require.ErrorIs(t, checker.Check(ctx), context.Canceled)
	assert.Equal(t, "v1.2.4", checker.Latest().TagName)
}

func TestStartStopsWhenContextIsCancelledBeforeInitialCheck(t *testing.T) {
	checker := New(true, "1.2.3", &http.Client{})
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		checker.Start(ctx)
		close(done)
	}()
	cancel()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("checker did not stop after context cancellation")
	}
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func rewriteClient(t *testing.T, target string) *http.Client {
	t.Helper()
	targetURL, err := url.Parse(target)
	require.NoError(t, err)

	return &http.Client{Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		clone := r.Clone(r.Context())
		clone.URL.Scheme = targetURL.Scheme
		clone.URL.Host = targetURL.Host
		return http.DefaultTransport.RoundTrip(clone)
	})}
}
