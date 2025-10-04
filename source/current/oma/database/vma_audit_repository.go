// Package database provides VMA audit repository implementation
package database

import (
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/vexxhost/migratekit-oma/models"
	"gorm.io/gorm"
)

// VMAAuditRepository handles database operations for VMA audit trail
type VMAAuditRepository struct {
	db *gorm.DB
}

// NewVMAAuditRepository creates a new VMA audit repository
func NewVMAAuditRepository(conn Connection) *VMAAuditRepository {
	return &VMAAuditRepository{
		db: conn.GetGormDB(),
	}
}

// LogEvent stores a VMA security event in the audit trail
func (vr *VMAAuditRepository) LogEvent(audit *models.VMAConnectionAudit) error {
	if vr.db == nil {
		return fmt.Errorf("database not available")
	}

	if err := vr.db.Create(audit).Error; err != nil {
		log.WithError(err).WithFields(log.Fields{
			"event_type":      audit.EventType,
			"enrollment_id":   audit.EnrollmentID,
			"vma_fingerprint": audit.VMAFingerprint,
		}).Error("Failed to log audit event")
		return fmt.Errorf("failed to log audit event: %w", err)
	}

	log.WithFields(log.Fields{
		"event_type":      audit.EventType,
		"enrollment_id":   audit.EnrollmentID,
		"vma_fingerprint": audit.VMAFingerprint,
		"source_ip":       audit.SourceIP,
	}).Debug("💾 Logged VMA audit event")

	return nil
}

// GetAuditLog retrieves audit log entries with filtering and pagination
func (vr *VMAAuditRepository) GetAuditLog(filter AuditLogFilter) ([]models.VMAConnectionAudit, error) {
	if vr.db == nil {
		return nil, fmt.Errorf("database not available")
	}

	query := vr.db.Model(&models.VMAConnectionAudit{})

	// Add filters
	if filter.EventType != "" {
		query = query.Where("event_type = ?", filter.EventType)
	}
	if filter.EnrollmentID != "" {
		query = query.Where("enrollment_id = ?", filter.EnrollmentID)
	}
	if filter.VMAFingerprint != "" {
		query = query.Where("vma_fingerprint = ?", filter.VMAFingerprint)
	}
	if filter.SourceIP != "" {
		query = query.Where("source_ip = ?", filter.SourceIP)
	}
	if filter.AdminUser != "" {
		query = query.Where("approved_by = ?", filter.AdminUser)
	}
	if !filter.StartTime.IsZero() {
		query = query.Where("created_at >= ?", filter.StartTime)
	}
	if !filter.EndTime.IsZero() {
		query = query.Where("created_at <= ?", filter.EndTime)
	}

	// Order and pagination
	query = query.Order("created_at DESC")
	if filter.Limit > 0 {
		query = query.Limit(filter.Limit)
	}
	if filter.Offset > 0 {
		query = query.Offset(filter.Offset)
	}

	var events []models.VMAConnectionAudit
	if err := query.Find(&events).Error; err != nil {
		return nil, fmt.Errorf("failed to query audit log: %w", err)
	}

	log.WithFields(log.Fields{
		"count":      len(events),
		"event_type": filter.EventType,
	}).Debug("💾 Retrieved audit log entries")

	return events, nil
}

// GetAuditStatistics returns audit statistics for monitoring
func (vr *VMAAuditRepository) GetAuditStatistics() (*AuditStatistics, error) {
	if vr.db == nil {
		return nil, fmt.Errorf("database not available")
	}

	stats := &AuditStatistics{}
	today := time.Now().Format("2006-01-02")

	// Get total events
	vr.db.Model(&models.VMAConnectionAudit{}).Count(&stats.TotalEvents)

	// Get today's statistics
	vr.db.Model(&models.VMAConnectionAudit{}).
		Where("event_type = ? AND DATE(created_at) = ?", models.AuditEventEnrollment, today).
		Count(&stats.EnrollmentsToday)

	vr.db.Model(&models.VMAConnectionAudit{}).
		Where("event_type = ? AND DATE(created_at) = ?", models.AuditEventApproval, today).
		Count(&stats.ApprovalsToday)

	vr.db.Model(&models.VMAConnectionAudit{}).
		Where("event_type = ? AND DATE(created_at) = ?", models.AuditEventRejection, today).
		Count(&stats.RejectionsToday)

	vr.db.Model(&models.VMAConnectionAudit{}).
		Where("event_type = ? AND DATE(created_at) = ?", models.AuditEventConnection, today).
		Count(&stats.ConnectionsToday)

	vr.db.Model(&models.VMAConnectionAudit{}).
		Where("event_type = ? AND DATE(created_at) = ?", models.AuditEventRevocation, today).
		Count(&stats.RevocationsToday)

	// Active connections
	vr.db.Model(&models.VMAActiveConnection{}).
		Where("connection_status = ?", models.ConnectionStatusConnected).
		Count(&stats.ActiveConnections)

	log.WithFields(log.Fields{
		"total_events":       stats.TotalEvents,
		"enrollments_today":  stats.EnrollmentsToday,
		"active_connections": stats.ActiveConnections,
	}).Debug("💾 Retrieved VMA audit statistics")

	return stats, nil
}

// AuditLogFilter represents filtering options for audit log queries
type AuditLogFilter struct {
	EventType      string    `json:"event_type,omitempty"`
	EnrollmentID   string    `json:"enrollment_id,omitempty"`
	VMAFingerprint string    `json:"vma_fingerprint,omitempty"`
	SourceIP       string    `json:"source_ip,omitempty"`
	AdminUser      string    `json:"admin_user,omitempty"`
	StartTime      time.Time `json:"start_time,omitempty"`
	EndTime        time.Time `json:"end_time,omitempty"`
	Limit          int       `json:"limit,omitempty"`
	Offset         int       `json:"offset,omitempty"`
}

// AuditStatistics represents audit statistics for monitoring
type AuditStatistics struct {
	TotalEvents       int64 `json:"total_events"`
	EnrollmentsToday  int64 `json:"enrollments_today"`
	ApprovalsToday    int64 `json:"approvals_today"`
	RejectionsToday   int64 `json:"rejections_today"`
	ConnectionsToday  int64 `json:"connections_today"`
	RevocationsToday  int64 `json:"revocations_today"`
	ActiveConnections int64 `json:"active_connections"`
}
