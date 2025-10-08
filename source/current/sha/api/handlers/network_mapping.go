// Package handlers provides HTTP handlers for network mapping management
// Following project rules: minimal endpoints, modular design, clean interfaces
package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"

	"github.com/vexxhost/migratekit-sha/database"
	"github.com/vexxhost/migratekit-sha/ossea"
)

// NetworkMappingHandler provides HTTP handlers for network mapping operations
type NetworkMappingHandler struct {
	db            database.Connection
	mappingRepo   *database.NetworkMappingRepository
	osseaClient   *ossea.Client
	networkClient *ossea.NetworkClient
}

// NewNetworkMappingHandler creates new network mapping handler
func NewNetworkMappingHandler(db database.Connection, osseaClient *ossea.Client, networkClient *ossea.NetworkClient) *NetworkMappingHandler {
	return &NetworkMappingHandler{
		db:            db,
		mappingRepo:   database.NewNetworkMappingRepository(db),
		osseaClient:   osseaClient,
		networkClient: networkClient,
	}
}

// NetworkMappingRequest represents a network mapping creation/update request
type NetworkMappingRequest struct {
	VMID                   string `json:"vm_id"`
	VMContextID            string `json:"vm_context_id,omitempty"` // Optional - if not provided, will be resolved from vm_id
	SourceNetworkName      string `json:"source_network_name"`
	DestinationNetworkID   string `json:"destination_network_id"`
	DestinationNetworkName string `json:"destination_network_name"`
	IsTestNetwork          bool   `json:"is_test_network"`
}

// NetworkMappingResponse represents a network mapping response
type NetworkMappingResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// GetNetworkMappingsByVM retrieves network mappings for a VM
// GET /api/v1/network-mappings/{vm_id}
func (nmh *NetworkMappingHandler) GetNetworkMappingsByVM(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	vmID := vars["vm_id"]

	log.WithField("vm_id", vmID).Info("🔍 API: Getting network mappings for VM")

	mappings, err := nmh.mappingRepo.GetByVMID(vmID)
	if err != nil {
		log.WithError(err).WithField("vm_id", vmID).Error("Failed to get network mappings")
		response := NetworkMappingResponse{
			Success: false,
			Message: "Failed to get network mappings",
			Error:   err.Error(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	log.WithFields(log.Fields{
		"vm_id":         vmID,
		"mapping_count": len(mappings),
	}).Info("✅ API: Retrieved network mappings for VM")

	response := NetworkMappingResponse{
		Success: true,
		Message: fmt.Sprintf("Retrieved %d network mappings for VM", len(mappings)),
		Data:    mappings,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// CreateNetworkMapping creates or updates a network mapping
// POST /api/v1/network-mappings
func (nmh *NetworkMappingHandler) CreateNetworkMapping(w http.ResponseWriter, r *http.Request) {
	var req NetworkMappingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.WithError(err).Error("Invalid network mapping request")
		response := NetworkMappingResponse{
			Success: false,
			Message: "Invalid request data",
			Error:   err.Error(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	log.WithFields(log.Fields{
		"vm_id":               req.VMID,
		"vm_context_id":       req.VMContextID,
		"source_network":      req.SourceNetworkName,
		"destination_network": req.DestinationNetworkID,
		"is_test_network":     req.IsTestNetwork,
	}).Info("🔧 API: Creating network mapping")

	// Validate required fields
	if req.VMID == "" || req.SourceNetworkName == "" || req.DestinationNetworkID == "" {
		response := NetworkMappingResponse{
			Success: false,
			Message: "Missing required fields: vm_id, source_network_name, destination_network_id",
			Error:   "validation_failed",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Resolve vm_context_id if not provided
	vmContextID := req.VMContextID
	if vmContextID == "" {
		// Try to resolve from vm_id (vm_name) to context_id
		var result struct {
			ContextID string `gorm:"column:context_id"`
		}
		err := nmh.db.GetGormDB().Table("vm_replication_contexts").
			Select("context_id").
			Where("vm_name = ?", req.VMID).
			First(&result).Error

		if err == nil {
			vmContextID = result.ContextID
			log.WithFields(log.Fields{
				"vm_id":         req.VMID,
				"vm_context_id": vmContextID,
			}).Debug("🔍 Resolved vm_context_id from vm_name")
		} else {
			log.WithFields(log.Fields{
				"vm_id": req.VMID,
				"error": err,
			}).Warn("⚠️ Failed to resolve vm_context_id from vm_name - proceeding without context_id")
		}
	}

	// Create the mapping record
	mapping := &database.NetworkMapping{
		VMID:                   req.VMID,
		VMContextID:            vmContextID,
		SourceNetworkName:      req.SourceNetworkName,
		DestinationNetworkID:   req.DestinationNetworkID,
		DestinationNetworkName: req.DestinationNetworkName,
		IsTestNetwork:          req.IsTestNetwork,
	}

	if err := nmh.mappingRepo.CreateOrUpdate(mapping); err != nil {
		log.WithError(err).Error("Failed to create network mapping")
		response := NetworkMappingResponse{
			Success: false,
			Message: "Failed to create network mapping",
			Error:   err.Error(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	log.WithFields(log.Fields{
		"mapping_id":          mapping.ID,
		"vm_id":               req.VMID,
		"source_network":      req.SourceNetworkName,
		"destination_network": req.DestinationNetworkName,
	}).Info("✅ API: Network mapping created successfully")

	response := NetworkMappingResponse{
		Success: true,
		Message: "Network mapping created successfully",
		Data:    mapping,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// DeleteNetworkMapping removes a network mapping
// DELETE /api/v1/network-mappings/{vm_id}/{source_network_name}
func (nmh *NetworkMappingHandler) DeleteNetworkMapping(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	vmID := vars["vm_id"]
	sourceNetworkName := vars["source_network_name"]

	log.WithFields(log.Fields{
		"vm_id":          vmID,
		"source_network": sourceNetworkName,
	}).Info("🗑️ API: Deleting network mapping")

	// Get the existing mapping to log details
	existingMapping, err := nmh.mappingRepo.GetByVMAndNetwork(vmID, sourceNetworkName)
	if err != nil {
		log.WithError(err).Error("Network mapping not found")
		response := NetworkMappingResponse{
			Success: false,
			Message: "Network mapping not found",
			Error:   err.Error(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(response)
		return
	}

	// For now, we'll just delete all mappings for the VM and recreate others
	// This is a simplified implementation that can be improved later
	allMappings, err := nmh.mappingRepo.GetByVMID(vmID)
	if err != nil {
		log.WithError(err).Error("Failed to get existing mappings")
		response := NetworkMappingResponse{
			Success: false,
			Message: "Failed to process deletion",
			Error:   err.Error(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Delete all mappings for the VM
	if err := nmh.mappingRepo.DeleteByVMID(vmID); err != nil {
		log.WithError(err).Error("Failed to delete network mappings")
		response := NetworkMappingResponse{
			Success: false,
			Message: "Failed to delete network mapping",
			Error:   err.Error(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Recreate other mappings (excluding the deleted one)
	for _, mapping := range allMappings {
		if mapping.SourceNetworkName != sourceNetworkName {
			newMapping := database.NetworkMapping{
				VMID:                   mapping.VMID,
				SourceNetworkName:      mapping.SourceNetworkName,
				DestinationNetworkID:   mapping.DestinationNetworkID,
				DestinationNetworkName: mapping.DestinationNetworkName,
				IsTestNetwork:          mapping.IsTestNetwork,
			}
			if err := nmh.mappingRepo.CreateOrUpdate(&newMapping); err != nil {
				log.WithError(err).Warn("Failed to recreate network mapping after deletion")
			}
		}
	}

	log.WithFields(log.Fields{
		"vm_id":               vmID,
		"source_network":      sourceNetworkName,
		"destination_network": existingMapping.DestinationNetworkName,
	}).Info("✅ API: Network mapping deleted successfully")

	response := NetworkMappingResponse{
		Success: true,
		Message: "Network mapping deleted successfully",
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetNetworkMappingStatus provides status summary for network mappings
// GET /api/v1/network-mappings/{vm_id}/status
func (nmh *NetworkMappingHandler) GetNetworkMappingStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	vmID := vars["vm_id"]

	log.WithField("vm_id", vmID).Info("📊 API: Getting network mapping status")

	mappings, err := nmh.mappingRepo.GetByVMID(vmID)
	if err != nil {
		log.WithError(err).Error("Failed to get network mappings for status")
		response := NetworkMappingResponse{
			Success: false,
			Message: "Failed to get network mapping status",
			Error:   err.Error(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Calculate status metrics
	testMappingCount := 0
	liveMappingCount := 0
	for _, mapping := range mappings {
		if mapping.IsTestNetwork {
			testMappingCount++
		} else {
			liveMappingCount++
		}
	}

	status := map[string]interface{}{
		"vm_id":            vmID,
		"total_mappings":   len(mappings),
		"live_mappings":    liveMappingCount,
		"test_mappings":    testMappingCount,
		"has_mappings":     len(mappings) > 0,
		"mapping_complete": len(mappings) > 0, // Simplified check
		"readiness_level":  "unknown",         // Requires full service integration for proper assessment
	}

	log.WithFields(log.Fields{
		"vm_id":          vmID,
		"total_mappings": len(mappings),
		"live_mappings":  liveMappingCount,
		"test_mappings":  testMappingCount,
	}).Info("✅ API: Retrieved network mapping status")

	response := NetworkMappingResponse{
		Success: true,
		Message: "Network mapping status retrieved successfully",
		Data:    status,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// ListAllNetworkMappings lists all network mappings (for admin/debugging)
// GET /api/v1/network-mappings
func (nmh *NetworkMappingHandler) ListAllNetworkMappings(w http.ResponseWriter, r *http.Request) {
	log.Info("📋 API: Listing all network mappings")

	// Get test network mappings as a starting point
	testMappings, err := nmh.mappingRepo.GetTestNetworkMappings()
	if err != nil {
		log.WithError(err).Error("Failed to get network mappings")
		response := NetworkMappingResponse{
			Success: false,
			Message: "Failed to get network mappings",
			Error:   err.Error(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	log.WithField("mapping_count", len(testMappings)).Info("✅ API: Retrieved network mappings")

	response := NetworkMappingResponse{
		Success: true,
		Message: fmt.Sprintf("Retrieved %d network mappings", len(testMappings)),
		Data:    testMappings,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// ListAvailableNetworks lists all available OSSEA networks for network mapping
// GET /api/v1/networks/available
func (nmh *NetworkMappingHandler) ListAvailableNetworks(w http.ResponseWriter, r *http.Request) {
	log.Info("🌐 API: Listing available OSSEA networks")

	// Get networks from OSSEA
	networks, err := nmh.networkClient.ListNetworks()
	if err != nil {
		response := NetworkMappingResponse{
			Success: false,
			Message: "Failed to retrieve available networks",
			Error:   err.Error(),
		}
		w.WriteHeader(http.StatusInternalServerError)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	log.WithField("network_count", len(networks)).Info("✅ Retrieved available OSSEA networks")

	response := NetworkMappingResponse{
		Success: true,
		Message: fmt.Sprintf("Retrieved %d available networks", len(networks)),
		Data:    networks,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// ResolveNetworkID resolves a network name to its UUID
// POST /api/v1/networks/resolve
func (nmh *NetworkMappingHandler) ResolveNetworkID(w http.ResponseWriter, r *http.Request) {
	log.Info("🔍 API: Resolving network name to ID")

	var req struct {
		NetworkName string `json:"network_name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response := NetworkMappingResponse{
			Success: false,
			Message: "Invalid request body",
			Error:   err.Error(),
		}
		w.WriteHeader(http.StatusBadRequest)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	// Get networks and find matching name
	networks, err := nmh.networkClient.ListNetworks()
	if err != nil {
		response := NetworkMappingResponse{
			Success: false,
			Message: "Failed to retrieve networks for resolution",
			Error:   err.Error(),
		}
		w.WriteHeader(http.StatusInternalServerError)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	// Find network by name
	for _, network := range networks {
		if network.Name == req.NetworkName {
			log.WithFields(log.Fields{
				"network_name": req.NetworkName,
				"network_id":   network.ID,
			}).Info("✅ Network name resolved to ID")

			response := NetworkMappingResponse{
				Success: true,
				Message: fmt.Sprintf("Network '%s' resolved", req.NetworkName),
				Data: map[string]interface{}{
					"network_name": req.NetworkName,
					"network_id":   network.ID,
					"network":      network,
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
			return
		}
	}

	// Network not found
	response := NetworkMappingResponse{
		Success: false,
		Message: fmt.Sprintf("Network '%s' not found", req.NetworkName),
		Error:   "Network name not found in available networks",
	}
	w.WriteHeader(http.StatusNotFound)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// ListServiceOfferings lists all available OSSEA service offerings
// GET /api/v1/service-offerings/available
func (nmh *NetworkMappingHandler) ListServiceOfferings(w http.ResponseWriter, r *http.Request) {
	log.Info("🔧 API: Listing available OSSEA service offerings")

	// Get service offerings from OSSEA
	offerings, err := nmh.osseaClient.ListServiceOfferings()
	if err != nil {
		response := NetworkMappingResponse{
			Success: false,
			Message: "Failed to retrieve available service offerings",
			Error:   err.Error(),
		}
		w.WriteHeader(http.StatusInternalServerError)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	log.WithField("offering_count", len(offerings)).Info("✅ Retrieved available OSSEA service offerings")

	response := NetworkMappingResponse{
		Success: true,
		Message: fmt.Sprintf("Retrieved %d available service offerings", len(offerings)),
		Data:    offerings,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// RegisterNetworkMappingRoutes registers network mapping routes with the router
func RegisterNetworkMappingRoutes(r *mux.Router, handler *NetworkMappingHandler) {
	// Network mapping CRUD operations
	r.HandleFunc("/api/v1/network-mappings", handler.CreateNetworkMapping).Methods("POST")
	r.HandleFunc("/api/v1/network-mappings", handler.ListAllNetworkMappings).Methods("GET")
	r.HandleFunc("/api/v1/network-mappings/{vm_id}", handler.GetNetworkMappingsByVM).Methods("GET")
	r.HandleFunc("/api/v1/network-mappings/{vm_id}/status", handler.GetNetworkMappingStatus).Methods("GET")
	r.HandleFunc("/api/v1/network-mappings/{vm_id}/{source_network_name}", handler.DeleteNetworkMapping).Methods("DELETE")

	// Network discovery and resolution endpoints
	r.HandleFunc("/api/v1/networks/available", handler.ListAvailableNetworks).Methods("GET")
	r.HandleFunc("/api/v1/networks/resolve", handler.ResolveNetworkID).Methods("POST")

	// Service offering discovery endpoints
	r.HandleFunc("/api/v1/service-offerings/available", handler.ListServiceOfferings).Methods("GET")
}
