package api

import (
	"encoding/json"
	"net/http"
)

// HealthResponse represents the health check response.
type HealthResponse struct {
	Status  string `json:"status"`
	Service string `json:"service"`
}

// Phase 1: static response, no DB ping.
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	resp := HealthResponse{
		Status:  "ok",
		Service: "baxi-api",
	}

	json.NewEncoder(w).Encode(resp)
}
