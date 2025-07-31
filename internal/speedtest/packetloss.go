// Copyright (c) 2024-2025, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package speedtest

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os/exec"
	"strings"
	"sync"
	"time"

	probing "github.com/prometheus-community/pro-bing"
	"github.com/rs/zerolog/log"

	"github.com/autobrr/netronome/internal/database"
	"github.com/autobrr/netronome/internal/notifications"
	"github.com/autobrr/netronome/internal/types"
)

// PacketLossMonitor represents a single packet loss monitor
type PacketLossMonitor struct {
	ID          int64
	Host        string
	Name        string
	PacketCount int
	Threshold   float64
	Enabled     bool
	Cancel      context.CancelFunc
	ctx         context.Context
}

// PacketLossService manages packet loss monitoring
type PacketLossService struct {
	monitors      map[int64]*PacketLossMonitor
	progress      map[int64]float64   // Track current progress for each monitor
	completed     map[int64]time.Time // Track recently completed tests
	mtrData       map[int64]string    // Store MTR JSON data temporarily
	mtrPrivileged map[int64]bool      // Track if MTR ran in privileged mode
	mu            sync.RWMutex
	db            database.Service
	notifier      *notifications.Notifier
	broadcast     func(types.PacketLossUpdate)
	scheduler     interface {
		UpdateMonitorSchedule(monitorID int64, interval string) error
	}
	maxConcurrent  int
	privilegedMode bool
}

// NewPacketLossService creates a new packet loss monitoring service
func NewPacketLossService(db database.Service, notifier *notifications.Notifier, broadcast func(types.PacketLossUpdate), maxConcurrent int, privilegedMode bool) *PacketLossService {
	if maxConcurrent <= 0 {
		maxConcurrent = 10
	}
	return &PacketLossService{
		monitors:       make(map[int64]*PacketLossMonitor),
		progress:       make(map[int64]float64),
		completed:      make(map[int64]time.Time),
		mtrData:        make(map[int64]string),
		mtrPrivileged:  make(map[int64]bool),
		db:             db,
		notifier:       notifier,
		broadcast:      broadcast,
		maxConcurrent:  maxConcurrent,
		privilegedMode: privilegedMode,
	}
}

// SetBroadcast sets the broadcast function for the service
func (s *PacketLossService) SetBroadcast(broadcast func(types.PacketLossUpdate)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.broadcast = broadcast
}

// SetScheduler sets the scheduler for the service
func (s *PacketLossService) SetScheduler(scheduler interface {
	UpdateMonitorSchedule(monitorID int64, interval string) error
}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.scheduler = scheduler
}

// StartMonitor starts monitoring for a specific monitor configuration
func (s *PacketLossService) StartMonitor(monitorID int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if already monitoring
	if _, exists := s.monitors[monitorID]; exists {
		return fmt.Errorf("monitor %d is already running", monitorID)
	}

	// Check concurrent limit
	if len(s.monitors) >= s.maxConcurrent {
		return fmt.Errorf("maximum concurrent monitors (%d) reached", s.maxConcurrent)
	}

	// Get monitor config from database
	monitorConfig, err := s.db.GetPacketLossMonitor(monitorID)
	if err != nil {
		return fmt.Errorf("failed to get monitor config: %w", err)
	}

	// Create monitor instance
	ctx, cancel := context.WithCancel(context.Background())
	monitor := &PacketLossMonitor{
		ID:          monitorConfig.ID,
		Host:        monitorConfig.Host,
		Name:        monitorConfig.Name,
		PacketCount: monitorConfig.PacketCount,
		Threshold:   monitorConfig.Threshold,
		Enabled:     true,
		Cancel:      cancel,
		ctx:         ctx,
	}

	// Store monitor
	s.monitors[monitorID] = monitor

	// Start monitoring in goroutine
	go s.runMonitor(monitor)

	log.Info().
		Int64("monitorID", monitorID).
		Str("host", monitorConfig.Host).
		Msg("Started packet loss monitor")

	return nil
}

// StopMonitor stops monitoring for a specific monitor
func (s *PacketLossService) StopMonitor(monitorID int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	monitor, exists := s.monitors[monitorID]
	if !exists {
		return fmt.Errorf("monitor %d is not running", monitorID)
	}

	// Cancel context to stop the monitor
	monitor.Cancel()

	// Remove from active monitors
	delete(s.monitors, monitorID)
	// Clear progress
	delete(s.progress, monitorID)

	log.Info().
		Int64("monitorID", monitorID).
		Str("host", monitor.Host).
		Msg("Stopped packet loss monitor")

	return nil
}

// runMonitor runs a single test for manual start
func (s *PacketLossService) runMonitor(monitor *PacketLossMonitor) {
	log.Info().
		Int64("monitorID", monitor.ID).
		Str("host", monitor.Host).
		Int("packetCount", monitor.PacketCount).
		Msg("Running manual packet loss test")

	// Run a single test when manually started
	s.runSingleTest(monitor)

	// Remove from active monitors after completion
	s.mu.Lock()
	delete(s.monitors, monitor.ID)
	delete(s.progress, monitor.ID)
	s.mu.Unlock()

	log.Info().
		Int64("monitorID", monitor.ID).
		Str("host", monitor.Host).
		Msg("Manual packet loss test completed")
}

// runSingleTest runs a single packet loss test
func (s *PacketLossService) runSingleTest(monitor *PacketLossMonitor) {
	log.Debug().
		Int64("monitorID", monitor.ID).
		Str("host", monitor.Host).
		Msg("Running packet loss test")

	// Initialize progress to 0
	s.mu.Lock()
	s.progress[monitor.ID] = 0
	s.mu.Unlock()

	// Broadcast start update
	if s.broadcast != nil {
		s.broadcast(types.PacketLossUpdate{
			Type:       "packetloss",
			MonitorID:  monitor.ID,
			Host:       monitor.Host,
			IsRunning:  true,
			IsComplete: false,
			Progress:   0,
		})
	}

	// Try MTR first if available
	if s.checkMTRAvailable() {
		log.Info().
			Int64("monitorID", monitor.ID).
			Str("host", monitor.Host).
			Msg("MTR is available, attempting MTR test")

		if result, err := s.runMTRTest(monitor); err == nil {
			s.processResults(monitor, result)
			return
		} else {
			log.Warn().
				Err(err).
				Int64("monitorID", monitor.ID).
				Str("host", monitor.Host).
				Msg("MTR test failed, falling back to ping")
		}
	}

	// Fall back to regular ping test
	s.runPingTest(monitor)
}

// checkMTRAvailable checks if MTR is available on the system
func (s *PacketLossService) checkMTRAvailable() bool {
	_, err := exec.LookPath("mtr")
	return err == nil
}

// runPingTest runs a traditional ping-based packet loss test
func (s *PacketLossService) runPingTest(monitor *PacketLossMonitor) {
	// Try privileged mode first if configured
	if s.privilegedMode {
		if err := s.runPingWithPrivilege(monitor, true); err == nil {
			return // Success
		} else if strings.Contains(err.Error(), "operation not permitted") {
			log.Warn().
				Err(err).
				Int64("monitorID", monitor.ID).
				Str("host", monitor.Host).
				Msg("Privileged ping failed, trying unprivileged mode")

			// Try unprivileged mode
			if err := s.runPingWithPrivilege(monitor, false); err == nil {
				return // Success with unprivileged
			}
			// If unprivileged also failed, let it fail naturally
			// The error has already been handled in runPingWithPrivilege
		} else {
			// Other error in privileged mode, try unprivileged as fallback
			s.runPingWithPrivilege(monitor, false)
		}
	} else {
		// Not using privileged mode, run unprivileged directly
		s.runPingWithPrivilege(monitor, false)
	}
}

// runPingWithPrivilege runs the ping test with specified privilege mode
func (s *PacketLossService) runPingWithPrivilege(monitor *PacketLossMonitor, usePrivileged bool) error {
	// Create a new pinger for this test
	pinger, err := probing.NewPinger(monitor.Host)
	if err != nil {
		log.Error().
			Err(err).
			Int64("monitorID", monitor.ID).
			Str("host", monitor.Host).
			Msg("Failed to create pinger")

		// Broadcast error
		if s.broadcast != nil {
			s.broadcast(types.PacketLossUpdate{
				Type:       "packetloss",
				MonitorID:  monitor.ID,
				Host:       monitor.Host,
				IsRunning:  false,
				IsComplete: true,
				Error:      fmt.Sprintf("Failed to create pinger: %v", err),
			})
		}
		return err
	}

	// Configure pinger
	pinger.Interval = 1 * time.Second // Send one packet per second
	pinger.Count = monitor.PacketCount
	pinger.Timeout = time.Duration(monitor.PacketCount*2) * time.Second // 2 seconds per packet timeout
	pinger.SetPrivileged(usePrivileged)

	log.Info().
		Int64("monitorID", monitor.ID).
		Str("host", monitor.Host).
		Bool("privilegedMode", usePrivileged).
		Msg("Configured pinger")

	// Create results channel for this test
	results := make(chan *probing.Statistics, 1)

	// Set up callbacks
	packetsReceived := 0
	packetsSent := 0

	// Track when packets are sent
	pinger.OnSend = func(pkt *probing.Packet) {
		packetsSent++
		progress := float64(packetsSent) / float64(monitor.PacketCount) * 100

		// Store progress based on sent packets
		s.mu.Lock()
		s.progress[monitor.ID] = progress
		s.mu.Unlock()

		log.Debug().
			Int64("monitorID", monitor.ID).
			Int("seq", pkt.Seq).
			Int("sent", packetsSent).
			Float64("progress", progress).
			Msg("Packet sent")

		// Broadcast progress update based on sent packets
		if s.broadcast != nil {
			s.broadcast(types.PacketLossUpdate{
				Type:        "packetloss",
				MonitorID:   monitor.ID,
				Host:        monitor.Host,
				IsRunning:   true,
				IsComplete:  false,
				Progress:    progress,
				PacketsSent: packetsSent,
				PacketsRecv: packetsReceived,
			})
		}
	}

	pinger.OnRecv = func(pkt *probing.Packet) {
		packetsReceived++

		log.Debug().
			Int64("monitorID", monitor.ID).
			Int("seq", pkt.Seq).
			Dur("rtt", pkt.Rtt).
			Int("ttl", pkt.TTL).
			Int("received", packetsReceived).
			Msg("Packet received")

		// Update with latest receive count
		if s.broadcast != nil {
			s.mu.RLock()
			progress := s.progress[monitor.ID]
			s.mu.RUnlock()

			s.broadcast(types.PacketLossUpdate{
				Type:        "packetloss",
				MonitorID:   monitor.ID,
				Host:        monitor.Host,
				IsRunning:   true,
				IsComplete:  false,
				Progress:    progress,
				PacketsSent: packetsSent,
				PacketsRecv: packetsReceived,
			})
		}
	}

	// Create context with timeout for goroutine management
	timeoutDuration := time.Duration(monitor.PacketCount*2) * time.Second
	pingerCtx, pingerCancel := context.WithTimeout(context.Background(), timeoutDuration)
	defer pingerCancel()

	// Channel to track pinger completion
	pingerDone := make(chan struct{})
	var pingerStats *probing.Statistics
	var pingerMutex sync.Mutex
	var pingerError error

	pinger.OnFinish = func(stats *probing.Statistics) {
		pingerMutex.Lock()
		pingerStats = stats
		pingerMutex.Unlock()

		log.Debug().
			Int64("monitorID", monitor.ID).
			Float64("packetLoss", stats.PacketLoss).
			Int("packetsSent", stats.PacketsSent).
			Int("packetsRecv", stats.PacketsRecv).
			Msg("Pinger OnFinish called")

		select {
		case results <- stats:
		default:
			// Channel full, should not happen with buffer of 1
		}
	}

	// Run the pinger in a goroutine with context management
	go func() {
		defer func() {
			log.Debug().
				Int64("monitorID", monitor.ID).
				Msg("Pinger goroutine exiting")
			close(pingerDone)
		}()

		log.Info().
			Int64("monitorID", monitor.ID).
			Str("host", monitor.Host).
			Dur("timeout", timeoutDuration).
			Msg("Starting pinger")

		err := pinger.Run()
		if err != nil {
			// Check if it's a permission error and we were using privileged mode
			if usePrivileged && strings.Contains(err.Error(), "operation not permitted") {
				log.Warn().
					Err(err).
					Int64("monitorID", monitor.ID).
					Str("host", monitor.Host).
					Msg("Privileged ping failed, will retry with unprivileged mode")

				// Store the error
				pingerMutex.Lock()
				pingerError = err
				pingerMutex.Unlock()
			} else {
				log.Error().
					Err(err).
					Int64("monitorID", monitor.ID).
					Str("host", monitor.Host).
					Msg("Pinger run failed")

				// Store the error
				pingerMutex.Lock()
				pingerError = err
				pingerMutex.Unlock()
			}
		}

		log.Debug().
			Int64("monitorID", monitor.ID).
			Msg("Pinger.Run() completed")
	}()

	// Wait for results with multiple completion paths
	completed := false
	defer func() {
		if !completed {
			log.Warn().
				Int64("monitorID", monitor.ID).
				Msg("Test completed via fallback cleanup")
		}
		// Always ensure pinger is stopped and progress is cleared
		pinger.Stop()
		pingerCancel()
		s.mu.Lock()
		delete(s.progress, monitor.ID)
		s.mu.Unlock()
	}()

	select {
	case <-monitor.ctx.Done():
		log.Info().
			Int64("monitorID", monitor.ID).
			Msg("Test cancelled via monitor context")
		completed = true
		return nil

	case stats := <-results:
		log.Info().
			Int64("monitorID", monitor.ID).
			Float64("packetLoss", stats.PacketLoss).
			Msg("Test completed via OnFinish callback")
		completed = true
		// Process results
		s.processResults(monitor, stats)

	case <-pingerCtx.Done():
		log.Warn().
			Int64("monitorID", monitor.ID).
			Str("host", monitor.Host).
			Msg("Test timed out - pinger context cancelled")
		completed = true

		// Check if we got stats from OnFinish before timeout
		pingerMutex.Lock()
		stats := pingerStats
		pingerMutex.Unlock()

		if stats != nil {
			log.Info().
				Int64("monitorID", monitor.ID).
				Msg("Using stats from OnFinish despite timeout")
			s.processResults(monitor, stats)
		} else {
			log.Warn().
				Int64("monitorID", monitor.ID).
				Msg("No stats from OnFinish, creating timeout result")
			// Create timeout result with 100% packet loss
			timeoutStats := &probing.Statistics{
				PacketsSent: monitor.PacketCount,
				PacketsRecv: 0,
				PacketLoss:  100.0,
				MinRtt:      0,
				MaxRtt:      0,
				AvgRtt:      0,
				StdDevRtt:   0,
			}
			s.processResults(monitor, timeoutStats)
		}

		// Broadcast timeout completion
		if s.broadcast != nil {
			packetLoss := 100.0
			if stats != nil {
				packetLoss = stats.PacketLoss
			}
			s.broadcast(types.PacketLossUpdate{
				Type:        "packetloss",
				MonitorID:   monitor.ID,
				Host:        monitor.Host,
				IsRunning:   false,
				IsComplete:  true,
				PacketLoss:  packetLoss,
				PacketsSent: monitor.PacketCount,
				PacketsRecv: 0,
			})
		}

	case <-pingerDone:
		log.Debug().
			Int64("monitorID", monitor.ID).
			Msg("Pinger goroutine completed, waiting for OnFinish")

		// Check if there was an immediate error that should be returned
		pingerMutex.Lock()
		errorToCheck := pingerError
		pingerMutex.Unlock()

		// If this was a privileged mode permission error, don't create a fallback result
		if errorToCheck != nil && usePrivileged && strings.Contains(errorToCheck.Error(), "operation not permitted") {
			log.Debug().
				Int64("monitorID", monitor.ID).
				Msg("Skipping fallback result creation for privileged mode permission error")
			completed = true
			// Error will be returned at the end of the function
			break
		}

		// Give OnFinish a short time to trigger after goroutine completes
		select {
		case stats := <-results:
			log.Info().
				Int64("monitorID", monitor.ID).
				Msg("Test completed via delayed OnFinish")
			completed = true
			s.processResults(monitor, stats)
		case <-time.After(1 * time.Second):
			log.Warn().
				Int64("monitorID", monitor.ID).
				Msg("OnFinish never called after pinger completed")
			completed = true

			// Use cached stats if available, otherwise create timeout result
			pingerMutex.Lock()
			stats := pingerStats
			pingerMutex.Unlock()

			if stats != nil {
				log.Info().
					Int64("monitorID", monitor.ID).
					Msg("Using cached stats from OnFinish")
				s.processResults(monitor, stats)
			} else {
				log.Warn().
					Int64("monitorID", monitor.ID).
					Msg("No cached stats, creating fallback result")
				timeoutStats := &probing.Statistics{
					PacketsSent: monitor.PacketCount,
					PacketsRecv: 0,
					PacketLoss:  100.0,
					MinRtt:      0,
					MaxRtt:      0,
					AvgRtt:      0,
					StdDevRtt:   0,
				}
				s.processResults(monitor, timeoutStats)
			}
		}
	}

	// Check if there was an error
	pingerMutex.Lock()
	finalErr := pingerError
	pingerMutex.Unlock()

	// Return the error if privileged mode failed
	if finalErr != nil && usePrivileged && strings.Contains(finalErr.Error(), "operation not permitted") {
		return finalErr
	}

	// Return nil on success
	return nil
}

// processResults processes the test results
func (s *PacketLossService) processResults(monitor *PacketLossMonitor, stats *probing.Statistics) {
	log.Info().
		Int64("monitorID", monitor.ID).
		Str("host", monitor.Host).
		Float64("packetLoss", stats.PacketLoss).
		Int("packetsSent", stats.PacketsSent).
		Int("packetsRecv", stats.PacketsRecv).
		Dur("minRtt", stats.MinRtt).
		Dur("maxRtt", stats.MaxRtt).
		Dur("avgRtt", stats.AvgRtt).
		Msg("Packet loss test completed")

	// Mark test as completed
	s.mu.Lock()
	s.completed[monitor.ID] = time.Now()

	// Clean up old completed entries (older than 1 minute)
	for id, completedTime := range s.completed {
		if time.Since(completedTime) > time.Minute {
			delete(s.completed, id)
		}
	}
	s.mu.Unlock()

	// Check if this was an MTR test
	usedMTR := false
	hopCount := 0
	var mtrDataStr *string
	privilegedMode := false

	s.mu.RLock()
	if mtrJSON, exists := s.mtrData[monitor.ID]; exists {
		usedMTR = true
		mtrDataStr = &mtrJSON
		// Extract hop count from Rtts hack
		if len(stats.Rtts) > 0 {
			hopCount = int(stats.Rtts[0])
		}
		// Get the privileged mode status
		if priv, exists := s.mtrPrivileged[monitor.ID]; exists {
			privilegedMode = priv
		}
	}
	s.mu.RUnlock()

	// Save results to database
	result := &types.PacketLossResult{
		MonitorID:      monitor.ID,
		PacketLoss:     stats.PacketLoss,
		MinRTT:         float64(stats.MinRtt.Milliseconds()),
		MaxRTT:         float64(stats.MaxRtt.Milliseconds()),
		AvgRTT:         float64(stats.AvgRtt.Milliseconds()),
		StdDevRTT:      float64(stats.StdDevRtt.Milliseconds()),
		PacketsSent:    stats.PacketsSent,
		PacketsRecv:    stats.PacketsRecv,
		UsedMTR:        usedMTR,
		HopCount:       hopCount,
		MTRData:        mtrDataStr,
		PrivilegedMode: privilegedMode,
		CreatedAt:      time.Now(),
	}

	// Clean up MTR data after use
	if usedMTR {
		s.mu.Lock()
		delete(s.mtrData, monitor.ID)
		delete(s.mtrPrivileged, monitor.ID)
		s.mu.Unlock()
	}

	if s.db != nil {
		if err := s.db.SavePacketLossResult(result); err != nil {
			log.Error().
				Err(err).
				Int64("monitorID", monitor.ID).
				Msg("Failed to save packet loss result")
		}
	}

	// Broadcast complete update
	if s.broadcast != nil {
		s.broadcast(types.PacketLossUpdate{
			Type:        "packetloss",
			MonitorID:   monitor.ID,
			Host:        monitor.Host,
			IsRunning:   false,
			IsComplete:  true,
			PacketLoss:  stats.PacketLoss,
			MinRTT:      result.MinRTT,
			MaxRTT:      result.MaxRTT,
			AvgRTT:      result.AvgRTT,
			StdDevRTT:   result.StdDevRTT,
			PacketsSent: stats.PacketsSent,
			PacketsRecv: stats.PacketsRecv,
			UsedMTR:     usedMTR,
			HopCount:    hopCount,
		})
	}

	// Check notification conditions and track state
	if s.notifier != nil && s.db != nil {
		// Get full monitor from database to check state
		dbMonitor, err := s.db.GetPacketLossMonitor(monitor.ID)
		if err != nil {
			log.Error().
				Err(err).
				Int64("monitorID", monitor.ID).
				Msg("Failed to get monitor from database for state tracking")
			return
		}

		// Determine current state
		var currentState string
		if stats.PacketLoss >= 100.0 {
			currentState = "down"
		} else if stats.PacketLoss > monitor.Threshold {
			currentState = "threshold_exceeded"
		} else {
			currentState = "ok"
		}

		// Get previous state from database
		previousState := dbMonitor.LastState
		if previousState == "" {
			previousState = "unknown"
		}

		// Determine state transitions and send appropriate notifications
		if previousState != currentState {
			// State has changed
			if currentState == "down" {
				// Monitor went down
				s.sendPacketLossNotification(monitor, stats, database.NotificationEventPacketLossDown)
			} else if currentState == "threshold_exceeded" {
				// Threshold exceeded (but not down)
				s.sendPacketLossNotification(monitor, stats, database.NotificationEventPacketLossHigh)
			} else if currentState == "ok" && (previousState == "down" || previousState == "threshold_exceeded") {
				// Monitor recovered
				s.sendPacketLossNotification(monitor, stats, database.NotificationEventPacketLossRecovered)
			}

			// Update state in database
			if err := s.db.UpdatePacketLossMonitorState(monitor.ID, currentState); err != nil {
				log.Error().
					Err(err).
					Int64("monitorID", monitor.ID).
					Str("state", currentState).
					Msg("Failed to update monitor state")
			}
		}
	}

	// Update monitor schedule if scheduler is available
	if s.scheduler != nil {
		// Get the monitor config to get the interval
		monitorConfig, err := s.db.GetPacketLossMonitor(monitor.ID)
		if err == nil && monitorConfig.Enabled {
			if err := s.scheduler.UpdateMonitorSchedule(monitor.ID, monitorConfig.Interval); err != nil {
				log.Error().
					Err(err).
					Int64("monitorID", monitor.ID).
					Msg("Failed to update monitor schedule after test completion")
			} else {
				log.Debug().
					Int64("monitorID", monitor.ID).
					Str("interval", monitorConfig.Interval).
					Msg("Updated monitor schedule after test completion")
			}
		}
	}
}

// sendPacketLossNotification sends a notification for packet loss events
func (s *PacketLossService) sendPacketLossNotification(monitor *PacketLossMonitor, stats *probing.Statistics, eventType string) {
	// Create packet loss notification data
	monitorName := monitor.Name
	if monitorName == "" {
		monitorName = monitor.Host
	}

	// Determine if monitor is down or recovered
	isDown := eventType == database.NotificationEventPacketLossDown
	isRecovered := eventType == database.NotificationEventPacketLossRecovered

	// Send notification with appropriate parameters
	if err := s.notifier.SendPacketLossNotification(monitorName, monitor.Host, stats.PacketLoss, isDown, isRecovered); err != nil {
		log.Error().
			Err(err).
			Int64("monitorID", monitor.ID).
			Str("eventType", eventType).
			Msg("Failed to send packet loss notification")
	} else {
		log.Info().
			Int64("monitorID", monitor.ID).
			Str("host", monitor.Host).
			Float64("packetLoss", stats.PacketLoss).
			Str("eventType", eventType).
			Msg("Packet loss notification sent")
	}
}

// GetMonitorStatus returns the current status of a monitor
func (s *PacketLossService) GetMonitorStatus(monitorID int64) (*types.PacketLossUpdate, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// First, get monitor configuration from database to check if it's enabled
	monitorConfig, err := s.db.GetPacketLossMonitor(monitorID)
	if err != nil {
		return nil, fmt.Errorf("failed to get monitor config: %w", err)
	}

	// Check if monitor is currently in memory (actively testing)
	activeMonitor, isInMemory := s.monitors[monitorID]
	progress := s.progress[monitorID]

	// Check if test was recently completed (within last 5 seconds)
	completedTime, wasCompleted := s.completed[monitorID]
	isRecentlyCompleted := wasCompleted && time.Since(completedTime) < 5*time.Second

	// Quad-state logic:
	// 1. Actively testing: in memory + has progress > 0
	// 2. Recently completed: marked as completed within last 5 seconds
	// 3. Scheduled monitoring: enabled in DB but not actively testing
	// 4. Disabled: not enabled in DB

	if isInMemory && progress > 0 {
		// State 1: Actively testing - show progress
		return &types.PacketLossUpdate{
			Type:       "packetloss",
			MonitorID:  monitorID,
			Host:       activeMonitor.Host,
			IsRunning:  true,
			IsComplete: false,
			Progress:   progress,
		}, nil
	} else if isRecentlyCompleted {
		// State 2: Recently completed - return completion status
		// Get latest result to show values
		result, err := s.db.GetLatestPacketLossResult(monitorID)
		if err != nil {
			// Return completion status without results
			return &types.PacketLossUpdate{
				Type:       "packetloss",
				MonitorID:  monitorID,
				Host:       monitorConfig.Host,
				IsRunning:  false,
				IsComplete: true,
			}, nil
		}

		// Return completion status with results
		return &types.PacketLossUpdate{
			Type:        "packetloss",
			MonitorID:   monitorID,
			Host:        monitorConfig.Host,
			IsRunning:   false,
			IsComplete:  true,
			PacketLoss:  result.PacketLoss,
			MinRTT:      result.MinRTT,
			MaxRTT:      result.MaxRTT,
			AvgRTT:      result.AvgRTT,
			PacketsSent: result.PacketsSent,
			PacketsRecv: result.PacketsRecv,
		}, nil
	} else if monitorConfig.Enabled {
		// State 3: Scheduled monitoring - enabled but waiting for next test
		// Get latest result to show last known values
		result, err := s.db.GetLatestPacketLossResult(monitorID)
		if err != nil {
			// No previous results, return basic monitoring status
			return &types.PacketLossUpdate{
				Type:       "packetloss",
				MonitorID:  monitorID,
				Host:       monitorConfig.Host,
				IsRunning:  false,
				IsComplete: false, // Not complete, just waiting for next test
			}, nil
		}

		// Return monitoring status with last known results
		return &types.PacketLossUpdate{
			Type:        "packetloss",
			MonitorID:   monitorID,
			Host:        monitorConfig.Host,
			IsRunning:   false,
			IsComplete:  false, // Not complete, just scheduled monitoring
			PacketLoss:  result.PacketLoss,
			MinRTT:      result.MinRTT,
			MaxRTT:      result.MaxRTT,
			AvgRTT:      result.AvgRTT,
			PacketsSent: result.PacketsSent,
			PacketsRecv: result.PacketsRecv,
		}, nil
	} else {
		// State 4: Disabled - monitor is not enabled
		return &types.PacketLossUpdate{
			Type:       "packetloss",
			MonitorID:  monitorID,
			Host:       monitorConfig.Host,
			IsRunning:  false,
			IsComplete: true, // Mark as complete since it's disabled
		}, nil
	}
}

// GetActiveMonitors returns all currently active monitors
func (s *PacketLossService) GetActiveMonitors() []int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	monitors := make([]int64, 0, len(s.monitors))
	for id := range s.monitors {
		monitors = append(monitors, id)
	}
	return monitors
}

// StartAllEnabledMonitors starts all enabled monitors from the database
func (s *PacketLossService) StartAllEnabledMonitors() error {
	monitors, err := s.db.GetEnabledPacketLossMonitors()
	if err != nil {
		return fmt.Errorf("failed to get enabled monitors: %w", err)
	}

	for _, monitor := range monitors {
		if err := s.StartMonitor(monitor.ID); err != nil {
			log.Error().
				Err(err).
				Int64("monitorID", monitor.ID).
				Str("host", monitor.Host).
				Msg("Failed to start monitor")
		}
	}

	return nil
}

// StopAllMonitors stops all active monitors
func (s *PacketLossService) StopAllMonitors() {
	s.mu.Lock()
	monitorIDs := make([]int64, 0, len(s.monitors))
	for id := range s.monitors {
		monitorIDs = append(monitorIDs, id)
	}
	s.mu.Unlock()

	for _, id := range monitorIDs {
		if err := s.StopMonitor(id); err != nil {
			log.Error().
				Err(err).
				Int64("monitorID", id).
				Msg("Failed to stop monitor")
		}
	}
}

// MTR JSON output structure
type mtrReport struct {
	Report struct {
		MTR struct {
			Src   string `json:"src"`
			Dst   string `json:"dst"`
			Tests int    `json:"tests"`
		} `json:"mtr"`
		Hubs []struct {
			Count int     `json:"count"`
			Host  string  `json:"host"`
			Loss  float64 `json:"Loss%"`
			Snt   int     `json:"Snt"`
			Last  float64 `json:"Last"`
			Avg   float64 `json:"Avg"`
			Best  float64 `json:"Best"`
			Wrst  float64 `json:"Wrst"`
			StDev float64 `json:"StDev"`
		} `json:"hubs"`
	} `json:"report"`
}

// runMTRTest runs an MTR test and returns statistics
func (s *PacketLossService) runMTRTest(monitor *PacketLossMonitor) (*probing.Statistics, error) {
	// Initialize GeoIP databases if not already done
	// This is a workaround since PacketLossService doesn't have access to the main service
	if countryDB == nil && asnDB == nil {
		log.Info().Msg("GeoIP databases not initialized for MTR. GeoIP enrichment will be unavailable.")
		log.Info().Msg("To enable GeoIP for MTR, ensure GeoIP is configured in the [geoip] section of your config file.")
	}

	// Broadcast that MTR is starting
	if s.broadcast != nil {
		s.broadcast(types.PacketLossUpdate{
			Type:       "packetloss",
			MonitorID:  monitor.ID,
			Host:       monitor.Host,
			IsRunning:  true,
			IsComplete: false,
			Progress:   50, // Show indeterminate progress since MTR doesn't provide real-time updates
			UsedMTR:    true,
		})
	}

	// Create timeout context
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(monitor.PacketCount*3)*time.Second)
	defer cancel()

	// Build MTR command with JSON output
	args := []string{
		"-4",                                         // Force IPv4
		"-j",                                         // JSON output
		"-c", fmt.Sprintf("%d", monitor.PacketCount), // Number of cycles
		"-i", "1", // 1 second interval
		"--no-dns", // Skip DNS resolution for speed
		monitor.Host,
	}

	// Track if we're using privileged mode
	actuallyPrivileged := s.privilegedMode

	// Try privileged mode first (ICMP)
	if s.privilegedMode {
		log.Info().
			Int64("monitorID", monitor.ID).
			Str("host", monitor.Host).
			Msg("Running MTR in privileged mode (ICMP)")
	} else {
		// Use UDP mode if unprivileged
		args = append([]string{"-u"}, args...)
		log.Info().
			Int64("monitorID", monitor.ID).
			Str("host", monitor.Host).
			Msg("Running MTR in unprivileged mode (UDP)")
	}

	// Helper function to run MTR with proper cleanup
	runMTRCommand := func(args []string) ([]byte, error) {
		cmd := exec.CommandContext(ctx, "mtr", args...)
		
		// Configure platform-specific process attributes
		configureMTRCommand(cmd)
		
		// Get stdout pipe BEFORE starting the command
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			return nil, fmt.Errorf("failed to get stdout pipe: %w", err)
		}
		
		// Start the command
		if err := cmd.Start(); err != nil {
			return nil, fmt.Errorf("failed to start MTR: %w", err)
		}
		
		// Log the process start
		log.Info().
			Int("pid", cmd.Process.Pid).
			Int64("monitorID", monitor.ID).
			Str("host", monitor.Host).
			Strs("args", args).
			Msg("Started MTR process")
		
		// Ensure cleanup on function exit
		defer func() {
			// Kill the process group to ensure all child processes are terminated
			if cmd.Process != nil {
				if err := killMTRProcessGroup(cmd.Process.Pid); err != nil {
					log.Error().
						Err(err).
						Int("pid", cmd.Process.Pid).
						Msg("Failed to kill MTR process group during cleanup")
				}
			}
		}()
		
		// Create channels for output and errors
		outputChan := make(chan []byte, 1)
		errChan := make(chan error, 1)
		
		// Run command in goroutine
		go func() {
			defer close(outputChan)
			defer close(errChan)
			
			// Read all output
			output, readErr := io.ReadAll(stdout)
			
			// Wait for command to finish
			waitErr := cmd.Wait()
			
			if readErr != nil {
				errChan <- fmt.Errorf("failed to read output: %w", readErr)
			} else if waitErr != nil {
				errChan <- fmt.Errorf("command failed: %w", waitErr)
			} else {
				outputChan <- output
			}
		}()
		
		// Wait for completion or timeout
		select {
		case <-ctx.Done():
			// Context timeout - kill the process group
			log.Warn().
				Int64("monitorID", monitor.ID).
				Str("host", monitor.Host).
				Int("timeout_seconds", monitor.PacketCount*3).
				Msg("MTR command timed out, killing process group")
			
			if cmd.Process != nil {
				pid := cmd.Process.Pid
				if err := killMTRProcessGroup(pid); err != nil {
					log.Error().
						Err(err).
						Int("pid", pid).
						Int64("monitorID", monitor.ID).
						Msg("Failed to kill MTR process group on timeout")
				} else {
					log.Info().
						Int("pid", pid).
						Int64("monitorID", monitor.ID).
						Msg("Successfully killed MTR process group")
				}
			}
			
			// Wait briefly for the goroutine to finish after kill
			select {
			case <-errChan:
				// Process terminated after kill
			case <-time.After(2 * time.Second):
				// Process didn't terminate gracefully, but we've done our best
				log.Error().
					Int64("monitorID", monitor.ID).
					Msg("MTR process did not terminate cleanly after kill")
			}
			return nil, fmt.Errorf("MTR command timed out after %d seconds", monitor.PacketCount*3)
		case err := <-errChan:
			return nil, err
		case output := <-outputChan:
			return output, nil
		}
	}

	// Run MTR command
	output, err := runMTRCommand(args)
	if err != nil {
		// If privileged mode failed, try UDP mode
		if s.privilegedMode {
			log.Warn().
				Err(err).
				Int64("monitorID", monitor.ID).
				Msg("MTR privileged mode failed, trying UDP mode")

			args = append([]string{"-u"}, args...)
			output, err = runMTRCommand(args)
			if err != nil {
				return nil, fmt.Errorf("MTR failed in both ICMP and UDP modes: %w", err)
			}
			// We fell back to UDP mode
			actuallyPrivileged = false
		} else {
			return nil, fmt.Errorf("MTR command failed: %w", err)
		}
	}

	// Store whether this MTR test ran in privileged mode
	s.mu.Lock()
	s.mtrPrivileged[monitor.ID] = actuallyPrivileged
	s.mu.Unlock()

	// Parse MTR JSON output
	var report mtrReport
	if err := json.Unmarshal(output, &report); err != nil {
		return nil, fmt.Errorf("failed to parse MTR output: %w", err)
	}

	// Find the last hop (destination)
	if len(report.Report.Hubs) == 0 {
		return nil, fmt.Errorf("MTR returned no hops")
	}

	// Get the last hop statistics
	lastHop := report.Report.Hubs[len(report.Report.Hubs)-1]

	// Create MTRData for storage
	mtrData := types.MTRData{
		Destination: monitor.Host,
		IP:          report.Report.MTR.Dst,
		HopCount:    len(report.Report.Hubs),
		Tests:       report.Report.MTR.Tests,
		Hops:        make([]types.MTRHop, 0, len(report.Report.Hubs)),
	}

	// Convert hops to our format
	for i, hop := range report.Report.Hubs {
		mtrHop := types.MTRHop{
			Number:     i + 1,
			Host:       hop.Host,
			PacketLoss: hop.Loss,
			Sent:       hop.Snt,
			Recv:       hop.Snt - int(float64(hop.Snt)*hop.Loss/100),
			Last:       hop.Last,
			Avg:        hop.Avg,
			Best:       hop.Best,
			Worst:      hop.Wrst,
			StdDev:     hop.StDev,
		}

		// Extract IP from host if it's in format "hostname (IP)"
		if strings.Contains(hop.Host, "(") && strings.Contains(hop.Host, ")") {
			parts := strings.Split(hop.Host, "(")
			if len(parts) > 1 {
				mtrHop.Host = strings.TrimSpace(parts[0])
				mtrHop.IP = strings.TrimSuffix(strings.TrimSpace(parts[1]), ")")
			}
		} else if hop.Host != "???" && hop.Host != "" {
			// If we only have host (which might be an IP), use it as IP
			if net.ParseIP(hop.Host) != nil {
				mtrHop.IP = hop.Host
			}
		}

		// Enrich with GeoIP data if we have an IP
		if mtrHop.IP != "" {
			mtrHop.CountryCode = getCountryFromIP(mtrHop.IP)
			mtrHop.AS = getASNFromIP(mtrHop.IP)

			// Debug logging
			if mtrHop.CountryCode != "" || mtrHop.AS != "" {
				log.Debug().
					Str("ip", mtrHop.IP).
					Str("countryCode", mtrHop.CountryCode).
					Str("as", mtrHop.AS).
					Msg("MTR hop enriched with GeoIP data")
			}
		}

		mtrData.Hops = append(mtrData.Hops, mtrHop)
	}

	// Store MTR data as JSON
	mtrJSON, _ := json.Marshal(mtrData)
	mtrDataStr := string(mtrJSON)

	// Create statistics compatible with existing ping results
	stats := &probing.Statistics{
		PacketsSent: lastHop.Snt,
		PacketsRecv: lastHop.Snt - int(float64(lastHop.Snt)*lastHop.Loss/100),
		PacketLoss:  lastHop.Loss,
		MinRtt:      time.Duration(lastHop.Best * float64(time.Millisecond)),
		MaxRtt:      time.Duration(lastHop.Wrst * float64(time.Millisecond)),
		AvgRtt:      time.Duration(lastHop.Avg * float64(time.Millisecond)),
		StdDevRtt:   time.Duration(lastHop.StDev * float64(time.Millisecond)),
	}

	// Mark this as MTR result
	stats.Rtts = []time.Duration{time.Duration(len(report.Report.Hubs))} // Hack to store hop count

	// Store MTR data for later retrieval
	s.mu.Lock()
	if s.mtrData == nil {
		s.mtrData = make(map[int64]string)
	}
	s.mtrData[monitor.ID] = mtrDataStr
	s.mu.Unlock()

	log.Info().
		Int64("monitorID", monitor.ID).
		Str("host", monitor.Host).
		Int("hopCount", len(report.Report.Hubs)).
		Float64("packetLoss", lastHop.Loss).
		Float64("avgRtt", lastHop.Avg).
		Msg("MTR test completed successfully")

	return stats, nil
}

// RunScheduledTest runs a single packet loss test for a monitor called by the scheduler
func (s *PacketLossService) RunScheduledTest(monitor *types.PacketLossMonitor) {
	log.Info().
		Int64("monitorID", monitor.ID).
		Str("host", monitor.Host).
		Msg("Running scheduled packet loss test")

	// Create a context for this test
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// Convert types.PacketLossMonitor to local PacketLossMonitor
	localMonitor := &PacketLossMonitor{
		ID:          monitor.ID,
		Host:        strings.TrimSpace(monitor.Host),
		Name:        monitor.Name,
		PacketCount: monitor.PacketCount,
		Threshold:   monitor.Threshold,
		Enabled:     monitor.Enabled,
		ctx:         ctx,
		Cancel:      cancel,
	}

	// Run the single test
	s.runSingleTest(localMonitor)
}
