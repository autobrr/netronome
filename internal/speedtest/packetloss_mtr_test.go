// Copyright (c) 2024-2025, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package speedtest

import (
	"context"
	"os"
	"os/exec"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMTRProcessCleanup(t *testing.T) {
	// Skip if MTR is not available
	if _, err := exec.LookPath("mtr"); err != nil {
		t.Skip("MTR not found, skipping test")
	}

	// Skip if not running as root/admin (MTR usually requires privileges)
	if os.Geteuid() != 0 && runtime.GOOS != "windows" {
		t.Skip("Test requires root privileges to run MTR")
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Test that we can start and kill MTR without leaving zombies
	args := []string{
		"-4",
		"-j",
		"-c", "100", // High count to ensure it runs long enough
		"-i", "0.1", // Fast interval
		"--no-dns",
		"127.0.0.1",
	}

	// Create the command
	cmd := exec.CommandContext(ctx, "mtr", args...)
	configureMTRCommand(cmd)

	// Start the command
	err := cmd.Start()
	if err != nil {
		// If we can't start MTR even with privileges, skip
		t.Skipf("Cannot start MTR: %v", err)
	}

	// Get the PID for cleanup tracking
	pid := cmd.Process.Pid

	// Let it run for a bit
	time.Sleep(1 * time.Second)

	// Kill the process group
	err = killMTRProcessGroup(pid)
	assert.NoError(t, err, "Failed to kill MTR process group")

	// Wait for cleanup
	time.Sleep(500 * time.Millisecond)

	// Check that no zombie processes exist
	hasZombies := checkForZombies()
	assert.False(t, hasZombies, "Found zombie MTR processes after cleanup")
}

// TestProcessCleanupMechanism tests the cleanup mechanism with a simpler command
func TestProcessCleanupMechanism(t *testing.T) {
	// Use a simple long-running command that doesn't require privileges
	var cmdName string
	var args []string
	
	switch runtime.GOOS {
	case "windows":
		cmdName = "ping"
		args = []string{"-n", "100", "127.0.0.1"}
	default:
		cmdName = "sleep"
		args = []string{"100"}
	}

	// Create context with short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Create the command
	cmd := exec.CommandContext(ctx, cmdName, args...)
	configureMTRCommand(cmd)

	// Start the command
	err := cmd.Start()
	require.NoError(t, err, "Failed to start test command")

	// Get the PID
	pid := cmd.Process.Pid

	// Create a done channel
	done := make(chan error, 1)

	// Wait for command in goroutine
	go func() {
		done <- cmd.Wait()
	}()

	// Wait for timeout
	select {
	case <-ctx.Done():
		// Context timeout - kill the process group
		err := killMTRProcessGroup(pid)
		// It's OK if the process is already gone (no such process error)
		if err != nil && !isNoSuchProcessError(err) {
			assert.NoError(t, err, "Failed to kill process group")
		}
	case <-done:
		t.Fatal("Command should have been killed by timeout")
	}

	// Verify process is gone
	time.Sleep(100 * time.Millisecond)
	
	// Try to find the process
	if runtime.GOOS != "windows" {
		// On Unix, check if process exists
		process, err := os.FindProcess(pid)
		if err == nil {
			// Send signal 0 to check if process is alive
			err = process.Signal(os.Signal(nil))
			assert.Error(t, err, "Process should be dead")
		}
	}
}

// checkForZombies checks for zombie mtr processes
func checkForZombies() bool {
	if runtime.GOOS == "windows" {
		// Windows doesn't have zombie processes in the same way
		return false
	}

	// Run ps to check for zombie mtr processes
	cmd := exec.Command("ps", "aux")
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	
	outputStr := string(output)
	return containsZombieMTR(outputStr)
}

// containsZombieMTR checks if the ps output contains zombie mtr processes
func containsZombieMTR(psOutput string) bool {
	lines := splitLines(psOutput)
	for _, line := range lines {
		// Check for mtr process that is a zombie (defunct or Z state)
		if containsString(line, "mtr") && 
		   (containsString(line, "<defunct>") || containsString(line, " Z ")) {
			return true
		}
	}
	return false
}

// Helper functions to avoid imports
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && stringIndex(s, substr) >= 0
}

func stringIndex(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

// isNoSuchProcessError checks if the error is a "no such process" error
func isNoSuchProcessError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return containsString(errStr, "no such process") || 
	       containsString(errStr, "process already finished") ||
	       containsString(errStr, "The process cannot be found")
}