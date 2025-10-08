// Package failover provides helper utilities for enhanced test failover cleanup
package failover

import (
	"context"
	"fmt"
	"time"

	"github.com/vexxhost/migratekit-sha/database"
	"github.com/vexxhost/migratekit-sha/joblog"
	"github.com/vexxhost/migratekit-sha/ossea"
)

// CleanupHelpers provides utility functions for enhanced test failover cleanup
type CleanupHelpers struct {
	db          database.Connection
	osseaClient *ossea.Client
	jobTracker  *joblog.Tracker
}

// NewCleanupHelpers creates a new cleanup helpers instance
func NewCleanupHelpers(db database.Connection, osseaClient *ossea.Client, jobTracker *joblog.Tracker) *CleanupHelpers {
	return &CleanupHelpers{
		db:          db,
		osseaClient: osseaClient,
		jobTracker:  jobTracker,
	}
}

// InitializeOSSEAClient initializes the CloudStack client from database configuration
func (ch *CleanupHelpers) InitializeOSSEAClient(ctx context.Context) (*ossea.Client, error) {
	logger := ch.jobTracker.Logger(ctx)
	logger.Info("🔧 Initializing OSSEA client from database configuration")

	var osseaConfig database.OSSEAConfig
	err := ch.db.GetGormDB().Where("is_active = true").First(&osseaConfig).Error
	if err != nil {
		logger.Error("Failed to get active OSSEA configuration", "error", err)
		return nil, fmt.Errorf("no active OSSEA configuration found: %w", err)
	}

	client := ossea.NewClient(
		osseaConfig.APIURL,
		osseaConfig.APIKey,
		osseaConfig.SecretKey,
		osseaConfig.Domain,
		osseaConfig.Zone,
	)

	logger.Info("✅ OSSEA client initialized successfully",
		"api_url", osseaConfig.APIURL,
		"zone", osseaConfig.Zone,
		"domain", osseaConfig.Domain,
	)

	return client, nil
}

// GetFailoverJobDetails retrieves failover job information including snapshot ID and test VM ID
func (ch *CleanupHelpers) GetFailoverJobDetails(ctx context.Context, vmNameOrID string) (string, string, string, error) {
	logger := ch.jobTracker.Logger(ctx)
	logger.Info("🔍 Retrieving failover job details", "vm_name_or_id", vmNameOrID)

	// Query failover_jobs table to find the job for this VM
	var failoverJob database.FailoverJob
	err := ch.db.GetGormDB().Where("job_id = ? OR vm_id = ? OR source_vm_name = ?", vmNameOrID, vmNameOrID, vmNameOrID).
		Order("created_at DESC").
		First(&failoverJob).Error

	if err != nil {
		logger.Error("Failed to find failover job", "error", err, "vm_name_or_id", vmNameOrID)
		return "", "", "", fmt.Errorf("no failover job found for VM %s: %w", vmNameOrID, err)
	}

	logger.Info("Retrieved failover job details",
		"failover_job_id", failoverJob.ID,
		"replication_job_id", failoverJob.ReplicationJobID,
		"job_type", failoverJob.JobType,
		"status", failoverJob.Status,
		"destination_vm_id", failoverJob.DestinationVMID,
	)

	// Extract snapshot ID from ossea_snapshot_id field (CloudStack volume snapshot)
	snapshotID := ""
	if failoverJob.OSSEASnapshotID != "" {
		snapshotID = failoverJob.OSSEASnapshotID
	}

	// Get test VM ID from destination_vm_id field
	testVMID := ""
	if failoverJob.DestinationVMID != "" {
		testVMID = failoverJob.DestinationVMID
	}

	if testVMID == "" {
		logger.Error("No destination VM ID found in failover job", "failover_job_id", failoverJob.ID)
		return "", "", "", fmt.Errorf("no destination VM ID found in failover job %s", failoverJob.ID)
	}

	logger.Info("✅ Failover job details retrieved successfully",
		"failover_job_id", failoverJob.ID,
		"snapshot_id", snapshotID,
		"test_vm_id", testVMID,
	)

	return failoverJob.JobID, snapshotID, testVMID, nil
}

// UpdateFailoverJobStatus updates the status of a failover job
func (ch *CleanupHelpers) UpdateFailoverJobStatus(ctx context.Context, failoverJobID, status string) error {
	logger := ch.jobTracker.Logger(ctx)
	logger.Info("📝 Updating failover job status", "failover_job_id", failoverJobID, "new_status", status)

	// Update the failover job status in the database
	result := ch.db.GetGormDB().Model(&database.FailoverJob{}).
		Where("job_id = ?", failoverJobID).
		Updates(map[string]interface{}{
			"status":     status,
			"updated_at": time.Now(),
		})

	if result.Error != nil {
		logger.Error("Failed to update failover job status", "error", result.Error, "failover_job_id", failoverJobID)
		return fmt.Errorf("failed to update failover job status: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		logger.Warn("No rows affected when updating failover job status", "failover_job_id", failoverJobID)
		return fmt.Errorf("no failover job found with ID %s", failoverJobID)
	}

	logger.Info("✅ Failover job status updated successfully",
		"failover_job_id", failoverJobID,
		"new_status", status,
		"rows_affected", result.RowsAffected,
	)

	return nil
}

// GetOMAVMID retrieves the SHA VM ID for volume reattachment
func (ch *CleanupHelpers) GetOMAVMID(ctx context.Context, vmNameOrID string) (string, error) {
	logger := ch.jobTracker.Logger(ctx)
	logger.Info("🔍 Retrieving SHA VM ID from database for volume reattachment", "vm_name_or_id", vmNameOrID)

	var shaVMID string
	err := ch.db.GetGormDB().Raw("SELECT oma_vm_id FROM ossea_configs WHERE is_active = 1 LIMIT 1").Scan(&shaVMID).Error
	if err != nil {
		logger.Error("Failed to query SHA VM ID from database", "error", err)
		return "", fmt.Errorf("failed to query SHA VM ID from ossea_configs: %w", err)
	}
	
	if shaVMID == "" {
		logger.Error("SHA VM ID is empty in database")
		return "", fmt.Errorf("SHA VM ID is empty in ossea_configs table")
	}
	
	logger.Info("Successfully retrieved SHA VM ID from database", "oma_vm_id", shaVMID)
	return shaVMID, nil
}
