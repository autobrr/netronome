// Copyright (c) 2024-2026, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package speedtest

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/autobrr/netronome/internal/types"
)

// Result describes a completed test and the identity of the server that actually ran it.
type Result struct {
	ID            int64     `json:"id"`
	Timestamp     time.Time `json:"timestamp"`
	Server        string    `json:"server"`
	ServerID      string    `json:"serverId,omitempty"`
	ServerHost    string    `json:"serverHost,omitempty"`
	ServerCity    string    `json:"serverCity,omitempty"`
	DownloadSpeed float64   `json:"downloadSpeed"`
	UploadSpeed   float64   `json:"uploadSpeed"`
	Latency       string    `json:"latency"`
	Jitter        float64   `json:"jitter"`
	Error         string    `json:"error,omitempty"`
	Download      float64   `json:"-"`
	Upload        float64   `json:"-"`
}

// ServerListOptions selects the Speedtest.net source used to populate the retained catalogue.
// Global and Location are mutually exclusive. Refresh always fetches from the selected source.
type ServerListOptions struct {
	Global   bool            // Global aggregates catalogues from known regions.
	Location *ServerLocation // Location requests the catalogue nearest this origin.
	Refresh  bool            // Refresh fetches the selected source even when it is already stored.
}

// ServerLocation identifies the geographic origin used to find nearby servers.
type ServerLocation struct {
	Latitude  float64
	Longitude float64
}

// ServerCatalogueStatus reports whether the selected source has been durably fetched.
type ServerCatalogueStatus struct {
	Stored    bool       `json:"stored"`
	UpdatedAt *time.Time `json:"updatedAt,omitempty"`
}

// PartialServerCatalogueError reports failed discovery sources after successful servers were retained.
type PartialServerCatalogueError struct {
	successfulLocations int
	totalLocations      int
	failures            []error
}

// Error summarizes how many regional discoveries succeeded and includes their failures.
func (e *PartialServerCatalogueError) Error() string {
	return fmt.Sprintf(
		"global speedtest catalogue updated from %d of %d regional locations: %v",
		e.successfulLocations,
		e.totalLocations,
		errors.Join(e.failures...),
	)
}

// Unwrap joins the regional discovery failures for errors.Is and errors.As matching.
func (e *PartialServerCatalogueError) Unwrap() error {
	return errors.Join(e.failures...)
}

// WarningMessages returns source-specific failures suitable for an API response.
func (e *PartialServerCatalogueError) WarningMessages() []string {
	warnings := make([]string, 0, len(e.failures))
	for _, failure := range e.failures {
		warnings = append(warnings, failure.Error())
	}
	return warnings
}

// ServerResponse describes a selectable speed test server returned by the server-list API.
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
	IsPublic     bool    `json:"isPublic"`
}

// ResultHandler handles database saves and notifications
type ResultHandler interface {
	SaveResult(ctx context.Context, result *Result, testType string, opts *types.TestOptions) error
	SendNotification(result *types.SpeedTestResult)
}
