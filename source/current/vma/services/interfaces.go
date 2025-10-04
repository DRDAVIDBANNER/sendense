// Package services provides service interfaces for VMA components
// This package breaks import cycles by defining interfaces that can be implemented by other packages
package services

import (
	"context"
	"time"
)

// VMSpecificationChecker defines the interface for VM specification change detection
type VMSpecificationChecker interface {
	// DetectVMSpecificationChanges compares current vCenter specs with stored VM data
	DetectVMSpecificationChanges(ctx context.Context, vmPath string, storedVMInfo *StoredVMInfo) (*VMSpecificationDiff, error)

	// GetChangesSummary returns a human-readable summary of changes
	GetChangesSummary(diff *VMSpecificationDiff) string

	// SerializeChanges converts the diff to JSON for storage/logging
	SerializeChanges(diff *VMSpecificationDiff) (string, error)
}

// VMwareDiscoveryProvider defines the interface for creating VMware discovery connections
type VMwareDiscoveryProvider interface {
	// CreateDiscovery creates a new VMware discovery service with the given credentials
	CreateDiscovery(vcenter, username, password, datacenter string) (VMwareDiscovery, error)
}

// VMwareDiscovery defines the interface for VMware operations
type VMwareDiscovery interface {
	// Connect establishes connection to vCenter
	Connect(ctx context.Context) error

	// Disconnect closes vCenter connection
	Disconnect()

	// GetVMDetails gets detailed information for a specific VM by path
	GetVMDetails(ctx context.Context, vmPath string) (*StoredVMInfo, error)
}

// StoredVMInfo represents VM information that can be stored and compared
type StoredVMInfo struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Path       string `json:"path"`
	Datacenter string `json:"datacenter"`
	CPUs       int    `json:"cpus"`
	MemoryMB   int    `json:"memory_mb"`
	PowerState string `json:"power_state"`
	OSType     string `json:"os_type"`
	VMXVersion string `json:"vmx_version"`

	// Additional VM metadata
	DisplayName        string `json:"display_name"`
	Annotation         string `json:"annotation"`
	FolderPath         string `json:"folder_path"`
	VMwareToolsStatus  string `json:"vmware_tools_status"`
	VMwareToolsVersion string `json:"vmware_tools_version"`

	Disks    []StoredDiskInfo    `json:"disks"`
	Networks []StoredNetworkInfo `json:"networks"`
}

// StoredDiskInfo represents VM disk information for comparison
type StoredDiskInfo struct {
	ID               string `json:"id"`
	Path             string `json:"path"`
	SizeGB           int    `json:"size_gb"`
	Datastore        string `json:"datastore"`
	VMDKPath         string `json:"vmdk_path"`
	ProvisioningType string `json:"provisioning_type"`
	Label            string `json:"label"`
	CapacityBytes    int64  `json:"capacity_bytes"`
	UnitNumber       int    `json:"unit_number"`
}

// StoredNetworkInfo represents VM network interface information for comparison
type StoredNetworkInfo struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Connected   bool   `json:"connected"`
	MACAddress  string `json:"mac_address"`
	Label       string `json:"label"`
	NetworkName string `json:"network_name"`
	AdapterType string `json:"adapter_type"`
}

// VMSpecificationDiff represents changes detected in VM configuration
type VMSpecificationDiff struct {
	HasChanges         bool                   `json:"has_changes"`
	VMID               string                 `json:"vm_id"`
	VMName             string                 `json:"vm_name"`
	CPUChanges         *FieldChange           `json:"cpu_changes,omitempty"`
	MemoryChanges      *FieldChange           `json:"memory_changes,omitempty"`
	NetworkChanges     []NetworkAdapterChange `json:"network_changes,omitempty"`
	PowerStateChange   *FieldChange           `json:"power_state_change,omitempty"`
	VMwareToolsChanges *FieldChange           `json:"vmware_tools_changes,omitempty"`
	DisplayNameChange  *FieldChange           `json:"display_name_change,omitempty"`
	AnnotationChange   *FieldChange           `json:"annotation_change,omitempty"`
	FolderPathChange   *FieldChange           `json:"folder_path_change,omitempty"`
	LastChecked        time.Time              `json:"last_checked"`
}

// FieldChange represents a change in a specific field
type FieldChange struct {
	Field    string      `json:"field"`
	OldValue interface{} `json:"old_value"`
	NewValue interface{} `json:"new_value"`
}

// NetworkAdapterChange represents changes in network adapter configuration
type NetworkAdapterChange struct {
	AdapterIndex int         `json:"adapter_index"`
	ChangeType   string      `json:"change_type"` // "modified", "added", "removed"
	Field        string      `json:"field,omitempty"`
	OldValue     interface{} `json:"old_value,omitempty"`
	NewValue     interface{} `json:"new_value,omitempty"`
}




