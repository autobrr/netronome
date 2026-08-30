// Copyright (c) 2024-2026, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package server

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func (s *Server) handleLatestVersion(c *gin.Context) {
	if s.updateChecker == nil {
		c.Status(http.StatusNoContent)
		return
	}

	latest := s.updateChecker.Latest()
	if latest == nil {
		c.Status(http.StatusNoContent)
		return
	}
	c.JSON(http.StatusOK, struct {
		TagName     string `json:"tag_name"`
		Name        string `json:"name,omitempty"`
		HTMLURL     string `json:"html_url"`
		PublishedAt string `json:"published_at"`
	}{
		TagName:     latest.TagName,
		Name:        latest.Name,
		HTMLURL:     latest.HTMLURL,
		PublishedAt: latest.PublishedAt.UTC().Format(time.RFC3339),
	})
}
