// Copyright (c) 2024-2025, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package server

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/autobrr/netronome/internal/auth"
	"github.com/autobrr/netronome/internal/broadcaster"
	"github.com/autobrr/netronome/internal/config"
	"github.com/autobrr/netronome/internal/database"
	"github.com/autobrr/netronome/internal/handlers"
	"github.com/autobrr/netronome/internal/scheduler"
	"github.com/autobrr/netronome/internal/speedtest"
	"github.com/autobrr/netronome/internal/types"
	"github.com/autobrr/netronome/web"
)

var _ broadcaster.Broadcaster = &Server{}

type Server struct {
	Router     *gin.Engine
	speedtest  speedtest.Service
	db         database.Service
	scheduler  scheduler.Service
	auth       *AuthHandler
	mu         sync.RWMutex
	lastUpdate *types.SpeedUpdate
	config     *config.Config
}

func NewServer(speedtest speedtest.Service, db database.Service, scheduler scheduler.Service, cfg *config.Config) *Server {
	// Set Gin mode from config
	if cfg.Server.GinMode != "" {
		gin.SetMode(cfg.Server.GinMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}
	gin.DefaultWriter = nil

	router := gin.New()

	// Initialize OIDC if configured
	oidcConfig, err := auth.NewOIDC(context.Background(), cfg.OIDC)
	if err != nil {
		log.Error().Err(err).Msg("failed to initialize OIDC")
		// Continue without OIDC
	}

	router.Use(LoggerMiddleware())
	router.Use(gin.Recovery())

	// CORS middleware with config
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

	s := &Server{
		Router:     router,
		speedtest:  speedtest,
		db:         db,
		scheduler:  scheduler,
		auth:       NewAuthHandler(db, oidcConfig, cfg.Session.Secret, cfg.Auth.Whitelist),
		lastUpdate: &types.SpeedUpdate{},
		config:     cfg,
	}

	// Register API routes first
	s.RegisterRoutes()

	// Use build.go for static file serving with embedded filesystem
	web.ServeStatic(router)

	return s
}

func (s *Server) BroadcastUpdate(update types.SpeedUpdate) {
	s.mu.Lock()
	s.lastUpdate = &update
	s.mu.Unlock()

	log.Debug().
		Bool("isScheduled", update.IsScheduled).
		Str("type", update.Type).
		Str("server", update.ServerName).
		Msg("Broadcasting speed test update")
}

func (s *Server) StartScheduler(ctx context.Context) {
	s.scheduler.Start(ctx)
	log.Info().Msg("Scheduler service started")
}

func (s *Server) RegisterRoutes() {
	baseURL := s.config.Server.BaseURL
	if baseURL == "" {
		baseURL = "/"
	}

	// Ensure baseURL starts with /
	if !strings.HasPrefix(baseURL, "/") {
		baseURL = "/" + baseURL
	}

	// remove trailing slash for route registration but preserve it in context
	routeBase := strings.TrimSuffix(baseURL, "/")

	// set base URL in context for all routes (preserve trailing slash)
	s.Router.Use(func(c *gin.Context) {
		c.Set("base_url", baseURL)
		c.Next()
	})

	// register api routes
	apiGroup := s.Router.Group(routeBase)
	if routeBase != "" {
		apiGroup = apiGroup.Group("")
	}
	api := apiGroup.Group("/api")
	{
		// public auth routes
		auth := api.Group("/auth")
		{
			auth.GET("/status", s.auth.CheckRegistrationStatus)
			auth.POST("/register", s.auth.Register)
			auth.POST("/login", s.auth.Login)
			if s.auth.oidc != nil {
				auth.GET("/oidc/login", s.auth.handleOIDCLogin)
				auth.GET("/oidc/callback", s.auth.handleOIDCCallback)
			}
		}

		// protected routes
		protected := api.Group("")
		protected.Use(RequireAuth(s.db, s.auth.oidc, s.config.Session.Secret, s.auth, s.config.Auth.Whitelist))
		{
			protected.POST("/auth/logout", s.auth.Logout)
			protected.GET("/auth/verify", s.auth.Verify)
			protected.GET("/auth/user", s.auth.GetUserInfo)

			protected.GET("/servers", s.handleGetServers)
			protected.POST("/speedtest", s.handleSpeedTest)
			protected.GET("/speedtest/status", s.handleSpeedTestStatus)
			protected.GET("/speedtest/history", s.handleSpeedTestHistory)
			protected.GET("/schedules", s.handleGetSchedules)
			protected.POST("/schedules", s.handleCreateSchedule)
			protected.PUT("/schedules/:id", s.handleUpdateSchedule)
			protected.DELETE("/schedules/:id", s.handleDeleteSchedule)

			iperfHandler := handlers.NewIperfHandler(s.db)
			protected.POST("/iperf/servers", iperfHandler.SaveServer)
			protected.GET("/iperf/servers", iperfHandler.GetServers)
			protected.DELETE("/iperf/servers/:id", iperfHandler.DeleteServer)
		}
	}

	// only register explicit routes if we have a base URL
	if routeBase != "" {
		// serve root path and index.html
		s.Router.GET(routeBase, web.ServeIndex)
		s.Router.GET(routeBase+"/", web.ServeIndex)
		s.Router.GET(routeBase+"/index.html", web.ServeIndex)
	}

	// register the catch-all handler for SPA routing
	web.ServeStatic(s.Router)
}

func LoggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		// skip certain endpoints to reduce noise
		if path == "/health" || path == "/api/speedtest/status" && c.Writer.Status() == 200 {
			return
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
