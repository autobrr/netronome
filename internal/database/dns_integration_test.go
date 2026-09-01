// Copyright (c) 2024-2026, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package database

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/autobrr/netronome/internal/types"
)

func TestDNSMonitor_CRUD(t *testing.T) {
	RunTestWithBothDatabases(t, func(t *testing.T, td *TestDatabase) {
		created, err := td.Service.CreateDNSMonitor(&types.DNSMonitor{
			Name:       "Router",
			Host:       "192.168.1.1",
			Protocol:   types.DNSProtocolUDP,
			Query:      "google.com",
			RecordType: "A",
			Interval:   "60s",
			Enabled:    true,
		})
		require.NoError(t, err)
		assert.Greater(t, created.ID, int64(0))

		retrieved, err := td.Service.GetDNSMonitor(created.ID)
		require.NoError(t, err)
		assert.Equal(t, "Router", retrieved.Name)
		assert.Equal(t, "192.168.1.1", retrieved.Host)
		assert.Equal(t, types.DNSProtocolUDP, retrieved.Protocol)
		assert.Equal(t, "google.com", retrieved.Query)
		assert.Equal(t, "A", retrieved.RecordType)
		assert.True(t, retrieved.Enabled)

		retrieved.Name = "Pi-hole"
		retrieved.Protocol = types.DNSProtocolDoT
		retrieved.RecordType = "AAAA"
		retrieved.Enabled = false
		require.NoError(t, td.Service.UpdateDNSMonitor(retrieved))

		updated, err := td.Service.GetDNSMonitor(created.ID)
		require.NoError(t, err)
		assert.Equal(t, "Pi-hole", updated.Name)
		assert.Equal(t, types.DNSProtocolDoT, updated.Protocol)
		assert.Equal(t, "AAAA", updated.RecordType)
		assert.False(t, updated.Enabled)

		monitors, err := td.Service.GetDNSMonitors()
		require.NoError(t, err)
		assert.Len(t, monitors, 1)

		require.NoError(t, td.Service.DeleteDNSMonitor(created.ID))

		_, err = td.Service.GetDNSMonitor(created.ID)
		assert.ErrorIs(t, err, ErrNotFound)
	})
}

func TestDNSMonitor_NotFound(t *testing.T) {
	RunTestWithBothDatabases(t, func(t *testing.T, td *TestDatabase) {
		_, err := td.Service.GetDNSMonitor(99999)
		assert.ErrorIs(t, err, ErrNotFound)

		assert.ErrorIs(t, td.Service.UpdateDNSMonitorState(99999, "down"), ErrNotFound)
		assert.ErrorIs(t, td.Service.DeleteDNSMonitor(99999), ErrNotFound)
	})
}

func TestUpdateDNSMonitorState(t *testing.T) {
	RunTestWithBothDatabases(t, func(t *testing.T, td *TestDatabase) {
		monitor := CreateTestDNSMonitor(t, td)

		require.NoError(t, td.Service.UpdateDNSMonitorState(monitor.ID, "down"))

		updated, err := td.Service.GetDNSMonitor(monitor.ID)
		require.NoError(t, err)
		assert.Equal(t, "down", updated.LastState)
		assert.NotNil(t, updated.LastStateChange)
	})
}

func TestUpdateDNSMonitorSchedule_KeepsConfiguration(t *testing.T) {
	RunTestWithBothDatabases(t, func(t *testing.T, td *TestDatabase) {
		monitor := CreateTestDNSMonitor(t, td)

		lastRun := time.Now().UTC().Truncate(time.Second)
		nextRun := lastRun.Add(time.Minute)
		require.NoError(t, td.Service.UpdateDNSMonitorSchedule(monitor.ID, &lastRun, nextRun))

		updated, err := td.Service.GetDNSMonitor(monitor.ID)
		require.NoError(t, err)
		require.NotNil(t, updated.LastRun)
		require.NotNil(t, updated.NextRun)
		assert.Equal(t, lastRun, updated.LastRun.UTC())
		assert.Equal(t, nextRun, updated.NextRun.UTC())

		// the configuration the user owns is untouched
		assert.Equal(t, monitor.Host, updated.Host)
		assert.Equal(t, monitor.Protocol, updated.Protocol)
		assert.Equal(t, monitor.Query, updated.Query)
		assert.Equal(t, monitor.RecordType, updated.RecordType)
		assert.Equal(t, monitor.Interval, updated.Interval)
		assert.Equal(t, monitor.Enabled, updated.Enabled)

		assert.ErrorIs(t, td.Service.UpdateDNSMonitorSchedule(99999, &lastRun, nextRun), ErrNotFound)
	})
}

func TestDNSResults_SaveAndLatest(t *testing.T) {
	RunTestWithBothDatabases(t, func(t *testing.T, td *TestDatabase) {
		monitor := CreateTestDNSMonitor(t, td)

		errText := "read udp 127.0.0.1:5353: i/o timeout"
		results := []types.DNSResult{
			{MonitorID: monitor.ID, ResponseTimeMs: 12.5, ResponseCode: "NOERROR", Success: true, CreatedAt: time.Now().Add(-2 * time.Hour)},
			{MonitorID: monitor.ID, ResponseTimeMs: 5000, ResponseCode: "", Success: false, Error: &errText, CreatedAt: time.Now().Add(-time.Hour)},
			{MonitorID: monitor.ID, ResponseTimeMs: 9.25, ResponseCode: "NOERROR", Success: true, CreatedAt: time.Now()},
		}

		for i := range results {
			require.NoError(t, td.Service.SaveDNSResult(&results[i]))
			assert.Greater(t, results[i].ID, int64(0))
		}

		latest, err := td.Service.GetLatestDNSResult(monitor.ID)
		require.NoError(t, err)
		assert.Equal(t, 9.25, latest.ResponseTimeMs)
		assert.True(t, latest.Success)
		assert.Nil(t, latest.Error)

		page, err := td.Service.GetDNSResults(monitor.ID, 1, 2)
		require.NoError(t, err)
		require.Len(t, page.Data, 2)
		assert.Equal(t, 3, page.Total)
		assert.Equal(t, 1, page.Page)
		assert.Equal(t, 2, page.Limit)
		require.NotNil(t, page.Data[1].Error)
		assert.Equal(t, errText, *page.Data[1].Error)

		second, err := td.Service.GetDNSResults(monitor.ID, 2, 2)
		require.NoError(t, err)
		require.Len(t, second.Data, 1)
		assert.NotEqual(t, page.Data[0].ID, second.Data[0].ID)
		assert.NotEqual(t, page.Data[1].ID, second.Data[0].ID)
	})
}

func TestDeleteDNSMonitor_RemovesResults(t *testing.T) {
	RunTestWithBothDatabases(t, func(t *testing.T, td *TestDatabase) {
		monitor := CreateTestDNSMonitor(t, td)

		result := &types.DNSResult{MonitorID: monitor.ID, ResponseTimeMs: 11, ResponseCode: "NOERROR", Success: true, CreatedAt: time.Now()}
		require.NoError(t, td.Service.SaveDNSResult(result))

		require.NoError(t, td.Service.DeleteDNSMonitor(monitor.ID))
		AssertRecordNotExists(t, td, "dns_results", "monitor_id", monitor.ID)
	})
}

func TestDNSNotificationEventsSeeded(t *testing.T) {
	RunTestWithBothDatabases(t, func(t *testing.T, td *TestDatabase) {
		events, err := td.Service.GetEventsByCategory(NotificationCategoryDNS)
		require.NoError(t, err)
		require.Len(t, events, 2)

		types := []string{events[0].EventType, events[1].EventType}
		assert.Contains(t, types, NotificationEventDNSDown)
		assert.Contains(t, types, NotificationEventDNSRecovered)
	})
}
