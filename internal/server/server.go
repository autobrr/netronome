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
	"github.com/autobrr/netronome/internal/vnstat"
	"github.com/autobrr/netronome/web"
)

var _ broadcaster.Broadcaster = &Server{}

type Server struct {
	Router               *gin.Engine
	speedtest            speedtest.Service
	packetLossService    *speedtest.PacketLossService
	vnstatService        *vnstat.Service
	db                   database.Service
	scheduler            scheduler.Service
	auth                 *AuthHandler
	mu                   sync.RWMutex
	lastUpdate           *types.SpeedUpdate
	lastTracerouteUpdate *types.TracerouteUpdate
	lastPacketLossUpdate *types.PacketLossUpdate
	lastVnstatUpdate     *types.VnstatUpdate
	config               *config.Config
}

func NewServer(speedtest speedtest.Service, db database.Service, scheduler scheduler.Service, cfg *config.Config, packetLossService *speedtest.PacketLossService, vnstatService *vnstat.Service) *Server {
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
	router.Use(ErrorHandlerMiddleware())

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
		Router:            router,
		speedtest:         speedtest,
		packetLossService: packetLossService,
		vnstatService:     vnstatService,
		db:                db,
		scheduler:         scheduler,
		auth:              NewAuthHandler(db, oidcConfig, cfg.Session.Secret, cfg.Auth.Whitelist),
		lastUpdate:        &types.SpeedUpdate{},
		config:            cfg,
	}

	// Don't register routes here - let the caller do it after setting up packet loss service
	return s
}

func (s *Server) BroadcastUpdate(update types.SpeedUpdate) {
	s.mu.Lock()
	s.lastUpdate = &update
	s.mu.Unlock()

	log.Trace().
		Bool("isScheduled", update.IsScheduled).
		Str("type", update.Type).
		Str("server", update.ServerName).
		Msg("Broadcasting speed test update")
}

func (s *Server) BroadcastTracerouteUpdate(update types.TracerouteUpdate) {
	s.mu.Lock()
	s.lastTracerouteUpdate = &update
	s.mu.Unlock()

	log.Debug().
		Bool("isScheduled", update.IsScheduled).
		Str("type", update.Type).
		Str("host", update.Host).
		Int("currentHop", update.CurrentHop).
		Int("totalHops", update.TotalHops).
		Float64("progress", update.Progress).
		Msg("Broadcasting traceroute update")
}

func (s *Server) BroadcastPacketLossUpdate(update types.PacketLossUpdate) {
	s.mu.Lock()
	s.lastPacketLossUpdate = &update
	s.mu.Unlock()

	log.Debug().
		Int64("monitorID", update.MonitorID).
		Str("type", update.Type).
		Str("host", update.Host).
		Bool("isRunning", update.IsRunning).
		Bool("isComplete", update.IsComplete).
		Float64("packetLoss", update.PacketLoss).
		Msg("Broadcasting packet loss update")
}

func (s *Server) BroadcastVnstatUpdate(update types.VnstatUpdate) {
	s.mu.Lock()
	s.lastVnstatUpdate = &update
	s.mu.Unlock()

	log.Debug().
		Int64("agentID", update.AgentID).
		Str("agentName", update.AgentName).
		Bool("connected", update.Connected).
		Int64("rxBytesPerSecond", update.RxBytesPerSecond).
		Int64("txBytesPerSecond", update.TxBytesPerSecond).
		Msg("Broadcasting vnstat update")
}

func (s *Server) SetPacketLossService(service *speedtest.PacketLossService) {
	s.mu.Lock()
	s.packetLossService = service
	s.mu.Unlock()
}

func (s *Server) SetVnstatService(service *vnstat.Service) {
	s.mu.Lock()
	s.vnstatService = service
	s.mu.Unlock()
}

func (s *Server) Initialize() {
	// Register API routes
	s.RegisterRoutes()

	// Use build.go for static file serving with embedded filesystem
	web.ServeStatic(s.Router)
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

		// public speedtest history
		api.GET("/speedtest/public/history", s.handlePublicSpeedTestHistory)

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
			protected.GET("/traceroute", s.handleTraceroute)
			protected.GET("/traceroute/status", s.handleTracerouteStatus)
			protected.GET("/schedules", s.handleGetSchedules)
			protected.POST("/schedules", s.handleCreateSchedule)
			protected.PUT("/schedules/:id", s.handleUpdateSchedule)
			protected.DELETE("/schedules/:id", s.handleDeleteSchedule)

			iperfHandler := handlers.NewIperfHandler(s.db)
			protected.POST("/iperf/servers", iperfHandler.SaveServer)
			protected.GET("/iperf/servers", iperfHandler.GetServers)
			protected.DELETE("/iperf/servers/:id", iperfHandler.DeleteServer)

			// Packet Loss monitoring routes
			if s.packetLossService != nil {
				packetLossHandler := handlers.NewPacketLossHandler(s.db, s.packetLossService, s.scheduler)
				protected.GET("/packetloss/monitors", packetLossHandler.GetMonitors)
				protected.POST("/packetloss/monitors", packetLossHandler.CreateMonitor)
				protected.PUT("/packetloss/monitors/:id", packetLossHandler.UpdateMonitor)
				protected.DELETE("/packetloss/monitors/:id", packetLossHandler.DeleteMonitor)
				protected.GET("/packetloss/monitors/:id/status", packetLossHandler.GetMonitorStatus)
				protected.GET("/packetloss/monitors/:id/history", packetLossHandler.GetMonitorHistory)
				protected.POST("/packetloss/monitors/:id/start", packetLossHandler.StartMonitor)
				protected.POST("/packetloss/monitors/:id/stop", packetLossHandler.StopMonitor)
			}

			// Vnstat monitoring routes
			if s.vnstatService != nil {
				vnstatHandler := handlers.NewVnstatHandler(s.db, s.vnstatService)
				protected.GET("/vnstat/agents", vnstatHandler.GetAgents)
				protected.POST("/vnstat/agents", vnstatHandler.CreateAgent)
				protected.GET("/vnstat/agents/:id", vnstatHandler.GetAgent)
				protected.PUT("/vnstat/agents/:id", vnstatHandler.UpdateAgent)
				protected.DELETE("/vnstat/agents/:id", vnstatHandler.DeleteAgent)
				protected.GET("/vnstat/agents/:id/status", vnstatHandler.GetAgentStatus)
				protected.GET("/vnstat/agents/:id/bandwidth", vnstatHandler.GetAgentBandwidth)
				protected.GET("/vnstat/agents/:id/usage", vnstatHandler.GetAgentUsage)
				protected.POST("/vnstat/agents/:id/start", vnstatHandler.StartAgent)
				protected.POST("/vnstat/agents/:id/stop", vnstatHandler.StopAgent)
			}
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

		//event.Msg("HTTP Request")
	}
}
