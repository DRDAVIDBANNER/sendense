// Package storage provides immutable repository wrapper for ransomware protection
// Following project rules: modular design, enterprise security, no simulations
package storage

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"

	log "github.com/sirupsen/logrus"
)

// ImmutableRepository wraps any Repository implementation to add immutability protection.
// Uses Linux chattr +i for filesystem-level ransomware protection.
// Enterprise feature: Prevents backup deletion/modification during retention period.
type ImmutableRepository struct {
	Repository                       // Embed underlying repository
	config     *ImmutableConfig      // Uses existing ImmutableConfig from repository_config.go
	chattrConfig *LinuxChattrConfig   // Linux-specific chattr configuration
	backupRepo BackupChainRepository
}

// LinuxChattrConfig defines Linux chattr-specific immutability settings.
// Used with ImmutableConfig when Type = ImmutableTypeLinuxChattr.
type LinuxChattrConfig struct {
	GracePeriodDays     int  `json:"grace_period_days"`      // Days before applying chattr +i
	ApplyToFullBackups  bool `json:"apply_to_full_backups"`  // Apply immutability to full backups
	ApplyToIncrementals bool `json:"apply_to_incrementals"`  // Apply immutability to incremental backups
}

// NewImmutableRepository creates an immutable repository wrapper.
func NewImmutableRepository(underlying Repository, config *ImmutableConfig, backupRepo BackupChainRepository) *ImmutableRepository {
	if config == nil {
		config = &ImmutableConfig{
			Type:             ImmutableTypeLinuxChattr,
			MinRetentionDays: 7,
			Config:           nil,
		}
	}

	// Parse Linux chattr-specific config
	var chattrConfig *LinuxChattrConfig
	if config.Type == ImmutableTypeLinuxChattr && config.Config != nil {
		// Try to parse chattr config from Config interface
		if configMap, ok := config.Config.(map[string]interface{}); ok {
			chattrConfig = &LinuxChattrConfig{
				GracePeriodDays:     getIntFromMap(configMap, "grace_period_days", 1),
				ApplyToFullBackups:  getBoolFromMap(configMap, "apply_to_full_backups", true),
				ApplyToIncrementals: getBoolFromMap(configMap, "apply_to_incrementals", false),
			}
		}
	}

	// Default chattr config if not provided
	if chattrConfig == nil {
		chattrConfig = &LinuxChattrConfig{
			GracePeriodDays:     1,
			ApplyToFullBackups:  true,
			ApplyToIncrementals: false,
		}
	}

	return &ImmutableRepository{
		Repository:   underlying,
		config:       config,
		chattrConfig: chattrConfig,
		backupRepo:   backupRepo,
	}
}

// Helper functions for parsing config map
func getIntFromMap(m map[string]interface{}, key string, defaultVal int) int {
	if val, ok := m[key]; ok {
		if intVal, ok := val.(float64); ok { // JSON numbers are float64
			return int(intVal)
		}
		if intVal, ok := val.(int); ok {
			return intVal
		}
	}
	return defaultVal
}

func getBoolFromMap(m map[string]interface{}, key string, defaultVal bool) bool {
	if val, ok := m[key]; ok {
		if boolVal, ok := val.(bool); ok {
			return boolVal
		}
	}
	return defaultVal
}

// CreateBackup creates a backup and schedules immutability application.
func (ir *ImmutableRepository) CreateBackup(ctx context.Context, req BackupRequest) (*Backup, error) {
	// Create backup via underlying repository
	backup, err := ir.Repository.CreateBackup(ctx, req)
	if err != nil {
		return nil, err
	}

	// If immutability enabled and no grace period, apply immediately
	if ir.config.Type == ImmutableTypeLinuxChattr && ir.shouldApplyImmutability(backup) {
		if ir.chattrConfig.GracePeriodDays == 0 {
			if err := ir.applyImmutability(backup.FilePath); err != nil {
				log.WithError(err).WithField("backup_id", backup.ID).Warn("Failed to apply immutability immediately")
			} else {
				log.WithField("backup_id", backup.ID).Info("Immutability applied immediately (no grace period)")
			}
		} else {
			log.WithFields(log.Fields{
				"backup_id":         backup.ID,
				"grace_period_days": ir.chattrConfig.GracePeriodDays,
			}).Info("Backup created, immutability will be applied after grace period")
		}
	}

	return backup, nil
}

// DeleteBackup enforces retention policy before allowing deletion.
func (ir *ImmutableRepository) DeleteBackup(ctx context.Context, backupID string) error {
	// Get backup metadata
	backup, err := ir.Repository.GetBackup(ctx, backupID)
	if err != nil {
		return err
	}

	// Check if immutability is enabled for this repository
	if ir.config.Type != ImmutableTypeLinuxChattr {
		// No immutability protection, allow deletion
		return ir.Repository.DeleteBackup(ctx, backupID)
	}

	// Check minimum retention period
	if err := ir.checkRetentionPeriod(backup); err != nil {
		return err
	}

	// Check if file is immutable
	isImmutable, err := ir.isFileImmutable(backup.FilePath)
	if err != nil {
		return fmt.Errorf("failed to check immutability status: %w", err)
	}

	// If immutable, remove immutability before deletion
	if isImmutable {
		log.WithField("backup_id", backupID).Info("Removing immutability before deletion")
		if err := ir.removeImmutability(backup.FilePath); err != nil {
			return fmt.Errorf("failed to remove immutability: %w", err)
		}
	}

	// Delete backup via underlying repository
	if err := ir.Repository.DeleteBackup(ctx, backupID); err != nil {
		// If deletion fails, re-apply immutability
		if isImmutable {
			if reapplyErr := ir.applyImmutability(backup.FilePath); reapplyErr != nil {
				log.WithError(reapplyErr).WithField("backup_id", backupID).Error("Failed to re-apply immutability after deletion failure")
			}
		}
		return err
	}

	log.WithField("backup_id", backupID).Info("Backup deleted successfully (immutability removed)")
	return nil
}

// shouldApplyImmutability determines if immutability should be applied to a backup.
func (ir *ImmutableRepository) shouldApplyImmutability(backup *Backup) bool {
	if ir.config.Type != ImmutableTypeLinuxChattr {
		return false
	}

	// Check backup type configuration
	if backup.BackupType == BackupTypeFull && ir.chattrConfig.ApplyToFullBackups {
		return true
	}
	if backup.BackupType == BackupTypeIncremental && ir.chattrConfig.ApplyToIncrementals {
		return true
	}

	return false
}

// checkRetentionPeriod validates backup age against minimum retention.
func (ir *ImmutableRepository) checkRetentionPeriod(backup *Backup) error {
	if ir.config.MinRetentionDays <= 0 {
		return nil // No minimum retention
	}

	backupAge := time.Since(backup.CreatedAt)
	minRetention := time.Duration(ir.config.MinRetentionDays) * 24 * time.Hour

	if backupAge < minRetention {
		return fmt.Errorf("cannot delete backup: minimum retention period not met (backup age: %v, minimum: %v)",
			backupAge.Round(time.Hour), minRetention)
	}

	return nil
}

// applyImmutability applies Linux chattr +i to a file.
// Requires CAP_LINUX_IMMUTABLE capability or root.
func (ir *ImmutableRepository) applyImmutability(filePath string) error {
	// Check if file exists
	if _, err := os.Stat(filePath); err != nil {
		return fmt.Errorf("file not found: %w", err)
	}

	// Execute chattr +i
	cmd := exec.Command("chattr", "+i", filePath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("chattr +i failed: %w (output: %s)", err, string(output))
	}

	log.WithField("file", filePath).Info("Immutability applied successfully")
	return nil
}

// removeImmutability removes Linux chattr +i from a file.
// Requires CAP_LINUX_IMMUTABLE capability or root.
func (ir *ImmutableRepository) removeImmutability(filePath string) error {
	// Check if file exists
	if _, err := os.Stat(filePath); err != nil {
		return fmt.Errorf("file not found: %w", err)
	}

	// Execute chattr -i
	cmd := exec.Command("chattr", "-i", filePath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("chattr -i failed: %w (output: %s)", err, string(output))
	}

	log.WithField("file", filePath).Info("Immutability removed successfully")
	return nil
}

// isFileImmutable checks if a file has the immutable attribute set.
func (ir *ImmutableRepository) isFileImmutable(filePath string) (bool, error) {
	// Execute lsattr to check immutable flag
	cmd := exec.Command("lsattr", filePath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false, fmt.Errorf("lsattr failed: %w (output: %s)", err, string(output))
	}

	// Check if output contains 'i' flag (immutable)
	// lsattr output format: "----i--------e------- /path/to/file"
	outputStr := string(output)
	if len(outputStr) > 0 {
		// The immutable flag is typically the 5th character
		flags := outputStr[:20] // First 20 chars contain flags
		for _, char := range flags {
			if char == 'i' {
				return true, nil
			}
		}
	}

	return false, nil
}

// ApplyImmutabilityToBackup manually applies immutability to a specific backup.
// Used by grace period worker or admin operations.
func (ir *ImmutableRepository) ApplyImmutabilityToBackup(ctx context.Context, backupID string) error {
	backup, err := ir.Repository.GetBackup(ctx, backupID)
	if err != nil {
		return err
	}

	if !ir.shouldApplyImmutability(backup) {
		return fmt.Errorf("immutability not configured for this backup type")
	}

	// Check if already immutable
	isImmutable, err := ir.isFileImmutable(backup.FilePath)
	if err != nil {
		return err
	}

	if isImmutable {
		log.WithField("backup_id", backupID).Info("Backup already immutable")
		return nil
	}

	// Apply immutability
	if err := ir.applyImmutability(backup.FilePath); err != nil {
		return err
	}

	log.WithField("backup_id", backupID).Info("Immutability applied to backup")
	return nil
}

// GetImmutabilityStatus returns the immutability status of a backup.
func (ir *ImmutableRepository) GetImmutabilityStatus(ctx context.Context, backupID string) (*ImmutabilityStatus, error) {
	backup, err := ir.Repository.GetBackup(ctx, backupID)
	if err != nil {
		return nil, err
	}

	isImmutable, err := ir.isFileImmutable(backup.FilePath)
	if err != nil {
		return nil, err
	}

	status := &ImmutabilityStatus{
		BackupID:         backupID,
		IsImmutable:      isImmutable,
		BackupCreatedAt:  backup.CreatedAt,
		MinRetentionDays: ir.config.MinRetentionDays,
		GracePeriodDays:  ir.chattrConfig.GracePeriodDays,
	}

	// Calculate grace period expiry
	if ir.chattrConfig.GracePeriodDays > 0 {
		graceExpiry := backup.CreatedAt.Add(time.Duration(ir.chattrConfig.GracePeriodDays) * 24 * time.Hour)
		status.GracePeriodExpiresAt = &graceExpiry
		status.GracePeriodActive = time.Now().Before(graceExpiry)
	}

	// Calculate retention expiry
	if ir.config.MinRetentionDays > 0 {
		retentionExpiry := backup.CreatedAt.Add(time.Duration(ir.config.MinRetentionDays) * 24 * time.Hour)
		status.RetentionExpiresAt = &retentionExpiry
		status.CanDelete = time.Now().After(retentionExpiry)
	} else {
		status.CanDelete = true
	}

	return status, nil
}

// ImmutabilityStatus represents the immutability state of a backup.
type ImmutabilityStatus struct {
	BackupID              string     `json:"backup_id"`
	IsImmutable           bool       `json:"is_immutable"`
	BackupCreatedAt       time.Time  `json:"backup_created_at"`
	MinRetentionDays      int        `json:"min_retention_days"`
	GracePeriodDays       int        `json:"grace_period_days"`
	GracePeriodExpiresAt  *time.Time `json:"grace_period_expires_at,omitempty"`
	GracePeriodActive     bool       `json:"grace_period_active"`
	RetentionExpiresAt    *time.Time `json:"retention_expires_at,omitempty"`
	CanDelete             bool       `json:"can_delete"`
}

// ProcessGracePeriodBackups processes backups whose grace period has expired.
// Should be called by a background worker.
func (ir *ImmutableRepository) ProcessGracePeriodBackups(ctx context.Context, vmContextID string) error {
	if ir.config.Type != ImmutableTypeLinuxChattr || ir.chattrConfig.GracePeriodDays == 0 {
		return nil // No grace period processing needed
	}

	// Get all backups for VM
	backups, err := ir.Repository.ListBackups(ctx, vmContextID)
	if err != nil {
		return err
	}

	graceExpiry := time.Now().Add(-time.Duration(ir.chattrConfig.GracePeriodDays) * 24 * time.Hour)
	processedCount := 0

	for _, backup := range backups {
		// Check if backup is old enough (grace period expired)
		if backup.CreatedAt.After(graceExpiry) {
			continue // Still in grace period
		}

		// Check if this backup type should be immutable
		if !ir.shouldApplyImmutability(backup) {
			continue
		}

		// Check if already immutable
		isImmutable, err := ir.isFileImmutable(backup.FilePath)
		if err != nil {
			log.WithError(err).WithField("backup_id", backup.ID).Warn("Failed to check immutability status")
			continue
		}

		if isImmutable {
			continue // Already immutable
		}

		// Apply immutability
		if err := ir.applyImmutability(backup.FilePath); err != nil {
			log.WithError(err).WithField("backup_id", backup.ID).Error("Failed to apply immutability after grace period")
			continue
		}

		log.WithFields(log.Fields{
			"backup_id":   backup.ID,
			"backup_age":  time.Since(backup.CreatedAt).Round(time.Hour),
		}).Info("Immutability applied after grace period")
		processedCount++
	}

	if processedCount > 0 {
		log.WithFields(log.Fields{
			"vm_context_id":    vmContextID,
			"backups_processed": processedCount,
		}).Info("Grace period processing complete")
	}

	return nil
}
