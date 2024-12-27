// Copyright (c) 2024, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/rs/zerolog/log"
	"golang.org/x/oauth2"
)

type OIDCConfig struct {
	provider     *oidc.Provider
	OAuth2Config oauth2.Config
	verifier     *oidc.IDTokenVerifier
}

type Claims struct {
	Subject  string `json:"sub"`
	Name     string `json:"name"`
	Username string `json:"preferred_username"`
}

func NewOIDC(ctx context.Context) (*OIDCConfig, error) {
	issuer := os.Getenv("OIDC_ISSUER")
	if issuer == "" {
		log.Debug().Msg("Using built-in authentication")
		return nil, nil
	}

	log.Debug().Str("issuer", issuer).Msg("Initializing OIDC provider")

	// Perform manual discovery
	endpoints, _, err := getProviderEndpoints(ctx, http.DefaultClient, issuer)
	if err != nil {
		log.Error().Err(err).Str("issuer", issuer).Msg("Failed to discover OIDC provider endpoints")
		return nil, fmt.Errorf("failed to discover OIDC provider endpoints: %w", err)
	}

	provider, err := oidc.NewProvider(ctx, issuer)
	if err != nil {
		log.Error().Err(err).Str("issuer", issuer).Msg("Failed to initialize OIDC provider")
		return nil, fmt.Errorf("failed to initialize OIDC provider: %w", err)
	}
	// log.Info().Str("issuer", issuer).Msg("OIDC provider initialized successfully")

	config := oauth2.Config{
		ClientID:     os.Getenv("OIDC_CLIENT_ID"),
		ClientSecret: os.Getenv("OIDC_CLIENT_SECRET"),
		RedirectURL:  os.Getenv("OIDC_REDIRECT_URL"),
		Endpoint:     endpoints,
		Scopes:       []string{oidc.ScopeOpenID, "profile"},
	}

	log.Trace().
		Str("clientID", config.ClientID).
		Str("redirectURL", config.RedirectURL).
		Str("authURL", endpoints.AuthURL).
		Str("tokenURL", endpoints.TokenURL).
		// Str("userinfoURL", userinfoURL).
		Strs("scopes", config.Scopes).
		Msg("OIDC configuration created")

	return &OIDCConfig{
		provider:     provider,
		OAuth2Config: config,
		verifier:     provider.Verifier(&oidc.Config{ClientID: config.ClientID}),
	}, nil
}

func getProviderEndpoints(ctx context.Context, client *http.Client, issuer string) (oauth2.Endpoint, string, error) {
	issuer = strings.TrimRight(issuer, "/")

	wellKnown := issuer + "/.well-known/openid-configuration"
	if strings.Contains(issuer, "/.well-known/openid-configuration") {
		wellKnown = issuer
	}

	log.Trace().Str("well_known_url", wellKnown).Msg("Fetching OIDC discovery document")

	req, err := http.NewRequestWithContext(ctx, "GET", wellKnown, nil)
	if err != nil {
		return oauth2.Endpoint{}, "", fmt.Errorf("creating discovery request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return oauth2.Endpoint{}, "", fmt.Errorf("fetching discovery document: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return oauth2.Endpoint{}, "", fmt.Errorf("reading discovery document: %w", err)
	}

	var discovery struct {
		Issuer        string   `json:"issuer"`
		AuthURL       string   `json:"authorization_endpoint"`
		TokenURL      string   `json:"token_endpoint"`
		UserinfoURL   string   `json:"userinfo_endpoint"`
		JWKSURL       string   `json:"jwks_uri"`
		ResponseTypes []string `json:"response_types_supported"`
		SubjectTypes  []string `json:"subject_types_supported"`
		SigningAlgs   []string `json:"id_token_signing_alg_values_supported"`
	}

	if err := json.Unmarshal(body, &discovery); err != nil {
		return oauth2.Endpoint{}, "", fmt.Errorf("parsing discovery document: %w", err)
	}

	log.Debug().
		Str("issuer", discovery.Issuer).
		Str("auth_url", discovery.AuthURL).
		Str("token_url", discovery.TokenURL).
		// Str("userinfo_url", discovery.UserinfoURL).
		// Str("jwks_url", discovery.JWKSURL).
		// Strs("response_types", discovery.ResponseTypes).
		// Strs("subject_types", discovery.SubjectTypes).
		// Strs("signing_algs", discovery.SigningAlgs).
		Msg("OIDC discovery successful")

	return oauth2.Endpoint{
		AuthURL:  discovery.AuthURL,
		TokenURL: discovery.TokenURL,
	}, discovery.UserinfoURL, nil
}

func (c *OIDCConfig) AuthURL() string {
	return c.OAuth2Config.AuthCodeURL("state")
}

func (c *OIDCConfig) VerifyToken(ctx context.Context, token string) error {
	idToken, err := c.verifier.Verify(ctx, token)
	if err != nil {
		return fmt.Errorf("invalid token: %w", err)
	}

	var claims struct {
		Subject string `json:"sub"`
		Expiry  int64  `json:"exp"`
	}
	if err := idToken.Claims(&claims); err != nil {
		return fmt.Errorf("failed to parse claims: %w", err)
	}

	if claims.Subject == "" {
		return fmt.Errorf("token missing subject claim")
	}

	now := time.Now()
	if claims.Expiry == 0 {
		// If expiry is missing, set it to 24 hours from the current time
		claims.Expiry = now.Add(24 * time.Hour).Unix()
	}

	if now.After(time.Unix(claims.Expiry, 0)) {
		return fmt.Errorf("token has expired")
	}

	return nil
}

func (c *OIDCConfig) GetClaims(ctx context.Context, token string) (*Claims, error) {
	idToken, err := c.verifier.Verify(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	var claims Claims
	if err := idToken.Claims(&claims); err != nil {
		return nil, fmt.Errorf("failed to parse claims: %w", err)
	}

	return &claims, nil
}
