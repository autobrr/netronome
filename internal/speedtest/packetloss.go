// Copyright (c) 2024-2025, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package speedtest

import (
	"context"
	"fmt"
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
	Interval    int
	PacketCount int
	Threshold   float64
	Enabled     bool
	Cancel      context.CancelFunc
	ctx         context.Context
}

// PacketLossService manages packet loss monitoring
type PacketLossService struct {
	monitors       map[int64]*PacketLossMonitor
	progress       map[int64]float64   // Track current progress for each monitor
	completed      map[int64]time.Time // Track recently completed tests
	mu             sync.RWMutex
	db             database.Service
	notifier       *notifications.Notifier
	broadcast      func(types.PacketLossUpdate)
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
		db:             db,
		notifier:       notifier,
		broadcast:      broadcast,
		maxConcurrent:  maxConcurrent,
		privilegedMode: privilegedMode,
	}
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
		Interval:    monitorConfig.Interval,
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

// runMonitor runs the continuous monitoring loop
func (s *PacketLossService) runMonitor(monitor *PacketLossMonitor) {
	log.Info().
		Int64("monitorID", monitor.ID).
		Str("host", monitor.Host).
		Int("interval", monitor.Interval).
		Int("packetCount", monitor.PacketCount).
		Msg("Starting continuous packet loss monitoring")

	ticker := time.NewTicker(time.Duration(monitor.Interval) * time.Second)
	defer ticker.Stop()

	// Run initial test immediately
	s.runSingleTest(monitor)

	for {
		select {
		case <-monitor.ctx.Done():
			log.Info().
				Int64("monitorID", monitor.ID).
				Str("host", monitor.Host).
				Msg("Monitor context cancelled, stopping")
			return
		case <-ticker.C:
			s.runSingleTest(monitor)
		}
	}
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
		return
	}

	// Configure pinger
	pinger.Interval = 1 * time.Second // Send one packet per second
	pinger.Count = monitor.PacketCount
	pinger.Timeout = time.Duration(monitor.PacketCount*2) * time.Second // 2 seconds per packet timeout
	pinger.SetPrivileged(s.privilegedMode)

	log.Info().
		Int64("monitorID", monitor.ID).
		Str("host", monitor.Host).
		Bool("privilegedMode", s.privilegedMode).
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
			log.Error().
				Err(err).
				Int64("monitorID", monitor.ID).
				Str("host", monitor.Host).
				Msg("Pinger run failed")
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
		return

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

	// Save results to database
	result := &types.PacketLossResult{
		MonitorID:   monitor.ID,
		PacketLoss:  stats.PacketLoss,
		MinRTT:      float64(stats.MinRtt.Milliseconds()),
		MaxRTT:      float64(stats.MaxRtt.Milliseconds()),
		AvgRTT:      float64(stats.AvgRtt.Milliseconds()),
		StdDevRTT:   float64(stats.StdDevRtt.Milliseconds()),
		PacketsSent: stats.PacketsSent,
		PacketsRecv: stats.PacketsRecv,
		CreatedAt:   time.Now(),
	}

	if err := s.db.SavePacketLossResult(result); err != nil {
		log.Error().
			Err(err).
			Int64("monitorID", monitor.ID).
			Msg("Failed to save packet loss result")
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
			PacketsSent: stats.PacketsSent,
			PacketsRecv: stats.PacketsRecv,
		})
	}

	// Check threshold and send notification if needed
	if s.notifier != nil && stats.PacketLoss > monitor.Threshold {
		s.sendPacketLossAlert(monitor, stats)
	}
}

// sendPacketLossAlert sends a notification for high packet loss
func (s *PacketLossService) sendPacketLossAlert(monitor *PacketLossMonitor, stats *probing.Statistics) {
	// Create a fake speed test result to reuse existing notification system
	// In the future, we should extend the notification system to support different types
	fakeResult := &types.SpeedTestResult{
		ServerName:    fmt.Sprintf("Packet Loss Monitor: %s", monitor.Name),
		TestType:      "packetloss",
		Latency:       fmt.Sprintf("%.2fms", stats.AvgRtt.Seconds()*1000),
		PacketLoss:    stats.PacketLoss,
		DownloadSpeed: 0, // Not applicable
		UploadSpeed:   0, // Not applicable
		CreatedAt:     time.Now(),
	}

	// Send notification
	s.notifier.SendNotification(fakeResult)

	log.Warn().
		Int64("monitorID", monitor.ID).
		Str("host", monitor.Host).
		Float64("packetLoss", stats.PacketLoss).
		Float64("threshold", monitor.Threshold).
		Msg("Packet loss threshold exceeded, notification sent")
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
