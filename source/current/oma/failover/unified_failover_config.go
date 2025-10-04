// Package failover provides unified failover configuration for both live and test scenarios
package failover

import (
	"fmt"
	"time"
)

// FailoverType defines the type of failover operation
type FailoverType string

const (
	FailoverTypeLive FailoverType = "live"
	FailoverTypeTest FailoverType = "test"
)

// VMNamingStrategy defines how the destination VM should be named
type VMNamingStrategy string

const (
	VMNamingExact    VMNamingStrategy = "exact"    // Use exact same name as source (live failover)
	VMNamingSuffixed VMNamingStrategy = "suffixed" // Add timestamp suffix (test failover)
)

// SnapshotType defines the type of snapshot to create for rollback protection
type SnapshotType string

const (
	SnapshotTypeCloudStack SnapshotType = "cloudstack" // CloudStack VM snapshots (test failover)
	SnapshotTypeLinstor    SnapshotType = "linstor"    // Linstor volume snapshots (live failover)
	SnapshotTypeNone       SnapshotType = "none"       // No snapshot protection
)

// NetworkStrategy defines which networks to use for the destination VM
type NetworkStrategy string

const (
	NetworkStrategyProduction NetworkStrategy = "production" // Production networks (live failover)
	NetworkStrategyIsolated   NetworkStrategy = "isolated"   // Test/isolated networks (test failover)
	NetworkStrategyCustom     NetworkStrategy = "custom"     // User-defined network mappings
)

// UnifiedFailoverConfig provides configuration for both live and test failover operations
// This replaces the separate EnhancedTestFailoverRequest and EnhancedFailoverRequest structures
type UnifiedFailoverConfig struct {
	// Core identification (VM-centric architecture compliance)
	ContextID     string `json:"context_id" binding:"required"`      // VM context ID (primary key)
	VMwareVMID    string `json:"vmware_vm_id" binding:"required"`    // VMware VM ID
	VMName        string `json:"vm_name" binding:"required"`         // VM name for backward compatibility
	FailoverJobID string `json:"failover_job_id" binding:"required"` // Failover job ID
	VCenterHost   string `json:"vcenter_host"`                       // vCenter hostname from VM context

	// Behavior configuration (determines live vs test behavior)
	FailoverType    FailoverType     `json:"failover_type" binding:"required"`    // LIVE or TEST
	VMNaming        VMNamingStrategy `json:"vm_naming" binding:"required"`        // EXACT or SUFFIXED
	SnapshotType    SnapshotType     `json:"snapshot_type" binding:"required"`    // CLOUDSTACK, LINSTOR, or NONE
	NetworkStrategy NetworkStrategy  `json:"network_strategy" binding:"required"` // PRODUCTION, ISOLATED, or CUSTOM

	// Optional behaviors (user configurable via GUI prompts)
	PowerOffSource   bool `json:"power_off_source"`   // Power off source VM (live failover only)
	PerformFinalSync bool `json:"perform_final_sync"` // Perform final incremental sync (live failover only)
	CleanupEnabled   bool `json:"cleanup_enabled"`    // Enable cleanup/rollback capability (both types)
	SkipValidation   bool `json:"skip_validation"`    // Skip pre-failover validation (emergency use)
	SkipVirtIO       bool `json:"skip_virtio"`        // Skip VirtIO driver injection (if not needed)

	// Timing and metadata
	Timestamp time.Time `json:"timestamp"`
	UserID    string    `json:"user_id,omitempty"` // User who initiated the failover
	Reason    string    `json:"reason,omitempty"`  // Reason for failover (audit trail)
}

// IsLiveFailover returns true if this is a live failover configuration
func (ufc *UnifiedFailoverConfig) IsLiveFailover() bool {
	return ufc.FailoverType == FailoverTypeLive
}

// IsTestFailover returns true if this is a test failover configuration
func (ufc *UnifiedFailoverConfig) IsTestFailover() bool {
	return ufc.FailoverType == FailoverTypeTest
}

// GetDestinationVMName returns the name to use for the destination VM
func (ufc *UnifiedFailoverConfig) GetDestinationVMName() string {
	switch ufc.VMNaming {
	case VMNamingExact:
		return ufc.VMName
	case VMNamingSuffixed:
		return fmt.Sprintf("%s-test-%d", ufc.VMName, ufc.Timestamp.Unix())
	default:
		return ufc.VMName
	}
}

// RequiresSourceVMPowerOff returns true if the source VM should be powered off
func (ufc *UnifiedFailoverConfig) RequiresSourceVMPowerOff() bool {
	return ufc.IsLiveFailover() && ufc.PowerOffSource
}

// RequiresFinalSync returns true if a final sync should be performed
func (ufc *UnifiedFailoverConfig) RequiresFinalSync() bool {
	return ufc.IsLiveFailover() && ufc.PerformFinalSync
}

// GetSnapshotName returns the snapshot name to use for rollback protection
func (ufc *UnifiedFailoverConfig) GetSnapshotName() string {
	switch ufc.SnapshotType {
	case SnapshotTypeCloudStack:
		return fmt.Sprintf("test-failover-%s-%d", ufc.VMName, ufc.Timestamp.Unix())
	case SnapshotTypeLinstor:
		return fmt.Sprintf("live-failover-%s-%d", ufc.VMName, ufc.Timestamp.Unix())
	default:
		return ""
	}
}

// Validate ensures the configuration is valid and consistent
func (ufc *UnifiedFailoverConfig) Validate() error {
	if ufc.ContextID == "" {
		return fmt.Errorf("context_id is required")
	}
	if ufc.VMwareVMID == "" {
		return fmt.Errorf("vmware_vm_id is required")
	}
	if ufc.VMName == "" {
		return fmt.Errorf("vm_name is required")
	}
	if ufc.FailoverJobID == "" {
		return fmt.Errorf("failover_job_id is required")
	}

	// Validate live failover specific requirements
	if ufc.IsLiveFailover() {
		if ufc.VMNaming != VMNamingExact {
			return fmt.Errorf("live failover must use exact VM naming")
		}
		fmt.Printf("🔍 DEBUG VALIDATION: NetworkStrategy = %v (Production=%v, Custom=%v)\n", ufc.NetworkStrategy, NetworkStrategyProduction, NetworkStrategyCustom)
		if ufc.NetworkStrategy != NetworkStrategyProduction && ufc.NetworkStrategy != NetworkStrategyCustom {
			return fmt.Errorf("live failover must use production or custom networks")
		}
		if ufc.SnapshotType != SnapshotTypeCloudStack && ufc.SnapshotType != SnapshotTypeNone {
			return fmt.Errorf("live failover must use CloudStack volume snapshots or no snapshots")
		}
	}

	// Validate test failover specific requirements
	if ufc.IsTestFailover() {
		if ufc.VMNaming != VMNamingSuffixed {
			return fmt.Errorf("test failover must use suffixed VM naming")
		}
		if ufc.NetworkStrategy != NetworkStrategyIsolated && ufc.NetworkStrategy != NetworkStrategyCustom {
			return fmt.Errorf("test failover must use isolated or custom networks")
		}
		if ufc.SnapshotType != SnapshotTypeCloudStack {
			return fmt.Errorf("test failover must use CloudStack snapshots")
		}
		// Test failover cannot power off source VM or perform final sync
		if ufc.PowerOffSource {
			return fmt.Errorf("test failover cannot power off source VM")
		}
		if ufc.PerformFinalSync {
			return fmt.Errorf("test failover cannot perform final sync")
		}
	}

	return nil
}

// NewLiveFailoverConfig creates a configuration for live failover with sensible defaults
func NewLiveFailoverConfig(contextID, vmwareVMID, vmName, failoverJobID string) *UnifiedFailoverConfig {
	return &UnifiedFailoverConfig{
		ContextID:        contextID,
		VMwareVMID:       vmwareVMID,
		VMName:           vmName,
		FailoverJobID:    failoverJobID,
		FailoverType:     FailoverTypeLive,
		VMNaming:         VMNamingExact,
		SnapshotType:     SnapshotTypeCloudStack, // IMPORTANT: Always use CloudStack volume snapshots
		NetworkStrategy:  NetworkStrategyProduction,
		PowerOffSource:   true,  // Default: power off source for live failover
		PerformFinalSync: true,  // Default: perform final sync for live failover
		CleanupEnabled:   true,  // Default: enable cleanup capability
		SkipValidation:   false, // Default: perform validation
		SkipVirtIO:       false, // Default: perform VirtIO injection
		Timestamp:        time.Now(),
	}
}

// NewTestFailoverConfig creates a configuration for test failover with sensible defaults
func NewTestFailoverConfig(contextID, vmwareVMID, vmName, failoverJobID string) *UnifiedFailoverConfig {
	return &UnifiedFailoverConfig{
		ContextID:        contextID,
		VMwareVMID:       vmwareVMID,
		VMName:           vmName,
		FailoverJobID:    failoverJobID,
		FailoverType:     FailoverTypeTest,
		VMNaming:         VMNamingSuffixed,
		SnapshotType:     SnapshotTypeCloudStack,
		NetworkStrategy:  NetworkStrategyIsolated,
		PowerOffSource:   false, // Test failover never powers off source
		PerformFinalSync: false, // Test failover never performs final sync
		CleanupEnabled:   true,  // Default: enable cleanup capability
		SkipValidation:   false, // Default: perform validation
		SkipVirtIO:       false, // Default: perform VirtIO injection
		Timestamp:        time.Now(),
	}
}
