// Package handlers provides HTTP handlers for OMA API endpoints
// Authentication handler following project rules: modular design, clean interfaces
package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/vexxhost/migratekit-oma/database"
)

// AuthHandler handles authentication endpoints
type AuthHandler struct {
	db       database.Connection
	sessions map[string]*Session // In-memory session storage for now
}

// Session represents an authenticated session
type Session struct {
	Token       string    `json:"token"`
	ApplianceID string    `json:"appliance_id"`
	ExpiresAt   time.Time `json:"expires_at"`
	CreatedAt   time.Time `json:"created_at"`
}

// AuthRequest represents authentication request
type AuthRequest struct {
	ApplianceID string `json:"appliance_id" binding:"required"`
	Token       string `json:"token" binding:"required"`
	Version     string `json:"version" binding:"required"`
}

// AuthResponse represents authentication response
type AuthResponse struct {
	Success      bool   `json:"success"`
	SessionToken string `json:"session_token,omitempty"`
	ExpiresAt    string `json:"expires_at,omitempty"`
	ErrorMessage string `json:"error_message,omitempty"`
}

// ErrorResponse represents a standard API error response
type ErrorResponse struct {
	Error     string `json:"error" example:"Invalid request"`
	Details   string `json:"details" example:"Missing required field"`
	Timestamp string `json:"timestamp" example:"2025-08-06T10:00:00Z"`
}

// NewAuthHandler creates a new authentication handler
func NewAuthHandler(db database.Connection) *AuthHandler {
	return &AuthHandler{
		db:       db,
		sessions: make(map[string]*Session),
	}
}

// Login handles authentication requests
// @Summary Authenticate VMA
// @Description Authenticate VMware appliance for secure OMA communication
// @Tags authentication
// @Accept json
// @Produce json
// @Param credentials body AuthRequest true "VMA authentication credentials"
// @Success 200 {object} AuthResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Router /api/v1/auth/login [post]
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req AuthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid request payload", err.Error())
		return
	}

	// Simple token validation (in production, use proper validation)
	if req.Token == "vma_test_token_abc123def456789012345678" {
		// Create long-lived session (10 years) to avoid frequent re-authentication
		sessionToken := "sess_longlived_dev_token_2025_2035_permanent"
		expiresAt := time.Now().Add(10 * 365 * 24 * time.Hour) // 10 years

		h.sessions[sessionToken] = &Session{
			Token:       sessionToken,
			ApplianceID: req.ApplianceID,
			ExpiresAt:   expiresAt,
			CreatedAt:   time.Now(),
		}

		log.WithField("appliance_id", req.ApplianceID).Info("VMA authenticated successfully")

		response := AuthResponse{
			Success:      true,
			SessionToken: sessionToken,
			ExpiresAt:    expiresAt.Format(time.RFC3339),
		}

		h.writeJSONResponse(w, http.StatusOK, response)
	} else {
		response := AuthResponse{
			Success:      false,
			ErrorMessage: "Invalid credentials",
		}
		h.writeJSONResponse(w, http.StatusUnauthorized, response)
	}
}

// ValidateToken validates a session token
func (h *AuthHandler) ValidateToken(token string) bool {
	session, exists := h.sessions[token]
	if !exists {
		return false
	}

	// Check if token is expired
	if time.Now().After(session.ExpiresAt) {
		delete(h.sessions, token)
		return false
	}

	return true
}

// Helper functions

// writeJSONResponse writes a standardized JSON response
func (h *AuthHandler) writeJSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.WithError(err).Error("Failed to encode JSON response")
	}
}

// writeErrorResponse writes a standardized error response
func (h *AuthHandler) writeErrorResponse(w http.ResponseWriter, statusCode int, message, details string) {
	response := map[string]interface{}{
		"error":     message,
		"details":   details,
		"timestamp": time.Now().Format(time.RFC3339),
	}
	h.writeJSONResponse(w, statusCode, response)
}
