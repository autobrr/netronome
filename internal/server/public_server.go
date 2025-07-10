// Copyright (c) 2024-2025, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package server

import (
	"fmt"
	"io"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/autobrr/netronome/internal/config"
	"github.com/autobrr/netronome/internal/database"
	"github.com/autobrr/netronome/web"
)

// PublicServer represents a minimal public-only server
type PublicServer struct {
	Router *gin.Engine
	db     database.Service
	config *config.Config
}

// NewPublicServer creates a new public-only server with minimal middleware
func NewPublicServer(db database.Service, cfg *config.Config) *PublicServer {
	// Set Gin mode from config
	if cfg.Server.GinMode != "" {
		gin.SetMode(cfg.Server.GinMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}
	gin.DefaultWriter = nil

	router := gin.New()

	// Apply only essential middleware
	router.Use(PublicLoggerMiddleware())
	router.Use(gin.Recovery())
	router.Use(ErrorHandlerMiddleware())

	// CORS middleware
	router.Use(func(c *gin.Context) {
		if c.Request.Method == "OPTIONS" {
			c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
			c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
			c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
			c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	s := &PublicServer{
		Router: router,
		db:     db,
		config: cfg,
	}

	// Register only public routes
	s.RegisterPublicRoutes()

	// Register selective frontend serving for public routes only
	s.RegisterPublicFrontend()

	return s
}

// RegisterPublicRoutes registers only the public endpoints
func (s *PublicServer) RegisterPublicRoutes() {
	// Health check endpoint
	s.Router.GET("/health", s.handleHealth)

	// Public API endpoints
	api := s.Router.Group("/api")
	{
		speedtest := api.Group("/speedtest")
		{
			public := speedtest.Group("/public")
			{
				public.GET("/history", s.handlePublicSpeedTestHistory)
			}
			// Stub endpoint to prevent frontend errors
			speedtest.GET("/status", s.handleSpeedTestStatus)
		}
		
		// Stub endpoints to prevent frontend errors and reduce noise
		api.GET("/servers", s.handleServersStub)
		api.GET("/schedules", s.handleSchedulesStub)
		
		auth := api.Group("/auth")
		{
			auth.GET("/status", s.handleAuthStatusStub)
		}
	}
}

// handleHealth provides a simple health check endpoint
func (s *PublicServer) handleHealth(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "healthy",
		"server": "public",
		"time":   time.Now().UTC().Format(time.RFC3339),
	})
}

// handlePublicSpeedTestHistory handles public speed test history requests
func (s *PublicServer) handlePublicSpeedTestHistory(c *gin.Context) {
	timeRange := c.DefaultQuery("timeRange", s.config.Pagination.DefaultTimeRange)
	page, _ := strconv.Atoi(c.DefaultQuery("page", strconv.Itoa(s.config.Pagination.DefaultPage)))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", strconv.Itoa(s.config.Pagination.DefaultLimit)))

	results, err := s.db.GetSpeedTests(c.Request.Context(), timeRange, page, limit)
	if err != nil {
		log.Error().Err(err).
			Str("timeRange", timeRange).
			Int("page", page).
			Int("limit", limit).
			Msg("Failed to retrieve speed test history")
		c.Status(http.StatusInternalServerError)
		_ = c.Error(fmt.Errorf("failed to retrieve speed test history: %w", err))
		return
	}

	c.JSON(http.StatusOK, results)
}

// PublicLoggerMiddleware is a simplified logger middleware for the public server
func PublicLoggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		// skip health endpoint to reduce noise
		if path == "/health" && c.Writer.Status() == 200 {
			return
		}

		// skip expected 404s for API endpoints and other paths not available on public server
		if c.Writer.Status() == 404 {
			// these are expected to fail on public server, don't log as warnings
			expectedPaths := []string{
				"/api/servers",
				"/api/schedules", 
				"/api/speedtest/status",
				"/api/traceroute",
				"/api/auth/status",
				"/api/auth/verify",
				"/api/iperf",
				"/polar-svg",
				"/vite.svg",
				"/manifest.json",
				"/robots.txt",
			}
			
			for _, expectedPath := range expectedPaths {
				if strings.HasPrefix(path, expectedPath) {
					return // skip logging these expected 404s
				}
			}
		}

		var event *zerolog.Event
		switch {
		case c.Writer.Status() >= 500:
			event = log.Error()
		case c.Writer.Status() >= 400:
			event = log.Warn()
		default:
			event = log.Info()
		}

		event.
			Str("server", "public").
			Str("method", c.Request.Method).
			Str("path", path).
			Int("status", c.Writer.Status()).
			Dur("latency", time.Since(start))

		if query != "" {
			event.Str("query", query)
		}

		if len(c.Errors) > 0 {
			event.Str("error", c.Errors.String())
		}

		if requestID := c.GetHeader("X-Request-ID"); requestID != "" {
			event.Str("request_id", requestID)
		}

		event.Msg("HTTP Request")
	}
}

// RegisterPublicFrontend registers selective frontend serving for public routes only
func (s *PublicServer) RegisterPublicFrontend() {
	// Add no-cache headers for HTML files
	s.Router.Use(func(c *gin.Context) {
		if strings.HasSuffix(c.Request.URL.Path, ".html") {
			c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
			c.Header("Pragma", "no-cache")
			c.Header("Expires", "0")
		}
		c.Next()
	})

	// Handle static assets needed for the React app
	s.Router.GET("/assets/*filepath", s.serveAssets)
	s.Router.GET("/favicon.ico", s.serveFavicon)

	// Handle only the /public route for the dashboard
	s.Router.GET("/public", s.servePublicDashboard)
	s.Router.GET("/public/*path", s.servePublicDashboard)

	// Handle root redirect to public
	s.Router.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/public")
	})

	// All other routes return 404
	s.Router.NoRoute(func(c *gin.Context) {
		c.Status(http.StatusNotFound)
	})
}

// serveAssets serves static assets from the embedded filesystem
func (s *PublicServer) serveAssets(c *gin.Context) {
	filepath := strings.TrimPrefix(c.Param("filepath"), "/")
	assetPath := "assets/" + filepath

	file, err := web.DistDirFS.Open(assetPath)
	if err != nil {
		log.Debug().Str("filepath", assetPath).Err(err).Msg("failed to open static file")
		c.Status(http.StatusNotFound)
		return
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		log.Debug().Str("filepath", assetPath).Err(err).Msg("failed to stat static file")
		c.Status(http.StatusInternalServerError)
		return
	}

	// Set content type based on file extension
	ext := strings.ToLower(path.Ext(filepath))
	var contentType string
	switch ext {
	case ".css":
		contentType = "text/css; charset=utf-8"
	case ".js":
		contentType = "text/javascript; charset=utf-8"
	case ".svg":
		contentType = "image/svg+xml"
	case ".png":
		contentType = "image/png"
	default:
		contentType = "application/octet-stream"
	}

	c.Header("Content-Type", contentType)
	c.Header("Cache-Control", "public, max-age=31536000")
	http.ServeContent(c.Writer, c.Request, filepath, stat.ModTime(), file.(io.ReadSeeker))
}

// serveFavicon serves the favicon
func (s *PublicServer) serveFavicon(c *gin.Context) {
	file, err := web.DistDirFS.Open("favicon.ico")
	if err != nil {
		c.Status(http.StatusNotFound)
		return
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	c.Header("Content-Type", "image/x-icon")
	c.Header("Cache-Control", "public, max-age=31536000")
	http.ServeContent(c.Writer, c.Request, "favicon.ico", stat.ModTime(), file.(io.ReadSeeker))
}

// servePublicDashboard serves the index.html for the public dashboard route only
func (s *PublicServer) servePublicDashboard(c *gin.Context) {
	file, err := web.DistDirFS.Open("index.html")
	if err != nil {
		c.Status(http.StatusNotFound)
		return
	}
	defer file.Close()

	// Read the file content
	content, err := io.ReadAll(file)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	// Replace the template variable with root base URL for public server
	html := strings.Replace(string(content), "{{.BaseURL}}", "/", 1)

	c.Header("Content-Type", "text/html; charset=utf-8")
	c.Header("Cache-Control", "no-cache")
	c.String(http.StatusOK, html)
}

// Stub handlers to prevent frontend errors and reduce API noise

// handleSpeedTestStatus returns a stub status indicating no test is running
func (s *PublicServer) handleSpeedTestStatus(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"running": false,
		"message": "Speed test status not available on public server",
	})
}

// handleServersStub returns empty server list
func (s *PublicServer) handleServersStub(c *gin.Context) {
	c.JSON(http.StatusOK, []interface{}{})
}

// handleSchedulesStub returns empty schedules list
func (s *PublicServer) handleSchedulesStub(c *gin.Context) {
	c.JSON(http.StatusOK, []interface{}{})
}

// handleAuthStatusStub returns unauthenticated status
func (s *PublicServer) handleAuthStatusStub(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"authenticated": false,
		"message":       "Authentication not available on public server",
	})
}