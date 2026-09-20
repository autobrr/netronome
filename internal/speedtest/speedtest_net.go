// Copyright (c) 2024-2026, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package speedtest

import (
	"cmp"
	"context"
	"encoding/json"
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
	"golang.org/x/sync/singleflight"

	"github.com/autobrr/netronome/internal/config"
	"github.com/autobrr/netronome/internal/database"
	"github.com/autobrr/netronome/internal/types"
)

const (
	// serverCatalogueFetchTimeout bounds each upstream server discovery request.
	serverCatalogueFetchTimeout     = 30 * time.Second
	serverCatalogueStoreTimeout     = 30 * time.Second
	serverCatalogueSettingKey       = "speedtest_retained_servers"
	serverCatalogueCoordinatePrefix = "location:"
	maxCoordinateCatalogueEntries   = 32
)

// serverCatalogueStore reads and atomically replaces the durable catalogue value.
type serverCatalogueStore interface {
	GetAppSetting(ctx context.Context, key string) (string, error)
	SetAppSetting(ctx context.Context, key, value string) error
}

// SpeedtestNetRunner executes Speedtest.net tests, coalesces discovery, and persistently retains servers.
type SpeedtestNetRunner struct {
	client              *st.Speedtest
	config              config.SpeedTestConfig
	progressCallback    func(types.SpeedUpdate)
	cacheMu             sync.RWMutex
	persistMu           sync.Mutex
	retainedServers     map[string]ServerResponse
	catalogueSources    map[string]time.Time
	catalogueStore      serverCatalogueStore
	catalogueLoadFailed atomic.Bool
	userLocation        *ServerLocation
	pendingCatalogue    *persistedServerCatalogue
	hasPendingWrite     atomic.Bool
	globalFetch         chan struct{}
	serverFetches       singleflight.Group
	catalogueRetries    singleflight.Group
	cacheDuration       time.Duration
	fetchServers        serverFetcher
	globalLocations     map[string]ServerLocation
}

type serverFetcher func(context.Context, *ServerLocation) ([]ServerResponse, *ServerLocation, error)

type persistedServerCatalogue struct {
	Servers      []ServerResponse     `json:"servers"`
	UserLocation *ServerLocation      `json:"userLocation,omitempty"`
	Sources      map[string]time.Time `json:"sources,omitempty"`
}

// NewSpeedtestNetRunner creates a runner with 30-minute source caching. A non-nil store loads and
// durably retains the catalogue across runner restarts; load failures are logged and retried before use.
func NewSpeedtestNetRunner(cfg config.SpeedTestConfig, store serverCatalogueStore) *SpeedtestNetRunner {
	runner := &SpeedtestNetRunner{
		client:           st.New(),
		config:           cfg,
		retainedServers:  make(map[string]ServerResponse),
		catalogueSources: make(map[string]time.Time),
		catalogueStore:   store,
		globalFetch:      make(chan struct{}, 1),
		cacheDuration:    30 * time.Minute,
		fetchServers:     fetchSpeedtestServers,
		globalLocations:  make(map[string]ServerLocation, len(st.Locations)),
	}
	for name, location := range st.Locations {
		if location != nil {
			runner.globalLocations[name] = ServerLocation{Latitude: location.Lat, Longitude: location.Lon}
		}
	}
	loadCtx, cancel := context.WithTimeout(context.Background(), serverCatalogueStoreTimeout)
	err := runner.loadPersistedServers(loadCtx)
	cancel()
	if err != nil {
		runner.catalogueLoadFailed.Store(true)
		log.Error().Err(err).Msg("Failed to load retained speedtest servers")
	}
	return runner
}

func (r *SpeedtestNetRunner) SetProgressCallback(callback func(types.SpeedUpdate)) {
	r.progressCallback = callback
}

// RunTest executes a Speedtest.net test against a requested server or the nearest available server.
// Requested IDs omitted from the public list are looked up directly before the test is rejected.
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
		if len(serverList) == 0 {
			return nil, fmt.Errorf("no speedtest servers available")
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

// GetServersWithOptions returns the durable Speedtest.net catalogue. An empty catalogue fetches
// from the selected source, while Refresh always fetches and persists before returning the update.
// Ordinary reads serve committed data immediately and retry pending persistence in the background.
// Concurrent fetches for one local or coordinate source share work while each waiter observes its
// own context.
func (r *SpeedtestNetRunner) GetServersWithOptions(ctx context.Context, options ServerListOptions) ([]ServerResponse, error) {
	if options.Global && options.Location != nil {
		return nil, fmt.Errorf("global and coordinate server searches are mutually exclusive")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := r.ensureCatalogueLoaded(ctx); err != nil {
		return nil, err
	}
	if !options.Refresh && r.catalogueStore != nil {
		r.retryPendingCatalogueAsync(ctx)
		if servers := r.loadRetainedServers(options.Location); len(servers) > 0 {
			return servers, nil
		}
		if r.hasPendingWrite.Load() {
			return nil, fmt.Errorf("no committed speedtest servers are available while a catalogue write is pending")
		}
	}
	if options.Global {
		return r.getGlobalServers(ctx, options.Refresh)
	}
	if options.Location != nil {
		key := serverCatalogueSourceKey(options)
		return r.getServersForLocation(ctx, key, options.Location, options.Refresh)
	}
	return r.getServersForLocation(ctx, "local", nil, options.Refresh)
}

// GetServerCatalogueStatus returns durable fetch metadata without contacting Speedtest.net.
func (r *SpeedtestNetRunner) GetServerCatalogueStatus(ctx context.Context, options ServerListOptions) (ServerCatalogueStatus, error) {
	if options.Global && options.Location != nil {
		return ServerCatalogueStatus{}, fmt.Errorf("global and coordinate server searches are mutually exclusive")
	}
	if err := ctx.Err(); err != nil {
		return ServerCatalogueStatus{}, err
	}
	if err := r.ensureCatalogueLoaded(ctx); err != nil {
		return ServerCatalogueStatus{}, err
	}
	if r.catalogueStore != nil {
		r.retryPendingCatalogueAsync(ctx)
	}

	r.cacheMu.RLock()
	updatedAt, stored := r.catalogueSources[serverCatalogueSourceKey(options)]
	r.cacheMu.RUnlock()
	if !stored {
		return ServerCatalogueStatus{}, nil
	}
	return ServerCatalogueStatus{Stored: true, UpdatedAt: new(updatedAt)}, nil
}

// serverCatalogueSourceKey returns the durable identity for one discovery origin.
func serverCatalogueSourceKey(options ServerListOptions) string {
	if options.Global {
		return "global"
	}
	if options.Location != nil {
		return serverCatalogueCoordinatePrefix + strconv.FormatFloat(options.Location.Latitude, 'f', -1, 64) + "," +
			strconv.FormatFloat(options.Location.Longitude, 'f', -1, 64)
	}
	return "local"
}

// getServersForLocation returns a copied catalogue and coalesces concurrent fetches by key.
func (r *SpeedtestNetRunner) getServersForLocation(ctx context.Context, key string, location *ServerLocation, refresh bool) ([]ServerResponse, error) {
	if !refresh && r.catalogueSourceFresh(key) {
		return r.loadRetainedServers(location), nil
	}

	result := r.serverFetches.DoChan(key, func() (any, error) {
		if !refresh && r.catalogueSourceFresh(key) {
			return r.loadRetainedServers(location), nil
		}

		fetchCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), serverCatalogueFetchTimeout)
		defer cancel()
		servers, userLocation, err := r.fetchServers(fetchCtx, location)
		if err != nil {
			return nil, err
		}
		persistCtx, persistCancel := context.WithTimeout(context.WithoutCancel(ctx), serverCatalogueStoreTimeout)
		err = r.updateServerCatalogue(persistCtx, servers, userLocation, key)
		persistCancel()
		if err != nil {
			return nil, err
		}
		retained := r.loadRetainedServers(location)

		log.Debug().
			Str("cache_key", key).
			Int("server_count", len(retained)).
			Msg("Retrieved and cached speedtest servers")
		return retained, nil
	})

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case fetched := <-result:
		if fetched.Err != nil {
			return nil, fetched.Err
		}
		return slices.Clone(fetched.Val.([]ServerResponse)), nil
	}
}

// getGlobalServers retains successful regional results and reports any failed regions to the caller.
func (r *SpeedtestNetRunner) getGlobalServers(ctx context.Context, refresh bool) ([]ServerResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if !refresh && r.catalogueSourceFresh("global") {
		return r.loadRetainedServers(nil), nil
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
	if !refresh && r.catalogueSourceFresh("global") {
		return r.loadRetainedServers(nil), nil
	}

	_, err := r.getServersForLocation(ctx, "local", nil, refresh)
	if err != nil {
		return nil, fmt.Errorf("fetch local speedtest servers: %w", err)
	}
	userLocation, ok := r.loadUserLocation()
	if !ok {
		return nil, fmt.Errorf("speedtest user location is unavailable")
	}

	locationNames := slices.Sorted(maps.Keys(r.globalLocations))
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

			fetchCtx, cancel := context.WithTimeout(ctx, serverCatalogueFetchTimeout)
			servers, _, fetchErr := r.fetchServers(fetchCtx, &coordinates)
			cancel()
			resultMu.Lock()
			defer resultMu.Unlock()
			if fetchErr != nil {
				failures = append(failures, fmt.Errorf("fetch servers near %s: %w", name, fetchErr))
				return
			}
			r.persistMu.Lock()
			r.stageServerCatalogueLocked(servers, nil, "")
			r.persistMu.Unlock()
			successfulLocations++
		})
	}
	wg.Wait()

	var persistErr error
	if successfulLocations > 0 {
		sourceKey := ""
		if len(failures) == 0 {
			sourceKey = "global"
		}
		persistCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), serverCatalogueStoreTimeout)
		persistErr = r.updateServerCatalogue(persistCtx, nil, nil, sourceKey)
		cancel()
	}
	if ctxErr := ctx.Err(); ctxErr != nil {
		return nil, errors.Join(ctxErr, persistErr)
	}
	if persistErr != nil {
		return nil, persistErr
	}
	if successfulLocations == 0 {
		if err := errors.Join(failures...); err != nil {
			return nil, fmt.Errorf("fetch global speedtest servers: %w", err)
		}
		return nil, fmt.Errorf("no global speedtest locations configured")
	}

	retained := r.loadRetainedServers(&userLocation)
	if len(failures) > 0 {
		partialErr := &PartialServerCatalogueError{
			successfulLocations: successfulLocations,
			totalLocations:      len(locationNames),
			failures:            failures,
		}
		log.Warn().Err(partialErr).Msg("Some global speedtest locations could not be fetched")
		return retained, partialErr
	}

	log.Info().
		Int("server_count", len(retained)).
		Int("locations", len(locationNames)).
		Bool("cached", len(failures) == 0).
		Msg("Retrieved global speedtest servers")
	return retained, nil
}

// fetchSpeedtestServers discovers nearby servers and returns the detected origin for local requests.
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

// serverResponses converts upstream servers, falling back when none respond to the library's ping.
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

// sortServersFromOrigin recalculates distance and orders servers nearest-first.
func sortServersFromOrigin(origin ServerLocation, servers []ServerResponse) []ServerResponse {
	for i := range servers {
		servers[i].Distance = haversineDistance(origin.Latitude, origin.Longitude, servers[i].Lat, servers[i].Lon)
	}

	slices.SortFunc(servers, func(a, b ServerResponse) int {
		return cmp.Compare(a.Distance, b.Distance)
	})
	return servers
}

// haversineDistance returns the great-circle distance in kilometres between two coordinates.
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

// catalogueSourceFresh reports whether a source was durably refreshed within the cache duration.
func (r *SpeedtestNetRunner) catalogueSourceFresh(key string) bool {
	r.cacheMu.RLock()
	updatedAt, ok := r.catalogueSources[key]
	r.cacheMu.RUnlock()
	return ok && !time.Now().After(updatedAt.Add(r.cacheDuration))
}

// ensureCatalogueLoaded retries a failed startup load before catalogue state can be used.
func (r *SpeedtestNetRunner) ensureCatalogueLoaded(ctx context.Context) error {
	if r.catalogueStore == nil {
		return nil
	}
	if !r.catalogueLoadFailed.Load() {
		return nil
	}

	r.persistMu.Lock()
	defer r.persistMu.Unlock()
	if !r.catalogueLoadFailed.Load() {
		return nil
	}
	if err := r.loadPersistedServers(ctx); err != nil {
		return err
	}
	r.catalogueLoadFailed.Store(false)
	return nil
}

// loadPersistedServers replaces committed in-memory state from durable storage.
func (r *SpeedtestNetRunner) loadPersistedServers(ctx context.Context) error {
	if r.catalogueStore == nil {
		return nil
	}

	raw, err := r.catalogueStore.GetAppSetting(ctx, serverCatalogueSettingKey)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			return nil
		}
		return fmt.Errorf("read retained speedtest servers: %w", err)
	}

	var catalogue persistedServerCatalogue
	if err := json.Unmarshal([]byte(raw), &catalogue); err != nil {
		return fmt.Errorf("decode retained speedtest servers: %w", err)
	}
	servers := make(map[string]ServerResponse, len(catalogue.Servers))
	for _, server := range catalogue.Servers {
		servers[server.ID] = server
	}
	sources := maps.Clone(catalogue.Sources)
	if sources == nil {
		sources = make(map[string]time.Time)
	}
	var userLocation *ServerLocation
	if catalogue.UserLocation != nil {
		location := *catalogue.UserLocation
		userLocation = &location
	}

	r.cacheMu.Lock()
	r.retainedServers = servers
	r.catalogueSources = sources
	r.userLocation = userLocation
	r.cacheMu.Unlock()
	return nil
}

// retryPendingCatalogue retries the latest failed durable write. The caller waits for completion.
func (r *SpeedtestNetRunner) retryPendingCatalogue(ctx context.Context) error {
	if r.catalogueStore == nil {
		return nil
	}

	r.persistMu.Lock()
	defer r.persistMu.Unlock()
	return r.persistPendingCatalogue(ctx)
}

// retryPendingCatalogueAsync coalesces background retries so reads never wait for durable storage.
func (r *SpeedtestNetRunner) retryPendingCatalogueAsync(ctx context.Context) {
	if r.catalogueStore == nil || !r.hasPendingWrite.Load() {
		return
	}

	r.catalogueRetries.DoChan("pending", func() (any, error) {
		retryCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), serverCatalogueStoreTimeout)
		defer cancel()
		if err := r.retryPendingCatalogue(retryCtx); err != nil {
			log.Warn().Err(err).Msg("Failed to retry retained speedtest catalogue write")
			return nil, err
		}
		return nil, nil
	})
}

// updateServerCatalogue stages an update and publishes it only after durable storage succeeds.
func (r *SpeedtestNetRunner) updateServerCatalogue(
	ctx context.Context,
	servers []ServerResponse,
	detectedLocation *ServerLocation,
	sourceKey string,
) error {
	r.persistMu.Lock()
	defer r.persistMu.Unlock()
	r.stageServerCatalogueLocked(servers, detectedLocation, sourceKey)
	return r.persistPendingCatalogue(ctx)
}

// stageServerCatalogueLocked merges an update into the pending snapshot. The caller must hold persistMu.
func (r *SpeedtestNetRunner) stageServerCatalogueLocked(
	servers []ServerResponse,
	detectedLocation *ServerLocation,
	sourceKey string,
) {
	serversByID := make(map[string]ServerResponse, len(servers))
	sources := make(map[string]time.Time)
	var userLocation *ServerLocation
	if r.pendingCatalogue != nil {
		for _, server := range r.pendingCatalogue.Servers {
			serversByID[server.ID] = server
		}
		maps.Copy(sources, r.pendingCatalogue.Sources)
		if r.pendingCatalogue.UserLocation != nil {
			location := *r.pendingCatalogue.UserLocation
			userLocation = &location
		}
	} else {
		r.cacheMu.RLock()
		maps.Copy(serversByID, r.retainedServers)
		maps.Copy(sources, r.catalogueSources)
		if r.userLocation != nil {
			location := *r.userLocation
			userLocation = &location
		}
		r.cacheMu.RUnlock()
	}
	for _, server := range servers {
		serversByID[server.ID] = server
	}
	if detectedLocation != nil {
		location := *detectedLocation
		userLocation = &location
	}
	if sourceKey != "" {
		sources[sourceKey] = time.Now().UTC()
	}
	coordinateSources := make([]string, 0)
	for key := range sources {
		if key != sourceKey && strings.HasPrefix(key, serverCatalogueCoordinatePrefix) {
			coordinateSources = append(coordinateSources, key)
		}
	}
	slices.SortFunc(coordinateSources, func(a, b string) int {
		if byUpdated := sources[a].Compare(sources[b]); byUpdated != 0 {
			return byUpdated
		}
		return cmp.Compare(a, b)
	})
	// Source timestamps drive status display; retained servers are never pruned here.
	coordinateEntries := len(coordinateSources)
	if strings.HasPrefix(sourceKey, serverCatalogueCoordinatePrefix) {
		coordinateEntries++
	}
	if overflow := coordinateEntries - maxCoordinateCatalogueEntries; overflow > 0 {
		for _, key := range coordinateSources[:overflow] {
			delete(sources, key)
		}
	}

	r.pendingCatalogue = &persistedServerCatalogue{
		Servers: slices.SortedFunc(maps.Values(serversByID), func(a, b ServerResponse) int {
			return cmp.Compare(a.ID, b.ID)
		}),
		UserLocation: userLocation,
		Sources:      sources,
	}
	r.hasPendingWrite.Store(true)
}

// persistPendingCatalogue stores and publishes the pending snapshot. The caller must hold persistMu.
func (r *SpeedtestNetRunner) persistPendingCatalogue(ctx context.Context) error {
	if r.pendingCatalogue == nil {
		return nil
	}

	raw, err := json.Marshal(r.pendingCatalogue)
	if err != nil {
		return fmt.Errorf("encode retained speedtest servers: %w", err)
	}
	if r.catalogueStore != nil {
		if err := r.catalogueStore.SetAppSetting(ctx, serverCatalogueSettingKey, string(raw)); err != nil {
			return fmt.Errorf("write retained speedtest servers: %w", err)
		}
	}

	servers := make(map[string]ServerResponse, len(r.pendingCatalogue.Servers))
	for _, server := range r.pendingCatalogue.Servers {
		servers[server.ID] = server
	}
	sources := maps.Clone(r.pendingCatalogue.Sources)
	var userLocation *ServerLocation
	if r.pendingCatalogue.UserLocation != nil {
		location := *r.pendingCatalogue.UserLocation
		userLocation = &location
	}
	r.cacheMu.Lock()
	r.retainedServers = servers
	r.catalogueSources = sources
	r.userLocation = userLocation
	r.cacheMu.Unlock()
	r.pendingCatalogue = nil
	r.hasPendingWrite.Store(false)
	return nil
}

// loadRetainedServers returns the durable server catalogue sorted from origin.
func (r *SpeedtestNetRunner) loadRetainedServers(origin *ServerLocation) []ServerResponse {
	r.cacheMu.RLock()
	servers := slices.Collect(maps.Values(r.retainedServers))
	if origin == nil && r.userLocation != nil {
		location := *r.userLocation
		origin = &location
	}
	r.cacheMu.RUnlock()

	if origin != nil {
		return sortServersFromOrigin(*origin, servers)
	}
	slices.SortFunc(servers, func(a, b ServerResponse) int {
		return cmp.Compare(a.Distance, b.Distance)
	})
	return servers
}

// loadUserLocation returns the last durably detected local origin.
func (r *SpeedtestNetRunner) loadUserLocation() (ServerLocation, bool) {
	r.cacheMu.RLock()
	defer r.cacheMu.RUnlock()
	if r.userLocation == nil {
		return ServerLocation{}, false
	}
	return *r.userLocation, true
}
