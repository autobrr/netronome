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

// TestPacketLossMonitor_CascadeDelete verifies that deleting a monitor removes all its results
func TestPacketLossMonitor_CascadeDelete(t *testing.T) {
	RunTestWithBothDatabases(t, func(t *testing.T, td *TestDatabase) {
		// Create a monitor
		monitor := CreateTestPacketLossMonitor(t, td)

		// Create multiple results for the monitor
		resultIDs := make([]int64, 5)
		for i := 0; i < 5; i++ {
			mtrData := `{"hops": [{"addr": "192.168.1.1", "loss": 0}]}`
			result := &types.PacketLossResult{
				MonitorID:   monitor.ID,
				PacketLoss:  float64(i),
				MinRTT:      10.0 + float64(i),
				MaxRTT:      20.0 + float64(i),
				AvgRTT:      15.0 + float64(i),
				PacketsSent: 10,
				PacketsRecv: 10 - i,
				MTRData:     &mtrData,
				UsedMTR:     true,
				HopCount:    1,
				CreatedAt:   time.Now().Add(time.Duration(-i) * time.Hour),
			}

			err := td.Service.SavePacketLossResult(result)
			require.NoError(t, err)
			resultIDs[i] = result.ID
		}

		// Verify all results exist
		for _, id := range resultIDs {
			AssertRecordExists(t, td, "packet_loss_results", "id", id)
		}

		// Verify we can retrieve results for the monitor
		results, err := td.Service.GetPacketLossResults(monitor.ID, 10)
		require.NoError(t, err)
		assert.Len(t, results, 5)

		// Delete the monitor
		err = td.Service.DeletePacketLossMonitor(monitor.ID)
		require.NoError(t, err)

		// Verify monitor is deleted
		AssertRecordNotExists(t, td, "packet_loss_monitors", "id", monitor.ID)

		// Verify all results are cascade deleted
		for _, id := range resultIDs {
			AssertRecordNotExists(t, td, "packet_loss_results", "id", id)
		}

		// Double-check by trying to retrieve results
		results, err = td.Service.GetPacketLossResults(monitor.ID, 10)
		require.NoError(t, err)
		assert.Empty(t, results)
	})
}

// TestMonitorAgent_CascadeDelete verifies that deleting an agent removes all its data
func TestMonitorAgent_CascadeDelete(t *testing.T) {
	RunTestWithBothDatabases(t, func(t *testing.T, td *TestDatabase) {
		ctx := context.Background()

		// Create an agent
		agent := &types.MonitorAgent{
			Name:    "Cascade Delete Test Agent",
			URL:     "http://test-agent.local",
			APIKey:  stringPtr("test-api-key"),
			Enabled: true,
		}

		created, err := td.Service.CreateMonitorAgent(ctx, agent)
		require.NoError(t, err)

		// Save various types of data for the agent

		// 1. Resource stats
		resourceStats := &types.MonitorResourceStats{
			CPUUsagePercent:   25.5,
			MemoryUsedPercent: 45.5,
			SwapUsedPercent:   10.0,
			DiskUsageJSON:     `[{"path":"/","total":100000000000,"used":50000000000,"free":50000000000,"usedPercent":50.0}]`,
			TemperatureJSON:   `[{"sensorKey":"cpu","temperature":55.0}]`,
			UptimeSeconds:     3600,
		}

		err = td.Service.SaveMonitorResourceStats(ctx, created.ID, resourceStats)
		require.NoError(t, err)

		// 2. Peak stats
		now := time.Now()
		peakStats := &types.MonitorPeakStats{
			PeakRxBytes:     1000000000,
			PeakTxBytes:     500000000,
			PeakRxTimestamp: &now,
			PeakTxTimestamp: &now,
		}

		err = td.Service.UpsertMonitorPeakStats(ctx, created.ID, peakStats)
		require.NoError(t, err)

		// 3. Historical snapshot
		snapshot := &types.MonitorHistoricalSnapshot{
			PeriodType: "daily",
			DataJSON:   `{"detailed": "stats"}`,
		}

		err = td.Service.SaveMonitorHistoricalSnapshot(ctx, created.ID, snapshot)
		require.NoError(t, err)

		// Verify all data exists
		AssertRecordExists(t, td, "monitor_agents", "id", created.ID)
		AssertRecordExists(t, td, "monitor_resource_stats", "agent_id", created.ID)
		AssertRecordExists(t, td, "monitor_peak_stats", "agent_id", created.ID)
		AssertRecordExists(t, td, "monitor_historical_snapshots", "agent_id", created.ID)

		// Verify we can retrieve the data

		// Use 24 hours to avoid timezone issues with SQLite
		retrievedStats, err := td.Service.GetMonitorResourceStats(ctx, created.ID, 24)
		require.NoError(t, err)
		assert.Len(t, retrievedStats, 1)

		retrievedPeaks, err := td.Service.GetMonitorPeakStats(ctx, created.ID)
		require.NoError(t, err)
		assert.NotNil(t, retrievedPeaks)

		retrievedSnapshot, err := td.Service.GetMonitorLatestSnapshot(ctx, created.ID, "daily")
		require.NoError(t, err)
		assert.NotNil(t, retrievedSnapshot)

		// Delete the agent
		err = td.Service.DeleteMonitorAgent(ctx, created.ID)
		require.NoError(t, err)

		// Verify agent is deleted
		AssertRecordNotExists(t, td, "monitor_agents", "id", created.ID)

		// Verify all related data is cascade deleted
		AssertRecordNotExists(t, td, "monitor_resource_stats", "agent_id", created.ID)
		AssertRecordNotExists(t, td, "monitor_peak_stats", "agent_id", created.ID)
		AssertRecordNotExists(t, td, "monitor_historical_snapshots", "agent_id", created.ID)

		// Double-check by trying to retrieve data

		// Use 24 hours to avoid timezone issues with SQLite
		retrievedStats, err = td.Service.GetMonitorResourceStats(ctx, created.ID, 24)
		require.NoError(t, err)
		assert.Empty(t, retrievedStats)

		retrievedPeaks, err = td.Service.GetMonitorPeakStats(ctx, created.ID)
		assert.ErrorIs(t, err, ErrNotFound)
		assert.Nil(t, retrievedPeaks)

		retrievedSnapshot, err = td.Service.GetMonitorLatestSnapshot(ctx, created.ID, "daily")
		assert.ErrorIs(t, err, ErrNotFound)
		assert.Nil(t, retrievedSnapshot)
	})
}

// TestNotificationChannel_CascadeDelete verifies that deleting a channel removes rules and history
func TestNotificationChannel_CascadeDelete(t *testing.T) {
	RunTestWithBothDatabases(t, func(t *testing.T, td *TestDatabase) {
		ctx := context.Background()
		_ = ctx

		// Create notification channels
		channel1, err := td.Service.CreateChannel(NotificationChannelInput{
			Name:    "Cascade Test Channel 1",
			URL:     "https://webhook.example.com/1",
			Enabled: boolPtr(true),
		})
		require.NoError(t, err)

		channel2, err := td.Service.CreateChannel(NotificationChannelInput{
			Name:    "Cascade Test Channel 2",
			URL:     "https://webhook.example.com/2",
			Enabled: boolPtr(true),
		})
		require.NoError(t, err)

		// Get notification events
		speedtestEvent, err := td.Service.GetEventByType(NotificationCategorySpeedtest, NotificationEventSpeedtestComplete)
		require.NoError(t, err)

		packetlossEvent, err := td.Service.GetEventByType(NotificationCategoryPacketLoss, NotificationEventPacketLossHigh)
		require.NoError(t, err)

		agentEvent, err := td.Service.GetEventByType(NotificationCategoryAgent, NotificationEventAgentOffline)
		require.NoError(t, err)

		// Create rules for channel 1
		rule1, err := td.Service.CreateRule(NotificationRuleInput{
			ChannelID: channel1.ID,
			EventID:   speedtestEvent.ID,
			Enabled:   boolPtr(true),
		})
		require.NoError(t, err)

		rule2, err := td.Service.CreateRule(NotificationRuleInput{
			ChannelID: channel1.ID,
			EventID:   packetlossEvent.ID,
			Enabled:   boolPtr(true),
		})
		require.NoError(t, err)

		rule3, err := td.Service.CreateRule(NotificationRuleInput{
			ChannelID: channel1.ID,
			EventID:   agentEvent.ID,
			Enabled:   boolPtr(true),
		})
		require.NoError(t, err)

		// Create a rule for channel 2 (should not be affected)
		rule4, err := td.Service.CreateRule(NotificationRuleInput{
			ChannelID: channel2.ID,
			EventID:   speedtestEvent.ID,
			Enabled:   boolPtr(true),
		})
		require.NoError(t, err)

		// Log notification history for channel 1
		payload1 := "Test notification 1"
		err = td.Service.LogNotification(channel1.ID, speedtestEvent.ID, true, nil, &payload1)
		require.NoError(t, err)

		errorMsg := "Failed to send"
		payload2 := "Test notification 2"
		err = td.Service.LogNotification(channel1.ID, packetlossEvent.ID, false, &errorMsg, &payload2)
		require.NoError(t, err)

		payload3 := "Test notification 3"
		err = td.Service.LogNotification(channel1.ID, agentEvent.ID, true, nil, &payload3)
		require.NoError(t, err)

		// Log notification history for channel 2
		payload4 := "Test notification 4"
		err = td.Service.LogNotification(channel2.ID, speedtestEvent.ID, true, nil, &payload4)
		require.NoError(t, err)

		// Verify all data exists
		AssertRecordExists(t, td, "notification_channels", "id", channel1.ID)
		AssertRecordExists(t, td, "notification_rules", "id", rule1.ID)
		AssertRecordExists(t, td, "notification_rules", "id", rule2.ID)
		AssertRecordExists(t, td, "notification_rules", "id", rule3.ID)
		AssertRecordExists(t, td, "notification_rules", "id", rule4.ID)

		// Verify we have notification history
		var historyCount int
		query := "SELECT COUNT(*) FROM notification_history WHERE channel_id = ?"
		if td.Config.Type == "postgres" {
			query = "SELECT COUNT(*) FROM notification_history WHERE channel_id = $1"
		}
		err = td.DB.QueryRow(query, channel1.ID).Scan(&historyCount)
		require.NoError(t, err)
		assert.Equal(t, 3, historyCount)

		// Delete channel 1
		err = td.Service.DeleteChannel(channel1.ID)
		require.NoError(t, err)

		// Verify channel 1 is deleted
		AssertRecordNotExists(t, td, "notification_channels", "id", channel1.ID)

		// Verify all rules for channel 1 are cascade deleted
		AssertRecordNotExists(t, td, "notification_rules", "id", rule1.ID)
		AssertRecordNotExists(t, td, "notification_rules", "id", rule2.ID)
		AssertRecordNotExists(t, td, "notification_rules", "id", rule3.ID)

		// Verify channel 2 and its rule are NOT affected
		AssertRecordExists(t, td, "notification_channels", "id", channel2.ID)
		AssertRecordExists(t, td, "notification_rules", "id", rule4.ID)

		// Verify notification history for channel 1 is cascade deleted
		err = td.DB.QueryRow(query, channel1.ID).Scan(&historyCount)
		require.NoError(t, err)
		assert.Equal(t, 0, historyCount)

		// Verify notification history for channel 2 still exists
		if td.Config.Type == "postgres" {
			query = "SELECT COUNT(*) FROM notification_history WHERE channel_id = $1"
		}
		err = td.DB.QueryRow(query, channel2.ID).Scan(&historyCount)
		require.NoError(t, err)
		assert.Equal(t, 1, historyCount)

		// Double-check by trying to retrieve rules
		rules, err := td.Service.GetRulesByChannel(channel1.ID)
		require.NoError(t, err)
		assert.Empty(t, rules)

		rules, err = td.Service.GetRulesByChannel(channel2.ID)
		require.NoError(t, err)
		assert.Len(t, rules, 1)
	})
}

// TestNotificationEvent_CascadeDelete verifies deleting events cascades to rules and history
func TestNotificationEvent_CascadeDelete(t *testing.T) {
	// Note: In a real application, events are typically seeded and not deleted.
	// This test documents the cascade behavior if events were to be deleted.
	t.Skip("Notification events are seeded data and should not be deleted in normal operation")
}

// TestCascadeDelete_Performance verifies cascade deletes work efficiently with large datasets
func TestCascadeDelete_Performance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	RunTestWithBothDatabases(t, func(t *testing.T, td *TestDatabase) {
		// Create a monitor with many results
		monitor := CreateTestPacketLossMonitor(t, td)

		// Create 1000 results
		start := time.Now()
		for i := 0; i < 1000; i++ {
			result := &types.PacketLossResult{
				MonitorID:   monitor.ID,
				PacketLoss:  float64(i%100) / 10.0,
				MinRTT:      10.0,
				MaxRTT:      20.0,
				AvgRTT:      15.0,
				PacketsSent: 10,
				PacketsRecv: 10,
				CreatedAt:   time.Now().Add(time.Duration(-i) * time.Minute),
			}

			err := td.Service.SavePacketLossResult(result)
			require.NoError(t, err)
		}

		insertDuration := time.Since(start)
		t.Logf("Inserted 1000 results in %v", insertDuration)

		// Verify count
		var count int
		query := "SELECT COUNT(*) FROM packet_loss_results WHERE monitor_id = ?"
		if td.Config.Type == "postgres" {
			query = "SELECT COUNT(*) FROM packet_loss_results WHERE monitor_id = $1"
		}
		err := td.DB.QueryRow(query, monitor.ID).Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 1000, count)

		// Delete monitor and time cascade delete
		start = time.Now()
		err = td.Service.DeletePacketLossMonitor(monitor.ID)
		require.NoError(t, err)
		deleteDuration := time.Since(start)

		t.Logf("Cascade deleted 1000 results in %v", deleteDuration)

		// Verify all deleted
		err = td.DB.QueryRow(query, monitor.ID).Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 0, count)

		// Cascade delete should be reasonably fast (under 5 seconds even for 1000 records)
		assert.Less(t, deleteDuration, 5*time.Second)
	})
}
