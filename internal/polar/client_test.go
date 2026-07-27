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
