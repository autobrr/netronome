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

// GetPacketLossMonitor retrieves a packet loss monitor by ID
func (s *service) GetPacketLossMonitor(monitorID int64) (*types.PacketLossMonitor, error) {
	query := s.sqlBuilder.
		Select("id", "host", "name", "interval", "packet_count", "enabled", "threshold", "last_run", "next_run", "last_state", "last_state_change", "created_at", "updated_at").
		From("packet_loss_monitors").
		Where(sq.Eq{"id": monitorID})

	monitor := &types.PacketLossMonitor{}
	err := query.RunWith(s.db).QueryRow().Scan(
		&monitor.ID,
		&monitor.Host,
		&monitor.Name,
		&monitor.Interval,
		&monitor.PacketCount,
		&monitor.Enabled,
		&monitor.Threshold,
		&monitor.LastRun,
		&monitor.NextRun,
		&monitor.LastState,
		&monitor.LastStateChange,
		&monitor.CreatedAt,
		&monitor.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get packet loss monitor: %w", err)
	}

	return monitor, nil
}

// GetEnabledPacketLossMonitors retrieves all enabled packet loss monitors
func (s *service) GetEnabledPacketLossMonitors() ([]*types.PacketLossMonitor, error) {
	query := s.sqlBuilder.
		Select("id", "host", "name", "interval", "packet_count", "enabled", "threshold", "last_run", "next_run", "last_state", "last_state_change", "created_at", "updated_at").
		From("packet_loss_monitors").
		Where(sq.Eq{"enabled": true}).
		OrderBy("created_at ASC")

	rows, err := query.RunWith(s.db).Query()
	if err != nil {
		return nil, fmt.Errorf("failed to get enabled packet loss monitors: %w", err)
	}
	defer rows.Close()

	var monitors []*types.PacketLossMonitor
	for rows.Next() {
		monitor := &types.PacketLossMonitor{}
		err := rows.Scan(
			&monitor.ID,
			&monitor.Host,
			&monitor.Name,
			&monitor.Interval,
			&monitor.PacketCount,
			&monitor.Enabled,
			&monitor.Threshold,
			&monitor.LastRun,
			&monitor.NextRun,
			&monitor.LastState,
			&monitor.LastStateChange,
			&monitor.CreatedAt,
			&monitor.UpdatedAt,
		)
		if err != nil {
			log.Error().Err(err).Msg("Failed to scan packet loss monitor")
			continue
		}
		monitors = append(monitors, monitor)
	}

	return monitors, nil
}

// SavePacketLossResult saves a packet loss test result
func (s *service) SavePacketLossResult(result *types.PacketLossResult) error {
	var id int64

	switch s.config.Type {
	case config.Postgres:
		query := s.sqlBuilder.
			Insert("packet_loss_results").
			Columns("monitor_id", "packet_loss", "min_rtt", "max_rtt", "avg_rtt", "std_dev_rtt", "packets_sent", "packets_recv", "used_mtr", "hop_count", "mtr_data", "privileged_mode", "created_at").
			Values(result.MonitorID, result.PacketLoss, result.MinRTT, result.MaxRTT, result.AvgRTT, result.StdDevRTT, result.PacketsSent, result.PacketsRecv, result.UsedMTR, result.HopCount, result.MTRData, result.PrivilegedMode, result.CreatedAt).
			Suffix("RETURNING id")

		sqlStr, args, err := query.ToSql()
		if err != nil {
			return fmt.Errorf("failed to build query: %w", err)
		}

		err = s.db.QueryRow(sqlStr, args...).Scan(&id)
		if err != nil {
			return fmt.Errorf("failed to save packet loss result: %w", err)
		}

	case config.SQLite:
		query := s.sqlBuilder.
			Insert("packet_loss_results").
			Columns("monitor_id", "packet_loss", "min_rtt", "max_rtt", "avg_rtt", "std_dev_rtt", "packets_sent", "packets_recv", "used_mtr", "hop_count", "mtr_data", "privileged_mode", "created_at").
			Values(result.MonitorID, result.PacketLoss, result.MinRTT, result.MaxRTT, result.AvgRTT, result.StdDevRTT, result.PacketsSent, result.PacketsRecv, result.UsedMTR, result.HopCount, result.MTRData, result.PrivilegedMode, result.CreatedAt)

		res, err := query.RunWith(s.db).Exec()
		if err != nil {
			return fmt.Errorf("failed to save packet loss result: %w", err)
		}

		id, err = res.LastInsertId()
		if err != nil {
			return fmt.Errorf("failed to get last insert ID: %w", err)
		}

	default:
		return fmt.Errorf("unsupported database type: %s", s.config.Type)
	}

	result.ID = id
	return nil
}

// GetLatestPacketLossResult retrieves the most recent packet loss result for a monitor
func (s *service) GetLatestPacketLossResult(monitorID int64) (*types.PacketLossResult, error) {
	query := s.sqlBuilder.
		Select("id", "monitor_id", "packet_loss", "min_rtt", "max_rtt", "avg_rtt", "std_dev_rtt", "packets_sent", "packets_recv", "used_mtr", "hop_count", "mtr_data", "privileged_mode", "created_at").
		From("packet_loss_results").
		Where(sq.Eq{"monitor_id": monitorID}).
		OrderBy("created_at DESC").
		Limit(1)

	result := &types.PacketLossResult{}
	err := query.RunWith(s.db).QueryRow().Scan(
		&result.ID,
		&result.MonitorID,
		&result.PacketLoss,
		&result.MinRTT,
		&result.MaxRTT,
		&result.AvgRTT,
		&result.StdDevRTT,
		&result.PacketsSent,
		&result.PacketsRecv,
		&result.UsedMTR,
		&result.HopCount,
		&result.MTRData,
		&result.PrivilegedMode,
		&result.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get latest packet loss result: %w", err)
	}

	return result, nil
}

// CreatePacketLossMonitor creates a new packet loss monitor
func (s *service) CreatePacketLossMonitor(monitor *types.PacketLossMonitor) (*types.PacketLossMonitor, error) {
	monitor.CreatedAt = time.Now()
	monitor.UpdatedAt = time.Now()

	query := s.sqlBuilder.
		Insert("packet_loss_monitors").
		Columns("host", "name", "interval", "packet_count", "enabled", "threshold", "created_at", "updated_at").
		Values(monitor.Host, monitor.Name, monitor.Interval, monitor.PacketCount, monitor.Enabled, monitor.Threshold, monitor.CreatedAt, monitor.UpdatedAt)

	if s.config.Type == config.Postgres {
		query = query.Suffix("RETURNING id")
		err := query.RunWith(s.db).QueryRow().Scan(&monitor.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to create packet loss monitor: %w", err)
		}
	} else {
		res, err := query.RunWith(s.db).Exec()
		if err != nil {
			return nil, fmt.Errorf("failed to create packet loss monitor: %w", err)
		}
		id, err := res.LastInsertId()
		if err != nil {
			return nil, fmt.Errorf("failed to get last insert ID: %w", err)
		}
		monitor.ID = id
	}

	return monitor, nil
}

// UpdatePacketLossMonitor updates an existing packet loss monitor
func (s *service) UpdatePacketLossMonitor(monitor *types.PacketLossMonitor) error {
	monitor.UpdatedAt = time.Now()

	data := map[string]interface{}{
		"host":         monitor.Host,
		"name":         monitor.Name,
		"interval":     monitor.Interval,
		"packet_count": monitor.PacketCount,
		"enabled":      monitor.Enabled,
		"threshold":    monitor.Threshold,
		"last_run":     monitor.LastRun,
		"next_run":     monitor.NextRun,
		"updated_at":   monitor.UpdatedAt,
	}

	query := s.sqlBuilder.
		Update("packet_loss_monitors").
		SetMap(data).
		Where(sq.Eq{"id": monitor.ID})

	res, err := query.RunWith(s.db).Exec()
	if err != nil {
		return fmt.Errorf("failed to update packet loss monitor: %w", err)
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

// DeletePacketLossMonitor deletes a packet loss monitor and its results
func (s *service) DeletePacketLossMonitor(monitorID int64) error {
	// Start a transaction to delete both monitor and results atomically
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Delete results first (due to foreign key constraint)
	deleteResults := s.sqlBuilder.
		Delete("packet_loss_results").
		Where(sq.Eq{"monitor_id": monitorID})

	_, err = deleteResults.RunWith(tx).Exec()
	if err != nil {
		return fmt.Errorf("failed to delete packet loss results: %w", err)
	}

	// Delete monitor
	deleteMonitor := s.sqlBuilder.
		Delete("packet_loss_monitors").
		Where(sq.Eq{"id": monitorID})

	res, err := deleteMonitor.RunWith(tx).Exec()
	if err != nil {
		return fmt.Errorf("failed to delete packet loss monitor: %w", err)
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

// GetPacketLossMonitors retrieves all packet loss monitors
func (s *service) GetPacketLossMonitors() ([]*types.PacketLossMonitor, error) {
	query := s.sqlBuilder.
		Select("id", "host", "name", "interval", "packet_count", "enabled", "threshold", "last_run", "next_run", "last_state", "last_state_change", "created_at", "updated_at").
		From("packet_loss_monitors").
		OrderBy("created_at DESC")

	rows, err := query.RunWith(s.db).Query()
	if err != nil {
		return nil, fmt.Errorf("failed to get packet loss monitors: %w", err)
	}
	defer rows.Close()

	var monitors []*types.PacketLossMonitor
	for rows.Next() {
		monitor := &types.PacketLossMonitor{}
		err := rows.Scan(
			&monitor.ID,
			&monitor.Host,
			&monitor.Name,
			&monitor.Interval,
			&monitor.PacketCount,
			&monitor.Enabled,
			&monitor.Threshold,
			&monitor.LastRun,
			&monitor.NextRun,
			&monitor.LastState,
			&monitor.LastStateChange,
			&monitor.CreatedAt,
			&monitor.UpdatedAt,
		)
		if err != nil {
			log.Error().Err(err).Msg("Failed to scan packet loss monitor")
			continue
		}
		monitors = append(monitors, monitor)
	}

	return monitors, nil
}

// GetPacketLossResults retrieves paginated packet loss result summaries for a monitor.
func (s *service) GetPacketLossResults(monitorID int64, page int, limit int) (*types.PaginatedPacketLossResults, error) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 25
	}

	countQuery := s.sqlBuilder.
		Select("COUNT(*)").
		From("packet_loss_results").
		Where(sq.Eq{"monitor_id": monitorID})

	var total int
	if err := countQuery.RunWith(s.db).QueryRow().Scan(&total); err != nil {
		return nil, fmt.Errorf("failed to count packet loss results: %w", err)
	}

	query := s.sqlBuilder.
		Select("id", "monitor_id", "packet_loss", "min_rtt", "max_rtt", "avg_rtt", "std_dev_rtt", "packets_sent", "packets_recv", "used_mtr", "hop_count", "privileged_mode", "created_at").
		From("packet_loss_results").
		Where(sq.Eq{"monitor_id": monitorID}).
		OrderBy("created_at DESC", "id DESC").
		Limit(uint64(limit)).
		Offset(uint64((page - 1) * limit))

	rows, err := query.RunWith(s.db).Query()
	if err != nil {
		return nil, fmt.Errorf("failed to get packet loss result summaries: %w", err)
	}
	defer rows.Close()

	results := make([]types.PacketLossResultSummary, 0, limit)
	for rows.Next() {
		var result types.PacketLossResultSummary
		err := rows.Scan(
			&result.ID,
			&result.MonitorID,
			&result.PacketLoss,
			&result.MinRTT,
			&result.MaxRTT,
			&result.AvgRTT,
			&result.StdDevRTT,
			&result.PacketsSent,
			&result.PacketsRecv,
			&result.UsedMTR,
			&result.HopCount,
			&result.PrivilegedMode,
			&result.CreatedAt,
		)
		if err != nil {
			log.Error().Err(err).Msg("Failed to scan packet loss result summary")
			continue
		}
		results = append(results, result)
	}

	return &types.PaginatedPacketLossResults{
		Data:  results,
		Total: total,
		Page:  page,
		Limit: limit,
	}, nil
}

// GetPacketLossResultDetail retrieves a single packet loss result including full MTR data.
func (s *service) GetPacketLossResultDetail(monitorID int64, resultID int64) (*types.PacketLossResult, error) {
	query := s.sqlBuilder.
		Select("id", "monitor_id", "packet_loss", "min_rtt", "max_rtt", "avg_rtt", "std_dev_rtt", "packets_sent", "packets_recv", "used_mtr", "hop_count", "mtr_data", "privileged_mode", "created_at").
		From("packet_loss_results").
		Where(sq.Eq{"monitor_id": monitorID, "id": resultID}).
		Limit(1)

	result := &types.PacketLossResult{}
	err := query.RunWith(s.db).QueryRow().Scan(
		&result.ID,
		&result.MonitorID,
		&result.PacketLoss,
		&result.MinRTT,
		&result.MaxRTT,
		&result.AvgRTT,
		&result.StdDevRTT,
		&result.PacketsSent,
		&result.PacketsRecv,
		&result.UsedMTR,
		&result.HopCount,
		&result.MTRData,
		&result.PrivilegedMode,
		&result.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get packet loss result detail: %w", err)
	}

	return result, nil
}

// UpdatePacketLossMonitorState updates the monitor state and timestamp
func (s *service) UpdatePacketLossMonitorState(monitorID int64, state string) error {
	query := s.sqlBuilder.
		Update("packet_loss_monitors").
		Set("last_state", state).
		Set("last_state_change", time.Now()).
		Where(sq.Eq{"id": monitorID})

	result, err := query.RunWith(s.db).Exec()
	if err != nil {
		return fmt.Errorf("failed to update packet loss monitor state: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}
