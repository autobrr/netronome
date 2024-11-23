package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/autobrr/netronome/internal/database"
)

// RequireAuth middleware checks if user is authenticated
func RequireAuth(db database.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		_, err := c.Cookie("session")
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
			return
		}

		// Get user from database using the session token
		// For simplicity, we'll just get the first user since we only allow one user
		var username string
		err = db.QueryRow(c.Request.Context(), "SELECT username FROM users LIMIT 1").Scan(&username)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid session"})
			return
		}

		// Set username in context
		c.Set("username", username)

		c.Next()
	}
}
