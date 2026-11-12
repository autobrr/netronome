// Copyright (c) 2024-2026, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package update

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"

	version "github.com/hashicorp/go-version"
	"github.com/rs/zerolog/log"
	"golang.org/x/mod/module"
)

const latestReleaseURL = "https://api.autobrr.com/repos/autobrr/netronome/releases/latest"

var gitDescribeSuffix = regexp.MustCompile(`-[0-9]+-g[0-9a-f]+$`)

// Release is the release metadata shown in the UI when an update is available.
type Release struct {
	TagName     string    `json:"tag_name"`
	Name        string    `json:"name,omitempty"`
	HTMLURL     string    `json:"html_url"`
	PublishedAt time.Time `json:"published_at"`
}

// Checker periodically checks the Netronome release feed and keeps the newest
// applicable release in memory.
type Checker struct {
	enabled        bool
	currentVersion string
	client         *http.Client

	mu     sync.RWMutex
	latest *Release
}

func New(enabled bool, currentVersion string, client *http.Client) *Checker {
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}

	return &Checker{
		enabled:        enabled,
		currentVersion: currentVersion,
		client:         client,
	}
}

// Start performs the initial check after two seconds, then checks every two
// hours until ctx is cancelled.
func (c *Checker) Start(ctx context.Context) {
	if !c.enabled || !isCheckableVersion(c.currentVersion) {
		return
	}

	initial := time.NewTimer(2 * time.Second)
	defer initial.Stop()

	select {
	case <-ctx.Done():
		return
	case <-initial.C:
		c.checkAndLog(ctx)
	}

	ticker := time.NewTicker(2 * time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.checkAndLog(ctx)
		}
	}
}

func (c *Checker) checkAndLog(ctx context.Context) {
	if err := c.Check(ctx); err != nil && ctx.Err() == nil {
		log.Warn().Err(err).Msg("Failed to check for Netronome updates")
	}
}

// Check fetches the latest release and updates the in-memory cache. Errors do
// not modify an existing cached update.
func (c *Checker) Check(ctx context.Context) error {
	if !c.enabled || !isCheckableVersion(c.currentVersion) {
		return nil
	}

	release, err := c.fetch(ctx)
	if err != nil {
		return err
	}

	isNewer, err := IsNewer(c.currentVersion, release.TagName)
	if err != nil {
		return fmt.Errorf("compare releases: %w", err)
	}

	c.mu.Lock()
	if isNewer {
		changed := c.latest == nil || c.latest.TagName != release.TagName
		c.latest = &release
		c.mu.Unlock()
		if changed {
			log.Info().Str("tag", release.TagName).Msg("New Netronome release detected")
		}
		return nil
	}
	c.latest = nil
	c.mu.Unlock()
	return nil
}

func (c *Checker) fetch(ctx context.Context) (Release, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, latestReleaseURL, nil)
	if err != nil {
		return Release{}, fmt.Errorf("create latest release request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", fmt.Sprintf("netronome/%s (%s %s)", c.currentVersion, runtime.GOOS, runtime.GOARCH))

	resp, err := c.client.Do(req)
	if err != nil {
		return Release{}, fmt.Errorf("fetch latest release: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return Release{}, fmt.Errorf("fetch latest release: unexpected status %s", resp.Status)
	}

	var release Release
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return Release{}, fmt.Errorf("decode latest release: %w", err)
	}
	releaseURL, err := url.Parse(release.HTMLURL)
	if err != nil || releaseURL.Scheme != "https" || releaseURL.Host == "" {
		return Release{}, fmt.Errorf("decode latest release: unsafe release URL %q", release.HTMLURL)
	}
	return release, nil
}

// Latest returns a copy of the cached update, if one is available.
func (c *Checker) Latest() *Release {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.latest == nil {
		return nil
	}
	latest := *c.latest
	return &latest
}

// IsNewer reports whether latest is a strictly newer release than current.
func IsNewer(current, latest string) (bool, error) {
	currentVersion, err := version.NewVersion(current)
	if err != nil {
		return false, fmt.Errorf("parse current version %q: %w", current, err)
	}
	latestVersion, err := version.NewVersion(latest)
	if err != nil {
		return false, fmt.Errorf("parse latest version %q: %w", latest, err)
	}

	if currentVersion.Prerelease() == "" && latestVersion.Prerelease() != "" {
		return false, nil
	}
	return latestVersion.GreaterThan(currentVersion), nil
}

// IsAgentOutdated reports whether server is a newer release than agent. A
// development build on either side, an empty version, or a version that does
// not parse is never outdated.
func IsAgentOutdated(agent, server string) bool {
	if !isCheckableVersion(agent) || !isCheckableVersion(server) {
		return false
	}
	outdated, err := IsNewer(agent, server)
	return err == nil && outdated
}

func isCheckableVersion(current string) bool {
	current = strings.ToLower(strings.TrimSpace(current))
	if current == "" || current == "dev" || current == "develop" || current == "main" || current == "latest" {
		return false
	}
	if strings.Contains(current, "(devel)") || strings.HasPrefix(current, "pr-") {
		return false
	}
	// git describe --always prints a bare commit hash when no tag is reachable.
	if !strings.Contains(current, ".") {
		return false
	}
	if strings.HasSuffix(current, "-dev") || strings.HasSuffix(current, "-develop") || strings.HasSuffix(current, "-dirty") {
		return false
	}
	if module.IsPseudoVersion("v" + strings.TrimPrefix(current, "v")) {
		return false
	}
	// git describe appends "-<commits>-g<hash>" when HEAD is past the tag.
	return !gitDescribeSuffix.MatchString(current)
}
