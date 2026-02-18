// Copyright (c) 2024-2026, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package database

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAppSettings_GetMissingReturnsNotFound(t *testing.T) {
	RunTestWithBothDatabases(t, func(t *testing.T, td *TestDatabase) {
		ctx := context.Background()

		_, err := td.Service.GetAppSetting(ctx, "dashboard_recent_speedtests_rows")
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrNotFound)
	})
}

func TestAppSettings_SetAndGet(t *testing.T) {
	RunTestWithBothDatabases(t, func(t *testing.T, td *TestDatabase) {
		ctx := context.Background()

		err := td.Service.SetAppSetting(ctx, "dashboard_recent_speedtests_rows", "20")
		require.NoError(t, err)

		value, err := td.Service.GetAppSetting(ctx, "dashboard_recent_speedtests_rows")
		require.NoError(t, err)
		assert.Equal(t, "20", value)
	})
}

func TestAppSettings_UpdateExisting(t *testing.T) {
	RunTestWithBothDatabases(t, func(t *testing.T, td *TestDatabase) {
		ctx := context.Background()

		err := td.Service.SetAppSetting(ctx, "dashboard_recent_speedtests_rows", "20")
		require.NoError(t, err)

		err = td.Service.SetAppSetting(ctx, "dashboard_recent_speedtests_rows", "100")
		require.NoError(t, err)

		value, err := td.Service.GetAppSetting(ctx, "dashboard_recent_speedtests_rows")
		require.NoError(t, err)
		assert.Equal(t, "100", value)
	})
}
