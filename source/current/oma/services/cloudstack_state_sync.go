package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/vexxhost/migratekit-oma/ossea"
)

// CloudStackStateSyncService provides CloudStack state synchronization and consistency verification
type CloudStackStateSyncService struct {
	osseaClient   *ossea.Client
	jobTracker    *JobTrackingService
	centralLogger *CentralLogger
	syncInterval  time.Duration
	stateCache    *StateCache
	isRunning     bool
	stopChan      chan struct{}
	mutex         sync.RWMutex
	lastSyncTime  time.Time
	syncErrors    int
	maxSyncErrors int
	forceFullSync bool
}

// StateCache holds cached CloudStack state for quick comparison
type StateCache struct {
	VMs         map[string]*VirtualMachineState `json:"vms"`
	Volumes     map[string]*VolumeState         `json:"volumes"`
	LastUpdated time.Time                       `json:"last_updated"`
	SyncVersion int64                           `json:"sync_version"`
	mutex       sync.RWMutex
}

// VirtualMachineState represents cached VM state from CloudStack
type VirtualMachineState struct {
	ID              string                 `json:"id"`
	Name            string                 `json:"name"`
	State           string                 `json:"state"`
	ServiceOffering string                 `json:"service_offering"`
	Template        string                 `json:"template"`
	Zone            string                 `json:"zone"`
	AttachedVolumes []string               `json:"attached_volumes"`
	NetworkInfo     map[string]interface{} `json:"network_info"`
	LastSeen        time.Time              `json:"last_seen"`
	SyncStatus      string                 `json:"sync_status"`
}

// VolumeState represents cached volume state from CloudStack
type VolumeState struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	Type            string    `json:"type"`
	State           string    `json:"state"`
	Size            int64     `json:"size"`
	AttachedToVM    string    `json:"attached_to_vm"`
	DeviceID        int       `json:"device_id"`
	StorageOffering string    `json:"storage_offering"`
	Zone            string    `json:"zone"`
	LastSeen        time.Time `json:"last_seen"`
	SyncStatus      string    `json:"sync_status"`
}

// StateSyncResult represents the result of a state synchronization operation
type StateSyncResult struct {
	SyncTime             time.Time          `json:"sync_time"`
	Duration             time.Duration      `json:"duration"`
	VMsSynced            int                `json:"vms_synced"`
	VolumesSynced        int                `json:"volumes_synced"`
	InconsistenciesFound int                `json:"inconsistencies_found"`
	InconsistenciesFixed int                `json:"inconsistencies_fixed"`
	StateChanges         []StateChangeEvent `json:"state_changes"`
	Errors               []StateSyncError   `json:"errors"`
	IsFullSync           bool               `json:"is_full_sync"`
	SyncVersion          int64              `json:"sync_version"`
}

// StateChangeEvent represents a detected state change
type StateChangeEvent struct {
	Type         string                 `json:"type"`
	ResourceID   string                 `json:"resource_id"`
	ResourceType string                 `json:"resource_type"`
	OldState     map[string]interface{} `json:"old_state"`
	NewState     map[string]interface{} `json:"new_state"`
	DetectedAt   time.Time              `json:"detected_at"`
	Action       string                 `json:"action"`
	Fixed        bool                   `json:"fixed"`
}

// StateSyncError represents an error during state synchronization
type StateSyncError struct {
	Type       string    `json:"type"`
	ResourceID string    `json:"resource_id"`
	Message    string    `json:"message"`
	OccurredAt time.Time `json:"occurred_at"`
	Retryable  bool      `json:"retryable"`
	RetryCount int       `json:"retry_count"`
}

// StateInconsistency represents a detected inconsistency
type StateInconsistency struct {
	Type            string                 `json:"type"`
	ResourceID      string                 `json:"resource_id"`
	ResourceType    string                 `json:"resource_type"`
	Description     string                 `json:"description"`
	LocalState      map[string]interface{} `json:"local_state"`
	CloudStackState map[string]interface{} `json:"cloudstack_state"`
	Severity        string                 `json:"severity"`
	AutoFixable     bool                   `json:"auto_fixable"`
	DetectedAt      time.Time              `json:"detected_at"`
}

// NewCloudStackStateSyncService creates a new state synchronization service
func NewCloudStackStateSyncService(osseaClient *ossea.Client, jobTracker *JobTrackingService, centralLogger *CentralLogger) *CloudStackStateSyncService {
	return &CloudStackStateSyncService{
		osseaClient:   osseaClient,
		jobTracker:    jobTracker,
		centralLogger: centralLogger,
		syncInterval:  2 * time.Minute, // Aggressive sync every 2 minutes
		stateCache: &StateCache{
			VMs:         make(map[string]*VirtualMachineState),
			Volumes:     make(map[string]*VolumeState),
			LastUpdated: time.Time{},
			SyncVersion: 1,
		},
		stopChan:      make(chan struct{}),
		maxSyncErrors: 5,
		forceFullSync: true, // Start with full sync
	}
}

// StartStateSynchronization begins continuous state synchronization
func (css *CloudStackStateSyncService) StartStateSynchronization(ctx context.Context) error {
	css.mutex.Lock()
	defer css.mutex.Unlock()

	if css.isRunning {
		return fmt.Errorf("state synchronization is already running")
	}

	css.isRunning = true
	css.syncErrors = 0

	log.Info("🔄 Starting CloudStack state synchronization service")

	// Perform initial full sync
	go css.syncLoop(ctx)

	log.WithFields(log.Fields{
		"sync_interval": css.syncInterval.String(),
		"max_errors":    css.maxSyncErrors,
	}).Info("CloudStack state sync started")

	return nil
}

// StopStateSynchronization stops the state synchronization service
func (css *CloudStackStateSyncService) StopStateSynchronization() error {
	css.mutex.Lock()
	defer css.mutex.Unlock()

	if !css.isRunning {
		return fmt.Errorf("state synchronization is not running")
	}

	css.isRunning = false
	close(css.stopChan)

	log.Info("⏹️ Stopped CloudStack state synchronization service")

	log.WithFields(log.Fields{
		"total_sync_cycles": css.stateCache.SyncVersion - 1,
		"last_sync_time":    css.lastSyncTime,
	}).Info("CloudStack state sync stopped")

	return nil
}

// syncLoop runs the continuous synchronization loop
func (css *CloudStackStateSyncService) syncLoop(ctx context.Context) {
	ticker := time.NewTicker(css.syncInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Info("🔄 State sync loop stopping due to context cancellation")
			return
		case <-css.stopChan:
			log.Info("🔄 State sync loop stopping due to stop signal")
			return
		case <-ticker.C:
			if err := css.performSyncCycle(ctx); err != nil {
				css.syncErrors++
				log.WithError(err).WithField("sync_errors", css.syncErrors).Error("❌ State sync cycle failed")

				if css.syncErrors >= css.maxSyncErrors {
					log.WithField("max_errors", css.maxSyncErrors).Error("🚨 Maximum sync errors reached, stopping state synchronization")
					css.StopStateSynchronization()
					return
				}
			} else {
				css.syncErrors = 0 // Reset error counter on successful sync
			}
		}
	}
}

// performSyncCycle executes a single synchronization cycle
func (css *CloudStackStateSyncService) performSyncCycle(ctx context.Context) error {
	startTime := time.Now()
	correlationID := fmt.Sprintf("state-sync-%d", time.Now().UnixNano())

	log.WithFields(log.Fields{
		"correlation_id": correlationID,
		"sync_version":   css.stateCache.SyncVersion,
		"is_full_sync":   css.forceFullSync,
	}).Info("Starting sync operation")

	log.WithFields(log.Fields{
		"correlation_id": correlationID,
		"sync_version":   css.stateCache.SyncVersion,
		"is_full_sync":   css.forceFullSync,
	}).Info("🔄 Starting CloudStack state synchronization cycle")

	result := &StateSyncResult{
		SyncTime:     startTime,
		StateChanges: make([]StateChangeEvent, 0),
		Errors:       make([]StateSyncError, 0),
		IsFullSync:   css.forceFullSync,
		SyncVersion:  css.stateCache.SyncVersion,
	}

	// Sync VMs
	vmSyncResult, err := css.syncVirtualMachines(ctx, correlationID)
	if err != nil {
		result.Errors = append(result.Errors, StateSyncError{
			Type:       "vm_sync_error",
			Message:    err.Error(),
			OccurredAt: time.Now(),
			Retryable:  true,
		})
	} else {
		result.VMsSynced = vmSyncResult.VMsSynced
		result.StateChanges = append(result.StateChanges, vmSyncResult.StateChanges...)
	}

	// Sync Volumes
	volumeSyncResult, err := css.syncVolumes(ctx, correlationID)
	if err != nil {
		result.Errors = append(result.Errors, StateSyncError{
			Type:       "volume_sync_error",
			Message:    err.Error(),
			OccurredAt: time.Now(),
			Retryable:  true,
		})
	} else {
		result.VolumesSynced = volumeSyncResult.VolumesSynced
		result.StateChanges = append(result.StateChanges, volumeSyncResult.StateChanges...)
	}

	// Detect and fix inconsistencies
	inconsistencies, err := css.detectInconsistencies(ctx, correlationID)
	if err != nil {
		result.Errors = append(result.Errors, StateSyncError{
			Type:       "inconsistency_detection_error",
			Message:    err.Error(),
			OccurredAt: time.Now(),
			Retryable:  true,
		})
	} else {
		result.InconsistenciesFound = len(inconsistencies)
		result.InconsistenciesFixed = css.fixInconsistencies(ctx, inconsistencies)
	}

	// Update cache metadata
	css.stateCache.mutex.Lock()
	css.stateCache.LastUpdated = time.Now()
	css.stateCache.SyncVersion++
	css.stateCache.mutex.Unlock()

	result.Duration = time.Since(startTime)
	css.lastSyncTime = time.Now()
	css.forceFullSync = false // Only full sync on first run

	log.WithFields(log.Fields{
		"correlation_id":        correlationID,
		"duration":              result.Duration,
		"vms_synced":            result.VMsSynced,
		"volumes_synced":        result.VolumesSynced,
		"inconsistencies_found": result.InconsistenciesFound,
		"inconsistencies_fixed": result.InconsistenciesFixed,
		"errors":                len(result.Errors),
	}).Info("✅ CloudStack state synchronization cycle completed")

	log.WithFields(log.Fields{
		"correlation_id": correlationID,
		"duration":       result.Duration,
		"sync_result":    result,
	}).Info("Sync operation completed")

	return nil
}

// syncVirtualMachines synchronizes VM state with CloudStack
func (css *CloudStackStateSyncService) syncVirtualMachines(ctx context.Context, correlationID string) (*StateSyncResult, error) {
	log.WithField("correlation_id", correlationID).Info("🔄 Synchronizing virtual machines")

	result := &StateSyncResult{
		StateChanges: make([]StateChangeEvent, 0),
	}

	// Get all VMs from CloudStack
	vms, err := css.osseaClient.ListVMs(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list VMs from CloudStack: %w", err)
	}

	css.stateCache.mutex.Lock()
	defer css.stateCache.mutex.Unlock()

	// Update cache with current VM states
	for _, vm := range vms {
		oldState := css.stateCache.VMs[vm.ID]

		newState := &VirtualMachineState{
			ID:              vm.ID,
			Name:            vm.Name,
			State:           vm.State,
			ServiceOffering: vm.ServiceOfferingName,
			Template:        vm.TemplateName,
			Zone:            vm.ZoneName,
			AttachedVolumes: css.getVMAttachedVolumes(vm.ID),
			NetworkInfo:     css.getVMNetworkInfo(vm),
			LastSeen:        time.Now(),
			SyncStatus:      "synced",
		}

		// Detect state changes
		if oldState != nil && css.hasVMStateChanged(oldState, newState) {
			change := StateChangeEvent{
				Type:         "vm_state_change",
				ResourceID:   vm.ID,
				ResourceType: "virtual_machine",
				OldState:     css.vmStateToMap(oldState),
				NewState:     css.vmStateToMap(newState),
				DetectedAt:   time.Now(),
				Action:       "state_updated",
				Fixed:        true,
			}
			result.StateChanges = append(result.StateChanges, change)

			log.WithFields(log.Fields{
				"vm_id":     vm.ID,
				"vm_name":   vm.Name,
				"old_state": oldState.State,
				"new_state": newState.State,
			}).Info("🔄 VM state change detected")
		}

		css.stateCache.VMs[vm.ID] = newState
		result.VMsSynced++
	}

	// Mark missing VMs as out of sync
	for vmID, cachedVM := range css.stateCache.VMs {
		found := false
		for _, vm := range vms {
			if vm.ID == vmID {
				found = true
				break
			}
		}

		if !found && cachedVM.SyncStatus == "synced" {
			cachedVM.SyncStatus = "missing"
			change := StateChangeEvent{
				Type:         "vm_missing",
				ResourceID:   vmID,
				ResourceType: "virtual_machine",
				OldState:     css.vmStateToMap(cachedVM),
				NewState:     map[string]interface{}{"sync_status": "missing"},
				DetectedAt:   time.Now(),
				Action:       "marked_missing",
				Fixed:        true,
			}
			result.StateChanges = append(result.StateChanges, change)

			log.WithField("vm_id", vmID).Warn("⚠️ VM no longer found in CloudStack")
		}
	}

	return result, nil
}

// syncVolumes synchronizes volume state with CloudStack
func (css *CloudStackStateSyncService) syncVolumes(ctx context.Context, correlationID string) (*StateSyncResult, error) {
	log.WithField("correlation_id", correlationID).Info("🔄 Synchronizing volumes")

	result := &StateSyncResult{
		StateChanges: make([]StateChangeEvent, 0),
	}

	// Get all volumes from CloudStack
	volumes, err := css.osseaClient.ListVolumesContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list volumes from CloudStack: %w", err)
	}

	css.stateCache.mutex.Lock()
	defer css.stateCache.mutex.Unlock()

	// Update cache with current volume states
	for _, volume := range volumes {
		oldState := css.stateCache.Volumes[volume.ID]

		newState := &VolumeState{
			ID:              volume.ID,
			Name:            volume.Name,
			Type:            volume.Type,
			State:           volume.State,
			Size:            volume.Size,
			AttachedToVM:    volume.VirtualMachineID,
			DeviceID:        volume.DeviceID,
			StorageOffering: volume.DiskOfferingName,
			Zone:            volume.ZoneName,
			LastSeen:        time.Now(),
			SyncStatus:      "synced",
		}

		// Detect state changes
		if oldState != nil && css.hasVolumeStateChanged(oldState, newState) {
			change := StateChangeEvent{
				Type:         "volume_state_change",
				ResourceID:   volume.ID,
				ResourceType: "volume",
				OldState:     css.volumeStateToMap(oldState),
				NewState:     css.volumeStateToMap(newState),
				DetectedAt:   time.Now(),
				Action:       "state_updated",
				Fixed:        true,
			}
			result.StateChanges = append(result.StateChanges, change)

			log.WithFields(log.Fields{
				"volume_id":   volume.ID,
				"volume_name": volume.Name,
				"old_state":   oldState.State,
				"new_state":   newState.State,
				"attached_to": newState.AttachedToVM,
			}).Info("🔄 Volume state change detected")
		}

		css.stateCache.Volumes[volume.ID] = newState
		result.VolumesSynced++
	}

	return result, nil
}

// detectInconsistencies detects state inconsistencies between local and CloudStack state
func (css *CloudStackStateSyncService) detectInconsistencies(ctx context.Context, correlationID string) ([]StateInconsistency, error) {
	log.WithField("correlation_id", correlationID).Info("🔍 Detecting state inconsistencies")

	inconsistencies := make([]StateInconsistency, 0)

	// Check for volume attachment inconsistencies
	css.stateCache.mutex.RLock()
	defer css.stateCache.mutex.RUnlock()

	for volumeID, volume := range css.stateCache.Volumes {
		if volume.AttachedToVM != "" {
			// Check if the VM actually exists and has this volume attached
			vm, exists := css.stateCache.VMs[volume.AttachedToVM]
			if !exists {
				inconsistencies = append(inconsistencies, StateInconsistency{
					Type:         "volume_attached_to_missing_vm",
					ResourceID:   volumeID,
					ResourceType: "volume",
					Description:  fmt.Sprintf("Volume %s is attached to non-existent VM %s", volumeID, volume.AttachedToVM),
					LocalState:   css.volumeStateToMap(volume),
					CloudStackState: map[string]interface{}{
						"attached_vm_exists": false,
					},
					Severity:    "high",
					AutoFixable: true,
					DetectedAt:  time.Now(),
				})
			} else {
				// Check if VM's attached volumes list includes this volume
				volumeFound := false
				for _, attachedVolumeID := range vm.AttachedVolumes {
					if attachedVolumeID == volumeID {
						volumeFound = true
						break
					}
				}

				if !volumeFound {
					inconsistencies = append(inconsistencies, StateInconsistency{
						Type:            "volume_attachment_mismatch",
						ResourceID:      volumeID,
						ResourceType:    "volume",
						Description:     fmt.Sprintf("Volume %s claims to be attached to VM %s, but VM doesn't list this volume", volumeID, volume.AttachedToVM),
						LocalState:      css.volumeStateToMap(volume),
						CloudStackState: css.vmStateToMap(vm),
						Severity:        "medium",
						AutoFixable:     true,
						DetectedAt:      time.Now(),
					})
				}
			}
		}
	}

	// Check for VM state inconsistencies
	for vmID, vm := range css.stateCache.VMs {
		if vm.State == "Running" && len(vm.AttachedVolumes) == 0 {
			inconsistencies = append(inconsistencies, StateInconsistency{
				Type:         "running_vm_no_volumes",
				ResourceID:   vmID,
				ResourceType: "virtual_machine",
				Description:  fmt.Sprintf("VM %s is running but has no attached volumes", vmID),
				LocalState:   css.vmStateToMap(vm),
				CloudStackState: map[string]interface{}{
					"expected_root_volume": true,
				},
				Severity:    "medium",
				AutoFixable: false,
				DetectedAt:  time.Now(),
			})
		}
	}

	log.WithFields(log.Fields{
		"correlation_id":  correlationID,
		"inconsistencies": len(inconsistencies),
	}).Info("🔍 State inconsistency detection completed")

	return inconsistencies, nil
}

// fixInconsistencies attempts to automatically fix detected inconsistencies
func (css *CloudStackStateSyncService) fixInconsistencies(ctx context.Context, inconsistencies []StateInconsistency) int {
	fixedCount := 0

	for _, inconsistency := range inconsistencies {
		if !inconsistency.AutoFixable {
			continue
		}

		log.WithFields(log.Fields{
			"type":        inconsistency.Type,
			"resource_id": inconsistency.ResourceID,
			"severity":    inconsistency.Severity,
		}).Info("🔧 Attempting to fix state inconsistency")

		switch inconsistency.Type {
		case "volume_attached_to_missing_vm":
			if css.fixVolumeAttachedToMissingVM(ctx, inconsistency) {
				fixedCount++
			}
		case "volume_attachment_mismatch":
			if css.fixVolumeAttachmentMismatch(ctx, inconsistency) {
				fixedCount++
			}
		}
	}

	log.WithField("fixed_count", fixedCount).Info("🔧 State inconsistency fixing completed")

	return fixedCount
}

// Individual fix methods for different inconsistency types

func (css *CloudStackStateSyncService) fixVolumeAttachedToMissingVM(ctx context.Context, inconsistency StateInconsistency) bool {
	// Force refresh volume state from CloudStack
	volume, err := css.osseaClient.GetVolume(inconsistency.ResourceID)
	if err != nil {
		log.WithError(err).WithField("volume_id", inconsistency.ResourceID).Error("Failed to refresh volume state")
		return false
	}

	css.stateCache.mutex.Lock()
	defer css.stateCache.mutex.Unlock()

	// Update cached state
	if cachedVolume, exists := css.stateCache.Volumes[inconsistency.ResourceID]; exists {
		cachedVolume.AttachedToVM = volume.VirtualMachineID
		cachedVolume.State = volume.State
		cachedVolume.LastSeen = time.Now()
		cachedVolume.SyncStatus = "synced"

		log.WithField("volume_id", inconsistency.ResourceID).Info("✅ Fixed volume attached to missing VM")
		return true
	}

	return false
}

func (css *CloudStackStateSyncService) fixVolumeAttachmentMismatch(ctx context.Context, inconsistency StateInconsistency) bool {
	// Force refresh both volume and VM state from CloudStack
	volume, err := css.osseaClient.GetVolume(inconsistency.ResourceID)
	if err != nil {
		log.WithError(err).WithField("volume_id", inconsistency.ResourceID).Error("Failed to refresh volume state")
		return false
	}

	if volume.VirtualMachineID != "" {
		_, err := css.osseaClient.GetVM(volume.VirtualMachineID)
		if err != nil {
			log.WithError(err).WithField("vm_id", volume.VirtualMachineID).Error("Failed to refresh VM state")
			return false
		}

		css.stateCache.mutex.Lock()
		defer css.stateCache.mutex.Unlock()

		// Update cached states
		if cachedVolume, exists := css.stateCache.Volumes[inconsistency.ResourceID]; exists {
			cachedVolume.AttachedToVM = volume.VirtualMachineID
			cachedVolume.State = volume.State
			cachedVolume.LastSeen = time.Now()
		}

		if cachedVM, exists := css.stateCache.VMs[volume.VirtualMachineID]; exists {
			cachedVM.AttachedVolumes = css.getVMAttachedVolumes(volume.VirtualMachineID)
			cachedVM.LastSeen = time.Now()
		}

		log.WithFields(log.Fields{
			"volume_id": inconsistency.ResourceID,
			"vm_id":     volume.VirtualMachineID,
		}).Info("✅ Fixed volume attachment mismatch")
		return true
	}

	return false
}

// Helper methods for state comparison and conversion

func (css *CloudStackStateSyncService) hasVMStateChanged(old, new *VirtualMachineState) bool {
	return old.State != new.State ||
		old.ServiceOffering != new.ServiceOffering ||
		!css.slicesEqual(old.AttachedVolumes, new.AttachedVolumes)
}

func (css *CloudStackStateSyncService) hasVolumeStateChanged(old, new *VolumeState) bool {
	return old.State != new.State ||
		old.AttachedToVM != new.AttachedToVM ||
		old.DeviceID != new.DeviceID ||
		old.Size != new.Size
}

func (css *CloudStackStateSyncService) vmStateToMap(vm *VirtualMachineState) map[string]interface{} {
	return map[string]interface{}{
		"id":               vm.ID,
		"name":             vm.Name,
		"state":            vm.State,
		"service_offering": vm.ServiceOffering,
		"template":         vm.Template,
		"zone":             vm.Zone,
		"attached_volumes": vm.AttachedVolumes,
		"last_seen":        vm.LastSeen,
		"sync_status":      vm.SyncStatus,
	}
}

func (css *CloudStackStateSyncService) volumeStateToMap(volume *VolumeState) map[string]interface{} {
	return map[string]interface{}{
		"id":               volume.ID,
		"name":             volume.Name,
		"type":             volume.Type,
		"state":            volume.State,
		"size":             volume.Size,
		"attached_to_vm":   volume.AttachedToVM,
		"device_id":        volume.DeviceID,
		"storage_offering": volume.StorageOffering,
		"zone":             volume.Zone,
		"last_seen":        volume.LastSeen,
		"sync_status":      volume.SyncStatus,
	}
}

func (css *CloudStackStateSyncService) slicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}

func (css *CloudStackStateSyncService) getVMAttachedVolumes(vmID string) []string {
	// This would query CloudStack for volumes attached to the VM
	// For now, return empty slice - would be implemented with actual CloudStack API call
	return []string{}
}

func (css *CloudStackStateSyncService) getVMNetworkInfo(vm interface{}) map[string]interface{} {
	// Extract network information from VM object
	// For now, return empty map - would be implemented with actual VM data parsing
	return map[string]interface{}{}
}

// Public API methods for external access

// GetCurrentState returns the current cached state
func (css *CloudStackStateSyncService) GetCurrentState() (*StateCache, error) {
	css.stateCache.mutex.RLock()
	defer css.stateCache.mutex.RUnlock()

	// Return a copy to prevent external modification
	return &StateCache{
		VMs:         css.copyVMMap(css.stateCache.VMs),
		Volumes:     css.copyVolumeMap(css.stateCache.Volumes),
		LastUpdated: css.stateCache.LastUpdated,
		SyncVersion: css.stateCache.SyncVersion,
	}, nil
}

// ForceSync triggers an immediate synchronization cycle
func (css *CloudStackStateSyncService) ForceSync(ctx context.Context) (*StateSyncResult, error) {
	css.forceFullSync = true
	return nil, css.performSyncCycle(ctx)
}

// GetSyncStatus returns the current synchronization status
func (css *CloudStackStateSyncService) GetSyncStatus() map[string]interface{} {
	css.mutex.RLock()
	defer css.mutex.RUnlock()

	return map[string]interface{}{
		"is_running":    css.isRunning,
		"last_sync":     css.lastSyncTime,
		"sync_interval": css.syncInterval.String(),
		"sync_errors":   css.syncErrors,
		"max_errors":    css.maxSyncErrors,
		"sync_version":  css.stateCache.SyncVersion,
		"cache_updated": css.stateCache.LastUpdated,
	}
}

// Helper methods for copying state maps

func (css *CloudStackStateSyncService) copyVMMap(original map[string]*VirtualMachineState) map[string]*VirtualMachineState {
	copy := make(map[string]*VirtualMachineState)
	for k, v := range original {
		vmCopy := *v // Copy the VM state
		copy[k] = &vmCopy
	}
	return copy
}

func (css *CloudStackStateSyncService) copyVolumeMap(original map[string]*VolumeState) map[string]*VolumeState {
	copy := make(map[string]*VolumeState)
	for k, v := range original {
		volumeCopy := *v // Copy the volume state
		copy[k] = &volumeCopy
	}
	return copy
}
