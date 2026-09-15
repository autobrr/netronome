// Copyright (c) 2024-2026, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package speedtest

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	st "github.com/showwin/speedtest-go/speedtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/autobrr/netronome/internal/config"
	"github.com/autobrr/netronome/internal/database"
)

type memoryServerCatalogueStore struct {
	mu              sync.Mutex
	values          map[string]string
	getErr          error
	setErr          error
	setStarted      chan struct{}
	releaseSet      chan struct{}
	lastSetDeadline time.Time
}

func (s *memoryServerCatalogueStore) GetAppSetting(ctx context.Context, key string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.getErr != nil {
		return "", s.getErr
	}
	value, ok := s.values[key]
	if !ok {
		return "", database.ErrNotFound
	}
	return value, nil
}

func (s *memoryServerCatalogueStore) SetAppSetting(ctx context.Context, key, value string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s.setStarted != nil {
		select {
		case s.setStarted <- struct{}{}:
		default:
		}
		select {
		case <-s.releaseSet:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastSetDeadline, _ = ctx.Deadline()
	if s.setErr != nil {
		return s.setErr
	}
	if s.values == nil {
		s.values = make(map[string]string)
	}
	s.values[key] = value
	return nil
}

func (s *memoryServerCatalogueStore) setError(err error) {
	s.mu.Lock()
	s.setErr = err
	s.mu.Unlock()
}

func (s *memoryServerCatalogueStore) setGetError(err error) {
	s.mu.Lock()
	s.getErr = err
	s.mu.Unlock()
}

func (s *memoryServerCatalogueStore) value(key string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.values[key]
}

func (s *memoryServerCatalogueStore) setDeadline() time.Time {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.lastSetDeadline
}

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

func serverIDs(servers []ServerResponse) []string {
	ids := make([]string, 0, len(servers))
	for _, server := range servers {
		ids = append(ids, server.ID)
	}
	return ids
}

func TestServerCacheReturnsCopies(t *testing.T) {
	runner := NewSpeedtestNetRunner(config.SpeedTestConfig{}, nil)
	runner.cacheDuration = time.Minute
	runner.storeServerCache("test", []ServerResponse{{ID: "1"}}, nil)

	first, ok := runner.loadServerCache("test")
	require.True(t, ok)
	first[0].ID = "changed"

	second, ok := runner.loadServerCache("test")
	require.True(t, ok)
	assert.Equal(t, "1", second[0].ID)
}

func TestServerCacheRefreshRetainsValidEntry(t *testing.T) {
	runner := NewSpeedtestNetRunner(config.SpeedTestConfig{}, nil)
	runner.cacheDuration = time.Minute
	cachedServers := []ServerResponse{{ID: "cached"}}
	runner.retainServers(cachedServers)
	runner.storeServerCache("local", cachedServers, &ServerLocation{})

	var fetchCount atomic.Int32
	runner.fetchServers = func(_ context.Context, _ *ServerLocation) ([]ServerResponse, *ServerLocation, error) {
		fetchCount.Add(1)
		return []ServerResponse{{ID: "fresh"}}, &ServerLocation{}, nil
	}

	servers, err := runner.GetServersWithOptions(t.Context(), ServerListOptions{Refresh: true})
	require.NoError(t, err)
	assert.ElementsMatch(t, []ServerResponse{{ID: "cached"}, {ID: "fresh"}}, servers)
	assert.Equal(t, int32(1), fetchCount.Load())
	sourceServers, ok := runner.loadServerCache("local")
	require.True(t, ok)
	assert.Equal(t, []ServerResponse{{ID: "fresh"}}, sourceServers)

	servers, err = runner.GetServersWithOptions(t.Context(), ServerListOptions{})
	require.NoError(t, err)
	assert.ElementsMatch(t, []ServerResponse{{ID: "cached"}, {ID: "fresh"}}, servers)
	assert.Equal(t, int32(1), fetchCount.Load())
}

func TestServerCatalogueRetainsServersAcrossSources(t *testing.T) {
	runner := NewSpeedtestNetRunner(config.SpeedTestConfig{}, nil)
	runner.fetchServers = func(_ context.Context, location *ServerLocation) ([]ServerResponse, *ServerLocation, error) {
		if location == nil {
			return []ServerResponse{{ID: "local", Sponsor: "original"}}, &ServerLocation{}, nil
		}
		return []ServerResponse{{ID: "local", Sponsor: "updated"}, {ID: "coordinate"}}, nil, nil
	}

	servers, err := runner.GetServersWithOptions(t.Context(), ServerListOptions{})
	require.NoError(t, err)
	assert.Equal(t, []ServerResponse{{ID: "local", Sponsor: "original"}}, servers)

	servers, err = runner.GetServersWithOptions(t.Context(), ServerListOptions{
		Location: &ServerLocation{Latitude: 1, Longitude: 1},
	})
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"local", "coordinate"}, serverIDs(servers))
	for _, server := range servers {
		if server.ID == "local" {
			assert.Equal(t, "updated", server.Sponsor)
		}
	}

	servers, err = runner.GetServersWithOptions(t.Context(), ServerListOptions{})
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"local", "coordinate"}, serverIDs(servers))
}

func TestServerCataloguePersistsAcrossRunnerRestarts(t *testing.T) {
	store := &memoryServerCatalogueStore{}
	runner := NewSpeedtestNetRunner(config.SpeedTestConfig{}, store)
	runner.fetchServers = func(_ context.Context, location *ServerLocation) ([]ServerResponse, *ServerLocation, error) {
		if location == nil {
			return []ServerResponse{{ID: "shared", Sponsor: "local", Lat: 0, Lon: 1}}, &ServerLocation{}, nil
		}
		return []ServerResponse{
			{ID: "shared", Sponsor: "coordinate", Lat: 0, Lon: 1},
			{ID: "coordinate", Lat: 0, Lon: 2},
		}, nil, nil
	}

	_, err := runner.GetServersWithOptions(t.Context(), ServerListOptions{Refresh: true})
	require.NoError(t, err)
	_, err = runner.GetServersWithOptions(t.Context(), ServerListOptions{
		Location: &ServerLocation{Latitude: 1, Longitude: 1},
		Refresh:  true,
	})
	require.NoError(t, err)
	localStatus, err := runner.GetServerCatalogueStatus(t.Context(), ServerListOptions{})
	require.NoError(t, err)
	require.True(t, localStatus.Stored)
	require.NotNil(t, localStatus.UpdatedAt)
	coordinateStatus, err := runner.GetServerCatalogueStatus(t.Context(), ServerListOptions{
		Location: &ServerLocation{Latitude: 1, Longitude: 1},
	})
	require.NoError(t, err)
	require.True(t, coordinateStatus.Stored)
	require.NotNil(t, coordinateStatus.UpdatedAt)
	globalStatus, err := runner.GetServerCatalogueStatus(t.Context(), ServerListOptions{Global: true})
	require.NoError(t, err)
	assert.False(t, globalStatus.Stored)

	restarted := NewSpeedtestNetRunner(config.SpeedTestConfig{}, store)
	restarted.fetchServers = func(_ context.Context, _ *ServerLocation) ([]ServerResponse, *ServerLocation, error) {
		t.Fatal("persisted catalogue should not refetch Speedtest.net")
		return nil, nil, nil
	}
	servers, err := restarted.GetServersWithOptions(t.Context(), ServerListOptions{})
	require.NoError(t, err)
	require.Equal(t, []string{"shared", "coordinate"}, serverIDs(servers))
	assert.Equal(t, "coordinate", servers[0].Sponsor)
	assert.InDelta(t, 111.2, servers[0].Distance, 0.2)
	assert.InDelta(t, 222.4, servers[1].Distance, 0.2)
	restartedCoordinateStatus, err := restarted.GetServerCatalogueStatus(t.Context(), ServerListOptions{
		Location: &ServerLocation{Latitude: 1, Longitude: 1},
	})
	require.NoError(t, err)
	assert.Equal(t, coordinateStatus, restartedCoordinateStatus)
	differentCoordinateStatus, err := restarted.GetServerCatalogueStatus(t.Context(), ServerListOptions{
		Location: &ServerLocation{Latitude: 2, Longitude: 2},
	})
	require.NoError(t, err)
	assert.False(t, differentCoordinateStatus.Stored)
}

func TestServerCatalogueRetriesTransientStartupLoadFailure(t *testing.T) {
	store := &memoryServerCatalogueStore{}
	runner := NewSpeedtestNetRunner(config.SpeedTestConfig{}, store)
	runner.fetchServers = func(_ context.Context, _ *ServerLocation) ([]ServerResponse, *ServerLocation, error) {
		return []ServerResponse{{ID: "durable"}}, &ServerLocation{}, nil
	}
	_, err := runner.GetServersWithOptions(t.Context(), ServerListOptions{Refresh: true})
	require.NoError(t, err)
	storedValue := store.value(serverCatalogueSettingKey)

	store.setGetError(errors.New("database unavailable"))
	restarted := NewSpeedtestNetRunner(config.SpeedTestConfig{}, store)
	restarted.fetchServers = func(_ context.Context, _ *ServerLocation) ([]ServerResponse, *ServerLocation, error) {
		t.Fatal("a failed startup load must prevent upstream discovery")
		return nil, nil, nil
	}
	_, err = restarted.GetServerCatalogueStatus(t.Context(), ServerListOptions{})
	require.ErrorContains(t, err, "read retained speedtest servers")
	assert.Equal(t, storedValue, store.value(serverCatalogueSettingKey))

	store.setGetError(nil)
	status, err := restarted.GetServerCatalogueStatus(t.Context(), ServerListOptions{})
	require.NoError(t, err)
	assert.True(t, status.Stored)
}

func TestServerCatalogueMalformedStartupValueIsNotOverwritten(t *testing.T) {
	const malformed = `{not-json`
	store := &memoryServerCatalogueStore{values: map[string]string{
		serverCatalogueSettingKey: malformed,
	}}
	runner := NewSpeedtestNetRunner(config.SpeedTestConfig{}, store)
	var fetchCount atomic.Int32
	runner.fetchServers = func(_ context.Context, _ *ServerLocation) ([]ServerResponse, *ServerLocation, error) {
		fetchCount.Add(1)
		return []ServerResponse{{ID: "fresh"}}, &ServerLocation{}, nil
	}

	_, err := runner.GetServersWithOptions(t.Context(), ServerListOptions{Refresh: true})
	require.ErrorContains(t, err, "decode retained speedtest servers")
	assert.Zero(t, fetchCount.Load())
	assert.Equal(t, malformed, store.value(serverCatalogueSettingKey))
}

func TestServerCataloguePersistenceGetsIndependentDeadline(t *testing.T) {
	store := &memoryServerCatalogueStore{}
	runner := NewSpeedtestNetRunner(config.SpeedTestConfig{}, store)
	var fetchDeadline time.Time
	runner.fetchServers = func(ctx context.Context, _ *ServerLocation) ([]ServerResponse, *ServerLocation, error) {
		fetchDeadline, _ = ctx.Deadline()
		time.Sleep(10 * time.Millisecond)
		return []ServerResponse{{ID: "fresh"}}, &ServerLocation{}, nil
	}

	_, err := runner.GetServersWithOptions(t.Context(), ServerListOptions{Refresh: true})
	require.NoError(t, err)
	assert.True(t, store.setDeadline().After(fetchDeadline))
}

func TestServerCataloguePersistenceFailureIsReturned(t *testing.T) {
	store := &memoryServerCatalogueStore{}
	runner := NewSpeedtestNetRunner(config.SpeedTestConfig{}, store)
	var fetchCount atomic.Int32
	runner.fetchServers = func(_ context.Context, _ *ServerLocation) ([]ServerResponse, *ServerLocation, error) {
		if fetchCount.Add(1) == 1 {
			return []ServerResponse{{ID: "durable"}}, &ServerLocation{}, nil
		}
		return []ServerResponse{{ID: "pending"}}, &ServerLocation{}, nil
	}

	_, err := runner.GetServersWithOptions(t.Context(), ServerListOptions{Refresh: true})
	require.NoError(t, err)
	store.setError(errors.New("database unavailable"))
	servers, err := runner.GetServersWithOptions(t.Context(), ServerListOptions{Refresh: true})
	require.ErrorContains(t, err, "write retained speedtest servers")
	assert.Nil(t, servers)
	assert.NotEqual(t, runner.catalogueVersion.Load(), runner.storedVersion.Load())

	servers, err = runner.GetServersWithOptions(t.Context(), ServerListOptions{})
	require.ErrorContains(t, err, "write retained speedtest servers")
	assert.Nil(t, servers)

	store.setError(nil)
	servers, err = runner.GetServersWithOptions(t.Context(), ServerListOptions{})
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"durable", "pending"}, serverIDs(servers))
	assert.Equal(t, runner.catalogueVersion.Load(), runner.storedVersion.Load())

	restarted := NewSpeedtestNetRunner(config.SpeedTestConfig{}, store)
	assert.ElementsMatch(t, []string{"durable", "pending"}, serverIDs(restarted.loadRetainedServers(nil)))
}

func TestServerCatalogueDoesNotExposePendingFirstWrite(t *testing.T) {
	store := &memoryServerCatalogueStore{
		setErr:     errors.New("database unavailable"),
		setStarted: make(chan struct{}, 1),
		releaseSet: make(chan struct{}),
	}
	runner := NewSpeedtestNetRunner(config.SpeedTestConfig{}, store)
	runner.fetchServers = func(_ context.Context, _ *ServerLocation) ([]ServerResponse, *ServerLocation, error) {
		return []ServerResponse{{ID: "pending"}}, &ServerLocation{}, nil
	}

	refreshDone := make(chan error, 1)
	go func() {
		_, err := runner.GetServersWithOptions(t.Context(), ServerListOptions{Refresh: true})
		refreshDone <- err
	}()
	<-store.setStarted
	assert.Empty(t, runner.loadAvailableServers(nil))
	close(store.releaseSet)
	require.ErrorContains(t, <-refreshDone, "write retained speedtest servers")

	servers, err := runner.GetServersWithOptions(t.Context(), ServerListOptions{})
	require.ErrorContains(t, err, "write retained speedtest servers")
	assert.Nil(t, servers)
}

func TestServerCatalogueSurvivesSourceCacheExpiry(t *testing.T) {
	runner := NewSpeedtestNetRunner(config.SpeedTestConfig{}, nil)
	oldServers := []ServerResponse{{ID: "old"}}
	runner.retainServers(oldServers)
	runner.storeServerCache("old", oldServers, nil)
	runner.cacheMu.Lock()
	runner.serverCache["old"] = serverCacheEntry{expiresAt: time.Now().Add(-time.Minute)}
	runner.cacheMu.Unlock()

	newServers := []ServerResponse{{ID: "new"}}
	runner.retainServers(newServers)
	runner.storeServerCache("new", newServers, nil)

	_, cached := runner.loadServerCache("old")
	assert.False(t, cached)
	assert.ElementsMatch(t, []string{"old", "new"}, serverIDs(runner.loadRetainedServers(nil)))
}

func TestServerCacheCoalescesConcurrentMissesByKey(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		runner := NewSpeedtestNetRunner(config.SpeedTestConfig{}, nil)
		var fetchCount atomic.Int32
		release := make(chan struct{})
		runner.fetchServers = func(_ context.Context, _ *ServerLocation) ([]ServerResponse, *ServerLocation, error) {
			fetchCount.Add(1)
			<-release
			return []ServerResponse{{ID: "1"}}, &ServerLocation{}, nil
		}

		type fetchResult struct {
			servers []ServerResponse
			err     error
		}
		results := make(chan fetchResult, 2)
		var wg sync.WaitGroup
		for range 2 {
			wg.Go(func() {
				servers, err := runner.getServersForLocation(t.Context(), "local", nil, false)
				results <- fetchResult{servers: servers, err: err}
			})
		}

		synctest.Wait()
		assert.Equal(t, int32(1), fetchCount.Load())
		close(release)
		wg.Wait()
		close(results)
		for result := range results {
			require.NoError(t, result.err)
			assert.Equal(t, []ServerResponse{{ID: "1"}}, result.servers)
		}
	})
}

func TestServerCacheFetchesDifferentKeysIndependently(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		runner := NewSpeedtestNetRunner(config.SpeedTestConfig{}, nil)
		var fetchCount atomic.Int32
		release := make(chan struct{})
		runner.fetchServers = func(_ context.Context, location *ServerLocation) ([]ServerResponse, *ServerLocation, error) {
			fetchCount.Add(1)
			<-release
			return []ServerResponse{{ID: fmt.Sprint(location.Latitude)}}, nil, nil
		}

		results := make(chan error, 2)
		var wg sync.WaitGroup
		for i := 1; i <= 2; i++ {
			location := &ServerLocation{Latitude: float64(i)}
			wg.Go(func() {
				_, err := runner.getServersForLocation(
					t.Context(),
					fmt.Sprintf("location:%d", i),
					location,
					false,
				)
				results <- err
			})
		}

		synctest.Wait()
		assert.Equal(t, int32(2), fetchCount.Load())
		close(release)
		wg.Wait()
		close(results)
		for err := range results {
			require.NoError(t, err)
		}
	})
}

func TestServerCacheCallerCancellationDoesNotAbortSharedFetch(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		runner := NewSpeedtestNetRunner(config.SpeedTestConfig{}, nil)
		fetchContextDone := make(chan (<-chan struct{}), 1)
		releaseFetch := make(chan struct{})
		runner.fetchServers = func(ctx context.Context, _ *ServerLocation) ([]ServerResponse, *ServerLocation, error) {
			fetchContextDone <- ctx.Done()
			select {
			case <-releaseFetch:
				return []ServerResponse{{ID: "1"}}, &ServerLocation{}, nil
			case <-ctx.Done():
				return nil, nil, ctx.Err()
			}
		}

		leaderCtx, cancelLeader := context.WithCancel(t.Context())
		leaderResult := make(chan error, 1)
		waiterResult := make(chan error, 1)
		var wg sync.WaitGroup
		wg.Go(func() {
			_, err := runner.getServersForLocation(leaderCtx, "local", nil, false)
			leaderResult <- err
		})
		sharedDone := <-fetchContextDone
		wg.Go(func() {
			_, err := runner.getServersForLocation(t.Context(), "local", nil, false)
			waiterResult <- err
		})

		synctest.Wait()
		cancelLeader()
		synctest.Wait()
		require.ErrorIs(t, <-leaderResult, context.Canceled)
		select {
		case <-sharedDone:
			t.Fatal("shared fetch context canceled with first caller")
		default:
		}

		close(releaseFetch)
		wg.Wait()
		require.NoError(t, <-waiterResult)
	})
}

func TestGlobalServersRejectsCompleteRegionalFailure(t *testing.T) {
	runner := NewSpeedtestNetRunner(config.SpeedTestConfig{}, nil)
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

	servers, err := runner.getGlobalServers(context.Background(), false)
	require.Error(t, err)
	assert.Nil(t, servers)
	_, cached := runner.loadServerCache("global")
	assert.False(t, cached)
}

func TestGlobalServersDoesNotCachePartialResults(t *testing.T) {
	runner := NewSpeedtestNetRunner(config.SpeedTestConfig{}, nil)
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

	servers, err := runner.getGlobalServers(context.Background(), false)
	require.NoError(t, err)
	assert.Len(t, servers, 2)
	_, cached := runner.loadServerCache("global")
	assert.False(t, cached)
}

func TestGlobalServersRefreshRetainsPreviousGlobalAndLocalServers(t *testing.T) {
	runner := NewSpeedtestNetRunner(config.SpeedTestConfig{}, nil)
	runner.cacheDuration = time.Minute
	runner.globalLocations = map[string]ServerLocation{
		"regional": {Latitude: 1, Longitude: 1},
	}
	cachedGlobal := []ServerResponse{{ID: "cached-global"}}
	cachedLocal := []ServerResponse{{ID: "cached-local"}}
	runner.retainServers(cachedGlobal)
	runner.retainServers(cachedLocal)
	runner.storeServerCache("global", cachedGlobal, nil)
	runner.storeServerCache("local", cachedLocal, &ServerLocation{})

	var fetchCount atomic.Int32
	runner.fetchServers = func(_ context.Context, location *ServerLocation) ([]ServerResponse, *ServerLocation, error) {
		fetchCount.Add(1)
		if location == nil {
			return []ServerResponse{{ID: "fresh-local"}}, &ServerLocation{}, nil
		}
		return []ServerResponse{{ID: "fresh-regional", Lat: location.Latitude, Lon: location.Longitude}}, nil, nil
	}

	servers, err := runner.GetServersWithOptions(t.Context(), ServerListOptions{Global: true, Refresh: true})
	require.NoError(t, err)
	require.Len(t, servers, 4)
	assert.ElementsMatch(t, []string{"cached-global", "cached-local", "fresh-local", "fresh-regional"}, serverIDs(servers))
	assert.Equal(t, int32(2), fetchCount.Load())

	cached, ok := runner.loadServerCache("global")
	require.True(t, ok)
	assert.ElementsMatch(t, servers, cached)
	status, err := runner.GetServerCatalogueStatus(t.Context(), ServerListOptions{Global: true})
	require.NoError(t, err)
	assert.True(t, status.Stored)
}

func TestGlobalServerCacheDoesNotOverwriteNewerRetainedMetadata(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		store := &memoryServerCatalogueStore{}
		runner := NewSpeedtestNetRunner(config.SpeedTestConfig{}, store)
		runner.globalLocations = map[string]ServerLocation{
			"duplicate": {Latitude: 1, Longitude: 1},
			"slow":      {Latitude: 2, Longitude: 2},
		}
		duplicateFetched := make(chan struct{}, 1)
		slowStarted := make(chan struct{}, 1)
		releaseSlow := make(chan struct{})
		var releaseOnce sync.Once
		release := func() {
			releaseOnce.Do(func() { close(releaseSlow) })
		}
		t.Cleanup(release)
		runner.fetchServers = func(_ context.Context, location *ServerLocation) ([]ServerResponse, *ServerLocation, error) {
			if location == nil {
				return []ServerResponse{{ID: "local"}}, &ServerLocation{}, nil
			}
			switch location.Latitude {
			case 1:
				duplicateFetched <- struct{}{}
				return []ServerResponse{{ID: "shared", Sponsor: "global-old"}}, nil, nil
			case 2:
				slowStarted <- struct{}{}
				<-releaseSlow
				return []ServerResponse{{ID: "slow"}}, nil, nil
			default:
				return []ServerResponse{{ID: "shared", Sponsor: "coordinate-new"}}, nil, nil
			}
		}

		type globalResult struct {
			servers []ServerResponse
			err     error
		}
		globalDone := make(chan globalResult, 1)
		go func() {
			servers, err := runner.GetServersWithOptions(t.Context(), ServerListOptions{Global: true, Refresh: true})
			globalDone <- globalResult{servers: servers, err: err}
		}()

		<-duplicateFetched
		<-slowStarted
		synctest.Wait()
		runner.cacheMu.RLock()
		retainedSponsor := runner.retainedServers["shared"].Sponsor
		runner.cacheMu.RUnlock()
		require.Equal(t, "global-old", retainedSponsor)

		_, err := runner.GetServersWithOptions(t.Context(), ServerListOptions{
			Location: &ServerLocation{Latitude: 3, Longitude: 3},
			Refresh:  true,
		})
		require.NoError(t, err)
		release()

		result := <-globalDone
		require.NoError(t, result.err)
		var shared ServerResponse
		for _, server := range result.servers {
			if server.ID == "shared" {
				shared = server
				break
			}
		}
		require.Equal(t, "shared", shared.ID)
		assert.Equal(t, "coordinate-new", shared.Sponsor)

		restarted := NewSpeedtestNetRunner(config.SpeedTestConfig{}, store)
		for _, server := range restarted.loadRetainedServers(nil) {
			if server.ID == "shared" {
				assert.Equal(t, "coordinate-new", server.Sponsor)
				return
			}
		}
		t.Fatal("persisted shared server not found")
	})
}

func TestGlobalServersRefreshPreservesCacheOnPartialFailure(t *testing.T) {
	runner := NewSpeedtestNetRunner(config.SpeedTestConfig{}, nil)
	runner.cacheDuration = time.Minute
	runner.globalLocations = map[string]ServerLocation{
		"success": {Latitude: 1, Longitude: 1},
		"failure": {Latitude: 2, Longitude: 2},
	}
	previous := []ServerResponse{{ID: "cached-global"}}
	runner.retainServers(previous)
	runner.storeServerCache("global", previous, nil)
	runner.fetchServers = func(_ context.Context, location *ServerLocation) ([]ServerResponse, *ServerLocation, error) {
		if location == nil {
			return []ServerResponse{{ID: "fresh-local"}}, &ServerLocation{}, nil
		}
		if location.Latitude == 1 {
			return []ServerResponse{{ID: "fresh-regional"}}, nil, nil
		}
		return nil, nil, errors.New("regional fetch failed")
	}

	servers, err := runner.GetServersWithOptions(t.Context(), ServerListOptions{Global: true, Refresh: true})
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"cached-global", "fresh-local", "fresh-regional"}, serverIDs(servers))

	cached, ok := runner.loadServerCache("global")
	require.True(t, ok)
	assert.Equal(t, previous, cached)
	assert.ElementsMatch(t, []string{"cached-global", "fresh-local", "fresh-regional"}, serverIDs(runner.loadRetainedServers(nil)))
}

func TestGlobalServersPersistsCompletedRegionsAfterCancellation(t *testing.T) {
	store := &memoryServerCatalogueStore{}
	runner := NewSpeedtestNetRunner(config.SpeedTestConfig{}, store)
	runner.globalLocations = map[string]ServerLocation{
		"completed": {Latitude: 1, Longitude: 1},
		"blocked":   {Latitude: 2, Longitude: 2},
	}
	completed := make(chan struct{}, 1)
	blocked := make(chan struct{}, 1)
	runner.fetchServers = func(ctx context.Context, location *ServerLocation) ([]ServerResponse, *ServerLocation, error) {
		if location == nil {
			return []ServerResponse{{ID: "local"}}, &ServerLocation{}, nil
		}
		if location.Latitude == 1 {
			completed <- struct{}{}
			return []ServerResponse{{ID: "regional"}}, nil, nil
		}
		blocked <- struct{}{}
		<-ctx.Done()
		return nil, nil, ctx.Err()
	}

	ctx, cancel := context.WithCancel(t.Context())
	t.Cleanup(cancel)
	type globalResult struct {
		servers []ServerResponse
		err     error
	}
	done := make(chan globalResult, 1)
	go func() {
		servers, err := runner.GetServersWithOptions(ctx, ServerListOptions{Global: true, Refresh: true})
		done <- globalResult{servers: servers, err: err}
	}()

	<-completed
	<-blocked
	cancel()
	result := <-done
	require.ErrorIs(t, result.err, context.Canceled)
	assert.Nil(t, result.servers)

	restarted := NewSpeedtestNetRunner(config.SpeedTestConfig{}, store)
	assert.ElementsMatch(t, []string{"local", "regional"}, serverIDs(restarted.loadRetainedServers(nil)))
}

func TestGlobalServersWaitHonorsContext(t *testing.T) {
	runner := NewSpeedtestNetRunner(config.SpeedTestConfig{}, nil)
	runner.globalFetch <- struct{}{}
	defer func() { <-runner.globalFetch }()

	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Millisecond)
	defer cancel()
	_, err := runner.getGlobalServers(ctx, false)
	require.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestServerCacheBoundsCoordinateEntriesAndSweepsExpiredEntries(t *testing.T) {
	runner := NewSpeedtestNetRunner(config.SpeedTestConfig{}, nil)
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
