// Copyright (c) 2024-2025, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/go-jose/go-jose/v4"
	"github.com/rs/zerolog/log"
	"golang.org/x/oauth2"

	"github.com/autobrr/netronome/internal/config"
)

type OIDCConfig struct {
	provider     *oidc.Provider
	OAuth2Config oauth2.Config
	verifier     *oidc.IDTokenVerifier
	httpClient   *loggingHTTPClient
	clientSecret string
}

type Claims struct {
	Subject  string `json:"sub"`
	Name     string `json:"name"`
	Username string `json:"preferred_username"`
}

// loggingHTTPClient wraps an http.Client to log requests and responses
type loggingHTTPClient struct {
	client *http.Client
}

func (l *loggingHTTPClient) Do(req *http.Request) (*http.Response, error) {
	log.Debug().Str("url", req.URL.String()).Msg("OIDC HTTP request")
	
	resp, err := l.client.Do(req)
	if err != nil {
		return nil, err
	}
	
	// Only log token endpoint responses for debugging JWE issues
	if strings.Contains(req.URL.Path, "token") {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		
		log.Debug().
			Int("status", resp.StatusCode).
			Msg("OIDC token response received")
		
		// Check for malformed JSON responses that could cause parsing issues
		if len(body) > 0 {
			bodyStr := string(body)
			decoder := json.NewDecoder(strings.NewReader(bodyStr))
			var temp interface{}
			if err := decoder.Decode(&temp); err == nil && decoder.More() {
				offset := decoder.InputOffset()
				if offset < int64(len(bodyStr)) {
					remaining := bodyStr[offset:]
					log.Warn().
						Str("extra_content", remaining).
						Msg("Extra content found after JSON in token response")
				}
			}
		}
		
		// Restore response body
		resp.Body = io.NopCloser(strings.NewReader(string(body)))
	}
	
	return resp, nil
}

func NewOIDC(ctx context.Context, cfg config.OIDCConfig) (*OIDCConfig, error) {
	if cfg.Issuer == "" {
		log.Debug().Msg("Using built-in authentication")
		return nil, nil
	}

	log.Debug().Str("issuer", cfg.Issuer).Msg("Initializing OIDC provider")

	// Create a custom HTTP client with logging
	loggingClient := &loggingHTTPClient{
		client: http.DefaultClient,
	}

	// Perform manual discovery
	endpoints, _, err := getProviderEndpoints(ctx, loggingClient, cfg.Issuer)
	if err != nil {
		log.Error().Err(err).Str("issuer", cfg.Issuer).Msg("Failed to discover OIDC provider endpoints")
		return nil, fmt.Errorf("failed to discover OIDC provider endpoints: %w", err)
	}

	// Create context with custom HTTP client for provider
	ctxWithClient := context.WithValue(ctx, oauth2.HTTPClient, loggingClient)
	provider, err := oidc.NewProvider(ctxWithClient, cfg.Issuer)
	if err != nil {
		log.Error().Err(err).Str("issuer", cfg.Issuer).Msg("Failed to initialize OIDC provider")
		return nil, fmt.Errorf("failed to initialize OIDC provider: %w", err)
	}

	config := oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		RedirectURL:  cfg.RedirectURL,
		Endpoint:     endpoints,
		Scopes:       []string{oidc.ScopeOpenID, "profile"},
	}

	log.Trace().
		Str("clientID", config.ClientID).
		Str("redirectURL", config.RedirectURL).
		Str("authURL", endpoints.AuthURL).
		Str("tokenURL", endpoints.TokenURL).
		Strs("scopes", config.Scopes).
		Msg("OIDC configuration created")

	return &OIDCConfig{
		provider:     provider,
		OAuth2Config: config,
		verifier:     provider.Verifier(&oidc.Config{ClientID: config.ClientID}),
		httpClient:   loggingClient,
		clientSecret: cfg.ClientSecret,
	}, nil
}

func getProviderEndpoints(ctx context.Context, client interface{ Do(*http.Request) (*http.Response, error) }, issuer string) (oauth2.Endpoint, string, error) {
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

func (c *OIDCConfig) GetHTTPClient() interface{ Do(*http.Request) (*http.Response, error) } {
	return c.httpClient
}

func (c *OIDCConfig) VerifyToken(ctx context.Context, token string) error {
	log.Debug().Msg("verifying ID token")
	
	// Check if this is a JWE (encrypted) token and decrypt if needed
	decryptedToken, err := c.decryptTokenIfNeeded(ctx, token)
	if err != nil {
		log.Error().Err(err).Msg("failed to decrypt JWE token")
		return fmt.Errorf("failed to decrypt token: %w", err)
	}
	
	// Use the decrypted token (or original if it wasn't encrypted)
	tokenToVerify := decryptedToken
	if tokenToVerify != token {
		log.Debug().Msg("successfully decrypted JWE token")
	}
	
	// Check if token is single-part (Pocket-ID format)
	tokenParts := strings.Split(tokenToVerify, ".")
	if len(tokenParts) == 1 {
		log.Debug().Msg("Pocket-ID format detected")
		return nil
	}
	
	idToken, err := c.verifier.Verify(ctx, tokenToVerify)
	if err != nil {
		log.Error().Err(err).Msg("token verification failed")
		return fmt.Errorf("invalid token: %w", err)
	}

	var claims struct {
		Subject string `json:"sub"`
		Expiry  int64  `json:"exp"`
	}
	if err := idToken.Claims(&claims); err != nil {
		log.Error().Err(err).Msg("failed to parse claims from verified token")
		return fmt.Errorf("failed to parse claims: %w", err)
	}
	log.Debug().Msg("token verified successfully")

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

// decryptTokenIfNeeded checks if the token is a JWE and decrypts it if necessary
func (c *OIDCConfig) decryptTokenIfNeeded(ctx context.Context, token string) (string, error) {
	// Check if this looks like a JWE by counting parts (JWE has 5 parts, JWT has 3)
	parts := strings.Split(token, ".")
	if len(parts) != 5 {
		return token, nil
	}
	
	// Try to parse as JWE with common algorithms
	jwe, err := jose.ParseEncrypted(token, []jose.KeyAlgorithm{jose.RSA_OAEP_256, jose.RSA_OAEP}, []jose.ContentEncryption{jose.A256CBC_HS512, jose.A256GCM})
	if err != nil {
		// If parsing as JWE fails, treat as regular JWT
		return token, nil
	}
	
	log.Debug().Msg("detected JWE encrypted token, attempting decryption")
	
	// Try decrypting with the client secret (for symmetric encryption)
	if decrypted, err := jwe.Decrypt([]byte(c.clientSecret)); err == nil {
		log.Debug().Msg("JWE token decrypted with client secret")
		return string(decrypted), nil
	}
	
	// If client secret doesn't work, try JWKS method (for asymmetric encryption)
	return c.decryptWithJWKS(ctx, jwe)
}

// decryptWithJWKS attempts to decrypt using keys from the JWKS endpoint
func (c *OIDCConfig) decryptWithJWKS(ctx context.Context, jwe *jose.JSONWebEncryption) (string, error) {
	// For now, we'll return an error as we need to implement proper key retrieval
	// The go-oidc library doesn't expose the KeySet directly in v3
	log.Debug().Msg("JWE decryption with JWKS not yet implemented")
	return "", fmt.Errorf("JWE decryption with JWKS not yet implemented - configure Authentik to use client secret encryption or disable encryption")
}

func (c *OIDCConfig) GetClaims(ctx context.Context, token string) (*Claims, error) {
	// Check if this is a JWE (encrypted) token and decrypt if needed
	decryptedToken, err := c.decryptTokenIfNeeded(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt token: %w", err)
	}
	
	idToken, err := c.verifier.Verify(ctx, decryptedToken)
	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	var claims Claims
	if err := idToken.Claims(&claims); err != nil {
		return nil, fmt.Errorf("failed to parse claims: %w", err)
	}

	return &claims, nil
}
