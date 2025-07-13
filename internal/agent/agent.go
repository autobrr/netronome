// Copyright (c) 2024-2025, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package agent

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"

	"github.com/autobrr/netronome/internal/config"
)

// Agent represents the vnstat SSE agent
type Agent struct {
	config     *config.AgentConfig
	clients    map[chan string]bool
	clientsMu  sync.RWMutex
	vnstatData chan string
}

// VnstatLiveData represents the JSON structure from vnstat --live --json
type VnstatLiveData struct {
	Index   int `json:"index"`
	Seconds int `json:"seconds"`
	Rx      struct {
		Ratestring       string `json:"ratestring"`
		Bytespersecond   int    `json:"bytespersecond"`
		Packetspersecond int    `json:"packetspersecond"`
		Bytes            int    `json:"bytes"`
		Packets          int    `json:"packets"`
		Totalbytes       int    `json:"totalbytes"`
		Totalpackets     int    `json:"totalpackets"`
	} `json:"rx"`
	Tx struct {
		Ratestring       string `json:"ratestring"`
		Bytespersecond   int    `json:"bytespersecond"`
		Packetspersecond int    `json:"packetspersecond"`
		Bytes            int    `json:"bytes"`
		Packets          int    `json:"packets"`
		Totalbytes       int    `json:"totalbytes"`
		Totalpackets     int    `json:"totalpackets"`
	} `json:"tx"`
}

// New creates a new Agent instance
func New(cfg *config.AgentConfig) *Agent {
	return &Agent{
		config:     cfg,
		clients:    make(map[chan string]bool),
		vnstatData: make(chan string, 100),
	}
}

// Start starts the agent server
func (a *Agent) Start(ctx context.Context) error {
	// Start vnstat monitoring
	go a.runVnstat(ctx)

	// Start broadcaster
	go a.broadcaster(ctx)

	// Set up Gin router
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())

	// CORS middleware
	router.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusOK)
			return
		}

		c.Next()
	})

	// Root endpoint
	router.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"service":  "vnstat SSE agent",
			"port":     a.config.Port,
			"endpoint": "/events?stream=live-data",
		})
	})

	// SSE endpoint
	router.GET("/events", a.handleSSE)

	// Start HTTP server
	addr := fmt.Sprintf(":%d", a.config.Port)
	server := &http.Server{
		Addr:    addr,
		Handler: router,
	}

	// Handle graceful shutdown
	go func() {
		<-ctx.Done()
		log.Info().Msg("Shutting down agent server...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Error().Err(err).Msg("Failed to gracefully shutdown server")
		}
	}()

	log.Info().Str("addr", addr).Msg("Starting vnstat SSE agent")
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("failed to start server: %w", err)
	}

	return nil
}

// handleSSE handles SSE connections
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

// broadcaster distributes data to all connected clients
func (a *Agent) broadcaster(ctx context.Context) {
	for {
		select {
		case data := <-a.vnstatData:
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

// runVnstat runs vnstat and sends data to the broadcast channel
func (a *Agent) runVnstat(ctx context.Context) {
	// Build vnstat command
	args := []string{"--live", "--json"}
	if a.config.Interface != "" {
		args = append(args, "--iface", a.config.Interface)
	}

	cmd := exec.CommandContext(ctx, "vnstat", args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Error().Err(err).Msg("Failed to create stdout pipe")
		return
	}

	if err := cmd.Start(); err != nil {
		log.Error().Err(err).Msg("Failed to start vnstat")
		return
	}

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()

		// Parse JSON to validate it
		var data VnstatLiveData
		if err := json.Unmarshal([]byte(line), &data); err != nil {
			log.Warn().Err(err).Str("line", line).Msg("Failed to parse vnstat JSON")
			continue
		}

		// Send to broadcaster
		select {
		case a.vnstatData <- line:
		default:
			// Channel full, skip
		}

		log.Debug().
			Str("rx", data.Rx.Ratestring).
			Str("tx", data.Tx.Ratestring).
			Msg("Broadcasting vnstat data")
	}

	if err := scanner.Err(); err != nil {
		log.Error().Err(err).Msg("Scanner error")
	}

	if err := cmd.Wait(); err != nil {
		log.Error().Err(err).Msg("vnstat command failed")
	}
}
