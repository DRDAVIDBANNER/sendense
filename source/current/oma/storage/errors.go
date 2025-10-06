package storage

import (
	"errors"
	"fmt"
)

var (
	// ErrBackupNotFound is returned when a backup cannot be found.
	ErrBackupNotFound = errors.New("backup not found")

	// ErrBackupChainNotFound is returned when a backup chain cannot be found.
	ErrBackupChainNotFound = errors.New("backup chain not found")

	// ErrBackupChainCorrupt is returned when a backup chain is broken or invalid.
	ErrBackupChainCorrupt = errors.New("backup chain is corrupt")

	// ErrRepositoryNotFound is returned when a repository cannot be found.
	ErrRepositoryNotFound = errors.New("repository not found")

	// ErrRepositoryFull is returned when repository has insufficient space.
	ErrRepositoryFull = errors.New("repository has insufficient space")

	// ErrRepositoryOffline is returned when repository is not accessible.
	ErrRepositoryOffline = errors.New("repository is offline or not mounted")

	// ErrInvalidBackupType is returned when backup type is not supported.
	ErrInvalidBackupType = errors.New("invalid backup type")

	// ErrParentBackupRequired is returned when incremental backup is missing parent.
	ErrParentBackupRequired = errors.New("parent backup required for incremental backup")

	// ErrParentBackupNotFound is returned when parent backup doesn't exist.
	ErrParentBackupNotFound = errors.New("parent backup not found")

	// ErrBackupHasDependents is returned when attempting to delete backup with children.
	ErrBackupHasDependents = errors.New("cannot delete backup: has dependent incrementals")

	// ErrImmutableBackup is returned when attempting to delete immutable backup.
	ErrImmutableBackup = errors.New("cannot delete immutable backup before retention period")

	// ErrBackupInProgress is returned when backup is currently running.
	ErrBackupInProgress = errors.New("backup operation in progress")

	// ErrMountFailed is returned when repository mount fails.
	ErrMountFailed = errors.New("failed to mount repository")

	// ErrUnmountFailed is returned when repository unmount fails.
	ErrUnmountFailed = errors.New("failed to unmount repository")

	// ErrQCOW2Operation is returned when QCOW2 operation fails.
	ErrQCOW2Operation = errors.New("QCOW2 operation failed")
)

// BackupError wraps an error with backup-specific context.
type BackupError struct {
	BackupID string
	Op       string // Operation that failed (create, delete, verify, etc.)
	Err      error
}

func (e *BackupError) Error() string {
	if e.BackupID != "" {
		return fmt.Sprintf("backup %s: %s: %v", e.BackupID, e.Op, e.Err)
	}
	return fmt.Sprintf("%s: %v", e.Op, e.Err)
}

func (e *BackupError) Unwrap() error {
	return e.Err
}

// RepositoryError wraps an error with repository-specific context.
type RepositoryError struct {
	RepositoryID string
	Op           string
	Err          error
}

func (e *RepositoryError) Error() string {
	if e.RepositoryID != "" {
		return fmt.Sprintf("repository %s: %s: %v", e.RepositoryID, e.Op, e.Err)
	}
	return fmt.Sprintf("%s: %v", e.Op, e.Err)
}

func (e *RepositoryError) Unwrap() error {
	return e.Err
}

// ChainError wraps an error with chain-specific context.
type ChainError struct {
	ChainID string
	Op      string
	Err     error
}

func (e *ChainError) Error() string {
	if e.ChainID != "" {
		return fmt.Sprintf("backup chain %s: %s: %v", e.ChainID, e.Op, e.Err)
	}
	return fmt.Sprintf("%s: %v", e.Op, e.Err)
}

func (e *ChainError) Unwrap() error {
	return e.Err
}

// InsufficientSpaceError provides details about storage capacity.
type InsufficientSpaceError struct {
	Required  int64
	Available int64
}

func (e *InsufficientSpaceError) Error() string {
	return fmt.Sprintf("insufficient space: required %d bytes, available %d bytes", 
		e.Required, e.Available)
}

