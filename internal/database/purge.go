// Copyright (c) 2024-2026, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package database

import (
	"context"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/rs/zerolog/log"
)

// PurgeHistoricalData deletes speed test and packet loss results older than the
// given cutoff. On SQLite it runs VACUUM afterwards so the on-disk file actually
// shrinks (VACUUM cannot run inside a transaction). A failed VACUUM is logged but
// does not fail the call since the rows are already gone.
func (s *service) PurgeHistoricalData(ctx context.Context, before time.Time) (speedTests int64, packetLoss int64, err error) {
	log.Info().Time("before", before).Msg("Purging historical data")

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, 0, err
	}
	defer tx.Rollback()

	stResult, err := s.sqlBuilder.Delete("speed_tests").Where(sq.Lt{"created_at": before}).RunWith(tx).ExecContext(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Failed to purge speed tests")
		return 0, 0, err
	}
	speedTests, _ = stResult.RowsAffected()

	plResult, err := s.sqlBuilder.Delete("packet_loss_results").Where(sq.Lt{"created_at": before}).RunWith(tx).ExecContext(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Failed to purge packet loss results")
		return 0, 0, err
	}
	packetLoss, _ = plResult.RowsAffected()

	if err := tx.Commit(); err != nil {
		log.Error().Err(err).Msg("Failed to commit purge transaction")
		return 0, 0, err
	}

	log.Info().Int64("speed_tests", speedTests).Int64("packet_loss", packetLoss).Msg("Purged historical data")

	if s.config.Type == "sqlite" {
		if _, err := s.db.ExecContext(ctx, "VACUUM"); err != nil {
			log.Warn().Err(err).Msg("Failed to VACUUM after purge; database file will not shrink")
		} else if _, err := s.db.ExecContext(ctx, "PRAGMA wal_checkpoint(TRUNCATE)"); err != nil {
			// in WAL mode the file only shrinks once the WAL is checkpointed
			log.Warn().Err(err).Msg("Failed to checkpoint WAL after purge")
		}
	}

	return speedTests, packetLoss, nil
}
