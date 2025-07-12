// Copyright (c) 2024-2025, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package notifications

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"

	"github.com/autobrr/netronome/internal/config"
	"github.com/autobrr/netronome/internal/types"
)

type Notifier struct {
	Cfg *config.NotificationConfig
}

func NewNotifier(cfg *config.NotificationConfig) *Notifier {
	return &Notifier{Cfg: cfg}
}

// PacketLossNotification represents packet loss monitoring data for notifications
type PacketLossNotification struct {
	MonitorName string
	Host        string
	PacketLoss  float64
	Threshold   float64
	AvgRTT      float64 // in milliseconds
	MinRTT      float64 // in milliseconds
	MaxRTT      float64 // in milliseconds
	PacketsSent int
	PacketsRecv int
	UsedMTR     bool
	HopCount    int
}

func (n *Notifier) SendNotification(result *types.SpeedTestResult) {
	if !n.Cfg.Enabled || n.Cfg.WebhookURL == "" {
		return
	}

	alerts := n.checkThresholds(result)

	// Only send notification if there are threshold breaches
	if len(alerts) > 0 {
		payload := n.buildPayload(result, alerts)

		if err := n.sendWebhook(payload); err != nil {
			log.Error().Err(err).Msg("Failed to send notification")
		}
	}
}

func (n *Notifier) checkThresholds(result *types.SpeedTestResult) []string {
	var alerts []string

	latency, err := strconv.ParseFloat(strings.TrimSuffix(result.Latency, "ms"), 64)
	if err != nil {
		log.Warn().Err(err).Msg("could not parse latency")
	} else {
		if latency > n.Cfg.PingThreshold {
			alerts = append(alerts, fmt.Sprintf("Ping is above threshold: %.2f > %.2f ms", latency, n.Cfg.PingThreshold))
		}
	}

	if result.DownloadSpeed < n.Cfg.DownloadThreshold {
		alerts = append(alerts, fmt.Sprintf("Download speed is below threshold: %.2f < %.2f Mbps", result.DownloadSpeed, n.Cfg.DownloadThreshold))
	}
	if result.UploadSpeed < n.Cfg.UploadThreshold {
		alerts = append(alerts, fmt.Sprintf("Upload speed is below threshold: %.2f < %.2f Mbps", result.UploadSpeed, n.Cfg.UploadThreshold))
	}

	return alerts
}

func (n *Notifier) buildPayload(result *types.SpeedTestResult, alerts []string) map[string]interface{} {
	testType := result.TestType
	if testType == "speedtest" {
		testType = "speedtest.net"
	}

	serverName := fmt.Sprintf("%s (%s)", result.ServerName, testType)

	embed := map[string]interface{}{
		"title":       "Speed Test Results",
		"description": "A summary of the latest speed test results.",
		"color":       0x00ff00, // Green
		"fields": []map[string]interface{}{
			{"name": "Server", "value": serverName, "inline": false},
			{"name": "Ping", "value": fmt.Sprintf("%v", result.Latency), "inline": false},
			{"name": "Download", "value": fmt.Sprintf("%.2f Mbps", result.DownloadSpeed), "inline": true},
			{"name": "Upload", "value": fmt.Sprintf("%.2f Mbps", result.UploadSpeed), "inline": true},
		},
	}

	content := ""
	if len(alerts) > 0 {
		embed["color"] = 0xff0000 // Red
		embed["fields"] = append(embed["fields"].([]map[string]interface{}), map[string]interface{}{
			"name":  "Alerts",
			"value": "- " + strings.Join(alerts, "\n- "),
		})
		if n.Cfg.DiscordMentionID != "" {
			content = fmt.Sprintf("<@%s>", n.Cfg.DiscordMentionID)
		}
	}

	return map[string]interface{}{
		"content": content,
		"embeds":  []map[string]interface{}{embed},
	}
}

func (n *Notifier) sendWebhook(payload map[string]interface{}) error {
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequest("POST", n.Cfg.WebhookURL, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned non-success status code: %d", resp.StatusCode)
	}

	log.Info().Msg("Notification sent successfully")
	return nil
}

// SendPacketLossNotification sends a notification specifically for packet loss monitoring
func (n *Notifier) SendPacketLossNotification(data *PacketLossNotification) {
	if !n.Cfg.Enabled || n.Cfg.WebhookURL == "" {
		return
	}

	// Build packet loss specific payload
	payload := n.buildPacketLossPayload(data)

	if err := n.sendWebhook(payload); err != nil {
		log.Error().Err(err).Msg("Failed to send packet loss notification")
	}
}

// buildPacketLossPayload creates a Discord/webhook payload for packet loss alerts
func (n *Notifier) buildPacketLossPayload(data *PacketLossNotification) map[string]interface{} {
	// Determine test mode string
	testMode := "Ping"
	if data.UsedMTR {
		testMode = fmt.Sprintf("MTR (%d hops)", data.HopCount)
	}

	// Create embed fields
	fields := []map[string]interface{}{
		{"name": "Monitor", "value": data.MonitorName, "inline": false},
		{"name": "Host", "value": data.Host, "inline": false},
		{"name": "Packet Loss", "value": fmt.Sprintf("%.1f%% (threshold: %.1f%%)", data.PacketLoss, data.Threshold), "inline": false},
		{"name": "Average RTT", "value": fmt.Sprintf("%.1fms", data.AvgRTT), "inline": true},
		{"name": "Min/Max RTT", "value": fmt.Sprintf("%.1fms / %.1fms", data.MinRTT, data.MaxRTT), "inline": true},
		{"name": "Packets", "value": fmt.Sprintf("%d/%d received", data.PacketsRecv, data.PacketsSent), "inline": false},
		{"name": "Test Mode", "value": testMode, "inline": true},
	}

	embed := map[string]interface{}{
		"title":       "Packet Loss Alert",
		"description": fmt.Sprintf("Monitor \"%s\" has exceeded the packet loss threshold", data.MonitorName),
		"color":       0xff0000, // Red for alert
		"fields":      fields,
	}

	// Add mention if configured
	content := ""
	if n.Cfg.DiscordMentionID != "" {
		content = fmt.Sprintf("<@%s>", n.Cfg.DiscordMentionID)
	}

	return map[string]interface{}{
		"content": content,
		"embeds":  []map[string]interface{}{embed},
	}
}
