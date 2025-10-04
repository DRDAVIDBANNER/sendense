// Package handlers provides HTTP handlers for OMA API endpoints
// Linstor configuration management following project rules: minimal endpoints, modular design
package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/vexxhost/migratekit-oma/config"
	"github.com/vexxhost/migratekit-oma/database"
)

// LinstorHandler handles Linstor configuration API endpoints
// Follows project rules: modular, well-structured, no monster code
type LinstorHandler struct {
	db     database.Connection
	config *config.LinstorConfigManager
}

// NewLinstorHandler creates a new Linstor configuration handler
func NewLinstorHandler(db database.Connection) *LinstorHandler {
	return &LinstorHandler{
		db:     db,
		config: config.NewLinstorConfigManager(db),
	}
}

// LinstorConfigRequest represents the API request structure for Linstor configuration
type LinstorConfigRequest struct {
	Name                     string `json:"name" validate:"required" example:"production-linstor"`
	APIURL                   string `json:"api_url" validate:"required" example:"http://10.245.241.101"`
	APIPort                  int    `json:"api_port" example:"3370"`
	APIProtocol              string `json:"api_protocol" example:"http"`
	APIKey                   string `json:"api_key,omitempty" example:"optional-api-key"`
	APISecret                string `json:"api_secret,omitempty" example:"optional-api-secret"`
	ConnectionTimeoutSeconds int    `json:"connection_timeout_seconds" example:"30"`
	RetryAttempts            int    `json:"retry_attempts" example:"3"`
	Description              string `json:"description,omitempty" example:"Production Linstor cluster for volume snapshots"`
}

// LinstorConfigResponse represents the API response structure for Linstor configuration
type LinstorConfigResponse struct {
	ID                       int    `json:"id" example:"1"`
	Name                     string `json:"name" example:"production-linstor"`
	APIURL                   string `json:"api_url" example:"http://10.245.241.101"`
	APIPort                  int    `json:"api_port" example:"3370"`
	APIProtocol              string `json:"api_protocol" example:"http"`
	APIKey                   string `json:"api_key" example:"optional-api-key"`
	ConnectionTimeoutSeconds int    `json:"connection_timeout_seconds" example:"30"`
	RetryAttempts            int    `json:"retry_attempts" example:"3"`
	Description              string `json:"description" example:"Production Linstor cluster"`
	CreatedAt                string `json:"created_at" example:"2025-09-02T15:00:00Z"`
	UpdatedAt                string `json:"updated_at" example:"2025-09-02T15:30:00Z"`
}

// LinstorConfigOperation represents different operations that can be performed
type LinstorConfigOperation struct {
	Action string                    `json:"action" validate:"required" example:"get"` // get, create, update, delete, test
	ID     *int                      `json:"id,omitempty" example:"1"`                 // required for get, update, delete
	Config *LinstorConfigRequest     `json:"config,omitempty"`                         // required for create, update, test
}

// LinstorConfigMultiResponse represents the unified response structure
type LinstorConfigMultiResponse struct {
	Action     string                      `json:"action" example:"get"`
	Success    bool                        `json:"success" example:"true"`
	Message    string                      `json:"message" example:"Operation completed successfully"`
	Config     *LinstorConfigResponse      `json:"config,omitempty"`      // single config for get, create, update
	Configs    []LinstorConfigResponse     `json:"configs,omitempty"`     // multiple configs for get all
	TestResult *LinstorTestResponse        `json:"test_result,omitempty"` // for test action
	Timestamp  string                      `json:"timestamp" example:"2025-09-02T15:00:00Z"`
}

// LinstorTestResponse represents the response for Linstor connection testing
type LinstorTestResponse struct {
	Success   bool   `json:"success" example:"true"`
	Message   string `json:"message" example:"Connection successful"`
	Version   string `json:"version,omitempty" example:"1.21.1"`
	Error     string `json:"error,omitempty" example:""`
	Timestamp string `json:"timestamp" example:"2025-09-02T15:00:00Z"`
}

// HandleConfig is the UNIFIED endpoint for all Linstor configuration operations
// PROJECT RULE: Single endpoint instead of 6 separate endpoints to avoid API sprawl
// @Summary Unified Linstor configuration management
// @Description Single endpoint for all Linstor configuration operations (get, create, update, delete, test)
// @Tags linstor-config
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param operation body LinstorConfigOperation true "Linstor configuration operation"
// @Success 200 {object} LinstorConfigMultiResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/linstor/config [post]
func (h *LinstorHandler) HandleConfig(w http.ResponseWriter, r *http.Request) {
	var op LinstorConfigOperation
	if err := json.NewDecoder(r.Body).Decode(&op); err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid request payload", err.Error())
		return
	}

	response := LinstorConfigMultiResponse{
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
func (h *LinstorHandler) handleGet(response *LinstorConfigMultiResponse, id *int) {
	if id != nil {
		// Get single config by ID
		log.WithField("config_id", *id).Info("Retrieving Linstor configuration")

		config, err := h.config.GetLinstorConfig(*id)
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
		log.Info("Retrieving all Linstor configurations")

		configs, err := h.config.ListLinstorConfigs()
		if err != nil {
			response.Success = false
			response.Message = err.Error()
			return
		}

		response.Configs = make([]LinstorConfigResponse, len(configs))
		for i, config := range configs {
			response.Configs[i] = *h.convertToResponse(&config)
		}

		response.Success = true
		response.Message = fmt.Sprintf("Retrieved %d configurations successfully", len(configs))
	}
}

// handleCreate processes create operations
func (h *LinstorHandler) handleCreate(response *LinstorConfigMultiResponse, configReq *LinstorConfigRequest) {
	if configReq == nil {
		response.Success = false
		response.Message = "Configuration data required for create action"
		return
	}

	log.WithField("config_name", configReq.Name).Info("Creating new Linstor configuration")

	// Convert request to config input
	input := h.convertToInput(configReq)

	// Create via config manager
	config, err := h.config.CreateLinstorConfig(input)
	if err != nil {
		response.Success = false
		response.Message = err.Error()
		return
	}

	response.Config = h.convertToResponse(config)
	response.Success = true
	response.Message = "Configuration created successfully"
}

// handleUpdate processes update operations
func (h *LinstorHandler) handleUpdate(response *LinstorConfigMultiResponse, id *int, configReq *LinstorConfigRequest) {
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
	}).Info("Updating Linstor configuration")

	// Convert request to config input
	input := h.convertToInput(configReq)

	// Update via config manager
	config, err := h.config.UpdateLinstorConfig(*id, input)
	if err != nil {
		response.Success = false
		response.Message = err.Error()
		return
	}

	response.Config = h.convertToResponse(config)
	response.Success = true
	response.Message = "Configuration updated successfully"
}

// handleDelete processes delete operations
func (h *LinstorHandler) handleDelete(response *LinstorConfigMultiResponse, id *int) {
	if id == nil {
		response.Success = false
		response.Message = "Configuration ID required for delete action"
		return
	}

	log.WithField("config_id", *id).Info("Deleting Linstor configuration")

	// Delete via config manager
	if err := h.config.DeleteLinstorConfig(*id); err != nil {
		response.Success = false
		response.Message = err.Error()
		return
	}

	response.Success = true
	response.Message = "Configuration deleted successfully"
}

// handleTest processes test operations
func (h *LinstorHandler) handleTest(response *LinstorConfigMultiResponse, configReq *LinstorConfigRequest) {
	if configReq != nil {
		// If configuration is provided in request, use it
		log.WithField("api_url", configReq.APIURL).Info("Testing provided Linstor configuration")
		// input := h.convertToInput(configReq) // Will be used when implementing actual test
	} else {
		// Try to get active configuration from database
		log.Info("Retrieving active Linstor configuration from database")
		config, err := h.config.GetDefaultLinstorConfig()
		if err != nil {
			response.Success = false
			response.Message = fmt.Sprintf("Failed to retrieve configuration: %v", err)
			return
		}
		// Use the config directly for testing
		_ = config // We'll implement actual testing later
	}

	// TODO: Implement actual Linstor API test
	testResult := LinstorTestResponse{
		Success:   true,
		Message:   "Linstor connection test not yet implemented",
		Timestamp: response.Timestamp,
	}

	response.TestResult = &testResult
	response.Success = true
	response.Message = "Connection test completed"
}

// Helper functions following project standards: small, focused, well-documented

// writeJSONResponse writes a standardized JSON response
func (h *LinstorHandler) writeJSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.WithError(err).Error("Failed to encode JSON response")
	}
}

// writeErrorResponse writes a standardized error response
func (h *LinstorHandler) writeErrorResponse(w http.ResponseWriter, statusCode int, message, details string) {
	response := map[string]interface{}{
		"error":     message,
		"details":   details,
		"timestamp": time.Now().Format(time.RFC3339),
	}
	h.writeJSONResponse(w, statusCode, response)
}

// convertToInput converts API request to config input
func (h *LinstorHandler) convertToInput(req *LinstorConfigRequest) *config.LinstorConfigInput {
	return &config.LinstorConfigInput{
		Name:                     req.Name,
		APIURL:                   req.APIURL,
		APIPort:                  req.APIPort,
		APIProtocol:              req.APIProtocol,
		APIKey:                   req.APIKey,
		APISecret:                req.APISecret,
		ConnectionTimeoutSeconds: req.ConnectionTimeoutSeconds,
		RetryAttempts:            req.RetryAttempts,
		Description:              req.Description,
	}
}

// convertToResponse converts database model to API response
func (h *LinstorHandler) convertToResponse(config *database.LinstorConfig) *LinstorConfigResponse {
	return &LinstorConfigResponse{
		ID:                       config.ID,
		Name:                     config.Name,
		APIURL:                   config.APIURL,
		APIPort:                  config.APIPort,
		APIProtocol:              config.APIProtocol,
		APIKey:                   config.APIKey,
		ConnectionTimeoutSeconds: config.ConnectionTimeoutSeconds,
		RetryAttempts:            config.RetryAttempts,
		Description:              config.Description,
		CreatedAt:                config.CreatedAt.Format(time.RFC3339),
		UpdatedAt:                config.UpdatedAt.Format(time.RFC3339),
	}
}
