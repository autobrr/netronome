// Copyright (c) 2024-2025, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package monitor

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"tailscale.com/tsnet"

	"github.com/autobrr/netronome/internal/config"
	"github.com/autobrr/netronome/internal/types"
)

// TailscaleDiscovery handles automatic discovery of Tailscale-connected agents
type TailscaleDiscovery struct {
	config       *config.TailscaleConfig
	tsnetServer  *tsnet.Server
	service      *Service
	discoveryTicker *time.Ticker
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
	if !td.config.Enabled || !td.config.Monitor.AutoDiscover {
		log.Debug().Msg("Tailscale discovery is disabled")
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
	log.Info().Str("hostname", hostname).Msg("Starting Tailscale discovery service...")
	if err := td.tsnetServer.Start(); err != nil {
		return fmt.Errorf("failed to start tsnet for discovery: %w", err)
	}

	// Start periodic discovery
	td.discoveryTicker = time.NewTicker(30 * time.Second)
	go td.runDiscovery(ctx)

	return nil
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
	localClient, err := td.tsnetServer.LocalClient()
	if err != nil {
		log.Warn().Err(err).Msg("Failed to get Tailscale local client for discovery")
		return
	}

	status, err := localClient.Status(ctx)
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

	// Create map of existing Tailscale hostnames
	existingMap := make(map[string]*types.MonitorAgent)
	for _, agent := range existingAgents {
		// Check if URL contains a Tailscale hostname
		if strings.Contains(agent.URL, ".ts.net") || strings.Contains(agent.URL, status.MagicDNSSuffix) {
			parts := strings.Split(agent.URL, "://")
			if len(parts) > 1 {
				hostParts := strings.Split(parts[1], ":")
				if len(hostParts) > 0 {
					existingMap[hostParts[0]] = agent
				}
			}
		}
	}

	// Look for new agents
	prefix := td.config.Monitor.DiscoveryPrefix
	for _, peer := range status.Peer {
		if !strings.HasPrefix(peer.HostName, prefix) {
			continue
		}

		// Skip if already exists
		if _, exists := existingMap[peer.HostName]; exists {
			continue
		}

		// Skip if not online
		if !peer.Online {
			continue
		}

		// Create new agent
		tailscaleIP := ""
		if len(peer.TailscaleIPs) > 0 {
			tailscaleIP = peer.TailscaleIPs[0].String()
		}

		emptyString := ""
		now := time.Now()
		newAgent := &types.MonitorAgent{
			Name:              peer.HostName,
			URL:               fmt.Sprintf("http://%s:8200", peer.HostName),
			APIKey:            &emptyString, // User will need to set this manually if required
			Enabled:           false, // Start disabled, user can enable
			IsTailscale:       true,
			TailscaleHostname: &peer.HostName,
			DiscoveredAt:      &now,
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
		}

		// Try to get more info from the agent
		if tailscaleIP != "" {
			// Optionally try to connect to get system info
			testURL := fmt.Sprintf("http://%s:8200/", tailscaleIP)
			client := &http.Client{Timeout: 5 * time.Second}
			if resp, err := client.Get(testURL); err == nil {
				resp.Body.Close()
				if resp.StatusCode == http.StatusOK {
					// Agent is reachable, use the hostname URL
					log.Info().
						Str("hostname", peer.HostName).
						Str("tailscale_ip", tailscaleIP).
						Msg("Discovered new Tailscale agent")
				}
			}
		}

		// Save to database
		if _, err := td.service.db.CreateMonitorAgent(ctx, newAgent); err != nil {
			log.Error().
				Err(err).
				Str("hostname", peer.HostName).
				Msg("Failed to save discovered agent")
		} else {
			log.Info().
				Str("name", newAgent.Name).
				Str("url", newAgent.URL).
				Msg("Added discovered Tailscale agent (disabled by default)")
			
			// Broadcast update
			td.service.broadcastFunc(types.MonitorUpdate{
				Type: types.MonitorUpdateTypeAgentDiscovered,
				Data: map[string]interface{}{
					"agent": newAgent,
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