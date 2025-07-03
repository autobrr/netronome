// Copyright (c) 2024-2025, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package speedtest

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
)

// PingResult represents the parsed output from ping command
type PingResult struct {
	Host            string
	PacketsSent     int
	PacketsReceived int
	PacketLoss      float64
	MinRTT          float64
	AvgRTT          float64
	MaxRTT          float64
	StddevRTT       float64
}

// RunPingTest executes a ping test against the specified host
func (s *service) RunPingTest(ctx context.Context, host string) (*PingResult, error) {
	if host == "" {
		return nil, fmt.Errorf("host is required for ping test")
	}

	// Strip port from host if present
	if strings.Contains(host, ":") {
		host = strings.Split(host, ":")[0]
	}

	log.Debug().
		Str("host", host).
		Int("count", s.config.IPerf.Ping.Count).
		Int("interval", s.config.IPerf.Ping.Interval).
		Msg("Starting ping test")

	// Check if ping is available
	if _, err := exec.LookPath("ping"); err != nil {
		return nil, fmt.Errorf("ping command not found: %w", err)
	}

	// Build ping command based on OS
	args := s.buildPingArgs(host)

	// Create a timeout context for the ping command
	timeout := time.Duration(s.config.IPerf.Ping.Timeout) * time.Second
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(timeoutCtx, "ping", args...)

	output, err := cmd.Output()
	if err != nil {
		// Check if the error was due to context timeout
		if timeoutCtx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("ping test timed out after %d seconds", s.config.IPerf.Ping.Timeout)
		}
		return nil, fmt.Errorf("ping failed: %w", err)
	}

	// Parse ping output
	result, err := s.parsePingOutput(string(output), host)
	if err != nil {
		return nil, fmt.Errorf("failed to parse ping output: %w", err)
	}

	log.Debug().
		Str("host", host).
		Float64("avg_rtt", result.AvgRTT).
		Float64("packet_loss", result.PacketLoss).
		Msg("Ping test completed")

	return result, nil
}

// buildPingArgs builds ping command arguments based on the operating system
func (s *service) buildPingArgs(host string) []string {
	var args []string

	switch runtime.GOOS {
	case "darwin", "linux":
		args = []string{
			"-c", strconv.Itoa(s.config.IPerf.Ping.Count), // packet count
			"-i", fmt.Sprintf("%.1f", float64(s.config.IPerf.Ping.Interval)/1000), // interval in seconds
			"-W", strconv.Itoa(s.config.IPerf.Ping.Timeout * 1000), // timeout in milliseconds
			host,
		}
	case "windows":
		args = []string{
			"-n", strconv.Itoa(s.config.IPerf.Ping.Count), // packet count
			"-l", "32", // packet size
			"-w", strconv.Itoa(s.config.IPerf.Ping.Timeout * 1000), // timeout in milliseconds
			host,
		}
	default:
		// Default to Linux/Unix style
		args = []string{
			"-c", strconv.Itoa(s.config.IPerf.Ping.Count),
			"-i", fmt.Sprintf("%.1f", float64(s.config.IPerf.Ping.Interval)/1000),
			"-W", strconv.Itoa(s.config.IPerf.Ping.Timeout * 1000),
			host,
		}
	}

	return args
}

// parsePingOutput parses ping command output based on the operating system
func (s *service) parsePingOutput(output, host string) (*PingResult, error) {
	result := &PingResult{
		Host: host,
	}

	lines := strings.Split(output, "\n")

	switch runtime.GOOS {
	case "darwin", "linux":
		return s.parseUnixPingOutput(lines, result)
	case "windows":
		return s.parseWindowsPingOutput(lines, result)
	default:
		return s.parseUnixPingOutput(lines, result)
	}
}

// parseUnixPingOutput parses Unix/Linux/macOS ping output
func (s *service) parseUnixPingOutput(lines []string, result *PingResult) (*PingResult, error) {
	// Look for statistics line (e.g., "5 packets transmitted, 5 received, 0% packet loss")
	statsRegex := regexp.MustCompile(`(\d+) packets transmitted, (\d+) (?:packets )?received, (?:\+\d+ errors, )?(\d+(?:\.\d+)?)% packet loss`)

	// Look for timing line - handle both formats:
	// "round-trip min/avg/max/stddev = 1.234/2.345/3.456/0.789 ms" (Linux)
	// "round-trip min/avg/max = 1.234/2.345/3.456 ms" (Alpine)
	timingRegexWithStddev := regexp.MustCompile(`round-trip min/avg/max/(?:stddev|mdev) = ([\d.]+)/([\d.]+)/([\d.]+)/([\d.]+) ms`)
	timingRegexSimple := regexp.MustCompile(`round-trip min/avg/max = ([\d.]+)/([\d.]+)/([\d.]+) ms`)

	for _, line := range lines {
		if match := statsRegex.FindStringSubmatch(line); match != nil {
			sent, _ := strconv.Atoi(match[1])
			received, _ := strconv.Atoi(match[2])
			loss, _ := strconv.ParseFloat(match[3], 64)

			result.PacketsSent = sent
			result.PacketsReceived = received
			result.PacketLoss = loss
		}

		// Try format with stddev first
		if match := timingRegexWithStddev.FindStringSubmatch(line); match != nil {
			min, _ := strconv.ParseFloat(match[1], 64)
			avg, _ := strconv.ParseFloat(match[2], 64)
			max, _ := strconv.ParseFloat(match[3], 64)
			stddev, _ := strconv.ParseFloat(match[4], 64)

			result.MinRTT = min
			result.AvgRTT = avg
			result.MaxRTT = max
			result.StddevRTT = stddev
		} else if match := timingRegexSimple.FindStringSubmatch(line); match != nil {
			// Alpine format without stddev
			min, _ := strconv.ParseFloat(match[1], 64)
			avg, _ := strconv.ParseFloat(match[2], 64)
			max, _ := strconv.ParseFloat(match[3], 64)

			result.MinRTT = min
			result.AvgRTT = avg
			result.MaxRTT = max
			result.StddevRTT = 0 // Not available in Alpine format
		}
	}

	if result.PacketsSent == 0 {
		return nil, fmt.Errorf("could not parse ping statistics")
	}

	return result, nil
}

// parseWindowsPingOutput parses Windows ping output
func (s *service) parseWindowsPingOutput(lines []string, result *PingResult) (*PingResult, error) {
	// Windows ping statistics format:
	// "Packets: Sent = 4, Received = 4, Lost = 0 (0% loss)"
	// "Approximate round trip times in milli-seconds:"
	// "Minimum = 1ms, Maximum = 4ms, Average = 2ms"

	statsRegex := regexp.MustCompile(`Packets: Sent = (\d+), Received = (\d+), Lost = \d+ \((\d+(?:\.\d+)?)% loss\)`)
	timingRegex := regexp.MustCompile(`Minimum = (\d+)ms, Maximum = (\d+)ms, Average = (\d+)ms`)

	for _, line := range lines {
		if match := statsRegex.FindStringSubmatch(line); match != nil {
			sent, _ := strconv.Atoi(match[1])
			received, _ := strconv.Atoi(match[2])
			loss, _ := strconv.ParseFloat(match[3], 64)

			result.PacketsSent = sent
			result.PacketsReceived = received
			result.PacketLoss = loss
		}

		if match := timingRegex.FindStringSubmatch(line); match != nil {
			min, _ := strconv.ParseFloat(match[1], 64)
			max, _ := strconv.ParseFloat(match[2], 64)
			avg, _ := strconv.ParseFloat(match[3], 64)

			result.MinRTT = min
			result.AvgRTT = avg
			result.MaxRTT = max
			// Windows doesn't provide stddev
			result.StddevRTT = 0
		}
	}

	if result.PacketsSent == 0 {
		return nil, fmt.Errorf("could not parse ping statistics")
	}

	return result, nil
}

// formatLatency formats the average RTT as a string for storage
func (result *PingResult) FormatLatency() string {
	if result.AvgRTT == 0 {
		return ""
	}
	return fmt.Sprintf("%.2f ms", result.AvgRTT)
}
