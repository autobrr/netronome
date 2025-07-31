// Copyright (c) 2024-2025, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

//go:build windows

package speedtest

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

// configureMTRCommand sets Windows-specific attributes for the MTR command
func configureMTRCommand(cmd *exec.Cmd) {
	// On Windows, we use CREATE_NEW_PROCESS_GROUP to create a new process group
	// This allows us to send signals to the entire group
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP,
	}
}

// killMTRProcessGroup kills the process for the MTR command on Windows
func killMTRProcessGroup(pid int) error {
	// On Windows, we can use taskkill to kill the process tree
	// The /T flag kills the process and all its children
	// The /F flag forces termination
	killCmd := exec.Command("taskkill", "/T", "/F", "/PID", fmt.Sprintf("%d", pid))
	err := killCmd.Run()
	
	// If taskkill fails, fall back to os.Process.Kill
	if err != nil {
		process, findErr := os.FindProcess(pid)
		if findErr != nil {
			return findErr
		}
		return process.Kill()
	}
	
	return nil
}