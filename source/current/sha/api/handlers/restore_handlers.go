// Package handlers provides REST API handlers for file-level restore operations
// Task 4: File-Level Restore (Phase 5 - API Integration)
// Complete REST API for mounting backups, browsing files, and downloading
package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"

	"github.com/vexxhost/migratekit-sha/database"
	"github.com/vexxhost/migratekit-sha/restore"
	"github.com/vexxhost/migratekit-sha/storage"
)

// RestoreHandlers handles file-level restore API requests
type RestoreHandlers struct {
	mountManager   *restore.MountManager
	fileBrowser    *restore.FileBrowser
	fileDownloader *restore.FileDownloader
	cleanupService *restore.CleanupService
	resourceMonitor *restore.ResourceMonitor
}

// NewRestoreHandlers creates a new restore handlers instance
// v2.16.0+: Passes DB connection to MountManager for backup_disks queries
func NewRestoreHandlers(
	db database.Connection,
	repositoryManager *storage.RepositoryManager,
) *RestoreHandlers {
	// Initialize repositories
	mountRepo := database.NewRestoreMountRepository(db)

	// Initialize services
	// v2.16.0+: Pass DB connection for backup_disks table queries
	mountManager := restore.NewMountManager(mountRepo, repositoryManager, db)
	fileBrowser := restore.NewFileBrowser(mountRepo)
	fileDownloader := restore.NewFileDownloader(fileBrowser)
	cleanupService := restore.NewCleanupService(mountRepo, mountManager)
	resourceMonitor := restore.NewResourceMonitor(mountRepo)

	// Start cleanup service
	if err := cleanupService.Start(); err != nil {
		log.WithError(err).Error("Failed to start cleanup service")
	}

	return &RestoreHandlers{
		mountManager:    mountManager,
		fileBrowser:     fileBrowser,
		fileDownloader:  fileDownloader,
		cleanupService:  cleanupService,
		resourceMonitor: resourceMonitor,
	}
}

// RegisterRoutes registers restore API routes
func (rh *RestoreHandlers) RegisterRoutes(r *mux.Router) {
	log.Info("🔗 Registering file-level restore API routes")

	restore := r.PathPrefix("/restore").Subrouter()
	
	// Mount operations
	restore.HandleFunc("/mount", rh.MountBackup).Methods("POST")
	restore.HandleFunc("/mounts", rh.ListMounts).Methods("GET")
	restore.HandleFunc("/{mount_id}", rh.UnmountBackup).Methods("DELETE")

	// File browsing
	restore.HandleFunc("/{mount_id}/files", rh.ListFiles).Methods("GET")
	restore.HandleFunc("/{mount_id}/file-info", rh.GetFileInfo).Methods("GET")

	// File downloads
	restore.HandleFunc("/{mount_id}/download", rh.DownloadFile).Methods("GET")
	restore.HandleFunc("/{mount_id}/download-directory", rh.DownloadDirectory).Methods("GET")

	// Resource monitoring
	restore.HandleFunc("/resources", rh.GetResourceStatus).Methods("GET")
	restore.HandleFunc("/cleanup-status", rh.GetCleanupStatus).Methods("GET")

	log.Info("✅ File-level restore API routes registered")
}

// MountBackup mounts a QCOW2 backup disk for file browsing
// v2.16.0+: Multi-disk support - specify disk_index to mount specific disk
// POST /api/v1/restore/mount
func (rh *RestoreHandlers) MountBackup(w http.ResponseWriter, r *http.Request) {
	log.Info("📥 Received mount backup request (v2.16.0+ multi-disk support)")

	// Parse request
	var req restore.MountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		rh.sendError(w, http.StatusBadRequest, fmt.Sprintf("invalid request: %v", err))
		return
	}

	// Validate request
	if req.BackupID == "" {
		rh.sendError(w, http.StatusBadRequest, "backup_id is required")
		return
	}

	// v2.16.0+: disk_index defaults to 0 for backward compatibility
	if req.DiskIndex < 0 {
		rh.sendError(w, http.StatusBadRequest, "disk_index must be >= 0")
		return
	}

	log.WithFields(log.Fields{
		"backup_id":  req.BackupID,
		"disk_index": req.DiskIndex,
	}).Info("Mounting backup disk with multi-disk support")

	// Mount backup
	mountInfo, err := rh.mountManager.MountBackup(r.Context(), &req)
	if err != nil {
		log.WithError(err).Error("Mount backup failed")
		rh.sendError(w, http.StatusInternalServerError, fmt.Sprintf("failed to mount backup: %v", err))
		return
	}

	log.WithFields(log.Fields{
		"mount_id":   mountInfo.MountID,
		"mount_path": mountInfo.MountPath,
	}).Info("✅ Backup mounted successfully")

	rh.sendJSON(w, http.StatusOK, mountInfo)
}

// UnmountBackup unmounts a QCOW2 backup
// DELETE /api/v1/restore/{mount_id}
func (rh *RestoreHandlers) UnmountBackup(w http.ResponseWriter, r *http.Request) {
	mountID := mux.Vars(r)["mount_id"]
	log.WithField("mount_id", mountID).Info("📤 Received unmount backup request")

	// Unmount backup
	if err := rh.mountManager.UnmountBackup(r.Context(), mountID); err != nil {
		log.WithError(err).Error("Unmount backup failed")
		rh.sendError(w, http.StatusInternalServerError, fmt.Sprintf("failed to unmount backup: %v", err))
		return
	}

	log.WithField("mount_id", mountID).Info("✅ Backup unmounted successfully")

	rh.sendJSON(w, http.StatusOK, map[string]string{
		"message": "backup unmounted successfully",
		"mount_id": mountID,
	})
}

// ListMounts lists all active restore mounts
// GET /api/v1/restore/mounts
func (rh *RestoreHandlers) ListMounts(w http.ResponseWriter, r *http.Request) {
	log.Debug("📋 Received list mounts request")

	mounts, err := rh.mountManager.ListMounts(r.Context())
	if err != nil {
		log.WithError(err).Error("List mounts failed")
		rh.sendError(w, http.StatusInternalServerError, fmt.Sprintf("failed to list mounts: %v", err))
		return
	}

	log.WithField("mount_count", len(mounts)).Debug("✅ Mounts listed successfully")

	rh.sendJSON(w, http.StatusOK, map[string]interface{}{
		"mounts": mounts,
		"count":  len(mounts),
	})
}

// ListFiles lists files and directories within a mounted backup
// GET /api/v1/restore/{mount_id}/files?path=/var/www&recursive=false
func (rh *RestoreHandlers) ListFiles(w http.ResponseWriter, r *http.Request) {
	mountID := mux.Vars(r)["mount_id"]
	path := r.URL.Query().Get("path")
	recursiveStr := r.URL.Query().Get("recursive")

	recursive := false
	if recursiveStr != "" {
		recursive, _ = strconv.ParseBool(recursiveStr)
	}

	log.WithFields(log.Fields{
		"mount_id":  mountID,
		"path":      path,
		"recursive": recursive,
	}).Info("📂 Received list files request")

	req := &restore.ListFilesRequest{
		MountID:   mountID,
		Path:      path,
		Recursive: recursive,
	}

	response, err := rh.fileBrowser.ListFiles(r.Context(), req)
	if err != nil {
		log.WithError(err).Error("List files failed")
		rh.sendError(w, http.StatusInternalServerError, fmt.Sprintf("failed to list files: %v", err))
		return
	}

	log.WithField("file_count", response.TotalCount).Info("✅ Files listed successfully")

	rh.sendJSON(w, http.StatusOK, response)
}

// GetFileInfo retrieves detailed information about a specific file
// GET /api/v1/restore/{mount_id}/file-info?path=/var/www/index.html
func (rh *RestoreHandlers) GetFileInfo(w http.ResponseWriter, r *http.Request) {
	mountID := mux.Vars(r)["mount_id"]
	filePath := r.URL.Query().Get("path")

	log.WithFields(log.Fields{
		"mount_id":  mountID,
		"file_path": filePath,
	}).Debug("📄 Received file info request")

	if filePath == "" {
		rh.sendError(w, http.StatusBadRequest, "path query parameter is required")
		return
	}

	fileInfo, err := rh.fileBrowser.GetFileInfo(r.Context(), mountID, filePath)
	if err != nil {
		log.WithError(err).Error("Get file info failed")
		rh.sendError(w, http.StatusInternalServerError, fmt.Sprintf("failed to get file info: %v", err))
		return
	}

	log.WithField("file_name", fileInfo.Name).Debug("✅ File info retrieved successfully")

	rh.sendJSON(w, http.StatusOK, fileInfo)
}

// DownloadFile downloads an individual file from a mounted backup
// GET /api/v1/restore/{mount_id}/download?path=/var/www/index.html
func (rh *RestoreHandlers) DownloadFile(w http.ResponseWriter, r *http.Request) {
	mountID := mux.Vars(r)["mount_id"]
	filePath := r.URL.Query().Get("path")

	log.WithFields(log.Fields{
		"mount_id":  mountID,
		"file_path": filePath,
	}).Info("📥 Received file download request")

	if filePath == "" {
		rh.sendError(w, http.StatusBadRequest, "path query parameter is required")
		return
	}

	req := &restore.FileDownloadRequest{
		MountID:  mountID,
		FilePath: filePath,
	}

	// Prepare file for download
	fileReader, downloadInfo, err := rh.fileDownloader.DownloadFile(r.Context(), req)
	if err != nil {
		log.WithError(err).Error("File download failed")
		rh.sendError(w, http.StatusInternalServerError, fmt.Sprintf("failed to prepare file download: %v", err))
		return
	}
	defer fileReader.Close()

	// Set headers
	w.Header().Set("Content-Type", downloadInfo.ContentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, downloadInfo.FileName))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", downloadInfo.Size))

	// Stream file to client
	written, err := copyWithContext(r.Context(), w, fileReader)
	if err != nil {
		log.WithError(err).Error("File streaming failed")
		return
	}

	log.WithFields(log.Fields{
		"file_name":      downloadInfo.FileName,
		"bytes_written":  written,
	}).Info("✅ File downloaded successfully")
}

// DownloadDirectory downloads a directory as an archive (ZIP or TAR.GZ)
// GET /api/v1/restore/{mount_id}/download-directory?path=/var/www&format=zip
func (rh *RestoreHandlers) DownloadDirectory(w http.ResponseWriter, r *http.Request) {
	mountID := mux.Vars(r)["mount_id"]
	dirPath := r.URL.Query().Get("path")
	archiveType := r.URL.Query().Get("format")

	log.WithFields(log.Fields{
		"mount_id":     mountID,
		"dir_path":     dirPath,
		"archive_type": archiveType,
	}).Info("📦 Received directory download request")

	if dirPath == "" {
		rh.sendError(w, http.StatusBadRequest, "path query parameter is required")
		return
	}

	// Default to ZIP if not specified
	if archiveType == "" {
		archiveType = "zip"
	}

	req := &restore.DirectoryDownloadRequest{
		MountID:     mountID,
		DirPath:     dirPath,
		ArchiveType: archiveType,
	}

	// Prepare directory archive for download
	archiveReader, downloadInfo, err := rh.fileDownloader.DownloadDirectory(r.Context(), req)
	if err != nil {
		log.WithError(err).Error("Directory download failed")
		rh.sendError(w, http.StatusInternalServerError, fmt.Sprintf("failed to prepare directory download: %v", err))
		return
	}
	defer archiveReader.Close()

	// Set headers
	w.Header().Set("Content-Type", downloadInfo.ContentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, downloadInfo.FileName))

	// Stream archive to client (size unknown for streaming archives)
	written, err := copyWithContext(r.Context(), w, archiveReader)
	if err != nil {
		log.WithError(err).Error("Archive streaming failed")
		return
	}

	log.WithFields(log.Fields{
		"archive_name":  downloadInfo.FileName,
		"archive_type":  archiveType,
		"bytes_written": written,
	}).Info("✅ Directory archive downloaded successfully")
}

// GetResourceStatus returns current resource utilization
// GET /api/v1/restore/resources
func (rh *RestoreHandlers) GetResourceStatus(w http.ResponseWriter, r *http.Request) {
	log.Debug("📊 Received resource status request")

	status, err := rh.resourceMonitor.GetResourceStatus(r.Context())
	if err != nil {
		log.WithError(err).Error("Get resource status failed")
		rh.sendError(w, http.StatusInternalServerError, fmt.Sprintf("failed to get resource status: %v", err))
		return
	}

	log.Debug("✅ Resource status retrieved successfully")

	rh.sendJSON(w, http.StatusOK, status)
}

// GetCleanupStatus returns cleanup service status
// GET /api/v1/restore/cleanup-status
func (rh *RestoreHandlers) GetCleanupStatus(w http.ResponseWriter, r *http.Request) {
	log.Debug("🧹 Received cleanup status request")

	status, err := rh.cleanupService.GetCleanupStatus(r.Context())
	if err != nil {
		log.WithError(err).Error("Get cleanup status failed")
		rh.sendError(w, http.StatusInternalServerError, fmt.Sprintf("failed to get cleanup status: %v", err))
		return
	}

	log.Debug("✅ Cleanup status retrieved successfully")

	rh.sendJSON(w, http.StatusOK, status)
}

// Helper: sendJSON sends JSON response
func (rh *RestoreHandlers) sendJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

// Helper: sendError sends error response
func (rh *RestoreHandlers) sendError(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]string{
		"error": message,
	})
}

// Helper: copyWithContext copies data with context cancellation support
func copyWithContext(ctx context.Context, dst io.Writer, src io.Reader) (int64, error) {
	// Create buffered reader for efficient copying
	buf := make([]byte, 32*1024) // 32KB buffer

	var written int64
	for {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return written, ctx.Err()
		default:
		}

		// Read from source
		nr, err := src.Read(buf)
		if nr > 0 {
			// Write to destination
			nw, werr := dst.Write(buf[0:nr])
			if nw > 0 {
				written += int64(nw)
			}
			if werr != nil {
				return written, werr
			}
			if nr != nw {
				return written, io.ErrShortWrite
			}
		}
		if err != nil {
			if err == io.EOF {
				return written, nil
			}
			return written, err
		}
	}
}
