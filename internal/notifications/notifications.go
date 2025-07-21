// Copyright (c) 2024-2025, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package notifications

import (
	"fmt"
	"strings"

	"github.com/containrrr/shoutrrr"
	"github.com/containrrr/shoutrrr/pkg/router"
	"github.com/rs/zerolog/log"

	"github.com/autobrr/netronome/internal/database"
)

type Notifier struct {
	db     database.NotificationService
	router *router.ServiceRouter
}

// NewNotifier creates a new notifier with database support
func NewNotifier(db database.NotificationService) (*Notifier, error) {
	return &Notifier{
		db: db,
	}, nil
}

// NewNotifierFromURLs creates a temporary notifier for testing
func NewNotifierFromURLs(urls []string) (*Notifier, error) {
	if len(urls) == 0 {
		return &Notifier{}, nil
	}

	r, err := shoutrrr.CreateSender(urls...)
	if err != nil {
		return nil, fmt.Errorf("failed to create shoutrrr router: %w", err)
	}

	return &Notifier{
		router: r,
	}, nil
}

// SendNotification sends a notification for a specific event
func (n *Notifier) SendNotification(category, eventType string, message string, value *float64) error {
	if n.db == nil {
		// For temporary notifiers (like testing), use the router directly
		if n.router != nil {
			errs := n.router.Send(message, nil)
			if len(errs) > 0 {
				return errs[0]
			}
			return nil
		}
		return fmt.Errorf("no database or router configured")
	}

	// Get enabled rules for this event
	rules, err := n.db.GetEnabledRulesForEvent(category, eventType)
	if err != nil {
		return fmt.Errorf("failed to get notification rules: %w", err)
	}

	if len(rules) == 0 {
		return nil
	}

	var lastError error
	successCount := 0

	for _, rule := range rules {
		// Check threshold if applicable
		if value != nil && rule.ThresholdValue != nil {
			if !n.db.CheckThreshold(&rule, *value) {
				continue
			}
		}

		// Send notification
		if rule.Channel == nil {
			log.Warn().
				Int64("ruleID", rule.ID).
				Int64("channelID", rule.ChannelID).
				Msg("Rule has no channel associated, skipping")
			continue
		}
		
		if rule.Channel.URL != "" {
			tempNotifier, err := shoutrrr.CreateSender(rule.Channel.URL)
			if err != nil {
				lastError = err
				errMsg := err.Error()
				if logErr := n.db.LogNotification(rule.ChannelID, rule.EventID, false, &errMsg, &message); logErr != nil {
					log.Error().Err(logErr).Msg("Failed to log notification error")
				}
				log.Error().
					Err(err).
					Int64("channelID", rule.ChannelID).
					Msg("Failed to create notifier for channel")
				continue
			}

			if tempNotifier == nil {
				lastError = fmt.Errorf("notifier is nil")
				errMsg := "notifier is nil"
				if logErr := n.db.LogNotification(rule.ChannelID, rule.EventID, false, &errMsg, &message); logErr != nil {
					log.Error().Err(logErr).Msg("Failed to log notification error")
				}
				continue
			}

			errs := tempNotifier.Send(message, nil)
			// Filter out nil errors from the slice
			var realErrors []error
			for _, err := range errs {
				if err != nil {
					realErrors = append(realErrors, err)
				}
			}
			
			if len(realErrors) > 0 {
				lastError = realErrors[0]
				errMsg := realErrors[0].Error()
				if logErr := n.db.LogNotification(rule.ChannelID, rule.EventID, false, &errMsg, &message); logErr != nil {
					log.Error().Err(logErr).Msg("Failed to log notification error")
				}
				log.Error().
					Err(realErrors[0]).
					Int64("channelID", rule.ChannelID).
					Msg("Failed to send notification")
			} else {
				successCount++
				if logErr := n.db.LogNotification(rule.ChannelID, rule.EventID, true, nil, &message); logErr != nil {
					log.Error().Err(logErr).Msg("Failed to log notification success")
				}
			}
		}
	}

	if successCount == 0 && lastError != nil {
		return fmt.Errorf("failed to send any notifications: %w", lastError)
	}

	if successCount > 0 {
		log.Info().
			Int("sent", successCount).
			Int("total", len(rules)).
			Str("category", category).
			Str("eventType", eventType).
			Msg("Notifications sent")
	}

	return nil
}

// SendSpeedTestNotification sends a speed test notification
func (n *Notifier) SendSpeedTestNotification(result *SpeedTestResult) error {
	// Check for various conditions
	if result.Failed {
		message := fmt.Sprintf("❌ **Speed Test Failed**\n\nProvider: %s\nServer: %s\nError: Test failed to complete",
			result.Provider, result.ServerName)
		return n.SendNotification(database.NotificationCategorySpeedtest, database.NotificationEventSpeedtestFailed, message, nil)
	}

	// Always send completion notification
	message := n.formatSpeedTestMessage(result)
	if err := n.SendNotification(database.NotificationCategorySpeedtest, database.NotificationEventSpeedtestComplete, message, nil); err != nil {
		log.Error().Err(err).Msg("Failed to send speed test completion notification")
	}

	// Check thresholds
	if result.Ping > 0 {
		if err := n.SendNotification(database.NotificationCategorySpeedtest, database.NotificationEventSpeedtestPingHigh, message, &result.Ping); err != nil {
			log.Error().Err(err).Msg("Failed to send high ping notification")
		}
	}

	if result.Download > 0 {
		downloadMbps := result.Download
		if err := n.SendNotification(database.NotificationCategorySpeedtest, database.NotificationEventSpeedtestDownloadLow, message, &downloadMbps); err != nil {
			log.Error().Err(err).Msg("Failed to send low download notification")
		}
	}

	if result.Upload > 0 {
		uploadMbps := result.Upload
		if err := n.SendNotification(database.NotificationCategorySpeedtest, database.NotificationEventSpeedtestUploadLow, message, &uploadMbps); err != nil {
			log.Error().Err(err).Msg("Failed to send low upload notification")
		}
	}

	return nil
}

// SendPacketLossNotification sends a packet loss notification
func (n *Notifier) SendPacketLossNotification(monitorName string, host string, packetLoss float64, isDown bool, isRecovered bool) error {
	if isRecovered {
		message := fmt.Sprintf("✅ **Monitor Recovered**\n\nMonitor: %s\nHost: %s\nStatus: Back Online",
			monitorName, host)
		return n.SendNotification(database.NotificationCategoryPacketLoss, database.NotificationEventPacketLossRecovered, message, nil)
	}

	if isDown {
		message := fmt.Sprintf("🔴 **Monitor Down**\n\nMonitor: %s\nHost: %s\nStatus: Unreachable (100%% packet loss)",
			monitorName, host)
		return n.SendNotification(database.NotificationCategoryPacketLoss, database.NotificationEventPacketLossDown, message, nil)
	}

	// High packet loss
	message := fmt.Sprintf("⚠️ **High Packet Loss**\n\nMonitor: %s\nHost: %s\nPacket Loss: %.1f%%",
		monitorName, host, packetLoss)
	return n.SendNotification(database.NotificationCategoryPacketLoss, database.NotificationEventPacketLossHigh, message, &packetLoss)
}

// SendAgentNotification sends an agent-related notification
func (n *Notifier) SendAgentNotification(agentName string, eventType string, value *float64) error {
	var message string
	
	switch eventType {
	case database.NotificationEventAgentOffline:
		message = fmt.Sprintf("🔴 **Agent Offline**\n\nAgent: %s\nStatus: Connection lost", agentName)
	case database.NotificationEventAgentOnline:
		message = fmt.Sprintf("✅ **Agent Online**\n\nAgent: %s\nStatus: Connection restored", agentName)
	case database.NotificationEventAgentHighBandwidth:
		if value != nil {
			message = fmt.Sprintf("📈 **High Bandwidth Usage**\n\nAgent: %s\nBandwidth: %.1f Mbps", agentName, *value)
		} else {
			message = fmt.Sprintf("📈 **High Bandwidth Usage**\n\nAgent: %s\nBandwidth: Unknown", agentName)
		}
	case database.NotificationEventAgentLowDisk:
		if value != nil {
			message = fmt.Sprintf("💾 **Low Disk Space**\n\nAgent: %s\nDisk Usage: %.1f%%", agentName, *value)
		} else {
			message = fmt.Sprintf("💾 **Low Disk Space**\n\nAgent: %s\nDisk Usage: Unknown", agentName)
		}
	case database.NotificationEventAgentHighCPU:
		if value != nil {
			message = fmt.Sprintf("🔥 **High CPU Usage**\n\nAgent: %s\nCPU: %.1f%%", agentName, *value)
		} else {
			message = fmt.Sprintf("🔥 **High CPU Usage**\n\nAgent: %s\nCPU: Unknown", agentName)
		}
	case database.NotificationEventAgentHighMemory:
		if value != nil {
			message = fmt.Sprintf("🧠 **High Memory Usage**\n\nAgent: %s\nMemory: %.1f%%", agentName, *value)
		} else {
			message = fmt.Sprintf("🧠 **High Memory Usage**\n\nAgent: %s\nMemory: Unknown", agentName)
		}
	default:
		return fmt.Errorf("unknown agent event type: %s", eventType)
	}

	if message == "" {
		return fmt.Errorf("empty notification message for event type: %s", eventType)
	}

	return n.SendNotification(database.NotificationCategoryAgent, eventType, message, value)
}

// SendTestNotification sends a test notification
func (n *Notifier) SendTestNotification() error {
	title := "Netronome Test"
	message := fmt.Sprintf("✅ **%s**\n\nThis is a test notification from Netronome. If you received this, your notifications are working correctly!", title)
	
	if n.router != nil {
		errs := n.router.Send(message, nil)
		if len(errs) > 0 {
			return errs[0]
		}
		return nil
	}
	
	return fmt.Errorf("no router configured")
}

// formatSpeedTestMessage formats a speed test result into a notification message
func (n *Notifier) formatSpeedTestMessage(result *SpeedTestResult) string {
	var sb strings.Builder

	if result.Failed {
		sb.WriteString("❌ **Speed Test Failed**\n\n")
	} else {
		sb.WriteString("📊 **Speed Test Results**\n\n")
	}

	if result.ServerName != "" {
		sb.WriteString(fmt.Sprintf("**Server:** %s\n", result.ServerName))
	}
	if result.Provider != "" {
		sb.WriteString(fmt.Sprintf("**Provider:** %s\n", result.Provider))
	}
	
	sb.WriteString("\n")

	if !result.Failed {
		if result.Download > 0 {
			sb.WriteString(fmt.Sprintf("⬇️ **Download:** %.2f Mbps\n", result.Download))
		}
		if result.Upload > 0 {
			sb.WriteString(fmt.Sprintf("⬆️ **Upload:** %.2f Mbps\n", result.Upload))
		}
		if result.Ping > 0 {
			sb.WriteString(fmt.Sprintf("📶 **Ping:** %.2f ms\n", result.Ping))
		}
		if result.Jitter > 0 {
			sb.WriteString(fmt.Sprintf("📊 **Jitter:** %.2f ms\n", result.Jitter))
		}
		if result.PacketLoss >= 0 {
			sb.WriteString(fmt.Sprintf("📉 **Packet Loss:** %.1f%%\n", result.PacketLoss))
		}
	} else {
		sb.WriteString("The speed test failed to complete.\n")
	}

	if result.ISP != "" {
		sb.WriteString(fmt.Sprintf("\n**ISP:** %s", result.ISP))
	}

	return sb.String()
}

// MigrateDiscordWebhook converts an old Discord webhook URL to Shoutrrr format
func MigrateDiscordWebhook(webhookURL string) string {
	if webhookURL == "" {
		return ""
	}

	// Already in Shoutrrr format
	if strings.HasPrefix(webhookURL, "discord://") {
		return webhookURL
	}

	// Parse Discord webhook URL
	// Format: https://discord.com/api/webhooks/{id}/{token}
	if strings.Contains(webhookURL, "discord.com/api/webhooks/") ||
		strings.Contains(webhookURL, "discordapp.com/api/webhooks/") {
		parts := strings.Split(webhookURL, "/")
		if len(parts) >= 2 {
			token := parts[len(parts)-1]
			id := parts[len(parts)-2]
			return fmt.Sprintf("discord://%s@%s", token, id)
		}
	}

	// Return as-is if we can't parse it
	return webhookURL
}

// SpeedTestResult represents the result of a speed test
type SpeedTestResult struct {
	ServerName string
	Provider   string
	Download   float64
	Upload     float64
	Ping       float64
	Jitter     float64
	PacketLoss float64
	ISP        string
	Failed     bool
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