package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// BackupChainMetadata represents the chain metadata stored in chain.json files.
type BackupChainMetadata struct {
	ChainID        string    `json:"chain_id"`
	VMContextID    string    `json:"vm_context_id"`
	DiskID         int       `json:"disk_id"`
	FullBackupID   string    `json:"full_backup_id"`
	BackupIDs      []string  `json:"backup_ids"` // Ordered list
	TotalBackups   int       `json:"total_backups"`
	TotalSizeBytes int64     `json:"total_size_bytes"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// BackupSidecarMetadata represents the backup metadata stored in sidecar JSON files.
type BackupSidecarMetadata struct {
	BackupID       string         `json:"backup_id"`
	VMContextID    string         `json:"vm_context_id"`
	VMName         string         `json:"vm_name"`
	DiskID         int            `json:"disk_id"`
	BackupType     BackupType     `json:"backup_type"`
	ParentBackupID string         `json:"parent_backup_id,omitempty"`
	ChangeID       string         `json:"change_id,omitempty"`
	SizeBytes      int64          `json:"size_bytes"`
	TotalBytes     int64          `json:"total_bytes"`
	CreatedAt      time.Time      `json:"created_at"`
	CompletedAt    *time.Time     `json:"completed_at,omitempty"`
	QCOW2Info      *QCOW2Info     `json:"qcow2_info,omitempty"`
	Metadata       BackupMetadata `json:"metadata"`
}

// QCOW2Info contains information about a QCOW2 file.
type QCOW2Info struct {
	VirtualSize  int64  `json:"virtual_size"`
	ActualSize   int64  `json:"actual_size"`
	BackingFile  string `json:"backing_file,omitempty"`
	Format       string `json:"format"` // qcow2
	Cluster      int64  `json:"cluster_size"`
	Compressed   bool   `json:"compressed"`
	Encrypted    bool   `json:"encrypted"`
	DirtyFlag    bool   `json:"dirty_flag"`
}

// SaveBackupMetadata writes backup metadata to a JSON sidecar file.
func SaveBackupMetadata(backup *Backup, metadata BackupMetadata, qcow2Info *QCOW2Info) error {
	sidecar := &BackupSidecarMetadata{
		BackupID:       backup.ID,
		VMContextID:    backup.VMContextID,
		VMName:         backup.VMName,
		DiskID:         backup.DiskID,
		BackupType:     backup.BackupType,
		ParentBackupID: backup.ParentBackupID,
		ChangeID:       backup.ChangeID,
		SizeBytes:      backup.SizeBytes,
		TotalBytes:     backup.TotalBytes,
		CreatedAt:      backup.CreatedAt,
		CompletedAt:    backup.CompletedAt,
		QCOW2Info:      qcow2Info,
		Metadata:       metadata,
	}

	sidecarPath := backup.FilePath + ".json"
	return saveJSONFile(sidecarPath, sidecar)
}

// LoadBackupMetadata reads backup metadata from a JSON sidecar file.
func LoadBackupMetadata(filePath string) (*BackupSidecarMetadata, error) {
	sidecarPath := filePath + ".json"
	
	data, err := os.ReadFile(sidecarPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read metadata: %w", err)
	}

	var sidecar BackupSidecarMetadata
	if err := json.Unmarshal(data, &sidecar); err != nil {
		return nil, fmt.Errorf("failed to parse metadata: %w", err)
	}

	return &sidecar, nil
}

// SaveChainMetadata writes chain metadata to chain.json file.
func SaveChainMetadata(chainPath string, chain *BackupChain) error {
	backupIDs := make([]string, len(chain.Backups))
	for i, backup := range chain.Backups {
		backupIDs[i] = backup.ID
	}

	metadata := &BackupChainMetadata{
		ChainID:        chain.ID,
		VMContextID:    chain.VMContextID,
		DiskID:         chain.DiskID,
		FullBackupID:   chain.FullBackupID,
		BackupIDs:      backupIDs,
		TotalBackups:   chain.TotalBackups,
		TotalSizeBytes: chain.TotalSizeBytes,
		CreatedAt:      chain.CreatedAt,
		UpdatedAt:      chain.UpdatedAt,
	}

	chainFilePath := filepath.Join(chainPath, "chain.json")
	return saveJSONFile(chainFilePath, metadata)
}

// LoadChainMetadata reads chain metadata from chain.json file.
func LoadChainMetadata(chainPath string) (*BackupChainMetadata, error) {
	chainFilePath := filepath.Join(chainPath, "chain.json")
	
	data, err := os.ReadFile(chainFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read chain metadata: %w", err)
	}

	var metadata BackupChainMetadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		return nil, fmt.Errorf("failed to parse chain metadata: %w", err)
	}

	return &metadata, nil
}

// saveJSONFile atomically writes JSON data to a file.
// Uses temporary file + rename for atomic write.
func saveJSONFile(path string, data interface{}) error {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Marshal JSON with indentation for readability
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	// Write to temporary file
	tempPath := path + ".tmp"
	if err := os.WriteFile(tempPath, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tempPath, path); err != nil {
		os.Remove(tempPath) // Clean up temp file
		return fmt.Errorf("failed to rename file: %w", err)
	}

	return nil
}

// GenerateBackupID generates a unique backup ID.
func GenerateBackupID(vmName string, diskID int) string {
	timestamp := time.Now().Format("20060102-150405")
	return fmt.Sprintf("backup-%s-disk%d-%s", vmName, diskID, timestamp)
}

// GenerateChainID generates a unique chain ID.
func GenerateChainID(vmContextID string, diskID int) string {
	return fmt.Sprintf("chain-%s-disk%d", vmContextID, diskID)
}

// GenerateCopyID generates a unique copy ID.
func GenerateCopyID() string {
	timestamp := time.Now().Format("20060102-150405-000")
	return fmt.Sprintf("copy-%s", timestamp)
}

// GetBackupPath returns the directory path for a VM's backups.
func GetBackupPath(basePath, vmContextID string, diskID int) string {
	return filepath.Join(basePath, vmContextID, fmt.Sprintf("disk-%d", diskID))
}

// GetBackupFilePath returns the full path for a backup file.
func GetBackupFilePath(basePath, vmContextID string, diskID int, backupID string) string {
	backupDir := GetBackupPath(basePath, vmContextID, diskID)
	return filepath.Join(backupDir, backupID+".qcow2")
}
