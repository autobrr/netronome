package server

import (
	"net/http"

	"github.com/autobrr/netronome/internal/server/encoder"

	"github.com/go-chi/chi/v5"
)

type healthHandler struct {
	//db      *database.DB
	//metrics *metrics.Metrics
}

func newHealthHandler() *healthHandler {
	return &healthHandler{}
}

func (h *healthHandler) Routes(r chi.Router) {
	r.Get("/liveness", h.handleLiveness)
	r.Get("/readiness", h.handleReadiness)
}

func (h *healthHandler) handleLiveness(w http.ResponseWriter, r *http.Request) {
	writeHealthy(w, r)
}

func (h *healthHandler) handleReadiness(w http.ResponseWriter, r *http.Request) {
	writeHealthy(w, r)
}

func writeHealthy(w http.ResponseWriter, r *http.Request) {
	encoder.PlainText(w, http.StatusOK, "OK")
}

func writeUnhealthy(w http.ResponseWriter, r *http.Request) {
	encoder.PlainText(w, http.StatusFailedDependency, "Unhealthy")
}
