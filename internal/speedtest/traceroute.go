// Copyright (c) 2024-2025, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package speedtest

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/oschwald/geoip2-golang"
	"github.com/rs/zerolog/log"
)

// Global GeoIP database instances
var (
	countryDB *geoip2.Reader
	asnDB     *geoip2.Reader
)

// Initialize GeoIP database
func (s *service) initGeoIP() {
	// Check if GeoIP is configured and enabled
	if s.fullConfig == nil || (s.fullConfig.GeoIP.CountryDatabasePath == "" && s.fullConfig.GeoIP.ASNDatabasePath == "") {
		log.Info().Msg("GeoIP not configured. Country and ASN detection disabled. Configure [geoip] section in config to enable.")
		return
	}

	// Load Country database if configured
	if s.fullConfig.GeoIP.CountryDatabasePath != "" {
		if db, err := geoip2.Open(s.fullConfig.GeoIP.CountryDatabasePath); err == nil {
			countryDB = db
			log.Info().Str("path", s.fullConfig.GeoIP.CountryDatabasePath).Msg("GeoIP Country database loaded successfully")
		} else {
			log.Warn().Str("path", s.fullConfig.GeoIP.CountryDatabasePath).Err(err).Msg("Failed to load GeoIP Country database")
		}
	}

	// Load ASN database if configured
	if s.fullConfig.GeoIP.ASNDatabasePath != "" {
		if db, err := geoip2.Open(s.fullConfig.GeoIP.ASNDatabasePath); err == nil {
			asnDB = db
			log.Info().Str("path", s.fullConfig.GeoIP.ASNDatabasePath).Msg("GeoIP ASN database loaded successfully")
		} else {
			log.Warn().Str("path", s.fullConfig.GeoIP.ASNDatabasePath).Err(err).Msg("Failed to load GeoIP ASN database")
		}
	}

	if countryDB == nil && asnDB == nil {
		log.Warn().Msg("No GeoIP databases loaded. See README for setup instructions.")
	}
}

// Get country code from IP address
func getCountryFromIP(ip string) string {
	if countryDB == nil {
		return ""
	}

	netIP := net.ParseIP(ip)
	if netIP == nil {
		return ""
	}

	record, err := countryDB.Country(netIP)
	if err != nil {
		return ""
	}

	return record.Country.IsoCode
}

// Get ASN information from IP address
func getASNFromIP(ip string) string {
	if asnDB == nil {
		return ""
	}

	netIP := net.ParseIP(ip)
	if netIP == nil {
		return ""
	}

	record, err := asnDB.ASN(netIP)
	if err != nil {
		return ""
	}

	if record.AutonomousSystemOrganization != "" {
		return fmt.Sprintf("AS%d %s", record.AutonomousSystemNumber, record.AutonomousSystemOrganization)
	}
	return fmt.Sprintf("AS%d", record.AutonomousSystemNumber)
}

// Resolve hostname to IP and get country
func getCountryFromHost(host string) string {
	if countryDB == nil {
		return ""
	}

	// If it's already an IP, use it directly
	if net.ParseIP(host) != nil {
		return getCountryFromIP(host)
	}

	// Resolve hostname to IP
	ips, err := net.LookupIP(host)
	if err != nil || len(ips) == 0 {
		return ""
	}

	// Use the first IP address
	return getCountryFromIP(ips[0].String())
}

// Resolve hostname to IP and get ASN
func getASNFromHost(host string) string {
	if asnDB == nil {
		return ""
	}

	// If it's already an IP, use it directly
	if net.ParseIP(host) != nil {
		return getASNFromIP(host)
	}

	// Resolve hostname to IP
	ips, err := net.LookupIP(host)
	if err != nil || len(ips) == 0 {
		return ""
	}

	// Use the first IP address
	return getASNFromIP(ips[0].String())
}

// TracerouteHop represents a single hop in the traceroute path
type TracerouteHop struct {
	Number      int     `json:"number"`
	Host        string  `json:"host"`
	IP          string  `json:"ip"`
	RTT1        float64 `json:"rtt1"`
	RTT2        float64 `json:"rtt2"`
	RTT3        float64 `json:"rtt3"`
	Timeout     bool    `json:"timeout"`
	AS          string  `json:"as,omitempty"`
	Location    string  `json:"location,omitempty"`
	CountryCode string  `json:"countryCode,omitempty"`
}

// TracerouteResult represents the complete traceroute results
type TracerouteResult struct {
	Destination string          `json:"destination"`
	IP          string          `json:"ip"`
	Hops        []TracerouteHop `json:"hops"`
	TotalHops   int             `json:"totalHops"`
	Complete    bool            `json:"complete"`
}

// RunTraceroute executes a traceroute test against the specified host
func (s *service) RunTraceroute(ctx context.Context, host string) (*TracerouteResult, error) {
	if host == "" {
		return nil, fmt.Errorf("host is required for traceroute test")
	}

	// Initialize GeoIP databases if not already done
	if countryDB == nil && asnDB == nil {
		s.initGeoIP()
	}

	// Extract hostname from URL if it's a full URL
	originalHost := host
	if strings.HasPrefix(host, "http://") || strings.HasPrefix(host, "https://") {
		parsedURL, err := url.Parse(host)
		if err != nil {
			return nil, fmt.Errorf("invalid URL: %w", err)
		}
		host = parsedURL.Hostname()
		if host == "" {
			return nil, fmt.Errorf("could not extract hostname from URL: %s", originalHost)
		}
		log.Info().
			Str("original_host", originalHost).
			Str("extracted_host", host).
			Msg("Extracted hostname from URL for traceroute")
	} else {
		// Strip port from host if present (for non-URL hosts)
		if strings.Contains(host, ":") {
			host = strings.Split(host, ":")[0]
		}
	}

	// Check for Docker environment indicators
	inDocker := s.isRunningInDocker()

	log.Info().
		Str("host", host).
		Bool("in_docker", inDocker).
		Str("os", runtime.GOOS).
		Msg("Starting traceroute test")

	// Check if traceroute command is available
	cmdName := "traceroute"
	if runtime.GOOS == "windows" {
		cmdName = "tracert"
	}

	if _, err := exec.LookPath(cmdName); err != nil {
		return nil, fmt.Errorf("%s command not found: %w", cmdName, err)
	}

	// Build traceroute command based on OS
	args := s.buildTracerouteArgs(host)

	log.Info().
		Str("host", host).
		Strs("args", args).
		Str("os", runtime.GOOS).
		Str("command", cmdName).
		Msg("Executing traceroute command")

	// Create a timeout context for the traceroute command
	timeout := 60 * time.Second // 60 second timeout for traceroute
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(timeoutCtx, cmdName, args...)

	output, err := cmd.Output()
	if err != nil {
		// Check if the error was due to context timeout
		if timeoutCtx.Err() == context.DeadlineExceeded {
			log.Error().
				Str("host", host).
				Msg("Traceroute test timed out")
			return nil, fmt.Errorf("traceroute test timed out after 60 seconds")
		}
		log.Error().Err(err).
			Str("host", host).
			Strs("args", args).
			Msg("Traceroute command execution failed")
		return nil, fmt.Errorf("traceroute failed: %w", err)
	}

	log.Info().
		Str("host", host).
		Int("output_length", len(output)).
		Msg("Traceroute command completed, parsing output")

	// Log the raw output for debugging (truncated if too long)
	outputStr := string(output)
	if len(outputStr) > 2000 {
		log.Debug().
			Str("host", host).
			Str("output_sample", outputStr[:2000]+"...").
			Msg("Traceroute raw output (truncated)")
	} else {
		log.Debug().
			Str("host", host).
			Str("output", outputStr).
			Msg("Traceroute raw output")
	}

	// Parse traceroute output
	result, err := s.parseTracerouteOutput(outputStr, originalHost)
	if err != nil {
		log.Error().Err(err).
			Str("host", host).
			Str("output_sample", outputStr[:min(len(outputStr), 500)]).
			Msg("Failed to parse traceroute output")
		return nil, fmt.Errorf("failed to parse traceroute output: %w", err)
	}

	log.Info().
		Str("host", host).
		Int("total_hops", result.TotalHops).
		Bool("complete", result.Complete).
		Msg("Traceroute test completed successfully")

	return result, nil
}

// buildTracerouteArgs builds traceroute command arguments based on the operating system
func (s *service) buildTracerouteArgs(host string) []string {
	var args []string

	switch runtime.GOOS {
	case "darwin", "linux":
		args = []string{
			"-w", "5", // Wait 5 seconds for response (more reliable)
			"-m", "30", // Max 30 hops
			"-q", "3", // 3 queries per hop for better statistics
			host,
		}

		log.Debug().
			Str("os", runtime.GOOS).
			Str("host", host).
			Strs("final_args", args).
			Bool("docker_detected", s.isRunningInDocker()).
			Msg("Built traceroute arguments for Unix/Linux/macOS")
	case "windows":
		args = []string{
			"-w", "5000", // Wait 5000 milliseconds for response (more reliable)
			"-h", "30", // Max 30 hops
			host,
		}

		log.Debug().
			Str("os", runtime.GOOS).
			Str("host", host).
			Strs("final_args", args).
			Msg("Built traceroute arguments for Windows")
	default:
		// Default to Linux/Unix style
		args = []string{
			"-w", "5",
			"-m", "30",
			"-q", "3",
			host,
		}
	}

	return args
}

// parseTracerouteOutput parses traceroute command output based on the operating system
func (s *service) parseTracerouteOutput(output, originalHost string) (*TracerouteResult, error) {
	result := &TracerouteResult{
		Destination: originalHost, // Use the original host/URL for display
		Hops:        []TracerouteHop{},
	}

	lines := strings.Split(output, "\n")

	switch runtime.GOOS {
	case "darwin", "linux":
		return s.parseUnixTracerouteOutput(lines, result)
	case "windows":
		return s.parseWindowsTracerouteOutput(lines, result)
	default:
		return s.parseUnixTracerouteOutput(lines, result)
	}
}

// parseUnixTracerouteOutput parses Unix/Linux/macOS traceroute output
func (s *service) parseUnixTracerouteOutput(lines []string, result *TracerouteResult) (*TracerouteResult, error) {
	// Unix traceroute format:
	// traceroute to google.com (172.217.14.110), 30 hops max, 60 byte packets
	//  1  192.168.1.1  0.123 ms  0.456 ms  0.789 ms
	//  2  10.0.0.1  5.123 ms  5.456 ms  5.789 ms
	//  3  * * *
	//  4  172.217.14.110  15.123 ms  15.456 ms  15.789 ms

	// Extract destination IP from first line
	if len(lines) > 0 {
		firstLine := lines[0]
		if strings.Contains(firstLine, "traceroute to") {
			// Extract IP from parentheses
			ipRegex := regexp.MustCompile(`\(([^)]+)\)`)
			if match := ipRegex.FindStringSubmatch(firstLine); match != nil {
				result.IP = match[1]
			}
		}
	}

	// Regex patterns for parsing hop lines
	// Updated to handle both hostname and IP, or just IP
	// Updated regex patterns for single query per hop
	hopRegex := regexp.MustCompile(`^\s*(\d+)\s+([^\s]+)\s+\(([^)]+)\)\s+([\d.]+)\s+ms`)
	hopRegexIPOnly := regexp.MustCompile(`^\s*(\d+)\s+([^\s]+)\s+([\d.]+)\s+ms`)
	// Also support the old 3-query format for backward compatibility
	hopRegex3 := regexp.MustCompile(`^\s*(\d+)\s+([^\s]+)\s+\(([^)]+)\)\s+([\d.]+)\s+ms\s+([\d.]+)\s+ms\s+([\d.]+)\s+ms`)
	hopRegexIPOnly3 := regexp.MustCompile(`^\s*(\d+)\s+([^\s]+)\s+([\d.]+)\s+ms\s+([\d.]+)\s+ms\s+([\d.]+)\s+ms`)
	timeoutRegex := regexp.MustCompile(`^\s*(\d+)\s+\*\s+\*\s+\*`)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "traceroute to") {
			continue
		}

		// Try to match timeout line first
		if match := timeoutRegex.FindStringSubmatch(line); match != nil {
			hopNum, _ := strconv.Atoi(match[1])
			hop := TracerouteHop{
				Number:      hopNum,
				Host:        "*",
				IP:          "*",
				Timeout:     true,
				CountryCode: "",
				AS:          "",
			}
			result.Hops = append(result.Hops, hop)
			continue
		}

		// Try to match hop line with hostname and IP first (3-query format)
		if match := hopRegex3.FindStringSubmatch(line); match != nil {
			hopNum, _ := strconv.Atoi(match[1])
			hostname := match[2]
			ip := match[3]
			rtt1, _ := strconv.ParseFloat(match[4], 64)
			rtt2, _ := strconv.ParseFloat(match[5], 64)
			rtt3, _ := strconv.ParseFloat(match[6], 64)

			hop := TracerouteHop{
				Number:      hopNum,
				Host:        hostname,
				IP:          ip,
				RTT1:        rtt1,
				RTT2:        rtt2,
				RTT3:        rtt3,
				Timeout:     false,
				CountryCode: getCountryFromHost(ip),
				AS:          getASNFromHost(ip),
			}
			result.Hops = append(result.Hops, hop)
		} else if match := hopRegexIPOnly3.FindStringSubmatch(line); match != nil {
			// Try to match hop line with IP only (3-query format)
			hopNum, _ := strconv.Atoi(match[1])
			ip := match[2]
			rtt1, _ := strconv.ParseFloat(match[3], 64)
			rtt2, _ := strconv.ParseFloat(match[4], 64)
			rtt3, _ := strconv.ParseFloat(match[5], 64)

			hop := TracerouteHop{
				Number:      hopNum,
				Host:        ip,
				IP:          ip,
				RTT1:        rtt1,
				RTT2:        rtt2,
				RTT3:        rtt3,
				Timeout:     false,
				CountryCode: getCountryFromHost(ip),
				AS:          getASNFromHost(ip),
			}
			result.Hops = append(result.Hops, hop)
		} else if match := hopRegex.FindStringSubmatch(line); match != nil {
			// Try to match hop line with hostname and IP (single query format)
			hopNum, _ := strconv.Atoi(match[1])
			hostname := match[2]
			ip := match[3]
			rtt1, _ := strconv.ParseFloat(match[4], 64)

			hop := TracerouteHop{
				Number:      hopNum,
				Host:        hostname,
				IP:          ip,
				RTT1:        rtt1,
				RTT2:        0, // No second query
				RTT3:        0, // No third query
				Timeout:     false,
				CountryCode: getCountryFromHost(ip),
				AS:          getASNFromHost(ip),
			}
			result.Hops = append(result.Hops, hop)
		} else if match := hopRegexIPOnly.FindStringSubmatch(line); match != nil {
			// Try to match hop line with IP only (single query format)
			hopNum, _ := strconv.Atoi(match[1])
			ip := match[2]
			rtt1, _ := strconv.ParseFloat(match[3], 64)

			hop := TracerouteHop{
				Number:      hopNum,
				Host:        ip,
				IP:          ip,
				RTT1:        rtt1,
				RTT2:        0, // No second query
				RTT3:        0, // No third query
				Timeout:     false,
				CountryCode: getCountryFromHost(ip),
				AS:          getASNFromHost(ip),
			}
			result.Hops = append(result.Hops, hop)
		}
	}

	result.TotalHops = len(result.Hops)
	result.Complete = result.TotalHops > 0

	return result, nil
}

// parseWindowsTracerouteOutput parses Windows tracert output
func (s *service) parseWindowsTracerouteOutput(lines []string, result *TracerouteResult) (*TracerouteResult, error) {
	// Windows tracert format:
	// Tracing route to google.com [172.217.14.110]
	// over a maximum of 30 hops:
	//
	//   1    <1 ms    <1 ms    <1 ms  192.168.1.1
	//   2     5 ms     5 ms     5 ms  10.0.0.1
	//   3     *        *        *     Request timed out.
	//   4    15 ms    15 ms    15 ms  172.217.14.110

	// Extract destination IP from first line
	if len(lines) > 0 {
		firstLine := lines[0]
		if strings.Contains(firstLine, "Tracing route to") {
			// Extract IP from brackets
			ipRegex := regexp.MustCompile(`\[([^\]]+)\]`)
			if match := ipRegex.FindStringSubmatch(firstLine); match != nil {
				result.IP = match[1]
			}
		}
	}

	// Regex patterns for parsing hop lines
	hopRegex := regexp.MustCompile(`^\s*(\d+)\s+(<?\d+)\s+ms\s+(<?\d+)\s+ms\s+(<?\d+)\s+ms\s+(.+)`)
	timeoutRegex := regexp.MustCompile(`^\s*(\d+)\s+\*\s+\*\s+\*\s+Request timed out\.`)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "Tracing route") || strings.HasPrefix(line, "over a maximum") {
			continue
		}

		// Try to match timeout line first
		if match := timeoutRegex.FindStringSubmatch(line); match != nil {
			hopNum, _ := strconv.Atoi(match[1])
			hop := TracerouteHop{
				Number:      hopNum,
				Host:        "*",
				IP:          "*",
				Timeout:     true,
				CountryCode: "",
				AS:          "",
			}
			result.Hops = append(result.Hops, hop)
			continue
		}

		// Try to match normal hop line
		if match := hopRegex.FindStringSubmatch(line); match != nil {
			hopNum, _ := strconv.Atoi(match[1])
			rtt1Str := strings.TrimPrefix(match[2], "<")
			rtt2Str := strings.TrimPrefix(match[3], "<")
			rtt3Str := strings.TrimPrefix(match[4], "<")
			ip := strings.TrimSpace(match[5])

			rtt1, _ := strconv.ParseFloat(rtt1Str, 64)
			rtt2, _ := strconv.ParseFloat(rtt2Str, 64)
			rtt3, _ := strconv.ParseFloat(rtt3Str, 64)

			hop := TracerouteHop{
				Number:      hopNum,
				Host:        ip,
				IP:          ip,
				RTT1:        rtt1,
				RTT2:        rtt2,
				RTT3:        rtt3,
				Timeout:     false,
				CountryCode: getCountryFromHost(ip),
				AS:          getASNFromHost(ip),
			}
			result.Hops = append(result.Hops, hop)
		}
	}

	result.TotalHops = len(result.Hops)
	result.Complete = result.TotalHops > 0

	return result, nil
}
