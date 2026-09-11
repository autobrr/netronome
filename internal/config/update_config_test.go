// Copyright (c) 2024-2026, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckForUpdatesConfiguration(t *testing.T) {
	t.Setenv("NETRONOME__CHECK_FOR_UPDATES", "false")

	path := filepath.Join(t.TempDir(), "config.toml")
	require.NoError(t, os.WriteFile(path, []byte("check_for_updates = true\n"), 0o600))

	cfg, err := Load(path)
	require.NoError(t, err)
	assert.False(t, cfg.CheckForUpdates)

	assert.True(t, New().CheckForUpdates)
}
