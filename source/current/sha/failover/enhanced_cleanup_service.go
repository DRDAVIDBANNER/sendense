// Package failover provides enhanced test failover cleanup orchestration with modular architecture
package failover

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/vexxhost/migratekit-sha/database"
	"github.com/vexxhost/migratekit-sha/joblog"
	"github.com/vexxhost/migratekit-sha/services"
)

// RollbackOptions defines configurable behaviors for failover rollback operations
type RollbackOptions struct {
	PowerOnSourceVM bool   `json:"power_on_source_vm"` // User configurable via GUI
	ForceCleanup    bool   `json:"force_cleanup"`      // Force cleanup even on errors
	FailoverType    string `json:"failover_type"`      // "test" or "live"
}

// SNAClient interface for power management operations with vCenter credentials
type SNAClient interface {
	PowerOnSourceVM(ctx context.Context, vmwareVMID, vcenter, username, password string) error
	PowerOffSourceVM(ctx context.Context, vmwareVMID, vcenter, username, password string) error
	GetVMPowerState(ctx context.Context, vmwareVMID, vcenter, username, password string) (string, error)
}

// RollbackDecision represents a user decision point during rollback
type RollbackDecision struct {
	DecisionID   string   `json:"decision_id"`
	Question     string   `json:"question"`
	Options      []string `json:"options"`
	DefaultValue string   `json:"default_value"`
	Required     bool     `json:"required"`
}

// EnhancedCleanupService orchestrates enhanced test failover cleanup with modular components
type EnhancedCleanupService struct {
	// Core dependencies
	jobTracker    *joblog.Tracker
	db            database.Connection
	vmContextRepo *database.VMReplicationContextRepository

	// Modular components
	vmCleanup       *VMCleanupOperations
	volumeCleanup   *VolumeCleanupOperations
	snapshotCleanup *SnapshotCleanupOperations
	helpers         *CleanupHelpers

	// 🆕 NEW: Multi-volume snapshot service for complete VM cleanup
	multiVolumeSnapshotService *MultiVolumeSnapshotService

	// SNA integration for live failover rollback
	snaClient SNAClient
}

// NewEnhancedCleanupService creates an enhanced cleanup service with modular architecture
func NewEnhancedCleanupService(db database.Connection, jobTracker *joblog.Tracker, snaClient SNAClient) *EnhancedCleanupService {
	// 🔧 CREDENTIAL FIX: Create helpers first (no longer needs pre-initialized client)
	helpers := NewCleanupHelpers(db, nil, jobTracker)
	
	// Create modular components with helpers for dynamic credential initialization
	vmCleanup := NewVMCleanupOperations(nil, jobTracker, helpers)
	volumeCleanup := NewVolumeCleanupOperations(jobTracker, db)
	snapshotCleanup := NewSnapshotCleanupOperations(nil, jobTracker, helpers)

	// 🆕 NEW: Initialize multi-volume snapshot service for complete cleanup
	multiVolumeSnapshotService := NewMultiVolumeSnapshotService(&db, nil, jobTracker)

	// Initialize VM context repository for status updates
	vmContextRepo := database.NewVMReplicationContextRepository(db)

	return &EnhancedCleanupService{
		jobTracker:                 jobTracker,
		db:                         db,
		vmContextRepo:              vmContextRepo,
		vmCleanup:                  vmCleanup,
		volumeCleanup:              volumeCleanup,
		snapshotCleanup:            snapshotCleanup,
		helpers:                    helpers,
		multiVolumeSnapshotService: multiVolumeSnapshotService, // 🆕 NEW: Multi-volume snapshot cleanup
		snaClient:                  snaClient,
	}
}

// ExecuteTestFailoverCleanupSteps executes individual cleanup steps under an existing job
func (ecs *EnhancedCleanupService) ExecuteTestFailoverCleanupSteps(ctx context.Context, jobID, contextID, vmNameOrID string) error {
	// Get logger with job context
	logger := ecs.jobTracker.Logger(ctx)
	logger.Info("Starting enhanced test failover cleanup steps", "vm_name_or_id", vmNameOrID, "job_id", jobID)

	// 🔧 CREDENTIAL FIX: No need for Phase 1 OSSEA client initialization
	// Components now initialize fresh client per operation via helpers.InitializeOSSEAClient()

	// PHASE 2: Retrieve failover job and snapshot information
	var failoverJobID, snapshotID, testVMID string
	if err := ecs.jobTracker.RunStep(ctx, jobID, "failover-job-retrieval", func(ctx context.Context) error {
		var err error
		failoverJobID, snapshotID, testVMID, err = ecs.helpers.GetFailoverJobDetails(ctx, vmNameOrID)
		return err
	}); err != nil {
		return fmt.Errorf("failover job retrieval failed: %w", err)
	}

	// PHASE 3: Stop test VM
	if err := ecs.jobTracker.RunStep(ctx, jobID, "test-vm-shutdown", func(ctx context.Context) error {
		return ecs.vmCleanup.StopTestVM(ctx, testVMID)
	}); err != nil {
		return fmt.Errorf("test VM shutdown failed: %w", err)
	}

	// PHASE 4: Detach volumes from test VM
	var volumeIDs []string
	if err := ecs.jobTracker.RunStep(ctx, jobID, "volume-detachment", func(ctx context.Context) error {
		var err error
		volumeIDs, err = ecs.volumeCleanup.DetachVolumesFromTestVM(ctx, testVMID)
		return err
	}); err != nil {
		return fmt.Errorf("volume detachment failed: %w", err)
	}

	// PHASE 5: Multi-Volume Snapshot Cleanup (NEW: Complete VM protection cleanup)
	// 🆕 ENHANCED: Clean up ALL volume snapshots, not just single snapshot
	if err := ecs.jobTracker.RunStep(ctx, jobID, "multi-volume-snapshot-cleanup", func(ctx context.Context) error {
		logger := ecs.jobTracker.Logger(ctx)
		logger.Info("🧹 Executing multi-volume snapshot cleanup",
			"context_id", contextID)

		err := ecs.multiVolumeSnapshotService.CleanupAllVolumeSnapshots(ctx, contextID)
		if err != nil {
			logger.Error("Multi-volume snapshot cleanup failed", "error", err)
			return fmt.Errorf("multi-volume snapshot cleanup failed: %w", err)
		}

		logger.Info("✅ Multi-volume snapshot cleanup completed successfully")
		return nil
	}); err != nil {
		return fmt.Errorf("multi-volume snapshot cleanup failed: %w", err)
	}

	// PHASE 5.1: Switch volumes back to 'oma' mode after cleanup
	// 🆕 CRITICAL: Restore volumes to normal operation mode
	if err := ecs.jobTracker.RunStep(ctx, jobID, "volume-mode-switch-oma", func(ctx context.Context) error {
		logger := ecs.jobTracker.Logger(ctx)
		logger.Info("🔄 Switching volumes back to SHA mode",
			"context_id", contextID)

		// Update all device mappings for this VM context back to 'oma' mode
		// Note: device_mappings table is managed by Volume Daemon, using raw SQL query
		result := ecs.db.GetGormDB().Exec(
			"UPDATE device_mappings SET operation_mode = ? WHERE vm_context_id = ?",
			"oma", contextID)

		if result.Error != nil {
			return fmt.Errorf("failed to switch volumes back to oma mode: %w", result.Error)
		}

		logger.Info("✅ Volumes switched back to SHA mode successfully",
			"context_id", contextID,
			"volumes_updated", result.RowsAffected)

		return nil
	}); err != nil {
		return fmt.Errorf("volume mode switch back to oma failed: %w", err)
	}

	// PHASE 5.5: Legacy CloudStack Volume Snapshot Rollback (for backward compatibility)
	// ⚠️ LEGACY: This handles old single-snapshot jobs, new jobs use multi-volume cleanup above
	if snapshotID != "" {
		logger.Info("🔄 Processing legacy single snapshot cleanup",
			"snapshot_id", snapshotID)

		if err := ecs.jobTracker.RunStep(ctx, jobID, "legacy-cloudstack-snapshot-rollback", func(ctx context.Context) error {
			return ecs.snapshotCleanup.RollbackCloudStackVolumeSnapshot(ctx, snapshotID)
		}); err != nil {
			return fmt.Errorf("legacy cloudstack snapshot rollback failed: %w", err)
		}

		// PHASE 6: Delete legacy CloudStack volume snapshot
		if err := ecs.jobTracker.RunStep(ctx, jobID, "legacy-cloudstack-snapshot-deletion", func(ctx context.Context) error {
			return ecs.snapshotCleanup.DeleteCloudStackVolumeSnapshot(ctx, snapshotID)
		}); err != nil {
			return fmt.Errorf("legacy cloudstack snapshot deletion failed: %w", err)
		}
	} else {
		logger.Info("ℹ️ No legacy snapshot found - multi-volume cleanup handled all snapshots")
	}

	// PHASE 7: Reattach volumes to SHA
	if err := ecs.jobTracker.RunStep(ctx, jobID, "volume-reattachment-to-oma", func(ctx context.Context) error {
		return ecs.volumeCleanup.ReattachVolumesToOMA(ctx, volumeIDs, vmNameOrID)
	}); err != nil {
		return fmt.Errorf("volume reattachment to SHA failed: %w", err)
	}

	// PHASE 8: Delete test VM
	if err := ecs.jobTracker.RunStep(ctx, jobID, "test-vm-deletion", func(ctx context.Context) error {
		return ecs.vmCleanup.DeleteTestVM(ctx, testVMID)
	}); err != nil {
		return fmt.Errorf("test VM deletion failed: %w", err)
	}

	// PHASE 9: Update failover job status
	if err := ecs.jobTracker.RunStep(ctx, jobID, "failover-job-status-update", func(ctx context.Context) error {
		return ecs.helpers.UpdateFailoverJobStatus(ctx, failoverJobID, "completed")
	}); err != nil {
		return fmt.Errorf("failover job status update failed: %w", err)
	}

	// PHASE 10: Update VM context status
	if err := ecs.jobTracker.RunStep(ctx, jobID, "vm-context-status-update", func(ctx context.Context) error {
		return ecs.vmContextRepo.UpdateVMContextStatus(contextID, "ready_for_failover")
	}); err != nil {
		return fmt.Errorf("VM context status update failed: %w", err)
	}

	logger.Info("✅ Enhanced test failover cleanup completed successfully",
		"context_id", contextID,
		"vm_name_or_id", vmNameOrID,
		"test_vm_id", testVMID,
		"volumes_processed", len(volumeIDs))

	return nil
}

// ExecuteTestFailoverCleanupWithTracking orchestrates the complete enhanced test failover cleanup process
func (ecs *EnhancedCleanupService) ExecuteTestFailoverCleanupWithTracking(ctx context.Context, contextID, vmNameOrID string, existingJobID string) error {
	// Use existing job ID instead of creating a new job
	jobID := existingJobID

	// Get logger with job context
	logger := ecs.jobTracker.Logger(ctx)
	logger.Info("Starting enhanced test failover cleanup with modular architecture", "vm_name_or_id", vmNameOrID)

	// 🔧 CREDENTIAL FIX: No need for Phase 1 OSSEA client initialization
	// Components now initialize fresh client per operation via helpers.InitializeOSSEAClient()

	// PHASE 2: Retrieve failover job and snapshot information
	var failoverJobID, snapshotID, testVMID string
	if err := ecs.jobTracker.RunStep(ctx, jobID, "failover-job-retrieval", func(ctx context.Context) error {
		var err error
		failoverJobID, snapshotID, testVMID, err = ecs.helpers.GetFailoverJobDetails(ctx, vmNameOrID)
		return err
	}); err != nil {
		return fmt.Errorf("failover job retrieval failed: %w", err)
	}

	// PHASE 3: Stop test VM
	if err := ecs.jobTracker.RunStep(ctx, jobID, "test-vm-shutdown", func(ctx context.Context) error {
		return ecs.vmCleanup.StopTestVM(ctx, testVMID)
	}); err != nil {
		return fmt.Errorf("test VM shutdown failed: %w", err)
	}

	// PHASE 4: Detach volumes from test VM
	var volumeIDs []string
	if err := ecs.jobTracker.RunStep(ctx, jobID, "volume-detachment", func(ctx context.Context) error {
		var err error
		volumeIDs, err = ecs.volumeCleanup.DetachVolumesFromTestVM(ctx, testVMID)
		return err
	}); err != nil {
		return fmt.Errorf("volume detachment failed: %w", err)
	}

	// PHASE 5: Multi-Volume Snapshot Cleanup (NEW: Complete VM protection cleanup)
	// 🆕 ENHANCED: Clean up ALL volume snapshots, not just single snapshot
	if err := ecs.jobTracker.RunStep(ctx, jobID, "multi-volume-snapshot-cleanup", func(ctx context.Context) error {
		logger := ecs.jobTracker.Logger(ctx)
		logger.Info("🧹 Executing multi-volume snapshot cleanup",
			"context_id", contextID)

		err := ecs.multiVolumeSnapshotService.CleanupAllVolumeSnapshots(ctx, contextID)
		if err != nil {
			logger.Error("Multi-volume snapshot cleanup failed", "error", err)
			return fmt.Errorf("multi-volume snapshot cleanup failed: %w", err)
		}

		logger.Info("✅ Multi-volume snapshot cleanup completed successfully")
		return nil
	}); err != nil {
		return fmt.Errorf("multi-volume snapshot cleanup failed: %w", err)
	}

	// PHASE 5.1: Switch volumes back to 'oma' mode after cleanup
	// 🆕 CRITICAL: Restore volumes to normal operation mode
	if err := ecs.jobTracker.RunStep(ctx, jobID, "volume-mode-switch-oma", func(ctx context.Context) error {
		logger := ecs.jobTracker.Logger(ctx)
		logger.Info("🔄 Switching volumes back to SHA mode",
			"context_id", contextID)

		// Update all device mappings for this VM context back to 'oma' mode
		// Note: device_mappings table is managed by Volume Daemon, using raw SQL query
		result := ecs.db.GetGormDB().Exec(
			"UPDATE device_mappings SET operation_mode = ? WHERE vm_context_id = ?",
			"oma", contextID)

		if result.Error != nil {
			return fmt.Errorf("failed to switch volumes back to oma mode: %w", result.Error)
		}

		logger.Info("✅ Volumes switched back to SHA mode successfully",
			"context_id", contextID,
			"volumes_updated", result.RowsAffected)

		return nil
	}); err != nil {
		return fmt.Errorf("volume mode switch back to oma failed: %w", err)
	}

	// PHASE 5.5: Legacy CloudStack Volume Snapshot Rollback (for backward compatibility)
	// ⚠️ LEGACY: This handles old single-snapshot jobs, new jobs use multi-volume cleanup above
	if snapshotID != "" {
		logger.Info("🔄 Processing legacy single snapshot cleanup",
			"snapshot_id", snapshotID)

		if err := ecs.jobTracker.RunStep(ctx, jobID, "legacy-cloudstack-snapshot-rollback", func(ctx context.Context) error {
			return ecs.snapshotCleanup.RollbackCloudStackVolumeSnapshot(ctx, snapshotID)
		}); err != nil {
			return fmt.Errorf("legacy CloudStack snapshot rollback failed: %w", err)
		}

		// PHASE 5.5: Delete the legacy snapshot after successful rollback
		if err := ecs.jobTracker.RunStep(ctx, jobID, "legacy-cloudstack-snapshot-deletion", func(ctx context.Context) error {
			return ecs.snapshotCleanup.DeleteCloudStackVolumeSnapshot(ctx, snapshotID)
		}); err != nil {
			return fmt.Errorf("legacy CloudStack snapshot deletion failed: %w", err)
		}
	} else {
		logger.Info("ℹ️ No legacy snapshot found - multi-volume cleanup handled all snapshots")
	}

	// PHASE 6: Reattach volumes to SHA
	if err := ecs.jobTracker.RunStep(ctx, jobID, "volume-reattachment-to-oma", func(ctx context.Context) error {
		return ecs.volumeCleanup.ReattachVolumesToOMA(ctx, volumeIDs, vmNameOrID)
	}); err != nil {
		return fmt.Errorf("volume reattachment to SHA failed: %w", err)
	}

	// PHASE 7: Delete test VM
	if err := ecs.jobTracker.RunStep(ctx, jobID, "test-vm-deletion", func(ctx context.Context) error {
		return ecs.vmCleanup.DeleteTestVM(ctx, testVMID)
	}); err != nil {
		return fmt.Errorf("test VM deletion failed: %w", err)
	}

	// PHASE 8: Update failover job status
	if err := ecs.jobTracker.RunStep(ctx, jobID, "failover-job-status-update", func(ctx context.Context) error {
		return ecs.helpers.UpdateFailoverJobStatus(ctx, failoverJobID, "cleanup")
	}); err != nil {
		return fmt.Errorf("failover job status update failed: %w", err)
	}

	// PHASE 9: Update VM context status back to ready_for_failover
	if contextID != "" {
		if err := ecs.jobTracker.RunStep(ctx, jobID, "vm-context-status-update", func(ctx context.Context) error {
			return ecs.vmContextRepo.UpdateVMContextStatus(contextID, "ready_for_failover")
		}); err != nil {
			// Log error but don't fail cleanup - same pattern as failover system
			logger := ecs.jobTracker.Logger(ctx)
			logger.Error("Failed to update VM context status to ready_for_failover", "error", err, "context_id", contextID)
		}
	}

	logger.Info("✅ Enhanced test failover cleanup completed successfully",
		"context_id", contextID,
		"vm_name_or_id", vmNameOrID,
		"test_vm_id", testVMID,
		"failover_job_id", failoverJobID,
		"snapshot_rollback", snapshotID != "",
		"volumes_processed", len(volumeIDs),
	)

	return nil
}

// ExecuteUnifiedFailoverRollback orchestrates rollback for both test and live failover with optional behaviors
func (ecs *EnhancedCleanupService) ExecuteUnifiedFailoverRollback(ctx context.Context, contextID, vmNameOrID, vmwareVMID string, options *RollbackOptions, externalJobID string) error {
	// Create JobLog job with external job ID correlation (copying unified failover pattern)
	ctx, jobID, err := ecs.jobTracker.StartJob(ctx, joblog.JobStart{
		JobType:       "cleanup",
		Operation:     fmt.Sprintf("enhanced-%s-failover-rollback", options.FailoverType),
		Owner:         stringPtr("system"),
		ContextID:     &contextID,          // Enhanced: Direct VM context correlation
		ExternalJobID: &externalJobID,      // Enhanced: GUI job ID correlation (same as unified failover)
		JobCategory:   stringPtr("system"), // Enhanced: High-level categorization
		Metadata: map[string]interface{}{
			"context_id":      contextID, // Backward compatibility
			"vm_name_or_id":   vmNameOrID,
			"vmware_vm_id":    vmwareVMID,
			"failover_type":   options.FailoverType,
			"power_on_source": options.PowerOnSourceVM,
			"force_cleanup":   options.ForceCleanup,
			"operation":       "unified-failover-rollback",
			"external_job_id": externalJobID, // Track external ID in metadata too
		},
	})
	if err != nil {
		return fmt.Errorf("failed to start rollback job: %w", err)
	}
	defer func() {
		if r := recover(); r != nil {
			// Handle panic and store sanitized summary
			err := fmt.Errorf("rollback panic: %v", r)
			ecs.jobTracker.EndJob(ctx, jobID, joblog.StatusFailed, err)
			ecs.storeRollbackSummary(ctx, jobID, externalJobID, contextID, "failed", err)
			panic(r) // Re-panic after logging
		}
	}()
	defer func() {
		// Store rollback summary on completion
		ecs.jobTracker.EndJob(ctx, jobID, joblog.StatusCompleted, nil)
		ecs.storeRollbackSummary(ctx, jobID, externalJobID, contextID, "completed", nil)
	}()

	// Get logger with job context
	logger := ecs.jobTracker.Logger(ctx)
	logger.Info("🔄 Starting unified failover rollback with optional behaviors",
		"failover_type", options.FailoverType,
		"power_on_source", options.PowerOnSourceVM,
		"vm_name_or_id", vmNameOrID,
		"vmware_vm_id", vmwareVMID)

	// PHASE 1: Standard cleanup (volumes, test VM, snapshots) - execute steps directly under this job
	logger.Info("📋 Phase 1: Standard cleanup operations")
	if err := ecs.ExecuteTestFailoverCleanupSteps(ctx, jobID, contextID, vmNameOrID); err != nil {
		if !options.ForceCleanup {
			return fmt.Errorf("standard cleanup failed during rollback: %w", err)
		}
		logger.Error("Standard cleanup failed but continuing due to force_cleanup option", "error", err)
	}

	// PHASE 2: Optional source VM power-on (only for live failover or user request)
	if options.PowerOnSourceVM && vmwareVMID != "" && ecs.snaClient != nil {
		logger.Info("⚡ Phase 2: Source VM power-on requested")

		if err := ecs.jobTracker.RunStep(ctx, jobID, "source-vm-power-on", func(ctx context.Context) error {
			// Get logger with step context for proper external_job_id correlation
			stepLogger := ecs.jobTracker.Logger(ctx)

			// 🔧 CREDENTIAL FIX: Get vCenter credentials from secure credential service (NO FALLBACKS)
			encryptionService, err := services.NewCredentialEncryptionService()
			if err != nil {
				stepLogger.Error("❌ Failed to initialize VMware credential encryption service for rollback power-on", "error", err.Error())
				return fmt.Errorf("failed to initialize VMware credential encryption service: %w. Please ensure encryption service is configured", err)
			}

			credentialService := services.NewVMwareCredentialService(&ecs.db, encryptionService)
			creds, err := credentialService.GetDefaultCredentials(ctx)
			if err != nil {
				stepLogger.Error("❌ Failed to retrieve VMware credentials from database for rollback power-on", "error", err.Error())
				return fmt.Errorf("failed to retrieve VMware credentials from database: %w. "+
					"Please ensure VMware credentials are configured in GUI (Settings → VMware Credentials)", err)
			}

			// Use service-managed credentials (NO fallback to hardcoded values)
			vcenterHost := creds.VCenterHost
			vcenterUsername := creds.Username
			vcenterPassword := creds.Password

			stepLogger.Info("✅ Using fresh VMware credentials from database for rollback power-on",
				"vcenter_host", vcenterHost,
				"username", vcenterUsername,
				"credential_source", "database")

			// First check current power state
			powerState, err := ecs.snaClient.GetVMPowerState(ctx, vmwareVMID, vcenterHost, vcenterUsername, vcenterPassword)
			if err != nil {
				return fmt.Errorf("failed to check source VM power state: %w", err)
			}

			stepLogger.Info("🔍 Current source VM power state", "power_state", powerState, "vmware_vm_id", vmwareVMID)

			if powerState == "poweredOff" {
				stepLogger.Info("🔌 Powering on source VM as part of rollback")
				if err := ecs.snaClient.PowerOnSourceVM(ctx, vmwareVMID, vcenterHost, vcenterUsername, vcenterPassword); err != nil {
					return fmt.Errorf("failed to power on source VM: %w", err)
				}

				// Wait for VM to actually power on (same logic as power-off wait)
				// VM power-on can take 30-120 seconds depending on OS boot time and VMware Tools
				stepLogger.Info("⏳ Waiting for VM to complete power-on sequence...")
				maxWaitTime := 180 * time.Second // 3 minutes for power-on (longer than power-off)
				pollInterval := 10 * time.Second // Check every 10 seconds
				startTime := time.Now()

				for time.Since(startTime) < maxWaitTime {
					time.Sleep(pollInterval)

					// Check current power state
					currentState, err := ecs.snaClient.GetVMPowerState(ctx, vmwareVMID, vcenterHost, vcenterUsername, vcenterPassword)
					if err != nil {
						// If we can't check state, continue waiting
						stepLogger.Warn("Failed to check power state during wait, continuing...", "error", err)
						continue
					}

					if currentState == "poweredOn" {
						stepLogger.Info("✅ Source VM powered on successfully", "final_state", currentState, "wait_duration", time.Since(startTime))
						return nil // Successfully powered on
					}

					stepLogger.Info("🔄 VM still powering on...", "current_state", currentState, "elapsed", time.Since(startTime))
				}

				// Timeout reached - check final state
				finalState, err := ecs.snaClient.GetVMPowerState(ctx, vmwareVMID, vcenterHost, vcenterUsername, vcenterPassword)
				if err == nil && finalState == "poweredOn" {
					stepLogger.Info("✅ Source VM powered on successfully (final check)", "final_state", finalState)
					return nil
				}

				// Still not powered on after timeout
				stepLogger.Warn("VM power-on timeout reached", "final_state", finalState, "timeout", maxWaitTime)
				return fmt.Errorf("VM power-on timeout: VM still %s after %v", finalState, maxWaitTime)
			} else {
				stepLogger.Info("ℹ️ Source VM already powered on, no action needed", "power_state", powerState)
			}

			return nil
		}); err != nil {
			// Log error but don't fail rollback - user can manually power on
			logger.Error("Failed to power on source VM during rollback", "error", err, "vmware_vm_id", vmwareVMID)
			if !options.ForceCleanup {
				return fmt.Errorf("rollback completed but failed to power on source VM: %w", err)
			}
		}
	} else {
		if options.PowerOnSourceVM {
			logger.Info("ℹ️ Source VM power-on requested but SNA client not available or VMware VM ID missing")
		} else {
			logger.Info("ℹ️ Source VM left in current state (user choice)")
		}
	}

	// NOTE: VM context status was already updated to ready_for_failover in Phase 9 (vm-context-status-update)
	// No additional status update needed here

	logger.Info("✅ Unified failover rollback completed successfully",
		"context_id", contextID,
		"vm_name_or_id", vmNameOrID,
		"vmware_vm_id", vmwareVMID,
		"failover_type", options.FailoverType,
		"source_vm_powered_on", options.PowerOnSourceVM,
		"force_cleanup_used", options.ForceCleanup,
	)

	return nil
}

// CreateRollbackDecision creates a decision point for user interaction during rollback
func (ecs *EnhancedCleanupService) CreateRollbackDecision(failoverType, vmName string) *RollbackDecision {
	switch failoverType {
	case "live":
		return &RollbackDecision{
			DecisionID:   "live-rollback-power-on",
			Question:     fmt.Sprintf("The source VM '%s' was powered off during live failover. Would you like to power it back on during rollback?", vmName),
			Options:      []string{"Yes, power on source VM", "No, leave powered off"},
			DefaultValue: "Yes, power on source VM",
			Required:     true,
		}
	case "test":
		return &RollbackDecision{
			DecisionID:   "test-cleanup-confirmation",
			Question:     fmt.Sprintf("Proceed with test failover cleanup for VM '%s'? This will remove the test VM and restore normal operations.", vmName),
			Options:      []string{"Yes, proceed with cleanup", "Cancel"},
			DefaultValue: "Yes, proceed with cleanup",
			Required:     true,
		}
	default:
		return &RollbackDecision{
			DecisionID:   "generic-rollback",
			Question:     fmt.Sprintf("Proceed with rollback for VM '%s'?", vmName),
			Options:      []string{"Yes", "No"},
			DefaultValue: "Yes",
			Required:     true,
		}
	}
}

// GetDefaultRollbackOptions returns sensible defaults for rollback options
func (ecs *EnhancedCleanupService) GetDefaultRollbackOptions(failoverType string) *RollbackOptions {
	switch failoverType {
	case "live":
		return &RollbackOptions{
			PowerOnSourceVM: true,  // Default to powering on for live failover rollback
			ForceCleanup:    false, // Don't force by default
			FailoverType:    "live",
		}
	case "test":
		return &RollbackOptions{
			PowerOnSourceVM: false, // Source VM wasn't touched in test failover
			ForceCleanup:    false, // Don't force by default
			FailoverType:    "test",
		}
	default:
		return &RollbackOptions{
			PowerOnSourceVM: false,
			ForceCleanup:    false,
			FailoverType:    failoverType,
		}
	}
}

// stringPtr returns a pointer to the given string
func stringPtr(s string) *string {
	return &s
}

// storeRollbackSummary saves a sanitized summary of the rollback operation for persistent GUI visibility
func (ecs *EnhancedCleanupService) storeRollbackSummary(
	ctx context.Context,
	jobID string,
	externalJobID string,
	contextID string,
	status string,
	err error,
) {
	logger := ecs.jobTracker.Logger(ctx)
	
	// Get complete job details from JobLog
	jobSummary, summaryErr := ecs.jobTracker.FindJobByAnyID(jobID)
	if summaryErr != nil {
		logger.Warn("Could not retrieve job summary for rollback storage", "error", summaryErr)
		return
	}
	
	// Build sanitized summary
	summary := map[string]interface{}{
		"job_id":           jobID,
		"external_job_id":  externalJobID,
		"operation_type":   "rollback",
		"status":           status,
		"progress":         jobSummary.Progress.StepCompletion,
		"timestamp":        time.Now(),
		"steps_completed":  jobSummary.Progress.CompletedSteps,
		"steps_total":      jobSummary.Progress.TotalSteps,
		"duration_seconds": jobSummary.Progress.RuntimeSeconds,
	}
	
	// Add sanitized error information if failed
	if status == "failed" && err != nil {
		// Find the failed step
		var failedStepName string
		for _, step := range jobSummary.Steps {
			if step.Status == joblog.StatusFailed {
				failedStepName = step.Name
				break
			}
		}
		
		if failedStepName != "" {
			// Sanitize the error for user display
			sanitized := SanitizeFailoverError(failedStepName, err)
			
			summary["failed_step"] = GetUserFriendlyStepName(failedStepName)
			summary["failed_step_internal"] = failedStepName
			summary["error_message"] = sanitized.UserMessage
			summary["error_category"] = sanitized.Category
			summary["error_severity"] = sanitized.Severity
			summary["actionable_steps"] = sanitized.ActionableSteps
			
			logger.Info("📋 Sanitized rollback error for GUI display",
				"failed_step", failedStepName,
				"user_message", sanitized.UserMessage,
				"category", sanitized.Category)
		}
	}
	
	// Serialize to JSON
	summaryJSON, jsonErr := json.Marshal(summary)
	if jsonErr != nil {
		logger.Error("Failed to marshal rollback summary", "error", jsonErr)
		return
	}
	
	// Store in VM context for persistent visibility
	updates := map[string]interface{}{
		"last_operation_summary": string(summaryJSON),
		"updated_at":             time.Now(),
	}
	
	dbErr := ecs.db.GetGormDB().Model(&database.VMReplicationContext{}).
		Where("context_id = ?", contextID).
		Updates(updates).Error
	
	if dbErr != nil {
		logger.Error("Failed to store rollback summary", "error", dbErr)
		return
	}
	
	logger.Info("✅ Stored sanitized rollback summary for persistent GUI visibility",
		"context_id", contextID,
		"status", status,
		"sanitized", status == "failed")
}
