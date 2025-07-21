// Copyright (c) 2024-2025, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package database

import (
	"database/sql"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"

	"github.com/autobrr/netronome/internal/config"
)

// CreateChannel creates a new notification channel
func (s *service) CreateChannel(input NotificationChannelInput) (*NotificationChannel, error) {
	now := time.Now()
	enabled := true
	if input.Enabled != nil {
		enabled = *input.Enabled
	}

	query := s.sqlBuilder.Insert("notification_channels").
		Columns("name", "url", "enabled", "created_at", "updated_at").
		Values(input.Name, input.URL, enabled, now, now)

	if s.config.Type == config.Postgres {
		query = query.Suffix("RETURNING id")
	}

	if s.config.Type == config.SQLite {
		result, err := query.RunWith(s.db).Exec()
		if err != nil {
			return nil, fmt.Errorf("failed to create notification channel: %w", err)
		}

		id, err := result.LastInsertId()
		if err != nil {
			return nil, fmt.Errorf("failed to get last insert id: %w", err)
		}

		return &NotificationChannel{
			ID:        id,
			Name:      input.Name,
			URL:       input.URL,
			Enabled:   enabled,
			CreatedAt: now,
			UpdatedAt: now,
		}, nil
	} else {
		// PostgreSQL
		var id int64
		err := query.RunWith(s.db).QueryRow().Scan(&id)
		if err != nil {
			return nil, fmt.Errorf("failed to create notification channel: %w", err)
		}

		return &NotificationChannel{
			ID:        id,
			Name:      input.Name,
			URL:       input.URL,
			Enabled:   enabled,
			CreatedAt: now,
			UpdatedAt: now,
		}, nil
	}
}

// GetChannels retrieves all notification channels
func (s *service) GetChannels() ([]NotificationChannel, error) {
	var channels []NotificationChannel

	rows, err := s.sqlBuilder.Select("id", "name", "url", "enabled", "created_at", "updated_at").
		From("notification_channels").
		OrderBy("created_at DESC").
		RunWith(s.db).
		Query()
	if err != nil {
		return nil, fmt.Errorf("failed to get notification channels: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var channel NotificationChannel
		if err := rows.Scan(&channel.ID, &channel.Name, &channel.URL, &channel.Enabled, &channel.CreatedAt, &channel.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan channel: %w", err)
		}
		channels = append(channels, channel)
	}

	return channels, nil
}

// GetEnabledChannels retrieves all enabled notification channels
func (s *service) GetEnabledChannels() ([]NotificationChannel, error) {
	var channels []NotificationChannel

	rows, err := s.sqlBuilder.Select("id", "name", "url", "enabled", "created_at", "updated_at").
		From("notification_channels").
		Where(sq.Eq{"enabled": true}).
		OrderBy("created_at DESC").
		RunWith(s.db).
		Query()
	if err != nil {
		return nil, fmt.Errorf("failed to get enabled notification channels: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var channel NotificationChannel
		if err := rows.Scan(&channel.ID, &channel.Name, &channel.URL, &channel.Enabled, &channel.CreatedAt, &channel.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan channel: %w", err)
		}
		channels = append(channels, channel)
	}

	return channels, nil
}

// GetChannel retrieves a single notification channel by ID
func (s *service) GetChannel(id int64) (*NotificationChannel, error) {
	var channel NotificationChannel

	err := s.sqlBuilder.Select("id", "name", "url", "enabled", "created_at", "updated_at").
		From("notification_channels").
		Where(sq.Eq{"id": id}).
		RunWith(s.db).
		QueryRow().
		Scan(&channel.ID, &channel.Name, &channel.URL, &channel.Enabled, &channel.CreatedAt, &channel.UpdatedAt)
	
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get notification channel: %w", err)
	}

	return &channel, nil
}

// UpdateChannel updates an existing notification channel
func (s *service) UpdateChannel(id int64, input NotificationChannelInput) (*NotificationChannel, error) {
	enabled := true
	if input.Enabled != nil {
		enabled = *input.Enabled
	}

	result, err := s.sqlBuilder.Update("notification_channels").
		Set("name", input.Name).
		Set("url", input.URL).
		Set("enabled", enabled).
		Set("updated_at", time.Now()).
		Where(sq.Eq{"id": id}).
		RunWith(s.db).
		Exec()
	
	if err != nil {
		return nil, fmt.Errorf("failed to update notification channel: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return nil, ErrNotFound
	}

	return s.GetChannel(id)
}

// DeleteChannel deletes a notification channel
func (s *service) DeleteChannel(id int64) error {
	result, err := s.sqlBuilder.Delete("notification_channels").
		Where(sq.Eq{"id": id}).
		RunWith(s.db).
		Exec()
	
	if err != nil {
		return fmt.Errorf("failed to delete notification channel: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

// GetEvents retrieves all notification events
func (s *service) GetEvents() ([]NotificationEvent, error) {
	var events []NotificationEvent

	rows, err := s.sqlBuilder.Select("id", "category", "event_type", "name", "description", "default_enabled", "supports_threshold", "threshold_unit", "created_at").
		From("notification_events").
		OrderBy("category", "name").
		RunWith(s.db).
		Query()
	if err != nil {
		return nil, fmt.Errorf("failed to get notification events: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var event NotificationEvent
		var description, thresholdUnit sql.NullString
		
		if err := rows.Scan(&event.ID, &event.Category, &event.EventType, &event.Name, &description, &event.DefaultEnabled, &event.SupportsThreshold, &thresholdUnit, &event.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan event: %w", err)
		}
		
		if description.Valid {
			event.Description = &description.String
		}
		if thresholdUnit.Valid {
			event.ThresholdUnit = &thresholdUnit.String
		}
		
		events = append(events, event)
	}

	return events, nil
}

// GetEventsByCategory retrieves notification events by category
func (s *service) GetEventsByCategory(category string) ([]NotificationEvent, error) {
	var events []NotificationEvent

	rows, err := s.sqlBuilder.Select("id", "category", "event_type", "name", "description", "default_enabled", "supports_threshold", "threshold_unit", "created_at").
		From("notification_events").
		Where(sq.Eq{"category": category}).
		OrderBy("name").
		RunWith(s.db).
		Query()
	if err != nil {
		return nil, fmt.Errorf("failed to get notification events: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var event NotificationEvent
		var description, thresholdUnit sql.NullString
		
		if err := rows.Scan(&event.ID, &event.Category, &event.EventType, &event.Name, &description, &event.DefaultEnabled, &event.SupportsThreshold, &thresholdUnit, &event.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan event: %w", err)
		}
		
		if description.Valid {
			event.Description = &description.String
		}
		if thresholdUnit.Valid {
			event.ThresholdUnit = &thresholdUnit.String
		}
		
		events = append(events, event)
	}

	return events, nil
}

// GetEvent retrieves a single notification event by ID
func (s *service) GetEvent(id int64) (*NotificationEvent, error) {
	var event NotificationEvent
	var description, thresholdUnit sql.NullString

	err := s.sqlBuilder.Select("id", "category", "event_type", "name", "description", "default_enabled", "supports_threshold", "threshold_unit", "created_at").
		From("notification_events").
		Where(sq.Eq{"id": id}).
		RunWith(s.db).
		QueryRow().
		Scan(&event.ID, &event.Category, &event.EventType, &event.Name, &description, &event.DefaultEnabled, &event.SupportsThreshold, &thresholdUnit, &event.CreatedAt)
	
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get notification event: %w", err)
	}

	if description.Valid {
		event.Description = &description.String
	}
	if thresholdUnit.Valid {
		event.ThresholdUnit = &thresholdUnit.String
	}

	return &event, nil
}

// GetEventByType retrieves a notification event by category and type
func (s *service) GetEventByType(category, eventType string) (*NotificationEvent, error) {
	var event NotificationEvent
	var description, thresholdUnit sql.NullString

	err := s.sqlBuilder.Select("id", "category", "event_type", "name", "description", "default_enabled", "supports_threshold", "threshold_unit", "created_at").
		From("notification_events").
		Where(sq.And{
			sq.Eq{"category": category},
			sq.Eq{"event_type": eventType},
		}).
		RunWith(s.db).
		QueryRow().
		Scan(&event.ID, &event.Category, &event.EventType, &event.Name, &description, &event.DefaultEnabled, &event.SupportsThreshold, &thresholdUnit, &event.CreatedAt)
	
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get notification event: %w", err)
	}

	if description.Valid {
		event.Description = &description.String
	}
	if thresholdUnit.Valid {
		event.ThresholdUnit = &thresholdUnit.String
	}

	return &event, nil
}

// CreateRule creates a new notification rule
func (s *service) CreateRule(input NotificationRuleInput) (*NotificationRule, error) {
	now := time.Now()
	enabled := false
	if input.Enabled != nil {
		enabled = *input.Enabled
	}

	query := s.sqlBuilder.Insert("notification_rules").
		Columns("channel_id", "event_id", "enabled", "threshold_value", "threshold_operator", "created_at", "updated_at").
		Values(input.ChannelID, input.EventID, enabled, input.ThresholdValue, input.ThresholdOperator, now, now)

	if s.config.Type == config.Postgres {
		query = query.Suffix("RETURNING id")
	}

	if s.config.Type == config.SQLite {
		result, err := query.RunWith(s.db).Exec()
		if err != nil {
			return nil, fmt.Errorf("failed to create notification rule: %w", err)
		}

		id, err := result.LastInsertId()
		if err != nil {
			return nil, fmt.Errorf("failed to get last insert id: %w", err)
		}

		return &NotificationRule{
			ID:                id,
			ChannelID:         input.ChannelID,
			EventID:           input.EventID,
			Enabled:           enabled,
			ThresholdValue:    input.ThresholdValue,
			ThresholdOperator: input.ThresholdOperator,
			CreatedAt:         now,
			UpdatedAt:         now,
		}, nil
	} else {
		// PostgreSQL
		var id int64
		err := query.RunWith(s.db).QueryRow().Scan(&id)
		if err != nil {
			return nil, fmt.Errorf("failed to create notification rule: %w", err)
		}

		return &NotificationRule{
			ID:                id,
			ChannelID:         input.ChannelID,
			EventID:           input.EventID,
			Enabled:           enabled,
			ThresholdValue:    input.ThresholdValue,
			ThresholdOperator: input.ThresholdOperator,
			CreatedAt:         now,
			UpdatedAt:         now,
		}, nil
	}
}

// GetRules retrieves all notification rules
func (s *service) GetRules() ([]NotificationRule, error) {
	var rules []NotificationRule

	rows, err := s.sqlBuilder.Select(
		"r.id", "r.channel_id", "r.event_id", "r.enabled", "r.threshold_value", "r.threshold_operator", "r.created_at", "r.updated_at",
		"c.id", "c.name", "c.url", "c.enabled", "c.created_at", "c.updated_at",
		"e.id", "e.category", "e.event_type", "e.name", "e.description", "e.default_enabled", "e.supports_threshold", "e.threshold_unit", "e.created_at",
	).
		From("notification_rules r").
		LeftJoin("notification_channels c ON r.channel_id = c.id").
		LeftJoin("notification_events e ON r.event_id = e.id").
		OrderBy("r.created_at DESC").
		RunWith(s.db).
		Query()
	if err != nil {
		return nil, fmt.Errorf("failed to get notification rules: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var rule NotificationRule
		var channel NotificationChannel
		var event NotificationEvent
		var thresholdValue sql.NullFloat64
		var thresholdOperator, eventDescription, eventThresholdUnit sql.NullString

		err := rows.Scan(
			&rule.ID, &rule.ChannelID, &rule.EventID, &rule.Enabled, &thresholdValue, &thresholdOperator, &rule.CreatedAt, &rule.UpdatedAt,
			&channel.ID, &channel.Name, &channel.URL, &channel.Enabled, &channel.CreatedAt, &channel.UpdatedAt,
			&event.ID, &event.Category, &event.EventType, &event.Name, &eventDescription, &event.DefaultEnabled, &event.SupportsThreshold, &eventThresholdUnit, &event.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan rule: %w", err)
		}

		if thresholdValue.Valid {
			rule.ThresholdValue = &thresholdValue.Float64
		}
		if thresholdOperator.Valid {
			rule.ThresholdOperator = &thresholdOperator.String
		}
		if eventDescription.Valid {
			event.Description = &eventDescription.String
		}
		if eventThresholdUnit.Valid {
			event.ThresholdUnit = &eventThresholdUnit.String
		}

		rule.Channel = &channel
		rule.Event = &event

		rules = append(rules, rule)
	}

	return rules, nil
}

// GetRulesByChannel retrieves notification rules for a specific channel
func (s *service) GetRulesByChannel(channelID int64) ([]NotificationRule, error) {
	var rules []NotificationRule

	rows, err := s.sqlBuilder.Select(
		"r.id", "r.channel_id", "r.event_id", "r.enabled", "r.threshold_value", "r.threshold_operator", "r.created_at", "r.updated_at",
		"e.id", "e.category", "e.event_type", "e.name", "e.description", "e.default_enabled", "e.supports_threshold", "e.threshold_unit", "e.created_at",
	).
		From("notification_rules r").
		LeftJoin("notification_events e ON r.event_id = e.id").
		Where(sq.Eq{"r.channel_id": channelID}).
		OrderBy("e.category", "e.name").
		RunWith(s.db).
		Query()
	if err != nil {
		return nil, fmt.Errorf("failed to get notification rules: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var rule NotificationRule
		var event NotificationEvent
		var thresholdValue sql.NullFloat64
		var thresholdOperator, eventDescription, eventThresholdUnit sql.NullString

		err := rows.Scan(
			&rule.ID, &rule.ChannelID, &rule.EventID, &rule.Enabled, &thresholdValue, &thresholdOperator, &rule.CreatedAt, &rule.UpdatedAt,
			&event.ID, &event.Category, &event.EventType, &event.Name, &eventDescription, &event.DefaultEnabled, &event.SupportsThreshold, &eventThresholdUnit, &event.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan rule: %w", err)
		}

		if thresholdValue.Valid {
			rule.ThresholdValue = &thresholdValue.Float64
		}
		if thresholdOperator.Valid {
			rule.ThresholdOperator = &thresholdOperator.String
		}
		if eventDescription.Valid {
			event.Description = &eventDescription.String
		}
		if eventThresholdUnit.Valid {
			event.ThresholdUnit = &eventThresholdUnit.String
		}

		rule.Event = &event

		rules = append(rules, rule)
	}

	return rules, nil
}

// UpdateRule updates a notification rule
func (s *service) UpdateRule(id int64, input NotificationRuleInput) (*NotificationRule, error) {
	update := s.sqlBuilder.Update("notification_rules").
		Set("updated_at", time.Now()).
		Where(sq.Eq{"id": id})

	if input.Enabled != nil {
		update = update.Set("enabled", *input.Enabled)
	}
	if input.ThresholdValue != nil {
		update = update.Set("threshold_value", *input.ThresholdValue)
	}
	if input.ThresholdOperator != nil {
		update = update.Set("threshold_operator", *input.ThresholdOperator)
	}

	result, err := update.RunWith(s.db).Exec()
	if err != nil {
		return nil, fmt.Errorf("failed to update notification rule: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return nil, ErrNotFound
	}

	// Get the updated rule
	var rule NotificationRule
	err = s.sqlBuilder.Select("id", "channel_id", "event_id", "enabled", "threshold_value", "threshold_operator", "created_at", "updated_at").
		From("notification_rules").
		Where(sq.Eq{"id": id}).
		RunWith(s.db).
		QueryRow().
		Scan(&rule.ID, &rule.ChannelID, &rule.EventID, &rule.Enabled, &rule.ThresholdValue, &rule.ThresholdOperator, &rule.CreatedAt, &rule.UpdatedAt)
	
	if err != nil {
		return nil, fmt.Errorf("failed to get updated rule: %w", err)
	}

	return &rule, nil
}

// GetRule retrieves a single notification rule by ID
func (s *service) GetRule(id int64) (*NotificationRule, error) {
	var rule NotificationRule
	var thresholdValue sql.NullFloat64
	var thresholdOperator sql.NullString

	err := s.sqlBuilder.Select("id", "channel_id", "event_id", "enabled", "threshold_value", "threshold_operator", "created_at", "updated_at").
		From("notification_rules").
		Where(sq.Eq{"id": id}).
		RunWith(s.db).
		QueryRow().
		Scan(&rule.ID, &rule.ChannelID, &rule.EventID, &rule.Enabled, &thresholdValue, &thresholdOperator, &rule.CreatedAt, &rule.UpdatedAt)
	
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get notification rule: %w", err)
	}

	if thresholdValue.Valid {
		rule.ThresholdValue = &thresholdValue.Float64
	}
	if thresholdOperator.Valid {
		rule.ThresholdOperator = &thresholdOperator.String
	}

	return &rule, nil
}

// DeleteRule deletes a notification rule
func (s *service) DeleteRule(id int64) error {
	result, err := s.sqlBuilder.Delete("notification_rules").
		Where(sq.Eq{"id": id}).
		RunWith(s.db).
		Exec()
	
	if err != nil {
		return fmt.Errorf("failed to delete notification rule: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

// GetRulesByEvent retrieves all notification rules for a specific event
func (s *service) GetRulesByEvent(eventID int64) ([]NotificationRule, error) {
	var rules []NotificationRule

	rows, err := s.sqlBuilder.Select(
		"r.id", "r.channel_id", "r.event_id", "r.enabled", "r.threshold_value", "r.threshold_operator", "r.created_at", "r.updated_at",
		"c.id", "c.name", "c.url", "c.enabled", "c.created_at", "c.updated_at",
	).
		From("notification_rules r").
		LeftJoin("notification_channels c ON r.channel_id = c.id").
		Where(sq.Eq{"r.event_id": eventID}).
		OrderBy("r.created_at DESC").
		RunWith(s.db).
		Query()
	if err != nil {
		return nil, fmt.Errorf("failed to get notification rules by event: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var rule NotificationRule
		var channel NotificationChannel
		var thresholdValue sql.NullFloat64
		var thresholdOperator sql.NullString

		err := rows.Scan(
			&rule.ID, &rule.ChannelID, &rule.EventID, &rule.Enabled, &thresholdValue, &thresholdOperator, &rule.CreatedAt, &rule.UpdatedAt,
			&channel.ID, &channel.Name, &channel.URL, &channel.Enabled, &channel.CreatedAt, &channel.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan rule: %w", err)
		}

		if thresholdValue.Valid {
			rule.ThresholdValue = &thresholdValue.Float64
		}
		if thresholdOperator.Valid {
			rule.ThresholdOperator = &thresholdOperator.String
		}

		rule.Channel = &channel
		rules = append(rules, rule)
	}

	return rules, nil
}

// GetEnabledRulesForEvent retrieves enabled notification rules for a specific event
func (s *service) GetEnabledRulesForEvent(category, eventType string) ([]NotificationRule, error) {
	var rules []NotificationRule

	rows, err := s.sqlBuilder.Select(
		"r.id", "r.channel_id", "r.event_id", "r.enabled", "r.threshold_value", "r.threshold_operator", "r.created_at", "r.updated_at",
		"c.id", "c.name", "c.url", "c.enabled", "c.created_at", "c.updated_at",
	).
		From("notification_rules r").
		Join("notification_channels c ON r.channel_id = c.id").
		Join("notification_events e ON r.event_id = e.id").
		Where(sq.And{
			sq.Eq{"r.enabled": true},
			sq.Eq{"c.enabled": true},
			sq.Eq{"e.category": category},
			sq.Eq{"e.event_type": eventType},
		}).
		RunWith(s.db).
		Query()
	if err != nil {
		return nil, fmt.Errorf("failed to get enabled rules: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var rule NotificationRule
		var channel NotificationChannel
		var thresholdValue sql.NullFloat64
		var thresholdOperator sql.NullString

		err := rows.Scan(
			&rule.ID, &rule.ChannelID, &rule.EventID, &rule.Enabled, &thresholdValue, &thresholdOperator, &rule.CreatedAt, &rule.UpdatedAt,
			&channel.ID, &channel.Name, &channel.URL, &channel.Enabled, &channel.CreatedAt, &channel.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan rule: %w", err)
		}

		if thresholdValue.Valid {
			rule.ThresholdValue = &thresholdValue.Float64
		}
		if thresholdOperator.Valid {
			rule.ThresholdOperator = &thresholdOperator.String
		}

		rule.Channel = &channel
		rules = append(rules, rule)
	}

	return rules, nil
}

// LogNotification logs a notification attempt
func (s *service) LogNotification(channelID, eventID int64, success bool, errorMessage, payload *string) error {
	_, err := s.sqlBuilder.Insert("notification_history").
		Columns("channel_id", "event_id", "success", "error_message", "payload", "created_at").
		Values(channelID, eventID, success, errorMessage, payload, time.Now()).
		RunWith(s.db).
		Exec()
	
	if err != nil {
		return fmt.Errorf("failed to log notification: %w", err)
	}

	return nil
}

// CheckThreshold checks if a value meets the threshold criteria
func (s *service) CheckThreshold(rule *NotificationRule, value float64) bool {
	if rule.ThresholdValue == nil || rule.ThresholdOperator == nil {
		return true // No threshold configured, always pass
	}

	threshold := *rule.ThresholdValue
	operator := *rule.ThresholdOperator

	switch operator {
	case "gt":
		return value > threshold
	case "lt":
		return value < threshold
	case "eq":
		return value == threshold
	case "gte":
		return value >= threshold
	case "lte":
		return value <= threshold
	default:
		return true // Unknown operator, default to pass
	}
}

// GetNotificationHistory retrieves notification history with optional limit
func (s *service) GetNotificationHistory(limit int) ([]NotificationHistory, error) {
	query := s.sqlBuilder.Select(
		"h.id", "h.channel_id", "h.event_id", "h.success", "h.error_message", "h.payload", "h.created_at",
		"c.name", "c.url",
		"e.category", "e.event_type", "e.name",
	).
		From("notification_history h").
		LeftJoin("notification_channels c ON h.channel_id = c.id").
		LeftJoin("notification_events e ON h.event_id = e.id").
		OrderBy("h.created_at DESC")

	if limit > 0 {
		query = query.Limit(uint64(limit))
	}

	rows, err := query.RunWith(s.db).Query()
	if err != nil {
		return nil, fmt.Errorf("failed to get notification history: %w", err)
	}
	defer rows.Close()

	var history []NotificationHistory
	for rows.Next() {
		var h NotificationHistory
		var errorMessage, payload sql.NullString
		var channelName, channelURL string
		var eventCategory, eventType, eventName string

		err := rows.Scan(
			&h.ID, &h.ChannelID, &h.EventID, &h.Success, &errorMessage, &payload, &h.CreatedAt,
			&channelName, &channelURL,
			&eventCategory, &eventType, &eventName,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan history: %w", err)
		}

		if errorMessage.Valid {
			h.ErrorMessage = &errorMessage.String
		}
		if payload.Valid {
			h.Payload = &payload.String
		}

		// Optional: Add channel and event info to the history object
		// This would require extending the NotificationHistory struct

		history = append(history, h)
	}

	return history, nil
}