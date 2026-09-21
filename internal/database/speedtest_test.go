// Copyright (c) 2024-2026, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package database

import (
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	sq "github.com/Masterminds/squirrel"
	"github.com/stretchr/testify/require"

	"github.com/autobrr/netronome/internal/config"
)

func TestSpeedTest_GetSkipsAliasLookupWithoutLegacyRows(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	service := &service{
		db:         db,
		config:     config.DatabaseConfig{Type: config.SQLite},
		sqlBuilder: sq.StatementBuilder,
	}

	mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) FROM speed_tests")).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, server_name, server_id, server_host, server_city, test_type, download_speed, upload_speed, latency, jitter, is_scheduled, created_at FROM speed_tests ORDER BY created_at DESC LIMIT 10 OFFSET 0")).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "server_name", "server_id", "server_host", "server_city", "test_type",
			"download_speed", "upload_speed", "latency", "jitter", "is_scheduled", "created_at",
		}).AddRow(
			1, "iperf.example.com", "iperf.example.com", "iperf.example.com", nil, "iperf3",
			100.0, 50.0, "10ms", nil, false, time.Now().UTC(),
		))

	results, err := service.GetSpeedTests(t.Context(), "all", 1, 10)
	require.NoError(t, err)
	require.Len(t, results.Data, 1)
	require.NoError(t, mock.ExpectationsWereMet())
}
