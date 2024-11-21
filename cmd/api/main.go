package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"speedtrackerr/internal/database"
	"speedtrackerr/internal/scheduler"
	"speedtrackerr/internal/server"
	"speedtrackerr/internal/speedtest"
	"speedtrackerr/internal/types"
)

func gracefulShutdown(
	apiServer *http.Server,
	db database.Service,
	scheduler scheduler.Service,
	schedulerCancel context.CancelFunc,
	done chan bool,
) {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	<-ctx.Done()

	log.Println("shutting down gracefully, press Ctrl+C again to force")

	// Cancel scheduler context
	schedulerCancel()

	// Stop the scheduler
	scheduler.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := apiServer.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown with error: %v", err)
	}

	if err := db.Close(); err != nil {
		log.Printf("Database connection closed with error: %v", err)
	}

	log.Println("Server exiting")
	done <- true
}

func main() {
	// Set Gin mode to release if not in development
	if os.Getenv("GIN_MODE") != "debug" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Initialize database service
	db := database.New()
	defer db.Close()

	// Check database health
	healthStatus := db.Health()
	if healthStatus["status"] != "up" {
		log.Fatalf("Database health check failed: %v", healthStatus["error"])
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
	schedulerService := scheduler.New(db, speedtestService)

	// Create context for the scheduler
	schedulerCtx, schedulerCancel := context.WithCancel(context.Background())
	defer schedulerCancel()

	// Start the scheduler in a goroutine
	go schedulerService.Start(schedulerCtx)

	// Create the server handler with all services
	serverHandler := server.NewServer(
		speedtestService,
		db,
		schedulerService,
	)

	// Now set the broadcast function to use the server handler
	speedServer.BroadcastUpdate = serverHandler.BroadcastUpdate

	apiServer := &http.Server{
		Addr:         ":8080",
		Handler:      serverHandler.Router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Create a done channel to signal when the shutdown is complete
	done := make(chan bool, 1)

	// Run graceful shutdown in a separate goroutine
	go gracefulShutdown(apiServer, db, schedulerService, schedulerCancel, done)

	log.Printf("Starting server on %s", apiServer.Addr)
	if err := apiServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server error: %v", err)
	}

	// Wait for the graceful shutdown to complete
	<-done
	log.Println("Graceful shutdown complete.")
}
