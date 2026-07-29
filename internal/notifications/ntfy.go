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

// ntfyConfig holds the parsed ntfy URL components. The fields mirror Shoutrrr's
// ntfy service config so that URLs written for Shoutrrr behave the same here.
type ntfyConfig struct {
	apiURL   string
	username string
	password string

	title    string
	priority string
	tags     []string
	actions  []string
	click    string
	attach   string
	filename string
	delay    string
	email    string
	icon     string
	cache    bool
	firebase bool
}

// ntfyPriorities maps the priority values Shoutrrr accepts to their canonical
// ntfy name. Shoutrrr matches the names case-insensitively and additionally
// accepts the numeric values and "urgent" as aliases.
var ntfyPriorities = map[string]string{
	"min":     "min",
	"low":     "low",
	"default": "default",
	"high":    "high",
	"max":     "max",
	"urgent":  "max",
	"1":       "min",
	"2":       "low",
	"3":       "default",
	"4":       "high",
	"5":       "max",
}

// ntfyQueryKeys lists the query parameters Shoutrrr's ntfy service supports.
// Keys are matched case-insensitively, matching Shoutrrr's PropKeyResolver, and
// "at"/"in" are aliases for "delay".
var ntfyQueryKeys = []string{
	"actions", "at", "attach", "cache", "click", "delay", "email",
	"filename", "firebase", "icon", "in", "priority", "scheme", "tags", "title",
}

// sendNtfy sends a notification directly to an ntfy server, bypassing Shoutrrr's
// ntfy implementation which has a bug where it removes the Content-Type header,
// causing newer ntfy servers to reject the plain-text body as invalid JSON.
// title overrides any title set in the URL, mirroring how Shoutrrr lets send
// params override the config.
func sendNtfy(ntfyURL string, title string, message string) error {
	cfg, err := parseNtfyURL(ntfyURL)
	if err != nil {
		return fmt.Errorf("failed to parse ntfy URL: %w", err)
	}

	if title != "" {
		cfg.title = title
	}

	req, err := http.NewRequest(http.MethodPost, cfg.apiURL, strings.NewReader(message))
	if err != nil {
		return fmt.Errorf("failed to create ntfy request: %w", err)
	}

	req.Header.Set("Content-Type", "text/plain")

	setHeaderIfNotEmpty(req.Header, "Title", cfg.title)
	setHeaderIfNotEmpty(req.Header, "Priority", cfg.priority)
	setHeaderIfNotEmpty(req.Header, "Tags", strings.Join(cfg.tags, ","))
	setHeaderIfNotEmpty(req.Header, "Delay", cfg.delay)
	setHeaderIfNotEmpty(req.Header, "Actions", strings.Join(cfg.actions, ";"))
	setHeaderIfNotEmpty(req.Header, "Click", cfg.click)
	setHeaderIfNotEmpty(req.Header, "Attach", cfg.attach)
	setHeaderIfNotEmpty(req.Header, "X-Icon", cfg.icon)
	setHeaderIfNotEmpty(req.Header, "Filename", cfg.filename)
	setHeaderIfNotEmpty(req.Header, "Email", cfg.email)

	if !cfg.cache {
		req.Header.Set("Cache", "no")
	}
	if !cfg.firebase {
		req.Header.Set("Firebase", "no")
	}

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
// into an ntfyConfig with the API endpoint URL, optional credentials and message
// options.
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

	// Escape raw ";" so action separators survive query parsing, as Shoutrrr does.
	parsed.RawQuery = strings.ReplaceAll(parsed.RawQuery, ";", "%3b")
	query, err := url.ParseQuery(parsed.RawQuery)
	if err != nil {
		return nil, fmt.Errorf("invalid ntfy URL query: %w", err)
	}

	cfg := &ntfyConfig{
		cache:    true,
		firebase: true,
	}

	scheme := "https"

	for key, vals := range query {
		if len(vals) == 0 {
			continue
		}
		value := vals[0]

		// Shoutrrr resolves query keys case-insensitively, so "?Scheme=http"
		// and "?scheme=http" are equivalent.
		switch strings.ToLower(key) {
		case "scheme":
			switch strings.ToLower(value) {
			case "http", "https":
				scheme = strings.ToLower(value)
			default:
				return nil, fmt.Errorf("invalid ntfy scheme %q: accepted values are http or https", value)
			}
		case "title":
			cfg.title = value
		case "priority":
			priority, ok := ntfyPriorities[strings.ToLower(value)]
			if !ok {
				return nil, fmt.Errorf("invalid ntfy priority %q: accepted values are 1-5, min, low, default, high, max or urgent", value)
			}
			cfg.priority = priority
		case "tags":
			cfg.tags = splitAndTrim(value, ",")
		case "actions":
			cfg.actions = splitAndTrim(value, ";")
		case "click":
			cfg.click = value
		case "attach":
			cfg.attach = value
		case "filename":
			cfg.filename = value
		case "delay", "at", "in":
			cfg.delay = value
		case "email":
			cfg.email = value
		case "icon":
			cfg.icon = value
		case "cache":
			cfg.cache, err = parseNtfyBool(key, value)
			if err != nil {
				return nil, err
			}
		case "firebase":
			cfg.firebase, err = parseNtfyBool(key, value)
			if err != nil {
				return nil, err
			}
		default:
			return nil, fmt.Errorf("%s is not a valid ntfy config key %v", key, ntfyQueryKeys)
		}
	}

	apiURL := &url.URL{
		Scheme: scheme,
		Host:   parsed.Host,
		Path:   "/" + topic,
	}
	cfg.apiURL = apiURL.String()

	if parsed.User != nil {
		cfg.username = parsed.User.Username()
		cfg.password, _ = parsed.User.Password()
	}

	return cfg, nil
}

// parseNtfyBool parses the boolean values Shoutrrr accepts for config keys.
func parseNtfyBool(key string, value string) (bool, error) {
	switch strings.ToLower(value) {
	case "1", "true", "yes":
		return true, nil
	case "0", "false", "no":
		return false, nil
	}
	return false, fmt.Errorf("invalid value %q for ntfy %s: accepted values are 1, true, yes or 0, false, no", value, strings.ToLower(key))
}

// splitAndTrim splits a separated list value and drops empty entries.
func splitAndTrim(value string, sep string) []string {
	var out []string
	for _, part := range strings.Split(value, sep) {
		if part = strings.TrimSpace(part); part != "" {
			out = append(out, part)
		}
	}
	return out
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
