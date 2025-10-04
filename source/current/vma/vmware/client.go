// Package vmware provides VMware client implementations for the VMA API
package vmware

import (
	"context"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/vexxhost/migratekit/source/current/vma/api"
	"github.com/vexxhost/migratekit/source/current/vma/client"
)

// RealVMwareClient provides real VMware operations for the VMA API
type RealVMwareClient struct {
	service *Service
}

// NewRealVMwareClient creates a new real VMware client
func NewRealVMwareClient(omaClient *client.Client) *RealVMwareClient {
	return &RealVMwareClient{
		service: NewService(omaClient),
	}
}

// NewRealVMwareClientWithConfig creates a new real VMware client with custom configuration
func NewRealVMwareClientWithConfig(omaClient *client.Client, config ServiceConfig) *RealVMwareClient {
	return &RealVMwareClient{
		service: NewServiceWithConfig(omaClient, config),
	}
}

// DeleteSnapshot deletes a VMware snapshot (not yet implemented)
func (c *RealVMwareClient) DeleteSnapshot(jobID string) error {
	log.WithField("job_id", jobID).Warn("VMware snapshot deletion not yet implemented - manual cleanup required")
	return fmt.Errorf("snapshot deletion not implemented - manual cleanup required for job %s", jobID)
}

// GetVMStatus gets the status of a VM (basic implementation)
func (c *RealVMwareClient) GetVMStatus(vmPath string) (string, error) {
	// For now, return running - real implementation would query vCenter
	log.WithField("vm_path", vmPath).Debug("Getting VM status (basic implementation)")
	return "running", nil
}

// DiscoverVMs discovers VMs from vCenter and returns them in API format
func (c *RealVMwareClient) DiscoverVMs(vcenter, username, password, datacenter string) (*api.VMInventory, error) {
	return c.DiscoverVMsWithFilter(vcenter, username, password, datacenter, "")
}

// DiscoverVMsWithFilter discovers VMs from vCenter with filtering and returns them in API format
func (c *RealVMwareClient) DiscoverVMsWithFilter(vcenter, username, password, datacenter, filter string) (*api.VMInventory, error) {
	log.WithFields(log.Fields{
		"vcenter":    vcenter,
		"datacenter": datacenter,
		"filter":     filter,
	}).Info("Discovering VMs from vCenter via API with filter")

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Use the service to discover VMs with filter (this will apply path correction)
	inventory, err := c.service.DiscoverVMsFromVCenter(ctx, vcenter, username, password, datacenter, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to discover VMs from vCenter: %w", err)
	}

	// Convert from models.VMInventoryRequest to api.VMInventory
	apiInventory := &api.VMInventory{
		VCenter: struct {
			Host       string `json:"host"`
			Datacenter string `json:"datacenter"`
		}{
			Host:       inventory.VCenter.Host,
			Datacenter: inventory.VCenter.Datacenter,
		},
		VMs: make([]api.VMInfo, len(inventory.VMs)),
	}

	// Convert VM info with complete disk and network details
	for i, vm := range inventory.VMs {
		// Convert disks
		apiDisks := make([]api.DiskInfo, len(vm.Disks))
		for j, disk := range vm.Disks {
			apiDisks[j] = api.DiskInfo{
				ID:               disk.ID,
				Label:            disk.Label,
				Path:             disk.Path,
				VMDKPath:         disk.VMDKPath,
				SizeGB:           disk.SizeGB,
				CapacityBytes:    disk.CapacityBytes,
				Datastore:        disk.Datastore,
				ProvisioningType: disk.ProvisioningType,
				UnitNumber:       disk.UnitNumber,
			}
		}

		// Convert networks
		apiNetworks := make([]api.NetworkInfo, len(vm.Networks))
		for j, network := range vm.Networks {
			apiNetworks[j] = api.NetworkInfo{
				Label:       network.Label,
				NetworkName: network.NetworkName,
				AdapterType: network.AdapterType,
				MACAddress:  network.MACAddress,
				Connected:   network.Connected,
			}
		}

		apiInventory.VMs[i] = api.VMInfo{
			ID:         vm.ID,
			Name:       vm.Name,
			Path:       vm.Path,
			Datacenter: vm.Datacenter,
			PowerState: vm.PowerState,
			GuestOS:    vm.OSType,
			MemoryMB:   vm.MemoryMB,
			NumCPU:     vm.CPUs,
			VMXVersion: vm.VMXVersion,
			Disks:      apiDisks,
			Networks:   apiNetworks,
		}
	}

	log.WithFields(log.Fields{
		"vm_count":     len(apiInventory.VMs),
		"vcenter_host": apiInventory.VCenter.Host,
		"datacenter":   apiInventory.VCenter.Datacenter,
	}).Info("VM discovery completed successfully via API")

	return apiInventory, nil
}

// StartReplication starts replication of specific VMs
func (c *RealVMwareClient) StartReplication(request *api.ReplicationRequest) (*api.ReplicationResponse, error) {
	log.WithFields(log.Fields{
		"job_id":   request.JobID,
		"vcenter":  request.VCenter,
		"vm_count": len(request.VMPaths),
	}).Info("Starting replication via API")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Start the replication job using the service
	err := c.service.StartReplicationJob(ctx, request.JobID, request.VCenter, request.Username, request.Password, request.VMPaths, request.NBDTargets)
	if err != nil {
		return nil, fmt.Errorf("failed to start replication: %w", err)
	}

	// Create response
	response := &api.ReplicationResponse{
		JobID:     request.JobID,
		Status:    "started",
		VMCount:   len(request.VMPaths),
		StartedAt: time.Now().UTC().Format(time.RFC3339),
	}

	log.WithFields(log.Fields{
		"job_id":   response.JobID,
		"status":   response.Status,
		"vm_count": response.VMCount,
	}).Info("Replication started successfully via API")

	return response, nil
}
