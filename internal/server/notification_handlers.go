// Copyright (c) 2024-2025, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package server

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"

	"github.com/autobrr/netronome/internal/database"
	"github.com/autobrr/netronome/internal/notifications"
)

// handleGetNotificationChannels retrieves all notification channels
func (s *Server) handleGetNotificationChannels(c *gin.Context) {
	channels, err := s.db.GetChannels()
	if err != nil {
		log.Error().Err(err).Msg("Failed to get notification channels")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get notification channels"})
		return
	}

	// Ensure we always return an array, never null
	if channels == nil {
		channels = []database.NotificationChannel{}
	}

	c.JSON(http.StatusOK, channels)
}

// handleCreateNotificationChannel creates a new notification channel
func (s *Server) handleCreateNotificationChannel(c *gin.Context) {
	var input database.NotificationChannelInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	channel, err := s.db.CreateChannel(input)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create notification channel")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create notification channel"})
		return
	}

	c.JSON(http.StatusCreated, channel)
}

// handleUpdateNotificationChannel updates a notification channel
func (s *Server) handleUpdateNotificationChannel(c *gin.Context) {
	channelID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid channel ID"})
		return
	}

	var input database.NotificationChannelInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	channel, err := s.db.UpdateChannel(channelID, input)
	if err != nil {
		log.Error().Err(err).Msg("Failed to update notification channel")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update notification channel"})
		return
	}

	c.JSON(http.StatusOK, channel)
}

// handleDeleteNotificationChannel deletes a notification channel
func (s *Server) handleDeleteNotificationChannel(c *gin.Context) {
	channelID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid channel ID"})
		return
	}

	if err := s.db.DeleteChannel(channelID); err != nil {
		log.Error().Err(err).Msg("Failed to delete notification channel")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete notification channel"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// handleGetNotificationEvents retrieves all notification events
func (s *Server) handleGetNotificationEvents(c *gin.Context) {
	category := c.Query("category")
	
	var events []database.NotificationEvent
	var err error
	
	if category != "" {
		events, err = s.db.GetEventsByCategory(category)
	} else {
		events, err = s.db.GetEvents()
	}
	
	if err != nil {
		log.Error().Err(err).Msg("Failed to get notification events")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get notification events"})
		return
	}

	// Ensure we always return an array, never null
	if events == nil {
		events = []database.NotificationEvent{}
	}

	c.JSON(http.StatusOK, events)
}

// handleGetNotificationRules retrieves notification rules
func (s *Server) handleGetNotificationRules(c *gin.Context) {
	channelIDStr := c.Query("channel_id")
	
	var rules []database.NotificationRule
	var err error
	
	if channelIDStr != "" {
		channelID, err := strconv.ParseInt(channelIDStr, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid channel ID"})
			return
		}
		rules, err = s.db.GetRulesByChannel(channelID)
	} else {
		rules, err = s.db.GetRules()
	}
	
	if err != nil {
		log.Error().Err(err).Msg("Failed to get notification rules")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get notification rules"})
		return
	}

	// Ensure we always return an array, never null
	if rules == nil {
		rules = []database.NotificationRule{}
	}

	c.JSON(http.StatusOK, rules)
}

// handleCreateNotificationRule creates a new notification rule
func (s *Server) handleCreateNotificationRule(c *gin.Context) {
	var input database.NotificationRuleInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	rule, err := s.db.CreateRule(input)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create notification rule")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create notification rule"})
		return
	}

	c.JSON(http.StatusCreated, rule)
}

// handleUpdateNotificationRule updates a notification rule
func (s *Server) handleUpdateNotificationRule(c *gin.Context) {
	ruleID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid rule ID"})
		return
	}

	var input database.NotificationRuleInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	rule, err := s.db.UpdateRule(ruleID, input)
	if err != nil {
		log.Error().Err(err).Msg("Failed to update notification rule")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update notification rule"})
		return
	}

	c.JSON(http.StatusOK, rule)
}

// handleDeleteNotificationRule deletes a notification rule
func (s *Server) handleDeleteNotificationRule(c *gin.Context) {
	ruleID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid rule ID"})
		return
	}

	if err := s.db.DeleteRule(ruleID); err != nil {
		log.Error().Err(err).Msg("Failed to delete notification rule")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete notification rule"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// handleTestNotification tests a notification channel
func (s *Server) handleTestNotification(c *gin.Context) {
	var req struct {
		ChannelID int64  `json:"channel_id"`
		URL       string `json:"url"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Get the URL to test
	var testURL string
	if req.URL != "" {
		testURL = req.URL
	} else if req.ChannelID > 0 {
		channel, err := s.db.GetChannel(req.ChannelID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Channel not found"})
			return
		}
		testURL = channel.URL
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Either channel_id or url must be provided"})
		return
	}

	// Create a temporary notifier to test the URL
	notifier, err := notifications.NewNotifierFromURLs([]string{testURL})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid notification URL"})
		return
	}

	// Send test notification
	if err := notifier.SendTestNotification(); err != nil {
		log.Error().Err(err).Str("url", testURL).Msg("Failed to send test notification")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send test notification", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Test notification sent successfully"})
}

// handleGetNotificationHistory retrieves notification history
func (s *Server) handleGetNotificationHistory(c *gin.Context) {
	limit := 100 // Default limit
	if limitStr := c.Query("limit"); limitStr != "" {
		parsedLimit, err := strconv.Atoi(limitStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid limit"})
			return
		}
		limit = parsedLimit
	}

	history, err := s.db.GetNotificationHistory(limit)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get notification history")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get notification history"})
		return
	}

	c.JSON(http.StatusOK, history)
}