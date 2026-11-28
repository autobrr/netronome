// Copyright (c) 2024-2026, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package server

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseServerListOptions(t *testing.T) {
	tests := []struct {
		name         string
		global       string
		latitude     string
		longitude    string
		refresh      string
		wantGlobal   bool
		wantLocation bool
		wantRefresh  bool
		wantError    bool
	}{
		{name: "local"},
		{name: "global", global: "true", wantGlobal: true},
		{name: "coordinates", latitude: "-27.4698", longitude: "153.0251", wantLocation: true},
		{name: "refresh", refresh: "true", wantRefresh: true},
		{name: "missing longitude", latitude: "1", wantError: true},
		{name: "invalid global", global: "sometimes", wantError: true},
		{name: "invalid refresh", refresh: "sometimes", wantError: true},
		{name: "global coordinates", global: "true", latitude: "1", longitude: "2", wantError: true},
		{name: "latitude out of range", latitude: "91", longitude: "2", wantError: true},
		{name: "longitude out of range", latitude: "1", longitude: "181", wantError: true},
		{name: "non finite latitude", latitude: "NaN", longitude: "2", wantError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			options, err := parseServerListOptions(tt.global, tt.latitude, tt.longitude, tt.refresh)
			if tt.wantError {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantGlobal, options.Global)
			assert.Equal(t, tt.wantLocation, options.Location != nil)
			assert.Equal(t, tt.wantRefresh, options.Refresh)
		})
	}
}
