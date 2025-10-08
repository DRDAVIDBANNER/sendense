// Package services provides SNA audit service for security event tracking
package services

import (
	"context"
	"encoding/json"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/vexxhost/migratekit-sha/database"
	"github.com/vexxhost/migratekit-sha/models"
)

// SNAAuditService handles security audit logging for SNA enrollment system
type SNAAuditService struct {
	auditRepo *database.SNAAuditRepository
}

// NewVMAAuditService creates a new SNA audit service
func NewVMAAuditService(auditRepo *database.SNAAuditRepository) *SNAAuditService {
	return &SNAAuditService{
		auditRepo: auditRepo,
	}
}

// LogEvent logs a SNA security event to the audit trail
func (vas *SNAAuditService) LogEvent(audit *models.SNAConnectionAudit) error {
	// Set timestamp if not provided
	if audit.CreatedAt.IsZero() {
		audit.CreatedAt = time.Now()
	}

	// Store audit event
	if err := vas.auditRepo.LogEvent(audit); err != nil {
		log.WithError(err).WithFields(log.Fields{
			"event_type":      audit.EventType,
			"enrollment_id":   audit.EnrollmentID,
			"vma_fingerprint": audit.SNAFingerprint,
			"source_ip":       audit.SourceIP,
		}).Error("Failed to log SNA audit event")
		return err
	}

	// Also log to application logs for immediate visibility
	vas.logToApplicationLog(audit)

	return nil
}

// LogEnrollmentEvent logs SNA enrollment-specific events
func (vas *SNAAuditService) LogEnrollmentEvent(ctx context.Context, eventType string, enrollmentID string, details map[string]interface{}) error {
	detailsJSON, _ := json.Marshal(details)
	detailsStr := string(detailsJSON)

	audit := &models.SNAConnectionAudit{
		EnrollmentID: &enrollmentID,
		EventType:    eventType,
		EventDetails: &detailsStr,
		CreatedAt:    time.Now(),
	}

	return vas.LogEvent(audit)
}

// LogConnectionEvent logs SNA connection events (connect, disconnect, etc.)
func (vas *SNAAuditService) LogConnectionEvent(ctx context.Context, eventType string, snaFingerprint string, sourceIP string, details map[string]interface{}) error {
	detailsJSON, _ := json.Marshal(details)
	detailsStr := string(detailsJSON)

	audit := &models.SNAConnectionAudit{
		EventType:      eventType,
		SNAFingerprint: &snaFingerprint,
		SourceIP:       &sourceIP,
		EventDetails:   &detailsStr,
		CreatedAt:      time.Now(),
	}

	return vas.LogEvent(audit)
}

// LogAdminAction logs admin actions (approve, reject, revoke)
func (vas *SNAAuditService) LogAdminAction(ctx context.Context, eventType string, enrollmentID string, adminUser string, details map[string]interface{}) error {
	detailsJSON, _ := json.Marshal(details)
	detailsStr := string(detailsJSON)

	audit := &models.SNAConnectionAudit{
		EnrollmentID: &enrollmentID,
		EventType:    eventType,
		ApprovedBy:   &adminUser,
		EventDetails: &detailsStr,
		CreatedAt:    time.Now(),
	}

	return vas.LogEvent(audit)
}

// GetAuditLog retrieves audit log entries with filtering and pagination
func (vas *SNAAuditService) GetAuditLog(filter database.AuditLogFilter) ([]models.SNAConnectionAudit, error) {
	events, err := vas.auditRepo.GetAuditLog(filter)
	if err != nil {
		return nil, err
	}

	log.WithFields(log.Fields{
		"count":      len(events),
		"event_type": filter.EventType,
		"limit":      filter.Limit,
		"offset":     filter.Offset,
	}).Debug("📋 Retrieved SNA audit log entries")

	return events, nil
}

// GetAuditStatistics returns audit statistics for monitoring
func (vas *SNAAuditService) GetAuditStatistics() (*database.AuditStatistics, error) {
	stats, err := vas.auditRepo.GetAuditStatistics()
	if err != nil {
		return nil, err
	}

	log.WithFields(log.Fields{
		"total_events":      stats.TotalEvents,
		"enrollments_today": stats.EnrollmentsToday,
		"approvals_today":   stats.ApprovalsToday,
		"connections_today": stats.ConnectionsToday,
	}).Debug("📊 Retrieved SNA audit statistics")

	return stats, nil
}

// logToApplicationLog writes audit events to application logs for immediate visibility
func (vas *SNAAuditService) logToApplicationLog(audit *models.SNAConnectionAudit) {
	fields := log.Fields{
		"event_type": audit.EventType,
		"timestamp":  audit.CreatedAt,
	}

	if audit.EnrollmentID != nil {
		fields["enrollment_id"] = *audit.EnrollmentID
	}
	if audit.SNAFingerprint != nil {
		fields["vma_fingerprint"] = *audit.SNAFingerprint
	}
	if audit.SourceIP != nil {
		fields["source_ip"] = *audit.SourceIP
	}
	if audit.ApprovedBy != nil {
		fields["admin_user"] = *audit.ApprovedBy
	}

	switch audit.EventType {
	case models.AuditEventEnrollment:
		log.WithFields(fields).Info("🔐 SNA enrollment initiated")
	case models.AuditEventVerification:
		log.WithFields(fields).Info("✅ SNA challenge verified")
	case models.AuditEventApproval:
		log.WithFields(fields).Info("✅ SNA enrollment approved")
	case models.AuditEventRejection:
		log.WithFields(fields).Info("❌ SNA enrollment rejected")
	case models.AuditEventConnection:
		log.WithFields(fields).Info("🔗 SNA tunnel connected")
	case models.AuditEventDisconnection:
		log.WithFields(fields).Info("🔌 SNA tunnel disconnected")
	case models.AuditEventRevocation:
		log.WithFields(fields).Info("🚫 SNA access revoked")
	default:
		log.WithFields(fields).Info("📋 SNA audit event")
	}
}
