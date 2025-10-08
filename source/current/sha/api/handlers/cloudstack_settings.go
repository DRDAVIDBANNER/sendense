package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	log "github.com/sirupsen/logrus"
	"github.com/vexxhost/migratekit-sha/database"
	"github.com/vexxhost/migratekit-sha/internal/validation"
	"github.com/vexxhost/migratekit-sha/ossea"
)

// CloudStackSettingsHandler handles CloudStack settings and validation
type CloudStackSettingsHandler struct {
	db database.Connection
}

// NewCloudStackSettingsHandler creates a new settings handler
func NewCloudStackSettingsHandler(db database.Connection) *CloudStackSettingsHandler {
	return &CloudStackSettingsHandler{
		db: db,
	}
}

// CloudStackValidationConnectionRequest represents a connection test request
type CloudStackValidationConnectionRequest struct {
	APIURL    string `json:"api_url"`
	APIKey    string `json:"api_key"`
	SecretKey string `json:"secret_key"`
}

// CloudStackValidationConnectionResponse represents connection test response
type CloudStackValidationConnectionResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Error   string `json:"error,omitempty"`
}

// DetectOMAVMResponse represents SHA VM detection response
type DetectOMAVMResponse struct {
	Success bool                  `json:"success"`
	SHAInfo *validation.SHAVMInfo `json:"oma_info,omitempty"`
	Message string                `json:"message"`
	Error   string                `json:"error,omitempty"`
}

// NetworksResponse represents networks list response
type NetworksResponse struct {
	Success  bool                    `json:"success"`
	Networks []validation.NetworkInfo `json:"networks"`
	Count    int                     `json:"count"`
	Error    string                  `json:"error,omitempty"`
}

// ValidateSettingsRequest represents validation request
type ValidateSettingsRequest struct {
	APIURL            string `json:"api_url"`
	APIKey            string `json:"api_key"`
	SecretKey         string `json:"secret_key"`
	SHAVMID           string `json:"oma_vm_id,omitempty"`
	ServiceOfferingID string `json:"service_offering_id,omitempty"`
	NetworkID         string `json:"network_id,omitempty"`
}

// ValidateSettingsResponse represents validation response
type ValidateSettingsResponse struct {
	Success bool                        `json:"success"`
	Result  *validation.ValidationResult `json:"result"`
	Message string                      `json:"message"`
}

// TestConnection tests CloudStack API connectivity
// POST /api/v1/settings/cloudstack/test-connection
func (h *CloudStackSettingsHandler) TestConnection(w http.ResponseWriter, r *http.Request) {
	var req CloudStackValidationConnectionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, CloudStackValidationConnectionResponse{
			Success: false,
			Message: "Invalid request format",
			Error:   err.Error(),
		})
		return
	}

	log.WithField("api_url", req.APIURL).Info("🔍 Testing CloudStack connection")

	// Create temporary client
	client := ossea.NewClient(req.APIURL, req.APIKey, req.SecretKey, "", "")

	// Try to list zones as a connectivity test
	zones, err := client.ListZones()
	if err != nil {
		log.WithError(err).Warn("CloudStack connection test failed")
		respondJSON(w, http.StatusOK, CloudStackValidationConnectionResponse{
			Success: false,
			Message: "Failed to connect to CloudStack. Please verify your API URL, API key, and secret key.",
			Error:   sanitizeError(err),
		})
		return
	}

	log.WithField("zones_found", len(zones)).Info("✅ CloudStack connection successful")
	respondJSON(w, http.StatusOK, CloudStackValidationConnectionResponse{
		Success: true,
		Message: fmt.Sprintf("Successfully connected to CloudStack. Found %d zone(s).", len(zones)),
	})
}

// DetectOMAVM attempts to auto-detect the SHA VM by MAC address
// POST /api/v1/settings/cloudstack/detect-oma-vm
func (h *CloudStackSettingsHandler) DetectOMAVM(w http.ResponseWriter, r *http.Request) {
	var req CloudStackValidationConnectionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, DetectOMAVMResponse{
			Success: false,
			Message: "Invalid request format",
			Error:   err.Error(),
		})
		return
	}

	log.Info("🔍 Attempting to auto-detect SHA VM by MAC address")

	// Create temporary client
	client := ossea.NewClient(req.APIURL, req.APIKey, req.SecretKey, "", "")
	ctx := context.Background()

	// Create validator and detect SHA VM
	validator := validation.NewCloudStackValidator(client)
	shaInfo, err := validator.DetectOMAVMID(ctx)
	if err != nil {
		log.WithError(err).Warn("SHA VM auto-detection failed")
		respondJSON(w, http.StatusOK, DetectOMAVMResponse{
			Success: false,
			Message: "Could not auto-detect SHA VM. Please enter the VM ID manually.",
			Error:   sanitizeError(err),
		})
		return
	}

	log.WithFields(log.Fields{
		"vm_id":       shaInfo.VMID,
		"vm_name":     shaInfo.VMName,
		"mac_address": shaInfo.MACAddress,
	}).Info("✅ SHA VM detected successfully")

	respondJSON(w, http.StatusOK, DetectOMAVMResponse{
		Success: true,
		SHAInfo: shaInfo,
		Message: fmt.Sprintf("SHA VM detected: %s", shaInfo.VMName),
	})
}

// ListNetworks lists all available CloudStack networks
// GET /api/v1/settings/cloudstack/networks
func (h *CloudStackSettingsHandler) ListNetworks(w http.ResponseWriter, r *http.Request) {
	log.Info("🔍 Listing available CloudStack networks")

	// Load active CloudStack config from database
	config, err := h.getActiveConfig()
	if err != nil {
		log.WithError(err).Error("Failed to load CloudStack configuration")
		respondJSON(w, http.StatusNotFound, NetworksResponse{
			Success: false,
			Error:   "No active CloudStack configuration found. Please configure CloudStack settings first.",
		})
		return
	}

	// Create client
	client := ossea.NewClient(config.APIURL, config.APIKey, config.SecretKey, "", "")
	ctx := context.Background()

	// Create validator and list networks
	validator := validation.NewCloudStackValidator(client)
	networks, err := validator.ListAvailableNetworks(ctx)
	if err != nil {
		log.WithError(err).Error("Failed to list networks")
		respondJSON(w, http.StatusInternalServerError, NetworksResponse{
			Success: false,
			Error:   sanitizeError(err),
		})
		return
	}

	log.WithField("network_count", len(networks)).Info("✅ Listed networks successfully")
	respondJSON(w, http.StatusOK, NetworksResponse{
		Success:  true,
		Networks: networks,
		Count:    len(networks),
	})
}

// ValidateSettings runs all CloudStack prerequisite validations
// POST /api/v1/settings/cloudstack/validate
func (h *CloudStackSettingsHandler) ValidateSettings(w http.ResponseWriter, r *http.Request) {
	var req ValidateSettingsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, ValidateSettingsResponse{
			Success: false,
			Message: "Invalid request format",
		})
		return
	}

	log.WithFields(log.Fields{
		"api_url":             req.APIURL,
		"has_oma_vm_id":       req.SHAVMID != "",
		"has_offering_id":     req.ServiceOfferingID != "",
		"has_network_id":      req.NetworkID != "",
	}).Info("🔍 Running CloudStack validation")

	// Create temporary client
	client := ossea.NewClient(req.APIURL, req.APIKey, req.SecretKey, "", "")
	ctx := context.Background()

	// Create validator and run all validations
	validator := validation.NewCloudStackValidator(client)
	result := validator.ValidateAll(ctx, req.SHAVMID, req.ServiceOfferingID, req.NetworkID)

	// Determine message based on result
	var message string
	switch result.OverallStatus {
	case "pass":
		message = "All validations passed. CloudStack is ready for VM replication."
	case "warning":
		message = "Some validations have warnings. Review the details before proceeding."
	case "fail":
		message = "Critical validation failures detected. Cannot proceed with replication until issues are resolved."
	default:
		message = "Validation completed with unknown status."
	}

	log.WithField("overall_status", result.OverallStatus).Info("✅ Validation complete")
	respondJSON(w, http.StatusOK, ValidateSettingsResponse{
		Success: result.OverallStatus != "fail",
		Result:  result,
		Message: message,
	})
}

// getActiveConfig retrieves the active CloudStack configuration from database
func (h *CloudStackSettingsHandler) getActiveConfig() (*CloudStackConfig, error) {
	type DBConfig struct {
		APIURL            string
		APIKey            string
		SecretKey         string
		NetworkID         string
		ServiceOfferingID string
		SHAVMID           string
	}

	var config DBConfig
	err := h.db.GetGormDB().Raw(`
		SELECT 
			api_url as APIURL,
			api_key as APIKey,
			secret_key as SecretKey,
			network_id as NetworkID,
			service_offering_id as ServiceOfferingID,
			oma_vm_id as SHAVMID
		FROM ossea_configs 
		WHERE is_active = 1 
		LIMIT 1
	`).Scan(&config).Error

	if err != nil {
		return nil, fmt.Errorf("failed to query database: %w", err)
	}

	if config.APIURL == "" {
		return nil, fmt.Errorf("no active configuration found")
	}

	return &CloudStackConfig{
		APIURL:            config.APIURL,
		APIKey:            config.APIKey,
		SecretKey:         config.SecretKey,
		NetworkID:         config.NetworkID,
		ServiceOfferingID: config.ServiceOfferingID,
		SHAVMID:           config.SHAVMID,
	}, nil
}

// CloudStackConfig holds CloudStack configuration
type CloudStackConfig struct {
	APIURL            string
	APIKey            string
	SecretKey         string
	NetworkID         string
	ServiceOfferingID string
	SHAVMID           string
}

// sanitizeError converts technical errors to user-friendly messages
func sanitizeError(err error) string {
	if err == nil {
		return ""
	}

	errStr := err.Error()
	
	// Common error patterns
	if contains(errStr, "401") || contains(errStr, "unable to verify user credentials") {
		return "Authentication failed. Please verify your API key and secret key are correct."
	}
	if contains(errStr, "404") || contains(errStr, "not found") {
		return "Resource not found. Please check your configuration."
	}
	if contains(errStr, "connection refused") || contains(errStr, "no such host") {
		return "Cannot connect to CloudStack. Please verify the API URL is correct and the server is accessible."
	}
	if contains(errStr, "timeout") {
		return "Connection timed out. CloudStack server may be slow or unreachable."
	}
	if contains(errStr, "iscustomized") {
		return "The selected compute offering does not support custom VM specifications. Please select an offering with customizable CPU, memory, and disk size."
	}
	if contains(errStr, "account") && contains(errStr, "does not match") {
		return "The API key belongs to a different CloudStack account than the SHA VM. Please use credentials from the same account."
	}

	// Default: return first line of error (avoid stack traces)
	return errStr
}

// contains checks if a string contains a substring (case-insensitive)
// DiscoverAllResources combines connection test, SHA VM detection, and resource discovery
// POST /api/v1/settings/cloudstack/discover-all
func (h *CloudStackSettingsHandler) DiscoverAllResources(w http.ResponseWriter, r *http.Request) {
	var req struct {
		APIURL    string `json:"api_url"`
		APIKey    string `json:"api_key"`
		SecretKey string `json:"secret_key"`
		Domain    string `json:"domain"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{
			"error": "Invalid request payload"})
		return
	}

	// Step 1: Test connection
	client := ossea.NewClient(req.APIURL, req.APIKey, req.SecretKey, req.Domain, "")
	ctx := context.Background()
	
	// Test with a simple API call
	zones, err := client.ListZones()
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{
			"error": fmt.Sprintf("Failed to connect to CloudStack: %v", sanitizeError(err))})
		return
	}

	if len(zones) == 0 {
		respondJSON(w, http.StatusBadRequest, map[string]string{
			"error": "Connected but no zones found in CloudStack"})
		return
	}

	// Step 2: Detect SHA VM by MAC address
	validator := validation.NewCloudStackValidator(client)
	shaInfo, err := validator.DetectOMAVMID(ctx)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("Failed to detect SHA VM: %v", sanitizeError(err))})
		return
	}

	// Step 3: Discover all resources
	// Zones
	zonesList := make([]map[string]interface{}, 0)
	for _, zone := range zones {
		zonesList = append(zonesList, map[string]interface{}{
			"id":   zone.ID,
			"name": zone.Name,
		})
	}

	// Templates (use "executable" filter to get all usable templates)
	templates, err := client.ListTemplates("executable")
	if err != nil {
		log.WithError(err).Warn("Failed to list templates, trying 'featured' filter")
		// Try featured templates as fallback
		templates, err = client.ListTemplates("featured")
		if err != nil {
			log.WithError(err).Warn("Failed to list featured templates, returning empty list")
			templates = []ossea.Template{} // Empty list on error
		}
	}
	templatesList := make([]map[string]interface{}, 0)
	const flexibleTemplateSizeThreshold = int64(2 * 1024 * 1024 * 1024) // 2 GB threshold
	
	for _, template := range templates {
		sizeGB := float64(template.Size) / (1024 * 1024 * 1024)
		
		// Only include ready templates with flexible root disk (Size < 2 GB)
		// CloudStack uses the template Size as the minimum root disk size
		// Templates with large sizes (e.g., 100 GB) will fail during failover when source VM has smaller disk
		// Flexible templates have very small sizes (< 2 GB), indicating they allow dynamic root disk sizing
		if template.IsReady && template.Size < flexibleTemplateSizeThreshold {
			log.WithFields(log.Fields{
				"template_name": template.Name,
				"template_id":   template.ID,
				"size_bytes":    template.Size,
				"size_gb":       sizeGB,
			}).Info("✅ Flexible template (Size < 2 GB) - allows dynamic root disk sizing")
			
			templatesList = append(templatesList, map[string]interface{}{
				"id":          template.ID,
				"name":        template.Name,
				"description": template.DisplayText,
				"os_type":     template.OSTypeName,
				"size_gb":     sizeGB,
			})
		} else if template.IsReady {
			log.WithFields(log.Fields{
				"template_name": template.Name,
				"size_gb":       sizeGB,
			}).Debug("❌ Template filtered out - fixed root disk size too large for failover flexibility")
		}
	}

	// Service Offerings
	serviceOfferings, err := client.ListServiceOfferings()
	if err != nil {
		serviceOfferings = []ossea.ServiceOffering{} // Empty list on error
	}
	offeringsList := make([]map[string]interface{}, 0)
	for _, offering := range serviceOfferings {
		offeringsList = append(offeringsList, map[string]interface{}{
			"id":          offering.ID,
			"name":        offering.Name,
			"description": offering.DisplayText,
			"cpu":         offering.CPUNumber,
			"memory":      offering.Memory,
		})
	}

	// Disk Offerings
	diskOfferings, err := client.ListDiskOfferings()
	if err != nil {
		diskOfferings = []ossea.DiskOffering{} // Empty list on error
	}
	diskOfferingsList := make([]map[string]interface{}, 0)
	for _, offering := range diskOfferings {
		diskOfferingsList = append(diskOfferingsList, map[string]interface{}{
			"id":           offering.ID,
			"name":         offering.Name,
			"description":  offering.DisplayText,
			"disk_size_gb": offering.DiskSize,
		})
	}

	// Networks
	networks, err := validator.ListAvailableNetworks(ctx)
	if err != nil {
		networks = []validation.NetworkInfo{} // Empty list on error
	}
	networksList := make([]map[string]interface{}, 0)
	for _, network := range networks {
		networksList = append(networksList, map[string]interface{}{
			"id":        network.ID,
			"name":      network.Name,
			"zone_id":   network.ZoneID,
			"zone_name": network.ZoneName,
			"state":     network.State,
		})
	}

	// Build response with all discovered resources
	response := map[string]interface{}{
		"oma_vm_id":         shaInfo.VMID,
		"oma_vm_name":       shaInfo.VMName,
		"zones":             zonesList,
		"templates":         templatesList,
		"service_offerings": offeringsList,
		"disk_offerings":    diskOfferingsList,
		"networks":          networksList,
	}

	log.WithFields(log.Fields{
		"oma_vm_id":         shaInfo.VMID,
		"zones":             len(zones),
		"templates":         len(templates),
		"service_offerings": len(serviceOfferings),
		"disk_offerings":    len(diskOfferings),
		"networks":          len(networks),
	}).Info("✅ CloudStack resource discovery completed successfully")

	respondJSON(w, http.StatusOK, response)
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && 
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || 
		 findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// respondJSON sends a JSON response
func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

