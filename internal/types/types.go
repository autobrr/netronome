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
	TestType    string  `json:"testType,omitempty"` // "speedtest", "iperf3", "librespeed"
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
	ID          int64      `db:"id" json:"id"`
	Host        string     `db:"host" json:"host"`
	Name        string     `db:"name" json:"name"`
	Interval    string     `db:"interval" json:"interval"` // Changed from int to string
	PacketCount int        `db:"packet_count" json:"packetCount"`
	Enabled     bool       `db:"enabled" json:"enabled"`
	Threshold   float64    `db:"threshold" json:"threshold"`
	LastRun     *time.Time `db:"last_run" json:"lastRun"` // New field
	NextRun     *time.Time `db:"next_run" json:"nextRun"` // New field
	CreatedAt   time.Time  `db:"created_at" json:"createdAt"`
	UpdatedAt   time.Time  `db:"updated_at" json:"updatedAt"`
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

// VnstatAgent represents a vnstat agent configuration
type VnstatAgent struct {
	ID            int64     `db:"id" json:"id"`
	Name          string    `db:"name" json:"name"`
	URL           string    `db:"url" json:"url"`
	Enabled       bool      `db:"enabled" json:"enabled"`
	Interface     *string   `db:"interface" json:"interface,omitempty"`
	RetentionDays int       `db:"retention_days" json:"retentionDays"`
	CreatedAt     time.Time `db:"created_at" json:"createdAt"`
	UpdatedAt     time.Time `db:"updated_at" json:"updatedAt"`
}

// VnstatBandwidth represents bandwidth data from vnstat
type VnstatBandwidth struct {
	ID                 int64     `db:"id" json:"id"`
	AgentID            int64     `db:"agent_id" json:"agentId"`
	RxBytesPerSecond   *int64    `db:"rx_bytes_per_second" json:"rxBytesPerSecond"`
	TxBytesPerSecond   *int64    `db:"tx_bytes_per_second" json:"txBytesPerSecond"`
	RxPacketsPerSecond *int      `db:"rx_packets_per_second" json:"rxPacketsPerSecond"`
	TxPacketsPerSecond *int      `db:"tx_packets_per_second" json:"txPacketsPerSecond"`
	RxRateString       *string   `db:"rx_rate_string" json:"rxRateString"`
	TxRateString       *string   `db:"tx_rate_string" json:"txRateString"`
	CreatedAt          time.Time `db:"created_at" json:"createdAt"`
}

// VnstatLiveData represents live data from vnstat agent
type VnstatLiveData struct {
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

// VnstatFullData represents the complete vnstat JSON export structure
type VnstatFullData struct {
	Vnstatversion string `json:"vnstatversion"`
	Jsonversion   string `json:"jsonversion"`
	Interfaces    []struct {
		Name    string `json:"name"`
		Alias   string `json:"alias"`
		Created struct {
			Date struct {
				Year  int `json:"year"`
				Month int `json:"month"`
				Day   int `json:"day"`
			} `json:"date"`
		} `json:"created"`
		Updated struct {
			Date struct {
				Year  int `json:"year"`
				Month int `json:"month"`
				Day   int `json:"day"`
			} `json:"date"`
			Time struct {
				Hour   int `json:"hour"`
				Minute int `json:"minute"`
			} `json:"time"`
		} `json:"updated"`
		Traffic struct {
			Total struct {
				Rx int64 `json:"rx"`
				Tx int64 `json:"tx"`
			} `json:"total"`
			Fiveminute []struct {
				ID   int `json:"id"`
				Date struct {
					Year  int `json:"year"`
					Month int `json:"month"`
					Day   int `json:"day"`
				} `json:"date"`
				Time struct {
					Hour   int `json:"hour"`
					Minute int `json:"minute"`
				} `json:"time"`
				Rx int64 `json:"rx"`
				Tx int64 `json:"tx"`
			} `json:"fiveminute"`
			Hour []struct {
				ID   int `json:"id"`
				Date struct {
					Year  int `json:"year"`
					Month int `json:"month"`
					Day   int `json:"day"`
				} `json:"date"`
				Hour int   `json:"hour"`
				Rx   int64 `json:"rx"`
				Tx   int64 `json:"tx"`
			} `json:"hour"`
			Day []struct {
				ID   int `json:"id"`
				Date struct {
					Year  int `json:"year"`
					Month int `json:"month"`
					Day   int `json:"day"`
				} `json:"date"`
				Rx int64 `json:"rx"`
				Tx int64 `json:"tx"`
			} `json:"day"`
			Month []struct {
				ID   int `json:"id"`
				Date struct {
					Year  int `json:"year"`
					Month int `json:"month"`
				} `json:"date"`
				Rx int64 `json:"rx"`
				Tx int64 `json:"tx"`
			} `json:"month"`
			Year []struct {
				ID   int `json:"id"`
				Date struct {
					Year int `json:"year"`
				} `json:"date"`
				Rx int64 `json:"rx"`
				Tx int64 `json:"tx"`
			} `json:"year"`
		} `json:"traffic"`
	} `json:"interfaces"`
}

// VnstatUpdate represents real-time vnstat updates
type VnstatUpdate struct {
	Type             string `json:"type"`
	AgentID          int64  `json:"agentId"`
	AgentName        string `json:"agentName"`
	RxBytesPerSecond int64  `json:"rxBytesPerSecond"`
	TxBytesPerSecond int64  `json:"txBytesPerSecond"`
	RxRateString     string `json:"rxRateString"`
	TxRateString     string `json:"txRateString"`
	Connected        bool   `json:"connected"`
}
