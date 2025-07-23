// Copyright (c) 2024-2025, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package agent

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/rs/zerolog/log"
)

// matchPath checks if a path matches a pattern, supporting both glob and prefix patterns
func matchPath(pattern, path string) bool {
	// Check if pattern ends with * for prefix matching
	if strings.HasSuffix(pattern, "*") {
		prefix := strings.TrimSuffix(pattern, "*")
		return strings.HasPrefix(path, prefix)
	}

	// Check if pattern contains any glob characters
	if strings.ContainsAny(pattern, "*?[]") {
		// Use filepath.Match for single-component patterns
		matched, err := filepath.Match(pattern, path)
		if err == nil && matched {
			return true
		}
		// Also try matching just the basename
		matched, err = filepath.Match(pattern, filepath.Base(path))
		return err == nil && matched
	}

	// Otherwise, exact match
	return pattern == path
}

// shouldIncludeDisk determines if a disk should be included in monitoring based on filter rules
func (a *Agent) shouldIncludeDisk(mountpoint, device, fstype string) bool {
	// First check if it's explicitly included - this takes precedence
	for _, pattern := range a.config.DiskIncludes {
		if matchPath(pattern, mountpoint) {
			log.Debug().Str("mount", mountpoint).Str("pattern", pattern).Msg("Disk included by pattern match")
			return true
		}
	}

	// Then check if it's explicitly excluded
	for _, pattern := range a.config.DiskExcludes {
		if matchPath(pattern, mountpoint) {
			log.Debug().Str("mount", mountpoint).Str("pattern", pattern).Msg("Disk excluded by pattern match")
			return false
		}
	}

	// Skip special filesystems by default (unless explicitly included above)
	if strings.HasPrefix(mountpoint, "/snap") ||
		strings.HasPrefix(mountpoint, "/run") ||
		strings.HasPrefix(mountpoint, "/dev") ||
		strings.HasPrefix(mountpoint, "/proc") ||
		strings.HasPrefix(mountpoint, "/sys") ||
		strings.HasPrefix(mountpoint, "/var/lib/docker/overlay") ||
		strings.HasPrefix(mountpoint, "/var/lib/containers/storage/overlay") ||
		strings.HasPrefix(device, "tmpfs") ||
		strings.HasPrefix(device, "devfs") ||
		strings.HasPrefix(device, "udev") ||
		strings.HasPrefix(device, "overlay") ||
		fstype == "squashfs" ||
		fstype == "devtmpfs" ||
		fstype == "overlay" ||
		fstype == "proc" ||
		fstype == "sysfs" ||
		fstype == "cgroup" ||
		fstype == "cgroup2" {
		return false
	}

	// Include everything else by default
	return true
}

// getDevicePaths returns a list of device paths to check for SMART data
func (a *Agent) getDevicePaths() []string {
	var devicePaths []string

	if runtime.GOOS == "darwin" {
		// macOS: Look for disk devices in /dev
		entries, err := os.ReadDir("/dev")
		if err != nil {
			log.Debug().Err(err).Msg("Failed to read /dev directory")
			return devicePaths
		}

		for _, entry := range entries {
			name := entry.Name()
			// macOS uses disk0, disk1, etc. and rdisk0, rdisk1, etc.
			// We want the raw disk devices (rdisk) for SMART access
			if strings.HasPrefix(name, "rdisk") && !strings.Contains(name, "s") {
				// Skip partitions (rdisk0s1, rdisk1s2, etc.)
				devicePaths = append(devicePaths, filepath.Join("/dev", name))
			}
			// Also check for nvme devices that might exist
			if strings.HasPrefix(name, "nvme") && !strings.Contains(name, "p") {
				devicePaths = append(devicePaths, filepath.Join("/dev", name))
			}
		}
	} else {
		// Linux: Look for traditional device names
		entries, err := os.ReadDir("/dev")
		if err != nil {
			log.Debug().Err(err).Msg("Failed to read /dev directory")
			return devicePaths
		}

		for _, entry := range entries {
			name := entry.Name()
			// Look for disk devices (sda, sdb, nvme0n1, etc.)
			if !strings.HasPrefix(name, "sd") && !strings.HasPrefix(name, "nvme") && !strings.HasPrefix(name, "hd") {
				continue
			}

			// Skip partitions (sda1, sdb2, nvme0n1p1, etc.)
			if strings.Contains(name, "p") && len(name) > 4 {
				continue
			}
			if len(name) > 3 && name[len(name)-1] >= '0' && name[len(name)-1] <= '9' {
				continue
			}

			devicePaths = append(devicePaths, filepath.Join("/dev", name))
		}
	}

	return devicePaths
}
