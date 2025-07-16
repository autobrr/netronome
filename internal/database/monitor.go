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

// CreateMonitorAgent creates a new monitoring agent
func (s *service) CreateMonitorAgent(ctx context.Context, agent *types.MonitorAgent) (*types.MonitorAgent, error) {
	now := time.Now()
	agent.CreatedAt = now
	agent.UpdatedAt = now

	query := s.sqlBuilder.
		Insert("monitor_agents").
		Columns("name", "url", "api_key", "enabled", "interface", "created_at", "updated_at").
		Values(agent.Name, agent.URL, agent.APIKey, agent.Enabled, agent.Interface, agent.CreatedAt, agent.UpdatedAt)

	if s.config.Type == config.Postgres {
		query = query.Suffix("RETURNING id")
		err := query.RunWith(s.db).QueryRowContext(ctx).Scan(&agent.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to create monitor agent: %w", err)
		}
	} else {
		res, err := query.RunWith(s.db).ExecContext(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to create monitor agent: %w", err)
		}
		id, err := res.LastInsertId()
		if err != nil {
			return nil, fmt.Errorf("failed to get last insert id: %w", err)
		}
		agent.ID = id
	}

	return agent, nil
}

// GetMonitorAgent retrieves a monitoring agent by ID
func (s *service) GetMonitorAgent(ctx context.Context, agentID int64) (*types.MonitorAgent, error) {
	query := s.sqlBuilder.
		Select("id", "name", "url", "api_key", "enabled", "interface", "created_at", "updated_at").
		From("monitor_agents").
		Where(sq.Eq{"id": agentID})

	var agent types.MonitorAgent
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
		return nil, fmt.Errorf("failed to get monitor agent: %w", err)
	}

	return &agent, nil
}

// GetMonitorAgents retrieves all monitoring agents
func (s *service) GetMonitorAgents(ctx context.Context, enabledOnly bool) ([]*types.MonitorAgent, error) {
	query := s.sqlBuilder.
		Select("id", "name", "url", "api_key", "enabled", "interface", "created_at", "updated_at").
		From("monitor_agents").
		OrderBy("created_at DESC")

	if enabledOnly {
		query = query.Where(sq.Eq{"enabled": true})
	}

	rows, err := query.RunWith(s.db).QueryContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get monitor agents: %w", err)
	}
	defer rows.Close()

	agents := make([]*types.MonitorAgent, 0)
	for rows.Next() {
		var agent types.MonitorAgent
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
			return nil, fmt.Errorf("failed to scan monitor agent: %w", err)
		}
		agents = append(agents, &agent)
	}

	return agents, nil
}

// UpdateMonitorAgent updates a monitoring agent
func (s *service) UpdateMonitorAgent(ctx context.Context, agent *types.MonitorAgent) error {
	agent.UpdatedAt = time.Now()

	query := s.sqlBuilder.
		Update("monitor_agents").
		Set("name", agent.Name).
		Set("url", agent.URL).
		Set("api_key", agent.APIKey).
		Set("enabled", agent.Enabled).
		Set("interface", agent.Interface).
		Set("updated_at", agent.UpdatedAt).
		Where(sq.Eq{"id": agent.ID})

	_, err := query.RunWith(s.db).ExecContext(ctx)
	if err != nil {
		return fmt.Errorf("failed to update monitor agent: %w", err)
	}

	return nil
}

// DeleteMonitorAgent deletes a monitoring agent
func (s *service) DeleteMonitorAgent(ctx context.Context, agentID int64) error {
	query := s.sqlBuilder.
		Delete("monitor_agents").
		Where(sq.Eq{"id": agentID})

	_, err := query.RunWith(s.db).ExecContext(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete monitor agent: %w", err)
	}

	return nil
}
