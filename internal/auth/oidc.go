// Copyright (c) 2024-2025, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
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
}

type Claims struct {
	Subject  string `json:"sub"`
	Name     string `json:"name"`
	Username string `json:"preferred_username"`
}

// PKCEParams holds PKCE parameters for OAuth2 flow
type PKCEParams struct {
	CodeVerifier  string
	CodeChallenge string
}

// GeneratePKCEParams generates PKCE code verifier and challenge
func GeneratePKCEParams() (*PKCEParams, error) {
	// Generate 43-128 character random string for code verifier
	verifierBytes := make([]byte, 96) // 96 bytes = 128 base64url characters
	if _, err := rand.Read(verifierBytes); err != nil {
		return nil, fmt.Errorf("failed to generate PKCE code verifier: %w", err)
	}

	codeVerifier := base64.RawURLEncoding.EncodeToString(verifierBytes)

	// Generate code challenge using SHA256
	challengeBytes := sha256.Sum256([]byte(codeVerifier))
	codeChallenge := base64.RawURLEncoding.EncodeToString(challengeBytes[:])

	return &PKCEParams{
		CodeVerifier:  codeVerifier,
		CodeChallenge: codeChallenge,
	}, nil
}

// isJWE checks if a token is in JWE format (5 parts separated by dots)
func isJWE(token string) bool {
	parts := strings.Split(token, ".")
	return len(parts) == 5
}

// decryptJWE attempts to decrypt a JWE token using the client secret as the key
func (c *OIDCConfig) decryptJWE(jweToken string) (string, error) {
	// Parse the JWE token
	jwe, err := jose.ParseEncrypted(jweToken, []jose.KeyAlgorithm{jose.DIRECT, jose.A128KW, jose.A192KW, jose.A256KW}, []jose.ContentEncryption{jose.A128GCM, jose.A192GCM, jose.A256GCM})
	if err != nil {
		return "", fmt.Errorf("failed to parse JWE token: %w", err)
	}

	// Try to decrypt with client secret as key
	clientSecret := c.OAuth2Config.ClientSecret
	if clientSecret == "" {
		return "", fmt.Errorf("client secret required for JWE decryption")
	}

	// Convert client secret to appropriate key length for AES
	key := []byte(clientSecret)
	if len(key) < 16 {
		// Pad with zeros if too short
		padded := make([]byte, 16)
		copy(padded, key)
		key = padded
	} else if len(key) > 32 {
		// Truncate if too long
		key = key[:32]
	} else if len(key) > 16 && len(key) < 24 {
		// Pad to 24 bytes
		padded := make([]byte, 24)
		copy(padded, key)
		key = padded
	} else if len(key) > 24 && len(key) < 32 {
		// Pad to 32 bytes
		padded := make([]byte, 32)
		copy(padded, key)
		key = padded
	}

	// Attempt decryption
	decrypted, err := jwe.Decrypt(key)
	if err != nil {
		// Try SHA256 hash of client secret as key (common pattern)
		hasher := sha256.New()
		hasher.Write([]byte(clientSecret))
		hashedKey := hasher.Sum(nil)

		// Try with full hash (32 bytes)
		decrypted, err = jwe.Decrypt(hashedKey)
		if err != nil {
			// Try with truncated hash (16 bytes)
			decrypted, err = jwe.Decrypt(hashedKey[:16])
			if err != nil {
				return "", fmt.Errorf("failed to decrypt JWE token with client secret or hash: %w", err)
			}
		}
	}

	log.Debug().Msg("Successfully decrypted JWE token")
	return string(decrypted), nil
}

func NewOIDC(ctx context.Context, cfg config.OIDCConfig) (*OIDCConfig, error) {
	if cfg.Issuer == "" {
		log.Debug().Msg("Using built-in authentication")
		return nil, nil
	}

	log.Debug().Str("issuer", cfg.Issuer).Msg("Initializing OIDC provider")

	// Perform manual discovery
	endpoints, _, err := getProviderEndpoints(ctx, http.DefaultClient, cfg.Issuer)
	if err != nil {
		log.Error().Err(err).Str("issuer", cfg.Issuer).Msg("Failed to discover OIDC provider endpoints")
		return nil, fmt.Errorf("failed to discover OIDC provider endpoints: %w", err)
	}

	provider, err := oidc.NewProvider(ctx, cfg.Issuer)
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
		Issuer      string `json:"issuer"`
		AuthURL     string `json:"authorization_endpoint"`
		TokenURL    string `json:"token_endpoint"`
		UserinfoURL string `json:"userinfo_endpoint"`
		JWKSURL     string `json:"jwks_uri"`
	}

	if err := json.Unmarshal(body, &discovery); err != nil {
		return oauth2.Endpoint{}, "", fmt.Errorf("parsing discovery document: %w", err)
	}

	log.Debug().
		Str("issuer", discovery.Issuer).
		Str("auth_url", discovery.AuthURL).
		Str("token_url", discovery.TokenURL).
		Msg("OIDC discovery successful")

	return oauth2.Endpoint{
		AuthURL:  discovery.AuthURL,
		TokenURL: discovery.TokenURL,
	}, discovery.UserinfoURL, nil
}

func (c *OIDCConfig) AuthURL() string {
	return c.OAuth2Config.AuthCodeURL("state")
}

// AuthURLWithPKCE generates an authorization URL with PKCE parameters
func (c *OIDCConfig) AuthURLWithPKCE(state string, pkce *PKCEParams) string {
	return c.OAuth2Config.AuthCodeURL(state,
		oauth2.SetAuthURLParam("code_challenge", pkce.CodeChallenge),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"))
}

// ExchangeCodeWithPKCE exchanges authorization code for tokens using PKCE
func (c *OIDCConfig) ExchangeCodeWithPKCE(ctx context.Context, code string, codeVerifier string) (*oauth2.Token, error) {
	return c.OAuth2Config.Exchange(ctx, code,
		oauth2.SetAuthURLParam("code_verifier", codeVerifier))
}

func (c *OIDCConfig) VerifyToken(ctx context.Context, token string) error {
	// Check if token is JWE and decrypt if necessary
	var verifyToken string = token
	if isJWE(token) {
		log.Debug().Msg("Detected JWE token, attempting decryption")
		decrypted, err := c.decryptJWE(token)
		if err != nil {
			log.Error().Err(err).Msg("Failed to decrypt JWE token")
			return fmt.Errorf("failed to decrypt JWE token: %w", err)
		}
		verifyToken = decrypted
		log.Debug().Msg("JWE token decrypted successfully")
	}

	idToken, err := c.verifier.Verify(ctx, verifyToken)
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
	// Check if token is JWE and decrypt if necessary
	var verifyToken string = token
	if isJWE(token) {
		log.Debug().Msg("Detected JWE token in GetClaims, attempting decryption")
		decrypted, err := c.decryptJWE(token)
		if err != nil {
			log.Error().Err(err).Msg("Failed to decrypt JWE token in GetClaims")
			return nil, fmt.Errorf("failed to decrypt JWE token: %w", err)
		}
		verifyToken = decrypted
		log.Debug().Msg("JWE token decrypted successfully in GetClaims")
	}

	idToken, err := c.verifier.Verify(ctx, verifyToken)
	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	var claims Claims
	if err := idToken.Claims(&claims); err != nil {
		return nil, fmt.Errorf("failed to parse claims: %w", err)
	}

	return &claims, nil
}
