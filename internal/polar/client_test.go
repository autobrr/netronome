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
// endpoint answers 404 too. Only a body that came from Polar may map to a
// denial sentinel; anything else must stay transient, or an endpoint change on
// Polar's side would revoke every install's license fleet-wide.
//
// The discriminator is Polar's typed envelope, not the wording of the detail:
// Polar answers a plain "Not found" for a key it cannot find and for an
// activation the customer released from their portal, which reads identically
// to FastAPI's generic 404 unless the "error" field is taken into account.
func TestNotFoundIsOnlyADenialForPolarShapedBodies(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		wantErr error // nil means transient: no denial sentinel allowed
	}{
		{"unknown key", `{"error":"ResourceNotFound","detail":"Not found"}`, ErrInvalidLicenseKey},
		{"released activation", `{"error":"ResourceNotFound","detail":"Not found"}`, ErrInvalidLicenseKey},
		{"revoked", `{"error":"ResourceNotFound","detail":"License key is no longer active."}`, ErrInvalidLicenseKey},
		{"expired", `{"error":"ResourceNotFound","detail":"License key has expired."}`, ErrLicenseExpired},
		{"condition mismatch", `{"error":"ResourceNotFound","detail":"License key does not match required conditions"}`, ErrConditionMismatch},
		{"envelope-less but names the key", `{"detail":"License key does not exist."}`, ErrInvalidLicenseKey},
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

// A 422 is Polar rejecting our payload - in practice a misconfigured
// organization id. Its detail is an array of field errors, not a string, so it
// must not be decoded as one, and it must not read as a bad license key.
func TestUnprocessableEntityIsAConfigError(t *testing.T) {
	body := `{"error":"RequestValidationError","detail":[{"type":"uuid_parsing","loc":["body","organization_id"]}]}`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()

	c := NewClient(WithOrganizationID("org_1"), WithHTTPClient(srv.Client()))
	c.baseURL = srv.URL

	_, err := c.Validate(context.Background(), ValidateRequest{Key: "key"})
	assert.ErrorIs(t, err, ErrBadRequestData)
	assert.NotErrorIs(t, err, ErrInvalidLicenseKey)

	_, err = c.Activate(context.Background(), ActivateRequest{Key: "key", Label: "test"})
	assert.ErrorIs(t, err, ErrBadRequestData)
	assert.NotErrorIs(t, err, ErrInvalidLicenseKey)
}
