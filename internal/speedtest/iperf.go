// Copyright (c) 2024-2025, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package speedtest

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os/exec"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/autobrr/netronome/internal/config"
	"github.com/autobrr/netronome/internal/types"
)

// IperfResult represents the parsed output from iperf3
type IperfResult struct {
	Start struct {
		Connected []struct {
			RemoteHost string `json:"remote_host"`
		} `json:"connected"`
	} `json:"start"`
	End struct {
		SumSent struct {
			BitsPerSecond float64 `json:"bits_per_second"`
			JitterMs      float64 `json:"jitter_ms"`
		} `json:"sum_sent"`
		SumReceived struct {
			BitsPerSecond float64 `json:"bits_per_second"`
			JitterMs      float64 `json:"jitter_ms"`
		} `json:"sum_received"`
	} `json:"end"`
	Error string `json:"error,omitempty"`
}

type IperfRunner struct {
	config           config.IperfConfig
	progressCallback func(types.SpeedUpdate)
	pingResult       *PingResult
}

func NewIperfRunner(cfg config.IperfConfig) *IperfRunner {
	return &IperfRunner{
		config: cfg,
	}
}

func (r *IperfRunner) GetTestType() string {
	return "iperf3"
}

func (r *IperfRunner) SetProgressCallback(callback func(types.SpeedUpdate)) {
	r.progressCallback = callback
}

func (r *IperfRunner) SetPingResult(pingResult *PingResult) {
	r.pingResult = pingResult
}

func (r *IperfRunner) GetServers() ([]ServerResponse, error) {
	return []ServerResponse{}, nil
}

func (r *IperfRunner) RunTest(ctx context.Context, opts *types.TestOptions) (*Result, error) {
	if opts.ServerHost == "" {
		return nil, fmt.Errorf("server host is required for iperf3 test")
	}

	log.Info().
		Str("server_host", opts.ServerHost).
		Bool("enable_download", opts.EnableDownload).
		Bool("enable_upload", opts.EnableUpload).
		Bool("enable_jitter", opts.EnableJitter).
		Msg("Starting iperf3 test")

	var downloadSpeed, uploadSpeed float64
	var jitterMs *float64
	var latency string = "0ms"

	// Use ping results if available
	if r.pingResult != nil {
		latency = r.pingResult.FormatLatency()
	}

	if opts.EnableDownload {
		downloadOpts := *opts
		downloadOpts.EnableDownload = true
		downloadOpts.EnableUpload = false

		downloadResult, err := r.runSingleIperfTest(ctx, &downloadOpts)
		if err != nil {
			return nil, fmt.Errorf("download test failed: %w", err)
		}
		downloadSpeed = downloadResult.DownloadSpeed
		if downloadResult.Jitter != nil {
			jitterMs = downloadResult.Jitter
		}
	}

	time.Sleep(2 * time.Second)

	if opts.EnableUpload {
		uploadOpts := *opts
		uploadOpts.EnableDownload = false
		uploadOpts.EnableUpload = true

		uploadResult, err := r.runSingleIperfTest(ctx, &uploadOpts)
		if err != nil {
			return nil, fmt.Errorf("upload test failed: %w", err)
		}
		uploadSpeed = uploadResult.UploadSpeed
	}

	// Skip jitter test for iperf3 - it's unreliable and causes timeouts
	// Users still get latency from ping which is more useful

	var jitterFloat float64
	if jitterMs != nil {
		jitterFloat = *jitterMs
	}

	// Use server name if provided, otherwise use the host:port
	serverName := opts.ServerName
	if serverName == "" {
		serverName = opts.ServerHost
	}

	result := &Result{
		Timestamp:     time.Now(),
		Server:        serverName,
		DownloadSpeed: downloadSpeed,
		UploadSpeed:   uploadSpeed,
		Latency:       latency,
		Jitter:        jitterFloat,
		Download:      downloadSpeed,
		Upload:        uploadSpeed,
	}

	return result, nil
}

// runSingleIperfTest executes a single iperf3 test (download OR upload)
func (r *IperfRunner) runSingleIperfTest(ctx context.Context, opts *types.TestOptions) (*types.SpeedTestResult, error) {
	if opts.ServerHost == "" {
		return nil, fmt.Errorf("server host is required for iperf3 test")
	}

	// Split host and port if port is included
	host := opts.ServerHost
	port := "5201" // Default to 5201 since we know it works
	if strings.Contains(host, ":") {
		parts := strings.Split(host, ":")
		host = parts[0]
		if len(parts) > 1 {
			port = parts[1]
		}
	}

	log.Debug().
		Str("host", host).
		Str("port", port).
		Bool("download", opts.EnableDownload).
		Msg("Starting iperf3 test")

	args := []string{
		"-c", host,
		"-p", port,
		"--json-stream", // Streaming JSON output for real-time updates
		"-i", "1",       // 1-second interval to match speedtest.net consistency
		"-t", strconv.Itoa(r.config.TestDuration), // Test duration in seconds
		"-P", strconv.Itoa(r.config.ParallelConns), // Number of parallel connections
		"--format", "m", // Force Mbps output
	}

	// Always use TCP mode for iperf3 speed tests
	// UDP mode with bandwidth limits gives misleading results

	if opts.EnableDownload {
		args = append(args, "-R")
	}

	testType := "upload"
	if opts.EnableDownload {
		testType = "download"
	}

	// Use server name if provided, otherwise use the host:port
	serverName := opts.ServerName
	if serverName == "" {
		serverName = fmt.Sprintf("%s:%s", host, port)
	}

	// Send initial status
	if r.progressCallback != nil {
		r.progressCallback(types.SpeedUpdate{
			Type:       testType,
			ServerName: serverName,
			Speed:      0,
			Progress:   0,
			IsComplete: false,
			TestType:   "iperf3",
		})
	}

	// Check if iperf3 is installed
	if _, err := exec.LookPath("iperf3"); err != nil {
		return nil, fmt.Errorf("iperf3 not found: please install iperf3 to use this feature")
	}

	// Create a timeout context for the iperf3 command
	timeout := time.Duration(r.config.Timeout) * time.Second
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(timeoutCtx, "iperf3", args...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	var output strings.Builder
	startTime := time.Now()
	totalDuration := time.Duration(r.config.TestDuration) * time.Second
	var lastUpdate atomic.Int64

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start iperf3: %w", err)
	}

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		output.WriteString(line + "\n")

		// Try to parse JSON streaming format for progress updates
		var streamData struct {
			Event string `json:"event"`
			Data  struct {
				Sum struct {
					Bits_per_second float64 `json:"bits_per_second"`
				} `json:"sum"`
			} `json:"data"`
		}

		if err := json.Unmarshal([]byte(line), &streamData); err == nil {
			// Only process interval events for real-time progress
			if streamData.Event == "interval" {
				// Apply the same 1-second throttling as speedtest.net
				now := time.Now().Unix()
				lastUpdateTime := lastUpdate.Load()

				if now-lastUpdateTime >= 1 {
					// Calculate progress based on elapsed time
					elapsed := time.Since(startTime)
					progress := math.Min(100, (elapsed.Seconds()/totalDuration.Seconds())*100)

					// Get speed from current interval
					currentSpeed := streamData.Data.Sum.Bits_per_second / 1_000_000 // Convert to Mbps

					log.Debug().
						Float64("progress", progress).
						Float64("speed", currentSpeed).
						Str("type", testType).
						Str("event", streamData.Event).
						Msg("iperf3 streaming progress update")

					if progress > 0 && r.progressCallback != nil {
						r.progressCallback(types.SpeedUpdate{
							Type:       testType,
							ServerName: serverName,
							Speed:      currentSpeed,
							Progress:   progress,
							IsComplete: false,
							TestType:   "iperf3",
						})
						lastUpdate.Store(now)
					}
				}
			}
		}
	}

	if err := cmd.Wait(); err != nil {
		// Check if the error was due to context timeout
		if timeoutCtx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("iperf3 test timed out after %d seconds", r.config.Timeout)
		}
		outputStr := output.String()
		// Parse and format JSON output for better error display
		var jsonOutput map[string]any
		if jsonErr := json.Unmarshal([]byte(outputStr), &jsonOutput); jsonErr == nil {
			if formattedJSON, formatErr := json.Marshal(jsonOutput); formatErr == nil {
				outputStr = string(formattedJSON)
			}
		}
		return nil, fmt.Errorf("iperf3 failed: %s - %w", outputStr, err)
	}

	// Parse final results from streaming JSON output
	// Find the last "end" event in the output
	var finalResult struct {
		Event string `json:"event"`
		Data  struct {
			SumSent struct {
				BitsPerSecond float64 `json:"bits_per_second"`
				JitterMs      float64 `json:"jitter_ms"`
			} `json:"sum_sent"`
			SumReceived struct {
				BitsPerSecond float64 `json:"bits_per_second"`
				JitterMs      float64 `json:"jitter_ms"`
			} `json:"sum_received"`
		} `json:"data"`
	}

	// Parse the output line by line to find the "end" event
	lines := strings.Split(output.String(), "\n")
	var foundFinalResult bool
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var lineData struct {
			Event string          `json:"event"`
			Data  json.RawMessage `json:"data"`
		}
		if err := json.Unmarshal([]byte(line), &lineData); err == nil {
			if lineData.Event == "end" {
				if err := json.Unmarshal(lineData.Data, &finalResult.Data); err == nil {
					foundFinalResult = true
					break
				}
			}
		}
	}

	if !foundFinalResult {
		return nil, fmt.Errorf("failed to find final results in iperf3 streaming output")
	}

	var speedMbps float64
	var jitterMs *float64

	if opts.EnableDownload {
		speedMbps = finalResult.Data.SumReceived.BitsPerSecond / 1_000_000
	} else {
		speedMbps = finalResult.Data.SumSent.BitsPerSecond / 1_000_000
	}

	// Send final update
	if r.progressCallback != nil {
		r.progressCallback(types.SpeedUpdate{
			Type:       testType,
			ServerName: serverName,
			Speed:      speedMbps,
			Progress:   100.0,
			IsComplete: true,
			TestType:   "iperf3",
		})
	}

	return &types.SpeedTestResult{
		ServerName: serverName,
		TestType:   "iperf3",
		DownloadSpeed: func() float64 {
			if opts.EnableDownload {
				return speedMbps
			}
			return 0
		}(),
		UploadSpeed: func() float64 {
			if !opts.EnableDownload {
				return speedMbps
			}
			return 0
		}(),
		Jitter:      jitterMs,
		IsScheduled: opts.IsScheduled,
	}, nil
}
