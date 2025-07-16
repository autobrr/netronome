// Copyright (c) 2024-2025, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/rs/zerolog/log"

	"github.com/autobrr/netronome/internal/types"
)

// UpsertMonitorSystemInfo inserts or updates system information for an agent
func (s *service) UpsertMonitorSystemInfo(ctx context.Context, agentID int64, info *types.MonitorSystemInfo) error {
	if s.config.Type == "sqlite" {
		// SQLite uses INSERT OR REPLACE
		query := s.sqlBuilder.
			Replace("monitor_agent_system_info").
			Columns("agent_id", "hostname", "kernel", "vnstat_version", "cpu_model", "cpu_cores", "cpu_threads", "total_memory", "updated_at").
			Values(agentID, info.Hostname, info.Kernel, info.VnstatVersion, info.CPUModel, info.CPUCores, info.CPUThreads, info.TotalMemory, time.Now())

		_, err := query.RunWith(s.db).ExecContext(ctx)
		return err
	}

	// PostgreSQL uses INSERT ... ON CONFLICT
	query := `
		INSERT INTO monitor_agent_system_info (agent_id, hostname, kernel, vnstat_version, cpu_model, cpu_cores, cpu_threads, total_memory, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (agent_id) DO UPDATE SET
			hostname = EXCLUDED.hostname,
			kernel = EXCLUDED.kernel,
			vnstat_version = EXCLUDED.vnstat_version,
			cpu_model = EXCLUDED.cpu_model,
			cpu_cores = EXCLUDED.cpu_cores,
			cpu_threads = EXCLUDED.cpu_threads,
			total_memory = EXCLUDED.total_memory,
			updated_at = EXCLUDED.updated_at`

	_, err := s.db.ExecContext(ctx, query, agentID, info.Hostname, info.Kernel, info.VnstatVersion, info.CPUModel, info.CPUCores, info.CPUThreads, info.TotalMemory, time.Now())
	return err
}

// GetMonitorSystemInfo retrieves system information for an agent
func (s *service) GetMonitorSystemInfo(ctx context.Context, agentID int64) (*types.MonitorSystemInfo, error) {
	query := s.sqlBuilder.
		Select("id", "agent_id", "hostname", "kernel", "vnstat_version", "cpu_model", "cpu_cores", "cpu_threads", "total_memory", "created_at", "updated_at").
		From("monitor_agent_system_info").
		Where(sq.Eq{"agent_id": agentID})

	var info types.MonitorSystemInfo
	err := query.RunWith(s.db).QueryRowContext(ctx).Scan(
		&info.ID, &info.AgentID, &info.Hostname, &info.Kernel, &info.VnstatVersion,
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

// SaveMonitorHistoricalSnapshot saves a vnstat data snapshot
func (s *service) SaveMonitorHistoricalSnapshot(ctx context.Context, agentID int64, snapshot *types.MonitorHistoricalSnapshot) error {
	// Compress JSON data if needed
	compressedData, err := json.Marshal(snapshot.DataJSON)
	if err != nil {
		return fmt.Errorf("failed to compress snapshot data: %w", err)
	}

	query := s.sqlBuilder.
		Insert("monitor_historical_snapshots").
		Columns("agent_id", "interface_name", "period_type", "data_json").
		Values(agentID, snapshot.InterfaceName, snapshot.PeriodType, string(compressedData))

	_, err = query.RunWith(s.db).ExecContext(ctx)
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
	// Start transaction
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Clean up bandwidth samples older than 24 hours
	bandwidthCutoff := time.Now().Add(-24 * time.Hour)
	deleteQuery := s.sqlBuilder.Delete("monitor_bandwidth_samples").Where(sq.Lt{"created_at": bandwidthCutoff})
	if _, err := deleteQuery.RunWith(tx).ExecContext(ctx); err != nil {
		log.Error().Err(err).Msg("Failed to cleanup bandwidth samples")
		return err
	}

	// Clean up resource stats older than 7 days
	resourceCutoff := time.Now().Add(-7 * 24 * time.Hour)
	deleteQuery = s.sqlBuilder.Delete("monitor_resource_stats").Where(sq.Lt{"created_at": resourceCutoff})
	if _, err := deleteQuery.RunWith(tx).ExecContext(ctx); err != nil {
		log.Error().Err(err).Msg("Failed to cleanup resource stats")
		return err
	}

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
		if _, err := tx.ExecContext(ctx, query); err != nil {
			log.Error().Err(err).Msg("Failed to cleanup historical snapshots")
			return err
		}
	} else {
		// PostgreSQL version
		query := `
			DELETE FROM monitor_historical_snapshots
			WHERE id NOT IN (
				SELECT DISTINCT ON (agent_id, period_type) id
				FROM monitor_historical_snapshots
				ORDER BY agent_id, period_type, created_at DESC
			)`
		if _, err := tx.ExecContext(ctx, query); err != nil {
			log.Error().Err(err).Msg("Failed to cleanup historical snapshots")
			return err
		}
	}

	return tx.Commit()
}