// Copyright (c) 2024-2025, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package server

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"

	"github.com/autobrr/netronome/internal/types"
)

func (s *Server) handleSpeedTest(c *gin.Context) {
	var opts types.TestOptions
	if err := c.ShouldBindJSON(&opts); err != nil {
		c.Status(http.StatusBadRequest)
		_ = c.Error(fmt.Errorf("invalid request body: %w", err))
		return
	}

	// Reset lastUpdate before starting new test
	s.mu.Lock()
	s.lastUpdate = &types.SpeedUpdate{}
	s.mu.Unlock()

	timeout := time.Duration(s.config.SpeedTest.Timeout) * time.Second
	if opts.UseLibrespeed {
		timeout = time.Duration(s.config.SpeedTest.Librespeed.Timeout) * time.Second
	}

	// Use configured timeout
	ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
	defer cancel()

	result, err := s.speedtest.RunTest(ctx, &opts)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		_ = c.Error(fmt.Errorf("failed to run speed test: %w", err))
		return
	}

	// Ensure final update is set
	s.mu.Lock()
	s.lastUpdate.IsComplete = true
	s.mu.Unlock()

	select {
	case <-ctx.Done():
		c.Status(http.StatusGatewayTimeout)
		_ = c.Error(fmt.Errorf("speed test timeout: %w", ctx.Err()))
		return
	default:
		c.JSON(http.StatusOK, result)
	}
}

func (s *Server) handleSpeedTestHistory(c *gin.Context) {
	timeRange := c.DefaultQuery("timeRange", s.config.Pagination.DefaultTimeRange)
	page, _ := strconv.Atoi(c.DefaultQuery("page", strconv.Itoa(s.config.Pagination.DefaultPage)))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", strconv.Itoa(s.config.Pagination.DefaultLimit)))

	results, err := s.db.GetSpeedTests(c.Request.Context(), timeRange, page, limit)
	if err != nil {
		log.Error().Err(err).
			Str("timeRange", timeRange).
			Int("page", page).
			Int("limit", limit).
			Msg("Failed to retrieve speed test history")
		c.Status(http.StatusInternalServerError)
		_ = c.Error(fmt.Errorf("failed to retrieve speed test history: %w", err))
		return
	}

	c.JSON(http.StatusOK, results)
}

func (s *Server) handlePublicSpeedTestHistory(c *gin.Context) {
	timeRange := c.DefaultQuery("timeRange", s.config.Pagination.DefaultTimeRange)
	page, _ := strconv.Atoi(c.DefaultQuery("page", strconv.Itoa(s.config.Pagination.DefaultPage)))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", strconv.Itoa(s.config.Pagination.DefaultLimit)))

	results, err := s.db.GetSpeedTests(c.Request.Context(), timeRange, page, limit)
	if err != nil {
		log.Error().Err(err).
			Str("timeRange", timeRange).
			Int("page", page).
			Int("limit", limit).
			Msg("Failed to retrieve speed test history")
		c.Status(http.StatusInternalServerError)
		_ = c.Error(fmt.Errorf("failed to retrieve speed test history: %w", err))
		return
	}

	c.JSON(http.StatusOK, results)
}

func (s *Server) handleGetServers(c *gin.Context) {
	testType := c.DefaultQuery("testType", "speedtest")

	servers, err := s.speedtest.GetServers(testType)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		_ = c.Error(fmt.Errorf("failed to get servers: %w", err))
		return
	}
	c.JSON(http.StatusOK, servers)
}

func (s *Server) handleSpeedTestStatus(c *gin.Context) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	log.Trace().
		Interface("lastUpdate", s.lastUpdate).
		Msg("Sending status update")

	c.JSON(http.StatusOK, s.lastUpdate)
}

func (s *Server) handleGetSchedules(c *gin.Context) {
	schedules, err := s.db.GetSchedules(c.Request.Context())
	if err != nil {
		c.Status(http.StatusInternalServerError)
		_ = c.Error(fmt.Errorf("failed to get schedules: %w", err))
		return
	}
	c.JSON(http.StatusOK, schedules)
}

func (s *Server) handleCreateSchedule(c *gin.Context) {
	var schedule types.Schedule
	if err := c.ShouldBindJSON(&schedule); err != nil {
		log.Debug().Err(err).Msg("Failed to bind JSON for schedule creation")
		c.Status(http.StatusBadRequest)
		_ = c.Error(fmt.Errorf("invalid schedule data: %w", err))
		return
	}

	createdSchedule, err := s.db.CreateSchedule(c.Request.Context(), schedule)
	if err != nil {
		log.Error().Err(err).
			Interface("schedule", schedule).
			Msg("Failed to create schedule")
		c.Status(http.StatusInternalServerError)
		_ = c.Error(fmt.Errorf("failed to create schedule: %w", err))
		return
	}

	c.JSON(http.StatusCreated, createdSchedule)
}

func (s *Server) handleUpdateSchedule(c *gin.Context) {
	var schedule types.Schedule
	if err := c.ShouldBindJSON(&schedule); err != nil {
		c.Status(http.StatusBadRequest)
		_ = c.Error(fmt.Errorf("invalid schedule data: %w", err))
		return
	}

	err := s.db.UpdateSchedule(c.Request.Context(), schedule)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		_ = c.Error(fmt.Errorf("failed to update schedule: %w", err))
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Schedule updated successfully"})
}

func (s *Server) handleDeleteSchedule(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.Status(http.StatusBadRequest)
		_ = c.Error(fmt.Errorf("invalid schedule ID: %w", err))
		return
	}

	err = s.db.DeleteSchedule(c.Request.Context(), int64(id))
	if err != nil {
		c.Status(http.StatusInternalServerError)
		_ = c.Error(fmt.Errorf("failed to delete schedule: %w", err))
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Schedule deleted successfully"})
}

func (s *Server) handleTraceroute(c *gin.Context) {
	host := c.Query("host")
	if host == "" {
		c.Status(http.StatusBadRequest)
		_ = c.Error(fmt.Errorf("host parameter is required"))
		return
	}

	// Reset lastTracerouteUpdate before starting new traceroute
	s.mu.Lock()
	s.lastTracerouteUpdate = &types.TracerouteUpdate{
		Type: "traceroute",
		Host: host,
	}
	s.mu.Unlock()

	// Create a timeout context for the traceroute
	timeout := 90 * time.Second // Extended timeout for traceroute
	ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
	defer cancel()

	result, err := s.speedtest.RunTraceroute(ctx, host)
	if err != nil {
		// Update status with error
		s.mu.Lock()
		s.lastTracerouteUpdate.IsComplete = true
		s.mu.Unlock()
		
		c.Status(http.StatusInternalServerError)
		_ = c.Error(fmt.Errorf("failed to run traceroute: %w", err))
		return
	}

	// Ensure final update is set
	s.mu.Lock()
	s.lastTracerouteUpdate.IsComplete = true
	s.lastTracerouteUpdate.Progress = 100.0
	s.lastTracerouteUpdate.TotalHops = result.TotalHops
	s.mu.Unlock()

	c.JSON(http.StatusOK, result)
}

func (s *Server) handleTracerouteStatus(c *gin.Context) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	log.Trace().
		Interface("lastTracerouteUpdate", s.lastTracerouteUpdate).
		Msg("Sending traceroute status update")

	if s.lastTracerouteUpdate == nil {
		c.JSON(http.StatusOK, &types.TracerouteUpdate{})
		return
	}

	c.JSON(http.StatusOK, s.lastTracerouteUpdate)
}
