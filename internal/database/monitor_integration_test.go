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

// Helper function for bool pointers
func boolPtr(b bool) *bool {
	return &b
}

// Helper function for time pointers
func timePtr(t time.Time) *time.Time {
	return &t
}

func TestMonitorAgent_CRUD(t *testing.T) {
	RunTestWithBothDatabases(t, func(t *testing.T, td *TestDatabase) {
		ctx := context.Background()

		// Create agent
		agent := &types.MonitorAgent{
			Name:    "Test Agent",
			URL:     "http://agent.example.com:8080",
			APIKey:  stringPtr("test-api-key-123"),
			Enabled: true,
		}

		created, err := td.Service.CreateMonitorAgent(ctx, agent)
		require.NoError(t, err)
		require.NotNil(t, created)
		assert.Greater(t, created.ID, int64(0))
		assert.Equal(t, agent.Name, created.Name)
		assert.Equal(t, agent.URL, created.URL)
		assert.NotZero(t, created.CreatedAt)
		assert.NotZero(t, created.UpdatedAt)

		// Get agent by ID
		retrieved, err := td.Service.GetMonitorAgent(ctx, created.ID)
		require.NoError(t, err)
		require.NotNil(t, retrieved)
		assert.Equal(t, created.ID, retrieved.ID)
		assert.Equal(t, created.Name, retrieved.Name)

		// Update agent
		retrieved.Name = "Updated Agent"
		retrieved.Enabled = false
		err = td.Service.UpdateMonitorAgent(ctx, retrieved)
		require.NoError(t, err)

		// Verify update
		updated, err := td.Service.GetMonitorAgent(ctx, retrieved.ID)
		require.NoError(t, err)
		assert.Equal(t, "Updated Agent", updated.Name)
		assert.False(t, updated.Enabled)

		// Delete agent
		err = td.Service.DeleteMonitorAgent(ctx, created.ID)
		require.NoError(t, err)

		// Verify deletion
		deleted, err := td.Service.GetMonitorAgent(ctx, created.ID)
		assert.Error(t, err)
		assert.Nil(t, deleted)
	})
}

func TestMonitorAgent_GetEnabledOnly(t *testing.T) {
	RunTestWithBothDatabases(t, func(t *testing.T, td *TestDatabase) {
		ctx := context.Background()

		// Create multiple agents
		agents := []*types.MonitorAgent{
			{
				Name:    "Enabled Agent 1",
				URL:     "http://agent1.example.com",
				APIKey:  stringPtr("key1"),
				Enabled: true,
			},
			{
				Name:    "Disabled Agent",
				URL:     "http://agent2.example.com",
				APIKey:  stringPtr("key2"),
				Enabled: false,
			},
			{
				Name:    "Enabled Agent 2",
				URL:     "http://agent3.example.com",
				APIKey:  stringPtr("key3"),
				Enabled: true,
			},
		}

		for _, agent := range agents {
			_, err := td.Service.CreateMonitorAgent(ctx, agent)
			require.NoError(t, err)
		}

		// Get all agents
		allAgents, err := td.Service.GetMonitorAgents(ctx, false)
		require.NoError(t, err)
		assert.Len(t, allAgents, 3)

		// Get enabled only
		enabledAgents, err := td.Service.GetMonitorAgents(ctx, true)
		require.NoError(t, err)
		assert.Len(t, enabledAgents, 2)

		// Verify all returned agents are enabled
		for _, agent := range enabledAgents {
			assert.True(t, agent.Enabled)
		}
	})
}

func TestMonitorAgent_SystemInfo(t *testing.T) {
	RunTestWithBothDatabases(t, func(t *testing.T, td *TestDatabase) {
		ctx := context.Background()

		// Create agent
		agent := &types.MonitorAgent{
			Name:    "Info Test Agent",
			URL:     "http://agent.example.com",
			APIKey:  stringPtr("test-key"),
			Enabled: true,
		}

		created, err := td.Service.CreateMonitorAgent(ctx, agent)
		require.NoError(t, err)

		// Create system info
		sysInfo := &types.MonitorSystemInfo{
			Hostname:      "test-host",
			Kernel:        "Linux 5.15.0 x86_64",
			VnstatVersion: "2.10",
			AgentVersion:  "1.0.0",
			CPUModel:      "Intel Core i7",
			CPUCores:      4,
			CPUThreads:    8,
			TotalMemory:   16000000000,
		}

		// Upsert system info
		err = td.Service.UpsertMonitorSystemInfo(ctx, created.ID, sysInfo)
		require.NoError(t, err)

		// Get system info
		retrieved, err := td.Service.GetMonitorSystemInfo(ctx, created.ID)
		require.NoError(t, err)
		require.NotNil(t, retrieved)
		assert.Equal(t, sysInfo.Hostname, retrieved.Hostname)
		assert.Equal(t, sysInfo.Kernel, retrieved.Kernel)

		// Update system info
		sysInfo.Kernel = "Linux 5.16.0 x86_64"
		err = td.Service.UpsertMonitorSystemInfo(ctx, created.ID, sysInfo)
		require.NoError(t, err)

		// Verify update
		updated, err := td.Service.GetMonitorSystemInfo(ctx, created.ID)
		require.NoError(t, err)
		assert.Equal(t, "Linux 5.16.0 x86_64", updated.Kernel)
	})
}

func TestMonitorAgent_Interfaces(t *testing.T) {
	RunTestWithBothDatabases(t, func(t *testing.T, td *TestDatabase) {
		ctx := context.Background()

		// Create agent
		agent := &types.MonitorAgent{
			Name:    "Interface Test Agent",
			URL:     "http://agent.example.com",
			APIKey:  stringPtr("test-key"),
			Enabled: true,
		}

		created, err := td.Service.CreateMonitorAgent(ctx, agent)
		require.NoError(t, err)

		// Create interfaces
		interfaces := []types.MonitorInterface{
			{
				Name:      "eth0",
				Alias:     "Ethernet",
				IPAddress: "192.168.1.100",
				LinkSpeed: 1000,
			},
			{
				Name:      "lo",
				Alias:     "Loopback",
				IPAddress: "127.0.0.1",
				LinkSpeed: 0,
			},
		}

		// Upsert interfaces
		err = td.Service.UpsertMonitorInterfaces(ctx, created.ID, interfaces)
		require.NoError(t, err)

		// Get interfaces
		retrieved, err := td.Service.GetMonitorInterfaces(ctx, created.ID)
		require.NoError(t, err)
		assert.Len(t, retrieved, 2)

		// Verify interface data
		foundInterfaces := make(map[string]bool)
		for _, iface := range retrieved {
			foundInterfaces[iface.Name] = true
		}
		assert.True(t, foundInterfaces["eth0"])
		assert.True(t, foundInterfaces["lo"])

		// Update interfaces - add more interfaces
		interfaces = append(interfaces, types.MonitorInterface{
			Name:      "wlan0",
			Alias:     "WiFi",
			IPAddress: "192.168.1.101",
			LinkSpeed: 300,
		})
		err = td.Service.UpsertMonitorInterfaces(ctx, created.ID, interfaces)
		require.NoError(t, err)

		// Verify update
		updated, err := td.Service.GetMonitorInterfaces(ctx, created.ID)
		require.NoError(t, err)
		assert.Len(t, updated, 3)
	})
}

func TestMonitorAgent_PeakStats(t *testing.T) {
	RunTestWithBothDatabases(t, func(t *testing.T, td *TestDatabase) {
		ctx := context.Background()

		// Create agent
		agent := &types.MonitorAgent{
			Name:    "Peak Stats Test Agent",
			URL:     "http://agent.example.com",
			APIKey:  stringPtr("test-key"),
			Enabled: true,
		}

		created, err := td.Service.CreateMonitorAgent(ctx, agent)
		require.NoError(t, err)

		// Create peak stats
		now := time.Now()
		peakStats := &types.MonitorPeakStats{
			PeakRxBytes:     1000000000,
			PeakTxBytes:     500000000,
			PeakRxTimestamp: &now,
			PeakTxTimestamp: &now,
		}

		// Upsert peak stats
		err = td.Service.UpsertMonitorPeakStats(ctx, created.ID, peakStats)
		require.NoError(t, err)

		// Get peak stats
		retrieved, err := td.Service.GetMonitorPeakStats(ctx, created.ID)
		require.NoError(t, err)
		require.NotNil(t, retrieved)
		assert.Equal(t, peakStats.PeakRxBytes, retrieved.PeakRxBytes)
		assert.Equal(t, peakStats.PeakTxBytes, retrieved.PeakTxBytes)

		// Update peak stats
		peakStats.PeakRxBytes = 2000000000
		err = td.Service.UpsertMonitorPeakStats(ctx, created.ID, peakStats)
		require.NoError(t, err)

		// Verify update
		updated, err := td.Service.GetMonitorPeakStats(ctx, created.ID)
		require.NoError(t, err)
		assert.Equal(t, int64(2000000000), updated.PeakRxBytes)
	})
}

func TestMonitorAgent_ResourceStats(t *testing.T) {
	RunTestWithBothDatabases(t, func(t *testing.T, td *TestDatabase) {
		ctx := context.Background()

		// Create agent
		agent := &types.MonitorAgent{
			Name:    "Resource Test Agent",
			URL:     "http://agent.example.com",
			APIKey:  stringPtr("test-key"),
			Enabled: true,
		}

		created, err := td.Service.CreateMonitorAgent(ctx, agent)
		require.NoError(t, err)

		// Save multiple resource stats
		for i := 0; i < 5; i++ {
			stats := &types.MonitorResourceStats{
				CPUUsagePercent:   float64(20 + i*5),
				MemoryUsedPercent: float64(40 + i*2),
				SwapUsedPercent:   float64(10 + i),
				DiskUsageJSON:     `{"disks":[{"path":"/","used":60,"total":100}]}`,
				TemperatureJSON:   `{"temps":[{"sensor":"cpu","temp":45}]}`,
				UptimeSeconds:     int64(3600 + i*100),
			}

			err = td.Service.SaveMonitorResourceStats(ctx, created.ID, stats)
			require.NoError(t, err)

			time.Sleep(10 * time.Millisecond)
		}

		// Get stats for last 24 hours to avoid timezone issues
		stats, err := td.Service.GetMonitorResourceStats(ctx, created.ID, 24)
		require.NoError(t, err)

		assert.Len(t, stats, 5)

		// Verify order and values
		for i := 1; i < len(stats); i++ {
			assert.True(t, stats[i-1].CreatedAt.After(stats[i].CreatedAt) ||
				stats[i-1].CreatedAt.Equal(stats[i].CreatedAt))
		}
	})
}

func TestMonitorAgent_HistoricalSnapshot(t *testing.T) {
	RunTestWithBothDatabases(t, func(t *testing.T, td *TestDatabase) {
		ctx := context.Background()

		// Create agent
		agent := &types.MonitorAgent{
			Name:    "Snapshot Test Agent",
			URL:     "http://agent.example.com",
			APIKey:  stringPtr("test-key"),
			Enabled: true,
		}

		created, err := td.Service.CreateMonitorAgent(ctx, agent)
		require.NoError(t, err)

		// Save daily snapshot
		dailySnapshot := &types.MonitorHistoricalSnapshot{
			InterfaceName: "eth0",
			PeriodType:    "daily",
			DataJSON: `{
				"rx_bytes": 1000000000,
				"tx_bytes": 500000000,
				"rx_packets": 1000000,
				"tx_packets": 500000,
				"avg_cpu_usage": 25.5,
				"avg_memory_usage": 45.0,
				"avg_temperature": 50.0,
				"peak_temperature": 65.0
			}`,
		}

		err = td.Service.SaveMonitorHistoricalSnapshot(ctx, created.ID, dailySnapshot)
		require.NoError(t, err)

		// Get latest daily snapshot
		retrieved, err := td.Service.GetMonitorLatestSnapshot(ctx, created.ID, "daily")
		require.NoError(t, err)
		require.NotNil(t, retrieved)
		assert.Equal(t, dailySnapshot.DataJSON, retrieved.DataJSON)
		assert.Equal(t, dailySnapshot.InterfaceName, retrieved.InterfaceName)

		// Save monthly snapshot
		monthlySnapshot := &types.MonitorHistoricalSnapshot{
			InterfaceName: "eth0",
			PeriodType:    "monthly",
			DataJSON: `{
				"rx_bytes": 30000000000,
				"tx_bytes": 15000000000,
				"rx_packets": 30000000,
				"tx_packets": 15000000,
				"avg_cpu_usage": 30.0,
				"avg_memory_usage": 50.0,
				"avg_temperature": 52.0,
				"peak_temperature": 70.0
			}`,
		}

		err = td.Service.SaveMonitorHistoricalSnapshot(ctx, created.ID, monthlySnapshot)
		require.NoError(t, err)

		// Get latest monthly snapshot
		retrieved, err = td.Service.GetMonitorLatestSnapshot(ctx, created.ID, "monthly")
		require.NoError(t, err)
		require.NotNil(t, retrieved)
		assert.Equal(t, monthlySnapshot.DataJSON, retrieved.DataJSON)
	})
}

func TestMonitorAgent_CleanupData(t *testing.T) {
	RunTestWithBothDatabases(t, func(t *testing.T, td *TestDatabase) {
		ctx := context.Background()

		// This test verifies the cleanup function doesn't error
		// In a real scenario, we'd need to create old data and verify it's cleaned
		err := td.Service.CleanupMonitorData(ctx)
		assert.NoError(t, err)
	})
}

func TestMonitorAgent_TailscaleFields(t *testing.T) {
	RunTestWithBothDatabases(t, func(t *testing.T, td *TestDatabase) {
		ctx := context.Background()

		// Create agent with Tailscale fields
		agent := &types.MonitorAgent{
			Name:              "Tailscale Agent",
			URL:               "http://agent.example.com",
			APIKey:            stringPtr("test-key"),
			Enabled:           true,
			IsTailscale:       true,
			TailscaleHostname: stringPtr("agent-ts"),
			Interface:         stringPtr("tailscale0"),
			DiscoveredAt:      timePtr(time.Now()),
		}

		created, err := td.Service.CreateMonitorAgent(ctx, agent)
		require.NoError(t, err)

		// Retrieve and verify Tailscale fields
		retrieved, err := td.Service.GetMonitorAgent(ctx, created.ID)
		require.NoError(t, err)
		assert.True(t, retrieved.IsTailscale)
		assert.NotNil(t, retrieved.TailscaleHostname)
		assert.Equal(t, *agent.TailscaleHostname, *retrieved.TailscaleHostname)
		assert.NotNil(t, retrieved.Interface)
		assert.Equal(t, *agent.Interface, *retrieved.Interface)
	})
}
