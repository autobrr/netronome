// Copyright (c) 2024-2026, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package handlers

import (
	"errors"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"

	"github.com/autobrr/netronome/internal/database"
	"github.com/autobrr/netronome/internal/dnsmonitor"
	"github.com/autobrr/netronome/internal/scheduler"
	"github.com/autobrr/netronome/internal/types"
)

// DNSHandler handles DNS monitoring endpoints
type DNSHandler struct {
	db        database.Service
	service   *dnsmonitor.Service
	scheduler scheduler.Service
}

func NewDNSHandler(db database.Service, service *dnsmonitor.Service, scheduler scheduler.Service) *DNSHandler {
	return &DNSHandler{db: db, service: service, scheduler: scheduler}
}

// dnsMonitorRequest is the monitor as the API takes it. Enabled is a pointer so
// that a body without the field keeps the default (a new monitor runs) instead
// of the zero value (a monitor that never runs).
type dnsMonitorRequest struct {
	Host       string `json:"host"`
	Name       string `json:"name"`
	Protocol   string `json:"protocol"`
	Query      string `json:"query"`
	RecordType string `json:"recordType"`
	Interval   string `json:"interval"`
	Enabled    *bool  `json:"enabled"`
}

// monitor applies the request to a monitor, with enabled as the fallback for a
// request that does not carry the field.
func (r dnsMonitorRequest) monitor(enabled bool) types.DNSMonitor {
	if r.Enabled != nil {
		enabled = *r.Enabled
	}

	return types.DNSMonitor{
		Host:       r.Host,
		Name:       r.Name,
		Protocol:   r.Protocol,
		Query:      r.Query,
		RecordType: r.RecordType,
		Interval:   r.Interval,
		Enabled:    enabled,
	}
}

// normalizeDNSMonitor fills in the defaults and reports the first invalid
// field. Its error text goes straight to the caller.
func normalizeDNSMonitor(monitor *types.DNSMonitor) error {
	monitor.Host = strings.TrimSpace(monitor.Host)
	if monitor.Host == "" {
		return errors.New("Host is required")
	}

	monitor.Query = strings.TrimSpace(monitor.Query)
	if monitor.Query == "" {
		monitor.Query = "google.com"
	}

	monitor.Protocol = strings.ToLower(strings.TrimSpace(monitor.Protocol))
	if monitor.Protocol == "" {
		monitor.Protocol = types.DNSProtocolUDP
	}
	switch monitor.Protocol {
	case types.DNSProtocolUDP, types.DNSProtocolTCP, types.DNSProtocolDoT:
	default:
		return errors.New("Protocol must be one of: udp, tcp, dot")
	}

	monitor.RecordType = strings.ToUpper(strings.TrimSpace(monitor.RecordType))
	if monitor.RecordType == "" {
		monitor.RecordType = "A"
	}
	if !slices.Contains(dnsmonitor.RecordTypes, monitor.RecordType) {
		return errors.New("Record type must be one of: " + strings.Join(dnsmonitor.RecordTypes, ", "))
	}

	if strings.TrimSpace(monitor.Interval) == "" {
		monitor.Interval = "60s"
	}

	return nil
}

// GetMonitors returns all DNS monitors
func (h *DNSHandler) GetMonitors(c *gin.Context) {
	monitors, err := h.db.GetDNSMonitors()
	if err != nil {
		log.Error().Err(err).Msg("Failed to get dns monitors")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get monitors"})
		return
	}

	c.JSON(http.StatusOK, monitors)
}

// CreateMonitor creates a new DNS monitor
func (h *DNSHandler) CreateMonitor(c *gin.Context) {
	var request dnsMonitorRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// a new monitor runs unless the caller says otherwise
	monitor := request.monitor(true)
	if err := normalizeDNSMonitor(&monitor); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if nextRun := h.scheduler.CalculateNextRun(monitor.Interval, time.Now()); !nextRun.IsZero() {
		monitor.NextRun = &nextRun
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid interval"})
		return
	}

	created, err := h.db.CreateDNSMonitor(&monitor)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create dns monitor")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create monitor"})
		return
	}

	c.JSON(http.StatusCreated, created)
}

// UpdateMonitor updates an existing DNS monitor
func (h *DNSHandler) UpdateMonitor(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid monitor ID"})
		return
	}

	var request dnsMonitorRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// keep the scheduling fields the user does not send
	monitor, err := h.db.GetDNSMonitor(id)
	if err != nil {
		if err == database.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Monitor not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get monitor"})
		return
	}

	updateData := request.monitor(monitor.Enabled)
	if err := normalizeDNSMonitor(&updateData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	monitor.Host = updateData.Host
	monitor.Name = updateData.Name
	monitor.Protocol = updateData.Protocol
	monitor.Query = updateData.Query
	monitor.RecordType = updateData.RecordType
	monitor.Enabled = updateData.Enabled

	if monitor.Interval != updateData.Interval {
		monitor.Interval = updateData.Interval
		nextRun := h.scheduler.CalculateNextRun(monitor.Interval, time.Now())
		if nextRun.IsZero() {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid interval"})
			return
		}
		monitor.NextRun = &nextRun
	}

	if err := h.db.UpdateDNSMonitor(monitor); err != nil {
		log.Error().Err(err).Int64("monitorID", id).Msg("Failed to update dns monitor")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update monitor"})
		return
	}

	c.JSON(http.StatusOK, monitor)
}

// DeleteMonitor deletes a DNS monitor and its results
func (h *DNSHandler) DeleteMonitor(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid monitor ID"})
		return
	}

	if err := h.db.DeleteDNSMonitor(id); err != nil {
		if err == database.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Monitor not found"})
			return
		}
		log.Error().Err(err).Int64("monitorID", id).Msg("Failed to delete dns monitor")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete monitor"})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

// GetMonitorStatus returns the last known status of a monitor
func (h *DNSHandler) GetMonitorStatus(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid monitor ID"})
		return
	}

	status, err := h.service.GetMonitorStatus(id)
	if err != nil {
		if err == database.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Monitor not found"})
			return
		}
		log.Error().Err(err).Int64("monitorID", id).Msg("Failed to get dns monitor status")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get monitor status"})
		return
	}

	c.JSON(http.StatusOK, status)
}

// GetMonitorHistory returns paginated results for a monitor
func (h *DNSHandler) GetMonitorHistory(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid monitor ID"})
		return
	}

	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil || page <= 0 {
		page = 1
	}

	limit, err := strconv.Atoi(c.DefaultQuery("limit", "25"))
	if err != nil || limit <= 0 {
		limit = 25
	}
	if limit > 100 {
		limit = 100
	}

	// an unknown monitor is a 404 here as it is on the status route, not an
	// empty page
	if _, err := h.db.GetDNSMonitor(id); err != nil {
		if err == database.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Monitor not found"})
			return
		}
		log.Error().Err(err).Int64("monitorID", id).Msg("Failed to get dns monitor")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get monitor history"})
		return
	}

	results, err := h.db.GetDNSResults(id, page, limit)
	if err != nil {
		log.Error().Err(err).Int64("monitorID", id).Msg("Failed to get dns results")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get monitor history"})
		return
	}

	c.JSON(http.StatusOK, results)
}
