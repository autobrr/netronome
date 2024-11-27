// Copyright (c) 2024, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package middleware

import (
	"database/sql"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"

	"github.com/autobrr/netronome/internal/database"
)

func RequireAuth(db database.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		_, err := c.Cookie("session")
		if err != nil {
			log.Debug().Err(err).Msg("No session cookie found")
			_ = c.Error(fmt.Errorf("authentication required: %w", err))
			c.Abort()
			return
		}

		// For simplicity, we'll just get the first user since we only allow one user
		var username string
		err = db.QueryRow(c.Request.Context(), "SELECT username FROM users LIMIT 1").Scan(&username)
		if err != nil {
			if err == sql.ErrNoRows {
				// Log a different message for no users found
				log.Debug().Msg("No users found in the database")
			} else {
				log.Error().Err(err).Msg("Failed to validate session")
				_ = c.Error(fmt.Errorf("invalid session: %w", err))
				c.Abort()
				return
			}
		}

		c.Set("username", username)
		c.Next()
	}
}
