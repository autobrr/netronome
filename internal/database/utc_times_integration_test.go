// Copyright (c) 2024-2026, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package database

import (
	"bytes"
	"encoding/json/v2"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/autobrr/netronome/internal/config"
	"github.com/autobrr/netronome/internal/types"
)

// The shared Postgres server runs in America/New_York (setupSharedPostgreSQL).
func useLocalZone(t *testing.T) {
	t.Helper()
	old := time.Local
	time.Local = time.FixedZone("UTC+2", 2*60*60)
	t.Cleanup(func() { time.Local = old })
}

func TestPostgresTimeColumnsHaveTimeZone(t *testing.T) {
	td := SetupTestDatabase(t, config.Postgres)
	defer td.Close()

	rows, err := td.DB.QueryContext(t.Context(), `SELECT table_name || '.' || column_name
		FROM information_schema.columns
		WHERE table_schema = 'public' AND data_type = 'timestamp without time zone'
		ORDER BY 1`)
	require.NoError(t, err)
	defer rows.Close()

	var columns []string
	for rows.Next() {
		var column string
		require.NoError(t, rows.Scan(&column))
		columns = append(columns, column)
	}
	require.NoError(t, rows.Err())

	assert.Empty(t, columns, "use TIMESTAMP WITH TIME ZONE for these columns (docs/adr/0004)")
}

func saveResultsAt(t *testing.T, td *TestDatabase, plMonitor, dnsMonitor int64, createdAt time.Time) map[ExportDataset]int64 {
	t.Helper()

	st, err := td.Service.SaveSpeedTest(t.Context(), types.SpeedTestResult{
		ServerName: "server", ServerID: "1", TestType: "speedtest", CreatedAt: createdAt,
	})
	require.NoError(t, err)

	pl := &types.PacketLossResult{MonitorID: plMonitor, PacketsSent: 10, PacketsRecv: 10, CreatedAt: createdAt}
	require.NoError(t, td.Service.SavePacketLossResult(pl))

	dns := &types.DNSResult{MonitorID: dnsMonitor, ResponseTimeMs: 12, ResponseCode: "NOERROR", Success: true, CreatedAt: createdAt}
	require.NoError(t, td.Service.SaveDNSResult(dns))

	return map[ExportDataset]int64{ExportSpeedTests: st.ID, ExportPacketLoss: pl.ID, ExportDNS: dns.ID}
}

func TestPurgeComparesInstants(t *testing.T) {
	useLocalZone(t)
	RunTestWithBothDatabases(t, func(t *testing.T, td *TestDatabase) {
		plMonitor := CreateTestPacketLossMonitor(t, td).ID
		dnsMonitor := CreateTestDNSMonitor(t, td).ID
		cutoff := time.Now().Add(-30 * 24 * time.Hour)

		older := saveResultsAt(t, td, plMonitor, dnsMonitor, cutoff.Add(-time.Hour))
		newer := saveResultsAt(t, td, plMonitor, dnsMonitor, cutoff.Add(time.Hour))

		speedTests, packetLoss, dnsResults, err := td.Service.PurgeHistoricalData(t.Context(), cutoff)
		require.NoError(t, err)
		assert.Equal(t, []int64{1, 1, 1}, []int64{speedTests, packetLoss, dnsResults})

		tables := map[ExportDataset]string{ExportSpeedTests: "speed_tests", ExportPacketLoss: "packet_loss_results", ExportDNS: "dns_results"}
		for dataset, table := range tables {
			AssertRecordNotExists(t, td, table, "id", older[dataset])
			AssertRecordExists(t, td, table, "id", newer[dataset])
		}
	})
}

func TestExportDateRangeComparesInstants(t *testing.T) {
	useLocalZone(t)
	RunTestWithBothDatabases(t, func(t *testing.T, td *TestDatabase) {
		plMonitor := CreateTestPacketLossMonitor(t, td).ID
		dnsMonitor := CreateTestDNSMonitor(t, td).ID
		to := time.Now().Add(-24 * time.Hour)
		from := to.Add(-24 * time.Hour)

		saveResultsAt(t, td, plMonitor, dnsMonitor, from.Add(-time.Hour))
		first := saveResultsAt(t, td, plMonitor, dnsMonitor, from.Add(time.Hour))
		last := saveResultsAt(t, td, plMonitor, dnsMonitor, to.Add(-time.Hour))
		saveResultsAt(t, td, plMonitor, dnsMonitor, to.Add(time.Hour))

		for _, dataset := range []ExportDataset{ExportSpeedTests, ExportPacketLoss, ExportDNS} {
			var buf bytes.Buffer
			require.NoError(t, td.Service.ExportResults(t.Context(), &buf, dataset, ExportJSON, from, to))
			var rows []struct {
				ID int64 `json:"id"`
			}
			require.NoError(t, json.Unmarshal(buf.Bytes(), &rows))
			got := make([]int64, len(rows))
			for i, r := range rows {
				got[i] = r.ID
			}
			assert.Equal(t, []int64{first[dataset], last[dataset]}, got, dataset)
		}
	})
}

func TestSpeedTestHistory24hComparesInstants(t *testing.T) {
	useLocalZone(t)
	RunTestWithBothDatabases(t, func(t *testing.T, td *TestDatabase) {
		now := time.Now()
		for _, age := range []time.Duration{23 * time.Hour, 25 * time.Hour} {
			_, err := td.Service.SaveSpeedTest(t.Context(), types.SpeedTestResult{
				ServerName: age.String(), ServerID: "1", TestType: "speedtest", CreatedAt: now.Add(-age),
			})
			require.NoError(t, err)
		}

		got, err := td.Service.GetSpeedTests(t.Context(), "24h", 1, 10)
		require.NoError(t, err)
		require.Len(t, got.Data, 1)
		assert.Equal(t, "23h0m0s", got.Data[0].ServerName)
	})
}

func TestMonitorResourceStatsWindowsCompareInstants(t *testing.T) {
	useLocalZone(t)
	RunTestWithBothDatabases(t, func(t *testing.T, td *TestDatabase) {
		agent, err := td.Service.CreateMonitorAgent(t.Context(), &types.MonitorAgent{Name: "agent", URL: "http://127.0.0.1:8200", Enabled: true})
		require.NoError(t, err)

		svc := td.Service.(*service)
		saveAt := func(createdAt time.Time, cpu float64) {
			t.Helper()
			_, err := svc.insert(t.Context(), "monitor_resource_stats", map[string]any{
				"agent_id": agent.ID, "cpu_usage_percent": cpu, "memory_used_percent": 0, "swap_used_percent": 0,
				"disk_usage_json": "[]", "temperature_json": "[]", "uptime_seconds": 0, "created_at": createdAt,
			})
			require.NoError(t, err)
		}
		cpuValues := func(hours int) []float64 {
			t.Helper()
			stats, err := td.Service.GetMonitorResourceStats(t.Context(), agent.ID, hours)
			require.NoError(t, err)
			got := make([]float64, len(stats))
			for i, s := range stats {
				got[i] = s.CPUUsagePercent
			}
			return got
		}

		// The agent poller saves with the database default time.
		require.NoError(t, td.Service.SaveMonitorResourceStats(t.Context(), agent.ID, &types.MonitorResourceStats{CPUUsagePercent: 1}))
		now := time.Now()
		saveAt(now.Add(-time.Hour), 2)
		saveAt(now.Add(-3*time.Hour), 3)
		saveAt(now.Add(-23*time.Hour), 4)
		saveAt(now.Add(-25*time.Hour), 5)

		assert.Equal(t, []float64{1, 2, 3, 4}, cpuValues(24), "24-hour window")

		require.NoError(t, td.Service.CleanupMonitorData(t.Context()))
		assert.Equal(t, []float64{1, 2}, cpuValues(24), "2-hour cleanup")
	})
}

func TestResultTimesReadBackInUTC(t *testing.T) {
	useLocalZone(t)
	RunTestWithBothDatabases(t, func(t *testing.T, td *TestDatabase) {
		createdAt := time.Date(2026, 11, 12, 10, 0, 0, 0, time.Local)
		want := "2026-11-12T08:00:00Z"

		plMonitor := CreateTestPacketLossMonitor(t, td).ID
		require.NoError(t, td.Service.SavePacketLossResult(&types.PacketLossResult{MonitorID: plMonitor, PacketsSent: 10, PacketsRecv: 10, CreatedAt: createdAt}))
		pl, err := td.Service.GetPacketLossResults(plMonitor, 1, 10)
		require.NoError(t, err)
		require.Len(t, pl.Data, 1)
		assert.Equal(t, want, pl.Data[0].CreatedAt.Format(time.RFC3339), "packet loss")

		dnsMonitor := CreateTestDNSMonitor(t, td).ID
		require.NoError(t, td.Service.SaveDNSResult(&types.DNSResult{MonitorID: dnsMonitor, Success: true, CreatedAt: createdAt}))
		dns, err := td.Service.GetDNSResults(dnsMonitor, 1, 10)
		require.NoError(t, err)
		require.Len(t, dns.Data, 1)
		assert.Equal(t, want, dns.Data[0].CreatedAt.Format(time.RFC3339), "DNS")
	})
}

func TestWrittenObjectTimesAreUTC(t *testing.T) {
	useLocalZone(t)
	RunTestWithBothDatabases(t, func(t *testing.T, td *TestDatabase) {
		assertUTC := func(name string, times ...time.Time) {
			t.Helper()
			for _, tm := range times {
				assert.Equal(t, time.UTC, tm.Location(), name)
			}
		}

		pl := CreateTestPacketLossMonitor(t, td)
		assertUTC("CreatePacketLossMonitor", pl.CreatedAt, pl.UpdatedAt)
		require.NoError(t, td.Service.UpdatePacketLossMonitor(pl))
		assertUTC("UpdatePacketLossMonitor", pl.UpdatedAt)

		dns := CreateTestDNSMonitor(t, td)
		assertUTC("CreateDNSMonitor", dns.CreatedAt, dns.UpdatedAt)
		require.NoError(t, td.Service.UpdateDNSMonitor(dns))
		assertUTC("UpdateDNSMonitor", dns.UpdatedAt)

		agent, err := td.Service.CreateMonitorAgent(t.Context(), &types.MonitorAgent{Name: "agent", URL: "http://127.0.0.1:8200", Enabled: true})
		require.NoError(t, err)
		assertUTC("CreateMonitorAgent", agent.CreatedAt, agent.UpdatedAt)
		require.NoError(t, td.Service.UpdateMonitorAgent(t.Context(), agent))
		assertUTC("UpdateMonitorAgent", agent.UpdatedAt)

		channelInput := NotificationChannelInput{Name: "channel", URL: "json://127.0.0.1:9000"}
		channel, err := td.Service.CreateChannel(channelInput)
		require.NoError(t, err)
		assertUTC("CreateChannel", channel.CreatedAt, channel.UpdatedAt)
		channel, err = td.Service.UpdateChannel(channel.ID, channelInput)
		require.NoError(t, err)
		assertUTC("UpdateChannel", channel.CreatedAt, channel.UpdatedAt)

		events, err := td.Service.GetEvents()
		require.NoError(t, err)
		require.NotEmpty(t, events)
		ruleInput := NotificationRuleInput{ChannelID: channel.ID, EventID: events[0].ID}
		rule, err := td.Service.CreateRule(ruleInput)
		require.NoError(t, err)
		assertUTC("CreateRule", rule.CreatedAt, rule.UpdatedAt)
		rule, err = td.Service.UpdateRule(rule.ID, ruleInput)
		require.NoError(t, err)
		assertUTC("UpdateRule", rule.CreatedAt, rule.UpdatedAt)
	})
}
