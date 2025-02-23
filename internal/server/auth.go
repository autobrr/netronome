// Copyright (c) 2024-2025, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"

	"github.com/autobrr/netronome/internal/auth"
	"github.com/autobrr/netronome/internal/database"
	"github.com/autobrr/netronome/internal/server/encoder"
	"github.com/autobrr/netronome/internal/utils"
)

type AuthHandler struct {
	db            database.Service
	oidc          *auth.OIDCConfig
	sessionTokens map[string]bool // Track valid memory sessions
	sessionMutex  sync.RWMutex
	sessionSecret string

	baseUrl string
}

func NewAuthHandler(db database.Service, oidc *auth.OIDCConfig, sessionSecret string, baseUrl string) *AuthHandler {
	return &AuthHandler{
		db:            db,
		oidc:          oidc,
		sessionTokens: make(map[string]bool),
		sessionSecret: sessionSecret,
		baseUrl:       baseUrl,
	}
}

func (h *AuthHandler) CheckRegistrationStatus(w http.ResponseWriter, r *http.Request) {
	var count int
	// TODO move method to db
	err := h.db.QueryRow(r.Context(), "SELECT COUNT(*) FROM users").Scan(&count)
	if err != nil && !isTableNotExistsError(err) {
		log.Error().Err(err).Msg("Failed to check existing users")
		encoder.JSON(w, http.StatusInternalServerError, encoder.H{
			"error": "Failed to check registration status",
		})
		return
	}

	encoder.JSON(w, http.StatusOK, encoder.H{
		"hasUsers":    count > 0,
		"oidcEnabled": h.oidc != nil,
	})
}

// refreshSession updates the session cookie with a new expiry time
func (h *AuthHandler) refreshSession(w http.ResponseWriter, r *http.Request, token string) {
	isSecure := r.Header.Get("X-Forwarded-Proto") == "https" || strings.HasPrefix(r.Proto, "HTTPS")

	var signedToken string
	if strings.Contains(token, ".") && h.sessionSecret != "" {
		// Token is already signed, verify and resign it
		rawToken, err := auth.VerifyToken(token, h.sessionSecret)
		if err != nil {
			log.Debug().Err(err).Msg("Failed to verify existing signed token")
			return
		}
		signedToken = auth.SignToken(rawToken, h.sessionSecret)
	} else {
		// Token is not signed yet, sign it
		signedToken = auth.SignToken(token, h.sessionSecret)
	}

	// Track memory-only sessions
	if h.sessionSecret == "" {
		h.sessionMutex.Lock()
		h.sessionTokens[signedToken] = true
		h.sessionMutex.Unlock()
	}

	log.Debug().
		Str("token", token).
		Str("signed_token", signedToken).
		Bool("secure", isSecure).
		Bool("memory_only", h.sessionSecret == "").
		Msg("Setting session cookie")

	//  Set domain to empty string to work with both localhost and IP addresses
	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    signedToken,
		MaxAge:   int((24 * time.Hour).Seconds()), // 24 hour expiry,
		Path:     h.baseUrl,
		Domain:   "",       // empty domain for maximum compatibility
		Secure:   isSecure, // secure flag only if HTTPS
		HttpOnly: true,     // httpOnly for security
	})
}

func (h *AuthHandler) isValidMemorySession(token string) bool {
	if !strings.HasPrefix(token, auth.MemoryOnlyPrefix) {
		return false
	}

	h.sessionMutex.RLock()
	valid := h.sessionTokens[token]
	h.sessionMutex.RUnlock()
	return valid
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var count int
	err := h.db.QueryRow(r.Context(), "SELECT COUNT(*) FROM users").Scan(&count)
	if err != nil && !isTableNotExistsError(err) {
		log.Error().Err(err).Msg("Failed to check existing users")
		encoder.JSON(w, http.StatusInternalServerError, encoder.H{
			"error": "Failed to check existing users",
		})
		return
	}

	if err == nil && count > 0 {
		encoder.JSON(w, http.StatusInternalServerError, encoder.H{
			"error": "registration disabled: user already exists",
		})
		return
	}

	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		encoder.JSON(w, http.StatusInternalServerError, encoder.H{
			"error": "could not decode request body",
		})
		return
	}

	//if err := auth.ValidatePassword(req.Password); err != nil {
	//	_ = c.Error(fmt.Errorf("invalid password: %w", err))
	//	return
	//}

	user, err := h.db.CreateUser(r.Context(), req.Username, req.Password)
	if err != nil {
		if errors.Is(err, database.ErrUserAlreadyExists) {
			encoder.JSON(w, http.StatusInternalServerError, encoder.H{
				"error": "username already exists",
			})
			return
		}
		log.Error().Err(err).Msg("Failed to create user")
		encoder.JSON(w, http.StatusInternalServerError, encoder.H{
			"error": "failed to create user",
		})
		return
	}

	sessionToken, err := utils.GenerateSecureToken(32)
	if err != nil {
		log.Error().Err(err).Msg("Failed to generate session token")
		encoder.JSON(w, http.StatusInternalServerError, encoder.H{
			"error": "failed to generate session token",
		})
		return
	}

	h.refreshSession(w, r, sessionToken)

	encoder.JSON(w, http.StatusCreated, encoder.H{
		"message": "User registered successfully",
		"user": encoder.H{
			"id":       user.ID,
			"username": user.Username,
		},
	})
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		encoder.JSON(w, http.StatusInternalServerError, encoder.H{
			"error": "could not decode request body",
		})
		return
	}

	user, err := h.db.GetUserByUsername(r.Context(), req.Username)
	if err != nil {
		if errors.Is(err, database.ErrUserNotFound) {
			encoder.JSON(w, http.StatusUnauthorized, encoder.H{
				"error": "invalid credentials",
			})
			return
		}
		log.Error().Err(err).Msg("Failed to get user")
		encoder.JSON(w, http.StatusInternalServerError, encoder.H{
			"error": "failed to get user",
		})
		return
	}

	if !h.db.ValidatePassword(user, req.Password) {
		encoder.JSON(w, http.StatusUnauthorized, encoder.H{
			"error": "invalid credentials",
		})
		return
	}

	sessionToken, err := utils.GenerateSecureToken(32)
	if err != nil {
		log.Error().Err(err).Msg("Failed to generate session token")
		encoder.JSON(w, http.StatusInternalServerError, encoder.H{
			"error": "failed to generate session token",
		})
		return
	}

	h.refreshSession(w, r, sessionToken)

	log.Debug().
		Str("username", user.Username).
		Str("session_token", sessionToken).
		Msg("User logged in successfully")

	encoder.JSON(w, http.StatusOK, encoder.H{
		"access_token": sessionToken,
		"token_type":   "Bearer",
		"expires_in":   int((24 * time.Hour).Seconds()),
		"user": encoder.H{
			"id":       user.ID,
			"username": user.Username,
		},
	})
}

func (h *AuthHandler) Verify(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session")
	if err != nil {
		log.Debug().Err(err).Msg("No session cookie found")
		encoder.JSON(w, http.StatusInternalServerError, encoder.H{
			"error": "no session found",
		})
		return
	}
	signedToken := cookie.Value

	// check if it's a memory session
	if strings.HasPrefix(signedToken, auth.MemoryOnlyPrefix) {
		if !h.isValidMemorySession(signedToken) {
			log.Debug().Msg("Invalid memory session")
			encoder.JSON(w, http.StatusInternalServerError, encoder.H{
				"error": "invalid memory session",
			})
			return
		}
	} else {
		_, err := auth.VerifyToken(signedToken, h.sessionSecret)
		if err != nil {
			log.Debug().Err(err).Msg("Invalid session token")
			encoder.JSON(w, http.StatusInternalServerError, encoder.H{
				"error": "invalid session",
			})
			return
		}
	}

	if h.oidc != nil {
		if err := h.oidc.VerifyToken(r.Context(), signedToken); err == nil {
			h.refreshSession(w, r, signedToken)
			encoder.JSON(w, http.StatusOK, encoder.H{
				"message": "Token is valid",
				"type":    "oidc",
			})
			return
		}
	}

	var username string
	// TODO move method to db
	err = h.db.QueryRow(r.Context(), "SELECT username FROM users LIMIT 1").Scan(&username)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to verify session")
		encoder.JSON(w, http.StatusInternalServerError, encoder.H{
			"error": "invalid session",
		})
		return
	}

	h.refreshSession(w, r, signedToken)

	encoder.JSON(w, http.StatusOK, encoder.H{
		"message": "Token is valid",
		"type":    "session",
	})
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	isSecure := r.Header.Get("X-Forwarded-Proto") == "https" || strings.HasPrefix(r.Proto, "HTTPS")

	// remove the session cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    "",
		MaxAge:   -1,
		Path:     h.baseUrl,
		Domain:   "",
		Secure:   isSecure,
		HttpOnly: true,
	})
	//
	// TODO use gorilla session cookiestore
	//h.sessionMutex.Lock()
	//if sessionToken, err := c.Cookie("session"); err == nil {
	//	delete(h.sessionTokens, sessionToken)
	//}
	//h.sessionMutex.Unlock()

	encoder.JSON(w, http.StatusOK, encoder.H{"message": "Logged out successfully"})
}

func (h *AuthHandler) GetUserInfo(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session")
	if err != nil {
		encoder.JSON(w, http.StatusInternalServerError, encoder.H{
			"error": "no session cookie found",
		})
		return
	}
	// TODO is this correct
	signedToken := cookie.Value

	// check if it's a memory session
	if strings.HasPrefix(signedToken, auth.MemoryOnlyPrefix) {
		if !h.isValidMemorySession(signedToken) {
			log.Debug().Msg("Invalid memory session")
			encoder.JSON(w, http.StatusInternalServerError, encoder.H{
				"error": "invalid memory session",
			})
			return
		}
	} else {
		_, err := auth.VerifyToken(signedToken, h.sessionSecret)
		if err != nil {
			log.Debug().Err(err).Msg("Invalid session token")
			encoder.JSON(w, http.StatusInternalServerError, encoder.H{
				"error": "invalid session",
			})
			return
		}
	}

	// try oidc first
	if h.oidc != nil {
		claims, err := h.oidc.GetClaims(r.Context(), signedToken)
		if err == nil {
			encoder.JSON(w, http.StatusOK, encoder.H{
				"user": encoder.H{
					"id":       0, // OIDC users don't have local IDs
					"username": claims.Subject,
				},
			})
			return
		}
	}

	// fall back to regular user lookup
	// TODO read from CTX
	//username := c.GetString("username")
	username := "username"
	if username == "" {
		encoder.JSON(w, http.StatusInternalServerError, encoder.H{
			"error": "no session found",
		})
		return
	}

	user, err := h.db.GetUserByUsername(r.Context(), username)
	if err != nil {
		encoder.JSON(w, http.StatusInternalServerError, encoder.H{
			"error": "failed to get user",
		})
		return
	}

	encoder.JSON(w, http.StatusOK, encoder.H{
		"user": encoder.H{
			"id":       user.ID,
			"username": user.Username,
		},
	})
}

func isTableNotExistsError(err error) bool {
	return err != nil && err.Error() == "SQL logic error: no such table: users (1)"
}

func RequireAuth(db database.Service, oidc *auth.OIDCConfig, sessionSecret string, handler *AuthHandler) gin.HandlerFunc {
	return func(c *gin.Context) {
		signedToken, err := c.Cookie("session")
		if err != nil {
			log.Debug().Err(err).Msg("No session cookie found")
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		// check if it's a memory session
		if strings.HasPrefix(signedToken, auth.MemoryOnlyPrefix) {
			if !handler.isValidMemorySession(signedToken) {
				log.Debug().Msg("Invalid memory session")
				c.AbortWithStatus(http.StatusUnauthorized)
				return
			}
		} else {
			_, err := auth.VerifyToken(signedToken, sessionSecret)
			if err != nil {
				log.Debug().Err(err).Msg("Invalid session token")
				c.AbortWithStatus(http.StatusUnauthorized)
				return
			}
		}

		// if oidc is configured, try to verify the token
		if oidc != nil {
			if err := oidc.VerifyToken(c.Request.Context(), signedToken); err == nil {
				c.Next()
				return
			}
		}

		// fall back to regular auth check
		var username string
		err = db.QueryRow(c.Request.Context(), "SELECT username FROM users LIMIT 1").Scan(&username)
		if err != nil {
			log.Debug().Err(err).Msg("Failed to find user")
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		log.Debug().
			Str("username", username).
			Bool("memory_only", strings.HasPrefix(signedToken, auth.MemoryOnlyPrefix)).
			Msg("User authenticated successfully")

		c.Set("username", username)
		c.Next()
	}
}

func (h *AuthHandler) storeSession(sessionToken string) error {
	h.sessionMutex.Lock()
	defer h.sessionMutex.Unlock()
	h.sessionTokens[sessionToken] = true
	return nil
}
