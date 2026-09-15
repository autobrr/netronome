// Copyright (c) 2024-2026, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package speedtest

import (
	"context"
	"fmt"
	"time"

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

	// Give the DB write up to 10s, detached from the caller's deadline/cancellation
	// (a test that overran the speedtest timeout must still persist its result) while keeping values.
	saveCtx, saveCancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
	defer saveCancel()

	dbResult, err := h.db.SaveSpeedTest(saveCtx, resultForStorage(result, testType, opts, time.Now().UTC()))
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

// resultForStorage maps a completed run to persistence fields, retaining legacy name-based ID fallbacks.
func resultForStorage(result *Result, testType string, opts *types.TestOptions, createdAt time.Time) types.SpeedTestResult {
	stored := types.SpeedTestResult{
		ServerName:    result.Server,
		TestType:      testType,
		DownloadSpeed: result.DownloadSpeed,
		UploadSpeed:   result.UploadSpeed,
		Latency:       result.Latency,
		IsScheduled:   opts.IsScheduled,
		CreatedAt:     createdAt,
	}
	if result.Jitter > 0 {
		stored.Jitter = &result.Jitter
	}

	switch testType {
	case "iperf3":
		stored.ServerHost = &opts.ServerHost
		stored.ServerID = fmt.Sprintf("iperf3-%s", opts.ServerHost)
	case "librespeed":
		serverID := result.ServerID
		if serverID == "" {
			serverID = result.Server
		}
		stored.ServerID = fmt.Sprintf("librespeed-%s", serverID)
		if result.ServerHost != "" {
			stored.ServerHost = &result.ServerHost
		}
	case "speedtest":
		stored.ServerID = result.ServerID
		if stored.ServerID == "" {
			stored.ServerID = result.Server
		}
		if result.ServerHost != "" {
			stored.ServerHost = &result.ServerHost
		}
		if result.ServerCity != "" {
			stored.ServerCity = &result.ServerCity
		}
	}

	return stored
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
			ISP:        "",    // ISP not available in types.SpeedTestResult
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
