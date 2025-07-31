// Copyright (c) 2024-2025, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package database

import (
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	sq "github.com/Masterminds/squirrel"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/autobrr/netronome/internal/config"
	"github.com/autobrr/netronome/internal/types"
)

// stringPtr returns a pointer to a string
func stringPtr(s string) *string {
	return &s
}

// Unit tests for database-specific behavior and error handling
// For comprehensive integration tests, see packetloss_integration_test.go

// mockResult simulates a PostgreSQL driver result that doesn't support LastInsertId
type mockPostgresResult struct {
	rowsAffected int64
}

func (m mockPostgresResult) LastInsertId() (int64, error) {
	return 0, errors.New("LastInsertId is not supported by this driver")
}

func (m mockPostgresResult) RowsAffected() (int64, error) {
	return m.rowsAffected, nil
}

// TestSavePacketLossResult_PostgreSQLReturning verifies the RETURNING clause behavior
func TestSavePacketLossResult_PostgreSQLReturning(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	s := &service{
		db:         db,
		config:     config.DatabaseConfig{Type: config.Postgres},
		sqlBuilder: sq.StatementBuilder.PlaceholderFormat(sq.Dollar),
	}

	result := &types.PacketLossResult{
		MonitorID:  1,
		PacketLoss: 5.5,
		CreatedAt:  time.Now(),
	}

	// Verify that PostgreSQL uses RETURNING clause
	mock.ExpectQuery(`INSERT INTO packet_loss_results .+ RETURNING id`).
		WithArgs(
			result.MonitorID,
			result.PacketLoss,
			sqlmock.AnyArg(), // MinRTT
			sqlmock.AnyArg(), // MaxRTT
			sqlmock.AnyArg(), // AvgRTT
			sqlmock.AnyArg(), // StdDevRTT
			sqlmock.AnyArg(), // PacketsSent
			sqlmock.AnyArg(), // PacketsRecv
			sqlmock.AnyArg(), // UsedMTR
			sqlmock.AnyArg(), // HopCount
			sqlmock.AnyArg(), // MTRData
			sqlmock.AnyArg(), // PrivilegedMode
			result.CreatedAt,
		).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(42))

	err = s.SavePacketLossResult(result)
	assert.NoError(t, err)
	assert.Equal(t, int64(42), result.ID)

	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestSavePacketLossResult_SQLiteLastInsertId verifies LastInsertId behavior
func TestSavePacketLossResult_SQLiteLastInsertId(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	s := &service{
		db:         db,
		config:     config.DatabaseConfig{Type: config.SQLite},
		sqlBuilder: sq.StatementBuilder,
	}

	result := &types.PacketLossResult{
		MonitorID:  1,
		PacketLoss: 5.5,
		CreatedAt:  time.Now(),
	}

	// Verify that SQLite uses LastInsertId
	mock.ExpectExec(`INSERT INTO packet_loss_results`).
		WithArgs(
			sqlmock.AnyArg(), // MonitorID
			sqlmock.AnyArg(), // PacketLoss
			sqlmock.AnyArg(), // MinRTT
			sqlmock.AnyArg(), // MaxRTT
			sqlmock.AnyArg(), // AvgRTT
			sqlmock.AnyArg(), // StdDevRTT
			sqlmock.AnyArg(), // PacketsSent
			sqlmock.AnyArg(), // PacketsRecv
			sqlmock.AnyArg(), // UsedMTR
			sqlmock.AnyArg(), // HopCount
			sqlmock.AnyArg(), // MTRData
			sqlmock.AnyArg(), // PrivilegedMode
			sqlmock.AnyArg(), // CreatedAt
		).
		WillReturnResult(sqlmock.NewResult(42, 1))

	err = s.SavePacketLossResult(result)
	assert.NoError(t, err)
	assert.Equal(t, int64(42), result.ID)

	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestPostgreSQLDriverLastInsertIdError demonstrates PostgreSQL driver behavior
func TestPostgreSQLDriverLastInsertIdError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// Mock PostgreSQL driver behavior - Exec returns a result that doesn't support LastInsertId
	mock.ExpectExec(`INSERT INTO test_table`).
		WillReturnResult(mockPostgresResult{rowsAffected: 1})

	// Execute a query that would typically need LastInsertId
	res, err := db.Exec(`INSERT INTO test_table (col) VALUES (?)`, "value")
	require.NoError(t, err)

	// Verify that LastInsertId fails as expected for PostgreSQL
	_, err = res.LastInsertId()
	assert.Error(t, err)
	assert.Equal(t, "LastInsertId is not supported by this driver", err.Error())
}

// TestSavePacketLossResult_UnsupportedDatabase verifies error handling for unsupported DB types
func TestSavePacketLossResult_UnsupportedDatabase(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	s := &service{
		db:         db,
		config:     config.DatabaseConfig{Type: "unsupported"},
		sqlBuilder: sq.StatementBuilder,
	}

	result := &types.PacketLossResult{
		MonitorID: 1,
		CreatedAt: time.Now(),
	}

	err = s.SavePacketLossResult(result)
	assert.Error(t, err)
	assert.Equal(t, "unsupported database type: unsupported", err.Error())
}

// TestSavePacketLossResult_QueryError verifies error handling when query fails
func TestSavePacketLossResult_QueryError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	s := &service{
		db:         db,
		config:     config.DatabaseConfig{Type: config.Postgres},
		sqlBuilder: sq.StatementBuilder.PlaceholderFormat(sq.Dollar),
	}

	result := &types.PacketLossResult{
		MonitorID: 1,
		CreatedAt: time.Now(),
	}

	// Simulate query error
	mock.ExpectQuery(`INSERT INTO packet_loss_results`).
		WillReturnError(errors.New("database connection lost"))

	err = s.SavePacketLossResult(result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database connection lost")
}
