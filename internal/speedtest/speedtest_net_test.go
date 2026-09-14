// Copyright (c) 2024-2026, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package speedtest

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	st "github.com/showwin/speedtest-go/speedtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/autobrr/netronome/internal/config"
)

func TestServerResponsesFallsBackToUnpingableServers(t *testing.T) {
	servers := st.Servers{
		&st.Server{ID: "2", Name: "Sydney", Sponsor: "Example", Lat: "-33.8688", Lon: "151.2093", Distance: 10, Latency: st.PingTimeout},
		&st.Server{ID: "1", Name: "Brisbane", Sponsor: "Example", Lat: "-27.4698", Lon: "153.0251", Distance: 5, Latency: st.PingTimeout},
	}

	got := serverResponses(servers)
	require.Len(t, got, 2)
	assert.Equal(t, "1", got[0].ID)
	assert.Equal(t, "Brisbane", got[0].Name)
}

func TestMergeServerListsDeduplicatesAndSortsFromUser(t *testing.T) {
	origin := ServerLocation{Latitude: 0, Longitude: 0}
	servers := []ServerResponse{
		{ID: "far", Lat: 0, Lon: 2},
		{ID: "near", Lat: 0, Lon: 1, Sponsor: "first"},
		{ID: "near", Lat: 0, Lon: 1, Sponsor: "latest"},
	}

	got := mergeServerLists(origin, servers)
	require.Len(t, got, 2)
	assert.Equal(t, "near", got[0].ID)
	assert.Equal(t, "latest", got[0].Sponsor)
	assert.InDelta(t, 111.2, got[0].Distance, 0.2)
	assert.Greater(t, got[1].Distance, got[0].Distance)
}

func TestServerCacheReturnsCopies(t *testing.T) {
	runner := NewSpeedtestNetRunner(config.SpeedTestConfig{})
	runner.cacheDuration = time.Minute
	runner.storeServerCache("test", []ServerResponse{{ID: "1"}}, nil)

	first, ok := runner.loadServerCache("test")
	require.True(t, ok)
	first[0].ID = "changed"

	second, ok := runner.loadServerCache("test")
	require.True(t, ok)
	assert.Equal(t, "1", second[0].ID)
}

func TestGlobalServersRejectsCompleteRegionalFailure(t *testing.T) {
	runner := NewSpeedtestNetRunner(config.SpeedTestConfig{})
	runner.globalLocations = map[string]ServerLocation{
		"one": {Latitude: 1, Longitude: 1},
		"two": {Latitude: 2, Longitude: 2},
	}
	runner.fetchServers = func(_ context.Context, location *ServerLocation) ([]ServerResponse, *ServerLocation, error) {
		if location == nil {
			return []ServerResponse{{ID: "local"}}, &ServerLocation{}, nil
		}
		return nil, nil, errors.New("regional fetch failed")
	}

	servers, err := runner.getGlobalServers(context.Background())
	require.Error(t, err)
	assert.Nil(t, servers)
	_, cached := runner.loadServerCache("global")
	assert.False(t, cached)
}

func TestGlobalServersDoesNotCachePartialResults(t *testing.T) {
	runner := NewSpeedtestNetRunner(config.SpeedTestConfig{})
	runner.globalLocations = map[string]ServerLocation{
		"success": {Latitude: 1, Longitude: 1},
		"failure": {Latitude: 2, Longitude: 2},
	}
	runner.fetchServers = func(_ context.Context, location *ServerLocation) ([]ServerResponse, *ServerLocation, error) {
		if location == nil {
			return []ServerResponse{{ID: "local"}}, &ServerLocation{}, nil
		}
		if location.Latitude == 1 {
			return []ServerResponse{{ID: "regional", Lat: 1, Lon: 1}}, nil, nil
		}
		return nil, nil, errors.New("regional fetch failed")
	}

	servers, err := runner.getGlobalServers(context.Background())
	require.NoError(t, err)
	assert.Len(t, servers, 2)
	_, cached := runner.loadServerCache("global")
	assert.False(t, cached)
}

func TestGlobalServersWaitHonorsContext(t *testing.T) {
	runner := NewSpeedtestNetRunner(config.SpeedTestConfig{})
	runner.globalFetch <- struct{}{}
	defer func() { <-runner.globalFetch }()

	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Millisecond)
	defer cancel()
	_, err := runner.getGlobalServers(ctx)
	require.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestServerCacheBoundsCoordinateEntriesAndSweepsExpiredEntries(t *testing.T) {
	runner := NewSpeedtestNetRunner(config.SpeedTestConfig{})
	runner.cacheDuration = time.Minute
	runner.storeServerCache("local", []ServerResponse{{ID: "local"}}, nil)

	runner.cacheMu.Lock()
	runner.serverCache["location:expired"] = serverCacheEntry{
		servers:   []ServerResponse{{ID: "expired"}},
		expiresAt: time.Now().Add(-time.Minute),
	}
	runner.cacheMu.Unlock()

	for i := range 40 {
		runner.storeServerCache(fmt.Sprintf("location:%d", i), []ServerResponse{{ID: fmt.Sprint(i)}}, nil)
	}

	runner.cacheMu.RLock()
	defer runner.cacheMu.RUnlock()
	coordinateEntries := 0
	for key := range runner.serverCache {
		if strings.HasPrefix(key, "location:") {
			coordinateEntries++
		}
	}
	assert.LessOrEqual(t, coordinateEntries, 32)
	assert.NotContains(t, runner.serverCache, "location:expired")
	assert.Contains(t, runner.serverCache, "local")
}
