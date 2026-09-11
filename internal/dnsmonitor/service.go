// Copyright (c) 2024-2026, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package dnsmonitor

import (
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/autobrr/netronome/internal/database"
	"github.com/autobrr/netronome/internal/notifications"
	"github.com/autobrr/netronome/internal/types"
)

// Monitor states. A monitor goes to StateDown on a failed check and to
// StateRecovered on the first success after that.
const (
	StateOK        = "ok"
	StateDown      = "down"
	StateRecovered = "recovered"
)

// Service runs DNS monitor checks and records what they find.
type Service struct {
	db        database.Service
	notifier  *notifications.Notifier
	mu        sync.RWMutex
	broadcast func(types.DNSUpdate)
}

func NewService(db database.Service, notifier *notifications.Notifier) *Service {
	return &Service{db: db, notifier: notifier}
}

// SetBroadcast sets the broadcast function for the service
func (s *Service) SetBroadcast(broadcast func(types.DNSUpdate)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.broadcast = broadcast
}

func (s *Service) send(update types.DNSUpdate) {
	s.mu.RLock()
	broadcast := s.broadcast
	s.mu.RUnlock()

	if broadcast != nil {
		broadcast(update)
	}
}

// RunCheck sends one query for the monitor, saves the result, moves the monitor
// between states, and sends a notification on a state change.
func (s *Service) RunCheck(monitor *types.DNSMonitor) {
	s.send(types.DNSUpdate{
		Type:      "dns",
		MonitorID: monitor.ID,
		Host:      monitor.Host,
		IsRunning: true,
		Enabled:   monitor.Enabled,
	})

	check := Probe(monitor)

	result := &types.DNSResult{
		MonitorID:      monitor.ID,
		ResponseTimeMs: float64(check.ResponseTime.Microseconds()) / 1000,
		ResponseCode:   check.ResponseCode,
		Success:        check.Success,
		CreatedAt:      time.Now(),
	}
	if check.Err != nil {
		errText := check.Err.Error()
		result.Error = &errText
	}

	log.Debug().
		Int64("monitorID", monitor.ID).
		Str("host", monitor.Host).
		Str("protocol", monitor.Protocol).
		Float64("responseTimeMs", result.ResponseTimeMs).
		Str("responseCode", result.ResponseCode).
		Bool("success", result.Success).
		Msg("DNS check completed")

	if err := s.db.SaveDNSResult(result); err != nil {
		log.Error().Err(err).Int64("monitorID", monitor.ID).Msg("Failed to save dns result")
	}

	state := s.applyState(monitor, check)

	update := types.DNSUpdate{
		Type:           "dns",
		MonitorID:      monitor.ID,
		Host:           monitor.Host,
		Enabled:        monitor.Enabled,
		Success:        result.Success,
		ResponseTimeMs: result.ResponseTimeMs,
		ResponseCode:   result.ResponseCode,
		State:          state,
	}
	if result.Error != nil {
		update.Error = *result.Error
	}
	s.send(update)
}

// applyState moves the monitor to its new state and notifies on a change. It
// returns the state the monitor is in after the check.
func (s *Service) applyState(monitor *types.DNSMonitor, check Check) string {
	previous := monitor.LastState

	state := StateOK
	switch {
	case !check.Success:
		state = StateDown
	case previous == StateDown:
		state = StateRecovered
	}

	if state == previous {
		return state
	}

	if err := s.db.UpdateDNSMonitorState(monitor.ID, state); err != nil {
		log.Error().Err(err).Int64("monitorID", monitor.ID).Str("state", state).Msg("Failed to update dns monitor state")
	}
	monitor.LastState = state

	if s.notifier == nil || (state != StateDown && state != StateRecovered) {
		return state
	}

	name := monitor.Name
	if name == "" {
		name = monitor.Host
	}

	detail := fmt.Sprintf("Response time: %.0f ms", float64(check.ResponseTime.Microseconds())/1000)
	if check.Err != nil {
		detail = check.Err.Error()
	}

	if err := s.notifier.SendDNSNotification(name, monitor.Host, detail, state == StateDown); err != nil {
		log.Error().Err(err).Int64("monitorID", monitor.ID).Str("state", state).Msg("Failed to send dns notification")
	}

	return state
}

// GetMonitorStatus returns the last known status of a monitor.
func (s *Service) GetMonitorStatus(monitorID int64) (*types.DNSUpdate, error) {
	monitor, err := s.db.GetDNSMonitor(monitorID)
	if err != nil {
		return nil, err
	}

	update := &types.DNSUpdate{
		Type:      "dns",
		MonitorID: monitor.ID,
		Host:      monitor.Host,
		Enabled:   monitor.Enabled,
		State:     monitor.LastState,
	}

	result, err := s.db.GetLatestDNSResult(monitorID)
	if err == database.ErrNotFound {
		return update, nil
	}
	if err != nil {
		return nil, err
	}

	update.Success = result.Success
	update.ResponseTimeMs = result.ResponseTimeMs
	update.ResponseCode = result.ResponseCode
	if result.Error != nil {
		update.Error = *result.Error
	}

	return update, nil
}
