// Copyright (c) 2024-2025, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"

	"github.com/autobrr/netronome/internal/database"
	"github.com/autobrr/netronome/internal/logger"
	"github.com/autobrr/netronome/internal/server/encoder"
)

type IperfHandler struct {
	log zerolog.Logger
	db  database.Service
}

func NewIperfHandler(db database.Service) *IperfHandler {
	return &IperfHandler{
		log: logger.Get().With().Str("module", "iperf_handler").Logger(),
		db:  db,
	}
}

func (h *IperfHandler) Routes(r chi.Router) {
	r.Post("/servers", h.SaveServer)
	r.Get("/servers", h.GetServers)
	r.Delete("/servers/{serverID}", h.DeleteServer)
}

func (h *IperfHandler) SaveServer(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name" binding:"required"`
		Host string `json:"host" binding:"required"`
		Port int    `json:"port" binding:"required"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		encoder.JSON(w, http.StatusBadRequest, encoder.H{
			"error": "could not decode request body",
		})
		return
	}

	server, err := h.db.SaveIperfServer(r.Context(), req.Name, req.Host, req.Port)
	if err != nil {
		h.log.Error().Err(err).Msg("Failed to save iperf server")
		encoder.JSON(w, http.StatusInternalServerError, encoder.H{
			"error": "failed to save iperf server",
		})
		return
	}

	encoder.JSON(w, http.StatusCreated, encoder.H{"message": "Iperf server saved successfully", "server": server})
}

func (h *IperfHandler) GetServers(w http.ResponseWriter, r *http.Request) {
	servers, err := h.db.GetIperfServers(r.Context())
	if err != nil {
		h.log.Error().Err(err).Msg("Failed to get iperf servers")
		encoder.JSON(w, http.StatusInternalServerError, encoder.H{
			"error": "failed to get iperf servers",
		})
		return
	}

	encoder.JSON(w, http.StatusOK, servers)
}

func (h *IperfHandler) DeleteServer(w http.ResponseWriter, r *http.Request) {
	serverID, err := strconv.Atoi(chi.URLParam(r, "serverID"))
	if err != nil {
		encoder.JSON(w, http.StatusBadRequest, encoder.H{
			"error": "invalid server ID",
		})
		return
	}

	err = h.db.DeleteIperfServer(r.Context(), serverID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			encoder.JSON(w, http.StatusNotFound, encoder.H{
				"error": "server not found",
			})
			return
		}
		h.log.Error().Err(err).Int("id", serverID).Msg("Failed to delete iperf server")
		encoder.JSON(w, http.StatusInternalServerError, encoder.H{
			"error": "failed to delete iperf server",
		})
		return
	}

	encoder.JSON(w, http.StatusOK, encoder.H{"message": "Server deleted successfully"})
}
