// Copyright (c) 2024-2025, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package database

import (
	"context"
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
		jitter := 2.5

		speedTest := types.SpeedTestResult{
			ServerName:    "Test Server",
			ServerID:      "test-123",
			ServerHost:    &serverHost,
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
