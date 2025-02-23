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
	"github.com/gorilla/sessions"
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
	config      *config.Config
	db          database.Service
	speedtest   speedtest.Service
	httpServer  http.Server
	mu          sync.RWMutex
	lastUpdate  *types.SpeedUpdate
	cookieStore *sessions.CookieStore
	oidcConfig  *auth.OIDCConfig
}

func NewServer(cfg *config.Config, db database.Service, speedtest speedtest.Service, oidcCfg *auth.OIDCConfig) *Server {
	s := &Server{
		speedtest:   speedtest,
		db:          db,
		lastUpdate:  &types.SpeedUpdate{},
		config:      cfg,
		cookieStore: sessions.NewCookieStore([]byte(cfg.Session.Secret)),
		oidcConfig:  oidcCfg,
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
		//AllowedHeaders:  []string{"Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With"},
		AllowCredentials:   true,
		OptionsPassthrough: true,
		Debug:              false,
	})

	r.Use(c.Handler)

	apiRouter := chi.NewRouter()

	apiRouter.Group(func(r chi.Router) {
		r.Route("/healthz", newHealthHandler().Routes)
		r.Route("/auth", NewAuthHandler(s.db, s.oidcConfig, s.config.Session.Secret, s.config.Server.BaseURL, s.cookieStore).Routes)

		r.Group(func(r chi.Router) {
			r.Use(IsAuthenticated(s.config.Server.BaseURL, s.cookieStore))

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
