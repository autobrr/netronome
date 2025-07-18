// Copyright (c) 2024-2025, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package monitor

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"tailscale.com/tsnet"

	"github.com/autobrr/netronome/internal/config"
	tailscale "github.com/autobrr/netronome/internal/tailscale"
	"github.com/autobrr/netronome/internal/types"
)

// TailscaleDiscovery handles automatic discovery of Tailscale-connected agents
type TailscaleDiscovery struct {
	config          *config.TailscaleConfig
	tsnetServer     *tsnet.Server
	tailscaleClient tailscale.Client
	service         *Service
	discoveryTicker *time.Ticker
	mode            tailscale.Mode
}

// NewTailscaleDiscovery creates a new Tailscale discovery service
func NewTailscaleDiscovery(cfg *config.TailscaleConfig, service *Service) *TailscaleDiscovery {
	return &TailscaleDiscovery{
		config:  cfg,
		service: service,
	}
}

// Start initializes Tailscale connection and begins discovery
func (td *TailscaleDiscovery) Start(ctx context.Context) error {
	if !td.config.Monitor.AutoDiscover {
		log.Debug().Msg("Tailscale discovery is disabled")
		return nil
	}

	// If Tailscale is not enabled, try to use host's tailscaled
	if !td.config.Enabled {
		log.Info().Msg("Tailscale not enabled, attempting to use host's tailscaled for discovery...")
		hostClient, err := tailscale.GetHostClient()
		if err != nil {
			log.Warn().Err(err).Msg("Failed to connect to host's tailscaled, discovery will not work")
			return nil // Don't fail startup, just disable discovery
		}
		td.tailscaleClient = hostClient
		td.mode = tailscale.ModeHost
		log.Info().Msg("Successfully connected to host's tailscaled for discovery")
		
		// Start periodic discovery
		td.startDiscoveryTimer(ctx)
		return nil
	}

	// Expand state directory path
	stateDir := td.config.StateDir
	if strings.HasPrefix(stateDir, "~/") {
		home, _ := os.UserHomeDir()
		stateDir = filepath.Join(home, stateDir[2:])
	}

	// Configure tsnet for discovery
	hostname := td.config.Hostname
	if hostname == "" {
		sysHostname, _ := os.Hostname()
		hostname = fmt.Sprintf("netronome-server-%s", sysHostname)
	}

	td.tsnetServer = &tsnet.Server{
		Dir:       stateDir,
		Hostname:  hostname,
		AuthKey:   td.config.AuthKey,
		Ephemeral: td.config.Ephemeral,
		Logf: func(format string, args ...interface{}) {
			log.Debug().Msgf("[tsnet-discovery] "+format, args...)
		},
	}

	if td.config.ControlURL != "" {
		td.tsnetServer.ControlURL = td.config.ControlURL
	}

	// Start tsnet
	log.Info().Str("hostname", hostname).Msg("Starting Tailscale discovery service (tsnet)...")
	if err := td.tsnetServer.Start(); err != nil {
		return fmt.Errorf("failed to start tsnet for discovery: %w", err)
	}

	// Get tsnet client
	var err error
	td.tailscaleClient, err = tailscale.GetTsnetClient(td.tsnetServer)
	if err != nil {
		return fmt.Errorf("failed to get tsnet client: %w", err)
	}
	td.mode = tailscale.ModeTsnet

	// Start periodic discovery
	td.startDiscoveryTimer(ctx)
	return nil
}

// startDiscoveryTimer starts the periodic discovery timer
func (td *TailscaleDiscovery) startDiscoveryTimer(ctx context.Context) {
	// Parse discovery interval
	interval := 5 * time.Minute // default
	if td.config.Monitor.DiscoveryInterval != "" {
		if parsed, err := time.ParseDuration(td.config.Monitor.DiscoveryInterval); err == nil {
			interval = parsed
		} else {
			log.Warn().Err(err).Str("interval", td.config.Monitor.DiscoveryInterval).Msg("Failed to parse discovery interval, using default")
		}
	}

	log.Info().Str("interval", interval.String()).Msg("Starting Tailscale discovery timer")
	td.discoveryTicker = time.NewTicker(interval)
	go td.runDiscovery(ctx)
}

// Stop stops the discovery service
func (td *TailscaleDiscovery) Stop() {
	if td.discoveryTicker != nil {
		td.discoveryTicker.Stop()
	}
	if td.tsnetServer != nil {
		td.tsnetServer.Close()
	}
}

// runDiscovery periodically discovers new Tailscale agents
func (td *TailscaleDiscovery) runDiscovery(ctx context.Context) {
	// Do initial discovery immediately
	td.discoverAgents(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-td.discoveryTicker.C:
			td.discoverAgents(ctx)
		}
	}
}

// discoverAgents discovers and registers new Tailscale agents
func (td *TailscaleDiscovery) discoverAgents(ctx context.Context) {
	if td.tailscaleClient == nil {
		log.Error().Msg("Tailscale client not initialized")
		return
	}

	status, err := td.tailscaleClient.Status(ctx)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to get Tailscale status for discovery")
		return
	}

	// Get existing agents from database
	existingAgents, err := td.service.db.GetMonitorAgents(ctx, false)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get existing agents for discovery")
		return
	}

	// Create map of existing agents by URL and Tailscale hostname
	existingMap := make(map[string]bool)
	for _, agent := range existingAgents {
		// Add URL to map
		existingMap[agent.URL] = true
		
		// Also check by Tailscale hostname if it's a Tailscale agent
		if agent.IsTailscale && agent.TailscaleHostname != nil {
			existingMap[*agent.TailscaleHostname] = true
		}
		
		// Extract hostname from URL for comparison
		if strings.Contains(agent.URL, "://") {
			parts := strings.Split(agent.URL, "://")
			if len(parts) > 1 {
				hostParts := strings.Split(parts[1], ":")
				if len(hostParts) > 0 {
					existingMap[hostParts[0]] = true
				}
			}
		}
	}

	// Get discovery port
	discoveryPort := td.config.Monitor.DiscoveryPort
	if discoveryPort == 0 {
		discoveryPort = 8200 // Default port
	}

	// Look for new agents by probing all online peers
	log.Debug().Msg("Tailscale discovery service checking for new agents")
	
	for _, peer := range status.Peer {
		// Skip if not online
		if !peer.Online {
			continue
		}

		// Skip if already exists (check by hostname and potential URL)
		if existingMap[peer.HostName] {
			continue
		}
		
		// Also check if the full URL already exists
		potentialURL := fmt.Sprintf("http://%s:%d", peer.HostName, discoveryPort)
		if existingMap[potentialURL] {
			continue
		}

		// Try to identify if this is a Netronome agent
		infoURL := fmt.Sprintf("http://%s:%d/netronome/info", peer.HostName, discoveryPort)
		client := &http.Client{Timeout: 3 * time.Second}
		
		resp, err := client.Get(infoURL)
		if err != nil {
			// Not a Netronome agent or not reachable
			continue
		}
		defer resp.Body.Close()
		
		if resp.StatusCode != http.StatusOK {
			continue
		}
		
		// Parse the response to verify it's a Netronome agent
		var agentInfo struct {
			Type     string `json:"type"`
			Version  string `json:"version"`
			Hostname string `json:"hostname"`
		}
		
		if err := json.NewDecoder(resp.Body).Decode(&agentInfo); err != nil {
			continue
		}
		
		if agentInfo.Type != "netronome-agent" {
			continue
		}

		// This is a valid Netronome agent!
		log.Info().
			Str("hostname", peer.HostName).
			Int("port", discoveryPort).
			Str("version", agentInfo.Version).
			Msg("Discovered new Netronome agent")

		// Create agent URL with SSE endpoint
		agentURL := potentialURL
		if !strings.HasSuffix(agentURL, "/events?stream=live-data") {
			if strings.HasSuffix(agentURL, "/") {
				agentURL = agentURL + "events?stream=live-data"
			} else {
				agentURL = agentURL + "/events?stream=live-data"
			}
		}

		// Create new agent entry
		emptyString := ""
		now := time.Now()
		newAgent := &types.MonitorAgent{
			Name:              peer.HostName,
			URL:               agentURL,
			APIKey:            &emptyString, // User will need to set this manually if required
			Enabled:           true, // Auto-discovered agents start enabled
			IsTailscale:       true,
			TailscaleHostname: &peer.HostName,
			DiscoveredAt:      &now,
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
		}

		// Save to database
		createdAgent, err := td.service.db.CreateMonitorAgent(ctx, newAgent)
		if err != nil {
			log.Error().
				Err(err).
				Str("hostname", peer.HostName).
				Msg("Failed to save discovered agent")
		} else {
			log.Info().
				Str("name", newAgent.Name).
				Str("url", newAgent.URL).
				Msg("Added discovered Tailscale agent")
			
			// Start monitoring the new agent immediately
			if err := td.service.StartAgent(createdAgent.ID); err != nil {
				log.Error().
					Err(err).
					Int64("agent_id", createdAgent.ID).
					Msg("Failed to start monitoring discovered agent")
			} else {
				log.Info().
					Int64("agent_id", createdAgent.ID).
					Str("name", createdAgent.Name).
					Msg("Started monitoring discovered agent")
			}
			
			// Broadcast update
			td.service.broadcastFunc(types.MonitorUpdate{
				Type: types.MonitorUpdateTypeAgentDiscovered,
				Data: map[string]interface{}{
					"agent": createdAgent,
				},
			})
		}
	}
}

// GetTailscaleStatus returns the current Tailscale status for the discovery service
func (td *TailscaleDiscovery) GetTailscaleStatus() (map[string]interface{}, error) {
	if td.tsnetServer == nil {
		return map[string]interface{}{
			"enabled": false,
		}, nil
	}

	localClient, err := td.tsnetServer.LocalClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get Tailscale local client: %w", err)
	}

	status, err := localClient.Status(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to get Tailscale status: %w", err)
	}

	result := map[string]interface{}{
		"enabled":  true,
		"hostname": td.tsnetServer.Hostname,
	}

	if status.Self != nil {
		var ips []string
		for _, ip := range status.Self.TailscaleIPs {
			ips = append(ips, ip.String())
		}
		result["tailscale_ips"] = ips
		result["online"] = status.Self.Online
		result["magic_dns_suffix"] = status.MagicDNSSuffix
		
		// Count discovered agents
		discoveredCount := 0
		for _, peer := range status.Peer {
			if strings.HasPrefix(peer.HostName, td.config.Monitor.DiscoveryPrefix) && peer.Online {
				discoveredCount++
			}
		}
		result["discovered_agents"] = discoveredCount
	}

	return result, nil
}