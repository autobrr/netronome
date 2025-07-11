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

	"github.com/autobrr/netronome/internal/config"
	"github.com/autobrr/netronome/internal/database"
	"github.com/autobrr/netronome/internal/logger"
	"github.com/autobrr/netronome/internal/notifications"
	"github.com/autobrr/netronome/internal/scheduler"
	"github.com/autobrr/netronome/internal/server"
	"github.com/autobrr/netronome/internal/speedtest"
)

var (
	configPath string
	rootCmd    = &cobra.Command{
		Use:   "netronome",
		Short: "Netronome is a network speed testing and monitoring tool",
		Long: `Netronome is a network speed testing and monitoring tool that helps you 
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
)

func init() {
	if err := godotenv.Load(); err != nil {
		// no .env file found
	}

	rootCmd.PersistentFlags().StringVar(&configPath, "config", "", "path to config file")

	rootCmd.AddCommand(serveCmd)
	rootCmd.AddCommand(generateConfigCmd)
	rootCmd.AddCommand(changePasswordCmd)
	rootCmd.AddCommand(createUserCmd)
}

func main() {
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
	schedulerSvc := scheduler.New(db, speedtestSvc, notifier)
	serverHandler := server.NewServer(speedtestSvc, db, schedulerSvc, cfg)

	speedtestSvc.SetBroadcastUpdate(serverHandler.BroadcastUpdate)
	speedtestSvc.SetBroadcastTracerouteUpdate(serverHandler.BroadcastTracerouteUpdate)

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
