// Copyright (c) 2024-2026, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package agent

import (
	"testing"

	"github.com/shirou/gopsutil/v4/disk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/autobrr/netronome/internal/config"
)

func TestBuildDiskReportEntries(t *testing.T) {
	tests := []struct {
		name                     string
		cfg                      config.AgentConfig
		partitions               []disk.PartitionStat
		usageFn                  func(string) (*disk.UsageStat, error)
		expectedMountpoints      []string
		expectedExplicitIncludes []bool
	}{
		{
			name: "explicit include bypasses size filter",
			cfg: config.AgentConfig{
				DiskIncludes: []string{"/var/log"},
			},
			partitions: []disk.PartitionStat{
				{
					Device:     "tmpfs",
					Mountpoint: "/var/log",
					Fstype:     "tmpfs",
				},
			},
			usageFn: func(mountpoint string) (*disk.UsageStat, error) {
				return &disk.UsageStat{
					Path:        mountpoint,
					Total:       128 * 1024 * 1024,
					Used:        64 * 1024 * 1024,
					Free:        64 * 1024 * 1024,
					UsedPercent: 50,
				}, nil
			},
			expectedMountpoints:      []string{"/var/log"},
			expectedExplicitIncludes: []bool{true},
		},
		{
			name: "dedupes non explicit bind mounts",
			cfg:  config.AgentConfig{},
			partitions: []disk.PartitionStat{
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
			},
			usageFn: func(mountpoint string) (*disk.UsageStat, error) {
				return &disk.UsageStat{
					Path:        mountpoint,
					Total:       32 * 1024 * 1024 * 1024,
					Used:        8 * 1024 * 1024 * 1024,
					Free:        24 * 1024 * 1024 * 1024,
					UsedPercent: 25,
				}, nil
			},
			expectedMountpoints:      []string{"/"},
			expectedExplicitIncludes: []bool{false},
		},
		{
			name: "keeps explicitly included bind mounts",
			cfg: config.AgentConfig{
				DiskIncludes: []string{"/var/hdd.log"},
			},
			partitions: []disk.PartitionStat{
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
			},
			usageFn: func(mountpoint string) (*disk.UsageStat, error) {
				return &disk.UsageStat{
					Path:        mountpoint,
					Total:       32 * 1024 * 1024 * 1024,
					Used:        8 * 1024 * 1024 * 1024,
					Free:        24 * 1024 * 1024 * 1024,
					UsedPercent: 25,
				}, nil
			},
			expectedMountpoints:      []string{"/", "/var/hdd.log"},
			expectedExplicitIncludes: []bool{false, true},
		},
		{
			name: "include wins over exclude precedence",
			cfg: config.AgentConfig{
				DiskIncludes: []string{"/var/hdd.log"},
				DiskExcludes: []string{"/var/hdd.log"},
			},
			partitions: []disk.PartitionStat{
				{
					Device:     "/dev/mmcblk0p2",
					Mountpoint: "/var/hdd.log",
					Fstype:     "ext4",
					Opts:       []string{"rw", "relatime", "bind"},
				},
			},
			usageFn: func(mountpoint string) (*disk.UsageStat, error) {
				return &disk.UsageStat{
					Path:        mountpoint,
					Total:       32 * 1024 * 1024 * 1024,
					Used:        8 * 1024 * 1024 * 1024,
					Free:        24 * 1024 * 1024 * 1024,
					UsedPercent: 25,
				}, nil
			},
			expectedMountpoints:      []string{"/var/hdd.log"},
			expectedExplicitIncludes: []bool{true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent := New(&tt.cfg)

			entries := agent.buildDiskReportEntries(tt.partitions, tt.usageFn)

			require.Len(t, entries, len(tt.expectedMountpoints))
			require.Len(t, tt.expectedExplicitIncludes, len(entries))
			for i, entry := range entries {
				assert.Equal(t, tt.expectedMountpoints[i], entry.Partition.Mountpoint)
				assert.Equal(t, tt.expectedExplicitIncludes[i], entry.ExplicitInclude)
			}
		})
	}
}
