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
	var packetLoss float64 = 0.0

	// Use ping results if available
	if r.pingResult != nil {
		latency = r.pingResult.FormatLatency()
		packetLoss = r.pingResult.PacketLoss
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
		if uploadResult.Jitter != nil {
			jitterMs = uploadResult.Jitter
		}
	}

	var jitterFloat float64
	if jitterMs != nil {
		jitterFloat = *jitterMs
	}

	result := &Result{
		Timestamp:     time.Now(),
		Server:        opts.ServerHost,
		DownloadSpeed: downloadSpeed,
		UploadSpeed:   uploadSpeed,
		Latency:       latency,
		PacketLoss:    packetLoss,
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
		"-J",        // JSON output
		"-i", "0.5", // Half-second interval for smoother updates
		"-t", strconv.Itoa(r.config.TestDuration), // Test duration in seconds
		"-P", strconv.Itoa(r.config.ParallelConns), // Number of parallel connections
		"--format", "m", // Force Mbps output
	}

	// Add UDP-specific arguments if jitter testing is enabled
	if opts.EnableJitter && r.config.EnableUDP {
		args = append(args, "-u") // UDP mode
		if r.config.UDPBandwidth != "" {
			args = append(args, "-b", r.config.UDPBandwidth) // Bandwidth limit
		}
	}

	if opts.EnableDownload {
		args = append(args, "-R")
	}

	testType := "upload"
	if opts.EnableDownload {
		testType = "download"
	}

	// Send initial status
	if r.progressCallback != nil {
		r.progressCallback(types.SpeedUpdate{
			Type:       testType,
			ServerName: fmt.Sprintf("%s:%s", host, port),
			Speed:      0,
			Progress:   0,
			IsComplete: false,
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

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start iperf3: %w", err)
	}

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		output.WriteString(line + "\n")

		// Try to parse JSON for progress updates
		var data struct {
			Start struct {
				Connected []struct{} `json:"connected"`
			} `json:"start"`
			Intervals []struct {
				Streams []struct {
					Bits_per_second float64 `json:"bits_per_second"`
				} `json:"streams"`
				Sum struct {
					Bits_per_second float64 `json:"bits_per_second"`
				} `json:"sum"`
			} `json:"intervals"`
		}

		if err := json.Unmarshal([]byte(line), &data); err == nil {
			// Calculate progress based on elapsed time
			elapsed := time.Since(startTime)
			progress := math.Min(100, (elapsed.Seconds()/totalDuration.Seconds())*100)

			// Get speed from the most recent interval
			var currentSpeed float64
			if len(data.Intervals) > 0 {
				lastInterval := data.Intervals[len(data.Intervals)-1]
				currentSpeed = lastInterval.Sum.Bits_per_second / 1_000_000 // Convert to Mbps
			}

			log.Debug().
				Float64("progress", progress).
				Float64("speed", currentSpeed).
				Str("type", testType).
				Msg("iperf3 progress update")

			if r.progressCallback != nil {
				r.progressCallback(types.SpeedUpdate{
					Type:       testType,
					ServerName: fmt.Sprintf("%s:%s", host, port),
					Speed:      currentSpeed,
					Progress:   progress,
					IsComplete: false,
				})
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

	// Parse final results from the complete output
	var finalResult struct {
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
	}

	if err := json.Unmarshal([]byte(output.String()), &finalResult); err != nil {
		return nil, fmt.Errorf("failed to parse iperf3 output: %w", err)
	}

	var speedMbps float64
	var jitterMs *float64
	if opts.EnableDownload {
		speedMbps = finalResult.End.SumReceived.BitsPerSecond / 1_000_000
		if opts.EnableJitter && finalResult.End.SumReceived.JitterMs > 0 {
			jitterMs = &finalResult.End.SumReceived.JitterMs
		}
	} else {
		speedMbps = finalResult.End.SumSent.BitsPerSecond / 1_000_000
		if opts.EnableJitter && finalResult.End.SumSent.JitterMs > 0 {
			jitterMs = &finalResult.End.SumSent.JitterMs
		}
	}

	// Send final update
	if r.progressCallback != nil {
		r.progressCallback(types.SpeedUpdate{
			Type:       testType,
			ServerName: fmt.Sprintf("%s:%s", host, port),
			Speed:      speedMbps,
			Progress:   100.0,
			IsComplete: true,
		})
	}

	return &types.SpeedTestResult{
		ServerName: fmt.Sprintf("%s:%s", host, port),
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
