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
	"net/http"
	"net/url"
	"slices"
	"strconv"
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
	serverCatalogueCoordinatePrefix = "location:"
	sourceKeyLocal                  = "local"
	sourceKeyGlobal                 = "global"
	speedtestServerListURL          = "https://www.speedtest.net/api/js/servers"
	// nearestServerLimit is how many retained servers the local and coordinate
	// views show, nearest first. The global view shows every retained server.
	nearestServerLimit = 10
)

// serverCatalogueStore owns durable Speedtest.net servers and discovery-source status.
type serverCatalogueStore interface {
	ListSpeedtestServers(ctx context.Context) ([]database.SpeedtestServer, error)
	GetSpeedtestServerSource(ctx context.Context, key string) (database.SpeedtestServerSource, bool, error)
	SaveSpeedtestServerCatalogue(
		ctx context.Context,
		servers []database.SpeedtestServer,
		source *database.SpeedtestServerSource,
	) error
}

// SpeedtestNetRunner executes Speedtest.net tests, coalesces discovery, and persistently retains servers.
type SpeedtestNetRunner struct {
	client           *st.Speedtest
	config           config.SpeedTestConfig
	progressCallback func(types.SpeedUpdate)
	catalogueStore   serverCatalogueStore
	globalFetch      chan struct{}
	serverFetches    singleflight.Group
	lastObservation  atomic.Int64
	fetchServers     serverFetcher
	globalLocations  map[string]ServerLocation
}

type serverFetcher func(context.Context, *ServerLocation) ([]ServerResponse, *ServerLocation, error)

// NewSpeedtestNetRunner creates a runner whose server catalogue is backed by store.
// Catalogue operations return an error when store is nil.
func NewSpeedtestNetRunner(cfg config.SpeedTestConfig, store serverCatalogueStore) *SpeedtestNetRunner {
	runner := &SpeedtestNetRunner{
		client:          st.New(),
		config:          cfg,
		catalogueStore:  store,
		globalFetch:     make(chan struct{}, 1),
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
			return nil, errors.New("no speedtest servers available")
		}
		slices.SortFunc(serverList, func(a, b *st.Server) int {
			return cmp.Compare(a.Distance, b.Distance)
		})
		selectedServer = serverList[0]
	}

	// The by-ID lookup at Speedtest.net returns no host, so fill it from the retained catalogue.
	if retained, ok := r.retainedServer(ctx, selectedServer.ID); ok {
		selectedServer.Host = cmp.Or(selectedServer.Host, retained.Host)
		selectedServer.Name = cmp.Or(selectedServer.Name, retained.Name)
		selectedServer.Country = cmp.Or(selectedServer.Country, retained.Country)
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
// Concurrent fetches for one local or coordinate source share work while each waiter observes its
// own context.
func (r *SpeedtestNetRunner) GetServersWithOptions(ctx context.Context, options ServerListOptions) ([]ServerResponse, error) {
	if options.Global && options.Location != nil {
		return nil, errors.New("global and coordinate server searches are mutually exclusive")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	key := serverCatalogueSourceKey(options)
	if !options.Refresh {
		stored, err := r.catalogueSourceStored(ctx, key)
		if err != nil {
			return nil, err
		}
		if stored {
			return r.loadRetainedServers(ctx, options.Location, viewLimit(options))
		}
	}
	if options.Global {
		return r.getGlobalServers(ctx, options.Refresh)
	}
	return r.getServersForLocation(ctx, key, options.Location, options.Refresh)
}

// viewLimit returns how many retained servers a source shows. Zero means all.
func viewLimit(options ServerListOptions) int {
	if options.Global {
		return 0
	}
	return nearestServerLimit
}

// GetServerCatalogueStatus returns durable fetch metadata without contacting Speedtest.net.
func (r *SpeedtestNetRunner) GetServerCatalogueStatus(ctx context.Context, options ServerListOptions) (ServerCatalogueStatus, error) {
	if options.Global && options.Location != nil {
		return ServerCatalogueStatus{}, errors.New("global and coordinate server searches are mutually exclusive")
	}
	if err := ctx.Err(); err != nil {
		return ServerCatalogueStatus{}, err
	}
	if r.catalogueStore == nil {
		return ServerCatalogueStatus{}, errors.New("speedtest server catalogue store is unavailable")
	}
	source, stored, err := r.catalogueStore.GetSpeedtestServerSource(ctx, serverCatalogueSourceKey(options))
	if err != nil {
		return ServerCatalogueStatus{}, fmt.Errorf("read speedtest server source status: %w", err)
	}
	if !stored {
		return ServerCatalogueStatus{}, nil
	}
	return ServerCatalogueStatus{Stored: true, UpdatedAt: new(source.UpdatedAt)}, nil
}

// serverCatalogueSourceKey returns the durable identity for one discovery origin.
func serverCatalogueSourceKey(options ServerListOptions) string {
	if options.Global {
		return sourceKeyGlobal
	}
	if options.Location != nil {
		return serverCatalogueCoordinatePrefix + strconv.FormatFloat(options.Location.Latitude, 'f', -1, 64) + "," +
			strconv.FormatFloat(options.Location.Longitude, 'f', -1, 64)
	}
	return sourceKeyLocal
}

// getServersForLocation fetches one local or coordinate source, coalescing concurrent fetches by key.
func (r *SpeedtestNetRunner) getServersForLocation(ctx context.Context, key string, location *ServerLocation, refresh bool) ([]ServerResponse, error) {
	result := r.serverFetches.DoChan(key, func() (any, error) {
		fetchCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), serverCatalogueFetchTimeout)
		defer cancel()
		servers, userLocation, err := r.fetchServers(fetchCtx, location)
		if err != nil {
			return nil, err
		}
		persistCtx, persistCancel := context.WithTimeout(context.WithoutCancel(ctx), serverCatalogueStoreTimeout)
		defer persistCancel()
		err = r.updateServerCatalogue(persistCtx, servers, userLocation, key, r.nextCatalogueObservation())
		if err != nil {
			return nil, err
		}
		retained, err := r.loadRetainedServers(persistCtx, location, nearestServerLimit)
		if err != nil {
			return nil, err
		}

		log.Debug().
			Str("source_key", key).
			Int("server_count", len(retained)).
			Msg("Retrieved and retained speedtest servers")
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
	select {
	case r.globalFetch <- struct{}{}:
		defer func() { <-r.globalFetch }()
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	// A caller that waited on the channel may find the fetch already done.
	if !refresh {
		stored, err := r.catalogueSourceStored(ctx, sourceKeyGlobal)
		if err != nil {
			return nil, err
		}
		if stored {
			return r.loadRetainedServers(ctx, nil, 0)
		}
	}

	_, localErr := r.getServersForLocation(ctx, sourceKeyLocal, nil, refresh)
	userLocation, ok, err := r.loadUserLocation(ctx)
	if err != nil {
		return nil, err
	}
	if !ok {
		if localErr != nil {
			return nil, fmt.Errorf("fetch local speedtest servers: %w", localErr)
		}
		return nil, errors.New("speedtest user location is unavailable")
	}

	locationNames := slices.Sorted(maps.Keys(r.globalLocations))
	failures := make([]error, 0, len(locationNames)+1)
	if localErr != nil {
		failures = append(failures, fmt.Errorf("fetch local speedtest servers: %w", localErr))
	}
	var discovered []ServerResponse
	successfulLocations := 0
	observedAt := r.nextCatalogueObservation()
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
			discovered = append(discovered, servers...)
			successfulLocations++
		})
	}
	wg.Wait()

	var persistErr error
	if successfulLocations > 0 {
		// A partial result still marks the source as fetched. Otherwise every
		// later read would fetch the world again until all regions succeed.
		// A cancelled fetch keeps its completed regions but stays unfetched,
		// so the next read finishes the job instead of reporting success.
		sourceKey := sourceKeyGlobal
		if ctx.Err() != nil {
			sourceKey = ""
		}
		persistCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), serverCatalogueStoreTimeout)
		persistErr = r.updateServerCatalogue(persistCtx, discovered, nil, sourceKey, observedAt)
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
		return nil, errors.New("no global speedtest locations configured")
	}

	retained, err := r.loadRetainedServers(ctx, &userLocation, 0)
	if err != nil {
		return nil, err
	}
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
	}

	serverList, err := fetchSpeedtestServerList(ctx, speedtestServerListURL, location)
	if err != nil {
		return nil, nil, fmt.Errorf("fetch speedtest servers: %w", err)
	}
	servers := serverResponses(serverList)
	if len(servers) == 0 {
		return nil, nil, errors.New("no speedtest servers found")
	}
	return servers, userLocation, nil
}

// fetchSpeedtestServerList retrieves discovery metadata without probing every returned server.
func fetchSpeedtestServerList(ctx context.Context, endpoint string, location *ServerLocation) (st.Servers, error) {
	serverURL, err := url.Parse(endpoint)
	if err != nil {
		return nil, fmt.Errorf("parse speedtest server URL: %w", err)
	}
	if location != nil {
		query := serverURL.Query()
		query.Set("lat", strconv.FormatFloat(location.Latitude, 'f', -1, 64))
		query.Set("lon", strconv.FormatFloat(location.Longitude, 'f', -1, 64))
		serverURL.RawQuery = query.Encode()
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, serverURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("create speedtest server request: %w", err)
	}
	request.Header.Set("User-Agent", st.DefaultUserAgent)

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("request speedtest servers: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request speedtest servers: status %s", response.Status)
	}

	var servers st.Servers
	if err := json.NewDecoder(response.Body).Decode(&servers); err != nil {
		return nil, fmt.Errorf("decode speedtest servers: %w", err)
	}
	return servers, nil
}

// serverResponses converts upstream discovery metadata into selectable servers.
func serverResponses(serverList st.Servers) []ServerResponse {
	response := make([]ServerResponse, 0, len(serverList))
	for _, server := range serverList {
		lat, _ := strconv.ParseFloat(server.Lat, 64)
		lon, _ := strconv.ParseFloat(server.Lon, 64)
		response = append(response, ServerResponse{
			ID:      server.ID,
			Name:    server.Name,
			Host:    server.Host,
			Country: server.Country,
			Sponsor: server.Sponsor,
			URL:     server.URL,
			Lat:     lat,
			Lon:     lon,
		})
	}
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

// catalogueSourceStored reports whether a source has completed a fetch at any time.
func (r *SpeedtestNetRunner) catalogueSourceStored(ctx context.Context, key string) (bool, error) {
	if r.catalogueStore == nil {
		return false, errors.New("speedtest server catalogue store is unavailable")
	}
	_, ok, err := r.catalogueStore.GetSpeedtestServerSource(ctx, key)
	if err != nil {
		return false, fmt.Errorf("read speedtest server source %q: %w", key, err)
	}
	return ok, nil
}

// nextCatalogueObservation returns a process-local monotonic UTC timestamp for refresh ordering.
func (r *SpeedtestNetRunner) nextCatalogueObservation() time.Time {
	now := time.Now().UTC().UnixNano()
	for {
		previous := r.lastObservation.Load()
		if now <= previous {
			now = previous + 1
		}
		if r.lastObservation.CompareAndSwap(previous, now) {
			return time.Unix(0, now).UTC()
		}
	}
}

// updateServerCatalogue atomically retains an observed batch and optional completed source.
func (r *SpeedtestNetRunner) updateServerCatalogue(
	ctx context.Context,
	servers []ServerResponse,
	detectedLocation *ServerLocation,
	sourceKey string,
	observedAt time.Time,
) error {
	if r.catalogueStore == nil {
		return errors.New("speedtest server catalogue store is unavailable")
	}

	storedServers := make([]database.SpeedtestServer, 0, len(servers))
	for _, server := range servers {
		storedServers = append(storedServers, database.SpeedtestServer{
			ID:         server.ID,
			Name:       server.Name,
			Host:       server.Host,
			Country:    server.Country,
			Sponsor:    server.Sponsor,
			URL:        server.URL,
			Latitude:   server.Lat,
			Longitude:  server.Lon,
			ObservedAt: observedAt,
		})
	}

	var source *database.SpeedtestServerSource
	if sourceKey != "" {
		source = &database.SpeedtestServerSource{Key: sourceKey, UpdatedAt: observedAt}
		if detectedLocation != nil {
			source.Latitude = new(detectedLocation.Latitude)
			source.Longitude = new(detectedLocation.Longitude)
		}
	}
	if err := r.catalogueStore.SaveSpeedtestServerCatalogue(ctx, storedServers, source); err != nil {
		return fmt.Errorf("write retained speedtest servers: %w", err)
	}
	return nil
}

// loadRetainedServers returns the retained servers sorted from origin. A limit
// above zero keeps only the nearest servers when an origin is known.
func (r *SpeedtestNetRunner) loadRetainedServers(ctx context.Context, origin *ServerLocation, limit int) ([]ServerResponse, error) {
	if r.catalogueStore == nil {
		return nil, errors.New("speedtest server catalogue store is unavailable")
	}
	storedServers, err := r.catalogueStore.ListSpeedtestServers(ctx)
	if err != nil {
		return nil, fmt.Errorf("read retained speedtest servers: %w", err)
	}
	servers := make([]ServerResponse, 0, len(storedServers))
	for _, server := range storedServers {
		servers = append(servers, ServerResponse{
			ID:      server.ID,
			Name:    server.Name,
			Host:    server.Host,
			Country: server.Country,
			Sponsor: server.Sponsor,
			URL:     server.URL,
			Lat:     server.Latitude,
			Lon:     server.Longitude,
		})
	}
	if origin == nil {
		location, ok, err := r.loadUserLocation(ctx)
		if err != nil {
			return nil, err
		}
		if ok {
			origin = &location
		}
	}

	if origin != nil {
		servers = sortServersFromOrigin(*origin, servers)
		if limit > 0 {
			servers = servers[:min(limit, len(servers))]
		}
		return servers, nil
	}
	slices.SortFunc(servers, func(a, b ServerResponse) int {
		return cmp.Compare(a.ID, b.ID)
	})
	return servers, nil
}

// retainedServer finds one retained server by ID.
func (r *SpeedtestNetRunner) retainedServer(ctx context.Context, id string) (database.SpeedtestServer, bool) {
	if r.catalogueStore == nil {
		return database.SpeedtestServer{}, false
	}
	servers, err := r.catalogueStore.ListSpeedtestServers(ctx)
	if err != nil {
		log.Debug().Err(err).Str("server_id", id).Msg("Could not read retained speedtest servers")
		return database.SpeedtestServer{}, false
	}
	// ponytail: linear scan over the retained pool, a store lookup if it grows past a few thousand rows
	i := slices.IndexFunc(servers, func(server database.SpeedtestServer) bool { return server.ID == id })
	if i < 0 {
		return database.SpeedtestServer{}, false
	}
	return servers[i], true
}

// loadUserLocation returns the last durably detected local origin.
func (r *SpeedtestNetRunner) loadUserLocation(ctx context.Context) (ServerLocation, bool, error) {
	if r.catalogueStore == nil {
		return ServerLocation{}, false, errors.New("speedtest server catalogue store is unavailable")
	}
	source, ok, err := r.catalogueStore.GetSpeedtestServerSource(ctx, sourceKeyLocal)
	if err != nil {
		return ServerLocation{}, false, fmt.Errorf("read local speedtest server source: %w", err)
	}
	if !ok || source.Latitude == nil || source.Longitude == nil {
		return ServerLocation{}, false, nil
	}
	return ServerLocation{Latitude: *source.Latitude, Longitude: *source.Longitude}, true, nil
}
