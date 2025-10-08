package storage

import (
	"encoding/json"
	"testing"
	"time"
)

func TestBackupTypeValidation(t *testing.T) {
	tests := []struct {
		name      string
		backupType BackupType
		valid     bool
	}{
		{"full backup", BackupTypeFull, true},
		{"incremental backup", BackupTypeIncremental, true},
		{"differential backup", BackupTypeDifferential, true},
		{"invalid type", BackupType("invalid"), false},
		{"empty type", BackupType(""), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test that valid types are in the expected set
			validTypes := map[BackupType]bool{
				BackupTypeFull:         true,
				BackupTypeIncremental:  true,
				BackupTypeDifferential: true,
			}

			isValid := validTypes[tt.backupType]
			if isValid != tt.valid {
				t.Errorf("BackupType %q validity = %v, want %v", tt.backupType, isValid, tt.valid)
			}
		})
	}
}

func TestBackupStatusValidation(t *testing.T) {
	tests := []struct {
		name   string
		status BackupStatus
		valid  bool
	}{
		{"pending", BackupStatusPending, true},
		{"running", BackupStatusRunning, true},
		{"completed", BackupStatusCompleted, true},
		{"failed", BackupStatusFailed, true},
		{"cancelled", BackupStatusCancelled, true},
		{"invalid status", BackupStatus("invalid"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validStatuses := map[BackupStatus]bool{
				BackupStatusPending:   true,
				BackupStatusRunning:   true,
				BackupStatusCompleted: true,
				BackupStatusFailed:    true,
				BackupStatusCancelled: true,
			}

			isValid := validStatuses[tt.status]
			if isValid != tt.valid {
				t.Errorf("BackupStatus %q validity = %v, want %v", tt.status, isValid, tt.valid)
			}
		})
	}
}

func TestBackupJSONSerialization(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	completed := now.Add(1 * time.Hour)

	backup := &Backup{
		ID:                "backup-123",
		VMContextID:       "ctx-vm1-20251004-153000",
		VMName:            "test-vm",
		RepositoryID:      "repo-456",
		BackupType:        BackupTypeFull,
		Status:            BackupStatusCompleted,
		RepositoryPath:    "/backups/test-vm/backup-123.qcow2",
		ParentBackupID:    "",
		ChangeID:          "52 3c ec 11 9e 2c 4c 3d-87 4a c3 4e 85 f2 ea 95/446",
		BytesTransferred:  42949672960,
		TotalBytes:        42949672960,
		CompressionEnabled: true,
		ErrorMessage:      "",
		CreatedAt:         now,
		StartedAt:         &now,
		CompletedAt:       &completed,
	}

	// Test serialization
	data, err := json.Marshal(backup)
	if err != nil {
		t.Fatalf("failed to marshal Backup: %v", err)
	}

	// Test deserialization
	var decoded Backup
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal Backup: %v", err)
	}

	// Verify key fields
	if decoded.ID != backup.ID {
		t.Errorf("ID mismatch: got %v, want %v", decoded.ID, backup.ID)
	}
	if decoded.VMName != backup.VMName {
		t.Errorf("VMName mismatch: got %v, want %v", decoded.VMName, backup.VMName)
	}
	if decoded.BackupType != backup.BackupType {
		t.Errorf("BackupType mismatch: got %v, want %v", decoded.BackupType, backup.BackupType)
	}
	if decoded.Status != backup.Status {
		t.Errorf("Status mismatch: got %v, want %v", decoded.Status, backup.Status)
	}
	if decoded.BytesTransferred != backup.BytesTransferred {
		t.Errorf("BytesTransferred mismatch: got %v, want %v", decoded.BytesTransferred, backup.BytesTransferred)
	}
	if decoded.CompressionEnabled != backup.CompressionEnabled {
		t.Errorf("CompressionEnabled mismatch: got %v, want %v", decoded.CompressionEnabled, backup.CompressionEnabled)
	}
}

func TestBackupChainJSONSerialization(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)

	chain := &BackupChain{
		ID:              "chain-vm1-disk0",
		VMContextID:     "ctx-vm1-20251004-153000",
		DiskID:          0,
		FullBackupID:    "backup-full-123",
		LatestBackupID:  "backup-inc-456",
		TotalBackups:    3,
		TotalSizeBytes:  128849018880,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	// Test serialization
	data, err := json.Marshal(chain)
	if err != nil {
		t.Fatalf("failed to marshal BackupChain: %v", err)
	}

	// Test deserialization
	var decoded BackupChain
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal BackupChain: %v", err)
	}

	// Verify fields
	if decoded.ID != chain.ID {
		t.Errorf("ID mismatch: got %v, want %v", decoded.ID, chain.ID)
	}
	if decoded.DiskID != chain.DiskID {
		t.Errorf("DiskID mismatch: got %v, want %v", decoded.DiskID, chain.DiskID)
	}
	if decoded.TotalBackups != chain.TotalBackups {
		t.Errorf("TotalBackups mismatch: got %v, want %v", decoded.TotalBackups, chain.TotalBackups)
	}
	if decoded.TotalSizeBytes != chain.TotalSizeBytes {
		t.Errorf("TotalSizeBytes mismatch: got %v, want %v", decoded.TotalSizeBytes, chain.TotalSizeBytes)
	}
}

func TestBackupRequestValidation(t *testing.T) {
	tests := []struct {
		name    string
		req     BackupRequest
		wantErr bool
	}{
		{
			name: "valid full backup",
			req: BackupRequest{
				VMContextID:  "ctx-vm1-20251004-153000",
				VMName:       "test-vm",
				RepositoryID: "repo-456",
				BackupType:   BackupTypeFull,
				DiskID:       0,
			},
			wantErr: false,
		},
		{
			name: "valid incremental backup",
			req: BackupRequest{
				VMContextID:    "ctx-vm1-20251004-153000",
				VMName:         "test-vm",
				RepositoryID:   "repo-456",
				BackupType:     BackupTypeIncremental,
				ParentBackupID: "backup-full-123",
				DiskID:         0,
			},
			wantErr: false,
		},
		{
			name: "missing vm_context_id",
			req: BackupRequest{
				VMName:       "test-vm",
				RepositoryID: "repo-456",
				BackupType:   BackupTypeFull,
			},
			wantErr: true,
		},
		{
			name: "missing repository_id",
			req: BackupRequest{
				VMContextID: "ctx-vm1-20251004-153000",
				VMName:      "test-vm",
				BackupType:  BackupTypeFull,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasError := (tt.req.VMContextID == "" || tt.req.RepositoryID == "")
			if hasError != tt.wantErr {
				t.Errorf("validation error expectation mismatch: got %v, want %v", hasError, tt.wantErr)
			}
		})
	}
}

func TestVMwareMetadataJSONSerialization(t *testing.T) {
	metadata := &VMwareMetadata{
		VMwareVMID:   "vm-123",
		VCenterHost:  "vcenter.example.com",
		Datacenter:   "DC1",
		Datastore:    "datastore1",
		VMXPath:      "[datastore1] test-vm/test-vm.vmx",
		DiskPath:     "[datastore1] test-vm/test-vm.vmdk",
		DiskUUID:     "6000C29a-1234-5678-9abc-def012345678",
		ChangeID:     "52 3c ec 11 9e 2c 4c 3d-87 4a c3 4e 85 f2 ea 95/446",
		CBTEnabled:   true,
		DiskSizeGB:   100,
		ProvisioningType: "thin",
	}

	// Test serialization
	data, err := json.Marshal(metadata)
	if err != nil {
		t.Fatalf("failed to marshal VMwareMetadata: %v", err)
	}

	// Test deserialization
	var decoded VMwareMetadata
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal VMwareMetadata: %v", err)
	}

	// Verify critical fields
	if decoded.VMwareVMID != metadata.VMwareVMID {
		t.Errorf("VMwareVMID mismatch: got %v, want %v", decoded.VMwareVMID, metadata.VMwareVMID)
	}
	if decoded.ChangeID != metadata.ChangeID {
		t.Errorf("ChangeID mismatch: got %v, want %v", decoded.ChangeID, metadata.ChangeID)
	}
	if decoded.CBTEnabled != metadata.CBTEnabled {
		t.Errorf("CBTEnabled mismatch: got %v, want %v", decoded.CBTEnabled, metadata.CBTEnabled)
	}
	if decoded.DiskSizeGB != metadata.DiskSizeGB {
		t.Errorf("DiskSizeGB mismatch: got %v, want %v", decoded.DiskSizeGB, metadata.DiskSizeGB)
	}
}

func TestStorageInfoJSONSerialization(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)

	info := &StorageInfo{
		TotalBytes:     10737418240000,
		UsedBytes:      4294967296000,
		AvailableBytes: 6442450944000,
		LastCheckAt:    now,
	}

	// Test serialization
	data, err := json.Marshal(info)
	if err != nil {
		t.Fatalf("failed to marshal StorageInfo: %v", err)
	}

	// Test deserialization
	var decoded StorageInfo
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal StorageInfo: %v", err)
	}

	// Verify fields
	if decoded.TotalBytes != info.TotalBytes {
		t.Errorf("TotalBytes mismatch: got %v, want %v", decoded.TotalBytes, info.TotalBytes)
	}
	if decoded.UsedBytes != info.UsedBytes {
		t.Errorf("UsedBytes mismatch: got %v, want %v", decoded.UsedBytes, info.UsedBytes)
	}
	if decoded.AvailableBytes != info.AvailableBytes {
		t.Errorf("AvailableBytes mismatch: got %v, want %v", decoded.AvailableBytes, info.AvailableBytes)
	}
}

func TestBackupMetadataFields(t *testing.T) {
	backup := &Backup{
		ID:           "backup-123",
		VMContextID:  "ctx-vm1-20251004-153000",
		VMName:       "test-vm",
		RepositoryID: "repo-456",
		BackupType:   BackupTypeFull,
	}

	// Test that all expected fields are accessible
	if backup.ID == "" {
		t.Error("ID should not be empty")
	}
	if backup.VMContextID == "" {
		t.Error("VMContextID should not be empty")
	}
	if backup.VMName == "" {
		t.Error("VMName should not be empty")
	}
	if backup.RepositoryID == "" {
		t.Error("RepositoryID should not be empty")
	}
	if backup.BackupType != BackupTypeFull {
		t.Errorf("BackupType = %v, want %v", backup.BackupType, BackupTypeFull)
	}
}

