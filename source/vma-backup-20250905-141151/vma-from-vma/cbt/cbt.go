// Package cbt provides CBT (Change Block Tracking) management utilities for VMware VMs
package cbt

import (
	"context"
	"fmt"
	"net/url"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/session/keepalive"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
)

// Manager handles CBT operations for VMware VMs
type Manager struct {
	vcenter  string
	username string
	password string
}

// NewManager creates a new CBT manager instance
func NewManager(vcenter, username, password string) *Manager {
	return &Manager{
		vcenter:  vcenter,
		username: username,
		password: password,
	}
}

// CBTStatus represents the CBT status of a VM
type CBTStatus struct {
	Enabled    bool   `json:"enabled"`
	VMName     string `json:"vm_name"`
	PowerState string `json:"power_state"`
	VMPath     string `json:"vm_path"`
}

// CheckCBTStatus checks if CBT is enabled on the specified VM
func (m *Manager) CheckCBTStatus(ctx context.Context, vmPath string) (*CBTStatus, error) {
	vm, vmMo, err := m.getVMAndProperties(ctx, vmPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get VM properties: %w", err)
	}

	_ = vm // Keep reference to avoid unused variable

	cbtEnabled := vmMo.Config.ChangeTrackingEnabled != nil && *vmMo.Config.ChangeTrackingEnabled

	status := &CBTStatus{
		Enabled:    cbtEnabled,
		VMName:     vmMo.Name,
		PowerState: string(vmMo.Runtime.PowerState),
		VMPath:     vmPath,
	}

	log.WithFields(log.Fields{
		"vm_path":     vmPath,
		"vm_name":     vmMo.Name,
		"cbt_enabled": cbtEnabled,
		"power_state": vmMo.Runtime.PowerState,
	}).Info("CBT status check completed")

	return status, nil
}

// EnsureCBTEnabled checks CBT status and enables it if not already enabled
// This is the critical method for migration - CBT disabled = migration failure
func (m *Manager) EnsureCBTEnabled(ctx context.Context, vmPath string) error {
	vm, vmMo, err := m.getVMAndProperties(ctx, vmPath)
	if err != nil {
		return fmt.Errorf("failed to get VM properties for CBT enablement: %w", err)
	}

	return m.ensureCBTEnabledWithClient(ctx, vmPath, vm, vmMo)
}

// EnsureCBTEnabledWithClient uses an existing authenticated govmomi client
func (m *Manager) EnsureCBTEnabledWithClient(ctx context.Context, vmPath string, client *govmomi.Client) error {
	vm, vmMo, err := m.getVMAndPropertiesWithClient(ctx, vmPath, client)
	if err != nil {
		return fmt.Errorf("failed to get VM properties for CBT enablement: %w", err)
	}

	return m.ensureCBTEnabledWithClient(ctx, vmPath, vm, vmMo)
}

// ensureCBTEnabledWithClient handles the actual CBT enablement logic
func (m *Manager) ensureCBTEnabledWithClient(ctx context.Context, vmPath string, vm *object.VirtualMachine, vmMo mo.VirtualMachine) error {
	// Check current CBT status
	cbtEnabled := vmMo.Config.ChangeTrackingEnabled != nil && *vmMo.Config.ChangeTrackingEnabled

	if cbtEnabled {
		log.WithFields(log.Fields{
			"vm_path": vmPath,
			"vm_name": vmMo.Name,
		}).Info("✅ CBT is already enabled - migration can proceed")
		return nil
	}

	// CBT is disabled - attempt to enable it
	log.WithFields(log.Fields{
		"vm_path":     vmPath,
		"vm_name":     vmMo.Name,
		"power_state": vmMo.Runtime.PowerState,
	}).Warn("🔧 CBT not enabled - attempting to enable via vCenter API")

	// CORRECTED: CBT can be enabled on running VMs
	if err := m.enableCBT(ctx, vm); err != nil {
		return fmt.Errorf("MIGRATION FAILURE: Could not enable CBT on VM %s (%s): %w", vmMo.Name, vmPath, err)
	}

	// Initialize CBT with temporary snapshot
	if err := m.initializeCBTWithSnapshot(ctx, vm, vmMo.Name); err != nil {
		return fmt.Errorf("MIGRATION FAILURE: CBT initialization failed for VM %s (%s): %w", vmMo.Name, vmPath, err)
	}

	log.WithFields(log.Fields{
		"vm_path": vmPath,
		"vm_name": vmMo.Name,
	}).Info("✅ CBT enabled and initialized successfully - migration can proceed")

	return nil
}

// getVMAndProperties establishes vCenter connection and retrieves VM properties
func (m *Manager) getVMAndProperties(ctx context.Context, vmPath string) (*object.VirtualMachine, mo.VirtualMachine, error) {
	// Create vCenter connection (same pattern as discovery service)
	u, err := url.Parse(fmt.Sprintf("https://%s/sdk", m.vcenter))
	if err != nil {
		return nil, mo.VirtualMachine{}, fmt.Errorf("failed to parse vCenter URL: %w", err)
	}

	// Set credentials using proper method (same as discovery service)
	u.User = url.UserPassword(m.username, m.password)

	client, err := govmomi.NewClient(ctx, u, true)
	if err != nil {
		return nil, mo.VirtualMachine{}, fmt.Errorf("failed to connect to vCenter %s: %w", m.vcenter, err)
	}
	defer client.Logout(ctx)

	// Set up keepalive
	client.RoundTripper = keepalive.NewHandlerSOAP(client.RoundTripper, 10*time.Minute, nil)

	// Find the VM
	finder := find.NewFinder(client.Client, true)
	vm, err := finder.VirtualMachine(ctx, vmPath)
	if err != nil {
		return nil, mo.VirtualMachine{}, fmt.Errorf("failed to find VM %s: %w", vmPath, err)
	}

	// Get current VM configuration
	var vmMo mo.VirtualMachine
	err = vm.Properties(ctx, vm.Reference(), []string{"config", "runtime"}, &vmMo)
	if err != nil {
		return nil, mo.VirtualMachine{}, fmt.Errorf("failed to get VM properties: %w", err)
	}

	return vm, vmMo, nil
}

// getVMAndPropertiesWithClient retrieves VM properties using an existing authenticated client
func (m *Manager) getVMAndPropertiesWithClient(ctx context.Context, vmPath string, client *govmomi.Client) (*object.VirtualMachine, mo.VirtualMachine, error) {
	// Find the VM using the existing client
	finder := find.NewFinder(client.Client, true)
	vm, err := finder.VirtualMachine(ctx, vmPath)
	if err != nil {
		return nil, mo.VirtualMachine{}, fmt.Errorf("failed to find VM %s: %w", vmPath, err)
	}

	// Get current VM configuration
	var vmMo mo.VirtualMachine
	err = vm.Properties(ctx, vm.Reference(), []string{"config", "runtime"}, &vmMo)
	if err != nil {
		return nil, mo.VirtualMachine{}, fmt.Errorf("failed to get VM properties: %w", err)
	}

	return vm, vmMo, nil
}

// enableCBT enables CBT on the VM via vCenter API
func (m *Manager) enableCBT(ctx context.Context, vm *object.VirtualMachine) error {
	log.Info("🔧 Enabling CBT via vCenter API...")

	cbtEnabled := true
	configSpec := types.VirtualMachineConfigSpec{
		ChangeTrackingEnabled: &cbtEnabled,
	}

	task, err := vm.Reconfigure(ctx, configSpec)
	if err != nil {
		return fmt.Errorf("failed to initiate CBT enablement: %w", err)
	}

	err = task.Wait(ctx)
	if err != nil {
		return fmt.Errorf("CBT enablement task failed: %w", err)
	}

	log.Info("✅ CBT enabled successfully via API")
	return nil
}

// initializeCBTWithSnapshot creates and deletes a temporary snapshot to initialize disk-level CBT
func (m *Manager) initializeCBTWithSnapshot(ctx context.Context, vm *object.VirtualMachine, vmName string) error {
	snapshotName := fmt.Sprintf("cbt-init-%d", time.Now().Unix())

	log.WithFields(log.Fields{
		"vm_name":       vmName,
		"snapshot_name": snapshotName,
	}).Info("📸 Creating temporary snapshot to initialize disk-level CBT")

	// Create snapshot
	task, err := vm.CreateSnapshot(ctx, snapshotName, "Temporary snapshot to initialize CBT for disks", false, false)
	if err != nil {
		return fmt.Errorf("failed to create CBT initialization snapshot: %w", err)
	}

	err = task.Wait(ctx)
	if err != nil {
		return fmt.Errorf("CBT initialization snapshot creation failed: %w", err)
	}

	log.Info("✅ Temporary snapshot created")

	// Wait for CBT to initialize
	log.Info("⏳ Waiting for CBT to initialize...")
	time.Sleep(5 * time.Second)

	// Delete the snapshot
	snapshotRef, err := vm.FindSnapshot(ctx, snapshotName)
	if err != nil {
		return fmt.Errorf("failed to find created snapshot: %w", err)
	}

	log.WithField("snapshot_name", snapshotName).Info("🗑️  Deleting temporary snapshot")

	consolidate := true
	task, err = vm.RemoveSnapshot(ctx, snapshotRef.Value, false, &consolidate)
	if err != nil {
		return fmt.Errorf("failed to delete CBT initialization snapshot: %w", err)
	}

	err = task.Wait(ctx)
	if err != nil {
		return fmt.Errorf("CBT initialization snapshot deletion failed: %w", err)
	}

	log.Info("✅ Temporary snapshot deleted - CBT initialized for all disks")
	return nil
}
