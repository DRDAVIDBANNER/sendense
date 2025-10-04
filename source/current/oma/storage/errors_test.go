package storage

import (
	"errors"
	"testing"
)

func TestRepositoryError(t *testing.T) {
	tests := []struct {
		name         string
		err          *RepositoryError
		expectedMsg  string
		expectedRepo string
	}{
		{
			name: "basic error",
			err: &RepositoryError{
				RepositoryID: "repo-123",
				Op:           "create",
				Err:          errors.New("disk full"),
			},
			expectedMsg:  "repository repo-123: create: disk full",
			expectedRepo: "repo-123",
		},
		{
			name: "error without repository ID",
			err: &RepositoryError{
				Op:  "list",
				Err: errors.New("database connection failed"),
			},
			expectedMsg:  "repository : list: database connection failed",
			expectedRepo: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.expectedMsg {
				t.Errorf("RepositoryError.Error() = %v, want %v", got, tt.expectedMsg)
			}
			if got := tt.err.RepositoryID; got != tt.expectedRepo {
				t.Errorf("RepositoryError.RepositoryID = %v, want %v", got, tt.expectedRepo)
			}
		})
	}
}

func TestRepositoryErrorUnwrap(t *testing.T) {
	innerErr := errors.New("inner error")
	repoErr := &RepositoryError{
		RepositoryID: "repo-123",
		Op:           "delete",
		Err:          innerErr,
	}

	if unwrapped := repoErr.Unwrap(); unwrapped != innerErr {
		t.Errorf("RepositoryError.Unwrap() = %v, want %v", unwrapped, innerErr)
	}
}

func TestBackupError(t *testing.T) {
	tests := []struct {
		name         string
		err          *BackupError
		expectedMsg  string
		expectedID   string
		expectedRepo string
	}{
		{
			name: "complete error",
			err: &BackupError{
				BackupID:     "backup-456",
				RepositoryID: "repo-123",
				Op:           "verify",
				Err:          errors.New("checksum mismatch"),
			},
			expectedMsg:  "backup backup-456 in repository repo-123: verify: checksum mismatch",
			expectedID:   "backup-456",
			expectedRepo: "repo-123",
		},
		{
			name: "error without backup ID",
			err: &BackupError{
				RepositoryID: "repo-123",
				Op:           "create",
				Err:          errors.New("insufficient space"),
			},
			expectedMsg:  "backup  in repository repo-123: create: insufficient space",
			expectedID:   "",
			expectedRepo: "repo-123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.expectedMsg {
				t.Errorf("BackupError.Error() = %v, want %v", got, tt.expectedMsg)
			}
			if got := tt.err.BackupID; got != tt.expectedID {
				t.Errorf("BackupError.BackupID = %v, want %v", got, tt.expectedID)
			}
			if got := tt.err.RepositoryID; got != tt.expectedRepo {
				t.Errorf("BackupError.RepositoryID = %v, want %v", got, tt.expectedRepo)
			}
		})
	}
}

func TestBackupErrorUnwrap(t *testing.T) {
	innerErr := errors.New("inner error")
	backupErr := &BackupError{
		BackupID:     "backup-456",
		RepositoryID: "repo-123",
		Op:           "restore",
		Err:          innerErr,
	}

	if unwrapped := backupErr.Unwrap(); unwrapped != innerErr {
		t.Errorf("BackupError.Unwrap() = %v, want %v", unwrapped, innerErr)
	}
}

func TestPredefinedErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
		msg  string
	}{
		{"ErrRepositoryNotFound", ErrRepositoryNotFound, "repository not found"},
		{"ErrBackupNotFound", ErrBackupNotFound, "backup not found"},
		{"ErrInsufficientSpace", ErrInsufficientSpace, "insufficient storage space"},
		{"ErrCorruptChain", ErrCorruptChain, "backup chain is corrupted"},
		{"ErrInvalidBackupType", ErrInvalidBackupType, "invalid backup type"},
		{"ErrBackupInProgress", ErrBackupInProgress, "backup operation in progress"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Error() != tt.msg {
				t.Errorf("%s.Error() = %v, want %v", tt.name, tt.err.Error(), tt.msg)
			}
		})
	}
}

func TestErrorsIsComparison(t *testing.T) {
	// Test that errors.Is works correctly with our custom errors
	repoErr := &RepositoryError{
		RepositoryID: "repo-123",
		Op:           "create",
		Err:          ErrInsufficientSpace,
	}

	if !errors.Is(repoErr, ErrInsufficientSpace) {
		t.Error("errors.Is should identify wrapped ErrInsufficientSpace")
	}

	backupErr := &BackupError{
		BackupID:     "backup-456",
		RepositoryID: "repo-123",
		Op:           "verify",
		Err:          ErrCorruptChain,
	}

	if !errors.Is(backupErr, ErrCorruptChain) {
		t.Error("errors.Is should identify wrapped ErrCorruptChain")
	}
}
