// Copyright (c) 2024-2025, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package speedtest

import (
	"context"
	"fmt"

	"github.com/rs/zerolog/log"

	"github.com/autobrr/netronome/internal/database"
	"github.com/autobrr/netronome/internal/notifications"
	"github.com/autobrr/netronome/internal/types"
)

type DefaultResultHandler struct {
	db       database.Service
	notifier *notifications.Notifier
}

func NewResultHandler(db database.Service, notifier *notifications.Notifier) *DefaultResultHandler {
	return &DefaultResultHandler{
		db:       db,
		notifier: notifier,
	}
}

func (h *DefaultResultHandler) SaveResult(ctx context.Context, result *Result, testType string, opts *types.TestOptions) error {
	log.Debug().
		Str("test_type", testType).
		Str("server", result.Server).
		Float64("download_speed", result.DownloadSpeed).
		Float64("upload_speed", result.UploadSpeed).
		Str("latency", result.Latency).
		Float64("jitter", result.Jitter).
		Msg("Preparing to save test results to database")

	var serverHost *string
	var serverID string
	
	switch testType {
	case "iperf3":
		serverHost = &opts.ServerHost
		serverID = fmt.Sprintf("iperf3-%s", opts.ServerHost)
	case "librespeed":
		serverHost = &result.Server
		serverID = fmt.Sprintf("librespeed-%s", result.Server)
	case "speedtest":
		// For speedtest.net, we'll need to extract host info from the result
		serverID = result.Server
	}

	var jitterPtr *float64
	if result.Jitter > 0 {
		jitterPtr = &result.Jitter
	}

	dbResult, err := h.db.SaveSpeedTest(ctx, types.SpeedTestResult{
		ServerName:    result.Server,
		ServerID:      serverID,
		ServerHost:    serverHost,
		TestType:      testType,
		DownloadSpeed: result.DownloadSpeed,
		UploadSpeed:   result.UploadSpeed,
		Latency:       result.Latency,
		Jitter:        jitterPtr,
		IsScheduled:   opts.IsScheduled,
	})
	if err != nil {
		log.Error().Err(err).
			Str("test_type", testType).
			Str("server", result.Server).
			Msg("Failed to save test result to database")
		return err
	}

	if dbResult != nil {
		result.ID = dbResult.ID
		log.Debug().
			Int64("result_id", dbResult.ID).
			Str("test_type", testType).
			Msg("Successfully saved test result to database")
		
		h.SendNotification(dbResult)
	}

	return nil
}

func (h *DefaultResultHandler) SendNotification(result *types.SpeedTestResult) {
	if h.notifier != nil {
		// Convert types.SpeedTestResult to notifications.SpeedTestResult
		notifResult := &notifications.SpeedTestResult{
			ServerName: result.ServerName,
			Provider:   result.TestType,
			Download:   result.DownloadSpeed,
			Upload:     result.UploadSpeed,
			Ping:       parsePingValue(result.Latency),
			Jitter:     getJitterValue(result.Jitter),
			ISP:        "", // ISP not available in types.SpeedTestResult
			Failed:     false, // Assuming successful test if we got here
		}
		h.notifier.SendSpeedTestNotification(notifResult)
	}
}

// parsePingValue extracts the numeric ping value from a latency string like "10.5ms"
func parsePingValue(latency string) float64 {
	if latency == "" {
		return 0
	}
	var value float64
	fmt.Sscanf(latency, "%fms", &value)
	return value
}

// getJitterValue safely dereferences a jitter pointer
func getJitterValue(jitter *float64) float64 {
	if jitter == nil {
		return 0
	}
	return *jitter
}