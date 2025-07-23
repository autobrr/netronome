// Copyright (c) 2024-2025, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package agent

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/autobrr/netronome/internal/config"
)

// New creates a new Agent instance
func New(cfg *config.AgentConfig) *Agent {
	return &Agent{
		config:      cfg,
		clients:     make(map[chan string]bool),
		monitorData: make(chan string, 100),
	}
}

// NewWithTailscale creates a new Agent instance with Tailscale support
func NewWithTailscale(cfg *config.AgentConfig, tsCfg *config.TailscaleConfig) *Agent {
	return &Agent{
		config:          cfg,
		tailscaleConfig: tsCfg,
		clients:         make(map[chan string]bool),
		monitorData:     make(chan string, 100),
		useTailscale:    tsCfg != nil && tsCfg.IsAgentMode(),
	}
}

// Start starts the agent server
func (a *Agent) Start(ctx context.Context) error {
	// If Tailscale is enabled, determine method
	if a.useTailscale && a.tailscaleConfig != nil {
		method, err := a.tailscaleConfig.GetEffectiveMethod()
		if err != nil {
			return fmt.Errorf("failed to determine Tailscale method: %w", err)
		}

		switch method {
		case "host":
			log.Info().Msg("Using host's tailscaled...")
			return a.startWithHostTailscale(ctx)
		case "tsnet":
			log.Info().Msg("Using embedded tsnet...")
			return a.startWithTailscale(ctx)
		default:
			return fmt.Errorf("unexpected Tailscale method: %s", method)
		}
	}

	// Start bandwidth monitoring
	go a.runBandwidthMonitor(ctx)

	// Start broadcaster
	go a.broadcaster(ctx)

	// Set up routes
	router := a.setupRoutes()

	// Start HTTP server
	addr := fmt.Sprintf("%s:%d", a.config.Host, a.config.Port)
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

	if a.config.APIKey != "" {
		log.Info().Str("addr", addr).Msg("Starting monitor SSE agent with API key authentication")
	} else {
		log.Info().Str("addr", addr).Msg("Starting monitor SSE agent without authentication")
	}

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("failed to start server: %w", err)
	}

	return nil
}