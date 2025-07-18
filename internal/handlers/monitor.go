// Copyright (c) 2024-2025, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"

	"github.com/autobrr/netronome/internal/config"
	"github.com/autobrr/netronome/internal/database"
	"github.com/autobrr/netronome/internal/types"
	"github.com/autobrr/netronome/internal/monitor"
)

// MonitorHandler handles monitoring endpoints
type MonitorHandler struct {
	db      database.Service
	service *monitor.Service
	config  *config.MonitorConfig
}

// NewMonitorHandler creates a new monitor handler
func NewMonitorHandler(db database.Service, service *monitor.Service, cfg *config.MonitorConfig) *MonitorHandler {
	return &MonitorHandler{
		db:      db,
		service: service,
		config:  cfg,
	}
}

// GetAgents returns all monitoring agents
func (h *MonitorHandler) GetAgents(c *gin.Context) {
	agents, err := h.db.GetMonitorAgents(c.Request.Context(), false)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get monitor agents")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get agents"})
		return
	}

	// Don't expose API keys to frontend
	for _, agent := range agents {
		if agent.APIKey != nil && *agent.APIKey != "" {
			masked := "configured"
			agent.APIKey = &masked
		}
	}

	c.JSON(http.StatusOK, agents)
}

// GetAgent returns a specific monitor agent
func (h *MonitorHandler) GetAgent(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid agent ID"})
		return
	}

	agent, err := h.db.GetMonitorAgent(c.Request.Context(), id)
	if err != nil {
		if err == database.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Agent not found"})
			return
		}
		log.Error().Err(err).Msg("Failed to get monitor agent")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get agent"})
		return
	}

	// Don't expose API key to frontend
	if agent.APIKey != nil && *agent.APIKey != "" {
		masked := "configured"
		agent.APIKey = &masked
	}

	c.JSON(http.StatusOK, agent)
}

// CreateAgent creates a new monitor agent
func (h *MonitorHandler) CreateAgent(c *gin.Context) {
	var agent types.MonitorAgent
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

	// Create agent in database
	createdAgent, err := h.db.CreateMonitorAgent(c.Request.Context(), &agent)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create monitor agent")
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

	// Don't expose API key to frontend
	if createdAgent.APIKey != nil && *createdAgent.APIKey != "" {
		masked := "configured"
		createdAgent.APIKey = &masked
	}

	c.JSON(http.StatusCreated, createdAgent)
}

// UpdateAgent updates an existing monitor agent
func (h *MonitorHandler) UpdateAgent(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid agent ID"})
		return
	}

	var agent types.MonitorAgent
	if err := c.ShouldBindJSON(&agent); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// If API key is "configured", fetch the existing one
	if agent.APIKey != nil && *agent.APIKey == "configured" {
		existingAgent, err := h.db.GetMonitorAgent(c.Request.Context(), id)
		if err == nil && existingAgent.APIKey != nil {
			agent.APIKey = existingAgent.APIKey
		}
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
	if err := h.db.UpdateMonitorAgent(c.Request.Context(), &agent); err != nil {
		log.Error().Err(err).Msg("Failed to update monitor agent")
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

	// Don't expose API key to frontend
	if agent.APIKey != nil && *agent.APIKey != "" {
		masked := "configured"
		agent.APIKey = &masked
	}

	c.JSON(http.StatusOK, agent)
}

// DeleteAgent deletes a monitor agent
func (h *MonitorHandler) DeleteAgent(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid agent ID"})
		return
	}

	// Stop monitoring first
	h.service.StopAgent(id)

	// Delete from database
	if err := h.db.DeleteMonitorAgent(c.Request.Context(), id); err != nil {
		log.Error().Err(err).Msg("Failed to delete monitor agent")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete agent"})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

// GetAgentStatus returns the connection status of an agent
func (h *MonitorHandler) GetAgentStatus(c *gin.Context) {
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

// StartAgent manually starts monitoring for an agent
func (h *MonitorHandler) StartAgent(c *gin.Context) {
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
func (h *MonitorHandler) StopAgent(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid agent ID"})
		return
	}

	h.service.StopAgent(id)
	c.JSON(http.StatusOK, gin.H{"message": "Agent stopped successfully"})
}

// GetAgentNativeVnstat returns the native bandwidth monitor JSON output from an agent for validation
func (h *MonitorHandler) GetAgentNativeVnstat(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid agent ID"})
		return
	}

	// Get the agent to retrieve its URL
	agent, err := h.db.GetMonitorAgent(c.Request.Context(), id)
	if err != nil {
		if err == database.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Agent not found"})
			return
		}
		log.Error().Err(err).Msg("Failed to get monitor agent")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get agent"})
		return
	}

	// Build the historical export URL from the agent's base URL
	baseURL := strings.TrimSuffix(agent.URL, "/events?stream=live-data")
	exportURL := baseURL + "/export/historical"

	// Get the interface parameter if provided
	iface := c.Query("interface")
	if iface != "" {
		exportURL += "?interface=" + iface
	}

	// Create HTTP request with API key if configured
	req, err := http.NewRequest("GET", exportURL, nil)
	if err != nil {
		log.Error().Err(err).Str("url", exportURL).Msg("Failed to create request")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create request"})
		return
	}

	// Add API key if configured
	if agent.APIKey != nil && *agent.APIKey != "" {
		req.Header.Set("X-API-Key", *agent.APIKey)
	}

	// Make HTTP request to agent's export endpoint
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Error().Err(err).Str("url", exportURL).Msg("Failed to fetch native bandwidth data from agent, falling back to cached data")
		
		// Fall back to cached historical snapshots from database
		// We need to reconstruct the vnstat JSON from individual period snapshots
		
		// Get latest snapshot to find interface name
		vnstatSnapshot, _ := h.db.GetMonitorLatestSnapshot(c.Request.Context(), id, "vnstat")
		interfaceName := "eth0" // default
		
		if vnstatSnapshot != nil {
			// The vnstat snapshot contains the full vnstat JSON
			var vnstatData map[string]interface{}
			if err := json.Unmarshal([]byte(vnstatSnapshot.DataJSON), &vnstatData); err == nil {
				vnstatData["from_cache"] = true
				vnstatData["cache_timestamp"] = vnstatSnapshot.CreatedAt.Format(time.RFC3339)
				c.JSON(http.StatusOK, vnstatData)
				return
			}
		}
		
		// If no full vnstat snapshot, try to reconstruct from period snapshots
		hourlySnapshot, _ := h.db.GetMonitorLatestSnapshot(c.Request.Context(), id, "hourly")
		dailySnapshot, _ := h.db.GetMonitorLatestSnapshot(c.Request.Context(), id, "daily") 
		monthlySnapshot, _ := h.db.GetMonitorLatestSnapshot(c.Request.Context(), id, "monthly")
		totalSnapshot, _ := h.db.GetMonitorLatestSnapshot(c.Request.Context(), id, "total")
		
		if hourlySnapshot == nil && dailySnapshot == nil && monthlySnapshot == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Agent is offline and no cached data available"})
			return
		}
		
		// Build the vnstat-like response structure
		response := map[string]interface{}{
			"vnstatversion": "2.11",
			"jsonversion": "2",
			"interfaces": []interface{}{},
			"from_cache": true,
		}
		
		// Get interface name from any available snapshot
		if hourlySnapshot != nil {
			interfaceName = hourlySnapshot.InterfaceName
		} else if dailySnapshot != nil {
			interfaceName = dailySnapshot.InterfaceName
		} else if monthlySnapshot != nil {
			interfaceName = monthlySnapshot.InterfaceName
		}
		
		// Build traffic data
		traffic := map[string]interface{}{}
		totalRx := int64(0)
		totalTx := int64(0)
		
		// Add hourly data if available
		if hourlySnapshot != nil {
			var hourlyData []interface{}
			if err := json.Unmarshal([]byte(hourlySnapshot.DataJSON), &hourlyData); err == nil {
				traffic["hour"] = hourlyData
			}
			response["cache_timestamp"] = hourlySnapshot.CreatedAt.Format(time.RFC3339)
		}
		
		// Add daily data if available
		if dailySnapshot != nil {
			var dailyData []interface{}
			if err := json.Unmarshal([]byte(dailySnapshot.DataJSON), &dailyData); err == nil {
				traffic["day"] = dailyData
			}
			if response["cache_timestamp"] == nil {
				response["cache_timestamp"] = dailySnapshot.CreatedAt.Format(time.RFC3339)
			}
		}
		
		// Add monthly data if available and calculate totals
		if monthlySnapshot != nil {
			var monthlyData []interface{}
			if err := json.Unmarshal([]byte(monthlySnapshot.DataJSON), &monthlyData); err == nil {
				traffic["month"] = monthlyData
				
				// Calculate total from all monthly data
				for _, month := range monthlyData {
					if m, ok := month.(map[string]interface{}); ok {
						if rx, ok := m["rx"].(float64); ok {
							totalRx += int64(rx)
						}
						if tx, ok := m["tx"].(float64); ok {
							totalTx += int64(tx)
						}
					}
				}
			}
			if response["cache_timestamp"] == nil {
				response["cache_timestamp"] = monthlySnapshot.CreatedAt.Format(time.RFC3339)
			}
		}
		
		// If we still don't have totals, try to calculate from daily data
		if totalRx == 0 && totalTx == 0 && dailySnapshot != nil {
			if dailyArray, ok := traffic["day"].([]interface{}); ok {
				for _, day := range dailyArray {
					if d, ok := day.(map[string]interface{}); ok {
						if rx, ok := d["rx"].(float64); ok {
							totalRx += int64(rx)
						}
						if tx, ok := d["tx"].(float64); ok {
							totalTx += int64(tx)
						}
					}
				}
			}
		}
		
		// Use saved total if available, otherwise use calculated total
		if totalSnapshot != nil {
			var totalData map[string]interface{}
			if err := json.Unmarshal([]byte(totalSnapshot.DataJSON), &totalData); err == nil {
				if rx, ok := totalData["rx"].(float64); ok {
					totalRx = int64(rx)
				}
				if tx, ok := totalData["tx"].(float64); ok {
					totalTx = int64(tx)
				}
			}
		}
		
		// Add the total to traffic data
		traffic["total"] = map[string]interface{}{
			"rx": totalRx,
			"tx": totalTx,
		}
		
		// Build interface object
		iface := map[string]interface{}{
			"name": interfaceName,
			"alias": "",
			"traffic": traffic,
		}
		
		response["interfaces"] = []interface{}{iface}
		
		c.JSON(http.StatusOK, response)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Error().Int("status", resp.StatusCode).Str("url", exportURL).Msg("Agent returned error status")
		c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("Agent returned status %d", resp.StatusCode)})
		return
	}

	// Read and parse the JSON response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Error().Err(err).Msg("Failed to read response body")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read response"})
		return
	}

	// Validate JSON
	var bandwidthData interface{}
	if err := json.Unmarshal(body, &bandwidthData); err != nil {
		log.Error().Err(err).Msg("Invalid JSON response from agent")
		c.JSON(http.StatusBadGateway, gin.H{"error": "Invalid JSON response from agent"})
		return
	}

	// Return the raw bandwidth monitor JSON data
	c.Header("Content-Type", "application/json")
	c.Data(http.StatusOK, "application/json", body)
}

// GetAgentSystemInfo returns system information from an agent
func (h *MonitorHandler) GetAgentSystemInfo(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid agent ID"})
		return
	}

	// Get the agent to retrieve its URL
	agent, err := h.db.GetMonitorAgent(c.Request.Context(), id)
	if err != nil {
		if err == database.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Agent not found"})
			return
		}
		log.Error().Err(err).Msg("Failed to get monitor agent")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get agent"})
		return
	}

	// Build the system info URL from the agent's base URL
	baseURL := strings.TrimSuffix(agent.URL, "/events?stream=live-data")
	systemURL := baseURL + "/system/info"

	// Create HTTP request with API key if configured
	req, err := http.NewRequest("GET", systemURL, nil)
	if err != nil {
		log.Error().Err(err).Str("url", systemURL).Msg("Failed to create request")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create request"})
		return
	}

	// Add API key if configured
	if agent.APIKey != nil && *agent.APIKey != "" {
		req.Header.Set("X-API-Key", *agent.APIKey)
	}

	// Make HTTP request to agent's system endpoint
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Error().Err(err).Str("url", systemURL).Msg("Failed to fetch system info from agent, falling back to cached data")
		
		// Fall back to cached system info from database
		systemInfo, dbErr := h.db.GetMonitorSystemInfo(c.Request.Context(), id)
		if dbErr != nil {
			if dbErr == database.ErrNotFound {
				c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Agent is offline and no cached data available"})
				return
			}
			log.Error().Err(dbErr).Msg("Failed to get cached system info")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get system info"})
			return
		}
		
		// Get interfaces
		interfaces, dbErr := h.db.GetMonitorInterfaces(c.Request.Context(), id)
		if dbErr != nil {
			log.Error().Err(dbErr).Msg("Failed to get cached interfaces")
		}
		
		// Build response matching the agent's system info format
		interfaceMap := make(map[string]interface{})
		for _, iface := range interfaces {
			interfaceMap[iface.Name] = map[string]interface{}{
				"name":        iface.Name,
				"alias":       iface.Alias,
				"ip_address":  iface.IPAddress,
				"link_speed":  iface.LinkSpeed,
				"is_up":       true, // We don't store this, assume up
			}
		}
		
		response := map[string]interface{}{
			"hostname":       systemInfo.Hostname,
			"kernel":         systemInfo.Kernel,
			"vnstat_version": systemInfo.VnstatVersion,
			"interfaces":     interfaceMap,
			"cpu_model":      systemInfo.CPUModel,
			"cpu_cores":      systemInfo.CPUCores,
			"cpu_threads":    systemInfo.CPUThreads,
			"total_memory":   systemInfo.TotalMemory,
			"updated_at":     systemInfo.UpdatedAt.Format(time.RFC3339),
			"from_cache":     true,
		}
		
		c.JSON(http.StatusOK, response)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Error().Int("status", resp.StatusCode).Str("url", systemURL).Msg("Agent returned error status")
		c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("Agent returned status %d", resp.StatusCode)})
		return
	}

	// Read and parse the JSON response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Error().Err(err).Msg("Failed to read response body")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read response"})
		return
	}

	// Validate JSON
	var systemData interface{}
	if err := json.Unmarshal(body, &systemData); err != nil {
		log.Error().Err(err).Msg("Invalid JSON response from agent")
		c.JSON(http.StatusBadGateway, gin.H{"error": "Invalid JSON response from agent"})
		return
	}

	// Return the system info JSON data
	c.Header("Content-Type", "application/json")
	c.Data(http.StatusOK, "application/json", body)
}

// GetAgentHardwareStats returns hardware statistics from an agent
func (h *MonitorHandler) GetAgentHardwareStats(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid agent ID"})
		return
	}

	// Get the agent to retrieve its URL
	agent, err := h.db.GetMonitorAgent(c.Request.Context(), id)
	if err != nil {
		if err == database.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Agent not found"})
			return
		}
		log.Error().Err(err).Msg("Failed to get monitor agent")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get agent"})
		return
	}

	// Build the hardware stats URL from the agent's base URL
	baseURL := strings.TrimSuffix(agent.URL, "/events?stream=live-data")
	hardwareURL := baseURL + "/system/hardware"

	// Create HTTP request with API key if configured
	req, err := http.NewRequest("GET", hardwareURL, nil)
	if err != nil {
		log.Error().Err(err).Str("url", hardwareURL).Msg("Failed to create request")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create request"})
		return
	}

	// Add API key if configured
	if agent.APIKey != nil && *agent.APIKey != "" {
		req.Header.Set("X-API-Key", *agent.APIKey)
	}

	// Make HTTP request to agent's hardware endpoint
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Error().Err(err).Str("url", hardwareURL).Msg("Failed to fetch hardware stats from agent, falling back to cached data")
		
		// Fall back to cached resource stats from database
		// Try to get the most recent resource stats (last 24 hours)
		resourceStats, dbErr := h.db.GetMonitorResourceStats(c.Request.Context(), id, 24)
		if dbErr != nil || len(resourceStats) == 0 {
			// If no resource stats, still try to return basic system info
			systemInfo, _ := h.db.GetMonitorSystemInfo(c.Request.Context(), id)
			
			response := map[string]interface{}{
				"cpu": map[string]interface{}{
					"usage_percent": 0,
					"cores":         0,
					"threads":       0,
					"model":         "Unknown",
					"frequency":     0,
				},
				"memory": map[string]interface{}{
					"total":        int64(0),
					"used":         int64(0),
					"free":         int64(0),
					"available":    int64(0),
					"used_percent": 0,
					"swap_total":   0,
					"swap_used":    0,
					"swap_percent": 0,
				},
				"disks":       []interface{}{},
				"temperature": []interface{}{},
				"uptime":      0,
				"updated_at":  time.Now().Format(time.RFC3339),
				"from_cache":  true,
			}
			
			// If we have system info, use it for CPU details
			if systemInfo != nil {
				response["cpu"].(map[string]interface{})["model"] = systemInfo.CPUModel
				response["cpu"].(map[string]interface{})["cores"] = systemInfo.CPUCores
				response["cpu"].(map[string]interface{})["threads"] = systemInfo.CPUThreads
				response["memory"].(map[string]interface{})["total"] = systemInfo.TotalMemory
			}
			
			c.JSON(http.StatusOK, response)
			return
		}
		
		// Get the most recent stats
		latestStats := resourceStats[0]
		
		// Get system info for CPU model
		systemInfo, _ := h.db.GetMonitorSystemInfo(c.Request.Context(), id)
		
		// Parse disk usage JSON
		var diskUsage []interface{}
		if latestStats.DiskUsageJSON != "" {
			json.Unmarshal([]byte(latestStats.DiskUsageJSON), &diskUsage)
		}
		
		// Parse temperature JSON
		var temperature []interface{}
		if latestStats.TemperatureJSON != "" {
			json.Unmarshal([]byte(latestStats.TemperatureJSON), &temperature)
		}
		
		cpuModel := "Unknown"
		cpuCores := 0
		cpuThreads := 0
		if systemInfo != nil {
			cpuModel = systemInfo.CPUModel
			cpuCores = systemInfo.CPUCores
			cpuThreads = systemInfo.CPUThreads
		}
		
		// Build response matching the agent's hardware stats format
		totalMemory := int64(0)
		if systemInfo != nil {
			totalMemory = systemInfo.TotalMemory
		}
		
		response := map[string]interface{}{
			"cpu": map[string]interface{}{
				"usage_percent": latestStats.CPUUsagePercent,
				"cores":         cpuCores,
				"threads":       cpuThreads,
				"model":         cpuModel,
				"frequency":     0, // Not stored in database
			},
			"memory": map[string]interface{}{
				"total":        totalMemory,
				"used":         int64(float64(totalMemory) * latestStats.MemoryUsedPercent / 100),
				"free":         int64(float64(totalMemory) * (100 - latestStats.MemoryUsedPercent) / 100),
				"available":    int64(float64(totalMemory) * (100 - latestStats.MemoryUsedPercent) / 100),
				"used_percent": latestStats.MemoryUsedPercent,
				"swap_total":   0, // Not directly stored
				"swap_used":    0, // Not directly stored
				"swap_percent": latestStats.SwapUsedPercent,
			},
			"disks":       diskUsage,
			"temperature": temperature,
			"uptime":      latestStats.UptimeSeconds,
			"updated_at":  latestStats.CreatedAt.Format(time.RFC3339),
			"from_cache":  true,
		}
		
		c.JSON(http.StatusOK, response)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Error().Int("status", resp.StatusCode).Str("url", hardwareURL).Msg("Agent returned error status")
		c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("Agent returned status %d", resp.StatusCode)})
		return
	}

	// Read and parse the JSON response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Error().Err(err).Msg("Failed to read response body")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read response"})
		return
	}

	// Validate JSON
	var hardwareData interface{}
	if err := json.Unmarshal(body, &hardwareData); err != nil {
		log.Error().Err(err).Msg("Invalid JSON response from agent")
		c.JSON(http.StatusBadGateway, gin.H{"error": "Invalid JSON response from agent"})
		return
	}

	// Return the hardware stats JSON data
	c.Header("Content-Type", "application/json")
	c.Data(http.StatusOK, "application/json", body)
}

// GetAgentPeakStats returns peak bandwidth statistics from an agent
func (h *MonitorHandler) GetAgentPeakStats(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid agent ID"})
		return
	}

	// Get the agent to retrieve its URL
	agent, err := h.db.GetMonitorAgent(c.Request.Context(), id)
	if err != nil {
		if err == database.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Agent not found"})
			return
		}
		log.Error().Err(err).Msg("Failed to get monitor agent")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get agent"})
		return
	}

	// Build the peak stats URL from the agent's base URL
	baseURL := strings.TrimSuffix(agent.URL, "/events?stream=live-data")
	peaksURL := baseURL + "/stats/peaks"

	// Create HTTP request with API key if configured
	req, err := http.NewRequest("GET", peaksURL, nil)
	if err != nil {
		log.Error().Err(err).Str("url", peaksURL).Msg("Failed to create request")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create request"})
		return
	}

	// Add API key if configured
	if agent.APIKey != nil && *agent.APIKey != "" {
		req.Header.Set("X-API-Key", *agent.APIKey)
	}

	// Make HTTP request to agent's peaks endpoint
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Error().Err(err).Str("url", peaksURL).Msg("Failed to fetch peak stats")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch peak stats"})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Error().Int("status", resp.StatusCode).Str("url", peaksURL).Msg("Agent returned error status")
		c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("Agent returned status %d", resp.StatusCode)})
		return
	}

	// Read and parse the JSON response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Error().Err(err).Msg("Failed to read response body")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read response"})
		return
	}

	// Validate JSON
	var peakData interface{}
	if err := json.Unmarshal(body, &peakData); err != nil {
		log.Error().Err(err).Msg("Invalid JSON response from agent")
		c.JSON(http.StatusBadGateway, gin.H{"error": "Invalid JSON response from agent"})
		return
	}

	// Return the peak stats JSON data
	c.Header("Content-Type", "application/json")
	c.Data(http.StatusOK, "application/json", body)
}
