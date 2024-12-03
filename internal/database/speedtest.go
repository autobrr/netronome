// Copyright (c) 2024, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package database

import (
	"context"
	"fmt"

	sq "github.com/Masterminds/squirrel"

	"github.com/autobrr/netronome/internal/types"
)

func (s *service) SaveSpeedTest(ctx context.Context, result types.SpeedTestResult) (*types.SpeedTestResult, error) {
	query := s.sqlBuilder.
		Insert("speed_tests").
		Columns(
			"server_name",
			"server_id",
			"test_type",
			"download_speed",
			"upload_speed",
			"latency",
			"packet_loss",
			"jitter",
			"is_scheduled",
			"created_at",
		).
		Values(
			result.ServerName,
			result.ServerID,
			result.TestType,
			result.DownloadSpeed,
			result.UploadSpeed,
			result.Latency,
			result.PacketLoss,
			result.Jitter,
			result.IsScheduled,
			sq.Expr("CURRENT_TIMESTAMP"),
		).
		Suffix("RETURNING id, created_at")

	err := query.RunWith(s.db).QueryRowContext(ctx).Scan(&result.ID, &result.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to save speed test result: %w", err)
	}

	return &result, nil
}

func (s *service) GetSpeedTests(ctx context.Context, timeRange string, page, limit int) (*types.PaginatedSpeedTests, error) {
	baseQuery := s.sqlBuilder.Select().From("speed_tests")

	var whereClause string
	switch s.config.Type {
	case Postgres:
		switch timeRange {
		case "1d":
			whereClause = "created_at >= NOW() - INTERVAL '1 day'"
		case "3d":
			whereClause = "created_at >= NOW() - INTERVAL '3 days'"
		case "1w":
			whereClause = "created_at >= NOW() - INTERVAL '7 days'"
		case "1m":
			whereClause = "created_at >= NOW() - INTERVAL '1 month'"
		}
	case SQLite:
		switch timeRange {
		case "1d":
			whereClause = "created_at >= datetime('now', '-1 day')"
		case "3d":
			whereClause = "created_at >= datetime('now', '-3 days')"
		case "1w":
			whereClause = "created_at >= datetime('now', '-7 days')"
		case "1m":
			whereClause = "created_at >= datetime('now', '-1 month')"
		}
	}

	if whereClause != "" {
		baseQuery = baseQuery.Where(whereClause)
	}

	// Count query
	countQuery := s.sqlBuilder.Select("COUNT(*)").From("speed_tests")
	if whereClause != "" {
		countQuery = countQuery.Where(whereClause)
	}

	var total int
	err := countQuery.RunWith(s.db).QueryRowContext(ctx).Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("failed to get total count: %w", err)
	}

	// Data query with pagination
	dataQuery := baseQuery.
		Columns(
			"id",
			"server_name",
			"server_id",
			"test_type",
			"download_speed",
			"upload_speed",
			"latency",
			"packet_loss",
			"jitter",
			"is_scheduled",
			"created_at",
		).
		OrderBy("created_at DESC").
		Limit(uint64(limit)).
		Offset(uint64((page - 1) * limit))

	rows, err := dataQuery.RunWith(s.db).QueryContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to query speed tests: %w", err)
	}
	defer rows.Close()

	var results []types.SpeedTestResult
	for rows.Next() {
		var result types.SpeedTestResult
		err := rows.Scan(
			&result.ID,
			&result.ServerName,
			&result.ServerID,
			&result.TestType,
			&result.DownloadSpeed,
			&result.UploadSpeed,
			&result.Latency,
			&result.PacketLoss,
			&result.Jitter,
			&result.IsScheduled,
			&result.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan speed test result: %w", err)
		}
		results = append(results, result)
	}

	return &types.PaginatedSpeedTests{
		Data:  results,
		Total: total,
		Page:  page,
		Limit: limit,
	}, nil
}
