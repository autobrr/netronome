// Copyright (c) 2024-2025, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package database

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/autobrr/netronome/internal/config"
)

func TestMigrations_FreshDatabase(t *testing.T) {
	RunTestWithBothDatabases(t, func(t *testing.T, td *TestDatabase) {
		// Get all tables dynamically from the database
		var tables []string

		switch td.Config.Type {
		case config.Postgres:
			rows, err := td.DB.Query(`
				SELECT table_name 
				FROM information_schema.tables 
				WHERE table_schema = 'public' 
				AND table_type = 'BASE TABLE'
				AND table_name != 'schema_migrations'
				ORDER BY table_name
			`)
			require.NoError(t, err)
			defer rows.Close()

			for rows.Next() {
				var tableName string
				err := rows.Scan(&tableName)
				require.NoError(t, err)
				tables = append(tables, tableName)
			}
			require.NoError(t, rows.Err())

		case config.SQLite:
			rows, err := td.DB.Query(`
				SELECT name FROM sqlite_master 
				WHERE type='table' 
				AND name NOT IN ('sqlite_sequence', 'schema_migrations')
				ORDER BY name
			`)
			require.NoError(t, err)
			defer rows.Close()

			for rows.Next() {
				var tableName string
				err := rows.Scan(&tableName)
				require.NoError(t, err)
				tables = append(tables, tableName)
			}
			require.NoError(t, rows.Err())
		}

		// Verify we have a reasonable number of tables
		assert.Greater(t, len(tables), 10, "Should have at least 10 tables after migrations")

		// Log the tables for debugging
		t.Logf("Found %d tables: %v", len(tables), tables)

		// Verify some core tables that should always exist
		coreTablesMap := map[string]bool{
			"users":                false,
			"speed_tests":          false,
			"packet_loss_monitors": false,
			"monitor_agents":       false,
		}

		for _, table := range tables {
			if _, exists := coreTablesMap[table]; exists {
				coreTablesMap[table] = true
			}
		}

		// Check all core tables were found
		for table, found := range coreTablesMap {
			assert.True(t, found, "Core table '%s' should exist", table)
		}
	})
}

func TestMigrations_SchemaVersion(t *testing.T) {
	RunTestWithBothDatabases(t, func(t *testing.T, td *TestDatabase) {
		// Check that schema_migrations table exists and has entries
		var count int
		err := td.DB.QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&count)
		require.NoError(t, err)
		assert.Greater(t, count, 0, "Should have migration entries")

		// Get the latest version string (which is the full filename)
		var latestVersion string
		err = td.DB.QueryRow("SELECT version FROM schema_migrations ORDER BY version DESC LIMIT 1").Scan(&latestVersion)
		require.NoError(t, err)
		assert.NotEmpty(t, latestVersion, "Should have a latest migration version")

		// Log the version format for debugging
		t.Logf("Latest migration version format: %s", latestVersion)

		// Count actual migration files to ensure they all ran
		migrationFiles, err := countMigrationFiles(td.Config.Type)
		require.NoError(t, err)

		// The count in schema_migrations should match the number of migration files
		assert.Equal(t, migrationFiles, count,
			"Number of applied migrations should match number of migration files")

		// Verify the latest version has the expected format
		if td.Config.Type == config.Postgres {
			assert.Contains(t, latestVersion, "_postgres.sql", "PostgreSQL migrations should end with _postgres.sql")
		} else {
			assert.Contains(t, latestVersion, ".sql", "SQLite migrations should end with .sql")
			assert.NotContains(t, latestVersion, "_postgres", "SQLite migrations should not contain _postgres")
		}
	})
}

// Helper function to count migration files
func countMigrationFiles(dbType config.DatabaseType) (int, error) {
	var suffix string
	var dir string

	switch dbType {
	case config.SQLite:
		suffix = ".sql"
		dir = "sqlite"
	case config.Postgres:
		suffix = "_postgres.sql"
		dir = "postgres"
	default:
		return 0, fmt.Errorf("unsupported database type: %v", dbType)
	}

	// Count files in the migrations directory
	entries, err := os.ReadDir(fmt.Sprintf("./migrations/%s", dir))
	if err != nil {
		return 0, err
	}

	count := 0
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), suffix) {
			count++
		}
	}

	return count, nil
}
