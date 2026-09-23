// Copyright (c) 2024-2026, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package monitor

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/autobrr/netronome/internal/types"
)

// agentStreamSuffix is the SSE path that an agent URL carries in the database.
const agentStreamSuffix = "/events?stream=live-data"

// Timeout ceilings per endpoint. The historical export reads the full vnstat
// database, so it gets more time. The info endpoint is a best-effort version
// lookup on the path of a browser request, so it gets less.
const (
	agentTimeout       = 30 * time.Second
	agentExportTimeout = 60 * time.Second
	agentInfoTimeout   = 5 * time.Second
)

// AgentBaseURL returns the root URL of an agent from the URL that the database
// stores, which ends with the SSE stream path.
func AgentBaseURL(agentURL string) string {
	return strings.TrimRight(strings.TrimSuffix(agentURL, agentStreamSuffix), "/")
}

// AgentStreamURL returns the agent URL with the SSE stream path attached.
func AgentStreamURL(agentURL string) string {
	if strings.HasSuffix(agentURL, agentStreamSuffix) {
		return agentURL
	}
	return strings.TrimRight(agentURL, "/") + agentStreamSuffix
}

// Readable arguments for the auth parameter of get.
const (
	withAPIKey = true
	noAPIKey   = false
)

// setAgentAuth adds the API key header to a request when the agent has a key.
func setAgentAuth(req *http.Request, apiKey *string) {
	if apiKey != nil && *apiKey != "" {
		req.Header.Set("X-API-Key", *apiKey)
	}
}

// AgentClient fetches from the HTTP endpoints of one agent.
type AgentClient struct {
	baseURL string
	apiKey  *string
	http    *http.Client
}

// NewAgentClient returns a client for the given agent.
func NewAgentClient(agent *types.MonitorAgent) *AgentClient {
	return &AgentClient{
		baseURL: AgentBaseURL(agent.URL),
		apiKey:  agent.APIKey,
		// http.DefaultClient has no timeout of its own, which is correct
		// here: each method sets its own ceiling on the request context, so
		// cancellation by the caller keeps working.
		http: http.DefaultClient,
	}
}

// AgentInfo is the response of the public agent info endpoint.
type AgentInfo struct {
	Version string `json:"version"`
}

// SystemInfo returns the raw response of the agent system info endpoint.
func (c *AgentClient) SystemInfo(ctx context.Context) ([]byte, error) {
	return c.get(ctx, "/system/info", agentTimeout, withAPIKey)
}

// HardwareStats returns the raw response of the agent hardware endpoint.
func (c *AgentClient) HardwareStats(ctx context.Context) ([]byte, error) {
	return c.get(ctx, "/system/hardware", agentTimeout, withAPIKey)
}

// PeakStats returns the raw response of the agent peak statistics endpoint.
func (c *AgentClient) PeakStats(ctx context.Context) ([]byte, error) {
	return c.get(ctx, "/stats/peaks", agentTimeout, withAPIKey)
}

// Historical returns the raw vnstat export of the agent. An empty interface
// name asks the agent for its default interface.
func (c *AgentClient) Historical(ctx context.Context, iface string) ([]byte, error) {
	path := "/export/historical"
	if iface != "" {
		path += "?interface=" + url.QueryEscape(iface)
	}
	return c.get(ctx, path, agentExportTimeout, withAPIKey)
}

// Info returns the version of the agent. The endpoint is public, so the
// request sends no API key.
func (c *AgentClient) Info(ctx context.Context) (AgentInfo, error) {
	body, err := c.get(ctx, "/netronome/info", agentInfoTimeout, noAPIKey)
	if err != nil {
		return AgentInfo{}, err
	}

	var info AgentInfo
	if err := json.Unmarshal(body, &info); err != nil {
		return AgentInfo{}, fmt.Errorf("decode agent info: %w", err)
	}
	return info, nil
}

// get sends one GET request to the agent and reads the full body. The timeout
// is a ceiling on top of the context of the caller, so cancellation by the
// caller still ends the request early.
func (c *AgentClient) get(ctx context.Context, path string, timeout time.Duration, auth bool) ([]byte, error) {
	requestURL := c.baseURL + path

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request for %s: %w", path, err)
	}
	if auth {
		setAgentAuth(req, c.apiKey)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch %s: %w", path, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, &StatusError{StatusCode: resp.StatusCode, URL: requestURL}
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	return body, nil
}
