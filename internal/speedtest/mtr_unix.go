// Copyright (c) 2024-2026, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

//go:build !windows

package speedtest

import (
	"fmt"
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

// buildMTRArgs builds Unix-specific MTR arguments
// On Unix, we can use the -j flag for JSON output directly
func buildMTRArgs(host string, packetCount int, privilegedMode bool, enableDNS bool) ([]string, string, error) {
	args := []string{
		"-4",                                 // Force IPv4
		"-j",                                 // JSON output
		"-c", fmt.Sprintf("%d", packetCount), // Number of cycles
		"-i", "1",                            // 1 second interval
	}

	if !enableDNS {
		args = append(args, "--no-dns")
	}

	args = append(args, host)

	// Add UDP mode if not privileged
	if !privilegedMode {
		args = append([]string{"-u"}, args...)
	}

	// Return empty string to indicate Unix doesn't need parsing
	return args, "", nil
}

// parseMTROutput parses Unix MTR JSON output (no-op since it's already JSON)
func parseMTROutput(outputData []byte, host string) ([]byte, error) {
	// This should never be called on Unix since we get JSON directly
	// But we provide it for interface consistency
	return nil, fmt.Errorf("parseMTROutput should not be called on Unix systems")
}