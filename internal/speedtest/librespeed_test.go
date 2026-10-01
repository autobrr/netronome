// Copyright (c) 2024-2026, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package speedtest

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/autobrr/netronome/internal/config"
	"github.com/autobrr/netronome/internal/types"
)

func TestResultFromLibrespeedMapsServerIdentity(t *testing.T) {
	timestamp := time.Date(2026, time.September, 15, 1, 2, 3, 0, time.UTC)
	result := resultFromLibrespeed(LibrespeedResult{
		Timestamp: timestamp,
		Server: LibrespeedServerInfo{
			Name: "Example LibreSpeed",
			URL:  "https://speed.example.com/",
		},
	}, "42")

	assert.Equal(t, timestamp, result.Timestamp)
	assert.Equal(t, "Example LibreSpeed", result.Server)
	assert.Equal(t, "42", result.ServerID)
	assert.Equal(t, "https://speed.example.com/", result.ServerHost)

	withoutHost := resultFromLibrespeed(LibrespeedResult{}, "")
	assert.Empty(t, withoutHost.ServerHost)
}

func TestLibrespeedServerIdentityIncludesCatalogueSource(t *testing.T) {
	assert.Equal(t, "public-42", librespeedServerIdentity("42", true))
	assert.Equal(t, "custom-42", librespeedServerIdentity("42", false))
	assert.Empty(t, librespeedServerIdentity("", true))
}

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
