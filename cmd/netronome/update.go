// Copyright (c) 2024-2025, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package main

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/creativeprojects/go-selfupdate"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update Netronome to the latest version",
	Long: `Check for and install the latest version of Netronome.
This command will download the latest release from GitHub and replace the current binary.`,
	RunE: runUpdate,
}

func runUpdate(cmd *cobra.Command, args []string) error {
	log.Info().Str("current_version", version).Msg("Checking for updates...")

	// Create GitHub repository source
	repo := selfupdate.ParseSlug("autobrr/netronome")

	// First, detect the latest release without validation to get the version
	updater, err := selfupdate.NewUpdater(selfupdate.Config{})
	if err != nil {
		return fmt.Errorf("failed to create updater: %w", err)
	}

	// Get latest release info
	release, found, err := updater.DetectLatest(cmd.Context(), repo)
	if err != nil {
		return fmt.Errorf("failed to detect latest version: %w", err)
	}

	if !found {
		log.Info().Msg("No updates found")
		return nil
	}

	// Compare versions
	if release.LessOrEqual(version) {
		log.Info().
			Str("current_version", version).
			Str("latest_version", release.Version()).
			Msg("Already running the latest version")
		return nil
	}

	log.Info().
		Str("current_version", version).
		Str("latest_version", release.Version()).
		Msg("New version available")

	// Now create updater with correct checksum filename
	versionStr := strings.TrimPrefix(release.Version(), "v")

	updaterWithChecksum, err := selfupdate.NewUpdater(selfupdate.Config{
		Validator: &selfupdate.ChecksumValidator{
			UniqueFilename: fmt.Sprintf("netronome_%s_checksums.txt", versionStr),
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create updater with checksum: %w", err)
	}

	// Perform the update
	exe, err := selfupdate.ExecutablePath()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	log.Info().Str("path", exe).Msg("Updating binary...")

	if err := updaterWithChecksum.UpdateTo(cmd.Context(), release, exe); err != nil {
		return fmt.Errorf("failed to update: %w", err)
	}

	log.Info().
		Str("version", release.Version()).
		Msg("Successfully updated to the latest version")

	// Inform about potential need to restart the service
	if runtime.GOOS == "linux" {
		log.Info().Msg("If running as a systemd service, restart with: systemctl restart netronome-agent")
	}

	return nil
}
