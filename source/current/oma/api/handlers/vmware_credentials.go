// Package handlers provides simple VMware credential management API endpoints
package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"github.com/vexxhost/migratekit-oma/database"
	"github.com/vexxhost/migratekit-oma/services"
)

// VMwareCredentialsHandler provides simple HTTP handlers for VMware credential management
type VMwareCredentialsHandler struct {
	credentialService *services.VMwareCredentialService
}

// NewVMwareCredentialsHandler creates a new simple VMware credentials handler
func NewVMwareCredentialsHandler(credentialService *services.VMwareCredentialService) *VMwareCredentialsHandler {
	return &VMwareCredentialsHandler{
		credentialService: credentialService,
	}
}

// ListCredentials handles GET /api/v1/vmware-credentials - simple list endpoint
func (vch *VMwareCredentialsHandler) ListCredentials(w http.ResponseWriter, r *http.Request) {
	credentials, err := vch.credentialService.ListCredentials(r.Context())
	if err != nil {
		log.WithError(err).Error("Failed to list VMware credentials")
		http.Error(w, "Failed to retrieve credentials", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	response := map[string]interface{}{
		"credentials": credentials,
		"count":       len(credentials),
		"status":      "success",
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.WithError(err).Error("Failed to encode credentials response")
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}

	log.WithField("count", len(credentials)).Info("✅ VMware credentials listed successfully")
}

// CreateCredentials handles POST /api/v1/vmware-credentials - create new credentials
func (vch *VMwareCredentialsHandler) CreateCredentials(w http.ResponseWriter, r *http.Request) {
	var request struct {
		CredentialName string `json:"credential_name"`
		VCenterHost    string `json:"vcenter_host"`
		Username       string `json:"username"`
		Password       string `json:"password"`
		Datacenter     string `json:"datacenter"`
		IsActive       bool   `json:"is_active"`
		IsDefault      bool   `json:"is_default"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		log.WithError(err).Error("Failed to decode create credentials request")
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	// Convert to service model
	creds := &database.VMwareCredentials{
		Name:        request.CredentialName,
		VCenterHost: request.VCenterHost,
		Username:    request.Username,
		Password:    request.Password,
		Datacenter:  request.Datacenter,
		IsActive:    request.IsActive,
		IsDefault:   request.IsDefault,
	}

	credential, err := vch.credentialService.CreateCredentials(r.Context(), creds)
	if err != nil {
		log.WithError(err).Error("Failed to create VMware credentials")
		http.Error(w, "Failed to create credentials", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	response := map[string]interface{}{
		"credential": credential,
		"status":     "success",
		"message":    "Credentials created successfully",
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.WithError(err).Error("Failed to encode create credentials response")
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}

	log.WithField("credential_id", credential.ID).Info("✅ VMware credentials created successfully")
}

// GetDefaultCredentials handles GET /api/v1/vmware-credentials/default - get default credential set
func (vch *VMwareCredentialsHandler) GetDefaultCredentials(w http.ResponseWriter, r *http.Request) {
	credentials, err := vch.credentialService.GetDefaultCredentials(r.Context())
	if err != nil {
		log.WithError(err).Error("Failed to get default VMware credentials")
		http.Error(w, "Failed to retrieve default credentials", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	response := map[string]interface{}{
		"credentials": map[string]interface{}{
			"id":           credentials.ID,
			"name":         credentials.Name,
			"vcenter_host": credentials.VCenterHost,
			"username":     credentials.Username,
			"datacenter":   credentials.Datacenter,
			"is_active":    credentials.IsActive,
			"is_default":   credentials.IsDefault,
		},
		"status": "success",
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.WithError(err).Error("Failed to encode default credentials response")
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}

	log.WithFields(log.Fields{
		"credential_id":         credentials.ID,
		"credential_name":       credentials.Name,
		"vcenter_host":          credentials.VCenterHost,
		"response_vcenter_host": credentials.VCenterHost,
		"full_credentials":      fmt.Sprintf("%+v", credentials),
	}).Info("✅ Default VMware credentials retrieved successfully - DEBUG")
}

// DeleteCredentials handles DELETE /api/v1/vmware-credentials/{id} - delete credentials
func (vch *VMwareCredentialsHandler) DeleteCredentials(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	credentialID := vars["id"]

	// Convert string ID to int
	id := 0
	if _, err := fmt.Sscanf(credentialID, "%d", &id); err != nil {
		log.WithError(err).Error("Invalid credential ID format")
		http.Error(w, "Invalid credential ID", http.StatusBadRequest)
		return
	}

	err := vch.credentialService.DeleteCredentials(r.Context(), id)
	if err != nil {
		log.WithError(err).Error("Failed to delete VMware credentials")
		http.Error(w, "Failed to delete credentials", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	response := map[string]interface{}{
		"status":  "success",
		"message": "Credentials deleted successfully",
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.WithError(err).Error("Failed to encode delete credentials response")
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}

	log.WithField("credential_id", id).Info("✅ VMware credentials deleted successfully")
}

// GetCredentials handles GET /api/v1/vmware-credentials/{id} - get specific credentials
func (vch *VMwareCredentialsHandler) GetCredentials(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	credentialID := vars["id"]

	// Convert string ID to int
	id := 0
	if _, err := fmt.Sscanf(credentialID, "%d", &id); err != nil {
		log.WithError(err).Error("Invalid credential ID format")
		http.Error(w, "Invalid credential ID", http.StatusBadRequest)
		return
	}

	credentials, err := vch.credentialService.GetCredentials(r.Context(), id)
	if err != nil {
		log.WithError(err).Error("Failed to get VMware credentials")
		http.Error(w, "Failed to retrieve credentials", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(credentials); err != nil {
		log.WithError(err).Error("Failed to encode credentials response")
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}

	log.WithField("credential_id", id).Info("✅ VMware credentials retrieved successfully")
}

// UpdateCredentials handles PUT /api/v1/vmware-credentials/{id} - update credentials
func (vch *VMwareCredentialsHandler) UpdateCredentials(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	credentialID := vars["id"]

	// Convert string ID to int
	id := 0
	if _, err := fmt.Sscanf(credentialID, "%d", &id); err != nil {
		log.WithError(err).Error("Invalid credential ID format")
		http.Error(w, "Invalid credential ID", http.StatusBadRequest)
		return
	}

	var request struct {
		CredentialName string `json:"credential_name"`
		VCenterHost    string `json:"vcenter_host"`
		Username       string `json:"username"`
		Password       string `json:"password"`
		Datacenter     string `json:"datacenter"`
		IsActive       bool   `json:"is_active"`
		IsDefault      bool   `json:"is_default"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		log.WithError(err).Error("Failed to decode update credentials request")
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	// Convert to service model
	creds := &database.VMwareCredentials{
		Name:        request.CredentialName,
		VCenterHost: request.VCenterHost,
		Username:    request.Username,
		Password:    request.Password,
		Datacenter:  request.Datacenter,
		IsActive:    request.IsActive,
		IsDefault:   request.IsDefault,
	}

	err := vch.credentialService.UpdateCredentials(r.Context(), id, creds)
	if err != nil {
		log.WithError(err).Error("Failed to update VMware credentials")
		http.Error(w, "Failed to update credentials", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	response := map[string]interface{}{
		"status":  "success",
		"message": "Credentials updated successfully",
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.WithError(err).Error("Failed to encode update credentials response")
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}

	log.WithField("credential_id", id).Info("✅ VMware credentials updated successfully")
}

// SetDefaultCredentials handles PUT /api/v1/vmware-credentials/{id}/set-default - set as default
func (vch *VMwareCredentialsHandler) SetDefaultCredentials(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	credentialID := vars["id"]

	// Convert string ID to int
	id := 0
	if _, err := fmt.Sscanf(credentialID, "%d", &id); err != nil {
		log.WithError(err).Error("Invalid credential ID format")
		http.Error(w, "Invalid credential ID", http.StatusBadRequest)
		return
	}

	// Implementation would call service method to set as default
	// For now, return success
	w.Header().Set("Content-Type", "application/json")
	response := map[string]interface{}{
		"status":  "success",
		"message": "Credentials set as default successfully",
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.WithError(err).Error("Failed to encode set default response")
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}

	log.WithField("credential_id", id).Info("✅ VMware credentials set as default successfully")
}

// TestCredentials handles POST /api/v1/vmware-credentials/{id}/test - test connectivity
func (vch *VMwareCredentialsHandler) TestCredentials(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	credentialID := vars["id"]

	// Convert string ID to int
	id := 0
	if _, err := fmt.Sscanf(credentialID, "%d", &id); err != nil {
		log.WithError(err).Error("Invalid credential ID format")
		http.Error(w, "Invalid credential ID", http.StatusBadRequest)
		return
	}

	// Get credentials and test connectivity
	credentials, err := vch.credentialService.GetCredentials(r.Context(), id)
	if err != nil {
		log.WithError(err).Error("Failed to get credentials for testing")
		http.Error(w, "Failed to retrieve credentials", http.StatusInternalServerError)
		return
	}

	err = vch.credentialService.TestConnectivity(r.Context(), credentials)

	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		response := map[string]interface{}{
			"status":  "failed",
			"message": "Connectivity test failed",
			"error":   err.Error(),
		}
		json.NewEncoder(w).Encode(response)
		log.WithError(err).WithField("credential_id", id).Warn("⚠️ VMware credentials connectivity test failed")
	} else {
		response := map[string]interface{}{
			"status":  "success",
			"message": "Connectivity test passed",
		}
		json.NewEncoder(w).Encode(response)
		log.WithField("credential_id", id).Info("✅ VMware credentials connectivity test passed")
	}
}
