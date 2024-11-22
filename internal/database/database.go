package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"strconv"
	"strings"
	"time"

	_ "github.com/joho/godotenv/autoload"
	_ "github.com/mattn/go-sqlite3"
	"github.com/rs/zerolog/log"

	"speedtrackerr/internal/database/migrations"
	"speedtrackerr/internal/types"
)

// Service represents a service that interacts with a database.
type Service interface {
	// Health returns a map of health status information.
	// The keys and values in the map are service-specific.
	Health() map[string]string

	// Close terminates the database connection.
	// It returns an error if the connection cannot be closed.
	Close() error

	// InitializeTables creates necessary database tables
	InitializeTables(ctx context.Context) error

	// SaveSpeedTest stores a new speed test result in the database
	SaveSpeedTest(ctx context.Context, result SpeedTestResult) (*SpeedTestResult, error)

	// GetSpeedTests retrieves speed test results
	GetSpeedTests(ctx context.Context, limit int) ([]SpeedTestResult, error)

	// CreateSchedule creates a new speed test schedule in the database
	CreateSchedule(ctx context.Context, schedule types.Schedule) (*types.Schedule, error)

	// GetSchedules retrieves speed test schedules
	GetSchedules(ctx context.Context) ([]types.Schedule, error)

	// UpdateSchedule updates a speed test schedule in the database
	UpdateSchedule(ctx context.Context, schedule types.Schedule) error

	// DeleteSchedule deletes a speed test schedule from the database
	DeleteSchedule(ctx context.Context, id int64) error
}

type service struct {
	db *sql.DB
}

var (
	dburl      = getDBURL()
	dbInstance *service
)

func getDBURL() string {
	// Try to get from environment variable, fallback to default if not set
	url := os.Getenv("SPEEDTRACKERR_DB_URL")
	if url == "" {
		// Default to a SQLite database in the current directory
		url = "speedtrackerr.db"
	}
	return url
}

func New() Service {
	// Reuse Connection
	if dbInstance != nil {
		return dbInstance
	}

	db, err := sql.Open("sqlite3", dburl)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to open database")
	}

	dbInstance = &service{
		db: db,
	}

	// Initialize tables
	ctx := context.Background()
	if err := dbInstance.InitializeTables(ctx); err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize tables")
	}

	return dbInstance
}

// Health checks the health of the database connection by pinging the database.
// It returns a map with keys indicating various health statistics.
func (s *service) Health() map[string]string {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	stats := make(map[string]string)

	// Ping the database
	err := s.db.PingContext(ctx)
	if err != nil {
		stats["status"] = "down"
		stats["error"] = fmt.Sprintf("db down: %v", err)
		log.Error().Err(err).Msg("Database is down")
		return stats
	}

	// Database is up, add more statistics
	stats["status"] = "up"
	stats["message"] = "It's healthy"

	// Get database stats (like open connections, in use, idle, etc.)
	dbStats := s.db.Stats()
	stats["open_connections"] = strconv.Itoa(dbStats.OpenConnections)
	stats["in_use"] = strconv.Itoa(dbStats.InUse)
	stats["idle"] = strconv.Itoa(dbStats.Idle)
	stats["wait_count"] = strconv.FormatInt(dbStats.WaitCount, 10)
	stats["wait_duration"] = dbStats.WaitDuration.String()
	stats["max_idle_closed"] = strconv.FormatInt(dbStats.MaxIdleClosed, 10)
	stats["max_lifetime_closed"] = strconv.FormatInt(dbStats.MaxLifetimeClosed, 10)

	// Evaluate stats to provide a health message
	if dbStats.OpenConnections > 40 { // Assuming 50 is the max for this example
		stats["message"] = "The database is experiencing heavy load."
	}

	if dbStats.WaitCount > 1000 {
		stats["message"] = "The database has a high number of wait events, indicating potential bottlenecks."
	}

	if dbStats.MaxIdleClosed > int64(dbStats.OpenConnections)/2 {
		stats["message"] = "Many idle connections are being closed, consider revising the connection pool settings."
	}

	if dbStats.MaxLifetimeClosed > int64(dbStats.OpenConnections)/2 {
		stats["message"] = "Many connections are being closed due to max lifetime, consider increasing max lifetime or revising the connection usage pattern."
	}

	return stats
}

// Close closes the database connection.
// It logs a message indicating the disconnection from the specific database.
// If the connection is successfully closed, it returns nil.
// If an error occurs while closing the connection, it returns the error.
func (s *service) Close() error {
	log.Info().Str("url", dburl).Msg("Disconnected from database")
	return s.db.Close()
}

// SpeedTestResult represents a stored speed test result in the database
type SpeedTestResult struct {
	ID            int64     `json:"id"`
	ServerName    string    `json:"serverName"`
	ServerID      string    `json:"serverId"`
	DownloadSpeed float64   `json:"downloadSpeed"`
	UploadSpeed   float64   `json:"uploadSpeed"`
	Latency       string    `json:"latency"`
	PacketLoss    float64   `json:"packetLoss"`
	Jitter        *float64  `json:"jitter"`
	CreatedAt     time.Time `json:"createdAt"`
}

// InitializeTables creates necessary database tables
func (s *service) InitializeTables(ctx context.Context) error {
	// Create schema_migrations table first
	query := `CREATE TABLE IF NOT EXISTS schema_migrations (
		version INTEGER PRIMARY KEY,
		applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);`

	if _, err := s.db.ExecContext(ctx, query); err != nil {
		return fmt.Errorf("failed to create schema_migrations table: %w", err)
	}

	// Get applied migrations
	applied, err := s.getAppliedMigrations(ctx)
	if err != nil {
		return fmt.Errorf("failed to get applied migrations: %w", err)
	}

	// Apply pending migrations
	for _, fileName := range migrations.MigrationFiles {
		version := getMigrationVersion(fileName)
		if !contains(applied, version) {
			if err := s.applyMigration(ctx, fileName, version); err != nil {
				return fmt.Errorf("failed to apply migration %s: %w", fileName, err)
			}
		}
	}

	return nil
}

func (s *service) getAppliedMigrations(ctx context.Context) ([]int, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT version FROM schema_migrations ORDER BY version")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var versions []int
	for rows.Next() {
		var version int
		if err := rows.Scan(&version); err != nil {
			return nil, err
		}
		versions = append(versions, version)
	}
	return versions, nil
}

func (s *service) applyMigration(ctx context.Context, fileName string, version int) error {
	// Read migration file
	content, err := fs.ReadFile(migrations.SchemaMigrations, fileName)
	if err != nil {
		return err
	}

	// Start transaction
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Apply migration
	if _, err := tx.ExecContext(ctx, string(content)); err != nil {
		return err
	}

	// Record migration
	if _, err := tx.ExecContext(ctx,
		"INSERT INTO schema_migrations (version) VALUES (?)",
		version,
	); err != nil {
		return err
	}

	return tx.Commit()
}

func getMigrationVersion(fileName string) int {
	parts := strings.Split(fileName, "_")
	if len(parts) > 0 {
		version, _ := strconv.Atoi(parts[0])
		return version
	}
	return 0
}

func contains(slice []int, item int) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// SaveSpeedTest stores a new speed test result in the database
func (s *service) SaveSpeedTest(ctx context.Context, result SpeedTestResult) (*SpeedTestResult, error) {
	query := `
	INSERT INTO speed_tests (
		server_name, server_id, download_speed, upload_speed, latency, packet_loss, jitter, created_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
	RETURNING id, created_at`

	err := s.db.QueryRowContext(
		ctx,
		query,
		result.ServerName,
		result.ServerID,
		result.DownloadSpeed,
		result.UploadSpeed,
		result.Latency,
		result.PacketLoss,
		result.Jitter,
	).Scan(&result.ID, &result.CreatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to save speed test result: %w", err)
	}

	return &result, nil
}

// GetSpeedTests retrieves speed test results
func (s *service) GetSpeedTests(ctx context.Context, limit int) ([]SpeedTestResult, error) {
	query := `
	SELECT id, server_name, server_id, download_speed, upload_speed, latency, packet_loss, jitter, created_at
	FROM speed_tests
	ORDER BY created_at DESC
	LIMIT ?`

	rows, err := s.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query speed tests: %w", err)
	}
	defer rows.Close()

	var results []SpeedTestResult
	for rows.Next() {
		var result SpeedTestResult
		err := rows.Scan(
			&result.ID,
			&result.ServerName,
			&result.ServerID,
			&result.DownloadSpeed,
			&result.UploadSpeed,
			&result.Latency,
			&result.PacketLoss,
			&result.Jitter,
			&result.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan speed test result: %w", err)
		}
		results = append(results, result)
	}

	return results, nil
}

// CreateSchedule creates a new speed test schedule in the database
func (s *service) CreateSchedule(ctx context.Context, schedule types.Schedule) (*types.Schedule, error) {
	serverIDs, err := json.Marshal(schedule.ServerIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal server IDs: %w", err)
	}

	options, err := json.Marshal(schedule.Options)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal options: %w", err)
	}

	query := `
	INSERT INTO schedules (
		server_ids, interval, next_run, enabled, options, created_at
	) VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
	RETURNING id, created_at`

	err = s.db.QueryRowContext(
		ctx,
		query,
		string(serverIDs),
		schedule.Interval,
		schedule.NextRun,
		schedule.Enabled,
		string(options),
	).Scan(&schedule.ID, &schedule.CreatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to create schedule: %w", err)
	}

	return &schedule, nil
}

// GetSchedules retrieves speed test schedules
func (s *service) GetSchedules(ctx context.Context) ([]types.Schedule, error) {
	query := `SELECT id, server_ids, interval, last_run, next_run, enabled, options, created_at 
			  FROM schedules`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query schedules: %w", err)
	}
	defer rows.Close()

	var schedules []types.Schedule
	for rows.Next() {
		var schedule types.Schedule
		var serverIDsJSON, optionsJSON string
		var lastRun sql.NullTime

		err := rows.Scan(
			&schedule.ID,
			&serverIDsJSON,
			&schedule.Interval,
			&lastRun,
			&schedule.NextRun,
			&schedule.Enabled,
			&optionsJSON,
			&schedule.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan schedule: %w", err)
		}

		if lastRun.Valid {
			schedule.LastRun = &lastRun.Time
		}

		if err := json.Unmarshal([]byte(serverIDsJSON), &schedule.ServerIDs); err != nil {
			return nil, fmt.Errorf("failed to unmarshal server IDs: %w", err)
		}

		if err := json.Unmarshal([]byte(optionsJSON), &schedule.Options); err != nil {
			return nil, fmt.Errorf("failed to unmarshal options: %w", err)
		}

		schedules = append(schedules, schedule)
	}

	return schedules, nil
}

// UpdateSchedule updates a speed test schedule in the database
func (s *service) UpdateSchedule(ctx context.Context, schedule types.Schedule) error {
	serverIDs, err := json.Marshal(schedule.ServerIDs)
	if err != nil {
		return err
	}

	options, err := json.Marshal(schedule.Options)
	if err != nil {
		return err
	}

	query := `
	UPDATE schedules
	SET server_ids = ?, interval = ?, next_run = ?, enabled = ?, options = ?
	WHERE id = ?`

	_, err = s.db.ExecContext(
		ctx,
		query,
		string(serverIDs),
		schedule.Interval,
		schedule.NextRun,
		schedule.Enabled,
		string(options),
		schedule.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update schedule: %w", err)
	}

	return nil
}

// DeleteSchedule deletes a speed test schedule from the database
func (s *service) DeleteSchedule(ctx context.Context, id int64) error {
	query := `
	DELETE FROM schedules
	WHERE id = ?`

	_, err := s.db.ExecContext(
		ctx,
		query,
		id,
	)

	if err != nil {
		return fmt.Errorf("failed to delete schedule: %w", err)
	}

	return nil
}
