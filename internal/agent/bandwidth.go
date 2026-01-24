// Copyright (c) 2024-2026, s0up and the autobrr contributors.
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
	var bandwidthData map[string]any
	if err := json.Unmarshal(output, &bandwidthData); err != nil {
		log.Error().Err(err).Msg("Failed to parse bandwidth data JSON")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to parse bandwidth data",
		})
		return
	}

	// Normalize vnstat 1.x JSON format to 2.x format
	// vnstat 1.x uses "hours"/"days"/"months" (plural) and lacks "time" fields
	// vnstat 2.x uses "hour"/"day"/"month" (singular) with "time" fields
	if jsonVer, ok := bandwidthData["jsonversion"].(string); ok && jsonVer == "1" {
		normalizeVnstatV1(bandwidthData)
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

// normalizeVnstatV1 converts vnstat JSON version 1 format to version 2 format.
// v1 uses plural keys ("hours", "days", "months") and hour entries lack "time" fields.
// v2 uses singular keys ("hour", "day", "month") and hour entries include "time" with hour/minute.
func normalizeVnstatV1(data map[string]any) {
	data["jsonversion"] = "2"

	interfaces, ok := data["interfaces"].([]any)
	if !ok {
		return
	}

	for _, iface := range interfaces {
		ifaceMap, ok := iface.(map[string]any)
		if !ok {
			continue
		}

		traffic, ok := ifaceMap["traffic"].(map[string]any)
		if !ok {
			continue
		}

		// Rename plural keys to singular and synthesize "time" for hour entries
		if hours, ok := traffic["hours"].([]any); ok {
			for _, entry := range hours {
				hourMap, ok := entry.(map[string]any)
				if !ok {
					continue
				}
				// v1 stores hour-of-day in "id" field (0-23)
				if id, ok := hourMap["id"].(float64); ok {
					hourMap["time"] = map[string]any{
						"hour":   int(id),
						"minute": 0,
					}
				}
				// v1 hour entries may lack "day" in date
				if dateMap, ok := hourMap["date"].(map[string]any); ok {
					if _, hasDay := dateMap["day"]; !hasDay {
						dateMap["day"] = float64(time.Now().Day())
					}
				}
			}
			traffic["hour"] = hours
			delete(traffic, "hours")
		}

		if days, ok := traffic["days"].([]any); ok {
			traffic["day"] = days
			delete(traffic, "days")
		}

		if months, ok := traffic["months"].([]any); ok {
			traffic["month"] = months
			delete(traffic, "months")
		}
	}
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
