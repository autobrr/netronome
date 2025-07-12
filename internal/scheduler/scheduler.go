// Copyright (c) 2024-2025, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package scheduler

import (
	"context"
	"math"
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/autobrr/netronome/internal/database"
	"github.com/autobrr/netronome/internal/notifications"
	"github.com/autobrr/netronome/internal/speedtest"
	"github.com/autobrr/netronome/internal/types"
)

type Service interface {
	Start(ctx context.Context)
	Stop()
}

type service struct {
	db         database.Service
	speedtest  speedtest.Service
	packetLoss *speedtest.PacketLossService
	notifier   *notifications.Notifier
	ticker     *time.Ticker
	done       chan bool
	mu         sync.Mutex
	running    bool
}

func New(db database.Service, speedtest speedtest.Service, packetLoss *speedtest.PacketLossService, notifier *notifications.Notifier) Service {
	return &service{
		db:         db,
		speedtest:  speedtest,
		packetLoss: packetLoss,
		notifier:   notifier,
		done:       make(chan bool),
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

	// Initialize schedules before starting
	s.initializeSchedules(ctx)
	s.initializePacketLossMonitors(ctx)

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
				s.checkAndRunPacketLossMonitors(ctx)
			}
		}
	}()
}

// initializeSchedules prepares schedules on startup without running tests immediately
func (s *service) initializeSchedules(ctx context.Context) {
	schedules, err := s.db.GetSchedules(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Error fetching schedules during initialization")
		return
	}

	now := time.Now()
	for _, schedule := range schedules {
		if !schedule.Enabled {
			continue
		}

		// Parse the interval (either duration or exact time)
		if !s.isValidScheduleInterval(schedule.Interval) {
			log.Error().
				Int64("schedule_id", schedule.ID).
				Str("interval", schedule.Interval).
				Msg("Invalid schedule interval during initialization")
			continue
		}

		// If NextRun is in the past, calculate new NextRun
		if schedule.NextRun.Before(now) {
			nextRun := s.calculateNextRun(schedule.Interval, now)
			if nextRun.IsZero() {
				log.Error().
					Int64("schedule_id", schedule.ID).
					Str("interval", schedule.Interval).
					Msg("Could not calculate next run time")
				continue
			}

			schedule.NextRun = nextRun

			log.Info().
				Int64("schedule_id", schedule.ID).
				Time("next_run", schedule.NextRun).
				Str("interval", schedule.Interval).
				Msg("Rescheduling test")

			if err := s.db.UpdateSchedule(ctx, schedule); err != nil {
				log.Error().
					Err(err).
					Int64("schedule_id", schedule.ID).
					Msg("Error updating schedule during initialization")
			}
		}
	}
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
			result, err := s.speedtest.RunTest(ctx, &schedule.Options)
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

			nextRun := s.calculateNextRun(schedule.Interval, now)
			if nextRun.IsZero() {
				log.Error().
					Int64("schedule_id", schedule.ID).
					Str("interval", schedule.Interval).
					Msg("Error calculating next run time")
				return
			}

			schedule.LastRun = &now
			schedule.NextRun = nextRun

			if err := s.db.UpdateSchedule(ctx, schedule); err != nil {
				log.Error().
					Err(err).
					Int64("schedule_id", schedule.ID).
					Msg("Error updating schedule")
			}
		}(schedule, testCtx, cancel)
	}
}

// isValidScheduleInterval checks if the interval is valid (duration or exact time)
func (s *service) isValidScheduleInterval(interval string) bool {
	if strings.HasPrefix(interval, "exact:") {
		// Extract time part and validate - supports multiple times
		timePart := strings.TrimPrefix(interval, "exact:")
		times := strings.Split(timePart, ",")

		if len(times) == 0 {
			return false
		}

		for _, timeStr := range times {
			parts := strings.Split(strings.TrimSpace(timeStr), ":")
			if len(parts) != 2 {
				return false
			}

			hour, err := strconv.Atoi(parts[0])
			if err != nil || hour < 0 || hour > 23 {
				return false
			}

			minute, err := strconv.Atoi(parts[1])
			if err != nil || minute < 0 || minute > 59 {
				return false
			}
		}

		return true
	} else {
		// Try to parse as duration
		_, err := time.ParseDuration(interval)
		return err == nil
	}
}

// calculateNextRun calculates the next run time based on interval type
func (s *service) calculateNextRun(interval string, from time.Time) time.Time {
	if strings.HasPrefix(interval, "exact:") {
		// Extract time part - supports multiple times separated by comma
		timePart := strings.TrimPrefix(interval, "exact:")
		times := strings.Split(timePart, ",")

		var nextRun time.Time
		minTimeDiff := time.Duration(math.MaxInt64)

		// Find the next upcoming time from the list
		for _, timeStr := range times {
			parts := strings.Split(strings.TrimSpace(timeStr), ":")
			if len(parts) != 2 {
				continue
			}

			hour, err := strconv.Atoi(parts[0])
			if err != nil || hour < 0 || hour > 23 {
				continue
			}

			minute, err := strconv.Atoi(parts[1])
			if err != nil || minute < 0 || minute > 59 {
				continue
			}

			// Check today
			todayRun := time.Date(from.Year(), from.Month(), from.Day(), hour, minute, 0, 0, from.Location())
			if todayRun.After(from) {
				diff := todayRun.Sub(from)
				if diff < minTimeDiff {
					minTimeDiff = diff
					nextRun = todayRun
				}
			}

			// Check tomorrow
			tomorrowRun := todayRun.Add(24 * time.Hour)
			diff := tomorrowRun.Sub(from)
			if diff < minTimeDiff {
				minTimeDiff = diff
				nextRun = tomorrowRun
			}
		}

		if nextRun.IsZero() {
			return time.Time{}
		}

		// Add small random jitter (1-60 seconds) to prevent thundering herd
		jitter := time.Duration(rand.Int63n(60)+1) * time.Second
		return nextRun.Add(jitter)
	} else {
		// Parse as duration
		duration, err := time.ParseDuration(interval)
		if err != nil {
			return time.Time{}
		}

		// Add small random jitter (1-300 seconds) to prevent thundering herd
		jitter := time.Duration(rand.Int63n(300)+1) * time.Second
		return from.Add(duration).Add(jitter)
	}
}

// initializePacketLossMonitors prepares packet loss monitors on startup
func (s *service) initializePacketLossMonitors(ctx context.Context) {
	monitors, err := s.db.GetPacketLossMonitors()
	if err != nil {
		log.Error().Err(err).Msg("Error fetching packet loss monitors during initialization")
		return
	}

	now := time.Now()
	for _, monitor := range monitors {
		if !monitor.Enabled {
			continue
		}

		// Parse the interval (either duration or exact time)
		if !s.isValidScheduleInterval(monitor.Interval) {
			log.Error().
				Int64("monitor_id", monitor.ID).
				Str("interval", monitor.Interval).
				Msg("Invalid monitor interval during initialization")
			continue
		}

		// If NextRun is nil or in the past, calculate new NextRun
		if monitor.NextRun == nil || monitor.NextRun.Before(now) {
			nextRun := s.calculateNextRun(monitor.Interval, now)
			if nextRun.IsZero() {
				log.Error().
					Int64("monitor_id", monitor.ID).
					Str("interval", monitor.Interval).
					Msg("Could not calculate next run time for monitor")
				continue
			}

			monitor.NextRun = &nextRun

			log.Info().
				Int64("monitor_id", monitor.ID).
				Time("next_run", nextRun).
				Str("interval", monitor.Interval).
				Msg("Rescheduling packet loss monitor")

			if err := s.db.UpdatePacketLossMonitor(monitor); err != nil {
				log.Error().
					Err(err).
					Int64("monitor_id", monitor.ID).
					Msg("Error updating monitor during initialization")
			}
		}
	}
}

// checkAndRunPacketLossMonitors checks for due packet loss monitors and runs them
func (s *service) checkAndRunPacketLossMonitors(ctx context.Context) {
	monitors, err := s.db.GetPacketLossMonitors()
	if err != nil {
		log.Error().
			Err(err).
			Msg("Error fetching packet loss monitors")
		return
	}

	now := time.Now()
	for _, monitor := range monitors {
		if !monitor.Enabled || monitor.NextRun == nil || monitor.NextRun.After(now) {
			continue
		}

		log.Info().
			Int64("monitor_id", monitor.ID).
			Str("host", monitor.Host).
			Time("next_run", *monitor.NextRun).
			Str("interval", monitor.Interval).
			Msg("Running scheduled packet loss test")

		// Create a timeout context for the test
		testCtx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		go func(monitor *types.PacketLossMonitor, ctx context.Context, cancel context.CancelFunc) {
			defer cancel()

			// Run the packet loss test
			if s.packetLoss != nil {
				s.packetLoss.RunScheduledTest(monitor)
			}

			// Calculate and update next run time
			nextRun := s.calculateNextRun(monitor.Interval, now)
			if nextRun.IsZero() {
				log.Error().
					Int64("monitor_id", monitor.ID).
					Str("interval", monitor.Interval).
					Msg("Error calculating next run time for monitor")
				return
			}

			monitor.LastRun = &now
			monitor.NextRun = &nextRun

			if err := s.db.UpdatePacketLossMonitor(monitor); err != nil {
				log.Error().
					Err(err).
					Int64("monitor_id", monitor.ID).
					Msg("Error updating monitor schedule")
			}
		}(monitor, testCtx, cancel)
	}
}
