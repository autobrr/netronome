// Copyright (c) 2024-2026, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package monitor

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type agentEndpoint int

const (
	endpointSystemInfo agentEndpoint = iota
	endpointHardwareStats
)

type endpointSupport struct {
	known     bool
	supported bool
}

type agentCapabilities struct {
	systemInfo    endpointSupport
	hardwareStats endpointSupport
}

type httpStatusError struct {
	StatusCode int
	URL        string
}

func (e *httpStatusError) Error() string {
	if e == nil {
		return "<nil>"
	}
	return fmt.Sprintf("unexpected status code: %d (url=%s)", e.StatusCode, e.URL)
}

func detectAgentCapabilities(ctx context.Context, baseURL string) (agentCapabilities, error) {
	rootURL := strings.TrimRight(baseURL, "/") + "/"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rootURL, nil)
	if err != nil {
		return agentCapabilities{}, fmt.Errorf("create root request: %w", err)
	}

	httpClient := &http.Client{Timeout: 5 * time.Second}
	resp, err := httpClient.Do(req)
	if err != nil {
		return agentCapabilities{}, fmt.Errorf("fetch root: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return agentCapabilities{}, &httpStatusError{StatusCode: resp.StatusCode, URL: rootURL}
	}

	var root struct {
		Endpoints map[string]any `json:"endpoints"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&root); err != nil {
		return agentCapabilities{}, fmt.Errorf("decode root json: %w", err)
	}
	if root.Endpoints == nil {
		return agentCapabilities{}, fmt.Errorf("root response missing endpoints")
	}

	_, hasSystem := root.Endpoints["system"]
	_, hasHardware := root.Endpoints["hardware"]

	return agentCapabilities{
		systemInfo:    endpointSupport{known: true, supported: hasSystem},
		hardwareStats: endpointSupport{known: true, supported: hasHardware},
	}, nil
}
