// Copyright (c) 2024-2026, s0up and the autobrr contributors.
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

	"github.com/autobrr/netronome/internal/config"
	"github.com/autobrr/netronome/internal/types"
)

const (
	// speedtest-go defaults its parallel stream count to runtime.NumCPU(), which
	// makes a result depend on how many cores the host has instead of on the
	// line. A single TCP stream cannot fill a fast or high-latency path, so a
	// 4-core box measured about half the upload of a 16-core box on the same
	// line. Pin the count so every host runs the same test. On a path where one
	// stream already saturates the line the extra streams change nothing, and
	// they cost almost no CPU.
	speedtestStreams = 16

	// Ookla reports the same distance for every server in a city, so sorting by
	// distance alone leaves a large tie that resolves arbitrarily. Servers this
	// close to the nearest one count as tied and the tie goes to the lowest
	// latency. Without the limit a distant server with a quick ping would beat a
	// much closer one.
	speedtestDistanceTieKm = 5.0
)

type SpeedtestNetRunner struct {
	// speedtest-go keeps its counters on the client, so two tests sharing one
	// client corrupt each other. Tests queue for this slot rather than being
	// rejected, but a caller that gives up while queued leaves without starting
	// a test its deadline has already passed.
	running          chan struct{}
	client           *st.Speedtest
	config           config.SpeedTestConfig
	progressCallback func(types.SpeedUpdate)
	serverCache      []ServerResponse
	cacheExpiry      time.Time
	cacheDuration    time.Duration
}

func NewSpeedtestNetRunner(cfg config.SpeedTestConfig) *SpeedtestNetRunner {
	return &SpeedtestNetRunner{
		running:       make(chan struct{}, 1),
		client:        st.New(st.WithUserConfig(&st.UserConfig{MaxConnections: speedtestStreams})),
		config:        cfg,
		cacheDuration: 30 * time.Minute,
		cacheExpiry:   time.Now(),
	}
}

// selectNearestServer returns the closest server, breaking distance ties on the
// latency that FetchServers() already measured. Latency is st.PingTimeout (-1)
// when that ping failed, so only positive values count. Sorts in place.
func selectNearestServer(servers []*st.Server) *st.Server {
	if len(servers) == 0 {
		return nil
	}

	sort.Slice(servers, func(i, j int) bool {
		return servers[i].Distance < servers[j].Distance
	})

	best := servers[0]
	for _, server := range servers[1:] {
		if server.Distance > servers[0].Distance+speedtestDistanceTieKm {
			break
		}
		if server.Latency > 0 && (best.Latency <= 0 || server.Latency < best.Latency) {
			best = server
		}
	}
	return best
}

func (r *SpeedtestNetRunner) GetTestType() string {
	return "speedtest"
}

func (r *SpeedtestNetRunner) SetProgressCallback(callback func(types.SpeedUpdate)) {
	r.progressCallback = callback
}

func (r *SpeedtestNetRunner) RunTest(ctx context.Context, opts *types.TestOptions) (*Result, error) {
	select {
	case r.running <- struct{}{}:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	defer func() { <-r.running }()

	// Deferred so the early returns below cannot leave stale state on the
	// client. Registered second, so it runs before the slot is released.
	defer r.client.Reset()

	log.Debug().
		Bool("isScheduled", opts.IsScheduled).
		Str("server_ids", fmt.Sprintf("%v", opts.ServerIDs)).
		Msg("Starting speedtest.net test")

	serverList, err := r.client.FetchServers()
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

		// Server not in FetchServers() list — try direct lookup by ID
		if selectedServer == nil {
			for _, requestedID := range opts.ServerIDs {
				server, err := r.client.FetchServerByIDContext(ctx, requestedID)
				if err == nil && server != nil {
					log.Info().
						Str("server_id", requestedID).
						Str("server_name", server.Name).
						Msg("Server not in public list, fetched directly by ID")
					selectedServer = server
					break
				}
				log.Debug().Err(err).Str("server_id", requestedID).Msg("Failed to fetch server by ID")
			}
		}
	}

	if selectedServer == nil {
		if len(opts.ServerIDs) > 0 {
			return nil, fmt.Errorf("requested server(s) %v not found in public list or by direct lookup", opts.ServerIDs)
		}
		selectedServer = selectNearestServer(serverList)
		if selectedServer == nil {
			return nil, fmt.Errorf("no speedtest servers available")
		}
	}

	log.Info().
		Str("server_ids", fmt.Sprintf("%v", opts.ServerIDs)).
		Str("server_name", selectedServer.Name).
		Str("server_host", selectedServer.Host).
		Str("server_country", selectedServer.Country).
		Str("provider", selectedServer.Sponsor).
		Bool("enable_download", opts.EnableDownload).
		Bool("enable_upload", opts.EnableUpload).
		Msg("Starting speedtest.net test")

	result := &Result{
		Timestamp: time.Now(),
		Server:    selectedServer.Sponsor, // Use provider/sponsor instead of city name
	}

	if err := selectedServer.PingTest(func(latency time.Duration) {
		if r.progressCallback != nil {
			r.progressCallback(types.SpeedUpdate{
				Type:        "ping",
				ServerName:  selectedServer.Sponsor, // Use provider/sponsor instead of city name
				Latency:     latency.String(),
				Progress:    100,
				IsComplete:  false,
				IsScheduled: opts.IsScheduled,
				TestType:    "speedtest",
			})
		}
	}); err != nil {
		result.Error = fmt.Sprintf("ping test failed: %v", err)
		return result, err
	}
	result.Latency = selectedServer.Latency.String()

	if r.progressCallback != nil {
		r.progressCallback(types.SpeedUpdate{
			Type:        "ping",
			ServerName:  selectedServer.Sponsor, // Use provider/sponsor instead of city name
			Latency:     selectedServer.Latency.String(),
			Progress:    100,
			IsComplete:  false,
			IsScheduled: opts.IsScheduled,
			TestType:    "speedtest",
		})
	}

	if opts.EnableDownload {
		var downloadStartTime time.Time
		var progress float64
		var lastUpdate atomic.Int64

		selectedServer.Context.SetCallbackDownload(func(speed st.ByteRate) {
			if downloadStartTime.IsZero() {
				downloadStartTime = time.Now()
			}

			now := time.Now().Unix()
			lastUpdateTime := lastUpdate.Load()

			if now-lastUpdateTime >= 1 {
				elapsed := time.Since(downloadStartTime).Seconds()
				progress = math.Min(100, (elapsed/10.0)*100)

				if progress > 0 && r.progressCallback != nil {
					r.progressCallback(types.SpeedUpdate{
						Type:        "download",
						ServerName:  selectedServer.Sponsor, // Use provider/sponsor instead of city name
						Speed:       speed.Mbps(),
						Progress:    progress,
						IsComplete:  progress >= 100,
						IsScheduled: opts.IsScheduled,
						TestType:    "speedtest",
					})
					lastUpdate.Store(now)
				}
			}
		})

		timeout := time.Duration(r.config.Timeout) * time.Second
		ctxTimeout, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		if err := selectedServer.DownloadTestContext(ctxTimeout); err != nil {
			if ctxTimeout.Err() == context.DeadlineExceeded {
				return nil, fmt.Errorf("download test timed out after %d seconds", r.config.Timeout)
			}
			return nil, fmt.Errorf("download test failed: %w", err)
		}

		result.DownloadSpeed = selectedServer.DLSpeed.Mbps()

		if r.progressCallback != nil {
			r.progressCallback(types.SpeedUpdate{
				Type:        "download",
				ServerName:  selectedServer.Sponsor, // Use provider/sponsor instead of city name
				Speed:       result.DownloadSpeed,
				Progress:    100,
				IsComplete:  true,
				IsScheduled: opts.IsScheduled,
				TestType:    "speedtest",
			})
		}
	}

	if opts.EnableUpload {
		var uploadStartTime time.Time
		var progress float64
		var lastUpdate atomic.Int64

		selectedServer.Context.SetCallbackUpload(func(speed st.ByteRate) {
			if uploadStartTime.IsZero() {
				uploadStartTime = time.Now()
			}

			now := time.Now().Unix()
			lastUpdateTime := lastUpdate.Load()

			if now-lastUpdateTime >= 1 {
				elapsed := time.Since(uploadStartTime).Seconds()
				progress = math.Min(100, (elapsed/10.0)*100)

				if progress > 0 && r.progressCallback != nil {
					r.progressCallback(types.SpeedUpdate{
						Type:        "upload",
						ServerName:  selectedServer.Sponsor, // Use provider/sponsor instead of city name
						Speed:       speed.Mbps(),
						Progress:    progress,
						IsComplete:  progress >= 100,
						IsScheduled: opts.IsScheduled,
						TestType:    "speedtest",
					})
					lastUpdate.Store(now)
				}
			}
		})

		timeout := time.Duration(r.config.Timeout) * time.Second
		uploadCtx, uploadCancel := context.WithTimeout(context.Background(), timeout)
		defer uploadCancel()

		if err := selectedServer.UploadTestContext(uploadCtx); err != nil {
			if uploadCtx.Err() == context.DeadlineExceeded {
				return nil, fmt.Errorf("upload test timed out after %d seconds", r.config.Timeout)
			}
			return nil, fmt.Errorf("upload test failed: %w", err)
		}

		result.UploadSpeed = selectedServer.ULSpeed.Mbps()

		if r.progressCallback != nil {
			r.progressCallback(types.SpeedUpdate{
				Type:        "upload",
				ServerName:  selectedServer.Sponsor, // Use provider/sponsor instead of city name
				Speed:       result.UploadSpeed,
				Progress:    100,
				IsComplete:  true,
				IsScheduled: opts.IsScheduled,
				TestType:    "speedtest",
			})
		}

		if r.progressCallback != nil {
			r.progressCallback(types.SpeedUpdate{
				Type:        "complete",
				ServerName:  selectedServer.Sponsor, // Use provider/sponsor instead of city name
				Speed:       result.UploadSpeed,
				Progress:    100,
				IsComplete:  true,
				IsScheduled: opts.IsScheduled,
				TestType:    "speedtest",
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
		Float64("download_mbps", result.DownloadSpeed).
		Float64("upload_mbps", result.UploadSpeed).
		Msg("Speedtest.net test complete")

	jitterFloat := selectedServer.Jitter.Seconds() * 1000
	result.Jitter = jitterFloat

	return result, nil
}

func (r *SpeedtestNetRunner) GetServers() ([]ServerResponse, error) {
	log.Trace().
		Int("cache_size", len(r.serverCache)).
		Time("cache_expiry", r.cacheExpiry).
		Bool("cache_valid", time.Now().Before(r.cacheExpiry)).
		Msg("Checking cache status")

	if len(r.serverCache) > 0 && time.Now().Before(r.cacheExpiry) {
		log.Debug().
			Int("server_count", len(r.serverCache)).
			Time("cache_expiry", r.cacheExpiry).
			Msg("Returning cached speedtest servers")
		return r.serverCache, nil
	}

	log.Debug().Msg("Cache miss, fetching fresh speedtest servers")

	_, err := r.client.FetchUserInfo()
	if err != nil {
		log.Error().Err(err).Msg("Failed to fetch speedtest user info")
		return nil, fmt.Errorf("failed to fetch user info: %w", err)
	}

	serverList, err := r.client.FetchServers()
	if err != nil {
		log.Error().Err(err).Msg("Failed to fetch speedtest servers")
		return nil, fmt.Errorf("failed to fetch servers: %w", err)
	}

	// Available() drops servers whose HTTP ping failed during FetchServers().
	// In restricted networks (e.g. Docker) all pings time out, leaving an empty
	// (non-nil) slice, so fall back to the unfiltered list to keep the selectable
	// server list populated.
	availableServers := serverList.Available()
	if availableServers == nil || len(*availableServers) == 0 {
		log.Warn().Msg("No pingable speedtest servers, falling back to unfiltered server list")
		availableServers = &serverList
	}

	if len(*availableServers) == 0 {
		log.Error().Msg("No speedtest servers found")
		return nil, fmt.Errorf("no speedtest servers found")
	}

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

	sort.Slice(response, func(i, j int) bool {
		return response[i].Distance < response[j].Distance
	})

	r.serverCache = response
	r.cacheExpiry = time.Now().Add(r.cacheDuration)

	log.Debug().
		Int("server_count", len(response)).
		Time("cache_expiry", r.cacheExpiry).
		Msg("Retrieved and cached speedtest servers")

	return response, nil
}
