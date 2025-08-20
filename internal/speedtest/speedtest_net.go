// Copyright (c) 2024-2025, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package speedtest

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/rs/zerolog/log"
	st "github.com/showwin/speedtest-go/speedtest"

	"github.com/autobrr/netronome/internal/config"
	"github.com/autobrr/netronome/internal/types"
)

type SpeedtestNetRunner struct {
	client                *st.Speedtest
	config                config.SpeedTestConfig
	progressCallback      func(types.SpeedUpdate)
	serverCache           []ServerResponse
	allServersCache       []ServerResponse            // Cache all servers from all locations
	locationCache         map[string][]ServerResponse // Cache servers by location
	cacheExpiry           time.Time
	allServersCacheExpiry time.Time
	locationCacheExpiry   map[string]time.Time // Cache expiry per location
	cacheDuration         time.Duration
	isInitialized         bool // Track if we've built the initial server lists
}

func NewSpeedtestNetRunner(cfg config.SpeedTestConfig) *SpeedtestNetRunner {
	return &SpeedtestNetRunner{
		client:                st.New(),
		config:                cfg,
		cacheDuration:         30 * time.Minute,
		cacheExpiry:           time.Now(),
		allServersCacheExpiry: time.Now(),
		locationCache:         make(map[string][]ServerResponse),
		locationCacheExpiry:   make(map[string]time.Time),
		isInitialized:         false,
	}
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
		// First, try to find the server in the default server list
		for _, server := range serverList {
			for _, requestedID := range opts.ServerIDs {
				if server.ID == requestedID {
					selectedServer = server
					log.Info().
						Str("server_id", server.ID).
						Str("server_name", server.Name).
						Str("provider", server.Sponsor).
						Msg("Found requested server in default server list")
					break
				}
			}
			if selectedServer != nil {
				break
			}
		}

		// If not found in default list, try to fetch the specific server by ID
		if selectedServer == nil {
			log.Info().
				Str("requested_id", opts.ServerIDs[0]).
				Msg("Server not found in default list, trying to fetch by ID")

			specificServer, err := st.FetchServerByID(opts.ServerIDs[0])
			if err != nil {
				log.Warn().
					Str("server_id", opts.ServerIDs[0]).
					Err(err).
					Msg("Failed to fetch specific server by ID, falling back to closest server")
			} else {
				selectedServer = specificServer
				log.Info().
					Str("server_id", specificServer.ID).
					Str("server_name", specificServer.Name).
					Str("provider", specificServer.Sponsor).
					Msg("Successfully fetched specific server by ID")
			}
		}
	}

	if selectedServer == nil {
		sort.Slice(serverList, func(i, j int) bool {
			return serverList[i].Distance < serverList[j].Distance
		})
		selectedServer = serverList[0]
		log.Info().
			Str("server_id", selectedServer.ID).
			Str("server_name", selectedServer.Name).
			Str("provider", selectedServer.Sponsor).
			Float64("distance", selectedServer.Distance).
			Msg("Selected closest server for testing")
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

	selectedServer.Context.Reset()

	jitterFloat := selectedServer.Jitter.Seconds() * 1000
	result.Jitter = jitterFloat

	return result, nil
}

func (r *SpeedtestNetRunner) GetServers() ([]ServerResponse, error) {
	// Check if we have local servers cached first
	if localServers, exists := r.locationCache["local"]; exists {
		log.Debug().
			Int("server_count", len(localServers)).
			Msg("Returning cached local servers")
		return localServers, nil
	}

	// Directly fetch local servers without initializing all locations
	// This is much faster than initializeAllServers()
	log.Debug().Msg("Fetching local servers directly")
	localServers, err := r.fetchLocalServers()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch local servers: %w", err)
	}

	// Cache the local servers for future use
	r.locationCache["local"] = localServers

	log.Debug().
		Int("server_count", len(localServers)).
		Msg("Fetched and cached local servers")

	return localServers, nil
}

func (r *SpeedtestNetRunner) GetAllServersWithLocationInfo() (map[string]interface{}, error) {
	// Initialize all servers if not done yet
	if err := r.initializeAllServers(); err != nil {
		return nil, fmt.Errorf("failed to initialize servers: %w", err)
	}

	// Create response with all server data and metadata
	locations, _ := r.GetAvailableLocations()

	// Generate a cache version based on locations and server counts
	cacheVersion := fmt.Sprintf("v1_%d_locations_%d_servers", len(locations), len(r.allServersCache))

	response := map[string]interface{}{
		"locations":    locations,
		"servers":      map[string][]ServerResponse{},
		"allServers":   r.allServersCache,
		"totalServers": len(r.allServersCache),
		"cacheVersion": cacheVersion,
		"lastUpdated":  time.Now().Format(time.RFC3339),
	}

	// Add servers grouped by location
	servers := make(map[string][]ServerResponse)
	for location, locationServers := range r.locationCache {
		servers[location] = locationServers
	}
	response["servers"] = servers

	log.Info().
		Int("total_servers", len(r.allServersCache)).
		Int("locations", len(r.locationCache)).
		Str("cache_version", cacheVersion).
		Msg("Returning comprehensive server data")

	return response, nil
}

func (r *SpeedtestNetRunner) GetServersFromMultipleLocations() ([]ServerResponse, error) {
	log.Info().Msg("Fetching servers from multiple global locations")

	// Define major cities for global server coverage
	// These correspond to the --city options available in speedtest-go
	cities := []string{
		"brasilia",
		"hongkong",
		"tokyo",
		"london",
		"moscow",
		"beijing",
		"paris",
		"sanfrancisco",
		"newyork",
		"yishun", // Singapore area
		"delhi",
		"monterrey",
		"berlin",
		"maputo",
		"honolulu",
		"seoul",
		"osaka",
		"shanghai",
		"urumqi",
		"ottawa",
		"capetown",
		"sydney",
		"perth",
		"warsaw",
		"kampala",
		"bangkok",
	}

	var allServers []ServerResponse
	serverMap := make(map[string]ServerResponse) // To avoid duplicates

	for _, cityName := range cities {
		log.Debug().
			Str("city", cityName).
			Msg("Fetching servers for location")

		// Create a new client with location-specific user config
		userConfig := &st.UserConfig{
			CityFlag: cityName,
		}

		locationClient := st.New(st.WithUserConfig(userConfig))

		// Fetch user info for this location
		_, err := locationClient.FetchUserInfo()
		if err != nil {
			log.Warn().
				Err(err).
				Str("city", cityName).
				Msg("Failed to fetch user info for location")
			continue
		}

		// Fetch servers for this location
		serverList, err := locationClient.FetchServers()
		if err != nil {
			log.Warn().
				Err(err).
				Str("city", cityName).
				Msg("Failed to fetch servers for location")
			continue
		}

		availableServers := serverList.Available()
		if availableServers == nil {
			log.Warn().
				Str("city", cityName).
				Msg("No available servers for location")
			continue
		}

		// Process servers and add to map to avoid duplicates
		for _, server := range *availableServers {
			// Use server ID as key to avoid duplicates
			if _, exists := serverMap[server.ID]; !exists {
				lat, _ := strconv.ParseFloat(server.Lat, 64)
				lon, _ := strconv.ParseFloat(server.Lon, 64)

				// Extract city name from server.Name field which typically contains "City (Country)" format
				cityName := server.Name
				if strings.Contains(server.Name, "(") {
					cityName = strings.TrimSpace(strings.Split(server.Name, "(")[0])
				}

				serverMap[server.ID] = ServerResponse{
					ID:           server.ID,
					Name:         server.Sponsor, // Provider name
					Host:         server.Host,
					Distance:     server.Distance,
					Country:      server.Country,
					Sponsor:      server.Sponsor,
					URL:          server.URL,
					Lat:          lat,
					Lon:          lon,
					City:         cityName, // Extracted city name
					IsIperf:      false,
					IsLibrespeed: false,
				}
			}
		}

		log.Info().
			Str("city", cityName).
			Int("servers_found", len(*availableServers)).
			Int("total_unique", len(serverMap)).
			Msg("Processed servers for location")
	}

	// Convert map to slice
	for _, server := range serverMap {
		allServers = append(allServers, server)
	}

	// Sort by distance (using original distance from each server's perspective)
	sort.Slice(allServers, func(i, j int) bool {
		return allServers[i].Distance < allServers[j].Distance
	})

	log.Info().
		Int("total_global_servers", len(allServers)).
		Msg("Successfully retrieved global server list")

	return allServers, nil
}

func (r *SpeedtestNetRunner) initializeAllServers() error {
	if r.isInitialized && len(r.allServersCache) > 0 && time.Now().Before(r.allServersCacheExpiry) {
		return nil // Already initialized and cache is valid
	}

	log.Info().Msg("Initializing comprehensive server list from all locations")

	// Define all locations we want to fetch from
	locations := []string{
		"brasilia",
		"hongkong",
		"tokyo",
		"london",
		"moscow",
		"beijing",
		"paris",
		"sanfrancisco",
		"newyork",
		"yishun", // Singapore area
		"delhi",
		"monterrey",
		"berlin",
		"maputo",
		"honolulu",
		"seoul",
		"osaka",
		"shanghai",
		"urumqi",
		"ottawa",
		"capetown",
		"sydney",
		"perth",
		"warsaw",
		"kampala",
		"bangkok",
	}

	var allServers []ServerResponse
	serverMap := make(map[string]ServerResponse) // To avoid duplicates
	locationServers := make(map[string][]ServerResponse)

	// First, get local servers
	log.Info().Msg("Fetching local servers")
	localServers, err := r.fetchLocalServers()
	if err != nil {
		log.Error().Err(err).Msg("Failed to fetch local servers")
		return err
	}

	// Add local servers to the map
	for _, server := range localServers {
		serverMap[server.ID] = server
	}
	locationServers["local"] = localServers

	log.Info().
		Int("local_servers", len(localServers)).
		Msg("Fetched local servers")

	// Then fetch from all global locations
	for _, location := range locations {
		log.Info().
			Str("location", location).
			Msg("Fetching servers for location")

		locationServerList, err := r.fetchServersForLocation(location)
		if err != nil {
			log.Warn().
				Err(err).
				Str("location", location).
				Msg("Failed to fetch servers for location, continuing with others")
			continue
		}

		// Add to location-specific cache
		locationServers[location] = locationServerList

		// Add to global server map (avoiding duplicates)
		for _, server := range locationServerList {
			if _, exists := serverMap[server.ID]; !exists {
				serverMap[server.ID] = server
			}
		}

		log.Info().
			Str("location", location).
			Int("location_servers", len(locationServerList)).
			Int("total_unique", len(serverMap)).
			Msg("Processed servers for location")
	}

	// Convert map to slice for allServersCache
	for _, server := range serverMap {
		allServers = append(allServers, server)
	}

	// Sort all servers by distance
	sort.Slice(allServers, func(i, j int) bool {
		return allServers[i].Distance < allServers[j].Distance
	})

	// Cache everything
	r.allServersCache = allServers
	r.locationCache = locationServers
	r.allServersCacheExpiry = time.Now().Add(r.cacheDuration)

	// Set expiry for all location caches
	expiryTime := time.Now().Add(r.cacheDuration)
	r.locationCacheExpiry["local"] = expiryTime
	for _, location := range locations {
		r.locationCacheExpiry[location] = expiryTime
	}

	r.isInitialized = true

	log.Info().
		Int("total_servers", len(allServers)).
		Int("locations", len(locationServers)).
		Msg("Successfully initialized comprehensive server list")

	return nil
}

func (r *SpeedtestNetRunner) fetchLocalServers() ([]ServerResponse, error) {
	_, err := r.client.FetchUserInfo()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user info: %w", err)
	}

	serverList, err := r.client.FetchServers()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch servers: %w", err)
	}

	availableServers := serverList.Available()
	if availableServers == nil {
		return []ServerResponse{}, nil
	}

	var servers []ServerResponse
	for _, server := range *availableServers {
		lat, _ := strconv.ParseFloat(server.Lat, 64)
		lon, _ := strconv.ParseFloat(server.Lon, 64)

		// Extract city name from server.Name field which typically contains "City (Country)" format
		cityName := server.Name
		if strings.Contains(server.Name, "(") {
			cityName = strings.TrimSpace(strings.Split(server.Name, "(")[0])
		}

		servers = append(servers, ServerResponse{
			ID:           server.ID,
			Name:         server.Sponsor, // Provider name
			Host:         server.Host,
			Distance:     server.Distance,
			Country:      server.Country,
			Sponsor:      server.Sponsor,
			URL:          server.URL,
			Lat:          lat,
			Lon:          lon,
			City:         cityName, // Extracted city name
			IsIperf:      false,
			IsLibrespeed: false,
		})
	}

	sort.Slice(servers, func(i, j int) bool {
		return servers[i].Distance < servers[j].Distance
	})

	return servers, nil
}

func (r *SpeedtestNetRunner) fetchServersForLocation(location string) ([]ServerResponse, error) {
	// Create a new client with location-specific user config
	userConfig := &st.UserConfig{
		CityFlag: location,
	}

	locationClient := st.New(st.WithUserConfig(userConfig))

	// Fetch user info for this location
	_, err := locationClient.FetchUserInfo()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user info for location %s: %w", location, err)
	}

	// Fetch servers for this location
	serverList, err := locationClient.FetchServers()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch servers for location %s: %w", location, err)
	}

	availableServers := serverList.Available()
	if availableServers == nil {
		return []ServerResponse{}, nil
	}

	// Convert servers to our response format
	var servers []ServerResponse
	for _, server := range *availableServers {
		lat, _ := strconv.ParseFloat(server.Lat, 64)
		lon, _ := strconv.ParseFloat(server.Lon, 64)

		// Extract city name from server.Name field which typically contains "City (Country)" format
		cityName := server.Name
		if strings.Contains(server.Name, "(") {
			cityName = strings.TrimSpace(strings.Split(server.Name, "(")[0])
		}

		servers = append(servers, ServerResponse{
			ID:           server.ID,
			Name:         server.Sponsor, // Provider name
			Host:         server.Host,
			Distance:     server.Distance,
			Country:      server.Country,
			Sponsor:      server.Sponsor,
			URL:          server.URL,
			Lat:          lat,
			Lon:          lon,
			City:         cityName, // Extracted city name
			IsIperf:      false,
			IsLibrespeed: false,
		})
	}

	// Sort by distance
	sort.Slice(servers, func(i, j int) bool {
		return servers[i].Distance < servers[j].Distance
	})

	return servers, nil
}

func (r *SpeedtestNetRunner) GetServersByLocation(location string) ([]ServerResponse, error) {
	// Check if we have cached servers for this location
	if servers, exists := r.locationCache[location]; exists {
		if expiry, expExists := r.locationCacheExpiry[location]; expExists && time.Now().Before(expiry) {
			log.Debug().
				Str("location", location).
				Int("server_count", len(servers)).
				Msg("Returning cached servers for location")
			return servers, nil
		}
	}

	log.Info().
		Str("location", location).
		Msg("Fetching servers for specific location")

	// Parse lat,lon coordinates from location string
	var lat, lon float64
	var err error

	if coords := strings.Split(location, ","); len(coords) == 2 {
		lat, err = strconv.ParseFloat(strings.TrimSpace(coords[0]), 64)
		if err != nil {
			return nil, fmt.Errorf("invalid latitude in location %s: %w", location, err)
		}
		lon, err = strconv.ParseFloat(strings.TrimSpace(coords[1]), 64)
		if err != nil {
			return nil, fmt.Errorf("invalid longitude in location %s: %w", location, err)
		}
	} else {
		return nil, fmt.Errorf("invalid location format %s, expected 'lat,lon'", location)
	}

	// Create a new client with location set to the specified coordinates
	customLocation := st.NewLocation("Custom", lat, lon)
	userConfig := &st.UserConfig{
		Location: customLocation,
	}
	locationClient := st.New(st.WithUserConfig(userConfig))

	// Fetch user info for this location
	_, err = locationClient.FetchUserInfo()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user info: %w", err)
	}

	// Fetch server list
	serverList, err := locationClient.FetchServers()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch servers: %w", err)
	}

	// Get all available servers instead of using FindServer which might limit results
	availableServers := serverList.Available()
	if availableServers == nil {
		log.Warn().
			Str("location", location).
			Msg("No servers available for location")
		return []ServerResponse{}, nil
	}

	log.Info().
		Str("location", location).
		Int("available_servers", len(*availableServers)).
		Msg("Found servers for location")

	// Convert to ServerResponse format
	var servers []ServerResponse
	for i, server := range *availableServers {
		serverLat, _ := strconv.ParseFloat(server.Lat, 64)
		serverLon, _ := strconv.ParseFloat(server.Lon, 64)

		// Debug: log server fields for first few servers to understand available data
		if i < 3 {
			log.Info().
				Str("id", server.ID).
				Str("name", server.Name).
				Str("sponsor", server.Sponsor).
				Str("country", server.Country).
				Str("lat", server.Lat).
				Str("lon", server.Lon).
				Str("host", server.Host).
				Str("url", server.URL).
				Float64("distance", server.Distance).
				Msg("Debug: Available server fields from location-based search")
		}

		// Extract city name from server.Name field which typically contains "City (Country)" format
		cityName := server.Name
		if strings.Contains(server.Name, "(") {
			cityName = strings.TrimSpace(strings.Split(server.Name, "(")[0])
		}

		servers = append(servers, ServerResponse{
			ID:           server.ID,
			Name:         server.Sponsor, // Provider name (like "Kordia", "Megatel")
			Host:         server.Host,
			Distance:     server.Distance,
			Country:      server.Country,
			Sponsor:      server.Sponsor,
			URL:          server.URL,
			Lat:          serverLat,
			Lon:          serverLon,
			City:         cityName, // Extracted city name (like "Auckland")
			IsIperf:      false,
			IsLibrespeed: false,
		})
	}

	// Sort by distance (closest first)
	sort.Slice(servers, func(i, j int) bool {
		return servers[i].Distance < servers[j].Distance
	})

	// Limit to top 20 servers (like the CLI example)
	if len(servers) > 20 {
		servers = servers[:20]
	}

	// Cache the results for this location (valid for 30 minutes)
	r.locationCache[location] = servers
	r.locationCacheExpiry[location] = time.Now().Add(30 * time.Minute)

	log.Info().
		Str("location", location).
		Int("server_count", len(servers)).
		Msg("Successfully fetched and cached servers for location")

	return servers, nil
}

// haversineDistance calculates the distance between two points on the Earth
// using the Haversine formula. Returns distance in kilometers.
func haversineDistance(lat1, lon1, lat2, lon2 float64) float64 {
	const R = 6371 // Earth's radius in kilometers

	// Convert degrees to radians
	lat1Rad := lat1 * math.Pi / 180
	lon1Rad := lon1 * math.Pi / 180
	lat2Rad := lat2 * math.Pi / 180
	lon2Rad := lon2 * math.Pi / 180

	// Calculate differences
	dlat := lat2Rad - lat1Rad
	dlon := lon2Rad - lon1Rad

	// Haversine formula
	a := math.Sin(dlat/2)*math.Sin(dlat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*
			math.Sin(dlon/2)*math.Sin(dlon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return R * c
}

func (r *SpeedtestNetRunner) GetAvailableLocations() ([]string, error) {
	// Return the list of supported locations
	locations := []string{
		"local",
		"brasilia",
		"hongkong",
		"tokyo",
		"london",
		"moscow",
		"beijing",
		"paris",
		"sanfrancisco",
		"newyork",
		"yishun", // Singapore area
		"delhi",
		"monterrey",
		"berlin",
		"maputo",
		"honolulu",
		"seoul",
		"osaka",
		"shanghai",
		"urumqi",
		"ottawa",
		"capetown",
		"sydney",
		"perth",
		"warsaw",
		"kampala",
		"bangkok",
	}

	return locations, nil
}
