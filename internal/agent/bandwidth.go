// Copyright (c) 2024-2025, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package agent

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

// handleHistoricalExport exports all historical bandwidth monitor data
func (a *Agent) handleHistoricalExport(c *gin.Context) {
	// Get optional interface parameter
	iface := c.Query("interface")
	if iface == "" {
		iface = a.config.Interface
	}

	// Build vnstat command for all historical data
	args := []string{"--json", "a"}
	if iface != "" {
		args = append(args, "--iface", iface)
	}

	// Execute vnstat command
	cmd := exec.Command("vnstat", args...)
	output, err := cmd.Output()
	if err != nil {
		log.Error().Err(err).Msg("Failed to export historical data")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed to export historical data",
			"details": err.Error(),
		})
		return
	}

	// Parse the JSON to add timezone information
	var bandwidthData map[string]interface{}
	if err := json.Unmarshal(output, &bandwidthData); err != nil {
		log.Error().Err(err).Msg("Failed to parse bandwidth data JSON")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to parse bandwidth data",
		})
		return
	}

	// Add server time information for timezone handling
	now := time.Now()
	bandwidthData["server_time"] = now.Format(time.RFC3339)
	bandwidthData["server_time_unix"] = now.Unix()
	_, offset := now.Zone()
	bandwidthData["timezone_offset"] = offset // Offset in seconds from UTC

	// Re-encode with timezone information
	enrichedOutput, err := json.Marshal(bandwidthData)
	if err != nil {
		log.Error().Err(err).Msg("Failed to encode bandwidth data")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to encode data",
		})
		return
	}

	// Set appropriate headers for JSON response
	c.Header("Content-Type", "application/json")
	c.Header("Content-Disposition", "inline; filename=\"monitor-historical.json\"")

	// Return the enriched JSON data
	c.Data(http.StatusOK, "application/json", enrichedOutput)
}

// runBandwidthMonitor runs vnstat command and sends data to the broadcast channel
func (a *Agent) runBandwidthMonitor(ctx context.Context) {
	// Build vnstat command
	args := []string{"--live", "--json"}
	if a.config.Interface != "" {
		args = append(args, "--iface", a.config.Interface)
	}

	cmd := exec.CommandContext(ctx, "vnstat", args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Error().Err(err).Msg("Failed to create stdout pipe")
		return
	}

	if err := cmd.Start(); err != nil {
		log.Error().Err(err).Msg("Failed to start vnstat")
		return
	}

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()

		// Parse JSON to validate it
		var data MonitorLiveData
		if err := json.Unmarshal([]byte(line), &data); err != nil {
			log.Warn().Err(err).Str("line", line).Msg("Failed to parse bandwidth data JSON")
			continue
		}

		// Track peak speeds
		a.peakMu.Lock()
		now := time.Now()
		if data.Rx.Bytespersecond > a.peakRx {
			a.peakRx = data.Rx.Bytespersecond
			a.peakRxTimestamp = now
		}
		if data.Tx.Bytespersecond > a.peakTx {
			a.peakTx = data.Tx.Bytespersecond
			a.peakTxTimestamp = now
		}
		a.peakMu.Unlock()

		// Send to broadcaster
		select {
		case a.monitorData <- line:
		default:
			// Channel full, skip
		}

		log.Trace().
			Str("rx", data.Rx.Ratestring).
			Str("tx", data.Tx.Ratestring).
			Msg("Broadcasting bandwidth monitor data")
	}

	if err := scanner.Err(); err != nil {
		log.Error().Err(err).Msg("Scanner error")
	}

	if err := cmd.Wait(); err != nil {
		log.Error().Err(err).Msg("vnstat command failed")
	}
}

// handlePeakStats returns peak bandwidth statistics
func (a *Agent) handlePeakStats(c *gin.Context) {
	a.peakMu.RLock()
	stats := PeakStats{
		PeakRx:          a.peakRx,
		PeakTx:          a.peakTx,
		PeakRxString:    formatBytesPerSecond(a.peakRx),
		PeakTxString:    formatBytesPerSecond(a.peakTx),
		PeakRxTimestamp: a.peakRxTimestamp,
		PeakTxTimestamp: a.peakTxTimestamp,
		UpdatedAt:       time.Now(),
	}
	a.peakMu.RUnlock()

	c.JSON(http.StatusOK, stats)
}

// formatBytesPerSecond formats bytes per second to human readable string
func formatBytesPerSecond(bytes int) string {
	if bytes == 0 {
		return "0 B/s"
	}

	const k = 1024
	sizes := []string{"B/s", "KiB/s", "MiB/s", "GiB/s", "TiB/s"}

	i := 0
	bytesFloat := float64(bytes)
	for bytesFloat >= k && i < len(sizes)-1 {
		bytesFloat /= k
		i++
	}

	return fmt.Sprintf("%.2f %s", bytesFloat, sizes[i])
}
