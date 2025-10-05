// Package storage provides SQL repository implementation for backup policies
// Following project rules: ALL database operations via repository pattern, no direct SQL in business logic
package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// SQLPolicyRepository implements PolicyRepository using SQL database.
type SQLPolicyRepository struct {
	db *sql.DB
}

// NewPolicyRepository creates a new SQL policy repository.
func NewPolicyRepository(db *sql.DB) *SQLPolicyRepository {
	return &SQLPolicyRepository{db: db}
}

// CreatePolicy creates a new backup policy.
func (r *SQLPolicyRepository) CreatePolicy(ctx context.Context, policy *BackupPolicy) error {
	query := `
		INSERT INTO backup_policies (
			id, name, enabled, primary_repository_id, retention_days, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?)
	`
	
	now := time.Now()
	policy.CreatedAt = now
	policy.UpdatedAt = now

	_, err := r.db.ExecContext(ctx, query,
		policy.ID,
		policy.Name,
		policy.Enabled,
		policy.PrimaryRepositoryID,
		policy.RetentionDays,
		policy.CreatedAt,
		policy.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create policy: %w", err)
	}

	return nil
}

// GetPolicy retrieves a policy by ID (without copy rules).
func (r *SQLPolicyRepository) GetPolicy(ctx context.Context, policyID string) (*BackupPolicy, error) {
	query := `
		SELECT id, name, enabled, primary_repository_id, retention_days, created_at, updated_at
		FROM backup_policies
		WHERE id = ?
	`

	policy := &BackupPolicy{}
	err := r.db.QueryRowContext(ctx, query, policyID).Scan(
		&policy.ID,
		&policy.Name,
		&policy.Enabled,
		&policy.PrimaryRepositoryID,
		&policy.RetentionDays,
		&policy.CreatedAt,
		&policy.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("policy not found: %s", policyID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get policy: %w", err)
	}

	return policy, nil
}

// GetPolicyByName retrieves a policy by name.
func (r *SQLPolicyRepository) GetPolicyByName(ctx context.Context, name string) (*BackupPolicy, error) {
	query := `
		SELECT id, name, enabled, primary_repository_id, retention_days, created_at, updated_at
		FROM backup_policies
		WHERE name = ?
	`

	policy := &BackupPolicy{}
	err := r.db.QueryRowContext(ctx, query, name).Scan(
		&policy.ID,
		&policy.Name,
		&policy.Enabled,
		&policy.PrimaryRepositoryID,
		&policy.RetentionDays,
		&policy.CreatedAt,
		&policy.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("policy not found: %s", name)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get policy: %w", err)
	}

	return policy, nil
}

// ListPolicies returns all backup policies (without copy rules).
func (r *SQLPolicyRepository) ListPolicies(ctx context.Context) ([]*BackupPolicy, error) {
	query := `
		SELECT id, name, enabled, primary_repository_id, retention_days, created_at, updated_at
		FROM backup_policies
		ORDER BY name
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list policies: %w", err)
	}
	defer rows.Close()

	var policies []*BackupPolicy
	for rows.Next() {
		policy := &BackupPolicy{}
		err := rows.Scan(
			&policy.ID,
			&policy.Name,
			&policy.Enabled,
			&policy.PrimaryRepositoryID,
			&policy.RetentionDays,
			&policy.CreatedAt,
			&policy.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan policy: %w", err)
		}
		policies = append(policies, policy)
	}

	return policies, nil
}

// UpdatePolicy updates an existing policy.
func (r *SQLPolicyRepository) UpdatePolicy(ctx context.Context, policy *BackupPolicy) error {
	query := `
		UPDATE backup_policies
		SET name = ?, enabled = ?, primary_repository_id = ?, retention_days = ?, updated_at = ?
		WHERE id = ?
	`

	policy.UpdatedAt = time.Now()

	result, err := r.db.ExecContext(ctx, query,
		policy.Name,
		policy.Enabled,
		policy.PrimaryRepositoryID,
		policy.RetentionDays,
		policy.UpdatedAt,
		policy.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update policy: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("policy not found: %s", policy.ID)
	}

	return nil
}

// DeletePolicy deletes a policy and its copy rules (CASCADE DELETE).
func (r *SQLPolicyRepository) DeletePolicy(ctx context.Context, policyID string) error {
	query := `DELETE FROM backup_policies WHERE id = ?`

	result, err := r.db.ExecContext(ctx, query, policyID)
	if err != nil {
		return fmt.Errorf("failed to delete policy: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("policy not found: %s", policyID)
	}

	return nil
}

// AddCopyRule creates a new copy rule.
func (r *SQLPolicyRepository) AddCopyRule(ctx context.Context, rule *BackupCopyRule) error {
	query := `
		INSERT INTO backup_copy_rules (
			id, policy_id, destination_repository_id, copy_mode, priority, enabled, verify_after_copy, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	rule.CreatedAt = time.Now()

	_, err := r.db.ExecContext(ctx, query,
		rule.ID,
		rule.PolicyID,
		rule.DestinationRepositoryID,
		rule.CopyMode,
		rule.Priority,
		rule.Enabled,
		rule.VerifyAfterCopy,
		rule.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create copy rule: %w", err)
	}

	return nil
}

// GetCopyRule retrieves a copy rule by ID.
func (r *SQLPolicyRepository) GetCopyRule(ctx context.Context, ruleID string) (*BackupCopyRule, error) {
	query := `
		SELECT id, policy_id, destination_repository_id, copy_mode, priority, enabled, verify_after_copy, created_at
		FROM backup_copy_rules
		WHERE id = ?
	`

	rule := &BackupCopyRule{}
	err := r.db.QueryRowContext(ctx, query, ruleID).Scan(
		&rule.ID,
		&rule.PolicyID,
		&rule.DestinationRepositoryID,
		&rule.CopyMode,
		&rule.Priority,
		&rule.Enabled,
		&rule.VerifyAfterCopy,
		&rule.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("copy rule not found: %s", ruleID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get copy rule: %w", err)
	}

	return rule, nil
}

// ListCopyRules returns all copy rules for a policy.
func (r *SQLPolicyRepository) ListCopyRules(ctx context.Context, policyID string) ([]*BackupCopyRule, error) {
	query := `
		SELECT id, policy_id, destination_repository_id, copy_mode, priority, enabled, verify_after_copy, created_at
		FROM backup_copy_rules
		WHERE policy_id = ?
		ORDER BY priority
	`

	rows, err := r.db.QueryContext(ctx, query, policyID)
	if err != nil {
		return nil, fmt.Errorf("failed to list copy rules: %w", err)
	}
	defer rows.Close()

	var rules []*BackupCopyRule
	for rows.Next() {
		rule := &BackupCopyRule{}
		err := rows.Scan(
			&rule.ID,
			&rule.PolicyID,
			&rule.DestinationRepositoryID,
			&rule.CopyMode,
			&rule.Priority,
			&rule.Enabled,
			&rule.VerifyAfterCopy,
			&rule.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan copy rule: %w", err)
		}
		rules = append(rules, rule)
	}

	return rules, nil
}

// UpdateCopyRule updates an existing copy rule.
func (r *SQLPolicyRepository) UpdateCopyRule(ctx context.Context, rule *BackupCopyRule) error {
	query := `
		UPDATE backup_copy_rules
		SET destination_repository_id = ?, copy_mode = ?, priority = ?, enabled = ?, verify_after_copy = ?
		WHERE id = ?
	`

	result, err := r.db.ExecContext(ctx, query,
		rule.DestinationRepositoryID,
		rule.CopyMode,
		rule.Priority,
		rule.Enabled,
		rule.VerifyAfterCopy,
		rule.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update copy rule: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("copy rule not found: %s", rule.ID)
	}

	return nil
}

// DeleteCopyRule deletes a copy rule.
func (r *SQLPolicyRepository) DeleteCopyRule(ctx context.Context, ruleID string) error {
	query := `DELETE FROM backup_copy_rules WHERE id = ?`

	result, err := r.db.ExecContext(ctx, query, ruleID)
	if err != nil {
		return fmt.Errorf("failed to delete copy rule: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("copy rule not found: %s", ruleID)
	}

	return nil
}

// CreateBackupCopy creates a new backup copy record.
func (r *SQLPolicyRepository) CreateBackupCopy(ctx context.Context, copy *BackupCopy) error {
	query := `
		INSERT INTO backup_copies (
			id, source_backup_id, repository_id, copy_rule_id, status, file_path, size_bytes,
			verification_status, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	copy.CreatedAt = time.Now()

	_, err := r.db.ExecContext(ctx, query,
		copy.ID,
		copy.SourceBackupID,
		copy.RepositoryID,
		sql.NullString{String: copy.CopyRuleID, Valid: copy.CopyRuleID != ""},
		copy.Status,
		copy.FilePath,
		copy.SizeBytes,
		copy.VerificationStatus,
		copy.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create backup copy: %w", err)
	}

	return nil
}

// GetBackupCopy retrieves a backup copy by ID.
func (r *SQLPolicyRepository) GetBackupCopy(ctx context.Context, copyID string) (*BackupCopy, error) {
	query := `
		SELECT id, source_backup_id, repository_id, copy_rule_id, status, file_path, size_bytes,
			   copy_started_at, copy_completed_at, verified_at, verification_status, error_message, created_at
		FROM backup_copies
		WHERE id = ?
	`

	copy := &BackupCopy{}
	var copyRuleID sql.NullString
	var copyStartedAt, copyCompletedAt, verifiedAt sql.NullTime

	err := r.db.QueryRowContext(ctx, query, copyID).Scan(
		&copy.ID,
		&copy.SourceBackupID,
		&copy.RepositoryID,
		&copyRuleID,
		&copy.Status,
		&copy.FilePath,
		&copy.SizeBytes,
		&copyStartedAt,
		&copyCompletedAt,
		&verifiedAt,
		&copy.VerificationStatus,
		&copy.ErrorMessage,
		&copy.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("backup copy not found: %s", copyID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get backup copy: %w", err)
	}

	// Handle nullable fields
	if copyRuleID.Valid {
		copy.CopyRuleID = copyRuleID.String
	}
	if copyStartedAt.Valid {
		copy.CopyStartedAt = &copyStartedAt.Time
	}
	if copyCompletedAt.Valid {
		copy.CopyCompletedAt = &copyCompletedAt.Time
	}
	if verifiedAt.Valid {
		copy.VerifiedAt = &verifiedAt.Time
	}

	return copy, nil
}

// ListBackupCopies returns all copies of a source backup.
func (r *SQLPolicyRepository) ListBackupCopies(ctx context.Context, sourceBackupID string) ([]*BackupCopy, error) {
	query := `
		SELECT id, source_backup_id, repository_id, copy_rule_id, status, file_path, size_bytes,
			   copy_started_at, copy_completed_at, verified_at, verification_status, error_message, created_at
		FROM backup_copies
		WHERE source_backup_id = ?
		ORDER BY created_at
	`

	rows, err := r.db.QueryContext(ctx, query, sourceBackupID)
	if err != nil {
		return nil, fmt.Errorf("failed to list backup copies: %w", err)
	}
	defer rows.Close()

	var copies []*BackupCopy
	for rows.Next() {
		copy := &BackupCopy{}
		var copyRuleID sql.NullString
		var copyStartedAt, copyCompletedAt, verifiedAt sql.NullTime

		err := rows.Scan(
			&copy.ID,
			&copy.SourceBackupID,
			&copy.RepositoryID,
			&copyRuleID,
			&copy.Status,
			&copy.FilePath,
			&copy.SizeBytes,
			&copyStartedAt,
			&copyCompletedAt,
			&verifiedAt,
			&copy.VerificationStatus,
			&copy.ErrorMessage,
			&copy.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan backup copy: %w", err)
		}

		// Handle nullable fields
		if copyRuleID.Valid {
			copy.CopyRuleID = copyRuleID.String
		}
		if copyStartedAt.Valid {
			copy.CopyStartedAt = &copyStartedAt.Time
		}
		if copyCompletedAt.Valid {
			copy.CopyCompletedAt = &copyCompletedAt.Time
		}
		if verifiedAt.Valid {
			copy.VerifiedAt = &verifiedAt.Time
		}

		copies = append(copies, copy)
	}

	return copies, nil
}

// ListPendingCopies returns all pending backup copies.
func (r *SQLPolicyRepository) ListPendingCopies(ctx context.Context) ([]*BackupCopy, error) {
	query := `
		SELECT id, source_backup_id, repository_id, copy_rule_id, status, file_path, size_bytes,
			   copy_started_at, copy_completed_at, verified_at, verification_status, error_message, created_at
		FROM backup_copies
		WHERE status = ?
		ORDER BY created_at
	`

	rows, err := r.db.QueryContext(ctx, query, BackupCopyStatusPending)
	if err != nil {
		return nil, fmt.Errorf("failed to list pending copies: %w", err)
	}
	defer rows.Close()

	var copies []*BackupCopy
	for rows.Next() {
		copy := &BackupCopy{}
		var copyRuleID sql.NullString
		var copyStartedAt, copyCompletedAt, verifiedAt sql.NullTime

		err := rows.Scan(
			&copy.ID,
			&copy.SourceBackupID,
			&copy.RepositoryID,
			&copyRuleID,
			&copy.Status,
			&copy.FilePath,
			&copy.SizeBytes,
			&copyStartedAt,
			&copyCompletedAt,
			&verifiedAt,
			&copy.VerificationStatus,
			&copy.ErrorMessage,
			&copy.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan backup copy: %w", err)
		}

		// Handle nullable fields
		if copyRuleID.Valid {
			copy.CopyRuleID = copyRuleID.String
		}
		if copyStartedAt.Valid {
			copy.CopyStartedAt = &copyStartedAt.Time
		}
		if copyCompletedAt.Valid {
			copy.CopyCompletedAt = &copyCompletedAt.Time
		}
		if verifiedAt.Valid {
			copy.VerifiedAt = &verifiedAt.Time
		}

		copies = append(copies, copy)
	}

	return copies, nil
}

// UpdateBackupCopyStatus updates the status of a backup copy.
func (r *SQLPolicyRepository) UpdateBackupCopyStatus(ctx context.Context, copyID string, status BackupCopyStatus, errorMsg string) error {
	var query string
	var args []interface{}

	if status == BackupCopyStatusCopying {
		query = `UPDATE backup_copies SET status = ?, copy_started_at = ?, error_message = ? WHERE id = ?`
		args = []interface{}{status, time.Now(), errorMsg, copyID}
	} else if status == BackupCopyStatusCompleted {
		query = `UPDATE backup_copies SET status = ?, copy_completed_at = ?, error_message = ? WHERE id = ?`
		args = []interface{}{status, time.Now(), errorMsg, copyID}
	} else {
		query = `UPDATE backup_copies SET status = ?, error_message = ? WHERE id = ?`
		args = []interface{}{status, errorMsg, copyID}
	}

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update backup copy status: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("backup copy not found: %s", copyID)
	}

	return nil
}

// UpdateBackupCopyVerification updates verification status of a backup copy.
func (r *SQLPolicyRepository) UpdateBackupCopyVerification(ctx context.Context, copyID string, verified bool) error {
	var verificationStatus VerificationStatus
	if verified {
		verificationStatus = VerificationStatusPassed
	} else {
		verificationStatus = VerificationStatusFailed
	}

	query := `UPDATE backup_copies SET verification_status = ?, verified_at = ? WHERE id = ?`

	result, err := r.db.ExecContext(ctx, query, verificationStatus, time.Now(), copyID)
	if err != nil {
		return fmt.Errorf("failed to update verification status: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("backup copy not found: %s", copyID)
	}

	return nil
}
