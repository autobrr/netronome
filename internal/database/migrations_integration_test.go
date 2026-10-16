// Copyright (c) 2024-2026, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package database

import (
	"database/sql"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/autobrr/netronome/internal/config"
	"github.com/autobrr/netronome/internal/database/migrations"
	"github.com/autobrr/netronome/pkg/migrator"
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

func TestMigrations_UpgradeSetsThresholdOperators(t *testing.T) {
	RunTestWithBothDatabases(t, func(t *testing.T, td *TestDatabase) {
		db := openEmptyDatabase(t, td.Config)

		files, err := migrations.GetMigrationFiles(migrations.DatabaseType(td.Config.Type))
		require.NoError(t, err)
		first := slices.IndexFunc(files, func(f string) bool { return strings.HasPrefix(path.Base(f), "025_") })
		require.Positive(t, first)

		migrate := func(files []string) {
			m := migrator.NewMigrate(db, migrator.WithEmbedFS(migrations.SchemaMigrations))
			for _, f := range files {
				m.Add(&migrator.Migration{Name: f, File: f})
			}
			require.NoError(t, m.Migrate())
		}

		migrate(files[:first])

		rule := func(channelID int, eventType, operator string) string {
			return fmt.Sprintf(`INSERT INTO notification_rules (channel_id, event_id, threshold_value, threshold_operator)
				SELECT %d, id, 500, %s FROM notification_events WHERE category = 'speedtest' AND event_type = '%s'`,
				channelID, operator, eventType)
		}
		for _, stmt := range []string{
			`INSERT INTO notification_channels (id, name, url) VALUES (1, 'first', 'generic://example.invalid'), (2, 'second', 'generic://example.invalid')`,
			rule(1, NotificationEventSpeedtestDownloadLow, "'gt'"),
			rule(1, NotificationEventSpeedtestUploadLow, "'lte'"),
			rule(1, NotificationEventSpeedtestPingHigh, "'gt'"),
			rule(2, NotificationEventSpeedtestDownloadLow, "NULL"),
			rule(2, NotificationEventSpeedtestPingHigh, "NULL"),
		} {
			_, err := db.Exec(stmt)
			require.NoError(t, err)
		}

		migrate(files)

		rows, err := db.Query(`SELECT r.channel_id, e.event_type, r.threshold_operator
			FROM notification_rules r JOIN notification_events e ON e.id = r.event_id`)
		require.NoError(t, err)
		defer rows.Close()

		got := map[string]*string{}
		for rows.Next() {
			var channelID int
			var eventType string
			var operator *string
			require.NoError(t, rows.Scan(&channelID, &eventType, &operator))
			got[fmt.Sprintf("%d/%s", channelID, eventType)] = operator
		}
		require.NoError(t, rows.Err())

		assert.Equal(t, map[string]*string{
			"1/download_low": new("lt"),
			"1/upload_low":   new("lte"),
			"1/ping_high":    new("gt"),
			"2/download_low": new("lt"),
			"2/ping_high":    new("gt"),
		}, got)
	})
}

// openEmptyDatabase opens a database with no tables, of the same type as cfg.
func openEmptyDatabase(t *testing.T, cfg config.DatabaseConfig) *sql.DB {
	t.Helper()

	if cfg.Type == config.SQLite {
		db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "upgrade.db"))
		require.NoError(t, err)
		t.Cleanup(func() { db.Close() })
		return db
	}

	admin, err := sql.Open("postgres", postgresDSN(cfg, cfg.DBName))
	require.NoError(t, err)
	t.Cleanup(func() { admin.Close() })

	_, err = admin.Exec("CREATE DATABASE upgrade_test")
	require.NoError(t, err)
	t.Cleanup(func() { admin.Exec("DROP DATABASE upgrade_test") })

	db, err := sql.Open("postgres", postgresDSN(cfg, "upgrade_test"))
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	return db
}

func postgresDSN(cfg config.DatabaseConfig, dbName string) string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, dbName, cfg.SSLMode)
}
