// Copyright (c) 2024, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package database

import (
	"context"
	"fmt"

	sq "github.com/Masterminds/squirrel"

	"github.com/autobrr/netronome/internal/config"
	"github.com/autobrr/netronome/internal/types"
)

func (s *service) SaveSpeedTest(ctx context.Context, result types.SpeedTestResult) (*types.SpeedTestResult, error) {
	data := map[string]interface{}{
		"server_name":    result.ServerName,
		"server_id":      result.ServerID,
		"test_type":      result.TestType,
		"download_speed": result.DownloadSpeed,
		"upload_speed":   result.UploadSpeed,
		"latency":        result.Latency,
		"packet_loss":    result.PacketLoss,
		"jitter":         result.Jitter,
		"is_scheduled":   result.IsScheduled,
		"created_at":     sq.Expr("CURRENT_TIMESTAMP"),
	}

	var id int64

	switch s.config.Type {
	case config.Postgres:
		query := s.sqlBuilder.Insert("speed_tests").
			SetMap(data).
			Suffix("RETURNING id")

		sqlStr, args, err := query.ToSql()
		if err != nil {
			return nil, fmt.Errorf("failed to build query: %w", err)
		}

		err = s.db.QueryRowContext(ctx, sqlStr, args...).Scan(&id)
		if err != nil {
			return nil, fmt.Errorf("failed to save speed test: %w", err)
		}

	case config.SQLite:
		res, err := s.insert(ctx, "speed_tests", data)
		if err != nil {
			return nil, fmt.Errorf("failed to save speed test: %w", err)
		}

		id, err = res.LastInsertId()
		if err != nil {
			return nil, fmt.Errorf("failed to get last insert ID: %w", err)
		}
	}

	result.ID = id
	return &result, nil
}

func (s *service) GetSpeedTests(ctx context.Context, timeRange string, page, limit int) (*types.PaginatedSpeedTests, error) {
	baseQuery := s.sqlBuilder.Select().From("speed_tests")

	if timeRange != "" {
		var timeExpr string
		switch s.config.Type {
		case config.Postgres:
			switch timeRange {
			case "1d":
				timeExpr = "NOW() - INTERVAL '1 day'"
			case "3d":
				timeExpr = "NOW() - INTERVAL '3 days'"
			case "1w":
				timeExpr = "NOW() - INTERVAL '7 days'"
			case "1m":
				timeExpr = "NOW() - INTERVAL '1 month'"
			}
		case config.SQLite:
			switch timeRange {
			case "1d":
				timeExpr = "datetime('now', '-1 day')"
			case "3d":
				timeExpr = "datetime('now', '-3 days')"
			case "1w":
				timeExpr = "datetime('now', '-7 days')"
			case "1m":
				timeExpr = "datetime('now', '-1 month')"
			}
		}
		if timeExpr != "" {
			baseQuery = baseQuery.Where("created_at >= " + timeExpr)
		}
	}

	countQuery := baseQuery.Columns("COUNT(*)")
	var total int
	err := countQuery.RunWith(s.db).QueryRowContext(ctx).Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("failed to get total count: %w", err)
	}

	// Get paginated results
	dataQuery := baseQuery.Columns(
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

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating speed test results: %w", err)
	}

	return &types.PaginatedSpeedTests{
		Data:  results,
		Total: total,
		Page:  page,
		Limit: limit,
	}, nil
}
