// Copyright (c) 2024, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package types

import "time"

type TestOptions struct {
	EnableDownload   bool     `json:"enableDownload"`
	EnableUpload     bool     `json:"enableUpload"`
	EnablePacketLoss bool     `json:"enablePacketLoss"`
	ServerIDs        []string `json:"serverIds"`
	IsScheduled      bool     `json:"isScheduled"`
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
