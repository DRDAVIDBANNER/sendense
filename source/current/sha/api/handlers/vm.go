// Package handlers provides HTTP handlers for SHA API endpoints
// VM inventory management following project rules: modular design, clean interfaces
package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"

	"github.com/vexxhost/migratekit-sha/database"
)

// VMHandler handles VM inventory endpoints
type VMHandler struct {
	db          database.Connection
	vmInventory []VMInfo // In-memory storage for now
}

// VMInfo represents VM information
type VMInfo struct {
	ID         string        `json:"id"`
	Name       string        `json:"name"`
	CPUs       int           `json:"cpus"`
	MemoryMB   int           `json:"memory_mb"`
	PowerState string        `json:"power_state"`
	OSType     string        `json:"os_type"`
	VMXVersion string        `json:"vmx_version"`
	Disks      []DiskInfo    `json:"disks"`
	Networks   []NetworkInfo `json:"networks"`
}

// DiskInfo represents disk information
type DiskInfo struct {
	ID               string `json:"id"`
	Label            string `json:"label"`
	CapacityBytes    int64  `json:"capacity_bytes"`
	UsedBytes        int64  `json:"used_bytes"`
	ProvisioningType string `json:"provisioning_type"`
	UnitNumber       int    `json:"unit_number"`
	VMDKPath         string `json:"vmdk_path"`
}

// NetworkInfo represents network interface information
type NetworkInfo struct {
	Label       string `json:"label"`
	NetworkName string `json:"network_name"`
	MACAddress  string `json:"mac_address"`
	AdapterType string `json:"adapter_type"`
	Connected   bool   `json:"connected"`
}

// VMInventoryRequest represents VM inventory submission
type VMInventoryRequest struct {
	VCenter struct {
		Host       string `json:"host"`
		Datacenter string `json:"datacenter"`
	} `json:"vcenter"`
	VMs       []VMInfo  `json:"vms"`
	Timestamp time.Time `json:"timestamp"`
}

// NewVMHandler creates a new VM handler
func NewVMHandler(db database.Connection) *VMHandler {
	return &VMHandler{
		db:          db,
		vmInventory: make([]VMInfo, 0),
	}
}

// List handles VM list requests
// @Summary List VMs
// @Description Get list of all VMs discovered from VMware inventory
// @Tags inventory
// @Security BearerAuth
// @Produce json
// @Success 200 {array} VMInfo
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/vms [get]
func (h *VMHandler) List(w http.ResponseWriter, r *http.Request) {
	h.writeJSONResponse(w, http.StatusOK, h.vmInventory)
}

// ReceiveInventory handles VM inventory submission
// @Summary Receive VM inventory
// @Description Accept VM inventory payload from VMware appliance
// @Tags inventory
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param inventory body VMInventoryRequest true "VM inventory data"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} ErrorResponse
// @Router /api/v1/vms/inventory [post]
func (h *VMHandler) ReceiveInventory(w http.ResponseWriter, r *http.Request) {
	var req VMInventoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid request payload", err.Error())
		return
	}

	// Store VMs in memory (TODO: store in database when DB integration is complete)
	h.vmInventory = req.VMs

	log.WithFields(log.Fields{
		"vcenter_host": req.VCenter.Host,
		"vm_count":     len(req.VMs),
		"timestamp":    req.Timestamp,
	}).Info("Received VM inventory from SNA")

	response := map[string]interface{}{
		"message":   "VM inventory received successfully",
		"vm_count":  len(req.VMs),
		"timestamp": time.Now().Format(time.RFC3339),
	}

	h.writeJSONResponse(w, http.StatusOK, response)
}

// GetByID handles VM lookup by ID
// @Summary Get VM by ID
// @Description Get detailed information about a specific VM
// @Tags inventory
// @Security BearerAuth
// @Produce json
// @Param id path string true "VM ID"
// @Success 200 {object} VMInfo
// @Failure 404 {object} ErrorResponse
// @Router /api/v1/vms/{id} [get]
func (h *VMHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	vmID := vars["id"]

	for _, vm := range h.vmInventory {
		if vm.ID == vmID {
			h.writeJSONResponse(w, http.StatusOK, vm)
			return
		}
	}

	h.writeErrorResponse(w, http.StatusNotFound, "VM not found", "")
}

// Helper functions

// writeJSONResponse writes a standardized JSON response
func (h *VMHandler) writeJSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.WithError(err).Error("Failed to encode JSON response")
	}
}

// writeErrorResponse writes a standardized error response
func (h *VMHandler) writeErrorResponse(w http.ResponseWriter, statusCode int, message, details string) {
	response := map[string]interface{}{
		"error":     message,
		"details":   details,
		"timestamp": time.Now().Format(time.RFC3339),
	}
	h.writeJSONResponse(w, statusCode, response)
}
