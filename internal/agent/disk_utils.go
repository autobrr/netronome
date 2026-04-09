// Copyright (c) 2024-2026, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/shirou/gopsutil/v4/disk"
)

const minDiskReportSize = 1024 * 1024 * 1024

type diskReportEntry struct {
	Partition       disk.PartitionStat
	Usage           *disk.UsageStat
	ExplicitInclude bool
}

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

func (a *Agent) diskFilterDecision(mountpoint, device, fstype string) (include bool, explicit bool) {
	// First check if it's explicitly included - this takes precedence
	for _, pattern := range a.config.DiskIncludes {
		if matchPath(pattern, mountpoint) {
			log.Debug().Str("mount", mountpoint).Str("pattern", pattern).Msg("Disk included by pattern match")
			return true, true
		}
	}

	// Then check if it's explicitly excluded
	for _, pattern := range a.config.DiskExcludes {
		if matchPath(pattern, mountpoint) {
			log.Debug().Str("mount", mountpoint).Str("pattern", pattern).Msg("Disk excluded by pattern match")
			return false, false
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
		return false, false
	}

	// Include everything else by default
	return true, false
}

// shouldIncludeDisk determines if a disk should be included in monitoring based on filter rules
func (a *Agent) shouldIncludeDisk(mountpoint, device, fstype string) bool {
	include, _ := a.diskFilterDecision(mountpoint, device, fstype)
	return include
}

func (a *Agent) buildDiskReportEntries(partitions []disk.PartitionStat, usageFn func(string) (*disk.UsageStat, error)) []diskReportEntry {
	entries := make([]diskReportEntry, 0, len(partitions))

	for _, partition := range partitions {
		include, explicit := a.diskFilterDecision(partition.Mountpoint, partition.Device, partition.Fstype)
		if !include {
			log.Debug().
				Str("mount", partition.Mountpoint).
				Str("device", partition.Device).
				Str("fstype", partition.Fstype).
				Msg("Skipping disk based on filter rules")
			continue
		}

		usage, err := usageFn(partition.Mountpoint)
		if err != nil {
			log.Debug().Err(err).Str("mount", partition.Mountpoint).Msg("Failed to get disk usage")
			continue
		}

		if !explicit && usage.Total < minDiskReportSize {
			log.Debug().
				Str("mount", partition.Mountpoint).
				Uint64("total", usage.Total).
				Msg("Skipping small disk")
			continue
		}

		entries = append(entries, diskReportEntry{
			Partition:       partition,
			Usage:           usage,
			ExplicitInclude: explicit,
		})
	}

	return dedupeDiskReportEntries(entries)
}

func dedupeDiskReportEntries(entries []diskReportEntry) []diskReportEntry {
	if len(entries) < 2 {
		return entries
	}

	grouped := make(map[string][]int, len(entries))
	for i, entry := range entries {
		grouped[diskReportSignature(entry)] = append(grouped[diskReportSignature(entry)], i)
	}

	keep := make([]bool, len(entries))

	for _, indexes := range grouped {
		if len(indexes) == 1 {
			keep[indexes[0]] = true
			continue
		}

		preferred := indexes[0]
		for _, idx := range indexes[1:] {
			if prefersDiskEntry(entries[idx], entries[preferred]) {
				preferred = idx
			}
		}

		keep[preferred] = true

		for _, idx := range indexes {
			if idx == preferred {
				continue
			}
			if entries[idx].ExplicitInclude || !isBindMount(entries[idx].Partition.Opts) {
				keep[idx] = true
			}
		}
	}

	deduped := make([]diskReportEntry, 0, len(entries))
	for i, entry := range entries {
		if keep[i] {
			deduped = append(deduped, entry)
		}
	}

	return deduped
}

func diskReportSignature(entry diskReportEntry) string {
	return fmt.Sprintf("%s|%s|%d|%d|%d", entry.Partition.Device, entry.Partition.Fstype, entry.Usage.Total, entry.Usage.Used, entry.Usage.Free)
}

func prefersDiskEntry(left, right diskReportEntry) bool {
	leftBind := isBindMount(left.Partition.Opts)
	rightBind := isBindMount(right.Partition.Opts)

	if leftBind != rightBind {
		return !leftBind
	}

	if len(left.Partition.Mountpoint) != len(right.Partition.Mountpoint) {
		return len(left.Partition.Mountpoint) < len(right.Partition.Mountpoint)
	}

	return left.Partition.Mountpoint < right.Partition.Mountpoint
}

func isBindMount(opts []string) bool {
	for _, opt := range opts {
		if opt == "bind" {
			return true
		}
	}

	return false
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
