// Copyright (c) 2024-2026, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package database

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/autobrr/netronome/internal/config"
	"github.com/autobrr/netronome/internal/types"
)

func TestSpeedTest_Save(t *testing.T) {
	RunTestWithBothDatabases(t, func(t *testing.T, td *TestDatabase) {
		ctx := context.Background()

		serverHost := "speedtest.example.com"
		serverCity := "Brisbane"
		jitter := 2.5

		speedTest := types.SpeedTestResult{
			ServerName:    "Test Server",
			ServerID:      "test-123",
			ServerHost:    &serverHost,
			ServerCity:    &serverCity,
			TestType:      "iperf3",
			DownloadSpeed: 100.5,
			UploadSpeed:   50.25,
			Latency:       "10.0ms",
			Jitter:        &jitter,
			IsScheduled:   false,
		}

		// Save speed test
		saved, err := td.Service.SaveSpeedTest(ctx, speedTest)
		require.NoError(t, err)
		require.NotNil(t, saved)
		assert.Greater(t, saved.ID, int64(0))
		assert.Equal(t, speedTest.ServerName, saved.ServerName)
		assert.Equal(t, speedTest.DownloadSpeed, saved.DownloadSpeed)

		// Verify saved in database
		AssertRecordExists(t, td, "speed_tests", "id", saved.ID)

		// Query back to verify created_at was set
		results, err := td.Service.GetSpeedTests(ctx, "all", 1, 10)
		require.NoError(t, err)
		require.Len(t, results.Data, 1)
		assert.NotZero(t, results.Data[0].CreatedAt)
		require.NotNil(t, results.Data[0].ServerCity)
		assert.Equal(t, serverCity, *results.Data[0].ServerCity)
	})
}

func TestSpeedtestServerCataloguePersistsNewestMetadataAndSourceState(t *testing.T) {
	RunTestWithBothDatabases(t, func(t *testing.T, td *TestDatabase) {
		older := time.Date(2026, time.September, 21, 0, 0, 0, 0, time.UTC)
		newer := older.Add(time.Minute)
		latitude := -27.4698
		longitude := 153.0251

		require.NoError(t, td.Service.SaveSpeedtestServerCatalogue(t.Context(), []SpeedtestServer{{
			ID:         "123",
			Name:       "Brisbane",
			Host:       "speed.example:8080",
			Country:    "Australia",
			Sponsor:    "new sponsor",
			URL:        "https://speed.example/upload",
			Latitude:   latitude,
			Longitude:  longitude,
			ObservedAt: newer,
		}}, &SpeedtestServerSource{
			Key:       "local",
			UpdatedAt: newer,
			Latitude:  &latitude,
			Longitude: &longitude,
		}))

		require.NoError(t, td.Service.SaveSpeedtestServerCatalogue(t.Context(), []SpeedtestServer{{
			ID:         "123",
			Name:       "Brisbane",
			Host:       "stale.example:8080",
			Country:    "Australia",
			Sponsor:    "stale sponsor",
			URL:        "https://stale.example/upload",
			Latitude:   latitude,
			Longitude:  longitude,
			ObservedAt: older,
		}}, &SpeedtestServerSource{
			Key:       "local",
			UpdatedAt: older,
		}))

		servers, err := td.Service.ListSpeedtestServers(t.Context())
		require.NoError(t, err)
		require.Len(t, servers, 1)
		assert.Equal(t, "new sponsor", servers[0].Sponsor)
		assert.Equal(t, "speed.example:8080", servers[0].Host)
		assert.Equal(t, newer, servers[0].ObservedAt)

		source, found, err := td.Service.GetSpeedtestServerSource(t.Context(), "local")
		require.NoError(t, err)
		require.True(t, found)
		assert.Equal(t, newer, source.UpdatedAt)
		require.NotNil(t, source.Latitude)
		require.NotNil(t, source.Longitude)
		assert.InDelta(t, latitude, *source.Latitude, 1e-9)
		assert.InDelta(t, longitude, *source.Longitude, 1e-9)
	})
}

func TestSpeedtestServerCatalogueRollsBackFailedSourceUpdate(t *testing.T) {
	RunTestWithBothDatabases(t, func(t *testing.T, td *TestDatabase) {
		observedAt := time.Date(2026, time.September, 21, 0, 0, 0, 0, time.UTC)
		latitude := -27.4698
		err := td.Service.SaveSpeedtestServerCatalogue(t.Context(), []SpeedtestServer{{
			ID:         "rollback",
			ObservedAt: observedAt,
		}}, &SpeedtestServerSource{
			Key:       "local",
			UpdatedAt: observedAt,
			Latitude:  &latitude,
		})
		require.Error(t, err)

		servers, listErr := td.Service.ListSpeedtestServers(t.Context())
		require.NoError(t, listErr)
		assert.Empty(t, servers)
		_, found, sourceErr := td.Service.GetSpeedtestServerSource(t.Context(), "local")
		require.NoError(t, sourceErr)
		assert.False(t, found)
	})
}

func TestSpeedtestServerCatalogueBoundsCoordinateSourceState(t *testing.T) {
	RunTestWithBothDatabases(t, func(t *testing.T, td *TestDatabase) {
		baseTime := time.Date(2026, time.September, 21, 0, 0, 0, 0, time.UTC)
		require.NoError(t, td.Service.SaveSpeedtestServerCatalogue(t.Context(), nil, &SpeedtestServerSource{
			Key:       "local",
			UpdatedAt: baseTime,
		}))
		for i := range 40 {
			require.NoError(t, td.Service.SaveSpeedtestServerCatalogue(t.Context(), nil, &SpeedtestServerSource{
				Key:       fmt.Sprintf("location:%d,%d", i, i),
				UpdatedAt: baseTime.Add(time.Duration(i+1) * time.Minute),
			}))
		}

		var coordinateSources int
		err := td.DB.QueryRowContext(t.Context(), `SELECT COUNT(*) FROM speedtest_server_sources WHERE source_key LIKE 'location:%'`).
			Scan(&coordinateSources)
		require.NoError(t, err)
		assert.Equal(t, maxSpeedtestCoordinateSources, coordinateSources)
		_, oldestFound, err := td.Service.GetSpeedtestServerSource(t.Context(), "location:0,0")
		require.NoError(t, err)
		assert.False(t, oldestFound)
		_, newestFound, err := td.Service.GetSpeedtestServerSource(t.Context(), "location:39,39")
		require.NoError(t, err)
		assert.True(t, newestFound)
		currentKey := "location:-27.4698,153.0251"
		require.NoError(t, td.Service.SaveSpeedtestServerCatalogue(t.Context(), nil, &SpeedtestServerSource{
			Key:       currentKey,
			UpdatedAt: baseTime.Add(-time.Hour),
		}))
		_, currentFound, err := td.Service.GetSpeedtestServerSource(t.Context(), currentKey)
		require.NoError(t, err)
		assert.True(t, currentFound, "the source just fetched must survive pruning despite future timestamps")
		err = td.DB.QueryRowContext(t.Context(), `SELECT COUNT(*) FROM speedtest_server_sources WHERE source_key LIKE 'location:%'`).
			Scan(&coordinateSources)
		require.NoError(t, err)
		assert.Equal(t, maxSpeedtestCoordinateSources, coordinateSources)
		_, localFound, err := td.Service.GetSpeedtestServerSource(t.Context(), "local")
		require.NoError(t, err)
		assert.True(t, localFound)
	})
}

func TestSpeedtestServerCatalogueSerializesConcurrentCoordinatePruning(t *testing.T) {
	if os.Getenv("SKIP_POSTGRES_TESTS") != "" {
		t.Skip("PostgreSQL tests skipped (SKIP_POSTGRES_TESTS is set)")
	}

	td := SetupTestDatabase(t, config.Postgres)
	defer td.Close()
	baseTime := time.Date(2026, time.September, 21, 0, 0, 0, 0, time.UTC)
	for i := range maxSpeedtestCoordinateSources - 1 {
		require.NoError(t, td.Service.SaveSpeedtestServerCatalogue(t.Context(), nil, &SpeedtestServerSource{
			Key:       fmt.Sprintf("location:old-%d", i),
			UpdatedAt: baseTime.Add(time.Duration(i) * time.Minute),
		}))
	}

	const concurrentWrites = 16
	start := make(chan struct{})
	errs := make(chan error, concurrentWrites)
	var writes sync.WaitGroup
	for i := range concurrentWrites {
		writes.Go(func() {
			<-start
			observedAt := baseTime.Add(time.Duration(maxSpeedtestCoordinateSources+i) * time.Minute)
			servers := []SpeedtestServer{
				{ID: "shared-a", ObservedAt: observedAt},
				{ID: "shared-b", ObservedAt: observedAt},
			}
			if i%2 != 0 {
				servers[0], servers[1] = servers[1], servers[0]
			}
			errs <- td.Service.SaveSpeedtestServerCatalogue(t.Context(), servers, &SpeedtestServerSource{
				Key:       fmt.Sprintf("location:new-%d", i),
				UpdatedAt: observedAt,
			})
		})
	}
	close(start)
	writes.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}

	var coordinateSources int
	require.NoError(t, td.DB.QueryRowContext(
		t.Context(),
		`SELECT COUNT(*) FROM speedtest_server_sources WHERE source_key LIKE 'location:%'`,
	).Scan(&coordinateSources))
	assert.Equal(t, maxSpeedtestCoordinateSources, coordinateSources)
	for i := range concurrentWrites {
		_, found, err := td.Service.GetSpeedtestServerSource(t.Context(), fmt.Sprintf("location:new-%d", i))
		require.NoError(t, err)
		assert.True(t, found)
	}
}

func TestSpeedTest_GetAliasesOnlyUnambiguousLegacyServerIdentities(t *testing.T) {
	RunTestWithBothDatabases(t, func(t *testing.T, td *TestDatabase) {
		baseTime := time.Date(2026, time.September, 20, 0, 0, 0, 0, time.UTC)
		results := []types.SpeedTestResult{
			{ServerName: "custom-42", ServerID: "librespeed-custom-42", ServerHost: new("https://first.example.com"), TestType: "librespeed", CreatedAt: baseTime.Add(-2 * time.Minute)},
			{ServerName: "custom-42", ServerID: "librespeed-custom-43", ServerHost: new("https://second.example.com"), TestType: "librespeed", CreatedAt: baseTime.Add(-time.Minute)},
			{ServerName: "Unique", ServerID: "101", ServerHost: new("unique.example.com"), TestType: "speedtest", CreatedAt: baseTime.Add(time.Minute)},
			{ServerName: "Unique", ServerID: "Unique", TestType: "speedtest", CreatedAt: baseTime.Add(2 * time.Minute)},
			{ServerName: "Shared", ServerID: "201", ServerHost: new("first.example.com"), TestType: "speedtest", CreatedAt: baseTime.Add(3 * time.Minute)},
			{ServerName: "Shared", ServerID: "202", ServerHost: new("second.example.com"), TestType: "speedtest", CreatedAt: baseTime.Add(4 * time.Minute)},
			{ServerName: "Libre", ServerID: "librespeed-public-42", ServerHost: new("https://libre.example.com"), TestType: "librespeed", CreatedAt: baseTime.Add(5 * time.Minute)},
			{ServerName: "Libre", ServerID: "librespeed-Libre", ServerHost: new("Libre"), TestType: "librespeed", CreatedAt: baseTime.Add(6 * time.Minute)},
			{ServerName: "Shared", ServerID: "Shared", TestType: "speedtest", CreatedAt: baseTime.Add(7 * time.Minute)},
		}
		ids := make(map[string]int64, len(results))
		for _, result := range results {
			saved, err := td.Service.SaveSpeedTest(t.Context(), result)
			require.NoError(t, err)
			ids[result.TestType+":"+result.ServerID] = saved.ID
		}

		page, err := td.Service.GetSpeedTests(t.Context(), "all", 1, 1)
		require.NoError(t, err)
		require.Len(t, page.Data, 1)
		assert.Equal(t, "Shared", page.Data[0].ServerID, "aliases must use identities outside the requested page")

		all, err := td.Service.GetSpeedTests(t.Context(), "all", 1, len(results))
		require.NoError(t, err)
		byID := make(map[int64]types.SpeedTestResult, len(all.Data))
		for _, result := range all.Data {
			byID[result.ID] = result
		}
		assert.Equal(t, "101", byID[ids["speedtest:Unique"]].ServerID)
		assert.Equal(t, "Shared", byID[ids["speedtest:Shared"]].ServerID)
		assert.Equal(t, "librespeed-public-42", byID[ids["librespeed:librespeed-Libre"]].ServerID)
		assert.Equal(t, "librespeed-custom-42", byID[ids["librespeed:librespeed-custom-42"]].ServerID)
		assert.Equal(t, "librespeed-custom-43", byID[ids["librespeed:librespeed-custom-43"]].ServerID)
	})
}

func TestSpeedTest_GetWithPagination(t *testing.T) {
	RunTestWithBothDatabases(t, func(t *testing.T, td *TestDatabase) {
		ctx := context.Background()

		// Create multiple speed tests
		baseTime := time.Now()
		for i := 0; i < 25; i++ {
			speedTest := types.SpeedTestResult{
				ServerName:    "Server " + string(rune('A'+i)),
				ServerID:      "srv-" + string(rune('0'+i)),
				ServerHost:    stringPtr("host" + string(rune('0'+i)) + ".example.com"),
				TestType:      "speedtest",
				DownloadSpeed: float64(100 + i),
				UploadSpeed:   float64(50 + i),
				Latency:       string(rune('0'+i)) + "0ms",
				IsScheduled:   i%2 == 0,
				CreatedAt:     baseTime.Add(time.Duration(-i) * time.Hour),
			}
			_, err := td.Service.SaveSpeedTest(ctx, speedTest)
			require.NoError(t, err)
		}

		// Test pagination - first page
		page1, err := td.Service.GetSpeedTests(ctx, "all", 1, 10)
		require.NoError(t, err)
		assert.Len(t, page1.Data, 10)
		assert.Equal(t, 25, page1.Total)
		// Calculate total pages
		totalPages := (page1.Total + page1.Limit - 1) / page1.Limit
		assert.Equal(t, 3, totalPages)
		assert.Equal(t, 1, page1.Page)

		// Test pagination - second page
		page2, err := td.Service.GetSpeedTests(ctx, "all", 2, 10)
		require.NoError(t, err)
		assert.Len(t, page2.Data, 10)
		assert.Equal(t, 2, page2.Page)

		// Test pagination - last page
		page3, err := td.Service.GetSpeedTests(ctx, "all", 3, 10)
		require.NoError(t, err)
		assert.Len(t, page3.Data, 5)
		assert.Equal(t, 3, page3.Page)

		// Verify ordering (newest first)
		for i := 1; i < len(page1.Data); i++ {
			assert.True(t,
				page1.Data[i-1].CreatedAt.After(page1.Data[i].CreatedAt) ||
					page1.Data[i-1].CreatedAt.Equal(page1.Data[i].CreatedAt),
			)
		}
	})
}

func TestSpeedTest_TimeRangeFilters(t *testing.T) {
	RunTestWithBothDatabases(t, func(t *testing.T, td *TestDatabase) {
		ctx := context.Background()

		// Create speed tests at different times
		now := time.Now()
		times := []time.Duration{
			-1 * time.Hour,       // 1 hour ago
			-36 * time.Hour,      // 1.5 days ago (clearly outside 24h)
			-8 * 24 * time.Hour,  // Last week (8 days ago)
			-35 * 24 * time.Hour, // Last month (35 days ago)
		}

		for i, duration := range times {
			createdAt := now.Add(duration)
			speedTest := types.SpeedTestResult{
				ServerName:    "Server" + string(rune('0'+i)),
				ServerID:      "id" + string(rune('0'+i)),
				TestType:      "iperf3",
				DownloadSpeed: 100.0,
				UploadSpeed:   50.0,
				CreatedAt:     createdAt,
			}
			_, err := td.Service.SaveSpeedTest(ctx, speedTest)
			require.NoError(t, err)
		}

		// Test "24h" filter
		results24h, err := td.Service.GetSpeedTests(ctx, "24h", 1, 100)
		require.NoError(t, err)

		assert.Equal(t, 1, results24h.Total) // Only the 1 hour ago test

		// Test "week" filter
		resultsWeek, err := td.Service.GetSpeedTests(ctx, "week", 1, 100)
		require.NoError(t, err)
		assert.Equal(t, 2, resultsWeek.Total) // 1 hour and 1.5 days ago

		// Test "month" filter
		resultsMonth, err := td.Service.GetSpeedTests(ctx, "month", 1, 100)
		require.NoError(t, err)
		assert.Equal(t, 3, resultsMonth.Total) // All except the oldest

		// Test "all" filter
		resultsAll, err := td.Service.GetSpeedTests(ctx, "all", 1, 100)
		require.NoError(t, err)
		assert.Equal(t, 4, resultsAll.Total) // All tests
	})
}

func TestSpeedTest_DifferentTestTypes(t *testing.T) {
	RunTestWithBothDatabases(t, func(t *testing.T, td *TestDatabase) {
		ctx := context.Background()

		testTypes := []string{"iperf3", "speedtest", "librespeed"}

		for _, testType := range testTypes {
			speedTest := types.SpeedTestResult{
				ServerName:    testType + " Server",
				ServerID:      testType + "-123",
				ServerHost:    stringPtr(testType + ".example.com"),
				TestType:      testType,
				DownloadSpeed: 100.0,
				UploadSpeed:   50.0,
				Latency:       "15.0ms",
			}

			saved, err := td.Service.SaveSpeedTest(ctx, speedTest)
			require.NoError(t, err)
			assert.Equal(t, testType, saved.TestType)
		}

		// Verify all test types were saved
		results, err := td.Service.GetSpeedTests(ctx, "all", 1, 100)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, results.Total, len(testTypes))

		// Check that all test types are present
		foundTypes := make(map[string]bool)
		for _, test := range results.Data {
			foundTypes[test.TestType] = true
		}

		for _, testType := range testTypes {
			assert.True(t, foundTypes[testType], "Test type %s should be present", testType)
		}
	})
}

func TestSpeedTest_ScheduledVsManual(t *testing.T) {
	RunTestWithBothDatabases(t, func(t *testing.T, td *TestDatabase) {
		ctx := context.Background()

		// Create scheduled test
		scheduledTest := types.SpeedTestResult{
			ServerName:    "Scheduled Server",
			ServerID:      "sched-123",
			TestType:      "iperf3",
			DownloadSpeed: 150.0,
			UploadSpeed:   75.0,
			IsScheduled:   true,
		}

		savedScheduled, err := td.Service.SaveSpeedTest(ctx, scheduledTest)
		require.NoError(t, err)
		assert.True(t, savedScheduled.IsScheduled)

		// Create manual test
		manualTest := types.SpeedTestResult{
			ServerName:    "Manual Server",
			ServerID:      "manual-123",
			TestType:      "speedtest",
			DownloadSpeed: 200.0,
			UploadSpeed:   100.0,
			IsScheduled:   false,
		}

		savedManual, err := td.Service.SaveSpeedTest(ctx, manualTest)
		require.NoError(t, err)
		assert.False(t, savedManual.IsScheduled)

		// Verify both are saved correctly
		var isScheduled bool

		// Build query based on database type
		var query string
		if td.Config.Type == config.Postgres {
			query = "SELECT is_scheduled FROM speed_tests WHERE id = $1"
		} else {
			query = "SELECT is_scheduled FROM speed_tests WHERE id = ?"
		}

		err = td.DB.QueryRow(query, savedScheduled.ID).Scan(&isScheduled)
		require.NoError(t, err)
		assert.True(t, isScheduled)

		err = td.DB.QueryRow(query, savedManual.ID).Scan(&isScheduled)
		require.NoError(t, err)
		assert.False(t, isScheduled)
	})
}

func TestSpeedTest_NullableFields(t *testing.T) {
	RunTestWithBothDatabases(t, func(t *testing.T, td *TestDatabase) {
		ctx := context.Background()

		// Create test with minimal fields (some fields might be nullable)
		minimalTest := types.SpeedTestResult{
			ServerName:    "Minimal Server",
			ServerID:      "min-123",
			TestType:      "speedtest",
			DownloadSpeed: 50.0,
			UploadSpeed:   25.0,
			// Latency and Jitter might be 0/null
		}

		saved, err := td.Service.SaveSpeedTest(ctx, minimalTest)
		require.NoError(t, err)

		// Retrieve and verify
		results, err := td.Service.GetSpeedTests(ctx, "all", 1, 10)
		require.NoError(t, err)
		require.Greater(t, len(results.Data), 0)

		// Find our test
		var found bool
		for _, test := range results.Data {
			if test.ID == saved.ID {
				found = true
				assert.Equal(t, minimalTest.ServerName, test.ServerName)
				assert.Equal(t, minimalTest.DownloadSpeed, test.DownloadSpeed)
				// Latency should be empty string if not set
				// Jitter could be nil or >= 0
				if test.Jitter != nil {
					assert.GreaterOrEqual(t, *test.Jitter, 0.0)
				}
				break
			}
		}
		assert.True(t, found, "Should find the saved test")
	})
}
