// Package database provides SNA enrollment repository implementation
package database

import (
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/vexxhost/migratekit-sha/models"
	"gorm.io/gorm"
)

// SNAEnrollmentRepository handles database operations for SNA enrollment system
type SNAEnrollmentRepository struct {
	db *gorm.DB
}

// NewVMAEnrollmentRepository creates a new SNA enrollment repository
func NewVMAEnrollmentRepository(conn Connection) *SNAEnrollmentRepository {
	return &SNAEnrollmentRepository{
		db: conn.GetGormDB(),
	}
}

// CreatePairingCode stores a new pairing code in the database
func (ver *SNAEnrollmentRepository) CreatePairingCode(code *models.SNAPairingCode) error {
	if ver.db == nil {
		return fmt.Errorf("database not available")
	}

	if err := ver.db.Create(code).Error; err != nil {
		log.WithError(err).WithField("pairing_code", code.PairingCode).Error("Failed to create pairing code")
		return fmt.Errorf("failed to create pairing code: %w", err)
	}

	log.WithFields(log.Fields{
		"pairing_code": code.PairingCode,
		"generated_by": code.GeneratedBy,
		"expires_at":   code.ExpiresAt,
	}).Debug("💾 Created pairing code record")

	return nil
}

// GetPairingCode retrieves a pairing code by code value
func (ver *SNAEnrollmentRepository) GetPairingCode(pairingCode string) (*models.SNAPairingCode, error) {
	if ver.db == nil {
		return nil, fmt.Errorf("database not available")
	}

	var code models.SNAPairingCode
	if err := ver.db.Where("pairing_code = ?", pairingCode).First(&code).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("pairing code not found")
		}
		return nil, fmt.Errorf("failed to get pairing code: %w", err)
	}

	return &code, nil
}

// MarkPairingCodeUsed marks a pairing code as used by an enrollment
func (ver *SNAEnrollmentRepository) MarkPairingCodeUsed(pairingCode string, enrollmentID string) error {
	if ver.db == nil {
		return fmt.Errorf("database not available")
	}

	updates := map[string]interface{}{
		"used_by_enrollment_id": enrollmentID,
		"used_at":               time.Now(),
	}

	if err := ver.db.Model(&models.SNAPairingCode{}).Where("pairing_code = ?", pairingCode).Updates(updates).Error; err != nil {
		return fmt.Errorf("failed to mark pairing code as used: %w", err)
	}

	log.WithFields(log.Fields{
		"pairing_code":  pairingCode,
		"enrollment_id": enrollmentID,
	}).Debug("💾 Marked pairing code as used")

	return nil
}

// CreateEnrollment creates a new SNA enrollment record
func (ver *SNAEnrollmentRepository) CreateEnrollment(enrollment *models.SNAEnrollment) error {
	if ver.db == nil {
		return fmt.Errorf("database not available")
	}

	if err := ver.db.Create(enrollment).Error; err != nil {
		log.WithError(err).WithField("enrollment_id", enrollment.ID).Error("Failed to create enrollment")
		return fmt.Errorf("failed to create enrollment: %w", err)
	}

	log.WithFields(log.Fields{
		"enrollment_id": enrollment.ID,
		"vma_name":      enrollment.SNAName,
		"status":        enrollment.Status,
	}).Debug("💾 Created SNA enrollment record")

	return nil
}

// GetEnrollment retrieves an enrollment by ID
func (ver *SNAEnrollmentRepository) GetEnrollment(enrollmentID string) (*models.SNAEnrollment, error) {
	if ver.db == nil {
		return nil, fmt.Errorf("database not available")
	}

	var enrollment models.SNAEnrollment
	if err := ver.db.Where("id = ?", enrollmentID).First(&enrollment).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("enrollment not found")
		}
		return nil, fmt.Errorf("failed to get enrollment: %w", err)
	}

	return &enrollment, nil
}

// UpdateEnrollment updates an existing enrollment record
func (ver *SNAEnrollmentRepository) UpdateEnrollment(enrollment *models.SNAEnrollment) error {
	if ver.db == nil {
		return fmt.Errorf("database not available")
	}

	if err := ver.db.Save(enrollment).Error; err != nil {
		log.WithError(err).WithField("enrollment_id", enrollment.ID).Error("Failed to update enrollment")
		return fmt.Errorf("failed to update enrollment: %w", err)
	}

	log.WithFields(log.Fields{
		"enrollment_id": enrollment.ID,
		"status":        enrollment.Status,
		"approved_by":   enrollment.ApprovedBy,
	}).Debug("💾 Updated SNA enrollment record")

	return nil
}

// GetEnrollmentsByStatus retrieves enrollments by status
func (ver *SNAEnrollmentRepository) GetEnrollmentsByStatus(status string) ([]models.SNAEnrollment, error) {
	if ver.db == nil {
		return nil, fmt.Errorf("database not available")
	}

	var enrollments []models.SNAEnrollment
	if err := ver.db.Where("status = ?", status).Order("created_at DESC").Find(&enrollments).Error; err != nil {
		return nil, fmt.Errorf("failed to query enrollments by status: %w", err)
	}

	log.WithFields(log.Fields{
		"status": status,
		"count":  len(enrollments),
	}).Debug("💾 Retrieved enrollments by status")

	return enrollments, nil
}

// CreateActiveConnection creates an active SNA connection record
func (ver *SNAEnrollmentRepository) CreateActiveConnection(connection *models.SNAActiveConnection) error {
	if ver.db == nil {
		return fmt.Errorf("database not available")
	}

	if err := ver.db.Create(connection).Error; err != nil {
		log.WithError(err).WithField("connection_id", connection.ID).Error("Failed to create active connection")
		return fmt.Errorf("failed to create active connection: %w", err)
	}

	log.WithFields(log.Fields{
		"connection_id": connection.ID,
		"enrollment_id": connection.EnrollmentID,
		"vma_name":      connection.SNAName,
	}).Debug("💾 Created active SNA connection record")

	return nil
}

// RevokeActiveConnection marks an active connection as revoked
func (ver *SNAEnrollmentRepository) RevokeActiveConnection(enrollmentID string, revokedBy string) error {
	if ver.db == nil {
		return fmt.Errorf("database not available")
	}

	now := time.Now()
	updates := map[string]interface{}{
		"connection_status": models.ConnectionStatusRevoked,
		"revoked_at":        now,
		"revoked_by":        revokedBy,
		"updated_at":        now,
	}

	if err := ver.db.Model(&models.SNAActiveConnection{}).Where("enrollment_id = ?", enrollmentID).Updates(updates).Error; err != nil {
		return fmt.Errorf("failed to revoke active connection: %w", err)
	}

	log.WithFields(log.Fields{
		"enrollment_id": enrollmentID,
		"revoked_by":    revokedBy,
	}).Debug("💾 Revoked active SNA connection")

	return nil
}

// DeleteExpiredEnrollments removes expired enrollment requests
func (ver *SNAEnrollmentRepository) DeleteExpiredEnrollments() (int, error) {
	if ver.db == nil {
		return 0, fmt.Errorf("database not available")
	}

	result := ver.db.Where("expires_at < ? AND status NOT IN (?)",
		time.Now(),
		[]string{models.EnrollmentStatusApproved, models.EnrollmentStatusRejected}).
		Delete(&models.SNAEnrollment{})

	if result.Error != nil {
		return 0, fmt.Errorf("failed to delete expired enrollments: %w", result.Error)
	}

	rowsAffected := int(result.RowsAffected)
	if rowsAffected > 0 {
		log.WithField("count", rowsAffected).Debug("💾 Deleted expired SNA enrollments")
	}

	return rowsAffected, nil
}

// GetActiveConnections retrieves all active SNA connections
func (ver *SNAEnrollmentRepository) GetActiveConnections() ([]models.SNAActiveConnection, error) {
	if ver.db == nil {
		return nil, fmt.Errorf("database not available")
	}

	var connections []models.SNAActiveConnection
	if err := ver.db.Where("connection_status = ?", models.ConnectionStatusConnected).
		Order("connected_at DESC").Find(&connections).Error; err != nil {
		return nil, fmt.Errorf("failed to get active connections: %w", err)
	}

	return connections, nil
}

// UpdateLastSeen updates the last seen timestamp for a SNA connection
func (ver *SNAEnrollmentRepository) UpdateLastSeen(enrollmentID string) error {
	if ver.db == nil {
		return fmt.Errorf("database not available")
	}

	updates := map[string]interface{}{
		"last_seen_at": time.Now(),
		"updated_at":   time.Now(),
	}

	if err := ver.db.Model(&models.SNAActiveConnection{}).Where("enrollment_id = ?", enrollmentID).Updates(updates).Error; err != nil {
		return fmt.Errorf("failed to update last seen: %w", err)
	}

	return nil
}

// UpdateConnectionStatus updates the connection status for a SNA
func (ver *SNAEnrollmentRepository) UpdateConnectionStatus(enrollmentID string, status string) error {
	if ver.db == nil {
		return fmt.Errorf("database not available")
	}

	updates := map[string]interface{}{
		"connection_status": status,
		"updated_at":        time.Now(),
	}

	if err := ver.db.Model(&models.SNAActiveConnection{}).Where("enrollment_id = ?", enrollmentID).Updates(updates).Error; err != nil {
		return fmt.Errorf("failed to update connection status: %w", err)
	}

	return nil
}
