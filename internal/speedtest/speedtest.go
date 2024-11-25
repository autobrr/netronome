// Copyright (c) 2024, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package speedtest

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strconv"
	"time"

	"github.com/rs/zerolog/log"
	st "github.com/showwin/speedtest-go/speedtest"

	"github.com/autobrr/netronome/internal/database"
	"github.com/autobrr/netronome/internal/types"
)

type Service interface {
	RunTest(opts *types.TestOptions) (*Result, error)
	GetServers() ([]ServerResponse, error)
}

type service struct {
	client *st.Speedtest
	server *Server
	db     database.Service
}

func New(server *Server, db database.Service) Service {
	return &service{
		client: st.New(),
		server: server,
		db:     db,
	}
}

func (s *service) RunTest(opts *types.TestOptions) (*Result, error) {
	log.Debug().
		Bool("isScheduled", opts.IsScheduled).
		Str("server_ids", fmt.Sprintf("%v", opts.ServerIDs)).
		Msg("Starting speed test")

	serverList, err := s.client.FetchServers()
	if err != nil {
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
		if s.server.BroadcastUpdate != nil {
			s.server.BroadcastUpdate(types.SpeedUpdate{
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

	if s.server.BroadcastUpdate != nil {
		s.server.BroadcastUpdate(types.SpeedUpdate{
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

		selectedServer.Context.SetCallbackDownload(func(speed st.ByteRate) {
			if downloadStartTime.IsZero() {
				downloadStartTime = time.Now()
			}

			// Update progress more frequently (every ~100ms)
			elapsed := time.Since(downloadStartTime).Seconds()
			progress = math.Min(100, (elapsed/10.0)*100)

			// Ensure we're not sending too many updates
			if progress > 0 && s.server.BroadcastUpdate != nil {
				s.server.BroadcastUpdate(types.SpeedUpdate{
					Type:        "download",
					ServerName:  selectedServer.Name,
					Speed:       speed.Mbps(),
					Progress:    progress,
					IsComplete:  progress >= 100,
					IsScheduled: opts.IsScheduled,
				})
			}
		})

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := selectedServer.DownloadTestContext(ctx); err != nil {
			return nil, fmt.Errorf("download test failed: %w", err)
		}

		result.DownloadSpeed = selectedServer.DLSpeed.Mbps()

		// Broadcast download completion
		if s.server.BroadcastUpdate != nil {
			s.server.BroadcastUpdate(types.SpeedUpdate{
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

		selectedServer.Context.SetCallbackUpload(func(speed st.ByteRate) {
			if uploadStartTime.IsZero() {
				uploadStartTime = time.Now()
			}

			// Update progress more frequently (every ~100ms)
			elapsed := time.Since(uploadStartTime).Seconds()
			progress = math.Min(100, (elapsed/10.0)*100)

			// Ensure we're not sending too many updates
			if progress > 0 && s.server.BroadcastUpdate != nil {
				s.server.BroadcastUpdate(types.SpeedUpdate{
					Type:        "upload",
					ServerName:  selectedServer.Name,
					Speed:       speed.Mbps(),
					Progress:    progress,
					IsComplete:  progress >= 100,
					IsScheduled: opts.IsScheduled,
				})
			}
		})

		// Perform the upload test
		if err := selectedServer.UploadTest(); err != nil {
			return nil, fmt.Errorf("upload test failed: %w", err)
		}

		// After the upload test is complete, set the final upload speed
		result.UploadSpeed = selectedServer.ULSpeed.Mbps()

		// Broadcast upload completion only after the upload test is done
		if s.server.BroadcastUpdate != nil {
			s.server.BroadcastUpdate(types.SpeedUpdate{
				Type:        "upload",
				ServerName:  selectedServer.Name,
				Speed:       result.UploadSpeed,
				Progress:    100,
				IsComplete:  true,
				IsScheduled: opts.IsScheduled,
			})
		}

		// Send final completion update
		if s.server.BroadcastUpdate != nil {
			s.server.BroadcastUpdate(types.SpeedUpdate{
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
		DownloadSpeed: result.DownloadSpeed,
		UploadSpeed:   result.UploadSpeed,
		Latency:       result.Latency,
		PacketLoss:    result.PacketLoss,
		Jitter:        &jitterFloat,
	})
	if err != nil {
		log.Error().Err(err).Msg("Failed to save result to database")
	}

	if dbResult != nil {
		result.ID = dbResult.ID
	}

	return result, nil
}

func (s *service) GetServers() ([]ServerResponse, error) {
	// Create new speedtest client
	client := st.New()

	// Get user info first to initialize the client
	_, err := client.FetchUserInfo()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user info: %w", err)
	}

	// Fetch servers using the initialized client
	serverList, err := client.FetchServers()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch servers: %w", err)
	}

	availableServers := serverList.Available()
	if availableServers == nil {
		return nil, fmt.Errorf("no available servers found")
	}

	// Convert to response format
	response := make([]ServerResponse, len(*availableServers))
	for i, server := range *availableServers {
		lat, _ := strconv.ParseFloat(server.Lat, 64)
		lon, _ := strconv.ParseFloat(server.Lon, 64)

		response[i] = ServerResponse{
			ID:       server.ID,
			Name:     server.Name,
			Host:     server.Host,
			Distance: server.Distance,
			Country:  server.Country,
			Sponsor:  server.Sponsor,
			URL:      server.URL,
			Lat:      lat,
			Lon:      lon,
		}
	}

	// Sort by distance
	sort.Slice(response, func(i, j int) bool {
		return response[i].Distance < response[j].Distance
	})

	log.Info().Int("server_count", len(response)).Msg("Retrieved speedtest servers")
	return response, nil
}
