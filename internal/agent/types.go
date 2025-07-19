// Copyright (c) 2024-2025, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package agent

import (
	"sync"
	"time"

	"tailscale.com/tsnet"

	"github.com/autobrr/netronome/internal/config"
)

// Agent represents the monitor SSE agent
type Agent struct {
	config          *config.AgentConfig
	tailscaleConfig *config.TailscaleConfig
	clients         map[chan string]bool
	clientsMu       sync.RWMutex
	monitorData     chan string
	peakRx          int       // Peak download speed in bytes/s
	peakTx          int       // Peak upload speed in bytes/s
	peakRxTimestamp time.Time // Timestamp when peak download was recorded
	peakTxTimestamp time.Time // Timestamp when peak upload was recorded
	peakMu          sync.RWMutex
	tsnetServer     *tsnet.Server
	useTailscale    bool
}

// MonitorLiveData represents the JSON structure from vnstat --live --json
type MonitorLiveData struct {
	Index   int `json:"index"`
	Seconds int `json:"seconds"`
	Rx      struct {
		Ratestring       string `json:"ratestring"`
		Bytespersecond   int    `json:"bytespersecond"`
		Packetspersecond int    `json:"packetspersecond"`
		Bytes            int    `json:"bytes"`
		Packets          int    `json:"packets"`
		Totalbytes       int    `json:"totalbytes"`
		Totalpackets     int    `json:"totalpackets"`
	} `json:"rx"`
	Tx struct {
		Ratestring       string `json:"ratestring"`
		Bytespersecond   int    `json:"bytespersecond"`
		Packetspersecond int    `json:"packetspersecond"`
		Bytes            int    `json:"bytes"`
		Packets          int    `json:"packets"`
		Totalbytes       int    `json:"totalbytes"`
		Totalpackets     int    `json:"totalpackets"`
	} `json:"tx"`
}

// SystemInfo represents system information
type SystemInfo struct {
	Hostname      string                   `json:"hostname"`
	Kernel        string                   `json:"kernel"`
	Uptime        int64                    `json:"uptime"` // seconds
	Interfaces    map[string]InterfaceInfo `json:"interfaces"`
	VnstatVersion string                   `json:"vnstat_version"`
	UpdatedAt     time.Time                `json:"updated_at"`
}

// InterfaceInfo represents network interface details
type InterfaceInfo struct {
	Name       string `json:"name"`
	Alias      string `json:"alias"`
	IPAddress  string `json:"ip_address"`
	LinkSpeed  int    `json:"link_speed"` // Mbps
	IsUp       bool   `json:"is_up"`
	BytesTotal int64  `json:"bytes_total"`
}

// PeakStats represents peak bandwidth statistics
type PeakStats struct {
	PeakRx          int       `json:"peak_rx"` // bytes/s
	PeakTx          int       `json:"peak_tx"` // bytes/s
	PeakRxString    string    `json:"peak_rx_string"`
	PeakTxString    string    `json:"peak_tx_string"`
	PeakRxTimestamp time.Time `json:"peak_rx_timestamp"`
	PeakTxTimestamp time.Time `json:"peak_tx_timestamp"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// HardwareStats represents system hardware statistics
type HardwareStats struct {
	CPU         CPUStats           `json:"cpu"`
	Memory      MemoryStats        `json:"memory"`
	Disks       []DiskStats        `json:"disks"`
	Temperature []TemperatureStats `json:"temperature,omitempty"`
	UpdatedAt   time.Time          `json:"updated_at"`
}

// CPUStats represents CPU usage statistics
type CPUStats struct {
	UsagePercent float64   `json:"usage_percent"`
	Cores        int       `json:"cores"`
	Threads      int       `json:"threads"`
	Model        string    `json:"model"`
	Frequency    float64   `json:"frequency"`          // MHz
	LoadAvg      []float64 `json:"load_avg,omitempty"` // 1, 5, 15 min
}

// MemoryStats represents memory usage statistics
type MemoryStats struct {
	Total       uint64  `json:"total"`
	Used        uint64  `json:"used"`
	Free        uint64  `json:"free"`
	Available   uint64  `json:"available"`
	UsedPercent float64 `json:"used_percent"`
	Cached      uint64  `json:"cached"`
	Buffers     uint64  `json:"buffers"`
	ZFSArc      uint64  `json:"zfs_arc"`
	SwapTotal   uint64  `json:"swap_total"`
	SwapUsed    uint64  `json:"swap_used"`
	SwapPercent float64 `json:"swap_percent"`
}

// DiskStats represents disk usage statistics
type DiskStats struct {
	Path        string  `json:"path"`
	Device      string  `json:"device"`
	Fstype      string  `json:"fstype"`
	Total       uint64  `json:"total"`
	Used        uint64  `json:"used"`
	Free        uint64  `json:"free"`
	UsedPercent float64 `json:"used_percent"`
	Model       string  `json:"model,omitempty"`  // Disk model name from SMART data
	Serial      string  `json:"serial,omitempty"` // Disk serial number from SMART data
}

// TemperatureStats represents temperature sensor data
type TemperatureStats struct {
	SensorKey   string  `json:"sensor_key"`
	Temperature float64 `json:"temperature"` // Celsius
	Label       string  `json:"label,omitempty"`
	Critical    float64 `json:"critical,omitempty"`
}
