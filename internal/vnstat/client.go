// Copyright (c) 2024-2025, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package vnstat

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/autobrr/netronome/internal/config"
	"github.com/autobrr/netronome/internal/database"
	"github.com/autobrr/netronome/internal/types"
)

// Client represents an SSE client connection to a vnstat agent
type Client struct {
	agent         *types.VnstatAgent
	db            database.Service
	broadcastFunc func(types.VnstatUpdate)

	mu        sync.Mutex
	connected bool
	lastData  *types.VnstatLiveData
	ctx       context.Context
	cancel    context.CancelFunc
}

// Service manages all vnstat clients
type Service struct {
	db            database.Service
	config        *config.VnstatConfig
	broadcastFunc func(types.VnstatUpdate)

	clientsMu sync.RWMutex
	clients   map[int64]*Client

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewService creates a new vnstat service
func NewService(db database.Service, cfg *config.VnstatConfig, broadcastFunc func(types.VnstatUpdate)) *Service {
	ctx, cancel := context.WithCancel(context.Background())

	return &Service{
		db:            db,
		config:        cfg,
		broadcastFunc: broadcastFunc,
		clients:       make(map[int64]*Client),
		ctx:           ctx,
		cancel:        cancel,
	}
}

// Start starts the vnstat service
func (s *Service) Start() error {
	if !s.config.Enabled {
		log.Info().Msg("Vnstat service is disabled")
		return nil
	}

	// Load all enabled agents from database
	agents, err := s.db.GetVnstatAgents(s.ctx, true)
	if err != nil {
		return fmt.Errorf("failed to load vnstat agents: %w", err)
	}

	log.Debug().Int("agent_count", len(agents)).Msg("Loaded vnstat agents from database")

	// Start monitoring each agent
	for _, agent := range agents {
		if err := s.StartAgent(agent.ID); err != nil {
			log.Error().Err(err).Int64("agent_id", agent.ID).Msg("Failed to start vnstat agent")
		}
	}

	// Start cleanup routine
	s.wg.Add(1)
	go s.cleanupRoutine()

	// Start aggregation routine
	s.wg.Add(1)
	go s.aggregationRoutine()

	log.Info().Int("agents", len(agents)).Msg("Vnstat service started")
	return nil
}

// Stop stops the vnstat service
func (s *Service) Stop() {
	log.Info().Msg("Stopping vnstat service")

	s.cancel()

	// Stop all clients
	s.clientsMu.Lock()
	for _, client := range s.clients {
		client.Stop()
	}
	s.clientsMu.Unlock()

	s.wg.Wait()
	log.Info().Msg("Vnstat service stopped")
}

// StartAgent starts monitoring a specific agent
func (s *Service) StartAgent(agentID int64) error {
	// Get agent from database
	agent, err := s.db.GetVnstatAgent(s.ctx, agentID)
	if err != nil {
		return fmt.Errorf("failed to get agent: %w", err)
	}

	if !agent.Enabled {
		return fmt.Errorf("agent is disabled")
	}

	// Stop existing client if any
	s.StopAgent(agentID)

	// Create new client
	client := &Client{
		agent:         agent,
		db:            s.db,
		broadcastFunc: s.broadcastFunc,
	}

	// Start monitoring
	client.Start()

	// Store client
	s.clientsMu.Lock()
	s.clients[agentID] = client
	s.clientsMu.Unlock()

	log.Info().Int64("agent_id", agentID).Str("url", agent.URL).Msg("Started vnstat agent")
	return nil
}

// StopAgent stops monitoring a specific agent
func (s *Service) StopAgent(agentID int64) {
	s.clientsMu.Lock()
	client, exists := s.clients[agentID]
	if exists {
		delete(s.clients, agentID)
	}
	s.clientsMu.Unlock()

	if exists {
		client.Stop()
		log.Info().Int64("agent_id", agentID).Msg("Stopped vnstat agent")
	}
}

// GetAgentStatus returns the status of an agent
func (s *Service) GetAgentStatus(agentID int64) (bool, *types.VnstatLiveData) {
	s.clientsMu.RLock()
	client, exists := s.clients[agentID]
	s.clientsMu.RUnlock()

	if !exists {
		return false, nil
	}

	return client.IsConnected()
}

// cleanupRoutine periodically cleans up old bandwidth data
func (s *Service) cleanupRoutine() {
	defer s.wg.Done()

	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			if err := s.db.CleanupOldVnstatData(s.ctx); err != nil {
				log.Error().Err(err).Msg("Failed to cleanup old vnstat data")
			}
		}
	}
}

// aggregationRoutine periodically aggregates bandwidth data into hourly buckets
func (s *Service) aggregationRoutine() {
	defer s.wg.Done()

	// Run aggregation every 10 minutes
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	// Run once on startup to catch any data that needs aggregating
	if err := s.db.AggregateVnstatBandwidthHourly(s.ctx); err != nil {
		log.Error().Err(err).Msg("Failed to aggregate vnstat bandwidth data on startup")
	}

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			if err := s.db.AggregateVnstatBandwidthHourly(s.ctx); err != nil {
				log.Error().Err(err).Msg("Failed to aggregate vnstat bandwidth data")
			}
		}
	}
}

// Client methods

// Start starts the client connection
func (c *Client) Start() {
	c.mu.Lock()
	if c.cancel != nil {
		c.cancel()
	}
	c.ctx, c.cancel = context.WithCancel(context.Background())
	c.mu.Unlock()

	go c.monitor()
}

// Stop stops the client connection
func (c *Client) Stop() {
	c.mu.Lock()
	if c.cancel != nil {
		c.cancel()
	}
	c.connected = false
	c.mu.Unlock()
}

// IsConnected returns the connection status and last data
func (c *Client) IsConnected() (bool, *types.VnstatLiveData) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.connected, c.lastData
}

// monitor connects to the SSE endpoint and processes data
func (c *Client) monitor() {
	reconnectDelay := time.Second
	maxReconnectDelay := time.Minute

	for {
		select {
		case <-c.ctx.Done():
			return
		default:
		}

		// Connect to SSE endpoint
		err := c.connectAndStream()
		if err != nil {
			log.Error().
				Err(err).
				Int64("agent_id", c.agent.ID).
				Str("url", c.agent.URL).
				Msg("Failed to connect to vnstat agent")

			// Update connection status
			c.mu.Lock()
			c.connected = false
			c.mu.Unlock()

			// Broadcast disconnection
			c.broadcastFunc(types.VnstatUpdate{
				Type:      "vnstat",
				AgentID:   c.agent.ID,
				AgentName: c.agent.Name,
				Connected: false,
			})

			// Wait before reconnecting
			select {
			case <-c.ctx.Done():
				return
			case <-time.After(reconnectDelay):
				// Exponential backoff
				reconnectDelay *= 2
				if reconnectDelay > maxReconnectDelay {
					reconnectDelay = maxReconnectDelay
				}
			}
		} else {
			// Reset reconnect delay on successful connection
			reconnectDelay = time.Second
		}
	}
}

// connectAndStream connects to the SSE endpoint and streams data
func (c *Client) connectAndStream() error {
	// Create request with context
	req, err := http.NewRequestWithContext(c.ctx, "GET", c.agent.URL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers for SSE
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 0, // No timeout for SSE connections
	}

	// Make request
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Update connection status
	c.mu.Lock()
	c.connected = true
	c.mu.Unlock()

	// Broadcast connection
	c.broadcastFunc(types.VnstatUpdate{
		Type:      "vnstat",
		AgentID:   c.agent.ID,
		AgentName: c.agent.Name,
		Connected: true,
	})

	log.Info().
		Int64("agent_id", c.agent.ID).
		Str("url", c.agent.URL).
		Msg("Connected to vnstat agent")

	// Read SSE stream
	scanner := bufio.NewScanner(resp.Body)
	var eventData string

	for scanner.Scan() {
		line := scanner.Text()

		// SSE format: lines starting with "data: " contain the actual data
		if strings.HasPrefix(line, "data:") {
			// Extract the data part (handle both "data: " and "data:")
			eventData = strings.TrimPrefix(line, "data:")
			eventData = strings.TrimSpace(eventData)
		} else if line == "" && eventData != "" {
			// Empty line indicates end of event
			c.processData(eventData)
			eventData = ""
		}

		// Check if context is cancelled
		select {
		case <-c.ctx.Done():
			return c.ctx.Err()
		default:
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scanner error: %w", err)
	}

	return fmt.Errorf("connection closed")
}

// processData processes incoming vnstat data
func (c *Client) processData(data string) {
	// Parse JSON data
	var liveData types.VnstatLiveData
	if err := json.Unmarshal([]byte(data), &liveData); err != nil {
		log.Warn().
			Err(err).
			Str("data", data).
			Msg("Failed to parse vnstat data")
		return
	}

	// Update last data
	c.mu.Lock()
	c.lastData = &liveData
	c.mu.Unlock()

	log.Trace().
		Int64("agent_id", c.agent.ID).
		Str("rx_rate", liveData.Rx.Ratestring).
		Str("tx_rate", liveData.Tx.Ratestring).
		Msg("Updated live data")

	// Convert to int64 for storage
	rxBytes := int64(liveData.Rx.Bytespersecond)
	txBytes := int64(liveData.Tx.Bytespersecond)
	rxPackets := liveData.Rx.Packetspersecond
	txPackets := liveData.Tx.Packetspersecond

	// Store in database
	bandwidth := &types.VnstatBandwidth{
		AgentID:            c.agent.ID,
		RxBytesPerSecond:   &rxBytes,
		TxBytesPerSecond:   &txBytes,
		RxPacketsPerSecond: &rxPackets,
		TxPacketsPerSecond: &txPackets,
		RxRateString:       &liveData.Rx.Ratestring,
		TxRateString:       &liveData.Tx.Ratestring,
	}

	if err := c.db.SaveVnstatBandwidth(context.Background(), bandwidth); err != nil {
		log.Error().
			Err(err).
			Int64("agent_id", c.agent.ID).
			Msg("Failed to save vnstat bandwidth data")
	}

	// Broadcast update
	c.broadcastFunc(types.VnstatUpdate{
		Type:             "vnstat",
		AgentID:          c.agent.ID,
		AgentName:        c.agent.Name,
		RxBytesPerSecond: rxBytes,
		TxBytesPerSecond: txBytes,
		RxRateString:     liveData.Rx.Ratestring,
		TxRateString:     liveData.Tx.Ratestring,
		Connected:        true,
	})
}
