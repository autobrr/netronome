// Copyright (c) 2024-2026, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package speedtest

import (
	"runtime"
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

func TestParseUnixHopLine(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("these lines are Unix traceroute output")
	}

	tests := []struct {
		name string
		line string
		want *TracerouteHop
	}{
		{
			name: "three replies with host name",
			line: " 1  router.example (192.0.2.1)  0.512 ms  0.450 ms  0.470 ms",
			want: &TracerouteHop{Number: 1, Host: "router.example", IP: "192.0.2.1", RTT1: 0.512, RTT2: 0.450, RTT3: 0.470},
		},
		{
			name: "no replies",
			line: " 2  * * *",
			want: &TracerouteHop{Number: 2, Host: "*", IP: "*", Timeout: true},
		},
		{
			name: "first probe lost before the host name",
			line: " 3  * 192.0.2.1 (192.0.2.1)  2.345 ms  2.456 ms",
			want: &TracerouteHop{Number: 3, Host: "192.0.2.1", IP: "192.0.2.1", RTT2: 2.345, RTT3: 2.456},
		},
		{
			name: "middle probe lost",
			line: " 4  router.example (192.0.2.1)  2.300 ms * 2.500 ms",
			want: &TracerouteHop{Number: 4, Host: "router.example", IP: "192.0.2.1", RTT1: 2.300, RTT3: 2.500},
		},
		{
			name: "lost probe without a host name",
			line: " 5  * 1.234 ms 1.456 ms",
			want: &TracerouteHop{Number: 5, Host: "*", IP: "*", RTT2: 1.234, RTT3: 1.456},
		},
		{
			name: "ip only line",
			line: " 6  192.0.2.1  10.100 ms  10.200 ms  10.300 ms",
			want: &TracerouteHop{Number: 6, Host: "192.0.2.1", IP: "192.0.2.1", RTT1: 10.100, RTT2: 10.200, RTT3: 10.300},
		},
		{
			name: "icmp annotation between replies",
			line: " 7  router.example (192.0.2.1)  2.300 ms !H 2.500 ms  2.600 ms",
			want: &TracerouteHop{Number: 7, Host: "router.example", IP: "192.0.2.1", RTT1: 2.300, RTT2: 2.500, RTT3: 2.600},
		},
		{
			name: "single reply",
			line: " 8  router.example (192.0.2.1)  1.250 ms",
			want: &TracerouteHop{Number: 8, Host: "router.example", IP: "192.0.2.1", RTT1: 1.250},
		},
		{
			name: "header line",
			line: "traceroute to example.com (192.0.2.1), 30 hops max, 60 byte packets",
		},
		{
			name: "hop number without replies",
			line: " 9  ",
		},
	}

	s := &service{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hop := s.parseHopLine(tt.line)
			require.Equal(t, tt.want, hop)
		})
	}
}
