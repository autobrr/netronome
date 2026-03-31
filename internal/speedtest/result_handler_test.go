// Copyright (c) 2024-2026, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package speedtest

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResultURLNormalization(t *testing.T) {
	tests := []struct {
		name      string
		resultURL string
		wantNil   bool
	}{
		{
			name:      "empty string produces nil",
			resultURL: "",
			wantNil:   true,
		},
		{
			name:      "non-empty string produces pointer",
			resultURL: "https://librespeed.org/results/?id=abc123",
			wantNil:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var resultURL *string
			if tt.resultURL != "" {
				resultURL = &tt.resultURL
			}

			if tt.wantNil {
				assert.Nil(t, resultURL)
			} else {
				assert.NotNil(t, resultURL)
				assert.Equal(t, tt.resultURL, *resultURL)
			}
		})
	}
}
