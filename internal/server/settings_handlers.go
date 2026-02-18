// Copyright (c) 2024-2026, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package server

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"

	"github.com/autobrr/netronome/internal/database"
)

const (
	dashboardRecentRowsSettingKey = "dashboard_recent_speedtests_rows"
	dashboardRecentRowsDefault    = 20
)

var allowedDashboardRecentRows = map[int]struct{}{
	10:  {},
	20:  {},
	50:  {},
	100: {},
}

type dashboardSettingsResponse struct {
	RecentSpeedtestsRows int `json:"recentSpeedtestsRows"`
}

type updateDashboardSettingsRequest struct {
	RecentSpeedtestsRows int `json:"recentSpeedtestsRows"`
}

func (s *Server) handleGetDashboardSettings(c *gin.Context) {
	value, err := s.db.GetAppSetting(c.Request.Context(), dashboardRecentRowsSettingKey)
	if err != nil {
		if err == database.ErrNotFound {
			c.JSON(http.StatusOK, dashboardSettingsResponse{RecentSpeedtestsRows: dashboardRecentRowsDefault})
			return
		}

		log.Error().Err(err).Msg("Failed to get dashboard settings")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get dashboard settings"})
		return
	}

	rows, err := strconv.Atoi(value)
	if err != nil {
		log.Warn().Err(err).Str("value", value).Msg("Invalid persisted dashboard row setting, using default")
		c.JSON(http.StatusOK, dashboardSettingsResponse{RecentSpeedtestsRows: dashboardRecentRowsDefault})
		return
	}

	if !isAllowedDashboardRecentRows(rows) {
		log.Warn().Int("rows", rows).Msg("Out-of-range persisted dashboard row setting, using default")
		c.JSON(http.StatusOK, dashboardSettingsResponse{RecentSpeedtestsRows: dashboardRecentRowsDefault})
		return
	}

	c.JSON(http.StatusOK, dashboardSettingsResponse{RecentSpeedtestsRows: rows})
}

func (s *Server) handleUpdateDashboardSettings(c *gin.Context) {
	var req updateDashboardSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	if !isAllowedDashboardRecentRows(req.RecentSpeedtestsRows) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "recentSpeedtestsRows must be one of: 10, 20, 50, 100"})
		return
	}

	if err := s.db.SetAppSetting(c.Request.Context(), dashboardRecentRowsSettingKey, strconv.Itoa(req.RecentSpeedtestsRows)); err != nil {
		log.Error().Err(err).Msg("Failed to update dashboard settings")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update dashboard settings"})
		return
	}

	c.JSON(http.StatusOK, dashboardSettingsResponse{RecentSpeedtestsRows: req.RecentSpeedtestsRows})
}

func isAllowedDashboardRecentRows(rows int) bool {
	_, ok := allowedDashboardRecentRows[rows]
	return ok
}
