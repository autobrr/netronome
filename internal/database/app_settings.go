// Copyright (c) 2024-2026, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	sq "github.com/Masterminds/squirrel"
)

func (s *service) GetAppSetting(ctx context.Context, key string) (string, error) {
	query := s.sqlBuilder.
		Select("value").
		From("app_settings").
		Where(sq.Eq{"key": key})

	var value string
	err := query.RunWith(s.db).QueryRowContext(ctx).Scan(&value)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrNotFound
		}
		return "", fmt.Errorf("failed to get app setting %q: %w", key, err)
	}

	return value, nil
}

func (s *service) SetAppSetting(ctx context.Context, key, value string) error {
	result, err := s.update(ctx, "app_settings", map[string]interface{}{"value": value}, sq.Eq{"key": key})
	if err != nil {
		return fmt.Errorf("failed to update app setting %q: %w", key, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to inspect updated app setting %q: %w", key, err)
	}

	if rowsAffected > 0 {
		return nil
	}

	if _, err := s.insert(ctx, "app_settings", map[string]interface{}{"key": key, "value": value}); err != nil {
		return fmt.Errorf("failed to insert app setting %q: %w", key, err)
	}

	return nil
}
