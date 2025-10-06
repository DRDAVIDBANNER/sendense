// Package handlers provides HTTP handlers for repository management endpoints
// Following project rules: modular design, minimal endpoints, clean separation
package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"

	"github.com/vexxhost/migratekit-oma/storage"
)

// RepositoryHandler handles backup repository management endpoints
type RepositoryHandler struct {
	db             *sql.DB
	repoManager    *storage.RepositoryManager
	configRepo     storage.ConfigRepository
	backupRepo     storage.BackupChainRepository
	mountManager   *storage.MountManager
}

// NewRepositoryHandler creates a new repository handler
func NewRepositoryHandler(db *sql.DB) (*RepositoryHandler, error) {
	// Initialize repositories for repository pattern
	configRepo := storage.NewConfigRepository(db)
	backupRepo := storage.NewBackupChainRepository(db)
	
	// Initialize mount manager for network storage
	mountManager := storage.NewMountManager()
	
	// Create repository manager
	repoManager, err := storage.NewRepositoryManager(configRepo, backupRepo, db, mountManager)
	if err != nil {
		return nil, fmt.Errorf("failed to create repository manager: %w", err)
	}

	return &RepositoryHandler{
		db:           db,
		repoManager:  repoManager,
		configRepo:   configRepo,
		backupRepo:   backupRepo,
		mountManager: mountManager,
	}, nil
}

// CreateRepositoryRequest represents the request to create a new repository
type CreateRepositoryRequest struct {
	Name             string                      `json:"name"`
	Type             storage.RepositoryType      `json:"type"`
	Enabled          bool                        `json:"enabled"`
	Config           json.RawMessage             `json:"config"` // Type-specific config as JSON
	IsImmutable      bool                        `json:"is_immutable"`
	MinRetentionDays int                         `json:"min_retention_days"`
}

// RepositoryResponse represents a repository in API responses
type RepositoryResponse struct {
	ID               string                 `json:"id"`
	Name             string                 `json:"name"`
	Type             storage.RepositoryType `json:"type"`
	Enabled          bool                   `json:"enabled"`
	Config           interface{}            `json:"config"`
	IsImmutable      bool                   `json:"is_immutable"`
	MinRetentionDays int                    `json:"min_retention_days"`
	StorageInfo      *storage.StorageInfo   `json:"storage_info,omitempty"`
	CreatedAt        string                 `json:"created_at"`
	UpdatedAt        string                 `json:"updated_at"`
}

// TestRepositoryRequest represents a request to test repository configuration
type TestRepositoryRequest struct {
	Type   storage.RepositoryType `json:"type"`
	Config json.RawMessage        `json:"config"`
}

// TestRepositoryResponse represents the result of testing a repository
type TestRepositoryResponse struct {
	Success      bool                 `json:"success"`
	Message      string               `json:"message"`
	StorageInfo  *storage.StorageInfo `json:"storage_info,omitempty"`
	ErrorDetails string               `json:"error_details,omitempty"`
}

// CreateRepository handles POST /api/v1/repositories
// Creates a new backup repository after validating configuration
func (h *RepositoryHandler) CreateRepository(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse request
	var req CreateRepositoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("Invalid request body: %v", err),
		})
		return
	}

	// Validate request
	if req.Name == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Repository name is required",
		})
		return
	}
	if req.Type == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Repository type is required",
		})
		return
	}

	// Parse type-specific config
	var config interface{}
	switch req.Type {
	case storage.RepositoryTypeLocal:
		var localConfig storage.LocalRepositoryConfig
		if err := json.Unmarshal(req.Config, &localConfig); err != nil {
			http.Error(w, fmt.Sprintf("Invalid local repository config: %v", err), http.StatusBadRequest)
			return
		}
		config = localConfig
	case storage.RepositoryTypeNFS:
		var nfsConfig storage.NFSRepositoryConfig
		if err := json.Unmarshal(req.Config, &nfsConfig); err != nil {
			http.Error(w, fmt.Sprintf("Invalid NFS repository config: %v", err), http.StatusBadRequest)
			return
		}
		config = nfsConfig
	case storage.RepositoryTypeCIFS, storage.RepositoryTypeSMB:
		var cifsConfig storage.CIFSRepositoryConfig
		if err := json.Unmarshal(req.Config, &cifsConfig); err != nil {
			http.Error(w, fmt.Sprintf("Invalid CIFS repository config: %v", err), http.StatusBadRequest)
			return
		}
		config = cifsConfig
	default:
		http.Error(w, fmt.Sprintf("Unsupported repository type: %s", req.Type), http.StatusBadRequest)
		return
	}

	// Create repository config
	repoConfig := &storage.RepositoryConfig{
		Name:             req.Name,
		Type:             req.Type,
		Enabled:          req.Enabled,
		Config:           config,
		IsImmutable:      req.IsImmutable,
		MinRetentionDays: req.MinRetentionDays,
	}

	// Register repository (includes connection test)
	if err := h.repoManager.RegisterRepository(ctx, repoConfig); err != nil {
		log.WithError(err).WithField("name", req.Name).Error("Failed to register repository")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("Failed to register repository: %v", err),
		})
		return
	}

	log.WithFields(log.Fields{
		"id":   repoConfig.ID,
		"name": repoConfig.Name,
		"type": repoConfig.Type,
	}).Info("Repository registered successfully")

	// Get storage info for response
	repo, err := h.repoManager.GetRepository(ctx, repoConfig.ID)
	var storageInfo *storage.StorageInfo
	if err == nil {
		storageInfo, _ = repo.GetStorageInfo(ctx)
	}

	// Build response
	response := RepositoryResponse{
		ID:               repoConfig.ID,
		Name:             repoConfig.Name,
		Type:             repoConfig.Type,
		Enabled:          repoConfig.Enabled,
		Config:           repoConfig.Config,
		IsImmutable:      repoConfig.IsImmutable,
		MinRetentionDays: repoConfig.MinRetentionDays,
		StorageInfo:      storageInfo,
		CreatedAt:        repoConfig.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:        repoConfig.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// ListRepositories handles GET /api/v1/repositories
// Lists all configured repositories with optional filtering
func (h *RepositoryHandler) ListRepositories(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get query parameters for filtering
	repoType := r.URL.Query().Get("type")
	enabledFilter := r.URL.Query().Get("enabled")

	// Get all repositories
	configs, err := h.repoManager.ListRepositories(ctx)
	if err != nil {
		log.WithError(err).Error("Failed to list repositories")
		http.Error(w, fmt.Sprintf("Failed to list repositories: %v", err), http.StatusInternalServerError)
		return
	}

	// Build responses with storage info
	responses := make([]RepositoryResponse, 0, len(configs))
	for _, config := range configs {
		// Apply filters
		if repoType != "" && string(config.Type) != repoType {
			continue
		}
		if enabledFilter == "true" && !config.Enabled {
			continue
		}
		if enabledFilter == "false" && config.Enabled {
			continue
		}

		// Get storage info
		repo, err := h.repoManager.GetRepository(ctx, config.ID)
		var storageInfo *storage.StorageInfo
		if err == nil {
			storageInfo, _ = repo.GetStorageInfo(ctx)
		}

		responses = append(responses, RepositoryResponse{
			ID:               config.ID,
			Name:             config.Name,
			Type:             config.Type,
			Enabled:          config.Enabled,
			Config:           config.Config,
			IsImmutable:      config.IsImmutable,
			MinRetentionDays: config.MinRetentionDays,
			StorageInfo:      storageInfo,
			CreatedAt:        config.CreatedAt.Format("2006-01-02T15:04:05Z"),
			UpdatedAt:        config.UpdatedAt.Format("2006-01-02T15:04:05Z"),
		})
	}

	// Return consistent API response format
	response := map[string]interface{}{
		"success":      true,
		"repositories": responses,
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetRepositoryStorage handles GET /api/v1/repositories/{id}/storage
// Forces an immediate storage capacity check for a repository
func (h *RepositoryHandler) GetRepositoryStorage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	repoID := vars["id"]

	if repoID == "" {
		http.Error(w, "Repository ID is required", http.StatusBadRequest)
		return
	}

	// Get repository instance
	repo, err := h.repoManager.GetRepository(ctx, repoID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Repository not found: %v", err), http.StatusNotFound)
		return
	}

	// Get fresh storage info
	storageInfo, err := repo.GetStorageInfo(ctx)
	if err != nil {
		log.WithError(err).WithField("repo_id", repoID).Error("Failed to get storage info")
		http.Error(w, fmt.Sprintf("Failed to get storage info: %v", err), http.StatusInternalServerError)
		return
	}

	// Update database with latest stats
	err = h.configRepo.UpdateStorageStats(ctx, repoID, storageInfo.TotalBytes, storageInfo.UsedBytes, storageInfo.AvailableBytes)
	if err != nil {
		log.WithError(err).WithField("repo_id", repoID).Warn("Failed to update storage stats in database")
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(storageInfo)
}

// TestRepository handles POST /api/v1/repositories/test
// Tests a repository configuration without saving it
func (h *RepositoryHandler) TestRepository(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse request
	var req TestRepositoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	// Parse type-specific config
	var config interface{}
	switch req.Type {
	case storage.RepositoryTypeLocal:
		var localConfig storage.LocalRepositoryConfig
		if err := json.Unmarshal(req.Config, &localConfig); err != nil {
			http.Error(w, fmt.Sprintf("Invalid local repository config: %v", err), http.StatusBadRequest)
			return
		}
		config = localConfig
	case storage.RepositoryTypeNFS:
		var nfsConfig storage.NFSRepositoryConfig
		if err := json.Unmarshal(req.Config, &nfsConfig); err != nil {
			http.Error(w, fmt.Sprintf("Invalid NFS repository config: %v", err), http.StatusBadRequest)
			return
		}
		config = nfsConfig
	case storage.RepositoryTypeCIFS, storage.RepositoryTypeSMB:
		var cifsConfig storage.CIFSRepositoryConfig
		if err := json.Unmarshal(req.Config, &cifsConfig); err != nil {
			http.Error(w, fmt.Sprintf("Invalid CIFS repository config: %v", err), http.StatusBadRequest)
			return
		}
		config = cifsConfig
	default:
		http.Error(w, fmt.Sprintf("Unsupported repository type: %s", req.Type), http.StatusBadRequest)
		return
	}

	// Create test repository config
	testConfig := &storage.RepositoryConfig{
		ID:      "test-repo",
		Name:    "Test Repository",
		Type:    req.Type,
		Enabled: true,
		Config:  config,
	}

	// Test the repository (validates and tests connection)
	err := h.repoManager.TestRepository(ctx, testConfig)
	if err != nil {
		log.WithError(err).WithField("type", req.Type).Info("Repository test failed")
		response := TestRepositoryResponse{
			Success:      false,
			Message:      "Repository test failed",
			ErrorDetails: err.Error(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK) // 200 with success:false to indicate validation failure
		json.NewEncoder(w).Encode(response)
		return
	}

	// Test passed - try to get storage info if possible
	// Note: We don't actually create the repository, just test connection
	response := TestRepositoryResponse{
		Success: true,
		Message: "Repository configuration is valid and connection test passed",
	}

	log.WithField("type", req.Type).Info("Repository test passed")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// DeleteRepository handles DELETE /api/v1/repositories/{id}
// Deletes a repository configuration (fails if backups exist)
func (h *RepositoryHandler) DeleteRepository(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	repoID := vars["id"]

	if repoID == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Repository ID is required",
		})
		return
	}

	// Delete repository (will fail if backups exist)
	err := h.repoManager.DeleteRepository(ctx, repoID)
	if err != nil {
		log.WithError(err).WithField("repo_id", repoID).Error("Failed to delete repository")
		
		// Check if error is due to existing backups
		if err.Error() == "cannot delete repository with existing backups" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusConflict)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"error":   "Cannot delete repository with existing backups",
			})
			return
		}
		
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("Failed to delete repository: %v", err),
		})
		return
	}

	log.WithField("repo_id", repoID).Info("Repository deleted successfully")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Repository deleted successfully",
		"id":      repoID,
	})
}

// RefreshStorage handles POST /api/v1/repositories/refresh-storage
// Refreshes storage info for all repositories
func (h *RepositoryHandler) RefreshStorage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get all repositories
	repos, err := h.repoManager.ListRepositories(ctx)
	if err != nil {
		log.WithError(err).Error("Failed to list repositories for refresh")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("Failed to list repositories: %v", err),
		})
		return
	}

	// Refresh storage info for each repository
	refreshedCount := 0
	failedCount := 0
	for _, repoConfig := range repos {
		repo, err := h.repoManager.GetRepository(ctx, repoConfig.ID)
		if err != nil {
			log.WithError(err).WithField("repo_id", repoConfig.ID).Warn("Failed to get repository for refresh")
			failedCount++
			continue
		}

		// GetStorageInfo automatically updates the database
		_, err = repo.GetStorageInfo(ctx)
		if err != nil {
			log.WithError(err).WithField("repo_id", repoConfig.ID).Warn("Failed to refresh storage info")
			failedCount++
			continue
		}

		refreshedCount++
	}

	log.WithFields(log.Fields{
		"refreshed": refreshedCount,
		"failed":    failedCount,
	}).Info("Storage refresh completed")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":         true,
		"message":         fmt.Sprintf("Storage information refreshed for %d repositories", refreshedCount),
		"refreshed_count": refreshedCount,
		"failed_count":    failedCount,
	})
}
