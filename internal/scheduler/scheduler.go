// Copyright (c) 2024, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package scheduler

import (
	"context"
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/autobrr/netronome/internal/database"
	"github.com/autobrr/netronome/internal/speedtest"
	"github.com/autobrr/netronome/internal/types"
)

type Service interface {
	Start(ctx context.Context)
	Stop()
}

type service struct {
	db        database.Service
	speedtest speedtest.Service
	ticker    *time.Ticker
	done      chan bool
	mu        sync.Mutex
	running   bool
}

func New(db database.Service, speedtest speedtest.Service) Service {
	return &service{
		db:        db,
		speedtest: speedtest,
		done:      make(chan bool),
	}
}

func (s *service) Start(ctx context.Context) {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return
	}
	s.running = true
	s.mu.Unlock()

	s.ticker = time.NewTicker(1 * time.Minute)

	// Run immediately on start
	s.checkAndRunScheduledTests(ctx)

	go func() {
		for {
			select {
			case <-ctx.Done():
				s.Stop()
				return
			case <-s.done:
				return
			case <-s.ticker.C:
				s.checkAndRunScheduledTests(ctx)
			}
		}
	}()

	log.Info().Msg("Scheduler service started")
}

func (s *service) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return
	}

	if s.ticker != nil {
		s.ticker.Stop()
	}
	s.done <- true
	s.running = false
	log.Info().Msg("Scheduler service stopped")
}

func (s *service) checkAndRunScheduledTests(ctx context.Context) {
	schedules, err := s.db.GetSchedules(ctx)
	if err != nil {
		log.Error().
			Err(err).
			Msg("Error fetching schedules")
		return
	}

	now := time.Now()
	for _, schedule := range schedules {
		if !schedule.Enabled || schedule.NextRun.After(now) {
			continue
		}

		log.Info().
			Int64("schedule_id", schedule.ID).
			Time("next_run", schedule.NextRun).
			Str("interval", schedule.Interval).
			Bool("is_iperf", schedule.Options.UseIperf).
			Msg("Running scheduled test")

		testCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		go func(schedule types.Schedule, ctx context.Context, cancel context.CancelFunc) {
			defer cancel()
			schedule.Options.IsScheduled = true
			result, err := s.speedtest.RunTest(&schedule.Options)
			if err != nil {
				log.Error().
					Err(err).
					Int64("schedule_id", schedule.ID).
					Msg("Error running scheduled test")
				return
			}

			log.Info().
				Int64("schedule_id", schedule.ID).
				Float64("download_speed", result.DownloadSpeed).
				Float64("upload_speed", result.UploadSpeed).
				Msg("Scheduled test completed")

			duration, err := time.ParseDuration(schedule.Interval)
			if err != nil {
				log.Error().
					Err(err).
					Int64("schedule_id", schedule.ID).
					Msg("Error parsing interval")
				return
			}

			schedule.LastRun = &now
			schedule.NextRun = now.Add(duration)

			if err := s.db.UpdateSchedule(ctx, schedule); err != nil {
				log.Error().
					Err(err).
					Int64("schedule_id", schedule.ID).
					Msg("Error updating schedule")
			}
		}(schedule, testCtx, cancel)
	}
}
