// Copyright (c) 2024-2025, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

//go:build !windows

package speedtest

import (
	"os/exec"
	"syscall"
)

// configureMTRCommand sets Unix-specific attributes for the MTR command
func configureMTRCommand(cmd *exec.Cmd) {
	// Set process group to ensure child processes are killed
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}
}

// killMTRProcessGroup kills the process group for the MTR command
func killMTRProcessGroup(pid int) error {
	// Send SIGKILL to the process group (negative PID)
	return syscall.Kill(-pid, syscall.SIGKILL)
}