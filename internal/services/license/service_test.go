// Copyright (c) 2024-2026, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package license

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/autobrr/netronome/internal/database"
	"github.com/autobrr/netronome/internal/polar"
)

type fakeStore struct {
	values map[string]string
}

func newFakeStore() *fakeStore { return &fakeStore{values: map[string]string{}} }

func (f *fakeStore) GetAppSetting(_ context.Context, key string) (string, error) {
	v, ok := f.values[key]
	if !ok {
		return "", database.ErrNotFound
	}
	return v, nil
}

func (f *fakeStore) SetAppSetting(_ context.Context, key, value string) error {
	f.values[key] = value
	return nil
}

type fakePolar struct {
	resp *polar.ValidateResp
	err  error

	activateResp *polar.ActivateKeyResponse
	activateErr  error
	deactivated  []polar.DeactivateRequest
}

func (f *fakePolar) Activate(context.Context, polar.ActivateRequest) (*polar.ActivateKeyResponse, error) {
	if f.activateErr != nil {
		return nil, f.activateErr
	}
	if f.activateResp == nil {
		return nil, errors.New("not used")
	}
	return f.activateResp, nil
}

func (f *fakePolar) Validate(context.Context, polar.ValidateRequest) (*polar.ValidateResp, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.resp, nil
}

func (f *fakePolar) Deactivate(_ context.Context, req polar.DeactivateRequest) error {
	f.deactivated = append(f.deactivated, req)
	return nil
}

func (f *fakePolar) IsClientConfigured() bool { return true }

func TestEntitled(t *testing.T) {
	now := time.Date(2026, 7, 27, 12, 0, 0, 0, time.UTC)
	past := now.Add(-time.Hour)
	future := now.Add(24 * time.Hour)

	tests := []struct {
		name    string
		license *License
		want    bool
	}{
		{"no license", nil, false},
		{"active, just validated", &License{Status: StatusActive, LastValidated: now}, true},
		{"invalid status", &License{Status: StatusInvalid, LastValidated: now}, false},
		{"inside grace period", &License{Status: StatusActive, LastValidated: now.Add(-offlineGracePeriod + time.Minute)}, true},
		{"exactly at grace boundary", &License{Status: StatusActive, LastValidated: now.Add(-offlineGracePeriod)}, true},
		{"past grace period", &License{Status: StatusActive, LastValidated: now.Add(-offlineGracePeriod - time.Minute)}, false},
		{"never validated", &License{Status: StatusActive}, false},
		{"expired", &License{Status: StatusActive, LastValidated: now, ExpiresAt: &past}, false},
		{"not yet expired", &License{Status: StatusActive, LastValidated: now, ExpiresAt: &future}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.license.Entitled(now))
		})
	}
}

func TestValidateKeepsEntitlementOnTransientErrors(t *testing.T) {
	tests := []struct {
		name        string
		polarErr    error
		polarResp   *polar.ValidateResp
		wantErr     bool
		wantStatus  string
		wantPremium bool
	}{
		{
			name:        "network error keeps entitlement",
			polarErr:    &net.OpError{Op: "dial", Err: errors.New("connection refused")},
			wantErr:     true,
			wantStatus:  StatusActive,
			wantPremium: true,
		},
		{
			name:        "rate limit keeps entitlement",
			polarErr:    polar.ErrRateLimitExceeded,
			wantErr:     true,
			wantStatus:  StatusActive,
			wantPremium: true,
		},
		{
			name:        "invalid key revokes entitlement",
			polarErr:    polar.ErrInvalidLicenseKey,
			wantStatus:  StatusInvalid,
			wantPremium: false,
		},
		{
			name:        "fingerprint mismatch revokes entitlement",
			polarErr:    polar.ErrConditionMismatch,
			wantStatus:  StatusInvalid,
			wantPremium: false,
		},
		{
			name:        "non-granted status revokes entitlement",
			polarResp:   &polar.ValidateResp{Status: "revoked"},
			wantStatus:  StatusInvalid,
			wantPremium: false,
		},
		{
			name:        "granted status refreshes validation",
			polarResp:   &polar.ValidateResp{Status: "granted"},
			wantStatus:  StatusActive,
			wantPremium: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			store := newFakeStore()

			// Stored a while ago but still inside the grace period.
			stored := License{
				LicenseKey:        "NETRONOME-1234-5678",
				Status:            StatusActive,
				ProductName:       ProductNamePremium,
				ActivatedAt:       time.Now().Add(-48 * time.Hour),
				PolarActivationID: "act_123",
				LastValidated:     time.Now().Add(-48 * time.Hour),
			}
			blob, err := json.Marshal(stored)
			require.NoError(t, err)
			store.values[SettingKey] = string(blob)

			svc := NewService(store, &fakePolar{resp: tt.polarResp, err: tt.polarErr}, t.TempDir())

			lic, err := svc.Validate(ctx)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			require.NotNil(t, lic)
			assert.Equal(t, tt.wantStatus, lic.Status)
			assert.Equal(t, tt.wantPremium, svc.HasPremiumAccess(ctx))

			// A transient failure must not rewrite the stored blob.
			if tt.wantErr {
				assert.Equal(t, string(blob), store.values[SettingKey])
			}
		})
	}
}

func TestActivate(t *testing.T) {
	ctx := context.Background()

	activateResp := &polar.ActivateKeyResponse{ID: "act_new"}
	activateResp.LicenseKey.CustomerID = "cus_1"
	activateResp.LicenseKey.BenefitID = "ben_1"

	t.Run("success persists license", func(t *testing.T) {
		store := newFakeStore()
		svc := NewService(store, &fakePolar{activateResp: activateResp}, t.TempDir())

		lic, err := svc.Activate(ctx, "NETRONOME-1234-5678")
		require.NoError(t, err)
		assert.Equal(t, "act_new", lic.PolarActivationID)
		assert.True(t, svc.HasPremiumAccess(ctx))

		var stored License
		require.NoError(t, json.Unmarshal([]byte(store.values[SettingKey]), &stored))
		assert.Equal(t, "act_new", stored.PolarActivationID)
		assert.Equal(t, "cus_1", stored.PolarCustomerID)
	})

	t.Run("activation limit maps to ErrActivationLimit", func(t *testing.T) {
		svc := NewService(newFakeStore(), &fakePolar{activateErr: polar.ErrActivationLimitExceeded}, t.TempDir())

		_, err := svc.Activate(ctx, "NETRONOME-1234-5678")
		assert.ErrorIs(t, err, ErrActivationLimit)
	})

	t.Run("replacing a license releases the old activation", func(t *testing.T) {
		store := newFakeStore()
		fp := &fakePolar{activateResp: activateResp}
		svc := NewService(store, fp, t.TempDir())

		blob, err := json.Marshal(License{LicenseKey: "OLD-KEY-1234-5678", Status: StatusActive, PolarActivationID: "act_old", LastValidated: time.Now()})
		require.NoError(t, err)
		store.values[SettingKey] = string(blob)

		_, err = svc.Activate(ctx, "NETRONOME-1234-5678")
		require.NoError(t, err)
		require.Len(t, fp.deactivated, 1)
		assert.Equal(t, "act_old", fp.deactivated[0].ActivationID)
	})
}

func TestGetNoLicenseIsNotAnError(t *testing.T) {
	svc := NewService(newFakeStore(), &fakePolar{}, t.TempDir())

	lic, err := svc.Get(context.Background())
	require.NoError(t, err)
	assert.Nil(t, lic)
	assert.False(t, svc.HasPremiumAccess(context.Background()))
}

func TestDeactivateClearsLicense(t *testing.T) {
	ctx := context.Background()
	store := newFakeStore()
	svc := NewService(store, &fakePolar{}, t.TempDir())

	blob, err := json.Marshal(License{LicenseKey: "NETRONOME-1234-5678", Status: StatusActive, PolarActivationID: "act_123", LastValidated: time.Now()})
	require.NoError(t, err)
	store.values[SettingKey] = string(blob)
	require.True(t, svc.HasPremiumAccess(ctx))

	require.NoError(t, svc.Deactivate(ctx))
	assert.False(t, svc.HasPremiumAccess(ctx))

	lic, err := svc.Get(ctx)
	require.NoError(t, err)
	assert.Nil(t, lic)
}
