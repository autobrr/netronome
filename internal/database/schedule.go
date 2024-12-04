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
	if len(schedule.ServerIDs) == 0 {
		return nil, fmt.Errorf("%w: server IDs required", ErrInvalidInput)
	}

	serverIDs, err := json.Marshal(schedule.ServerIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal server IDs: %w", err)
	}

	options, err := json.Marshal(schedule.Options)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal options: %w", err)
	}

	data := map[string]interface{}{
		"server_ids": string(serverIDs),
		"interval":   schedule.Interval,
		"next_run":   schedule.NextRun,
		"enabled":    schedule.Enabled,
		"options":    string(options),
		"created_at": sq.Expr("CURRENT_TIMESTAMP"),
	}

	query := s.sqlBuilder.
		Insert("schedules").
		SetMap(data).
		Suffix("RETURNING id, created_at")

	err = query.RunWith(s.db).QueryRowContext(ctx).Scan(&schedule.ID, &schedule.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create schedule: %w", err)
	}

	return &schedule, nil
}

func (s *service) GetSchedules(ctx context.Context) ([]types.Schedule, error) {
	query := s.sqlBuilder.
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
		From("schedules").
		OrderBy("created_at DESC")

	rows, err := query.RunWith(s.db).QueryContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get schedules: %w", err)
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

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating schedules: %w", err)
	}

	return schedules, nil
}

func (s *service) UpdateSchedule(ctx context.Context, schedule types.Schedule) error {
	if schedule.ID <= 0 {
		return fmt.Errorf("%w: invalid schedule ID", ErrInvalidInput)
	}

	if len(schedule.ServerIDs) == 0 {
		return fmt.Errorf("%w: server IDs required", ErrInvalidInput)
	}

	serverIDs, err := json.Marshal(schedule.ServerIDs)
	if err != nil {
		return fmt.Errorf("failed to marshal server IDs: %w", err)
	}

	options, err := json.Marshal(schedule.Options)
	if err != nil {
		return fmt.Errorf("failed to marshal options: %w", err)
	}

	data := map[string]interface{}{
		"server_ids": string(serverIDs),
		"interval":   schedule.Interval,
		"next_run":   schedule.NextRun,
		"enabled":    schedule.Enabled,
		"options":    string(options),
	}

	query := s.sqlBuilder.
		Update("schedules").
		SetMap(data).
		Where(sq.Eq{"id": schedule.ID})

	result, err := query.RunWith(s.db).ExecContext(ctx)
	if err != nil {
		return fmt.Errorf("failed to update schedule: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}

	if affected == 0 {
		return ErrNotFound
	}

	return nil
}

func (s *service) DeleteSchedule(ctx context.Context, id int64) error {
	if id <= 0 {
		return fmt.Errorf("%w: invalid schedule ID", ErrInvalidInput)
	}

	query := s.sqlBuilder.
		Delete("schedules").
		Where(sq.Eq{"id": id})

	result, err := query.RunWith(s.db).ExecContext(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete schedule: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}

	if affected == 0 {
		return ErrNotFound
	}

	return nil
}
