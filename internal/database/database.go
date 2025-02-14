// Copyright (c) 2024-2025, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package database

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	sq "github.com/Masterminds/squirrel"
	_ "github.com/joho/godotenv/autoload"
	_ "github.com/lib/pq"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	_ "modernc.org/sqlite"

	"github.com/autobrr/netronome/internal/config"
	"github.com/autobrr/netronome/internal/database/migrations"
	"github.com/autobrr/netronome/internal/types"
	"github.com/autobrr/netronome/pkg/migrator"
)

// Common errors
var (
	ErrNotFound     = fmt.Errorf("record not found")
	ErrInvalidInput = fmt.Errorf("invalid input")
)

// ZerologAdapter adapts zerolog.Logger to migrator.Logger
type ZerologAdapter struct {
	logger zerolog.Logger
}

func (z *ZerologAdapter) Printf(format string, args ...interface{}) {
	z.logger.Info().Msgf(format, args...)
}

// Service represents the core database functionality
type Service interface {
	Health() map[string]string
	Close() error
	InitializeTables(ctx context.Context) error
	QueryRow(ctx context.Context, query string, args ...interface{}) *sql.Row

	// User operations
	CreateUser(ctx context.Context, username, password string) (*User, error)
	GetUserByUsername(ctx context.Context, username string) (*User, error)
	ValidatePassword(user *User, password string) bool

	// SpeedTest operations
	SaveSpeedTest(ctx context.Context, result types.SpeedTestResult) (*types.SpeedTestResult, error)
	GetSpeedTests(ctx context.Context, timeRange string, page int, limit int) (*types.PaginatedSpeedTests, error)

	// Schedule operations
	CreateSchedule(ctx context.Context, schedule types.Schedule) (*types.Schedule, error)
	GetSchedules(ctx context.Context) ([]types.Schedule, error)
	UpdateSchedule(ctx context.Context, schedule types.Schedule) error
	DeleteSchedule(ctx context.Context, id int64) error

	// IPerf operations
	SaveIperfServer(ctx context.Context, name, host string, port int) (*types.SavedIperfServer, error)
	GetIperfServers(ctx context.Context) ([]types.SavedIperfServer, error)
	DeleteIperfServer(ctx context.Context, id int) error
}

type service struct {
	db         *sql.DB
	config     config.DatabaseConfig
	sqlBuilder sq.StatementBuilderType
}

// Common query building methods
func (s *service) insert(ctx context.Context, table string, data map[string]interface{}) (sql.Result, error) {
	cols := make([]string, 0, len(data))
	vals := make([]interface{}, 0, len(data))

	for col, val := range data {
		cols = append(cols, col)
		vals = append(vals, val)
	}

	query := s.sqlBuilder.
		Insert(table).
		Columns(cols...).
		Values(vals...)

	return query.RunWith(s.db).ExecContext(ctx)
}

func (s *service) update(ctx context.Context, table string, data map[string]interface{}, where sq.Eq) (sql.Result, error) {
	query := s.sqlBuilder.Update(table)

	for col, val := range data {
		query = query.Set(col, val)
	}

	query = query.Where(where)
	return query.RunWith(s.db).ExecContext(ctx)
}

func (s *service) delete(ctx context.Context, table string, where sq.Eq) (sql.Result, error) {
	query := s.sqlBuilder.
		Delete(table).
		Where(where)

	return query.RunWith(s.db).ExecContext(ctx)
}

func (s *service) select_(ctx context.Context, table string, columns []string, where sq.Eq) (*sql.Rows, error) {
	query := s.sqlBuilder.
		Select(columns...).
		From(table).
		Where(where)

	return query.RunWith(s.db).QueryContext(ctx)
}

func (s *service) count(ctx context.Context, table string, where sq.Eq) (int, error) {
	query := s.sqlBuilder.
		Select("COUNT(*)").
		From(table).
		Where(where)

	var count int
	err := query.RunWith(s.db).QueryRowContext(ctx).Scan(&count)
	return count, err
}

func (s *service) QueryRow(ctx context.Context, query string, args ...interface{}) *sql.Row {
	return s.db.QueryRowContext(ctx, query, args...)
}

var dbInstance *service

func New(cfg config.DatabaseConfig) Service {
	if dbInstance != nil {
		return dbInstance
	}

	var db *sql.DB
	var err error
	var builder sq.StatementBuilderType

	switch cfg.Type {
	case config.Postgres:
		dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
			cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName, cfg.SSLMode)
		db, err = sql.Open("postgres", dsn)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to open PostgreSQL database")
		}
		builder = sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
	case config.SQLite:
		absPath, err := filepath.Abs(cfg.Path)
		if err != nil {
			log.Fatal().Err(err).Str("path", cfg.Path).Msg("Failed to get absolute database path")
		}
		cfg.Path = absPath
		dbDir := filepath.Dir(cfg.Path)

		if err := os.MkdirAll(dbDir, 0o755); err != nil {
			log.Fatal().Err(err).Str("path", dbDir).Msg("Failed to create database directory")
		}

		if err := os.Chmod(dbDir, 0o755); err != nil {
			log.Fatal().Err(err).Msg("Failed to set database directory permissions")
		}

		db, err = sql.Open("sqlite", fmt.Sprintf("file:%s?cache=shared&mode=rwc", cfg.Path))
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to open SQLite database")
		}

		if err := initializeSQLite(db); err != nil {
			log.Fatal().Err(err).Msg("Failed to initialize SQLite database")
		}

		if err := os.Chmod(cfg.Path, 0o640); err != nil {
			log.Fatal().Err(err).Msg("Failed to set database file permissions")
		}
		builder = sq.StatementBuilder.PlaceholderFormat(sq.Question)
	}

	builder = builder.RunWith(db)

	dbInstance = &service{
		db:         db,
		config:     cfg,
		sqlBuilder: builder,
	}

	return dbInstance
}

func initializeSQLite(db *sql.DB) error {
	if err := db.Ping(); err != nil {
		return fmt.Errorf("failed to create database file: %w", err)
	}

	pragmas := []string{
		"PRAGMA busy_timeout = 5000",
		"PRAGMA journal_mode = wal",
		"PRAGMA analysis_limit = 400",
		"PRAGMA wal_checkpoint(TRUNCATE)",
		"PRAGMA foreign_keys = ON",
	}

	for _, pragma := range pragmas {
		if _, err := db.Exec(pragma); err != nil {
			return fmt.Errorf("failed to execute %s: %w", pragma, err)
		}
	}

	return nil
}

func (s *service) Health() map[string]string {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	stats := make(map[string]string)

	if err := s.db.PingContext(ctx); err != nil {
		stats["status"] = "down"
		stats["error"] = fmt.Sprintf("db down: %v", err)
		log.Error().Err(err).Msg("Database is down")
		return stats
	}

	stats["status"] = "up"
	stats["message"] = "It's healthy"
	stats["type"] = string(s.config.Type)

	dbStats := s.db.Stats()
	stats["open_connections"] = strconv.Itoa(dbStats.OpenConnections)
	stats["in_use"] = strconv.Itoa(dbStats.InUse)
	stats["idle"] = strconv.Itoa(dbStats.Idle)
	stats["wait_count"] = strconv.FormatInt(dbStats.WaitCount, 10)
	stats["wait_duration"] = dbStats.WaitDuration.String()
	stats["max_idle_closed"] = strconv.FormatInt(dbStats.MaxIdleClosed, 10)
	stats["max_lifetime_closed"] = strconv.FormatInt(dbStats.MaxLifetimeClosed, 10)

	s.evaluateHealthStats(dbStats, stats)

	return stats
}

func (s *service) evaluateHealthStats(dbStats sql.DBStats, stats map[string]string) {
	if dbStats.OpenConnections > 40 {
		stats["message"] = "The database is experiencing heavy load."
	}
	if dbStats.WaitCount > 1000 {
		stats["message"] = "The database has a high number of wait events, indicating potential bottlenecks."
	}
	if dbStats.MaxIdleClosed > int64(dbStats.OpenConnections)/2 {
		stats["message"] = "Many idle connections are being closed, consider revising the connection pool settings."
	}
	if dbStats.MaxLifetimeClosed > int64(dbStats.OpenConnections)/2 {
		stats["message"] = "Many connections are being closed due to max lifetime, consider increasing max lifetime."
	}
}

func (s *service) Close() error {
	log.Info().
		Str("type", string(s.config.Type)).
		Msg("Disconnected from database")
	return s.db.Close()
}

func getMigrationVersion(fileName string) int {
	parts := strings.Split(fileName, "/")
	if len(parts) > 0 {
		fileName = parts[len(parts)-1]
	}

	parts = strings.Split(fileName, "_")
	if len(parts) > 0 {
		if v, err := strconv.Atoi(parts[0]); err == nil {
			return v
		}
	}
	return 0
}

func (s *service) InitializeTables(ctx context.Context) error {
	logger := &ZerologAdapter{logger: log.Logger}
	m := migrator.NewMigrate(s.db,
		migrator.WithLogger(logger),
		migrator.WithEmbedFS(migrations.SchemaMigrations),
	)

	migrationFiles, err := migrations.GetMigrationFiles(migrations.DatabaseType(s.config.Type))
	if err != nil {
		return fmt.Errorf("failed to get migration files: %w", err)
	}

	// log.Trace().Interface("migration_files", migrationFiles).Msg("Found migration files")

	for _, fileName := range migrationFiles {
		// version := getMigrationVersion(fileName)
		// log.Trace().Str("file", fileName).Int("version", version).Msg("Adding migration")
		m.Add(&migrator.Migration{
			Name: fileName,
			File: fileName,
		})
	}

	if err := m.Migrate(); err != nil {
		return fmt.Errorf("failed to apply migrations: %w", err)
	}

	return nil
}
