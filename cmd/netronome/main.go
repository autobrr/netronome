// Copyright (c) 2024, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/autobrr/netronome/internal/config"
	"github.com/autobrr/netronome/internal/database"
	"github.com/autobrr/netronome/internal/logger"
	"github.com/autobrr/netronome/internal/scheduler"
	"github.com/autobrr/netronome/internal/server"
	"github.com/autobrr/netronome/internal/speedtest"
	"github.com/autobrr/netronome/internal/types"
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
)

func init() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Debug().Err(err).Msg("No .env file found")
	}

	rootCmd.PersistentFlags().StringVar(&configPath, "config", config.DefaultConfigPath, "path to config file")

	rootCmd.AddCommand(serveCmd)
	rootCmd.AddCommand(generateConfigCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func generateConfig(cmd *cobra.Command, args []string) error {
	cfg := config.New()

	// Check if file already exists
	if _, err := os.Stat(configPath); err == nil {
		return fmt.Errorf("config file already exists at %s", configPath)
	}

	// Create config file
	f, err := os.Create(configPath)
	if err != nil {
		return fmt.Errorf("failed to create config file: %w", err)
	}
	defer f.Close()

	// Write default config
	if err := cfg.WriteToml(f); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	log.Info().Str("path", configPath).Msg("Generated default config file")
	return nil
}

func runServer(cmd *cobra.Command, args []string) error {
	// Load configuration
	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Initialize logger with config
	logger.Init(cfg.Logging, cfg.Server)

	// Initialize database
	db := database.New(cfg.Database)
	if err := db.InitializeTables(context.Background()); err != nil {
		return fmt.Errorf("failed to initialize database tables: %w", err)
	}

	// Create speedtest server with the broadcast function first
	speedServer := &speedtest.Server{
		BroadcastUpdate: func(update types.SpeedUpdate) {
			// We'll set this after creating the server handler
		},
	}

	// Create server handler with all services
	serverHandler := server.NewServer(speedtest.New(speedServer, db, cfg.SpeedTest), db, scheduler.New(db, speedtest.New(speedServer, db, cfg.SpeedTest)), cfg)

	// Now set the broadcast function to use the server's broadcast method
	speedServer.BroadcastUpdate = serverHandler.BroadcastUpdate

	// Create HTTP server
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	srv := &http.Server{
		Addr:    addr,
		Handler: serverHandler.Router,
	}

	// Start server
	go func() {
		log.Info().Str("addr", addr).Msg("Starting server")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("Failed to start server")
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info().Msg("Shutting down server...")

	// The context is used to inform the server it has 5 seconds to finish
	// the request it is currently handling
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		return fmt.Errorf("server forced to shutdown: %w", err)
	}

	log.Info().Msg("Server exiting")
	return nil
}
