// Package services provides database-based VM information service for failover operations
package services

import (
	"encoding/json"
	"fmt"

	"github.com/vexxhost/migratekit-oma/database"
	"github.com/vexxhost/migratekit-oma/models"

	log "github.com/sirupsen/logrus"
)

// DatabaseVMInfoService provides VM information from database (from completed migrations)
type DatabaseVMInfoService struct {
	db         database.Connection
	vmDiskRepo *database.VMDiskRepository
}

// NewDatabaseVMInfoService creates a new database-based VM info service
func NewDatabaseVMInfoService(db database.Connection) *DatabaseVMInfoService {
	return &DatabaseVMInfoService{
		db:         db,
		vmDiskRepo: database.NewVMDiskRepository(db),
	}
}

// GetVMNetworkConfiguration retrieves VM network configuration from database
func (dvis *DatabaseVMInfoService) GetVMNetworkConfiguration(vmID string) ([]SourceNetworkInfo, error) {
	log.WithField("vm_id", vmID).Info("🔍 Getting VM network configuration from database")

	// Find the most recent completed replication job for this VM
	job, err := dvis.getMostRecentCompletedJob(vmID)
	if err != nil {
		return nil, fmt.Errorf("failed to find completed replication job for VM %s: %w", vmID, err)
	}

	// Get VM disks to find network configuration
	vmDisks, err := dvis.vmDiskRepo.GetByJobID(job.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get VM disks for job %s: %w", job.ID, err)
	}

	if len(vmDisks) == 0 {
		return nil, fmt.Errorf("no VM disks found for job %s", job.ID)
	}

	// Network config is stored in the first disk record
	firstDisk := vmDisks[0]
	if firstDisk.NetworkConfig == "" {
		log.WithField("vm_id", vmID).Warn("⚠️ No network configuration stored in database - using defaults")
		// Return default network configuration
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

	// Parse stored network configuration
	var networks []models.NetworkInfo
	if err := json.Unmarshal([]byte(firstDisk.NetworkConfig), &networks); err != nil {
		return nil, fmt.Errorf("failed to parse network configuration for VM %s: %w", vmID, err)
	}

	// Convert to SourceNetworkInfo format
	var sourceNetworks []SourceNetworkInfo
	for _, network := range networks {
		sourceNetwork := SourceNetworkInfo{
			NetworkName:     network.NetworkName,
			AdapterType:     network.AdapterType,
			MACAddress:      network.MACAddress,
			ConnectionState: getConnectionState(network.Connected),
			VLANID:          "",           // VLAN ID not currently stored
			IPConfiguration: "configured", // Default since IP details not stored
		}
		sourceNetworks = append(sourceNetworks, sourceNetwork)
	}

	log.WithFields(log.Fields{
		"vm_id":         vmID,
		"job_id":        job.ID,
		"network_count": len(sourceNetworks),
	}).Info("✅ Retrieved VM network configuration from database")

	return sourceNetworks, nil
}

// ValidateVMExists validates that a VM exists in the database (has completed migration)
func (dvis *DatabaseVMInfoService) ValidateVMExists(vmID string) error {
	log.WithField("vm_id", vmID).Debug("Validating VM exists in database")

	// Check if we have any replication jobs for this VM using direct DB query
	var jobCount int64
	err := dvis.db.GetGormDB().Model(&database.ReplicationJob{}).
		Where("source_vm_id = ?", vmID).Count(&jobCount).Error
	if err != nil {
		return fmt.Errorf("failed to query replication jobs for VM %s: %w", vmID, err)
	}

	if jobCount == 0 {
		return fmt.Errorf("VM %s not found in database - no replication jobs exist", vmID)
	}

	// Check if at least one job completed successfully  
	var completedJobCount int64
	err = dvis.db.GetGormDB().Model(&database.ReplicationJob{}).
		Where("source_vm_id = ? AND status = ?", vmID, "completed").Count(&completedJobCount).Error
	if err != nil {
		return fmt.Errorf("failed to query completed jobs for VM %s: %w", vmID, err)
	}

	if completedJobCount == 0 {
		return fmt.Errorf("VM %s has no completed replication jobs - failover requires completed migration", vmID)
	}

	log.WithField("vm_id", vmID).Debug("✅ VM exists and has completed migration")
	return nil
}

// GetVMSpecification is a placeholder - will be implemented when failover types are available
func (dvis *DatabaseVMInfoService) GetVMSpecification(vmID string) (interface{}, error) {
	return nil, fmt.Errorf("VM specification retrieval not yet implemented")
}

// Helper methods

// getMostRecentCompletedJob finds the most recent completed replication job for a VM
func (dvis *DatabaseVMInfoService) getMostRecentCompletedJob(vmID string) (*database.ReplicationJob, error) {
	var job database.ReplicationJob
	err := dvis.db.GetGormDB().Where("source_vm_id = ? AND status = ?", vmID, "completed").
		Order("created_at DESC").First(&job).Error
	if err != nil {
		return nil, fmt.Errorf("no completed replication job found for VM %s: %w", vmID, err)
	}
	return &job, nil
}

// getConnectionStateDB converts boolean connected state to string
func getConnectionStateDB(connected bool) string {
	if connected {
		return "connected"
	}
	return "disconnected"
}
