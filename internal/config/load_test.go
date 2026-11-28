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

func TestLoadConfigFile(t *testing.T) {
	tests := []struct {
		name     string
		explicit bool   // pass the file path to Load, or find it on a default path
		contents string // empty means no file
		wantErr  bool
	}{
		{name: "explicit path with bad TOML", explicit: true, contents: "[agent\n", wantErr: true},
		{name: "default path with bad TOML", contents: "[agent\n", wantErr: true},
		{name: "no file uses defaults and environment"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Point every default path at an empty temp directory.
			dir := t.TempDir()
			t.Setenv("HOME", dir)
			t.Setenv("XDG_CONFIG_HOME", dir)
			t.Chdir(dir)
			t.Setenv("NETRONOME__AGENT_API_KEY", "env-key")

			path := ""
			if tt.contents != "" {
				require.NoError(t, os.WriteFile(filepath.Join(dir, "config.toml"), []byte(tt.contents), 0o600))
			}
			if tt.explicit {
				path = filepath.Join(dir, "config.toml")
			}

			cfg, err := Load(path)
			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, cfg)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, "env-key", cfg.Agent.APIKey)
		})
	}
}
