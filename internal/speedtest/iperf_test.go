// Copyright (c) 2024-2026, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package speedtest

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseIperfFinalMetrics_StreamOutput(t *testing.T) {
	output := `{"event":"interval","data":{"sum":{"bits_per_second":123456789}}}
{"event":"end","data":{"sum_sent":{"bits_per_second":1200000000,"jitter_ms":0.50},"sum_received":{"bits_per_second":980000000,"jitter_ms":1.25}}}
`

	downloadSpeed, downloadJitter, err := parseIperfFinalMetrics(output, true)
	require.NoError(t, err)
	assert.InDelta(t, 980.0, downloadSpeed, 0.0001)
	require.NotNil(t, downloadJitter)
	assert.InDelta(t, 1.25, *downloadJitter, 0.0001)

	uploadSpeed, uploadJitter, err := parseIperfFinalMetrics(output, false)
	require.NoError(t, err)
	assert.InDelta(t, 1200.0, uploadSpeed, 0.0001)
	require.NotNil(t, uploadJitter)
	assert.InDelta(t, 0.50, *uploadJitter, 0.0001)
}

func TestParseIperfFinalMetrics_FallbackToMonolithicJSON(t *testing.T) {
	output := `{
  "end": {
    "sum_sent": {
      "bits_per_second": 250000000,
      "jitter_ms": 0.75
    },
    "sum_received": {
      "bits_per_second": 500000000,
      "jitter_ms": 1.50
    }
  }
}`

	speed, jitter, err := parseIperfFinalMetrics(output, true)
	require.NoError(t, err)
	assert.InDelta(t, 500.0, speed, 0.0001)
	require.NotNil(t, jitter)
	assert.InDelta(t, 1.50, *jitter, 0.0001)
}

func TestParseIperfFinalMetrics_ZeroJitterReturnsNilPointer(t *testing.T) {
	output := `{"event":"end","data":{"sum_sent":{"bits_per_second":100000000,"jitter_ms":0},"sum_received":{"bits_per_second":100000000,"jitter_ms":0}}}`

	speed, jitter, err := parseIperfFinalMetrics(output, true)
	require.NoError(t, err)
	assert.InDelta(t, 100.0, speed, 0.0001)
	assert.Nil(t, jitter)
}

func TestParseIperfFinalMetrics_InvalidOutput(t *testing.T) {
	_, _, err := parseIperfFinalMetrics("not-json", true)
	require.Error(t, err)
}

func TestParseIperfFinalMetrics_LargeMonolithicJSON(t *testing.T) {
	largePadding := strings.Repeat("a", 70*1024)
	output := fmt.Sprintf(`{"end":{"sum_sent":{"bits_per_second":250000000,"jitter_ms":0.75},"sum_received":{"bits_per_second":500000000,"jitter_ms":1.50}},"padding":"%s"}`, largePadding)

	speed, jitter, err := parseIperfFinalMetrics(output, true)
	require.NoError(t, err)
	assert.InDelta(t, 500.0, speed, 0.0001)
	require.NotNil(t, jitter)
	assert.InDelta(t, 1.50, *jitter, 0.0001)
}

func TestFormatIperfFailureOutput(t *testing.T) {
	assert.Equal(t, "stdout={\"ok\":true} stderr=unknown option --json-stream", formatIperfFailureOutput("{\"ok\":true}", "unknown option --json-stream"))
	assert.Equal(t, "stdout={\"ok\":true}", formatIperfFailureOutput("{\"ok\":true}", ""))
	assert.Equal(t, "stderr=unknown option --json-stream", formatIperfFailureOutput("", "unknown option --json-stream"))
	assert.Equal(t, "no output", formatIperfFailureOutput("", ""))
}
