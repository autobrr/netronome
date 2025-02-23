// Copyright (c) 2024-2025, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package server

import (
	"context"
	"net"
	"net/http"
	"strconv"
	"sync"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/rs/cors"
	"github.com/rs/zerolog/log"

	"github.com/autobrr/netronome/internal/auth"
	"github.com/autobrr/netronome/internal/config"
	"github.com/autobrr/netronome/internal/database"
	"github.com/autobrr/netronome/internal/handlers"
	"github.com/autobrr/netronome/internal/speedtest"
	"github.com/autobrr/netronome/internal/types"
)

type Server struct {
	config     *config.Config
	db         database.Service
	speedtest  speedtest.Service
	httpServer http.Server
	auth       *AuthHandler
	mu         sync.RWMutex
	lastUpdate *types.SpeedUpdate
}

func NewServer(cfg *config.Config, db database.Service, speedtest speedtest.Service) *Server {
	// Initialize OIDC if configured
	oidcConfig, err := auth.NewOIDC(context.Background(), cfg.OIDC)
	if err != nil {
		log.Error().Err(err).Msg("failed to initialize OIDC")
		// Continue without OIDC
	}

	s := &Server{
		speedtest:  speedtest,
		db:         db,
		auth:       NewAuthHandler(db, oidcConfig, cfg.Session.Secret, cfg.Server.BaseURL),
		lastUpdate: &types.SpeedUpdate{},
		config:     cfg,
	}

	return s
}

func (s *Server) ListenAndServe() error {
	addr := net.JoinHostPort(s.config.Server.Host, strconv.Itoa(s.config.Server.Port))
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	s.httpServer = http.Server{
		Handler: s.Handler(),
	}

	return s.httpServer.Serve(listener)
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}

func (s *Server) Handler() http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	// TODO add logger
	r.Use(LoggerMiddleware(log.With().Logger()))

	c := cors.New(cors.Options{
		AllowOriginFunc: func(origin string) bool { return true },
		AllowedMethods:  []string{"HEAD", "OPTIONS", "GET", "POST", "PUT", "PATCH", "DELETE"},
		//c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		AllowCredentials:   true,
		OptionsPassthrough: true,
		Debug:              false,
	})

	r.Use(c.Handler)

	apiRouter := chi.NewRouter()

	apiRouter.Group(func(r chi.Router) {
		r.Route("/healthz", newHealthHandler().Routes)

		r.Route("/auth", func(r chi.Router) {
			r.Post("/register", s.auth.Register)
			r.Post("/login", s.auth.Login)

			r.Group(func(r chi.Router) {
				// TODO auth middleware

				r.Get("/status", s.auth.CheckRegistrationStatus)

				r.Post("/logout", s.auth.Logout)
				r.Get("/verify", s.auth.Verify)
				r.Get("/user", s.auth.GetUserInfo)
			})

			if s.auth.oidc != nil {
				r.Route("/oidc", func(r chi.Router) {
					r.Get("/login", s.auth.handleOIDCLogin)
					r.Get("/callback", s.auth.handleOIDCCallback)
				})
			}
		})

		r.Group(func(r chi.Router) {
			// TODO auth middleware
			// protected.Use(RequireAuth(s.db, s.auth.oidc, s.config.Session.Secret, s.auth))

			r.Route("/servers", func(r chi.Router) {
				r.Get("/", s.getServers)
			})

			r.Route("/iperf", handlers.NewIperfHandler(s.db).Routes)

			r.Route("/speedtest", func(r chi.Router) {
				r.Post("/", s.handleSpeedTest)
				r.Get("/status", s.handleSpeedTestStatus)
				r.Get("/history", s.handleSpeedTestHistory)
			})

			r.Route("/schedules", func(r chi.Router) {
				r.Get("/", s.handleGetSchedules)
				r.Post("/", s.handleCreateSchedule)

				r.Route("/{scheduleID}", func(r chi.Router) {
					r.Put("/", s.handleUpdateSchedule)
					r.Delete("/", s.handleDeleteSchedule)
				})
			})
		})
	})

	r.Mount("/api", apiRouter)

	// Use build.go for static file serving with embedded filesystem
	//web.ServeStatic(router)

	//// only register explicit routes if we have a base URL
	//if routeBase != "" {
	//	// serve root path and index.html
	//	s.Router.GET(routeBase, web.ServeIndex)
	//	s.Router.GET(routeBase+"/", web.ServeIndex)
	//	s.Router.GET(routeBase+"/index.html", web.ServeIndex)
	//}

	return r
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

//func LoggerMiddleware() gin.HandlerFunc {
//	return func(c *gin.Context) {
//		start := time.Now()
//		path := c.Request.URL.Path
//		query := c.Request.URL.RawQuery
//
//		c.Next()
//
//		// skip certain endpoints to reduce noise
//		if path == "/health" || path == "/api/speedtest/status" && c.Writer.Status() == 200 {
//			return
//		}
//
//		var event *zerolog.Event
//		switch {
//		case c.Writer.Status() >= 500:
//			event = log.Error()
//		case c.Writer.Status() >= 400:
//			event = log.Warn()
//		default:
//			event = log.Info()
//		}
//
//		event.
//			Str("method", c.Request.Method).
//			Str("path", path).
//			Int("status", c.Writer.Status()).
//			Dur("latency", time.Since(start))
//
//		if query != "" {
//			event.Str("query", query)
//		}
//
//		if len(c.Errors) > 0 {
//			event.Str("error", c.Errors.String())
//		}
//
//		if requestID := c.GetHeader("X-Request-ID"); requestID != "" {
//			event.Str("request_id", requestID)
//		}
//
//		event.Msg("HTTP Request")
//	}
//}
