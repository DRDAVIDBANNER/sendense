package database

import (
	"time"
)

// OSSEAConfig represents OSSEA connection configuration
type OSSEAConfig struct {
	ID   int    `json:"id" gorm:"primaryKey"`
	Name string `json:"name" gorm:"uniqueIndex;not null"`

	// OSSEA API configuration
	APIURL    string `json:"api_url" gorm:"not null"`
	APIKey    string `json:"api_key" gorm:"not null"`
	SecretKey string `json:"secret_key" gorm:"not null"`
	Domain    string `json:"domain"`
	Zone      string `json:"zone" gorm:"not null"`

	// OSSEA-specific settings
	TemplateID        string `json:"template_id"`
	NetworkID         string `json:"network_id"`
	ServiceOfferingID string `json:"service_offering_id"`
	DiskOfferingID    string `json:"disk_offering_id"`

	// SHA VM identification in OSSEA
	SHAVMID string `json:"oma_vm_id"` // The VM ID of this SHA appliance in OSSEA

	// Metadata
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	IsActive  bool      `json:"is_active" gorm:"default:true"`
}

// OSSEAVolume represents a volume in OSSEA
type OSSEAVolume struct {
	ID          int    `json:"id" gorm:"primaryKey"`
	VMContextID string `json:"vm_context_id" gorm:"index"`            // VM-Centric Architecture: Link to vm_replication_contexts
	VolumeID    string `json:"volume_id" gorm:"uniqueIndex;not null"` // OSSEA volume UUID
	VolumeName  string `json:"volume_name" gorm:"not null"`
	SizeGB      int    `json:"size_gb" gorm:"not null"`

	// Configuration reference (foreign key temporarily disabled)
	OSSEAConfigID int `json:"ossea_config_id"`
	// OSSEAConfig   *OSSEAConfig `json:"ossea_config,omitempty" gorm:"foreignKey:OSSEAConfigID"`

	// Volume metadata
	VolumeType string `json:"volume_type"`                      // ROOT, DATADISK, etc.
	DevicePath string `json:"device_path"`                      // Mount path on SHA appliance
	MountPoint string `json:"mount_point"`                      // Where it's mounted locally
	Status     string `json:"status" gorm:"default:'creating'"` // creating, available, attached, detached, error

	// Timestamps
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Relationships (temporarily disabled)
	// VMDisks      []VMDisk      `json:"vm_disks,omitempty" gorm:"foreignKey:OSSEAVolumeID"`
	// VolumeMounts []VolumeMount `json:"volume_mounts,omitempty" gorm:"foreignKey:OSSEAVolumeID"`
}

// ReplicationJob represents enhanced replication job with OSSEA integration
type ReplicationJob struct {
	ID          string `json:"id" gorm:"primaryKey"`       // Job ID from API
	VMContextID string `json:"vm_context_id" gorm:"index"` // VM-Centric Architecture: Link to vm_replication_contexts

	// Source VM information
	SourceVMID   string `json:"source_vm_id" gorm:"not null"`
	SourceVMName string `json:"source_vm_name" gorm:"not null"`
	SourceVMPath string `json:"source_vm_path" gorm:"not null"`
	VCenterHost  string `json:"vcenter_host" gorm:"not null"`
	Datacenter   string `json:"datacenter" gorm:"not null"`

	// Job configuration
	ReplicationType string `json:"replication_type" gorm:"not null"` // initial, incremental
	TargetNetwork   string `json:"target_network"`
	Status          string `json:"status" gorm:"default:'pending'"` // pending, running, completed, failed, cancelled

	// Progress tracking
	ProgressPercent  float64 `json:"progress_percent" gorm:"default:0.0"`
	CurrentOperation string  `json:"current_operation"`
	BytesTransferred int64   `json:"bytes_transferred" gorm:"default:0"`
	TotalBytes       int64   `json:"total_bytes" gorm:"default:0"`
	TransferSpeedBps int64   `json:"transfer_speed_bps" gorm:"default:0"`
	ErrorMessage     string  `json:"error_message"`

	// CBT and incremental sync
	ChangeID         string `json:"change_id"`          // VMware CBT ChangeID
	PreviousChangeID string `json:"previous_change_id"` // For incremental sync
	SnapshotID       string `json:"snapshot_id"`        // VMware snapshot reference

	// Dynamic allocation
	NBDPort       int    `json:"nbd_port"`
	NBDExportName string `json:"nbd_export_name"`
	TargetDevice  string `json:"target_device"`

	// OSSEA configuration (foreign key temporarily disabled)
	OSSEAConfigID int `json:"ossea_config_id"`
	// OSSEAConfig   *OSSEAConfig `json:"ossea_config,omitempty" gorm:"foreignKey:OSSEAConfigID"`

	// SNA Progress Integration (v1.5.0)
	SNASyncType            string     `json:"vma_sync_type" gorm:"column:vma_sync_type"`
	SNACurrentPhase        string     `json:"vma_current_phase" gorm:"column:vma_current_phase"`
	SNAThroughputMBps      float64    `json:"vma_throughput_mbps" gorm:"column:vma_throughput_mbps;default:0.0"`
	SNAETASeconds          *int       `json:"vma_eta_seconds" gorm:"column:vma_eta_seconds"`
	SNALastPollAt          *time.Time `json:"vma_last_poll_at" gorm:"column:vma_last_poll_at"`
	SNAErrorClassification string     `json:"vma_error_classification" gorm:"column:vma_error_classification"`
	SNAErrorDetails        string     `json:"vma_error_details" gorm:"column:vma_error_details"`

	// Scheduler Integration - ALL VMs referenced by context_id
	ScheduleExecutionID *string `json:"schedule_execution_id" gorm:"type:varchar(64);index;comment:Links job to schedule execution that created it"`
	ScheduledBy         *string `json:"scheduled_by" gorm:"type:varchar(255);index;comment:Which scheduler component created this job"`
	VMGroupID           *string `json:"vm_group_id" gorm:"type:varchar(64);index;comment:Machine group this job belongs to"`

	// Timestamps
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	StartedAt   *time.Time `json:"started_at"`
	CompletedAt *time.Time `json:"completed_at"`

	// Relationships (temporarily disabled)
	// VMDisks      []VMDisk      `json:"vm_disks,omitempty" gorm:"foreignKey:JobID"`
	// VolumeMounts []VolumeMount `json:"volume_mounts,omitempty" gorm:"foreignKey:JobID"`
}

// VMDisk represents disk information for a VM
type VMDisk struct {
	ID          int     `json:"id" gorm:"primaryKey"`
	JobID       *string `json:"job_id" gorm:"type:varchar(191)"` // Nullable - NULL when populated from discovery, set when replication starts
	VMContextID string  `json:"vm_context_id" gorm:"index"`      // VM-Centric Architecture: Link to vm_replication_contexts

	// Source disk info (from VMware)
	DiskID           string `json:"disk_id" gorm:"not null"`
	VMDKPath         string `json:"vmdk_path" gorm:"not null"`
	SizeGB           int    `json:"size_gb" gorm:"not null"`
	Datastore        string `json:"datastore"`
	UnitNumber       int    `json:"unit_number"`
	Label            string `json:"label"`
	CapacityBytes    int64  `json:"capacity_bytes"`
	ProvisioningType string `json:"provisioning_type"`

	// Target OSSEA volume mapping (foreign key temporarily disabled)
	OSSEAVolumeID int `json:"ossea_volume_id"`
	// OSSEAVolume   *OSSEAVolume `json:"ossea_volume,omitempty" gorm:"foreignKey:OSSEAVolumeID"`

	// VM specification fields for failover (populated on first disk of VM)
	CPUCount       int    `json:"cpu_count"`                       // Number of vCPUs
	MemoryMB       int    `json:"memory_mb"`                       // Memory in MB
	OSType         string `json:"os_type"`                         // Guest OS type
	VMToolsVersion string `json:"vm_tools_version"`                // VMware Tools version
	NetworkConfig  string `json:"network_config" gorm:"type:TEXT"` // JSON-encoded network configuration
	DisplayName    string `json:"display_name"`                    // VM display name
	Annotation     string `json:"annotation" gorm:"type:TEXT"`     // VM annotation/notes
	PowerState     string `json:"power_state"`                     // poweredOn, poweredOff, suspended
	VMwareUUID     string `json:"vmware_uuid"`                     // VMware instance UUID
	BIOSSetup      string `json:"bios_setup" gorm:"type:TEXT"`     // JSON-encoded BIOS/firmware settings

	// Sync tracking per disk
	DiskChangeID        string  `json:"disk_change_id"`                       // CBT ChangeID for this specific disk
	SyncStatus          string  `json:"sync_status" gorm:"default:'pending'"` // pending, syncing, completed, failed
	SyncProgressPercent float64 `json:"sync_progress_percent" gorm:"default:0.0"`
	BytesSynced         int64   `json:"bytes_synced" gorm:"default:0"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Relationships - FK constraint enabled for data integrity
	ReplicationJob *ReplicationJob `json:"replication_job,omitempty" gorm:"foreignKey:JobID"`
	// CBTHistory     []CBTHistory    `json:"cbt_history,omitempty" gorm:"foreignKey:DiskID;references:DiskID"`
}

// VolumeMount represents volume mount tracking on SHA appliance
type VolumeMount struct {
	ID            int    `json:"id" gorm:"primaryKey"`
	OSSEAVolumeID int    `json:"ossea_volume_id" gorm:"not null"`
	JobID         string `json:"job_id" gorm:"not null"`

	// Mount details
	DevicePath  string `json:"device_path" gorm:"not null"`             // e.g., /dev/vdb, /dev/vdc
	MountPoint  string `json:"mount_point"`                             // e.g., /mnt/migration/job-123-disk-0
	MountStatus string `json:"mount_status" gorm:"default:'unmounted'"` // unmounted, mounting, mounted, unmount_pending, error

	// Mount options and metadata
	FilesystemType string `json:"filesystem_type"` // ext4, xfs, ntfs, etc.
	MountOptions   string `json:"mount_options"`   // rw,noatime,etc.
	IsReadOnly     bool   `json:"is_read_only" gorm:"default:false"`

	// Tracking
	MountedAt   *time.Time `json:"mounted_at"`
	UnmountedAt *time.Time `json:"unmounted_at"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`

	// Relationships (temporarily disabled)
	// OSSEAVolume    *OSSEAVolume    `json:"ossea_volume,omitempty" gorm:"foreignKey:OSSEAVolumeID"`
	// ReplicationJob *ReplicationJob `json:"replication_job,omitempty" gorm:"foreignKey:JobID"`
}

// CBTHistory represents CBT ChangeID history for incremental sync
type CBTHistory struct {
	ID          int    `json:"id" gorm:"primaryKey"`
	JobID       string `json:"job_id" gorm:"not null"`
	VMContextID string `json:"vm_context_id" gorm:"index"` // VM-Centric Architecture: Link to vm_replication_contexts
	DiskID      string `json:"disk_id" gorm:"not null"`

	// CBT tracking
	ChangeID         string `json:"change_id" gorm:"not null"`
	PreviousChangeID string `json:"previous_change_id"`
	SyncType         string `json:"sync_type" gorm:"not null"` // full, incremental

	// Sync results
	BlocksChanged       int   `json:"blocks_changed"`
	BytesTransferred    int64 `json:"bytes_transferred"`
	SyncDurationSeconds int   `json:"sync_duration_seconds"`
	SyncSuccess         bool  `json:"sync_success" gorm:"default:false"`

	CreatedAt time.Time `json:"created_at"`

	// Relationships (temporarily disabled foreign key constraint due to GORM migration order issues)
	// VMDisk *VMDisk `json:"vm_disk,omitempty" gorm:"foreignKey:DiskID;references:DiskID"`
}

// NBDExport represents an NBD export record for volume access
type NBDExport struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	JobID      string    `gorm:"not null;index" json:"job_id"`             // Foreign key to replication_jobs.id
	VolumeID   string    `gorm:"not null;index" json:"volume_id"`          // Foreign key to ossea_volumes.volume_id
	ExportName string    `gorm:"not null;unique" json:"export_name"`       // NBD export name
	Port       int       `gorm:"not null" json:"port"`                     // NBD server port
	DevicePath string    `gorm:"not null" json:"device_path"`              // Block device path (e.g., /dev/vdb)
	ConfigPath string    `gorm:"not null" json:"config_path"`              // NBD config file path
	Status     string    `gorm:"not null;default:'pending'" json:"status"` // pending, active, stopped, error
	CreatedAt  time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt  time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

// FailoverJob represents VM failover operations (live or test)
type FailoverJob struct {
	ID               int    `json:"id" gorm:"primaryKey"`
	VMContextID      string `json:"vm_context_id" gorm:"index"`         // VM-Centric Architecture: Link to vm_replication_contexts
	JobID            string `json:"job_id" gorm:"uniqueIndex;not null"` // Format: failover-YYYYMMDD-HHMMSS
	VMID             string `json:"vm_id" gorm:"not null;index"`        // Source VM ID from VMware
	ReplicationJobID string `json:"replication_job_id" gorm:"index"`    // FK to replication_jobs.id (temporarily no constraint)

	// Failover configuration
	JobType      string `json:"job_type" gorm:"not null"`        // live, test
	Status       string `json:"status" gorm:"default:'pending'"` // pending, validating, snapshotting, creating_vm, switching_volume, powering_on, completed, failed, cleanup, reverting
	SourceVMName string `json:"source_vm_name" gorm:"not null"`  // Original VM name
	SourceVMSpec string `json:"source_vm_spec" gorm:"type:TEXT"` // JSON-encoded VM specifications

	// OSSEA destination
	DestinationVMID     string `json:"destination_vm_id"`                 // Created VM ID in OSSEA
	OSSEASnapshotID     string `json:"ossea_snapshot_id"`                 // Snapshot before failover
	LinstorSnapshotName string `json:"linstor_snapshot_name"`             // Linstor snapshot name for rollback
	LinstorConfigID     *int   `json:"linstor_config_id"`                 // FK to linstor_configs.id
	NetworkMappings     string `json:"network_mappings" gorm:"type:TEXT"` // JSON-encoded network mappings

	// Error handling
	ErrorMessage string `json:"error_message" gorm:"type:TEXT"`

	// Timestamps
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	StartedAt   *time.Time `json:"started_at"`
	CompletedAt *time.Time `json:"completed_at"`

	// Relationships (temporarily disabled)
	// ReplicationJob *ReplicationJob `json:"replication_job,omitempty" gorm:"foreignKey:ReplicationJobID"`
}

// LinstorConfig represents Linstor API configuration for snapshot operations
type LinstorConfig struct {
	ID   int    `json:"id" gorm:"primaryKey"`
	Name string `json:"name" gorm:"uniqueIndex;not null"`

	// Linstor API configuration
	APIURL      string `json:"api_url" gorm:"not null"`
	APIPort     int    `json:"api_port" gorm:"default:3370"`
	APIProtocol string `json:"api_protocol" gorm:"default:'http'"`

	// Optional authentication
	APIKey    string `json:"api_key"`
	APISecret string `json:"api_secret"`

	// Connection settings
	ConnectionTimeoutSeconds int `json:"connection_timeout_seconds" gorm:"default:30"`
	RetryAttempts            int `json:"retry_attempts" gorm:"default:3"`

	// Metadata
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	IsActive    bool      `json:"is_active" gorm:"default:true"`
}

// NetworkMapping represents network mapping configuration per VM
type NetworkMapping struct {
	ID                     int    `json:"id" gorm:"primaryKey"`
	VMID                   string `json:"vm_id" gorm:"not null;index"`              // VMware VM ID
	VMContextID            string `json:"vm_context_id" gorm:"index"`               // VM-Centric Architecture: Link to vm_replication_contexts
	SourceNetworkName      string `json:"source_network_name" gorm:"not null"`      // Source network name from VMware
	DestinationNetworkID   string `json:"destination_network_id" gorm:"not null"`   // OSSEA network ID
	DestinationNetworkName string `json:"destination_network_name" gorm:"not null"` // OSSEA network name for display
	IsTestNetwork          bool   `json:"is_test_network" gorm:"default:false"`     // Whether this is a test Layer 2 network

	// Metadata
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Unique constraint on VM + source network combination
	// Index: unique_vm_network (vm_id, source_network_name)
}

// TableName methods for custom table names (if needed)
func (OSSEAConfig) TableName() string {
	return "ossea_configs"
}

func (OSSEAVolume) TableName() string {
	return "ossea_volumes"
}

func (ReplicationJob) TableName() string {
	return "replication_jobs"
}

func (VMDisk) TableName() string {
	return "vm_disks"
}

func (VolumeMount) TableName() string {
	return "volume_mounts"
}

func (CBTHistory) TableName() string {
	return "cbt_history"
}

func (NBDExport) TableName() string {
	return "nbd_exports"
}

func (FailoverJob) TableName() string {
	return "failover_jobs"
}

func (NetworkMapping) TableName() string {
	return "network_mappings"
}

// VMReplicationContext represents the VM-Centric Architecture master table
type VMReplicationContext struct {
	ContextID           string     `json:"context_id" gorm:"column:context_id;primaryKey;type:varchar(64);default:uuid()"`
	VMName              string     `json:"vm_name" gorm:"column:vm_name;not null;index"`
	VMwareVMID          string     `json:"vmware_vm_id" gorm:"column:vmware_vm_id;not null;index"`
	VMPath              string     `json:"vm_path" gorm:"column:vm_path;not null;type:varchar(500)"`
	VCenterHost         string     `json:"vcenter_host" gorm:"column:vcenter_host;not null;index"`
	Datacenter          string     `json:"datacenter" gorm:"column:datacenter;not null"`
	CurrentStatus       string     `json:"current_status" gorm:"column:current_status;type:enum('discovered','replicating','ready_for_failover','failed_over_test','failed_over_live','completed','failed','cleanup_required');default:'discovered';index"`
	CurrentJobID        *string    `json:"current_job_id" gorm:"column:current_job_id;type:varchar(191);index"`
	TotalJobsRun        int        `json:"total_jobs_run" gorm:"column:total_jobs_run;default:0"`
	SuccessfulJobs      int        `json:"successful_jobs" gorm:"column:successful_jobs;default:0"`
	FailedJobs          int        `json:"failed_jobs" gorm:"column:failed_jobs;default:0"`
	LastSuccessfulJobID *string    `json:"last_successful_job_id" gorm:"column:last_successful_job_id;type:varchar(191);index"`
	CPUCount            *int       `json:"cpu_count" gorm:"column:cpu_count"`
	MemoryMB            *int       `json:"memory_mb" gorm:"column:memory_mb"`
	OSType              *string    `json:"os_type" gorm:"column:os_type"`
	PowerState          *string    `json:"power_state" gorm:"column:power_state"`
	VMToolsVersion      *string    `json:"vm_tools_version" gorm:"column:vm_tools_version"`
	CreatedAt           time.Time  `json:"created_at" gorm:"column:created_at;autoCreateTime"`
	UpdatedAt           time.Time  `json:"updated_at" gorm:"column:updated_at;autoUpdateTime"`
	FirstJobAt          *time.Time `json:"first_job_at" gorm:"column:first_job_at"`
	LastJobAt           *time.Time `json:"last_job_at" gorm:"column:last_job_at"`
	LastStatusChange    time.Time  `json:"last_status_change" gorm:"column:last_status_change;autoCreateTime"`

	// Scheduler Integration - Added by scheduler migration
	AutoAdded          bool       `json:"auto_added" gorm:"column:auto_added;default:false;index"`
	LastScheduledJobID *string    `json:"last_scheduled_job_id" gorm:"column:last_scheduled_job_id;type:varchar(255)"`
	NextScheduledAt    *time.Time `json:"next_scheduled_at" gorm:"column:next_scheduled_at;index"`
	SchedulerEnabled   bool       `json:"scheduler_enabled" gorm:"column:scheduler_enabled;default:true;index"`
	
	// Configuration References - For CloudStack and VMware credentials
	OSSEAConfigID *int `json:"ossea_config_id" gorm:"column:ossea_config_id"`
	CredentialID  *int `json:"credential_id" gorm:"column:credential_id"`
	
	// Operation Summary - For persistent visibility of failover/rollback operations
	LastOperationSummary *string `json:"last_operation_summary" gorm:"column:last_operation_summary;type:json"`
}

func (VMReplicationContext) TableName() string {
	return "vm_replication_contexts"
}

// =============================================================================
// SCHEDULER SYSTEM MODELS - All VM references use context_id
// =============================================================================

// ReplicationSchedule represents automated replication scheduling configuration
type ReplicationSchedule struct {
	ID          string  `json:"id" gorm:"primaryKey;type:varchar(64);default:uuid()"`
	Name        string  `json:"name" gorm:"not null;uniqueIndex;type:varchar(255)"`
	Description *string `json:"description" gorm:"type:text"`

	// Schedule configuration
	CronExpression string `json:"cron_expression" gorm:"not null;type:varchar(100);comment:Cron expression for schedule timing"`
	ScheduleType   string `json:"schedule_type" gorm:"type:enum('cron','chain');default:'cron';not null"`
	Timezone       string `json:"timezone" gorm:"default:'UTC';type:varchar(50)"`

	// Chain scheduling
	ChainParentScheduleID *string `json:"chain_parent_schedule_id" gorm:"type:varchar(64);index"`
	ChainDelayMinutes     int     `json:"chain_delay_minutes" gorm:"default:0"`

	// Job configuration
	ReplicationType   string `json:"replication_type" gorm:"type:enum('full','incremental','auto');default:'auto'"`
	MaxConcurrentJobs int    `json:"max_concurrent_jobs" gorm:"default:1"`
	RetryAttempts     int    `json:"retry_attempts" gorm:"default:3"`
	RetryDelayMinutes int    `json:"retry_delay_minutes" gorm:"default:30"`

	// Control flags
	Enabled       bool `json:"enabled" gorm:"default:true;index"`
	SkipIfRunning bool `json:"skip_if_running" gorm:"default:true"`

	// Metadata
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`
	CreatedBy string    `json:"created_by" gorm:"default:'system';type:varchar(255)"`

	// Relationships - cascade properly through context_id
	Groups        []VMMachineGroup      `json:"groups,omitempty" gorm:"foreignKey:ScheduleID;references:ID"`
	Executions    []ScheduleExecution   `json:"executions,omitempty" gorm:"foreignKey:ScheduleID;references:ID"`
	ChainChildren []ReplicationSchedule `json:"chain_children,omitempty" gorm:"foreignKey:ChainParentScheduleID;references:ID"`
}

func (ReplicationSchedule) TableName() string {
	return "replication_schedules"
}

// VMMachineGroup represents groups of VMs organized for scheduling
type VMMachineGroup struct {
	ID          string  `json:"id" gorm:"primaryKey;type:varchar(64);default:uuid()"`
	Name        string  `json:"name" gorm:"not null;uniqueIndex;type:varchar(255)"`
	Description *string `json:"description" gorm:"type:text"`
	ScheduleID  *string `json:"schedule_id" gorm:"type:varchar(64);index"`

	// Group settings
	MaxConcurrentVMs int `json:"max_concurrent_vms" gorm:"default:5"`
	Priority         int `json:"priority" gorm:"default:0;index"`

	// Metadata
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`
	CreatedBy string    `json:"created_by" gorm:"default:'system';type:varchar(255)"`

	// Relationships - ALL VM links via context_id
	Schedule    *ReplicationSchedule `json:"schedule,omitempty" gorm:"foreignKey:ScheduleID;references:ID"`
	Memberships []VMGroupMembership  `json:"memberships,omitempty" gorm:"foreignKey:GroupID;references:ID"`
	Executions  []ScheduleExecution  `json:"executions,omitempty" gorm:"foreignKey:GroupID;references:ID"`
}

func (VMMachineGroup) TableName() string {
	return "vm_machine_groups"
}

// VMGroupMembership represents VM membership in machine groups - CRITICAL: Uses context_id
type VMGroupMembership struct {
	ID          string `json:"id" gorm:"primaryKey;type:varchar(64);default:uuid()"`
	GroupID     string `json:"group_id" gorm:"not null;type:varchar(64);index"`
	VMContextID string `json:"vm_context_id" gorm:"not null;type:varchar(64);index;comment:References vm_replication_contexts.context_id"`

	// Per-VM settings
	Enabled            bool    `json:"enabled" gorm:"default:true;index"`
	Priority           int     `json:"priority" gorm:"default:0;index"`
	ScheduleOverrideID *string `json:"schedule_override_id" gorm:"type:varchar(64);index"`

	// Metadata
	AddedAt time.Time `json:"added_at" gorm:"autoCreateTime"`
	AddedBy string    `json:"added_by" gorm:"default:'system';type:varchar(255)"`

	// Relationships - ENSURE context_id linkage
	Group            *VMMachineGroup       `json:"group,omitempty" gorm:"foreignKey:GroupID;references:ID"`
	VMContext        *VMReplicationContext `json:"vm_context,omitempty" gorm:"foreignKey:VMContextID;references:ContextID"`
	ScheduleOverride *ReplicationSchedule  `json:"schedule_override,omitempty" gorm:"foreignKey:ScheduleOverrideID;references:ID"`
}

func (VMGroupMembership) TableName() string {
	return "vm_group_memberships"
}

// ScheduleExecution represents execution tracking for scheduled operations
type ScheduleExecution struct {
	ID         string  `json:"id" gorm:"primaryKey;type:varchar(64);default:uuid()"`
	ScheduleID string  `json:"schedule_id" gorm:"not null;type:varchar(64);index"`
	GroupID    *string `json:"group_id" gorm:"type:varchar(64);index"`

	// Execution timing
	ScheduledAt time.Time  `json:"scheduled_at" gorm:"not null;index"`
	StartedAt   *time.Time `json:"started_at" gorm:"index"`
	CompletedAt *time.Time `json:"completed_at"`

	// Execution status
	Status string `json:"status" gorm:"type:enum('scheduled','running','completed','failed','skipped','cancelled');default:'scheduled';index"`

	// Job statistics - tracks VMs by context_id
	VMsEligible   int `json:"vms_eligible" gorm:"default:0"`
	JobsCreated   int `json:"jobs_created" gorm:"default:0"`
	JobsCompleted int `json:"jobs_completed" gorm:"default:0"`
	JobsFailed    int `json:"jobs_failed" gorm:"default:0"`
	JobsSkipped   int `json:"jobs_skipped" gorm:"default:0"`

	// Execution details
	ExecutionDetails *string `json:"execution_details" gorm:"type:json;comment:Detailed execution information (job IDs, VM context_ids, etc.)"`
	ErrorMessage     *string `json:"error_message" gorm:"type:text"`
	ErrorDetails     *string `json:"error_details" gorm:"type:json"`

	// Performance metrics
	ExecutionDurationSeconds *int `json:"execution_duration_seconds"`

	// Metadata
	CreatedAt   time.Time `json:"created_at" gorm:"autoCreateTime"`
	TriggeredBy string    `json:"triggered_by" gorm:"default:'scheduler';type:varchar(255)"`

	// Relationships
	Schedule *ReplicationSchedule `json:"schedule,omitempty" gorm:"foreignKey:ScheduleID;references:ID"`
	Group    *VMMachineGroup      `json:"group,omitempty" gorm:"foreignKey:GroupID;references:ID"`
	Jobs     []ReplicationJob     `json:"jobs,omitempty" gorm:"foreignKey:ScheduleExecutionID;references:ID"`
}

func (ScheduleExecution) TableName() string {
	return "schedule_executions"
}

// VMware Credential Management Models

// VMwareCredential represents centralized VMware vCenter credential management
type VMwareCredential struct {
	ID                int        `json:"id" gorm:"primaryKey"`
	CredentialName    string     `json:"credential_name" gorm:"uniqueIndex;not null"`
	VCenterHost       string     `json:"vcenter_host" gorm:"column:vcenter_host;not null"`
	Username          string     `json:"username" gorm:"not null"`
	PasswordEncrypted string     `json:"-" gorm:"type:TEXT;not null"` // Never expose in JSON
	Datacenter        string     `json:"datacenter" gorm:"not null"`
	IsActive          bool       `json:"is_active" gorm:"default:true"`
	IsDefault         bool       `json:"is_default" gorm:"default:false"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
	CreatedBy         string     `json:"created_by"`
	LastUsed          *time.Time `json:"last_used"`
	UsageCount        int        `json:"usage_count" gorm:"default:0"`
}

// VMwareCredentials represents decrypted credentials for operations (never stored)
type VMwareCredentials struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	VCenterHost string `json:"vcenter_host"`
	Username    string `json:"username"`
	Password    string `json:"-"` // Never expose in JSON
	Datacenter  string `json:"datacenter"`
	IsActive    bool   `json:"is_active"`
	IsDefault   bool   `json:"is_default"`
}
