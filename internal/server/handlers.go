// Copyright (c) 2024-2025, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package server

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"

	"github.com/autobrr/netronome/internal/server/encoder"
	"github.com/autobrr/netronome/internal/types"
)

func (s *Server) handleSpeedTest(w http.ResponseWriter, r *http.Request) {
	var opts types.TestOptions
	if err := json.NewDecoder(r.Body).Decode(&opts); err != nil {
		encoder.JSON(w, http.StatusBadRequest, encoder.H{
			"error": "could not decode request body",
		})
		return
	}

	// TODO move last speedtest to db
	// Reset lastUpdate before starting new test
	s.mu.Lock()
	s.lastUpdate = &types.SpeedUpdate{}
	s.mu.Unlock()

	// Use configured timeout
	ctx, cancel := context.WithTimeout(r.Context(), time.Duration(s.config.SpeedTest.Timeout)*time.Second)
	defer cancel()

	result, err := s.speedtest.RunTest(&opts)
	if err != nil {
		encoder.JSON(w, http.StatusInternalServerError, encoder.H{
			"error": "failed to run speed test",
		})
		return
	}

	// Ensure final update is set
	s.mu.Lock()
	s.lastUpdate.IsComplete = true
	s.mu.Unlock()

	select {
	case <-ctx.Done():
		encoder.JSON(w, http.StatusInternalServerError, encoder.H{
			"error": "speed test timed out " + ctx.Err().Error(),
		})
		return
	default:
		encoder.JSON(w, http.StatusOK, result)
	}
}

func (s *Server) handleSpeedTestHistory(w http.ResponseWriter, r *http.Request) {
	timeRange := s.config.Pagination.DefaultTimeRange
	if val := chi.URLParam(r, "timeRange"); val != "" {
		timeRange = val
	}

	page := s.config.Pagination.DefaultPage
	if val := chi.URLParam(r, "page"); val != "" {
		page, _ = strconv.Atoi(val)
	}

	limit := s.config.Pagination.DefaultLimit
	if val := chi.URLParam(r, "limit"); val != "" {
		limit, _ = strconv.Atoi(val)
	}

	results, err := s.db.GetSpeedTests(r.Context(), timeRange, page, limit)
	if err != nil {
		log.Error().Err(err).
			Str("timeRange", timeRange).
			Int("page", page).
			Int("limit", limit).
			Msg("Failed to retrieve speed test history")
		encoder.JSON(w, http.StatusInternalServerError, encoder.H{
			"error": "failed to retrieve speed test history",
		})
		return
	}

	encoder.JSON(w, http.StatusOK, results)
}

func (s *Server) getServers(w http.ResponseWriter, r *http.Request) {
	servers, err := s.speedtest.GetServers(r.Context())
	if err != nil {
		encoder.JSON(w, http.StatusInternalServerError, encoder.H{
			"error": "Failed to get servers",
		})
		return
	}

	encoder.JSON(w, http.StatusOK, servers)
}

func (s *Server) handleSpeedTestStatus(w http.ResponseWriter, _ *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	log.Trace().
		Interface("lastUpdate", s.lastUpdate).
		Msg("Sending status update")

	// TODO get from db
	encoder.JSON(w, http.StatusOK, s.lastUpdate)
}

func (s *Server) handleGetSchedules(w http.ResponseWriter, r *http.Request) {
	schedules, err := s.db.GetSchedules(r.Context())
	if err != nil {
		encoder.JSON(w, http.StatusInternalServerError, encoder.H{
			"error": "Failed to get schedules",
		})
		return
	}
	encoder.JSON(w, http.StatusOK, schedules)
}

func (s *Server) handleCreateSchedule(w http.ResponseWriter, r *http.Request) {
	var schedule types.Schedule
	if err := json.NewDecoder(r.Body).Decode(&schedule); err != nil {
		log.Debug().Err(err).Msg("Failed to bind JSON for schedule creation")
		encoder.JSON(w, http.StatusBadRequest, encoder.H{
			"error": "invalid schedule data",
		})
		return
	}

	createdSchedule, err := s.db.CreateSchedule(r.Context(), schedule)
	if err != nil {
		log.Error().Err(err).
			Interface("schedule", schedule).
			Msg("Failed to create schedule")
		encoder.JSON(w, http.StatusInternalServerError, encoder.H{
			"error": "Failed to create schedule",
		})
		return
	}

	encoder.JSON(w, http.StatusCreated, createdSchedule)
}

func (s *Server) handleUpdateSchedule(w http.ResponseWriter, r *http.Request) {
	var schedule types.Schedule
	if err := json.NewDecoder(r.Body).Decode(&schedule); err != nil {
		log.Debug().Err(err).Msg("Failed to bind JSON for schedule creation")
		encoder.JSON(w, http.StatusBadRequest, encoder.H{
			"error": "invalid schedule data",
		})
		return
	}

	err := s.db.UpdateSchedule(r.Context(), schedule)
	if err != nil {
		encoder.JSON(w, http.StatusInternalServerError, encoder.H{
			"error": "failed to update schedule",
		})
		return
	}

	encoder.JSON(w, http.StatusOK, encoder.H{"message": "Schedule updated successfully"})
}

func (s *Server) handleDeleteSchedule(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "scheduleID"))
	if err != nil {
		encoder.JSON(w, http.StatusInternalServerError, encoder.H{
			"error": "invalid schedule ID",
		})
		return
	}

	err = s.db.DeleteSchedule(r.Context(), int64(id))
	if err != nil {
		encoder.JSON(w, http.StatusInternalServerError, encoder.H{
			"error": "failed to delete schedule",
		})
		return
	}

	encoder.JSON(w, http.StatusOK, encoder.H{"message": "Schedule deleted successfully"})
}
