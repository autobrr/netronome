// Copyright (c) 2024, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package main

import (
	"context"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"

	"github.com/autobrr/netronome/internal/database"
	"github.com/autobrr/netronome/internal/logger"
	"github.com/autobrr/netronome/internal/scheduler"
	"github.com/autobrr/netronome/internal/server"
	"github.com/autobrr/netronome/internal/speedtest"
	"github.com/autobrr/netronome/internal/types"
	"github.com/autobrr/netronome/web"
)

func main() {
	// Initialize logger
	logger.Init()

	// Set Gin mode to release if not in development
	if os.Getenv("GIN_MODE") != "debug" {
		gin.SetMode(gin.ReleaseMode)
	}

	log.Info().Msg("Starting netronome API server")

	// Initialize database service
	db := database.New()
	defer func() {
		log.Info().Msg("Closing database connection...")
		if err := db.Close(); err != nil {
			log.Printf("Database connection closed with error: %v", err)
		}
	}()

	// Check database health
	healthStatus := db.Health()
	if healthStatus["status"] != "up" {
		log.Fatal().Msgf("Database health check failed: %v", healthStatus["error"])
	}

	// Create speedtest server with the broadcast function first
	speedServer := &speedtest.Server{
		BroadcastUpdate: func(update types.SpeedUpdate) {
			// We'll set this after creating the server handler
		},
	}

	// Create speedtest service with database
	speedtestService := speedtest.New(speedServer, db)

	// Initialize scheduler service
	schedulerSvc := scheduler.New(db, speedtestService)

	// Create a context that will be canceled on shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start the scheduler service
	schedulerSvc.Start(ctx)
	defer schedulerSvc.Stop()

	// Create the server handler with all services
	serverHandler := server.NewServer(
		speedtestService,
		db,
		schedulerSvc,
	)

	// Now set the broadcast function to use the server handler
	speedServer.BroadcastUpdate = serverHandler.BroadcastUpdate

	// Set up static file serving
	web.ServeStatic(serverHandler.Router)

	apiServer := &http.Server{
		Addr:         ":8080",
		Handler:      serverHandler.Router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	log.Info().Msgf("Starting server on %s", apiServer.Addr)
	if err := apiServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal().Msgf("Server error: %v", err)
	}

	log.Info().Msg("Server exiting")
}
