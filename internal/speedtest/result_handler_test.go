// Copyright (c) 2024-2026, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package speedtest

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/autobrr/netronome/internal/types"
)

func TestResultForStorage(t *testing.T) {
	createdAt := time.Date(2026, time.September, 15, 1, 2, 3, 0, time.UTC)
	tests := []struct {
		name     string
		result   Result
		testType string
		opts     types.TestOptions
		at       time.Time
		want     types.SpeedTestResult
	}{
		{
			name: "speedtest stable identity",
			result: Result{
				Server:     "Example ISP",
				ServerID:   "1234",
				ServerHost: "speed.example.com:8080",
				ServerCity: "Brisbane",
				Jitter:     1.5,
			},
			testType: "speedtest",
			opts:     types.TestOptions{IsScheduled: true},
			at:       createdAt,
			want: types.SpeedTestResult{
				ServerName:  "Example ISP",
				ServerID:    "1234",
				ServerHost:  new("speed.example.com:8080"),
				ServerCity:  new("Brisbane"),
				TestType:    "speedtest",
				Jitter:      new(1.5),
				IsScheduled: true,
				CreatedAt:   createdAt,
			},
		},
		{
			name:     "speedtest legacy fallback",
			result:   Result{Server: "Example ISP"},
			testType: "speedtest",
			want: types.SpeedTestResult{
				ServerName: "Example ISP",
				ServerID:   "Example ISP",
				TestType:   "speedtest",
			},
		},
		{
			name: "librespeed public stable identity",
			result: Result{
				Server:     "Example LibreSpeed",
				ServerID:   librespeedServerIdentity("42", true),
				ServerHost: "https://speed.example.com/",
			},
			testType: "librespeed",
			want: types.SpeedTestResult{
				ServerName: "Example LibreSpeed",
				ServerID:   "librespeed-public-42",
				ServerHost: new("https://speed.example.com/"),
				TestType:   "librespeed",
			},
		},
		{
			name:     "librespeed custom stable identity",
			result:   Result{Server: "Custom", ServerID: librespeedServerIdentity("42", false)},
			testType: "librespeed",
			want: types.SpeedTestResult{
				ServerName: "Custom",
				ServerID:   "librespeed-custom-42",
				TestType:   "librespeed",
			},
		},
		{
			name:     "librespeed legacy fallback",
			result:   Result{Server: "Example LibreSpeed"},
			testType: "librespeed",
			want: types.SpeedTestResult{
				ServerName: "Example LibreSpeed",
				ServerID:   "librespeed-Example LibreSpeed",
				TestType:   "librespeed",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resultForStorage(&tt.result, tt.testType, &tt.opts, tt.at)
			assert.Equal(t, tt.want, got)
		})
	}
}
