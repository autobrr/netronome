// Copyright (c) 2024-2025, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package agent

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"tailscale.com/tsnet"

	ts "github.com/autobrr/netronome/internal/tailscale"
)

// startWithTailscale starts the agent server with Tailscale
func (a *Agent) startWithTailscale(ctx context.Context) error {
	// Start bandwidth monitoring
	go a.runBandwidthMonitor(ctx)

	// Start broadcaster
	go a.broadcaster(ctx)

	// Configure tsnet
	hostname := a.tailscaleConfig.Hostname
	if hostname == "" {
		// Generate default hostname
		sysHostname, _ := os.Hostname()
		hostname = fmt.Sprintf("netronome-agent-%s", sysHostname)
	}

	// Expand state directory path
	stateDir := a.tailscaleConfig.StateDir
	if strings.HasPrefix(stateDir, "~/") {
		home, _ := os.UserHomeDir()
		stateDir = filepath.Join(home, stateDir[2:])
	}

	// Create state directory if it doesn't exist
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		return fmt.Errorf("failed to create tsnet state directory: %w", err)
	}

	a.tsnetServer = &tsnet.Server{
		Dir:       stateDir,
		Hostname:  hostname,
		AuthKey:   a.tailscaleConfig.AuthKey,
		Ephemeral: a.tailscaleConfig.Ephemeral,
		Logf: func(format string, args ...interface{}) {
			log.Debug().Msgf("[tsnet] "+format, args...)
		},
	}

	// Set control URL if specified
	if a.tailscaleConfig.ControlURL != "" {
		a.tsnetServer.ControlURL = a.tailscaleConfig.ControlURL
	}

	// Start tsnet server
	log.Info().Str("hostname", hostname).Msg("Starting Tailscale node...")
	if err := a.tsnetServer.Start(); err != nil {
		return fmt.Errorf("failed to start tsnet: %w", err)
	}

	// Set up routes
	router := a.setupRoutes()

	// Listen on Tailscale network
	port := a.config.Port
	if a.tailscaleConfig.AgentPort > 0 {
		port = a.tailscaleConfig.AgentPort
	}
	ln, err := a.tsnetServer.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return fmt.Errorf("failed to listen on Tailscale: %w", err)
	}

	server := &http.Server{
		Handler: router,
	}

	// Handle graceful shutdown
	go func() {
		<-ctx.Done()
		log.Info().Msg("Shutting down Tailscale agent server...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Error().Err(err).Msg("Failed to gracefully shutdown server")
		}
		if a.tsnetServer != nil {
			a.tsnetServer.Close()
		}
	}()

	localClient, err := a.tsnetServer.LocalClient()
	if err != nil {
		log.Warn().Err(err).Msg("Failed to get Tailscale local client")
	} else {
		status, err := localClient.Status(ctx)
		if err != nil {
			log.Warn().Err(err).Msg("Failed to get Tailscale status")
		} else if status.Self != nil {
			logEvent := log.Info().
				Str("hostname", hostname).
				Int("port", port)

			if len(status.Self.TailscaleIPs) > 0 {
				logEvent = logEvent.Str("tailscale_ip", status.Self.TailscaleIPs[0].String())
			}

			logEvent.Msg("Monitor SSE agent listening on Tailscale network")

			if a.config.APIKey != "" {
				log.Info().Msg("API key authentication enabled")
			}
		}
	}

	if err := server.Serve(ln); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("failed to serve: %w", err)
	}

	return nil
}

// startWithHostTailscale starts the agent using the host's existing tailscaled
func (a *Agent) startWithHostTailscale(ctx context.Context) error {
	// Start bandwidth monitoring
	go a.runBandwidthMonitor(ctx)
	// Start broadcaster
	go a.broadcaster(ctx)

	// Get host tailscale client
	hostClient, err := ts.GetHostClient()
	if err != nil {
		return fmt.Errorf("failed to connect to host tailscaled: %w", err)
	}

	// Get our tailscale status
	hostname, ips, err := ts.GetSelfInfo(hostClient)
	if err != nil {
		return fmt.Errorf("failed to get Tailscale info: %w", err)
	}

	log.Info().
		Str("hostname", hostname).
		Strs("tailscale_ips", ips).
		Msg("Using host's tailscaled for agent")

	// Start normal server but bind to Tailscale IP
	router := a.setupRoutes()
	server := &http.Server{
		Handler: router,
	}

	// Listen on Tailscale network
	port := a.config.Port
	if a.tailscaleConfig.AgentPort > 0 {
		port = a.tailscaleConfig.AgentPort
	}
	ln, err := ts.ListenOnTailscale(hostClient, port)
	if err != nil {
		// Fallback to all interfaces if we can't bind to specific Tailscale IP
		log.Warn().Err(err).Msg("Failed to bind to Tailscale IP, using all interfaces")
		addr := fmt.Sprintf("%s:%d", a.config.Host, port)
		ln, err = net.Listen("tcp", addr)
		if err != nil {
			return fmt.Errorf("failed to listen: %w", err)
		}
	}

	// Set up graceful shutdown
	go func() {
		<-ctx.Done()
		log.Info().Msg("Shutting down agent server...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Error().Err(err).Msg("Failed to gracefully shutdown server")
		}
	}()

	log.Info().
		Str("address", ln.Addr().String()).
		Int("port", port).
		Msg("Monitor SSE agent listening via host's tailscaled")

	if a.config.APIKey != "" {
		log.Info().Msg("API key authentication enabled")
	}

	if err := server.Serve(ln); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("failed to serve: %w", err)
	}

	return nil
}

// GetTailscaleStatus returns the current Tailscale connection status
func (a *Agent) GetTailscaleStatus() (map[string]interface{}, error) {
	if !a.useTailscale || a.tailscaleConfig == nil {
		return map[string]interface{}{
			"enabled": false,
			"status":  "disabled",
		}, nil
	}

	method, _ := a.tailscaleConfig.GetEffectiveMethod()
	result := map[string]interface{}{
		"enabled": true,
		"method":  method,
	}

	// Handle tsnet mode
	if method == "tsnet" && a.tsnetServer != nil {
		localClient, err := a.tsnetServer.LocalClient()
		if err != nil {
			return nil, fmt.Errorf("failed to get Tailscale local client: %w", err)
		}

		status, err := localClient.Status(context.Background())
		if err != nil {
			return nil, fmt.Errorf("failed to get Tailscale status: %w", err)
		}

		if status.Self != nil {
			// Use the actual Tailscale machine name (DNSName without suffix)
			hostname := status.Self.DNSName
			// Trim the MagicDNS suffix to get just the machine name
			if hostname != "" && strings.Contains(hostname, ".") {
				hostname = strings.Split(hostname, ".")[0]
			}
			// Fallback to HostName if DNSName is empty
			if hostname == "" {
				hostname = status.Self.HostName
			}
			result["hostname"] = hostname

			var ips []string
			for _, ip := range status.Self.TailscaleIPs {
				ips = append(ips, ip.String())
			}
			result["tailscale_ips"] = ips
			result["online"] = status.Self.Online
			result["status"] = "connected"
		} else {
			// Fallback to configured hostname if not connected
			result["hostname"] = a.tsnetServer.Hostname
			result["status"] = "connecting"
		}
	} else if method == "host" {
		// Handle host mode
		hostClient, err := ts.GetHostClient()
		if err != nil {
			result["status"] = "host_unavailable"
			result["error"] = err.Error()
		} else {
			hostname, ips, err := ts.GetSelfInfo(hostClient)
			if err != nil {
				result["status"] = "error"
				result["error"] = err.Error()
			} else {
				result["hostname"] = hostname
				result["tailscale_ips"] = ips
				result["status"] = "connected"
			}
		}
	}

	return result, nil
}
