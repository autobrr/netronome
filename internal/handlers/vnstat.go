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

	"github.com/autobrr/netronome/internal/config"
	"github.com/autobrr/netronome/internal/database"
	"github.com/autobrr/netronome/internal/types"
	"github.com/autobrr/netronome/internal/vnstat"
)

// VnstatHandler handles vnstat monitoring endpoints
type VnstatHandler struct {
	db      database.Service
	service *vnstat.Service
	config  *config.VnstatConfig
}

// NewVnstatHandler creates a new vnstat handler
func NewVnstatHandler(db database.Service, service *vnstat.Service, cfg *config.VnstatConfig) *VnstatHandler {
	return &VnstatHandler{
		db:      db,
		service: service,
		config:  cfg,
	}
}

// GetAgents returns all vnstat agents
func (h *VnstatHandler) GetAgents(c *gin.Context) {
	agents, err := h.db.GetVnstatAgents(c.Request.Context(), false)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get vnstat agents")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get agents"})
		return
	}

	c.JSON(http.StatusOK, agents)
}

// GetAgent returns a specific vnstat agent
func (h *VnstatHandler) GetAgent(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid agent ID"})
		return
	}

	agent, err := h.db.GetVnstatAgent(c.Request.Context(), id)
	if err != nil {
		if err == database.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Agent not found"})
			return
		}
		log.Error().Err(err).Msg("Failed to get vnstat agent")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get agent"})
		return
	}

	c.JSON(http.StatusOK, agent)
}

// CreateAgent creates a new vnstat agent
func (h *VnstatHandler) CreateAgent(c *gin.Context) {
	var agent types.VnstatAgent
	if err := c.ShouldBindJSON(&agent); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Validate input
	agent.Name = strings.TrimSpace(agent.Name)
	agent.URL = strings.TrimSpace(agent.URL)

	if agent.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Name is required"})
		return
	}
	if agent.URL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "URL is required"})
		return
	}

	// Ensure URL has the correct SSE endpoint
	if !strings.HasSuffix(agent.URL, "/events?stream=live-data") {
		if strings.HasSuffix(agent.URL, "/") {
			agent.URL = agent.URL + "events?stream=live-data"
		} else {
			agent.URL = agent.URL + "/events?stream=live-data"
		}
	}

	// Set default retention if not provided
	if agent.RetentionDays <= 0 {
		agent.RetentionDays = h.config.DefaultRetentionDays
	}

	// Create agent in database
	createdAgent, err := h.db.CreateVnstatAgent(c.Request.Context(), &agent)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create vnstat agent")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create agent"})
		return
	}

	// Start monitoring if enabled
	if createdAgent.Enabled {
		if err := h.service.StartAgent(createdAgent.ID); err != nil {
			log.Error().Err(err).Int64("agent_id", createdAgent.ID).Msg("Failed to start agent monitoring")
			// Don't fail the request, agent was created successfully
		}
	}

	c.JSON(http.StatusCreated, createdAgent)
}

// UpdateAgent updates an existing vnstat agent
func (h *VnstatHandler) UpdateAgent(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid agent ID"})
		return
	}

	var agent types.VnstatAgent
	if err := c.ShouldBindJSON(&agent); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Set the ID from URL
	agent.ID = id

	// Validate input
	agent.Name = strings.TrimSpace(agent.Name)
	agent.URL = strings.TrimSpace(agent.URL)

	if agent.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Name is required"})
		return
	}
	if agent.URL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "URL is required"})
		return
	}

	// Ensure URL has the correct SSE endpoint
	if !strings.HasSuffix(agent.URL, "/events?stream=live-data") {
		if strings.HasSuffix(agent.URL, "/") {
			agent.URL = agent.URL + "events?stream=live-data"
		} else {
			agent.URL = agent.URL + "/events?stream=live-data"
		}
	}

	// Update agent in database
	if err := h.db.UpdateVnstatAgent(c.Request.Context(), &agent); err != nil {
		log.Error().Err(err).Msg("Failed to update vnstat agent")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update agent"})
		return
	}

	// Restart monitoring with new settings
	h.service.StopAgent(id)
	if agent.Enabled {
		if err := h.service.StartAgent(id); err != nil {
			log.Error().Err(err).Int64("agent_id", id).Msg("Failed to restart agent monitoring")
			// Don't fail the request, agent was updated successfully
		}
	}

	c.JSON(http.StatusOK, agent)
}

// DeleteAgent deletes a vnstat agent
func (h *VnstatHandler) DeleteAgent(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid agent ID"})
		return
	}

	// Stop monitoring first
	h.service.StopAgent(id)

	// Delete from database
	if err := h.db.DeleteVnstatAgent(c.Request.Context(), id); err != nil {
		log.Error().Err(err).Msg("Failed to delete vnstat agent")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete agent"})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

// GetAgentStatus returns the connection status of an agent
func (h *VnstatHandler) GetAgentStatus(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid agent ID"})
		return
	}

	connected, liveData := h.service.GetAgentStatus(id)

	status := gin.H{
		"connected": connected,
	}

	if liveData != nil {
		status["liveData"] = liveData
	}

	log.Trace().
		Int64("agent_id", id).
		Bool("connected", connected).
		Bool("has_live_data", liveData != nil).
		Msg("Agent status requested")

	c.JSON(http.StatusOK, status)
}

// GetAgentBandwidth returns bandwidth history for an agent
func (h *VnstatHandler) GetAgentBandwidth(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid agent ID"})
		return
	}

	// Parse query parameters
	limitStr := c.DefaultQuery("limit", "1000")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 1000
	}

	// Parse time range
	startStr := c.Query("start")
	endStr := c.Query("end")

	var startTime, endTime time.Time

	if startStr != "" {
		startTime, err = time.Parse(time.RFC3339, startStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid start time format"})
			return
		}
	} else {
		// Default to last 24 hours
		startTime = time.Now().Add(-24 * time.Hour)
	}

	if endStr != "" {
		endTime, err = time.Parse(time.RFC3339, endStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid end time format"})
			return
		}
	} else {
		endTime = time.Now()
	}

	// Get bandwidth history
	history, err := h.db.GetVnstatBandwidthHistory(c.Request.Context(), id, startTime, endTime, limit)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get vnstat bandwidth history")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get bandwidth history"})
		return
	}

	c.JSON(http.StatusOK, history)
}

// StartAgent manually starts monitoring for an agent
func (h *VnstatHandler) StartAgent(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid agent ID"})
		return
	}

	if err := h.service.StartAgent(id); err != nil {
		log.Error().Err(err).Int64("agent_id", id).Msg("Failed to start agent")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Agent started successfully"})
}

// StopAgent manually stops monitoring for an agent
func (h *VnstatHandler) StopAgent(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid agent ID"})
		return
	}

	h.service.StopAgent(id)
	c.JSON(http.StatusOK, gin.H{"message": "Agent stopped successfully"})
}

// GetAgentUsage returns bandwidth usage statistics for an agent
func (h *VnstatHandler) GetAgentUsage(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid agent ID"})
		return
	}

	usage, err := h.db.GetVnstatBandwidthUsage(c.Request.Context(), id)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get vnstat bandwidth usage")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get bandwidth usage"})
		return
	}

	c.JSON(http.StatusOK, usage)
}

// ImportHistoricalData imports historical vnstat data from an agent
func (h *VnstatHandler) ImportHistoricalData(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid agent ID"})
		return
	}

	// Get the agent to retrieve its URL
	agent, err := h.db.GetVnstatAgent(c.Request.Context(), id)
	if err != nil {
		if err == database.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Agent not found"})
			return
		}
		log.Error().Err(err).Msg("Failed to get vnstat agent")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get agent"})
		return
	}

	// Check if import is already in progress
	importStatus := h.service.GetImportStatus(id)
	if importStatus != nil && importStatus.InProgress {
		c.JSON(http.StatusConflict, gin.H{
			"error":  "Import already in progress",
			"status": importStatus,
		})
		return
	}

	// Start the import in background
	go h.service.ImportHistoricalData(agent)

	c.JSON(http.StatusAccepted, gin.H{
		"message": "Historical data import started",
		"agentId": id,
	})
}

// GetImportStatus returns the status of a historical data import
func (h *VnstatHandler) GetImportStatus(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid agent ID"})
		return
	}

	status := h.service.GetImportStatus(id)
	if status == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "No import status found"})
		return
	}

	c.JSON(http.StatusOK, status)
}
