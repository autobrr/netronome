// Copyright (c) 2024-2025, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package monitor

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/autobrr/netronome/internal/config"
	"github.com/autobrr/netronome/internal/database"
	"github.com/autobrr/netronome/internal/types"
)

// Notifier interface for sending notifications
type Notifier interface {
	SendAgentNotification(agentName string, eventType string, value *float64) error
}

// Client represents an SSE client connection to a monitor agent
type Client struct {
	agent         *types.MonitorAgent
	db            database.Service
	broadcastFunc func(types.MonitorUpdate)
	notifier      Notifier

	mu        sync.Mutex
	connected bool
	lastData  *types.MonitorLiveData
	ctx       context.Context
	cancel    context.CancelFunc

	// Peak tracking
	peakRx          int64
	peakTx          int64
	peakRxTimestamp time.Time
	peakTxTimestamp time.Time

	// Resource state tracking for notifications
	lastCPUNotificationTime       time.Time
	lastMemoryNotificationTime    time.Time
	lastDiskNotificationTime      time.Time
	lastBandwidthNotificationTime time.Time
	lastTempNotificationTime      time.Time
}

// Service manages all monitoring clients
type Service struct {
	db                 database.Service
	config             *config.MonitorConfig
	tailscaleConfig    *config.TailscaleConfig
	broadcastFunc      func(types.MonitorUpdate)
	tailscaleDiscovery *TailscaleDiscovery
	notifier           Notifier

	clientsMu   sync.RWMutex
	clients     map[int64]*Client
	agentStates map[int64]bool // Track connection state per agent

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// Background collection tickers
	resourceTicker *time.Ticker
	snapshotTicker *time.Ticker
	cleanupTicker  *time.Ticker
}

// NewService creates a new monitor service
func NewService(db database.Service, cfg *config.MonitorConfig, broadcastFunc func(types.MonitorUpdate), notifier Notifier) *Service {
	ctx, cancel := context.WithCancel(context.Background())

	return &Service{
		db:            db,
		config:        cfg,
		broadcastFunc: broadcastFunc,
		notifier:      notifier,
		clients:       make(map[int64]*Client),
		agentStates:   make(map[int64]bool),
		ctx:           ctx,
		cancel:        cancel,
	}
}

// NewServiceWithTailscale creates a new monitor service with Tailscale support
func NewServiceWithTailscale(db database.Service, cfg *config.MonitorConfig, tsCfg *config.TailscaleConfig, broadcastFunc func(types.MonitorUpdate), notifier Notifier) *Service {
	ctx, cancel := context.WithCancel(context.Background())

	service := &Service{
		db:              db,
		config:          cfg,
		tailscaleConfig: tsCfg,
		broadcastFunc:   broadcastFunc,
		notifier:        notifier,
		clients:         make(map[int64]*Client),
		agentStates:     make(map[int64]bool),
		ctx:             ctx,
		cancel:          cancel,
	}

	// Create Tailscale discovery if auto-discover is enabled
	// This works with both host and tsnet modes
	if tsCfg != nil && tsCfg.IsServerDiscoveryMode() {
		service.tailscaleDiscovery = NewTailscaleDiscovery(tsCfg, service)
	}

	return service
}

// Start starts the monitor service
func (s *Service) Start() error {
	if !s.config.Enabled {
		log.Info().Msg("Monitor service is disabled")
		return nil
	}

	// Load all enabled agents from database
	agents, err := s.db.GetMonitorAgents(s.ctx, true)
	if err != nil {
		return fmt.Errorf("failed to load monitor agents: %w", err)
	}

	log.Debug().Int("agent_count", len(agents)).Msg("Loaded monitor agents from database")

	// Start monitoring each agent
	for _, agent := range agents {
		if err := s.StartAgent(agent.ID); err != nil {
			log.Error().Err(err).Int64("agent_id", agent.ID).Msg("Failed to start monitor agent")
		}
	}

	// Start background collection tasks
	s.startBackgroundCollectors()

	// Start Tailscale discovery if enabled
	if s.tailscaleDiscovery != nil {
		if err := s.tailscaleDiscovery.Start(s.ctx); err != nil {
			log.Error().Err(err).Msg("Failed to start Tailscale discovery")
		} else {
			log.Info().Msg("Tailscale discovery service started")
		}
	}

	// Do an initial collection for all connected agents
	go func() {
		// Wait a bit for agents to connect
		time.Sleep(5 * time.Second)
		s.collectResourceStats()
		s.collectHistoricalSnapshots()
	}()

	log.Info().Int("agents", len(agents)).Msg("Monitor service started")
	return nil
}

// Stop stops the monitor service
func (s *Service) Stop() {
	log.Info().Msg("Stopping monitor service")

	s.cancel()

	// Stop Tailscale discovery if running
	if s.tailscaleDiscovery != nil {
		s.tailscaleDiscovery.Stop()
	}

	// Stop background collectors
	s.stopBackgroundCollectors()

	// Stop all clients
	s.clientsMu.Lock()
	for _, client := range s.clients {
		client.Stop()
	}
	s.clientsMu.Unlock()

	s.wg.Wait()

	// Run cleanup before shutdown
	log.Info().Msg("Running monitor data cleanup before shutdown")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := s.db.CleanupMonitorData(ctx); err != nil {
		log.Error().Err(err).Msg("Failed to cleanup monitor data on shutdown")
	} else {
		log.Info().Msg("Monitor data cleanup completed")
	}

	log.Info().Msg("Monitor service stopped")
}

// broadcastWithNotification wraps the broadcast function and handles notifications
func (s *Service) broadcastWithNotification(update types.MonitorUpdate) {
	// Don't send notifications if the service is shutting down
	select {
	case <-s.ctx.Done():
		// Service is shutting down, just broadcast without notifications
		s.broadcastFunc(update)
		return
	default:
	}

	// Get previous connection state
	s.clientsMu.Lock()
	wasConnected, exists := s.agentStates[update.AgentID]
	if !exists {
		// If no previous state, assume it was connected
		wasConnected = true
	}
	s.agentStates[update.AgentID] = update.Connected
	s.clientsMu.Unlock()

	// Check if agent went offline or came back online
	if wasConnected && !update.Connected {
		// Agent went offline
		log.Warn().
			Int64("agentID", update.AgentID).
			Str("agentName", update.AgentName).
			Msg("Agent went offline")

		if s.notifier != nil {
			err := s.notifier.SendAgentNotification(update.AgentName, database.NotificationEventAgentOffline, nil)
			if err != nil {
				log.Error().Err(err).Msg("Failed to send agent offline notification")
			}
		}
	} else if !wasConnected && update.Connected {
		// Agent came back online
		log.Info().
			Int64("agentID", update.AgentID).
			Str("agentName", update.AgentName).
			Msg("Agent came back online")

		if s.notifier != nil {
			err := s.notifier.SendAgentNotification(update.AgentName, database.NotificationEventAgentOnline, nil)
			if err != nil {
				log.Error().Err(err).Msg("Failed to send agent online notification")
			}
		}
	}

	// Always broadcast the update
	s.broadcastFunc(update)
}

// StartAgent starts monitoring a specific agent
func (s *Service) StartAgent(agentID int64) error {
	// Get agent from database
	agent, err := s.db.GetMonitorAgent(s.ctx, agentID)
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
		broadcastFunc: s.broadcastWithNotification,
		notifier:      s.notifier,
	}

	// Start monitoring
	client.Start()

	// Store client
	s.clientsMu.Lock()
	s.clients[agentID] = client
	s.clientsMu.Unlock()

	log.Info().Int64("agent_id", agentID).Str("url", agent.URL).Msg("Started monitor agent")
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
		log.Info().Int64("agent_id", agentID).Msg("Stopped monitor agent")
	}
}

// GetAgentStatus returns the status of an agent
func (s *Service) GetAgentStatus(agentID int64) (bool, *types.MonitorLiveData) {
	s.clientsMu.RLock()
	client, exists := s.clients[agentID]
	s.clientsMu.RUnlock()

	if !exists {
		return false, nil
	}

	return client.IsConnected()
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
func (c *Client) IsConnected() (bool, *types.MonitorLiveData) {
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
			// Don't log error if context was cancelled (normal shutdown)
			if errors.Is(err, context.Canceled) {
				log.Debug().
					Int64("agent_id", c.agent.ID).
					Msg("Agent connection cancelled")
			} else {
				log.Error().
					Err(err).
					Int64("agent_id", c.agent.ID).
					Str("url", c.agent.URL).
					Msg("Failed to connect to monitor agent")
			}

			// Update connection status
			c.mu.Lock()
			c.connected = false
			c.mu.Unlock()

			// Broadcast disconnection
			c.broadcastFunc(types.MonitorUpdate{
				Type:      "monitor",
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

	// Add API key if configured
	if c.agent.APIKey != nil && *c.agent.APIKey != "" {
		req.Header.Set("X-API-Key", *c.agent.APIKey)
	}

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
	c.broadcastFunc(types.MonitorUpdate{
		Type:      "monitor",
		AgentID:   c.agent.ID,
		AgentName: c.agent.Name,
		Connected: true,
	})

	log.Info().
		Int64("agent_id", c.agent.ID).
		Str("url", c.agent.URL).
		Msg("Connected to monitor agent")

	// Fetch initial peak stats after connection
	go c.fetchInitialPeakStats()

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

// processData processes incoming bandwidth monitor data
func (c *Client) processData(data string) {
	// Parse JSON data
	var liveData types.MonitorLiveData
	if err := json.Unmarshal([]byte(data), &liveData); err != nil {
		log.Warn().
			Err(err).
			Str("data", data).
			Msg("Failed to parse bandwidth monitor data")
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

	// Convert to int64 for broadcast
	rxBytes := int64(liveData.Rx.Bytespersecond)
	txBytes := int64(liveData.Tx.Bytespersecond)

	// Broadcast update
	c.broadcastFunc(types.MonitorUpdate{
		Type:             "monitor",
		AgentID:          c.agent.ID,
		AgentName:        c.agent.Name,
		RxBytesPerSecond: rxBytes,
		TxBytesPerSecond: txBytes,
		RxRateString:     liveData.Rx.Ratestring,
		TxRateString:     liveData.Tx.Ratestring,
		Connected:        true,
	})

	// Update peak stats if this is a new peak
	c.updatePeakStats(rxBytes, txBytes)

	// Check bandwidth threshold for notifications
	if c.notifier != nil {
		// Convert bytes per second to Mbps for threshold checking
		totalBandwidthMbps := float64(rxBytes+txBytes) * 8 / 1_000_000

		// Rate limit notifications to once per hour
		notificationCooldown := 1 * time.Hour
		now := time.Now()

		if totalBandwidthMbps > 0 && now.Sub(c.lastBandwidthNotificationTime) > notificationCooldown {
			if err := c.notifier.SendAgentNotification(
				c.agent.Name,
				database.NotificationEventAgentHighBandwidth,
				&totalBandwidthMbps,
			); err != nil {
				log.Error().Err(err).Msg("Failed to send high bandwidth notification")
			} else {
				c.lastBandwidthNotificationTime = now
			}
		}
	}
}

// updatePeakStats updates peak bandwidth statistics if current values are higher
func (c *Client) updatePeakStats(rxBytes, txBytes int64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	updated := false

	if rxBytes > c.peakRx {
		c.peakRx = rxBytes
		c.peakRxTimestamp = now
		updated = true
	}

	if txBytes > c.peakTx {
		c.peakTx = txBytes
		c.peakTxTimestamp = now
		updated = true
	}

	// Save to database if we have new peaks
	if updated {
		stats := &types.MonitorPeakStats{
			AgentID:         c.agent.ID,
			PeakRxBytes:     c.peakRx,
			PeakTxBytes:     c.peakTx,
			PeakRxTimestamp: &c.peakRxTimestamp,
			PeakTxTimestamp: &c.peakTxTimestamp,
		}
		if err := c.db.UpsertMonitorPeakStats(context.Background(), c.agent.ID, stats); err != nil {
			log.Warn().Err(err).Int64("agent_id", c.agent.ID).Msg("Failed to update peak stats")
		}
	}
}

// startBackgroundCollectors starts background data collection tasks
func (s *Service) startBackgroundCollectors() {
	// Bandwidth samples are collected in real-time via SSE, no separate ticker needed

	// Resource stats collection every 30 seconds
	s.resourceTicker = time.NewTicker(30 * time.Second)
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		for {
			select {
			case <-s.ctx.Done():
				return
			case <-s.resourceTicker.C:
				s.collectResourceStats()
			}
		}
	}()

	// Historical snapshot collection every hour
	s.snapshotTicker = time.NewTicker(1 * time.Hour)
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		for {
			select {
			case <-s.ctx.Done():
				return
			case <-s.snapshotTicker.C:
				s.collectHistoricalSnapshots()
			}
		}
	}()

	// Data cleanup every hour
	s.cleanupTicker = time.NewTicker(1 * time.Hour)
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		for {
			select {
			case <-s.ctx.Done():
				return
			case <-s.cleanupTicker.C:
				if err := s.db.CleanupMonitorData(s.ctx); err != nil {
					log.Error().Err(err).Msg("Failed to cleanup monitor data")
				}
			}
		}
	}()
}

// stopBackgroundCollectors stops all background collection tasks
func (s *Service) stopBackgroundCollectors() {
	if s.resourceTicker != nil {
		s.resourceTicker.Stop()
	}
	if s.snapshotTicker != nil {
		s.snapshotTicker.Stop()
	}
	if s.cleanupTicker != nil {
		s.cleanupTicker.Stop()
	}
}

// collectResourceStats collects hardware resource stats for all connected agents
func (s *Service) collectResourceStats() {
	s.clientsMu.RLock()
	agents := make([]*Client, 0, len(s.clients))
	for _, client := range s.clients {
		if connected, _ := client.IsConnected(); connected {
			agents = append(agents, client)
		}
	}
	s.clientsMu.RUnlock()

	for _, client := range agents {
		go s.fetchAndStoreResourceStats(client)
	}
}

// fetchAndStoreResourceStats fetches system info and hardware stats from an agent
func (s *Service) fetchAndStoreResourceStats(client *Client) {
	// Fetch system info
	if err := s.fetchSystemInfo(client); err != nil {
		log.Error().Err(err).Int64("agent_id", client.agent.ID).Msg("Failed to fetch system info")
	}

	// Fetch hardware stats
	if err := s.fetchHardwareStats(client); err != nil {
		log.Error().Err(err).Int64("agent_id", client.agent.ID).Msg("Failed to fetch hardware stats")
	}
}

// fetchSystemInfo fetches and stores system information from an agent
func (s *Service) fetchSystemInfo(client *Client) error {
	baseURL := strings.TrimSuffix(client.agent.URL, "/events?stream=live-data")
	systemURL := baseURL + "/system/info"

	req, err := http.NewRequestWithContext(client.ctx, "GET", systemURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	if client.agent.APIKey != nil && *client.agent.APIKey != "" {
		req.Header.Set("X-API-Key", *client.agent.APIKey)
	}

	httpClient := &http.Client{Timeout: 30 * time.Second}
	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to fetch system info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Read the response body first for debugging
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	// Log the raw response for debugging
	log.Trace().
		Int64("agent_id", client.agent.ID).
		Str("response", string(body)).
		Msg("System info raw response")

	var systemInfo struct {
		Hostname      string                 `json:"hostname"`
		Kernel        string                 `json:"kernel"`
		Uptime        int64                  `json:"uptime"`
		VnstatVersion string                 `json:"vnstat_version"`
		Interfaces    map[string]interface{} `json:"interfaces"`
		UpdatedAt     time.Time              `json:"updated_at"`
	}

	if err := json.Unmarshal(body, &systemInfo); err != nil {
		return fmt.Errorf("failed to decode system info: %w", err)
	}

	// Fetch agent version from /netronome/info endpoint
	agentVersion := ""
	infoURL := baseURL + "/netronome/info"
	infoReq, err := http.NewRequestWithContext(client.ctx, "GET", infoURL, nil)
	if err == nil {
		// Don't add API key for this endpoint as it's public
		infoResp, err := httpClient.Do(infoReq)
		if err == nil && infoResp.StatusCode == http.StatusOK {
			defer infoResp.Body.Close()
			var agentInfo struct {
				Version string `json:"version"`
			}
			if err := json.NewDecoder(infoResp.Body).Decode(&agentInfo); err == nil {
				agentVersion = agentInfo.Version
			}
		}
	}

	// Log parsed values for debugging
	log.Debug().
		Int64("agent_id", client.agent.ID).
		Str("hostname", systemInfo.Hostname).
		Str("kernel", systemInfo.Kernel).
		Str("vnstat_version", systemInfo.VnstatVersion).
		Str("agent_version", agentVersion).
		Int("interface_count", len(systemInfo.Interfaces)).
		Msg("Parsed system info values")

	// Store system info
	sysInfo := &types.MonitorSystemInfo{
		AgentID:       client.agent.ID,
		Hostname:      systemInfo.Hostname,
		Kernel:        systemInfo.Kernel,
		VnstatVersion: systemInfo.VnstatVersion,
		AgentVersion:  &agentVersion,
	}

	if err := s.db.UpsertMonitorSystemInfo(client.ctx, client.agent.ID, sysInfo); err != nil {
		return fmt.Errorf("failed to store system info: %w", err)
	}

	// Store interfaces
	var interfaces []types.MonitorInterface
	for ifaceName, ifaceData := range systemInfo.Interfaces {
		if ifaceMap, ok := ifaceData.(map[string]interface{}); ok {
			iface := types.MonitorInterface{
				AgentID: client.agent.ID,
				Name:    ifaceName,
			}

			if alias, ok := ifaceMap["alias"].(string); ok {
				iface.Alias = alias
			}
			if ip, ok := ifaceMap["ip_address"].(string); ok {
				iface.IPAddress = ip
			}
			if speed, ok := ifaceMap["link_speed"].(float64); ok {
				iface.LinkSpeed = int(speed)
			}

			interfaces = append(interfaces, iface)
		}
	}

	if err := s.db.UpsertMonitorInterfaces(client.ctx, client.agent.ID, interfaces); err != nil {
		return fmt.Errorf("failed to store interfaces: %w", err)
	}

	return nil
}

// fetchHardwareStats fetches and stores hardware statistics from an agent
func (s *Service) fetchHardwareStats(client *Client) error {
	baseURL := strings.TrimSuffix(client.agent.URL, "/events?stream=live-data")
	hardwareURL := baseURL + "/system/hardware"

	req, err := http.NewRequestWithContext(client.ctx, "GET", hardwareURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	if client.agent.APIKey != nil && *client.agent.APIKey != "" {
		req.Header.Set("X-API-Key", *client.agent.APIKey)
	}

	httpClient := &http.Client{Timeout: 30 * time.Second}
	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to fetch hardware stats: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var hardwareStats struct {
		CPU struct {
			UsagePercent float64 `json:"usage_percent"`
			Model        string  `json:"model"`
			Cores        int     `json:"cores"`
			Threads      int     `json:"threads"`
		} `json:"cpu"`
		Memory struct {
			UsedPercent float64 `json:"used_percent"`
			SwapPercent float64 `json:"swap_percent"`
		} `json:"memory"`
		Disks []struct {
			Path        string  `json:"path"`
			Device      string  `json:"device"`
			Fstype      string  `json:"fstype"`
			Total       uint64  `json:"total"`
			Used        uint64  `json:"used"`
			Free        uint64  `json:"free"`
			UsedPercent float64 `json:"used_percent"`
		} `json:"disks"`
		Temperature []struct {
			SensorKey   string  `json:"sensor_key"`
			Temperature float64 `json:"temperature"`
			Label       string  `json:"label"`
		} `json:"temperature"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&hardwareStats); err != nil {
		return fmt.Errorf("failed to decode hardware stats: %w", err)
	}

	// Update system info with CPU details
	if hardwareStats.CPU.Model != "" {
		sysInfo := &types.MonitorSystemInfo{
			AgentID:    client.agent.ID,
			CPUModel:   hardwareStats.CPU.Model,
			CPUCores:   hardwareStats.CPU.Cores,
			CPUThreads: hardwareStats.CPU.Threads,
		}
		if err := s.db.UpsertMonitorSystemInfo(client.ctx, client.agent.ID, sysInfo); err != nil {
			log.Warn().Err(err).Msg("Failed to update CPU info")
		}
	}

	// Store resource stats
	diskJSON, _ := json.Marshal(hardwareStats.Disks)
	tempJSON, _ := json.Marshal(hardwareStats.Temperature)

	stats := &types.MonitorResourceStats{
		AgentID:           client.agent.ID,
		CPUUsagePercent:   hardwareStats.CPU.UsagePercent,
		MemoryUsedPercent: hardwareStats.Memory.UsedPercent,
		SwapUsedPercent:   hardwareStats.Memory.SwapPercent,
		DiskUsageJSON:     string(diskJSON),
		TemperatureJSON:   string(tempJSON),
		UptimeSeconds:     0, // Will be set from system info
	}

	if err := s.db.SaveMonitorResourceStats(client.ctx, client.agent.ID, stats); err != nil {
		return fmt.Errorf("failed to store resource stats: %w", err)
	}

	// Check thresholds and send notifications if needed
	if client.notifier != nil {
		// Rate limit notifications to once per hour per type
		notificationCooldown := 1 * time.Hour
		now := time.Now()

		// Check CPU usage threshold
		// The notification service will check if CPU exceeds the configured threshold
		if hardwareStats.CPU.UsagePercent > 0 && now.Sub(client.lastCPUNotificationTime) > notificationCooldown {
			if err := client.notifier.SendAgentNotification(
				client.agent.Name,
				database.NotificationEventAgentHighCPU,
				&hardwareStats.CPU.UsagePercent,
			); err != nil {
				log.Error().Err(err).Msg("Failed to send high CPU notification")
			} else {
				client.lastCPUNotificationTime = now
			}
		}

		// Check memory usage threshold
		if hardwareStats.Memory.UsedPercent > 0 && now.Sub(client.lastMemoryNotificationTime) > notificationCooldown {
			if err := client.notifier.SendAgentNotification(
				client.agent.Name,
				database.NotificationEventAgentHighMemory,
				&hardwareStats.Memory.UsedPercent,
			); err != nil {
				log.Error().Err(err).Msg("Failed to send high memory notification")
			} else {
				client.lastMemoryNotificationTime = now
			}
		}

		// Check disk usage thresholds - find highest usage
		var highestDiskUsage float64
		for _, disk := range hardwareStats.Disks {
			if disk.UsedPercent > highestDiskUsage {
				highestDiskUsage = disk.UsedPercent
			}
		}

		if highestDiskUsage > 0 && now.Sub(client.lastDiskNotificationTime) > notificationCooldown {
			if err := client.notifier.SendAgentNotification(
				client.agent.Name,
				database.NotificationEventAgentLowDisk,
				&highestDiskUsage,
			); err != nil {
				log.Error().Err(err).Msg("Failed to send low disk notification")
			} else {
				client.lastDiskNotificationTime = now
			}
		}

		// Check temperature thresholds - find highest temperature with sensor details
		var highestTemp float64
		var highestTempSensor string
		var highestTempLabel string

		// Log temperature sensor data for debugging
		log.Debug().
			Int("temp_sensor_count", len(hardwareStats.Temperature)).
			Str("agent", client.agent.Name).
			Msg("Processing temperature sensors for notifications")

		for _, temp := range hardwareStats.Temperature {
			log.Debug().
				Str("sensor_key", temp.SensorKey).
				Str("label", temp.Label).
				Float64("temperature", temp.Temperature).
				Msg("Temperature sensor data")

			if temp.Temperature > highestTemp {
				highestTemp = temp.Temperature
				highestTempSensor = temp.SensorKey
				highestTempLabel = temp.Label
			}
		}

		if highestTemp > 0 && now.Sub(client.lastTempNotificationTime) > notificationCooldown {
			// Build sensor info for notification
			sensorInfo := highestTempSensor
			if highestTempLabel != "" {
				sensorInfo = highestTempLabel
			}

			// If we still don't have sensor info, use a generic label
			if sensorInfo == "" {
				sensorInfo = "Unknown Sensor"
				log.Warn().
					Str("agent", client.agent.Name).
					Msg("Temperature sensor has no label or key")
			}

			// Log what we're about to send
			log.Info().
				Str("agent", client.agent.Name).
				Str("sensor_key", highestTempSensor).
				Str("sensor_label", highestTempLabel).
				Str("sensor_info", sensorInfo).
				Float64("temperature", highestTemp).
				Msg("Preparing temperature notification")

			// Build agent name with sensor info for temperature notifications
			agentNameWithSensor := fmt.Sprintf("%s|%s", client.agent.Name, sensorInfo)

			log.Info().
				Str("agent_with_sensor", agentNameWithSensor).
				Str("sensor_info", sensorInfo).
				Msg("Sending temperature notification with sensor details")

			// Send notification with sensor info embedded in agent name
			if err := client.notifier.SendAgentNotification(
				agentNameWithSensor,
				database.NotificationEventAgentHighTemp,
				&highestTemp,
			); err != nil {
				log.Error().Err(err).Msg("Failed to send high temperature notification")
			} else {
				client.lastTempNotificationTime = now
				log.Info().
					Str("agent", client.agent.Name).
					Str("sensor", sensorInfo).
					Float64("temperature", highestTemp).
					Msg("Temperature notification sent successfully with sensor details")
			}
		}
	}

	return nil
}

// collectHistoricalSnapshots collects bandwidth historical data for all connected agents
func (s *Service) collectHistoricalSnapshots() {
	s.clientsMu.RLock()
	agents := make([]*Client, 0, len(s.clients))
	for _, client := range s.clients {
		if connected, _ := client.IsConnected(); connected {
			agents = append(agents, client)
		}
	}
	s.clientsMu.RUnlock()

	for _, client := range agents {
		go s.fetchAndStoreHistoricalData(client)
	}
}

// fetchAndStoreHistoricalData fetches and stores vnstat historical data from an agent
func (s *Service) fetchAndStoreHistoricalData(client *Client) {
	baseURL := strings.TrimSuffix(client.agent.URL, "/events?stream=live-data")
	historicalURL := baseURL + "/export/historical"

	req, err := http.NewRequestWithContext(client.ctx, "GET", historicalURL, nil)
	if err != nil {
		log.Error().Err(err).Int64("agent_id", client.agent.ID).Msg("Failed to create historical request")
		return
	}

	if client.agent.APIKey != nil && *client.agent.APIKey != "" {
		req.Header.Set("X-API-Key", *client.agent.APIKey)
	}

	httpClient := &http.Client{Timeout: 60 * time.Second}
	resp, err := httpClient.Do(req)
	if err != nil {
		log.Error().Err(err).Int64("agent_id", client.agent.ID).Msg("Failed to fetch historical data")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Error().Int("status", resp.StatusCode).Int64("agent_id", client.agent.ID).Msg("Historical endpoint returned error")
		return
	}

	// Parse the vnstat JSON data
	var vnstatData map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&vnstatData); err != nil {
		log.Error().Err(err).Int64("agent_id", client.agent.ID).Msg("Failed to decode historical data")
		return
	}

	// First, save the complete vnstat data snapshot
	vnstatJSON, err := json.Marshal(vnstatData)
	if err == nil {
		snapshot := &types.MonitorHistoricalSnapshot{
			AgentID:       client.agent.ID,
			InterfaceName: "all", // Indicates this is the full vnstat data
			PeriodType:    "vnstat",
			DataJSON:      string(vnstatJSON),
		}
		if err := s.db.SaveMonitorHistoricalSnapshot(client.ctx, client.agent.ID, snapshot); err != nil {
			log.Warn().Err(err).Msg("Failed to save full vnstat snapshot")
		} else {
			log.Debug().Int64("agent_id", client.agent.ID).Msg("Saved full vnstat snapshot")
		}
	}

	// Extract interfaces data
	interfaces, ok := vnstatData["interfaces"].([]interface{})
	if !ok {
		log.Warn().Int64("agent_id", client.agent.ID).Msg("No interfaces found in historical data")
		return
	}

	// Process each interface
	for _, ifaceData := range interfaces {
		iface, ok := ifaceData.(map[string]interface{})
		if !ok {
			continue
		}

		ifaceName, _ := iface["name"].(string)
		if ifaceName == "" {
			continue
		}

		// Extract and store different period types
		traffic, ok := iface["traffic"].(map[string]interface{})
		if !ok {
			continue
		}

		// Store hourly data
		if hourly, ok := traffic["hour"].([]interface{}); ok && len(hourly) > 0 {
			hourlyJSON, err := json.Marshal(hourly)
			if err == nil {
				snapshot := &types.MonitorHistoricalSnapshot{
					AgentID:       client.agent.ID,
					InterfaceName: ifaceName,
					PeriodType:    "hourly",
					DataJSON:      string(hourlyJSON),
				}
				if err := s.db.SaveMonitorHistoricalSnapshot(client.ctx, client.agent.ID, snapshot); err != nil {
					log.Warn().Err(err).Str("period", "hourly").Msg("Failed to save historical snapshot")
				}
			}
		}

		// Store daily data
		if daily, ok := traffic["day"].([]interface{}); ok && len(daily) > 0 {
			dailyJSON, err := json.Marshal(daily)
			if err == nil {
				snapshot := &types.MonitorHistoricalSnapshot{
					AgentID:       client.agent.ID,
					InterfaceName: ifaceName,
					PeriodType:    "daily",
					DataJSON:      string(dailyJSON),
				}
				if err := s.db.SaveMonitorHistoricalSnapshot(client.ctx, client.agent.ID, snapshot); err != nil {
					log.Warn().Err(err).Str("period", "daily").Msg("Failed to save historical snapshot")
				}
			}
		}

		// Store monthly data
		if monthly, ok := traffic["month"].([]interface{}); ok && len(monthly) > 0 {
			monthlyJSON, err := json.Marshal(monthly)
			if err == nil {
				snapshot := &types.MonitorHistoricalSnapshot{
					AgentID:       client.agent.ID,
					InterfaceName: ifaceName,
					PeriodType:    "monthly",
					DataJSON:      string(monthlyJSON),
				}
				if err := s.db.SaveMonitorHistoricalSnapshot(client.ctx, client.agent.ID, snapshot); err != nil {
					log.Warn().Err(err).Str("period", "monthly").Msg("Failed to save historical snapshot")
				}
			}
		}

		// Store total data if available
		if total, ok := traffic["total"].(map[string]interface{}); ok {
			totalJSON, err := json.Marshal(total)
			if err == nil {
				snapshot := &types.MonitorHistoricalSnapshot{
					AgentID:       client.agent.ID,
					InterfaceName: ifaceName,
					PeriodType:    "total",
					DataJSON:      string(totalJSON),
				}
				if err := s.db.SaveMonitorHistoricalSnapshot(client.ctx, client.agent.ID, snapshot); err != nil {
					log.Warn().Err(err).Str("period", "total").Msg("Failed to save total snapshot")
				}
			}
		}

		// Also check for and update total memory if available from vnstat data
		if created, ok := iface["created"].(map[string]interface{}); ok {
			if memory, ok := created["memory"].(float64); ok && memory > 0 {
				// Convert MB to bytes
				totalMemory := int64(memory * 1024 * 1024)
				sysInfo := &types.MonitorSystemInfo{
					AgentID:     client.agent.ID,
					TotalMemory: totalMemory,
				}
				if err := s.db.UpsertMonitorSystemInfo(client.ctx, client.agent.ID, sysInfo); err != nil {
					log.Warn().Err(err).Msg("Failed to update total memory from vnstat")
				}
			}
		}
	}

	log.Debug().Int64("agent_id", client.agent.ID).Msg("Successfully collected historical snapshots")
}

// fetchInitialPeakStats fetches and stores initial peak bandwidth statistics from an agent
func (c *Client) fetchInitialPeakStats() {
	baseURL := strings.TrimSuffix(c.agent.URL, "/events?stream=live-data")
	peaksURL := baseURL + "/stats/peaks"

	req, err := http.NewRequestWithContext(c.ctx, "GET", peaksURL, nil)
	if err != nil {
		log.Warn().Err(err).Int64("agent_id", c.agent.ID).Msg("Failed to create peaks request")
		return
	}

	if c.agent.APIKey != nil && *c.agent.APIKey != "" {
		req.Header.Set("X-API-Key", *c.agent.APIKey)
	}

	httpClient := &http.Client{Timeout: 30 * time.Second}
	resp, err := httpClient.Do(req)
	if err != nil {
		log.Warn().Err(err).Int64("agent_id", c.agent.ID).Msg("Failed to fetch peak stats")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Warn().Int("status", resp.StatusCode).Int64("agent_id", c.agent.ID).Msg("Peak stats endpoint returned error")
		return
	}

	var peakStats struct {
		PeakRx          int    `json:"peak_rx"`
		PeakTx          int    `json:"peak_tx"`
		PeakRxTimestamp string `json:"peak_rx_timestamp"`
		PeakTxTimestamp string `json:"peak_tx_timestamp"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&peakStats); err != nil {
		log.Warn().Err(err).Int64("agent_id", c.agent.ID).Msg("Failed to decode peak stats")
		return
	}

	// Parse timestamps
	rxTime, _ := time.Parse(time.RFC3339, peakStats.PeakRxTimestamp)
	txTime, _ := time.Parse(time.RFC3339, peakStats.PeakTxTimestamp)

	// Update local peak tracking
	c.mu.Lock()
	c.peakRx = int64(peakStats.PeakRx)
	c.peakTx = int64(peakStats.PeakTx)
	c.peakRxTimestamp = rxTime
	c.peakTxTimestamp = txTime
	c.mu.Unlock()

	// Store in database
	stats := &types.MonitorPeakStats{
		AgentID:         c.agent.ID,
		PeakRxBytes:     int64(peakStats.PeakRx),
		PeakTxBytes:     int64(peakStats.PeakTx),
		PeakRxTimestamp: &rxTime,
		PeakTxTimestamp: &txTime,
	}

	if err := c.db.UpsertMonitorPeakStats(context.Background(), c.agent.ID, stats); err != nil {
		log.Warn().Err(err).Int64("agent_id", c.agent.ID).Msg("Failed to store initial peak stats")
	} else {
		log.Debug().Int64("agent_id", c.agent.ID).Msg("Stored initial peak stats")
	}
}

// GetTailscaleStatus returns the Tailscale status from the discovery service
func (s *Service) GetTailscaleStatus() (map[string]interface{}, error) {
	if s.tailscaleDiscovery == nil {
		return map[string]interface{}{
			"enabled": false,
			"status":  "Tailscale discovery not configured",
		}, nil
	}

	return s.tailscaleDiscovery.GetTailscaleStatus()
}
