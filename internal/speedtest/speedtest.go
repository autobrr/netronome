// Copyright (c) 2024-2025, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package speedtest

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/rs/zerolog/log"
	st "github.com/showwin/speedtest-go/speedtest"

	"github.com/autobrr/netronome/internal/broadcaster"
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
	SetBroadcastUpdate(broadcastUpdate func(types.SpeedUpdate))
}

type service struct {
	client          *st.Speedtest
	db              database.Service
	config          config.SpeedTestConfig
	notifier        *notifications.Notifier
	broadcaster     broadcaster.Broadcaster
	broadcastUpdate func(types.SpeedUpdate)
	serverCache     []ServerResponse
	cacheExpiry     time.Time
	cacheDuration   time.Duration
}

func New(db database.Service, cfg config.SpeedTestConfig, notifier *notifications.Notifier) Service {
	svc := &service{
		client:        st.New(),
		db:            db,
		config:        cfg,
		notifier:      notifier,
		cacheDuration: 30 * time.Minute,
		cacheExpiry:   time.Now(),
	}

	// log.Debug().Msg("Initialized speedtest service")
	return svc
}

func (s *service) SetBroadcastUpdate(broadcastUpdate func(types.SpeedUpdate)) {
	s.broadcastUpdate = broadcastUpdate
}

func (s *service) RunTest(ctx context.Context, opts *types.TestOptions) (*Result, error) {
	log.Debug().
		Bool("isScheduled", opts.IsScheduled).
		Bool("useIperf", opts.UseIperf).
		Str("server_ids", fmt.Sprintf("%v", opts.ServerIDs)).
		Str("server_host", opts.ServerHost).
		Msg("Starting speed test")

	if opts.UseLibrespeed {
		log.Info().Msg("Starting librespeed test")
		return s.RunLibrespeedTest(ctx, opts)
	}

	if opts.UseIperf && opts.ServerHost != "" {
		log.Info().
			Str("server_host", opts.ServerHost).
			Bool("enable_download", opts.EnableDownload).
			Bool("enable_upload", opts.EnableUpload).
			Bool("enable_ping", opts.EnablePing).
			Bool("enable_jitter", opts.EnableJitter).
			Msg("Starting iperf3 test")

		var downloadSpeed, uploadSpeed float64
		var jitterMs *float64
		var latency string = "0ms"

		// Run ping test for iperf3 if enabled to get accurate latency
		if opts.EnablePing {
			pingResult, err := s.RunPingTest(ctx, opts.ServerHost)
			if err != nil {
				log.Warn().Err(err).Msg("Ping test failed, continuing with speed tests")
			} else {
				latency = pingResult.FormatLatency()
				log.Debug().
					Str("latency", latency).
					Float64("packet_loss", pingResult.PacketLoss).
					Msg("Ping test completed")
			}
		}

		// Run download test if enabled
		if opts.EnableDownload {
			downloadOpts := *opts
			downloadOpts.EnableDownload = true
			downloadOpts.EnableUpload = false

			downloadResult, err := s.RunIperfTest(context.Background(), &downloadOpts)
			if err != nil {
				return nil, fmt.Errorf("download test failed: %w", err)
			}
			downloadSpeed = downloadResult.DownloadSpeed
			if downloadResult.Jitter != nil {
				jitterMs = downloadResult.Jitter
			}
		}

		// Short pause between tests
		time.Sleep(2 * time.Second)

		// Run upload test if enabled
		if opts.EnableUpload {
			uploadOpts := *opts
			uploadOpts.EnableDownload = false
			uploadOpts.EnableUpload = true

			uploadResult, err := s.RunIperfTest(context.Background(), &uploadOpts)
			if err != nil {
				return nil, fmt.Errorf("upload test failed: %w", err)
			}
			uploadSpeed = uploadResult.UploadSpeed
			// Prefer jitter from upload test if available
			if uploadResult.Jitter != nil {
				jitterMs = uploadResult.Jitter
			}
		}

		var jitterFloat float64
		if jitterMs != nil {
			jitterFloat = *jitterMs
		}

		result := &Result{
			Timestamp:     time.Now(),
			Server:        opts.ServerHost,
			DownloadSpeed: downloadSpeed,
			UploadSpeed:   uploadSpeed,
			Latency:       latency,
			PacketLoss:    0.0,
			Jitter:        jitterFloat,
			Download:      downloadSpeed,
			Upload:        uploadSpeed,
		}

		// Save to database
		dbResult, err := s.db.SaveSpeedTest(context.Background(), types.SpeedTestResult{
			ServerName:    opts.ServerHost,
			ServerID:      fmt.Sprintf("iperf3-%s", opts.ServerHost),
			ServerHost:    &opts.ServerHost,
			TestType:      "iperf3",
			DownloadSpeed: downloadSpeed,
			UploadSpeed:   uploadSpeed,
			Latency:       latency,
			PacketLoss:    0.0,
			Jitter:        jitterMs,
			IsScheduled:   opts.IsScheduled,
		})
		if err != nil {
			log.Error().Err(err).Msg("Failed to save iperf3 result to database")
		}

		if dbResult != nil {
			result.ID = dbResult.ID
			s.notifier.SendNotification(dbResult)
		}

		return result, nil
	}

	serverList, err := s.client.FetchServers()
	if err != nil {
		log.Error().Err(err).Msg("Failed to fetch servers")
		return nil, fmt.Errorf("failed to fetch servers: %w", err)
	}

	var selectedServer *st.Server
	if len(opts.ServerIDs) > 0 {
		for _, server := range serverList {
			for _, requestedID := range opts.ServerIDs {
				if server.ID == requestedID {
					selectedServer = server
					break
				}
			}
			if selectedServer != nil {
				break
			}
		}
	}

	if selectedServer == nil {
		sort.Slice(serverList, func(i, j int) bool {
			return serverList[i].Distance < serverList[j].Distance
		})
		selectedServer = serverList[0]
	}

	log.Info().
		Str("server_ids", fmt.Sprintf("%v", opts.ServerIDs)).
		Str("server_name", selectedServer.Name).
		Str("server_host", selectedServer.Host).
		Str("server_country", selectedServer.Country).
		Str("provider", selectedServer.Sponsor).
		Bool("enable_download", opts.EnableDownload).
		Bool("enable_upload", opts.EnableUpload).
		Msg("Starting speed test")

	result := &Result{
		Timestamp: time.Now(),
		Server:    selectedServer.Name,
	}

	if err := selectedServer.PingTest(func(latency time.Duration) {
		if s.broadcastUpdate != nil {
			s.broadcastUpdate(types.SpeedUpdate{
				Type:        "ping",
				ServerName:  selectedServer.Name,
				Latency:     latency.String(),
				Progress:    100,
				IsComplete:  false,
				IsScheduled: opts.IsScheduled,
			})
		}
	}); err != nil {
		result.Error = fmt.Sprintf("ping test failed: %v", err)
		return result, err
	}
	result.Latency = selectedServer.Latency.String()

	if s.broadcastUpdate != nil {
		s.broadcastUpdate(types.SpeedUpdate{
			Type:        "ping",
			ServerName:  selectedServer.Name,
			Latency:     selectedServer.Latency.String(),
			Progress:    100,
			IsComplete:  false,
			IsScheduled: opts.IsScheduled,
		})
	}

	if opts.EnableDownload {
		var downloadStartTime time.Time
		var progress float64
		var lastUpdate atomic.Int64 // For rate limiting updates

		selectedServer.Context.SetCallbackDownload(func(speed st.ByteRate) {
			if downloadStartTime.IsZero() {
				downloadStartTime = time.Now()
			}

			now := time.Now().Unix()
			lastUpdateTime := lastUpdate.Load()

			if now-lastUpdateTime >= 1 {
				elapsed := time.Since(downloadStartTime).Seconds()
				progress = math.Min(100, (elapsed/10.0)*100)

				if progress > 0 && s.broadcastUpdate != nil {
					s.broadcastUpdate(types.SpeedUpdate{
						Type:        "download",
						ServerName:  selectedServer.Name,
						Speed:       speed.Mbps(),
						Progress:    progress,
						IsComplete:  progress >= 100,
						IsScheduled: opts.IsScheduled,
					})
					lastUpdate.Store(now)
				}
			}
		})

		timeout := time.Duration(s.config.Timeout) * time.Second
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		if err := selectedServer.DownloadTestContext(ctx); err != nil {
			if ctx.Err() == context.DeadlineExceeded {
				return nil, fmt.Errorf("download test timed out after %d seconds", s.config.Timeout)
			}
			return nil, fmt.Errorf("download test failed: %w", err)
		}

		result.DownloadSpeed = selectedServer.DLSpeed.Mbps()

		// Broadcast download completion
		if s.broadcastUpdate != nil {
			s.broadcastUpdate(types.SpeedUpdate{
				Type:        "download",
				ServerName:  selectedServer.Name,
				Speed:       result.DownloadSpeed,
				Progress:    100,
				IsComplete:  true,
				IsScheduled: opts.IsScheduled,
			})
		}
	}

	if opts.EnableUpload {
		var uploadStartTime time.Time
		var progress float64
		var lastUpdate atomic.Int64 // For rate limiting updates

		selectedServer.Context.SetCallbackUpload(func(speed st.ByteRate) {
			if uploadStartTime.IsZero() {
				uploadStartTime = time.Now()
			}

			now := time.Now().Unix()
			lastUpdateTime := lastUpdate.Load()

			// Only broadcast once per second
			if now-lastUpdateTime >= 1 {
				elapsed := time.Since(uploadStartTime).Seconds()
				progress = math.Min(100, (elapsed/10.0)*100)

				if progress > 0 && s.broadcastUpdate != nil {
					s.broadcastUpdate(types.SpeedUpdate{
						Type:        "upload",
						ServerName:  selectedServer.Name,
						Speed:       speed.Mbps(),
						Progress:    progress,
						IsComplete:  progress >= 100,
						IsScheduled: opts.IsScheduled,
					})
					lastUpdate.Store(now)
				}
			}
		})

		timeout := time.Duration(s.config.Timeout) * time.Second
		uploadCtx, uploadCancel := context.WithTimeout(context.Background(), timeout)
		defer uploadCancel()

		if err := selectedServer.UploadTestContext(uploadCtx); err != nil {
			if uploadCtx.Err() == context.DeadlineExceeded {
				return nil, fmt.Errorf("upload test timed out after %d seconds", s.config.Timeout)
			}
			return nil, fmt.Errorf("upload test failed: %w", err)
		}

		// After the upload test is complete, set the final upload speed
		result.UploadSpeed = selectedServer.ULSpeed.Mbps()

		// Broadcast upload completion only after the upload test is done
		if s.broadcastUpdate != nil {
			s.broadcastUpdate(types.SpeedUpdate{
				Type:        "upload",
				ServerName:  selectedServer.Name,
				Speed:       result.UploadSpeed,
				Progress:    100,
				IsComplete:  true,
				IsScheduled: opts.IsScheduled,
			})
		}

		// Send final completion update with both speeds
		if s.broadcastUpdate != nil {
			s.broadcastUpdate(types.SpeedUpdate{
				Type:        "complete",
				ServerName:  selectedServer.Name,
				Speed:       result.UploadSpeed,
				Progress:    100,
				IsComplete:  true,
				IsScheduled: opts.IsScheduled,
			})
		}
	}

	log.Info().
		Str("server", selectedServer.Name).
		Str("server_host", selectedServer.Host).
		Str("server_country", selectedServer.Country).
		Str("provider", selectedServer.Sponsor).
		Str("server_url", selectedServer.URL).
		Str("latency", result.Latency).
		Float64("packet_loss", result.PacketLoss).
		Float64("download_mbps", result.DownloadSpeed).
		Float64("upload_mbps", result.UploadSpeed).
		Msg("Speed test complete")

	selectedServer.Context.Reset()

	// Update the database save operation
	jitterFloat := selectedServer.Jitter.Seconds() * 1000
	dbResult, err := s.db.SaveSpeedTest(context.Background(), types.SpeedTestResult{
		ServerName:    selectedServer.Name,
		ServerID:      selectedServer.ID,
		ServerHost:    &selectedServer.Host,
		TestType:      "speedtest",
		DownloadSpeed: result.DownloadSpeed,
		UploadSpeed:   result.UploadSpeed,
		Latency:       result.Latency,
		PacketLoss:    result.PacketLoss,
		Jitter:        &jitterFloat,
		IsScheduled:   opts.IsScheduled,
	})
	if err != nil {
		log.Error().Err(err).Msg("Failed to save result to database")
	}

	if dbResult != nil {
		result.ID = dbResult.ID
		s.notifier.SendNotification(dbResult)
	}

	return result, nil
}

func (s *service) GetServers(testType string) ([]ServerResponse, error) {
	if testType == "librespeed" {
		return s.GetLibrespeedServers()
	}

	log.Trace().
		Int("cache_size", len(s.serverCache)).
		Time("cache_expiry", s.cacheExpiry).
		Bool("cache_valid", time.Now().Before(s.cacheExpiry)).
		Msg("Checking cache status")

	if len(s.serverCache) > 0 && time.Now().Before(s.cacheExpiry) {
		log.Debug().
			Int("server_count", len(s.serverCache)).
			Time("cache_expiry", s.cacheExpiry).
			Msg("Returning cached speedtest servers")
		return s.serverCache, nil
	}

	log.Debug().Msg("Cache miss, fetching fresh speedtest servers")

	// FetchUserInfo returns information about caller determined by speedtest.net
	_, err := s.client.FetchUserInfo()
	if err != nil {
		log.Error().Err(err).Msg("Failed to fetch speedtest user info")
		return nil, fmt.Errorf("failed to fetch user info: %w", err)
	}

	// Fetch servers using the initialized client
	serverList, err := s.client.FetchServers()
	if err != nil {
		log.Error().Err(err).Msg("Failed to fetch speedtest servers")
		return nil, fmt.Errorf("failed to fetch servers: %w", err)
	}

	availableServers := serverList.Available()
	if availableServers == nil {
		log.Error().Msg("No available speedtest servers found")
		return nil, fmt.Errorf("no available servers found")
	}

	// Convert to response format
	response := make([]ServerResponse, len(*availableServers))
	for i, server := range *availableServers {
		lat, _ := strconv.ParseFloat(server.Lat, 64)
		lon, _ := strconv.ParseFloat(server.Lon, 64)

		response[i] = ServerResponse{
			ID:           server.ID,
			Name:         server.Name,
			Host:         server.Host,
			Distance:     server.Distance,
			Country:      server.Country,
			Sponsor:      server.Sponsor,
			URL:          server.URL,
			Lat:          lat,
			Lon:          lon,
			IsIperf:      false,
			IsLibrespeed: false,
		}
	}

	// Sort by distance
	sort.Slice(response, func(i, j int) bool {
		return response[i].Distance < response[j].Distance
	})

	// Update cache
	s.serverCache = response
	s.cacheExpiry = time.Now().Add(s.cacheDuration)

	log.Debug().
		Int("server_count", len(response)).
		Time("cache_expiry", s.cacheExpiry).
		Msg("Retrieved and cached speedtest servers")

	return response, nil
}
