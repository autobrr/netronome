package database

import (
	"context"
	"fmt"

	sq "github.com/Masterminds/squirrel"

	"github.com/autobrr/netronome/internal/types"
)

func (s *service) SaveSpeedTest(ctx context.Context, result types.SpeedTestResult) (*types.SpeedTestResult, error) {
	query := sqlBuilder.
		Insert("speed_tests").
		Columns(
			"server_name",
			"server_id",
			"download_speed",
			"upload_speed",
			"latency",
			"packet_loss",
			"jitter",
			"created_at",
		).
		Values(
			result.ServerName,
			result.ServerID,
			result.DownloadSpeed,
			result.UploadSpeed,
			result.Latency,
			result.PacketLoss,
			result.Jitter,
			sq.Expr("CURRENT_TIMESTAMP"),
		).
		Suffix("RETURNING id, created_at")

	err := query.RunWith(s.db).QueryRowContext(ctx).Scan(&result.ID, &result.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to save speed test result: %w", err)
	}

	return &result, nil
}

func (s *service) GetSpeedTests(ctx context.Context, limit int) ([]types.SpeedTestResult, error) {
	query := sqlBuilder.
		Select(
			"id",
			"server_name",
			"server_id",
			"download_speed",
			"upload_speed",
			"latency",
			"packet_loss",
			"jitter",
			"created_at",
		).
		From("speed_tests").
		OrderBy("created_at DESC").
		Limit(uint64(limit))

	rows, err := query.RunWith(s.db).QueryContext(ctx)
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
