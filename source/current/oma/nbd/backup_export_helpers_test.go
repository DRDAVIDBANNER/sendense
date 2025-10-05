package nbd

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestBuildBackupExportName tests the backup export name generation
func TestBuildBackupExportName(t *testing.T) {
	tests := []struct {
		name        string
		vmContextID string
		diskID      int
		backupType  string
		timestamp   time.Time
		wantPrefix  string
		wantLen     int
	}{
		{
			name:        "short context ID",
			vmContextID: "ctx-pgtest2-20251005-120000",
			diskID:      0,
			backupType:  "full",
			timestamp:   time.Date(2025, 10, 5, 12, 0, 0, 0, time.UTC),
			wantPrefix:  "backup-ctx-pgtest2-20251005-120000-disk0-full-",
			wantLen:     57, // Should be under 64
		},
		{
			name:        "multi-disk VM",
			vmContextID: "ctx-pgtest2-20251005-120000",
			diskID:      3,
			backupType:  "incr",
			timestamp:   time.Date(2025, 10, 5, 13, 30, 45, 0, time.UTC),
			wantPrefix:  "backup-ctx-pgtest2-20251005-120000-disk3-incr-",
			wantLen:     57, // Should be under 64
		},
		{
			name:        "very long context ID (truncation test)",
			vmContextID: "ctx-very-long-vm-name-that-might-cause-export-name-to-exceed-nbd-limit-12345",
			diskID:      0,
			backupType:  "full",
			timestamp:   time.Date(2025, 10, 5, 12, 0, 0, 0, time.UTC),
			wantPrefix:  "backup-ctx-very-long-vm-name-that-might-cause-export",
			wantLen:     63, // Should be exactly 63 (NBD limit)
		},
		{
			name:        "incremental backup",
			vmContextID: "ctx-wintest-20251005-140000",
			diskID:      1,
			backupType:  "incr",
			timestamp:   time.Date(2025, 10, 5, 14, 15, 30, 0, time.UTC),
			wantPrefix:  "backup-ctx-wintest-20251005-140000-disk1-incr-",
			wantLen:     56,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildBackupExportName(tt.vmContextID, tt.diskID, tt.backupType, tt.timestamp)

			// Check length constraint
			if len(got) > 63 {
				t.Errorf("BuildBackupExportName() length = %d, want <= 63", len(got))
			}

			// Check prefix
			if len(got) >= len(tt.wantPrefix) && got[:len(tt.wantPrefix)] != tt.wantPrefix {
				t.Errorf("BuildBackupExportName() prefix = %s, want %s", got[:len(tt.wantPrefix)], tt.wantPrefix)
			}

			// Check total length is reasonable
			if len(got) < 30 {
				t.Errorf("BuildBackupExportName() length = %d, too short (minimum 30 expected)", len(got))
			}

			t.Logf("✅ Export name: %s (length: %d)", got, len(got))
		})
	}
}

// TestIsBackupExport tests the backup export detection
func TestIsBackupExport(t *testing.T) {
	tests := []struct {
		name       string
		exportName string
		want       bool
	}{
		{
			name:       "valid backup export",
			exportName: "backup-ctx-pgtest2-20251005-120000-disk0-full-20251005T120000",
			want:       true,
		},
		{
			name:       "valid incremental backup",
			exportName: "backup-ctx-pgtest2-20251005-120000-disk1-incr-20251005T130000",
			want:       true,
		},
		{
			name:       "migration export (not backup)",
			exportName: "migration-vm-a1b2c3d4-e5f6-7890-abcd-ef1234567890-disk0",
			want:       false,
		},
		{
			name:       "invalid format",
			exportName: "some-random-export-name",
			want:       false,
		},
		{
			name:       "empty string",
			exportName: "",
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsBackupExport(tt.exportName); got != tt.want {
				t.Errorf("IsBackupExport() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestParseBackupExportName tests parsing of backup export names
func TestParseBackupExportName(t *testing.T) {
	tests := []struct {
		name            string
		exportName      string
		wantVMContextID string
		wantDiskID      int
		wantBackupType  string
		wantError       bool
	}{
		{
			name:            "valid full backup",
			exportName:      "backup-ctx-pgtest2-20251005-120000-disk0-full-20251005T120000",
			wantVMContextID: "ctx-pgtest2-20251005-120000",
			wantDiskID:      0,
			wantBackupType:  "full",
			wantError:       false,
		},
		{
			name:            "valid incremental backup",
			exportName:      "backup-ctx-wintest-20251005-140000-disk1-incr-20251005T140000",
			wantVMContextID: "ctx-wintest-20251005-140000",
			wantDiskID:      1,
			wantBackupType:  "incr",
			wantError:       false,
		},
		{
			name:       "invalid format (not backup)",
			exportName: "migration-vm-test-disk0",
			wantError:  true,
		},
		{
			name:       "malformed backup name",
			exportName: "backup-incomplete",
			wantError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotVMContextID, gotDiskID, gotBackupType, _, err := ParseBackupExportName(tt.exportName)

			if (err != nil) != tt.wantError {
				t.Errorf("ParseBackupExportName() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if !tt.wantError {
				if gotVMContextID != tt.wantVMContextID {
					t.Errorf("ParseBackupExportName() vmContextID = %v, want %v", gotVMContextID, tt.wantVMContextID)
				}
				if gotDiskID != tt.wantDiskID {
					t.Errorf("ParseBackupExportName() diskID = %v, want %v", gotDiskID, tt.wantDiskID)
				}
				if gotBackupType != tt.wantBackupType {
					t.Errorf("ParseBackupExportName() backupType = %v, want %v", gotBackupType, tt.wantBackupType)
				}
			}
		})
	}
}

// TestGetQCOW2FileSize tests QCOW2 file size detection
func TestGetQCOW2FileSize(t *testing.T) {
	// Create a temporary test QCOW2 file
	tmpDir := t.TempDir()
	qcow2Path := filepath.Join(tmpDir, "test-backup.qcow2")

	// Create a 1GB QCOW2 file for testing
	cmd := []string{
		"qemu-img", "create", "-f", "qcow2",
		qcow2Path, "1G",
	}

	if err := os.WriteFile(filepath.Join(tmpDir, "create-test.sh"), []byte("#!/bin/bash\n"+cmd[0]+" "+cmd[1]+" "+cmd[2]+" "+cmd[3]+" "+cmd[4]+" "+cmd[5]), 0755); err != nil {
		t.Fatalf("Failed to write test script: %v", err)
	}

	// Execute qemu-img create
	if output, err := execCommand("qemu-img", "create", "-f", "qcow2", qcow2Path, "1G"); err != nil {
		t.Fatalf("Failed to create test QCOW2 file: %v, output: %s", err, output)
	}

	// Test size detection
	size, err := GetQCOW2FileSize(qcow2Path)
	if err != nil {
		t.Fatalf("GetQCOW2FileSize() error = %v", err)
	}

	// 1GB = 1073741824 bytes
	expectedSize := int64(1073741824)
	if size != expectedSize {
		t.Errorf("GetQCOW2FileSize() size = %d, want %d", size, expectedSize)
	}

	t.Logf("✅ QCOW2 file size detected: %d bytes (1 GB)", size)
}

// TestValidateQCOW2File tests QCOW2 file validation
func TestValidateQCOW2File(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name      string
		setupFile func() string
		wantError bool
	}{
		{
			name: "valid QCOW2 file",
			setupFile: func() string {
				qcow2Path := filepath.Join(tmpDir, "valid-test.qcow2")
				if output, err := execCommand("qemu-img", "create", "-f", "qcow2", qcow2Path, "100M"); err != nil {
					t.Fatalf("Failed to create test file: %v, output: %s", err, output)
				}
				return qcow2Path
			},
			wantError: false,
		},
		{
			name: "non-existent file",
			setupFile: func() string {
				return filepath.Join(tmpDir, "non-existent.qcow2")
			},
			wantError: true,
		},
		{
			name: "invalid QCOW2 file (text file)",
			setupFile: func() string {
				invalidPath := filepath.Join(tmpDir, "invalid.qcow2")
				if err := os.WriteFile(invalidPath, []byte("not a qcow2 file"), 0644); err != nil {
					t.Fatalf("Failed to create invalid file: %v", err)
				}
				return invalidPath
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filePath := tt.setupFile()
			err := ValidateQCOW2File(filePath)

			if (err != nil) != tt.wantError {
				t.Errorf("ValidateQCOW2File() error = %v, wantError %v", err, tt.wantError)
			}

			if !tt.wantError {
				t.Logf("✅ QCOW2 file validation passed: %s", filePath)
			}
		})
	}
}

// Helper function to execute commands
func execCommand(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	output, err := cmd.CombinedOutput()
	return string(output), err
}
