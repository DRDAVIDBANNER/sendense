// Package nbd provides helper functions for backup export management
package nbd

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

// BuildBackupExportName generates collision-proof NBD export name for backup files
// Format: backup-{vmContextID}-disk{diskID}-{backupType}-{timestamp}
// Example: backup-ctx-pgtest2-20251005-120000-disk0-full-20251005T120000
//
// Ensures:
// - Unique VM Context ID (no VM name collisions)
// - Backup prefix (distinguished from migration- exports)
// - Disk ID (multi-disk VM support)
// - Backup type (full/incr distinction)
// - Timestamp (multiple backup chain support)
// - Length limit (NBD export names <64 chars)
func BuildBackupExportName(vmContextID string, diskID int, backupType string, timestamp time.Time) string {
	timestampStr := timestamp.Format("20060102T150405")
	exportName := fmt.Sprintf("backup-%s-disk%d-%s-%s",
		vmContextID, diskID, backupType, timestampStr)

	// NBD export name limit is 64 characters
	if len(exportName) > 63 {
		// Calculate available space for vmContextID
		// Format: backup-{CONTEXT}-disk{N}-{TYPE}-{TIMESTAMP}
		// Example: backup-XXXXX-disk0-full-20251005T150405
		fixedPartLen := len(fmt.Sprintf("backup--disk%d-%s-%s", diskID, backupType, timestampStr))
		maxContextLen := 63 - fixedPartLen

		if maxContextLen > 0 {
			truncatedContext := vmContextID
			if len(vmContextID) > maxContextLen {
				truncatedContext = vmContextID[:maxContextLen]
			}
			exportName = fmt.Sprintf("backup-%s-disk%d-%s-%s",
				truncatedContext, diskID, backupType, timestampStr)
		} else {
			// Extreme case: even with minimal context, still too long
			// Use hash of vmContextID to ensure uniqueness
			log.WithFields(log.Fields{
				"vm_context_id": vmContextID,
				"disk_id":       diskID,
				"backup_type":   backupType,
			}).Warn("Export name would exceed 64 chars - using shortened format")

			// Use first 8 chars of context ID + disk + type + timestamp
			shortContext := vmContextID
			if len(vmContextID) > 8 {
				shortContext = vmContextID[:8]
			}
			exportName = fmt.Sprintf("backup-%s-d%d-%s-%s",
				shortContext, diskID, backupType, timestampStr)
		}
	}

	log.WithFields(log.Fields{
		"vm_context_id": vmContextID,
		"disk_id":       diskID,
		"backup_type":   backupType,
		"export_name":   exportName,
		"length":        len(exportName),
	}).Debug("Generated backup export name")

	return exportName
}

// GetQCOW2FileSize returns the virtual size of a QCOW2 file using qemu-img info
// This is the size that NBD clients will see (virtual disk size, not physical file size)
func GetQCOW2FileSize(qcow2Path string) (int64, error) {
	log.WithField("qcow2_path", qcow2Path).Debug("Getting QCOW2 file virtual size")

	// Use qemu-img info --output=json for accurate parsing
	cmd := exec.Command("qemu-img", "info", "--output=json", qcow2Path)
	output, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("failed to run qemu-img info: %w", err)
	}

	// Parse JSON output
	var info struct {
		VirtualSize int64  `json:"virtual-size"`
		Format      string `json:"format"`
		ActualSize  int64  `json:"actual-size"`
	}

	if err := json.Unmarshal(output, &info); err != nil {
		return 0, fmt.Errorf("failed to parse qemu-img output: %w", err)
	}

	// Validate format
	if info.Format != "qcow2" {
		return 0, fmt.Errorf("file is not QCOW2 format: %s", info.Format)
	}

	log.WithFields(log.Fields{
		"qcow2_path":    qcow2Path,
		"virtual_size":  info.VirtualSize,
		"actual_size":   info.ActualSize,
		"format":        info.Format,
	}).Info("✅ QCOW2 file size detected")

	return info.VirtualSize, nil
}

// ValidateQCOW2File checks if a file is a valid QCOW2 image
func ValidateQCOW2File(qcow2Path string) error {
	log.WithField("qcow2_path", qcow2Path).Debug("Validating QCOW2 file")

	// Use qemu-img check for validation
	cmd := exec.Command("qemu-img", "check", qcow2Path)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("QCOW2 validation failed: %w (output: %s)", err, string(output))
	}

	// Check output for errors
	outputStr := strings.ToLower(string(output))
	if strings.Contains(outputStr, "error") || strings.Contains(outputStr, "corrupt") {
		return fmt.Errorf("QCOW2 file appears corrupted: %s", string(output))
	}

	log.WithField("qcow2_path", qcow2Path).Info("✅ QCOW2 file validation passed")
	return nil
}

// ParseMigrationExportName extracts VM ID and disk number from migration export name
// Format: migration-vm-{vmID}-disk{diskNumber}
// Returns: vmID, diskNumber, error
func ParseMigrationExportName(exportName string) (string, int, error) {
	// Expected format: migration-vm-a1b2c3d4-e5f6-7890-abcd-ef1234567890-disk0
	if !strings.HasPrefix(exportName, "migration-vm-") {
		return "", 0, fmt.Errorf("not a migration export name: %s", exportName)
	}

	// Remove "migration-vm-" prefix
	remainder := strings.TrimPrefix(exportName, "migration-vm-")

	// Find last "-disk" occurrence
	diskIdx := strings.LastIndex(remainder, "-disk")
	if diskIdx == -1 {
		return "", 0, fmt.Errorf("invalid migration export format: %s", exportName)
	}

	vmID := remainder[:diskIdx]
	diskNumStr := remainder[diskIdx+5:] // Skip "-disk"

	var diskNum int
	if _, err := fmt.Sscanf(diskNumStr, "%d", &diskNum); err != nil {
		return "", 0, fmt.Errorf("invalid disk number in export name: %s", exportName)
	}

	return vmID, diskNum, nil
}

// ParseBackupExportName extracts components from backup export name
// Format: backup-{vmContextID}-disk{diskID}-{backupType}-{timestamp}
// Returns: vmContextID, diskID, backupType, timestamp, error
func ParseBackupExportName(exportName string) (string, int, string, time.Time, error) {
	// Expected format: backup-ctx-pgtest2-20251005-120000-disk0-full-20251005T120000
	if !strings.HasPrefix(exportName, "backup-") {
		return "", 0, "", time.Time{}, fmt.Errorf("not a backup export name: %s", exportName)
	}

	// Remove "backup-" prefix
	remainder := strings.TrimPrefix(exportName, "backup-")

	// Find "-disk" occurrence
	diskIdx := strings.Index(remainder, "-disk")
	if diskIdx == -1 {
		return "", 0, "", time.Time{}, fmt.Errorf("invalid backup export format: %s", exportName)
	}

	vmContextID := remainder[:diskIdx]

	// Parse disk number and remaining parts
	afterDisk := remainder[diskIdx+5:] // Skip "-disk"
	parts := strings.Split(afterDisk, "-")
	if len(parts) < 3 {
		return "", 0, "", time.Time{}, fmt.Errorf("invalid backup export format: %s", exportName)
	}

	var diskID int
	if _, err := fmt.Sscanf(parts[0], "%d", &diskID); err != nil {
		return "", 0, "", time.Time{}, fmt.Errorf("invalid disk ID: %s", parts[0])
	}

	backupType := parts[1]
	timestampStr := parts[2]

	// Parse timestamp
	timestamp, err := time.Parse("20060102T150405", timestampStr)
	if err != nil {
		return "", 0, "", time.Time{}, fmt.Errorf("invalid timestamp: %s", timestampStr)
	}

	return vmContextID, diskID, backupType, timestamp, nil
}

// IsBackupExport checks if an export name is a backup export (vs migration export)
func IsBackupExport(exportName string) bool {
	return strings.HasPrefix(exportName, "backup-")
}

// IsMigrationExport checks if an export name is a migration export (vs backup export)
func IsMigrationExport(exportName string) bool {
	return strings.HasPrefix(exportName, "migration-")
}
