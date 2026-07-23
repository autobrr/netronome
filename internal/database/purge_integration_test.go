// Copyright (c) 2024-2026, s0up and the autobrr contributors.
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

func TestPurgeHistoricalData(t *testing.T) {
	RunTestWithBothDatabases(t, func(t *testing.T, td *TestDatabase) {
		ctx := context.Background()

		now := time.Now()
		old := now.Add(-30 * 24 * time.Hour)   // 30 days ago
		recent := now.Add(-1 * time.Hour)      // 1 hour ago
		cutoff := now.Add(-7 * 24 * time.Hour) // 7 days ago (between old and recent)

		// speed_tests: one old, one recent
		_, err := td.Service.SaveSpeedTest(ctx, types.SpeedTestResult{
			ServerName: "old", ServerID: "old", TestType: "speedtest", CreatedAt: old,
		})
		require.NoError(t, err)
		recentST, err := td.Service.SaveSpeedTest(ctx, types.SpeedTestResult{
			ServerName: "recent", ServerID: "recent", TestType: "speedtest", CreatedAt: recent,
		})
		require.NoError(t, err)

		// packet_loss_results: needs a monitor (FK), one old, one recent
		monitor := CreateTestPacketLossMonitor(t, td)
		oldPL := &types.PacketLossResult{MonitorID: monitor.ID, PacketsSent: 10, PacketsRecv: 10, CreatedAt: old}
		require.NoError(t, td.Service.SavePacketLossResult(oldPL))
		recentPL := &types.PacketLossResult{MonitorID: monitor.ID, PacketsSent: 10, PacketsRecv: 10, CreatedAt: recent}
		require.NoError(t, td.Service.SavePacketLossResult(recentPL))

		// Purge everything older than the cutoff.
		speedTests, packetLoss, err := td.Service.PurgeHistoricalData(ctx, cutoff)
		require.NoError(t, err)
		assert.Equal(t, int64(1), speedTests)
		assert.Equal(t, int64(1), packetLoss)

		// Only the recent rows should survive.
		AssertRecordExists(t, td, "speed_tests", "id", recentST.ID)
		AssertRecordExists(t, td, "packet_loss_results", "id", recentPL.ID)

		var stCount, plCount int
		require.NoError(t, td.DB.QueryRow("SELECT COUNT(*) FROM speed_tests").Scan(&stCount))
		require.NoError(t, td.DB.QueryRow("SELECT COUNT(*) FROM packet_loss_results").Scan(&plCount))
		assert.Equal(t, 1, stCount)
		assert.Equal(t, 1, plCount)
	})
}
