package service

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/vexxhost/migratekit-volume-daemon/models"
)

// AutoRecoveryService provides automatic state recovery mechanisms for the Volume Daemon
type AutoRecoveryService struct {
	stateRecovery    *StateRecoveryService
	volumeService    VolumeManagementService
	repo             VolumeRepository
	recoveryInterval time.Duration
	isRunning        bool
	stopChan         chan struct{}
	wg               sync.WaitGroup
	mutex            sync.RWMutex
	lastRecoveryRun  time.Time
	recoveryStats    *RecoveryStats
}

// RecoveryStats tracks recovery statistics
type RecoveryStats struct {
	TotalRuns             int           `json:"total_runs"`
	LastRunTime           time.Time     `json:"last_run_time"`
	LastRunDuration       time.Duration `json:"last_run_duration"`
	TotalVolumesRecovered int           `json:"total_volumes_recovered"`
	TotalMappingsCreated  int           `json:"total_mappings_created"`
	TotalMappingsFixed    int           `json:"total_mappings_fixed"`
	LastRunErrors         []string      `json:"last_run_errors"`
	IsHealthy             bool          `json:"is_healthy"`
}

// NewAutoRecoveryService creates a new automatic recovery service
func NewAutoRecoveryService(
	stateRecovery *StateRecoveryService,
	volumeService VolumeManagementService,
	repo VolumeRepository,
) *AutoRecoveryService {
	return &AutoRecoveryService{
		stateRecovery:    stateRecovery,
		volumeService:    volumeService,
		repo:             repo,
		recoveryInterval: 5 * time.Minute, // Default 5-minute recovery checks
		stopChan:         make(chan struct{}),
		recoveryStats: &RecoveryStats{
			IsHealthy: true,
		},
	}
}

// Start begins the automatic recovery process
func (ars *AutoRecoveryService) Start(ctx context.Context) error {
	ars.mutex.Lock()
	defer ars.mutex.Unlock()

	if ars.isRunning {
		return fmt.Errorf("auto recovery service is already running")
	}

	ars.isRunning = true
	log.WithField("interval", ars.recoveryInterval).Info("🔄 Starting automatic state recovery service")

	// Start the recovery worker
	ars.wg.Add(1)
	go ars.recoveryWorker(ctx)

	return nil
}

// Stop gracefully stops the automatic recovery process
func (ars *AutoRecoveryService) Stop() error {
	ars.mutex.Lock()
	defer ars.mutex.Unlock()

	if !ars.isRunning {
		return fmt.Errorf("auto recovery service is not running")
	}

	log.Info("🛑 Stopping automatic state recovery service")
	close(ars.stopChan)
	ars.wg.Wait()
	ars.isRunning = false

	return nil
}

// IsRunning returns whether the auto recovery service is currently running
func (ars *AutoRecoveryService) IsRunning() bool {
	ars.mutex.RLock()
	defer ars.mutex.RUnlock()
	return ars.isRunning
}

// GetStats returns the current recovery statistics
func (ars *AutoRecoveryService) GetStats() *RecoveryStats {
	ars.mutex.RLock()
	defer ars.mutex.RUnlock()

	// Return a copy to prevent race conditions
	statsCopy := *ars.recoveryStats
	return &statsCopy
}

// TriggerManualRecovery manually triggers a recovery run
func (ars *AutoRecoveryService) TriggerManualRecovery(ctx context.Context) (*RecoveryResult, error) {
	log.Info("🔄 Manual recovery triggered")
	return ars.performRecoveryRun(ctx)
}

// RecoverVolumeOnDemand attempts to recover a specific volume when a "mapping not found" error occurs
func (ars *AutoRecoveryService) RecoverVolumeOnDemand(ctx context.Context, volumeID string) (*models.DeviceMapping, error) {
	log.WithField("volume_id", volumeID).Info("🔄 On-demand volume recovery triggered")

	// Attempt single volume recovery
	mapping, err := ars.stateRecovery.RecoverSingleVolume(ctx, volumeID)
	if err != nil {
		log.WithError(err).WithField("volume_id", volumeID).Error("❌ On-demand recovery failed")
		return nil, err
	}

	log.WithFields(log.Fields{
		"volume_id":   volumeID,
		"device_path": mapping.DevicePath,
	}).Info("✅ On-demand recovery successful")

	return mapping, nil
}

// EnhancedVolumeOperation wraps volume operations with automatic recovery
func (ars *AutoRecoveryService) EnhancedVolumeOperation(ctx context.Context, operation func() error, volumeID string) error {
	// First attempt
	err := operation()
	if err == nil {
		return nil
	}

	// Check if error indicates missing mapping
	if ars.isMappingNotFoundError(err) {
		log.WithFields(log.Fields{
			"volume_id": volumeID,
			"error":     err.Error(),
		}).Warn("🔄 Mapping not found error detected, attempting recovery")

		// Attempt recovery
		_, recoveryErr := ars.RecoverVolumeOnDemand(ctx, volumeID)
		if recoveryErr != nil {
			log.WithError(recoveryErr).WithField("volume_id", volumeID).Error("❌ Recovery failed")
			return fmt.Errorf("operation failed and recovery failed: original error: %w, recovery error: %v", err, recoveryErr)
		}

		// Retry the operation
		retryErr := operation()
		if retryErr != nil {
			log.WithError(retryErr).WithField("volume_id", volumeID).Error("❌ Operation failed even after recovery")
			return fmt.Errorf("operation failed after recovery: %w", retryErr)
		}

		log.WithField("volume_id", volumeID).Info("✅ Operation succeeded after recovery")
		return nil
	}

	// Not a mapping error, return original error
	return err
}

// SetRecoveryInterval updates the recovery interval
func (ars *AutoRecoveryService) SetRecoveryInterval(interval time.Duration) {
	ars.mutex.Lock()
	defer ars.mutex.Unlock()
	ars.recoveryInterval = interval
	log.WithField("new_interval", interval).Info("🔄 Updated recovery interval")
}

// Private methods

// recoveryWorker runs the periodic recovery checks
func (ars *AutoRecoveryService) recoveryWorker(ctx context.Context) {
	defer ars.wg.Done()

	ticker := time.NewTicker(ars.recoveryInterval)
	defer ticker.Stop()

	log.WithField("interval", ars.recoveryInterval).Info("🔄 Recovery worker started")

	for {
		select {
		case <-ctx.Done():
			log.Info("🔄 Recovery worker stopped due to context cancellation")
			return
		case <-ars.stopChan:
			log.Info("🔄 Recovery worker stopped")
			return
		case <-ticker.C:
			// Perform recovery check
			if err := ars.performPeriodicCheck(ctx); err != nil {
				log.WithError(err).Error("❌ Periodic recovery check failed")
			}
		}
	}
}

// performPeriodicCheck performs a periodic recovery check
func (ars *AutoRecoveryService) performPeriodicCheck(ctx context.Context) error {
	log.Debug("🔍 Performing periodic recovery check")

	// Check for stale operations that might indicate lost mappings
	staleOperations, err := ars.findStaleOperations(ctx)
	if err != nil {
		return fmt.Errorf("failed to find stale operations: %w", err)
	}

	if len(staleOperations) > 0 {
		log.WithField("stale_count", len(staleOperations)).Warn("🚨 Found stale operations, triggering recovery")

		// Trigger full recovery
		result, err := ars.performRecoveryRun(ctx)
		if err != nil {
			return fmt.Errorf("recovery run failed: %w", err)
		}

		log.WithFields(log.Fields{
			"volumes_recovered": result.VolumesRecovered,
			"mappings_created":  result.MappingsCreated,
			"mappings_fixed":    result.MappingsFixed,
		}).Info("✅ Periodic recovery completed")
	} else {
		log.Debug("✅ No issues detected in periodic check")
	}

	return nil
}

// performRecoveryRun executes a full recovery run
func (ars *AutoRecoveryService) performRecoveryRun(ctx context.Context) (*RecoveryResult, error) {
	startTime := time.Now()

	log.Info("🔄 Starting recovery run")

	// Perform full system recovery
	result, err := ars.stateRecovery.PerformFullSystemRecovery(ctx)
	if err != nil {
		ars.updateStats(startTime, result, []string{err.Error()})
		return nil, fmt.Errorf("full system recovery failed: %w", err)
	}

	// Update statistics
	ars.updateStats(startTime, result, result.Errors)

	log.WithFields(log.Fields{
		"volumes_recovered": result.VolumesRecovered,
		"volumes_orphaned":  result.VolumesOrphaned,
		"mappings_created":  result.MappingsCreated,
		"mappings_fixed":    result.MappingsFixed,
		"duration":          result.Duration,
		"error_count":       len(result.Errors),
	}).Info("🎯 Recovery run completed")

	return result, nil
}

// findStaleOperations looks for operations that might indicate lost mappings
func (ars *AutoRecoveryService) findStaleOperations(ctx context.Context) ([]*models.VolumeOperation, error) {
	// Look for operations that have been stuck in "executing" state for too long
	cutoffTime := time.Now().Add(-10 * time.Minute) // Operations older than 10 minutes

	// TODO: Implement GetOperationsByStatus in repository
	// For now, return empty slice to avoid errors
	operations := make([]*models.VolumeOperation, 0)

	var staleOperations []*models.VolumeOperation
	for _, op := range operations {
		if op.CreatedAt.Before(cutoffTime) {
			staleOperations = append(staleOperations, op)
		}
	}

	return staleOperations, nil
}

// updateStats updates the recovery statistics
func (ars *AutoRecoveryService) updateStats(startTime time.Time, result *RecoveryResult, errors []string) {
	ars.mutex.Lock()
	defer ars.mutex.Unlock()

	ars.recoveryStats.TotalRuns++
	ars.recoveryStats.LastRunTime = startTime
	ars.recoveryStats.LastRunDuration = time.Since(startTime)
	ars.recoveryStats.LastRunErrors = errors

	if result != nil {
		ars.recoveryStats.TotalVolumesRecovered += result.VolumesRecovered
		ars.recoveryStats.TotalMappingsCreated += result.MappingsCreated
		ars.recoveryStats.TotalMappingsFixed += result.MappingsFixed
	}

	// Update health status
	ars.recoveryStats.IsHealthy = len(errors) == 0

	ars.lastRecoveryRun = startTime
}

// isMappingNotFoundError checks if an error indicates a missing mapping
func (ars *AutoRecoveryService) isMappingNotFoundError(err error) bool {
	if err == nil {
		return false
	}

	errorMsg := err.Error()
	return strings.Contains(errorMsg, "mapping not found") ||
		strings.Contains(errorMsg, "device mapping") ||
		strings.Contains(errorMsg, "volume not found")
}

// GetHealthStatus returns the health status of the auto recovery service
func (ars *AutoRecoveryService) GetHealthStatus() map[string]interface{} {
	ars.mutex.RLock()
	defer ars.mutex.RUnlock()

	timeSinceLastRun := time.Since(ars.lastRecoveryRun)

	return map[string]interface{}{
		"is_running":              ars.isRunning,
		"recovery_interval":       ars.recoveryInterval.String(),
		"last_recovery_run":       ars.lastRecoveryRun,
		"time_since_last_run":     timeSinceLastRun.String(),
		"is_healthy":              ars.recoveryStats.IsHealthy,
		"total_runs":              ars.recoveryStats.TotalRuns,
		"total_volumes_recovered": ars.recoveryStats.TotalVolumesRecovered,
		"total_mappings_created":  ars.recoveryStats.TotalMappingsCreated,
		"total_mappings_fixed":    ars.recoveryStats.TotalMappingsFixed,
		"last_run_duration":       ars.recoveryStats.LastRunDuration.String(),
		"error_count":             len(ars.recoveryStats.LastRunErrors),
		"errors":                  ars.recoveryStats.LastRunErrors,
	}
}
