// Copyright (c) 2024-2026, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package speedtest

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
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
	mu               sync.Mutex
	servers          map[string]database.SpeedtestServer
	sources          map[string]database.SpeedtestServerSource
	saveErr          error
	lastSaveDeadline time.Time
}

func (s *memoryServerCatalogueStore) ListSpeedtestServers(ctx context.Context) ([]database.SpeedtestServer, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	servers := make([]database.SpeedtestServer, 0, len(s.servers))
	for _, server := range s.servers {
		servers = append(servers, server)
	}
	return servers, nil
}

func (s *memoryServerCatalogueStore) GetSpeedtestServerSource(
	ctx context.Context,
	key string,
) (database.SpeedtestServerSource, bool, error) {
	if err := ctx.Err(); err != nil {
		return database.SpeedtestServerSource{}, false, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	source, ok := s.sources[key]
	return source, ok, nil
}

func (s *memoryServerCatalogueStore) SaveSpeedtestServerCatalogue(
	ctx context.Context,
	servers []database.SpeedtestServer,
	source *database.SpeedtestServerSource,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastSaveDeadline, _ = ctx.Deadline()
	if s.saveErr != nil {
		return s.saveErr
	}
	if s.servers == nil {
		s.servers = make(map[string]database.SpeedtestServer)
	}
	if s.sources == nil {
		s.sources = make(map[string]database.SpeedtestServerSource)
	}
	for _, server := range servers {
		current, ok := s.servers[server.ID]
		if !ok || server.ObservedAt.After(current.ObservedAt) {
			s.servers[server.ID] = server
		}
	}
	if source != nil {
		current, ok := s.sources[source.Key]
		if !ok || source.UpdatedAt.After(current.UpdatedAt) {
			s.sources[source.Key] = *source
		}
	}
	return nil
}

func (s *memoryServerCatalogueStore) setError(err error) {
	s.mu.Lock()
	s.saveErr = err
	s.mu.Unlock()
}

func (s *memoryServerCatalogueStore) setSourceUpdatedAt(key string, updatedAt time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	source := s.sources[key]
	source.UpdatedAt = updatedAt
	s.sources[key] = source
}

func (s *memoryServerCatalogueStore) setDeadline() time.Time {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.lastSaveDeadline
}

func TestFetchSpeedtestServerListPreservesDiscoveryMetadataWithoutProbes(t *testing.T) {
	var requestCount atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		requestCount.Add(1)
		assert.Equal(t, http.MethodGet, request.Method)
		assert.Equal(t, st.DefaultUserAgent, request.Header.Get("User-Agent"))
		assert.Equal(t, "-27.4698", request.URL.Query().Get("lat"))
		assert.Equal(t, "153.0251", request.URL.Query().Get("lon"))
		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write([]byte(`[
			{"id":"2","name":"Sydney","sponsor":"Example Two","host":"unresolved.invalid:8080","url":"http://unresolved.invalid:8080/speedtest/upload.php","country":"Australia","lat":"-33.8688","lon":"151.2093","distance":10},
			{"id":"1","name":"Brisbane","sponsor":"Example One","host":"unresolved.invalid:8080","url":"http://unresolved.invalid:8080/speedtest/upload.php","country":"Australia","lat":"-27.4698","lon":"153.0251","distance":5}
		]`))
		assert.NoError(t, err)
	}))
	t.Cleanup(server.Close)

	servers, err := fetchSpeedtestServerList(t.Context(), server.URL, &ServerLocation{
		Latitude:  -27.4698,
		Longitude: 153.0251,
	})
	require.NoError(t, err)
	assert.Equal(t, int32(1), requestCount.Load())

	got := serverResponses(servers)
	require.Len(t, got, 2)
	assert.Equal(t, "1", got[1].ID)
	assert.Equal(t, "Brisbane", got[1].Name)
	assert.Equal(t, "Example One", got[1].Sponsor)
	assert.Equal(t, "unresolved.invalid:8080", got[1].Host)
	assert.Equal(t, "http://unresolved.invalid:8080/speedtest/upload.php", got[1].URL)
	assert.Equal(t, "Australia", got[1].Country)
	assert.InDelta(t, -27.4698, got[1].Lat, 1e-9)
	assert.InDelta(t, 153.0251, got[1].Lon, 1e-9)
}

func TestFetchSpeedtestServerListRejectsUnsuccessfulResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "try again later", http.StatusServiceUnavailable)
	}))
	t.Cleanup(server.Close)

	_, err := fetchSpeedtestServerList(t.Context(), server.URL, nil)
	require.ErrorContains(t, err, "status 503 Service Unavailable")
}

func TestSortServersFromOrigin(t *testing.T) {
	origin := ServerLocation{Latitude: 0, Longitude: 0}
	servers := []ServerResponse{
		{ID: "far", Lat: 0, Lon: 2},
		{ID: "near", Lat: 0, Lon: 1},
	}

	got := sortServersFromOrigin(origin, servers)
	require.Len(t, got, 2)
	assert.Equal(t, "near", got[0].ID)
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

func TestCatalogueSourceStoredTracksPersistedSource(t *testing.T) {
	store := &memoryServerCatalogueStore{}
	runner := NewSpeedtestNetRunner(config.SpeedTestConfig{}, store)

	stored, err := runner.catalogueSourceStored(t.Context(), "test")
	require.NoError(t, err)
	assert.False(t, stored)

	require.NoError(t, runner.updateServerCatalogue(t.Context(), nil, nil, "test", runner.nextCatalogueObservation()))
	stored, err = runner.catalogueSourceStored(t.Context(), "test")
	require.NoError(t, err)
	assert.True(t, stored)
}

func TestSourceViewLimitsNearestServersExceptGlobal(t *testing.T) {
	runner := NewSpeedtestNetRunner(config.SpeedTestConfig{}, &memoryServerCatalogueStore{})
	pool := make([]ServerResponse, 0, nearestServerLimit+2)
	for i := range nearestServerLimit + 2 {
		pool = append(pool, ServerResponse{ID: fmt.Sprint(i), Lat: float64(i), Lon: 0})
	}
	origin := &ServerLocation{Latitude: 0, Longitude: 0}
	require.NoError(t, runner.updateServerCatalogue(t.Context(), pool, origin, sourceKeyLocal, runner.nextCatalogueObservation()))
	require.NoError(t, runner.updateServerCatalogue(t.Context(), nil, nil, sourceKeyGlobal, runner.nextCatalogueObservation()))
	runner.fetchServers = func(context.Context, *ServerLocation) ([]ServerResponse, *ServerLocation, error) {
		return nil, nil, errors.New("unexpected fetch")
	}

	local, err := runner.GetServersWithOptions(t.Context(), ServerListOptions{})
	require.NoError(t, err)
	require.Len(t, local, nearestServerLimit)
	assert.Equal(t, "0", local[0].ID)

	global, err := runner.GetServersWithOptions(t.Context(), ServerListOptions{Global: true})
	require.NoError(t, err)
	assert.Len(t, global, nearestServerLimit+2)
}

func TestUnstoredSourceFetchesEvenWhenPoolIsNotEmpty(t *testing.T) {
	runner := NewSpeedtestNetRunner(config.SpeedTestConfig{}, &memoryServerCatalogueStore{})
	runner.globalLocations = map[string]ServerLocation{"regional": {Latitude: 1, Longitude: 1}}
	require.NoError(t, runner.updateServerCatalogue(t.Context(), []ServerResponse{{ID: "local"}}, &ServerLocation{}, sourceKeyLocal, runner.nextCatalogueObservation()))
	runner.fetchServers = func(_ context.Context, location *ServerLocation) ([]ServerResponse, *ServerLocation, error) {
		if location == nil {
			return []ServerResponse{{ID: "local"}}, &ServerLocation{}, nil
		}
		return []ServerResponse{{ID: "regional"}}, nil, nil
	}

	servers, err := runner.GetServersWithOptions(t.Context(), ServerListOptions{Global: true})
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"local", "regional"}, serverIDs(servers))
}

func TestCatalogueSourceRefreshRetainsValidEntry(t *testing.T) {
	store := &memoryServerCatalogueStore{}
	runner := NewSpeedtestNetRunner(config.SpeedTestConfig{}, store)
	cachedServers := []ServerResponse{{ID: "cached"}}
	require.NoError(t, runner.updateServerCatalogue(t.Context(), cachedServers, &ServerLocation{}, "local", runner.nextCatalogueObservation()))

	var fetchCount atomic.Int32
	runner.fetchServers = func(_ context.Context, _ *ServerLocation) ([]ServerResponse, *ServerLocation, error) {
		fetchCount.Add(1)
		return []ServerResponse{{ID: "fresh"}}, &ServerLocation{}, nil
	}

	servers, err := runner.GetServersWithOptions(t.Context(), ServerListOptions{Refresh: true})
	require.NoError(t, err)
	assert.ElementsMatch(t, []ServerResponse{{ID: "cached"}, {ID: "fresh"}}, servers)
	assert.Equal(t, int32(1), fetchCount.Load())
	stored, err := runner.catalogueSourceStored(t.Context(), "local")
	require.NoError(t, err)
	assert.True(t, stored)

	servers, err = runner.GetServersWithOptions(t.Context(), ServerListOptions{})
	require.NoError(t, err)
	assert.ElementsMatch(t, []ServerResponse{{ID: "cached"}, {ID: "fresh"}}, servers)
	assert.Equal(t, int32(1), fetchCount.Load())
}

func TestServerCatalogueRetainsServersAcrossSources(t *testing.T) {
	runner := NewSpeedtestNetRunner(config.SpeedTestConfig{}, &memoryServerCatalogueStore{})
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
		Refresh:  true,
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

func TestServerCataloguePersistenceFailureDoesNotBlockCommittedReads(t *testing.T) {
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
	store.setError(nil)
	servers, err = runner.GetServersWithOptions(t.Context(), ServerListOptions{})
	require.NoError(t, err)
	assert.Equal(t, []string{"durable"}, serverIDs(servers))

	status, err := runner.GetServerCatalogueStatus(t.Context(), ServerListOptions{})
	require.NoError(t, err)
	assert.True(t, status.Stored)
	stored, err := runner.loadRetainedServers(t.Context(), nil, 0)
	require.NoError(t, err)
	assert.Equal(t, []string{"durable"}, serverIDs(stored))
}

func TestServerCatalogueSurvivesSourceCacheExpiry(t *testing.T) {
	store := &memoryServerCatalogueStore{}
	runner := NewSpeedtestNetRunner(config.SpeedTestConfig{}, store)
	oldServers := []ServerResponse{{ID: "old"}}
	require.NoError(t, runner.updateServerCatalogue(t.Context(), oldServers, nil, "old", runner.nextCatalogueObservation()))
	store.setSourceUpdatedAt("old", time.Now().Add(-time.Hour))

	newServers := []ServerResponse{{ID: "new"}}
	require.NoError(t, runner.updateServerCatalogue(t.Context(), newServers, nil, "new", runner.nextCatalogueObservation()))

	servers, err := runner.loadRetainedServers(t.Context(), nil, 0)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"old", "new"}, serverIDs(servers))
}

func TestServerCacheCoalescesConcurrentMissesByKey(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		runner := NewSpeedtestNetRunner(config.SpeedTestConfig{}, &memoryServerCatalogueStore{})
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
		runner := NewSpeedtestNetRunner(config.SpeedTestConfig{}, &memoryServerCatalogueStore{})
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
		runner := NewSpeedtestNetRunner(config.SpeedTestConfig{}, &memoryServerCatalogueStore{})
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
	runner := NewSpeedtestNetRunner(config.SpeedTestConfig{}, &memoryServerCatalogueStore{})
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

	servers, err := runner.getGlobalServers(t.Context(), false)
	require.Error(t, err)
	assert.Nil(t, servers)
	stored, storedErr := runner.catalogueSourceStored(t.Context(), "global")
	require.NoError(t, storedErr)
	assert.False(t, stored)
}

func TestGlobalServersReportsPartialFailureAndRetainsSuccessfulResults(t *testing.T) {
	runner := NewSpeedtestNetRunner(config.SpeedTestConfig{}, &memoryServerCatalogueStore{})
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

	servers, err := runner.getGlobalServers(t.Context(), false)
	require.ErrorContains(t, err, "updated from 1 of 2 regional locations")
	require.ErrorContains(t, err, "fetch servers near failure")
	partialErr, ok := errors.AsType[*PartialServerCatalogueError](err)
	require.True(t, ok)
	assert.Equal(t, []string{"fetch servers near failure: regional fetch failed"}, partialErr.WarningMessages())
	assert.Len(t, servers, 2)
	stored, storedErr := runner.catalogueSourceStored(t.Context(), "global")
	require.NoError(t, storedErr)
	assert.False(t, stored)
	status, statusErr := runner.GetServerCatalogueStatus(t.Context(), ServerListOptions{Global: true})
	require.NoError(t, statusErr)
	assert.False(t, status.Stored)
}

func TestGlobalServersUsesStoredLocationWhenLocalRefreshFails(t *testing.T) {
	store := &memoryServerCatalogueStore{}
	runner := NewSpeedtestNetRunner(config.SpeedTestConfig{}, store)
	runner.globalLocations = map[string]ServerLocation{
		"regional": {Latitude: 1, Longitude: 1},
	}
	require.NoError(t, runner.updateServerCatalogue(
		t.Context(),
		[]ServerResponse{{ID: "cached-local"}},
		&ServerLocation{Latitude: -27.4698, Longitude: 153.0251},
		"local",
		runner.nextCatalogueObservation(),
	))

	runner.fetchServers = func(_ context.Context, location *ServerLocation) ([]ServerResponse, *ServerLocation, error) {
		if location == nil {
			return nil, nil, errors.New("local discovery failed")
		}
		return []ServerResponse{{ID: "fresh-regional", Lat: location.Latitude, Lon: location.Longitude}}, nil, nil
	}

	servers, err := runner.GetServersWithOptions(t.Context(), ServerListOptions{Global: true, Refresh: true})
	require.ErrorContains(t, err, "fetch local speedtest servers: local discovery failed")
	partialErr, ok := errors.AsType[*PartialServerCatalogueError](err)
	require.True(t, ok)
	assert.Equal(t, []string{"fetch local speedtest servers: local discovery failed"}, partialErr.WarningMessages())
	assert.ElementsMatch(t, []string{"cached-local", "fresh-regional"}, serverIDs(servers))
	status, statusErr := runner.GetServerCatalogueStatus(t.Context(), ServerListOptions{Global: true})
	require.NoError(t, statusErr)
	assert.True(t, status.Stored)
}

func TestGlobalServersRefreshRetainsPreviousGlobalAndLocalServers(t *testing.T) {
	runner := NewSpeedtestNetRunner(config.SpeedTestConfig{}, &memoryServerCatalogueStore{})
	runner.globalLocations = map[string]ServerLocation{
		"regional": {Latitude: 1, Longitude: 1},
	}
	cachedGlobal := []ServerResponse{{ID: "cached-global"}}
	cachedLocal := []ServerResponse{{ID: "cached-local"}}
	require.NoError(t, runner.updateServerCatalogue(t.Context(), cachedGlobal, nil, "global", runner.nextCatalogueObservation()))
	require.NoError(t, runner.updateServerCatalogue(t.Context(), cachedLocal, &ServerLocation{}, "local", runner.nextCatalogueObservation()))

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

	stored, storedErr := runner.catalogueSourceStored(t.Context(), "global")
	require.NoError(t, storedErr)
	assert.True(t, stored)
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
		persisted, err := restarted.loadRetainedServers(t.Context(), nil, 0)
		require.NoError(t, err)
		for _, server := range persisted {
			if server.ID == "shared" {
				assert.Equal(t, "coordinate-new", server.Sponsor)
				return
			}
		}
		t.Fatal("persisted shared server not found")
	})
}

func TestGlobalServersRefreshPreservesCacheOnPartialFailure(t *testing.T) {
	runner := NewSpeedtestNetRunner(config.SpeedTestConfig{}, &memoryServerCatalogueStore{})
	runner.globalLocations = map[string]ServerLocation{
		"success": {Latitude: 1, Longitude: 1},
		"failure": {Latitude: 2, Longitude: 2},
	}
	previous := []ServerResponse{{ID: "cached-global"}}
	require.NoError(t, runner.updateServerCatalogue(t.Context(), previous, nil, "global", runner.nextCatalogueObservation()))
	statusBefore, err := runner.GetServerCatalogueStatus(t.Context(), ServerListOptions{Global: true})
	require.NoError(t, err)
	require.NotNil(t, statusBefore.UpdatedAt)
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
	require.ErrorContains(t, err, "updated from 1 of 2 regional locations")
	assert.ElementsMatch(t, []string{"cached-global", "fresh-local", "fresh-regional"}, serverIDs(servers))

	stored, storedErr := runner.catalogueSourceStored(t.Context(), "global")
	require.NoError(t, storedErr)
	assert.True(t, stored)
	retained, retainedErr := runner.loadRetainedServers(t.Context(), nil, 0)
	require.NoError(t, retainedErr)
	assert.ElementsMatch(t, []string{"cached-global", "fresh-local", "fresh-regional"}, serverIDs(retained))
	statusAfter, statusErr := runner.GetServerCatalogueStatus(t.Context(), ServerListOptions{Global: true})
	require.NoError(t, statusErr)
	assert.Equal(t, statusBefore.UpdatedAt, statusAfter.UpdatedAt)
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
	retained, retainedErr := restarted.loadRetainedServers(t.Context(), nil, 0)
	require.NoError(t, retainedErr)
	assert.ElementsMatch(t, []string{"local", "regional"}, serverIDs(retained))
}

func TestGlobalServersWaitHonorsContext(t *testing.T) {
	runner := NewSpeedtestNetRunner(config.SpeedTestConfig{}, &memoryServerCatalogueStore{})
	runner.fetchServers = func(context.Context, *ServerLocation) ([]ServerResponse, *ServerLocation, error) {
		return nil, nil, errors.New("unexpected fetch")
	}
	runner.globalFetch <- struct{}{}
	defer func() { <-runner.globalFetch }()

	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Millisecond)
	defer cancel()
	_, err := runner.getGlobalServers(ctx, false)
	require.ErrorIs(t, err, context.DeadlineExceeded)
}
