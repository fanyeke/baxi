package api

import (
	"encoding/json"
	"net/http"
)

const apiVersion = "0.6.0"

// HealthResponse represents the health check response.
type HealthResponse struct {
	Status      string `json:"status"`
	Version     string `json:"version"`
	DBConnected bool   `json:"db_connected"`
}

// handleHealth returns the service health status, including database connectivity.
// Always returns 200; db_connected reflects whether the database is reachable.
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	dbConnected := false
	if s.pool != nil {
		if err := s.pool.Ping(r.Context()); err == nil {
			dbConnected = true
		}
	}

	resp := HealthResponse{
		Status:      "ok",
		Version:     apiVersion,
		DBConnected: dbConnected,
	}

	json.NewEncoder(w).Encode(resp)
}
