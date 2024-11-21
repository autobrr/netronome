package types

import "time"

type TestOptions struct {
	EnableDownload   bool     `json:"enableDownload"`
	EnableUpload     bool     `json:"enableUpload"`
	EnablePacketLoss bool     `json:"enablePacketLoss"`
	ServerIDs        []string `json:"serverIds"`
}

type SpeedUpdate struct {
	Type       string  `json:"type"`
	ServerName string  `json:"serverName"`
	Speed      float64 `json:"speed"`
	Progress   float64 `json:"progress"`
	IsComplete bool    `json:"isComplete"`
	Latency    string  `json:"latency,omitempty"`
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
