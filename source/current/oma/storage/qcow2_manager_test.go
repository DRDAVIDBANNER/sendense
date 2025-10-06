package storage

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestNewQCOW2Manager(t *testing.T) {
	qm := NewQCOW2Manager()
	if qm == nil {
		t.Fatal("NewQCOW2Manager returned nil")
	}
}

func TestCreateFullBackup(t *testing.T) {
	// Create temporary directory for test
	tmpDir, err := os.MkdirTemp("", "sendense-qcow2-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	qm := NewQCOW2Manager()
	ctx := context.Background()

	outputPath := filepath.Join(tmpDir, "test-backup.qcow2")
	sizeBytes := int64(10737418240) // 10 GB

	// Note: This test requires qemu-img to be installed
	// In a real environment, we would skip this test if qemu-img is not available
	if !isQemuImgAvailable() {
		t.Skip("qemu-img not available, skipping test")
	}

	err = qm.CreateFullBackup(ctx, outputPath, sizeBytes, true)
	if err != nil {
		t.Fatalf("CreateFullBackup failed: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Error("backup file was not created")
	}
}

func TestCreateIncrementalBackup(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "sendense-qcow2-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	qm := NewQCOW2Manager()
	ctx := context.Background()

	if !isQemuImgAvailable() {
		t.Skip("qemu-img not available, skipping test")
	}

	// Create base backup first
	basePath := filepath.Join(tmpDir, "base.qcow2")
	err = qm.CreateFullBackup(ctx, basePath, 10737418240, true)
	if err != nil {
		t.Fatalf("CreateFullBackup failed: %v", err)
	}

	// Create incremental backup
	incrPath := filepath.Join(tmpDir, "incremental.qcow2")
	err = qm.CreateIncrementalBackup(ctx, incrPath, basePath, true)
	if err != nil {
		t.Fatalf("CreateIncrementalBackup failed: %v", err)
	}

	// Verify incremental file was created
	if _, err := os.Stat(incrPath); os.IsNotExist(err) {
		t.Error("incremental backup file was not created")
	}

	// Verify it has a backing file
	info, err := qm.GetInfo(ctx, incrPath)
	if err != nil {
		t.Fatalf("GetInfo failed: %v", err)
	}

	if info.BackingFile == "" {
		t.Error("incremental backup should have backing file")
	}
}

func TestGetInfo(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "sendense-qcow2-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	qm := NewQCOW2Manager()
	ctx := context.Background()

	if !isQemuImgAvailable() {
		t.Skip("qemu-img not available, skipping test")
	}

	// Create a test backup
	backupPath := filepath.Join(tmpDir, "test.qcow2")
	sizeBytes := int64(10737418240)
	err = qm.CreateFullBackup(ctx, backupPath, sizeBytes, true)
	if err != nil {
		t.Fatalf("CreateFullBackup failed: %v", err)
	}

	// Get info
	info, err := qm.GetInfo(ctx, backupPath)
	if err != nil {
		t.Fatalf("GetInfo failed: %v", err)
	}

	if info.Format != "qcow2" {
		t.Errorf("format = %v, want qcow2", info.Format)
	}
	if info.VirtualSize <= 0 {
		t.Error("virtual size should be positive")
	}
}

func TestGetInfoNonExistent(t *testing.T) {
	qm := NewQCOW2Manager()
	ctx := context.Background()

	if !isQemuImgAvailable() {
		t.Skip("qemu-img not available, skipping test")
	}

	_, err := qm.GetInfo(ctx, "/nonexistent/path/backup.qcow2")
	if err == nil {
		t.Error("GetInfo should fail for non-existent file")
	}
}

func TestVerifyBackup(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "sendense-qcow2-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	qm := NewQCOW2Manager()
	ctx := context.Background()

	if !isQemuImgAvailable() {
		t.Skip("qemu-img not available, skipping test")
	}

	// Create a test backup
	backupPath := filepath.Join(tmpDir, "test.qcow2")
	err = qm.CreateFullBackup(ctx, backupPath, 10737418240, true)
	if err != nil {
		t.Fatalf("CreateFullBackup failed: %v", err)
	}

	// Verify it
	err = qm.VerifyBackup(ctx, backupPath)
	if err != nil {
		t.Fatalf("VerifyBackup failed: %v", err)
	}
}

func TestVerifyCorruptedBackup(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "sendense-qcow2-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	qm := NewQCOW2Manager()
	ctx := context.Background()

	if !isQemuImgAvailable() {
		t.Skip("qemu-img not available, skipping test")
	}

	// Create a corrupted file (not a valid QCOW2)
	corruptPath := filepath.Join(tmpDir, "corrupt.qcow2")
	err = os.WriteFile(corruptPath, []byte("not a qcow2 file"), 0644)
	if err != nil {
		t.Fatalf("failed to create corrupt file: %v", err)
	}

	// Verify should fail
	err = qm.VerifyBackup(ctx, corruptPath)
	if err == nil {
		t.Error("VerifyBackup should fail for corrupted file")
	}
}

func TestCreateBackupWithInvalidPath(t *testing.T) {
	qm := NewQCOW2Manager()
	ctx := context.Background()

	if !isQemuImgAvailable() {
		t.Skip("qemu-img not available, skipping test")
	}

	// Try to create backup in non-existent directory
	invalidPath := "/nonexistent/directory/backup.qcow2"
	err := qm.CreateFullBackup(ctx, invalidPath, 10737418240, true)
	if err == nil {
		t.Error("CreateFullBackup should fail for invalid path")
	}
}

func TestCreateIncrementalWithMissingBase(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "sendense-qcow2-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	qm := NewQCOW2Manager()
	ctx := context.Background()

	if !isQemuImgAvailable() {
		t.Skip("qemu-img not available, skipping test")
	}

	incrPath := filepath.Join(tmpDir, "incremental.qcow2")
	missingBase := filepath.Join(tmpDir, "nonexistent-base.qcow2")

	err = qm.CreateIncrementalBackup(ctx, incrPath, missingBase, true)
	if err == nil {
		t.Error("CreateIncrementalBackup should fail when base file is missing")
	}
}

func TestQCOW2EdgeCases(t *testing.T) {
	qm := NewQCOW2Manager()
	ctx := context.Background()

	if !isQemuImgAvailable() {
		t.Skip("qemu-img not available, skipping test")
	}

	t.Run("zero size backup", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "sendense-qcow2-test-*")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		backupPath := filepath.Join(tmpDir, "zero.qcow2")
		err = qm.CreateFullBackup(ctx, backupPath, 0, true)
		if err == nil {
			t.Error("CreateFullBackup should fail for zero size")
		}
	})

	t.Run("negative size backup", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "sendense-qcow2-test-*")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		backupPath := filepath.Join(tmpDir, "negative.qcow2")
		err = qm.CreateFullBackup(ctx, backupPath, -1000, true)
		if err == nil {
			t.Error("CreateFullBackup should fail for negative size")
		}
	})
}

// Helper function to check if qemu-img is available
func isQemuImgAvailable() bool {
	_, err := os.Stat("/usr/bin/qemu-img")
	if err == nil {
		return true
	}
	// Try alternative path
	_, err = os.Stat("/usr/local/bin/qemu-img")
	return err == nil
}

func TestQCOW2InfoParsing(t *testing.T) {
	// Test that we can parse qemu-img info output correctly
	tests := []struct {
		name     string
		output   string
		expected *QCOW2Info
	}{
		{
			name: "basic info",
			output: `{
				"format": "qcow2",
				"virtual-size": 10737418240,
				"actual-size": 196608,
				"cluster-size": 65536
			}`,
			expected: &QCOW2Info{
				Format:      "qcow2",
				VirtualSize: 10737418240,
				ActualSize:  196608,
			},
		},
		{
			name: "with backing file",
			output: `{
				"format": "qcow2",
				"virtual-size": 10737418240,
				"actual-size": 196608,
				"backing-filename": "/backups/base.qcow2"
			}`,
			expected: &QCOW2Info{
				Format:      "qcow2",
				VirtualSize: 10737418240,
				ActualSize:  196608,
				BackingFile: "/backups/base.qcow2",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This would test the JSON parsing logic
			// In actual implementation, we would parse the qemu-img output
			t.Log("JSON parsing test placeholder")
		})
	}
}

