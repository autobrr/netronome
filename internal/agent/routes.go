// Copyright (c) 2024-2025, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package agent

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// setupRoutes configures all HTTP routes for the agent
func (a *Agent) setupRoutes() *gin.Engine {
	// Set up Gin router
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())

	// CORS middleware - simplified version that should work
	router.Use(corsMiddleware())

	// Root endpoint (public, no auth required)
	router.GET("/", a.handleRoot)

	// Create authenticated group for protected endpoints
	protected := router.Group("/")
	if a.config.APIKey != "" {
		protected.Use(a.authMiddleware())
	}

	// SSE endpoint (protected)
	protected.GET("/events", a.handleSSE)

	// Historical data export endpoint (protected)
	protected.GET("/export/historical", a.handleHistoricalExport)

	// System info endpoint (protected)
	protected.GET("/system/info", a.handleSystemInfo)

	// Peak stats endpoint (protected)
	protected.GET("/stats/peaks", a.handlePeakStats)

	// Hardware stats endpoint (protected)
	protected.GET("/system/hardware", a.handleHardwareStats)

	// Tailscale status endpoint (protected)
	protected.GET("/tailscale/status", a.handleTailscaleStatus)

	return router
}

// handleRoot handles the root endpoint
func (a *Agent) handleRoot(c *gin.Context) {
	response := gin.H{
		"service": "monitor SSE agent",
		"host":    a.config.Host,
		"port":    a.config.Port,
		"endpoints": gin.H{
			"live":       "/events?stream=live-data",
			"historical": "/export/historical",
			"system":     "/system/info",
			"hardware":   "/system/hardware",
			"peaks":      "/stats/peaks",
			"tailscale":  "/tailscale/status",
		},
	}

	// Indicate if authentication is required
	if a.config.APIKey != "" {
		response["authentication"] = "required"
		response["auth_methods"] = []string{"X-API-Key header", "apikey query parameter"}
	} else {
		response["authentication"] = "none"
	}

	c.JSON(http.StatusOK, response)
}

// handleTailscaleStatus handles the Tailscale status endpoint
func (a *Agent) handleTailscaleStatus(c *gin.Context) {
	status, err := a.GetTailscaleStatus()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, status)
}