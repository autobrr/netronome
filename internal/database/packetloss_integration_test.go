// Copyright (c) 2024-2025, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package database

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/autobrr/netronome/internal/types"
)

func TestPacketLossMonitor_CRUD(t *testing.T) {
	RunTestWithBothDatabases(t, func(t *testing.T, td *TestDatabase) {
		// Create
		monitor := &types.PacketLossMonitor{
			Name:        "Test Monitor",
			Host:        "8.8.8.8",
			Interval:    "60s",
			PacketCount: 10,
			Enabled:     true,
			Threshold:   5.0,
			LastState:   "healthy",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		created, err := td.Service.CreatePacketLossMonitor(monitor)
		require.NoError(t, err)
		require.NotNil(t, created)
		assert.Greater(t, created.ID, int64(0))
		assert.Equal(t, monitor.Name, created.Name)
		assert.Equal(t, monitor.Host, created.Host)

		// Read
		retrieved, err := td.Service.GetPacketLossMonitor(created.ID)
		require.NoError(t, err)
		require.NotNil(t, retrieved)
		assert.Equal(t, created.ID, retrieved.ID)
		assert.Equal(t, created.Name, retrieved.Name)

		// Update
		retrieved.Name = "Updated Monitor"
		retrieved.Enabled = false
		err = td.Service.UpdatePacketLossMonitor(retrieved)
		require.NoError(t, err)

		// Verify update
		updated, err := td.Service.GetPacketLossMonitor(retrieved.ID)
		require.NoError(t, err)
		assert.Equal(t, "Updated Monitor", updated.Name)
		assert.False(t, updated.Enabled)

		// Delete
		err = td.Service.DeletePacketLossMonitor(created.ID)
		require.NoError(t, err)

		// Verify deletion
		deleted, err := td.Service.GetPacketLossMonitor(created.ID)
		assert.Error(t, err)
		assert.Nil(t, deleted)
	})
}

func TestGetEnabledPacketLossMonitors(t *testing.T) {
	RunTestWithBothDatabases(t, func(t *testing.T, td *TestDatabase) {
		// Create multiple monitors
		monitors := []types.PacketLossMonitor{
			{
				Name:        "Enabled Monitor 1",
				Host:        "8.8.8.8",
				Interval:    "30s",
				PacketCount: 5,
				Enabled:     true,
				Threshold:   10.0,
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			},
			{
				Name:        "Disabled Monitor",
				Host:        "1.1.1.1",
				Interval:    "60s",
				PacketCount: 10,
				Enabled:     false,
				Threshold:   5.0,
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			},
			{
				Name:        "Enabled Monitor 2",
				Host:        "4.4.4.4",
				Interval:    "120s",
				PacketCount: 20,
				Enabled:     true,
				Threshold:   15.0,
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			},
		}

		for _, m := range monitors {
			monitor := m // capture loop variable
			_, err := td.Service.CreatePacketLossMonitor(&monitor)
			require.NoError(t, err)
		}

		// Get only enabled monitors
		enabled, err := td.Service.GetEnabledPacketLossMonitors()
		require.NoError(t, err)
		assert.Len(t, enabled, 2)

		// Verify all returned monitors are enabled
		for _, m := range enabled {
			assert.True(t, m.Enabled)
		}
	})
}

func TestSavePacketLossResult(t *testing.T) {
	RunTestWithBothDatabases(t, func(t *testing.T, td *TestDatabase) {
		// Create a monitor first
		monitor := CreateTestPacketLossMonitor(t, td)

		// Save result
		result := &types.PacketLossResult{
			MonitorID:      monitor.ID,
			PacketLoss:     2.5,
			MinRTT:         10.1,
			MaxRTT:         50.5,
			AvgRTT:         25.3,
			StdDevRTT:      5.2,
			PacketsSent:    100,
			PacketsRecv:    97,
			UsedMTR:        true,
			HopCount:       10,
			MTRData:        stringPtr(`{"hops":[]}`),
			PrivilegedMode: true,
			CreatedAt:      time.Now(),
		}

		err := td.Service.SavePacketLossResult(result)
		require.NoError(t, err)
		assert.Greater(t, result.ID, int64(0))

		// Verify saved
		AssertRecordExists(t, td, "packet_loss_results", "id", result.ID)
	})
}

func TestGetLatestPacketLossResult(t *testing.T) {
	RunTestWithBothDatabases(t, func(t *testing.T, td *TestDatabase) {
		// Create a monitor
		monitor := CreateTestPacketLossMonitor(t, td)

		// Save multiple results
		results := []types.PacketLossResult{
			{
				MonitorID:   monitor.ID,
				PacketLoss:  1.0,
				AvgRTT:      20.0,
				PacketsSent: 100,
				PacketsRecv: 99,
				CreatedAt:   time.Now().Add(-2 * time.Hour),
			},
			{
				MonitorID:   monitor.ID,
				PacketLoss:  2.0,
				AvgRTT:      25.0,
				PacketsSent: 100,
				PacketsRecv: 98,
				CreatedAt:   time.Now().Add(-1 * time.Hour),
			},
			{
				MonitorID:   monitor.ID,
				PacketLoss:  0.0,
				AvgRTT:      15.0,
				PacketsSent: 100,
				PacketsRecv: 100,
				CreatedAt:   time.Now(),
			},
		}

		for _, r := range results {
			result := r // capture loop variable
			err := td.Service.SavePacketLossResult(&result)
			require.NoError(t, err)
		}

		// Get latest result
		latest, err := td.Service.GetLatestPacketLossResult(monitor.ID)
		require.NoError(t, err)
		require.NotNil(t, latest)

		// Should be the most recent one (0% packet loss)
		assert.Equal(t, 0.0, latest.PacketLoss)
		assert.Equal(t, 15.0, latest.AvgRTT)
	})
}

func TestGetPacketLossResults(t *testing.T) {
	RunTestWithBothDatabases(t, func(t *testing.T, td *TestDatabase) {
		// Create a monitor
		monitor := CreateTestPacketLossMonitor(t, td)

		// Save multiple results
		for i := 0; i < 10; i++ {
			result := &types.PacketLossResult{
				MonitorID:   monitor.ID,
				PacketLoss:  float64(i),
				AvgRTT:      20.0 + float64(i),
				PacketsSent: 100,
				PacketsRecv: 100 - i,
				CreatedAt:   time.Now().Add(time.Duration(-i) * time.Hour),
			}
			err := td.Service.SavePacketLossResult(result)
			require.NoError(t, err)
		}

		// Get results with limit
		results, err := td.Service.GetPacketLossResults(monitor.ID, 5)
		require.NoError(t, err)
		assert.Len(t, results, 5)

		// Verify order (should be newest first)
		for i := 1; i < len(results); i++ {
			assert.True(t, results[i-1].CreatedAt.After(results[i].CreatedAt) ||
				results[i-1].CreatedAt.Equal(results[i].CreatedAt))
		}
	})
}

func TestUpdatePacketLossMonitorState(t *testing.T) {
	RunTestWithBothDatabases(t, func(t *testing.T, td *TestDatabase) {
		// Create a monitor
		monitor := CreateTestPacketLossMonitor(t, td)

		// Update state
		newState := "degraded"
		err := td.Service.UpdatePacketLossMonitorState(monitor.ID, newState)
		require.NoError(t, err)

		// Verify state change
		updated, err := td.Service.GetPacketLossMonitor(monitor.ID)
		require.NoError(t, err)
		assert.Equal(t, newState, updated.LastState)
		assert.NotNil(t, updated.LastStateChange)
	})
}

func TestPacketLossMonitor_NotFound(t *testing.T) {
	RunTestWithBothDatabases(t, func(t *testing.T, td *TestDatabase) {
		// Try to get non-existent monitor
		monitor, err := td.Service.GetPacketLossMonitor(99999)
		assert.Error(t, err)
		assert.Nil(t, monitor)

		// Try to update non-existent monitor
		err = td.Service.UpdatePacketLossMonitorState(99999, "healthy")
		assert.Error(t, err)

		// Try to delete non-existent monitor
		_ = td.Service.DeletePacketLossMonitor(99999)
		// Deletion might not return error for non-existent records
		// depending on implementation
	})
}

func TestSavePacketLossResult_WithMTRData(t *testing.T) {
	RunTestWithBothDatabases(t, func(t *testing.T, td *TestDatabase) {
		// Create a monitor
		monitor := CreateTestPacketLossMonitor(t, td)

		// Complex MTR data
		mtrData := `{
			"hops": [
				{"addr": "192.168.1.1", "loss": 0, "avg": 1.2},
				{"addr": "10.0.0.1", "loss": 0, "avg": 5.3},
				{"addr": "8.8.8.8", "loss": 0, "avg": 15.4}
			]
		}`

		result := &types.PacketLossResult{
			MonitorID:      monitor.ID,
			PacketLoss:     0.0,
			MinRTT:         1.0,
			MaxRTT:         20.0,
			AvgRTT:         10.0,
			StdDevRTT:      2.5,
			PacketsSent:    100,
			PacketsRecv:    100,
			UsedMTR:        true,
			HopCount:       3,
			MTRData:        &mtrData,
			PrivilegedMode: true,
			CreatedAt:      time.Now(),
		}

		err := td.Service.SavePacketLossResult(result)
		require.NoError(t, err)

		// Verify MTR data was saved correctly
		latest, err := td.Service.GetLatestPacketLossResult(monitor.ID)
		require.NoError(t, err)
		require.NotNil(t, latest.MTRData)
		assert.Contains(t, *latest.MTRData, "192.168.1.1")
		assert.Equal(t, 3, latest.HopCount)
	})
}
