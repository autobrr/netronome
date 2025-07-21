// Copyright (c) 2024-2025, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package database

import (
	"time"
)

type NotificationChannel struct {
	ID        int64     `json:"id" db:"id"`
	Name      string    `json:"name" db:"name"`
	URL       string    `json:"url" db:"url"`
	Enabled   bool      `json:"enabled" db:"enabled"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

type NotificationEvent struct {
	ID                int64   `json:"id" db:"id"`
	Category          string  `json:"category" db:"category"`
	EventType         string  `json:"event_type" db:"event_type"`
	Name              string  `json:"name" db:"name"`
	Description       *string `json:"description" db:"description"`
	DefaultEnabled    bool    `json:"default_enabled" db:"default_enabled"`
	SupportsThreshold bool    `json:"supports_threshold" db:"supports_threshold"`
	ThresholdUnit     *string `json:"threshold_unit" db:"threshold_unit"`
	CreatedAt         time.Time `json:"created_at" db:"created_at"`
}

type NotificationRule struct {
	ID                 int64     `json:"id" db:"id"`
	ChannelID          int64     `json:"channel_id" db:"channel_id"`
	EventID            int64     `json:"event_id" db:"event_id"`
	Enabled            bool      `json:"enabled" db:"enabled"`
	ThresholdValue     *float64  `json:"threshold_value" db:"threshold_value"`
	ThresholdOperator  *string   `json:"threshold_operator" db:"threshold_operator"`
	CreatedAt          time.Time `json:"created_at" db:"created_at"`
	UpdatedAt          time.Time `json:"updated_at" db:"updated_at"`
	
	// Joined fields for queries
	Channel *NotificationChannel `json:"channel,omitempty" db:"-"`
	Event   *NotificationEvent   `json:"event,omitempty" db:"-"`
}

type NotificationHistory struct {
	ID           int64     `json:"id" db:"id"`
	ChannelID    int64     `json:"channel_id" db:"channel_id"`
	EventID      int64     `json:"event_id" db:"event_id"`
	Success      bool      `json:"success" db:"success"`
	ErrorMessage *string   `json:"error_message" db:"error_message"`
	Payload      *string   `json:"payload" db:"payload"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}

type NotificationChannelInput struct {
	Name    string `json:"name" validate:"required"`
	URL     string `json:"url" validate:"required"`
	Enabled *bool  `json:"enabled"`
}

type NotificationRuleInput struct {
	ChannelID         int64    `json:"channel_id" validate:"required"`
	EventID           int64    `json:"event_id" validate:"required"`
	Enabled           *bool    `json:"enabled"`
	ThresholdValue    *float64 `json:"threshold_value"`
	ThresholdOperator *string  `json:"threshold_operator" validate:"omitempty,oneof=gt lt eq gte lte"`
}

// NotificationEventCategory constants
const (
	NotificationCategorySpeedtest  = "speedtest"
	NotificationCategoryPacketLoss = "packetloss"
	NotificationCategoryAgent      = "agent"
)

// NotificationEventType constants
const (
	// Speedtest events
	NotificationEventSpeedtestComplete    = "complete"
	NotificationEventSpeedtestPingHigh    = "ping_high"
	NotificationEventSpeedtestDownloadLow = "download_low"
	NotificationEventSpeedtestUploadLow   = "upload_low"
	NotificationEventSpeedtestFailed      = "failed"
	
	// Packet loss events
	NotificationEventPacketLossHigh      = "threshold_exceeded"
	NotificationEventPacketLossDown      = "monitor_down"
	NotificationEventPacketLossRecovered = "monitor_recovered"
	
	// Agent events
	NotificationEventAgentOffline       = "offline"
	NotificationEventAgentOnline        = "online"
	NotificationEventAgentHighBandwidth = "high_bandwidth"
	NotificationEventAgentLowDisk       = "disk_space_low"
	NotificationEventAgentHighCPU       = "cpu_high"
	NotificationEventAgentHighMemory    = "memory_high"
	NotificationEventAgentHighTemp      = "temperature_high"
)

// ThresholdOperator constants
const (
	ThresholdOperatorGT  = "gt"  // greater than
	ThresholdOperatorLT  = "lt"  // less than
	ThresholdOperatorEQ  = "eq"  // equal
	ThresholdOperatorGTE = "gte" // greater than or equal
	ThresholdOperatorLTE = "lte" // less than or equal
)

// NotificationService interface for database operations
type NotificationService interface {
	// Channels
	CreateChannel(input NotificationChannelInput) (*NotificationChannel, error)
	GetChannel(id int64) (*NotificationChannel, error)
	GetChannels() ([]NotificationChannel, error)
	GetEnabledChannels() ([]NotificationChannel, error)
	UpdateChannel(id int64, input NotificationChannelInput) (*NotificationChannel, error)
	DeleteChannel(id int64) error
	
	// Events
	GetEvents() ([]NotificationEvent, error)
	GetEventsByCategory(category string) ([]NotificationEvent, error)
	GetEvent(id int64) (*NotificationEvent, error)
	GetEventByType(category, eventType string) (*NotificationEvent, error)
	
	// Rules
	CreateRule(input NotificationRuleInput) (*NotificationRule, error)
	GetRule(id int64) (*NotificationRule, error)
	GetRules() ([]NotificationRule, error)
	GetRulesByChannel(channelID int64) ([]NotificationRule, error)
	GetRulesByEvent(eventID int64) ([]NotificationRule, error)
	GetEnabledRulesForEvent(category, eventType string) ([]NotificationRule, error)
	UpdateRule(id int64, input NotificationRuleInput) (*NotificationRule, error)
	DeleteRule(id int64) error
	
	// History
	LogNotification(channelID, eventID int64, success bool, errorMessage *string, payload *string) error
	GetNotificationHistory(limit int) ([]NotificationHistory, error)
	
	// Utility
	CheckThreshold(rule *NotificationRule, value float64) bool
}