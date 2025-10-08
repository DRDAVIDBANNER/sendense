// Package handlers provides HTTP handlers for backup policy management endpoints
// Following project rules: modular design, minimal endpoints, clean separation
package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"

	"github.com/vexxhost/migratekit-sha/storage"
)

// PolicyHandler handles backup policy management endpoints.
// Enterprise feature: 3-2-1 backup rule support with multi-repository copies.
type PolicyHandler struct {
	db            *sql.DB
	policyManager *storage.PolicyManager
	policyRepo    storage.PolicyRepository
	configRepo    storage.ConfigRepository
}

// NewPolicyHandler creates a new policy handler.
func NewPolicyHandler(db *sql.DB) (*PolicyHandler, error) {
	policyRepo := storage.NewPolicyRepository(db)
	configRepo := storage.NewConfigRepository(db)
	policyManager := storage.NewPolicyManager(policyRepo, configRepo)

	return &PolicyHandler{
		db:            db,
		policyManager: policyManager,
		policyRepo:    policyRepo,
		configRepo:    configRepo,
	}, nil
}

// CreatePolicyRequest represents the request to create a backup policy.
type CreatePolicyRequest struct {
	Name                string                       `json:"name"`
	Enabled             bool                         `json:"enabled"`
	PrimaryRepositoryID string                       `json:"primary_repository_id"`
	RetentionDays       int                          `json:"retention_days"`
	CopyRules           []*storage.BackupCopyRule    `json:"copy_rules"`
}

// PolicyResponse represents a policy in API responses.
type PolicyResponse struct {
	ID                  string                    `json:"id"`
	Name                string                    `json:"name"`
	Enabled             bool                      `json:"enabled"`
	PrimaryRepositoryID string                    `json:"primary_repository_id"`
	RetentionDays       int                       `json:"retention_days"`
	CopyRules           []*storage.BackupCopyRule `json:"copy_rules"`
	CreatedAt           string                    `json:"created_at"`
	UpdatedAt           string                    `json:"updated_at"`
}

// CreatePolicy handles POST /api/v1/policies
func (h *PolicyHandler) CreatePolicy(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse request
	var req CreatePolicyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	// Validate request
	if req.Name == "" {
		http.Error(w, "Policy name is required", http.StatusBadRequest)
		return
	}
	if req.PrimaryRepositoryID == "" {
		http.Error(w, "Primary repository ID is required", http.StatusBadRequest)
		return
	}

	// Generate policy ID
	policyID := uuid.New().String()

	// Create policy object
	policy := &storage.BackupPolicy{
		ID:                  policyID,
		Name:                req.Name,
		Enabled:             req.Enabled,
		PrimaryRepositoryID: req.PrimaryRepositoryID,
		RetentionDays:       req.RetentionDays,
		CopyRules:           req.CopyRules,
	}

	// Generate IDs for copy rules
	for i := range policy.CopyRules {
		if policy.CopyRules[i].ID == "" {
			policy.CopyRules[i].ID = uuid.New().String()
		}
	}

	// Create policy via manager (includes validation)
	if err := h.policyManager.CreatePolicy(ctx, policy); err != nil {
		log.WithError(err).WithField("name", req.Name).Error("Failed to create policy")
		http.Error(w, fmt.Sprintf("Failed to create policy: %v", err), http.StatusInternalServerError)
		return
	}

	log.WithFields(log.Fields{
		"id":          policyID,
		"name":        req.Name,
		"copy_rules":  len(policy.CopyRules),
	}).Info("Backup policy created successfully")

	// Build response
	response := PolicyResponse{
		ID:                  policy.ID,
		Name:                policy.Name,
		Enabled:             policy.Enabled,
		PrimaryRepositoryID: policy.PrimaryRepositoryID,
		RetentionDays:       policy.RetentionDays,
		CopyRules:           policy.CopyRules,
		CreatedAt:           policy.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:           policy.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// ListPolicies handles GET /api/v1/policies
func (h *PolicyHandler) ListPolicies(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get all policies
	policies, err := h.policyManager.ListPolicies(ctx)
	if err != nil {
		log.WithError(err).Error("Failed to list policies")
		http.Error(w, fmt.Sprintf("Failed to list policies: %v", err), http.StatusInternalServerError)
		return
	}

	// Build responses
	responses := make([]PolicyResponse, 0, len(policies))
	for _, policy := range policies {
		responses = append(responses, PolicyResponse{
			ID:                  policy.ID,
			Name:                policy.Name,
			Enabled:             policy.Enabled,
			PrimaryRepositoryID: policy.PrimaryRepositoryID,
			RetentionDays:       policy.RetentionDays,
			CopyRules:           policy.CopyRules,
			CreatedAt:           policy.CreatedAt.Format("2006-01-02T15:04:05Z"),
			UpdatedAt:           policy.UpdatedAt.Format("2006-01-02T15:04:05Z"),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(responses)
}

// GetPolicy handles GET /api/v1/policies/{id}
func (h *PolicyHandler) GetPolicy(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	policyID := vars["id"]

	if policyID == "" {
		http.Error(w, "Policy ID is required", http.StatusBadRequest)
		return
	}

	// Get policy
	policy, err := h.policyManager.GetPolicy(ctx, policyID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Policy not found: %v", err), http.StatusNotFound)
		return
	}

	// Build response
	response := PolicyResponse{
		ID:                  policy.ID,
		Name:                policy.Name,
		Enabled:             policy.Enabled,
		PrimaryRepositoryID: policy.PrimaryRepositoryID,
		RetentionDays:       policy.RetentionDays,
		CopyRules:           policy.CopyRules,
		CreatedAt:           policy.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:           policy.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// DeletePolicy handles DELETE /api/v1/policies/{id}
func (h *PolicyHandler) DeletePolicy(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	policyID := vars["id"]

	if policyID == "" {
		http.Error(w, "Policy ID is required", http.StatusBadRequest)
		return
	}

	// Delete policy
	if err := h.policyManager.DeletePolicy(ctx, policyID); err != nil {
		log.WithError(err).WithField("policy_id", policyID).Error("Failed to delete policy")
		http.Error(w, fmt.Sprintf("Failed to delete policy: %v", err), http.StatusInternalServerError)
		return
	}

	log.WithField("policy_id", policyID).Info("Backup policy deleted successfully")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Policy deleted successfully",
		"id":      policyID,
	})
}

// BackupCopyResponse represents a backup copy in API responses.
type BackupCopyResponse struct {
	ID                 string `json:"id"`
	SourceBackupID     string `json:"source_backup_id"`
	RepositoryID       string `json:"repository_id"`
	CopyRuleID         string `json:"copy_rule_id,omitempty"`
	Status             string `json:"status"`
	FilePath           string `json:"file_path"`
	SizeBytes          int64  `json:"size_bytes"`
	CopyStartedAt      string `json:"copy_started_at,omitempty"`
	CopyCompletedAt    string `json:"copy_completed_at,omitempty"`
	VerifiedAt         string `json:"verified_at,omitempty"`
	VerificationStatus string `json:"verification_status"`
	ErrorMessage       string `json:"error_message,omitempty"`
	CreatedAt          string `json:"created_at"`
}

// GetBackupCopies handles GET /api/v1/backups/{id}/copies
func (h *PolicyHandler) GetBackupCopies(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	backupID := vars["id"]

	if backupID == "" {
		http.Error(w, "Backup ID is required", http.StatusBadRequest)
		return
	}

	// Get backup copies
	copies, err := h.policyRepo.ListBackupCopies(ctx, backupID)
	if err != nil {
		log.WithError(err).WithField("backup_id", backupID).Error("Failed to list backup copies")
		http.Error(w, fmt.Sprintf("Failed to list backup copies: %v", err), http.StatusInternalServerError)
		return
	}

	// Build responses
	responses := make([]BackupCopyResponse, 0, len(copies))
	for _, copy := range copies {
		resp := BackupCopyResponse{
			ID:                 copy.ID,
			SourceBackupID:     copy.SourceBackupID,
			RepositoryID:       copy.RepositoryID,
			CopyRuleID:         copy.CopyRuleID,
			Status:             string(copy.Status),
			FilePath:           copy.FilePath,
			SizeBytes:          copy.SizeBytes,
			VerificationStatus: string(copy.VerificationStatus),
			ErrorMessage:       copy.ErrorMessage,
			CreatedAt:          copy.CreatedAt.Format("2006-01-02T15:04:05Z"),
		}

		if copy.CopyStartedAt != nil {
			resp.CopyStartedAt = copy.CopyStartedAt.Format("2006-01-02T15:04:05Z")
		}
		if copy.CopyCompletedAt != nil {
			resp.CopyCompletedAt = copy.CopyCompletedAt.Format("2006-01-02T15:04:05Z")
		}
		if copy.VerifiedAt != nil {
			resp.VerifiedAt = copy.VerifiedAt.Format("2006-01-02T15:04:05Z")
		}

		responses = append(responses, resp)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(responses)
}

// TriggerBackupCopy handles POST /api/v1/backups/{id}/copy
func (h *PolicyHandler) TriggerBackupCopy(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	backupID := vars["id"]

	if backupID == "" {
		http.Error(w, "Backup ID is required", http.StatusBadRequest)
		return
	}

	// Parse request to get destination repository
	var req struct {
		RepositoryID string `json:"repository_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	if req.RepositoryID == "" {
		http.Error(w, "Repository ID is required", http.StatusBadRequest)
		return
	}

	// Create manual backup copy record
	copyID := uuid.New().String()
	copy := &storage.BackupCopy{
		ID:                 copyID,
		SourceBackupID:     backupID,
		RepositoryID:       req.RepositoryID,
		Status:             storage.BackupCopyStatusPending,
		VerificationStatus: storage.VerificationStatusPending,
	}

	if err := h.policyRepo.CreateBackupCopy(ctx, copy); err != nil {
		log.WithError(err).WithFields(log.Fields{
			"backup_id":     backupID,
			"repository_id": req.RepositoryID,
		}).Error("Failed to create manual backup copy")
		http.Error(w, fmt.Sprintf("Failed to create backup copy: %v", err), http.StatusInternalServerError)
		return
	}

	log.WithFields(log.Fields{
		"copy_id":       copyID,
		"backup_id":     backupID,
		"repository_id": req.RepositoryID,
	}).Info("Manual backup copy triggered")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Backup copy triggered successfully",
		"copy_id": copyID,
		"status":  "pending",
	})
}
