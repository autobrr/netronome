// Copyright (c) 2024-2026, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package database

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLatencyMs(t *testing.T) {
	tests := []struct {
		name    string
		latency string
		want    any
	}{
		{name: "speedtest.net duration", latency: "12.345ms", want: 12.345},
		{name: "microseconds", latency: "850µs", want: 0.85},
		{name: "seconds", latency: "1.2s", want: 1200.0},
		{name: "librespeed plain milliseconds", latency: "12.35", want: 12.35},
		{name: "iperf3 ping", latency: "15.20 ms", want: 15.2},
		{name: "iperf3 without ping data", latency: "0ms", want: 0.0},
		{name: "empty", latency: "", want: nil},
		{name: "garbage", latency: "fast", want: nil},
		{name: "not a number", latency: "NaN", want: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := latencyMs(tt.latency)
			if tt.want == nil {
				assert.Nil(t, got)
				return
			}
			assert.InDelta(t, tt.want, got, 1e-9)
		})
	}
}
