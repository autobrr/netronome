// Copyright (c) 2024-2026, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package database

import (
	"bytes"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/autobrr/netronome/internal/types"
)

// saveExportSpeedTests stores a complete result and a sparse result, newest first.
func saveExportSpeedTests(t *testing.T, td *TestDatabase) (full, sparse *types.SpeedTestResult) {
	t.Helper()
	ctx := t.Context()

	sparse, err := td.Service.SaveSpeedTest(ctx, types.SpeedTestResult{
		ServerName: "=1+1",
		ServerID:   "librespeed:7",
		TestType:   "librespeed",
		Latency:    "fast",
		CreatedAt:  time.Date(2026, 10, 2, 11, 30, 0, 0, time.UTC),
	})
	require.NoError(t, err)

	full, err = td.Service.SaveSpeedTest(ctx, types.SpeedTestResult{
		ServerName:    "Oslo Fiber",
		ServerID:      "1234",
		ServerHost:    new("speed.example.test:8080"),
		ServerCity:    new("Oslo"),
		TestType:      "speedtest",
		DownloadSpeed: 500.5,
		UploadSpeed:   100.25,
		Latency:       "12.345ms",
		Jitter:        new(1.5),
		IsScheduled:   true,
		CreatedAt:     time.Date(2026, 10, 1, 10, 0, 0, 0, time.UTC),
	})
	require.NoError(t, err)

	return full, sparse
}

func TestExportResultsSpeedTestsCSV(t *testing.T) {
	RunTestWithBothDatabases(t, func(t *testing.T, td *TestDatabase) {
		full, sparse := saveExportSpeedTests(t, td)

		var buf bytes.Buffer
		require.NoError(t, td.Service.ExportResults(t.Context(), &buf, ExportSpeedTests, ExportCSV, time.Time{}, time.Time{}))

		want := "id,created_at,test_type,server_name,server_id,server_host,server_city,download_mbps,upload_mbps,latency_ms,jitter_ms,is_scheduled\n" +
			fmt.Sprintf("%d,2026-10-01T10:00:00Z,speedtest,Oslo Fiber,1234,speed.example.test:8080,Oslo,500.5,100.25,12.345,1.5,true\n", full.ID) +
			fmt.Sprintf("%d,2026-10-02T11:30:00Z,librespeed,'=1+1,librespeed:7,,,0,0,,,false\n", sparse.ID)
		assert.Equal(t, want, buf.String())
	})
}

func TestExportResultsSpeedTestsJSON(t *testing.T) {
	RunTestWithBothDatabases(t, func(t *testing.T, td *TestDatabase) {
		full, sparse := saveExportSpeedTests(t, td)

		var buf bytes.Buffer
		require.NoError(t, td.Service.ExportResults(t.Context(), &buf, ExportSpeedTests, ExportJSON, time.Time{}, time.Time{}))

		want := fmt.Sprintf(`[
			{"id": %d, "created_at": "2026-10-01T10:00:00Z", "test_type": "speedtest", "server_name": "Oslo Fiber", "server_id": "1234",
			 "server_host": "speed.example.test:8080", "server_city": "Oslo", "download_mbps": 500.5, "upload_mbps": 100.25,
			 "latency_ms": 12.345, "jitter_ms": 1.5, "is_scheduled": true},
			{"id": %d, "created_at": "2026-10-02T11:30:00Z", "test_type": "librespeed", "server_name": "=1+1", "server_id": "librespeed:7",
			 "server_host": null, "server_city": null, "download_mbps": 0, "upload_mbps": 0,
			 "latency_ms": null, "jitter_ms": null, "is_scheduled": false}
		]`, full.ID, sparse.ID)
		assert.JSONEq(t, want, buf.String())
		assert.Less(t, bytes.Index(buf.Bytes(), []byte(`"id"`)), bytes.Index(buf.Bytes(), []byte(`"created_at"`)), "keys keep the column order")
	})
}

func TestExportResultsEmptyJSON(t *testing.T) {
	RunTestWithBothDatabases(t, func(t *testing.T, td *TestDatabase) {
		var buf bytes.Buffer
		require.NoError(t, td.Service.ExportResults(t.Context(), &buf, ExportSpeedTests, ExportJSON, time.Time{}, time.Time{}))
		assert.JSONEq(t, `[]`, buf.String())
	})
}

func TestExportResultsPacketLossCSV(t *testing.T) {
	RunTestWithBothDatabases(t, func(t *testing.T, td *TestDatabase) {
		monitor := CreateTestPacketLossMonitor(t, td)
		result := &types.PacketLossResult{
			MonitorID: monitor.ID, PacketLoss: 10, MinRTT: 1.5, AvgRTT: 2.5, MaxRTT: 4, StdDevRTT: 0.5,
			PacketsSent: 10, PacketsRecv: 9, UsedMTR: true, HopCount: 7,
			CreatedAt: time.Date(2026, 10, 3, 8, 0, 0, 0, time.UTC),
		}
		require.NoError(t, td.Service.SavePacketLossResult(result))

		var buf bytes.Buffer
		require.NoError(t, td.Service.ExportResults(t.Context(), &buf, ExportPacketLoss, ExportCSV, time.Time{}, time.Time{}))

		want := "id,created_at,monitor_id,monitor_name,monitor_host,packet_loss_percent,min_rtt_ms,avg_rtt_ms,max_rtt_ms,std_dev_rtt_ms,packets_sent,packets_received,used_mtr,hop_count\n" +
			fmt.Sprintf("%d,2026-10-03T08:00:00Z,%d,Test Monitor,8.8.8.8,10,1.5,2.5,4,0.5,10,9,true,7\n", result.ID, monitor.ID)
		assert.Equal(t, want, buf.String())
	})
}

func TestExportResultsDNSJSON(t *testing.T) {
	RunTestWithBothDatabases(t, func(t *testing.T, td *TestDatabase) {
		monitor := CreateTestDNSMonitor(t, td)
		result := &types.DNSResult{
			MonitorID: monitor.ID, ResponseTimeMs: 2000, ResponseCode: "", Success: false, Error: new("i/o timeout"),
			CreatedAt: time.Date(2026, 10, 4, 9, 0, 0, 0, time.UTC),
		}
		require.NoError(t, td.Service.SaveDNSResult(result))

		var buf bytes.Buffer
		require.NoError(t, td.Service.ExportResults(t.Context(), &buf, ExportDNS, ExportJSON, time.Time{}, time.Time{}))

		want := fmt.Sprintf(`[{"id": %d, "created_at": "2026-10-04T09:00:00Z", "monitor_id": %d, "monitor_name": "Test DNS Monitor",
			"resolver": "127.0.0.1:5353", "protocol": "udp", "query": "example.invalid", "record_type": "A",
			"response_time_ms": 2000, "response_code": "", "success": false, "error": "i/o timeout"}]`, result.ID, monitor.ID)
		assert.JSONEq(t, want, buf.String())
	})
}

func TestExportResultsDateRange(t *testing.T) {
	RunTestWithBothDatabases(t, func(t *testing.T, td *TestDatabase) {
		monitor := CreateTestDNSMonitor(t, td)
		from := time.Date(2026, 10, 17, 0, 0, 0, 0, time.UTC)
		to := time.Date(2026, 10, 18, 0, 0, 0, 0, time.UTC)
		plus2 := time.FixedZone("+02:00", 2*60*60)

		save := func(createdAt time.Time, ms float64) {
			t.Helper()
			require.NoError(t, td.Service.SaveDNSResult(&types.DNSResult{MonitorID: monitor.ID, ResponseTimeMs: ms, Success: true, CreatedAt: createdAt}))
		}
		save(to, 4)                                         // excluded: to is exclusive
		save(from.Add(-time.Second), 1)                     // excluded: before from
		save(time.Date(2026, 10, 17, 1, 0, 0, 0, plus2), 2) // excluded: 23:00 UTC on October 16, stored with a local offset
		save(from.Add(12*time.Hour), 3)                     // included
		save(from, 5)                                       // included: from is inclusive

		export := func(from, to time.Time) []float64 {
			t.Helper()
			var buf bytes.Buffer
			require.NoError(t, td.Service.ExportResults(t.Context(), &buf, ExportDNS, ExportJSON, from, to))
			var rows []struct {
				ResponseTimeMs float64 `json:"response_time_ms"`
			}
			require.NoError(t, json.Unmarshal(buf.Bytes(), &rows))
			got := make([]float64, len(rows))
			for i, r := range rows {
				got[i] = r.ResponseTimeMs
			}
			return got
		}

		assert.Equal(t, []float64{5, 3}, export(from, to), "range, oldest first")
		assert.Equal(t, []float64{2, 1, 5, 3}, export(time.Time{}, to), "no lower limit")
		assert.Equal(t, []float64{5, 3, 4}, export(from, time.Time{}), "no upper limit")
		assert.Equal(t, []float64{2, 1, 5, 3, 4}, export(time.Time{}, time.Time{}), "no limits")
	})
}

func TestExportResultsRejectsUnknownInput(t *testing.T) {
	RunTestWithBothDatabases(t, func(t *testing.T, td *TestDatabase) {
		var buf bytes.Buffer
		assert.ErrorIs(t, td.Service.ExportResults(t.Context(), &buf, "bandwidth", ExportCSV, time.Time{}, time.Time{}), ErrInvalidInput)
		assert.ErrorIs(t, td.Service.ExportResults(t.Context(), &buf, ExportDNS, "xml", time.Time{}, time.Time{}), ErrInvalidInput)
		assert.Empty(t, buf.String())
	})
}
