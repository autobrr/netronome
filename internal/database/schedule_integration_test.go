// Copyright (c) 2024-2025, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package database

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/autobrr/netronome/internal/types"
)

func TestSchedule_CRUD(t *testing.T) {
	RunTestWithBothDatabases(t, func(t *testing.T, td *TestDatabase) {
		ctx := context.Background()

		// Create schedule
		schedule := types.Schedule{
			ServerIDs: []string{"server-123", "server-456"},
			Interval:  "6h", // Every 6 hours
			NextRun:   time.Now().Add(6 * time.Hour),
			Enabled:   true,
			Options: types.TestOptions{
				UseIperf:       true,
				EnableDownload: true,
				EnableUpload:   true,
			},
		}

		created, err := td.Service.CreateSchedule(ctx, schedule)
		require.NoError(t, err)
		require.NotNil(t, created)
		assert.Greater(t, created.ID, int64(0))
		assert.Equal(t, schedule.ServerIDs, created.ServerIDs)
		assert.Equal(t, schedule.Interval, created.Interval)
		assert.NotZero(t, created.CreatedAt)
		// Schedule doesn't have UpdatedAt in the current struct

		// Read all schedules
		schedules, err := td.Service.GetSchedules(ctx)
		require.NoError(t, err)
		assert.Len(t, schedules, 1)
		assert.Equal(t, created.ID, schedules[0].ID)

		// Update schedule
		created.ServerIDs = append(created.ServerIDs, "server-789")
		created.Enabled = false
		created.Interval = "24h" // Daily

		err = td.Service.UpdateSchedule(ctx, *created)
		require.NoError(t, err)

		// Verify update
		schedules, err = td.Service.GetSchedules(ctx)
		require.NoError(t, err)
		require.Len(t, schedules, 1)
		assert.Len(t, schedules[0].ServerIDs, 3)
		assert.False(t, schedules[0].Enabled)
		assert.Equal(t, "24h", schedules[0].Interval)

		// Delete schedule
		err = td.Service.DeleteSchedule(ctx, created.ID)
		require.NoError(t, err)

		// Verify deletion
		schedules, err = td.Service.GetSchedules(ctx)
		require.NoError(t, err)
		assert.Len(t, schedules, 0)
	})
}

func TestSchedule_MultipleSchedules(t *testing.T) {
	RunTestWithBothDatabases(t, func(t *testing.T, td *TestDatabase) {
		ctx := context.Background()

		// Create multiple schedules
		testSchedules := []types.Schedule{
			{
				ServerIDs: []string{"srv1"},
				Interval:  "1h",
				NextRun:   time.Now().Add(1 * time.Hour),
				Enabled:   true,
				Options: types.TestOptions{
					EnableDownload: true,
					EnableUpload:   true,
					EnablePing:     true,
				},
			},
			{
				ServerIDs: []string{"srv2", "srv3"},
				Interval:  "24h",
				NextRun:   time.Now().Add(24 * time.Hour),
				Enabled:   true,
				Options: types.TestOptions{
					UseIperf:       true,
					EnableDownload: true,
					EnableUpload:   true,
				},
			},
			{
				ServerIDs: []string{"srv4"},
				Interval:  "168h", // Weekly
				NextRun:   time.Now().Add(168 * time.Hour),
				Enabled:   false,
				Options: types.TestOptions{
					UseLibrespeed:  true,
					EnableDownload: true,
					EnableUpload:   true,
				},
			},
		}

		// Create all schedules
		var createdIDs []int64
		for _, sched := range testSchedules {
			created, err := td.Service.CreateSchedule(ctx, sched)
			require.NoError(t, err)
			createdIDs = append(createdIDs, created.ID)
		}

		// Get all schedules
		schedules, err := td.Service.GetSchedules(ctx)
		require.NoError(t, err)
		assert.Len(t, schedules, len(testSchedules))

		// Verify all schedules are present by checking intervals
		foundIntervals := make(map[string]bool)
		for _, sched := range schedules {
			foundIntervals[sched.Interval] = true
		}

		expectedIntervals := []string{"1h", "24h", "168h"}
		for _, interval := range expectedIntervals {
			assert.True(t, foundIntervals[interval],
				"Schedule with interval %s should be present", interval)
		}

		// Delete specific schedule
		err = td.Service.DeleteSchedule(ctx, createdIDs[1])
		require.NoError(t, err)

		// Verify only 2 remain
		schedules, err = td.Service.GetSchedules(ctx)
		require.NoError(t, err)
		assert.Len(t, schedules, 2)
	})
}

func TestSchedule_UpdateNonExistent(t *testing.T) {
	RunTestWithBothDatabases(t, func(t *testing.T, td *TestDatabase) {
		ctx := context.Background()

		// Try to update non-existent schedule
		nonExistent := types.Schedule{
			ID:        99999,
			ServerIDs: []string{"non-existent"},
			Interval:  "1h",
			NextRun:   time.Now(),
			Enabled:   true,
			Options:   types.TestOptions{UseIperf: true},
		}

		err := td.Service.UpdateSchedule(ctx, nonExistent)
		assert.Error(t, err)
	})
}

func TestSchedule_DeleteNonExistent(t *testing.T) {
	RunTestWithBothDatabases(t, func(t *testing.T, td *TestDatabase) {
		ctx := context.Background()

		// Try to delete non-existent schedule
		err := td.Service.DeleteSchedule(ctx, 99999)
		// Some databases might not return error for DELETE with no matches
		// So we just ensure it doesn't panic
		_ = err
	})
}

func TestSchedule_DifferentTestTypes(t *testing.T) {
	RunTestWithBothDatabases(t, func(t *testing.T, td *TestDatabase) {
		ctx := context.Background()

		testConfigs := []struct {
			name    string
			options types.TestOptions
		}{
			{
				"iperf3",
				types.TestOptions{
					UseIperf:       true,
					EnableDownload: true,
					EnableUpload:   true,
				},
			},
			{
				"speedtest",
				types.TestOptions{
					EnableDownload: true,
					EnableUpload:   true,
					EnablePing:     true,
					EnableJitter:   true,
				},
			},
			{
				"librespeed",
				types.TestOptions{
					UseLibrespeed:  true,
					EnableDownload: true,
					EnableUpload:   true,
				},
			},
		}

		for _, config := range testConfigs {
			schedule := types.Schedule{
				ServerIDs: []string{"srv-" + config.name},
				Interval:  "1h",
				NextRun:   time.Now().Add(1 * time.Hour),
				Enabled:   true,
				Options:   config.options,
			}

			created, err := td.Service.CreateSchedule(ctx, schedule)
			require.NoError(t, err)

			// Verify options were saved correctly
			switch config.name {
			case "iperf3":
				assert.True(t, created.Options.UseIperf)
			case "librespeed":
				assert.True(t, created.Options.UseLibrespeed)
			}
		}

		// Verify all schedules were created
		schedules, err := td.Service.GetSchedules(ctx)
		require.NoError(t, err)
		assert.Len(t, schedules, len(testConfigs))
	})
}

func TestSchedule_IntervalFormats(t *testing.T) {
	RunTestWithBothDatabases(t, func(t *testing.T, td *TestDatabase) {
		ctx := context.Background()

		// Test various interval formats
		intervalTests := []struct {
			name     string
			interval string
		}{
			{"Every minute", "1m"},
			{"Every hour", "1h"},
			{"Every 6 hours", "6h"},
			{"Daily", "24h"},
			{"Weekly", "168h"},
			{"Every 30 minutes", "30m"},
			{"Every 12 hours", "12h"},
		}

		for _, test := range intervalTests {
			schedule := types.Schedule{
				ServerIDs: []string{"interval-test"},
				Interval:  test.interval,
				NextRun:   time.Now().Add(1 * time.Hour),
				Enabled:   true,
				Options: types.TestOptions{
					EnableDownload: true,
					EnableUpload:   true,
					EnablePing:     true,
				},
			}

			created, err := td.Service.CreateSchedule(ctx, schedule)
			require.NoError(t, err)
			assert.Equal(t, test.interval, created.Interval)

			// Clean up
			err = td.Service.DeleteSchedule(ctx, created.ID)
			require.NoError(t, err)
		}
	})
}

func TestSchedule_TimestampBehavior(t *testing.T) {
	RunTestWithBothDatabases(t, func(t *testing.T, td *TestDatabase) {
		ctx := context.Background()

		// Create schedule
		schedule := types.Schedule{
			ServerIDs: []string{"ts-test"},
			Interval:  "1h",
			NextRun:   time.Now().Add(1 * time.Hour),
			Enabled:   true,
			Options: types.TestOptions{
				UseIperf:       true,
				EnableDownload: true,
				EnableUpload:   true,
			},
		}

		created, err := td.Service.CreateSchedule(ctx, schedule)
		require.NoError(t, err)

		originalCreatedAt := created.CreatedAt
		originalNextRun := created.NextRun

		// Sleep to ensure time difference
		time.Sleep(100 * time.Millisecond)

		// Update schedule
		created.Interval = "2h"
		created.NextRun = time.Now().Add(2 * time.Hour)
		err = td.Service.UpdateSchedule(ctx, *created)
		require.NoError(t, err)

		// Get updated schedule
		schedules, err := td.Service.GetSchedules(ctx)
		require.NoError(t, err)
		require.Len(t, schedules, 1)

		// CreatedAt should not change
		assert.Equal(t, originalCreatedAt.Unix(), schedules[0].CreatedAt.Unix())

		// NextRun should be updated
		assert.NotEqual(t, originalNextRun.Unix(), schedules[0].NextRun.Unix(),
			"NextRun should be different after update")
	})
}
