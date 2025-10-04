package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// QCOW2Manager handles QCOW2 file operations using qemu-img.
type QCOW2Manager struct {
	qemuImgPath string
}

// NewQCOW2Manager creates a new QCOW2Manager.
func NewQCOW2Manager() (*QCOW2Manager, error) {
	// Find qemu-img binary
	qemuImgPath, err := exec.LookPath("qemu-img")
	if err != nil {
		return nil, fmt.Errorf("qemu-img not found: %w (install qemu-utils package)", err)
	}

	return &QCOW2Manager{
		qemuImgPath: qemuImgPath,
	}, nil
}

// CreateFull creates a new QCOW2 file for a full backup.
func (q *QCOW2Manager) CreateFull(ctx context.Context, path string, sizeBytes int64) error {
	// Ensure parent directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return &BackupError{
			Op:  "create_directory",
			Err: fmt.Errorf("failed to create directory: %w", err),
		}
	}

	// Convert size to human-readable format for qemu-img
	sizeStr := fmt.Sprintf("%d", sizeBytes)

	// Create QCOW2 file
	// qemu-img create -f qcow2 <path> <size>
	cmd := exec.CommandContext(ctx, q.qemuImgPath, "create", "-f", "qcow2", path, sizeStr)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return &BackupError{
			Op:  "create_full",
			Err: fmt.Errorf("qemu-img create failed: %s: %w", string(output), err),
		}
	}

	// Set proper permissions
	if err := os.Chmod(path, 0644); err != nil {
		return &BackupError{
			Op:  "set_permissions",
			Err: fmt.Errorf("failed to set permissions: %w", err),
		}
	}

	return nil
}

// CreateIncremental creates a new QCOW2 file with a backing file for incremental backup.
func (q *QCOW2Manager) CreateIncremental(ctx context.Context, path string, backingFile string) error {
	// Verify backing file exists
	if _, err := os.Stat(backingFile); err != nil {
		return &BackupError{
			Op:  "verify_backing_file",
			Err: fmt.Errorf("backing file not found: %w", err),
		}
	}

	// Ensure parent directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return &BackupError{
			Op:  "create_directory",
			Err: fmt.Errorf("failed to create directory: %w", err),
		}
	}

	// Create QCOW2 file with backing file
	// qemu-img create -f qcow2 -b <backing> -F qcow2 <path>
	cmd := exec.CommandContext(ctx, q.qemuImgPath, "create",
		"-f", "qcow2",
		"-b", backingFile,
		"-F", "qcow2",
		path)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return &BackupError{
			Op:  "create_incremental",
			Err: fmt.Errorf("qemu-img create failed: %s: %w", string(output), err),
		}
	}

	// Set proper permissions
	if err := os.Chmod(path, 0644); err != nil {
		return &BackupError{
			Op:  "set_permissions",
			Err: fmt.Errorf("failed to set permissions: %w", err),
		}
	}

	return nil
}

// GetInfo retrieves information about a QCOW2 file.
func (q *QCOW2Manager) GetInfo(ctx context.Context, path string) (*QCOW2Info, error) {
	// qemu-img info --output=json <path>
	cmd := exec.CommandContext(ctx, q.qemuImgPath, "info", "--output=json", path)
	output, err := cmd.Output()
	if err != nil {
		return nil, &BackupError{
			Op:  "get_info",
			Err: fmt.Errorf("qemu-img info failed: %w", err),
		}
	}

	// Parse JSON output
	var rawInfo struct {
		VirtualSize   int64  `json:"virtual-size"`
		ActualSize    int64  `json:"actual-size"`
		BackingFile   string `json:"backing-filename,omitempty"`
		Format        string `json:"format"`
		ClusterSize   int64  `json:"cluster-size"`
		Compressed    bool   `json:"compressed"`
		Encrypted     bool   `json:"encrypted"`
		DirtyFlag     bool   `json:"dirty-flag"`
	}

	if err := json.Unmarshal(output, &rawInfo); err != nil {
		return nil, &BackupError{
			Op:  "parse_info",
			Err: fmt.Errorf("failed to parse qemu-img output: %w", err),
		}
	}

	return &QCOW2Info{
		VirtualSize: rawInfo.VirtualSize,
		ActualSize:  rawInfo.ActualSize,
		BackingFile: rawInfo.BackingFile,
		Format:      rawInfo.Format,
		Cluster:     rawInfo.ClusterSize,
		Compressed:  rawInfo.Compressed,
		Encrypted:   rawInfo.Encrypted,
		DirtyFlag:   rawInfo.DirtyFlag,
	}, nil
}

// Verify checks a QCOW2 file for corruption.
func (q *QCOW2Manager) Verify(ctx context.Context, path string) error {
	// qemu-img check <path>
	cmd := exec.CommandContext(ctx, q.qemuImgPath, "check", path)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return &BackupError{
			Op:  "verify",
			Err: fmt.Errorf("qemu-img check failed: %s: %w", string(output), err),
		}
	}

	// Parse output for errors or leaks
	outputStr := string(output)
	if strings.Contains(outputStr, "ERROR") {
		return &BackupError{
			Op:  "verify",
			Err: fmt.Errorf("QCOW2 file has errors: %s", outputStr),
		}
	}

	return nil
}

// GetActualSize returns the actual disk space used by a QCOW2 file.
func (q *QCOW2Manager) GetActualSize(ctx context.Context, path string) (int64, error) {
	info, err := q.GetInfo(ctx, path)
	if err != nil {
		return 0, err
	}
	return info.ActualSize, nil
}

// GetVirtualSize returns the virtual size of a QCOW2 file.
func (q *QCOW2Manager) GetVirtualSize(ctx context.Context, path string) (int64, error) {
	info, err := q.GetInfo(ctx, path)
	if err != nil {
		return 0, err
	}
	return info.VirtualSize, nil
}

// Rebase changes the backing file of a QCOW2 image (for chain consolidation).
func (q *QCOW2Manager) Rebase(ctx context.Context, path string, newBackingFile string) error {
	// qemu-img rebase -u -b <new_backing> <path>
	cmd := exec.CommandContext(ctx, q.qemuImgPath, "rebase",
		"-u",  // Unsafe mode (just change backing file reference)
		"-b", newBackingFile,
		path)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return &BackupError{
			Op:  "rebase",
			Err: fmt.Errorf("qemu-img rebase failed: %s: %w", string(output), err),
		}
	}

	return nil
}

// Commit merges an incremental into its backing file (for chain consolidation).
func (q *QCOW2Manager) Commit(ctx context.Context, path string) error {
	// qemu-img commit <path>
	cmd := exec.CommandContext(ctx, q.qemuImgPath, "commit", path)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return &BackupError{
			Op:  "commit",
			Err: fmt.Errorf("qemu-img commit failed: %s: %w", string(output), err),
		}
	}

	return nil
}

// Convert converts a QCOW2 file to another format (for restore operations).
func (q *QCOW2Manager) Convert(ctx context.Context, sourcePath, destPath, format string) error {
	// qemu-img convert -f qcow2 -O <format> <source> <dest>
	cmd := exec.CommandContext(ctx, q.qemuImgPath, "convert",
		"-f", "qcow2",
		"-O", format,
		sourcePath,
		destPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return &BackupError{
			Op:  "convert",
			Err: fmt.Errorf("qemu-img convert failed: %s: %w", string(output), err),
		}
	}

	return nil
}

// Resize changes the virtual size of a QCOW2 file.
func (q *QCOW2Manager) Resize(ctx context.Context, path string, newSizeBytes int64) error {
	newSizeStr := fmt.Sprintf("%d", newSizeBytes)
	
	// qemu-img resize <path> <new_size>
	cmd := exec.CommandContext(ctx, q.qemuImgPath, "resize", path, newSizeStr)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return &BackupError{
			Op:  "resize",
			Err: fmt.Errorf("qemu-img resize failed: %s: %w", string(output), err),
		}
	}

	return nil
}

// Snapshot creates a VM-style snapshot in a QCOW2 file (internal snapshots).
func (q *QCOW2Manager) Snapshot(ctx context.Context, path, snapshotName string) error {
	// qemu-img snapshot -c <snapshot_name> <path>
	cmd := exec.CommandContext(ctx, q.qemuImgPath, "snapshot", "-c", snapshotName, path)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return &BackupError{
			Op:  "snapshot",
			Err: fmt.Errorf("qemu-img snapshot failed: %s: %w", string(output), err),
		}
	}

	return nil
}

// ListSnapshots lists all snapshots in a QCOW2 file.
func (q *QCOW2Manager) ListSnapshots(ctx context.Context, path string) ([]string, error) {
	// qemu-img snapshot -l <path>
	cmd := exec.CommandContext(ctx, q.qemuImgPath, "snapshot", "-l", path)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, &BackupError{
			Op:  "list_snapshots",
			Err: fmt.Errorf("qemu-img snapshot failed: %s: %w", string(output), err),
		}
	}

	// Parse snapshot list (skip header lines)
	lines := strings.Split(string(output), "\n")
	snapshots := []string{}
	for i, line := range lines {
		if i < 2 || strings.TrimSpace(line) == "" {
			continue // Skip header and empty lines
		}
		// Extract snapshot name (second column)
		fields := strings.Fields(line)
		if len(fields) >= 2 {
			snapshots = append(snapshots, fields[1])
		}
	}

	return snapshots, nil
}

// parseSize parses a human-readable size string (e.g., "10G", "500M") to bytes.
func parseSize(sizeStr string) (int64, error) {
	sizeStr = strings.TrimSpace(sizeStr)
	if sizeStr == "" {
		return 0, fmt.Errorf("empty size string")
	}

	// Handle plain numbers (bytes)
	if size, err := strconv.ParseInt(sizeStr, 10, 64); err == nil {
		return size, nil
	}

	// Handle suffixes (K, M, G, T)
	suffix := sizeStr[len(sizeStr)-1:]
	valueStr := sizeStr[:len(sizeStr)-1]
	value, err := strconv.ParseFloat(valueStr, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid size format: %s", sizeStr)
	}

	multipliers := map[string]int64{
		"K": 1024,
		"M": 1024 * 1024,
		"G": 1024 * 1024 * 1024,
		"T": 1024 * 1024 * 1024 * 1024,
	}

	multiplier, ok := multipliers[suffix]
	if !ok {
		return 0, fmt.Errorf("unknown size suffix: %s", suffix)
	}

	return int64(value * float64(multiplier)), nil
}
