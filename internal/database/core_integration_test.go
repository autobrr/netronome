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

func TestDatabaseHealth(t *testing.T) {
	RunTestWithBothDatabases(t, func(t *testing.T, td *TestDatabase) {
		health := td.Service.Health()

		assert.Equal(t, "up", health["status"])
		assert.Contains(t, health, "message")
		assert.Contains(t, health, "type")
		assert.Contains(t, health, "open_connections")
		assert.Contains(t, health, "in_use")
		assert.Contains(t, health, "idle")

		// Verify database type is reported correctly
		if td.Config.Type == "postgres" {
			assert.Equal(t, "postgres", health["type"])
		} else {
			assert.Equal(t, "sqlite", health["type"])
		}
	})
}

func TestUserManagement(t *testing.T) {
	RunTestWithBothDatabases(t, func(t *testing.T, td *TestDatabase) {
		ctx := context.Background()

		// Create user
		user, err := td.Service.CreateUser(ctx, "testuser", "testpass123")
		require.NoError(t, err)
		require.NotNil(t, user)
		assert.Equal(t, "testuser", user.Username)
		assert.NotEmpty(t, user.PasswordHash)
		assert.NotZero(t, user.ID)

		// Get user by username
		retrieved, err := td.Service.GetUserByUsername(ctx, "testuser")
		require.NoError(t, err)
		require.NotNil(t, retrieved)
		assert.Equal(t, user.ID, retrieved.ID)
		assert.Equal(t, user.Username, retrieved.Username)

		// Validate password
		assert.True(t, td.Service.ValidatePassword(retrieved, "testpass123"))
		assert.False(t, td.Service.ValidatePassword(retrieved, "wrongpass"))

		// Update password
		err = td.Service.UpdatePassword(ctx, "testuser", "newpass456")
		require.NoError(t, err)

		// Verify new password
		updated, err := td.Service.GetUserByUsername(ctx, "testuser")
		require.NoError(t, err)
		assert.True(t, td.Service.ValidatePassword(updated, "newpass456"))
		assert.False(t, td.Service.ValidatePassword(updated, "testpass123"))

		// Test duplicate username
		_, err = td.Service.CreateUser(ctx, "testuser", "anotherpass")
		assert.Error(t, err)

		// Test non-existent user
		_, err = td.Service.GetUserByUsername(ctx, "nonexistent")
		assert.Error(t, err)

		// Test update password for non-existent user
		err = td.Service.UpdatePassword(ctx, "nonexistent", "newpass")
		assert.Error(t, err)
	})
}


func TestTransactionBehavior(t *testing.T) {
	RunTestWithBothDatabases(t, func(t *testing.T, td *TestDatabase) {
		ctx := context.Background()

		// Test that operations are isolated within transactions
		// Create a user
		user, err := td.Service.CreateUser(ctx, "tx_test_user", "password")
		require.NoError(t, err)

		// Create a monitor that references the user (through context)
		monitor := CreateTestPacketLossMonitor(t, td)

		// Verify both exist
		_, err = td.Service.GetUserByUsername(ctx, "tx_test_user")
		require.NoError(t, err)

		_, err = td.Service.GetPacketLossMonitor(monitor.ID)
		require.NoError(t, err)

		// In a real transaction scenario, if one operation fails,
		// all should be rolled back. This tests basic consistency.
		AssertRecordExists(t, td, "users", "id", user.ID)
		AssertRecordExists(t, td, "packet_loss_monitors", "id", monitor.ID)
	})
}

func TestDatabaseTimeout(t *testing.T) {
	RunTestWithBothDatabases(t, func(t *testing.T, td *TestDatabase) {
		// Create a context with very short timeout
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
		defer cancel()

		// Sleep to ensure context is cancelled
		time.Sleep(10 * time.Millisecond)

		// Try to perform operation with cancelled context
		_, err := td.Service.CreateUser(ctx, "timeout_user", "password")
		assert.Error(t, err)

		// Verify the operation didn't succeed
		ctx2 := context.Background()
		_, err = td.Service.GetUserByUsername(ctx2, "timeout_user")
		assert.Error(t, err)
	})
}

func TestQueryRowMethod(t *testing.T) {
	RunTestWithBothDatabases(t, func(t *testing.T, td *TestDatabase) {
		ctx := context.Background()

		// Test the QueryRow method directly
		var count int
		row := td.Service.QueryRow(ctx, "SELECT COUNT(*) FROM users")
		err := row.Scan(&count)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, count, 0)
	})
}

func TestDatabaseConstraints(t *testing.T) {
	RunTestWithBothDatabases(t, func(t *testing.T, td *TestDatabase) {
		// Test foreign key constraints
		// Create a packet loss result for non-existent monitor
		result := &types.PacketLossResult{
			MonitorID:  99999, // Non-existent monitor
			PacketLoss: 5.0,
			CreatedAt:  time.Now(),
		}

		err := td.Service.SavePacketLossResult(result)
		assert.Error(t, err) // Should fail due to foreign key constraint

		// Test unique constraints
		ctx := context.Background()
		_, err = td.Service.CreateUser(ctx, "unique_user", "password")
		require.NoError(t, err)

		// Try to create another user with same username
		_, err = td.Service.CreateUser(ctx, "unique_user", "different_password")
		assert.Error(t, err) // Should fail due to unique constraint
	})
}

func TestLargeDataHandling(t *testing.T) {
	RunTestWithBothDatabases(t, func(t *testing.T, td *TestDatabase) {
		// Create a monitor
		monitor := CreateTestPacketLossMonitor(t, td)

		// Create a large MTR data string
		largeMTRData := `{
			"hops": [`

		for i := 0; i < 100; i++ {
			if i > 0 {
				largeMTRData += ","
			}
			largeMTRData += `{"addr": "10.0.0.` + string(rune('0'+i%10)) + `", "loss": 0, "avg": 1.5}`
		}
		largeMTRData += `]}`

		// Save result with large data
		result := &types.PacketLossResult{
			MonitorID:  monitor.ID,
			PacketLoss: 0.0,
			MTRData:    &largeMTRData,
			UsedMTR:    true,
			HopCount:   100,
			CreatedAt:  time.Now(),
		}

		err := td.Service.SavePacketLossResult(result)
		require.NoError(t, err)

		// Retrieve and verify
		latest, err := td.Service.GetLatestPacketLossResult(monitor.ID)
		require.NoError(t, err)
		require.NotNil(t, latest.MTRData)
		assert.Contains(t, *latest.MTRData, "10.0.0")
		assert.Equal(t, 100, latest.HopCount)
	})
}
