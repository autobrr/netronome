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
}

func New(db database.Service, speedtest speedtest.Service) Service {
	return &service{
		db:        db,
		speedtest: speedtest,
		done:      make(chan bool),
	}
}

func (s *service) Start(ctx context.Context) {
	s.ticker = time.NewTicker(1 * time.Minute)
	go func() {
		for {
			select {
			case <-s.ticker.C:
				s.checkAndRunScheduledTests(ctx)
			case <-s.done:
				s.ticker.Stop()
				return
			}
		}
	}()
}

func (s *service) Stop() {
	s.done <- true
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

		testCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		go func(schedule types.Schedule, ctx context.Context, cancel context.CancelFunc) {
			defer cancel()
			schedule.Options.IsScheduled = true
			_, err := s.speedtest.RunTest(&schedule.Options)
			if err != nil {
				log.Error().
					Err(err).
					Msg("Error running scheduled test")
				return
			}

			duration, err := time.ParseDuration(schedule.Interval)
			if err != nil {
				log.Error().
					Err(err).
					Msg("Error parsing interval")
				return
			}

			schedule.LastRun = &now
			schedule.NextRun = now.Add(duration)

			if err := s.db.UpdateSchedule(ctx, schedule); err != nil {
				log.Error().
					Err(err).
					Msg("Error updating schedule")
			}
		}(schedule, testCtx, cancel)
	}
}
