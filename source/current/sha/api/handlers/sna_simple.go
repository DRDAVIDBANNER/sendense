// Package handlers provides simple SNA enrollment endpoints
package handlers

import (
	"encoding/json"
	"net/http"

	log "github.com/sirupsen/logrus"
)

// SNASimpleHandler provides basic SNA enrollment endpoints
type SNASimpleHandler struct{}

// NewVMASimpleHandler creates a new simple SNA handler
func NewVMASimpleHandler() *SNASimpleHandler {
	return &SNASimpleHandler{}
}

// GeneratePairingCode generates a SNA pairing code
func (vsh *SNASimpleHandler) GeneratePairingCode(w http.ResponseWriter, r *http.Request) {
	log.Info("🔑 SNA pairing code generation requested")

	// For now, return a mock response
	response := map[string]interface{}{
		"pairing_code": "AX7K-PJ3F-TH2Q",
		"expires_at":   "2025-09-29T06:10:00Z",
		"valid_for":    600,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// ListPendingEnrollments lists pending SNA enrollments
func (vsh *SNASimpleHandler) ListPendingEnrollments(w http.ResponseWriter, r *http.Request) {
	log.Info("📋 Pending SNA enrollments requested")

	// For now, return empty list
	response := map[string]interface{}{
		"count":       0,
		"enrollments": []interface{}{},
		"message":     "SNA enrollment system - Phase 1 implementation",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}






