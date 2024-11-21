package scheduler

import (
	"context"
	"log"
	"sync"
	"time"

	"speedtrackerr/internal/database"
	"speedtrackerr/internal/speedtest"
	"speedtrackerr/internal/types"
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
		log.Printf("Error fetching schedules: %v", err)
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
			_, err := s.speedtest.RunTest(&schedule.Options)
			if err != nil {
				log.Printf("Error running scheduled test: %v", err)
				return
			}

			duration, err := time.ParseDuration(schedule.Interval)
			if err != nil {
				log.Printf("Error parsing interval: %v", err)
				return
			}

			schedule.LastRun = &now
			schedule.NextRun = now.Add(duration)

			if err := s.db.UpdateSchedule(ctx, schedule); err != nil {
				log.Printf("Error updating schedule: %v", err)
			}
		}(schedule, testCtx, cancel)
	}
}
