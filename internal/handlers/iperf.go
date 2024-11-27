package handlers

import (
	"database/sql"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"

	"github.com/autobrr/netronome/internal/database"
	"github.com/autobrr/netronome/internal/logger"
	"github.com/autobrr/netronome/internal/types"
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

func (h *IperfHandler) SaveServer(c *gin.Context) {
	var req struct {
		Name string `json:"name" binding:"required"`
		Host string `json:"host" binding:"required"`
		Port int    `json:"port" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	server, err := h.db.SaveIperfServer(c.Request.Context(), req.Name, req.Host, req.Port)
	if err != nil {
		h.log.Error().Err(err).Msg("Failed to save iperf server")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save iperf server"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Iperf server saved successfully", "server": server})
}

func (h *IperfHandler) GetServers(c *gin.Context) {
	servers, err := h.db.GetIperfServers(c.Request.Context())
	if err != nil {
		h.log.Error().Err(err).Msg("Failed to get iperf servers")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get iperf servers"})
		return
	}

	// If no servers found, return empty array instead of null
	if servers == nil {
		servers = []types.SavedIperfServer{}
	}

	c.JSON(http.StatusOK, servers)
}

func (h *IperfHandler) DeleteServer(c *gin.Context) {
	id := c.Param("id")

	serverID, err := strconv.Atoi(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid server ID"})
		return
	}

	err = h.db.DeleteIperfServer(c.Request.Context(), serverID)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Server not found"})
			return
		}
		h.log.Error().Err(err).Int("id", serverID).Msg("Failed to delete iperf server")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete iperf server"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Server deleted successfully"})
}
