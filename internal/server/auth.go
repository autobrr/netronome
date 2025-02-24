// Copyright (c) 2024-2025, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/sessions"
	"github.com/rs/zerolog/log"

	"github.com/autobrr/netronome/internal/auth"
	"github.com/autobrr/netronome/internal/database"
	"github.com/autobrr/netronome/internal/server/encoder"
)

type AuthHandler struct {
	db            database.Service
	oidc          *auth.OIDCConfig
	cookieStore   *sessions.CookieStore
	sessionSecret string

	baseUrl string
}

func NewAuthHandler(db database.Service, oidc *auth.OIDCConfig, sessionSecret string, baseUrl string, cookieStore *sessions.CookieStore) *AuthHandler {
	return &AuthHandler{
		db:            db,
		oidc:          oidc,
		sessionSecret: sessionSecret,
		baseUrl:       baseUrl,
		cookieStore:   cookieStore,
	}
}

func (h *AuthHandler) Routes(r chi.Router) {
	r.Post("/login", h.Login)
	r.Post("/register", h.Register)

	r.Group(func(r chi.Router) {
		r.Use(IsAuthenticated(h.baseUrl, h.cookieStore))

		r.Get("/status", h.CheckRegistrationStatus)

		r.Post("/logout", h.Logout)
		r.Get("/verify", h.Verify)
		r.Get("/user", h.GetUserInfo)
	})

	r.Route("/oidc", func(r chi.Router) {
		r.Get("/login", h.handleOIDCLogin)
		r.Get("/callback", h.handleOIDCCallback)
	})
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
	//if h.sessionSecret == "" {
	//	h.sessionMutex.Lock()
	//	h.sessionTokens[signedToken] = true
	//	h.sessionMutex.Unlock()
	//}

	// TODO sessions handling

	log.Debug().
		Str("token", token).
		Str("signed_token", signedToken).
		Bool("secure", isSecure).
		Bool("memory_only", h.sessionSecret == "").
		Msg("Setting session cookie")

	//  Set domain to empty string to work with both localhost and IP addresses
	//http.SetCookie(w, &http.Cookie{
	//	Name:     "session",
	//	Value:    signedToken,
	//	MaxAge:   int((24 * time.Hour).Seconds()), // 24 hour expiry,
	//	Path:     h.baseUrl,
	//	Domain:   "",       // empty domain for maximum compatibility
	//	Secure:   isSecure, // secure flag only if HTTPS
	//	HttpOnly: true,     // httpOnly for security
	//})

	session, err := h.cookieStore.Get(r, "user_session")
	if err != nil {
		encoder.JSON(w, http.StatusUnauthorized, encoder.H{
			"error": "invalid credentials",
		})
		return
	}

	session.Values["authenticated"] = true
	session.Values["created_at"] = time.Now().Unix()
	//session.Values["id"] = user.ID
	//session.Values["username"] = user.Username
	session.Values["auth_method"] = "password"

	if created, ok := session.Values["created"].(int64); ok {
		// created is a unix timestamp MaxAge is in seconds
		maxAge := time.Duration(session.Options.MaxAge) * time.Second
		expires := time.Unix(created, 0).Add(maxAge)

		if time.Until(expires) <= 7*24*time.Hour { // 7 days
			//s.log.Info().Msgf("Cookie is expiring in less than 7 days on %s - extending session", expires.Format("2006-01-02 15:04:05"))

			session.Values["created"] = time.Now().Unix()

			// Call session.Save as needed - since it writes a header (the Set-Cookie
			// header), making sure you call it before writing out a body is important.
			// https://github.com/gorilla/sessions/issues/178#issuecomment-447674812
			if err := session.Save(r, w); err != nil {
				//s.log.Error().Err(err).Msgf("could not store session: %s", r.RemoteAddr)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
	}

	//session.Options.HttpOnly = true
	//session.Options.SameSite = http.SameSiteLaxMode
	session.Options.MaxAge = int((24 * time.Hour).Seconds()) // 24 hour expiry,
	//session.Options.Path = h.baseUrl
	//session.Options.Domain = "" // empty domain for maximum compatibility

	//if r.Header.Get("X-Forwarded-Proto") == "https" || strings.HasPrefix(r.Proto, "HTTPS") {
	//	session.Options.Secure = true
	//	session.Options.SameSite = http.SameSiteStrictMode
	//}

	if err := session.Save(r, w); err != nil {
		encoder.JSON(w, http.StatusInternalServerError, encoder.H{
			"error": "failed to save session",
		})
		return
	}
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

	//sessionToken, err := utils.GenerateSecureToken(32)
	//if err != nil {
	//	log.Error().Err(err).Msg("Failed to generate session token")
	//	encoder.JSON(w, http.StatusInternalServerError, encoder.H{
	//		"error": "failed to generate session token",
	//	})
	//	return
	//}
	//
	//h.refreshSession(w, r, sessionToken)

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

	session, err := h.cookieStore.Get(r, "user_session")
	if err != nil {
		encoder.JSON(w, http.StatusUnauthorized, encoder.H{
			"error": "invalid credentials",
		})
		return
	}

	session.Values["authenticated"] = true
	session.Values["created_at"] = time.Now().Unix()
	session.Values["id"] = user.ID
	session.Values["username"] = user.Username
	session.Values["auth_method"] = "password"

	session.Options.HttpOnly = true
	session.Options.SameSite = http.SameSiteLaxMode
	session.Options.MaxAge = int((24 * time.Hour).Seconds()) // 24 hour expiry,
	session.Options.Path = h.baseUrl
	session.Options.Domain = "" // empty domain for maximum compatibility

	if r.Header.Get("X-Forwarded-Proto") == "https" || strings.HasPrefix(r.Proto, "HTTPS") {
		session.Options.Secure = true
		session.Options.SameSite = http.SameSiteStrictMode
	}

	if err := session.Save(r, w); err != nil {
		encoder.JSON(w, http.StatusInternalServerError, encoder.H{
			"error": "failed to save session",
		})
		return
	}

	log.Debug().
		Str("username", user.Username).
		//Str("session_token", sessionToken).
		Msg("User logged in successfully")

	encoder.JSON(w, http.StatusOK, encoder.H{
		//"access_token": sessionToken,
		//"token_type":   "Bearer",
		"expires_in": int((24 * time.Hour).Seconds()),
		"user": encoder.H{
			"id":       user.ID,
			"username": user.Username,
		},
	})
}

func (h *AuthHandler) Verify(w http.ResponseWriter, r *http.Request) {
	session, ok := r.Context().Value("user_session").(*sessions.Session)
	if !ok || session == nil {
		encoder.JSON(w, http.StatusInternalServerError, encoder.H{
			"error": "no session found",
		})
		return
	}

	if !session.Values["authenticated"].(bool) == true {
		encoder.JSON(w, http.StatusUnauthorized, encoder.H{
			"error": "no session found",
		})
		return
	}

	//encoder.NoContent(w)

	//// check if it's a memory session
	//if strings.HasPrefix(signedToken, auth.MemoryOnlyPrefix) {
	//	if !h.isValidMemorySession(signedToken) {
	//		log.Debug().Msg("Invalid memory session")
	//		encoder.JSON(w, http.StatusInternalServerError, encoder.H{
	//			"error": "invalid memory session",
	//		})
	//		return
	//	}
	//} else {
	//	_, err := auth.VerifyToken(signedToken, h.sessionSecret)
	//	if err != nil {
	//		log.Debug().Err(err).Msg("Invalid session token")
	//		encoder.JSON(w, http.StatusInternalServerError, encoder.H{
	//			"error": "invalid session",
	//		})
	//		return
	//	}
	//}

	if h.oidc != nil {
		// TODO what to do here with OIDC??
		//if err := h.oidc.VerifyToken(r.Context(), signedToken); err == nil {
		//	h.refreshSession(w, r, signedToken)
		//	encoder.JSON(w, http.StatusOK, encoder.H{
		//		"message": "Token is valid",
		//		"type":    "oidc",
		//	})
		//	return
		//}
	}

	//var username string
	//// TODO move method to db
	//err = h.db.QueryRow(r.Context(), "SELECT username FROM users LIMIT 1").Scan(&username)
	//if err != nil {
	//	log.Debug().Err(err).Msg("Failed to verify session")
	//	encoder.JSON(w, http.StatusInternalServerError, encoder.H{
	//		"error": "invalid session",
	//	})
	//	return
	//}
	//
	//h.refreshSession(w, r, signedToken)

	encoder.JSON(w, http.StatusOK, encoder.H{
		"message": "Token is valid",
		"type":    "session",
	})
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	session, ok := r.Context().Value("user_session").(*sessions.Session)
	if !ok || session == nil {
		encoder.JSON(w, http.StatusInternalServerError, encoder.H{
			"error": "no session found",
		})
		return
	}

	session.Values["authenticated"] = false

	session.Options.MaxAge = -1

	if err := session.Save(r, w); err != nil {
		encoder.JSON(w, http.StatusInternalServerError, encoder.H{
			"error": "failed to save session",
		})
		return
	}

	encoder.NoContent(w)
	//encoder.JSON(w, http.StatusOK, encoder.H{"message": "Logged out successfully"})
}

func (h *AuthHandler) GetUserInfo(w http.ResponseWriter, r *http.Request) {
	session, ok := r.Context().Value("user_session").(*sessions.Session)
	if !ok || session == nil {
		encoder.JSON(w, http.StatusInternalServerError, encoder.H{
			"error": "no session found",
		})
		return
	}

	username, ok := session.Values["username"].(string)
	if !ok || username == "" {
		encoder.JSON(w, http.StatusInternalServerError, encoder.H{
			"error": "no session found",
		})
		return
	}

	//cookie, err := r.Cookie("session")
	//if err != nil {
	//	encoder.JSON(w, http.StatusInternalServerError, encoder.H{
	//		"error": "no session cookie found",
	//	})
	//	return
	//}
	//// TODO is this correct
	//signedToken := cookie.Value
	//
	//// check if it's a memory session
	//if strings.HasPrefix(signedToken, auth.MemoryOnlyPrefix) {
	//	if !h.isValidMemorySession(signedToken) {
	//		log.Debug().Msg("Invalid memory session")
	//		encoder.JSON(w, http.StatusInternalServerError, encoder.H{
	//			"error": "invalid memory session",
	//		})
	//		return
	//	}
	//} else {
	//	_, err := auth.VerifyToken(signedToken, h.sessionSecret)
	//	if err != nil {
	//		log.Debug().Err(err).Msg("Invalid session token")
	//		encoder.JSON(w, http.StatusInternalServerError, encoder.H{
	//			"error": "invalid session",
	//		})
	//		return
	//	}
	//}

	// try oidc first
	if h.oidc != nil {
		// TODO what to do here??
		signedToken := ""
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
