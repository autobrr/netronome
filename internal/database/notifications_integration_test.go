// Copyright (c) 2024-2025, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package database

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNotificationChannel_CRUD(t *testing.T) {
	RunTestWithBothDatabases(t, func(t *testing.T, td *TestDatabase) {
		ctx := context.Background()
		_ = ctx

		// Create channel
		input := NotificationChannelInput{
			Name:    "Test Channel",
			URL:     "https://discord.com/api/webhooks/123456",
			Enabled: boolPtr(true),
		}

		created, err := td.Service.CreateChannel(input)
		require.NoError(t, err)
		require.NotNil(t, created)
		assert.Greater(t, created.ID, int64(0))
		assert.Equal(t, input.Name, created.Name)
		assert.Equal(t, input.URL, created.URL)
		assert.True(t, created.Enabled)
		assert.NotZero(t, created.CreatedAt)
		assert.NotZero(t, created.UpdatedAt)

		// Get channel by ID
		retrieved, err := td.Service.GetChannel(created.ID)
		require.NoError(t, err)
		require.NotNil(t, retrieved)
		assert.Equal(t, created.ID, retrieved.ID)
		assert.Equal(t, created.Name, retrieved.Name)

		// Update channel
		updateInput := NotificationChannelInput{
			Name:    "Updated Channel",
			URL:     "https://slack.com/api/webhooks/789",
			Enabled: boolPtr(false),
		}

		updated, err := td.Service.UpdateChannel(created.ID, updateInput)
		require.NoError(t, err)
		assert.Equal(t, "Updated Channel", updated.Name)
		assert.False(t, updated.Enabled)

		// Delete channel
		err = td.Service.DeleteChannel(created.ID)
		require.NoError(t, err)

		// Verify deletion
		deleted, err := td.Service.GetChannel(created.ID)
		assert.Error(t, err)
		assert.Nil(t, deleted)
	})
}

func TestNotificationChannel_GetAll(t *testing.T) {
	RunTestWithBothDatabases(t, func(t *testing.T, td *TestDatabase) {
		ctx := context.Background()
		_ = ctx

		// Create multiple channels
		channels := []NotificationChannelInput{
			{
				Name:    "Discord",
				URL:     "https://discord.com/webhook1",
				Enabled: boolPtr(true),
			},
			{
				Name:    "Slack",
				URL:     "https://slack.com/webhook1",
				Enabled: boolPtr(true),
			},
			{
				Name:    "Disabled Channel",
				URL:     "https://example.com/webhook",
				Enabled: boolPtr(false),
			},
		}

		for _, ch := range channels {
			_, err := td.Service.CreateChannel(ch)
			require.NoError(t, err)
		}

		// Get all channels
		all, err := td.Service.GetChannels()
		require.NoError(t, err)
		assert.Len(t, all, 3)

		// Get enabled channels only
		enabled, err := td.Service.GetEnabledChannels()
		require.NoError(t, err)
		assert.Len(t, enabled, 2)

		// Verify all enabled channels are actually enabled
		for _, ch := range enabled {
			assert.True(t, ch.Enabled)
		}
	})
}

func TestNotificationRule_CRUD(t *testing.T) {
	RunTestWithBothDatabases(t, func(t *testing.T, td *TestDatabase) {
		ctx := context.Background()
		_ = ctx

		// First create a channel
		channel, err := td.Service.CreateChannel(NotificationChannelInput{
			Name:    "Test Channel for Rules",
			URL:     "https://example.com/webhook",
			Enabled: boolPtr(true),
		})
		require.NoError(t, err)

		// Get a speedtest event
		event, err := td.Service.GetEventByType(NotificationCategorySpeedtest, NotificationEventSpeedtestDownloadLow)
		require.NoError(t, err)
		require.NotNil(t, event)

		// Create rule
		ruleInput := NotificationRuleInput{
			ChannelID:         channel.ID,
			EventID:           event.ID,
			Enabled:           boolPtr(true),
			ThresholdValue:    float64Ptr(100.0),
			ThresholdOperator: stringPtr("lt"),
		}

		created, err := td.Service.CreateRule(ruleInput)
		require.NoError(t, err)
		require.NotNil(t, created)
		assert.Greater(t, created.ID, int64(0))
		assert.Equal(t, channel.ID, created.ChannelID)
		assert.Equal(t, event.ID, created.EventID)
		assert.True(t, created.Enabled)
		assert.NotNil(t, created.ThresholdValue)
		assert.Equal(t, 100.0, *created.ThresholdValue)

		// Get rule by ID
		retrieved, err := td.Service.GetRule(created.ID)
		require.NoError(t, err)
		require.NotNil(t, retrieved)
		assert.Equal(t, created.ID, retrieved.ID)

		// Update rule
		updateInput := NotificationRuleInput{
			ChannelID:         channel.ID,
			EventID:           event.ID,
			Enabled:           boolPtr(false),
			ThresholdValue:    float64Ptr(200.0),
			ThresholdOperator: stringPtr("lt"),
		}

		updated, err := td.Service.UpdateRule(created.ID, updateInput)
		require.NoError(t, err)
		assert.False(t, updated.Enabled)
		assert.Equal(t, 200.0, *updated.ThresholdValue)

		// Delete rule
		err = td.Service.DeleteRule(created.ID)
		require.NoError(t, err)

		// Verify deletion
		deleted, err := td.Service.GetRule(created.ID)
		assert.Error(t, err)
		assert.Nil(t, deleted)
	})
}

func TestNotificationRule_GetByChannel(t *testing.T) {
	RunTestWithBothDatabases(t, func(t *testing.T, td *TestDatabase) {
		ctx := context.Background()
		_ = ctx

		// Create two channels
		channel1, err := td.Service.CreateChannel(NotificationChannelInput{
			Name:    "Channel 1",
			URL:     "https://example.com/webhook1",
			Enabled: boolPtr(true),
		})
		require.NoError(t, err)

		channel2, err := td.Service.CreateChannel(NotificationChannelInput{
			Name:    "Channel 2",
			URL:     "https://example.com/webhook2",
			Enabled: boolPtr(true),
		})
		require.NoError(t, err)

		// Get some events
		speedtestEvent, err := td.Service.GetEventByType(NotificationCategorySpeedtest, NotificationEventSpeedtestComplete)
		require.NoError(t, err)

		packetlossEvent, err := td.Service.GetEventByType(NotificationCategoryPacketLoss, NotificationEventPacketLossHigh)
		require.NoError(t, err)

		// Create rules for channel 1
		_, err = td.Service.CreateRule(NotificationRuleInput{
			ChannelID: channel1.ID,
			EventID:   speedtestEvent.ID,
			Enabled:   boolPtr(true),
		})
		require.NoError(t, err)

		_, err = td.Service.CreateRule(NotificationRuleInput{
			ChannelID: channel1.ID,
			EventID:   packetlossEvent.ID,
			Enabled:   boolPtr(true),
		})
		require.NoError(t, err)

		// Create rule for channel 2
		_, err = td.Service.CreateRule(NotificationRuleInput{
			ChannelID: channel2.ID,
			EventID:   speedtestEvent.ID,
			Enabled:   boolPtr(true),
		})
		require.NoError(t, err)

		// Get rules by channel
		channel1Rules, err := td.Service.GetRulesByChannel(channel1.ID)
		require.NoError(t, err)
		assert.Len(t, channel1Rules, 2)

		channel2Rules, err := td.Service.GetRulesByChannel(channel2.ID)
		require.NoError(t, err)
		assert.Len(t, channel2Rules, 1)
	})
}

func TestNotificationEvent_GetByCategory(t *testing.T) {
	RunTestWithBothDatabases(t, func(t *testing.T, td *TestDatabase) {
		ctx := context.Background()
		_ = ctx

		// Get speedtest events
		speedtestEvents, err := td.Service.GetEventsByCategory(NotificationCategorySpeedtest)
		require.NoError(t, err)
		assert.NotEmpty(t, speedtestEvents)

		// Verify all are speedtest category
		for _, event := range speedtestEvents {
			assert.Equal(t, NotificationCategorySpeedtest, event.Category)
		}

		// Get packet loss events
		packetlossEvents, err := td.Service.GetEventsByCategory(NotificationCategoryPacketLoss)
		require.NoError(t, err)
		assert.NotEmpty(t, packetlossEvents)

		// Get agent events
		agentEvents, err := td.Service.GetEventsByCategory(NotificationCategoryAgent)
		require.NoError(t, err)
		assert.NotEmpty(t, agentEvents)
	})
}

func TestNotificationRule_EnabledNotifications(t *testing.T) {
	RunTestWithBothDatabases(t, func(t *testing.T, td *TestDatabase) {
		ctx := context.Background()
		_ = ctx

		// Create channels
		enabledChannel, err := td.Service.CreateChannel(NotificationChannelInput{
			Name:    "Enabled Channel",
			URL:     "https://example.com/enabled",
			Enabled: boolPtr(true),
		})
		require.NoError(t, err)

		disabledChannel, err := td.Service.CreateChannel(NotificationChannelInput{
			Name:    "Disabled Channel",
			URL:     "https://example.com/disabled",
			Enabled: boolPtr(false),
		})
		require.NoError(t, err)

		// Get different events for testing
		event1, err := td.Service.GetEventByType(NotificationCategorySpeedtest, NotificationEventSpeedtestComplete)
		require.NoError(t, err)

		event2, err := td.Service.GetEventByType(NotificationCategorySpeedtest, NotificationEventSpeedtestFailed)
		require.NoError(t, err)

		// Create rules with different combinations
		// Enabled channel, enabled rule for event1
		_, err = td.Service.CreateRule(NotificationRuleInput{
			ChannelID: enabledChannel.ID,
			EventID:   event1.ID,
			Enabled:   boolPtr(true),
		})
		require.NoError(t, err)

		// Disabled channel, enabled rule for event1
		_, err = td.Service.CreateRule(NotificationRuleInput{
			ChannelID: disabledChannel.ID,
			EventID:   event1.ID,
			Enabled:   boolPtr(true),
		})
		require.NoError(t, err)

		// Enabled channel, disabled rule for event2
		_, err = td.Service.CreateRule(NotificationRuleInput{
			ChannelID: enabledChannel.ID,
			EventID:   event2.ID,
			Enabled:   boolPtr(false),
		})
		require.NoError(t, err)

		// Get enabled rules for event1
		enabledRules, err := td.Service.GetEnabledRulesForEvent(event1.Category, event1.EventType)
		require.NoError(t, err)

		// Should only have one truly enabled rule (enabled channel + enabled rule)
		assert.Len(t, enabledRules, 1)
		assert.Equal(t, enabledChannel.ID, enabledRules[0].ChannelID)
		assert.True(t, enabledRules[0].Enabled)

		// Get enabled rules for event2
		enabledRulesEvent2, err := td.Service.GetEnabledRulesForEvent(event2.Category, event2.EventType)
		require.NoError(t, err)

		// Should have no enabled rules for event2 (rule is disabled)
		assert.Len(t, enabledRulesEvent2, 0)
	})
}

func TestNotificationHistory_Create(t *testing.T) {
	RunTestWithBothDatabases(t, func(t *testing.T, td *TestDatabase) {
		ctx := context.Background()
		_ = ctx

		// Create channel
		channel, err := td.Service.CreateChannel(NotificationChannelInput{
			Name:    "History Test Channel",
			URL:     "https://example.com/webhook",
			Enabled: boolPtr(true),
		})
		require.NoError(t, err)

		// Get event
		event, err := td.Service.GetEventByType(NotificationCategorySpeedtest, NotificationEventSpeedtestComplete)
		require.NoError(t, err)

		// Log notification attempts
		// Success
		payload := `{"message": "Test completed successfully"}`
		err = td.Service.LogNotification(channel.ID, event.ID, true, nil, &payload)
		require.NoError(t, err)

		// Failure
		errorMsg := "Failed to send notification: timeout"
		err = td.Service.LogNotification(channel.ID, event.ID, false, &errorMsg, &payload)
		require.NoError(t, err)

		// Get notification history
		history, err := td.Service.GetNotificationHistory(10)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(history), 2)

		// Find our entries
		var successEntry, failEntry *NotificationHistory
		for i := range history {
			if history[i].ChannelID == channel.ID && history[i].EventID == event.ID {
				if history[i].Success {
					successEntry = &history[i]
				} else {
					failEntry = &history[i]
				}
			}
		}

		require.NotNil(t, successEntry)
		require.NotNil(t, failEntry)

		assert.True(t, successEntry.Success)
		assert.Nil(t, successEntry.ErrorMessage)
		assert.NotNil(t, successEntry.Payload)

		assert.False(t, failEntry.Success)
		assert.NotNil(t, failEntry.ErrorMessage)
		assert.Equal(t, errorMsg, *failEntry.ErrorMessage)
	})
}

func TestNotification_CheckThreshold(t *testing.T) {
	RunTestWithBothDatabases(t, func(t *testing.T, td *TestDatabase) {
		ctx := context.Background()
		_ = ctx

		testCases := []struct {
			name      string
			operator  string
			threshold float64
			value     float64
			expected  bool
		}{
			{"Greater Than - True", "gt", 100.0, 150.0, true},
			{"Greater Than - False", "gt", 100.0, 50.0, false},
			{"Less Than - True", "lt", 100.0, 50.0, true},
			{"Less Than - False", "lt", 100.0, 150.0, false},
			{"Equal - True", "eq", 100.0, 100.0, true},
			{"Equal - False", "eq", 100.0, 99.0, false},
			{"Greater or Equal - True (Greater)", "gte", 100.0, 150.0, true},
			{"Greater or Equal - True (Equal)", "gte", 100.0, 100.0, true},
			{"Greater or Equal - False", "gte", 100.0, 99.0, false},
			{"Less or Equal - True (Less)", "lte", 100.0, 50.0, true},
			{"Less or Equal - True (Equal)", "lte", 100.0, 100.0, true},
			{"Less or Equal - False", "lte", 100.0, 101.0, false},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				rule := &NotificationRule{
					ThresholdValue:    &tc.threshold,
					ThresholdOperator: &tc.operator,
				}

				result := td.Service.CheckThreshold(rule, tc.value)
				assert.Equal(t, tc.expected, result)
			})
		}

		// Test with nil threshold - should always return true
		ruleNoThreshold := &NotificationRule{
			ThresholdValue:    nil,
			ThresholdOperator: nil,
		}
		assert.True(t, td.Service.CheckThreshold(ruleNoThreshold, 999.0))
	})
}

// Helper function for float64 pointers
func float64Ptr(f float64) *float64 {
	return &f
}
