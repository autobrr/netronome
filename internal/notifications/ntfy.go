// Copyright (c) 2024-2026, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package notifications

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// ntfyHTTPClient is a dedicated HTTP client for ntfy requests with a reasonable timeout.
var ntfyHTTPClient = &http.Client{
	Timeout: 30 * time.Second,
}

// ntfyConfig holds the parsed ntfy URL components.
type ntfyConfig struct {
	apiURL   string
	username string
	password string
}

// sendNtfy sends a notification directly to an ntfy server, bypassing Shoutrrr's
// ntfy implementation which has a bug where it removes the Content-Type header,
// causing newer ntfy servers to reject the plain-text body as invalid JSON.
func sendNtfy(ntfyURL string, message string) error {
	cfg, err := parseNtfyURL(ntfyURL)
	if err != nil {
		return fmt.Errorf("failed to parse ntfy URL: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, cfg.apiURL, strings.NewReader(message))
	if err != nil {
		return fmt.Errorf("failed to create ntfy request: %w", err)
	}

	req.Header.Set("Content-Type", "text/plain")

	if cfg.username != "" || cfg.password != "" {
		req.SetBasicAuth(cfg.username, cfg.password)
	}

	resp, err := ntfyHTTPClient.Do(req)
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
// into an ntfyConfig with the API endpoint URL and optional credentials.
func parseNtfyURL(ntfyURL string) (*ntfyConfig, error) {
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

	scheme := "https"
	if q := parsed.Query(); q.Get("scheme") == "http" {
		scheme = "http"
	}

	apiURL := &url.URL{
		Scheme: scheme,
		Host:   parsed.Host,
		Path:   "/" + topic,
	}

	cfg := &ntfyConfig{
		apiURL: apiURL.String(),
	}

	if parsed.User != nil {
		cfg.username = parsed.User.Username()
		cfg.password, _ = parsed.User.Password()
	}

	return cfg, nil
}

// isNtfyURL checks whether a notification URL uses the ntfy scheme.
func isNtfyURL(u string) bool {
	return strings.HasPrefix(u, "ntfy://")
}
