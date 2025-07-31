// Copyright (c) 2024-2025, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

// Package database contains database tests and test helpers.
//
// Common Test Patterns:
//
// 1. Time Range Query Testing:
//    When testing time-based queries (24h, week, month), create records at:
//    - now, -23h, -25h (for 24h boundary)
//    - -6d, -8d (for week boundary)
//    - -29d, -32d (for month boundary)
//
// 2. CASCADE DELETE Testing:
//    - Create parent record
//    - Create multiple child records
//    - Delete parent
//    - Verify children are deleted
//
// 3. Unique Constraint Testing:
//    - Insert first record (should succeed)
//    - Insert duplicate (should fail)
//    - Insert different value (should succeed)
//
// These patterns are tested comprehensively in:
// - Time ranges: speedtest_integration_test.go
// - CASCADE: monitor_integration_test.go, packetloss_integration_test.go
// - Constraints: core_integration_test.go
//
// Avoid duplicating these tests in new files.
//
// Performance Tips:
// - Use RunTestWithSQLiteOnly() for simple tests that don't need PostgreSQL
// - Set SKIP_POSTGRES_TESTS=1 to skip PostgreSQL tests during local development
// - PostgreSQL tests take ~6 seconds each due to embedded database initialization

package database

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	"github.com/stretchr/testify/require"

	"github.com/autobrr/netronome/internal/config"
	"github.com/autobrr/netronome/internal/types"
)

// Shared PostgreSQL instance for all tests
var (
	sharedPostgres    *embeddedpostgres.EmbeddedPostgres
	sharedConfig      config.DatabaseConfig
	sharedInitOnce    sync.Once
	sharedInitError   error
	sharedCleanupOnce sync.Once
)

// TestMain manages the shared PostgreSQL instance for the entire test suite
func TestMain(m *testing.M) {
	// Setup shared PostgreSQL instance
	setupSharedPostgreSQL()

	// Run all tests
	code := m.Run()

	// Cleanup shared PostgreSQL instance
	cleanupSharedPostgreSQL()

	os.Exit(code)
}

// setupSharedPostgreSQL initializes the shared PostgreSQL instance once
func setupSharedPostgreSQL() {
	sharedInitOnce.Do(func() {
		// Skip if PostgreSQL tests are disabled
		if os.Getenv("SKIP_POSTGRES_TESTS") != "" {
			log.Println("Skipping PostgreSQL setup (SKIP_POSTGRES_TESTS is set)")
			return
		}

		// Get available port
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			sharedInitError = fmt.Errorf("failed to find available port: %w", err)
			return
		}
		port := listener.Addr().(*net.TCPAddr).Port
		listener.Close()

		// Create temp directory for shared postgres
		tempDir, err := os.MkdirTemp("", "netronome-shared-postgres-*")
		if err != nil {
			sharedInitError = fmt.Errorf("failed to create temp dir: %w", err)
			return
		}

		log.Printf("Starting shared PostgreSQL on port %d", port)

		// Setup embedded PostgreSQL
		postgres := embeddedpostgres.NewDatabase(
			embeddedpostgres.DefaultConfig().
				Username("test").
				Password("test").
				Database("testdb").
				Port(uint32(port)).
				RuntimePath(filepath.Join(tempDir, "postgres-runtime")),
		)

		if err := postgres.Start(); err != nil {
			sharedInitError = fmt.Errorf("failed to start embedded postgres: %w", err)
			return
		}

		sharedPostgres = postgres
		sharedConfig = config.DatabaseConfig{
			Type:     config.Postgres,
			Host:     "localhost",
			Port:     port,
			User:     "test",
			Password: "test",
			DBName:   "testdb",
			SSLMode:  "disable",
		}

		// Initialize schema once
		log.Println("Initializing shared PostgreSQL schema...")
		dbInstance = nil // Reset singleton for clean initialization
		service := New(sharedConfig)
		defer service.Close()

		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		if err := service.InitializeTables(ctx); err != nil {
			sharedInitError = fmt.Errorf("failed to initialize tables: %w", err)
			postgres.Stop()
			sharedPostgres = nil
			return
		}

		log.Println("Shared PostgreSQL ready for tests")
	})
}

// ensureSharedPostgreSQL checks that the shared PostgreSQL instance is ready
func ensureSharedPostgreSQL() error {
	if sharedInitError != nil {
		return sharedInitError
	}
	if sharedPostgres == nil {
		return fmt.Errorf("shared PostgreSQL not initialized")
	}
	return nil
}

// cleanupSharedPostgreSQL stops the shared PostgreSQL instance
func cleanupSharedPostgreSQL() {
	sharedCleanupOnce.Do(func() {
		if sharedPostgres != nil {
			log.Println("Stopping shared PostgreSQL...")
			if err := sharedPostgres.Stop(); err != nil {
				log.Printf("Warning: failed to stop shared PostgreSQL: %v", err)
			}
			sharedPostgres = nil
		}
	})
}

// Tables to clear between tests (in dependency order to handle foreign keys)
var testTablesToClear = []string{
	"notification_history",
	"notification_rules",
	"notification_channels",
	"packet_loss_results",
	"packet_loss_monitors",
	"monitor_historical_snapshots",
	"monitor_resource_stats",
	"monitor_peak_stats",
	"monitor_agent_interfaces",
	"monitor_agent_system_info",
	"monitor_agents",
	"speedtest_results",
	"saved_iperf_servers",
	"users",
	// Keep: notification_events, notification_categories, schema_migrations
}

// cleanTestData removes all test data but keeps schema and seed data
func cleanTestData(t *testing.T, db *sql.DB) {
	t.Helper()

	// Use TRUNCATE CASCADE for speed (PostgreSQL)
	for _, table := range testTablesToClear {
		_, err := db.Exec(fmt.Sprintf("TRUNCATE TABLE %s CASCADE", table))
		if err != nil {
			// Some tables might not exist or have no data, that's ok
			t.Logf("Note: TRUNCATE %s failed (may be empty): %v", table, err)
		}
	}

	// Reset sequences to ensure consistent IDs across tests
	resetSequences(t, db)
}

// resetSequences resets PostgreSQL sequences to start from 1
func resetSequences(t *testing.T, db *sql.DB) {
	t.Helper()

	// Get all sequences in the database
	rows, err := db.Query(`
		SELECT schemaname, sequencename 
		FROM pg_sequences 
		WHERE schemaname = 'public'
	`)
	if err != nil {
		t.Logf("Warning: failed to get sequences: %v", err)
		return
	}
	defer rows.Close()

	var sequences []string
	for rows.Next() {
		var schema, seqName string
		if err := rows.Scan(&schema, &seqName); err != nil {
			continue
		}
		sequences = append(sequences, seqName)
	}

	// Reset each sequence to start from 1
	for _, seq := range sequences {
		_, err := db.Exec(fmt.Sprintf("ALTER SEQUENCE %s RESTART WITH 1", seq))
		if err != nil {
			t.Logf("Warning: failed to reset sequence %s: %v", seq, err)
		}
	}
}


// TestDatabase provides a test database instance
type TestDatabase struct {
	Service  Service
	DB       *sql.DB
	Config   config.DatabaseConfig
	postgres *embeddedpostgres.EmbeddedPostgres
	cleanup  func()
}

// Close cleans up the test database
func (td *TestDatabase) Close() error {
	if td.cleanup != nil {
		td.cleanup()
	}
	if td.Service != nil {
		td.Service.Close()
	}
	if td.postgres != nil {
		return td.postgres.Stop()
	}
	return nil
}

// SetupTestDatabase creates a test database instance
func SetupTestDatabase(t *testing.T, dbType config.DatabaseType) *TestDatabase {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	td := &TestDatabase{}

	switch dbType {
	case config.Postgres:
		// Use shared PostgreSQL instance
		if err := ensureSharedPostgreSQL(); err != nil {
			if os.Getenv("SKIP_POSTGRES_TESTS") != "" {
				t.Skip("PostgreSQL tests skipped (SKIP_POSTGRES_TESTS is set)")
			}
			t.Fatalf("Shared PostgreSQL not available: %v", err)
		}

		td.Config = sharedConfig
		// No postgres field - using shared instance, no individual cleanup needed

	case config.SQLite:
		// Setup SQLite in temporary directory
		tempDir := t.TempDir()
		dbPath := filepath.Join(tempDir, "test.db")

		td.Config = config.DatabaseConfig{
			Type: config.SQLite,
			Path: dbPath,
		}

		td.cleanup = func() {
			os.Remove(dbPath)
		}

	default:
		t.Fatalf("Unsupported database type: %v", dbType)
	}

	// Reset singleton for testing
	dbInstance = nil

	// Create service instance
	td.Service = New(td.Config)

	// Get the underlying *sql.DB for direct access if needed
	if svc, ok := td.Service.(*service); ok {
		td.DB = svc.db
	}

	// Initialize or clean tables based on database type
	if td.Config.Type == config.Postgres {
		// For PostgreSQL, clean existing data instead of initializing fresh tables
		cleanTestData(t, td.DB)
	} else {
		// For SQLite, initialize fresh tables as before
		if err := td.Service.InitializeTables(ctx); err != nil {
			t.Fatalf("Failed to initialize tables: %v", err)
		}
	}

	return td
}

// RunTestWithBothDatabases runs a test function against both SQLite and PostgreSQL
func RunTestWithBothDatabases(t *testing.T, testFunc func(t *testing.T, td *TestDatabase)) {
	t.Run("SQLite", func(t *testing.T) {
		td := SetupTestDatabase(t, config.SQLite)
		defer td.Close()
		testFunc(t, td)
	})

	// Skip PostgreSQL tests if SKIP_POSTGRES_TESTS is set (for faster local development)
	if os.Getenv("SKIP_POSTGRES_TESTS") != "" {
		t.Log("Skipping PostgreSQL tests (SKIP_POSTGRES_TESTS is set)")
		return
	}

	t.Run("PostgreSQL", func(t *testing.T) {
		td := SetupTestDatabase(t, config.Postgres)
		defer td.Close()
		testFunc(t, td)
	})
}

// RunTestWithSQLiteOnly runs a test function against SQLite only (for faster development)
func RunTestWithSQLiteOnly(t *testing.T, testFunc func(t *testing.T, td *TestDatabase)) {
	td := SetupTestDatabase(t, config.SQLite)
	defer td.Close()
	testFunc(t, td)
}

// AssertRecordExists checks if a record exists in the database
func AssertRecordExists(t *testing.T, td *TestDatabase, table string, column string, value any) {
	t.Helper()

	// Build query based on database type
	var query string
	if td.Config.Type == config.Postgres {
		query = fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE %s = $1", table, column)
	} else {
		query = fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE %s = ?", table, column)
	}

	var count int
	err := td.DB.QueryRow(query, value).Scan(&count)
	require.NoError(t, err)
	require.Greater(t, count, 0, "Expected record to exist in %s where %s = %v", table, column, value)
}

// AssertRecordNotExists checks if a record does not exist in the database
func AssertRecordNotExists(t *testing.T, td *TestDatabase, table string, column string, value any) {
	t.Helper()

	// Build query based on database type
	var query string
	if td.Config.Type == config.Postgres {
		query = fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE %s = $1", table, column)
	} else {
		query = fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE %s = ?", table, column)
	}

	var count int
	err := td.DB.QueryRow(query, value).Scan(&count)
	require.NoError(t, err)
	require.Equal(t, 0, count, "Expected no records in %s where %s = %v", table, column, value)
}

// CreateTestUser creates a test user in the database
func CreateTestUser(t *testing.T, td *TestDatabase, username, password string) *User {
	t.Helper()

	ctx := context.Background()
	user, err := td.Service.CreateUser(ctx, username, password)
	require.NoError(t, err)
	require.NotNil(t, user)
	return user
}

// CreateTestPacketLossMonitor creates a test packet loss monitor
func CreateTestPacketLossMonitor(t *testing.T, td *TestDatabase) *types.PacketLossMonitor {
	t.Helper()

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

	result, err := td.Service.CreatePacketLossMonitor(monitor)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Greater(t, result.ID, int64(0))

	return result
}
