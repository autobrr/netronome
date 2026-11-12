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

func TestSpeedTestRangeStart(t *testing.T) {
	endOfMarch := time.Date(2026, 3, 31, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name      string
		timeRange string
		now       time.Time
		want      time.Time
		ok        bool
	}{
		{"24h", "24h", endOfMarch, time.Date(2026, 3, 30, 12, 0, 0, 0, time.UTC), true},
		{"1d", "1d", endOfMarch, time.Date(2026, 3, 30, 12, 0, 0, 0, time.UTC), true},
		{"3d", "3d", endOfMarch, time.Date(2026, 3, 28, 12, 0, 0, 0, time.UTC), true},
		{"week", "week", endOfMarch, time.Date(2026, 3, 24, 12, 0, 0, 0, time.UTC), true},
		{"1w", "1w", endOfMarch, time.Date(2026, 3, 24, 12, 0, 0, 0, time.UTC), true},
		{"month at end of month", "month", endOfMarch, time.Date(2026, 2, 28, 12, 0, 0, 0, time.UTC), true},
		{"1m at end of month in leap year", "1m", time.Date(2028, 3, 31, 12, 0, 0, 0, time.UTC), time.Date(2028, 2, 29, 12, 0, 0, 0, time.UTC), true},
		{"month in mid month", "month", time.Date(2026, 3, 15, 12, 0, 0, 0, time.UTC), time.Date(2026, 2, 15, 12, 0, 0, 0, time.UTC), true},
		{"all", "all", endOfMarch, time.Time{}, false},
		{"unknown", "unknown", endOfMarch, time.Time{}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := speedTestRangeStart(tt.timeRange, tt.now)
			require.Equal(t, tt.ok, ok)
			require.Equal(t, tt.want, got)
		})
	}
}
