// Copyright (c) 2024-2025, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestIsWhitelisted(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name       string
		remoteAddr string
		whitelist  []string
		expected   bool
	}{
		{
			name:       "IP in whitelist",
			remoteAddr: "192.168.1.100",
			whitelist:  []string{"192.168.1.0/24"},
			expected:   true,
		},
		{
			name:       "IP not in whitelist",
			remoteAddr: "10.0.0.5",
			whitelist:  []string{"192.168.1.0/24"},
			expected:   false,
		},
		{
			name:       "Multiple whitelists",
			remoteAddr: "10.0.0.5",
			whitelist:  []string{"192.168.1.0/24", "10.0.0.0/8"},
			expected:   true,
		},
		{
			name:       "Empty whitelist",
			remoteAddr: "192.168.1.100",
			whitelist:  []string{},
			expected:   false,
		},
		{
			name:       "Invalid CIDR",
			remoteAddr: "192.168.1.100",
			whitelist:  []string{"invalid"},
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = &http.Request{RemoteAddr: tt.remoteAddr + ":12345"}

			if got := isWhitelisted(c, tt.whitelist); got != tt.expected {
				t.Errorf("isWhitelisted() = %v, want %v", got, tt.expected)
			}
		})
	}
}
