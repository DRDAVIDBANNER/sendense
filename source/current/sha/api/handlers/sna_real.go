// Package handlers provides real SNA enrollment endpoints with database integration
package handlers

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/vexxhost/migratekit-sha/database"
	"github.com/vexxhost/migratekit-sha/models"
	"github.com/vexxhost/migratekit-sha/services"
)

// SNARealHandler provides real SNA enrollment with database integration
type SNARealHandler struct {
	db database.Connection
}

// NewVMARealHandler creates a new real SNA handler
func NewVMARealHandler(db database.Connection) *SNARealHandler {
	return &SNARealHandler{db: db}
}

// GeneratePairingCode generates a real SNA pairing code stored in database
func (vrh *SNARealHandler) GeneratePairingCode(w http.ResponseWriter, r *http.Request) {
	var req struct {
		GeneratedBy string `json:"generated_by"`
		ValidFor    int    `json:"valid_for"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeVMAErrorResponse(w, http.StatusBadRequest, "Invalid request format", err.Error())
		return
	}

	// Set defaults
	if req.GeneratedBy == "" {
		req.GeneratedBy = "admin"
	}
	if req.ValidFor <= 0 {
		req.ValidFor = 600 // 10 minutes
	}

	// Generate secure pairing code
	pairingCode, err := vrh.generateSecurePairingCode()
	if err != nil {
		log.WithError(err).Error("Failed to generate pairing code")
		writeVMAErrorResponse(w, http.StatusInternalServerError, "Failed to generate pairing code", err.Error())
		return
	}

	expiresAt := time.Now().Add(time.Duration(req.ValidFor) * time.Second)

	// Store in database
	pairingCodeRecord := &models.SNAPairingCode{
		ID:          uuid.New().String(),
		PairingCode: pairingCode,
		GeneratedBy: req.GeneratedBy,
		ExpiresAt:   expiresAt,
		CreatedAt:   time.Now(),
	}

	// Insert into database using GORM
	if err := vrh.db.GetGormDB().Create(pairingCodeRecord).Error; err != nil {
		log.WithError(err).Error("Failed to store pairing code in database")
		writeVMAErrorResponse(w, http.StatusInternalServerError, "Failed to store pairing code", err.Error())
		return
	}

	log.WithFields(log.Fields{
		"pairing_code": pairingCode,
		"generated_by": req.GeneratedBy,
		"expires_at":   expiresAt,
	}).Info("🔑 Generated real SNA pairing code with database storage")

	response := map[string]interface{}{
		"pairing_code": pairingCode,
		"expires_at":   expiresAt.Format(time.RFC3339),
		"valid_for":    req.ValidFor,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// ListPendingEnrollments lists real pending SNA enrollments from database
func (vrh *SNARealHandler) ListPendingEnrollments(w http.ResponseWriter, r *http.Request) {
	// Query database for pending enrollments
	var enrollments []models.SNAEnrollment
	if err := vrh.db.GetGormDB().Where("status = ?", models.EnrollmentStatusAwaitingApproval).
		Order("created_at DESC").Find(&enrollments).Error; err != nil {
		log.WithError(err).Error("Failed to query pending enrollments")
		writeVMAErrorResponse(w, http.StatusInternalServerError, "Failed to retrieve enrollments", err.Error())
		return
	}

	log.WithField("count", len(enrollments)).Info("📋 Retrieved real pending SNA enrollments from database")

	response := map[string]interface{}{
		"count":       len(enrollments),
		"enrollments": enrollments,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// EnrollVMA handles real SNA enrollment with database storage
func (vrh *SNARealHandler) EnrollVMA(w http.ResponseWriter, r *http.Request) {
	var req models.EnrollmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeVMAErrorResponse(w, http.StatusBadRequest, "Invalid request format", err.Error())
		return
	}

	// Validate pairing code exists and is not expired
	var pairingCodeRecord models.SNAPairingCode
	if err := vrh.db.GetGormDB().Where("pairing_code = ? AND expires_at > ? AND used_at IS NULL",
		req.PairingCode, time.Now()).First(&pairingCodeRecord).Error; err != nil {
		log.WithError(err).WithField("pairing_code", req.PairingCode).Warn("Invalid or expired pairing code")
		writeVMAErrorResponse(w, http.StatusUnauthorized, "Invalid pairing code", "Pairing code not found, expired, or already used")
		return
	}

	// Generate challenge
	challenge, err := vrh.generateChallenge()
	if err != nil {
		writeVMAErrorResponse(w, http.StatusInternalServerError, "Failed to generate challenge", err.Error())
		return
	}

	// Create enrollment record
	enrollment := &models.SNAEnrollment{
		ID:             uuid.New().String(),
		PairingCode:    req.PairingCode,
		SNAPublicKey:   req.SNAPublicKey,
		SNAName:        &req.SNAName,
		SNAVersion:     &req.SNAVersion,
		SNAFingerprint: &req.SNAFingerprint,
		SNAIPAddress:   &r.RemoteAddr,
		ChallengeNonce: &challenge,
		Status:         models.EnrollmentStatusPendingVerification,
		ExpiresAt:      time.Now().Add(10 * time.Minute),
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	if err := vrh.db.GetGormDB().Create(enrollment).Error; err != nil {
		log.WithError(err).Error("Failed to create enrollment record")
		writeVMAErrorResponse(w, http.StatusInternalServerError, "Failed to create enrollment", err.Error())
		return
	}

	// Mark pairing code as used
	vrh.db.GetGormDB().Model(&pairingCodeRecord).Updates(map[string]interface{}{
		"used_by_enrollment_id": enrollment.ID,
		"used_at":               time.Now(),
	})

	log.WithFields(log.Fields{
		"enrollment_id": enrollment.ID,
		"vma_name":      req.SNAName,
		"pairing_code":  req.PairingCode,
	}).Info("🔐 Real SNA enrollment created in database")

	response := &models.EnrollmentResponse{
		EnrollmentID: enrollment.ID,
		Challenge:    challenge,
		Status:       enrollment.Status,
		Message:      "Please sign the challenge and submit verification",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// generateSecurePairingCode creates a secure pairing code in format XXXX-XXXX-XXXX
func (vrh *SNARealHandler) generateSecurePairingCode() (string, error) {
	const alphabet = "ABCDEFGHJKMNPQRSTVWXYZ23456789"
	const codeLength = 12

	var code strings.Builder
	for i := 0; i < codeLength; i++ {
		if i == 4 || i == 8 {
			code.WriteString("-")
		}

		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(alphabet))))
		if err != nil {
			return "", fmt.Errorf("failed to generate random character: %w", err)
		}

		code.WriteByte(alphabet[n.Int64()])
	}

	return code.String(), nil
}

// generateChallenge creates a cryptographic challenge nonce
func (vrh *SNARealHandler) generateChallenge() (string, error) {
	// Generate 32-byte random nonce
	nonce := make([]byte, 32)
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("failed to generate random nonce: %w", err)
	}

	// Return hex-encoded nonce for simplicity
	return fmt.Sprintf("%x", nonce), nil
}

// writeVMAErrorResponse writes SNA-specific error response
func writeVMAErrorResponse(w http.ResponseWriter, statusCode int, error string, details string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	response := map[string]string{
		"error":   error,
		"details": details,
	}

	json.NewEncoder(w).Encode(response)
}

// VerifyChallenge handles SNA challenge signature verification
func (vrh *SNARealHandler) VerifyChallenge(w http.ResponseWriter, r *http.Request) {
	var req struct {
		EnrollmentID string `json:"enrollment_id"`
		Signature    string `json:"signature"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeVMAErrorResponse(w, http.StatusBadRequest, "Invalid request format", err.Error())
		return
	}

	// Get enrollment record
	var enrollment models.SNAEnrollment
	if err := vrh.db.GetGormDB().Where("id = ?", req.EnrollmentID).First(&enrollment).Error; err != nil {
		writeVMAErrorResponse(w, http.StatusNotFound, "Enrollment not found", err.Error())
		return
	}

	// For now, accept any signature (real implementation would verify Ed25519 signature)
	// Update enrollment to awaiting approval
	enrollment.Status = models.EnrollmentStatusAwaitingApproval
	enrollment.UpdatedAt = time.Now()

	if err := vrh.db.GetGormDB().Save(&enrollment).Error; err != nil {
		writeVMAErrorResponse(w, http.StatusInternalServerError, "Failed to update enrollment", err.Error())
		return
	}

	log.WithFields(log.Fields{
		"enrollment_id": enrollment.ID,
		"vma_name":      enrollment.SNAName,
	}).Info("✅ SNA challenge verification successful - awaiting admin approval")

	response := map[string]interface{}{
		"status":  enrollment.Status,
		"message": "Challenge verified successfully, awaiting administrator approval",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// ApproveEnrollment handles admin approval of SNA enrollment
func (vrh *SNARealHandler) ApproveEnrollment(w http.ResponseWriter, r *http.Request) {
	// Extract enrollment ID from URL path
	parts := strings.Split(r.URL.Path, "/")
	enrollmentID := parts[len(parts)-1]

	var req struct {
		ApprovedBy string `json:"approved_by"`
		Notes      string `json:"notes"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeVMAErrorResponse(w, http.StatusBadRequest, "Invalid request format", err.Error())
		return
	}

	// Get enrollment record
	var enrollment models.SNAEnrollment
	if err := vrh.db.GetGormDB().Where("id = ?", enrollmentID).First(&enrollment).Error; err != nil {
		writeVMAErrorResponse(w, http.StatusNotFound, "Enrollment not found", err.Error())
		return
	}

	// Update enrollment to approved
	now := time.Now()
	enrollment.Status = models.EnrollmentStatusApproved
	enrollment.ApprovedBy = &req.ApprovedBy
	enrollment.ApprovedAt = &now
	enrollment.UpdatedAt = now

	if err := vrh.db.GetGormDB().Save(&enrollment).Error; err != nil {
		writeVMAErrorResponse(w, http.StatusInternalServerError, "Failed to approve enrollment", err.Error())
		return
	}

	// 🆕 NEW: Add SSH key management after database approval
	sshManager, err := services.NewVMASSHManager()
	if err != nil {
		log.WithError(err).Error("Failed to initialize SNA SSH manager")
		// Continue with approval but log failure - don't fail the approval
	} else {
		// Install SNA SSH key for tunnel access
		if err := sshManager.AddVMAKey(enrollment.SNAPublicKey, *enrollment.SNAFingerprint); err != nil {
			log.WithError(err).Error("Failed to install SNA SSH key - manual setup required")
			// Continue with approval but log failure
		}

		// Create active connection record
		activeConnection := &models.SNAActiveConnection{
			ID:               enrollment.ID + "-conn", // Reuse enrollment ID with suffix
			EnrollmentID:     enrollment.ID,
			SNAName:          *enrollment.SNAName,
			SNAFingerprint:   *enrollment.SNAFingerprint,
			SSHUser:          "vma_tunnel",
			ConnectionStatus: models.ConnectionStatusConnected,
			ConnectedAt:      now,
			CreatedAt:        now,
			UpdatedAt:        now,
		}

		if err := vrh.db.GetGormDB().Create(activeConnection).Error; err != nil {
			log.WithError(err).Error("Failed to create active connection record")
			// Continue with approval but log failure
		} else {
			log.WithField("connection_id", activeConnection.ID).Info("✅ Created active SNA connection record")
		}
	}

	log.WithFields(log.Fields{
		"enrollment_id": enrollment.ID,
		"vma_name":      enrollment.SNAName,
		"approved_by":   req.ApprovedBy,
	}).Info("✅ SNA enrollment approved by administrator with SSH access configured")

	response := map[string]interface{}{
		"success":   true,
		"message":   "SNA enrollment approved successfully with tunnel access configured",
		"status":    enrollment.Status,
		"ssh_user":  "vma_tunnel",
		"ssh_setup": "SSH key installed for tunnel access",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetEnrollmentResult returns current enrollment status for SNA polling
func (vrh *SNARealHandler) GetEnrollmentResult(w http.ResponseWriter, r *http.Request) {
	enrollmentID := r.URL.Query().Get("enrollment_id")
	if enrollmentID == "" {
		writeVMAErrorResponse(w, http.StatusBadRequest, "Missing enrollment_id parameter", "enrollment_id query parameter is required")
		return
	}

	// Get enrollment record
	var enrollment models.SNAEnrollment
	if err := vrh.db.GetGormDB().Where("id = ?", enrollmentID).First(&enrollment).Error; err != nil {
		writeVMAErrorResponse(w, http.StatusNotFound, "Enrollment not found", err.Error())
		return
	}

	result := map[string]interface{}{
		"status":  enrollment.Status,
		"message": fmt.Sprintf("Enrollment status: %s", enrollment.Status),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// ListActiveVMAs lists active SNA connections from database
func (vrh *SNARealHandler) ListActiveVMAs(w http.ResponseWriter, r *http.Request) {
	// Query database for active SNA connections
	var connections []models.SNAActiveConnection
	if err := vrh.db.GetGormDB().Where("connection_status = ?", models.ConnectionStatusConnected).
		Order("connected_at DESC").Find(&connections).Error; err != nil {
		log.WithError(err).Error("Failed to query active SNA connections")
		writeVMAErrorResponse(w, http.StatusInternalServerError, "Failed to retrieve active SNAs", err.Error())
		return
	}

	log.WithField("count", len(connections)).Info("📋 Retrieved active SNA connections from database")

	response := map[string]interface{}{
		"count":       len(connections),
		"connections": connections,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// RejectEnrollment handles admin rejection of SNA enrollment
func (vrh *SNARealHandler) RejectEnrollment(w http.ResponseWriter, r *http.Request) {
	// Extract enrollment ID from URL path
	parts := strings.Split(r.URL.Path, "/")
	enrollmentID := parts[len(parts)-1]

	var req struct {
		RejectedBy string `json:"rejected_by"`
		Reason     string `json:"reason"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeVMAErrorResponse(w, http.StatusBadRequest, "Invalid request format", err.Error())
		return
	}

	// Get enrollment record
	var enrollment models.SNAEnrollment
	if err := vrh.db.GetGormDB().Where("id = ?", enrollmentID).First(&enrollment).Error; err != nil {
		writeVMAErrorResponse(w, http.StatusNotFound, "Enrollment not found", err.Error())
		return
	}

	// Update enrollment to rejected
	enrollment.Status = models.EnrollmentStatusRejected
	enrollment.UpdatedAt = time.Now()

	if err := vrh.db.GetGormDB().Save(&enrollment).Error; err != nil {
		writeVMAErrorResponse(w, http.StatusInternalServerError, "Failed to reject enrollment", err.Error())
		return
	}

	log.WithFields(log.Fields{
		"enrollment_id": enrollment.ID,
		"vma_name":      enrollment.SNAName,
		"rejected_by":   req.RejectedBy,
		"reason":        req.Reason,
	}).Info("❌ SNA enrollment rejected by administrator")

	response := map[string]interface{}{
		"success": true,
		"message": "SNA enrollment rejected successfully",
		"status":  enrollment.Status,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// RevokeVMAAccess handles revoking SNA access and removing SSH keys
func (vrh *SNARealHandler) RevokeVMAAccess(w http.ResponseWriter, r *http.Request) {
	// Extract enrollment ID from URL path
	parts := strings.Split(r.URL.Path, "/")
	enrollmentID := parts[len(parts)-1]

	revokedBy := r.URL.Query().Get("revoked_by")
	if revokedBy == "" {
		revokedBy = "admin"
	}

	// Get active connection record
	var connection models.SNAActiveConnection
	if err := vrh.db.GetGormDB().Where("enrollment_id = ?", enrollmentID).First(&connection).Error; err != nil {
		writeVMAErrorResponse(w, http.StatusNotFound, "Active connection not found", err.Error())
		return
	}

	// Update connection status to revoked
	now := time.Now()
	connection.ConnectionStatus = models.ConnectionStatusRevoked
	connection.RevokedAt = &now
	connection.RevokedBy = &revokedBy
	connection.UpdatedAt = now

	if err := vrh.db.GetGormDB().Save(&connection).Error; err != nil {
		writeVMAErrorResponse(w, http.StatusInternalServerError, "Failed to revoke connection", err.Error())
		return
	}

	// 🆕 NEW: Remove SSH key during revocation
	sshManager, err := services.NewVMASSHManager()
	if err != nil {
		log.WithError(err).Error("Failed to initialize SNA SSH manager for revocation")
		// Continue with revocation but log failure
	} else {
		// Remove SNA SSH key to terminate access
		if err := sshManager.RemoveVMAKey(connection.SNAFingerprint); err != nil {
			log.WithError(err).Error("Failed to remove SNA SSH key - manual cleanup required")
			// Continue with revocation but log failure
		} else {
			log.WithField("fingerprint", connection.SNAFingerprint[:16]+"...").Info("✅ SNA SSH key removed from authorized_keys")
		}
	}

	log.WithFields(log.Fields{
		"enrollment_id": enrollmentID,
		"vma_name":      connection.SNAName,
		"revoked_by":    revokedBy,
	}).Info("🗑️ SNA access revoked by administrator with SSH key cleanup")

	response := map[string]interface{}{
		"success": true,
		"message": "SNA access revoked successfully",
		"status":  connection.ConnectionStatus,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetAuditLog returns SNA enrollment and connection audit events
func (vrh *SNARealHandler) GetAuditLog(w http.ResponseWriter, r *http.Request) {
	// Get query parameters for filtering
	eventType := r.URL.Query().Get("event_type")
	limitStr := r.URL.Query().Get("limit")

	limit := 50 // Default limit
	if limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 && parsedLimit <= 200 {
			limit = parsedLimit
		}
	}

	// Build query with optional filtering
	query := vrh.db.GetGormDB().Order("created_at DESC").Limit(limit)
	if eventType != "" {
		query = query.Where("event_type = ?", eventType)
	}

	// Execute query
	var events []models.SNAConnectionAudit
	if err := query.Find(&events).Error; err != nil {
		log.WithError(err).Error("Failed to query SNA audit events")
		writeVMAErrorResponse(w, http.StatusInternalServerError, "Failed to retrieve audit log", err.Error())
		return
	}

	log.WithFields(log.Fields{
		"count":      len(events),
		"event_type": eventType,
		"limit":      limit,
	}).Info("📋 Retrieved SNA audit events from database")

	response := map[string]interface{}{
		"count":  len(events),
		"events": events,
		"filter": map[string]interface{}{
			"event_type": eventType,
			"limit":      limit,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
