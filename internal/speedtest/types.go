// Copyright (c) 2024-2025, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package speedtest

import (
	"context"
	"time"

	"github.com/autobrr/netronome/internal/types"
)

type Result struct {
	ID            int64     `json:"id"`
	Timestamp     time.Time `json:"timestamp"`
	Server        string    `json:"server"`
	DownloadSpeed float64   `json:"downloadSpeed"`
	UploadSpeed   float64   `json:"uploadSpeed"`
	Latency       string    `json:"latency"`
	PacketLoss    float64   `json:"packetLoss"`
	Jitter        float64   `json:"jitter"`
	Error         string    `json:"error,omitempty"`
	Download      float64   `json:"-"`
	Upload        float64   `json:"-"`
}

type ServerResponse struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	Host         string  `json:"host"`
	Distance     float64 `json:"distance"`
	Country      string  `json:"country"`
	Sponsor      string  `json:"sponsor"`
	URL          string  `json:"url"`
	Lat          float64 `json:"lat,string"`
	Lon          float64 `json:"lon,string"`
	IsIperf      bool    `json:"isIperf"`
	IsLibrespeed bool    `json:"isLibrespeed"`
}

type ProgressUpdate struct {
	ServerName   string  `json:"serverName"`
	TestType     string  `json:"testType"`
	CurrentSpeed float64 `json:"currentSpeed"`
}

type SpeedUpdate struct {
	Type        string  `json:"type"`
	ServerName  string  `json:"serverName"`
	Speed       float64 `json:"speed"`
	Progress    float64 `json:"progress"`
	IsComplete  bool    `json:"isComplete"`
	Latency     string  `json:"latency,omitempty"`
	IsScheduled bool    `json:"isScheduled"`
}

// TestRunner interface for different speed test implementations
type TestRunner interface {
	// RunTest executes a speed test and returns the result
	RunTest(ctx context.Context, opts *types.TestOptions) (*Result, error)
	
	// GetServers returns available servers for this test type
	GetServers() ([]ServerResponse, error)
	
	// GetTestType returns the test type identifier
	GetTestType() string
	
	// SetProgressCallback sets the callback for progress updates
	SetProgressCallback(callback func(types.SpeedUpdate))
}

// ResultHandler handles database saves and notifications
type ResultHandler interface {
	SaveResult(ctx context.Context, result *Result, testType string, opts *types.TestOptions) error
	SendNotification(result *types.SpeedTestResult)
}

// ProgressBroadcaster handles real-time progress updates
type ProgressBroadcaster interface {
	BroadcastUpdate(update types.SpeedUpdate)
}
