// Package models defines data structures for VMA enrollment system
package models

import (
	"time"
)

// VMAEnrollment represents a VMA enrollment request with approval workflow
type VMAEnrollment struct {
	ID             string     `json:"id" gorm:"primaryKey;column:id"`
	PairingCode    string     `json:"pairing_code" gorm:"column:pairing_code"`
	VMAPublicKey   string     `json:"vma_public_key" gorm:"column:vma_public_key"`
	VMAName        *string    `json:"vma_name" gorm:"column:vma_name"`
	VMAVersion     *string    `json:"vma_version" gorm:"column:vma_version"`
	VMAFingerprint *string    `json:"vma_fingerprint" gorm:"column:vma_fingerprint"`
	VMAIPAddress   *string    `json:"vma_ip_address" gorm:"column:vma_ip_address"`
	ChallengeNonce *string    `json:"challenge_nonce" gorm:"column:challenge_nonce"`
	Status         string     `json:"status" gorm:"column:status"`
	ApprovedBy     *string    `json:"approved_by" gorm:"column:approved_by"`
	ApprovedAt     *time.Time `json:"approved_at" gorm:"column:approved_at"`
	ExpiresAt      time.Time  `json:"expires_at" gorm:"column:expires_at"`
	CreatedAt      time.Time  `json:"created_at" gorm:"column:created_at"`
	UpdatedAt      time.Time  `json:"updated_at" gorm:"column:updated_at"`
}

// TableName specifies the table name for GORM
func (VMAEnrollment) TableName() string {
	return "vma_enrollments"
}

// VMAConnectionAudit represents audit trail for VMA connection events
type VMAConnectionAudit struct {
	ID             int64     `json:"id" db:"id"`
	EnrollmentID   *string   `json:"enrollment_id" db:"enrollment_id"`
	EventType      string    `json:"event_type" db:"event_type"`
	VMAFingerprint *string   `json:"vma_fingerprint" db:"vma_fingerprint"`
	SourceIP       *string   `json:"source_ip" db:"source_ip"`
	UserAgent      *string   `json:"user_agent" db:"user_agent"`
	ApprovedBy     *string   `json:"approved_by" db:"approved_by"`
	EventDetails   *string   `json:"event_details" db:"event_details"` // JSON string
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
}

// TableName specifies the table name for GORM
func (VMAConnectionAudit) TableName() string {
	return "vma_connection_audit"
}

// VMAActiveConnection represents an active VMA tunnel connection
type VMAActiveConnection struct {
	ID               string     `json:"id" db:"id"`
	EnrollmentID     string     `json:"enrollment_id" db:"enrollment_id"`
	VMAName          string     `json:"vma_name" gorm:"column:vma_name"`
	VMAFingerprint   string     `json:"vma_fingerprint" gorm:"column:vma_fingerprint"`
	SSHUser          string     `json:"ssh_user" db:"ssh_user"`
	ConnectionStatus string     `json:"connection_status" db:"connection_status"`
	LastSeenAt       *time.Time `json:"last_seen_at" db:"last_seen_at"`
	ConnectedAt      time.Time  `json:"connected_at" db:"connected_at"`
	RevokedAt        *time.Time `json:"revoked_at" db:"revoked_at"`
	RevokedBy        *string    `json:"revoked_by" db:"revoked_by"`
	CreatedAt        time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at" db:"updated_at"`
}

// TableName specifies the table name for GORM
func (VMAActiveConnection) TableName() string {
	return "vma_active_connections"
}

// VMAPairingCode represents generated pairing codes for audit
type VMAPairingCode struct {
	ID                 string     `json:"id" gorm:"primaryKey;column:id"`
	PairingCode        string     `json:"pairing_code" gorm:"uniqueIndex;column:pairing_code"`
	GeneratedBy        string     `json:"generated_by" gorm:"column:generated_by"`
	UsedByEnrollmentID *string    `json:"used_by_enrollment_id" gorm:"column:used_by_enrollment_id"`
	ExpiresAt          time.Time  `json:"expires_at" gorm:"column:expires_at"`
	UsedAt             *time.Time `json:"used_at" gorm:"column:used_at"`
	CreatedAt          time.Time  `json:"created_at" gorm:"column:created_at"`
}

// TableName specifies the table name for GORM
func (VMAPairingCode) TableName() string {
	return "vma_pairing_codes"
}

// EnrollmentRequest represents the initial VMA enrollment request
type EnrollmentRequest struct {
	PairingCode    string `json:"pairing_code" binding:"required" example:"AX7K-PJ3F-TH2Q"`
	VMAPublicKey   string `json:"vma_public_key" binding:"required" example:"ssh-ed25519 AAAAC3NzaC1lZDI1NTE5..."`
	VMAName        string `json:"vma_name" example:"Production VMA 01"`
	VMAVersion     string `json:"vma_version" example:"v2.20.1"`
	VMAFingerprint string `json:"vma_fingerprint" example:"SHA256:abc123..."`
}

// EnrollmentResponse represents the response to initial enrollment
type EnrollmentResponse struct {
	EnrollmentID string `json:"enrollment_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Challenge    string `json:"challenge" example:"base64-encoded-32-byte-nonce"`
	Status       string `json:"status" example:"pending_verification"`
	Message      string `json:"message" example:"Please sign the challenge and submit verification"`
}

// VerificationRequest represents challenge signature verification
type VerificationRequest struct {
	EnrollmentID string `json:"enrollment_id" binding:"required" example:"550e8400-e29b-41d4-a716-446655440000"`
	Signature    string `json:"signature" binding:"required" example:"base64-encoded-ed25519-signature"`
}

// VerificationResponse represents verification result
type VerificationResponse struct {
	Status  string `json:"status" example:"awaiting_approval"`
	Message string `json:"message" example:"Enrollment verified, awaiting admin approval"`
}

// EnrollmentResult represents the final enrollment result
type EnrollmentResult struct {
	Status      string `json:"status" example:"approved"`
	SSHUser     string `json:"ssh_user,omitempty" example:"vma_tunnel"`
	SSHOptions  string `json:"ssh_options,omitempty" example:"restrict,permitopen=..."`
	HostKeyHash string `json:"host_key_hash,omitempty" example:"SHA256:abc123..."`
	Message     string `json:"message,omitempty" example:"Enrollment approved, connection authorized"`
}

// AdminApprovalRequest represents admin approval action
type AdminApprovalRequest struct {
	ApprovedBy string `json:"approved_by" binding:"required" example:"admin@company.com"`
	Notes      string `json:"notes" example:"Approved for production migration project"`
}

// AdminRejectionRequest represents admin rejection action
type AdminRejectionRequest struct {
	RejectedBy string `json:"rejected_by" binding:"required" example:"admin@company.com"`
	Reason     string `json:"reason" binding:"required" example:"Unauthorized VMA enrollment attempt"`
}

// PairingCodeRequest represents pairing code generation request
type PairingCodeRequest struct {
	GeneratedBy string `json:"generated_by" binding:"required" example:"admin@company.com"`
	ValidFor    int    `json:"valid_for" example:"600"` // Seconds, default 10 minutes
}

// PairingCodeResponse represents generated pairing code
type PairingCodeResponse struct {
	PairingCode string    `json:"pairing_code" example:"AX7K-PJ3F-TH2Q"`
	ExpiresAt   time.Time `json:"expires_at" example:"2025-09-28T21:45:00Z"`
	ValidFor    int       `json:"valid_for" example:"600"` // Seconds
}

// Enrollment status constants
const (
	EnrollmentStatusPendingVerification = "pending_verification"
	EnrollmentStatusAwaitingApproval    = "awaiting_approval"
	EnrollmentStatusApproved            = "approved"
	EnrollmentStatusRejected            = "rejected"
	EnrollmentStatusExpired             = "expired"
)

// Connection status constants
const (
	ConnectionStatusConnected    = "connected"
	ConnectionStatusDisconnected = "disconnected"
	ConnectionStatusRevoked      = "revoked"
)

// Audit event types
const (
	AuditEventEnrollment    = "enrollment"
	AuditEventVerification  = "verification"
	AuditEventApproval      = "approval"
	AuditEventRejection     = "rejection"
	AuditEventConnection    = "connection"
	AuditEventDisconnection = "disconnection"
	AuditEventRevocation    = "revocation"
)
