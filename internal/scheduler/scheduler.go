// Copyright (c) 2024-2026, s0up and the autobrr contributors.
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
	"github.com/autobrr/netronome/internal/dnsmonitor"
	"github.com/autobrr/netronome/internal/notifications"
	"github.com/autobrr/netronome/internal/speedtest"
)

type Service interface {
	Start(ctx context.Context)
	Stop()
	UpdateMonitorSchedule(monitorID int64, interval string) error
	CalculateNextRun(interval string, from time.Time) time.Time
}

type service struct {
	db         database.Service
	speedtest  speedtest.Service
	packetLoss *speedtest.PacketLossService
	dns        *dnsmonitor.Service
	notifier   *notifications.Notifier
	ticker     *time.Ticker
	done       chan bool
	mu         sync.Mutex
	running    bool
	inFlight   sync.Map // runKey -> struct{}, one entry per test that still runs
}

// runKey names one schedule or monitor in inFlight.
type runKey struct {
	kind string
	id   int64
}

func New(db database.Service, speedtest speedtest.Service, packetLoss *speedtest.PacketLossService, dns *dnsmonitor.Service, notifier *notifications.Notifier) Service {
	return &service{
		db:         db,
		speedtest:  speedtest,
		packetLoss: packetLoss,
		dns:        dns,
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
	s.initializeDNSMonitors()

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
				s.checkAndRunDNSMonitors()
			}
		}
	}()
}

// initializeSchedules prepares schedules on startup without running tests immediately.
// This function recalculates next run times for all enabled schedules.
//
// Important behavior:
// - Missed runs are NOT executed (no catch-up mechanism)
// - For exact times (e.g., "exact:14:00"), finds the next occurrence
// - For durations (e.g., "1h"), schedules from the current time
// - This prevents network flooding after downtime and ensures fresh data

func (s *service) initializeSchedules(ctx context.Context) {
	schedules, err := s.db.GetSchedules(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Error fetching schedules during initialization")
		return
	}

	now := time.Now().UTC()
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
			nextRun := s.calculateNextRun(schedule.Interval, now, false)
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

	now := time.Now().UTC()
	for _, schedule := range schedules {
		if !schedule.Enabled || schedule.NextRun.After(now) {
			continue
		}

		scheduledStart := schedule.NextRun.UTC()
		if schedule.NextRun.IsZero() {
			scheduledStart = now
		}

		// claim the next run before the test starts, so the next tick does
		// not start the test again and a failed test waits for its next slot
		nextRun := s.nextRunAfter(schedule.Interval, scheduledStart, now, false)
		if nextRun.IsZero() {
			log.Error().
				Int64("schedule_id", schedule.ID).
				Str("interval", schedule.Interval).
				Msg("Error calculating next run time")
			continue
		}
		schedule.NextRun = nextRun
		if err := s.db.UpdateSchedule(ctx, schedule); err != nil {
			log.Error().
				Err(err).
				Int64("schedule_id", schedule.ID).
				Msg("Error updating schedule")
			continue
		}

		// if the previous test still runs, skip this slot
		key := runKey{"speedtest", schedule.ID}
		if _, running := s.inFlight.LoadOrStore(key, struct{}{}); running {
			log.Warn().
				Int64("schedule_id", schedule.ID).
				Msg("Previous scheduled test still running, skipping this run")
			continue
		}

		log.Info().
			Int64("schedule_id", schedule.ID).
			Time("scheduled_start_utc", scheduledStart).
			Time("next_run_utc", nextRun).
			Str("interval", schedule.Interval).
			Bool("is_iperf", schedule.Options.UseIperf).
			Msg("Running scheduled test")

		go func() {
			defer s.inFlight.Delete(key)
			testCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer cancel()

			schedule.Options.IsScheduled = true
			result, err := s.speedtest.RunTest(testCtx, &schedule.Options)
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
		}()
	}
}

// nextRunAfter returns the next run that follows scheduledStart. If that time
// is not after now, it returns the next run that follows now.
func (s *service) nextRunAfter(interval string, scheduledStart, now time.Time, skipJitter bool) time.Time {
	nextRun := s.calculateNextRun(interval, scheduledStart, skipJitter)
	if !nextRun.IsZero() && !nextRun.After(now) {
		nextRun = s.calculateNextRun(interval, now, skipJitter)
	}
	return nextRun
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
		// Try to parse as duration, but first normalize custom units
		normalizedInterval := s.normalizeDuration(interval)
		_, err := time.ParseDuration(normalizedInterval)
		return err == nil
	}
}

// normalizeDuration converts custom duration units (d, w) to Go-compatible units
func (s *service) normalizeDuration(interval string) string {
	// Handle days (d) - convert to hours
	if strings.HasSuffix(interval, "d") {
		if daysStr := strings.TrimSuffix(interval, "d"); daysStr != "" {
			if days, err := strconv.Atoi(daysStr); err == nil {
				return strconv.Itoa(days*24) + "h"
			}
		}
	}

	// Handle weeks (w) - convert to hours
	if strings.HasSuffix(interval, "w") {
		if weeksStr := strings.TrimSuffix(interval, "w"); weeksStr != "" {
			if weeks, err := strconv.Atoi(weeksStr); err == nil {
				return strconv.Itoa(weeks*24*7) + "h"
			}
		}
	}

	// Return as-is for standard Go durations
	return interval
}

// calculateNextRun calculates the next run time based on interval type.
// Supports two interval formats:
// 1. Duration-based: Standard Go duration strings (e.g., "30s", "5m", "1h")
//   - Next run = current time + duration + optional random jitter (1-300 seconds)
//
// 2. Exact time: "exact:HH:MM" or "exact:HH:MM,HH:MM" for multiple times
//   - Next run = next occurrence of specified time + optional random jitter (1-60 seconds)
//
// The jitter prevents thundering herd problem when multiple monitors have the same interval.
// Set skipJitter=true for packet loss monitors that need precise timing.
func (s *service) calculateNextRun(interval string, from time.Time, skipJitter bool) time.Time {
	// Ensure we're working in UTC
	from = from.UTC()

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

			// Check today - interpret times as UTC since frontend sends UTC
			todayRun := time.Date(from.Year(), from.Month(), from.Day(), hour, minute, 0, 0, time.UTC)
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

		if skipJitter {
			// Return precise time for packet loss monitors
			return nextRun
		} else {
			// Add small random jitter (1-60 seconds) to prevent thundering herd
			jitter := time.Duration(rand.Int63n(60)+1) * time.Second
			return nextRun.Add(jitter)
		}
	} else {
		// Parse as duration - normalize custom units first
		normalizedInterval := s.normalizeDuration(interval)
		duration, err := time.ParseDuration(normalizedInterval)
		if err != nil {
			return time.Time{}
		}

		if skipJitter {
			// Return precise time for packet loss monitors
			return from.Add(duration)
		} else {
			// Add small random jitter (1-300 seconds) to prevent thundering herd
			jitter := time.Duration(rand.Int63n(300)+1) * time.Second
			return from.Add(duration).Add(jitter)
		}
	}
}

// initializePacketLossMonitors prepares packet loss monitors on startup.
// This function recalculates next run times for all enabled monitors.
//
// Important behavior:
// - Missed runs are NOT executed (no catch-up mechanism)
// - For exact times (e.g., "exact:14:00"), finds the next occurrence
// - For durations (e.g., "1h"), schedules from the current time
// - This prevents network flooding after downtime and ensures fresh data
//
// Example: If a monitor scheduled for "exact:14:00" starts at 15:00,
// it will be scheduled for 14:00 the next day, not run immediately.
func (s *service) initializePacketLossMonitors(ctx context.Context) {
	monitors, err := s.db.GetPacketLossMonitors()
	if err != nil {
		log.Error().Err(err).Msg("Error fetching packet loss monitors during initialization")
		return
	}

	now := time.Now().UTC()
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
			nextRun := s.calculateNextRun(monitor.Interval, now, true)
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

	now := time.Now().UTC()
	log.Info().
		Time("scheduler_check_time_utc", now).
		Int("total_monitors", len(monitors)).
		Msg("Checking packet loss monitors for due tests")

	for _, monitor := range monitors {
		if !monitor.Enabled {
			log.Debug().
				Int64("monitor_id", monitor.ID).
				Str("host", monitor.Host).
				Msg("Monitor disabled, skipping")
			continue
		}

		if monitor.NextRun == nil {
			log.Warn().
				Int64("monitor_id", monitor.ID).
				Str("host", monitor.Host).
				Msg("Monitor has nil NextRun, skipping")
			continue
		}

		nextRunUTC := monitor.NextRun.UTC()
		isOverdue := nextRunUTC.Before(now) || nextRunUTC.Equal(now)
		timeDiff := nextRunUTC.Sub(now)

		if !isOverdue {
			log.Debug().
				Int64("monitor_id", monitor.ID).
				Str("host", monitor.Host).
				Time("next_run_utc", nextRunUTC).
				Time("now_utc", now).
				Dur("time_until_due", timeDiff).
				Msg("Monitor not yet due")
			continue
		}

		// claim the next run before the test starts, so the next tick does
		// not start the test again. Count from the scheduled start to keep
		// the interval steady.
		scheduledStart := nextRunUTC
		nextRun := s.nextRunAfter(monitor.Interval, scheduledStart, now, true)
		if nextRun.IsZero() {
			log.Error().
				Int64("monitor_id", monitor.ID).
				Str("interval", monitor.Interval).
				Time("scheduled_start", scheduledStart).
				Msg("Error calculating next run time for monitor")
			continue
		}
		monitor.LastRun = &scheduledStart
		monitor.NextRun = &nextRun
		if err := s.db.UpdatePacketLossMonitor(monitor); err != nil {
			log.Error().
				Err(err).
				Int64("monitor_id", monitor.ID).
				Msg("Error updating monitor schedule")
			continue
		}

		key := runKey{"packetloss", monitor.ID}
		if _, running := s.inFlight.LoadOrStore(key, struct{}{}); running {
			log.Warn().
				Int64("monitor_id", monitor.ID).
				Str("host", monitor.Host).
				Msg("Previous packet loss test still running, skipping this run")
			continue
		}

		log.Info().
			Int64("monitor_id", monitor.ID).
			Str("host", monitor.Host).
			Time("scheduled_start_time_utc", scheduledStart).
			Dur("delay", now.Sub(scheduledStart)).
			Time("next_run_utc", nextRun).
			Str("interval", monitor.Interval).
			Msg("Starting scheduled packet loss test")

		go func() {
			defer s.inFlight.Delete(key)
			if s.packetLoss != nil {
				s.packetLoss.RunScheduledTest(monitor)
			}
		}()
	}
}

// UpdateMonitorSchedule updates a monitor's last_run and next_run times after a test completes
func (s *service) UpdateMonitorSchedule(monitorID int64, interval string) error {
	now := time.Now().UTC()

	// Get the monitor to update
	monitor, err := s.db.GetPacketLossMonitor(monitorID)
	if err != nil {
		return err
	}

	// Calculate next run time
	nextRun := s.calculateNextRun(interval, now, true)
	if nextRun.IsZero() {
		log.Error().
			Int64("monitor_id", monitorID).
			Str("interval", interval).
			Msg("Error calculating next run time for monitor")
		return nil // Don't fail, just log
	}

	// Update the monitor
	lastRun := now
	monitor.LastRun = &lastRun
	monitor.NextRun = &nextRun

	log.Debug().
		Int64("monitor_id", monitorID).
		Time("last_run", now).
		Time("next_run", nextRun).
		Str("interval", interval).
		Msg("Updating monitor schedule after test completion")

	return s.db.UpdatePacketLossMonitor(monitor)
}

// initializeDNSMonitors gives every enabled DNS monitor a next run time on
// startup. Missed runs are not executed, as with the other schedules.
func (s *service) initializeDNSMonitors() {
	if s.dns == nil {
		return
	}

	monitors, err := s.db.GetDNSMonitors()
	if err != nil {
		log.Error().Err(err).Msg("Error fetching dns monitors during initialization")
		return
	}

	now := time.Now().UTC()
	for _, monitor := range monitors {
		if !monitor.Enabled || (monitor.NextRun != nil && monitor.NextRun.After(now)) {
			continue
		}

		nextRun := s.calculateNextRun(monitor.Interval, now, true)
		if nextRun.IsZero() {
			log.Error().
				Int64("monitor_id", monitor.ID).
				Str("interval", monitor.Interval).
				Msg("Could not calculate next run time for dns monitor")
			continue
		}

		if err := s.db.UpdateDNSMonitorSchedule(monitor.ID, monitor.LastRun, nextRun); err != nil {
			log.Error().Err(err).Int64("monitor_id", monitor.ID).Msg("Error updating dns monitor during initialization")
		}
	}
}

// checkAndRunDNSMonitors runs every enabled DNS monitor that is due
func (s *service) checkAndRunDNSMonitors() {
	if s.dns == nil {
		return
	}

	monitors, err := s.db.GetDNSMonitors()
	if err != nil {
		log.Error().Err(err).Msg("Error fetching dns monitors")
		return
	}

	now := time.Now().UTC()
	for _, monitor := range monitors {
		if !monitor.Enabled || monitor.NextRun == nil || monitor.NextRun.UTC().After(now) {
			continue
		}

		// claim the next run before the check starts, so the next tick does
		// not start the check again
		scheduledStart := monitor.NextRun.UTC()
		nextRun := s.nextRunAfter(monitor.Interval, scheduledStart, now, true)
		if nextRun.IsZero() {
			log.Error().
				Int64("monitor_id", monitor.ID).
				Str("interval", monitor.Interval).
				Msg("Error calculating next run time for dns monitor")
			continue
		}
		if err := s.db.UpdateDNSMonitorSchedule(monitor.ID, &scheduledStart, nextRun); err != nil {
			log.Error().Err(err).Int64("monitor_id", monitor.ID).Msg("Error updating dns monitor schedule")
			continue
		}

		key := runKey{"dns", monitor.ID}
		if _, running := s.inFlight.LoadOrStore(key, struct{}{}); running {
			continue
		}

		go func() {
			defer s.inFlight.Delete(key)
			s.dns.RunCheck(monitor)
		}()
	}
}

// CalculateNextRun is a public wrapper for calculateNextRun
func (s *service) CalculateNextRun(interval string, from time.Time) time.Time {
	return s.calculateNextRun(interval, from, false)
}
