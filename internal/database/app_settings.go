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
	query := s.sqlBuilder.
		Insert("app_settings").
		Columns("key", "value").
		Values(key, value).
		Suffix("ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value")

	if _, err := query.RunWith(s.db).ExecContext(ctx); err != nil {
		return fmt.Errorf("failed to upsert app setting %q: %w", key, err)
	}

	return nil
}
