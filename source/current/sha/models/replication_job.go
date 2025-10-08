package models

import "time"

// ReplicationJob represents a VM replication job with dynamic allocation
type ReplicationJob struct {
	ID               string  `json:"id" binding:"required"`
	SourceVM         VMInfo  `json:"source_vm" binding:"required"`
	TargetNetwork    string  `json:"target_network" binding:"required"`
	ReplicationType  string  `json:"replication_type" binding:"required,oneof=initial incremental"`
	Status           string  `json:"status"`
	Progress         float64 `json:"progress"`
	CurrentOperation string  `json:"current_operation"`

	// Dynamic allocation fields (populated by SHA)
	NBDPort       int    `json:"nbd_port"`
	NBDExportName string `json:"nbd_export_name"`
	TargetDevice  string `json:"target_device"`

	// Tracking fields
	BytesTransferred int64  `json:"bytes_transferred"`
	TotalBytes       int64  `json:"total_bytes"`
	TransferSpeedBps int64  `json:"transfer_speed_bps"`
	ChangeID         string `json:"change_id"`
	ErrorMessage     string `json:"error_message"`

	// Timestamps
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// VMInfo represents VM information from VMware discovery
type VMInfo struct {
	ID         string        `json:"id" binding:"required"`
	Name       string        `json:"name" binding:"required"`
	Path       string        `json:"path" binding:"required"`
	Datacenter string        `json:"datacenter" binding:"required"`
	CPUs       int           `json:"cpus" binding:"required"`
	MemoryMB   int           `json:"memory_mb" binding:"required"`
	Disks      []DiskInfo    `json:"disks" binding:"required"`
	Networks   []NetworkInfo `json:"networks"`
	VMXVersion string        `json:"vmx_version"`
	PowerState string        `json:"power_state"`
	OSType     string        `json:"os_type"`

	// Additional VM metadata for failover system
	DisplayName        string `json:"display_name"`         // VM display name (may differ from name)
	Annotation         string `json:"annotation"`           // VM notes/description
	FolderPath         string `json:"folder_path"`          // vCenter folder path
	VMwareToolsStatus  string `json:"vmware_tools_status"`  // VMware Tools status
	VMwareToolsVersion string `json:"vmware_tools_version"` // VMware Tools version
}

// DiskInfo represents VM disk information
type DiskInfo struct {
	ID               string `json:"id"`
	Path             string `json:"path" binding:"required"`
	SizeGB           int    `json:"size_gb" binding:"required"`
	Datastore        string `json:"datastore" binding:"required"`
	VMDKPath         string `json:"vmdk_path"`
	ProvisioningType string `json:"provisioning_type"`
	Label            string `json:"label"`
	CapacityBytes    int64  `json:"capacity_bytes"`
	UnitNumber       int    `json:"unit_number"`
}

// NetworkInfo represents VM network interface information
type NetworkInfo struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Connected   bool   `json:"connected"`
	MACAddress  string `json:"mac_address"`
	Label       string `json:"label"`
	NetworkName string `json:"network_name"`
	AdapterType string `json:"adapter_type"`
}

// VCenterInfo represents vCenter server information
type VCenterInfo struct {
	Host              string `json:"host"`
	Version           string `json:"version"`
	Datacenter        string `json:"datacenter"`
	TotalVMs          int    `json:"total_vms"`
	ConnectionHealthy bool   `json:"connection_healthy"`
}
