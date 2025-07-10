// Copyright (c) 2024-2025, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package types

import (
	"time"
)

type TestOptions struct {
	EnableDownload   bool     `json:"enableDownload"`
	EnableUpload     bool     `json:"enableUpload"`
	EnablePacketLoss bool     `json:"enablePacketLoss"`
	EnablePing       bool     `json:"enablePing"`
	EnableJitter     bool     `json:"enableJitter"`
	ServerIDs        []string `json:"serverIds"`
	IsScheduled      bool     `json:"isScheduled"`
	UseIperf         bool     `json:"useIperf"`
	UseLibrespeed    bool     `json:"useLibrespeed"`
	ServerHost       string   `json:"serverHost"`
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

type Schedule struct {
	ID        int64       `json:"id"`
	ServerIDs []string    `json:"serverIds"`
	Interval  string      `json:"interval"`
	LastRun   *time.Time  `json:"lastRun,omitempty"`
	NextRun   time.Time   `json:"nextRun"`
	Enabled   bool        `json:"enabled"`
	Options   TestOptions `json:"options"`
	CreatedAt time.Time   `json:"createdAt"`
}

type SpeedTestResult struct {
	ID            int64     `json:"id"`
	ServerName    string    `json:"serverName"`
	ServerID      string    `json:"serverId"`
	ServerHost    *string   `json:"serverHost,omitempty"`
	TestType      string    `json:"testType"`
	DownloadSpeed float64   `json:"downloadSpeed"`
	UploadSpeed   float64   `json:"uploadSpeed"`
	Latency       string    `json:"latency,omitempty"`
	PacketLoss    float64   `json:"packetLoss,omitempty"`
	Jitter        *float64  `json:"jitter,omitempty"`
	IsScheduled   bool      `json:"isScheduled"`
	CreatedAt     time.Time `json:"createdAt"`
}

type PaginatedSpeedTests struct {
	Data  []SpeedTestResult `json:"data"`
	Total int               `json:"total"`
	Page  int               `json:"page"`
	Limit int               `json:"limit"`
}

type SavedIperfServer struct {
	ID        int       `db:"id" json:"id"`
	Name      string    `db:"name" json:"name"`
	Host      string    `db:"host" json:"host"`
	Port      int       `db:"port" json:"port"`
	CreatedAt time.Time `db:"created_at" json:"createdAt"`
	UpdatedAt time.Time `db:"updated_at" json:"updatedAt"`
}

type TracerouteUpdate struct {
	Type            string      `json:"type"`
	Host            string      `json:"host"`
	Progress        float64     `json:"progress"`
	IsComplete      bool        `json:"isComplete"`
	CurrentHop      int         `json:"currentHop"`
	TotalHops       int         `json:"totalHops"`
	IsScheduled     bool        `json:"isScheduled"`
	Hops            interface{} `json:"hops"` // Will be []TracerouteHop from speedtest package
	Destination     string      `json:"destination"`
	IP              string      `json:"ip"`
	TerminatedEarly bool        `json:"terminatedEarly,omitempty"`
}

// MTRHop represents a single hop in MTR results
type MTRHop struct {
	Number      int     `json:"number"`
	Host        string  `json:"host"`
	IP          string  `json:"ip,omitempty"`
	PacketLoss  float64 `json:"loss"`
	Sent        int     `json:"sent"`
	Recv        int     `json:"recv"`
	Last        float64 `json:"last"`
	Avg         float64 `json:"avg"`
	Best        float64 `json:"best"`
	Worst       float64 `json:"worst"`
	StdDev      float64 `json:"stddev"`
	CountryCode string  `json:"countryCode,omitempty"`
	AS          string  `json:"as,omitempty"`
}

// MTRData represents the complete MTR test results
type MTRData struct {
	Destination string   `json:"destination"`
	IP          string   `json:"ip"`
	HopCount    int      `json:"hopCount"`
	Tests       int      `json:"tests"`
	Hops        []MTRHop `json:"hops"`
}

type PacketLossUpdate struct {
	Type        string  `json:"type"`
	MonitorID   int64   `json:"monitorId"`
	Host        string  `json:"host"`
	IsRunning   bool    `json:"isRunning"`
	IsComplete  bool    `json:"isComplete"`
	Progress    float64 `json:"progress"`
	PacketLoss  float64 `json:"packetLoss,omitempty"`
	MinRTT      float64 `json:"minRtt,omitempty"`
	MaxRTT      float64 `json:"maxRtt,omitempty"`
	AvgRTT      float64 `json:"avgRtt,omitempty"`
	StdDevRTT   float64 `json:"stdDevRtt,omitempty"`
	PacketsSent int     `json:"packetsSent,omitempty"`
	PacketsRecv int     `json:"packetsRecv,omitempty"`
	UsedMTR     bool    `json:"usedMtr,omitempty"`
	HopCount    int     `json:"hopCount,omitempty"`
	Error       string  `json:"error,omitempty"`
}

type PacketLossMonitor struct {
	ID          int64     `db:"id" json:"id"`
	Host        string    `db:"host" json:"host"`
	Name        string    `db:"name" json:"name"`
	Interval    int       `db:"interval" json:"interval"`
	PacketCount int       `db:"packet_count" json:"packetCount"`
	Enabled     bool      `db:"enabled" json:"enabled"`
	Threshold   float64   `db:"threshold" json:"threshold"`
	CreatedAt   time.Time `db:"created_at" json:"createdAt"`
	UpdatedAt   time.Time `db:"updated_at" json:"updatedAt"`
}

type PacketLossResult struct {
	ID             int64     `db:"id" json:"id"`
	MonitorID      int64     `db:"monitor_id" json:"monitorId"`
	PacketLoss     float64   `db:"packet_loss" json:"packetLoss"`
	MinRTT         float64   `db:"min_rtt" json:"minRtt"`
	MaxRTT         float64   `db:"max_rtt" json:"maxRtt"`
	AvgRTT         float64   `db:"avg_rtt" json:"avgRtt"`
	StdDevRTT      float64   `db:"std_dev_rtt" json:"stdDevRtt"`
	PacketsSent    int       `db:"packets_sent" json:"packetsSent"`
	PacketsRecv    int       `db:"packets_recv" json:"packetsRecv"`
	UsedMTR        bool      `db:"used_mtr" json:"usedMtr"`
	HopCount       int       `db:"hop_count" json:"hopCount"`
	MTRData        *string   `db:"mtr_data" json:"mtrData,omitempty"`
	PrivilegedMode bool      `db:"privileged_mode" json:"privilegedMode"`
	CreatedAt      time.Time `db:"created_at" json:"createdAt"`
}
