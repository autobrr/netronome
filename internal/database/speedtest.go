// Copyright (c) 2024-2026, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package database

import (
	"cmp"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"time"

	sq "github.com/Masterminds/squirrel"

	"github.com/autobrr/netronome/internal/config"
	"github.com/autobrr/netronome/internal/types"
)

// SpeedtestServer is retained Speedtest.net discovery metadata. ObservedAt orders
// competing refreshes so an older request cannot overwrite newer server details.
type SpeedtestServer struct {
	ID         string
	Name       string
	Host       string
	Country    string
	Sponsor    string
	URL        string
	Latitude   float64
	Longitude  float64
	ObservedAt time.Time
}

// SpeedtestServerSource records the last completed discovery for one source.
// Latitude and Longitude are populated together for the detected local origin.
type SpeedtestServerSource struct {
	Key       string
	UpdatedAt time.Time
	Latitude  *float64
	Longitude *float64
}

// ListSpeedtestServers returns every retained Speedtest.net server ordered by ID.
func (s *service) ListSpeedtestServers(ctx context.Context) ([]SpeedtestServer, error) {
	rows, err := s.sqlBuilder.
		Select("id", "name", "host", "country", "sponsor", "url", "latitude", "longitude", "observed_at").
		From("speedtest_servers").
		OrderBy("id").
		RunWith(s.db).
		QueryContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to query speedtest servers: %w", err)
	}
	defer rows.Close()

	servers := make([]SpeedtestServer, 0)
	for rows.Next() {
		var server SpeedtestServer
		if err := rows.Scan(
			&server.ID,
			&server.Name,
			&server.Host,
			&server.Country,
			&server.Sponsor,
			&server.URL,
			&server.Latitude,
			&server.Longitude,
			&server.ObservedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan speedtest server: %w", err)
		}
		server.ObservedAt = server.ObservedAt.UTC()
		servers = append(servers, server)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate speedtest servers: %w", err)
	}
	return servers, nil
}

// GetSpeedtestServerSource returns durable discovery metadata for key. The bool is
// false when the source has never completed successfully.
func (s *service) GetSpeedtestServerSource(ctx context.Context, key string) (SpeedtestServerSource, bool, error) {
	query := s.sqlBuilder.
		Select("source_key", "updated_at", "latitude", "longitude").
		From("speedtest_server_sources").
		Where(sq.Eq{"source_key": key})

	var source SpeedtestServerSource
	var latitude, longitude sql.NullFloat64
	err := query.RunWith(s.db).QueryRowContext(ctx).Scan(
		&source.Key,
		&source.UpdatedAt,
		&latitude,
		&longitude,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return SpeedtestServerSource{}, false, nil
	}
	if err != nil {
		return SpeedtestServerSource{}, false, fmt.Errorf("failed to query speedtest server source %q: %w", key, err)
	}

	source.UpdatedAt = source.UpdatedAt.UTC()
	if latitude.Valid && longitude.Valid {
		source.Latitude = new(latitude.Float64)
		source.Longitude = new(longitude.Float64)
	}
	return source, true, nil
}

// SaveSpeedtestServerCatalogue atomically retains discovered servers and, when
// source is non-nil, records that the source refresh completed successfully.
// Older observations never replace newer rows; any error rolls back the batch.
func (s *service) SaveSpeedtestServerCatalogue(
	ctx context.Context,
	servers []SpeedtestServer,
	source *SpeedtestServerSource,
) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin speedtest catalogue transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	orderedServers := slices.Clone(servers)
	slices.SortFunc(orderedServers, func(a, b SpeedtestServer) int {
		return cmp.Compare(a.ID, b.ID)
	})
	for _, server := range orderedServers {
		query := s.sqlBuilder.
			Insert("speedtest_servers").
			Columns("id", "name", "host", "country", "sponsor", "url", "latitude", "longitude", "observed_at").
			Values(
				server.ID,
				server.Name,
				server.Host,
				server.Country,
				server.Sponsor,
				server.URL,
				server.Latitude,
				server.Longitude,
				server.ObservedAt.UTC(),
			).
			Suffix(`ON CONFLICT (id) DO UPDATE SET
				name = EXCLUDED.name,
				host = EXCLUDED.host,
				country = EXCLUDED.country,
				sponsor = EXCLUDED.sponsor,
				url = EXCLUDED.url,
				latitude = EXCLUDED.latitude,
				longitude = EXCLUDED.longitude,
				observed_at = EXCLUDED.observed_at
			WHERE speedtest_servers.observed_at < EXCLUDED.observed_at`)
		if _, err := query.RunWith(tx).ExecContext(ctx); err != nil {
			return fmt.Errorf("failed to upsert speedtest server %q: %w", server.ID, err)
		}
	}

	if source != nil {
		query := s.sqlBuilder.
			Insert("speedtest_server_sources").
			Columns("source_key", "updated_at", "latitude", "longitude").
			Values(source.Key, source.UpdatedAt.UTC(), source.Latitude, source.Longitude).
			Suffix(`ON CONFLICT (source_key) DO UPDATE SET
				updated_at = EXCLUDED.updated_at,
				latitude = EXCLUDED.latitude,
				longitude = EXCLUDED.longitude
			WHERE speedtest_server_sources.updated_at < EXCLUDED.updated_at`)
		if _, err := query.RunWith(tx).ExecContext(ctx); err != nil {
			return fmt.Errorf("failed to upsert speedtest server source %q: %w", source.Key, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit speedtest catalogue: %w", err)
	}
	return nil
}

func (s *service) SaveSpeedTest(ctx context.Context, result types.SpeedTestResult) (*types.SpeedTestResult, error) {
	data := map[string]interface{}{
		"server_name":    result.ServerName,
		"server_id":      result.ServerID,
		"server_host":    result.ServerHost,
		"server_city":    result.ServerCity,
		"test_type":      result.TestType,
		"download_speed": result.DownloadSpeed,
		"upload_speed":   result.UploadSpeed,
		"latency":        result.Latency,
		"jitter":         result.Jitter,
		"is_scheduled":   result.IsScheduled,
	}

	// Use provided created_at if available, otherwise default to current UTC time
	if result.CreatedAt.IsZero() {
		result.CreatedAt = time.Now().UTC()
	} else {
		result.CreatedAt = result.CreatedAt.UTC()
	}
	data["created_at"] = result.CreatedAt

	var id int64

	switch s.config.Type {
	case config.Postgres:
		query := s.sqlBuilder.Insert("speed_tests").
			SetMap(data).
			Suffix("RETURNING id")

		sqlStr, args, err := query.ToSql()
		if err != nil {
			return nil, fmt.Errorf("failed to build query: %w", err)
		}

		err = s.db.QueryRowContext(ctx, sqlStr, args...).Scan(&id)
		if err != nil {
			return nil, fmt.Errorf("failed to save speed test: %w", err)
		}

	case config.SQLite:
		res, err := s.insert(ctx, "speed_tests", data)
		if err != nil {
			return nil, fmt.Errorf("failed to save speed test: %w", err)
		}

		id, err = res.LastInsertId()
		if err != nil {
			return nil, fmt.Errorf("failed to get last insert ID: %w", err)
		}
	}

	result.ID = id
	return &result, nil
}

type speedTestResultIdentity struct {
	testType   string
	serverName string
	serverID   string
	serverHost *string
}

type speedTestResultName struct {
	testType   string
	serverName string
}

func (i speedTestResultIdentity) nameKey() speedTestResultName {
	return speedTestResultName{testType: i.testType, serverName: i.serverName}
}

// isLegacy reports whether an identity has the storage shape used before stable server IDs.
func (i speedTestResultIdentity) isLegacy() bool {
	switch i.testType {
	case "speedtest":
		return i.serverHost == nil && i.serverID == i.serverName
	case "librespeed":
		return i.serverHost != nil && *i.serverHost == i.serverName && i.serverID == types.LibrespeedServerIDPrefix+i.serverName
	default:
		return false
	}
}

// speedTestResultAliases resolves requested legacy names only when all stored history agrees on one stable ID.
func (s *service) speedTestResultAliases(ctx context.Context, legacyNames map[speedTestResultName]struct{}) (map[speedTestResultName]string, error) {
	nameConditions := make(sq.Or, 0, len(legacyNames))
	for name := range legacyNames {
		nameConditions = append(nameConditions, sq.And{
			sq.Eq{"test_type": name.testType},
			sq.Eq{"server_name": name.serverName},
		})
	}

	rows, err := s.sqlBuilder.
		Select("test_type", "server_name", "server_id", "server_host").
		Distinct().
		From("speed_tests").
		Where(nameConditions).
		RunWith(s.db).
		QueryContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to query speed test server identities: %w", err)
	}
	defer rows.Close()

	stableIDsByName := make(map[speedTestResultName]map[string]struct{})
	for rows.Next() {
		var identity speedTestResultIdentity
		if err := rows.Scan(&identity.testType, &identity.serverName, &identity.serverID, &identity.serverHost); err != nil {
			return nil, fmt.Errorf("failed to scan speed test server identity: %w", err)
		}
		if identity.serverName == "" || identity.serverID == "" || identity.isLegacy() {
			continue
		}
		stableIDs := stableIDsByName[identity.nameKey()]
		if stableIDs == nil {
			stableIDs = make(map[string]struct{})
			stableIDsByName[identity.nameKey()] = stableIDs
		}
		stableIDs[identity.serverID] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate speed test server identities: %w", err)
	}

	aliases := make(map[speedTestResultName]string)
	for nameKey, stableIDs := range stableIDsByName {
		if len(stableIDs) != 1 {
			continue
		}
		for stableID := range stableIDs {
			aliases[nameKey] = stableID
		}
	}
	return aliases, nil
}

func (s *service) GetSpeedTests(ctx context.Context, timeRange string, page, limit int) (*types.PaginatedSpeedTests, error) {
	baseQuery := s.sqlBuilder.Select().From("speed_tests")

	if timeRange != "all" {
		var timeExpr string
		switch s.config.Type {
		case config.Postgres:
			switch timeRange {
			case "24h", "1d":
				timeExpr = "NOW() - INTERVAL '1 day'"
			case "3d":
				timeExpr = "NOW() - INTERVAL '3 days'"
			case "week", "1w":
				timeExpr = "NOW() - INTERVAL '7 days'"
			case "month", "1m":
				timeExpr = "NOW() - INTERVAL '1 month'"
			}
		case config.SQLite:
			switch timeRange {
			case "24h", "1d":
				timeExpr = "datetime('now', '-1 day')"
			case "3d":
				timeExpr = "datetime('now', '-3 days')"
			case "week", "1w":
				timeExpr = "datetime('now', '-7 days')"
			case "month", "1m":
				timeExpr = "datetime('now', '-1 month')"
			}
		}
		if timeExpr != "" {
			baseQuery = baseQuery.Where("created_at >= " + timeExpr)
		}
	}

	countQuery := baseQuery.Columns("COUNT(*)")
	var total int
	err := countQuery.RunWith(s.db).QueryRowContext(ctx).Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("failed to get total count: %w", err)
	}
	// Get paginated results
	dataQuery := baseQuery.Columns(
		"id",
		"server_name",
		"server_id",
		"server_host",
		"server_city",
		"test_type",
		"download_speed",
		"upload_speed",
		"latency",
		"jitter",
		"is_scheduled",
		"created_at",
	).
		OrderBy("created_at DESC").
		Limit(uint64(limit)).
		Offset(uint64((page - 1) * limit))

	rows, err := dataQuery.RunWith(s.db).QueryContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to query speed tests: %w", err)
	}
	defer rows.Close()

	results := make([]types.SpeedTestResult, 0)
	legacyNames := make(map[speedTestResultName]struct{})
	for rows.Next() {
		var result types.SpeedTestResult
		err := rows.Scan(
			&result.ID,
			&result.ServerName,
			&result.ServerID,
			&result.ServerHost,
			&result.ServerCity,
			&result.TestType,
			&result.DownloadSpeed,
			&result.UploadSpeed,
			&result.Latency,
			&result.Jitter,
			&result.IsScheduled,
			&result.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan speed test result: %w", err)
		}

		result.CreatedAt = result.CreatedAt.UTC()
		identity := speedTestResultIdentity{
			testType:   result.TestType,
			serverName: result.ServerName,
			serverID:   result.ServerID,
			serverHost: result.ServerHost,
		}
		if identity.isLegacy() {
			legacyNames[identity.nameKey()] = struct{}{}
		}
		results = append(results, result)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating speed test results: %w", err)
	}

	if len(legacyNames) > 0 {
		aliases, err := s.speedTestResultAliases(ctx, legacyNames)
		if err != nil {
			return nil, err
		}
		for i := range results {
			identity := speedTestResultIdentity{
				testType:   results[i].TestType,
				serverName: results[i].ServerName,
				serverID:   results[i].ServerID,
				serverHost: results[i].ServerHost,
			}
			if identity.isLegacy() {
				if stableID, ok := aliases[identity.nameKey()]; ok {
					results[i].ServerID = stableID
				}
			}
		}
	}

	return &types.PaginatedSpeedTests{
		Data:  results,
		Total: total,
		Page:  page,
		Limit: limit,
	}, nil
}
