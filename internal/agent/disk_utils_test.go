// Copyright (c) 2024-2026, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package agent

import (
	"testing"

	"github.com/shirou/gopsutil/v4/disk"
	"github.com/stretchr/testify/assert"

	"github.com/autobrr/netronome/internal/config"
)

func TestBuildDiskReportEntries_ExplicitIncludeBypassesSizeFilter(t *testing.T) {
	agent := New(&config.AgentConfig{
		DiskIncludes: []string{"/var/log"},
	})

	partitions := []disk.PartitionStat{
		{
			Device:     "tmpfs",
			Mountpoint: "/var/log",
			Fstype:     "tmpfs",
		},
	}

	entries := agent.buildDiskReportEntries(partitions, func(mountpoint string) (*disk.UsageStat, error) {
		return &disk.UsageStat{
			Path:        mountpoint,
			Total:       128 * 1024 * 1024,
			Used:        64 * 1024 * 1024,
			Free:        64 * 1024 * 1024,
			UsedPercent: 50,
		}, nil
	})

	assert.Len(t, entries, 1)
	assert.Equal(t, "/var/log", entries[0].Partition.Mountpoint)
	assert.True(t, entries[0].ExplicitInclude)
}

func TestBuildDiskReportEntries_DedupesNonExplicitBindMounts(t *testing.T) {
	agent := New(&config.AgentConfig{})

	partitions := []disk.PartitionStat{
		{
			Device:     "/dev/mmcblk0p2",
			Mountpoint: "/",
			Fstype:     "ext4",
			Opts:       []string{"rw", "relatime"},
		},
		{
			Device:     "/dev/mmcblk0p2",
			Mountpoint: "/var/hdd.log",
			Fstype:     "ext4",
			Opts:       []string{"rw", "relatime", "bind"},
		},
	}

	entries := agent.buildDiskReportEntries(partitions, func(mountpoint string) (*disk.UsageStat, error) {
		return &disk.UsageStat{
			Path:        mountpoint,
			Total:       32 * 1024 * 1024 * 1024,
			Used:        8 * 1024 * 1024 * 1024,
			Free:        24 * 1024 * 1024 * 1024,
			UsedPercent: 25,
		}, nil
	})

	assert.Len(t, entries, 1)
	assert.Equal(t, "/", entries[0].Partition.Mountpoint)
}

func TestBuildDiskReportEntries_KeepsExplicitlyIncludedBindMounts(t *testing.T) {
	agent := New(&config.AgentConfig{
		DiskIncludes: []string{"/var/hdd.log"},
	})

	partitions := []disk.PartitionStat{
		{
			Device:     "/dev/mmcblk0p2",
			Mountpoint: "/",
			Fstype:     "ext4",
			Opts:       []string{"rw", "relatime"},
		},
		{
			Device:     "/dev/mmcblk0p2",
			Mountpoint: "/var/hdd.log",
			Fstype:     "ext4",
			Opts:       []string{"rw", "relatime", "bind"},
		},
	}

	entries := agent.buildDiskReportEntries(partitions, func(mountpoint string) (*disk.UsageStat, error) {
		return &disk.UsageStat{
			Path:        mountpoint,
			Total:       32 * 1024 * 1024 * 1024,
			Used:        8 * 1024 * 1024 * 1024,
			Free:        24 * 1024 * 1024 * 1024,
			UsedPercent: 25,
		}, nil
	})

	assert.Len(t, entries, 2)
	assert.Equal(t, "/", entries[0].Partition.Mountpoint)
	assert.Equal(t, "/var/hdd.log", entries[1].Partition.Mountpoint)
	assert.True(t, entries[1].ExplicitInclude)
}
