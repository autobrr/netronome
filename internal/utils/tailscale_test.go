// Copyright (c) 2024-2025, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package utils

import "testing"

func TestIsTailscaleIP(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected bool
	}{
		{
			name:     "Tailscale CGNAT IPv4",
			url:      "http://100.64.0.1:8200/events?stream=live-data",
			expected: true,
		},
		{
			name:     "Tailscale CGNAT IPv4 edge case",
			url:      "http://100.127.255.255:8200/events?stream=live-data",
			expected: true,
		},
		{
			name:     "Tailscale IPv6",
			url:      "http://[fd7a:115c:a1e0::1]:8200/events?stream=live-data",
			expected: true,
		},
		{
			name:     "Regular private IP",
			url:      "http://192.168.1.100:8200/events?stream=live-data",
			expected: false,
		},
		{
			name:     "Public IP",
			url:      "http://8.8.8.8:8200/events?stream=live-data",
			expected: false,
		},
		{
			name:     "Invalid IP range",
			url:      "http://100.63.255.255:8200/events?stream=live-data",
			expected: false,
		},
		{
			name:     "Invalid URL",
			url:      "not-a-url",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsTailscaleIP(tt.url)
			if result != tt.expected {
				t.Errorf("IsTailscaleIP(%s) = %v, expected %v", tt.url, result, tt.expected)
			}
		})
	}
}

func TestIsTailscaleHostname(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected bool
	}{
		{
			name:     "Tailscale MagicDNS .ts.net",
			url:      "http://my-device.ts.net:8200/events?stream=live-data",
			expected: true,
		},
		{
			name:     "Tailscale MagicDNS beta",
			url:      "http://my-device.beta.tailscale.net:8200/events?stream=live-data",
			expected: true,
		},
		{
			name:     "Tailscale MagicDNS alpha",
			url:      "http://my-device.alpha.tailscale.net:8200/events?stream=live-data",
			expected: true,
		},
		{
			name:     "Regular hostname",
			url:      "http://example.com:8200/events?stream=live-data",
			expected: false,
		},
		{
			name:     "Regular localhost",
			url:      "http://localhost:8200/events?stream=live-data",
			expected: false,
		},
		{
			name:     "Invalid URL",
			url:      "not-a-url",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsTailscaleHostname(tt.url)
			if result != tt.expected {
				t.Errorf("IsTailscaleHostname(%s) = %v, expected %v", tt.url, result, tt.expected)
			}
		})
	}
}

func TestIsTailscaleURL(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected bool
	}{
		{
			name:     "Tailscale CGNAT IP",
			url:      "http://100.64.0.1:8200/events?stream=live-data",
			expected: true,
		},
		{
			name:     "Tailscale MagicDNS",
			url:      "http://my-device.ts.net:8200/events?stream=live-data",
			expected: true,
		},
		{
			name:     "Regular IP",
			url:      "http://192.168.1.100:8200/events?stream=live-data",
			expected: false,
		},
		{
			name:     "Regular hostname",
			url:      "http://example.com:8200/events?stream=live-data",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsTailscaleURL(tt.url)
			if result != tt.expected {
				t.Errorf("IsTailscaleURL(%s) = %v, expected %v", tt.url, result, tt.expected)
			}
		})
	}
}