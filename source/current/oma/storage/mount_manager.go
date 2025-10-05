package storage

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// MountManager handles network storage mounting operations (NFS, CIFS/SMB)
// SECURITY: All mount operations require root privileges or CAP_SYS_ADMIN
type MountManager struct {
	mu            sync.RWMutex
	mountedPaths  map[string]*MountInfo // mount point -> mount info
	mountTimeout  time.Duration         // timeout for mount operations
}

// MountInfo tracks information about a mounted filesystem
type MountInfo struct {
	MountPoint    string    // Local mount point path
	RepositoryID  string    // Repository ID that owns this mount
	Type          string    // "nfs" or "cifs"
	RemoteHost    string    // NFS server or CIFS server
	RemotePath    string    // NFS export path or CIFS share
	MountedAt     time.Time // When the mount was established
	MountOptions  []string  // Mount options used
}

// NFSMountConfig holds configuration for NFS mounting
type NFSMountConfig struct {
	Server       string   // NFS server hostname or IP
	ExportPath   string   // NFS export path (e.g., "/exports/backups")
	MountPoint   string   // Local mount point (e.g., "/mnt/nfs-repo-001")
	Options      []string // Mount options (e.g., ["vers=4", "rw", "hard"])
	RepositoryID string   // Repository ID for tracking
}

// CIFSMountConfig holds configuration for CIFS/SMB mounting
type CIFSMountConfig struct {
	Server       string   // CIFS server hostname or IP
	ShareName    string   // CIFS share name (e.g., "backups")
	MountPoint   string   // Local mount point (e.g., "/mnt/cifs-repo-001")
	Username     string   // CIFS username
	Password     string   // CIFS password (SECURITY: never logged)
	Domain       string   // Windows domain (optional)
	Options      []string // Additional mount options
	RepositoryID string   // Repository ID for tracking
}

// NewMountManager creates a new mount manager
func NewMountManager() *MountManager {
	return &MountManager{
		mountedPaths: make(map[string]*MountInfo),
		mountTimeout: 30 * time.Second,
	}
}

// MountNFS mounts an NFS share
// Returns error if mount fails or path already mounted
func (m *MountManager) MountNFS(ctx context.Context, config NFSMountConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Validate configuration
	if config.Server == "" {
		return fmt.Errorf("NFS server cannot be empty")
	}
	if config.ExportPath == "" {
		return fmt.Errorf("NFS export path cannot be empty")
	}
	if config.MountPoint == "" {
		return fmt.Errorf("mount point cannot be empty")
	}

	// Check if already mounted
	if _, exists := m.mountedPaths[config.MountPoint]; exists {
		return fmt.Errorf("mount point %s already in use", config.MountPoint)
	}

	// Check if path is already mounted in system
	isMounted, err := m.isPathMounted(config.MountPoint)
	if err != nil {
		return fmt.Errorf("failed to check mount status: %w", err)
	}
	if isMounted {
		return fmt.Errorf("mount point %s already mounted in system", config.MountPoint)
	}

	// Create mount point directory if it doesn't exist
	if err := os.MkdirAll(config.MountPoint, 0755); err != nil {
		return fmt.Errorf("failed to create mount point directory: %w", err)
	}

	// Build NFS remote path (server:/export/path)
	remotePath := fmt.Sprintf("%s:%s", config.Server, config.ExportPath)

	// Default NFS options if none provided
	options := config.Options
	if len(options) == 0 {
		options = []string{"vers=4", "rw", "hard", "intr", "timeo=600", "retrans=2"}
	}

	// Build mount command
	// mount -t nfs -o vers=4,rw,hard server:/export/path /mnt/mount-point
	args := []string{"-t", "nfs", "-o", strings.Join(options, ",")}
	args = append(args, remotePath, config.MountPoint)

	// Execute mount with timeout
	cmdCtx, cancel := context.WithTimeout(ctx, m.mountTimeout)
	defer cancel()

	cmd := exec.CommandContext(cmdCtx, "mount", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("NFS mount failed: %w (output: %s)", err, string(output))
	}

	// Record mount info
	m.mountedPaths[config.MountPoint] = &MountInfo{
		MountPoint:    config.MountPoint,
		RepositoryID:  config.RepositoryID,
		Type:          "nfs",
		RemoteHost:    config.Server,
		RemotePath:    config.ExportPath,
		MountedAt:     time.Now(),
		MountOptions:  options,
	}

	return nil
}

// MountCIFS mounts a CIFS/SMB share
// Returns error if mount fails or path already mounted
// SECURITY: Password is never logged
func (m *MountManager) MountCIFS(ctx context.Context, config CIFSMountConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Validate configuration
	if config.Server == "" {
		return fmt.Errorf("CIFS server cannot be empty")
	}
	if config.ShareName == "" {
		return fmt.Errorf("CIFS share name cannot be empty")
	}
	if config.MountPoint == "" {
		return fmt.Errorf("mount point cannot be empty")
	}
	if config.Username == "" {
		return fmt.Errorf("CIFS username cannot be empty")
	}

	// Check if already mounted
	if _, exists := m.mountedPaths[config.MountPoint]; exists {
		return fmt.Errorf("mount point %s already in use", config.MountPoint)
	}

	// Check if path is already mounted in system
	isMounted, err := m.isPathMounted(config.MountPoint)
	if err != nil {
		return fmt.Errorf("failed to check mount status: %w", err)
	}
	if isMounted {
		return fmt.Errorf("mount point %s already mounted in system", config.MountPoint)
	}

	// Create mount point directory if it doesn't exist
	if err := os.MkdirAll(config.MountPoint, 0755); err != nil {
		return fmt.Errorf("failed to create mount point directory: %w", err)
	}

	// Build CIFS remote path (//server/share)
	remotePath := fmt.Sprintf("//%s/%s", config.Server, config.ShareName)

	// Build mount options (credentials passed via options for security)
	options := []string{
		fmt.Sprintf("username=%s", config.Username),
	}
	
	// Add password to options (mount will read it securely)
	if config.Password != "" {
		options = append(options, fmt.Sprintf("password=%s", config.Password))
	}

	// Add domain if provided
	if config.Domain != "" {
		options = append(options, fmt.Sprintf("domain=%s", config.Domain))
	}

	// Add default CIFS options
	defaultOptions := []string{"vers=3.0", "rw", "file_mode=0644", "dir_mode=0755"}
	options = append(options, defaultOptions...)

	// Add user-provided options
	options = append(options, config.Options...)

	// Build mount command
	// mount -t cifs -o username=user,password=pass //server/share /mnt/mount-point
	args := []string{"-t", "cifs", "-o", strings.Join(options, ",")}
	args = append(args, remotePath, config.MountPoint)

	// Execute mount with timeout
	cmdCtx, cancel := context.WithTimeout(ctx, m.mountTimeout)
	defer cancel()

	cmd := exec.CommandContext(cmdCtx, "mount", args...)
	_, err = cmd.CombinedOutput()
	if err != nil {
		// SECURITY: Don't include command output as it may contain password
		return fmt.Errorf("CIFS mount failed: %w", err)
	}

	// Record mount info (without password)
	safeOptions := []string{"username=" + config.Username}
	if config.Domain != "" {
		safeOptions = append(safeOptions, "domain="+config.Domain)
	}
	safeOptions = append(safeOptions, defaultOptions...)

	m.mountedPaths[config.MountPoint] = &MountInfo{
		MountPoint:    config.MountPoint,
		RepositoryID:  config.RepositoryID,
		Type:          "cifs",
		RemoteHost:    config.Server,
		RemotePath:    config.ShareName,
		MountedAt:     time.Now(),
		MountOptions:  safeOptions, // Safe options without password
	}

	return nil
}

// Unmount unmounts a filesystem at the given mount point
// If force is true, performs lazy unmount (dangerous with active I/O)
func (m *MountManager) Unmount(ctx context.Context, mountPoint string, force bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if we have record of this mount
	mountInfo, exists := m.mountedPaths[mountPoint]
	if !exists {
		return fmt.Errorf("mount point %s not found in manager", mountPoint)
	}

	// Check if path is actually mounted in system
	isMounted, err := m.isPathMounted(mountPoint)
	if err != nil {
		return fmt.Errorf("failed to check mount status: %w", err)
	}
	if !isMounted {
		// Clean up our records even if not actually mounted
		delete(m.mountedPaths, mountPoint)
		return fmt.Errorf("mount point %s not mounted in system (cleaned up records)", mountPoint)
	}

	// Build unmount command
	var args []string
	if force {
		// Lazy unmount: detach now, cleanup when no longer busy
		args = []string{"-l", mountPoint}
	} else {
		// Regular unmount: fail if busy
		args = []string{mountPoint}
	}

	// Execute unmount with timeout
	cmdCtx, cancel := context.WithTimeout(ctx, m.mountTimeout)
	defer cancel()

	cmd := exec.CommandContext(cmdCtx, "umount", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("unmount failed for %s (type: %s): %w (output: %s)",
			mountPoint, mountInfo.Type, err, string(output))
	}

	// Remove from our records
	delete(m.mountedPaths, mountPoint)

	return nil
}

// IsMounted checks if a mount point is currently managed and mounted
func (m *MountManager) IsMounted(mountPoint string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	_, exists := m.mountedPaths[mountPoint]
	return exists
}

// GetMountInfo returns information about a mounted filesystem
func (m *MountManager) GetMountInfo(mountPoint string) (*MountInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	info, exists := m.mountedPaths[mountPoint]
	if !exists {
		return nil, fmt.Errorf("mount point %s not found", mountPoint)
	}

	// Return a copy to prevent external modification
	infoCopy := *info
	return &infoCopy, nil
}

// GetMountsByRepository returns all mount points for a repository
func (m *MountManager) GetMountsByRepository(repositoryID string) []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var mountPoints []string
	for mountPoint, info := range m.mountedPaths {
		if info.RepositoryID == repositoryID {
			mountPoints = append(mountPoints, mountPoint)
		}
	}

	return mountPoints
}

// ListMounted returns all currently managed mounts
func (m *MountManager) ListMounted() map[string]*MountInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Return copies to prevent external modification
	mounts := make(map[string]*MountInfo, len(m.mountedPaths))
	for path, info := range m.mountedPaths {
		infoCopy := *info
		mounts[path] = &infoCopy
	}

	return mounts
}

// isPathMounted checks if a path is mounted in the system (reads /proc/mounts)
// This is a low-level check that doesn't rely on our internal state
func (m *MountManager) isPathMounted(path string) (bool, error) {
	file, err := os.Open("/proc/mounts")
	if err != nil {
		return false, fmt.Errorf("failed to open /proc/mounts: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) >= 2 {
			mountPoint := fields[1]
			if mountPoint == path {
				return true, nil
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return false, fmt.Errorf("error reading /proc/mounts: %w", err)
	}

	return false, nil
}

// UnmountAll unmounts all managed mounts (useful for shutdown)
// Returns map of mount points to errors (empty if all succeed)
func (m *MountManager) UnmountAll(ctx context.Context, force bool) map[string]error {
	m.mu.Lock()
	defer m.mu.Unlock()

	errors := make(map[string]error)

	// Collect mount points first (to avoid modifying map during iteration)
	mountPoints := make([]string, 0, len(m.mountedPaths))
	for mountPoint := range m.mountedPaths {
		mountPoints = append(mountPoints, mountPoint)
	}

	// Unmount each (temporarily unlock for each operation)
	for _, mountPoint := range mountPoints {
		m.mu.Unlock()
		err := m.Unmount(ctx, mountPoint, force)
		m.mu.Lock()

		if err != nil {
			errors[mountPoint] = err
		}
	}

	return errors
}
