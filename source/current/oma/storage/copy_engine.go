// Package storage provides backup copy engine for multi-repository replication
// Following project rules: modular design, worker pool pattern, enterprise reliability
package storage

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

// BackupCopyEngine orchestrates automatic backup replication to secondary repositories.
// Enterprise 3-2-1 backup rule: 3 copies, 2 media types, 1 offsite.
// Worker pool processes copy jobs concurrently with verification.
type BackupCopyEngine struct {
	policyRepo   PolicyRepository
	repoManager  *RepositoryManager
	maxWorkers   int
	checkInterval time.Duration
	workers      []*copyWorker
	stopChan     chan struct{}
	wg           sync.WaitGroup
}

// copyWorker represents a single copy worker goroutine.
type copyWorker struct {
	id       int
	engine   *BackupCopyEngine
	stopChan chan struct{}
}

// NewBackupCopyEngine creates a new backup copy engine.
func NewBackupCopyEngine(policyRepo PolicyRepository, repoManager *RepositoryManager) *BackupCopyEngine {
	return &BackupCopyEngine{
		policyRepo:    policyRepo,
		repoManager:   repoManager,
		maxWorkers:    3,                // 3 concurrent copy jobs max (job sheet spec)
		checkInterval: 30 * time.Second, // Check for pending copies every 30 seconds
		stopChan:      make(chan struct{}),
	}
}

// Start begins the backup copy engine worker pool.
func (bce *BackupCopyEngine) Start(ctx context.Context) {
	log.WithField("workers", bce.maxWorkers).Info("Backup copy engine started")

	// Create worker pool
	for i := 0; i < bce.maxWorkers; i++ {
		worker := &copyWorker{
			id:       i + 1,
			engine:   bce,
			stopChan: make(chan struct{}),
		}
		bce.workers = append(bce.workers, worker)

		bce.wg.Add(1)
		go worker.run(ctx)
	}

	// Monitor and restart workers if needed
	bce.wg.Wait()
	log.Info("Backup copy engine stopped")
}

// Stop stops the backup copy engine and all workers.
func (bce *BackupCopyEngine) Stop() {
	log.Info("Stopping backup copy engine...")
	close(bce.stopChan)

	// Stop all workers
	for _, worker := range bce.workers {
		close(worker.stopChan)
	}

	bce.wg.Wait()
}

// run is the main worker loop.
func (w *copyWorker) run(ctx context.Context) {
	defer w.engine.wg.Done()

	log.WithField("worker_id", w.id).Info("Copy worker started")
	ticker := time.NewTicker(w.engine.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			w.processPendingCopies(ctx)
		case <-w.stopChan:
			log.WithField("worker_id", w.id).Info("Copy worker stopped")
			return
		case <-w.engine.stopChan:
			log.WithField("worker_id", w.id).Info("Copy worker stopped (engine shutdown)")
			return
		case <-ctx.Done():
			log.WithField("worker_id", w.id).Info("Copy worker context cancelled")
			return
		}
	}
}

// processPendingCopies processes one pending copy job.
func (w *copyWorker) processPendingCopies(ctx context.Context) {
	// Get one pending copy
	copies, err := w.engine.policyRepo.ListPendingCopies(ctx)
	if err != nil {
		log.WithError(err).WithField("worker_id", w.id).Error("Failed to list pending copies")
		return
	}

	if len(copies) == 0 {
		return // No pending copies
	}

	// Process first pending copy
	copy := copies[0]
	
	log.WithFields(log.Fields{
		"worker_id":        w.id,
		"copy_id":          copy.ID,
		"source_backup_id": copy.SourceBackupID,
		"repository_id":    copy.RepositoryID,
	}).Info("Processing backup copy")

	// Execute copy
	if err := w.executeCopy(ctx, copy); err != nil {
		log.WithError(err).WithFields(log.Fields{
			"worker_id": w.id,
			"copy_id":   copy.ID,
		}).Error("Copy failed")

		// Update status to failed
		_ = w.engine.policyRepo.UpdateBackupCopyStatus(ctx, copy.ID, BackupCopyStatusFailed, err.Error())
		return
	}

	log.WithFields(log.Fields{
		"worker_id": w.id,
		"copy_id":   copy.ID,
	}).Info("Backup copy completed successfully")
}

// executeCopy executes a backup copy operation.
func (w *copyWorker) executeCopy(ctx context.Context, copy *BackupCopy) error {
	// Update status to copying
	if err := w.engine.policyRepo.UpdateBackupCopyStatus(ctx, copy.ID, BackupCopyStatusCopying, ""); err != nil {
		return fmt.Errorf("failed to update status to copying: %w", err)
	}

	// Get source backup to find file path
	sourceBackup, err := w.engine.repoManager.GetBackupFromAnyRepository(ctx, copy.SourceBackupID)
	if err != nil {
		return fmt.Errorf("failed to get source backup: %w", err)
	}

	// Get destination repository config to determine file path
	destConfig, err := w.engine.repoManager.GetRepositoryConfig(ctx, copy.RepositoryID)
	if err != nil {
		return fmt.Errorf("failed to get destination config: %w", err)
	}

	// Build destination file path
	destPath, err := w.buildDestinationPath(destConfig, sourceBackup)
	if err != nil {
		return fmt.Errorf("failed to build destination path: %w", err)
	}

	// Ensure destination directory exists
	destDir := filepath.Dir(destPath)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Copy file
	if err := w.copyFile(sourceBackup.FilePath, destPath); err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}

	// Update copy record with file path and size
	copy.FilePath = destPath
	fileInfo, err := os.Stat(destPath)
	if err != nil {
		return fmt.Errorf("failed to stat destination file: %w", err)
	}
	copy.SizeBytes = fileInfo.Size()

	// Update status to verifying
	if err := w.engine.policyRepo.UpdateBackupCopyStatus(ctx, copy.ID, BackupCopyStatusVerifying, ""); err != nil {
		return fmt.Errorf("failed to update status to verifying: %w", err)
	}

	// Verify copy
	if err := w.verifyCopy(sourceBackup.FilePath, destPath); err != nil {
		return fmt.Errorf("verification failed: %w", err)
	}

	// Update verification status
	if err := w.engine.policyRepo.UpdateBackupCopyVerification(ctx, copy.ID, true); err != nil {
		return fmt.Errorf("failed to update verification status: %w", err)
	}

	// Update status to completed
	if err := w.engine.policyRepo.UpdateBackupCopyStatus(ctx, copy.ID, BackupCopyStatusCompleted, ""); err != nil {
		return fmt.Errorf("failed to update status to completed: %w", err)
	}

	log.WithFields(log.Fields{
		"copy_id":   copy.ID,
		"size_bytes": copy.SizeBytes,
		"dest_path": destPath,
	}).Info("Copy verified and completed")

	return nil
}

// buildDestinationPath builds the destination file path for a backup copy.
func (w *copyWorker) buildDestinationPath(config *RepositoryConfig, backup *Backup) (string, error) {
	// Get base path from repository config
	var basePath string
	switch config.Type {
	case RepositoryTypeLocal:
		if localConfig, ok := config.Config.(LocalRepositoryConfig); ok {
			basePath = localConfig.Path
		} else if configMap, ok := config.Config.(map[string]interface{}); ok {
			if path, ok := configMap["path"].(string); ok {
				basePath = path
			}
		}
	case RepositoryTypeNFS:
		if nfsConfig, ok := config.Config.(NFSRepositoryConfig); ok {
			basePath = nfsConfig.MountPoint
		} else if configMap, ok := config.Config.(map[string]interface{}); ok {
			if mountPoint, ok := configMap["mount_point"].(string); ok {
				basePath = mountPoint
			}
		}
	case RepositoryTypeCIFS, RepositoryTypeSMB:
		if cifsConfig, ok := config.Config.(CIFSRepositoryConfig); ok {
			basePath = cifsConfig.MountPoint
		} else if configMap, ok := config.Config.(map[string]interface{}); ok {
			if mountPoint, ok := configMap["mount_point"].(string); ok {
				basePath = mountPoint
			}
		}
	default:
		return "", fmt.Errorf("unsupported repository type: %s", config.Type)
	}

	if basePath == "" {
		return "", fmt.Errorf("could not determine base path for repository")
	}

	// Build path: basePath/vm-uuid/disk-N/filename
	filename := filepath.Base(backup.FilePath)
	destPath := filepath.Join(basePath, backup.VMContextID, fmt.Sprintf("disk-%d", backup.DiskID), filename)

	return destPath, nil
}

// copyFile copies a file from source to destination.
// Uses cp --reflink=auto for CoW filesystem optimization (job sheet requirement).
func (w *copyWorker) copyFile(src, dest string) error {
	// Try cp --reflink=auto for CoW filesystem optimization (XFS, Btrfs)
	cmd := exec.Command("cp", "--reflink=auto", src, dest)
	if output, err := cmd.CombinedOutput(); err == nil {
		log.WithFields(log.Fields{
			"src":  src,
			"dest": dest,
		}).Debug("File copied using cp --reflink=auto")
		return nil
	} else {
		log.WithFields(log.Fields{
			"src":    src,
			"dest":   dest,
			"error":  err,
			"output": string(output),
		}).Debug("cp --reflink=auto failed, falling back to standard copy")
	}

	// Fallback to standard Go file copy
	return w.standardCopy(src, dest)
}

// standardCopy performs standard file copy using Go io.Copy.
func (w *copyWorker) standardCopy(src, dest string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer destFile.Close()

	// Copy file contents
	written, err := io.Copy(destFile, sourceFile)
	if err != nil {
		return fmt.Errorf("failed to copy file contents: %w", err)
	}

	// Sync to ensure data is written to disk
	if err := destFile.Sync(); err != nil {
		return fmt.Errorf("failed to sync destination file: %w", err)
	}

	log.WithFields(log.Fields{
		"src":           src,
		"dest":          dest,
		"bytes_written": written,
	}).Debug("File copied using standard copy")

	return nil
}

// verifyCopy verifies a backup copy using SHA256 checksums.
func (w *copyWorker) verifyCopy(src, dest string) error {
	// Calculate source checksum
	srcChecksum, err := w.calculateChecksum(src)
	if err != nil {
		return fmt.Errorf("failed to calculate source checksum: %w", err)
	}

	// Calculate destination checksum
	destChecksum, err := w.calculateChecksum(dest)
	if err != nil {
		return fmt.Errorf("failed to calculate destination checksum: %w", err)
	}

	// Compare checksums
	if srcChecksum != destChecksum {
		return fmt.Errorf("checksum mismatch: source=%s, dest=%s", srcChecksum, destChecksum)
	}

	log.WithFields(log.Fields{
		"src":      src,
		"dest":     dest,
		"checksum": srcChecksum,
	}).Debug("Copy verification passed")

	return nil
}

// calculateChecksum calculates SHA256 checksum of a file.
func (w *copyWorker) calculateChecksum(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

// TriggerCopyForBackup manually triggers copy jobs for a backup.
// Used when OnBackupComplete is called.
func (bce *BackupCopyEngine) TriggerCopyForBackup(ctx context.Context, backupID string, policyID string) error {
	// This is called by PolicyManager.OnBackupComplete
	// Copy jobs should already be created, so this is a no-op
	// The worker pool will pick them up automatically
	log.WithFields(log.Fields{
		"backup_id": backupID,
		"policy_id": policyID,
	}).Debug("Copy jobs triggered for backup")
	return nil
}
