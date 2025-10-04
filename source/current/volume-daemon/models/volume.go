package models

import (
	"time"
)

// VolumeOperation represents a volume operation in progress
type VolumeOperation struct {
	ID          string                 `json:"id" db:"id"`
	Type        VolumeOperationType    `json:"type" db:"type"`
	Status      OperationStatus        `json:"status" db:"status"`
	VolumeID    string                 `json:"volume_id" db:"volume_id"`
	VMID        *string                `json:"vm_id,omitempty" db:"vm_id"`
	Request     map[string]interface{} `json:"request" db:"request"`
	Response    map[string]interface{} `json:"response,omitempty" db:"response"`
	Error       *string                `json:"error,omitempty" db:"error"`
	CreatedAt   time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at" db:"updated_at"`
	CompletedAt *time.Time             `json:"completed_at,omitempty" db:"completed_at"`
}

// VolumeOperationType defines the type of volume operation
type VolumeOperationType string

const (
	OperationCreate  VolumeOperationType = "create"
	OperationAttach  VolumeOperationType = "attach"
	OperationDetach  VolumeOperationType = "detach"
	OperationDelete  VolumeOperationType = "delete"
	OperationCleanup VolumeOperationType = "cleanup"
)

// OperationStatus defines the status of a volume operation
type OperationStatus string

const (
	StatusPending   OperationStatus = "pending"
	StatusExecuting OperationStatus = "executing"
	StatusCompleted OperationStatus = "completed"
	StatusFailed    OperationStatus = "failed"
	StatusCancelled OperationStatus = "cancelled"
)

// OperationMode defines the type of volume operation mode
type OperationMode string

const (
	OperationModeOMA      OperationMode = "oma"
	OperationModeFailover OperationMode = "failover"
)

// DeviceMapping represents a real-time mapping between CloudStack volume and Linux device
type DeviceMapping struct {
	ID                        string    `json:"id" db:"id"`
	VMContextID               *string   `json:"vm_context_id" db:"vm_context_id"`         // VM-Centric Architecture integration
	VolumeUUID                string    `json:"volume_uuid" db:"volume_uuid"`             // Updated to match normalized schema
	VolumeIDNumeric           *int64    `json:"volume_id_numeric" db:"volume_id_numeric"` // CloudStack numeric ID
	VMID                      string    `json:"vm_id" db:"vm_id"`
	OperationMode             string    `json:"operation_mode" db:"operation_mode"`
	CloudStackDeviceID        *int      `json:"cloudstack_device_id" db:"cloudstack_device_id"`
	RequiresDeviceCorrelation bool      `json:"requires_device_correlation" db:"requires_device_correlation"`
	DevicePath                string    `json:"device_path" db:"device_path"`
	CloudStackState           string    `json:"cloudstack_state" db:"cloudstack_state"`
	LinuxState                string    `json:"linux_state" db:"linux_state"`
	Size                      int64     `json:"size" db:"size"`
	LastSync                  time.Time `json:"last_sync" db:"last_sync"`

	// Multi-Volume Snapshot Enhancement: Per-volume snapshot tracking
	OSSEASnapshotID   *string    `json:"ossea_snapshot_id,omitempty" db:"ossea_snapshot_id"`
	SnapshotCreatedAt *time.Time `json:"snapshot_created_at,omitempty" db:"snapshot_created_at"`
	SnapshotStatus    string     `json:"snapshot_status" db:"snapshot_status"`

	// Persistent Device Naming Enhancement: Stable device names for NBD export consistency
	PersistentDeviceName *string `json:"persistent_device_name,omitempty" db:"persistent_device_name"`
	SymlinkPath          *string `json:"symlink_path,omitempty" db:"symlink_path"`

	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// VolumeStatus represents the current status of a volume
type VolumeStatus struct {
	VolumeID       string                 `json:"volume_id"`
	VMID           *string                `json:"vm_id,omitempty"`
	DevicePath     *string                `json:"device_path,omitempty"`
	State          string                 `json:"state"`
	Size           int64                  `json:"size"`
	CloudStackData map[string]interface{} `json:"cloudstack_data"`
	LinuxData      map[string]interface{} `json:"linux_data,omitempty"`
	LastOperation  *VolumeOperation       `json:"last_operation,omitempty"`
}

// CreateVolumeRequest represents a request to create a new volume
type CreateVolumeRequest struct {
	Name           string            `json:"name" validate:"required"`
	Size           int64             `json:"size" validate:"required,min=1"`
	DiskOfferingID string            `json:"disk_offering_id" validate:"required"`
	ZoneID         string            `json:"zone_id" validate:"required"`
	Metadata       map[string]string `json:"metadata,omitempty"`
}

// AttachVolumeRequest represents a request to attach a volume
type AttachVolumeRequest struct {
	VolumeID string `json:"volume_id" validate:"required"`
	VMID     string `json:"vm_id" validate:"required"`
}

// CleanupRequest represents a request to cleanup test failover resources
type CleanupRequest struct {
	TestVMID   string `json:"test_vm_id" validate:"required"`
	VolumeID   string `json:"volume_id" validate:"required"`
	OMAVMID    string `json:"oma_vm_id" validate:"required"`
	DeleteVM   bool   `json:"delete_vm"`
	ForceClean bool   `json:"force_clean"`
}

// OperationFilter represents filter criteria for listing operations
type OperationFilter struct {
	Type     *VolumeOperationType `json:"type,omitempty"`
	Status   *OperationStatus     `json:"status,omitempty"`
	VolumeID *string              `json:"volume_id,omitempty"`
	VMID     *string              `json:"vm_id,omitempty"`
	Since    *time.Time           `json:"since,omitempty"`
	Until    *time.Time           `json:"until,omitempty"`
	Limit    int                  `json:"limit,omitempty"`
}

// HealthStatus represents the health status of the volume daemon
type HealthStatus struct {
	Status           string            `json:"status"`
	Timestamp        time.Time         `json:"timestamp"`
	CloudStackHealth string            `json:"cloudstack_health"`
	DatabaseHealth   string            `json:"database_health"`
	DeviceMonitor    string            `json:"device_monitor"`
	Details          map[string]string `json:"details,omitempty"`
}

// ServiceMetrics represents metrics for the volume daemon
type ServiceMetrics struct {
	Timestamp           time.Time              `json:"timestamp"`
	TotalOperations     int64                  `json:"total_operations"`
	PendingOperations   int64                  `json:"pending_operations"`
	ActiveMappings      int64                  `json:"active_mappings"`
	OperationsByType    map[string]int64       `json:"operations_by_type"`
	OperationsByStatus  map[string]int64       `json:"operations_by_status"`
	AverageResponseTime float64                `json:"average_response_time_ms"`
	ErrorRate           float64                `json:"error_rate_percent"`
	Details             map[string]interface{} `json:"details,omitempty"`
}

// NBD Export Management Models

// NBDExportInfo represents NBD export information
type NBDExportInfo struct {
	ID         string            `json:"id"`
	VolumeID   string            `json:"volume_id"`
	ExportName string            `json:"export_name"`
	DevicePath string            `json:"device_path"`
	Port       int               `json:"port"`
	Status     NBDExportStatus   `json:"status"`
	CreatedAt  time.Time         `json:"created_at"`
	UpdatedAt  time.Time         `json:"updated_at"`
	Metadata   map[string]string `json:"metadata"`
}

// NBDExportStatus represents the status of an NBD export
type NBDExportStatus string

const (
	NBDExportStatusPending NBDExportStatus = "pending"
	NBDExportStatusActive  NBDExportStatus = "active"
	NBDExportStatusFailed  NBDExportStatus = "failed"
)

// NBDExportRequest represents a request to create an NBD export
type NBDExportRequest struct {
	VolumeID   string            `json:"volume_id"`
	VMName     string            `json:"vm_name"`
	VMID       string            `json:"vm_id"`
	DiskNumber int               `json:"disk_number"`
	DevicePath string            `json:"device_path"`
	ReadOnly   bool              `json:"read_only"`
	Metadata   map[string]string `json:"metadata"`
}

// NBDExportFilter represents filters for listing NBD exports
type NBDExportFilter struct {
	VolumeID *string          `json:"volume_id,omitempty"`
	Status   *NBDExportStatus `json:"status,omitempty"`
	VMName   *string          `json:"vm_name,omitempty"`
	Limit    int              `json:"limit,omitempty"`
}

// TrackSnapshotRequest represents a request to track a volume snapshot
type TrackSnapshotRequest struct {
	VolumeUUID     string `json:"volume_uuid" binding:"required"`
	VMContextID    string `json:"vm_context_id" binding:"required"`
	SnapshotID     string `json:"snapshot_id" binding:"required"`
	SnapshotName   string `json:"snapshot_name,omitempty"`
	DiskID         string `json:"disk_id,omitempty"`
	SnapshotStatus string `json:"snapshot_status,omitempty"`
}

// UpdateSnapshotRequest represents a request to update snapshot information
type UpdateSnapshotRequest struct {
	VolumeUUID     string  `json:"volume_uuid"`
	SnapshotID     *string `json:"snapshot_id,omitempty"`
	SnapshotName   *string `json:"snapshot_name,omitempty"`
	SnapshotStatus *string `json:"snapshot_status,omitempty"`
}

// VolumeSnapshotInfo represents snapshot information for a volume
type VolumeSnapshotInfo struct {
	VolumeUUID        string     `json:"volume_uuid"`
	VMContextID       string     `json:"vm_context_id"`
	DevicePath        string     `json:"device_path"`
	OperationMode     string     `json:"operation_mode"`
	SnapshotID        *string    `json:"snapshot_id,omitempty"`
	SnapshotCreatedAt *time.Time `json:"snapshot_created_at,omitempty"`
	SnapshotStatus    string     `json:"snapshot_status"`
}
