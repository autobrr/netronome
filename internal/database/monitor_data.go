// Copyright (c) 2024-2025, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package database

import (
	"context"
	"database/sql"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/rs/zerolog/log"

	"github.com/autobrr/netronome/internal/types"
)

// UpsertMonitorSystemInfo inserts or updates system information for an agent
func (s *service) UpsertMonitorSystemInfo(ctx context.Context, agentID int64, info *types.MonitorSystemInfo) error {
	// First check if record exists
	_, err := s.GetMonitorSystemInfo(ctx, agentID)
	if err != nil && err != ErrNotFound {
		return err
	}

	if err == ErrNotFound {
		// Insert new record
		query := s.sqlBuilder.
			Insert("monitor_agent_system_info").
			Columns("agent_id", "hostname", "kernel", "vnstat_version", "agent_version", "cpu_model", "cpu_cores", "cpu_threads", "total_memory", "created_at", "updated_at").
			Values(agentID, info.Hostname, info.Kernel, info.VnstatVersion, info.AgentVersion, info.CPUModel, info.CPUCores, info.CPUThreads, info.TotalMemory, time.Now(), time.Now())

		_, err := query.RunWith(s.db).ExecContext(ctx)
		return err
	}

	// Update existing record - only update non-empty fields
	update := s.sqlBuilder.Update("monitor_agent_system_info").
		Set("updated_at", time.Now()).
		Where(sq.Eq{"agent_id": agentID})

	hasUpdates := false

	if info.Hostname != "" {
		update = update.Set("hostname", info.Hostname)
		hasUpdates = true
	}
	if info.Kernel != "" {
		update = update.Set("kernel", info.Kernel)
		hasUpdates = true
	}
	if info.VnstatVersion != "" {
		update = update.Set("vnstat_version", info.VnstatVersion)
		hasUpdates = true
	}
	if info.AgentVersion != "" {
		update = update.Set("agent_version", info.AgentVersion)
		hasUpdates = true
	}
	if info.CPUModel != "" {
		update = update.Set("cpu_model", info.CPUModel)
		hasUpdates = true
	}
	if info.CPUCores > 0 {
		update = update.Set("cpu_cores", info.CPUCores)
		hasUpdates = true
	}
	if info.CPUThreads > 0 {
		update = update.Set("cpu_threads", info.CPUThreads)
		hasUpdates = true
	}
	if info.TotalMemory > 0 {
		update = update.Set("total_memory", info.TotalMemory)
		hasUpdates = true
	}

	if !hasUpdates {
		return nil // Nothing to update
	}

	_, err = update.RunWith(s.db).ExecContext(ctx)
	return err
}

// GetMonitorSystemInfo retrieves system information for an agent
func (s *service) GetMonitorSystemInfo(ctx context.Context, agentID int64) (*types.MonitorSystemInfo, error) {
	query := s.sqlBuilder.
		Select("id", "agent_id", "hostname", "kernel", "vnstat_version", "agent_version", "cpu_model", "cpu_cores", "cpu_threads", "total_memory", "created_at", "updated_at").
		From("monitor_agent_system_info").
		Where(sq.Eq{"agent_id": agentID})

	var info types.MonitorSystemInfo
	err := query.RunWith(s.db).QueryRowContext(ctx).Scan(
		&info.ID, &info.AgentID, &info.Hostname, &info.Kernel, &info.VnstatVersion, &info.AgentVersion,
		&info.CPUModel, &info.CPUCores, &info.CPUThreads, &info.TotalMemory,
		&info.CreatedAt, &info.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	return &info, err
}

// UpsertMonitorInterfaces inserts or updates network interfaces for an agent
func (s *service) UpsertMonitorInterfaces(ctx context.Context, agentID int64, interfaces []types.MonitorInterface) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Delete existing interfaces for this agent
	deleteQuery := s.sqlBuilder.Delete("monitor_agent_interfaces").Where(sq.Eq{"agent_id": agentID})
	if _, err := deleteQuery.RunWith(tx).ExecContext(ctx); err != nil {
		return err
	}

	// Insert new interfaces
	for _, iface := range interfaces {
		insertQuery := s.sqlBuilder.
			Insert("monitor_agent_interfaces").
			Columns("agent_id", "name", "alias", "ip_address", "link_speed", "updated_at").
			Values(agentID, iface.Name, iface.Alias, iface.IPAddress, iface.LinkSpeed, time.Now())

		if _, err := insertQuery.RunWith(tx).ExecContext(ctx); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// GetMonitorInterfaces retrieves network interfaces for an agent
func (s *service) GetMonitorInterfaces(ctx context.Context, agentID int64) ([]types.MonitorInterface, error) {
	query := s.sqlBuilder.
		Select("id", "agent_id", "name", "alias", "ip_address", "link_speed", "created_at", "updated_at").
		From("monitor_agent_interfaces").
		Where(sq.Eq{"agent_id": agentID}).
		OrderBy("name")

	rows, err := query.RunWith(s.db).QueryContext(ctx)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var interfaces []types.MonitorInterface
	for rows.Next() {
		var iface types.MonitorInterface
		if err := rows.Scan(
			&iface.ID, &iface.AgentID, &iface.Name, &iface.Alias,
			&iface.IPAddress, &iface.LinkSpeed, &iface.CreatedAt, &iface.UpdatedAt,
		); err != nil {
			return nil, err
		}
		interfaces = append(interfaces, iface)
	}

	return interfaces, rows.Err()
}

// SaveMonitorBandwidthSample saves a bandwidth sample for an agent
func (s *service) SaveMonitorBandwidthSample(ctx context.Context, agentID int64, rxBytes, txBytes int64) error {
	query := s.sqlBuilder.
		Insert("monitor_bandwidth_samples").
		Columns("agent_id", "rx_bytes_per_second", "tx_bytes_per_second").
		Values(agentID, rxBytes, txBytes)

	_, err := query.RunWith(s.db).ExecContext(ctx)
	return err
}

// GetMonitorBandwidthSamples retrieves bandwidth samples for an agent
func (s *service) GetMonitorBandwidthSamples(ctx context.Context, agentID int64, hours int) ([]types.MonitorBandwidthSample, error) {
	since := time.Now().Add(-time.Duration(hours) * time.Hour)

	query := s.sqlBuilder.
		Select("id", "agent_id", "rx_bytes_per_second", "tx_bytes_per_second", "created_at").
		From("monitor_bandwidth_samples").
		Where(sq.And{
			sq.Eq{"agent_id": agentID},
			sq.GtOrEq{"created_at": since},
		}).
		OrderBy("created_at DESC")

	rows, err := query.RunWith(s.db).QueryContext(ctx)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var samples []types.MonitorBandwidthSample
	for rows.Next() {
		var sample types.MonitorBandwidthSample
		if err := rows.Scan(
			&sample.ID, &sample.AgentID, &sample.RxBytesPerSecond,
			&sample.TxBytesPerSecond, &sample.CreatedAt,
		); err != nil {
			return nil, err
		}
		samples = append(samples, sample)
	}

	return samples, rows.Err()
}

// UpsertMonitorPeakStats inserts or updates peak stats for an agent
func (s *service) UpsertMonitorPeakStats(ctx context.Context, agentID int64, stats *types.MonitorPeakStats) error {
	// First, get existing peaks to compare
	existingStats, err := s.GetMonitorPeakStats(ctx, agentID)
	if err != nil && err != ErrNotFound {
		return err
	}

	// If no existing stats, insert new
	if err == ErrNotFound {
		query := s.sqlBuilder.
			Insert("monitor_peak_stats").
			Columns("agent_id", "peak_rx_bytes", "peak_tx_bytes", "peak_rx_timestamp", "peak_tx_timestamp").
			Values(agentID, stats.PeakRxBytes, stats.PeakTxBytes, stats.PeakRxTimestamp, stats.PeakTxTimestamp)

		_, err := query.RunWith(s.db).ExecContext(ctx)
		return err
	}

	// Update only if new peaks are higher
	needsUpdate := false
	if stats.PeakRxBytes > existingStats.PeakRxBytes {
		existingStats.PeakRxBytes = stats.PeakRxBytes
		existingStats.PeakRxTimestamp = stats.PeakRxTimestamp
		needsUpdate = true
	}
	if stats.PeakTxBytes > existingStats.PeakTxBytes {
		existingStats.PeakTxBytes = stats.PeakTxBytes
		existingStats.PeakTxTimestamp = stats.PeakTxTimestamp
		needsUpdate = true
	}

	if needsUpdate {
		query := s.sqlBuilder.
			Update("monitor_peak_stats").
			Set("peak_rx_bytes", existingStats.PeakRxBytes).
			Set("peak_tx_bytes", existingStats.PeakTxBytes).
			Set("peak_rx_timestamp", existingStats.PeakRxTimestamp).
			Set("peak_tx_timestamp", existingStats.PeakTxTimestamp).
			Where(sq.Eq{"agent_id": agentID})

		_, err = query.RunWith(s.db).ExecContext(ctx)
	}

	return err
}

// GetMonitorPeakStats retrieves peak stats for an agent
func (s *service) GetMonitorPeakStats(ctx context.Context, agentID int64) (*types.MonitorPeakStats, error) {
	query := s.sqlBuilder.
		Select("id", "agent_id", "peak_rx_bytes", "peak_tx_bytes", "peak_rx_timestamp", "peak_tx_timestamp", "created_at").
		From("monitor_peak_stats").
		Where(sq.Eq{"agent_id": agentID}).
		OrderBy("created_at DESC").
		Limit(1)

	var stats types.MonitorPeakStats
	err := query.RunWith(s.db).QueryRowContext(ctx).Scan(
		&stats.ID, &stats.AgentID, &stats.PeakRxBytes, &stats.PeakTxBytes,
		&stats.PeakRxTimestamp, &stats.PeakTxTimestamp, &stats.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	return &stats, err
}

// SaveMonitorResourceStats saves resource usage stats for an agent
func (s *service) SaveMonitorResourceStats(ctx context.Context, agentID int64, stats *types.MonitorResourceStats) error {
	query := s.sqlBuilder.
		Insert("monitor_resource_stats").
		Columns("agent_id", "cpu_usage_percent", "memory_used_percent", "swap_used_percent", "disk_usage_json", "temperature_json", "uptime_seconds").
		Values(agentID, stats.CPUUsagePercent, stats.MemoryUsedPercent, stats.SwapUsedPercent, stats.DiskUsageJSON, stats.TemperatureJSON, stats.UptimeSeconds)

	_, err := query.RunWith(s.db).ExecContext(ctx)
	return err
}

// GetMonitorResourceStats retrieves resource stats for an agent
func (s *service) GetMonitorResourceStats(ctx context.Context, agentID int64, hours int) ([]types.MonitorResourceStats, error) {
	since := time.Now().Add(-time.Duration(hours) * time.Hour)

	query := s.sqlBuilder.
		Select("id", "agent_id", "cpu_usage_percent", "memory_used_percent", "swap_used_percent", "disk_usage_json", "temperature_json", "uptime_seconds", "created_at").
		From("monitor_resource_stats").
		Where(sq.And{
			sq.Eq{"agent_id": agentID},
			sq.GtOrEq{"created_at": since},
		}).
		OrderBy("created_at DESC")

	rows, err := query.RunWith(s.db).QueryContext(ctx)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []types.MonitorResourceStats
	for rows.Next() {
		var stat types.MonitorResourceStats
		if err := rows.Scan(
			&stat.ID, &stat.AgentID, &stat.CPUUsagePercent, &stat.MemoryUsedPercent,
			&stat.SwapUsedPercent, &stat.DiskUsageJSON, &stat.TemperatureJSON,
			&stat.UptimeSeconds, &stat.CreatedAt,
		); err != nil {
			return nil, err
		}
		stats = append(stats, stat)
	}

	return stats, rows.Err()
}

// SaveMonitorHistoricalSnapshot saves a bandwidth monitoring data snapshot
func (s *service) SaveMonitorHistoricalSnapshot(ctx context.Context, agentID int64, snapshot *types.MonitorHistoricalSnapshot) error {
	query := s.sqlBuilder.
		Insert("monitor_historical_snapshots").
		Columns("agent_id", "interface_name", "period_type", "data_json").
		Values(agentID, snapshot.InterfaceName, snapshot.PeriodType, snapshot.DataJSON)

	_, err := query.RunWith(s.db).ExecContext(ctx)
	return err
}

// GetMonitorLatestSnapshot retrieves the latest snapshot for an agent
func (s *service) GetMonitorLatestSnapshot(ctx context.Context, agentID int64, periodType string) (*types.MonitorHistoricalSnapshot, error) {
	query := s.sqlBuilder.
		Select("id", "agent_id", "interface_name", "period_type", "data_json", "created_at").
		From("monitor_historical_snapshots").
		Where(sq.And{
			sq.Eq{"agent_id": agentID},
			sq.Eq{"period_type": periodType},
		}).
		OrderBy("created_at DESC").
		Limit(1)

	var snapshot types.MonitorHistoricalSnapshot
	err := query.RunWith(s.db).QueryRowContext(ctx).Scan(
		&snapshot.ID, &snapshot.AgentID, &snapshot.InterfaceName,
		&snapshot.PeriodType, &snapshot.DataJSON, &snapshot.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	return &snapshot, err
}

// CleanupMonitorData removes old data based on retention policies
func (s *service) CleanupMonitorData(ctx context.Context) error {
	log.Info().Msg("Starting monitor data cleanup")
	
	// Start transaction
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Clean up resource stats older than 2 hours
	// We collect every 30 seconds, so 2 hours gives us ~240 data points which is plenty
	resourceCutoff := time.Now().Add(-2 * time.Hour)
	log.Debug().Time("cutoff", resourceCutoff).Msg("Cleaning up resource stats")
	
	deleteQuery := s.sqlBuilder.Delete("monitor_resource_stats").Where(sq.Lt{"created_at": resourceCutoff})
	result, err := deleteQuery.RunWith(tx).ExecContext(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Failed to cleanup resource stats")
		return err
	}
	
	rowsDeleted, _ := result.RowsAffected()
	log.Info().Int64("rows_deleted", rowsDeleted).Msg("Cleaned up resource stats")

	// Clean up old historical snapshots - keep only the latest of each type per agent
	// This is more complex and requires a subquery
	if s.config.Type == "sqlite" {
		query := `
			DELETE FROM monitor_historical_snapshots
			WHERE id NOT IN (
				SELECT MAX(id)
				FROM monitor_historical_snapshots
				GROUP BY agent_id, period_type
			)`
		result, err := tx.ExecContext(ctx, query)
		if err != nil {
			log.Error().Err(err).Msg("Failed to cleanup historical snapshots")
			return err
		}
		rowsDeleted, _ := result.RowsAffected()
		log.Info().Int64("snapshots_deleted", rowsDeleted).Msg("Cleaned up historical snapshots")
	} else {
		// PostgreSQL version
		query := `
			DELETE FROM monitor_historical_snapshots
			WHERE id NOT IN (
				SELECT DISTINCT ON (agent_id, period_type) id
				FROM monitor_historical_snapshots
				ORDER BY agent_id, period_type, created_at DESC
			)`
		result, err := tx.ExecContext(ctx, query)
		if err != nil {
			log.Error().Err(err).Msg("Failed to cleanup historical snapshots")
			return err
		}
		rowsDeleted, _ := result.RowsAffected()
		log.Info().Int64("snapshots_deleted", rowsDeleted).Msg("Cleaned up historical snapshots")
	}

	if err := tx.Commit(); err != nil {
		log.Error().Err(err).Msg("Failed to commit cleanup transaction")
		return err
	}
	
	log.Info().Msg("Monitor data cleanup completed successfully")
	return nil
}
