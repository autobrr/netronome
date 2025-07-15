// Copyright (c) 2024-2025, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"

	"github.com/autobrr/netronome/internal/config"
	"github.com/autobrr/netronome/internal/types"
)

// CreateVnstatAgent creates a new vnstat agent
func (s *service) CreateVnstatAgent(ctx context.Context, agent *types.VnstatAgent) (*types.VnstatAgent, error) {
	now := time.Now()
	agent.CreatedAt = now
	agent.UpdatedAt = now

	query := s.sqlBuilder.
		Insert("vnstat_agents").
		Columns("name", "url", "api_key", "enabled", "interface", "created_at", "updated_at").
		Values(agent.Name, agent.URL, agent.APIKey, agent.Enabled, agent.Interface, agent.CreatedAt, agent.UpdatedAt)

	if s.config.Type == config.Postgres {
		query = query.Suffix("RETURNING id")
		err := query.RunWith(s.db).QueryRowContext(ctx).Scan(&agent.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to create vnstat agent: %w", err)
		}
	} else {
		res, err := query.RunWith(s.db).ExecContext(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to create vnstat agent: %w", err)
		}
		id, err := res.LastInsertId()
		if err != nil {
			return nil, fmt.Errorf("failed to get last insert id: %w", err)
		}
		agent.ID = id
	}

	return agent, nil
}

// GetVnstatAgent retrieves a vnstat agent by ID
func (s *service) GetVnstatAgent(ctx context.Context, agentID int64) (*types.VnstatAgent, error) {
	query := s.sqlBuilder.
		Select("id", "name", "url", "api_key", "enabled", "interface", "created_at", "updated_at").
		From("vnstat_agents").
		Where(sq.Eq{"id": agentID})

	var agent types.VnstatAgent
	err := query.RunWith(s.db).QueryRowContext(ctx).Scan(
		&agent.ID,
		&agent.Name,
		&agent.URL,
		&agent.APIKey,
		&agent.Enabled,
		&agent.Interface,
		&agent.CreatedAt,
		&agent.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get vnstat agent: %w", err)
	}

	return &agent, nil
}

// GetVnstatAgents retrieves all vnstat agents
func (s *service) GetVnstatAgents(ctx context.Context, enabledOnly bool) ([]*types.VnstatAgent, error) {
	query := s.sqlBuilder.
		Select("id", "name", "url", "api_key", "enabled", "interface", "created_at", "updated_at").
		From("vnstat_agents").
		OrderBy("created_at DESC")

	if enabledOnly {
		query = query.Where(sq.Eq{"enabled": true})
	}

	rows, err := query.RunWith(s.db).QueryContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get vnstat agents: %w", err)
	}
	defer rows.Close()

	agents := make([]*types.VnstatAgent, 0)
	for rows.Next() {
		var agent types.VnstatAgent
		err := rows.Scan(
			&agent.ID,
			&agent.Name,
			&agent.URL,
			&agent.APIKey,
			&agent.Enabled,
			&agent.Interface,
			&agent.CreatedAt,
			&agent.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan vnstat agent: %w", err)
		}
		agents = append(agents, &agent)
	}

	return agents, nil
}

// UpdateVnstatAgent updates a vnstat agent
func (s *service) UpdateVnstatAgent(ctx context.Context, agent *types.VnstatAgent) error {
	agent.UpdatedAt = time.Now()

	query := s.sqlBuilder.
		Update("vnstat_agents").
		Set("name", agent.Name).
		Set("url", agent.URL).
		Set("api_key", agent.APIKey).
		Set("enabled", agent.Enabled).
		Set("interface", agent.Interface).
		Set("updated_at", agent.UpdatedAt).
		Where(sq.Eq{"id": agent.ID})

	_, err := query.RunWith(s.db).ExecContext(ctx)
	if err != nil {
		return fmt.Errorf("failed to update vnstat agent: %w", err)
	}

	return nil
}

// DeleteVnstatAgent deletes a vnstat agent
func (s *service) DeleteVnstatAgent(ctx context.Context, agentID int64) error {
	query := s.sqlBuilder.
		Delete("vnstat_agents").
		Where(sq.Eq{"id": agentID})

	_, err := query.RunWith(s.db).ExecContext(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete vnstat agent: %w", err)
	}

	return nil
}
