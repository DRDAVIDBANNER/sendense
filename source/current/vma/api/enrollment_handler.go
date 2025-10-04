// Package api provides VMA enrollment API endpoints
package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/vexxhost/migratekit/source/current/vma/services"
)

// EnrollmentHandler handles VMA enrollment operations
type EnrollmentHandler struct {
	enrollmentClient *services.VMAEnrollmentClient
}

// NewEnrollmentHandler creates a new enrollment handler
func NewEnrollmentHandler(enrollmentClient *services.VMAEnrollmentClient) *EnrollmentHandler {
	return &EnrollmentHandler{
		enrollmentClient: enrollmentClient,
	}
}

// EnrollWithOMARequest represents VMA enrollment request
type EnrollWithOMARequest struct {
	OMAHost     string `json:"oma_host" validate:"required"`
	OMAPort     int    `json:"oma_port"`
	PairingCode string `json:"pairing_code" validate:"required"`
	VMAName     string `json:"vma_name"`
	VMAVersion  string `json:"vma_version"`
}

// EnrollWithOMAResponse represents VMA enrollment response
type EnrollWithOMAResponse struct {
	Success          bool   `json:"success"`
	EnrollmentID     string `json:"enrollment_id,omitempty"`
	Status           string `json:"status,omitempty"`
	Message          string `json:"message"`
	TunnelConfigured bool   `json:"tunnel_configured"`
}

// EnrollWithOMA handles VMA enrollment with OMA
// @Summary Enroll VMA with OMA
// @Description Performs complete VMA enrollment process with OMA using pairing code
// @Tags VMA Enrollment
// @Accept json
// @Produce json
// @Param request body EnrollWithOMARequest true "Enrollment request"
// @Success 200 {object} EnrollWithOMAResponse "Enrollment successful"
// @Failure 400 {object} ErrorResponse "Invalid request"
// @Failure 401 {object} ErrorResponse "Enrollment failed"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /api/v1/enrollment/enroll [post]
func (eh *EnrollmentHandler) EnrollWithOMA(w http.ResponseWriter, r *http.Request) {
	var req EnrollWithOMARequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.WithError(err).Error("Invalid enrollment request")
		writeErrorResponse(w, http.StatusBadRequest, "Invalid request format", err.Error())
		return
	}

	// Validate required fields
	if req.OMAHost == "" || req.PairingCode == "" {
		log.Error("Missing required enrollment fields")
		writeErrorResponse(w, http.StatusBadRequest, "Missing required fields", "oma_host and pairing_code are required")
		return
	}

	// Set defaults
	if req.OMAPort == 0 {
		req.OMAPort = 443 // Default HTTPS port for enrollment API
	}
	if req.VMAName == "" {
		req.VMAName = "VMA-" + generateRandomSuffix()
	}
	if req.VMAVersion == "" {
		req.VMAVersion = "v2.20.1" // TODO: Get from build version
	}

	log.WithFields(log.Fields{
		"oma_host":     req.OMAHost,
		"oma_port":     req.OMAPort,
		"pairing_code": req.PairingCode,
		"vma_name":     req.VMAName,
	}).Info("🔐 Starting VMA enrollment process")

	// Create enrollment client for this OMA
	enrollmentClient := services.NewVMAEnrollmentClient(req.OMAHost, req.OMAPort)

	// Perform enrollment
	config, err := enrollmentClient.EnrollWithOMA(req.PairingCode, req.VMAName, req.VMAVersion)
	if err != nil {
		log.WithError(err).WithFields(log.Fields{
			"oma_host":     req.OMAHost,
			"pairing_code": req.PairingCode,
		}).Error("VMA enrollment failed")

		writeErrorResponse(w, http.StatusUnauthorized, "Enrollment failed", err.Error())
		return
	}

	// Configure tunnel with enrollment credentials
	tunnelConfigured := false
	if err := enrollmentClient.ConfigureTunnel(config); err != nil {
		log.WithError(err).Warn("Failed to configure tunnel - enrollment successful but manual tunnel setup required")
	} else {
		tunnelConfigured = true
	}

	log.WithFields(log.Fields{
		"enrollment_id":     config.EnrollmentID,
		"oma_host":          config.OMAHost,
		"ssh_user":          config.SSHUser,
		"tunnel_configured": tunnelConfigured,
	}).Info("🎉 VMA enrollment completed successfully")

	response := &EnrollWithOMAResponse{
		Success:          true,
		EnrollmentID:     config.EnrollmentID,
		Status:           "approved",
		Message:          "VMA enrollment completed successfully",
		TunnelConfigured: tunnelConfigured,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetEnrollmentStatus returns current enrollment status
// @Summary Get enrollment status
// @Description Returns current VMA enrollment status
// @Tags VMA Enrollment
// @Produce json
// @Param enrollment_id query string true "Enrollment ID"
// @Success 200 {object} services.EnrollmentResult "Enrollment status"
// @Failure 400 {object} ErrorResponse "Invalid request"
// @Failure 404 {object} ErrorResponse "Enrollment not found"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /api/v1/enrollment/status [get]
func (eh *EnrollmentHandler) GetEnrollmentStatus(w http.ResponseWriter, r *http.Request) {
	enrollmentID := r.URL.Query().Get("enrollment_id")
	if enrollmentID == "" {
		writeErrorResponse(w, http.StatusBadRequest, "Missing enrollment_id parameter", "enrollment_id query parameter is required")
		return
	}

	// Get enrollment status
	result, err := eh.enrollmentClient.GetEnrollmentStatus(enrollmentID)
	if err != nil {
		log.WithError(err).WithField("enrollment_id", enrollmentID).Error("Failed to get enrollment status")
		writeErrorResponse(w, http.StatusNotFound, "Enrollment not found", err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// generateRandomSuffix generates a random suffix for default VMA names
func generateRandomSuffix() string {
	// Simple random suffix - in production would use crypto/rand
	return fmt.Sprintf("%d", time.Now().Unix()%10000)
}

// writeErrorResponse writes a standardized error response
func writeErrorResponse(w http.ResponseWriter, statusCode int, error string, details string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	response := map[string]string{
		"error":   error,
		"details": details,
	}

	json.NewEncoder(w).Encode(response)
}






