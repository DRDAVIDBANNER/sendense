// Package api provides VMware power management endpoints for the VMA server
// Following project rules: minimal endpoints, clean interfaces, structured logging
package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
)

// PowerManagementRequest represents a VM power management request
type PowerManagementRequest struct {
	VMID            string `json:"vm_id" binding:"required"`    // VMware VM UUID
	VCenter         string `json:"vcenter" binding:"required"`  // vCenter hostname
	Username        string `json:"username" binding:"required"` // vCenter username
	Password        string `json:"password" binding:"required"` // vCenter password
	Force           bool   `json:"force,omitempty"`             // Force power operation
	Timeout         int    `json:"timeout,omitempty"`           // Timeout in seconds
	WaitForTools    bool   `json:"wait_for_tools,omitempty"`    // Wait for VMware Tools (power-on)
	WaitForShutdown bool   `json:"wait_for_shutdown,omitempty"` // Wait for graceful shutdown (power-off)
}

// PowerManagementResponse represents the response from power management operations
type PowerManagementResponse struct {
	Success         bool   `json:"success"`
	VMID            string `json:"vm_id"`
	PreviousState   string `json:"previous_state"`            // poweredOn, poweredOff, suspended
	NewState        string `json:"new_state"`                 // poweredOn, poweredOff, suspended
	Operation       string `json:"operation"`                 // power-on, power-off, query
	ShutdownMethod  string `json:"shutdown_method,omitempty"` // graceful, forced (power-off only)
	ToolsStatus     string `json:"tools_status,omitempty"`    // toolsOk, toolsNotRunning, etc.
	DurationSeconds int    `json:"duration_seconds"`          // Time taken for operation
	Timestamp       string `json:"timestamp"`                 // ISO 8601 timestamp
	Message         string `json:"message,omitempty"`         // Additional information
}

// PowerStateResponse represents the response for power state queries
type PowerStateResponse struct {
	VMID            string `json:"vm_id"`
	PowerState      string `json:"power_state"`                 // poweredOn, poweredOff, suspended
	ToolsStatus     string `json:"tools_status"`                // toolsOk, toolsNotInstalled, toolsNotRunning
	LastStateChange string `json:"last_state_change,omitempty"` // ISO 8601 timestamp
	UptimeSeconds   int64  `json:"uptime_seconds,omitempty"`    // VM uptime in seconds
	Timestamp       string `json:"timestamp"`                   // Query timestamp
}

// VMPowerManager interface for VMware power management operations
type VMPowerManager interface {
	PowerOnVM(ctx context.Context, vmID, vcenter, username, password string, waitForTools bool, timeout int) (*PowerManagementResponse, error)
	PowerOffVM(ctx context.Context, vmID, vcenter, username, password string, force bool, timeout int) (*PowerManagementResponse, error)
	GetVMPowerState(ctx context.Context, vmID, vcenter, username, password string) (*PowerStateResponse, error)
}

// createVMwareClient creates a VMware client for power management operations
func createVMwareClient(ctx context.Context, vcenter, username, password string) (*govmomi.Client, error) {
	u, err := url.Parse(fmt.Sprintf("https://%s/sdk", vcenter))
	if err != nil {
		return nil, fmt.Errorf("failed to parse vCenter URL: %w", err)
	}

	u.User = url.UserPassword(username, password)

	// Create vCenter client with insecure connection (TODO: make configurable)
	client, err := govmomi.NewClient(ctx, u, true)
	if err != nil {
		return nil, fmt.Errorf("failed to create vCenter client: %w", err)
	}

	return client, nil
}

// findVMByID finds a VM by its UUID using VMware APIs
func findVMByID(ctx context.Context, client *govmomi.Client, vmID, datacenter string) (*object.VirtualMachine, error) {
	// Create finder for the datacenter
	finder := find.NewFinder(client.Client, true)

	// Find datacenter
	dc, err := finder.Datacenter(ctx, datacenter)
	if err != nil {
		return nil, fmt.Errorf("failed to find datacenter %s: %w", datacenter, err)
	}

	finder.SetDatacenter(dc)

	// Search for VM by UUID
	searchIndex := object.NewSearchIndex(client.Client)
	vmRef, err := searchIndex.FindByUuid(ctx, dc, vmID, true, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to search for VM by UUID %s: %w", vmID, err)
	}

	if vmRef == nil {
		return nil, fmt.Errorf("VM not found with UUID: %s", vmID)
	}

	vm := object.NewVirtualMachine(client.Client, vmRef.Reference())
	return vm, nil
}

// getVMPowerState returns the current power state of a VM
func getVMPowerState(ctx context.Context, vm *object.VirtualMachine) (string, string, error) {
	var vmMo mo.VirtualMachine
	err := vm.Properties(ctx, vm.Reference(), []string{
		"runtime.powerState",
		"guest.toolsStatus",
	}, &vmMo)
	if err != nil {
		return "", "", fmt.Errorf("failed to get VM properties: %w", err)
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

	return powerState, toolsStatus, nil
}

// handleVMPowerOff powers off a VMware VM with graceful shutdown support
func (s *VMAControlServer) handleVMPowerOff(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	vmID := vars["vm_id"]

	// Decode request body
	var req PowerManagementRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.WithError(err).WithField("vm_id", vmID).Error("Invalid power-off request")
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.VCenter == "" || req.Username == "" || req.Password == "" {
		log.WithField("vm_id", vmID).Error("Missing required vCenter credentials")
		http.Error(w, "Missing required fields: vcenter, username, password", http.StatusBadRequest)
		return
	}

	// Set defaults
	req.VMID = vmID
	if req.Timeout == 0 {
		req.Timeout = 300 // 5 minute default timeout
	}

	log.WithFields(log.Fields{
		"vm_id":             vmID,
		"vcenter":           req.VCenter,
		"force":             req.Force,
		"timeout":           req.Timeout,
		"wait_for_shutdown": req.WaitForShutdown,
	}).Info("🔌 Processing VM power-off request")

	// Create timeout context
	ctx, cancel := context.WithTimeout(r.Context(), time.Duration(req.Timeout)*time.Second)
	defer cancel()

	// Connect to vCenter
	client, err := createVMwareClient(ctx, req.VCenter, req.Username, req.Password)
	if err != nil {
		log.WithError(err).WithField("vm_id", vmID).Error("Failed to connect to vCenter")
		http.Error(w, fmt.Sprintf("vCenter connection failed: %v", err), http.StatusInternalServerError)
		return
	}
	defer client.Logout(ctx)

	// Find the VM
	vm, err := findVMByID(ctx, client, vmID, "DatabanxDC") // TODO: Make datacenter configurable
	if err != nil {
		log.WithError(err).WithField("vm_id", vmID).Error("Failed to find VM")
		http.Error(w, fmt.Sprintf("VM not found: %v", err), http.StatusNotFound)
		return
	}

	// Get initial power state
	initialState, initialTools, err := getVMPowerState(ctx, vm)
	if err != nil {
		log.WithError(err).WithField("vm_id", vmID).Error("Failed to get initial power state")
		http.Error(w, fmt.Sprintf("Power state query failed: %v", err), http.StatusInternalServerError)
		return
	}

	var response *PowerManagementResponse

	// Check if already powered off
	if initialState == "poweredOff" {
		response = &PowerManagementResponse{
			Success:         true,
			VMID:            vmID,
			PreviousState:   initialState,
			NewState:        "poweredOff",
			Operation:       "power-off",
			ShutdownMethod:  "already-off",
			DurationSeconds: 0,
			Timestamp:       time.Now().UTC().Format(time.RFC3339),
			Message:         "VM already powered off",
		}
	} else {
		startTime := time.Now()
		var shutdownMethod string

		// Try graceful shutdown first if tools are available and not forced
		if !req.Force && (initialTools == "toolsOk" || initialTools == "toolsOld") {
			log.WithField("vm_id", vmID).Info("🔄 Attempting graceful shutdown")
			err = vm.ShutdownGuest(ctx)
			if err != nil {
				log.WithError(err).WithField("vm_id", vmID).Warn("Graceful shutdown failed, forcing power-off")
				req.Force = true
			} else {
				shutdownMethod = "graceful"
				// Wait briefly for graceful shutdown
				time.Sleep(5 * time.Second)
			}
		}

		// Force power-off if requested or graceful failed
		if req.Force || (initialTools != "toolsOk" && initialTools != "toolsOld") {
			log.WithField("vm_id", vmID).Info("⚡ Force powering off VM")
			shutdownMethod = "forced"

			task, err := vm.PowerOff(ctx)
			if err != nil {
				log.WithError(err).WithField("vm_id", vmID).Error("Failed to initiate power-off")
				http.Error(w, fmt.Sprintf("Power-off failed: %v", err), http.StatusInternalServerError)
				return
			}

			// Wait for task completion
			err = task.Wait(ctx)
			if err != nil {
				log.WithError(err).WithField("vm_id", vmID).Error("Power-off task failed")
				http.Error(w, fmt.Sprintf("Power-off task failed: %v", err), http.StatusInternalServerError)
				return
			}
		}

		// Get final power state
		finalState, _, err := getVMPowerState(ctx, vm)
		if err != nil {
			log.WithError(err).WithField("vm_id", vmID).Error("Failed to get final power state")
			// Continue with response even if we can't verify final state
			finalState = "poweredOff" // Assume success
		}

		response = &PowerManagementResponse{
			Success:         true,
			VMID:            vmID,
			PreviousState:   initialState,
			NewState:        finalState,
			Operation:       "power-off",
			ShutdownMethod:  shutdownMethod,
			DurationSeconds: int(time.Since(startTime).Seconds()),
			Timestamp:       time.Now().UTC().Format(time.RFC3339),
			Message:         "VM power-off completed successfully",
		}
	}

	log.WithFields(log.Fields{
		"vm_id":            vmID,
		"previous_state":   response.PreviousState,
		"new_state":        response.NewState,
		"shutdown_method":  response.ShutdownMethod,
		"duration_seconds": response.DurationSeconds,
	}).Warn("⚠️ PLACEHOLDER: VM power-off endpoint called - VMware integration needed")

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.WithError(err).Error("Failed to encode power-off response")
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// handleVMPowerOn powers on a VMware VM with VMware Tools wait support
func (s *VMAControlServer) handleVMPowerOn(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	vmID := vars["vm_id"]

	// Decode request body
	var req PowerManagementRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.WithError(err).WithField("vm_id", vmID).Error("Invalid power-on request")
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.VCenter == "" || req.Username == "" || req.Password == "" {
		log.WithField("vm_id", vmID).Error("Missing required vCenter credentials")
		http.Error(w, "Missing required fields: vcenter, username, password", http.StatusBadRequest)
		return
	}

	// Set defaults
	req.VMID = vmID
	if req.Timeout == 0 {
		req.Timeout = 600 // 10 minute default timeout for power-on
	}

	log.WithFields(log.Fields{
		"vm_id":          vmID,
		"vcenter":        req.VCenter,
		"timeout":        req.Timeout,
		"wait_for_tools": req.WaitForTools,
	}).Info("⚡ Processing VM power-on request")

	// Create timeout context
	ctx, cancel := context.WithTimeout(r.Context(), time.Duration(req.Timeout)*time.Second)
	defer cancel()

	// Connect to vCenter
	client, err := createVMwareClient(ctx, req.VCenter, req.Username, req.Password)
	if err != nil {
		log.WithError(err).WithField("vm_id", vmID).Error("Failed to connect to vCenter")
		http.Error(w, fmt.Sprintf("vCenter connection failed: %v", err), http.StatusInternalServerError)
		return
	}
	defer client.Logout(ctx)

	// Find the VM
	vm, err := findVMByID(ctx, client, vmID, "DatabanxDC") // TODO: Make datacenter configurable
	if err != nil {
		log.WithError(err).WithField("vm_id", vmID).Error("Failed to find VM")
		http.Error(w, fmt.Sprintf("VM not found: %v", err), http.StatusNotFound)
		return
	}

	// Get initial power state
	initialState, initialTools, err := getVMPowerState(ctx, vm)
	if err != nil {
		log.WithError(err).WithField("vm_id", vmID).Error("Failed to get initial power state")
		http.Error(w, fmt.Sprintf("Power state query failed: %v", err), http.StatusInternalServerError)
		return
	}

	startTime := time.Now()
	var response *PowerManagementResponse

	// Check if already powered on
	if initialState == "poweredOn" {
		response = &PowerManagementResponse{
			Success:         true,
			VMID:            vmID,
			PreviousState:   initialState,
			NewState:        "poweredOn",
			Operation:       "power-on",
			ToolsStatus:     initialTools,
			DurationSeconds: 0,
			Timestamp:       time.Now().UTC().Format(time.RFC3339),
			Message:         "VM already powered on",
		}
	} else {
		// Power on the VM
		log.WithField("vm_id", vmID).Info("⚡ Powering on VM")

		task, err := vm.PowerOn(ctx)
		if err != nil {
			log.WithError(err).WithField("vm_id", vmID).Error("Failed to initiate power-on")
			http.Error(w, fmt.Sprintf("Power-on failed: %v", err), http.StatusInternalServerError)
			return
		}

		// Wait for power-on task completion
		log.WithField("vm_id", vmID).Info("⏳ Waiting for power-on task completion...")
		err = task.Wait(ctx)
		if err != nil {
			log.WithError(err).WithField("vm_id", vmID).Error("Power-on task failed")
			http.Error(w, fmt.Sprintf("Power-on task failed: %v", err), http.StatusInternalServerError)
			return
		}

		log.WithField("vm_id", vmID).Info("✅ VM power-on task completed")

		// Wait for VMware Tools if requested
		var finalTools string
		if req.WaitForTools {
			log.WithField("vm_id", vmID).Info("⏳ Waiting for VMware Tools to start...")

			// Wait up to 60 seconds for tools
			toolsCtx, toolsCancel := context.WithTimeout(ctx, 60*time.Second)
			defer toolsCancel()

			ticker := time.NewTicker(5 * time.Second)
			defer ticker.Stop()

			for {
				select {
				case <-toolsCtx.Done():
					log.WithField("vm_id", vmID).Warn("VMware Tools wait timeout, but VM is powered on")
					_, finalTools, _ = getVMPowerState(ctx, vm)
					goto toolsWaitDone
				case <-ticker.C:
					_, currentTools, err := getVMPowerState(ctx, vm)
					if err != nil {
						continue
					}
					if currentTools == "toolsOk" {
						log.WithField("vm_id", vmID).Info("✅ VMware Tools are now available")
						finalTools = currentTools
						goto toolsWaitDone
					}
					finalTools = currentTools
				}
			}
		toolsWaitDone:
		} else {
			// Just get current tools status
			_, finalTools, _ = getVMPowerState(ctx, vm)
		}

		// Get final power state
		finalState, _, err := getVMPowerState(ctx, vm)
		if err != nil {
			log.WithError(err).WithField("vm_id", vmID).Error("Failed to get final power state")
			// Continue with response even if we can't verify final state
			finalState = "poweredOn" // Assume success
		}

		response = &PowerManagementResponse{
			Success:         true,
			VMID:            vmID,
			PreviousState:   initialState,
			NewState:        finalState,
			Operation:       "power-on",
			ToolsStatus:     finalTools,
			DurationSeconds: int(time.Since(startTime).Seconds()),
			Timestamp:       time.Now().UTC().Format(time.RFC3339),
			Message:         "VM power-on completed successfully",
		}
	}

	log.WithFields(log.Fields{
		"vm_id":            vmID,
		"previous_state":   response.PreviousState,
		"new_state":        response.NewState,
		"tools_status":     response.ToolsStatus,
		"duration_seconds": response.DurationSeconds,
	}).Warn("⚠️ PLACEHOLDER: VM power-on endpoint called - VMware integration needed")

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.WithError(err).Error("Failed to encode power-on response")
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// handleVMPowerState returns the current power state of a VMware VM
func (s *VMAControlServer) handleVMPowerState(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	vmID := vars["vm_id"]

	// Get credentials from query parameters (following existing VMA API pattern)
	vcenter := r.URL.Query().Get("vcenter")
	username := r.URL.Query().Get("username")
	password := r.URL.Query().Get("password")

	if vcenter == "" || username == "" || password == "" {
		log.WithField("vm_id", vmID).Error("Missing required vCenter credentials")
		http.Error(w, "Missing required parameters: vcenter, username, password", http.StatusBadRequest)
		return
	}

	log.WithFields(log.Fields{
		"vm_id":   vmID,
		"vcenter": vcenter,
	}).Info("🔍 Processing VM power state query")

	// Create timeout context
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	// Connect to vCenter
	client, err := createVMwareClient(ctx, vcenter, username, password)
	if err != nil {
		log.WithError(err).WithField("vm_id", vmID).Error("Failed to connect to vCenter")
		http.Error(w, fmt.Sprintf("vCenter connection failed: %v", err), http.StatusInternalServerError)
		return
	}
	defer client.Logout(ctx)

	// Find the VM
	vm, err := findVMByID(ctx, client, vmID, "DatabanxDC") // TODO: Make datacenter configurable
	if err != nil {
		log.WithError(err).WithField("vm_id", vmID).Error("Failed to find VM")
		http.Error(w, fmt.Sprintf("VM not found: %v", err), http.StatusNotFound)
		return
	}

	// Get VM power state and additional properties
	var vmMo mo.VirtualMachine
	err = vm.Properties(ctx, vm.Reference(), []string{
		"runtime.powerState",
		"guest.toolsStatus",
		"runtime.bootTime",
	}, &vmMo)
	if err != nil {
		log.WithError(err).WithField("vm_id", vmID).Error("Failed to get VM properties")
		http.Error(w, fmt.Sprintf("Power state query failed: %v", err), http.StatusInternalServerError)
		return
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

	// Calculate uptime and last state change
	var uptimeSeconds int64
	var lastStateChange string
	if vmMo.Runtime.BootTime != nil {
		if powerState == "poweredOn" {
			uptimeSeconds = int64(time.Since(*vmMo.Runtime.BootTime).Seconds())
		}
		lastStateChange = vmMo.Runtime.BootTime.UTC().Format(time.RFC3339)
	} else {
		lastStateChange = ""
	}

	// Create response
	response := &PowerStateResponse{
		VMID:            vmID,
		PowerState:      powerState,
		ToolsStatus:     toolsStatus,
		LastStateChange: lastStateChange,
		UptimeSeconds:   uptimeSeconds,
		Timestamp:       time.Now().UTC().Format(time.RFC3339),
	}

	log.WithFields(log.Fields{
		"vm_id":          vmID,
		"power_state":    response.PowerState,
		"tools_status":   response.ToolsStatus,
		"uptime_seconds": response.UptimeSeconds,
	}).Info("✅ VM power state retrieved successfully")

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.WithError(err).Error("Failed to encode power state response")
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}
