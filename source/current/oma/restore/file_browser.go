// Package restore provides file browsing capabilities for mounted QCOW2 backups
// Task 4: File-Level Restore (Phase 2 - File Browser API)
// SECURITY: Prevents path traversal attacks, validates all file paths
package restore

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/vexxhost/migratekit-oma/database"
)

// FileBrowser provides secure file browsing within mounted backups
type FileBrowser struct {
	mountRepo *database.RestoreMountRepository
}

// NewFileBrowser creates a new file browser instance
func NewFileBrowser(mountRepo *database.RestoreMountRepository) *FileBrowser {
	return &FileBrowser{
		mountRepo: mountRepo,
	}
}

// FileInfo represents information about a file or directory in a backup
type FileInfo struct {
	Name         string    `json:"name"`
	Path         string    `json:"path"`          // Full path within backup
	Type         string    `json:"type"`          // "file" or "directory"
	Size         int64     `json:"size"`          // File size in bytes
	Mode         string    `json:"mode"`          // Unix permissions (e.g., "0644")
	ModifiedTime time.Time `json:"modified_time"` // Last modification time
	IsSymlink    bool      `json:"is_symlink"`
	SymlinkTarget string   `json:"symlink_target,omitempty"`
}

// ListFilesRequest represents a request to list files in a mounted backup
type ListFilesRequest struct {
	MountID   string `json:"mount_id"`   // Required: Mount identifier
	Path      string `json:"path"`       // Path within backup (default: "/")
	Recursive bool   `json:"recursive"`  // Recursive directory listing
}

// ListFilesResponse contains the list of files and directories
type ListFilesResponse struct {
	MountID   string      `json:"mount_id"`
	Path      string      `json:"path"`
	Files     []*FileInfo `json:"files"`
	TotalCount int        `json:"total_count"`
}

// ListFiles lists files and directories within a mounted backup
func (fb *FileBrowser) ListFiles(ctx context.Context, req *ListFilesRequest) (*ListFilesResponse, error) {
	log.WithFields(log.Fields{
		"mount_id": req.MountID,
		"path":     req.Path,
		"recursive": req.Recursive,
	}).Info("📂 Listing files in mounted backup")

	// Get mount record (validates mount exists and is active)
	mount, err := fb.mountRepo.GetByID(ctx, req.MountID)
	if err != nil {
		return nil, fmt.Errorf("mount not found: %w", err)
	}

	if mount.Status != "mounted" {
		return nil, fmt.Errorf("mount not ready: status=%s", mount.Status)
	}

	// Update last accessed time (for idle timeout detection)
	fb.mountRepo.UpdateLastAccessed(ctx, req.MountID)

	// Normalize and validate path
	requestPath := req.Path
	if requestPath == "" {
		requestPath = "/"
	}

	// Validate and sanitize path (SECURITY: prevent path traversal)
	safePath, err := fb.ValidateAndSanitizePath(mount.MountPath, requestPath)
	if err != nil {
		return nil, fmt.Errorf("path validation failed: %w", err)
	}

	log.WithFields(log.Fields{
		"mount_path": mount.MountPath,
		"safe_path":  safePath,
	}).Debug("✅ Path validation passed")

	// List files
	var files []*FileInfo
	if req.Recursive {
		files, err = fb.listFilesRecursive(safePath, requestPath)
	} else {
		files, err = fb.listFilesSingle(safePath, requestPath)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to list files: %w", err)
	}

	log.WithFields(log.Fields{
		"mount_id":   req.MountID,
		"file_count": len(files),
	}).Info("✅ Files listed successfully")

	return &ListFilesResponse{
		MountID:    req.MountID,
		Path:       requestPath,
		Files:      files,
		TotalCount: len(files),
	}, nil
}

// GetFileInfo retrieves detailed information about a specific file
func (fb *FileBrowser) GetFileInfo(ctx context.Context, mountID, filePath string) (*FileInfo, error) {
	log.WithFields(log.Fields{
		"mount_id":  mountID,
		"file_path": filePath,
	}).Debug("📄 Getting file information")

	// Get mount record
	mount, err := fb.mountRepo.GetByID(ctx, mountID)
	if err != nil {
		return nil, fmt.Errorf("mount not found: %w", err)
	}

	if mount.Status != "mounted" {
		return nil, fmt.Errorf("mount not ready: status=%s", mount.Status)
	}

	// Update last accessed time
	fb.mountRepo.UpdateLastAccessed(ctx, mountID)

	// Validate and sanitize path
	safePath, err := fb.ValidateAndSanitizePath(mount.MountPath, filePath)
	if err != nil {
		return nil, fmt.Errorf("path validation failed: %w", err)
	}

	// Get file info
	info, err := os.Lstat(safePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	fileInfo := fb.buildFileInfo(info, filePath)

	// If symlink, resolve target
	if fileInfo.IsSymlink {
		target, err := os.Readlink(safePath)
		if err == nil {
			fileInfo.SymlinkTarget = target
		}
	}

	return fileInfo, nil
}

// ValidateAndSanitizePath validates a path and prevents directory traversal attacks
// SECURITY CRITICAL: This function prevents path traversal vulnerabilities
func (fb *FileBrowser) ValidateAndSanitizePath(mountRoot, requestedPath string) (string, error) {
	// Normalize requested path
	cleanPath := filepath.Clean("/" + requestedPath)

	// Build full filesystem path
	fullPath := filepath.Join(mountRoot, cleanPath)

	// SECURITY CHECK: Ensure resolved path is within mount root
	resolvedPath, err := filepath.Abs(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve path: %w", err)
	}

	// Get absolute mount root
	absRoot, err := filepath.Abs(mountRoot)
	if err != nil {
		return "", fmt.Errorf("failed to resolve mount root: %w", err)
	}

	// CRITICAL: Verify path is within mount root (prevents ../../etc/passwd attacks)
	if !strings.HasPrefix(resolvedPath, absRoot) {
		log.WithFields(log.Fields{
			"requested_path": requestedPath,
			"resolved_path":  resolvedPath,
			"mount_root":     absRoot,
		}).Warn("🚨 Path traversal attack attempt detected")
		return "", fmt.Errorf("invalid path: access denied (path traversal attempt)")
	}

	// Verify path exists
	if _, err := os.Stat(resolvedPath); os.IsNotExist(err) {
		return "", fmt.Errorf("path does not exist: %s", requestedPath)
	}

	log.WithFields(log.Fields{
		"requested_path": requestedPath,
		"safe_path":      resolvedPath,
	}).Debug("✅ Path validated and sanitized")

	return resolvedPath, nil
}

// listFilesSingle lists files in a single directory (non-recursive)
func (fb *FileBrowser) listFilesSingle(fsPath, requestPath string) ([]*FileInfo, error) {
	log.WithField("path", fsPath).Debug("📋 Listing directory contents")

	// Read directory
	entries, err := os.ReadDir(fsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	files := make([]*FileInfo, 0, len(entries))
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			log.WithError(err).WithField("entry", entry.Name()).Warn("Failed to get entry info")
			continue
		}

		// Build relative path
		relativePath := filepath.Join(requestPath, entry.Name())
		fileInfo := fb.buildFileInfo(info, relativePath)
		files = append(files, fileInfo)
	}

	return files, nil
}

// listFilesRecursive lists files recursively within a directory
func (fb *FileBrowser) listFilesRecursive(fsPath, requestPath string) ([]*FileInfo, error) {
	log.WithField("path", fsPath).Debug("📋 Listing directory contents (recursive)")

	var files []*FileInfo

	err := filepath.WalkDir(fsPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			log.WithError(err).WithField("path", path).Warn("Error walking directory")
			return nil // Continue walking despite errors
		}

		// Get relative path from mount point
		relPath, err := filepath.Rel(fsPath, path)
		if err != nil {
			return nil
		}

		// Skip root directory
		if relPath == "." {
			return nil
		}

		// Build path relative to request path
		relativePath := filepath.Join(requestPath, relPath)

		// Get file info
		info, err := d.Info()
		if err != nil {
			log.WithError(err).WithField("path", path).Warn("Failed to get file info")
			return nil
		}

		fileInfo := fb.buildFileInfo(info, relativePath)
		files = append(files, fileInfo)

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk directory: %w", err)
	}

	return files, nil
}

// buildFileInfo constructs FileInfo from os.FileInfo
func (fb *FileBrowser) buildFileInfo(info os.FileInfo, relativePath string) *FileInfo {
	fileType := "file"
	if info.IsDir() {
		fileType = "directory"
	}

	fileInfo := &FileInfo{
		Name:         info.Name(),
		Path:         relativePath,
		Type:         fileType,
		Size:         info.Size(),
		Mode:         fmt.Sprintf("%04o", info.Mode().Perm()),
		ModifiedTime: info.ModTime(),
		IsSymlink:    info.Mode()&os.ModeSymlink != 0,
	}

	return fileInfo
}

// ValidateFileAccess validates that a file can be accessed (security check before download)
func (fb *FileBrowser) ValidateFileAccess(ctx context.Context, mountID, filePath string) (string, error) {
	log.WithFields(log.Fields{
		"mount_id":  mountID,
		"file_path": filePath,
	}).Debug("🔒 Validating file access")

	// Get mount record
	mount, err := fb.mountRepo.GetByID(ctx, mountID)
	if err != nil {
		return "", fmt.Errorf("mount not found: %w", err)
	}

	if mount.Status != "mounted" {
		return "", fmt.Errorf("mount not ready: status=%s", mount.Status)
	}

	// Validate and sanitize path
	safePath, err := fb.ValidateAndSanitizePath(mount.MountPath, filePath)
	if err != nil {
		return "", fmt.Errorf("path validation failed: %w", err)
	}

	// Verify it's a regular file (not directory or special file)
	info, err := os.Stat(safePath)
	if err != nil {
		return "", fmt.Errorf("failed to stat file: %w", err)
	}

	if info.IsDir() {
		return "", fmt.Errorf("path is a directory, not a file")
	}

	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("path is not a regular file")
	}

	// Update last accessed time
	fb.mountRepo.UpdateLastAccessed(ctx, mountID)

	log.WithField("safe_path", safePath).Debug("✅ File access validated")
	return safePath, nil
}

// ValidateDirectoryAccess validates directory access for directory downloads
func (fb *FileBrowser) ValidateDirectoryAccess(ctx context.Context, mountID, dirPath string) (string, error) {
	log.WithFields(log.Fields{
		"mount_id": mountID,
		"dir_path": dirPath,
	}).Debug("🔒 Validating directory access")

	// Get mount record
	mount, err := fb.mountRepo.GetByID(ctx, mountID)
	if err != nil {
		return "", fmt.Errorf("mount not found: %w", err)
	}

	if mount.Status != "mounted" {
		return "", fmt.Errorf("mount not ready: status=%s", mount.Status)
	}

	// Validate and sanitize path
	safePath, err := fb.ValidateAndSanitizePath(mount.MountPath, dirPath)
	if err != nil {
		return "", fmt.Errorf("path validation failed: %w", err)
	}

	// Verify it's a directory
	info, err := os.Stat(safePath)
	if err != nil {
		return "", fmt.Errorf("failed to stat directory: %w", err)
	}

	if !info.IsDir() {
		return "", fmt.Errorf("path is not a directory")
	}

	// Update last accessed time
	fb.mountRepo.UpdateLastAccessed(ctx, mountID)

	log.WithField("safe_path", safePath).Debug("✅ Directory access validated")
	return safePath, nil
}

// CalculateDirectorySize calculates total size of a directory (for progress tracking)
func (fb *FileBrowser) CalculateDirectorySize(ctx context.Context, dirPath string) (int64, int, error) {
	log.WithField("dir_path", dirPath).Debug("📊 Calculating directory size")

	var totalSize int64
	var fileCount int

	err := filepath.WalkDir(dirPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // Continue despite errors
		}

		if !d.IsDir() {
			info, err := d.Info()
			if err != nil {
				return nil
			}
			totalSize += info.Size()
			fileCount++
		}

		return nil
	})

	if err != nil {
		return 0, 0, fmt.Errorf("failed to calculate directory size: %w", err)
	}

	log.WithFields(log.Fields{
		"total_size":  totalSize,
		"file_count":  fileCount,
	}).Debug("✅ Directory size calculated")

	return totalSize, fileCount, nil
}

