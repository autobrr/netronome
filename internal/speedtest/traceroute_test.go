// Copyright (c) 2024-2026, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package speedtest

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeTracerouteHost(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "hostname with port",
			input:    "example.com:8443",
			expected: "example.com",
		},
		{
			name:     "ipv6 literal",
			input:    "2001:db8::1",
			expected: "2001:db8::1",
		},
		{
			name:     "bracketed ipv6 with port",
			input:    "[2001:db8::1]:8080",
			expected: "2001:db8::1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, normalizeTracerouteHost(tt.input))
		})
	}
}

func TestParseHopLineIPv6Address(t *testing.T) {
	s := &service{}

	hop := s.parseHopLine(" 2  2001:db8::1  4.321 ms  4.567 ms  4.890 ms")
	require.NotNil(t, hop)

	assert.Equal(t, 2, hop.Number)
	assert.Equal(t, "2001:db8::1", hop.Host)
	assert.Equal(t, "2001:db8::1", hop.IP)
	assert.Equal(t, 4.321, hop.RTT1)
	assert.Equal(t, 4.567, hop.RTT2)
	assert.Equal(t, 4.890, hop.RTT3)
	assert.False(t, hop.Timeout)
}

func TestParseUnixTracerouteOutputIPv6Address(t *testing.T) {
	s := &service{}
	lines := []string{
		"traceroute to ipv6.google.com (2a00:1450:400f:802::200e), 30 hops max, 60 byte packets",
		" 1  2001:db8::1  1.100 ms  1.200 ms  1.300 ms",
	}

	result, err := s.parseUnixTracerouteOutput(lines, &TracerouteResult{
		Destination: "ipv6.google.com",
		Hops:        []TracerouteHop{},
	})
	require.NoError(t, err)
	require.Len(t, result.Hops, 1)
	assert.Equal(t, "2001:db8::1", result.Hops[0].IP)
}
