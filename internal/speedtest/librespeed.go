// Copyright (c) 2024-2026, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package speedtest

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/autobrr/netronome/internal/config"
	"github.com/autobrr/netronome/internal/types"
)

const (
	librespeedPublicServersURL = "https://librespeed.org/backend-servers/servers.json"
	publicServerCacheDuration  = 30 * time.Minute
)

type LibrespeedResult struct {
	Timestamp      time.Time            `json:"timestamp"`
	Server         LibrespeedServerInfo `json:"server"`
	Client         Client               `json:"client"`
	BytesSent      int                  `json:"bytes_sent"`
	BytesReceived  int                  `json:"bytes_received"`
	Ping           float64              `json:"ping"`
	Jitter         float64              `json:"jitter"`
	Upload         float64              `json:"upload"`
	Download       float64              `json:"download"`
	Share          string               `json:"share"`
	ProcessingTime float64              `json:"processing_time"`
}

type LibrespeedServerInfo struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

type Client struct {
	IP       string `json:"ip"`
	Hostname string `json:"hostname"`
	City     string `json:"city"`
	Region   string `json:"region"`
	Country  string `json:"country"`
	Loc      string `json:"loc"`
	Org      string `json:"org"`
	Postal   string `json:"postal"`
	Timezone string `json:"timezone"`
}

type LibrespeedServer struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Server   string `json:"server"`
	DlURL    string `json:"dlURL"`
	UlURL    string `json:"ulURL"`
	PingURL  string `json:"pingURL"`
	GetIpURL string `json:"getIpURL"`
}

// LibrespeedPublicServer represents a server from librespeed.org public list
type LibrespeedPublicServer struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Server      string `json:"server"`
	DlURL       string `json:"dlURL"`
	UlURL       string `json:"ulURL"`
	PingURL     string `json:"pingURL"`
	GetIpURL    string `json:"getIpURL"`
	SponsorName string `json:"sponsorName"`
	SponsorURL  string `json:"sponsorURL"`
}

type LibrespeedRunner struct {
	config           config.LibrespeedConfig
	progressCallback func(types.SpeedUpdate)

	// Public server cache
	publicServerCache []LibrespeedPublicServer
	cacheExpiry       time.Time
	cacheMu           sync.RWMutex
}

func NewLibrespeedRunner(cfg config.LibrespeedConfig) *LibrespeedRunner {
	return &LibrespeedRunner{
		config: cfg,
	}
}

func (r *LibrespeedRunner) GetTestType() string {
	return "librespeed"
}

func (r *LibrespeedRunner) SetProgressCallback(callback func(types.SpeedUpdate)) {
	r.progressCallback = callback
}

func (r *LibrespeedRunner) RunTest(ctx context.Context, opts *types.TestOptions) (*Result, error) {
	log.Debug().Bool("isPublicServer", opts.IsPublicServer).Msg("starting librespeed test")

	// Check if librespeed-cli is installed first
	if _, err := exec.LookPath("librespeed-cli"); err != nil {
		return nil, fmt.Errorf("librespeed-cli not found: please install librespeed-cli to use this feature")
	}

	args := r.buildArgs(opts)

	log.Debug().Strs("args", args).Msg("librespeed-cli arguments")

	cmd := exec.CommandContext(ctx, "librespeed-cli", args...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Error().Err(err).Str("output", string(output)).Msg("librespeed-cli failed")
		return nil, fmt.Errorf("librespeed-cli failed: %v: %s", err, string(output))
	}

	log.Debug().Str("output", string(output)).Msg("librespeed-cli output")

	var librespeedResults []LibrespeedResult
	if err := json.Unmarshal(output, &librespeedResults); err != nil {
		log.Error().Err(err).Str("output", string(output)).Msg("failed to parse librespeed-cli output")
		return nil, fmt.Errorf("failed to parse librespeed-cli output: %w", err)
	}

	if len(librespeedResults) == 0 {
		if len(opts.ServerIDs) > 0 {
			return nil, fmt.Errorf("librespeed-cli returned no results for server %s", opts.ServerIDs[0])
		}
		return nil, fmt.Errorf("librespeed-cli returned no results")
	}

	librespeedResult := librespeedResults[0]

	log.Info().Interface("result", librespeedResult).Msg("librespeed test complete")

	result := &Result{
		Timestamp:     librespeedResult.Timestamp,
		Server:        librespeedResult.Server.Name,
		DownloadSpeed: librespeedResult.Download,
		UploadSpeed:   librespeedResult.Upload,
		Latency:       fmt.Sprintf("%.2f", librespeedResult.Ping),
		Jitter:        librespeedResult.Jitter,
		ResultURL:     librespeedResult.Share,
	}

	// Final completion update
	if r.progressCallback != nil {
		r.progressCallback(types.SpeedUpdate{
			Type:        "complete",
			ServerName:  result.Server,
			Speed:       result.DownloadSpeed,
			Progress:    100,
			IsComplete:  true,
			IsScheduled: opts.IsScheduled,
			TestType:    "librespeed",
		})
	}

	return result, nil
}

func (r *LibrespeedRunner) buildArgs(opts *types.TestOptions) []string {
	args := []string{"--json"}

	if r.config.ShareResults {
		args = append(args, "--share")
	}

	if opts.IsPublicServer {
		// Keep CLI server IDs in sync with our fetched public list.
		args = append(args, "--server-json", librespeedPublicServersURL)
	} else {
		args = append(args, "--local-json", r.config.ServersPath)
	}

	if len(opts.ServerIDs) > 0 {
		args = append(args, "--server", opts.ServerIDs[0])
	}

	return args
}

func (r *LibrespeedRunner) GetServers() ([]ServerResponse, error) {
	var allServers []ServerResponse
	var customErr, publicErr error

	// 1. Load custom servers from local JSON
	customServers, customErr := r.getCustomServers()
	if customErr != nil {
		log.Warn().Err(customErr).Msg("failed to load custom librespeed servers")
	} else {
		allServers = append(allServers, customServers...)
	}

	// 2. Fetch public servers from librespeed.org
	publicServers, publicErr := r.fetchPublicServers()
	if publicErr != nil {
		log.Warn().Err(publicErr).Msg("failed to fetch public librespeed servers")
	} else {
		for _, server := range publicServers {
			// Validate server data from external source
			if server.ID <= 0 || server.Name == "" {
				log.Debug().Int("id", server.ID).Str("name", server.Name).Msg("skipping invalid public server")
				continue
			}

			country := parseCountryFromName(server.Name)
			sponsor := server.SponsorName
			if sponsor == "" {
				sponsor = server.Name
			}

			allServers = append(allServers, ServerResponse{
				ID:           strconv.Itoa(server.ID),
				Name:         server.Name,
				Host:         server.Server,
				Country:      country,
				Sponsor:      sponsor,
				IsLibrespeed: true,
				IsPublic:     true,
			})
		}
	}

	// Return error if both sources failed and we have no servers
	if len(allServers) == 0 && (customErr != nil || publicErr != nil) {
		return nil, fmt.Errorf("failed to load librespeed servers: custom=%v, public=%v", customErr, publicErr)
	}

	return allServers, nil
}

// getCustomServers loads servers from the local librespeed-servers.json file
func (r *LibrespeedRunner) getCustomServers() ([]ServerResponse, error) {
	jsonFile, err := os.Open(r.config.ServersPath)
	if err != nil {
		if os.IsNotExist(err) {
			log.Debug().Str("path", r.config.ServersPath).Msg("librespeed-servers.json not found, skipping custom servers")
			return []ServerResponse{}, nil
		}
		return nil, fmt.Errorf("failed to open librespeed-servers.json at %s: %w", r.config.ServersPath, err)
	}
	defer jsonFile.Close()

	var servers []LibrespeedServer
	if err := json.NewDecoder(jsonFile).Decode(&servers); err != nil {
		return nil, fmt.Errorf("failed to decode librespeed-servers.json: %w", err)
	}

	response := make([]ServerResponse, len(servers))
	for i, server := range servers {
		response[i] = ServerResponse{
			ID:           strconv.Itoa(server.ID),
			Name:         server.Name,
			Host:         server.Server,
			Country:      "Custom",
			Sponsor:      server.Name,
			IsLibrespeed: true,
			IsPublic:     false,
		}
	}

	return response, nil
}

// fetchPublicServers fetches servers from librespeed.org with caching
func (r *LibrespeedRunner) fetchPublicServers() ([]LibrespeedPublicServer, error) {
	// Check cache first (with read lock)
	r.cacheMu.RLock()
	if len(r.publicServerCache) > 0 && time.Now().Before(r.cacheExpiry) {
		servers := r.publicServerCache
		r.cacheMu.RUnlock()
		log.Debug().Int("count", len(servers)).Msg("returning cached public librespeed servers")
		return servers, nil
	}
	r.cacheMu.RUnlock()

	// Fetch from librespeed.org
	log.Debug().Str("url", librespeedPublicServersURL).Msg("fetching public librespeed servers")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get(librespeedPublicServersURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch public servers: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch public servers: status %d", resp.StatusCode)
	}

	// Limit response size to 10MB to prevent memory exhaustion from malicious responses
	limitedReader := io.LimitReader(resp.Body, 10*1024*1024)

	var servers []LibrespeedPublicServer
	if err := json.NewDecoder(limitedReader).Decode(&servers); err != nil {
		return nil, fmt.Errorf("failed to decode public servers: %w", err)
	}

	// Update cache (with write lock)
	r.cacheMu.Lock()
	r.publicServerCache = servers
	r.cacheExpiry = time.Now().Add(publicServerCacheDuration)
	r.cacheMu.Unlock()

	log.Info().Int("count", len(servers)).Msg("fetched public librespeed servers")
	return servers, nil
}

// parseCountryFromName extracts country from server names like "Amsterdam, Netherlands (Clouvider)"
func parseCountryFromName(name string) string {
	parts := strings.Split(name, ",")
	if len(parts) >= 2 {
		// Get the second part and strip any parenthetical info
		countryPart := strings.TrimSpace(parts[1])
		if idx := strings.Index(countryPart, "("); idx > 0 {
			countryPart = strings.TrimSpace(countryPart[:idx])
		}
		return countryPart
	}
	return "Unknown"
}
