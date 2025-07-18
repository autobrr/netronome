// Copyright (c) 2024-2025, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/autobrr/netronome/internal/agent"
	"github.com/autobrr/netronome/internal/config"
	"github.com/autobrr/netronome/internal/database"
	"github.com/autobrr/netronome/internal/logger"
	"github.com/autobrr/netronome/internal/notifications"
	"github.com/autobrr/netronome/internal/scheduler"
	"github.com/autobrr/netronome/internal/server"
	"github.com/autobrr/netronome/internal/speedtest"
	"github.com/autobrr/netronome/internal/monitor"
)

var (
	// Build-time variables (set via ldflags)
	version   = "dev"
	buildTime = "unknown"
	commit    = "unknown"

	configPath string
	rootCmd    = &cobra.Command{
		Use:   "netronome",
		Short: "Netronome is a network performance testing and monitoring tool",
		Long: `Netronome is a network performance testing and monitoring tool that helps you 
track and analyze your network performance over time.`,
		CompletionOptions: cobra.CompletionOptions{
			DisableDefaultCmd: true,
		},
	}

	serveCmd = &cobra.Command{
		Use:   "serve",
		Short: "Start the Netronome server",
		RunE:  runServer,
	}

	generateConfigCmd = &cobra.Command{
		Use:   "generate-config",
		Short: "Generate a default configuration file",
		RunE:  generateConfig,
	}

	changePasswordCmd = &cobra.Command{
		Use:   "change-password [username]",
		Short: "Change password for a user",
		Args:  cobra.ExactArgs(1),
		RunE:  changePassword,
	}

	createUserCmd = &cobra.Command{
		Use:   "create-user [username]",
		Short: "Create a new user",
		Args:  cobra.ExactArgs(1),
		RunE:  createUser,
	}

	agentCmd = &cobra.Command{
		Use:   "agent",
		Short: "Start the monitoring agent",
		Long: `Start a monitoring agent that broadcasts bandwidth and system usage data.
This agent can be monitored by a remote Netronome server.

Examples:
  # Start agent with default settings
  netronome agent

  # Start agent with custom port and API key
  netronome agent --port 8300 --api-key mysecretkey

  # Start agent monitoring specific interface
  netronome agent --interface eth0

  # Start agent with Tailscale for secure connectivity
  netronome agent --tailscale

  # Start agent with Tailscale and custom hostname
  netronome agent --tailscale --tailscale-hostname my-server

  # Start agent with Tailscale and auth key for headless deployment
  netronome agent --tailscale --tailscale-auth-key tskey-auth-xxx

  # Use configuration file
  netronome agent --config /etc/netronome/agent.toml`,
		RunE: runAgent,
	}
)

func init() {
	if err := godotenv.Load(); err != nil {
		// no .env file found
	}

	rootCmd.PersistentFlags().StringVar(&configPath, "config", "", "path to config file")

	agentCmd.Flags().StringP("host", "H", "0.0.0.0", "IP address to bind to")
	agentCmd.Flags().IntP("port", "p", 8200, "port to listen on")
	agentCmd.Flags().StringP("interface", "i", "", "network interface to monitor (empty for all)")
	agentCmd.Flags().StringP("api-key", "k", "", "API key for authentication")
	agentCmd.Flags().StringP("log-level", "l", "", "log level (trace, debug, info, warn, error)")
	agentCmd.Flags().StringSlice("disk-include", []string{}, "additional disk mount points to monitor (e.g., /mnt/storage)")
	agentCmd.Flags().StringSlice("disk-exclude", []string{}, "disk mount points to exclude from monitoring (e.g., /boot)")
	agentCmd.Flags().Bool("tailscale", false, "enable Tailscale for secure connectivity")
	agentCmd.Flags().String("tailscale-hostname", "", "custom Tailscale hostname (default: netronome-agent-<hostname>)")
	agentCmd.Flags().String("tailscale-auth-key", "", "Tailscale auth key for automatic registration")
	agentCmd.Flags().String("tailscale-state-dir", "", "directory for Tailscale state (default: ~/.config/netronome/tsnet)")
	agentCmd.Flags().Bool("tailscale-prefer-host", false, "prefer using host's tailscaled if available")

	rootCmd.AddCommand(serveCmd)
	rootCmd.AddCommand(generateConfigCmd)
	rootCmd.AddCommand(changePasswordCmd)
	rootCmd.AddCommand(createUserCmd)
	rootCmd.AddCommand(agentCmd)
	rootCmd.AddCommand(updateCmd)
	rootCmd.AddCommand(versionCmd)
}

func main() {
	// Initialize version information with build-time values
	SetVersion(version, buildTime, commit)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func generateConfig(cmd *cobra.Command, args []string) error {
	logger.Init(config.LoggingConfig{Level: "info"}, config.ServerConfig{}, false)

	cfg := config.New()

	if configPath == "" {
		homeDir, err := os.UserHomeDir()
		if err == nil {
			configDir := filepath.Join(homeDir, ".config")
			netronomeDir := filepath.Join(configDir, config.AppName)
			if err := os.MkdirAll(netronomeDir, 0755); err == nil {
				configPath = filepath.Join(netronomeDir, "config.toml")
			} else {
				// Fall back to platform-specific user config dir
				if configDir, err := os.UserConfigDir(); err == nil {
					netronomeDir := filepath.Join(configDir, config.AppName)
					if err := os.MkdirAll(netronomeDir, 0755); err == nil {
						configPath = filepath.Join(netronomeDir, "config.toml")
					} else {
						log.Warn().
							Err(err).
							Msg("could not create config directory, falling back to working directory")
						configPath = "config.toml"
					}
				} else {
					log.Warn().
						Err(err).
						Msg("could not determine user config directory, falling back to working directory")
					configPath = "config.toml"
				}
			}
		}
	}

	if _, err := os.Stat(configPath); err == nil {
		return fmt.Errorf("config file already exists at %s", configPath)
	}

	f, err := os.Create(configPath)
	if err != nil {
		return fmt.Errorf("failed to create config file: %w", err)
	}
	defer f.Close()

	if err := cfg.WriteToml(f); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	log.Info().Str("path", configPath).Msg("Generated default config file")
	return nil
}

func runServer(cmd *cobra.Command, args []string) error {
	// initialize logger with default settings first (silent)
	logger.Init(config.LoggingConfig{Level: "info"}, config.ServerConfig{}, true)

	// ensure config exists
	configPath, err := config.EnsureConfig(configPath)
	if err != nil {
		return fmt.Errorf("failed to ensure config exists: %w", err)
	}

	// load configuration
	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// reinitialize logger with loaded config (not silent)
	logger.Init(cfg.Logging, cfg.Server, false)

	// initialize database
	db := database.New(cfg.Database)
	if err := db.InitializeTables(context.Background()); err != nil {
		return fmt.Errorf("failed to initialize database tables: %w", err)
	}

	// create notifier
	notifier := notifications.NewNotifier(&cfg.Notifications)

	// create server handler with all services
	speedtestSvc := speedtest.New(db, cfg.SpeedTest, notifier, cfg)

	// Create packet loss service variable
	var packetLossService *speedtest.PacketLossService

	// create packet loss service if enabled
	if cfg.PacketLoss.Enabled {
		// We'll set the actual broadcaster after creating the server
		packetLossService = speedtest.NewPacketLossService(db, notifier, nil, cfg.PacketLoss.MaxConcurrentMonitors, cfg.PacketLoss.PrivilegedMode)
	}

	// Create monitor service variable
	var monitorService *monitor.Service

	// Now create scheduler with packet loss service
	schedulerSvc := scheduler.New(db, speedtestSvc, packetLossService, notifier)

	// create server handler with packet loss service and monitor service
	serverHandler := server.NewServer(speedtestSvc, db, schedulerSvc, cfg, packetLossService, monitorService)

	speedtestSvc.SetBroadcastUpdate(serverHandler.BroadcastUpdate)
	speedtestSvc.SetBroadcastTracerouteUpdate(serverHandler.BroadcastTracerouteUpdate)

	// Set the broadcaster for packet loss service
	if packetLossService != nil {
		packetLossService.SetBroadcast(serverHandler.BroadcastPacketLossUpdate)
		packetLossService.SetScheduler(schedulerSvc)

		// Don't start monitors here - let the scheduler handle them
	}

	// Create and set monitor service if enabled
	if cfg.Monitor.Enabled {
		// Use Tailscale-enabled service if auto-discovery is enabled
		// This works even if [tailscale] enabled = false (uses host's tailscaled)
		if cfg.Tailscale.Monitor.AutoDiscover {
			monitorService = monitor.NewServiceWithTailscale(db, &cfg.Monitor, &cfg.Tailscale, serverHandler.BroadcastMonitorUpdate)
			if cfg.Tailscale.Enabled {
				log.Info().Msg("Monitor service created with Tailscale discovery support (tsnet)")
			} else {
				log.Info().Msg("Monitor service created with Tailscale discovery support (host tailscaled)")
			}
		} else {
			monitorService = monitor.NewService(db, &cfg.Monitor, serverHandler.BroadcastMonitorUpdate)
		}
		serverHandler.SetMonitorService(monitorService)

		// Start monitor service
		if err := monitorService.Start(); err != nil {
			log.Error().Err(err).Msg("Failed to start monitor service")
		}
	}

	// Initialize server (register routes and static files)
	serverHandler.Initialize()

	serverHandler.StartScheduler(context.Background())

	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	srv := &http.Server{
		Addr:    addr,
		Handler: serverHandler.Router,
	}

	go func() {
		log.Info().Str("addr", addr).Msg("Starting server")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("Failed to start server")
		}
	}()

	// wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info().Msg("Shutting down server...")

	// the context is used to inform the server it has 5 seconds to finish
	// the request it is currently handling
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		return fmt.Errorf("server forced to shutdown: %w", err)
	}

	// Stop monitor service if running
	if monitorService != nil {
		monitorService.Stop()
	}

	// Close database connection to ensure WAL is checkpointed
	if err := db.Close(); err != nil {
		log.Error().Err(err).Msg("Failed to close database")
	}

	log.Info().Msg("Server exiting")
	return nil
}

func changePassword(cmd *cobra.Command, args []string) error {
	logger.Init(config.LoggingConfig{Level: "info"}, config.ServerConfig{}, false)

	configPath, err := config.EnsureConfig(configPath)
	if err != nil {
		return fmt.Errorf("failed to ensure config exists: %w", err)
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	db := database.New(cfg.Database)
	if err := db.InitializeTables(context.Background()); err != nil {
		return fmt.Errorf("failed to initialize database tables: %w", err)
	}
	defer db.Close()

	username := args[0]

	var password []byte

	// check if we're running in a terminal
	if term.IsTerminal(int(syscall.Stdin)) {
		fmt.Print("Enter new password: ")
		password, err = term.ReadPassword(int(syscall.Stdin))
		if err != nil {
			return fmt.Errorf("failed to read password: %w", err)
		}
		fmt.Println()
	} else {
		password, err = io.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("failed to read password from stdin: %w", err)
		}
		password = bytes.TrimSpace(password)
	}

	if len(password) == 0 {
		return fmt.Errorf("password cannot be empty")
	}

	// Validate password
	//if err := auth.ValidatePassword(string(password)); err != nil {
	//	return fmt.Errorf("invalid password: %w", err)
	//}

	if err := db.UpdatePassword(context.Background(), username, string(password)); err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	log.Info().Str("username", username).Msg("Password updated successfully")
	return nil
}

func createUser(cmd *cobra.Command, args []string) error {
	logger.Init(config.LoggingConfig{Level: "info"}, config.ServerConfig{}, false)

	configPath, err := config.EnsureConfig(configPath)
	if err != nil {
		return fmt.Errorf("failed to ensure config exists: %w", err)
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	db := database.New(cfg.Database)
	if err := db.InitializeTables(context.Background()); err != nil {
		return fmt.Errorf("failed to initialize database tables: %w", err)
	}
	defer db.Close()

	username := args[0]

	var password []byte

	// check if we're running in a terminal
	if term.IsTerminal(int(syscall.Stdin)) {
		fmt.Print("Enter password: ")
		password, err = term.ReadPassword(int(syscall.Stdin))
		if err != nil {
			return fmt.Errorf("failed to read password: %w", err)
		}
		fmt.Println()
	} else {
		password, err = io.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("failed to read password from stdin: %w", err)
		}
		password = bytes.TrimSpace(password)
	}

	if len(password) == 0 {
		return fmt.Errorf("password cannot be empty")
	}

	// Validate password
	//if err := auth.ValidatePassword(string(password)); err != nil {
	//	return fmt.Errorf("invalid password: %w", err)
	//}

	user, err := db.CreateUser(context.Background(), username, string(password))
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	log.Info().Str("username", username).Int64("id", user.ID).Msg("User created successfully")
	return nil
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
	tailscalePreferHost, _ := cmd.Flags().GetBool("tailscale-prefer-host")

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
		if cmd.Flags().Changed("tailscale-prefer-host") {
			cfg.Tailscale.PreferHost = tailscalePreferHost
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
