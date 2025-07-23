// Copyright (c) 2024-2025, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package agent

import (
	"context"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
)

// handleSSE handles Server-Sent Events connections for real-time data streaming
func (a *Agent) handleSSE(c *gin.Context) {
	stream := c.Query("stream")
	if stream != "live-data" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid stream parameter"})
		return
	}

	// Create a client channel
	clientChan := make(chan string, 100)

	// Register client
	a.clientsMu.Lock()
	a.clients[clientChan] = true
	a.clientsMu.Unlock()

	// Clean up on disconnect
	defer func() {
		a.clientsMu.Lock()
		delete(a.clients, clientChan)
		a.clientsMu.Unlock()
		close(clientChan)
	}()

	c.Stream(func(w io.Writer) bool {
		select {
		case data := <-clientChan:
			c.SSEvent("message", data)
			return true
		case <-c.Request.Context().Done():
			return false
		}
	})
}

// broadcaster distributes monitoring data to all connected SSE clients
func (a *Agent) broadcaster(ctx context.Context) {
	for {
		select {
		case data := <-a.monitorData:
			a.clientsMu.RLock()
			for client := range a.clients {
				select {
				case client <- data:
				default:
					// Client buffer full, skip
				}
			}
			a.clientsMu.RUnlock()
		case <-ctx.Done():
			return
		}
	}
}
