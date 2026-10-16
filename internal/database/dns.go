// Copyright (c) 2024-2026, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package database

import (
	"database/sql"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/rs/zerolog/log"

	"github.com/autobrr/netronome/internal/config"
	"github.com/autobrr/netronome/internal/types"
)

var dnsMonitorColumns = []string{
	"id", "host", "name", "protocol", "query", "record_type", "interval", "enabled",
	"last_run", "next_run", "last_state", "last_state_change", "created_at", "updated_at",
}

func scanDNSMonitor(scan func(dest ...any) error) (*types.DNSMonitor, error) {
	monitor := &types.DNSMonitor{}
	err := scan(
		&monitor.ID,
		&monitor.Host,
		&monitor.Name,
		&monitor.Protocol,
		&monitor.Query,
		&monitor.RecordType,
		&monitor.Interval,
		&monitor.Enabled,
		&monitor.LastRun,
		&monitor.NextRun,
		&monitor.LastState,
		&monitor.LastStateChange,
		&monitor.CreatedAt,
		&monitor.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return monitor, nil
}

// CreateDNSMonitor creates a new DNS monitor
func (s *service) CreateDNSMonitor(monitor *types.DNSMonitor) (*types.DNSMonitor, error) {
	monitor.CreatedAt = time.Now()
	monitor.UpdatedAt = time.Now()

	query := s.sqlBuilder.
		Insert("dns_monitors").
		Columns("host", "name", "protocol", "query", "record_type", "interval", "enabled", "next_run", "created_at", "updated_at").
		Values(monitor.Host, monitor.Name, monitor.Protocol, monitor.Query, monitor.RecordType, monitor.Interval, monitor.Enabled, monitor.NextRun, monitor.CreatedAt, monitor.UpdatedAt)

	if s.config.Type == config.Postgres {
		query = query.Suffix("RETURNING id")
		if err := query.RunWith(s.db).QueryRow().Scan(&monitor.ID); err != nil {
			return nil, fmt.Errorf("failed to create dns monitor: %w", err)
		}
		return monitor, nil
	}

	res, err := query.RunWith(s.db).Exec()
	if err != nil {
		return nil, fmt.Errorf("failed to create dns monitor: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get last insert ID: %w", err)
	}
	monitor.ID = id

	return monitor, nil
}

// GetDNSMonitor retrieves a DNS monitor by ID
func (s *service) GetDNSMonitor(monitorID int64) (*types.DNSMonitor, error) {
	query := s.sqlBuilder.
		Select(dnsMonitorColumns...).
		From("dns_monitors").
		Where(sq.Eq{"id": monitorID})

	monitor, err := scanDNSMonitor(query.RunWith(s.db).QueryRow().Scan)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get dns monitor: %w", err)
	}

	return monitor, nil
}

// GetDNSMonitors retrieves all DNS monitors
func (s *service) GetDNSMonitors() ([]*types.DNSMonitor, error) {
	query := s.sqlBuilder.
		Select(dnsMonitorColumns...).
		From("dns_monitors").
		OrderBy("created_at DESC")

	rows, err := query.RunWith(s.db).Query()
	if err != nil {
		return nil, fmt.Errorf("failed to get dns monitors: %w", err)
	}
	defer rows.Close()

	var monitors []*types.DNSMonitor
	for rows.Next() {
		monitor, err := scanDNSMonitor(rows.Scan)
		if err != nil {
			log.Error().Err(err).Msg("Failed to scan dns monitor")
			continue
		}
		monitors = append(monitors, monitor)
	}

	return monitors, rows.Err()
}

// UpdateDNSMonitor updates an existing DNS monitor
func (s *service) UpdateDNSMonitor(monitor *types.DNSMonitor) error {
	monitor.UpdatedAt = time.Now()

	query := s.sqlBuilder.
		Update("dns_monitors").
		SetMap(map[string]interface{}{
			"host":        monitor.Host,
			"name":        monitor.Name,
			"protocol":    monitor.Protocol,
			"query":       monitor.Query,
			"record_type": monitor.RecordType,
			"interval":    monitor.Interval,
			"enabled":     monitor.Enabled,
			"last_run":    monitor.LastRun,
			"next_run":    monitor.NextRun,
			"updated_at":  monitor.UpdatedAt,
		}).
		Where(sq.Eq{"id": monitor.ID})

	res, err := query.RunWith(s.db).Exec()
	if err != nil {
		return fmt.Errorf("failed to update dns monitor: %w", err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}
	if rows == 0 {
		return ErrNotFound
	}

	return nil
}

// UpdateDNSMonitorSchedule writes only the run times. The scheduler holds a
// monitor it read one tick ago, so a full row write there would put the stale
// configuration back over an edit the user made while the check ran.
func (s *service) UpdateDNSMonitorSchedule(monitorID int64, lastRun *time.Time, nextRun time.Time) error {
	query := s.sqlBuilder.
		Update("dns_monitors").
		Set("last_run", lastRun).
		Set("next_run", nextRun).
		Where(sq.Eq{"id": monitorID})

	res, err := query.RunWith(s.db).Exec()
	if err != nil {
		return fmt.Errorf("failed to update dns monitor schedule: %w", err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}
	if rows == 0 {
		return ErrNotFound
	}

	return nil
}

// UpdateDNSMonitorState updates the monitor state and timestamp
func (s *service) UpdateDNSMonitorState(monitorID int64, state string) error {
	query := s.sqlBuilder.
		Update("dns_monitors").
		Set("last_state", state).
		Set("last_state_change", time.Now()).
		Where(sq.Eq{"id": monitorID})

	res, err := query.RunWith(s.db).Exec()
	if err != nil {
		return fmt.Errorf("failed to update dns monitor state: %w", err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return ErrNotFound
	}

	return nil
}

// DeleteDNSMonitor deletes a DNS monitor and its results
func (s *service) DeleteDNSMonitor(monitorID int64) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// delete results first, SQLite only enforces the cascade with foreign keys on
	if _, err := s.sqlBuilder.Delete("dns_results").Where(sq.Eq{"monitor_id": monitorID}).RunWith(tx).Exec(); err != nil {
		return fmt.Errorf("failed to delete dns results: %w", err)
	}

	res, err := s.sqlBuilder.Delete("dns_monitors").Where(sq.Eq{"id": monitorID}).RunWith(tx).Exec()
	if err != nil {
		return fmt.Errorf("failed to delete dns monitor: %w", err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}
	if rows == 0 {
		return ErrNotFound
	}

	return tx.Commit()
}

// SaveDNSResult saves a DNS check result
func (s *service) SaveDNSResult(result *types.DNSResult) error {
	query := s.sqlBuilder.
		Insert("dns_results").
		Columns("monitor_id", "response_time_ms", "response_code", "success", "error", "created_at").
		Values(result.MonitorID, result.ResponseTimeMs, result.ResponseCode, result.Success, result.Error, result.CreatedAt)

	if s.config.Type == config.Postgres {
		query = query.Suffix("RETURNING id")
		if err := query.RunWith(s.db).QueryRow().Scan(&result.ID); err != nil {
			return fmt.Errorf("failed to save dns result: %w", err)
		}
		return nil
	}

	res, err := query.RunWith(s.db).Exec()
	if err != nil {
		return fmt.Errorf("failed to save dns result: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert ID: %w", err)
	}
	result.ID = id

	return nil
}

// GetLatestDNSResult retrieves the most recent result for a monitor
func (s *service) GetLatestDNSResult(monitorID int64) (*types.DNSResult, error) {
	query := s.sqlBuilder.
		Select("id", "monitor_id", "response_time_ms", "response_code", "success", "error", "created_at").
		From("dns_results").
		Where(sq.Eq{"monitor_id": monitorID}).
		OrderBy("created_at DESC", "id DESC").
		Limit(1)

	result := &types.DNSResult{}
	err := query.RunWith(s.db).QueryRow().Scan(
		&result.ID,
		&result.MonitorID,
		&result.ResponseTimeMs,
		&result.ResponseCode,
		&result.Success,
		&result.Error,
		&result.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get latest dns result: %w", err)
	}

	return result, nil
}

// GetDNSResults retrieves paginated results for a monitor
func (s *service) GetDNSResults(monitorID int64, page int, limit int) (*types.PaginatedDNSResults, error) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 25
	}

	countQuery := s.sqlBuilder.
		Select("COUNT(*)").
		From("dns_results").
		Where(sq.Eq{"monitor_id": monitorID})

	var total int
	if err := countQuery.RunWith(s.db).QueryRow().Scan(&total); err != nil {
		return nil, fmt.Errorf("failed to count dns results: %w", err)
	}

	query := s.sqlBuilder.
		Select("id", "monitor_id", "response_time_ms", "response_code", "success", "error", "created_at").
		From("dns_results").
		Where(sq.Eq{"monitor_id": monitorID}).
		OrderBy("created_at DESC", "id DESC").
		Limit(uint64(limit)).
		Offset(uint64((page - 1) * limit))

	rows, err := query.RunWith(s.db).Query()
	if err != nil {
		return nil, fmt.Errorf("failed to get dns results: %w", err)
	}
	defer rows.Close()

	results := make([]types.DNSResult, 0, limit)
	for rows.Next() {
		var result types.DNSResult
		err := rows.Scan(
			&result.ID,
			&result.MonitorID,
			&result.ResponseTimeMs,
			&result.ResponseCode,
			&result.Success,
			&result.Error,
			&result.CreatedAt,
		)
		if err != nil {
			log.Error().Err(err).Msg("Failed to scan dns result")
			continue
		}
		results = append(results, result)
	}

	return &types.PaginatedDNSResults{
		Data:  results,
		Total: total,
		Page:  page,
		Limit: limit,
	}, rows.Err()
}
