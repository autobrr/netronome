// Copyright (c) 2024-2025, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package speedtest

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/autobrr/netronome/internal/types"
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

func (s *service) GetLibrespeedServers() ([]ServerResponse, error) {
	jsonFile, err := os.Open(s.config.Librespeed.ServersPath)
	if err != nil {
		if os.IsNotExist(err) {
			log.Warn().Str("path", s.config.Librespeed.ServersPath).Msg("librespeed-servers.json not found")
			// Return empty array instead of error when file doesn't exist
			return []ServerResponse{}, nil
		} else {
			return nil, fmt.Errorf("failed to open librespeed-servers.json at %s: %w", s.config.Librespeed.ServersPath, err)
		}
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
		}
	}

	return response, nil
}

func (s *service) RunLibrespeedTest(ctx context.Context, opts *types.TestOptions) (*Result, error) {
	log.Debug().Msg("starting librespeed test")

	args := []string{"--json", "--local-json", s.config.Librespeed.ServersPath}

	if len(opts.ServerIDs) > 0 {
		args = append(args, "--server", opts.ServerIDs[0])
	}

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
	}

	dbResult, err := s.db.SaveSpeedTest(ctx, types.SpeedTestResult{
		ServerName:    result.Server,
		ServerID:      fmt.Sprintf("librespeed-%s", librespeedResult.Server.URL),
		ServerHost:    &librespeedResult.Server.URL,
		TestType:      "librespeed",
		DownloadSpeed: result.DownloadSpeed,
		UploadSpeed:   result.UploadSpeed,
		Latency:       result.Latency,
		Jitter:        &result.Jitter,
		IsScheduled:   opts.IsScheduled,
	})
	if err != nil {
		log.Error().Err(err).Msg("Failed to save librespeed result to database")
	}

	if dbResult != nil {
		result.ID = dbResult.ID
	}

	// Final completion update
	if s.broadcastUpdate != nil {
		s.broadcastUpdate(types.SpeedUpdate{
			Type:        "complete",
			ServerName:  result.Server,
			Speed:       result.DownloadSpeed, // Or another relevant metric
			Progress:    100,
			IsComplete:  true,
			IsScheduled: opts.IsScheduled,
		})
	}

	return result, nil
}
