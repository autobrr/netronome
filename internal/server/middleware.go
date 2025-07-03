// Copyright (c) 2024-2025, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// ErrorResponse represents the structure of error responses
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// ErrorHandlerMiddleware handles errors set by handlers and returns proper JSON responses
func ErrorHandlerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// Handle errors after request processing
		if len(c.Errors) > 0 {
			err := c.Errors.Last()
			
			// Determine status code if not already set
			status := c.Writer.Status()
			if status == http.StatusOK {
				status = http.StatusInternalServerError
			}

			// Send JSON error response
			c.JSON(status, ErrorResponse{
				Error:   "request_failed",
				Message: err.Error(),
			})
		}
	}
}