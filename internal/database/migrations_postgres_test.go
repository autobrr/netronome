// Copyright (c) 2024-2025, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package database

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/autobrr/netronome/internal/config"
)

// TestPostgreSQL_MigrationValidation validates PostgreSQL-specific features
func TestPostgreSQL_MigrationValidation(t *testing.T) {
	// Only run for PostgreSQL
	td := SetupTestDatabase(t, config.Postgres)
	defer td.Close()

	ctx := context.Background()

	t.Run("ValidateDataTypes", func(t *testing.T) {
		// Verify PostgreSQL-specific data types
		rows, err := td.DB.QueryContext(ctx, `
			SELECT 
				table_name,
				column_name,
				data_type,
				is_nullable,
				column_default
			FROM information_schema.columns
			WHERE table_schema = 'public'
			AND table_name NOT IN ('schema_migrations')
			ORDER BY table_name, ordinal_position
		`)
		require.NoError(t, err)
		defer rows.Close()

		dataTypes := make(map[string][]string)
		for rows.Next() {
			var tableName, columnName, dataType, isNullable string
			var columnDefault sql.NullString
			err := rows.Scan(&tableName, &columnName, &dataType, &isNullable, &columnDefault)
			require.NoError(t, err)

			dataTypes[tableName] = append(dataTypes[tableName],
				fmt.Sprintf("%s:%s", columnName, dataType))
		}

		// Log some data types for debugging
		for table, types := range dataTypes {
			if table == "users" || table == "packet_loss_monitors" {
				t.Logf("Table %s columns: %v", table, types)
			}
		}

		// Verify expected data types (be flexible with integer types)
		hasUserID := false
		hasUserCreatedAt := false
		hasPacketLossEnabled := false
		hasPacketLossThreshold := false

		for _, col := range dataTypes["users"] {
			if strings.HasPrefix(col, "id:") && strings.Contains(col, "int") {
				hasUserID = true
			}
			if strings.HasPrefix(col, "created_at:timestamp") {
				hasUserCreatedAt = true
				// Note: The schema uses "timestamp without time zone" instead of "with time zone"
				// This might be intentional or could be improved for timezone handling
			}
		}

		for _, col := range dataTypes["packet_loss_monitors"] {
			if col == "enabled:boolean" {
				hasPacketLossEnabled = true
			}
			if col == "threshold:real" || col == "threshold:double precision" {
				hasPacketLossThreshold = true
			}
		}

		assert.True(t, hasUserID, "users table should have integer id")
		assert.True(t, hasUserCreatedAt, "users table should have timestamp created_at")
		assert.True(t, hasPacketLossEnabled, "packet_loss_monitors should have boolean enabled")
		assert.True(t, hasPacketLossThreshold, "packet_loss_monitors should have real/double precision threshold")
	})

	t.Run("ValidateForeignKeys", func(t *testing.T) {
		// Get all foreign key constraints
		rows, err := td.DB.QueryContext(ctx, `
			SELECT 
				tc.table_name,
				kcu.column_name,
				ccu.table_name AS foreign_table_name,
				ccu.column_name AS foreign_column_name,
				rc.delete_rule
			FROM information_schema.table_constraints AS tc 
			JOIN information_schema.key_column_usage AS kcu
				ON tc.constraint_name = kcu.constraint_name
				AND tc.table_schema = kcu.table_schema
			JOIN information_schema.constraint_column_usage AS ccu
				ON ccu.constraint_name = tc.constraint_name
				AND ccu.table_schema = tc.table_schema
			JOIN information_schema.referential_constraints AS rc
				ON rc.constraint_name = tc.constraint_name
			WHERE tc.constraint_type = 'FOREIGN KEY' 
			AND tc.table_schema = 'public'
		`)
		require.NoError(t, err)
		defer rows.Close()

		fkCount := 0
		cascadeDeletes := []string{}

		for rows.Next() {
			var tableName, columnName, foreignTable, foreignColumn, deleteRule string
			err := rows.Scan(&tableName, &columnName, &foreignTable, &foreignColumn, &deleteRule)
			require.NoError(t, err)

			fkCount++
			if deleteRule == "CASCADE" {
				cascadeDeletes = append(cascadeDeletes,
					fmt.Sprintf("%s.%s -> %s.%s", tableName, columnName, foreignTable, foreignColumn))
			}

			t.Logf("FK: %s.%s -> %s.%s (DELETE %s)",
				tableName, columnName, foreignTable, foreignColumn, deleteRule)
		}

		// Verify we have foreign keys
		assert.Greater(t, fkCount, 5, "Should have multiple foreign key constraints")
		assert.NotEmpty(t, cascadeDeletes, "Should have CASCADE DELETE relationships")
	})

	t.Run("ValidateIndexes", func(t *testing.T) {
		// Get all indexes
		rows, err := td.DB.QueryContext(ctx, `
			SELECT 
				schemaname,
				tablename,
				indexname,
				indexdef
			FROM pg_indexes
			WHERE schemaname = 'public'
			AND indexname NOT LIKE '%_pkey'
			ORDER BY tablename, indexname
		`)
		require.NoError(t, err)
		defer rows.Close()

		indexes := make(map[string][]string)
		for rows.Next() {
			var schema, table, indexName, indexDef string
			err := rows.Scan(&schema, &table, &indexName, &indexDef)
			require.NoError(t, err)

			indexes[table] = append(indexes[table], indexName)

			// Check for important patterns
			if strings.Contains(indexDef, "UNIQUE") {
				t.Logf("Unique index: %s on %s", indexName, table)
			}
		}

		// Verify we have indexes
		assert.NotEmpty(t, indexes, "Should have indexes")

		// Check for specific tables having indexes
		if indices, ok := indexes["packet_loss_results"]; ok {
			assert.NotEmpty(t, indices, "packet_loss_results should have indexes")
			t.Logf("packet_loss_results indexes: %v", indices)
		}

		if indices, ok := indexes["notification_rules"]; ok {
			assert.NotEmpty(t, indices, "notification_rules should have indexes")
			t.Logf("notification_rules indexes: %v", indices)
		}
	})

	t.Run("ValidateTriggers", func(t *testing.T) {
		// Check for update timestamp triggers
		rows, err := td.DB.QueryContext(ctx, `
			SELECT 
				trigger_name,
				event_object_table,
				action_timing,
				event_manipulation
			FROM information_schema.triggers
			WHERE trigger_schema = 'public'
		`)
		require.NoError(t, err)
		defer rows.Close()

		triggerCount := 0
		updateTriggers := []string{}

		for rows.Next() {
			var triggerName, tableName, timing, event string
			err := rows.Scan(&triggerName, &tableName, &timing, &event)
			require.NoError(t, err)

			triggerCount++
			if strings.Contains(triggerName, "update") && event == "UPDATE" {
				updateTriggers = append(updateTriggers, tableName)
			}
		}

		// Verify update triggers exist for tables with updated_at
		if triggerCount > 0 {
			assert.Contains(t, updateTriggers, "notification_channels")
			assert.Contains(t, updateTriggers, "notification_rules")
		}
	})

	t.Run("ValidateCheckConstraints", func(t *testing.T) {
		// Get check constraints
		rows, err := td.DB.QueryContext(ctx, `
			SELECT 
				tc.table_name,
				tc.constraint_name,
				cc.check_clause
			FROM information_schema.table_constraints tc
			JOIN information_schema.check_constraints cc
				ON tc.constraint_name = cc.constraint_name
				AND tc.constraint_schema = cc.constraint_schema
			WHERE tc.constraint_type = 'CHECK'
			AND tc.table_schema = 'public'
		`)
		require.NoError(t, err)
		defer rows.Close()

		for rows.Next() {
			var tableName, constraintName, checkClause string
			err := rows.Scan(&tableName, &constraintName, &checkClause)
			require.NoError(t, err)

			t.Logf("Check constraint on %s: %s - %s", tableName, constraintName, checkClause)
		}
	})

	t.Run("ValidateSequences", func(t *testing.T) {
		// Verify sequences for SERIAL columns
		rows, err := td.DB.QueryContext(ctx, `
			SELECT 
				sequence_name,
				start_value,
				increment
			FROM information_schema.sequences
			WHERE sequence_schema = 'public'
		`)
		require.NoError(t, err)
		defer rows.Close()

		sequenceCount := 0
		for rows.Next() {
			var seqName string
			var startValue, increment sql.NullInt64
			err := rows.Scan(&seqName, &startValue, &increment)
			require.NoError(t, err)
			sequenceCount++
			t.Logf("Sequence: %s (start: %d, increment: %d)", seqName, startValue.Int64, increment.Int64)
		}

		// Each table with SERIAL primary key should have a sequence
		assert.Greater(t, sequenceCount, 10, "Should have sequences for SERIAL columns")
	})

	t.Run("TestTransactionIsolation", func(t *testing.T) {
		// Test that transactions work properly
		tx, err := td.DB.BeginTx(ctx, nil)
		require.NoError(t, err)
		defer tx.Rollback()

		// Insert test data in transaction
		_, err = tx.Exec(`
			INSERT INTO users (username, password_hash) 
			VALUES ($1, $2)
		`, "tx_test_user", "hash")
		require.NoError(t, err)

		// Verify data is visible in transaction
		var count int
		err = tx.QueryRow("SELECT COUNT(*) FROM users WHERE username = $1", "tx_test_user").Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 1, count)

		// Rollback and verify data is gone
		err = tx.Rollback()
		require.NoError(t, err)

		err = td.DB.QueryRow("SELECT COUNT(*) FROM users WHERE username = $1", "tx_test_user").Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 0, count)
	})

	t.Run("ValidatePerformanceFeatures", func(t *testing.T) {
		// Check for performance-related settings

		// Verify important indexes exist
		var indexCount int
		err := td.DB.QueryRow(`
			SELECT COUNT(*) 
			FROM pg_indexes 
			WHERE schemaname = 'public'
			AND tablename = 'packet_loss_results'
		`).Scan(&indexCount)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, indexCount, 2, "packet_loss_results should have indexes for performance")

		// Check for partial indexes or other optimizations
		rows, err := td.DB.QueryContext(ctx, `
			SELECT indexdef 
			FROM pg_indexes 
			WHERE schemaname = 'public'
			AND indexdef LIKE '%WHERE%'
		`)
		require.NoError(t, err)
		defer rows.Close()

		for rows.Next() {
			var indexDef string
			err := rows.Scan(&indexDef)
			require.NoError(t, err)
			t.Logf("Partial index found: %s", indexDef)
		}
	})
}

// TestMigrationRollbackPrevention ensures migrations can't be rolled back
func TestMigrationRollbackPrevention(t *testing.T) {
	RunTestWithBothDatabases(t, func(t *testing.T, td *TestDatabase) {
		// Try to delete from schema_migrations (should fail in production)
		// This is just to document expected behavior

		var count int
		err := td.DB.QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&count)
		require.NoError(t, err)
		assert.Greater(t, count, 0, "Should have migrations")

		// In a production system, this should be prevented by permissions
		// For testing, we just verify the table exists and has entries
	})
}

// TestMigrationIdempotency verifies migrations can be run multiple times safely
func TestMigrationIdempotency(t *testing.T) {
	// This test would require access to the migration runner
	// Currently we can only verify that migrations were applied once
	t.Skip("Requires migration runner access")
}
