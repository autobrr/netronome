// Copyright (c) 2024-2026, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package server

import (
	"context"
	"math"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/autobrr/netronome/internal/config"
	"github.com/autobrr/netronome/internal/database"
	"github.com/autobrr/netronome/internal/types"
)

type historyStubDB struct {
	database.Service
	page              int
	limit             int
	notificationLimit int
}

func (db *historyStubDB) GetSpeedTests(_ context.Context, _ string, page, limit int) (*types.PaginatedSpeedTests, error) {
	db.page = page
	db.limit = limit
	return &types.PaginatedSpeedTests{Page: page, Limit: limit}, nil
}

func (db *historyStubDB) GetNotificationHistory(limit int) ([]database.NotificationHistory, error) {
	db.notificationLimit = limit
	return []database.NotificationHistory{}, nil
}

func newHistoryTestServer(t *testing.T, db database.Service) *Server {
	t.Helper()
	gin.SetMode(gin.TestMode)
	cfg := config.New()
	cfg.Pagination.DefaultPage = 1
	cfg.Pagination.DefaultLimit = 20
	server := NewServer(nil, db, nil, cfg, nil, nil, nil, nil, nil)
	server.Router.GET("/speedtest/history", server.handleSpeedTestHistory)
	server.Router.GET("/speedtest/public/history", server.handlePublicSpeedTestHistory)
	server.Router.GET("/notifications/history", server.handleGetNotificationHistory)
	return server
}

func TestSpeedTestHistoryKeepsPageValuesInRange(t *testing.T) {
	tests := []struct {
		target    string
		wantPage  int
		wantLimit int
	}{
		{target: "/speedtest/history", wantPage: 1, wantLimit: 20},
		{target: "/speedtest/history?page=3&limit=5", wantPage: 3, wantLimit: 5},
		{target: "/speedtest/history?page=0&limit=0", wantPage: 1, wantLimit: 20},
		{target: "/speedtest/history?page=-4&limit=-1", wantPage: 1, wantLimit: 20},
		{target: "/speedtest/history?page=abc&limit=xyz", wantPage: 1, wantLimit: 20},
		{target: "/speedtest/history?limit=100000", wantPage: 1, wantLimit: 100},
		{target: "/speedtest/history?page=99999999999999999999", wantPage: 1, wantLimit: 20},
		{target: "/speedtest/history?page=" + strconv.Itoa(math.MaxInt) + "&limit=101", wantPage: math.MaxInt/100 + 1, wantLimit: 100},
		{target: "/speedtest/history?page=1&limit=1", wantPage: 1, wantLimit: 1},
		{target: "/speedtest/history?page=5&limit=1", wantPage: 5, wantLimit: 1},
		{target: "/speedtest/public/history?page=2&limit=1", wantPage: 2, wantLimit: 1},
		{target: "/speedtest/public/history", wantPage: 1, wantLimit: 20},
		{target: "/speedtest/public/history?limit=-1", wantPage: 1, wantLimit: 20},
		{target: "/speedtest/public/history?timeRange=all&limit=100000", wantPage: 1, wantLimit: 100},
	}

	for _, tt := range tests {
		t.Run(tt.target, func(t *testing.T) {
			db := &historyStubDB{}
			server := newHistoryTestServer(t, db)

			response := httptest.NewRecorder()
			request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, tt.target, nil)
			server.Router.ServeHTTP(response, request)

			require.Equal(t, http.StatusOK, response.Code)
			assert.Equal(t, tt.wantPage, db.page)
			assert.Equal(t, tt.wantLimit, db.limit)
			assert.GreaterOrEqual(t, (db.page-1)*db.limit, 0)
		})
	}
}

func TestNotificationHistoryKeepsLimitInRange(t *testing.T) {
	tests := []struct {
		target    string
		wantLimit int
	}{
		{target: "/notifications/history", wantLimit: 100},
		{target: "/notifications/history?limit=10", wantLimit: 10},
		{target: "/notifications/history?limit=0", wantLimit: 100},
		{target: "/notifications/history?limit=-5", wantLimit: 100},
		{target: "/notifications/history?limit=abc", wantLimit: 100},
		{target: "/notifications/history?limit=100000", wantLimit: 100},
	}

	for _, tt := range tests {
		t.Run(tt.target, func(t *testing.T) {
			db := &historyStubDB{}
			server := newHistoryTestServer(t, db)

			response := httptest.NewRecorder()
			request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, tt.target, nil)
			server.Router.ServeHTTP(response, request)

			require.Equal(t, http.StatusOK, response.Code)
			assert.Equal(t, tt.wantLimit, db.notificationLimit)
		})
	}
}
