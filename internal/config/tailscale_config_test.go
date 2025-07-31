// Copyright (c) 2024-2025, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTailscaleConfig_AutoDetection(t *testing.T) {
	tests := []struct {
		name            string
		config          TailscaleConfig
		expectedMethod  string
		expectError     bool
		errorContains   string
	}{
		{
			name: "auto mode with auth key uses tsnet",
			config: TailscaleConfig{
				Enabled:  true,
				Method:   "auto",
				AuthKey:  "tskey-auth-test",
			},
			expectedMethod: "tsnet",
		},
		{
			name: "auto mode without auth key uses host",
			config: TailscaleConfig{
				Enabled: true,
				Method:  "auto",
				AuthKey: "",
			},
			expectedMethod: "host",
		},
		{
			name: "explicit tsnet mode requires auth key",
			config: TailscaleConfig{
				Enabled: true,
				Method:  "tsnet",
				AuthKey: "",
			},
			expectError:   true,
			errorContains: "auth_key is required when method is 'tsnet'",
		},
		{
			name: "explicit tsnet mode with auth key",
			config: TailscaleConfig{
				Enabled: true,
				Method:  "tsnet",
				AuthKey: "tskey-auth-test",
			},
			expectedMethod: "tsnet",
		},
		{
			name: "explicit host mode ignores auth key",
			config: TailscaleConfig{
				Enabled: true,
				Method:  "host",
				AuthKey: "tskey-auth-test", // Should be ignored
			},
			expectedMethod: "host",
		},
		{
			name: "disabled tailscale ignores all settings",
			config: TailscaleConfig{
				Enabled: false,
				Method:  "invalid",
				AuthKey: "test",
			},
			expectedMethod: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			method, err := tt.config.GetEffectiveMethod()
			
			if tt.expectError {
				require.Error(t, err, "expected error but got none")
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains, "error message mismatch")
				}
			} else {
				require.NoError(t, err, "unexpected error")
				assert.Equal(t, tt.expectedMethod, method, "method mismatch")
			}
		})
	}
}

func TestTailscaleConfig_Validation(t *testing.T) {
	tests := []struct {
		name          string
		config        TailscaleConfig
		expectError   bool
		errorContains string
	}{
		{
			name: "valid config with tsnet",
			config: TailscaleConfig{
				Enabled:           true,
				Method:            "tsnet",
				AuthKey:           "tskey-auth-test",
				Hostname:          "test-node",
				Ephemeral:         true,
				AgentPort:         8200,
				AutoDiscover:      true,
				DiscoveryInterval: "5m",
			},
			expectError: false,
		},
		{
			name: "valid config with host",
			config: TailscaleConfig{
				Enabled:           true,
				Method:            "host",
				AgentPort:         8200,
				AutoDiscover:      true,
				DiscoveryInterval: "5m",
			},
			expectError: false,
		},
		{
			name: "invalid method",
			config: TailscaleConfig{
				Enabled: true,
				Method:  "invalid",
			},
			expectError:   true,
			errorContains: "invalid method",
		},
		{
			name: "invalid discovery interval",
			config: TailscaleConfig{
				Enabled:           true,
				Method:            "host",
				AutoDiscover:      true,
				DiscoveryInterval: "invalid",
			},
			expectError:   true,
			errorContains: "invalid discovery interval",
		},
		{
			name: "negative agent port",
			config: TailscaleConfig{
				Enabled:   true,
				Method:    "host",
				AgentPort: -1,
			},
			expectError:   true,
			errorContains: "invalid agent port",
		},
		{
			name: "agent port too high",
			config: TailscaleConfig{
				Enabled:   true,
				Method:    "host",
				AgentPort: 70000,
			},
			expectError:   true,
			errorContains: "invalid agent port",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			
			if tt.expectError {
				require.Error(t, err, "expected error but got none")
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains, "error message mismatch")
				}
			} else {
				require.NoError(t, err, "unexpected error")
			}
		})
	}
}

func TestTailscaleConfig_EnvironmentOverrides(t *testing.T) {
	// Save original env and restore after test
	originalEnv := os.Environ()
	defer func() {
		os.Clearenv()
		for _, env := range originalEnv {
			pair := splitEnvPair(env)
			os.Setenv(pair[0], pair[1])
		}
	}()

	tests := []struct {
		name     string
		envVars  map[string]string
		initial  TailscaleConfig
		expected TailscaleConfig
	}{
		{
			name: "env overrides all settings",
			envVars: map[string]string{
				"NETRONOME__TAILSCALE_ENABLED":             "true",
				"NETRONOME__TAILSCALE_METHOD":              "tsnet",
				"NETRONOME__TAILSCALE_AUTH_KEY":            "tskey-env-test",
				"NETRONOME__TAILSCALE_HOSTNAME":            "env-hostname",
				"NETRONOME__TAILSCALE_EPHEMERAL":           "true",
				"NETRONOME__TAILSCALE_STATE_DIR":           "/custom/state",
				"NETRONOME__TAILSCALE_CONTROL_URL":         "https://headscale.example.com",
				"NETRONOME__TAILSCALE_AGENT_PORT":          "8300",
				"NETRONOME__TAILSCALE_AUTO_DISCOVER":       "false",
				"NETRONOME__TAILSCALE_DISCOVERY_INTERVAL":  "10m",
				"NETRONOME__TAILSCALE_DISCOVERY_PORT":      "8400",
				"NETRONOME__TAILSCALE_DISCOVERY_PREFIX":    "prod-",
			},
			initial: TailscaleConfig{
				Enabled:  false,
				Method:   "host",
				AuthKey:  "original-key",
			},
			expected: TailscaleConfig{
				Enabled:           true,
				Method:            "tsnet",
				AuthKey:           "tskey-env-test",
				Hostname:          "env-hostname",
				Ephemeral:         true,
				StateDir:          "/custom/state",
				ControlURL:        "https://headscale.example.com",
				AgentPort:         8300,
				AutoDiscover:      false,
				DiscoveryInterval: "10m",
				DiscoveryPort:     8400,
				DiscoveryPrefix:   "prod-",
			},
		},
		{
			name: "partial env overrides",
			envVars: map[string]string{
				"NETRONOME__TAILSCALE_METHOD":   "host",
				"NETRONOME__TAILSCALE_AUTH_KEY": "", // Should clear auth key
			},
			initial: TailscaleConfig{
				Enabled:      true,
				Method:       "tsnet",
				AuthKey:      "original-key",
				AgentPort:    8200,
				AutoDiscover: true,
			},
			expected: TailscaleConfig{
				Enabled:      true,
				Method:       "host",
				AuthKey:      "",
				AgentPort:    8200,
				AutoDiscover: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear env
			os.Clearenv()
			
			// Set test env vars
			for k, v := range tt.envVars {
				os.Setenv(k, v)
			}
			
			// Apply env overrides
			cfg := tt.initial
			cfg.loadFromEnv()
			
			// Compare
			assert.Equal(t, tt.expected.Enabled, cfg.Enabled, "Enabled field mismatch")
			assert.Equal(t, tt.expected.Method, cfg.Method, "Method field mismatch")
			assert.Equal(t, tt.expected.AuthKey, cfg.AuthKey, "AuthKey field mismatch")
			assert.Equal(t, tt.expected.Hostname, cfg.Hostname, "Hostname field mismatch")
			assert.Equal(t, tt.expected.Ephemeral, cfg.Ephemeral, "Ephemeral field mismatch")
			assert.Equal(t, tt.expected.StateDir, cfg.StateDir, "StateDir field mismatch")
			assert.Equal(t, tt.expected.ControlURL, cfg.ControlURL, "ControlURL field mismatch")
			assert.Equal(t, tt.expected.AgentPort, cfg.AgentPort, "AgentPort field mismatch")
			assert.Equal(t, tt.expected.AutoDiscover, cfg.AutoDiscover, "AutoDiscover field mismatch")
			assert.Equal(t, tt.expected.DiscoveryInterval, cfg.DiscoveryInterval, "DiscoveryInterval field mismatch")
			assert.Equal(t, tt.expected.DiscoveryPort, cfg.DiscoveryPort, "DiscoveryPort field mismatch")
			assert.Equal(t, tt.expected.DiscoveryPrefix, cfg.DiscoveryPrefix, "DiscoveryPrefix field mismatch")
		})
	}
}

func TestTailscaleConfig_BackwardCompatibility(t *testing.T) {
	tests := []struct {
		name     string
		oldStyle Config
		expected TailscaleConfig
	}{
		{
			name: "old style with prefer_host true",
			oldStyle: Config{
				Tailscale: TailscaleConfig{
					Enabled:    true,
					AuthKey:    "",
					PreferHost: true,
					Agent: TailscaleAgentConfig{
						Enabled: true,
						Port:    8200,
					},
					Monitor: TailscaleMonitorConfig{
						AutoDiscover:      true,
						DiscoveryInterval: "5m",
						DiscoveryPort:     8200,
					},
				},
			},
			expected: TailscaleConfig{
				Enabled:           true,
				Method:            "host",
				AuthKey:           "",
				AgentPort:         8200,
				AutoDiscover:      true,
				DiscoveryInterval: "5m",
				DiscoveryPort:     8200,
			},
		},
		{
			name: "old style with auth key",
			oldStyle: Config{
				Tailscale: TailscaleConfig{
					Enabled:    true,
					AuthKey:    "tskey-auth-old",
					Hostname:   "old-node",
					PreferHost: false,
					Agent: TailscaleAgentConfig{
						Enabled: true,
						Port:    8300,
					},
					Monitor: TailscaleMonitorConfig{
						AutoDiscover:      false,
						DiscoveryPrefix:   "netronome-agent-",
					},
				},
			},
			expected: TailscaleConfig{
				Enabled:         true,
				Method:          "tsnet",
				AuthKey:         "tskey-auth-old",
				Hostname:        "old-node",
				AgentPort:       8300,
				AutoDiscover:    false,
				DiscoveryPrefix: "netronome-agent-",
			},
		},
		{
			name: "old style monitor only config",
			oldStyle: Config{
				Tailscale: TailscaleConfig{
					Enabled: false,
					Monitor: TailscaleMonitorConfig{
						AutoDiscover:      true,
						DiscoveryInterval: "2m",
						DiscoveryPort:     8200,
					},
				},
			},
			expected: TailscaleConfig{
				Enabled:           false,
				Method:            "host",
				AutoDiscover:      true,
				DiscoveryInterval: "2m",
				DiscoveryPort:     8200,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			migrated := tt.oldStyle.Tailscale.MigrateFromOldFormat()
			
			assert.Equal(t, tt.expected.Enabled, migrated.Enabled, "Enabled field mismatch")
			assert.Equal(t, tt.expected.Method, migrated.Method, "Method field mismatch")
			assert.Equal(t, tt.expected.AuthKey, migrated.AuthKey, "AuthKey field mismatch")
			assert.Equal(t, tt.expected.Hostname, migrated.Hostname, "Hostname field mismatch")
			assert.Equal(t, tt.expected.AgentPort, migrated.AgentPort, "AgentPort field mismatch")
			assert.Equal(t, tt.expected.AutoDiscover, migrated.AutoDiscover, "AutoDiscover field mismatch")
			assert.Equal(t, tt.expected.DiscoveryInterval, migrated.DiscoveryInterval, "DiscoveryInterval field mismatch")
			assert.Equal(t, tt.expected.DiscoveryPort, migrated.DiscoveryPort, "DiscoveryPort field mismatch")
			assert.Equal(t, tt.expected.DiscoveryPrefix, migrated.DiscoveryPrefix, "DiscoveryPrefix field mismatch")
		})
	}
}

func TestTailscaleConfig_IsAgentMode(t *testing.T) {
	tests := []struct {
		name     string
		config   TailscaleConfig
		expected bool
	}{
		{
			name: "enabled with valid method is agent mode",
			config: TailscaleConfig{
				Enabled: true,
				Method:  "host",
			},
			expected: true,
		},
		{
			name: "disabled is not agent mode",
			config: TailscaleConfig{
				Enabled: false,
				Method:  "host",
			},
			expected: false,
		},
		{
			name: "enabled with auto and no auth key is agent mode",
			config: TailscaleConfig{
				Enabled: true,
				Method:  "auto",
				AuthKey: "",
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.IsAgentMode()
			assert.Equal(t, tt.expected, result, "IsAgentMode() result mismatch")
		})
	}
}

func TestTailscaleConfig_IsServerDiscoveryMode(t *testing.T) {
	tests := []struct {
		name     string
		config   TailscaleConfig
		expected bool
	}{
		{
			name: "auto discover enabled is discovery mode",
			config: TailscaleConfig{
				Enabled:      true,
				AutoDiscover: true,
			},
			expected: true,
		},
		{
			name: "auto discover disabled is not discovery mode",
			config: TailscaleConfig{
				Enabled:      true,
				AutoDiscover: false,
			},
			expected: false,
		},
		{
			name: "disabled with auto discover is not discovery mode",
			config: TailscaleConfig{
				Enabled:      false,
				AutoDiscover: true,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.IsServerDiscoveryMode()
			assert.Equal(t, tt.expected, result, "IsServerDiscoveryMode() result mismatch")
		})
	}
}

// Helper functions
func splitEnvPair(env string) []string {
	for i := 0; i < len(env); i++ {
		if env[i] == '=' {
			return []string{env[:i], env[i+1:]}
		}
	}
	return []string{env, ""}
}