// Copyright (c) 2024-2026, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package handlers

import (
	"context"
	"encoding/json/v2"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/autobrr/netronome/internal/config"
	"github.com/autobrr/netronome/internal/database"
	"github.com/autobrr/netronome/internal/types"
)

func useLocalZone(t *testing.T) {
	t.Helper()
	old := time.Local
	time.Local = time.FixedZone("UTC+2", 2*60*60)
	t.Cleanup(func() { time.Local = old })
}

type monitorStubDB struct {
	database.Service
	agents      []*types.MonitorAgent
	versions    map[int64]string
	versionsErr error
	saved       map[int64]*types.MonitorSystemInfo
}

func (db monitorStubDB) GetMonitorAgents(context.Context, bool) ([]*types.MonitorAgent, error) {
	return db.agents, nil
}

func (db monitorStubDB) GetMonitorAgentVersions(context.Context) (map[int64]string, error) {
	return db.versions, db.versionsErr
}

func (db monitorStubDB) GetMonitorAgent(_ context.Context, id int64) (*types.MonitorAgent, error) {
	for _, agent := range db.agents {
		if agent.ID == id {
			return agent, nil
		}
	}
	return nil, database.ErrNotFound
}

func (db monitorStubDB) GetMonitorSystemInfo(_ context.Context, id int64) (*types.MonitorSystemInfo, error) {
	if info, ok := db.saved[id]; ok {
		return info, nil
	}
	return nil, database.ErrNotFound
}

func (db monitorStubDB) GetMonitorInterfaces(context.Context, int64) ([]types.MonitorInterface, error) {
	return nil, nil
}

func (db monitorStubDB) GetMonitorResourceStats(context.Context, int64, int) ([]types.MonitorResourceStats, error) {
	return nil, nil
}

func newMonitorTestRouter(db database.Service, serverVersion string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	handler := NewMonitorHandler(db, nil, &config.MonitorConfig{}, serverVersion)
	router := gin.New()
	router.GET("/api/monitor/agents", handler.GetAgents)
	router.GET("/api/monitor/agents/:id/system", handler.GetAgentSystemInfo)
	router.GET("/api/monitor/agents/:id/hardware", handler.GetAgentHardwareStats)
	return router
}

func serveMonitor(t *testing.T, router *gin.Engine, target string) *httptest.ResponseRecorder {
	t.Helper()
	response := httptest.NewRecorder()
	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, target, nil)
	router.ServeHTTP(response, request)
	return response
}

func TestGetAgentsReportsSavedVersionAndOutdated(t *testing.T) {
	db := monitorStubDB{
		agents: []*types.MonitorAgent{
			{ID: 1, Name: "older", URL: "http://127.0.0.1:8200"},
			{ID: 2, Name: "equal", URL: "http://127.0.0.1:8201"},
			{ID: 3, Name: "unknown", URL: "http://127.0.0.1:8202"},
		},
		versions: map[int64]string{1: "0.9.0", 2: "v0.10.0"},
	}

	response := serveMonitor(t, newMonitorTestRouter(db, "0.10.0"), "/api/monitor/agents")

	require.Equal(t, http.StatusOK, response.Code)
	var agents []struct {
		ID           int64   `json:"id"`
		AgentVersion *string `json:"agentVersion"`
		Outdated     *bool   `json:"outdated"`
	}
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &agents))
	require.Len(t, agents, 3)

	older, equal, unknown := agents[0], agents[1], agents[2]
	require.NotNil(t, older.AgentVersion)
	assert.Equal(t, "0.9.0", *older.AgentVersion)
	require.NotNil(t, older.Outdated)
	assert.True(t, *older.Outdated)

	require.NotNil(t, equal.AgentVersion)
	assert.Equal(t, "v0.10.0", *equal.AgentVersion)
	require.NotNil(t, equal.Outdated)
	assert.False(t, *equal.Outdated)

	assert.Nil(t, unknown.AgentVersion)
	require.NotNil(t, unknown.Outdated)
	assert.False(t, *unknown.Outdated)
}

func TestGetAgentsListsAgentsWhenVersionsFail(t *testing.T) {
	db := monitorStubDB{
		agents:      []*types.MonitorAgent{{ID: 1, Name: "agent", URL: "http://127.0.0.1:8200"}},
		versionsErr: errors.New("versions unavailable"),
	}

	response := serveMonitor(t, newMonitorTestRouter(db, "0.10.0"), "/api/monitor/agents")

	require.Equal(t, http.StatusOK, response.Code)
	assert.JSONEq(t, `[{"id":1,"name":"agent","url":"http://127.0.0.1:8200","enabled":false,"isTailscale":false,"createdAt":"0001-01-01T00:00:00Z","updatedAt":"0001-01-01T00:00:00Z","agentVersion":null,"outdated":false}]`, response.Body.String())
}

func TestGetAgentsKeepsAPIKeyMasked(t *testing.T) {
	agent := &types.MonitorAgent{ID: 1, Name: "keyed", URL: "http://127.0.0.1:8200"}
	agent.APIKey = new("unmasked-placeholder")
	db := monitorStubDB{agents: []*types.MonitorAgent{agent}}

	response := serveMonitor(t, newMonitorTestRouter(db, "0.10.0"), "/api/monitor/agents")

	require.Equal(t, http.StatusOK, response.Code)
	assert.Contains(t, response.Body.String(), `"apiKey":"configured"`)
	assert.NotContains(t, response.Body.String(), "unmasked-placeholder")
}

func TestGetAgentSystemInfoReportsOutdatedForLiveAgent(t *testing.T) {
	agent := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/system/info":
			_, _ = io.WriteString(w, `{"hostname":"host","kernel":"6.1"}`)
		case "/netronome/info":
			_, _ = io.WriteString(w, `{"version":"0.9.0"}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer agent.Close()

	db := monitorStubDB{agents: []*types.MonitorAgent{{ID: 1, Name: "live", URL: agent.URL}}}

	response := serveMonitor(t, newMonitorTestRouter(db, "0.10.0"), "/api/monitor/agents/1/system")

	require.Equal(t, http.StatusOK, response.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body))
	assert.Equal(t, "0.9.0", body["agent_version"])
	assert.Equal(t, true, body["outdated"])
}

func TestGetAgentSystemInfoReportsOutdatedForSavedData(t *testing.T) {
	offline := httptest.NewServer(http.NotFoundHandler())
	offlineURL := offline.URL
	offline.Close()

	tests := []struct {
		name         string
		agentVersion *string
		want         bool
	}{
		{name: "older saved version", agentVersion: new("0.9.0"), want: true},
		{name: "equal saved version", agentVersion: new("0.10.0"), want: false},
		{name: "no saved version", agentVersion: nil, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := monitorStubDB{
				agents: []*types.MonitorAgent{{ID: 1, Name: "offline", URL: offlineURL}},
				saved: map[int64]*types.MonitorSystemInfo{
					1: {AgentID: 1, Hostname: "host", AgentVersion: tt.agentVersion, UpdatedAt: time.Now()},
				},
			}

			response := serveMonitor(t, newMonitorTestRouter(db, "0.10.0"), "/api/monitor/agents/1/system")

			require.Equal(t, http.StatusOK, response.Code)
			var body map[string]any
			require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body))
			assert.Equal(t, true, body["from_cache"])
			assert.Equal(t, tt.want, body["outdated"])
		})
	}
}

func TestGetAgentHardwareStatsWithoutSavedSampleAnswersInUTC(t *testing.T) {
	useLocalZone(t)

	offline := httptest.NewServer(http.NotFoundHandler())
	offlineURL := offline.URL
	offline.Close()
	db := monitorStubDB{agents: []*types.MonitorAgent{{ID: 1, Name: "offline", URL: offlineURL}}}

	response := serveMonitor(t, newMonitorTestRouter(db, "0.10.0"), "/api/monitor/agents/1/hardware")

	require.Equal(t, http.StatusOK, response.Code)
	var body struct {
		UpdatedAt string `json:"updated_at"`
		FromCache bool   `json:"from_cache"`
	}
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body))
	assert.True(t, body.FromCache)
	assert.True(t, strings.HasSuffix(body.UpdatedAt, "Z"), "updated_at %q", body.UpdatedAt)
}
