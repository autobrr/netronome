// Copyright (c) 2024-2026, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package speedtest

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/autobrr/netronome/internal/database"
	"github.com/autobrr/netronome/internal/types"
)

// captureService is a database.Service test double that records the
// SpeedTestResult passed to SaveSpeedTest. Embedding the interface gives nil
// implementations for every other method (none are exercised by SaveResult).
type captureService struct {
	database.Service
	captured types.SpeedTestResult
}

func (c *captureService) SaveSpeedTest(_ context.Context, result types.SpeedTestResult) (*types.SpeedTestResult, error) {
	c.captured = result
	return &result, nil
}

// TestSaveResultNormalizesResultURL exercises the production normalization in
// DefaultResultHandler.SaveResult: an empty ResultURL must persist as a nil
// *string, a non-empty one as a pointer to that value.
func TestSaveResultNormalizesResultURL(t *testing.T) {
	tests := []struct {
		name      string
		resultURL string
		wantNil   bool
	}{
		{
			name:      "empty string persists as nil",
			resultURL: "",
			wantNil:   true,
		},
		{
			name:      "non-empty string persists as pointer",
			resultURL: "https://librespeed.org/results/?id=abc123",
			wantNil:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &captureService{}
			h := NewResultHandler(svc, nil)

			err := h.SaveResult(context.Background(), &Result{
				Server:    "test-server",
				ResultURL: tt.resultURL,
			}, "librespeed", &types.TestOptions{})
			require.NoError(t, err)

			if tt.wantNil {
				assert.Nil(t, svc.captured.ResultURL)
			} else {
				require.NotNil(t, svc.captured.ResultURL)
				assert.Equal(t, tt.resultURL, *svc.captured.ResultURL)
			}
		})
	}
}
