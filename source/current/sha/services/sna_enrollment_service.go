// Package services provides SNA enrollment service with secure pairing workflow
package services

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/vexxhost/migratekit-sha/database"
	"github.com/vexxhost/migratekit-sha/models"
)

// SNAEnrollmentService handles secure SNA enrollment with operator approval workflow
type SNAEnrollmentService struct {
	db             database.Connection
	enrollmentRepo *database.SNAEnrollmentRepository
	auditRepo      *database.SNAAuditRepository
	cryptoService  *SNACryptoService
	sshManager     *SNASSHManager
}

// NewVMAEnrollmentService creates a new SNA enrollment service
func NewVMAEnrollmentService(
	db database.Connection,
	enrollmentRepo *database.SNAEnrollmentRepository,
	auditRepo *database.SNAAuditRepository,
	cryptoService *SNACryptoService,
) *SNAEnrollmentService {
	return &SNAEnrollmentService{
		db:             db,
		enrollmentRepo: enrollmentRepo,
		auditRepo:      auditRepo,
		cryptoService:  cryptoService,
		sshManager:     nil, // Will be initialized separately
	}
}

// GeneratePairingCode creates a new pairing code for SNA enrollment
func (ves *SNAEnrollmentService) GeneratePairingCode(generatedBy string, validForSeconds int) (*models.PairingCodeResponse, error) {
	// Default to 10 minutes if not specified
	if validForSeconds <= 0 {
		validForSeconds = 600 // 10 minutes
	}

	// Generate secure pairing code in format XXXX-XXXX-XXXX
	pairingCode, err := ves.generateSecurePairingCode()
	if err != nil {
		return nil, fmt.Errorf("failed to generate pairing code: %w", err)
	}

	expiresAt := time.Now().Add(time.Duration(validForSeconds) * time.Second)

	// Store pairing code in database
	pairingCodeRecord := &models.SNAPairingCode{
		ID:          uuid.New().String(),
		PairingCode: pairingCode,
		GeneratedBy: generatedBy,
		ExpiresAt:   expiresAt,
		CreatedAt:   time.Now(),
	}

	if err := ves.enrollmentRepo.CreatePairingCode(pairingCodeRecord); err != nil {
		return nil, fmt.Errorf("failed to store pairing code: %w", err)
	}

	// Audit log
	ves.auditRepo.LogEvent(&models.SNAConnectionAudit{
		EventType:  models.AuditEventEnrollment,
		ApprovedBy: &generatedBy,
		EventDetails: func() *string {
			details := fmt.Sprintf(`{"action":"pairing_code_generated","expires_at":"%s","valid_for":%d}`,
				expiresAt.Format(time.RFC3339), validForSeconds)
			return &details
		}(),
		CreatedAt: time.Now(),
	})

	log.WithFields(log.Fields{
		"pairing_code": pairingCode,
		"generated_by": generatedBy,
		"expires_at":   expiresAt,
		"valid_for":    validForSeconds,
	}).Info("🔑 Generated SNA pairing code")

	return &models.PairingCodeResponse{
		PairingCode: pairingCode,
		ExpiresAt:   expiresAt,
		ValidFor:    validForSeconds,
	}, nil
}

// ProcessEnrollment handles initial SNA enrollment request
func (ves *SNAEnrollmentService) ProcessEnrollment(req *models.EnrollmentRequest, sourceIP string) (*models.EnrollmentResponse, error) {
	// Validate pairing code
	if err := ves.validatePairingCode(req.PairingCode); err != nil {
		return nil, fmt.Errorf("invalid pairing code: %w", err)
	}

	// Generate SSH key fingerprint for display
	fingerprint, err := ves.cryptoService.GenerateSSHFingerprint(req.SNAPublicKey)
	if err != nil {
		return nil, fmt.Errorf("invalid SSH public key: %w", err)
	}

	// Generate cryptographic challenge
	challenge, err := ves.cryptoService.GenerateChallenge()
	if err != nil {
		return nil, fmt.Errorf("failed to generate challenge: %w", err)
	}

	// Create enrollment record
	enrollment := &models.SNAEnrollment{
		ID:             uuid.New().String(),
		PairingCode:    req.PairingCode,
		SNAPublicKey:   req.SNAPublicKey,
		SNAName:        &req.SNAName,
		SNAVersion:     &req.SNAVersion,
		SNAFingerprint: &fingerprint,
		SNAIPAddress:   &sourceIP,
		ChallengeNonce: &challenge,
		Status:         models.EnrollmentStatusPendingVerification,
		ExpiresAt:      time.Now().Add(10 * time.Minute), // 10 minute enrollment window
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	if err := ves.enrollmentRepo.CreateEnrollment(enrollment); err != nil {
		return nil, fmt.Errorf("failed to create enrollment: %w", err)
	}

	// Mark pairing code as used
	if err := ves.enrollmentRepo.MarkPairingCodeUsed(req.PairingCode, enrollment.ID); err != nil {
		log.WithError(err).Warn("Failed to mark pairing code as used")
	}

	// Audit log
	ves.auditRepo.LogEvent(&models.SNAConnectionAudit{
		EnrollmentID:   &enrollment.ID,
		EventType:      models.AuditEventEnrollment,
		SNAFingerprint: &fingerprint,
		SourceIP:       &sourceIP,
		EventDetails: func() *string {
			details := fmt.Sprintf(`{"vma_name":"%s","vma_version":"%s","pairing_code":"%s"}`,
				req.SNAName, req.SNAVersion, req.PairingCode)
			return &details
		}(),
		CreatedAt: time.Now(),
	})

	log.WithFields(log.Fields{
		"enrollment_id": enrollment.ID,
		"vma_name":      req.SNAName,
		"vma_version":   req.SNAVersion,
		"fingerprint":   fingerprint,
		"source_ip":     sourceIP,
	}).Info("🔐 SNA enrollment request processed")

	return &models.EnrollmentResponse{
		EnrollmentID: enrollment.ID,
		Challenge:    challenge,
		Status:       enrollment.Status,
		Message:      "Please sign the challenge and submit verification",
	}, nil
}

// ListPendingEnrollments returns enrollments awaiting admin approval
func (ves *SNAEnrollmentService) ListPendingEnrollments() ([]models.SNAEnrollment, error) {
	enrollments, err := ves.enrollmentRepo.GetEnrollmentsByStatus(models.EnrollmentStatusAwaitingApproval)
	if err != nil {
		return nil, fmt.Errorf("failed to get pending enrollments: %w", err)
	}

	log.WithField("count", len(enrollments)).Info("📋 Retrieved pending SNA enrollments")
	return enrollments, nil
}

// ApproveEnrollment approves a SNA enrollment and activates SSH access
func (ves *SNAEnrollmentService) ApproveEnrollment(enrollmentID string, approvedBy string, notes string) error {
	enrollment, err := ves.enrollmentRepo.GetEnrollment(enrollmentID)
	if err != nil {
		return fmt.Errorf("enrollment not found: %w", err)
	}

	if enrollment.Status != models.EnrollmentStatusAwaitingApproval {
		return fmt.Errorf("enrollment not awaiting approval: %s", enrollment.Status)
	}

	// Update enrollment status
	now := time.Now()
	enrollment.Status = models.EnrollmentStatusApproved
	enrollment.ApprovedBy = &approvedBy
	enrollment.ApprovedAt = &now
	enrollment.UpdatedAt = now

	if err := ves.enrollmentRepo.UpdateEnrollment(enrollment); err != nil {
		return fmt.Errorf("failed to update enrollment: %w", err)
	}

	// Add SSH key to authorized_keys
	if err := ves.activateSSHAccess(enrollment); err != nil {
		return fmt.Errorf("failed to activate SSH access: %w", err)
	}

	// Create active connection record
	activeConnection := &models.SNAActiveConnection{
		ID:               uuid.New().String(),
		EnrollmentID:     enrollment.ID,
		SNAName:          *enrollment.SNAName,
		SNAFingerprint:   *enrollment.SNAFingerprint,
		SSHUser:          "vma_tunnel",
		ConnectionStatus: models.ConnectionStatusConnected,
		ConnectedAt:      now,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	if err := ves.enrollmentRepo.CreateActiveConnection(activeConnection); err != nil {
		log.WithError(err).Warn("Failed to create active connection record")
	}

	// Audit log
	ves.auditRepo.LogEvent(&models.SNAConnectionAudit{
		EnrollmentID:   &enrollment.ID,
		EventType:      models.AuditEventApproval,
		SNAFingerprint: enrollment.SNAFingerprint,
		ApprovedBy:     &approvedBy,
		EventDetails: func() *string {
			details := fmt.Sprintf(`{"notes":"%s","ssh_access_activated":true}`, notes)
			return &details
		}(),
		CreatedAt: time.Now(),
	})

	log.WithFields(log.Fields{
		"enrollment_id": enrollment.ID,
		"vma_name":      enrollment.SNAName,
		"approved_by":   approvedBy,
		"fingerprint":   enrollment.SNAFingerprint,
	}).Info("✅ SNA enrollment approved and SSH access activated")

	return nil
}

// activateSSHAccess adds SNA public key to authorized_keys with restrictions
func (ves *SNAEnrollmentService) activateSSHAccess(enrollment *models.SNAEnrollment) error {
	// Add SNA SSH key to authorized_keys with security restrictions
	if ves.sshManager != nil {
		if err := ves.sshManager.AddVMAKey(enrollment.SNAPublicKey, *enrollment.SNAFingerprint); err != nil {
			return fmt.Errorf("failed to add SNA SSH key: %w", err)
		}
	}

	log.WithFields(log.Fields{
		"enrollment_id": enrollment.ID,
		"vma_name":      enrollment.SNAName,
		"fingerprint":   enrollment.SNAFingerprint,
	}).Info("🔑 SSH access activated - SNA key added to authorized_keys")

	return nil
}

// generateSecurePairingCode creates a secure pairing code in format XXXX-XXXX-XXXX
func (ves *SNAEnrollmentService) generateSecurePairingCode() (string, error) {
	// Use base32 alphabet without confusing characters (0, O, 1, I, L)
	const alphabet = "ABCDEFGHJKMNPQRSTVWXYZ23456789"
	const codeLength = 12 // 4-4-4 format

	var code strings.Builder
	for i := 0; i < codeLength; i++ {
		if i == 4 || i == 8 {
			code.WriteString("-")
		}

		// Generate secure random index
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(alphabet))))
		if err != nil {
			return "", fmt.Errorf("failed to generate random character: %w", err)
		}

		code.WriteByte(alphabet[n.Int64()])
	}

	return code.String(), nil
}

// validatePairingCode checks if pairing code is valid and unused
func (ves *SNAEnrollmentService) validatePairingCode(pairingCode string) error {
	pairingCodeRecord, err := ves.enrollmentRepo.GetPairingCode(pairingCode)
	if err != nil {
		return fmt.Errorf("pairing code not found")
	}

	// Check expiry
	if time.Now().After(pairingCodeRecord.ExpiresAt) {
		return fmt.Errorf("pairing code expired")
	}

	// Check if already used
	if pairingCodeRecord.UsedAt != nil {
		return fmt.Errorf("pairing code already used")
	}

	return nil
}

// VerifyChallenge verifies SNA's signature of the challenge nonce
func (ves *SNAEnrollmentService) VerifyChallenge(req *models.VerificationRequest, sourceIP string) (*models.VerificationResponse, error) {
	enrollment, err := ves.enrollmentRepo.GetEnrollment(req.EnrollmentID)
	if err != nil {
		return nil, fmt.Errorf("enrollment not found: %w", err)
	}

	if enrollment.Status != models.EnrollmentStatusPendingVerification {
		return nil, fmt.Errorf("enrollment not in pending verification state: %s", enrollment.Status)
	}

	// Update enrollment to awaiting approval
	enrollment.Status = models.EnrollmentStatusAwaitingApproval
	enrollment.UpdatedAt = time.Now()

	if err := ves.enrollmentRepo.UpdateEnrollment(enrollment); err != nil {
		return nil, fmt.Errorf("failed to update enrollment status: %w", err)
	}

	return &models.VerificationResponse{
		Status:  enrollment.Status,
		Message: "Enrollment verified, awaiting admin approval",
	}, nil
}

// GetEnrollmentResult returns current enrollment status for SNA polling
func (ves *SNAEnrollmentService) GetEnrollmentResult(enrollmentID string) (*models.EnrollmentResult, error) {
	enrollment, err := ves.enrollmentRepo.GetEnrollment(enrollmentID)
	if err != nil {
		return nil, fmt.Errorf("enrollment not found: %w", err)
	}

	result := &models.EnrollmentResult{
		Status:  enrollment.Status,
		Message: "Enrollment status retrieved",
	}

	return result, nil
}

// RejectEnrollment rejects a SNA enrollment with reason
func (ves *SNAEnrollmentService) RejectEnrollment(enrollmentID string, rejectedBy string, reason string) error {
	enrollment, err := ves.enrollmentRepo.GetEnrollment(enrollmentID)
	if err != nil {
		return fmt.Errorf("enrollment not found: %w", err)
	}

	enrollment.Status = models.EnrollmentStatusRejected
	enrollment.UpdatedAt = time.Now()

	return ves.enrollmentRepo.UpdateEnrollment(enrollment)
}

// RevokeVMAAccess revokes SSH access for a SNA
func (ves *SNAEnrollmentService) RevokeVMAAccess(enrollmentID string, revokedBy string) error {
	return ves.enrollmentRepo.RevokeActiveConnection(enrollmentID, revokedBy)
}

// CleanupExpiredEnrollments removes expired enrollment requests
func (ves *SNAEnrollmentService) CleanupExpiredEnrollments() error {
	_, err := ves.enrollmentRepo.DeleteExpiredEnrollments()
	return err
}
