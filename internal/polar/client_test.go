// Copyright (c) 2024-2026, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package polar

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

// A 403 is an authoritative denial: it must map to a denial sentinel even when
// the body is empty or malformed, or Service.Validate would treat it as
// transient and let entitlement ride the offline grace period forever.
func TestForbiddenMapsToDenialSentinels(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		wantErr error
	}{
		{"activation limit", `{"detail":"License key activation limit already reached"}`, ErrActivationLimitExceeded},
		{"expired", `{"detail":"License key has expired"}`, ErrLicenseExpired},
		{"revoked", `{"detail":"License key is revoked"}`, ErrInvalidLicenseKey},
		{"disabled", `{"detail":"License key is disabled"}`, ErrInvalidLicenseKey},
		{"malformed body", `<html>forbidden</html>`, ErrInvalidLicenseKey},
		{"empty body", ``, ErrInvalidLicenseKey},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusForbidden)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer srv.Close()

			c := NewClient(WithOrganizationID("org_1"), WithHTTPClient(srv.Client()))
			c.baseURL = srv.URL

			_, err := c.Validate(context.Background(), ValidateRequest{Key: "key"})
			assert.ErrorIs(t, err, tt.wantErr, "Validate")

			_, err = c.Activate(context.Background(), ActivateRequest{Key: "key", Label: "test"})
			assert.ErrorIs(t, err, tt.wantErr, "Activate")
		})
	}
}

// A 404, unlike a 403, is not reserved for denials: a retired or moved
// endpoint answers 404 too. Only a body that talks about the license key may
// map to a denial sentinel; anything else must stay transient, or an endpoint
// change on Polar's side would revoke every install's license fleet-wide.
func TestNotFoundIsOnlyADenialForLicenseShapedBodies(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		wantErr error // nil means transient: no denial sentinel allowed
	}{
		{"condition mismatch", `{"detail":"License key does not match required conditions"}`, ErrConditionMismatch},
		{"unknown key", `{"detail":"License key does not exist."}`, ErrInvalidLicenseKey},
		{"generic route 404", `{"detail":"Not Found"}`, nil},
		{"malformed body", `<html>not found</html>`, nil},
		{"empty body", ``, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer srv.Close()

			c := NewClient(WithOrganizationID("org_1"), WithHTTPClient(srv.Client()))
			c.baseURL = srv.URL

			_, err := c.Validate(context.Background(), ValidateRequest{Key: "key"})
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}
			assert.Error(t, err)
			for _, denial := range []error{ErrInvalidLicenseKey, ErrConditionMismatch, ErrLicenseExpired, ErrLicenseNotActivated} {
				assert.NotErrorIs(t, err, denial)
			}
		})
	}
}
