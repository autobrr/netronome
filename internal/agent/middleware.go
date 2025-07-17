// Copyright (c) 2024-2025, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package agent

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// corsMiddleware handles CORS headers
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "*")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "*")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// authMiddleware validates API key from header or query parameter
func (a *Agent) authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check header first
		apiKey := c.GetHeader("X-API-Key")
		// If not in header, check query parameter
		if apiKey == "" {
			apiKey = c.Query("apikey")
		}
		// Validate API key
		if apiKey != a.config.APIKey {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing API key"})
			c.Abort()
			return
		}
		c.Next()
	}
}