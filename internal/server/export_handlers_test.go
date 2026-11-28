// Copyright (c) 2024-2026, s0up and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package server

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/autobrr/netronome/internal/config"
	"github.com/autobrr/netronome/internal/database"
)

type exportStubDB struct {
	database.Service
	export func(w io.Writer, dataset database.ExportDataset, format database.ExportFormat, from, to time.Time) error
}

func (s exportStubDB) ExportResults(_ context.Context, w io.Writer, dataset database.ExportDataset, format database.ExportFormat, from, to time.Time) error {
	return s.export(w, dataset, format, from, to)
}

func newExportTestServer(t *testing.T, db database.Service, whitelist []string) *Server {
	t.Helper()
	gin.SetMode(gin.TestMode)
	cfg := config.New()
	cfg.Auth.Whitelist = whitelist
	server := NewServer(nil, db, nil, cfg, nil, nil, nil, nil, nil)
	server.RegisterRoutes()
	return server
}

func serveExport(t *testing.T, server *Server, target string) *httptest.ResponseRecorder {
	t.Helper()
	response := httptest.NewRecorder()
	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, target, nil)
	request.RemoteAddr = "127.0.0.1:12345"
	server.Router.ServeHTTP(response, request)
	return response
}

func TestExportRejectsInvalidInput(t *testing.T) {
	db := exportStubDB{export: func(io.Writer, database.ExportDataset, database.ExportFormat, time.Time, time.Time) error {
		t.Error("export must not run for invalid input")
		return nil
	}}
	server := newExportTestServer(t, db, []string{"127.0.0.1/32"})

	for _, target := range []string{
		"/api/export/bandwidth?format=csv",
		"/api/export/speedtests?format=xml",
		"/api/export/speedtests",
		"/api/export/speedtests?format=csv&from=2026-10-17",
		"/api/export/speedtests?format=csv&to=yesterday",
	} {
		t.Run(target, func(t *testing.T) {
			response := serveExport(t, server, target)
			assert.Equal(t, http.StatusBadRequest, response.Code)
			assert.Equal(t, "application/json; charset=utf-8", response.Header().Get("Content-Type"))
			assert.Contains(t, response.Body.String(), `"error"`)
		})
	}
}

func TestExportRequiresAuth(t *testing.T) {
	db := exportStubDB{export: func(io.Writer, database.ExportDataset, database.ExportFormat, time.Time, time.Time) error {
		t.Error("export must not run without auth")
		return nil
	}}
	server := newExportTestServer(t, db, nil)

	response := serveExport(t, server, "/api/export/speedtests?format=csv")
	assert.Equal(t, http.StatusUnauthorized, response.Code)
}

func TestExportWritesFile(t *testing.T) {
	var gotDataset database.ExportDataset
	var gotFormat database.ExportFormat
	var gotFrom, gotTo time.Time
	db := exportStubDB{export: func(w io.Writer, dataset database.ExportDataset, format database.ExportFormat, from, to time.Time) error {
		gotDataset, gotFormat, gotFrom, gotTo = dataset, format, from, to
		_, err := io.WriteString(w, "[]")
		return err
	}}
	server := newExportTestServer(t, db, []string{"127.0.0.1/32"})

	before := time.Now().Format(time.DateOnly)
	response := serveExport(t, server, "/api/export/dns?format=json&from=2026-10-17T00:00:00%2B02:00")
	after := time.Now().Format(time.DateOnly)

	require.Equal(t, http.StatusOK, response.Code)
	assert.Equal(t, "application/json", response.Header().Get("Content-Type"))
	assert.Contains(t, []string{
		`attachment; filename="netronome-dns-` + before + `.json"`,
		`attachment; filename="netronome-dns-` + after + `.json"`,
	}, response.Header().Get("Content-Disposition"))
	assert.Equal(t, "[]", response.Body.String())
	assert.Equal(t, database.ExportDNS, gotDataset)
	assert.Equal(t, database.ExportJSON, gotFormat)
	assert.True(t, gotFrom.Equal(time.Date(2026, 10, 16, 22, 0, 0, 0, time.UTC)), "from: %v", gotFrom)
	assert.True(t, gotTo.IsZero(), "to: %v", gotTo)

	response = serveExport(t, server, "/api/export/speedtests?format=csv")
	require.Equal(t, http.StatusOK, response.Code)
	assert.Equal(t, "text/csv; charset=utf-8", response.Header().Get("Content-Type"))
	assert.Contains(t, response.Header().Get("Content-Disposition"), `.csv"`)
}

func TestExportAbortsConnectionOnError(t *testing.T) {
	db := exportStubDB{export: func(w io.Writer, _ database.ExportDataset, _ database.ExportFormat, _, _ time.Time) error {
		if _, err := io.WriteString(w, "id,created_at\n1,2026-10-17T00:00:00Z\n"); err != nil {
			return err
		}
		return errors.New("disk I/O error")
	}}
	server := newExportTestServer(t, db, []string{"127.0.0.1/32"})
	httpServer := httptest.NewServer(server.Router)
	defer httpServer.Close()

	request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, httpServer.URL+"/api/export/speedtests?format=csv", nil)
	require.NoError(t, err)
	response, err := http.DefaultClient.Do(request)
	require.NoError(t, err)
	defer response.Body.Close()

	require.Equal(t, http.StatusOK, response.StatusCode)
	_, err = io.ReadAll(response.Body)
	assert.ErrorIs(t, err, io.ErrUnexpectedEOF, "a failed export must not look like a complete file")
}
