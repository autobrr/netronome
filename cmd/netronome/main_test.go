// Copyright (c) 2024-2026, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// A config file that does not decode must stop the agent. It must not start on
// defaults, because the defaults have no API key.
func TestRunAgentFailsOnBadConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	require.NoError(t, os.WriteFile(path, []byte("[agent\n"), 0o600))

	old := configPath
	configPath = path
	t.Cleanup(func() { configPath = old })

	require.ErrorContains(t, runAgent(agentCmd, nil), "failed to load configuration")
}
