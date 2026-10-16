// Copyright (c) 2024-2026, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package notifications

import (
	"fmt"
	"io"
	"maps"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/nicholas-fedor/shoutrrr/pkg/format"
	"github.com/nicholas-fedor/shoutrrr/pkg/services/push/ntfy"
	"github.com/rs/zerolog/log"
)

// notificationHTTPClient is the shared HTTP client for all notification
// requests, from both the direct ntfy path and the shoutrrr router.
var notificationHTTPClient = &http.Client{
	Timeout: 30 * time.Second,
}

// sendNtfy sends a notification directly to an ntfy server. It does not use
// the ntfy service in Shoutrrr, because Shoutrrr fails the whole channel on
// the first query key that it cannot parse. parseNtfyURL logs and ignores
// these keys instead. Stored URLs predate this parsing and must continue to
// work.
//
// All other behavior copies the config and request shape of Shoutrrr, so URLs
// written for Shoutrrr do the same here. The title argument overrides a title
// from the URL, as send params do in Shoutrrr.
func sendNtfy(ntfyURL string, title string, message string) error {
	cfg, err := parseNtfyURL(ntfyURL)
	if err != nil {
		return fmt.Errorf("failed to parse ntfy URL: %w", err)
	}

	if title != "" {
		cfg.Title = title
	}

	// The credentials go in the Basic auth header below, not in the URL,
	// because credentials in the URL can leak into logged errors.
	scheme := cfg.Scheme
	if cfg.DisableTLS {
		scheme = "http"
	}
	apiURL := url.URL{Scheme: scheme, Host: cfg.Host, Path: "/" + cfg.Topic}

	req, err := http.NewRequest(http.MethodPost, apiURL.String(), strings.NewReader(message))
	if err != nil {
		return fmt.Errorf("failed to create ntfy request: %w", err)
	}

	if cfg.Username != "" || cfg.Password != "" {
		req.SetBasicAuth(cfg.Username, cfg.Password)
	}

	req.Header.Set("Content-Type", "text/plain; charset=utf-8")

	// Mirrors shoutrrr/pkg/services/ntfy.sendAPI.
	setHeaderIfNotEmpty(req.Header, "Title", cfg.Title)
	setHeaderIfNotEmpty(req.Header, "Priority", cfg.Priority.String())
	setHeaderIfNotEmpty(req.Header, "Tags", strings.Join(cfg.Tags, ","))
	setHeaderIfNotEmpty(req.Header, "Delay", cfg.Delay)
	setHeaderIfNotEmpty(req.Header, "Actions", strings.Join(cfg.Actions, ";"))
	setHeaderIfNotEmpty(req.Header, "Click", cfg.Click)
	setHeaderIfNotEmpty(req.Header, "Attach", cfg.Attach)
	setHeaderIfNotEmpty(req.Header, "X-Icon", cfg.Icon)
	setHeaderIfNotEmpty(req.Header, "Filename", cfg.Filename)
	setHeaderIfNotEmpty(req.Header, "Email", cfg.Email)

	if !cfg.Cache {
		req.Header.Set("Cache", "no")
	}
	if !cfg.Firebase {
		req.Header.Set("Firebase", "no")
	}

	resp, err := notificationHTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send ntfy notification: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("ntfy server returned %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// parseNtfyURL converts a Shoutrrr ntfy URL (ntfy://[user:pass@]host/topic[?params])
// into Shoutrrr's own ntfy config, so query keys, aliases, casing, defaults and
// value parsing match Shoutrrr exactly.
func parseNtfyURL(ntfyURL string) (*ntfy.Config, error) {
	parsed, err := url.Parse(ntfyURL)
	if err != nil {
		return nil, err
	}

	topic := strings.TrimPrefix(parsed.Path, "/")
	if topic == "" {
		// Avoid echoing credentials from the URL back into logs/errors.
		return nil, fmt.Errorf("ntfy URL must include a topic")
	}
	if parsed.Host == "" {
		return nil, fmt.Errorf("ntfy URL must include a host")
	}

	cfg := &ntfy.Config{}
	pkr := format.NewPropKeyResolver(cfg)
	if err := pkr.SetDefaultProps(cfg); err != nil {
		return nil, err
	}

	cfg.Host = parsed.Host
	cfg.Topic = topic
	if parsed.User != nil {
		cfg.Username = parsed.User.Username()
		cfg.Password, _ = parsed.User.Password()
	}

	// Escape raw ";" so action separators survive query parsing, as Shoutrrr does.
	parsed.RawQuery = strings.ReplaceAll(parsed.RawQuery, ";", "%3b")
	query := parsed.Query()
	// Sorted so colliding keys ("?scheme=http&Scheme=https", "?delay=…&at=…")
	// resolve the same way on every send instead of by map order.
	for _, key := range slices.Sorted(maps.Keys(query)) {
		// A key we do not understand keeps its default rather than failing the
		// whole channel — stored URLs predate this parsing and must keep working.
		if err := pkr.Set(key, query[key][0]); err != nil {
			log.Warn().Str("key", key).Err(err).Msg("ignoring ntfy URL option")
		}
	}
	cfg.Scheme = strings.ToLower(cfg.Scheme)

	return cfg, nil
}

func setHeaderIfNotEmpty(headers http.Header, key string, value string) {
	if value != "" {
		headers.Set(key, value)
	}
}

// isNtfyURL checks whether a notification URL uses the ntfy scheme.
func isNtfyURL(u string) bool {
	return strings.HasPrefix(u, "ntfy://")
}
