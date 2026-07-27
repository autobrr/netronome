// Copyright (c) 2024-2026, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package handlers

import (
	"context"
	"errors"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"

	"github.com/autobrr/netronome/internal/database"
	"github.com/autobrr/netronome/internal/services/license"
)

const (
	themeSettingKey       = "theme"
	publicThemeSettingKey = "public_theme"

	// defaultTheme is the only free theme. The theme catalogue lives in the
	// frontend bundle, so the server cannot know which ids exist: it treats
	// ANY id other than the default as premium-gated.
	defaultTheme = "netronome"
)

// themeIDPattern whitelists theme ids so junk never reaches app_settings.
var themeIDPattern = regexp.MustCompile(`^[a-z0-9-]{1,40}$`)

// LicenseService is the minimal surface these handlers need from
// internal/services/license. Get returns (nil, nil) when unlicensed.
type LicenseService interface {
	Activate(ctx context.Context, licenseKey string) (*license.License, error)
	Deactivate(ctx context.Context) error
	HasPremiumAccess(ctx context.Context) bool
	Get(ctx context.Context) (*license.License, error)
}

// SettingsStore is the app_settings key/value surface, satisfied by
// database.Service.
type SettingsStore interface {
	GetAppSetting(ctx context.Context, key string) (string, error)
	SetAppSetting(ctx context.Context, key, value string) error
}

// LicenseHandler handles license and theme endpoints
type LicenseHandler struct {
	db      SettingsStore
	service LicenseService
}

// NewLicenseHandler creates a new license handler
func NewLicenseHandler(db SettingsStore, service LicenseService) *LicenseHandler {
	return &LicenseHandler{
		db:      db,
		service: service,
	}
}

type activateLicenseRequest struct {
	LicenseKey string `json:"licenseKey"`
}

type licenseDetailsResponse struct {
	LicenseKey  string  `json:"licenseKey"`
	Status      string  `json:"status"`
	ProductName string  `json:"productName"`
	ActivatedAt string  `json:"activatedAt"`
	ExpiresAt   *string `json:"expiresAt"`
}

type licenseResponse struct {
	HasPremiumAccess bool                    `json:"hasPremiumAccess"`
	License          *licenseDetailsResponse `json:"license"`
}

type themeSettingsResponse struct {
	Theme       string `json:"theme"`
	PublicTheme string `json:"publicTheme"`
}

type updateThemeSettingsRequest struct {
	Theme       string `json:"theme"`
	PublicTheme string `json:"publicTheme"`
}

type publicThemeResponse struct {
	Theme string `json:"theme"`
}

// GetLicense returns the current license state
func (h *LicenseHandler) GetLicense(c *gin.Context) {
	ctx := c.Request.Context()

	lic, err := h.service.Get(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get license")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get license"})
		return
	}

	c.JSON(http.StatusOK, h.buildLicenseResponse(ctx, lic))
}

// ActivateLicense activates a license key
func (h *LicenseHandler) ActivateLicense(c *gin.Context) {
	var req activateLicenseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	req.LicenseKey = strings.TrimSpace(req.LicenseKey)
	if req.LicenseKey == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "License key is required"})
		return
	}

	ctx := c.Request.Context()

	// Never log the raw key.
	lic, err := h.service.Activate(ctx, req.LicenseKey)
	if err != nil {
		log.Error().Err(err).Msg("Failed to activate license")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, h.buildLicenseResponse(ctx, lic))
}

// DeactivateLicense removes the stored license
func (h *LicenseHandler) DeactivateLicense(c *gin.Context) {
	if err := h.service.Deactivate(c.Request.Context()); err != nil {
		log.Error().Err(err).Msg("Failed to deactivate license")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to deactivate license"})
		return
	}

	c.JSON(http.StatusOK, licenseResponse{HasPremiumAccess: false, License: nil})
}

// GetThemeSettings returns the configured themes, resolved through entitlement
func (h *LicenseHandler) GetThemeSettings(c *gin.Context) {
	ctx := c.Request.Context()
	hasPremiumAccess := h.service.HasPremiumAccess(ctx)

	c.JSON(http.StatusOK, themeSettingsResponse{
		Theme:       gateTheme(h.readTheme(ctx, themeSettingKey), hasPremiumAccess),
		PublicTheme: gateTheme(h.readTheme(ctx, publicThemeSettingKey), hasPremiumAccess),
	})
}

// UpdateThemeSettings stores the themes, rejecting premium ids without entitlement
func (h *LicenseHandler) UpdateThemeSettings(c *gin.Context) {
	var req updateThemeSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	req.Theme = strings.TrimSpace(req.Theme)
	req.PublicTheme = strings.TrimSpace(req.PublicTheme)

	if !themeIDPattern.MatchString(req.Theme) || !themeIDPattern.MatchString(req.PublicTheme) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid theme id"})
		return
	}

	ctx := c.Request.Context()

	// Server is authoritative: anything but the default theme needs entitlement.
	if req.Theme != defaultTheme || req.PublicTheme != defaultTheme {
		if !h.service.HasPremiumAccess(ctx) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Premium access is required for this theme"})
			return
		}
	}

	if err := h.db.SetAppSetting(ctx, themeSettingKey, req.Theme); err != nil {
		log.Error().Err(err).Msg("Failed to update theme setting")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update theme settings"})
		return
	}

	if err := h.db.SetAppSetting(ctx, publicThemeSettingKey, req.PublicTheme); err != nil {
		log.Error().Err(err).Msg("Failed to update public theme setting")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update theme settings"})
		return
	}

	c.JSON(http.StatusOK, themeSettingsResponse{Theme: req.Theme, PublicTheme: req.PublicTheme})
}

// GetPublicTheme returns the theme for unauthenticated pages. It resolves
// through entitlement so an unlicensed instance can never leak a premium theme.
func (h *LicenseHandler) GetPublicTheme(c *gin.Context) {
	ctx := c.Request.Context()

	c.JSON(http.StatusOK, publicThemeResponse{
		Theme: gateTheme(h.readTheme(ctx, publicThemeSettingKey), h.service.HasPremiumAccess(ctx)),
	})
}

func (h *LicenseHandler) buildLicenseResponse(ctx context.Context, lic *license.License) licenseResponse {
	resp := licenseResponse{HasPremiumAccess: h.service.HasPremiumAccess(ctx)}
	if lic == nil {
		return resp
	}

	resp.License = &licenseDetailsResponse{
		LicenseKey:  maskLicenseKey(lic.LicenseKey),
		Status:      lic.Status,
		ProductName: lic.ProductName,
		ActivatedAt: lic.ActivatedAt.Format(time.RFC3339),
	}

	if lic.ExpiresAt != nil {
		expiresAt := lic.ExpiresAt.Format(time.RFC3339)
		resp.License.ExpiresAt = &expiresAt
	}

	return resp
}

// readTheme returns a stored theme id, falling back to the default when it is
// missing or was persisted as junk.
func (h *LicenseHandler) readTheme(ctx context.Context, key string) string {
	value, err := h.db.GetAppSetting(ctx, key)
	if err != nil {
		if !errors.Is(err, database.ErrNotFound) {
			log.Error().Err(err).Str("key", key).Msg("Failed to get theme setting, using default")
		}
		return defaultTheme
	}

	if !themeIDPattern.MatchString(value) {
		log.Warn().Str("key", key).Str("value", value).Msg("Invalid persisted theme id, using default")
		return defaultTheme
	}

	return value
}

// gateTheme downgrades a premium theme id to the default when entitlement is
// not currently active.
func gateTheme(theme string, hasPremiumAccess bool) string {
	if theme != defaultTheme && !hasPremiumAccess {
		return defaultTheme
	}
	return theme
}

// maskLicenseKey returns the first 8 characters plus "***". Short keys are
// masked entirely so nothing usable is ever returned.
func maskLicenseKey(key string) string {
	if len(key) <= 8 {
		return "***"
	}
	return key[:8] + "***"
}
