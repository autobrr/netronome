// Copyright (c) 2024-2026, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package speedtest

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"maps"
	"math"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rs/zerolog/log"
	st "github.com/showwin/speedtest-go/speedtest"

	"github.com/autobrr/netronome/internal/config"
	"github.com/autobrr/netronome/internal/types"
)

// SpeedtestNetRunner executes Speedtest.net tests and caches discovered server catalogues.
type SpeedtestNetRunner struct {
	client           *st.Speedtest
	config           config.SpeedTestConfig
	progressCallback func(types.SpeedUpdate)
	cacheMu          sync.RWMutex
	serverCache      map[string]serverCacheEntry
	userLocation     *ServerLocation
	globalFetch      chan struct{}
	cacheDuration    time.Duration
	fetchServers     serverFetcher
	globalLocations  map[string]ServerLocation
}

type serverFetcher func(context.Context, *ServerLocation) ([]ServerResponse, *ServerLocation, error)

type serverCacheEntry struct {
	servers   []ServerResponse
	expiresAt time.Time
}

// NewSpeedtestNetRunner creates a runner with bounded, 30-minute server catalogue caching.
func NewSpeedtestNetRunner(cfg config.SpeedTestConfig) *SpeedtestNetRunner {
	runner := &SpeedtestNetRunner{
		client:          st.New(),
		config:          cfg,
		serverCache:     make(map[string]serverCacheEntry),
		globalFetch:     make(chan struct{}, 1),
		cacheDuration:   30 * time.Minute,
		fetchServers:    fetchSpeedtestServers,
		globalLocations: make(map[string]ServerLocation, len(st.Locations)),
	}
	for name, location := range st.Locations {
		if location != nil {
			runner.globalLocations[name] = ServerLocation{Latitude: location.Lat, Longitude: location.Lon}
		}
	}
	return runner
}

func (r *SpeedtestNetRunner) GetTestType() string {
	return "speedtest"
}

func (r *SpeedtestNetRunner) SetProgressCallback(callback func(types.SpeedUpdate)) {
	r.progressCallback = callback
}

func (r *SpeedtestNetRunner) RunTest(ctx context.Context, opts *types.TestOptions) (*Result, error) {
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
		slices.SortFunc(serverList, func(a, b *st.Server) int {
			return cmp.Compare(a.Distance, b.Distance)
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
		Msg("Starting speedtest.net test")

	result := &Result{
		Timestamp:  time.Now(),
		Server:     selectedServer.Sponsor, // Use provider/sponsor instead of city name
		ServerID:   selectedServer.ID,
		ServerHost: selectedServer.Host,
		ServerCity: selectedServer.Name,
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

	selectedServer.Context.Reset()

	jitterFloat := selectedServer.Jitter.Seconds() * 1000
	result.Jitter = jitterFloat

	return result, nil
}

// GetServers returns servers near the runner's detected location.
func (r *SpeedtestNetRunner) GetServers() ([]ServerResponse, error) {
	return r.GetServersWithOptions(context.Background(), ServerListOptions{})
}

// GetServersWithOptions returns the local, global, or coordinate-based Speedtest.net catalogue.
func (r *SpeedtestNetRunner) GetServersWithOptions(ctx context.Context, options ServerListOptions) ([]ServerResponse, error) {
	if options.Global && options.Location != nil {
		return nil, fmt.Errorf("global and coordinate server searches are mutually exclusive")
	}
	if options.Global {
		return r.getGlobalServers(ctx)
	}
	if options.Location != nil {
		key := "location:" + strconv.FormatFloat(options.Location.Latitude, 'f', -1, 64) + "," +
			strconv.FormatFloat(options.Location.Longitude, 'f', -1, 64)
		return r.getServersForLocation(ctx, key, options.Location)
	}
	return r.getServersForLocation(ctx, "local", nil)
}

func (r *SpeedtestNetRunner) getServersForLocation(ctx context.Context, key string, location *ServerLocation) ([]ServerResponse, error) {
	if servers, ok := r.loadServerCache(key); ok {
		return servers, nil
	}

	servers, userLocation, err := r.fetchServers(ctx, location)
	if err != nil {
		return nil, err
	}
	r.storeServerCache(key, servers, userLocation)

	log.Debug().
		Str("cache_key", key).
		Int("server_count", len(servers)).
		Msg("Retrieved and cached speedtest servers")
	return slices.Clone(servers), nil
}

func (r *SpeedtestNetRunner) getGlobalServers(ctx context.Context) ([]ServerResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if servers, ok := r.loadServerCache("global"); ok {
		return servers, nil
	}

	select {
	case r.globalFetch <- struct{}{}:
		defer func() { <-r.globalFetch }()
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if servers, ok := r.loadServerCache("global"); ok {
		return servers, nil
	}

	localServers, err := r.getServersForLocation(ctx, "local", nil)
	if err != nil {
		return nil, fmt.Errorf("fetch local speedtest servers: %w", err)
	}
	userLocation, ok := r.loadUserLocation()
	if !ok {
		return nil, fmt.Errorf("speedtest user location is unavailable")
	}

	locationNames := slices.Sorted(maps.Keys(r.globalLocations))
	allServers := slices.Clone(localServers)
	var failures []error
	successfulLocations := 0
	var resultMu sync.Mutex
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 4)

	for _, name := range locationNames {
		coordinates := r.globalLocations[name]
		wg.Go(func() {
			select {
			case semaphore <- struct{}{}:
				defer func() { <-semaphore }()
			case <-ctx.Done():
				resultMu.Lock()
				failures = append(failures, ctx.Err())
				resultMu.Unlock()
				return
			}

			servers, _, fetchErr := r.fetchServers(ctx, &coordinates)
			resultMu.Lock()
			defer resultMu.Unlock()
			if fetchErr != nil {
				failures = append(failures, fmt.Errorf("fetch servers near %s: %w", name, fetchErr))
				return
			}
			successfulLocations++
			allServers = append(allServers, servers...)
		})
	}
	wg.Wait()

	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if len(failures) > 0 {
		log.Warn().Err(errors.Join(failures...)).Msg("Some global speedtest locations could not be fetched")
	}
	if successfulLocations == 0 {
		if err := errors.Join(failures...); err != nil {
			return nil, fmt.Errorf("fetch global speedtest servers: %w", err)
		}
		return nil, fmt.Errorf("no global speedtest locations configured")
	}

	servers := mergeServerLists(userLocation, allServers)
	if len(servers) == 0 {
		if err := errors.Join(failures...); err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("no speedtest servers found")
	}
	if len(failures) == 0 {
		r.storeServerCache("global", servers, nil)
	}

	log.Info().
		Int("server_count", len(servers)).
		Int("locations", len(locationNames)).
		Bool("cached", len(failures) == 0).
		Msg("Retrieved global speedtest servers")
	return slices.Clone(servers), nil
}

func fetchSpeedtestServers(ctx context.Context, location *ServerLocation) ([]ServerResponse, *ServerLocation, error) {
	client := st.New()
	var userLocation *ServerLocation
	if location == nil {
		user, err := client.FetchUserInfoContext(ctx)
		if err != nil {
			return nil, nil, fmt.Errorf("fetch speedtest user info: %w", err)
		}
		latitude, latErr := strconv.ParseFloat(user.Lat, 64)
		longitude, lonErr := strconv.ParseFloat(user.Lon, 64)
		if latErr != nil || lonErr != nil {
			return nil, nil, fmt.Errorf("parse speedtest user location: %w", errors.Join(latErr, lonErr))
		}
		userLocation = &ServerLocation{Latitude: latitude, Longitude: longitude}
	} else {
		client = st.New(st.WithUserConfig(&st.UserConfig{Location: &st.Location{
			Name: "custom",
			Lat:  location.Latitude,
			Lon:  location.Longitude,
		}}))
	}

	serverList, err := client.FetchServerListContext(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("fetch speedtest servers: %w", err)
	}
	servers := serverResponses(serverList)
	if len(servers) == 0 {
		return nil, nil, fmt.Errorf("no speedtest servers found")
	}
	return servers, userLocation, nil
}

func serverResponses(serverList st.Servers) []ServerResponse {
	availableServers := serverList.Available()
	if availableServers == nil || len(*availableServers) == 0 {
		log.Warn().Msg("No pingable speedtest servers, falling back to unfiltered server list")
		availableServers = &serverList
	}

	response := make([]ServerResponse, 0, len(*availableServers))
	for _, server := range *availableServers {
		lat, _ := strconv.ParseFloat(server.Lat, 64)
		lon, _ := strconv.ParseFloat(server.Lon, 64)
		response = append(response, ServerResponse{
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
		})
	}
	slices.SortFunc(response, func(a, b ServerResponse) int {
		return cmp.Compare(a.Distance, b.Distance)
	})
	return response
}

func mergeServerLists(origin ServerLocation, servers []ServerResponse) []ServerResponse {
	unique := make(map[string]ServerResponse, len(servers))
	for _, server := range servers {
		server.Distance = haversineDistance(origin.Latitude, origin.Longitude, server.Lat, server.Lon)
		unique[server.ID] = server
	}

	merged := slices.Collect(maps.Values(unique))
	slices.SortFunc(merged, func(a, b ServerResponse) int {
		return cmp.Compare(a.Distance, b.Distance)
	})
	return merged
}

func haversineDistance(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadiusKM = 6371
	lat1Rad := lat1 * math.Pi / 180
	lat2Rad := lat2 * math.Pi / 180
	deltaLat := (lat2 - lat1) * math.Pi / 180
	deltaLon := (lon2 - lon1) * math.Pi / 180

	a := math.Sin(deltaLat/2)*math.Sin(deltaLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*math.Sin(deltaLon/2)*math.Sin(deltaLon/2)
	return earthRadiusKM * 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
}

func (r *SpeedtestNetRunner) loadServerCache(key string) ([]ServerResponse, bool) {
	r.cacheMu.RLock()
	entry, ok := r.serverCache[key]
	r.cacheMu.RUnlock()
	if !ok || time.Now().After(entry.expiresAt) {
		return nil, false
	}
	return slices.Clone(entry.servers), true
}

func (r *SpeedtestNetRunner) storeServerCache(key string, servers []ServerResponse, userLocation *ServerLocation) {
	r.cacheMu.Lock()
	defer r.cacheMu.Unlock()
	now := time.Now()
	for cacheKey, entry := range r.serverCache {
		if !now.Before(entry.expiresAt) {
			delete(r.serverCache, cacheKey)
		}
	}

	const maxCoordinateCacheEntries = 32
	if strings.HasPrefix(key, "location:") {
		coordinateEntries := 0
		oldestKey := ""
		var oldestExpiry time.Time
		for cacheKey, entry := range r.serverCache {
			if cacheKey == key || !strings.HasPrefix(cacheKey, "location:") {
				continue
			}
			coordinateEntries++
			if oldestKey == "" || entry.expiresAt.Before(oldestExpiry) {
				oldestKey = cacheKey
				oldestExpiry = entry.expiresAt
			}
		}
		if coordinateEntries >= maxCoordinateCacheEntries {
			delete(r.serverCache, oldestKey)
		}
	}

	r.serverCache[key] = serverCacheEntry{
		servers:   slices.Clone(servers),
		expiresAt: now.Add(r.cacheDuration),
	}
	if userLocation != nil {
		location := *userLocation
		r.userLocation = &location
	}
}

func (r *SpeedtestNetRunner) loadUserLocation() (ServerLocation, bool) {
	r.cacheMu.RLock()
	defer r.cacheMu.RUnlock()
	if r.userLocation == nil {
		return ServerLocation{}, false
	}
	return *r.userLocation, true
}
