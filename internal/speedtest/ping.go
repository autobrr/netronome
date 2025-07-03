// Copyright (c) 2024-2025, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package speedtest

import (
	"context"
	"fmt"
	"os"
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

	// Check for Docker environment indicators
	inDocker := s.isRunningInDocker()

	log.Info().
		Str("host", host).
		Int("count", s.config.IPerf.Ping.Count).
		Int("interval", s.config.IPerf.Ping.Interval).
		Bool("in_docker", inDocker).
		Str("os", runtime.GOOS).
		Msg("Starting ping test")

	// Check if ping is available
	if _, err := exec.LookPath("ping"); err != nil {
		return nil, fmt.Errorf("ping command not found: %w", err)
	}

	// Build ping command based on OS
	args := s.buildPingArgs(host)

	log.Info().
		Str("host", host).
		Strs("args", args).
		Str("os", runtime.GOOS).
		Int("timeout_seconds", s.config.IPerf.Ping.Timeout).
		Msg("Executing ping command")

	// Create a timeout context for the ping command
	timeout := time.Duration(s.config.IPerf.Ping.Timeout) * time.Second
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(timeoutCtx, "ping", args...)

	output, err := cmd.Output()
	if err != nil {
		// Check if the error was due to context timeout
		if timeoutCtx.Err() == context.DeadlineExceeded {
			log.Error().
				Str("host", host).
				Int("timeout_seconds", s.config.IPerf.Ping.Timeout).
				Msg("Ping test timed out")
			return nil, fmt.Errorf("ping test timed out after %d seconds", s.config.IPerf.Ping.Timeout)
		}
		log.Error().Err(err).
			Str("host", host).
			Strs("args", args).
			Msg("Ping command execution failed")
		return nil, fmt.Errorf("ping failed: %w", err)
	}

	log.Info().
		Str("host", host).
		Int("output_length", len(output)).
		Msg("Ping command completed, parsing output")

	// Log the raw output for debugging (truncated if too long)
	outputStr := string(output)
	if len(outputStr) > 1000 {
		log.Debug().
			Str("host", host).
			Str("output_sample", outputStr[:1000]+"...").
			Msg("Ping raw output (truncated)")
	} else {
		log.Debug().
			Str("host", host).
			Str("output", outputStr).
			Msg("Ping raw output")
	}

	// Parse ping output
	result, err := s.parsePingOutput(outputStr, host)
	if err != nil {
		log.Error().Err(err).
			Str("host", host).
			Str("output_sample", outputStr[:min(len(outputStr), 500)]).
			Msg("Failed to parse ping output")
		return nil, fmt.Errorf("failed to parse ping output: %w", err)
	}

	log.Info().
		Str("host", host).
		Float64("avg_rtt", result.AvgRTT).
		Float64("packet_loss", result.PacketLoss).
		Int("packets_sent", result.PacketsSent).
		Int("packets_received", result.PacketsReceived).
		Msg("Ping test completed successfully")

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
		}

		// Add additional flags for Docker environments if needed
		if s.isRunningInDocker() && runtime.GOOS == "linux" {
			// In some Docker configurations, we might need different options
			// Keep the same args for now but log the Docker detection
			log.Info().
				Str("host", host).
				Msg("Detected Docker environment, using standard Linux ping args")
		}

		args = append(args, host)

		log.Debug().
			Str("os", runtime.GOOS).
			Str("host", host).
			Int("count", s.config.IPerf.Ping.Count).
			Int("interval_ms", s.config.IPerf.Ping.Interval).
			Int("timeout_ms", s.config.IPerf.Ping.Timeout*1000).
			Strs("final_args", args).
			Bool("docker_detected", s.isRunningInDocker()).
			Msg("Built ping arguments for Unix/Linux/macOS")
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

	// Look for timing line with stddev/mdev (macOS, some Linux distributions)
	timingWithStddevRegex := regexp.MustCompile(`round-trip min/avg/max/(?:stddev|mdev) = ([\d.]+)/([\d.]+)/([\d.]+)/([\d.]+) ms`)

	// Look for timing line without stddev (Alpine Linux and some minimal distributions)
	timingWithoutStddevRegex := regexp.MustCompile(`round-trip min/avg/max = ([\d.]+)/([\d.]+)/([\d.]+) ms`)

	log.Debug().
		Str("host", result.Host).
		Int("total_lines", len(lines)).
		Msg("Starting to parse Unix ping output")

	for i, line := range lines {
		log.Debug().
			Str("host", result.Host).
			Int("line_number", i+1).
			Str("line_content", line).
			Msg("Processing ping output line")

		if match := statsRegex.FindStringSubmatch(line); match != nil {
			sent, _ := strconv.Atoi(match[1])
			received, _ := strconv.Atoi(match[2])
			loss, _ := strconv.ParseFloat(match[3], 64)

			result.PacketsSent = sent
			result.PacketsReceived = received
			result.PacketLoss = loss

			log.Info().
				Str("host", result.Host).
				Int("packets_sent", sent).
				Int("packets_received", received).
				Float64("packet_loss", loss).
				Str("matched_line", line).
				Msg("Successfully parsed ping statistics")
		}

		// Try matching with stddev first
		if match := timingWithStddevRegex.FindStringSubmatch(line); match != nil {
			min, _ := strconv.ParseFloat(match[1], 64)
			avg, _ := strconv.ParseFloat(match[2], 64)
			max, _ := strconv.ParseFloat(match[3], 64)
			stddev, _ := strconv.ParseFloat(match[4], 64)

			result.MinRTT = min
			result.AvgRTT = avg
			result.MaxRTT = max
			result.StddevRTT = stddev

			log.Info().
				Str("host", result.Host).
				Float64("min_rtt", min).
				Float64("avg_rtt", avg).
				Float64("max_rtt", max).
				Float64("stddev_rtt", stddev).
				Str("matched_line", line).
				Msg("Successfully parsed ping timing statistics with stddev")
		} else if match := timingWithoutStddevRegex.FindStringSubmatch(line); match != nil {
			// Handle Alpine Linux format without stddev
			min, _ := strconv.ParseFloat(match[1], 64)
			avg, _ := strconv.ParseFloat(match[2], 64)
			max, _ := strconv.ParseFloat(match[3], 64)

			result.MinRTT = min
			result.AvgRTT = avg
			result.MaxRTT = max
			result.StddevRTT = 0 // No stddev available in Alpine format

			log.Info().
				Str("host", result.Host).
				Float64("min_rtt", min).
				Float64("avg_rtt", avg).
				Float64("max_rtt", max).
				Str("matched_line", line).
				Msg("Successfully parsed ping timing statistics without stddev (Alpine format)")
		}
	}

	if result.PacketsSent == 0 {
		log.Error().
			Str("host", result.Host).
			Int("total_lines", len(lines)).
			Msg("Failed to parse ping statistics - no packets sent found")
		return nil, fmt.Errorf("could not parse ping statistics")
	}

	log.Info().
		Str("host", result.Host).
		Int("packets_sent", result.PacketsSent).
		Int("packets_received", result.PacketsReceived).
		Float64("packet_loss", result.PacketLoss).
		Float64("avg_rtt", result.AvgRTT).
		Msg("Unix ping parsing completed successfully")

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

// isRunningInDocker checks if the application is running inside a Docker container
func (s *service) isRunningInDocker() bool {
	// Check for .dockerenv file
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return true
	}

	// Check for Docker-specific environment variables
	if os.Getenv("DOCKER_CONTAINER") != "" ||
		os.Getenv("HOSTNAME") != "" && strings.HasPrefix(os.Getenv("HOSTNAME"), "docker-") {
		return true
	}

	// Check for container-specific cgroup entries
	if data, err := os.ReadFile("/proc/1/cgroup"); err == nil {
		content := string(data)
		if strings.Contains(content, "docker") || strings.Contains(content, "containerd") {
			return true
		}
	}

	return false
}
