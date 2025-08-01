// Copyright (c) 2024-2025, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/autobrr/netronome/internal/agent"
	"github.com/autobrr/netronome/internal/config"
	"github.com/autobrr/netronome/internal/logger"
)

var (
	// Build-time variables (set via ldflags)
	version   = "dev"
	buildTime = "unknown"
	commit    = "unknown"

	configPath string
	rootCmd    = &cobra.Command{
		Use:   "netronome-agent",
		Short: "Netronome Agent - Lightweight system monitoring agent",
		Long: `Netronome Agent is a lightweight system monitoring agent that collects and broadcasts
bandwidth and system usage data. It can be monitored by a remote Netronome server.`,
		CompletionOptions: cobra.CompletionOptions{
			DisableDefaultCmd: true,
		},
		RunE: runAgent,
	}

	versionCmd = &cobra.Command{
		Use:   "version",
		Short: "Show version information",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Printf("netronome-agent version %s\n", version)
			fmt.Printf("Built on %s\n", buildTime)
			fmt.Printf("Commit %s\n", commit)
			return nil
		},
	}
)

func init() {
	if err := godotenv.Load(); err != nil {
		// no .env file found
	}

	rootCmd.PersistentFlags().StringVar(&configPath, "config", "", "path to config file")

	rootCmd.Flags().StringP("host", "H", "0.0.0.0", "IP address to bind to")
	rootCmd.Flags().IntP("port", "p", 8200, "port to listen on")
	rootCmd.Flags().StringP("interface", "i", "", "network interface to monitor (empty for all)")
	rootCmd.Flags().StringP("api-key", "k", "", "API key for authentication")
	rootCmd.Flags().StringP("log-level", "l", "", "log level (trace, debug, info, warn, error)")
	rootCmd.Flags().StringSlice("disk-include", []string{}, "additional disk mount points to monitor (e.g., /mnt/storage)")
	rootCmd.Flags().StringSlice("disk-exclude", []string{}, "disk mount points to exclude from monitoring (e.g., /boot)")
	rootCmd.Flags().Bool("tailscale", false, "enable Tailscale for secure connectivity")
	rootCmd.Flags().String("tailscale-hostname", "", "custom Tailscale hostname (default: netronome-agent-<hostname>)")
	rootCmd.Flags().String("tailscale-auth-key", "", "Tailscale auth key for automatic registration")
	rootCmd.Flags().String("tailscale-state-dir", "", "directory for Tailscale state (default: ~/.config/netronome/tsnet)")
	rootCmd.Flags().String("tailscale-method", "auto", "Tailscale method: auto, host, or tsnet (default: auto)")

	rootCmd.AddCommand(versionCmd)
}

func main() {
	// Initialize version information with build-time values
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runAgent(cmd *cobra.Command, args []string) error {
	host, _ := cmd.Flags().GetString("host")
	port, _ := cmd.Flags().GetInt("port")
	iface, _ := cmd.Flags().GetString("interface")
	apiKey, _ := cmd.Flags().GetString("api-key")
	logLevel, _ := cmd.Flags().GetString("log-level")
	diskIncludes, _ := cmd.Flags().GetStringSlice("disk-include")
	diskExcludes, _ := cmd.Flags().GetStringSlice("disk-exclude")

	// Load config if provided
	var cfg *config.Config
	if configPath != "" {
		var err error
		cfg, err = config.Load(configPath)
		if err != nil {
			// Initialize logger with default settings if config load fails
			logger.Init(config.LoggingConfig{Level: "info"}, config.ServerConfig{}, false)
			log.Warn().Err(err).Msg("Failed to load config, using defaults")
			cfg = config.New()
		}
	} else {
		cfg = config.New()
	}

	// Override log level from command line flag if provided
	if cmd.Flags().Changed("log-level") && logLevel != "" {
		cfg.Logging.Level = logLevel
	}

	// Initialize logger with config settings
	logger.Init(cfg.Logging, cfg.Server, false)

	// Override with command line flags
	if cmd.Flags().Changed("host") {
		cfg.Agent.Host = host
	}
	if cmd.Flags().Changed("port") {
		cfg.Agent.Port = port
	}
	if cmd.Flags().Changed("interface") {
		cfg.Agent.Interface = iface
	}
	if cmd.Flags().Changed("api-key") {
		cfg.Agent.APIKey = apiKey
	}
	if cmd.Flags().Changed("disk-include") {
		cfg.Agent.DiskIncludes = diskIncludes
	}
	if cmd.Flags().Changed("disk-exclude") {
		cfg.Agent.DiskExcludes = diskExcludes
	}

	// Handle Tailscale flags
	useTailscale, _ := cmd.Flags().GetBool("tailscale")
	tailscaleHostname, _ := cmd.Flags().GetString("tailscale-hostname")
	tailscaleAuthKey, _ := cmd.Flags().GetString("tailscale-auth-key")
	tailscaleStateDir, _ := cmd.Flags().GetString("tailscale-state-dir")
	tailscaleMethod, _ := cmd.Flags().GetString("tailscale-method")

	// Create agent service
	var agentService *agent.Agent
	if useTailscale || cfg.Tailscale.Agent.Enabled {
		// Override Tailscale config with command line flags
		if cmd.Flags().Changed("tailscale") {
			cfg.Tailscale.Enabled = useTailscale
			cfg.Tailscale.Agent.Enabled = useTailscale
		}
		if cmd.Flags().Changed("tailscale-hostname") {
			cfg.Tailscale.Hostname = tailscaleHostname
		}
		if cmd.Flags().Changed("tailscale-auth-key") {
			cfg.Tailscale.AuthKey = tailscaleAuthKey
		}
		if cmd.Flags().Changed("tailscale-state-dir") {
			cfg.Tailscale.StateDir = tailscaleStateDir
		}
		if cmd.Flags().Changed("tailscale-method") {
			cfg.Tailscale.Method = tailscaleMethod
		}

		agentService = agent.NewWithTailscale(&cfg.Agent, &cfg.Tailscale)
		log.Info().Msg("Starting agent with Tailscale support")
	} else {
		agentService = agent.New(&cfg.Agent)
	}

	// Setup signal handling
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle interrupt signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		log.Info().Msg("Received interrupt signal")
		cancel()
	}()

	// Start the agent
	return agentService.Start(ctx)
}