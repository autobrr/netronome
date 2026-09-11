// Copyright (c) 2024-2026, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package license

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/keygen-sh/machineid"
	"github.com/rs/zerolog/log"
)

const (
	appID           = "netronome"
	fingerprintFile = ".device-id"
)

// GetDeviceID returns a stable fingerprint for this install. The value is
// persisted under configDir on first use so it survives machineid backend
// changes (container rebuilds, host ID rotation) that would otherwise
// invalidate the Polar activation.
func GetDeviceID(configDir string) (string, error) {
	path := filepath.Join(configDir, fingerprintFile)

	if content, err := os.ReadFile(path); err == nil {
		if existing := strings.TrimSpace(string(content)); existing != "" {
			log.Trace().Str("path", path).Msg("using existing fingerprint")
			return existing, nil
		}
	}

	baseID, err := machineid.ProtectedID(appID)
	if err != nil {
		log.Warn().Err(err).Msg("failed to get machine ID, using fallback")
		baseID = fallbackMachineID()
	}

	sum := sha256.Sum256([]byte(appID + "-" + baseID))
	fingerprint := hex.EncodeToString(sum[:])

	return persistFingerprint(path, fingerprint), nil
}

// fallbackMachineID is used when the platform machine ID is unreadable, which
// happens in minimal containers. It is weaker but stable for a given host.
func fallbackMachineID() string {
	hostInfo := fmt.Sprintf("%s-%s", runtime.GOOS, runtime.GOARCH)
	if hostname, err := os.Hostname(); err == nil {
		hostInfo = fmt.Sprintf("%s-%s", hostInfo, hostname)
	}

	sum := sha256.Sum256([]byte(hostInfo))
	return hex.EncodeToString(sum[:])[:32]
}

// persistFingerprint best-effort writes the fingerprint. A read-only config dir
// is not fatal: the fingerprint is deterministic, it just gets recomputed.
func persistFingerprint(path, fingerprint string) string {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		log.Warn().Err(err).Str("path", path).Msg("failed to create fingerprint directory")
		return fingerprint
	}

	if err := os.WriteFile(path, []byte(fingerprint), 0600); err != nil {
		log.Warn().Err(err).Str("path", path).Msg("failed to persist fingerprint")
		return fingerprint
	}

	log.Trace().Str("path", path).Msg("persisted new fingerprint")

	return fingerprint
}
