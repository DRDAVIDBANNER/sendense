// Package failover provides VM cleanup operations for enhanced test failover cleanup
package failover

import (
	"context"
	"fmt"
	"time"

	"github.com/vexxhost/migratekit-oma/joblog"
	"github.com/vexxhost/migratekit-oma/ossea"
)

// VMCleanupOperations handles VM-related cleanup operations
type VMCleanupOperations struct {
	jobTracker *joblog.Tracker
	helpers    *CleanupHelpers
}

// NewVMCleanupOperations creates a new VM cleanup operations handler
// Note: Takes CleanupHelpers for dynamic credential initialization
func NewVMCleanupOperations(osseaClient *ossea.Client, jobTracker *joblog.Tracker, helpers *CleanupHelpers) *VMCleanupOperations {
	return &VMCleanupOperations{
		jobTracker: jobTracker,
		helpers:    helpers,
	}
}

// StopTestVM stops a test VM with proper error handling and JobLog integration
func (vco *VMCleanupOperations) StopTestVM(ctx context.Context, testVMID string) error {
	logger := vco.jobTracker.Logger(ctx)
	logger.Info("🛑 Stopping test VM for cleanup", "test_vm_id", testVMID)

	// 🔧 CREDENTIAL FIX: Initialize fresh OSSEA client from database
	osseaClient, err := vco.helpers.InitializeOSSEAClient(ctx)
	if err != nil {
		logger.Error("❌ Failed to initialize OSSEA client for VM stop", "error", err.Error())
		return fmt.Errorf("failed to initialize OSSEA client: %w", err)
	}

	// Get VM details first to check current state
	vm, err := osseaClient.GetVMDetailed(testVMID)
	if err != nil {
		logger.Error("Failed to get VM details for shutdown", "error", err, "test_vm_id", testVMID)
		return fmt.Errorf("failed to get VM details: %w", err)
	}

	logger.Info("Retrieved VM details for shutdown",
		"test_vm_id", testVMID,
		"vm_name", vm.Name,
		"current_state", vm.State,
	)

	// Check if VM is already stopped
	if vm.State == "Stopped" {
		logger.Info("VM is already stopped, skipping shutdown", "test_vm_id", testVMID)
		return nil
	}

	// Stop the VM using CloudStack API
	err = osseaClient.StopVM(testVMID, false)
	if err != nil {
		logger.Error("Failed to stop test VM", "error", err, "test_vm_id", testVMID)
		return fmt.Errorf("failed to stop test VM: %w", err)
	}

	// Wait for VM to reach stopped state with timeout
	timeout := 5 * time.Minute
	logger.Info("⏳ Waiting for VM to stop", "test_vm_id", testVMID, "timeout", timeout)

	err = osseaClient.WaitForVMState(testVMID, "Stopped", timeout)
	if err != nil {
		logger.Error("VM failed to stop within timeout", "error", err, "test_vm_id", testVMID, "timeout", timeout)
		return fmt.Errorf("VM failed to stop within timeout: %w", err)
	}

	logger.Info("✅ Test VM stopped successfully", "test_vm_id", testVMID)
	return nil
}

// DeleteTestVM deletes a test VM with proper cleanup verification
func (vco *VMCleanupOperations) DeleteTestVM(ctx context.Context, testVMID string) error {
	logger := vco.jobTracker.Logger(ctx)
	logger.Info("🗑️ Deleting test VM", "test_vm_id", testVMID)

	// 🔧 CREDENTIAL FIX: Initialize fresh OSSEA client from database
	osseaClient, err := vco.helpers.InitializeOSSEAClient(ctx)
	if err != nil {
		logger.Error("❌ Failed to initialize OSSEA client for VM deletion", "error", err.Error())
		return fmt.Errorf("failed to initialize OSSEA client: %w", err)
	}

	// Verify VM exists before deletion
	vm, err := osseaClient.GetVMDetailed(testVMID)
	if err != nil {
		logger.Error("Failed to get VM details for deletion", "error", err, "test_vm_id", testVMID)
		return fmt.Errorf("failed to get VM details for deletion: %w", err)
	}

	logger.Info("Retrieved VM details for deletion",
		"test_vm_id", testVMID,
		"vm_name", vm.Name,
		"state", vm.State,
	)

	// Ensure VM is stopped before deletion
	if vm.State != "Stopped" {
		logger.Warn("VM is not stopped, attempting to stop before deletion", "test_vm_id", testVMID, "current_state", vm.State)
		if err := vco.StopTestVM(ctx, testVMID); err != nil {
			return fmt.Errorf("failed to stop VM before deletion: %w", err)
		}
	}

	// Delete the VM using CloudStack API
	err = osseaClient.DeleteVM(testVMID, true)
	if err != nil {
		logger.Error("Failed to delete test VM", "error", err, "test_vm_id", testVMID)
		return fmt.Errorf("failed to delete test VM: %w", err)
	}

	logger.Info("✅ Test VM deleted successfully", "test_vm_id", testVMID)
	return nil
}
