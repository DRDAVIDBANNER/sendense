// Package vmware provides VMware vCenter power management operations
// Following project rules: clean interfaces, structured logging, timeout protection
package vmware

import (
	"context"
	"fmt"
	"net/url"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
)

// PowerManager provides VMware VM power management operations
type PowerManager struct {
	config Config
	client *govmomi.Client
}

// PowerState represents VM power state information
type PowerState struct {
	State           string    `json:"state"`             // poweredOn, poweredOff, suspended
	ToolsStatus     string    `json:"tools_status"`      // toolsOk, toolsNotInstalled, toolsNotRunning
	LastStateChange time.Time `json:"last_state_change"` // Last power state change
	UptimeSeconds   int64     `json:"uptime_seconds"`    // VM uptime in seconds
}

// PowerOperation represents the result of a power operation
type PowerOperation struct {
	Success        bool          `json:"success"`
	PreviousState  string        `json:"previous_state"`
	NewState       string        `json:"new_state"`
	ShutdownMethod string        `json:"shutdown_method,omitempty"` // graceful, forced
	ToolsStatus    string        `json:"tools_status,omitempty"`
	Duration       time.Duration `json:"duration"`
	TaskInfo       string        `json:"task_info,omitempty"` // VMware task information
}

// NewPowerManager creates a new VMware power management client
func NewPowerManager(config Config) *PowerManager {
	return &PowerManager{
		config: config,
	}
}

// Connect establishes connection to vCenter (reuses Discovery connection logic)
func (pm *PowerManager) Connect(ctx context.Context) error {
	u, err := url.Parse(fmt.Sprintf("https://%s/sdk", pm.config.Host))
	if err != nil {
		return fmt.Errorf("failed to parse vCenter URL: %w", err)
	}

	u.User = url.UserPassword(pm.config.Username, pm.config.Password)

	// Create vCenter client
	client, err := govmomi.NewClient(ctx, u, pm.config.Insecure)
	if err != nil {
		return fmt.Errorf("failed to create vCenter client: %w", err)
	}

	pm.client = client

	log.WithFields(log.Fields{
		"vcenter":    pm.config.Host,
		"datacenter": pm.config.Datacenter,
	}).Info("✅ Connected to vCenter for power management")

	return nil
}

// Disconnect closes the vCenter connection
func (pm *PowerManager) Disconnect() {
	if pm.client != nil {
		pm.client.Logout(context.Background())
		pm.client = nil
	}
}

// findVMByID finds a VM by its UUID
func (pm *PowerManager) findVMByID(ctx context.Context, vmID string) (*object.VirtualMachine, error) {
	if pm.client == nil {
		return nil, fmt.Errorf("not connected to vCenter")
	}

	// Create finder for the datacenter
	finder := find.NewFinder(pm.client.Client, true)

	// Find datacenter
	dc, err := finder.Datacenter(ctx, pm.config.Datacenter)
	if err != nil {
		return nil, fmt.Errorf("failed to find datacenter %s: %w", pm.config.Datacenter, err)
	}

	finder.SetDatacenter(dc)

	// Search for VM by UUID
	searchIndex := object.NewSearchIndex(pm.client.Client)
	vmRef, err := searchIndex.FindByUuid(ctx, dc, vmID, true, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to search for VM by UUID %s: %w", vmID, err)
	}

	if vmRef == nil {
		return nil, fmt.Errorf("VM not found with UUID: %s", vmID)
	}

	vm := object.NewVirtualMachine(pm.client.Client, vmRef.Reference())
	return vm, nil
}

// GetVMPowerState returns the current power state of a VM
func (pm *PowerManager) GetVMPowerState(ctx context.Context, vmID string) (*PowerState, error) {
	log.WithField("vm_id", vmID).Info("🔍 Getting VM power state")

	vm, err := pm.findVMByID(ctx, vmID)
	if err != nil {
		return nil, err
	}

	// Get VM properties
	var vmMo mo.VirtualMachine
	err = vm.Properties(ctx, vm.Reference(), []string{
		"runtime.powerState",
		"guest.toolsStatus",
		"runtime.bootTime",
	}, &vmMo)
	if err != nil {
		return nil, fmt.Errorf("failed to get VM properties: %w", err)
	}

	// Convert power state
	var powerState string
	switch vmMo.Runtime.PowerState {
	case types.VirtualMachinePowerStatePoweredOn:
		powerState = "poweredOn"
	case types.VirtualMachinePowerStatePoweredOff:
		powerState = "poweredOff"
	case types.VirtualMachinePowerStateSuspended:
		powerState = "suspended"
	default:
		powerState = "unknown"
	}

	// Convert tools status
	var toolsStatus string
	if vmMo.Guest != nil && vmMo.Guest.ToolsStatus != "" {
		toolsStatus = string(vmMo.Guest.ToolsStatus)
	} else {
		toolsStatus = "toolsNotInstalled"
	}

	// Calculate uptime
	var uptimeSeconds int64
	if vmMo.Runtime.BootTime != nil && powerState == "poweredOn" {
		uptimeSeconds = int64(time.Since(*vmMo.Runtime.BootTime).Seconds())
	}

	result := &PowerState{
		State:         powerState,
		ToolsStatus:   toolsStatus,
		UptimeSeconds: uptimeSeconds,
	}

	if vmMo.Runtime.BootTime != nil {
		result.LastStateChange = *vmMo.Runtime.BootTime
	}

	log.WithFields(log.Fields{
		"vm_id":          vmID,
		"power_state":    result.State,
		"tools_status":   result.ToolsStatus,
		"uptime_seconds": result.UptimeSeconds,
	}).Info("✅ VM power state retrieved")

	return result, nil
}

// PowerOnVM powers on a VMware VM with optional VMware Tools wait
func (pm *PowerManager) PowerOnVM(ctx context.Context, vmID string, waitForTools bool, timeout time.Duration) (*PowerOperation, error) {
	startTime := time.Now()

	log.WithFields(log.Fields{
		"vm_id":          vmID,
		"wait_for_tools": waitForTools,
		"timeout":        timeout,
	}).Info("⚡ Starting VM power-on operation")

	// Get initial state
	initialState, err := pm.GetVMPowerState(ctx, vmID)
	if err != nil {
		return nil, fmt.Errorf("failed to get initial power state: %w", err)
	}

	// Check if already powered on
	if initialState.State == "poweredOn" {
		log.WithField("vm_id", vmID).Info("✅ VM already powered on")
		return &PowerOperation{
			Success:       true,
			PreviousState: initialState.State,
			NewState:      "poweredOn",
			ToolsStatus:   initialState.ToolsStatus,
			Duration:      time.Since(startTime),
		}, nil
	}

	vm, err := pm.findVMByID(ctx, vmID)
	if err != nil {
		return nil, err
	}

	// Create timeout context
	powerCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Power on the VM
	task, err := vm.PowerOn(powerCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to initiate power-on: %w", err)
	}

	log.WithField("vm_id", vmID).Info("⚡ VM power-on task initiated, waiting for completion...")

	// Wait for task completion
	err = task.Wait(powerCtx)
	if err != nil {
		return nil, fmt.Errorf("power-on task failed: %w", err)
	}

	log.WithField("vm_id", vmID).Info("✅ VM power-on task completed")

	// Wait for VMware Tools if requested
	if waitForTools {
		log.WithField("vm_id", vmID).Info("⏳ Waiting for VMware Tools to start...")
		err = pm.waitForTools(powerCtx, vm, 30*time.Second)
		if err != nil {
			log.WithError(err).WithField("vm_id", vmID).Warn("VMware Tools wait failed, but VM is powered on")
		}
	}

	// Get final state
	finalState, err := pm.GetVMPowerState(ctx, vmID)
	if err != nil {
		return nil, fmt.Errorf("failed to get final power state: %w", err)
	}

	result := &PowerOperation{
		Success:       true,
		PreviousState: initialState.State,
		NewState:      finalState.State,
		ToolsStatus:   finalState.ToolsStatus,
		Duration:      time.Since(startTime),
		TaskInfo:      fmt.Sprintf("task-%s", task.Reference().Value),
	}

	log.WithFields(log.Fields{
		"vm_id":            vmID,
		"previous_state":   result.PreviousState,
		"new_state":        result.NewState,
		"tools_status":     result.ToolsStatus,
		"duration_seconds": result.Duration.Seconds(),
	}).Info("✅ VM power-on operation completed successfully")

	return result, nil
}

// PowerOffVM powers off a VMware VM with graceful shutdown support
func (pm *PowerManager) PowerOffVM(ctx context.Context, vmID string, force bool, timeout time.Duration) (*PowerOperation, error) {
	startTime := time.Now()

	log.WithFields(log.Fields{
		"vm_id":   vmID,
		"force":   force,
		"timeout": timeout,
	}).Info("🔌 Starting VM power-off operation")

	// Get initial state
	initialState, err := pm.GetVMPowerState(ctx, vmID)
	if err != nil {
		return nil, fmt.Errorf("failed to get initial power state: %w", err)
	}

	// Check if already powered off
	if initialState.State == "poweredOff" {
		log.WithField("vm_id", vmID).Info("✅ VM already powered off")
		return &PowerOperation{
			Success:       true,
			PreviousState: initialState.State,
			NewState:      "poweredOff",
			Duration:      time.Since(startTime),
		}, nil
	}

	vm, err := pm.findVMByID(ctx, vmID)
	if err != nil {
		return nil, err
	}

	// Create timeout context
	powerCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var task *object.Task
	var shutdownMethod string

	if !force && initialState.ToolsStatus == "toolsOk" {
		// Try graceful shutdown first
		log.WithField("vm_id", vmID).Info("🔄 Attempting graceful shutdown via VMware Tools")

		err = vm.ShutdownGuest(powerCtx)
		if err != nil {
			log.WithError(err).WithField("vm_id", vmID).Warn("Graceful shutdown failed, falling back to force power-off")
			force = true
		} else {
			shutdownMethod = "graceful"

			// Wait for graceful shutdown with shorter timeout
			gracefulCtx, gracefulCancel := context.WithTimeout(powerCtx, timeout/2)
			defer gracefulCancel()

			if err := pm.waitForPowerOff(gracefulCtx, vm); err != nil {
				log.WithError(err).WithField("vm_id", vmID).Warn("Graceful shutdown timeout, forcing power-off")
				force = true
			}
		}
	}

	if force || initialState.ToolsStatus != "toolsOk" {
		// Force power-off
		log.WithField("vm_id", vmID).Info("⚡ Force powering off VM")
		shutdownMethod = "forced"

		task, err = vm.PowerOff(powerCtx)
		if err != nil {
			return nil, fmt.Errorf("failed to initiate force power-off: %w", err)
		}

		log.WithField("vm_id", vmID).Info("🔌 VM force power-off task initiated, waiting for completion...")

		// Wait for task completion
		err = task.Wait(powerCtx)
		if err != nil {
			return nil, fmt.Errorf("force power-off task failed: %w", err)
		}
	}

	log.WithField("vm_id", vmID).Info("✅ VM power-off task completed")

	// Get final state
	finalState, err := pm.GetVMPowerState(ctx, vmID)
	if err != nil {
		return nil, fmt.Errorf("failed to get final power state: %w", err)
	}

	result := &PowerOperation{
		Success:        true,
		PreviousState:  initialState.State,
		NewState:       finalState.State,
		ShutdownMethod: shutdownMethod,
		Duration:       time.Since(startTime),
	}

	if task != nil {
		result.TaskInfo = fmt.Sprintf("task-%s", task.Reference().Value)
	}

	log.WithFields(log.Fields{
		"vm_id":            vmID,
		"previous_state":   result.PreviousState,
		"new_state":        result.NewState,
		"shutdown_method":  result.ShutdownMethod,
		"duration_seconds": result.Duration.Seconds(),
	}).Info("✅ VM power-off operation completed successfully")

	return result, nil
}

// waitForTools waits for VMware Tools to become available
func (pm *PowerManager) waitForTools(ctx context.Context, vm *object.VirtualMachine, timeout time.Duration) error {
	log.Info("⏳ Waiting for VMware Tools to become available...")

	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-timeoutCtx.Done():
			return fmt.Errorf("timeout waiting for VMware Tools")
		case <-ticker.C:
			var vmMo mo.VirtualMachine
			err := vm.Properties(ctx, vm.Reference(), []string{"guest.toolsStatus"}, &vmMo)
			if err != nil {
				continue
			}

			if vmMo.Guest != nil && vmMo.Guest.ToolsStatus == types.VirtualMachineToolsStatusToolsOk {
				log.Info("✅ VMware Tools are now available")
				return nil
			}
		}
	}
}

// waitForPowerOff waits for VM to power off (for graceful shutdown)
func (pm *PowerManager) waitForPowerOff(ctx context.Context, vm *object.VirtualMachine) error {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			var vmMo mo.VirtualMachine
			err := vm.Properties(ctx, vm.Reference(), []string{"runtime.powerState"}, &vmMo)
			if err != nil {
				continue
			}

			if vmMo.Runtime.PowerState == types.VirtualMachinePowerStatePoweredOff {
				return nil
			}
		}
	}
}


