// Copyright (c) 2024-2026, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package server

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/autobrr/netronome/internal/auth"
)

func TestPKCEVerifierRoundTrip(t *testing.T) {
	h := NewAuthHandler(nil, nil, false, "", nil)

	h.storePKCEVerifier("state-1", "verifier-1")

	verifier, ok := h.getPKCEVerifier("state-1")
	require.True(t, ok)
	assert.Equal(t, "verifier-1", verifier)

	_, ok = h.getPKCEVerifier("state-1")
	assert.False(t, ok, "verifier must be single use")
}

func TestPKCEVerifierExpires(t *testing.T) {
	h := NewAuthHandler(nil, nil, false, "", nil)

	h.pkceVerifiers["state-1"] = pkceEntry{
		verifier: "verifier-1",
		expires:  time.Now().Add(-time.Second),
	}

	_, ok := h.getPKCEVerifier("state-1")
	assert.False(t, ok, "expired verifier must not be accepted")
	assert.NotContains(t, h.pkceVerifiers, "state-1", "expired verifier must be removed")
}

func TestStorePKCEVerifierSweepsExpiredEntries(t *testing.T) {
	h := NewAuthHandler(nil, nil, false, "", nil)

	for i := range 10 {
		h.pkceVerifiers[fmt.Sprintf("stale-%d", i)] = pkceEntry{
			verifier: "old",
			expires:  time.Now().Add(-time.Minute),
		}
	}

	h.storePKCEVerifier("fresh", "verifier")

	assert.Len(t, h.pkceVerifiers, 1, "abandoned logins must not accumulate")
	assert.Contains(t, h.pkceVerifiers, "fresh")
}

func TestStorePKCEVerifierEnforcesCap(t *testing.T) {
	h := NewAuthHandler(nil, nil, false, "", nil)

	for i := range pkceVerifierMax+2 {
		h.storePKCEVerifier(fmt.Sprintf("state-%d", i), "verifier")
	}

	assert.LessOrEqual(t, len(h.pkceVerifiers), pkceVerifierMax)

	// The newest state survives the eviction.
	_, ok := h.getPKCEVerifier(fmt.Sprintf("state-%d", pkceVerifierMax+1))
	assert.True(t, ok)
}

func TestMemorySessionExpiry(t *testing.T) {
	tests := []struct {
		name    string
		expires time.Time
		valid   bool
	}{
		{name: "before expiry", expires: time.Now().Add(time.Hour), valid: true},
		{name: "after expiry", expires: time.Now().Add(-time.Second), valid: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewAuthHandler(nil, nil, false, "", nil)

			token := auth.MemoryOnlyPrefix + "session-token"
			h.sessionTokens[token] = memorySession{
				claims:  SessionClaims{Version: sessionClaimsVersion, Type: sessionTypeLocal},
				expires: tt.expires,
			}

			assert.Equal(t, tt.valid, h.isValidMemorySession(token))

			claims, ok := h.getMemorySessionClaims(token)
			require.Equal(t, tt.valid, ok)
			if tt.valid {
				assert.Equal(t, sessionTypeLocal, claims.Type)
			}
		})
	}
}

func TestTrackMemorySessionSweepsExpiredEntries(t *testing.T) {
	h := NewAuthHandler(nil, nil, false, "", nil)

	for i := range 10 {
		h.sessionTokens[fmt.Sprintf("stale-%d", i)] = memorySession{
			claims:  SessionClaims{Version: sessionClaimsVersion, Type: sessionTypeLocal},
			expires: time.Now().Add(-time.Minute),
		}
	}

	h.trackMemorySession("", auth.MemoryOnlyPrefix+"fresh", nil)

	assert.Len(t, h.sessionTokens, 1, "expired memory sessions must not accumulate")
	assert.True(t, h.isValidMemorySession(auth.MemoryOnlyPrefix+"fresh"))
}

func TestTrackMemorySessionExtendsExpiry(t *testing.T) {
	h := NewAuthHandler(nil, nil, false, "", nil)

	token := auth.MemoryOnlyPrefix + "session-token"
	h.sessionTokens[token] = memorySession{
		claims:  SessionClaims{Version: sessionClaimsVersion, Type: sessionTypeOIDC, Username: "alice"},
		expires: time.Now().Add(time.Minute),
	}

	h.trackMemorySession("", token, nil)

	entry := h.sessionTokens[token]
	assert.Equal(t, "alice", entry.claims.Username, "existing claims must survive a refresh")
	assert.Greater(t, time.Until(entry.expires), time.Hour, "refresh must extend the expiry")
}
