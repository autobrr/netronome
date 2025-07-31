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
	// This is important even in containers to manage child processes
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}
}

// killMTRProcessGroup kills the process group for the MTR command
func killMTRProcessGroup(pid int) error {
	// Send SIGKILL to the process group (negative PID)
	err := syscall.Kill(-pid, syscall.SIGKILL)
	
	// If the process doesn't exist, that's fine - it already exited
	if err == syscall.ESRCH {
		return nil
	}
	
	// On macOS, we might get EPERM if the process already exited
	if err == syscall.EPERM {
		// Check if the process still exists
		if checkErr := syscall.Kill(-pid, 0); checkErr == syscall.ESRCH {
			// Process doesn't exist, so it's fine
			return nil
		}
	}
	
	return err
}