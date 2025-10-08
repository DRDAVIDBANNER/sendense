// Package api provides SNA enrollment API endpoints
package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/vexxhost/migratekit/source/current/sna/services"
)

// EnrollmentHandler handles SNA enrollment operations
type EnrollmentHandler struct {
	enrollmentClient *services.SNAEnrollmentClient
}

// NewEnrollmentHandler creates a new enrollment handler
func NewEnrollmentHandler(enrollmentClient *services.SNAEnrollmentClient) *EnrollmentHandler {
	return &EnrollmentHandler{
		enrollmentClient: enrollmentClient,
	}
}

// EnrollWithOMARequest represents SNA enrollment request
type EnrollWithOMARequest struct {
	SHAHost     string `json:"oma_host" validate:"required"`
	SHAPort     int    `json:"oma_port"`
	PairingCode string `json:"pairing_code" validate:"required"`
	SNAName     string `json:"vma_name"`
	SNAVersion  string `json:"vma_version"`
}

// EnrollWithOMAResponse represents SNA enrollment response
type EnrollWithOMAResponse struct {
	Success          bool   `json:"success"`
	EnrollmentID     string `json:"enrollment_id,omitempty"`
	Status           string `json:"status,omitempty"`
	Message          string `json:"message"`
	TunnelConfigured bool   `json:"tunnel_configured"`
}

// EnrollWithOMA handles SNA enrollment with SHA
// @Summary Enroll SNA with SHA
// @Description Performs complete SNA enrollment process with SHA using pairing code
// @Tags SNA Enrollment
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
	if req.SHAHost == "" || req.PairingCode == "" {
		log.Error("Missing required enrollment fields")
		writeErrorResponse(w, http.StatusBadRequest, "Missing required fields", "oma_host and pairing_code are required")
		return
	}

	// Set defaults
	if req.SHAPort == 0 {
		req.SHAPort = 443 // Default HTTPS port for enrollment API
	}
	if req.SNAName == "" {
		req.SNAName = "SNA-" + generateRandomSuffix()
	}
	if req.SNAVersion == "" {
		req.SNAVersion = "v2.20.1" // TODO: Get from build version
	}

	log.WithFields(log.Fields{
		"oma_host":     req.SHAHost,
		"oma_port":     req.SHAPort,
		"pairing_code": req.PairingCode,
		"vma_name":     req.SNAName,
	}).Info("🔐 Starting SNA enrollment process")

	// Create enrollment client for this SHA
	enrollmentClient := services.NewVMAEnrollmentClient(req.SHAHost, req.SHAPort)

	// Perform enrollment
	config, err := enrollmentClient.EnrollWithOMA(req.PairingCode, req.SNAName, req.SNAVersion)
	if err != nil {
		log.WithError(err).WithFields(log.Fields{
			"oma_host":     req.SHAHost,
			"pairing_code": req.PairingCode,
		}).Error("SNA enrollment failed")

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
		"oma_host":          config.SHAHost,
		"ssh_user":          config.SSHUser,
		"tunnel_configured": tunnelConfigured,
	}).Info("🎉 SNA enrollment completed successfully")

	response := &EnrollWithOMAResponse{
		Success:          true,
		EnrollmentID:     config.EnrollmentID,
		Status:           "approved",
		Message:          "SNA enrollment completed successfully",
		TunnelConfigured: tunnelConfigured,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetEnrollmentStatus returns current enrollment status
// @Summary Get enrollment status
// @Description Returns current SNA enrollment status
// @Tags SNA Enrollment
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

// generateRandomSuffix generates a random suffix for default SNA names
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






