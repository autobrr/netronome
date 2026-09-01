// Copyright (c) 2024-2026, s0up and the autobrr contributors.
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
	"github.com/autobrr/netronome/internal/config"
	"github.com/autobrr/netronome/internal/database"
	"github.com/autobrr/netronome/internal/dnsmonitor"
	"github.com/autobrr/netronome/internal/handlers"
	"github.com/autobrr/netronome/internal/monitor"
	"github.com/autobrr/netronome/internal/notifications"
	"github.com/autobrr/netronome/internal/scheduler"
	"github.com/autobrr/netronome/internal/services/license"
	"github.com/autobrr/netronome/internal/speedtest"
	"github.com/autobrr/netronome/internal/types"
	"github.com/autobrr/netronome/internal/update"
	"github.com/autobrr/netronome/web"
)

type Server struct {
	Router               *gin.Engine
	speedtest            speedtest.Service
	packetLossService    *speedtest.PacketLossService
	dnsService           *dnsmonitor.Service
	monitorService       *monitor.Service
	db                   database.Service
	scheduler            scheduler.Service
	auth                 *AuthHandler
	notifier             *notifications.Notifier
	mu                   sync.RWMutex
	lastUpdate           *types.SpeedUpdate
	lastTracerouteUpdate *types.TracerouteUpdate
	lastPacketLossUpdate *types.PacketLossUpdate
	lastDNSUpdate        *types.DNSUpdate
	lastMonitorUpdate    *types.MonitorUpdate
	config               *config.Config
	licenseService       *license.Service
	updateChecker        *update.Checker
}

func (s *Server) SetUpdateChecker(checker *update.Checker) {
	s.updateChecker = checker
}

func NewServer(speedtest speedtest.Service, db database.Service, scheduler scheduler.Service, cfg *config.Config, packetLossService *speedtest.PacketLossService, dnsService *dnsmonitor.Service, monitorService *monitor.Service, notifier *notifications.Notifier, licenseService *license.Service) *Server {
	// Set Gin mode from config
	if cfg.Server.GinMode != "" {
		gin.SetMode(cfg.Server.GinMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}
	gin.DefaultWriter = nil

	router := gin.New()

	if err := router.SetTrustedProxies(cfg.Auth.TrustedProxies); err != nil {
		log.Error().Err(err).Msg("failed to set trusted proxies")
	}

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
		dnsService:        dnsService,
		monitorService:    monitorService,
		db:                db,
		scheduler:         scheduler,
		auth:              NewAuthHandler(db, oidcConfig, cfg.OIDC.Issuer != "", cfg.Session.Secret, cfg.Auth.Whitelist),
		notifier:          notifier,
		lastUpdate:        &types.SpeedUpdate{},
		config:            cfg,
		licenseService:    licenseService,
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

func (s *Server) BroadcastDNSUpdate(update types.DNSUpdate) {
	s.mu.Lock()
	s.lastDNSUpdate = &update
	s.mu.Unlock()

	log.Debug().
		Int64("monitorID", update.MonitorID).
		Str("host", update.Host).
		Bool("isRunning", update.IsRunning).
		Bool("success", update.Success).
		Str("responseCode", update.ResponseCode).
		Float64("responseTimeMs", update.ResponseTimeMs).
		Msg("Broadcasting dns update")
}

func (s *Server) BroadcastMonitorUpdate(update types.MonitorUpdate) {
	s.mu.Lock()
	s.lastMonitorUpdate = &update
	s.mu.Unlock()

	log.Trace().
		Int64("agentID", update.AgentID).
		Str("agentName", update.AgentName).
		Bool("connected", update.Connected).
		Int64("rxBytesPerSecond", update.RxBytesPerSecond).
		Int64("txBytesPerSecond", update.TxBytesPerSecond).
		Msg("Broadcasting monitor update")
}

func (s *Server) SetMonitorService(service *monitor.Service) {
	s.mu.Lock()
	s.monitorService = service
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
			if s.auth.oidcConfigured {
				auth.GET("/oidc/login", s.auth.handleOIDCLogin)
				auth.GET("/oidc/callback", s.auth.handleOIDCCallback)
			}
		}

		// public speedtest history
		api.GET("/speedtest/public/history", s.handlePublicSpeedTestHistory)

		licenseHandler := handlers.NewLicenseHandler(s.db, s.licenseService)

		// the public dashboard needs its theme without being authenticated.
		// The handler resolves it through entitlement, so an unlicensed
		// instance can never serve a premium theme here.
		api.GET("/public/theme", licenseHandler.GetPublicTheme)

		// protected routes
		protected := api.Group("")
		protected.Use(RequireAuth(s.db, s.auth.oidc, s.config.Session.Secret, s.auth, s.config.Auth.Whitelist))
		{
			protected.POST("/auth/logout", s.auth.Logout)
			protected.GET("/auth/verify", s.auth.Verify)
			protected.GET("/auth/user", s.auth.GetUserInfo)
			protected.GET("/version/latest", s.handleLatestVersion)

			protected.GET("/license", licenseHandler.GetLicense)
			protected.POST("/license/activate", licenseHandler.ActivateLicense)
			protected.POST("/license/deactivate", licenseHandler.DeactivateLicense)
			protected.GET("/settings/theme", licenseHandler.GetThemeSettings)
			protected.PUT("/settings/theme", licenseHandler.UpdateThemeSettings)

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
				protected.GET("/packetloss/monitors/:id/history/:resultId", packetLossHandler.GetMonitorHistoryDetail)
				protected.POST("/packetloss/monitors/:id/start", packetLossHandler.StartMonitor)
				protected.POST("/packetloss/monitors/:id/stop", packetLossHandler.StopMonitor)
			}

			// DNS monitoring routes
			if s.dnsService != nil {
				dnsHandler := handlers.NewDNSHandler(s.db, s.dnsService, s.scheduler)
				protected.GET("/dns/monitors", dnsHandler.GetMonitors)
				protected.POST("/dns/monitors", dnsHandler.CreateMonitor)
				protected.PUT("/dns/monitors/:id", dnsHandler.UpdateMonitor)
				protected.DELETE("/dns/monitors/:id", dnsHandler.DeleteMonitor)
				protected.GET("/dns/monitors/:id/status", dnsHandler.GetMonitorStatus)
				protected.GET("/dns/monitors/:id/history", dnsHandler.GetMonitorHistory)
			}

			// Vnstat monitoring routes
			if s.monitorService != nil {
				monitorHandler := handlers.NewMonitorHandler(s.db, s.monitorService, &s.config.Monitor)
				protected.GET("/monitor/agents", monitorHandler.GetAgents)
				protected.POST("/monitor/agents", monitorHandler.CreateAgent)
				protected.GET("/monitor/agents/:id", monitorHandler.GetAgent)
				protected.PUT("/monitor/agents/:id", monitorHandler.UpdateAgent)
				protected.DELETE("/monitor/agents/:id", monitorHandler.DeleteAgent)
				protected.GET("/monitor/agents/:id/status", monitorHandler.GetAgentStatus)
				protected.POST("/monitor/agents/:id/start", monitorHandler.StartAgent)
				protected.POST("/monitor/agents/:id/stop", monitorHandler.StopAgent)
				protected.GET("/monitor/agents/:id/native", monitorHandler.GetAgentNativeVnstat)
				protected.GET("/monitor/agents/:id/system", monitorHandler.GetAgentSystemInfo)
				protected.GET("/monitor/agents/:id/hardware", monitorHandler.GetAgentHardwareStats)
				protected.GET("/monitor/agents/:id/peaks", monitorHandler.GetAgentPeakStats)
				protected.GET("/monitor/tailscale/status", monitorHandler.GetTailscaleStatus)
			}

			// Notification routes
			protected.GET("/notifications/channels", s.handleGetNotificationChannels)
			protected.POST("/notifications/channels", s.handleCreateNotificationChannel)
			protected.PUT("/notifications/channels/:id", s.handleUpdateNotificationChannel)
			protected.DELETE("/notifications/channels/:id", s.handleDeleteNotificationChannel)

			protected.GET("/notifications/events", s.handleGetNotificationEvents)

			protected.GET("/notifications/rules", s.handleGetNotificationRules)
			protected.POST("/notifications/rules", s.handleCreateNotificationRule)
			protected.PUT("/notifications/rules/:id", s.handleUpdateNotificationRule)
			protected.DELETE("/notifications/rules/:id", s.handleDeleteNotificationRule)

			protected.POST("/notifications/test", s.handleTestNotification)
			protected.GET("/notifications/history", s.handleGetNotificationHistory)

			protected.GET("/settings/dashboard", s.handleGetDashboardSettings)
			protected.PUT("/settings/dashboard", s.handleUpdateDashboardSettings)

			protected.POST("/history/purge", s.handlePurgeHistory)
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
