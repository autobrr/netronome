// Copyright (c) 2024-2026, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package polar

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

var (
	ErrNoOrganizationID        = errors.New("organization ID not configured")
	ErrLicenseExpired          = errors.New("license expired")
	ErrLicenseNotActivated     = errors.New("license not activated")
	ErrInvalidLicenseKey       = errors.New("license key is not valid")
	ErrConditionMismatch       = errors.New("license key does not match required conditions")
	ErrActivationLimitExceeded = errors.New("license key activation limit already reached")
	ErrActivationNotSupported  = errors.New("license key does not support activations")
	ErrBadRequestData          = errors.New("bad request data")
	ErrCouldNotUnmarshalData   = errors.New("could not unmarshal data")
	ErrRateLimitExceeded       = errors.New("rate limit exceeded")
)

const (
	polarAPIBaseURL        = "https://api.polar.sh"
	polarSandboxAPIBaseURL = "https://sandbox-api.polar.sh"
	validateEndpoint       = "/v1/customer-portal/license-keys/validate"
	activateEndpoint       = "/v1/customer-portal/license-keys/activate"
	deactivateEndpoint     = "/v1/customer-portal/license-keys/deactivate"

	requestTimeout = 30 * time.Second

	// maxResponseBytes bounds how much of a response body we will read.
	maxResponseBytes = 1 << 20
)

type ValidateResp struct {
	ID               string    `json:"id"`
	OrganizationID   string    `json:"organization_id"`
	CustomerID       string    `json:"customer_id"`
	BenefitID        string    `json:"benefit_id"`
	Key              string    `json:"key"`
	DisplayKey       string    `json:"display_key"`
	Status           string    `json:"status"`
	LimitActivations int       `json:"limit_activations"`
	Usage            int       `json:"usage"`
	LimitUsage       int       `json:"limit_usage"`
	Validations      int       `json:"validations"`
	LastValidatedAt  time.Time `json:"last_validated_at"`
	ExpiresAt        time.Time `json:"expires_at"`
	Activation       struct {
		ID           string         `json:"id"`
		LicenseKeyID string         `json:"license_key_id"`
		Label        string         `json:"label"`
		Meta         map[string]any `json:"meta"`
		CreatedAt    time.Time      `json:"created_at"`
		ModifiedAt   any            `json:"modified_at"`
	} `json:"activation"`
}

func (v *ValidateResp) ValidLicense() bool {
	return v.Status == "granted"
}

type ActivateKeyResponse struct {
	ID           string         `json:"id"`
	LicenseKeyID string         `json:"license_key_id"`
	Label        string         `json:"label"`
	Meta         map[string]any `json:"meta"`
	CreatedAt    time.Time      `json:"created_at"`
	ModifiedAt   time.Time      `json:"modified_at"`
	LicenseKey   struct {
		ID               string     `json:"id"`
		OrganizationID   string     `json:"organization_id"`
		CustomerID       string     `json:"customer_id"`
		BenefitID        string     `json:"benefit_id"`
		Key              string     `json:"key"`
		DisplayKey       string     `json:"display_key"`
		Status           string     `json:"status"`
		LimitActivations int        `json:"limit_activations"`
		Usage            int        `json:"usage"`
		LimitUsage       int        `json:"limit_usage"`
		Validations      int        `json:"validations"`
		LastValidatedAt  *time.Time `json:"last_validated_at"`
		ExpiresAt        *time.Time `json:"expires_at"`
	} `json:"license_key"`
}

// ErrorResponse is Polar's typed error envelope: {"error":"ResourceNotFound",
// "detail":"..."}. The Error field is the useful half - a generic FastAPI or
// proxy error carries only a detail, so its presence is what tells us Polar
// itself answered.
//
// It deliberately does not cover 422, whose detail is an array of field errors
// rather than a string and therefore fails to decode here.
type ErrorResponse struct {
	Error  string `json:"error"`
	Detail string `json:"detail"`
}

// denialFromDetail maps a Polar error detail to a denial sentinel. The strings
// come from Polar's license_key service; an unrecognised or empty detail still
// resolves to a denial, since callers only reach here on a status code that
// already established one.
func denialFromDetail(detail string) error {
	lower := strings.ToLower(detail)

	switch {
	case detail == "License key activation limit already reached":
		return ErrActivationLimitExceeded
	case detail == "License key does not match required conditions":
		return ErrConditionMismatch
	case strings.Contains(lower, "does not support activations"):
		return errors.Wrap(ErrActivationNotSupported, detail)
	case strings.Contains(lower, "expired"):
		return errors.Wrap(ErrLicenseExpired, detail)
	case detail == "":
		return ErrInvalidLicenseKey
	default:
		return errors.Wrap(ErrInvalidLicenseKey, detail)
	}
}

// isPolarError reports whether body came from Polar itself, as opposed to a
// generic 404/403 emitted by a proxy or a retired route.
//
// Polar's typed envelope always carries an "error" field, which a bare FastAPI
// or proxy error does not - that is the primary signal. A detail that names
// the license key is accepted too, so the check survives Polar dropping the
// envelope without every install losing its license at once.
func isPolarError(body []byte) (ErrorResponse, bool) {
	var response ErrorResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return ErrorResponse{}, false
	}

	return response, response.Error != "" || strings.Contains(strings.ToLower(response.Detail), "license key")
}

// Client wraps the Polar API for license management
type Client struct {
	baseURL        string
	environment    string
	organizationID string
	userAgent      string

	httpClient *http.Client
}

type OptFunc func(*Client)

// WithOrganizationID sets the organization ID to use for requests.
func WithOrganizationID(organizationID string) OptFunc {
	return func(c *Client) {
		c.organizationID = organizationID
	}
}

// WithEnvironment sets the environment to use for requests.
// Valid values are "production", "sandbox" and "development".
func WithEnvironment(env string) OptFunc {
	return func(c *Client) {
		switch env {
		case "production":
			c.baseURL = polarAPIBaseURL
			c.environment = env
		case "sandbox":
			c.baseURL = polarSandboxAPIBaseURL
			c.environment = env
		case "development":
			c.baseURL = "http://localhost:8080"
			c.environment = env
		case "":
			// unset: keep the production default
		default:
			log.Warn().Str("environment", env).Msg("Unknown Polar environment, using production")
		}
	}
}

func WithUserAgent(userAgent string) OptFunc {
	return func(c *Client) {
		c.userAgent = userAgent
	}
}

// WithHTTPClient sets a custom HTTP client to use for requests
func WithHTTPClient(httpClient *http.Client) OptFunc {
	return func(c *Client) {
		c.httpClient = httpClient
	}
}

// NewClient creates a new Polar API client with the default HTTP client
func NewClient(opts ...OptFunc) *Client {
	c := &Client{
		baseURL:        polarAPIBaseURL,
		environment:    "production",
		organizationID: "",
		userAgent:      "polar-go",

		httpClient: &http.Client{
			Timeout: requestTimeout,
			Transport: &http.Transport{
				MaxIdleConns:        10,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     30 * time.Second,
			},
		},
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

type ActivateRequest struct {
	// License key
	Key string `json:"key"`

	// Set a label to associate with this specific activation.
	Label string `json:"label"`

	// Organization ID
	OrganizationID string `json:"organization_id"`

	// JSON object with custom conditions to validate against in the future, e.g IP, mac address, major version etc.
	Conditions map[string]any `json:"conditions,omitempty"`

	// JSON object with metadata to store for the users activation.
	Meta map[string]any `json:"meta,omitempty"`
}

func (r *ActivateRequest) Validate() []error {
	var err []error
	if r.Key == "" {
		err = append(err, errors.New("key is required"))
	}
	if r.Label == "" {
		err = append(err, errors.New("label is required"))
	}
	if r.OrganizationID == "" {
		err = append(err, ErrNoOrganizationID)
	}

	return err
}

func (r *ActivateRequest) SetMeta(k string, v any) {
	if r.Meta == nil {
		r.Meta = make(map[string]any)
	}
	r.Meta[k] = v
}

func (r *ActivateRequest) SetCondition(k string, v any) {
	if r.Conditions == nil {
		r.Conditions = make(map[string]any)
	}
	r.Conditions[k] = v
}

// Activate activates a license key against Polar API
func (c *Client) Activate(ctx context.Context, activateReq ActivateRequest) (*ActivateKeyResponse, error) {
	if activateReq.OrganizationID == "" {
		activateReq.OrganizationID = c.organizationID
	}

	if err := activateReq.Validate(); len(err) > 0 {
		return nil, errors.Wrap(ErrBadRequestData, fmt.Sprintf("invalid request: %v", err))
	}

	jsonData, err := json.Marshal(activateReq)
	if err != nil {
		return nil, ErrBadRequestData
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+activateEndpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	switch resp.StatusCode {
	case http.StatusCreated, http.StatusOK:
		break

	case http.StatusForbidden:
		// Best-effort parse: activation is the only call Polar answers 403 to,
		// and it is an authoritative denial (revoked, disabled, expired, no
		// activation support, limit reached) even when the body is empty or
		// malformed.
		response, _ := isPolarError(body)
		return nil, denialFromDetail(response.Detail)

	case http.StatusNotFound:
		// Only Polar's own typed 404 means "no such key" - a generic 404 is a
		// retired route or a proxy, which must not read as a bad key.
		if _, ok := isPolarError(body); ok {
			return nil, ErrInvalidLicenseKey
		}
		return nil, fmt.Errorf("unexpected 404 from polar: %s", string(body))

	case http.StatusUnprocessableEntity:
		// Our request was malformed - in practice a misconfigured organization
		// id, since the key itself is an opaque string to Polar.
		return nil, errors.Wrap(ErrBadRequestData, "polar rejected the request payload")

	case http.StatusTooManyRequests:
		return nil, ErrRateLimitExceeded

	default:
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var response ActivateKeyResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, ErrCouldNotUnmarshalData
	}

	return &response, nil
}

type ValidateRequest struct {
	Key            string         `json:"key"`
	ActivationID   string         `json:"activation_id,omitempty"`
	OrganizationID string         `json:"organization_id"`
	Conditions     map[string]any `json:"conditions,omitempty"`
	IncrementUsage int            `json:"increment_usage,omitempty"`
}

func (r *ValidateRequest) SetCondition(k string, v any) {
	if r.Conditions == nil {
		r.Conditions = make(map[string]any)
	}
	r.Conditions[k] = v
}

func (r *ValidateRequest) Validate() []error {
	var err []error
	if r.Key == "" {
		err = append(err, errors.New("key is required"))
	}
	if r.OrganizationID == "" {
		err = append(err, ErrNoOrganizationID)
	}

	return err
}

// Validate a license key against Polar API
func (c *Client) Validate(ctx context.Context, validateReq ValidateRequest) (*ValidateResp, error) {
	if validateReq.OrganizationID == "" {
		validateReq.OrganizationID = c.organizationID
	}

	if err := validateReq.Validate(); len(err) > 0 {
		return nil, errors.Wrap(ErrBadRequestData, fmt.Sprintf("invalid request: %v", err))
	}

	jsonData, err := json.Marshal(validateReq)
	if err != nil {
		return nil, ErrBadRequestData
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+validateEndpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	switch resp.StatusCode {
	case http.StatusCreated, http.StatusOK:
		break

	case http.StatusForbidden:
		// Polar's spec documents no 403 on this endpoint, but if one ever
		// arrives it is Polar answering authoritatively, not us failing to
		// ask - map it to a denial even when the body is empty or malformed.
		response, _ := isPolarError(body)
		return nil, denialFromDetail(response.Detail)

	case http.StatusNotFound:
		// Every validation denial lands here: revoked, disabled, expired,
		// condition mismatch, unknown key, and an activation the customer
		// released from the Polar portal.
		//
		// A 404 is not reserved for denials though - a retired or moved
		// endpoint answers 404 too (FastAPI's generic {"detail":"Not Found"}),
		// and treating that as authoritative would revoke every install's
		// license fleet-wide. Polar's typed envelope is what separates the
		// two: an unknown key answers {"error":"ResourceNotFound",
		// "detail":"Not found"}, a dead route has no "error" field at all.
		if response, ok := isPolarError(body); ok {
			return nil, denialFromDetail(response.Detail)
		}
		return nil, fmt.Errorf("unexpected 404 from polar: %s", string(body))

	case http.StatusUnprocessableEntity:
		return nil, errors.Wrap(ErrBadRequestData, "polar rejected the request payload")

	case http.StatusTooManyRequests:
		return nil, ErrRateLimitExceeded

	default:
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var response ValidateResp
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, ErrCouldNotUnmarshalData
	}

	return &response, nil
}

type DeactivateRequest struct {
	Key            string `json:"key"`
	ActivationID   string `json:"activation_id"`
	OrganizationID string `json:"organization_id"`
}

func (r *DeactivateRequest) Validate() []error {
	var err []error
	if r.Key == "" {
		err = append(err, errors.New("key is required"))
	}
	if r.ActivationID == "" {
		err = append(err, errors.New("activation_id is required"))
	}
	if r.OrganizationID == "" {
		err = append(err, ErrNoOrganizationID)
	}

	return err
}

func (c *Client) Deactivate(ctx context.Context, deactivateReq DeactivateRequest) error {
	if deactivateReq.OrganizationID == "" {
		deactivateReq.OrganizationID = c.organizationID
	}

	if err := deactivateReq.Validate(); len(err) > 0 {
		return errors.Wrap(ErrBadRequestData, fmt.Sprintf("invalid request: %v", err))
	}

	log.Debug().
		Str("licenseKey", maskLicenseKey(deactivateReq.Key)).
		Str("activationId", maskID(deactivateReq.ActivationID)).
		Str("baseURL", c.baseURL).
		Msg("Deactivating Polar license activation")

	jsonData, err := json.Marshal(deactivateReq)
	if err != nil {
		return ErrBadRequestData
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+deactivateEndpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	log.Trace().
		Int("status", resp.StatusCode).
		Str("licenseKey", maskLicenseKey(deactivateReq.Key)).
		Str("activationId", maskID(deactivateReq.ActivationID)).
		Msg("Polar license deactivation response received")

	switch resp.StatusCode {
	case http.StatusOK, http.StatusCreated, http.StatusAccepted, http.StatusNoContent:
		log.Info().
			Str("licenseKey", maskLicenseKey(deactivateReq.Key)).
			Str("activationId", maskID(deactivateReq.ActivationID)).
			Msg("Polar license deactivated successfully")
		return nil
	case http.StatusForbidden:
		response, _ := isPolarError(body)
		log.Warn().
			Str("licenseKey", maskLicenseKey(deactivateReq.Key)).
			Str("activationId", maskID(deactivateReq.ActivationID)).
			Str("detail", response.Detail).
			Msg("Polar license deactivation forbidden")
		return denialFromDetail(response.Detail)
	case http.StatusNotFound:
		// Polar answers a bare {"error":"ResourceNotFound","detail":"Not found"}
		// whether it is the key or the activation that is gone. Either way
		// there is no seat left to release, which is what callers care about.
		log.Info().
			Str("licenseKey", maskLicenseKey(deactivateReq.Key)).
			Str("activationId", maskID(deactivateReq.ActivationID)).
			Msg("Polar license activation not found (already deactivated)")
		return ErrLicenseNotActivated
	case http.StatusUnprocessableEntity:
		return errors.Wrap(ErrBadRequestData, "polar rejected the request payload")
	case http.StatusTooManyRequests:
		log.Warn().
			Str("licenseKey", maskLicenseKey(deactivateReq.Key)).
			Str("activationId", maskID(deactivateReq.ActivationID)).
			Msg("Polar license deactivation rate limited")
		return ErrRateLimitExceeded
	default:
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
}

// Helper functions

// maskLicenseKey masks a license key for logging (shows first 8 chars + ***)
func maskLicenseKey(key string) string {
	if len(key) <= 8 {
		return "***"
	}
	return key[:8] + "***"
}

// maskID masks an ID for logging (shows first 8 chars + ***)
func maskID(id string) string {
	if len(id) <= 8 {
		return "***"
	}
	return id[:8] + "***"
}

// IsClientConfigured checks if the Polar client is properly configured
func (c *Client) IsClientConfigured() bool {
	return c.organizationID != ""
}
