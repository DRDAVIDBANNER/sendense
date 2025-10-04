package service

import (
	"context"
	"time"

	"github.com/vexxhost/migratekit-volume-daemon/models"
)

// VolumeManagementService is the core interface for centralized volume management
// ALL volume operations MUST go through this service - no direct CloudStack API calls allowed
type VolumeManagementService interface {
	// Volume Operations (ONLY way to interact with CloudStack volumes)
	CreateVolume(ctx context.Context, req models.CreateVolumeRequest) (*models.VolumeOperation, error)
	AttachVolume(ctx context.Context, volumeID, vmID string) (*models.VolumeOperation, error)
	AttachVolumeAsRoot(ctx context.Context, volumeID, vmID string) (*models.VolumeOperation, error)
	DetachVolume(ctx context.Context, volumeID string) (*models.VolumeOperation, error)
	DeleteVolume(ctx context.Context, volumeID string) (*models.VolumeOperation, error)

	// Cleanup Operations
	CleanupTestFailover(ctx context.Context, req models.CleanupRequest) (*models.VolumeOperation, error)

	// Real-time Status Queries
	GetVolumeStatus(ctx context.Context, volumeID string) (*models.VolumeStatus, error)
	GetDeviceMapping(ctx context.Context, volumeID string) (*models.DeviceMapping, error)
	GetVolumeForDevice(ctx context.Context, devicePath string) (*models.DeviceMapping, error)
	ListVolumesForVM(ctx context.Context, vmID string) ([]models.VolumeStatus, error)

	// Operation Tracking
	GetOperation(ctx context.Context, operationID string) (*models.VolumeOperation, error)
	ListOperations(ctx context.Context, filter models.OperationFilter) ([]models.VolumeOperation, error)
	WaitForOperation(ctx context.Context, operationID string, timeout time.Duration) (*models.VolumeOperation, error)

	// NBD Export Management (NEW - integrated with volume lifecycle)
	CreateNBDExport(ctx context.Context, volumeID, vmName, vmID string, diskNumber int) (*models.NBDExportInfo, error)
	DeleteNBDExport(ctx context.Context, volumeID string) error
	GetNBDExport(ctx context.Context, volumeID string) (*models.NBDExportInfo, error)
	ListNBDExports(ctx context.Context, filter models.NBDExportFilter) ([]*models.NBDExportInfo, error)
	ValidateNBDExports(ctx context.Context) error

	// Administrative
	GetHealth(ctx context.Context) (*models.HealthStatus, error)
	GetMetrics(ctx context.Context) (*models.ServiceMetrics, error)
	ForceSync(ctx context.Context) error

	// Snapshot tracking
	TrackVolumeSnapshot(ctx context.Context, req *models.TrackSnapshotRequest) error
	GetVMVolumeSnapshots(ctx context.Context, vmContextID string) ([]models.VolumeSnapshotInfo, error)
	ClearVMVolumeSnapshots(ctx context.Context, vmContextID string) (int, error)
	UpdateVolumeSnapshot(ctx context.Context, req *models.UpdateSnapshotRequest) error

	// Lifecycle
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}

// CloudStackFactory defines the interface for creating CloudStack clients
type CloudStackFactory interface {
	CreateClient(ctx context.Context) (CloudStackClient, error)
	TestConnection(ctx context.Context) error
}

// CloudStackClient defines the interface for CloudStack operations
type CloudStackClient interface {
	// Volume operations
	CreateVolume(ctx context.Context, req models.CreateVolumeRequest) (string, error)
	AttachVolume(ctx context.Context, volumeID, vmID string) error
	AttachVolumeAsRoot(ctx context.Context, volumeID, vmID string) error
	DetachVolume(ctx context.Context, volumeID string) error
	DeleteVolume(ctx context.Context, volumeID string) error

	// Volume queries
	GetVolume(ctx context.Context, volumeID string) (map[string]interface{}, error)
	ListVolumes(ctx context.Context, vmID string) ([]map[string]interface{}, error)

	// VM operations for cleanup
	GetVMPowerState(ctx context.Context, vmID string) (string, error)
	ValidateVMPoweredOff(ctx context.Context, vmID string) error
	PowerOffVM(ctx context.Context, vmID string) error
	DeleteVM(ctx context.Context, vmID string) error

	// Health check
	Ping(ctx context.Context) error
}

// DeviceMonitor defines the interface for Linux device monitoring
type DeviceMonitor interface {
	// Device monitoring
	StartMonitoring(ctx context.Context) error
	StopMonitoring(ctx context.Context) error

	// Device queries
	GetDevices(ctx context.Context) ([]DeviceInfo, error)
	GetDeviceByPath(ctx context.Context, devicePath string) (*DeviceInfo, error)
	WaitForDevice(ctx context.Context, timeout time.Duration) (*DeviceEvent, error)

	// Health check
	IsHealthy(ctx context.Context) bool
}

// DeviceInfo represents information about a Linux block device
type DeviceInfo struct {
	Path       string            `json:"path"`       // e.g., "/dev/vdb"
	Size       int64             `json:"size"`       // Size in bytes
	Controller string            `json:"controller"` // Virtio controller info
	Metadata   map[string]string `json:"metadata"`   // Additional device metadata
}

// DeviceEvent represents a device change event
type DeviceEvent struct {
	Type       DeviceEventType `json:"type"`
	DevicePath string          `json:"device_path"`
	DeviceInfo *DeviceInfo     `json:"device_info,omitempty"`
	Timestamp  time.Time       `json:"timestamp"`
}

// DeviceEventType defines the type of device event
type DeviceEventType string

const (
	DeviceAdded   DeviceEventType = "added"
	DeviceRemoved DeviceEventType = "removed"
	DeviceChanged DeviceEventType = "changed"
)

// VolumeRepository defines the interface for volume operation persistence
type VolumeRepository interface {
	// Operation management
	CreateOperation(ctx context.Context, op *models.VolumeOperation) error
	UpdateOperation(ctx context.Context, op *models.VolumeOperation) error
	GetOperation(ctx context.Context, operationID string) (*models.VolumeOperation, error)
	ListOperations(ctx context.Context, filter models.OperationFilter) ([]models.VolumeOperation, error)

	// Device mapping management
	CreateMapping(ctx context.Context, mapping *models.DeviceMapping) error
	UpdateMapping(ctx context.Context, mapping *models.DeviceMapping) error
	UpdateDeviceMapping(ctx context.Context, mapping *models.DeviceMapping) error
	DeleteMapping(ctx context.Context, volumeID string) error
	GetMapping(ctx context.Context, volumeID string) (*models.DeviceMapping, error)
	GetMappingByDevice(ctx context.Context, devicePath string) (*models.DeviceMapping, error)
	GetDeviceMappingByVolumeUUID(ctx context.Context, volumeUUID string) (*models.DeviceMapping, error)
	GetDeviceMappingsByVMContext(ctx context.Context, vmContextID string) ([]models.DeviceMapping, error)
	ListMappingsForVM(ctx context.Context, vmID string) ([]models.DeviceMapping, error)

	// Health check
	Ping(ctx context.Context) error

	// Configuration queries
	GetOMAVMID(ctx context.Context) (string, error)
}
