// Package services provides database-based VM validation for failover operations
package services

import (
	"fmt"

	"github.com/vexxhost/migratekit-oma/database"

	log "github.com/sirupsen/logrus"
)

// Note: SourceNetworkInfo is defined in network_mapping_service.go to avoid duplication

// SimpleDatabaseVMInfoService provides basic VM validation from database
type SimpleDatabaseVMInfoService struct {
	db database.Connection
}

// NewSimpleDatabaseVMInfoService creates a simple database-based VM info service
func NewSimpleDatabaseVMInfoService(db database.Connection) *SimpleDatabaseVMInfoService {
	return &SimpleDatabaseVMInfoService{
		db: db,
	}
}

// ValidateVMExists validates that a VM exists in the database (has completed migration)
func (sdvis *SimpleDatabaseVMInfoService) ValidateVMExists(vmID string) error {
	log.WithField("vm_id", vmID).Info("🔍 Validating VM exists in database (has completed migration)")

	// Check if we have any completed replication jobs OR high-progress jobs for this VM
	var completedJobCount int64
	err := sdvis.db.GetGormDB().Model(&database.ReplicationJob{}).
		Where("source_vm_id = ? AND status = ?", vmID, "completed").Count(&completedJobCount).Error
	if err != nil {
		return fmt.Errorf("failed to query completed jobs for VM %s: %w", vmID, err)
	}

	// If no completed jobs, check for high-progress replicating jobs (95%+)
	if completedJobCount == 0 {
		var highProgressJobCount int64
		err := sdvis.db.GetGormDB().Model(&database.ReplicationJob{}).
			Where("source_vm_id = ? AND status = ? AND progress_percent >= ?", vmID, "replicating", 95.0).
			Count(&highProgressJobCount).Error
		if err != nil {
			return fmt.Errorf("failed to query high-progress jobs for VM %s: %w", vmID, err)
		}

		if highProgressJobCount == 0 {
			return fmt.Errorf("VM %s has no completed migration - failover requires completed replication job or 95%+ progress", vmID)
		}

		log.WithFields(log.Fields{
			"vm_id":                   vmID,
			"high_progress_job_count": highProgressJobCount,
		}).Info("✅ VM validated - has high-progress migration (95%+) ready for failover")
		return nil
	}

	log.WithFields(log.Fields{
		"vm_id":               vmID,
		"completed_job_count": completedJobCount,
	}).Info("✅ VM validated - has completed migration data in database")

	return nil
}

// GetVMNetworkConfiguration returns basic network info (placeholder implementation)
func (sdvis *SimpleDatabaseVMInfoService) GetVMNetworkConfiguration(vmID string) ([]SourceNetworkInfo, error) {
	log.WithField("vm_id", vmID).Info("🔍 Getting basic VM network configuration from database")

	// For now, return basic default network configuration
	// This will be enhanced once VM specifications are properly stored
	return []SourceNetworkInfo{
		{
			NetworkName:     "VM Network",
			AdapterType:     "vmxnet3",
			MACAddress:      "00:50:56:xx:xx:xx",
			ConnectionState: "connected",
			VLANID:          "",
			IPConfiguration: "dhcp",
		},
	}, nil
}

// GetVMDetails returns basic VM details from database that can be used to build specifications
// This method accepts either VM name or VMware VM ID and attempts both lookup strategies
func (sdvis *SimpleDatabaseVMInfoService) GetVMDetails(vmID string) (map[string]interface{}, error) {
	log.WithField("vm_id", vmID).Info("🔍 Getting VM details from database")

	// Get the latest VM disk record which contains VM specs
	var vmDisk database.VMDisk
	
	// First try: lookup by source_vm_name (for VM names like "pgtest1")
	err := sdvis.db.GetGormDB().
		Where("job_id IN (SELECT id FROM replication_jobs WHERE source_vm_name = ?)", vmID).
		Order("created_at DESC").
		First(&vmDisk).Error

	if err != nil {
		// Second try: lookup by source_vm_id (for VMware UUIDs)
		err = sdvis.db.GetGormDB().
			Where("job_id IN (SELECT id FROM replication_jobs WHERE source_vm_id = ?)", vmID).
			Order("created_at DESC").
			First(&vmDisk).Error
		
		if err != nil {
			return nil, fmt.Errorf("failed to get VM specifications for %s: %w", vmID, err)
		}
	}

	// Get replication job to get VM name
	var job database.ReplicationJob
	err = sdvis.db.GetGormDB().Where("source_vm_id = ?", vmID).
		Order("created_at DESC").First(&job).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get replication job for VM %s: %w", vmID, err)
	}

	// Return VM details as a map to avoid circular import
	details := map[string]interface{}{
		"name":              job.SourceVMName,
		"display_name":      vmDisk.DisplayName,
		"cpu_count":         vmDisk.CPUCount,
		"memory_mb":         vmDisk.MemoryMB,
		"os_type":           vmDisk.OSType,
		"power_state":       vmDisk.PowerState,
		"annotation":        vmDisk.Annotation,
		"network_config":    vmDisk.NetworkConfig,
		"vmdk_path":         vmDisk.VMDKPath,
		"capacity_bytes":    vmDisk.CapacityBytes,
		"provisioning_type": vmDisk.ProvisioningType,
	}

	log.WithFields(log.Fields{
		"vm_id":     vmID,
		"vm_name":   details["name"],
		"cpu_count": details["cpu_count"],
		"memory_mb": details["memory_mb"],
	}).Info("✅ VM details retrieved from database")

	return details, nil
}

// GetVMDetailsByContextID returns VM details using VM-centric architecture context_id
func (sdvis *SimpleDatabaseVMInfoService) GetVMDetailsByContextID(contextID string) (map[string]interface{}, error) {
	log.WithField("context_id", contextID).Info("🔍 Getting VM details by context ID from database")

	// Get the latest VM disk record using VM-centric architecture
	var vmDisk database.VMDisk
	err := sdvis.db.GetGormDB().
		Where("vm_context_id = ?", contextID).
		Order("created_at DESC").
		First(&vmDisk).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get VM specifications for context %s: %w", contextID, err)
	}

	// Build VM details map with all available information
	details := map[string]interface{}{
		"name":         vmDisk.DisplayName,
		"display_name": vmDisk.DisplayName,
		"cpu_count":    vmDisk.CPUCount,
		"memory_mb":    vmDisk.MemoryMB,
		"os_type":      vmDisk.OSType,
		"power_state":  vmDisk.PowerState,
		"vmware_uuid":  vmDisk.VMwareUUID,
		"disk_id":      vmDisk.DiskID,
		"size_gb":      vmDisk.SizeGB,
		"datastore":    vmDisk.Datastore,
		"context_id":   contextID,
	}

	log.WithFields(log.Fields{
		"context_id":   contextID,
		"vm_name":      details["name"],
		"cpu_count":    details["cpu_count"],
		"memory_mb":    details["memory_mb"],
		"os_type":      details["os_type"],
	}).Info("✅ VM details retrieved successfully using context ID")

	return details, nil
}
