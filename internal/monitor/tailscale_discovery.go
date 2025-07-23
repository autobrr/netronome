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
	"tailscale.com/ipn/ipnstate"
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
	if !td.config.IsServerDiscoveryMode() {
		log.Debug().Msg("Tailscale discovery is disabled")
		return nil
	}

	// Determine the effective method
	method, err := td.config.GetEffectiveMethod()
	if err != nil {
		return fmt.Errorf("failed to determine Tailscale method: %w", err)
	}

	switch method {
	case "host":
		return td.startHostMode(ctx)
	case "tsnet":
		return td.startTsnetMode(ctx)
	default:
		return fmt.Errorf("unknown Tailscale method: %s", method)
	}
}

// startHostMode initializes discovery using host's tailscaled
func (td *TailscaleDiscovery) startHostMode(ctx context.Context) error {
	log.Info().Msg("Using host's tailscaled for discovery...")
	
	hostClient, err := tailscale.GetHostClient()
	if err != nil {
		log.Warn().Err(err).Msg("Failed to connect to host's tailscaled, discovery will not work")
		return nil // Don't fail startup, just disable discovery
	}
	
	td.tailscaleClient = hostClient
	td.mode = tailscale.ModeHost
	log.Info().Msg("Successfully connected to host's tailscaled for discovery")
	
	td.startDiscoveryTimer(ctx)
	return nil
}

// startTsnetMode initializes discovery using embedded tsnet
func (td *TailscaleDiscovery) startTsnetMode(ctx context.Context) error {
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
	client, err := tailscale.GetTsnetClient(td.tsnetServer)
	if err != nil {
		return fmt.Errorf("failed to get tsnet client: %w", err)
	}
	
	td.tailscaleClient = client
	td.mode = tailscale.ModeTsnet
	
	td.startDiscoveryTimer(ctx)
	return nil
}

// startDiscoveryTimer starts the periodic discovery timer
func (td *TailscaleDiscovery) startDiscoveryTimer(ctx context.Context) {
	// Parse discovery interval
	interval := 5 * time.Minute // default
	if td.config.DiscoveryInterval != "" {
		if parsed, err := time.ParseDuration(td.config.DiscoveryInterval); err == nil {
			interval = parsed
		} else {
			log.Warn().Err(err).Str("interval", td.config.DiscoveryInterval).Msg("Failed to parse discovery interval, using default")
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

	// Create map of existing Tailscale agents by hostname
	existingTailscaleAgents := make(map[string]bool)
	for _, agent := range existingAgents {
		if agent.IsTailscale && agent.TailscaleHostname != nil {
			existingTailscaleAgents[*agent.TailscaleHostname] = true
		}
	}

	// Get discovery port
	discoveryPort := td.config.DiscoveryPort
	if discoveryPort == 0 {
		discoveryPort = 8200 // Default port
	}

	log.Debug().Msg("Tailscale discovery service checking for new agents")

	// Build list of all peers to check (including self)
	peersToCheck := td.buildPeersList(status)

	// Check each peer
	client := &http.Client{Timeout: 3 * time.Second}
	
	for _, peer := range peersToCheck {
		// Skip if already exists
		if existingTailscaleAgents[peer.HostName] {
			continue
		}

		// Try to discover and add the agent
		td.tryDiscoverAgent(ctx, client, peer, discoveryPort)
	}
}

// buildPeersList builds a list of all peers to check, including self
func (td *TailscaleDiscovery) buildPeersList(status *ipnstate.Status) []struct {
	HostName string
	DNSName  string
	Online   bool
} {
	var peersToCheck []struct {
		HostName string
		DNSName  string
		Online   bool
	}

	// Add self if online
	if status.Self != nil && status.Self.Online {
		hostname := td.extractHostname(status.Self.DNSName, status.Self.HostName)
		peersToCheck = append(peersToCheck, struct {
			HostName string
			DNSName  string
			Online   bool
		}{
			HostName: hostname,
			DNSName:  status.Self.DNSName,
			Online:   true,
		})
	}

	// Add all online peers
	for _, peer := range status.Peer {
		if !peer.Online {
			continue
		}
		
		hostname := td.extractHostname(peer.DNSName, peer.HostName)
		peersToCheck = append(peersToCheck, struct {
			HostName string
			DNSName  string
			Online   bool
		}{
			HostName: hostname,
			DNSName:  peer.DNSName,
			Online:   true,
		})
	}
	
	return peersToCheck
}

// extractHostname extracts the machine name from DNS name, falling back to hostname
func (td *TailscaleDiscovery) extractHostname(dnsName, hostName string) string {
	if dnsName == "" {
		return hostName
	}
	
	// Trim the MagicDNS suffix to get just the machine name
	if idx := strings.Index(dnsName, "."); idx > 0 {
		return dnsName[:idx]
	}
	
	return dnsName
}

// tryDiscoverAgent attempts to discover and add a single agent
func (td *TailscaleDiscovery) tryDiscoverAgent(ctx context.Context, client *http.Client, peer struct {
	HostName string
	DNSName  string
	Online   bool
}, discoveryPort int) {
	// Check if this is a Netronome agent with Tailscale enabled
	infoURL := fmt.Sprintf("http://%s:%d/netronome/info", peer.HostName, discoveryPort)
	
	resp, err := client.Get(infoURL)
	if err != nil {
		// Not a Netronome agent or not reachable
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return
	}

	// Parse the response
	var agentInfo struct {
		Type           string `json:"type"`
		Version        string `json:"version"`
		UsingTailscale bool   `json:"using_tailscale"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&agentInfo); err != nil {
		return
	}

	// Only add if it's a Netronome agent with Tailscale enabled
	if agentInfo.Type != "netronome-agent" || !agentInfo.UsingTailscale {
		return
	}

	// This is a valid Tailscale-enabled Netronome agent!
	log.Info().
		Str("hostname", peer.HostName).
		Int("port", discoveryPort).
		Str("version", agentInfo.Version).
		Msg("Discovered new Tailscale-enabled Netronome agent")

	// Create and save the agent
	td.createAndSaveAgent(ctx, peer, discoveryPort)
}

// createAndSaveAgent creates a new agent entry and saves it to the database
func (td *TailscaleDiscovery) createAndSaveAgent(ctx context.Context, peer struct {
	HostName string
	DNSName  string
	Online   bool
}, discoveryPort int) {
	// Create agent URL with SSE endpoint using short hostname
	agentURL := fmt.Sprintf("http://%s:%d/events?stream=live-data", peer.HostName, discoveryPort)

	// Create new agent entry
	emptyString := ""
	now := time.Now()
	newAgent := &types.MonitorAgent{
		Name:              peer.HostName,
		URL:               agentURL,
		APIKey:            &emptyString, // User will need to set this manually if required
		Enabled:           true,         // Auto-discovered agents start enabled
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
		return
	}

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
		return
	}
	
	log.Info().
		Int64("agent_id", createdAgent.ID).
		Str("name", createdAgent.Name).
		Msg("Started monitoring discovered agent")

	// Broadcast update
	td.service.broadcastFunc(types.MonitorUpdate{
		Type: types.MonitorUpdateTypeAgentDiscovered,
		Data: map[string]interface{}{
			"agent": createdAgent,
		},
	})
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

	if status.Self == nil {
		return result, nil
	}

	// Add self information
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

	return result, nil
}
