// Package storage provides backup policy management for multi-repository backup copies
// Following project rules: modular design, repository pattern, no simulations
package storage

import (
	"context"
	"fmt"
	"time"
)

// PolicyRepository defines database operations for backup policies.
// Following project rules: ALL database operations via repository pattern.
type PolicyRepository interface {
	// Policy CRUD operations
	CreatePolicy(ctx context.Context, policy *BackupPolicy) error
	GetPolicy(ctx context.Context, policyID string) (*BackupPolicy, error)
	GetPolicyByName(ctx context.Context, name string) (*BackupPolicy, error)
	ListPolicies(ctx context.Context) ([]*BackupPolicy, error)
	UpdatePolicy(ctx context.Context, policy *BackupPolicy) error
	DeletePolicy(ctx context.Context, policyID string) error

	// Copy rule operations
	AddCopyRule(ctx context.Context, rule *BackupCopyRule) error
	GetCopyRule(ctx context.Context, ruleID string) (*BackupCopyRule, error)
	ListCopyRules(ctx context.Context, policyID string) ([]*BackupCopyRule, error)
	UpdateCopyRule(ctx context.Context, rule *BackupCopyRule) error
	DeleteCopyRule(ctx context.Context, ruleID string) error

	// Copy tracking operations
	CreateBackupCopy(ctx context.Context, copy *BackupCopy) error
	GetBackupCopy(ctx context.Context, copyID string) (*BackupCopy, error)
	ListBackupCopies(ctx context.Context, sourceBackupID string) ([]*BackupCopy, error)
	ListPendingCopies(ctx context.Context) ([]*BackupCopy, error)
	UpdateBackupCopyStatus(ctx context.Context, copyID string, status BackupCopyStatus, errorMsg string) error
	UpdateBackupCopyVerification(ctx context.Context, copyID string, verified bool) error
}

// PolicyManager manages backup policies and orchestrates copy operations.
// Validates configurations and triggers copy jobs.
type PolicyManager struct {
	policyRepo PolicyRepository
	configRepo ConfigRepository // For repository validation
}

// NewPolicyManager creates a new policy manager.
func NewPolicyManager(policyRepo PolicyRepository, configRepo ConfigRepository) *PolicyManager {
	return &PolicyManager{
		policyRepo: policyRepo,
		configRepo: configRepo,
	}
}

// CreatePolicy creates a new backup policy with validation.
func (pm *PolicyManager) CreatePolicy(ctx context.Context, policy *BackupPolicy) error {
	// Validate primary repository exists
	_, err := pm.configRepo.GetByID(ctx, policy.PrimaryRepositoryID)
	if err != nil {
		return fmt.Errorf("primary repository not found: %w", err)
	}

	// Validate copy rules
	if err := pm.validateCopyRules(ctx, policy); err != nil {
		return fmt.Errorf("invalid copy rules: %w", err)
	}

	// Create policy
	if err := pm.policyRepo.CreatePolicy(ctx, policy); err != nil {
		return fmt.Errorf("failed to create policy: %w", err)
	}

	// Create copy rules
	for _, rule := range policy.CopyRules {
		rule.PolicyID = policy.ID
		if err := pm.policyRepo.AddCopyRule(ctx, rule); err != nil {
			return fmt.Errorf("failed to create copy rule: %w", err)
		}
	}

	return nil
}

// GetPolicy retrieves a policy by ID with its copy rules.
func (pm *PolicyManager) GetPolicy(ctx context.Context, policyID string) (*BackupPolicy, error) {
	policy, err := pm.policyRepo.GetPolicy(ctx, policyID)
	if err != nil {
		return nil, err
	}

	// Load copy rules
	rules, err := pm.policyRepo.ListCopyRules(ctx, policyID)
	if err != nil {
		return nil, fmt.Errorf("failed to load copy rules: %w", err)
	}
	policy.CopyRules = rules

	return policy, nil
}

// ListPolicies returns all backup policies.
func (pm *PolicyManager) ListPolicies(ctx context.Context) ([]*BackupPolicy, error) {
	policies, err := pm.policyRepo.ListPolicies(ctx)
	if err != nil {
		return nil, err
	}

	// Load copy rules for each policy
	for _, policy := range policies {
		rules, err := pm.policyRepo.ListCopyRules(ctx, policy.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to load copy rules for policy %s: %w", policy.ID, err)
		}
		policy.CopyRules = rules
	}

	return policies, nil
}

// DeletePolicy deletes a policy and its copy rules.
func (pm *PolicyManager) DeletePolicy(ctx context.Context, policyID string) error {
	// Check if policy is in use by any backups
	// TODO: Add check once backup_jobs.policy_id is added
	
	return pm.policyRepo.DeletePolicy(ctx, policyID)
}

// validateCopyRules validates copy rule configuration.
func (pm *PolicyManager) validateCopyRules(ctx context.Context, policy *BackupPolicy) error {
	if len(policy.CopyRules) == 0 {
		return nil // No copy rules is valid
	}

	// Check for duplicate destinations
	seen := make(map[string]bool)
	for _, rule := range policy.CopyRules {
		// Validate destination repository exists
		_, err := pm.configRepo.GetByID(ctx, rule.DestinationRepositoryID)
		if err != nil {
			return fmt.Errorf("destination repository %s not found", rule.DestinationRepositoryID)
		}

		// Check for circular dependency (copying to same repository as primary)
		if rule.DestinationRepositoryID == policy.PrimaryRepositoryID {
			return fmt.Errorf("copy rule cannot use primary repository as destination")
		}

		// Check for duplicate destinations
		if seen[rule.DestinationRepositoryID] {
			return fmt.Errorf("duplicate copy rule for repository %s", rule.DestinationRepositoryID)
		}
		seen[rule.DestinationRepositoryID] = true

		// Validate priority is positive
		if rule.Priority < 1 {
			return fmt.Errorf("copy rule priority must be >= 1")
		}
	}

	return nil
}

// OnBackupComplete is called when a backup finishes successfully.
// Creates copy jobs based on the policy's copy rules.
func (pm *PolicyManager) OnBackupComplete(ctx context.Context, backup *Backup, policyID string) error {
	// Get policy with copy rules
	policy, err := pm.GetPolicy(ctx, policyID)
	if err != nil {
		return fmt.Errorf("failed to get policy: %w", err)
	}

	if !policy.Enabled {
		return nil // Policy disabled, skip copy
	}

	// Create copy jobs for immediate copy rules
	for _, rule := range policy.CopyRules {
		if !rule.Enabled {
			continue
		}

		if rule.CopyMode == CopyModeImmediate {
			// Create backup copy record
			copy := &BackupCopy{
				ID:                 fmt.Sprintf("copy-%s-%s", backup.ID, rule.DestinationRepositoryID),
				SourceBackupID:     backup.ID,
				RepositoryID:       rule.DestinationRepositoryID,
				CopyRuleID:         rule.ID,
				Status:             BackupCopyStatusPending,
				VerificationStatus: VerificationStatusPending,
				CreatedAt:          time.Now(),
			}

			if err := pm.policyRepo.CreateBackupCopy(ctx, copy); err != nil {
				return fmt.Errorf("failed to create backup copy: %w", err)
			}
		}
	}

	return nil
}
