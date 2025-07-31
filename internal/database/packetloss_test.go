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

func TestSavePacketLossResult_PostgreSQL(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	s := &service{
		db:         db,
		config:     config.DatabaseConfig{Type: config.Postgres},
		sqlBuilder: sq.StatementBuilder.PlaceholderFormat(sq.Dollar),
	}

	result := &types.PacketLossResult{
		MonitorID:      1,
		PacketLoss:     5.5,
		MinRTT:         10.1,
		MaxRTT:         50.5,
		AvgRTT:         25.3,
		StdDevRTT:      5.2,
		PacketsSent:    100,
		PacketsRecv:    95,
		UsedMTR:        true,
		HopCount:       10,
		MTRData:        stringPtr(`{"hops":[]}`),
		PrivilegedMode: true,
		CreatedAt:      time.Now(),
	}

	// Test the new code with RETURNING clause
	mock.ExpectQuery(`INSERT INTO packet_loss_results`).
		WithArgs(
			result.MonitorID,
			result.PacketLoss,
			result.MinRTT,
			result.MaxRTT,
			result.AvgRTT,
			result.StdDevRTT,
			result.PacketsSent,
			result.PacketsRecv,
			result.UsedMTR,
			result.HopCount,
			result.MTRData,
			result.PrivilegedMode,
			result.CreatedAt,
		).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(42))

	err = s.SavePacketLossResult(result)
	assert.NoError(t, err)
	assert.Equal(t, int64(42), result.ID)
	
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSavePacketLossResult_SQLite(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	s := &service{
		db:         db,
		config:     config.DatabaseConfig{Type: config.SQLite},
		sqlBuilder: sq.StatementBuilder,
	}

	result := &types.PacketLossResult{
		MonitorID:      1,
		PacketLoss:     5.5,
		MinRTT:         10.1,
		MaxRTT:         50.5,
		AvgRTT:         25.3,
		StdDevRTT:      5.2,
		PacketsSent:    100,
		PacketsRecv:    95,
		UsedMTR:        true,
		HopCount:       10,
		MTRData:        stringPtr(`{"hops":[]}`),
		PrivilegedMode: true,
		CreatedAt:      time.Now(),
	}

	// Test SQLite with LastInsertId
	mock.ExpectExec(`INSERT INTO packet_loss_results`).
		WithArgs(
			result.MonitorID,
			result.PacketLoss,
			result.MinRTT,
			result.MaxRTT,
			result.AvgRTT,
			result.StdDevRTT,
			result.PacketsSent,
			result.PacketsRecv,
			result.UsedMTR,
			result.HopCount,
			result.MTRData,
			result.PrivilegedMode,
			result.CreatedAt,
		).
		WillReturnResult(sqlmock.NewResult(42, 1))

	err = s.SavePacketLossResult(result)
	assert.NoError(t, err)
	assert.Equal(t, int64(42), result.ID)
	
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestSavePacketLossResult_OldCode_PostgreSQLFailure demonstrates how the old code would fail
func TestSavePacketLossResult_OldCode_PostgreSQLFailure(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// Simulate the old code behavior (without database type check)
	result := &types.PacketLossResult{
		MonitorID:   1,
		PacketLoss:  5.5,
	}

	// Mock PostgreSQL driver behavior - Exec returns a result that doesn't support LastInsertId
	mock.ExpectExec(`INSERT INTO packet_loss_results`).
		WillReturnResult(mockPostgresResult{rowsAffected: 1})

	// This simulates what the old code would have done
	res, err := db.Exec(`INSERT INTO packet_loss_results (monitor_id, packet_loss) VALUES (?, ?)`, 
		result.MonitorID, result.PacketLoss)
	require.NoError(t, err)

	// This is where the old code would fail with PostgreSQL
	_, err = res.LastInsertId()
	assert.Error(t, err)
	assert.Equal(t, "LastInsertId is not supported by this driver", err.Error())
}

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
		MonitorID:  1,
		CreatedAt:  time.Now(),
	}

	err = s.SavePacketLossResult(result)
	assert.Error(t, err)
	assert.Equal(t, "unsupported database type: unsupported", err.Error())
}