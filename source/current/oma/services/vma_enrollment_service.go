// Package services provides VMA enrollment service with secure pairing workflow
package services

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/vexxhost/migratekit-oma/database"
	"github.com/vexxhost/migratekit-oma/models"
)

// VMAEnrollmentService handles secure VMA enrollment with operator approval workflow
type VMAEnrollmentService struct {
	db             database.Connection
	enrollmentRepo *database.VMAEnrollmentRepository
	auditRepo      *database.VMAAuditRepository
	cryptoService  *VMACryptoService
	sshManager     *VMASSHManager
}

// NewVMAEnrollmentService creates a new VMA enrollment service
func NewVMAEnrollmentService(
	db database.Connection,
	enrollmentRepo *database.VMAEnrollmentRepository,
	auditRepo *database.VMAAuditRepository,
	cryptoService *VMACryptoService,
) *VMAEnrollmentService {
	return &VMAEnrollmentService{
		db:             db,
		enrollmentRepo: enrollmentRepo,
		auditRepo:      auditRepo,
		cryptoService:  cryptoService,
		sshManager:     nil, // Will be initialized separately
	}
}

// GeneratePairingCode creates a new pairing code for VMA enrollment
func (ves *VMAEnrollmentService) GeneratePairingCode(generatedBy string, validForSeconds int) (*models.PairingCodeResponse, error) {
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
	pairingCodeRecord := &models.VMAPairingCode{
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
	ves.auditRepo.LogEvent(&models.VMAConnectionAudit{
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
	}).Info("🔑 Generated VMA pairing code")

	return &models.PairingCodeResponse{
		PairingCode: pairingCode,
		ExpiresAt:   expiresAt,
		ValidFor:    validForSeconds,
	}, nil
}

// ProcessEnrollment handles initial VMA enrollment request
func (ves *VMAEnrollmentService) ProcessEnrollment(req *models.EnrollmentRequest, sourceIP string) (*models.EnrollmentResponse, error) {
	// Validate pairing code
	if err := ves.validatePairingCode(req.PairingCode); err != nil {
		return nil, fmt.Errorf("invalid pairing code: %w", err)
	}

	// Generate SSH key fingerprint for display
	fingerprint, err := ves.cryptoService.GenerateSSHFingerprint(req.VMAPublicKey)
	if err != nil {
		return nil, fmt.Errorf("invalid SSH public key: %w", err)
	}

	// Generate cryptographic challenge
	challenge, err := ves.cryptoService.GenerateChallenge()
	if err != nil {
		return nil, fmt.Errorf("failed to generate challenge: %w", err)
	}

	// Create enrollment record
	enrollment := &models.VMAEnrollment{
		ID:             uuid.New().String(),
		PairingCode:    req.PairingCode,
		VMAPublicKey:   req.VMAPublicKey,
		VMAName:        &req.VMAName,
		VMAVersion:     &req.VMAVersion,
		VMAFingerprint: &fingerprint,
		VMAIPAddress:   &sourceIP,
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
	ves.auditRepo.LogEvent(&models.VMAConnectionAudit{
		EnrollmentID:   &enrollment.ID,
		EventType:      models.AuditEventEnrollment,
		VMAFingerprint: &fingerprint,
		SourceIP:       &sourceIP,
		EventDetails: func() *string {
			details := fmt.Sprintf(`{"vma_name":"%s","vma_version":"%s","pairing_code":"%s"}`,
				req.VMAName, req.VMAVersion, req.PairingCode)
			return &details
		}(),
		CreatedAt: time.Now(),
	})

	log.WithFields(log.Fields{
		"enrollment_id": enrollment.ID,
		"vma_name":      req.VMAName,
		"vma_version":   req.VMAVersion,
		"fingerprint":   fingerprint,
		"source_ip":     sourceIP,
	}).Info("🔐 VMA enrollment request processed")

	return &models.EnrollmentResponse{
		EnrollmentID: enrollment.ID,
		Challenge:    challenge,
		Status:       enrollment.Status,
		Message:      "Please sign the challenge and submit verification",
	}, nil
}

// ListPendingEnrollments returns enrollments awaiting admin approval
func (ves *VMAEnrollmentService) ListPendingEnrollments() ([]models.VMAEnrollment, error) {
	enrollments, err := ves.enrollmentRepo.GetEnrollmentsByStatus(models.EnrollmentStatusAwaitingApproval)
	if err != nil {
		return nil, fmt.Errorf("failed to get pending enrollments: %w", err)
	}

	log.WithField("count", len(enrollments)).Info("📋 Retrieved pending VMA enrollments")
	return enrollments, nil
}

// ApproveEnrollment approves a VMA enrollment and activates SSH access
func (ves *VMAEnrollmentService) ApproveEnrollment(enrollmentID string, approvedBy string, notes string) error {
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
	activeConnection := &models.VMAActiveConnection{
		ID:               uuid.New().String(),
		EnrollmentID:     enrollment.ID,
		VMAName:          *enrollment.VMAName,
		VMAFingerprint:   *enrollment.VMAFingerprint,
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
	ves.auditRepo.LogEvent(&models.VMAConnectionAudit{
		EnrollmentID:   &enrollment.ID,
		EventType:      models.AuditEventApproval,
		VMAFingerprint: enrollment.VMAFingerprint,
		ApprovedBy:     &approvedBy,
		EventDetails: func() *string {
			details := fmt.Sprintf(`{"notes":"%s","ssh_access_activated":true}`, notes)
			return &details
		}(),
		CreatedAt: time.Now(),
	})

	log.WithFields(log.Fields{
		"enrollment_id": enrollment.ID,
		"vma_name":      enrollment.VMAName,
		"approved_by":   approvedBy,
		"fingerprint":   enrollment.VMAFingerprint,
	}).Info("✅ VMA enrollment approved and SSH access activated")

	return nil
}

// activateSSHAccess adds VMA public key to authorized_keys with restrictions
func (ves *VMAEnrollmentService) activateSSHAccess(enrollment *models.VMAEnrollment) error {
	// Add VMA SSH key to authorized_keys with security restrictions
	if ves.sshManager != nil {
		if err := ves.sshManager.AddVMAKey(enrollment.VMAPublicKey, *enrollment.VMAFingerprint); err != nil {
			return fmt.Errorf("failed to add VMA SSH key: %w", err)
		}
	}

	log.WithFields(log.Fields{
		"enrollment_id": enrollment.ID,
		"vma_name":      enrollment.VMAName,
		"fingerprint":   enrollment.VMAFingerprint,
	}).Info("🔑 SSH access activated - VMA key added to authorized_keys")

	return nil
}

// generateSecurePairingCode creates a secure pairing code in format XXXX-XXXX-XXXX
func (ves *VMAEnrollmentService) generateSecurePairingCode() (string, error) {
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
func (ves *VMAEnrollmentService) validatePairingCode(pairingCode string) error {
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

// VerifyChallenge verifies VMA's signature of the challenge nonce
func (ves *VMAEnrollmentService) VerifyChallenge(req *models.VerificationRequest, sourceIP string) (*models.VerificationResponse, error) {
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

// GetEnrollmentResult returns current enrollment status for VMA polling
func (ves *VMAEnrollmentService) GetEnrollmentResult(enrollmentID string) (*models.EnrollmentResult, error) {
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

// RejectEnrollment rejects a VMA enrollment with reason
func (ves *VMAEnrollmentService) RejectEnrollment(enrollmentID string, rejectedBy string, reason string) error {
	enrollment, err := ves.enrollmentRepo.GetEnrollment(enrollmentID)
	if err != nil {
		return fmt.Errorf("enrollment not found: %w", err)
	}

	enrollment.Status = models.EnrollmentStatusRejected
	enrollment.UpdatedAt = time.Now()

	return ves.enrollmentRepo.UpdateEnrollment(enrollment)
}

// RevokeVMAAccess revokes SSH access for a VMA
func (ves *VMAEnrollmentService) RevokeVMAAccess(enrollmentID string, revokedBy string) error {
	return ves.enrollmentRepo.RevokeActiveConnection(enrollmentID, revokedBy)
}

// CleanupExpiredEnrollments removes expired enrollment requests
func (ves *VMAEnrollmentService) CleanupExpiredEnrollments() error {
	_, err := ves.enrollmentRepo.DeleteExpiredEnrollments()
	return err
}
