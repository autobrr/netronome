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

func TestBuildArgsIncludesShareWhenEnabled(t *testing.T) {
	runner := NewLibrespeedRunner(config.LibrespeedConfig{
		ServersPath:  "/tmp/local-servers.json",
		ShareResults: true,
	})

	args := runner.buildArgs(&types.TestOptions{
		IsPublicServer: true,
		ServerIDs:      []string{"123"},
	})

	assert.Contains(t, args, "--share")
}

func TestBuildArgsExcludesShareWhenDisabled(t *testing.T) {
	runner := NewLibrespeedRunner(config.LibrespeedConfig{
		ServersPath:  "/tmp/local-servers.json",
		ShareResults: false,
	})

	args := runner.buildArgs(&types.TestOptions{
		IsPublicServer: true,
		ServerIDs:      []string{"123"},
	})

	assert.NotContains(t, args, "--share")
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
