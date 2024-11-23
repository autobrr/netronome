// Copyright (c) 2024, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	sq "github.com/Masterminds/squirrel"

	"github.com/autobrr/netronome/internal/types"
)

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

func (s *service) GetSchedules(ctx context.Context) ([]types.Schedule, error) {
	query := sqlBuilder.
		Select(
			"id",
			"server_ids",
			"interval",
			"last_run",
			"next_run",
			"enabled",
			"options",
			"created_at",
		).
		From("schedules")

	rows, err := query.RunWith(s.db).QueryContext(ctx)
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

func (s *service) UpdateSchedule(ctx context.Context, schedule types.Schedule) error {
	serverIDs, err := json.Marshal(schedule.ServerIDs)
	if err != nil {
		return fmt.Errorf("failed to marshal server IDs: %w", err)
	}

	options, err := json.Marshal(schedule.Options)
	if err != nil {
		return fmt.Errorf("failed to marshal options: %w", err)
	}

	query := sqlBuilder.
		Update("schedules").
		Set("server_ids", string(serverIDs)).
		Set("interval", schedule.Interval).
		Set("next_run", schedule.NextRun).
		Set("enabled", schedule.Enabled).
		Set("options", string(options)).
		Where(sq.Eq{"id": schedule.ID})

	_, err = query.RunWith(s.db).ExecContext(ctx)
	if err != nil {
		return fmt.Errorf("failed to update schedule: %w", err)
	}

	return nil
}

func (s *service) DeleteSchedule(ctx context.Context, id int64) error {
	query := `DELETE FROM schedules WHERE id = ?`

	_, err := s.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete schedule: %w", err)
	}

	return nil
}
