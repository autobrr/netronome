// Copyright (c) 2024, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"

	"github.com/autobrr/netronome/internal/auth"
	"github.com/autobrr/netronome/internal/database"
)

func RequireAuth(db database.Service, oidc *auth.OIDCConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionToken, err := c.Cookie("session")
		if err != nil {
			log.Debug().Err(err).Msg("No session cookie found")
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		// If OIDC is configured, try to verify the token
		if oidc != nil {
			if err := oidc.VerifyToken(c.Request.Context(), sessionToken); err == nil {
				c.Next()
				return
			}
		}

		// Fall back to regular auth check
		var username string
		err = db.QueryRow(c.Request.Context(), "SELECT username FROM users LIMIT 1").Scan(&username)
		if err != nil {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		c.Set("username", username)
		c.Next()
	}
}
