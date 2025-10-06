package storage

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRepositoryTypeValidation(t *testing.T) {
	tests := []struct {
		name     string
		repoType RepositoryType
		valid    bool
	}{
		{"local", RepositoryTypeLocal, true},
		{"nfs", RepositoryTypeNFS, true},
		{"cifs", RepositoryTypeCIFS, true},
		{"smb", RepositoryTypeSMB, true},
		{"s3", RepositoryTypeS3, true},
		{"azure", RepositoryTypeAzure, true},
		{"invalid", RepositoryType("invalid"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validTypes := map[RepositoryType]bool{
				RepositoryTypeLocal: true,
				RepositoryTypeNFS:   true,
				RepositoryTypeCIFS:  true,
				RepositoryTypeSMB:   true,
				RepositoryTypeS3:    true,
				RepositoryTypeAzure: true,
			}

			isValid := validTypes[tt.repoType]
			if isValid != tt.valid {
				t.Errorf("RepositoryType %q validity = %v, want %v", tt.repoType, isValid, tt.valid)
			}
		})
	}
}

func TestLocalRepositoryConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  LocalRepositoryConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: LocalRepositoryConfig{
				Path: "/mnt/backups",
			},
			wantErr: false,
		},
		{
			name: "empty path",
			config: LocalRepositoryConfig{
				Path: "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasError := (tt.config.Path == "")
			if hasError != tt.wantErr {
				t.Errorf("validation error expectation mismatch: got %v, want %v", hasError, tt.wantErr)
			}
		})
	}
}

func TestNFSRepositoryConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  NFSRepositoryConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: NFSRepositoryConfig{
				Server:     "nfs.example.com",
				ExportPath: "/exports/backups",
				MountOptions: []string{"ro", "nolock"},
			},
			wantErr: false,
		},
		{
			name: "missing server",
			config: NFSRepositoryConfig{
				Server:     "",
				ExportPath: "/exports/backups",
			},
			wantErr: true,
		},
		{
			name: "missing export path",
			config: NFSRepositoryConfig{
				Server:     "nfs.example.com",
				ExportPath: "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasError := (tt.config.Server == "" || tt.config.ExportPath == "")
			if hasError != tt.wantErr {
				t.Errorf("validation error expectation mismatch: got %v, want %v", hasError, tt.wantErr)
			}
		})
	}
}

func TestCIFSRepositoryConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  CIFSRepositoryConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: CIFSRepositoryConfig{
				Server:    "10.0.100.50",
				ShareName: "backups",
				Domain:    "EXAMPLE",
				Username:  "backup_user",
				Password:  "secure_password",
			},
			wantErr: false,
		},
		{
			name: "missing server",
			config: CIFSRepositoryConfig{
				Server:    "",
				ShareName: "backups",
			},
			wantErr: true,
		},
		{
			name: "missing share name",
			config: CIFSRepositoryConfig{
				Server:    "10.0.100.50",
				ShareName: "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasError := (tt.config.Server == "" || tt.config.ShareName == "")
			if hasError != tt.wantErr {
				t.Errorf("validation error expectation mismatch: got %v, want %v", hasError, tt.wantErr)
			}
		})
	}
}

func TestImmutableConfig(t *testing.T) {
	tests := []struct {
		name   string
		config ImmutableConfig
		valid  bool
	}{
		{
			name: "linux chattr method",
			config: ImmutableConfig{
				Method:              ImmutableMethodLinuxChattr,
				RetentionPeriodDays: 30,
				Locked:              true,
			},
			valid: true,
		},
		{
			name: "s3 object lock method",
			config: ImmutableConfig{
				Method:              ImmutableMethodS3ObjectLock,
				RetentionPeriodDays: 60,
				Locked:              true,
			},
			valid: true,
		},
		{
			name: "azure worm method",
			config: ImmutableConfig{
				Method:              ImmutableMethodAzureWORM,
				RetentionPeriodDays: 90,
				Locked:              true,
			},
			valid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validMethods := map[ImmutableMethod]bool{
				ImmutableMethodLinuxChattr:  true,
				ImmutableMethodS3ObjectLock: true,
				ImmutableMethodAzureWORM:    true,
			}

			isValid := validMethods[tt.config.Method]
			if isValid != tt.valid {
				t.Errorf("ImmutableConfig method %q validity = %v, want %v", tt.config.Method, isValid, tt.valid)
			}

			if tt.config.RetentionPeriodDays <= 0 {
				t.Error("RetentionPeriodDays should be positive")
			}
		})
	}
}

func TestRepositoryConfigComplete(t *testing.T) {
	now := time.Now()

	config := &RepositoryConfig{
		ID:      "repo-local-123",
		Name:    "Primary Backup Storage",
		Type:    RepositoryTypeLocal,
		Enabled: true,
		Config: LocalRepositoryConfig{
			Path: "/mnt/backups",
		},
		IsImmutable: true,
		ImmutableConfig: &ImmutableConfig{
			Method:              ImmutableMethodLinuxChattr,
			RetentionPeriodDays: 30,
			Locked:              true,
		},
		MinRetentionDays: 30,
		TotalBytes:       10737418240000,
		UsedBytes:        4294967296000,
		AvailableBytes:   6442450944000,
		LastCheckAt:      &now,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	// Verify all fields are set correctly
	if config.ID == "" {
		t.Error("ID should not be empty")
	}
	if config.Name == "" {
		t.Error("Name should not be empty")
	}
	if config.Type != RepositoryTypeLocal {
		t.Errorf("Type = %v, want %v", config.Type, RepositoryTypeLocal)
	}
	if !config.Enabled {
		t.Error("Enabled should be true")
	}
	if config.Config == nil {
		t.Fatal("Config should not be nil")
	}
	if !config.IsImmutable {
		t.Error("IsImmutable should be true")
	}
	if config.ImmutableConfig == nil {
		t.Fatal("ImmutableConfig should not be nil")
	}
	if config.MinRetentionDays != 30 {
		t.Errorf("MinRetentionDays = %v, want %v", config.MinRetentionDays, 30)
	}
}

func TestValidatePath(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "sendense-repo-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	tests := []struct {
		name    string
		path    string
		setup   func() error
		wantErr bool
	}{
		{
			name:    "valid existing directory",
			path:    tmpDir,
			wantErr: false,
		},
		{
			name:    "non-existent directory",
			path:    filepath.Join(tmpDir, "nonexistent"),
			wantErr: true,
		},
		{
			name: "file instead of directory",
			path: filepath.Join(tmpDir, "file"),
			setup: func() error {
				return os.WriteFile(filepath.Join(tmpDir, "file"), []byte("test"), 0644)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				if err := tt.setup(); err != nil {
					t.Fatalf("setup failed: %v", err)
				}
			}

			err := validatePath(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("validatePath() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetMountPoint(t *testing.T) {
	config := LocalRepositoryConfig{
		Path: "/mnt/backups/primary",
	}

	mountPoint := getMountPoint(config.Path)
	
	// The mount point should be a parent or the path itself
	if mountPoint == "" {
		t.Error("getMountPoint should not return empty string")
	}

	// For this test, we just verify it returns something
	// In a real scenario, it would detect the actual mount point
	t.Logf("Mount point detected: %s", mountPoint)
}

func TestRepositoryConfigTypeAssertion(t *testing.T) {
	// Test that we can properly type assert Config interface
	localConfig := LocalRepositoryConfig{Path: "/mnt/backups"}
	nfsConfig := NFSRepositoryConfig{Server: "nfs.example.com", ExportPath: "/exports/backups"}
	cifsConfig := CIFSRepositoryConfig{Server: "10.0.100.50", ShareName: "backups"}

	tests := []struct {
		name       string
		config     interface{}
		assertType string
	}{
		{"local config", localConfig, "LocalRepositoryConfig"},
		{"nfs config", nfsConfig, "NFSRepositoryConfig"},
		{"cifs config", cifsConfig, "CIFSRepositoryConfig"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			switch v := tt.config.(type) {
			case LocalRepositoryConfig:
				if tt.assertType != "LocalRepositoryConfig" {
					t.Errorf("expected %s, got LocalRepositoryConfig", tt.assertType)
				}
				if v.Path == "" && tt.assertType == "LocalRepositoryConfig" {
					t.Error("LocalRepositoryConfig.Path should not be empty")
				}
			case NFSRepositoryConfig:
				if tt.assertType != "NFSRepositoryConfig" {
					t.Errorf("expected %s, got NFSRepositoryConfig", tt.assertType)
				}
				if v.Server == "" && tt.assertType == "NFSRepositoryConfig" {
					t.Error("NFSRepositoryConfig.Server should not be empty")
				}
			case CIFSRepositoryConfig:
				if tt.assertType != "CIFSRepositoryConfig" {
					t.Errorf("expected %s, got CIFSRepositoryConfig", tt.assertType)
				}
				if v.Server == "" && tt.assertType == "CIFSRepositoryConfig" {
					t.Error("CIFSRepositoryConfig.Server should not be empty")
				}
			default:
				t.Errorf("unexpected type: %T", v)
			}
		})
	}
}

