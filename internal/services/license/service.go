// Copyright (c) 2024-2026, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package license

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"

	"github.com/autobrr/netronome/internal/database"
	"github.com/autobrr/netronome/internal/polar"
)

const (
	// SettingKey is the app_settings key holding the whole license blob.
	SettingKey = "license"

	// ProductNamePremium is the only product we sell: one-time premium access.
	ProductNamePremium = "premium-access"

	StatusActive  = "active"
	StatusInvalid = "invalid"

	activationLabel = "Netronome Premium License"

	// offlineGracePeriod is how long entitlement survives without a successful
	// revalidation against Polar.
	offlineGracePeriod = 7 * 24 * time.Hour
)

var (
	ErrNotConfigured   = errors.New("polar client not configured")
	ErrKeyRequired     = errors.New("license key is required")
	ErrActivationLimit = errors.New("license key activation limit already reached")
)

// SettingsStore is the key/value slice of the database service we need.
// internal/database.Service satisfies it.
type SettingsStore interface {
	GetAppSetting(ctx context.Context, key string) (string, error)
	SetAppSetting(ctx context.Context, key, value string) error
}

// PolarClient is the subset of *polar.Client the service calls, so tests can
// swap in a fake.
type PolarClient interface {
	Activate(ctx context.Context, req polar.ActivateRequest) (*polar.ActivateKeyResponse, error)
	Validate(ctx context.Context, req polar.ValidateRequest) (*polar.ValidateResp, error)
	Deactivate(ctx context.Context, req polar.DeactivateRequest) error
	IsClientConfigured() bool
}

// License is the persisted blob. One license per install, stored as JSON under
// the "license" app setting.
type License struct {
	LicenseKey        string     `json:"licenseKey"`
	Status            string     `json:"status"`
	ProductName       string     `json:"productName"`
	ActivatedAt       time.Time  `json:"activatedAt"`
	ExpiresAt         *time.Time `json:"expiresAt"`
	PolarActivationID string     `json:"polarActivationId"`
	PolarCustomerID   string     `json:"polarCustomerId"`
	PolarProductID    string     `json:"polarProductId"`
	LastValidated     time.Time  `json:"lastValidated"`
}

// Entitled reports whether the license grants premium access at t. A license we
// have not been able to revalidate for longer than the grace period lapses,
// which is what makes the check server-authoritative rather than a local flag.
func (l *License) Entitled(t time.Time) bool {
	if l == nil || l.Status != StatusActive {
		return false
	}
	if l.ExpiresAt != nil && t.After(*l.ExpiresAt) {
		return false
	}
	// A zero LastValidated never happens for a legitimately activated license
	// (Activate stamps it), so treat it as lapsed rather than valid forever.
	if l.LastValidated.IsZero() || t.Sub(l.LastValidated) > offlineGracePeriod {
		return false
	}
	return true
}

type Service struct {
	store     SettingsStore
	polar     PolarClient
	configDir string

	// mu serialises read-modify-write of the single blob.
	mu sync.Mutex
}

func NewService(store SettingsStore, polarClient PolarClient, configDir string) *Service {
	return &Service{
		store:     store,
		polar:     polarClient,
		configDir: configDir,
	}
}

// Get returns the stored license, or (nil, nil) when none is stored.
func (s *Service) Get(ctx context.Context) (*License, error) {
	raw, err := s.store.GetAppSetting(ctx, SettingKey)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			return nil, nil
		}
		return nil, errors.Wrap(err, "failed to read license")
	}

	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}

	var lic License
	if err := json.Unmarshal([]byte(raw), &lic); err != nil {
		return nil, errors.Wrap(err, "failed to decode stored license")
	}

	return &lic, nil
}

// HasPremiumAccess reports whether premium features are unlocked. A read error
// means we cannot prove entitlement, so we fall back to the free tier.
func (s *Service) HasPremiumAccess(ctx context.Context) bool {
	lic, err := s.Get(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to load license, denying premium access")
		return false
	}

	return lic.Entitled(time.Now())
}

// Activate activates licenseKey against Polar and stores the result.
func (s *Service) Activate(ctx context.Context, licenseKey string) (*License, error) {
	licenseKey = strings.TrimSpace(licenseKey)
	if licenseKey == "" {
		return nil, ErrKeyRequired
	}

	if s.polar == nil || !s.polar.IsClientConfigured() {
		return nil, ErrNotConfigured
	}

	fingerprint, err := GetDeviceID(s.configDir)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get device id")
	}

	req := polar.ActivateRequest{Key: licenseKey, Label: activationLabel}
	req.SetCondition("fingerprint", fingerprint)
	req.SetMeta("product", activationLabel)

	resp, err := s.polar.Activate(ctx, req)
	switch {
	case errors.Is(err, polar.ErrActivationLimitExceeded):
		return nil, ErrActivationLimit
	case err != nil:
		return nil, errors.Wrap(err, "failed to activate license")
	}

	now := time.Now()
	lic := &License{
		LicenseKey:        licenseKey,
		Status:            StatusActive,
		ProductName:       ProductNamePremium,
		ActivatedAt:       now,
		ExpiresAt:         resp.LicenseKey.ExpiresAt,
		PolarActivationID: resp.ID,
		PolarCustomerID:   resp.LicenseKey.CustomerID,
		PolarProductID:    resp.LicenseKey.BenefitID,
		LastValidated:     now,
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Replacing a key must release the previous Polar activation, or its seat
	// leaks forever. Done after the new activation succeeded so a failed swap
	// never drops a working license; best-effort, like Deactivate.
	if prev, err := s.Get(ctx); err == nil && prev != nil &&
		prev.PolarActivationID != "" && prev.PolarActivationID != lic.PolarActivationID {
		derr := s.polar.Deactivate(ctx, polar.DeactivateRequest{
			Key:          prev.LicenseKey,
			ActivationID: prev.PolarActivationID,
		})
		if derr != nil && !errors.Is(derr, polar.ErrLicenseNotActivated) && !errors.Is(derr, polar.ErrInvalidLicenseKey) {
			log.Warn().Err(derr).Str("licenseKey", MaskLicenseKey(prev.LicenseKey)).
				Msg("failed to release previous activation")
		}
	}

	if err := s.save(ctx, lic); err != nil {
		return nil, err
	}

	log.Info().Str("licenseKey", MaskLicenseKey(licenseKey)).Msg("license activated")

	return lic, nil
}

// Validate re-checks the stored license against Polar.
//
// Transient failures (network, rate limit, 5xx, missing config) leave the
// stored license untouched and are returned as an error: entitlement then rides
// the offline grace period. Only an authoritative negative from Polar flips the
// license to invalid.
func (s *Service) Validate(ctx context.Context) (*License, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	lic, err := s.Get(ctx)
	if err != nil || lic == nil {
		return nil, err
	}

	if s.polar == nil || !s.polar.IsClientConfigured() {
		return lic, ErrNotConfigured
	}

	fingerprint, err := GetDeviceID(s.configDir)
	if err != nil {
		return lic, errors.Wrap(err, "failed to get device id")
	}

	req := polar.ValidateRequest{Key: lic.LicenseKey, ActivationID: lic.PolarActivationID}
	req.SetCondition("fingerprint", fingerprint)

	resp, err := s.polar.Validate(ctx, req)
	if err != nil {
		if !isDenial(err) {
			log.Warn().Err(err).Str("licenseKey", MaskLicenseKey(lic.LicenseKey)).
				Msg("license validation failed, keeping current entitlement")
			return lic, err
		}

		log.Error().Err(err).Str("licenseKey", MaskLicenseKey(lic.LicenseKey)).
			Msg("license rejected by polar")
		return s.markInvalid(ctx, lic)
	}

	if !resp.ValidLicense() {
		log.Error().Str("status", resp.Status).Str("licenseKey", MaskLicenseKey(lic.LicenseKey)).
			Msg("license is not granted")
		return s.markInvalid(ctx, lic)
	}

	lic.Status = StatusActive
	lic.LastValidated = time.Now()
	if !resp.ExpiresAt.IsZero() {
		expiresAt := resp.ExpiresAt
		lic.ExpiresAt = &expiresAt
	}

	if err := s.save(ctx, lic); err != nil {
		return lic, err
	}

	return lic, nil
}

// Deactivate releases the activation with Polar and clears the local license.
// The local license is cleared even if the remote call fails, otherwise a
// revoked or offline install could never be reset.
func (s *Service) Deactivate(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	lic, err := s.Get(ctx)
	if err != nil {
		return err
	}
	if lic == nil {
		return nil
	}

	if lic.PolarActivationID != "" && s.polar != nil && s.polar.IsClientConfigured() {
		err := s.polar.Deactivate(ctx, polar.DeactivateRequest{
			Key:          lic.LicenseKey,
			ActivationID: lic.PolarActivationID,
		})
		if err != nil && !errors.Is(err, polar.ErrLicenseNotActivated) && !errors.Is(err, polar.ErrInvalidLicenseKey) {
			log.Warn().Err(err).Str("licenseKey", MaskLicenseKey(lic.LicenseKey)).
				Msg("failed to deactivate license remotely, clearing it locally anyway")
		}
	}

	if err := s.store.SetAppSetting(ctx, SettingKey, ""); err != nil {
		return errors.Wrap(err, "failed to clear license")
	}

	log.Info().Str("licenseKey", MaskLicenseKey(lic.LicenseKey)).Msg("license deactivated")

	return nil
}

func (s *Service) save(ctx context.Context, lic *License) error {
	blob, err := json.Marshal(lic)
	if err != nil {
		return errors.Wrap(err, "failed to encode license")
	}

	if err := s.store.SetAppSetting(ctx, SettingKey, string(blob)); err != nil {
		return errors.Wrap(err, "failed to store license")
	}

	return nil
}

func (s *Service) markInvalid(ctx context.Context, lic *License) (*License, error) {
	lic.Status = StatusInvalid
	lic.LastValidated = time.Now()

	if err := s.save(ctx, lic); err != nil {
		return lic, err
	}

	return lic, nil
}

// isDenial reports whether err is Polar telling us the license is not valid, as
// opposed to us failing to ask.
func isDenial(err error) bool {
	switch {
	// ErrActivationLimitExceeded is deliberately absent: the key hitting its
	// limit does not prove OUR activation is invalid, so it rides the grace
	// period like any other transient failure.
	case errors.Is(err, polar.ErrInvalidLicenseKey),
		errors.Is(err, polar.ErrConditionMismatch),
		errors.Is(err, polar.ErrLicenseExpired),
		errors.Is(err, polar.ErrLicenseNotActivated):
		return true
	default:
		return false
	}
}

// MaskLicenseKey shows the first 8 chars only, for logs and API responses.
func MaskLicenseKey(key string) string {
	if len(key) <= 8 {
		return "***"
	}
	return key[:8] + "***"
}
