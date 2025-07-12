// Copyright (c) 2024-2025, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package speedtest

import (
	"context"
	"fmt"

	"github.com/rs/zerolog/log"

	"github.com/autobrr/netronome/internal/config"
	"github.com/autobrr/netronome/internal/database"
	"github.com/autobrr/netronome/internal/notifications"
	"github.com/autobrr/netronome/internal/types"
)

type Service interface {
	RunTest(ctx context.Context, opts *types.TestOptions) (*Result, error)
	GetServers(testType string) ([]ServerResponse, error)
	GetLibrespeedServers() ([]ServerResponse, error)
	RunLibrespeedTest(ctx context.Context, opts *types.TestOptions) (*Result, error)
	RunTraceroute(ctx context.Context, host string) (*TracerouteResult, error)
	SetBroadcastUpdate(broadcastUpdate func(types.SpeedUpdate))
	SetBroadcastTracerouteUpdate(broadcastUpdate func(types.TracerouteUpdate))
}

type service struct {
	db                        database.Service
	config                    config.SpeedTestConfig
	fullConfig                *config.Config
	notifier                  *notifications.Notifier
	broadcastUpdate           func(types.SpeedUpdate)
	broadcastTracerouteUpdate func(types.TracerouteUpdate)

	// New architecture components
	speedtestNetRunner *SpeedtestNetRunner
	iperfRunner        *IperfRunner
	librespeedRunner   *LibrespeedRunner
	resultHandler      ResultHandler
}

func New(db database.Service, cfg config.SpeedTestConfig, notifier *notifications.Notifier, fullConfig *config.Config) Service {
	svc := &service{
		db:         db,
		config:     cfg,
		fullConfig: fullConfig,
		notifier:   notifier,
	}

	// Initialize new architecture components
	svc.resultHandler = NewResultHandler(db, notifier)
	svc.speedtestNetRunner = NewSpeedtestNetRunner(cfg)
	svc.iperfRunner = NewIperfRunner(cfg.IPerf)
	svc.librespeedRunner = NewLibrespeedRunner(cfg.Librespeed)

	// Initialize GeoIP databases for all speedtest features (traceroute, MTR, etc.)
	svc.initGeoIP()

	// log.Debug().Msg("Initialized speedtest service")
	return svc
}

func (s *service) SetBroadcastUpdate(broadcastUpdate func(types.SpeedUpdate)) {
	s.broadcastUpdate = broadcastUpdate
}

func (s *service) SetBroadcastTracerouteUpdate(broadcastUpdate func(types.TracerouteUpdate)) {
	s.broadcastTracerouteUpdate = broadcastUpdate
}

func (s *service) GetLibrespeedServers() ([]ServerResponse, error) {
	return s.librespeedRunner.GetServers()
}

func (s *service) RunLibrespeedTest(ctx context.Context, opts *types.TestOptions) (*Result, error) {
	s.librespeedRunner.SetProgressCallback(s.broadcastUpdate)
	result, err := s.librespeedRunner.RunTest(ctx, opts)
	if err != nil {
		return nil, err
	}

	// Save result using the result handler
	if err := s.resultHandler.SaveResult(ctx, result, "librespeed", opts); err != nil {
		log.Error().Err(err).Msg("Failed to save librespeed result")
	}

	return result, nil
}

func (s *service) RunIperfTest(ctx context.Context, opts *types.TestOptions) (*types.SpeedTestResult, error) {
	s.iperfRunner.SetProgressCallback(s.broadcastUpdate)
	return s.iperfRunner.runSingleIperfTest(ctx, opts)
}

func (s *service) RunTest(ctx context.Context, opts *types.TestOptions) (*Result, error) {
	log.Debug().
		Bool("isScheduled", opts.IsScheduled).
		Bool("useIperf", opts.UseIperf).
		Bool("useLibrespeed", opts.UseLibrespeed).
		Str("server_ids", fmt.Sprintf("%v", opts.ServerIDs)).
		Str("server_host", opts.ServerHost).
		Msg("Starting speed test coordination")

	if opts.UseLibrespeed {
		log.Info().Msg("Using librespeed runner (handles ping natively)")
		return s.RunLibrespeedTest(ctx, opts)
	}

	if opts.UseIperf && opts.ServerHost != "" {
		log.Info().Str("server_host", opts.ServerHost).Msg("Using iperf3 runner")

		// Only iperf3 needs a separate ping test since it doesn't handle ping natively
		pingResult, err := s.RunPingTest(ctx, opts.ServerHost)
		if err != nil {
			log.Warn().Err(err).
				Str("server_host", opts.ServerHost).
				Msg("Ping test failed, continuing with speed tests")
		} else {
			log.Debug().
				Str("latency", pingResult.FormatLatency()).
				Float64("packet_loss", pingResult.PacketLoss).
				Str("server_host", opts.ServerHost).
				Msg("Ping test completed successfully")

			// Pass ping results to iperf runner
			s.iperfRunner.SetPingResult(pingResult)
		}

		// Use the new iperf runner
		s.iperfRunner.SetProgressCallback(s.broadcastUpdate)
		result, err := s.iperfRunner.RunTest(ctx, opts)
		if err != nil {
			return nil, fmt.Errorf("iperf3 test failed: %w", err)
		}

		// Save the result
		if err := s.resultHandler.SaveResult(ctx, result, "iperf3", opts); err != nil {
			log.Error().Err(err).Msg("Failed to save iperf3 result")
		}

		return result, nil
	}

	// Default to speedtest.net runner
	log.Info().Msg("Using speedtest.net runner (handles ping natively)")
	s.speedtestNetRunner.SetProgressCallback(s.broadcastUpdate)
	result, err := s.speedtestNetRunner.RunTest(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("speedtest.net test failed: %w", err)
	}

	// Save the result
	if err := s.resultHandler.SaveResult(ctx, result, "speedtest", opts); err != nil {
		log.Error().Err(err).Msg("Failed to save speedtest.net result")
	}

	return result, nil
}

func (s *service) GetServers(testType string) ([]ServerResponse, error) {
	switch testType {
	case "librespeed":
		return s.GetLibrespeedServers()
	case "iperf3":
		return s.iperfRunner.GetServers()
	case "speedtest":
		return s.speedtestNetRunner.GetServers()
	default:
		return s.speedtestNetRunner.GetServers()
	}
}
