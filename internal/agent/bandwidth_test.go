// Copyright (c) 2024-2025, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package agent

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeVnstatV1(t *testing.T) {
	// Simulated vnstat 1.18 JSON output
	v1JSON := `{
		"vnstatversion": "1.18",
		"jsonversion": "1",
		"interfaces": [{
			"id": "eth0",
			"nick": "",
			"traffic": {
				"total": {"rx": 1000000, "tx": 500000},
				"hours": [
					{"id": 0, "date": {"year": 2025, "month": 1, "day": 6}, "rx": 100, "tx": 50},
					{"id": 14, "date": {"year": 2025, "month": 1, "day": 6}, "rx": 200, "tx": 100}
				],
				"days": [
					{"id": 0, "date": {"year": 2025, "month": 1, "day": 6}, "rx": 5000, "tx": 2500}
				],
				"months": [
					{"id": 0, "date": {"year": 2025, "month": 1}, "rx": 100000, "tx": 50000}
				]
			}
		}]
	}`

	var data map[string]any
	require.NoError(t, json.Unmarshal([]byte(v1JSON), &data))

	normalizeVnstatV1(data)

	// jsonversion should be updated to "2"
	assert.Equal(t, "2", data["jsonversion"])

	interfaces := data["interfaces"].([]any)
	iface := interfaces[0].(map[string]any)
	traffic := iface["traffic"].(map[string]any)

	// "hours" should be renamed to "hour"
	assert.Nil(t, traffic["hours"], "old 'hours' key should be removed")
	hourData, ok := traffic["hour"].([]any)
	require.True(t, ok, "should have 'hour' key")
	assert.Len(t, hourData, 2)

	// Hour entries should have "time" field added
	firstHour := hourData[0].(map[string]any)
	timeField, ok := firstHour["time"].(map[string]any)
	require.True(t, ok, "hour entry should have 'time' field")
	assert.Equal(t, 0, timeField["hour"].(int))
	assert.Equal(t, 0, timeField["minute"].(int))

	secondHour := hourData[1].(map[string]any)
	timeField2 := secondHour["time"].(map[string]any)
	assert.Equal(t, 14, timeField2["hour"].(int))

	// "days" should be renamed to "day"
	assert.Nil(t, traffic["days"], "old 'days' key should be removed")
	dayData, ok := traffic["day"].([]any)
	require.True(t, ok, "should have 'day' key")
	assert.Len(t, dayData, 1)

	// "months" should be renamed to "month"
	assert.Nil(t, traffic["months"], "old 'months' key should be removed")
	monthData, ok := traffic["month"].([]any)
	require.True(t, ok, "should have 'month' key")
	assert.Len(t, monthData, 1)
}

func TestNormalizeVnstatV1_HourWithoutDay(t *testing.T) {
	// vnstat 1.18 may omit "day" from hour entries
	v1JSON := `{
		"vnstatversion": "1.18",
		"jsonversion": "1",
		"interfaces": [{
			"id": "eth0",
			"nick": "",
			"traffic": {
				"total": {"rx": 1000, "tx": 500},
				"hours": [
					{"id": 10, "date": {"year": 2025, "month": 1}, "rx": 100, "tx": 50}
				]
			}
		}]
	}`

	var data map[string]any
	require.NoError(t, json.Unmarshal([]byte(v1JSON), &data))

	normalizeVnstatV1(data)

	interfaces := data["interfaces"].([]any)
	iface := interfaces[0].(map[string]any)
	traffic := iface["traffic"].(map[string]any)
	hourData := traffic["hour"].([]any)
	firstHour := hourData[0].(map[string]any)

	timeField := firstHour["time"].(map[string]any)
	assert.Equal(t, 10, timeField["hour"].(int))

	// Should have day added to date
	dateField := firstHour["date"].(map[string]any)
	assert.NotNil(t, dateField["day"], "day should be added to date when missing")
}

func TestNormalizeVnstatV1_NoInterfaces(t *testing.T) {
	data := map[string]any{
		"vnstatversion": "1.18",
		"jsonversion":   "1",
	}

	// Should not panic
	normalizeVnstatV1(data)
	assert.Equal(t, "2", data["jsonversion"])
}

func TestNormalizeVnstatV1_EmptyTraffic(t *testing.T) {
	data := map[string]any{
		"vnstatversion": "1.18",
		"jsonversion":   "1",
		"interfaces": []any{
			map[string]any{
				"id":      "eth0",
				"traffic": map[string]any{},
			},
		},
	}

	// Should not panic
	normalizeVnstatV1(data)
	assert.Equal(t, "2", data["jsonversion"])
}

func TestFormatBytesPerSecond(t *testing.T) {
	tests := []struct {
		name     string
		input    int
		expected string
	}{
		{"zero", 0, "0 B/s"},
		{"bytes", 512, "512.00 B/s"},
		{"kibibytes", 1024, "1.00 KiB/s"},
		{"mebibytes", 1048576, "1.00 MiB/s"},
		{"gibibytes", 1073741824, "1.00 GiB/s"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, formatBytesPerSecond(tt.input))
		})
	}
}
