// Copyright (c) 2024, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/autobrr/netronome/internal/database"
)

func RequireAuth(db database.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		_, err := c.Cookie("session")
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
			return
		}

		// For simplicity, we'll just get the first user since we only allow one user
		var username string
		err = db.QueryRow(c.Request.Context(), "SELECT username FROM users LIMIT 1").Scan(&username)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid session"})
			return
		}

		c.Set("username", username)

		c.Next()
	}
}
