// Copyright (c) 2024-2026, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package speedtest

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/autobrr/netronome/internal/types"
)

func TestResultForStorageKeepsSpeedtestServerIdentity(t *testing.T) {
	createdAt := time.Date(2026, time.September, 15, 1, 2, 3, 0, time.UTC)
	result := &Result{
		Server:     "Example ISP",
		ServerID:   "1234",
		ServerHost: "speed.example.com:8080",
		ServerCity: "Brisbane",
		Jitter:     1.5,
	}

	got := resultForStorage(result, "speedtest", &types.TestOptions{IsScheduled: true}, createdAt)
	assert.Equal(t, "Example ISP", got.ServerName)
	assert.Equal(t, "1234", got.ServerID)
	require.NotNil(t, got.ServerHost)
	assert.Equal(t, "speed.example.com:8080", *got.ServerHost)
	require.NotNil(t, got.ServerCity)
	assert.Equal(t, "Brisbane", *got.ServerCity)
	require.NotNil(t, got.Jitter)
	assert.Equal(t, 1.5, *got.Jitter)
	assert.True(t, got.IsScheduled)
	assert.Equal(t, createdAt, got.CreatedAt)
}

func TestResultForStoragePreservesLegacySpeedtestFallback(t *testing.T) {
	got := resultForStorage(&Result{Server: "Example ISP"}, "speedtest", &types.TestOptions{}, time.Time{})
	assert.Equal(t, "Example ISP", got.ServerID)
	assert.Nil(t, got.ServerHost)
	assert.Nil(t, got.ServerCity)
}

func TestResultForStorageKeepsLibrespeedServerIdentity(t *testing.T) {
	got := resultForStorage(
		&Result{
			Server:     "Example LibreSpeed",
			ServerID:   "42",
			ServerHost: "https://speed.example.com/",
		},
		"librespeed",
		&types.TestOptions{},
		time.Time{},
	)
	assert.Equal(t, "librespeed-42", got.ServerID)
	assert.Equal(t, "Example LibreSpeed", got.ServerName)
	require.NotNil(t, got.ServerHost)
	assert.Equal(t, "https://speed.example.com/", *got.ServerHost)
}

func TestResultForStoragePreservesLegacyLibrespeedFallback(t *testing.T) {
	got := resultForStorage(
		&Result{Server: "Example LibreSpeed"},
		"librespeed",
		&types.TestOptions{},
		time.Time{},
	)
	assert.Equal(t, "librespeed-Example LibreSpeed", got.ServerID)
	assert.Nil(t, got.ServerHost)
}
