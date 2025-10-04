// Package services provides VMA audit service for security event tracking
package services

import (
	"context"
	"encoding/json"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/vexxhost/migratekit-oma/database"
	"github.com/vexxhost/migratekit-oma/models"
)

// VMAAuditService handles security audit logging for VMA enrollment system
type VMAAuditService struct {
	auditRepo *database.VMAAuditRepository
}

// NewVMAAuditService creates a new VMA audit service
func NewVMAAuditService(auditRepo *database.VMAAuditRepository) *VMAAuditService {
	return &VMAAuditService{
		auditRepo: auditRepo,
	}
}

// LogEvent logs a VMA security event to the audit trail
func (vas *VMAAuditService) LogEvent(audit *models.VMAConnectionAudit) error {
	// Set timestamp if not provided
	if audit.CreatedAt.IsZero() {
		audit.CreatedAt = time.Now()
	}

	// Store audit event
	if err := vas.auditRepo.LogEvent(audit); err != nil {
		log.WithError(err).WithFields(log.Fields{
			"event_type":      audit.EventType,
			"enrollment_id":   audit.EnrollmentID,
			"vma_fingerprint": audit.VMAFingerprint,
			"source_ip":       audit.SourceIP,
		}).Error("Failed to log VMA audit event")
		return err
	}

	// Also log to application logs for immediate visibility
	vas.logToApplicationLog(audit)

	return nil
}

// LogEnrollmentEvent logs VMA enrollment-specific events
func (vas *VMAAuditService) LogEnrollmentEvent(ctx context.Context, eventType string, enrollmentID string, details map[string]interface{}) error {
	detailsJSON, _ := json.Marshal(details)
	detailsStr := string(detailsJSON)

	audit := &models.VMAConnectionAudit{
		EnrollmentID: &enrollmentID,
		EventType:    eventType,
		EventDetails: &detailsStr,
		CreatedAt:    time.Now(),
	}

	return vas.LogEvent(audit)
}

// LogConnectionEvent logs VMA connection events (connect, disconnect, etc.)
func (vas *VMAAuditService) LogConnectionEvent(ctx context.Context, eventType string, vmaFingerprint string, sourceIP string, details map[string]interface{}) error {
	detailsJSON, _ := json.Marshal(details)
	detailsStr := string(detailsJSON)

	audit := &models.VMAConnectionAudit{
		EventType:      eventType,
		VMAFingerprint: &vmaFingerprint,
		SourceIP:       &sourceIP,
		EventDetails:   &detailsStr,
		CreatedAt:      time.Now(),
	}

	return vas.LogEvent(audit)
}

// LogAdminAction logs admin actions (approve, reject, revoke)
func (vas *VMAAuditService) LogAdminAction(ctx context.Context, eventType string, enrollmentID string, adminUser string, details map[string]interface{}) error {
	detailsJSON, _ := json.Marshal(details)
	detailsStr := string(detailsJSON)

	audit := &models.VMAConnectionAudit{
		EnrollmentID: &enrollmentID,
		EventType:    eventType,
		ApprovedBy:   &adminUser,
		EventDetails: &detailsStr,
		CreatedAt:    time.Now(),
	}

	return vas.LogEvent(audit)
}

// GetAuditLog retrieves audit log entries with filtering and pagination
func (vas *VMAAuditService) GetAuditLog(filter database.AuditLogFilter) ([]models.VMAConnectionAudit, error) {
	events, err := vas.auditRepo.GetAuditLog(filter)
	if err != nil {
		return nil, err
	}

	log.WithFields(log.Fields{
		"count":      len(events),
		"event_type": filter.EventType,
		"limit":      filter.Limit,
		"offset":     filter.Offset,
	}).Debug("📋 Retrieved VMA audit log entries")

	return events, nil
}

// GetAuditStatistics returns audit statistics for monitoring
func (vas *VMAAuditService) GetAuditStatistics() (*database.AuditStatistics, error) {
	stats, err := vas.auditRepo.GetAuditStatistics()
	if err != nil {
		return nil, err
	}

	log.WithFields(log.Fields{
		"total_events":      stats.TotalEvents,
		"enrollments_today": stats.EnrollmentsToday,
		"approvals_today":   stats.ApprovalsToday,
		"connections_today": stats.ConnectionsToday,
	}).Debug("📊 Retrieved VMA audit statistics")

	return stats, nil
}

// logToApplicationLog writes audit events to application logs for immediate visibility
func (vas *VMAAuditService) logToApplicationLog(audit *models.VMAConnectionAudit) {
	fields := log.Fields{
		"event_type": audit.EventType,
		"timestamp":  audit.CreatedAt,
	}

	if audit.EnrollmentID != nil {
		fields["enrollment_id"] = *audit.EnrollmentID
	}
	if audit.VMAFingerprint != nil {
		fields["vma_fingerprint"] = *audit.VMAFingerprint
	}
	if audit.SourceIP != nil {
		fields["source_ip"] = *audit.SourceIP
	}
	if audit.ApprovedBy != nil {
		fields["admin_user"] = *audit.ApprovedBy
	}

	switch audit.EventType {
	case models.AuditEventEnrollment:
		log.WithFields(fields).Info("🔐 VMA enrollment initiated")
	case models.AuditEventVerification:
		log.WithFields(fields).Info("✅ VMA challenge verified")
	case models.AuditEventApproval:
		log.WithFields(fields).Info("✅ VMA enrollment approved")
	case models.AuditEventRejection:
		log.WithFields(fields).Info("❌ VMA enrollment rejected")
	case models.AuditEventConnection:
		log.WithFields(fields).Info("🔗 VMA tunnel connected")
	case models.AuditEventDisconnection:
		log.WithFields(fields).Info("🔌 VMA tunnel disconnected")
	case models.AuditEventRevocation:
		log.WithFields(fields).Info("🚫 VMA access revoked")
	default:
		log.WithFields(fields).Info("📋 VMA audit event")
	}
}
