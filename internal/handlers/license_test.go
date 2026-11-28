// Copyright (c) 2024-2026, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package handlers

import (
	"context"
	"encoding/json/v2"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/autobrr/netronome/internal/services/license"
)

type licenseStubService struct {
	LicenseService
	license *license.License
}

func (s licenseStubService) Get(context.Context) (*license.License, error) {
	return s.license, nil
}

func (s licenseStubService) HasPremiumAccess(context.Context) bool {
	return true
}

func TestGetLicenseAnswersInUTC(t *testing.T) {
	useLocalZone(t)
	gin.SetMode(gin.TestMode)
	handler := NewLicenseHandler(nil, licenseStubService{license: &license.License{
		LicenseKey: "placeholder-key", Status: license.StatusActive, ActivatedAt: time.Now(), ExpiresAt: new(time.Now().Add(24 * time.Hour)),
	}})
	router := gin.New()
	router.GET("/api/license", handler.GetLicense)

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/license", nil))

	require.Equal(t, http.StatusOK, response.Code)
	var body struct {
		License struct {
			ActivatedAt string `json:"activatedAt"`
			ExpiresAt   string `json:"expiresAt"`
		} `json:"license"`
	}
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body))
	assert.True(t, strings.HasSuffix(body.License.ActivatedAt, "Z"), "activatedAt %q", body.License.ActivatedAt)
	assert.True(t, strings.HasSuffix(body.License.ExpiresAt, "Z"), "expiresAt %q", body.License.ExpiresAt)
}
