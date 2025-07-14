// Copyright (c) 2024-2025, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/rs/zerolog/log"

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
		Columns("name", "url", "enabled", "interface", "retention_days", "created_at", "updated_at").
		Values(agent.Name, agent.URL, agent.Enabled, agent.Interface, agent.RetentionDays, agent.CreatedAt, agent.UpdatedAt)

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
		Select("id", "name", "url", "enabled", "interface", "retention_days", "created_at", "updated_at").
		From("vnstat_agents").
		Where(sq.Eq{"id": agentID})

	var agent types.VnstatAgent
	err := query.RunWith(s.db).QueryRowContext(ctx).Scan(
		&agent.ID,
		&agent.Name,
		&agent.URL,
		&agent.Enabled,
		&agent.Interface,
		&agent.RetentionDays,
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
		Select("id", "name", "url", "enabled", "interface", "retention_days", "created_at", "updated_at").
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
			&agent.Enabled,
			&agent.Interface,
			&agent.RetentionDays,
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
		Set("enabled", agent.Enabled).
		Set("interface", agent.Interface).
		Set("retention_days", agent.RetentionDays).
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

// SaveVnstatBandwidth saves bandwidth data
func (s *service) SaveVnstatBandwidth(ctx context.Context, bandwidth *types.VnstatBandwidth) error {
	now := time.Now()
	bandwidth.CreatedAt = now

	// For SQLite, format the timestamp properly
	var createdAt interface{}
	if s.config.Type == config.SQLite {
		createdAt = now.Format(time.RFC3339)
	} else {
		createdAt = now
	}

	query := s.sqlBuilder.
		Insert("vnstat_bandwidth").
		Columns("agent_id", "rx_bytes_per_second", "tx_bytes_per_second", "rx_packets_per_second", "tx_packets_per_second", "rx_rate_string", "tx_rate_string", "created_at").
		Values(bandwidth.AgentID, bandwidth.RxBytesPerSecond, bandwidth.TxBytesPerSecond, bandwidth.RxPacketsPerSecond, bandwidth.TxPacketsPerSecond, bandwidth.RxRateString, bandwidth.TxRateString, createdAt)

	if s.config.Type == config.Postgres {
		query = query.Suffix("RETURNING id")
		err := query.RunWith(s.db).QueryRowContext(ctx).Scan(&bandwidth.ID)
		if err != nil {
			return fmt.Errorf("failed to save vnstat bandwidth: %w", err)
		}
	} else {
		res, err := query.RunWith(s.db).ExecContext(ctx)
		if err != nil {
			return fmt.Errorf("failed to save vnstat bandwidth: %w", err)
		}
		id, err := res.LastInsertId()
		if err != nil {
			log.Warn().Err(err).Msg("Failed to get last insert ID for vnstat bandwidth")
		} else {
			bandwidth.ID = id
		}
	}

	return nil
}

// GetVnstatBandwidthHistory retrieves bandwidth history for an agent
func (s *service) GetVnstatBandwidthHistory(ctx context.Context, agentID int64, startTime, endTime time.Time, limit int) ([]*types.VnstatBandwidth, error) {
	// Format times for comparison based on database type
	var startTimeStr, endTimeStr interface{}
	if s.config.Type == config.SQLite {
		startTimeStr = startTime.Format(time.RFC3339)
		endTimeStr = endTime.Format(time.RFC3339)
	} else {
		startTimeStr = startTime
		endTimeStr = endTime
	}

	query := s.sqlBuilder.
		Select("id", "agent_id", "rx_bytes_per_second", "tx_bytes_per_second", "rx_packets_per_second", "tx_packets_per_second", "rx_rate_string", "tx_rate_string", "created_at").
		From("vnstat_bandwidth").
		Where(sq.And{
			sq.Eq{"agent_id": agentID},
			sq.GtOrEq{"created_at": startTimeStr},
			sq.LtOrEq{"created_at": endTimeStr},
		}).
		OrderBy("created_at DESC")

	if limit > 0 {
		query = query.Limit(uint64(limit))
	}

	rows, err := query.RunWith(s.db).QueryContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get vnstat bandwidth history: %w", err)
	}
	defer rows.Close()

	results := make([]*types.VnstatBandwidth, 0)
	for rows.Next() {
		var bandwidth types.VnstatBandwidth
		var createdAtStr string

		err := rows.Scan(
			&bandwidth.ID,
			&bandwidth.AgentID,
			&bandwidth.RxBytesPerSecond,
			&bandwidth.TxBytesPerSecond,
			&bandwidth.RxPacketsPerSecond,
			&bandwidth.TxPacketsPerSecond,
			&bandwidth.RxRateString,
			&bandwidth.TxRateString,
			&createdAtStr,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan vnstat bandwidth: %w", err)
		}

		// Parse the timestamp based on database type
		if s.config.Type == config.SQLite {
			bandwidth.CreatedAt, err = time.Parse(time.RFC3339, createdAtStr)
			if err != nil {
				// Try parsing the old format for backward compatibility
				bandwidth.CreatedAt, _ = time.Parse("2006-01-02 15:04:05.999999 -0700 MST", createdAtStr)
			}
		} else {
			// For PostgreSQL, the driver handles time.Time directly
			bandwidth.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
		}

		results = append(results, &bandwidth)
	}

	return results, nil
}

// CleanupOldVnstatData removes old bandwidth data based on retention settings
func (s *service) CleanupOldVnstatData(ctx context.Context) error {
	// Get all agents with their retention settings
	agents, err := s.GetVnstatAgents(ctx, false)
	if err != nil {
		return fmt.Errorf("failed to get agents for cleanup: %w", err)
	}

	for _, agent := range agents {
		cutoffTime := time.Now().AddDate(0, 0, -agent.RetentionDays)

		// Format cutoff time for comparison based on database type
		var cutoffTimeStr interface{}
		if s.config.Type == config.SQLite {
			cutoffTimeStr = cutoffTime.Format(time.RFC3339)
		} else {
			cutoffTimeStr = cutoffTime
		}

		query := s.sqlBuilder.
			Delete("vnstat_bandwidth").
			Where(sq.And{
				sq.Eq{"agent_id": agent.ID},
				sq.Lt{"created_at": cutoffTimeStr},
			})

		result, err := query.RunWith(s.db).ExecContext(ctx)
		if err != nil {
			log.Error().
				Err(err).
				Int64("agent_id", agent.ID).
				Msg("Failed to cleanup old vnstat data")
			continue
		}

		if affected, err := result.RowsAffected(); err == nil && affected > 0 {
			log.Info().
				Int64("agent_id", agent.ID).
				Int64("rows_deleted", affected).
				Msg("Cleaned up old vnstat data")
		}
	}

	return nil
}

// AggregateVnstatBandwidthHourly aggregates bandwidth data into hourly buckets
func (s *service) AggregateVnstatBandwidthHourly(ctx context.Context) error {
	return s.aggregateVnstatBandwidthHourly(ctx, false)
}

// ForceAggregateVnstatBandwidthHourly aggregates all available bandwidth data regardless of age
func (s *service) ForceAggregateVnstatBandwidthHourly(ctx context.Context) error {
	return s.aggregateVnstatBandwidthHourly(ctx, true)
}

// aggregateVnstatBandwidthHourly is the internal implementation
func (s *service) aggregateVnstatBandwidthHourly(ctx context.Context, force bool) error {
	// Get all agents
	agents, err := s.GetVnstatAgents(ctx, false)
	if err != nil {
		return fmt.Errorf("failed to get agents for aggregation: %w", err)
	}

	log.Info().Int("agent_count", len(agents)).Bool("force", force).Msg("Starting bandwidth aggregation")

	for _, agent := range agents {
		var cutoffTime time.Time
		if force {
			// Force mode: process all data regardless of age
			cutoffTime = time.Time{}
		} else {
			// Normal mode: only process data older than 2 hours
			cutoffTime = time.Now().Add(-2 * time.Hour)
		}

		// Format cutoff time for comparison based on database type
		var cutoffTimeStr interface{}
		if s.config.Type == config.SQLite {
			cutoffTimeStr = cutoffTime.Format(time.RFC3339)
		} else {
			cutoffTimeStr = cutoffTime
		}

		// Build query based on force mode
		rawDataQuery := s.sqlBuilder.
			Select("rx_bytes_per_second", "tx_bytes_per_second", "created_at").
			From("vnstat_bandwidth").
			Where(sq.Eq{"agent_id": agent.ID}).
			OrderBy("created_at ASC")

		if !force {
			// Normal mode: only process data older than 2 hours
			rawDataQuery = rawDataQuery.Where(sq.Lt{"created_at": cutoffTimeStr})
		}

		sql, args, _ := rawDataQuery.ToSql()
		log.Debug().Str("sql", sql).Interface("args", args).Int64("agent_id", agent.ID).Msg("Executing aggregation query")

		rows, err := rawDataQuery.RunWith(s.db).QueryContext(ctx)
		if err != nil {
			log.Error().Err(err).Int64("agent_id", agent.ID).Msg("Failed to query raw bandwidth data for aggregation")
			continue
		}

		// Group data by hour
		hourlyData := make(map[string]*struct {
			samples []struct {
				rxRate int64
				txRate int64
				time   time.Time
			}
		})

		for rows.Next() {
			var rxBytes, txBytes *int64
			var createdAtStr string

			err := rows.Scan(&rxBytes, &txBytes, &createdAtStr)
			if err != nil {
				log.Error().Err(err).Msg("Failed to scan raw bandwidth data")
				continue
			}

			// Parse timestamp
			var createdAt time.Time
			if s.config.Type == config.SQLite {
				createdAt, err = time.Parse(time.RFC3339, createdAtStr)
				if err != nil {
					log.Error().Err(err).Str("timestamp", createdAtStr).Msg("Failed to parse timestamp")
					continue
				}
			} else {
				createdAt, err = time.Parse(time.RFC3339, createdAtStr)
				if err != nil {
					log.Error().Err(err).Str("timestamp", createdAtStr).Msg("Failed to parse timestamp")
					continue
				}
			}

			// Truncate to hour boundary
			hourStart := createdAt.Truncate(time.Hour)
			hourKey := hourStart.Format(time.RFC3339)

			if hourlyData[hourKey] == nil {
				hourlyData[hourKey] = &struct {
					samples []struct {
						rxRate int64
						txRate int64
						time   time.Time
					}
				}{}
			}

			// Store the sample with its timestamp
			sample := struct {
				rxRate int64
				txRate int64
				time   time.Time
			}{
				time: createdAt,
			}
			if rxBytes != nil {
				sample.rxRate = *rxBytes
			}
			if txBytes != nil {
				sample.txRate = *txBytes
			}
			hourlyData[hourKey].samples = append(hourlyData[hourKey].samples, sample)
		}
		rows.Close()

		log.Info().Int64("agent_id", agent.ID).Int("hourly_buckets", len(hourlyData)).Msg("Processed raw data into hourly buckets")

		// Insert/update hourly aggregations
		for hourKey, data := range hourlyData {
			hourStart, _ := time.Parse(time.RFC3339, hourKey)

			// Calculate total bytes for this hour
			var totalRx, totalTx int64

			if len(data.samples) == 1 {
				// Single sample in the hour - assume it represents the full hour at that rate
				totalRx = data.samples[0].rxRate * 3600
				totalTx = data.samples[0].txRate * 3600
			} else if len(data.samples) > 1 {
				// Multiple samples - calculate based on time intervals
				for i := 0; i < len(data.samples); i++ {
					var duration int64
					if i < len(data.samples)-1 {
						// Duration until next sample
						duration = int64(data.samples[i+1].time.Sub(data.samples[i].time).Seconds())
					} else {
						// Last sample - assume 5 seconds for live data
						duration = 5
					}
					totalRx += data.samples[i].rxRate * duration
					totalTx += data.samples[i].txRate * duration
				}
			}

			// Format hour start for database
			var hourStartDB interface{}
			if s.config.Type == config.SQLite {
				hourStartDB = hourStart.Format(time.RFC3339)
			} else {
				hourStartDB = hourStart
			}

			// Insert or update hourly record
			_, err := s.sqlBuilder.
				Insert("vnstat_bandwidth_hourly").
				Columns("agent_id", "hour_start", "total_rx_bytes", "total_tx_bytes", "updated_at").
				Values(agent.ID, hourStartDB, totalRx, totalTx, time.Now()).
				Suffix("ON CONFLICT (agent_id, hour_start) DO UPDATE SET total_rx_bytes = ?, total_tx_bytes = ?, updated_at = ?", totalRx, totalTx, time.Now()).
				RunWith(s.db).ExecContext(ctx)

			if err != nil {
				// Try SQLite upsert syntax instead
				if s.config.Type == config.SQLite {
					_, err = s.sqlBuilder.
						Insert("vnstat_bandwidth_hourly").
						Columns("agent_id", "hour_start", "total_rx_bytes", "total_tx_bytes", "updated_at").
						Values(agent.ID, hourStartDB, totalRx, totalTx, time.Now()).
						Suffix("ON CONFLICT (agent_id, hour_start) DO UPDATE SET total_rx_bytes = excluded.total_rx_bytes, total_tx_bytes = excluded.total_tx_bytes, updated_at = excluded.updated_at").
						RunWith(s.db).ExecContext(ctx)
				}

				if err != nil {
					log.Error().Err(err).Int64("agent_id", agent.ID).Str("hour", hourKey).Msg("Failed to insert/update hourly aggregation")
				}
			}
		}

		// Delete raw data that has been aggregated
		if len(hourlyData) > 0 {
			_, err = s.sqlBuilder.
				Delete("vnstat_bandwidth").
				Where(sq.And{
					sq.Eq{"agent_id": agent.ID},
					sq.Lt{"created_at": cutoffTimeStr},
				}).
				RunWith(s.db).ExecContext(ctx)

			if err != nil {
				log.Error().Err(err).Int64("agent_id", agent.ID).Msg("Failed to delete aggregated raw data")
			} else {
				log.Info().Int64("agent_id", agent.ID).Int("hours", len(hourlyData)).Msg("Aggregated bandwidth data into hourly buckets")
			}
		}
	}

	return nil
}

// GetVnstatBandwidthUsage calculates usage statistics using both raw and aggregated data
func (s *service) GetVnstatBandwidthUsage(ctx context.Context, agentID int64) (map[string]struct {
	Download int64 `json:"download"`
	Upload   int64 `json:"upload"`
	Total    int64 `json:"total"`
}, error) {
	now := time.Now()
	usage := make(map[string]struct {
		Download int64 `json:"download"`
		Upload   int64 `json:"upload"`
		Total    int64 `json:"total"`
	})

	// This Hour - use raw data
	currentHour := now.Truncate(time.Hour)
	var currentHourStart interface{}
	if s.config.Type == config.SQLite {
		currentHourStart = currentHour.Format(time.RFC3339)
	} else {
		currentHourStart = currentHour
	}

	var thisHourRx, thisHourTx int64
	err := s.sqlBuilder.
		Select("COALESCE(SUM(COALESCE(rx_bytes_per_second, 0)), 0)", "COALESCE(SUM(COALESCE(tx_bytes_per_second, 0)), 0)").
		From("vnstat_bandwidth").
		Where(sq.And{
			sq.Eq{"agent_id": agentID},
			sq.GtOrEq{"created_at": currentHourStart},
		}).
		RunWith(s.db).QueryRowContext(ctx).Scan(&thisHourRx, &thisHourTx)

	if err != nil {
		log.Error().Err(err).Msg("Failed to calculate this hour usage")
	} else {
		usage["This Hour"] = struct {
			Download int64 `json:"download"`
			Upload   int64 `json:"upload"`
			Total    int64 `json:"total"`
		}{thisHourRx, thisHourTx, thisHourRx + thisHourTx}
	}

	// Last Hour - use hourly aggregation
	lastHour := currentHour.Add(-time.Hour)
	var lastHourStart interface{}
	if s.config.Type == config.SQLite {
		lastHourStart = lastHour.Format(time.RFC3339)
	} else {
		lastHourStart = lastHour
	}

	var lastHourRx, lastHourTx int64
	err = s.sqlBuilder.
		Select("COALESCE(total_rx_bytes, 0)", "COALESCE(total_tx_bytes, 0)").
		From("vnstat_bandwidth_hourly").
		Where(sq.And{
			sq.Eq{"agent_id": agentID},
			sq.Eq{"hour_start": lastHourStart},
		}).
		RunWith(s.db).QueryRowContext(ctx).Scan(&lastHourRx, &lastHourTx)

	if err != nil && err != sql.ErrNoRows {
		log.Error().Err(err).Msg("Failed to calculate last hour usage")
	} else {
		usage["Last Hour"] = struct {
			Download int64 `json:"download"`
			Upload   int64 `json:"upload"`
			Total    int64 `json:"total"`
		}{lastHourRx, lastHourTx, lastHourRx + lastHourTx}
	}

	// Today - sum hourly aggregations (excluding current hour) + current hour raw data
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	var todayStartDB interface{}
	if s.config.Type == config.SQLite {
		todayStartDB = todayStart.Format(time.RFC3339)
		currentHourStart = currentHour.Format(time.RFC3339)
	} else {
		todayStartDB = todayStart
		currentHourStart = currentHour
	}

	var todayRx, todayTx int64
	err = s.sqlBuilder.
		Select("COALESCE(SUM(total_rx_bytes), 0)", "COALESCE(SUM(total_tx_bytes), 0)").
		From("vnstat_bandwidth_hourly").
		Where(sq.And{
			sq.Eq{"agent_id": agentID},
			sq.GtOrEq{"hour_start": todayStartDB},
			sq.Lt{"hour_start": currentHourStart}, // Exclude current hour to avoid double counting
		}).
		RunWith(s.db).QueryRowContext(ctx).Scan(&todayRx, &todayTx)

	if err != nil {
		log.Error().Err(err).Msg("Failed to calculate today usage from hourly data")
	}

	// Add current hour raw data to today's total
	todayRx += thisHourRx
	todayTx += thisHourTx

	usage["Today"] = struct {
		Download int64 `json:"download"`
		Upload   int64 `json:"upload"`
		Total    int64 `json:"total"`
	}{todayRx, todayTx, todayRx + todayTx}

	// This Month - sum all hourly aggregations for current month (excluding current hour) + current hour
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	var monthStartDB interface{}
	if s.config.Type == config.SQLite {
		monthStartDB = monthStart.Format(time.RFC3339)
	} else {
		monthStartDB = monthStart
	}

	var monthRx, monthTx int64
	err = s.sqlBuilder.
		Select("COALESCE(SUM(total_rx_bytes), 0)", "COALESCE(SUM(total_tx_bytes), 0)").
		From("vnstat_bandwidth_hourly").
		Where(sq.And{
			sq.Eq{"agent_id": agentID},
			sq.GtOrEq{"hour_start": monthStartDB},
			sq.Lt{"hour_start": currentHourStart}, // Exclude current hour to avoid double counting
		}).
		RunWith(s.db).QueryRowContext(ctx).Scan(&monthRx, &monthTx)

	if err != nil {
		log.Error().Err(err).Msg("Failed to calculate this month usage")
	}

	// Add current hour raw data to month's total
	monthRx += thisHourRx
	monthTx += thisHourTx

	usage["This Month"] = struct {
		Download int64 `json:"download"`
		Upload   int64 `json:"upload"`
		Total    int64 `json:"total"`
	}{monthRx, monthTx, monthRx + monthTx}

	// All Time - sum all hourly aggregations (excluding current hour) + current hour
	var allTimeRx, allTimeTx int64
	err = s.sqlBuilder.
		Select("COALESCE(SUM(total_rx_bytes), 0)", "COALESCE(SUM(total_tx_bytes), 0)").
		From("vnstat_bandwidth_hourly").
		Where(sq.And{
			sq.Eq{"agent_id": agentID},
			sq.Lt{"hour_start": currentHourStart}, // Exclude current hour to avoid double counting
		}).
		RunWith(s.db).QueryRowContext(ctx).Scan(&allTimeRx, &allTimeTx)

	if err != nil {
		log.Error().Err(err).Msg("Failed to calculate all time usage")
	}

	// Add current hour raw data to all time total
	allTimeRx += thisHourRx
	allTimeTx += thisHourTx

	usage["All Time"] = struct {
		Download int64 `json:"download"`
		Upload   int64 `json:"upload"`
		Total    int64 `json:"total"`
	}{allTimeRx, allTimeTx, allTimeRx + allTimeTx}

	return usage, nil
}

// BulkInsertVnstatBandwidth inserts multiple bandwidth records in a single transaction
func (s *service) BulkInsertVnstatBandwidth(ctx context.Context, records []types.VnstatBandwidth) error {
	if len(records) == 0 {
		return nil
	}

	// Start transaction
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	// Process in batches to avoid memory issues
	const batchSize = 1000
	for i := 0; i < len(records); i += batchSize {
		end := i + batchSize
		if end > len(records) {
			end = len(records)
		}
		batch := records[i:end]

		// Build bulk insert query
		query := s.sqlBuilder.
			Insert("vnstat_bandwidth").
			Columns("agent_id", "rx_bytes_per_second", "tx_bytes_per_second", "rx_packets_per_second", "tx_packets_per_second", "rx_rate_string", "tx_rate_string", "created_at")

		for _, record := range batch {
			var createdAt interface{}
			if s.config.Type == config.SQLite {
				createdAt = record.CreatedAt.Format(time.RFC3339)
			} else {
				createdAt = record.CreatedAt
			}

			query = query.Values(
				record.AgentID,
				record.RxBytesPerSecond,
				record.TxBytesPerSecond,
				record.RxPacketsPerSecond,
				record.TxPacketsPerSecond,
				record.RxRateString,
				record.TxRateString,
				createdAt,
			)
		}

		// Execute batch insert
		if _, err := query.RunWith(tx).ExecContext(ctx); err != nil {
			return fmt.Errorf("failed to insert batch: %w", err)
		}

		log.Debug().Int("batch_size", len(batch)).Int("total_processed", end).Msg("Inserted vnstat bandwidth batch")
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	log.Info().Int("total_records", len(records)).Msg("Bulk inserted vnstat bandwidth records")
	return nil
}
