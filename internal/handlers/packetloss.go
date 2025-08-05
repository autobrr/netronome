// Copyright (c) 2024-2025, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package handlers

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"

	"github.com/autobrr/netronome/internal/database"
	"github.com/autobrr/netronome/internal/scheduler"
	"github.com/autobrr/netronome/internal/speedtest"
	"github.com/autobrr/netronome/internal/types"
)

// PacketLossHandler handles packet loss monitoring endpoints
type PacketLossHandler struct {
	db        database.Service
	service   *speedtest.PacketLossService
	scheduler scheduler.Service
}

// NewPacketLossHandler creates a new packet loss handler
func NewPacketLossHandler(db database.Service, service *speedtest.PacketLossService, scheduler scheduler.Service) *PacketLossHandler {
	return &PacketLossHandler{
		db:        db,
		service:   service,
		scheduler: scheduler,
	}
}

// GetMonitors returns all packet loss monitors
func (h *PacketLossHandler) GetMonitors(c *gin.Context) {
	monitors, err := h.db.GetPacketLossMonitors()
	if err != nil {
		log.Error().Err(err).Msg("Failed to get packet loss monitors")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get monitors"})
		return
	}

	c.JSON(http.StatusOK, monitors)
}

// CreateMonitor creates a new packet loss monitor
func (h *PacketLossHandler) CreateMonitor(c *gin.Context) {
	var monitor types.PacketLossMonitor
	if err := c.ShouldBindJSON(&monitor); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Validate and clean input
	monitor.Host = strings.TrimSpace(monitor.Host)
	if monitor.Host == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Host is required"})
		return
	}
	if monitor.Interval == "" {
		monitor.Interval = "60s" // Default to 60 seconds
	}
	if monitor.PacketCount <= 0 {
		monitor.PacketCount = 10 // Default to 10 packets
	}
	if monitor.Threshold <= 0 {
		monitor.Threshold = 5.0 // Default to 5% packet loss threshold
	}

	// Calculate initial next_run time
	now := time.Now()
	nextRun := h.scheduler.CalculateNextRun(monitor.Interval, now)
	if !nextRun.IsZero() {
		monitor.NextRun = &nextRun
		log.Debug().
			Time("next_run", nextRun).
			Str("interval", monitor.Interval).
			Msg("Setting initial next_run for new monitor")
	}

	// Create monitor in database
	createdMonitor, err := h.db.CreatePacketLossMonitor(&monitor)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create packet loss monitor")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create monitor"})
		return
	}

	// Note: The scheduler will handle starting the monitor based on its next_run time

	c.JSON(http.StatusCreated, createdMonitor)
}

// UpdateMonitor updates an existing packet loss monitor
func (h *PacketLossHandler) UpdateMonitor(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid monitor ID"})
		return
	}

	var updateData types.PacketLossMonitor
	if err := c.ShouldBindJSON(&updateData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Validate and clean input
	updateData.Host = strings.TrimSpace(updateData.Host)

	// Get existing monitor to preserve fields not updated by the user
	existingMonitor, err := h.db.GetPacketLossMonitor(id)
	if err != nil {
		if err == database.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Monitor not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get monitor"})
		}
		return
	}

	// Log the current state for debugging
	log.Debug().
		Int64("id", existingMonitor.ID).
		Interface("lastRun", existingMonitor.LastRun).
		Interface("nextRun", existingMonitor.NextRun).
		Msg("Retrieved existing monitor state")

	// Update only the fields that should be changed by the user
	existingMonitor.Host = updateData.Host
	existingMonitor.Name = updateData.Name
	existingMonitor.PacketCount = updateData.PacketCount
	existingMonitor.Enabled = updateData.Enabled
	existingMonitor.Threshold = updateData.Threshold

	// If the interval changed, we should reset next_run so the scheduler recalculates it
	if existingMonitor.Interval != updateData.Interval {
		existingMonitor.Interval = updateData.Interval
		// Clear next_run to force recalculation by the scheduler
		existingMonitor.NextRun = nil
	}

	// Log what we're about to save
	log.Debug().
		Int64("id", existingMonitor.ID).
		Interface("lastRun", existingMonitor.LastRun).
		Interface("nextRun", existingMonitor.NextRun).
		Str("interval", existingMonitor.Interval).
		Bool("enabled", existingMonitor.Enabled).
		Msg("Updating monitor with preserved scheduling fields")

	// Update monitor in database
	if err := h.db.UpdatePacketLossMonitor(existingMonitor); err != nil {
		log.Error().Err(err).Msg("Failed to update packet loss monitor")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update monitor"})
		return
	}

	// Note: The scheduler will handle monitoring based on the enabled state and next_run time

	c.JSON(http.StatusOK, existingMonitor)
}

// DeleteMonitor deletes a packet loss monitor
func (h *PacketLossHandler) DeleteMonitor(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid monitor ID"})
		return
	}

	// Stop monitoring if active
	if err := h.service.StopMonitor(id); err != nil {
		log.Warn().Err(err).Int64("monitorID", id).Msg("Monitor might not be running")
	}

	// Delete from database
	if err := h.db.DeletePacketLossMonitor(id); err != nil {
		if err == database.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Monitor not found"})
		} else {
			log.Error().Err(err).Msg("Failed to delete packet loss monitor")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete monitor"})
		}
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

// GetMonitorStatus returns the current status of a monitor
func (h *PacketLossHandler) GetMonitorStatus(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid monitor ID"})
		return
	}

	status, err := h.service.GetMonitorStatus(id)
	if err != nil {
		log.Error().Err(err).Int64("monitorID", id).Msg("Failed to get monitor status")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get monitor status"})
		return
	}

	c.JSON(http.StatusOK, status)
}

// GetMonitorHistory returns historical results for a monitor
func (h *PacketLossHandler) GetMonitorHistory(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid monitor ID"})
		return
	}

	// Get limit from query parameter
	limitStr := c.DefaultQuery("limit", "100")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 100
	}
	if limit > 1000 {
		limit = 1000 // Cap at 1000 results
	}

	results, err := h.db.GetPacketLossResults(id, limit)
	if err != nil {
		log.Error().Err(err).Int64("monitorID", id).Msg("Failed to get packet loss results")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get monitor history"})
		return
	}

	c.JSON(http.StatusOK, results)
}

// StartMonitor manually starts monitoring for a specific monitor
func (h *PacketLossHandler) StartMonitor(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid monitor ID"})
		return
	}

	// Get the monitor first
	monitor, err := h.db.GetPacketLossMonitor(id)
	if err != nil {
		if err == database.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Monitor not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get monitor"})
		}
		return
	}

	// Enable the monitor if it's disabled
	if !monitor.Enabled {
		monitor.Enabled = true
		if err := h.db.UpdatePacketLossMonitor(monitor); err != nil {
			log.Error().Err(err).Int64("monitorID", id).Msg("Failed to enable monitor")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to enable monitor"})
			return
		}
	}

	// Update last_run and next_run times before running the test
	now := time.Now()
	monitor.LastRun = &now
	
	// Calculate next run time based on the interval
	nextRun := h.scheduler.CalculateNextRun(monitor.Interval, now)
	if !nextRun.IsZero() {
		monitor.NextRun = &nextRun
	}
	
	// Update the monitor with new schedule times
	if err := h.db.UpdatePacketLossMonitor(monitor); err != nil {
		log.Error().Err(err).Int64("monitorID", id).Msg("Failed to update monitor schedule")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update monitor schedule"})
		return
	}

	// Run a test immediately
	h.service.RunScheduledTest(monitor)

	c.JSON(http.StatusOK, gin.H{"message": "Test started successfully"})
}

// StopMonitor disables monitoring for a specific monitor
func (h *PacketLossHandler) StopMonitor(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid monitor ID"})
		return
	}

	// Get the monitor
	monitor, err := h.db.GetPacketLossMonitor(id)
	if err != nil {
		if err == database.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Monitor not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get monitor"})
		}
		return
	}

	// Disable the monitor
	monitor.Enabled = false
	if err := h.db.UpdatePacketLossMonitor(monitor); err != nil {
		log.Error().Err(err).Int64("monitorID", id).Msg("Failed to disable monitor")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Monitor disabled successfully"})
}
