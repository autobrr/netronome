// Copyright (c) 2024-2026, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package speedtest

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/autobrr/netronome/internal/config"
	"github.com/autobrr/netronome/internal/types"
)

func TestBuildArgsUsesServerJSONForPublicServers(t *testing.T) {
	runner := NewLibrespeedRunner(config.LibrespeedConfig{
		ServersPath: "/tmp/local-servers.json",
	})

	args := runner.buildArgs(&types.TestOptions{
		IsPublicServer: true,
		ServerIDs:      []string{"123"},
	})

	assert.Equal(t, []string{
		"--json",
		"--server-json", librespeedPublicServersURL,
		"--server", "123",
	}, args)
}

func TestBuildArgsShareFlag(t *testing.T) {
	tests := []struct {
		name         string
		shareResults bool
		wantShare    bool
	}{
		{"included when enabled", true, true},
		{"excluded when disabled", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner := NewLibrespeedRunner(config.LibrespeedConfig{
				ServersPath:  "/tmp/local-servers.json",
				ShareResults: tt.shareResults,
			})

			args := runner.buildArgs(&types.TestOptions{
				IsPublicServer: true,
				ServerIDs:      []string{"123"},
			})

			if tt.wantShare {
				assert.Contains(t, args, "--share")
			} else {
				assert.NotContains(t, args, "--share")
			}
		})
	}
}

func TestSanitizeResultURL(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{"valid https URL", "https://librespeed.org/results/?id=abc123", "https://librespeed.org/results/?id=abc123"},
		{"valid http URL", "http://librespeed.org/results/?id=abc123", "http://librespeed.org/results/?id=abc123"},
		{"empty string", "", ""},
		{"no scheme", "librespeed.org/results", ""},
		{"javascript scheme", "javascript:alert(1)", ""},
		{"ftp scheme", "ftp://example.com/file", ""},
		{"no host", "https:///path", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, sanitizeResultURL(tt.raw))
		})
	}
}

func TestBuildArgsUsesLocalJSONForCustomServers(t *testing.T) {
	runner := NewLibrespeedRunner(config.LibrespeedConfig{
		ServersPath: "/tmp/local-servers.json",
	})

	args := runner.buildArgs(&types.TestOptions{
		IsPublicServer: false,
		ServerIDs:      []string{"42"},
	})

	assert.Equal(t, []string{
		"--json",
		"--local-json", "/tmp/local-servers.json",
		"--server", "42",
	}, args)
}
