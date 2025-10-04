// Package handlers provides streamlined OSSEA configuration with auto-discovery
package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/vexxhost/migratekit-oma/database"
	"github.com/vexxhost/migratekit-oma/ossea"
)

// StreamlinedOSSEAConfigHandler provides simplified OSSEA configuration
type StreamlinedOSSEAConfigHandler struct {
	db database.Connection
}

// NewStreamlinedOSSEAConfigHandler creates a new streamlined config handler
func NewStreamlinedOSSEAConfigHandler(db database.Connection) *StreamlinedOSSEAConfigHandler {
	return &StreamlinedOSSEAConfigHandler{
		db: db,
	}
}

// CloudStackConnectionRequest represents the simplified connection input
type CloudStackConnectionRequest struct {
	BaseURL   string `json:"base_url" binding:"required"` // e.g., "10.245.241.101:8080"
	APIKey    string `json:"api_key" binding:"required"`
	SecretKey string `json:"secret_key" binding:"required"`
	Domain    string `json:"domain,omitempty"` // Optional domain name (e.g., "151")
}

// CloudStackResourcesResponse represents discovered resources
type CloudStackResourcesResponse struct {
	Success          bool                    `json:"success"`
	Zones            []ZoneOption            `json:"zones"`
	Domains          []DomainOption          `json:"domains"`
	Templates        []TemplateOption        `json:"templates"`
	ServiceOfferings []ServiceOfferingOption `json:"service_offerings"`
	DiskOfferings    []DiskOfferingOption    `json:"disk_offerings"`
	Networks         []NetworkOption         `json:"networks"`
}

// Resource option types for dropdowns
type ZoneOption struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

type DomainOption struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Path string `json:"path,omitempty"`
}

type TemplateOption struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	OSType      string `json:"os_type"`
}

type ServiceOfferingOption struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	CPU         int    `json:"cpu"`
	Memory      int    `json:"memory"`
}

type NetworkOption struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	ZoneID      string `json:"zone_id,omitempty"`
}

type DiskOfferingOption struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	DiskSize    int64  `json:"disk_size_gb"`
}

// Complete configuration request
type StreamlinedConfigRequest struct {
	BaseURL           string `json:"base_url" binding:"required"`
	APIKey            string `json:"api_key" binding:"required"`
	SecretKey         string `json:"secret_key" binding:"required"`
	ZoneID            string `json:"zone_id" binding:"required"`
	DomainName        string `json:"domain_name" binding:"required"`
	TemplateID        string `json:"template_id" binding:"required"`
	ServiceOfferingID string `json:"service_offering_id" binding:"required"`
	DiskOfferingID    string `json:"disk_offering_id" binding:"required"`
	NetworkID         string `json:"network_id" binding:"required"`
	Domain            string `json:"domain,omitempty"` // Use manually entered domain
	OMAVmID           string `json:"oma_vm_id" binding:"required"`
}

// TestConnection tests CloudStack connection and discovers resources
// POST /api/v1/ossea/discover-resources
func (h *StreamlinedOSSEAConfigHandler) DiscoverResources(w http.ResponseWriter, r *http.Request) {
	var req CloudStackConnectionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request format", http.StatusBadRequest)
		return
	}

	// Safe API key logging (handle short keys)
	apiKeyPreview := req.APIKey
	if len(req.APIKey) > 10 {
		apiKeyPreview = req.APIKey[:10] + "..."
	}

	log.WithFields(log.Fields{
		"base_url": req.BaseURL,
		"api_key":  apiKeyPreview,
	}).Info("🔍 Discovering CloudStack resources")

	// Build full API URL with smart protocol detection
	var fullAPIURL string
	baseURL := req.BaseURL

	// Fix common typo: https// -> https://
	if strings.HasPrefix(baseURL, "https//") {
		baseURL = strings.Replace(baseURL, "https//", "https://", 1)
	} else if strings.HasPrefix(baseURL, "http//") {
		baseURL = strings.Replace(baseURL, "http//", "http://", 1)
	}

	// Add protocol if not present
	if !strings.HasPrefix(baseURL, "http://") && !strings.HasPrefix(baseURL, "https://") {
		fullAPIURL = "http://" + baseURL + "/client/api"
	} else {
		fullAPIURL = baseURL + "/client/api"
	}

	// Create temporary OSSEA client for discovery
	client := ossea.NewClient(
		fullAPIURL,
		req.APIKey,
		req.SecretKey,
		req.Domain, // Use provided domain or empty for ROOT
		"",         // Zone will be discovered
	)

	response := CloudStackResourcesResponse{
		Success: false,
	}

	// Test connection and discover zones
	zones, err := client.ListZones()
	if err != nil {
		log.WithError(err).Error("Failed to discover zones")
		http.Error(w, "Failed to connect to CloudStack: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Convert zones to options
	for _, zone := range zones {
		response.Zones = append(response.Zones, ZoneOption{
			ID:   zone.ID,
			Name: zone.Name,
		})
	}

	// Discover templates (use first zone for template discovery)
	if len(zones) > 0 {
		templates, err := client.ListTemplates("featured")
		if err != nil {
			log.WithError(err).Warn("Failed to discover templates")
		} else {
			for _, template := range templates {
				response.Templates = append(response.Templates, TemplateOption{
					ID:          template.ID,
					Name:        template.DisplayText,
					Description: template.OSTypeName,
					OSType:      template.OSTypeName,
				})
			}
		}
	}

	// Discover service offerings
	offerings, err := client.ListServiceOfferings()
	if err != nil {
		log.WithError(err).Warn("Failed to discover service offerings")
	} else {
		for _, offering := range offerings {
			response.ServiceOfferings = append(response.ServiceOfferings, ServiceOfferingOption{
				ID:          offering.ID,
				Name:        offering.DisplayText,
				Description: offering.DisplayText + " - " + offering.Name,
				CPU:         offering.CPUNumber,
				Memory:      offering.Memory,
			})
		}
	}

	// Discover disk offerings
	diskOfferings, err := client.ListDiskOfferings()
	if err != nil {
		log.WithError(err).Warn("Failed to discover disk offerings")
	} else {
		for _, offering := range diskOfferings {
			response.DiskOfferings = append(response.DiskOfferings, DiskOfferingOption{
				ID:          offering.ID,
				Name:        offering.DisplayText,
				Description: offering.DisplayText,
				DiskSize:    offering.DiskSize,
			})
		}
	}

	// Discover networks
	networks, err := client.ListNetworks()
	if err != nil {
		log.WithError(err).Warn("Failed to discover networks")
	} else {
		for _, network := range networks {
			response.Networks = append(response.Networks, NetworkOption{
				ID:          network.ID,
				Name:        network.Name,
				Description: network.DisplayText,
				ZoneID:      network.ZoneID,
			})
		}
	}

	// Use provided domain or discover domains
	if req.Domain != "" {
		// User provided domain name - use it directly
		response.Domains = []DomainOption{
			{
				ID:   req.Domain, // Use domain name as ID for simplicity
				Name: req.Domain,
				Path: "/" + req.Domain,
			},
		}
	} else {
		// No domain provided - use common defaults
		response.Domains = []DomainOption{
			{
				ID:   "ROOT",
				Name: "ROOT",
				Path: "/",
			},
		}
	}

	response.Success = true

	log.WithFields(log.Fields{
		"zones":             len(response.Zones),
		"domains":           len(response.Domains),
		"templates":         len(response.Templates),
		"service_offerings": len(response.ServiceOfferings),
	}).Info("✅ CloudStack resource discovery completed")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// SaveStreamlinedConfig saves the streamlined configuration
// POST /api/v1/ossea/config-streamlined
func (h *StreamlinedOSSEAConfigHandler) SaveStreamlinedConfig(w http.ResponseWriter, r *http.Request) {
	var req StreamlinedConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request format", http.StatusBadRequest)
		return
	}

	log.WithField("base_url", req.BaseURL).Info("💾 Saving streamlined OSSEA configuration")

	// Build full API URL with smart protocol detection
	var fullAPIURL string
	baseURL := req.BaseURL

	// Fix common typo: https// -> https://
	if strings.HasPrefix(baseURL, "https//") {
		baseURL = strings.Replace(baseURL, "https//", "https://", 1)
	} else if strings.HasPrefix(baseURL, "http//") {
		baseURL = strings.Replace(baseURL, "http//", "http://", 1)
	}

	// Add protocol if not present
	if !strings.HasPrefix(baseURL, "http://") && !strings.HasPrefix(baseURL, "https://") {
		fullAPIURL = "http://" + baseURL + "/client/api"
	} else {
		fullAPIURL = baseURL + "/client/api"
	}

	// Create OSSEA configuration
	config := &database.OSSEAConfig{
		Name:              "production-ossea",
		APIURL:            fullAPIURL,
		APIKey:            req.APIKey,
		SecretKey:         req.SecretKey,
		Domain:            req.Domain,
		Zone:              req.ZoneID,
		TemplateID:        req.TemplateID,
		NetworkID:         req.NetworkID,
		ServiceOfferingID: req.ServiceOfferingID,
		DiskOfferingID:    req.DiskOfferingID,
		OMAVMID:           req.OMAVmID,
		IsActive:          true,
	}

	// Save to database (update existing if it exists)
	repo := database.NewOSSEAConfigRepository(h.db)

	// Try to get existing config first
	existingConfig, err := repo.GetByName("production-ossea")
	if err == nil {
		// Update existing configuration
		err = repo.Update(existingConfig.ID, config)
	} else {
		// Create new configuration
		err = repo.Create(config)
	}

	if err != nil {
		log.WithError(err).Error("Failed to save OSSEA configuration")
		http.Error(w, "Failed to save configuration: "+err.Error(), http.StatusInternalServerError)
		return
	}

	log.Info("✅ Streamlined OSSEA configuration saved successfully")

	response := map[string]interface{}{
		"success": true,
		"message": "OSSEA configuration saved successfully",
		"config": map[string]interface{}{
			"api_url":             fullAPIURL,
			"zone_id":             req.ZoneID,
			"domain":              req.DomainName,
			"template_id":         req.TemplateID,
			"service_offering_id": req.ServiceOfferingID,
			"oma_vm_id":           req.OMAVmID,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
