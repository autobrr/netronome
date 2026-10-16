// Copyright (c) 2024-2026, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package notifications

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateNotificationURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		rawURL  string
		wantErr string
	}{
		{
			name:    "pushover shorthand missing password token",
			rawURL:  "pushover://API_TOKEN@USER_KEY",
			wantErr: "token missing",
		},
		{
			name:   "pushover proper shoutrrr format",
			rawURL: "pushover://shoutrrr:API_TOKEN@USER_KEY",
		},
		{
			name:    "ntfy missing topic",
			rawURL:  "ntfy://example.com",
			wantErr: "must include a topic",
		},
		{
			name:    "ntfy missing host",
			rawURL:  "ntfy:///alerts",
			wantErr: "must include a host",
		},
		{
			name:   "ntfy scheme override with shoutrrr casing",
			rawURL: "ntfy://:tk_token@ntfy/htpc?Scheme=http",
		},
		{
			name:    "trims whitespace before validating",
			rawURL:  "  pushover://API_TOKEN@USER_KEY  ",
			wantErr: "token missing",
		},
		{
			// This URL caused a panic in shoutrrr v0.8.0 (slice bounds). The
			// panic stopped the whole validation request.
			name:    "opsgenie missing host does not panic",
			rawURL:  "opsgenie://APIKEY",
			wantErr: "API key missing",
		},
		{
			name:    "unknown service scheme",
			rawURL:  "carrierpigeon://coop.example.com/loft",
			wantErr: "unknown service",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := ValidateNotificationURL(tt.rawURL)
			if tt.wantErr == "" {
				require.NoError(t, err)
				return
			}

			require.Error(t, err)
			require.Contains(t, err.Error(), tt.wantErr)
		})
	}
}
