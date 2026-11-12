// Copyright (c) 2024-2026, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package handlers

import (
	"encoding/json/v2"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/autobrr/netronome/internal/database"
	"github.com/autobrr/netronome/internal/scheduler"
	"github.com/autobrr/netronome/internal/types"
)

type packetLossStubDB struct {
	database.Service
	existing *types.PacketLossMonitor
}

func (db packetLossStubDB) CreatePacketLossMonitor(monitor *types.PacketLossMonitor) (*types.PacketLossMonitor, error) {
	monitor.ID = 1
	return monitor, nil
}

func (db packetLossStubDB) GetPacketLossMonitor(int64) (*types.PacketLossMonitor, error) {
	return db.existing, nil
}

func (db packetLossStubDB) UpdatePacketLossMonitor(*types.PacketLossMonitor) error {
	return nil
}

func TestPacketLossMonitorAnswersHaveNextRunInUTC(t *testing.T) {
	useLocalZone(t)

	gin.SetMode(gin.TestMode)
	db := packetLossStubDB{existing: &types.PacketLossMonitor{ID: 1, Host: "127.0.0.1", Interval: "1h", PacketCount: 10, Enabled: true}}
	handler := NewPacketLossHandler(db, nil, scheduler.New(db, nil, nil, nil, nil))
	router := gin.New()
	router.POST("/api/packetloss/monitors", handler.CreateMonitor)
	router.PUT("/api/packetloss/monitors/:id", handler.UpdateMonitor)

	tests := []struct {
		method, target string
		wantStatus     int
	}{
		{http.MethodPost, "/api/packetloss/monitors", http.StatusCreated},
		{http.MethodPut, "/api/packetloss/monitors/1", http.StatusOK},
	}
	for _, tt := range tests {
		t.Run(tt.method, func(t *testing.T) {
			body := strings.NewReader(`{"host":"127.0.0.1","interval":"60s","packetCount":10,"enabled":true,"threshold":5}`)
			request := httptest.NewRequestWithContext(t.Context(), tt.method, tt.target, body)
			request.Header.Set("Content-Type", "application/json")
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)

			require.Equal(t, tt.wantStatus, response.Code, response.Body.String())
			var monitor struct {
				NextRun string `json:"nextRun"`
			}
			require.NoError(t, json.Unmarshal(response.Body.Bytes(), &monitor))
			assert.True(t, strings.HasSuffix(monitor.NextRun, "Z"), "nextRun %q", monitor.NextRun)
		})
	}
}
