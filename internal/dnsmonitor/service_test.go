// Copyright (c) 2024-2026, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package dnsmonitor

import (
	"context"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/autobrr/netronome/internal/config"
	"github.com/autobrr/netronome/internal/database"
	"github.com/autobrr/netronome/internal/types"
)

// testDB is one SQLite database with the schema applied. The database package
// keeps a single instance per process, so every test here shares it and works
// on its own monitor rows.
var testDB database.Service

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "netronome-dnsmonitor")
	if err != nil {
		panic(err)
	}

	testDB = database.New(config.DatabaseConfig{
		Type: config.SQLite,
		Path: filepath.Join(dir, "dnsmonitor_test.db"),
	})
	if err := testDB.InitializeTables(context.Background()); err != nil {
		panic(err)
	}

	code := m.Run()

	_ = testDB.Close()
	_ = os.RemoveAll(dir)
	os.Exit(code)
}

// startFlakyResolver answers while healthy is true and returns SERVFAIL
// otherwise, so a test can make the same monitor pass and fail.
func startFlakyResolver(t *testing.T, healthy *atomic.Bool) string {
	return startUDPServer(t, dns.HandlerFunc(func(w dns.ResponseWriter, req *dns.Msg) {
		if !healthy.Load() {
			rcode(dns.RcodeServerFailure)(w, req)
			return
		}
		answer(w, req)
	}))
}

func TestRunCheck_RecordsResultAndStates(t *testing.T) {
	db := testDB

	var healthy atomic.Bool
	healthy.Store(true)
	addr := startFlakyResolver(t, &healthy)

	monitor, err := db.CreateDNSMonitor(&types.DNSMonitor{
		Name:       "Local resolver",
		Host:       addr,
		Protocol:   types.DNSProtocolUDP,
		Query:      "example.invalid",
		RecordType: "A",
		Interval:   "60s",
		Enabled:    true,
	})
	require.NoError(t, err)

	service := NewService(db, nil)

	// a healthy resolver records a successful check and the normal state
	service.RunCheck(monitor)

	result, err := db.GetLatestDNSResult(monitor.ID)
	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.Equal(t, "NOERROR", result.ResponseCode)
	assert.Positive(t, result.ResponseTimeMs)
	assert.Nil(t, result.Error)
	assert.Equal(t, StateOK, reloadState(t, db, monitor.ID))

	// a failing resolver moves the monitor to down and keeps the reason
	healthy.Store(false)
	service.RunCheck(monitor)

	result, err = db.GetLatestDNSResult(monitor.ID)
	require.NoError(t, err)
	assert.False(t, result.Success)
	assert.Equal(t, "SERVFAIL", result.ResponseCode)
	require.NotNil(t, result.Error)
	assert.Contains(t, *result.Error, "SERVFAIL")
	assert.Equal(t, StateDown, reloadState(t, db, monitor.ID))

	// the next success recovers it, through a fresh service and a monitor read
	// back from the database, as a restart would do
	healthy.Store(true)
	reloaded, err := db.GetDNSMonitor(monitor.ID)
	require.NoError(t, err)
	NewService(db, nil).RunCheck(reloaded)

	assert.Equal(t, StateRecovered, reloadState(t, db, monitor.ID))

	history, err := db.GetDNSResults(monitor.ID, 1, 25)
	require.NoError(t, err)
	assert.Equal(t, 3, history.Total)
}

func TestRunCheck_BroadcastsStartAndResult(t *testing.T) {
	db := testDB
	addr := startUDPServer(t, dns.HandlerFunc(answer))

	monitor, err := db.CreateDNSMonitor(&types.DNSMonitor{
		Host:       addr,
		Protocol:   types.DNSProtocolUDP,
		Query:      "example.invalid",
		RecordType: "A",
		Interval:   "60s",
		Enabled:    true,
	})
	require.NoError(t, err)

	var updates []types.DNSUpdate
	service := NewService(db, nil)
	service.SetBroadcast(func(update types.DNSUpdate) { updates = append(updates, update) })

	service.RunCheck(monitor)

	require.Len(t, updates, 2)
	assert.True(t, updates[0].IsRunning)
	assert.False(t, updates[1].IsRunning)
	assert.True(t, updates[1].Success)
	assert.Equal(t, "NOERROR", updates[1].ResponseCode)
	assert.Equal(t, StateOK, updates[1].State)
}

func TestGetMonitorStatus_WithoutResults(t *testing.T) {
	db := testDB

	monitor, err := db.CreateDNSMonitor(&types.DNSMonitor{
		Host:       "127.0.0.1:5353",
		Protocol:   types.DNSProtocolUDP,
		Query:      "example.invalid",
		RecordType: "A",
		Interval:   "60s",
		Enabled:    false,
	})
	require.NoError(t, err)

	status, err := NewService(db, nil).GetMonitorStatus(monitor.ID)
	require.NoError(t, err)
	assert.Equal(t, monitor.ID, status.MonitorID)
	assert.False(t, status.Enabled)
	assert.False(t, status.Success)
	assert.Zero(t, status.ResponseTimeMs)
}

func reloadState(t *testing.T, db database.Service, monitorID int64) string {
	t.Helper()

	monitor, err := db.GetDNSMonitor(monitorID)
	require.NoError(t, err)
	return monitor.LastState
}
