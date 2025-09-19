// Copyright (c) 2024-2025, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

//go:build windows

package speedtest

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
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
	// First, try to check if the process still exists
	process, findErr := os.FindProcess(pid)
	if findErr != nil {
		// Process doesn't exist, which is fine
		return nil
	}

	// Try to kill the process directly first (gentle approach)
	if err := process.Kill(); err == nil {
		return nil
	}

	// If direct kill fails, try taskkill with process tree termination
	// The /T flag kills the process and all its children
	// The /F flag forces termination
	killCmd := exec.Command("taskkill", "/T", "/F", "/PID", fmt.Sprintf("%d", pid))
	killCmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow: true, // Hide the taskkill window
	}

	err := killCmd.Run()
	if err != nil {
		// Check if the error is because the process doesn't exist
		if exitErr, ok := err.(*exec.ExitError); ok {
			// taskkill exit code 128 means "process not found"
			if exitErr.ExitCode() == 128 {
				return nil // Process already gone, that's fine
			}
		}
		// For other errors, just log and continue - the process might have exited naturally
		return fmt.Errorf("taskkill failed (process may have already exited): %w", err)
	}

	return nil
}

// buildMTRArgs builds Windows-specific MTR arguments
// Windows MTR doesn't support -j for JSON output, so we use -r for report mode
// and -w for wide format, then capture stdout and parse it
func buildMTRArgs(host string, packetCount int, privilegedMode bool) ([]string, string, error) {
	args := []string{
		"-4",                                 // Force IPv4
		"-r",                                 // Report mode
		"-w",                                 // Wide report, don't truncate hostnames
		"-n",                                 // No DNS lookups (numeric output only)
		"-c", fmt.Sprintf("%d", packetCount), // Number of cycles
		"-i", "1", // 1 second interval
		"-t", "2", // 2 second timeout per hop to prevent hanging on unresponsive hops
	}

	// Add UDP mode if not privileged (Windows MTR defaults to ICMP in privileged mode)
	if !privilegedMode {
		args = append([]string{"-u"}, args...)
	}

	// Add target host
	args = append(args, host)

	// Return "windows" as a flag to indicate this platform needs text parsing
	return args, "windows", nil
}

// parseMTROutput parses Windows MTR text output and converts it to JSON format
func parseMTROutput(outputData []byte, host string) ([]byte, error) {
	scanner := bufio.NewScanner(strings.NewReader(string(outputData)))
	var hops []mtrHopData
	var srcIP string

	// Regular expression to parse MTR output lines
	// Format: 1.|-- 192.168.1.1    0.0%   10   1.2   1.5   1.2   2.1   0.3
	hopRegex := regexp.MustCompile(`^\s*(\d+)\.\|\s*--\s+(\S+)\s+(\d+\.\d+)%\s+(\d+)\s+(\d+(?:\.\d+)?)\s+(\d+(?:\.\d+)?)\s+(\d+(?:\.\d+)?)\s+(\d+(?:\.\d+)?)\s+(\d+(?:\.\d+)?)`)

	// Look for HOST line to extract source IP
	hostRegex := regexp.MustCompile(`^HOST:\s+(\S+)`)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		// Extract source hostname/IP from HOST line
		if hostMatch := hostRegex.FindStringSubmatch(line); hostMatch != nil {
			srcIP = hostMatch[1]
			continue
		}

		// Parse hop lines
		if matches := hopRegex.FindStringSubmatch(line); matches != nil {
			hopNum, _ := strconv.Atoi(matches[1])
			hopHost := matches[2]
			loss, _ := strconv.ParseFloat(matches[3], 64)
			snt, _ := strconv.Atoi(matches[4])
			last, _ := strconv.ParseFloat(matches[5], 64)
			avg, _ := strconv.ParseFloat(matches[6], 64)
			best, _ := strconv.ParseFloat(matches[7], 64)
			wrst, _ := strconv.ParseFloat(matches[8], 64)
			stdev, _ := strconv.ParseFloat(matches[9], 64)

			hop := mtrHopData{
				Count: hopNum,
				Host:  hopHost,
				Loss:  loss,
				Snt:   snt,
				Last:  last,
				Avg:   avg,
				Best:  best,
				Wrst:  wrst,
				StDev: stdev,
			}
			hops = append(hops, hop)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read MTR output: %w", err)
	}

	if len(hops) == 0 {
		return nil, fmt.Errorf("no valid MTR hops found in output")
	}

	// If we couldn't extract source IP, use localhost
	if srcIP == "" {
		srcIP = "localhost"
	}

	// Build MTR report structure matching the existing format
	report := mtrReport{
		Report: struct {
			MTR struct {
				Src   string `json:"src"`
				Dst   string `json:"dst"`
				Tests int    `json:"tests"`
			} `json:"mtr"`
			Hubs []struct {
				Count int     `json:"count"`
				Host  string  `json:"host"`
				Loss  float64 `json:"Loss%"`
				Snt   int     `json:"Snt"`
				Last  float64 `json:"Last"`
				Avg   float64 `json:"Avg"`
				Best  float64 `json:"Best"`
				Wrst  float64 `json:"Wrst"`
				StDev float64 `json:"StDev"`
			} `json:"hubs"`
		}{
			MTR: struct {
				Src   string `json:"src"`
				Dst   string `json:"dst"`
				Tests int    `json:"tests"`
			}{
				Src:   srcIP,
				Dst:   host,
				Tests: len(hops),
			},
			Hubs: make([]struct {
				Count int     `json:"count"`
				Host  string  `json:"host"`
				Loss  float64 `json:"Loss%"`
				Snt   int     `json:"Snt"`
				Last  float64 `json:"Last"`
				Avg   float64 `json:"Avg"`
				Best  float64 `json:"Best"`
				Wrst  float64 `json:"Wrst"`
				StDev float64 `json:"StDev"`
			}, len(hops)),
		},
	}

	// Copy hop data to the report structure
	for i, hop := range hops {
		report.Report.Hubs[i] = struct {
			Count int     `json:"count"`
			Host  string  `json:"host"`
			Loss  float64 `json:"Loss%"`
			Snt   int     `json:"Snt"`
			Last  float64 `json:"Last"`
			Avg   float64 `json:"Avg"`
			Best  float64 `json:"Best"`
			Wrst  float64 `json:"Wrst"`
			StDev float64 `json:"StDev"`
		}{
			Count: hop.Count,
			Host:  hop.Host,
			Loss:  hop.Loss,
			Snt:   hop.Snt,
			Last:  hop.Last,
			Avg:   hop.Avg,
			Best:  hop.Best,
			Wrst:  hop.Wrst,
			StDev: hop.StDev,
		}
	}

	return json.Marshal(report)
}

// mtrHopData represents a single hop in the parsed MTR output
type mtrHopData struct {
	Count int
	Host  string
	Loss  float64
	Snt   int
	Last  float64
	Avg   float64
	Best  float64
	Wrst  float64
	StDev float64
}
