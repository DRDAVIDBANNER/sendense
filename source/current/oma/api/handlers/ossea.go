// Package handlers provides HTTP handlers for OMA API endpoints
// OSSEA configuration management following project rules: minimal endpoints, modular design
package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/vexxhost/migratekit-oma/database"
)

// OSSEAHandler handles OSSEA configuration API endpoints
// Follows project rules: modular, well-structured, no monster code
type OSSEAHandler struct {
	db   database.Connection
	repo *database.OSSEAConfigRepository
}

// NewOSSEAHandler creates a new OSSEA configuration handler
func NewOSSEAHandler(db database.Connection) *OSSEAHandler {
	return &OSSEAHandler{
		db:   db,
		repo: database.NewOSSEAConfigRepository(db),
	}
}

// OSSEAConfigRequest represents the API request structure for OSSEA configuration
type OSSEAConfigRequest struct {
	Name              string `json:"name" validate:"required" example:"production-ossea"`
	APIURL            string `json:"api_url" validate:"required" example:"http://10.245.241.101:8080/client/api"`
	APIKey            string `json:"api_key" validate:"required" example:"your-api-key"`
	SecretKey         string `json:"secret_key" validate:"required" example:"your-secret-key"`
	Zone              string `json:"zone" validate:"required" example:"OSSEA-Zone"`
	Domain            string `json:"domain,omitempty" example:"ROOT"`
	OMAVMID           string `json:"oma_vm_id" validate:"required" example:"8a4400e5-c92a-4bc4-8bff-4b6b0b6a018c"`
	TemplateID        string `json:"template_id,omitempty" example:"template-123"`
	NetworkID         string `json:"network_id,omitempty" example:"network-456"`
	ServiceOfferingID string `json:"service_offering_id,omitempty" example:"offering-789"`
	DiskOfferingID    string `json:"disk_offering_id,omitempty" example:"c813c642-d946-49e1-9289-c616dd70206a"`
}

// OSSEAConfigResponse represents the API response structure for OSSEA configuration
type OSSEAConfigResponse struct {
	ID                int    `json:"id" example:"1"`
	Name              string `json:"name" example:"production-ossea"`
	APIURL            string `json:"api_url" example:"http://10.245.241.101:8080/client/api"`
	APIKey            string `json:"api_key" example:"your-api-key"`
	Zone              string `json:"zone" example:"OSSEA-Zone"`
	Domain            string `json:"domain" example:"ROOT"`
	OMAVMID           string `json:"oma_vm_id" example:"8a4400e5-c92a-4bc4-8bff-4b6b0b6a018c"`
	TemplateID        string `json:"template_id" example:"template-123"`
	NetworkID         string `json:"network_id" example:"network-456"`
	ServiceOfferingID string `json:"service_offering_id" example:"offering-789"`
	DiskOfferingID    string `json:"disk_offering_id" example:"c813c642-d946-49e1-9289-c616dd70206a"`
	CreatedAt         string `json:"created_at" example:"2025-01-15T10:00:00Z"`
	UpdatedAt         string `json:"updated_at" example:"2025-01-15T10:30:00Z"`
}

// OSSEAConfigOperation represents different operations that can be performed
type OSSEAConfigOperation struct {
	Action string              `json:"action" validate:"required" example:"get"` // get, create, update, delete, test
	ID     *int                `json:"id,omitempty" example:"1"`                 // required for get, update, delete
	Config *OSSEAConfigRequest `json:"config,omitempty"`                         // required for create, update, test
}

// OSSEAConfigMultiResponse represents the unified response structure
type OSSEAConfigMultiResponse struct {
	Action     string                  `json:"action" example:"get"`
	Success    bool                    `json:"success" example:"true"`
	Message    string                  `json:"message" example:"Operation completed successfully"`
	Config     *OSSEAConfigResponse    `json:"config,omitempty"`      // single config for get, create, update
	Configs    []OSSEAConfigResponse   `json:"configs,omitempty"`     // multiple configs for get all
	TestResult *TestConnectionResponse `json:"test_result,omitempty"` // for test action
	Timestamp  string                  `json:"timestamp" example:"2025-01-15T10:00:00Z"`
}

// TestConnectionResponse represents the response for OSSEA connection testing
type TestConnectionResponse struct {
	Success   bool   `json:"success" example:"true"`
	Message   string `json:"message" example:"Connection successful"`
	Zone      string `json:"zone,omitempty" example:"OSSEA-Zone"`
	Error     string `json:"error,omitempty" example:""`
	Timestamp string `json:"timestamp" example:"2025-01-15T10:00:00Z"`
}

// HandleConfig is the UNIFIED endpoint for all OSSEA configuration operations
// PROJECT RULE: Single endpoint instead of 6 separate endpoints to avoid API sprawl
// @Summary Unified OSSEA configuration management
// @Description Single endpoint for all OSSEA configuration operations (get, create, update, delete, test)
// @Tags ossea-config
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param operation body OSSEAConfigOperation true "OSSEA configuration operation"
// @Success 200 {object} OSSEAConfigMultiResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/ossea/config [post]
func (h *OSSEAHandler) HandleConfig(w http.ResponseWriter, r *http.Request) {
	var op OSSEAConfigOperation
	if err := json.NewDecoder(r.Body).Decode(&op); err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid request payload", err.Error())
		return
	}

	response := OSSEAConfigMultiResponse{
		Action:    op.Action,
		Timestamp: time.Now().Format(time.RFC3339),
	}

	switch op.Action {
	case "get":
		h.handleGet(&response, op.ID)
	case "create":
		h.handleCreate(&response, op.Config)
	case "update":
		h.handleUpdate(&response, op.ID, op.Config)
	case "delete":
		h.handleDelete(&response, op.ID)
	case "test":
		h.handleTest(&response, op.Config)
	default:
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid action", "Supported actions: get, create, update, delete, test")
		return
	}

	statusCode := http.StatusOK
	if op.Action == "create" && response.Success {
		statusCode = http.StatusCreated
	}

	h.writeJSONResponse(w, statusCode, response)
}

// Handler methods following project rules: small, focused functions

// handleGet processes get operations (single config or all configs)
func (h *OSSEAHandler) handleGet(response *OSSEAConfigMultiResponse, id *int) {
	if id != nil {
		// Get single config by ID
		log.WithField("config_id", *id).Info("Retrieving OSSEA configuration")

		if h.repo == nil || h.db.GetGormDB() == nil {
			response.Success = false
			response.Message = "Database not available (using memory mode)"
			return
		}

		config, err := h.repo.GetByID(*id)
		if err != nil {
			response.Success = false
			response.Message = err.Error()
			return
		}

		response.Config = h.convertToResponse(config)
		response.Success = true
		response.Message = "Configuration retrieved successfully"
	} else {
		// Get all configs
		log.Info("Retrieving all OSSEA configurations")

		if h.repo == nil || h.db.GetGormDB() == nil {
			response.Configs = make([]OSSEAConfigResponse, 0)
			response.Success = true
			response.Message = "Retrieved 0 configurations (database not available - using memory mode)"
			return
		}

		configs, err := h.repo.GetAll()
		if err != nil {
			response.Success = false
			response.Message = err.Error()
			return
		}

		response.Configs = make([]OSSEAConfigResponse, len(configs))
		for i, config := range configs {
			response.Configs[i] = *h.convertToResponse(&config)
		}

		response.Success = true
		response.Message = fmt.Sprintf("Retrieved %d configurations successfully", len(configs))
	}
}

// handleCreate processes create operations
func (h *OSSEAHandler) handleCreate(response *OSSEAConfigMultiResponse, configReq *OSSEAConfigRequest) {
	if configReq == nil {
		response.Success = false
		response.Message = "Configuration data required for create action"
		return
	}

	log.WithField("config_name", configReq.Name).Info("Creating new OSSEA configuration")

	if h.repo == nil || h.db.GetGormDB() == nil {
		response.Success = false
		response.Message = "Database not available (using memory mode)"
		return
	}

	// Convert request to database model
	config := h.convertToModel(configReq)

	// Save to database
	if err := h.repo.Create(config); err != nil {
		response.Success = false
		response.Message = err.Error()
		return
	}

	response.Config = h.convertToResponse(config)
	response.Success = true
	response.Message = "Configuration created successfully"
}

// handleUpdate processes update operations
func (h *OSSEAHandler) handleUpdate(response *OSSEAConfigMultiResponse, id *int, configReq *OSSEAConfigRequest) {
	if id == nil {
		response.Success = false
		response.Message = "Configuration ID required for update action"
		return
	}
	if configReq == nil {
		response.Success = false
		response.Message = "Configuration data required for update action"
		return
	}

	log.WithFields(log.Fields{
		"config_id":   *id,
		"config_name": configReq.Name,
	}).Info("Updating OSSEA configuration")

	if h.repo == nil || h.db.GetGormDB() == nil {
		response.Success = false
		response.Message = "Database not available (using memory mode)"
		return
	}

	// Convert request to database model
	config := h.convertToModel(configReq)

	// Update in database
	if err := h.repo.Update(*id, config); err != nil {
		response.Success = false
		response.Message = err.Error()
		return
	}

	response.Config = h.convertToResponse(config)
	response.Success = true
	response.Message = "Configuration updated successfully"
}

// handleDelete processes delete operations
func (h *OSSEAHandler) handleDelete(response *OSSEAConfigMultiResponse, id *int) {
	if id == nil {
		response.Success = false
		response.Message = "Configuration ID required for delete action"
		return
	}

	log.WithField("config_id", *id).Info("Deleting OSSEA configuration")

	if h.repo == nil || h.db.GetGormDB() == nil {
		response.Success = false
		response.Message = "Database not available (using memory mode)"
		return
	}

	// Delete from database
	if err := h.repo.Delete(*id); err != nil {
		response.Success = false
		response.Message = err.Error()
		return
	}

	response.Success = true
	response.Message = "Configuration deleted successfully"
}

// handleTest processes test operations
func (h *OSSEAHandler) handleTest(response *OSSEAConfigMultiResponse, configReq *OSSEAConfigRequest) {
	if h.repo == nil || h.db.GetGormDB() == nil {
		response.Success = false
		response.Message = "Database not available"
		return
	}

	var config *database.OSSEAConfig

	if configReq != nil {
		// If configuration is provided in request, use it
		log.WithField("api_url", configReq.APIURL).Info("Testing provided OSSEA configuration")
		config = h.convertToModel(configReq)
	} else {
		// Try to get active configuration from database
		log.Info("Retrieving active OSSEA configuration from database")
		configs, err := h.repo.GetAll()
		if err != nil {
			response.Success = false
			response.Message = fmt.Sprintf("Failed to retrieve configuration: %v", err)
			return
		}
		if len(configs) == 0 {
			response.Success = false
			response.Message = "No active OSSEA configuration found"
			return
		}
		config = &configs[0] // GetAll returns only active configs, sorted by most recent
	}

	// Test the connection using repository
	success, message, err := h.repo.TestConnection(config)
	if err != nil {
		response.Success = false
		response.Message = err.Error()
		return
	}

	testResult := TestConnectionResponse{
		Success:   success,
		Message:   message,
		Zone:      config.Zone,
		Timestamp: response.Timestamp,
	}

	response.TestResult = &testResult
	response.Success = success
	response.Message = message
}

// Helper functions following project standards: small, focused, well-documented

// writeJSONResponse writes a standardized JSON response
func (h *OSSEAHandler) writeJSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.WithError(err).Error("Failed to encode JSON response")
	}
}

// writeErrorResponse writes a standardized error response
func (h *OSSEAHandler) writeErrorResponse(w http.ResponseWriter, statusCode int, message, details string) {
	response := map[string]interface{}{
		"error":     message,
		"details":   details,
		"timestamp": time.Now().Format(time.RFC3339),
	}
	h.writeJSONResponse(w, statusCode, response)
}

// convertToModel converts API request to database model
func (h *OSSEAHandler) convertToModel(req *OSSEAConfigRequest) *database.OSSEAConfig {
	return &database.OSSEAConfig{
		Name:              req.Name,
		APIURL:            req.APIURL,
		APIKey:            req.APIKey,
		SecretKey:         req.SecretKey,
		Domain:            req.Domain,
		Zone:              req.Zone,
		TemplateID:        req.TemplateID,
		NetworkID:         req.NetworkID,
		ServiceOfferingID: req.ServiceOfferingID,
		DiskOfferingID:    req.DiskOfferingID,
		OMAVMID:           req.OMAVMID,
		IsActive:          true,
	}
}

// convertToResponse converts database model to API response
func (h *OSSEAHandler) convertToResponse(config *database.OSSEAConfig) *OSSEAConfigResponse {
	return &OSSEAConfigResponse{
		ID:                config.ID,
		Name:              config.Name,
		APIURL:            config.APIURL,
		APIKey:            config.APIKey,
		Zone:              config.Zone,
		Domain:            config.Domain,
		OMAVMID:           config.OMAVMID,
		TemplateID:        config.TemplateID,
		NetworkID:         config.NetworkID,
		ServiceOfferingID: config.ServiceOfferingID,
		DiskOfferingID:    config.DiskOfferingID,
		CreatedAt:         config.CreatedAt.Format(time.RFC3339),
		UpdatedAt:         config.UpdatedAt.Format(time.RFC3339),
	}
}
