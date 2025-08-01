// Copyright (c) 2024-2025, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package database

import (
	"context"
	"fmt"

	sq "github.com/Masterminds/squirrel"

	"github.com/autobrr/netronome/internal/config"
	"github.com/autobrr/netronome/internal/types"
)

func (s *service) SaveIperfServer(ctx context.Context, name, host string, port int) (*types.SavedIperfServer, error) {
	if name == "" || host == "" || port <= 0 {
		return nil, ErrInvalidInput
	}

	data := map[string]interface{}{
		"name":       name,
		"host":       host,
		"port":       port,
		"created_at": sq.Expr("CURRENT_TIMESTAMP"),
		"updated_at": sq.Expr("CURRENT_TIMESTAMP"),
	}

	var id int64

	switch s.config.Type {
	case config.Postgres:
		query := s.sqlBuilder.Insert("saved_iperf_servers").
			SetMap(data).
			Suffix("RETURNING id")

		sqlStr, args, err := query.ToSql()
		if err != nil {
			return nil, fmt.Errorf("failed to build query: %w", err)
		}

		err = s.db.QueryRowContext(ctx, sqlStr, args...).Scan(&id)
		if err != nil {
			return nil, fmt.Errorf("failed to save iperf server: %w", err)
		}

	case config.SQLite:
		res, err := s.insert(ctx, "saved_iperf_servers", data)
		if err != nil {
			return nil, fmt.Errorf("failed to save iperf server: %w", err)
		}

		id, err = res.LastInsertId()
		if err != nil {
			return nil, fmt.Errorf("failed to get last insert ID: %w", err)
		}
	}

	query := s.sqlBuilder.
		Select(
			"id",
			"name",
			"host",
			"port",
			"created_at",
			"updated_at",
		).
		From("saved_iperf_servers").
		Where(sq.Eq{"id": id})

	var server types.SavedIperfServer
	err := query.RunWith(s.db).QueryRowContext(ctx).Scan(
		&server.ID,
		&server.Name,
		&server.Host,
		&server.Port,
		&server.CreatedAt,
		&server.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get saved iperf server: %w", err)
	}

	return &server, nil
}

func (s *service) GetIperfServers(ctx context.Context) ([]types.SavedIperfServer, error) {
	query := s.sqlBuilder.
		Select(
			"id",
			"name",
			"host",
			"port",
			"created_at",
			"updated_at",
		).
		From("saved_iperf_servers").
		OrderBy("created_at DESC")

	rows, err := query.RunWith(s.db).QueryContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get iperf servers: %w", err)
	}
	defer rows.Close()

	servers := make([]types.SavedIperfServer, 0)
	for rows.Next() {
		var server types.SavedIperfServer
		err := rows.Scan(
			&server.ID,
			&server.Name,
			&server.Host,
			&server.Port,
			&server.CreatedAt,
			&server.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan iperf server: %w", err)
		}
		servers = append(servers, server)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating iperf servers: %w", err)
	}

	return servers, nil
}

func (s *service) DeleteIperfServer(ctx context.Context, id int) error {
	if id <= 0 {
		return ErrInvalidInput
	}

	query := s.sqlBuilder.
		Delete("saved_iperf_servers").
		Where(sq.Eq{"id": id})

	result, err := query.RunWith(s.db).ExecContext(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete iperf server: %w", err)
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
