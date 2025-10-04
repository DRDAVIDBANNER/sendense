// Package vmware provides VMware service adapters that implement VMA service interfaces
package vmware

import (
	"context"

	"github.com/vexxhost/migratekit/internal/oma/models"
	"github.com/vexxhost/migratekit/source/current/vma/services"
)

// SpecificationServiceAdapter adapts VMSpecificationService to the services interface
type SpecificationServiceAdapter struct {
	discovery *Discovery
	service   *VMSpecificationService
}

// NewSpecificationServiceAdapter creates a new specification service adapter
func NewSpecificationServiceAdapter(discovery *Discovery) *SpecificationServiceAdapter {
	return &SpecificationServiceAdapter{
		discovery: discovery,
		service:   NewVMSpecificationService(discovery),
	}
}

// DetectVMSpecificationChanges implements services.VMSpecificationChecker
func (a *SpecificationServiceAdapter) DetectVMSpecificationChanges(ctx context.Context, vmPath string, storedVMInfo *services.StoredVMInfo) (*services.VMSpecificationDiff, error) {
	// Convert services.StoredVMInfo to models.VMInfo
	modelVMInfo := a.convertStoredVMInfoToModel(storedVMInfo)

	// Use the existing service to detect changes
	diff, err := a.service.DetectVMSpecificationChanges(ctx, vmPath, modelVMInfo)
	if err != nil {
		return nil, err
	}

	// Convert back to services.VMSpecificationDiff
	return a.convertDiffToServices(diff), nil
}

// GetChangesSummary implements services.VMSpecificationChecker
func (a *SpecificationServiceAdapter) GetChangesSummary(diff *services.VMSpecificationDiff) string {
	// Convert to internal diff format
	internalDiff := a.convertServicesaDiffToInternal(diff)
	return a.service.GetChangesSummary(internalDiff)
}

// SerializeChanges implements services.VMSpecificationChecker
func (a *SpecificationServiceAdapter) SerializeChanges(diff *services.VMSpecificationDiff) (string, error) {
	// Convert to internal diff format
	internalDiff := a.convertServicesaDiffToInternal(diff)
	return a.service.SerializeChanges(internalDiff)
}

// convertStoredVMInfoToModel converts services.StoredVMInfo to models.VMInfo
func (a *SpecificationServiceAdapter) convertStoredVMInfoToModel(stored *services.StoredVMInfo) *models.VMInfo {
	// Convert disks
	var disks []models.DiskInfo
	for _, disk := range stored.Disks {
		disks = append(disks, models.DiskInfo{
			ID:               disk.ID,
			Path:             disk.Path,
			SizeGB:           disk.SizeGB,
			Datastore:        disk.Datastore,
			VMDKPath:         disk.VMDKPath,
			ProvisioningType: disk.ProvisioningType,
			Label:            disk.Label,
			CapacityBytes:    disk.CapacityBytes,
			UnitNumber:       disk.UnitNumber,
		})
	}

	// Convert networks
	var networks []models.NetworkInfo
	for _, net := range stored.Networks {
		networks = append(networks, models.NetworkInfo{
			Name:        net.Name,
			Type:        net.Type,
			Connected:   net.Connected,
			MACAddress:  net.MACAddress,
			Label:       net.Label,
			NetworkName: net.NetworkName,
			AdapterType: net.AdapterType,
		})
	}

	return &models.VMInfo{
		ID:                 stored.ID,
		Name:               stored.Name,
		Path:               stored.Path,
		Datacenter:         stored.Datacenter,
		CPUs:               stored.CPUs,
		MemoryMB:           stored.MemoryMB,
		PowerState:         stored.PowerState,
		OSType:             stored.OSType,
		VMXVersion:         stored.VMXVersion,
		DisplayName:        stored.DisplayName,
		Annotation:         stored.Annotation,
		FolderPath:         stored.FolderPath,
		VMwareToolsStatus:  stored.VMwareToolsStatus,
		VMwareToolsVersion: stored.VMwareToolsVersion,
		Disks:              disks,
		Networks:           networks,
	}
}

// convertDiffToServices converts internal VMSpecificationDiff to services.VMSpecificationDiff
func (a *SpecificationServiceAdapter) convertDiffToServices(internal *VMSpecificationDiff) *services.VMSpecificationDiff {
	serviceDiff := &services.VMSpecificationDiff{
		HasChanges:  internal.HasChanges,
		VMID:        internal.VMID,
		VMName:      internal.VMName,
		LastChecked: internal.LastChecked,
	}

	// Convert field changes
	if internal.CPUChanges != nil {
		serviceDiff.CPUChanges = &services.FieldChange{
			Field:    internal.CPUChanges.Field,
			OldValue: internal.CPUChanges.OldValue,
			NewValue: internal.CPUChanges.NewValue,
		}
	}

	if internal.MemoryChanges != nil {
		serviceDiff.MemoryChanges = &services.FieldChange{
			Field:    internal.MemoryChanges.Field,
			OldValue: internal.MemoryChanges.OldValue,
			NewValue: internal.MemoryChanges.NewValue,
		}
	}

	if internal.PowerStateChange != nil {
		serviceDiff.PowerStateChange = &services.FieldChange{
			Field:    internal.PowerStateChange.Field,
			OldValue: internal.PowerStateChange.OldValue,
			NewValue: internal.PowerStateChange.NewValue,
		}
	}

	if internal.VMwareToolsChanges != nil {
		serviceDiff.VMwareToolsChanges = &services.FieldChange{
			Field:    internal.VMwareToolsChanges.Field,
			OldValue: internal.VMwareToolsChanges.OldValue,
			NewValue: internal.VMwareToolsChanges.NewValue,
		}
	}

	if internal.DisplayNameChange != nil {
		serviceDiff.DisplayNameChange = &services.FieldChange{
			Field:    internal.DisplayNameChange.Field,
			OldValue: internal.DisplayNameChange.OldValue,
			NewValue: internal.DisplayNameChange.NewValue,
		}
	}

	if internal.AnnotationChange != nil {
		serviceDiff.AnnotationChange = &services.FieldChange{
			Field:    internal.AnnotationChange.Field,
			OldValue: internal.AnnotationChange.OldValue,
			NewValue: internal.AnnotationChange.NewValue,
		}
	}

	if internal.FolderPathChange != nil {
		serviceDiff.FolderPathChange = &services.FieldChange{
			Field:    internal.FolderPathChange.Field,
			OldValue: internal.FolderPathChange.OldValue,
			NewValue: internal.FolderPathChange.NewValue,
		}
	}

	// Convert network changes
	for _, netChange := range internal.NetworkChanges {
		serviceDiff.NetworkChanges = append(serviceDiff.NetworkChanges, services.NetworkAdapterChange{
			AdapterIndex: netChange.AdapterIndex,
			ChangeType:   netChange.ChangeType,
			Field:        netChange.Field,
			OldValue:     netChange.OldValue,
			NewValue:     netChange.NewValue,
		})
	}

	return serviceDiff
}

// convertServicesaDiffToInternal converts services.VMSpecificationDiff back to internal format
func (a *SpecificationServiceAdapter) convertServicesaDiffToInternal(serviceDiff *services.VMSpecificationDiff) *VMSpecificationDiff {
	internal := &VMSpecificationDiff{
		HasChanges:  serviceDiff.HasChanges,
		VMID:        serviceDiff.VMID,
		VMName:      serviceDiff.VMName,
		LastChecked: serviceDiff.LastChecked,
	}

	// Convert field changes back
	if serviceDiff.CPUChanges != nil {
		internal.CPUChanges = &FieldChange{
			Field:    serviceDiff.CPUChanges.Field,
			OldValue: serviceDiff.CPUChanges.OldValue,
			NewValue: serviceDiff.CPUChanges.NewValue,
		}
	}

	if serviceDiff.MemoryChanges != nil {
		internal.MemoryChanges = &FieldChange{
			Field:    serviceDiff.MemoryChanges.Field,
			OldValue: serviceDiff.MemoryChanges.OldValue,
			NewValue: serviceDiff.MemoryChanges.NewValue,
		}
	}

	// Add other conversions as needed...

	return internal
}

// DiscoveryProviderAdapter adapts VMware discovery to the services interface
type DiscoveryProviderAdapter struct{}

// NewDiscoveryProviderAdapter creates a new discovery provider adapter
func NewDiscoveryProviderAdapter() *DiscoveryProviderAdapter {
	return &DiscoveryProviderAdapter{}
}

// CreateDiscovery implements services.VMwareDiscoveryProvider
func (p *DiscoveryProviderAdapter) CreateDiscovery(vcenter, username, password, datacenter string) (services.VMwareDiscovery, error) {
	config := Config{
		Host:       vcenter,
		Username:   username,
		Password:   password,
		Datacenter: datacenter,
		Insecure:   true,
	}

	discovery := NewDiscovery(config)
	return &DiscoveryAdapter{discovery: discovery}, nil
}

// DiscoveryAdapter adapts Discovery to the services interface
type DiscoveryAdapter struct {
	discovery *Discovery
}

// Connect implements services.VMwareDiscovery
func (d *DiscoveryAdapter) Connect(ctx context.Context) error {
	return d.discovery.Connect(ctx)
}

// Disconnect implements services.VMwareDiscovery
func (d *DiscoveryAdapter) Disconnect() {
	d.discovery.Disconnect()
}

// GetVMDetails implements services.VMwareDiscovery
func (d *DiscoveryAdapter) GetVMDetails(ctx context.Context, vmPath string) (*services.StoredVMInfo, error) {
	vmInfo, err := d.discovery.GetVMDetails(ctx, vmPath)
	if err != nil {
		return nil, err
	}

	// Convert models.VMInfo to services.StoredVMInfo
	return d.convertModelToStoredVMInfo(vmInfo), nil
}

// convertModelToStoredVMInfo converts models.VMInfo to services.StoredVMInfo
func (d *DiscoveryAdapter) convertModelToStoredVMInfo(model *models.VMInfo) *services.StoredVMInfo {
	// Convert disks
	var disks []services.StoredDiskInfo
	for _, disk := range model.Disks {
		disks = append(disks, services.StoredDiskInfo{
			ID:               disk.ID,
			Path:             disk.Path,
			SizeGB:           disk.SizeGB,
			Datastore:        disk.Datastore,
			VMDKPath:         disk.VMDKPath,
			ProvisioningType: disk.ProvisioningType,
			Label:            disk.Label,
			CapacityBytes:    disk.CapacityBytes,
			UnitNumber:       disk.UnitNumber,
		})
	}

	// Convert networks
	var networks []services.StoredNetworkInfo
	for _, net := range model.Networks {
		networks = append(networks, services.StoredNetworkInfo{
			Name:        net.Name,
			Type:        net.Type,
			Connected:   net.Connected,
			MACAddress:  net.MACAddress,
			Label:       net.Label,
			NetworkName: net.NetworkName,
			AdapterType: net.AdapterType,
		})
	}

	return &services.StoredVMInfo{
		ID:                 model.ID,
		Name:               model.Name,
		Path:               model.Path,
		Datacenter:         model.Datacenter,
		CPUs:               model.CPUs,
		MemoryMB:           model.MemoryMB,
		PowerState:         model.PowerState,
		OSType:             model.OSType,
		VMXVersion:         model.VMXVersion,
		DisplayName:        model.DisplayName,
		Annotation:         model.Annotation,
		FolderPath:         model.FolderPath,
		VMwareToolsStatus:  model.VMwareToolsStatus,
		VMwareToolsVersion: model.VMwareToolsVersion,
		Disks:              disks,
		Networks:           networks,
	}
}
