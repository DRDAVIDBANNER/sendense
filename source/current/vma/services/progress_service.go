package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/vexxhost/migratekit/source/current/vma/progress"
)

// ProgressUpdateRequest represents a progress update request from migratekit
type ProgressUpdateRequest struct {
	Stage            progress.ReplicationStage  `json:"stage"`
	Status           progress.ReplicationStatus `json:"status,omitempty"`
	BytesTransferred int64                      `json:"bytes_transferred"`
	TotalBytes       int64                      `json:"total_bytes,omitempty"`
	ThroughputBPS    int64                      `json:"throughput_bps"`
	Percent          float64                    `json:"percent,omitempty"`
	DiskID           string                     `json:"disk_id,omitempty"`
	SyncType         string                     `json:"sync_type,omitempty"` // 🎯 FIX: Sync type from migratekit
}

// ProgressService provides replication progress tracking functionality
type ProgressService struct {
	// In-memory storage for active job progress
	// In a production system, this might be backed by a database or cache
	jobProgress map[string]*progress.ReplicationProgress
	// Mutex to protect concurrent access to jobProgress map
	mutex sync.RWMutex
}

// NewProgressService creates a new progress service instance
func NewProgressService() *ProgressService {
	return &ProgressService{
		jobProgress: make(map[string]*progress.ReplicationProgress),
	}
}

// GetJobProgress retrieves progress information for a specific job
func (s *ProgressService) GetJobProgress(ctx context.Context, jobID string) (*progress.ReplicationProgress, error) {
	// Use read lock for safe concurrent access
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	log.WithFields(log.Fields{
		"job_id":   jobID,
		"map_size": len(s.jobProgress),
	}).Debug("🔍 DEBUG: GetJobProgress called")

	// Look up job progress from in-memory store
	jobProg, exists := s.jobProgress[jobID]
	if !exists {
		log.WithField("job_id", jobID).Debug("🔍 DEBUG: Job NOT FOUND in map")
		// Print all job IDs in the map for debugging
		jobIDs := make([]string, 0, len(s.jobProgress))
		for id := range s.jobProgress {
			jobIDs = append(jobIDs, id)
		}
		log.WithField("current_jobs", jobIDs).Debug("🔍 DEBUG: Current jobs in map")
		return nil, fmt.Errorf("job not found: %s", jobID)
	}

	log.WithField("job_id", jobID).Debug("🔍 DEBUG: Job FOUND in map")

	// 🚨 CRITICAL FIX: Don't modify shared data under read lock
	// Return a copy with updated timestamp instead of modifying original
	result := *jobProg
	result.UpdatedAt = time.Now()

	return &result, nil
}

// UpdateJobProgress updates progress information for a job
func (s *ProgressService) UpdateJobProgress(ctx context.Context, jobID string, update *progress.ReplicationProgress) error {
	// Use write lock for safe concurrent access
	s.mutex.Lock()
	defer s.mutex.Unlock()

	update.UpdatedAt = time.Now()
	s.jobProgress[jobID] = update
	return nil
}

// StartJobTracking initializes progress tracking for a new job
func (s *ProgressService) StartJobTracking(ctx context.Context, jobID string) error {
	// Use write lock for safe concurrent access
	s.mutex.Lock()
	defer s.mutex.Unlock()

	jobProg := &progress.ReplicationProgress{
		JobID:     jobID,
		Stage:     progress.StageDiscover,
		Status:    progress.StatusQueued,
		StartedAt: time.Now(),
		UpdatedAt: time.Now(),
		Aggregate: progress.AggregateProgress{
			TotalBytes:       0,
			BytesTransferred: 0,
			ThroughputBPS:    0,
			Percent:          0.0,
		},
		CBT: progress.CBTInfo{
			Type:             progress.CBTTypeFull,
			PreviousChangeID: "",
			ChangeID:         "",
		},
		NBD: progress.NBDInfo{
			Exports: []progress.NBDExport{},
		},
		Disks: []progress.DiskProgress{},
	}

	s.jobProgress[jobID] = jobProg
	return nil
}

// UpdateJobStage updates the current stage of a job
func (s *ProgressService) UpdateJobStage(ctx context.Context, jobID string, stage progress.ReplicationStage, status progress.ReplicationStatus) error {
	jobProg, exists := s.jobProgress[jobID]
	if !exists {
		return fmt.Errorf("job not found: %s", jobID)
	}

	jobProg.Stage = stage
	jobProg.Status = status
	jobProg.UpdatedAt = time.Now()

	return nil
}

// UpdateCBTInfo updates CBT information for a job
func (s *ProgressService) UpdateCBTInfo(ctx context.Context, jobID string, cbtType progress.CBTType, previousChangeID, changeID string) error {
	jobProg, exists := s.jobProgress[jobID]
	if !exists {
		return fmt.Errorf("job not found: %s", jobID)
	}

	jobProg.CBT = progress.CBTInfo{
		Type:             cbtType,
		PreviousChangeID: previousChangeID,
		ChangeID:         changeID,
	}
	jobProg.UpdatedAt = time.Now()

	return nil
}

// AddDisk adds a disk to be tracked for a job
func (s *ProgressService) AddDisk(ctx context.Context, jobID, diskID, label string, plannedBytes int64) error {
	jobProg, exists := s.jobProgress[jobID]
	if !exists {
		return fmt.Errorf("job not found: %s", jobID)
	}

	diskProg := progress.DiskProgress{
		ID:               diskID,
		Label:            label,
		PlannedBytes:     plannedBytes,
		BytesTransferred: 0,
		ThroughputBPS:    0,
		Percent:          0.0,
		Status:           progress.DiskStatusQueued,
	}

	jobProg.Disks = append(jobProg.Disks, diskProg)

	// Update aggregate total bytes
	jobProg.Aggregate.TotalBytes += plannedBytes
	jobProg.UpdatedAt = time.Now()

	return nil
}

// UpdateDiskProgress updates progress for a specific disk
func (s *ProgressService) UpdateDiskProgress(ctx context.Context, jobID, diskID string, bytesTransferred int64, throughputBPS int64, status progress.DiskStatus) error {
	jobProg, exists := s.jobProgress[jobID]
	if !exists {
		return fmt.Errorf("job not found: %s", jobID)
	}

	// Find the disk and update its progress
	for i := range jobProg.Disks {
		if jobProg.Disks[i].ID == diskID {
			jobProg.Disks[i].BytesTransferred = bytesTransferred
			jobProg.Disks[i].ThroughputBPS = throughputBPS
			jobProg.Disks[i].Status = status

			// Calculate percentage
			if jobProg.Disks[i].PlannedBytes > 0 {
				jobProg.Disks[i].Percent = float64(bytesTransferred) / float64(jobProg.Disks[i].PlannedBytes) * 100.0
			}

			break
		}
	}

	// Recalculate aggregate progress
	s.recalculateAggregateProgress(jobProg)
	jobProg.UpdatedAt = time.Now()

	return nil
}

// AddNBDExport adds an NBD export to the job tracking
func (s *ProgressService) AddNBDExport(ctx context.Context, jobID, exportName, device string, connected bool) error {
	jobProg, exists := s.jobProgress[jobID]
	if !exists {
		return fmt.Errorf("job not found: %s", jobID)
	}

	now := time.Now()
	export := progress.NBDExport{
		Name:      exportName,
		Device:    device,
		Connected: connected,
		StartedAt: &now,
	}

	jobProg.NBD.Exports = append(jobProg.NBD.Exports, export)
	jobProg.UpdatedAt = time.Now()

	return nil
}

// FinishJobTracking marks a job as completed and cleans up tracking
func (s *ProgressService) FinishJobTracking(ctx context.Context, jobID string, success bool) error {
	jobProg, exists := s.jobProgress[jobID]
	if !exists {
		return fmt.Errorf("job not found: %s", jobID)
	}

	if success {
		jobProg.Status = progress.StatusSucceeded
		jobProg.Stage = progress.StagePersistChangeIDs
		jobProg.Aggregate.Percent = 100.0
	} else {
		jobProg.Status = progress.StatusFailed
	}

	jobProg.UpdatedAt = time.Now()
	return nil
}

// recalculateAggregateProgress recalculates the aggregate progress from individual disks
func (s *ProgressService) recalculateAggregateProgress(jobProg *progress.ReplicationProgress) {
	var totalBytesTransferred int64
	var totalPlannedBytes int64
	var totalThroughput int64

	for _, disk := range jobProg.Disks {
		totalBytesTransferred += disk.BytesTransferred
		totalPlannedBytes += disk.PlannedBytes
		totalThroughput += disk.ThroughputBPS
	}

	jobProg.Aggregate.BytesTransferred = totalBytesTransferred
	jobProg.Aggregate.TotalBytes = totalPlannedBytes // 🚨 FIX: Calculate total bytes from disk planned bytes
	jobProg.Aggregate.ThroughputBPS = totalThroughput

	if jobProg.Aggregate.TotalBytes > 0 {
		jobProg.Aggregate.Percent = float64(totalBytesTransferred) / float64(jobProg.Aggregate.TotalBytes) * 100.0
	}
}

// CalculatePlannedBytes calculates the planned bytes for a job based on CBT or allocated extents
func (s *ProgressService) CalculatePlannedBytes(ctx context.Context, jobID string, cbtType progress.CBTType, previousChangeID, currentChangeID string) (int64, error) {
	// This would typically call VMware APIs to calculate planned bytes
	// For full replication: sum allocated extents per VMDK (VDDK QueryAllocatedBlocks or VMware Layout API)
	// For incremental: sum CBT changed extents between previousChangeID and currentChangeID

	// TODO: Implement actual VMware API calls
	// For now, return a placeholder value
	// This should be replaced with real VDDK or VMware Layout API calls

	if cbtType == progress.CBTTypeIncremental && previousChangeID != "" {
		// Incremental: query CBT changed blocks
		// return s.queryCBTChangedBlocks(ctx, previousChangeID, currentChangeID)
		return 0, fmt.Errorf("CBT calculation not yet implemented")
	} else {
		// Full: query allocated extents
		// return s.queryAllocatedExtents(ctx, jobID)
		return 0, fmt.Errorf("allocated extents calculation not yet implemented")
	}
}

// UpdateJobProgressFromMigratekit updates progress information from migratekit callbacks
func (s *ProgressService) UpdateJobProgressFromMigratekit(ctx context.Context, jobID string, update *ProgressUpdateRequest) error {
	// Use write lock for safe concurrent access
	s.mutex.Lock()
	defer s.mutex.Unlock()

	jobProg, exists := s.jobProgress[jobID]
	if !exists {
		return fmt.Errorf("job not found: %s", jobID)
	}

	// Update stage and status if provided
	if update.Stage != "" {
		jobProg.Stage = update.Stage
	}
	if update.Status != "" {
		jobProg.Status = update.Status
	}

	// 🎯 FIX: Update CBT sync type from migratekit
	if update.SyncType != "" {
		switch update.SyncType {
		case "full", "initial":
			jobProg.CBT.Type = progress.CBTTypeFull
		case "incremental":
			jobProg.CBT.Type = progress.CBTTypeIncremental
		}
	}

	// 🎯 MULTI-DISK PROGRESS HANDLING
	// If DiskID is provided, update per-disk progress and recalculate aggregate
	if update.DiskID != "" {
		s.updateDiskProgress(jobProg, update)
		s.recalculateAggregateProgress(jobProg)
	} else {
		// Legacy single-disk behavior: update aggregate directly
		s.updateAggregateProgress(jobProg, update)
	}

	jobProg.UpdatedAt = time.Now()
	return nil
}

// updateDiskProgress updates progress for a specific disk, creating it if it doesn't exist
func (s *ProgressService) updateDiskProgress(jobProg *progress.ReplicationProgress, update *ProgressUpdateRequest) {
	// Find existing disk or create new one
	diskIndex := -1
	for i := range jobProg.Disks {
		if jobProg.Disks[i].ID == update.DiskID {
			diskIndex = i
			break
		}
	}

	// Create new disk if not found
	if diskIndex == -1 {
		newDisk := progress.DiskProgress{
			ID:               update.DiskID,
			Label:            fmt.Sprintf("Disk %s", update.DiskID),
			PlannedBytes:     update.TotalBytes,
			BytesTransferred: 0,
			ThroughputBPS:    0,
			Percent:          0.0,
			Status:           progress.DiskStatusStreaming,
		}
		jobProg.Disks = append(jobProg.Disks, newDisk)
		diskIndex = len(jobProg.Disks) - 1
	}

	// Update disk progress
	disk := &jobProg.Disks[diskIndex]
	if update.BytesTransferred > 0 {
		disk.BytesTransferred = update.BytesTransferred
	}
	if update.TotalBytes > 0 {
		disk.PlannedBytes = update.TotalBytes
	}
	if update.ThroughputBPS >= 0 {
		disk.ThroughputBPS = update.ThroughputBPS
	}

	// Calculate disk percentage
	if disk.PlannedBytes > 0 {
		disk.Percent = float64(disk.BytesTransferred) / float64(disk.PlannedBytes) * 100.0
	}
}

// updateAggregateProgress updates aggregate progress directly (legacy single-disk behavior)
func (s *ProgressService) updateAggregateProgress(jobProg *progress.ReplicationProgress, update *ProgressUpdateRequest) {
	if update.BytesTransferred > 0 {
		jobProg.Aggregate.BytesTransferred = update.BytesTransferred
	}
	if update.TotalBytes > 0 {
		jobProg.Aggregate.TotalBytes = update.TotalBytes
	}
	if update.ThroughputBPS >= 0 {
		jobProg.Aggregate.ThroughputBPS = update.ThroughputBPS
	}
	if update.Percent > 0 {
		jobProg.Aggregate.Percent = update.Percent
	} else if jobProg.Aggregate.TotalBytes > 0 {
		// Calculate percentage from bytes if not provided
		jobProg.Aggregate.Percent = float64(jobProg.Aggregate.BytesTransferred) / float64(jobProg.Aggregate.TotalBytes) * 100.0
	}
}

// FindActiveJobForVMDisk attempts to find an active job ID that corresponds to a VM-disk format job ID
// This is a fallback method to handle job ID mismatches between migratekit and VMA
func (s *ProgressService) FindActiveJobForVMDisk(ctx context.Context, vmDiskJobID string) (string, error) {
	// Simple strategy: find the most recently updated active job
	// In production, you might want more sophisticated mapping based on VM path or other criteria
	var mostRecentJobID string
	var mostRecentTime time.Time

	for jobID, jobProg := range s.jobProgress {
		// Only consider jobs that are currently active (not completed or failed)
		if jobProg.Status == progress.StatusStreaming || jobProg.Status == progress.StatusSnapshot || jobProg.Status == progress.StatusPreparing {
			if jobProg.UpdatedAt.After(mostRecentTime) {
				mostRecentTime = jobProg.UpdatedAt
				mostRecentJobID = jobID
			}
		}
	}

	if mostRecentJobID == "" {
		return "", fmt.Errorf("no active job found for VM-disk format: %s", vmDiskJobID)
	}

	return mostRecentJobID, nil
}

// FindJobByNBDExport finds a job ID by its NBD export name
func (s *ProgressService) FindJobByNBDExport(ctx context.Context, exportName string) (string, error) {
	// Look for jobs that have this NBD export name
	for jobID, jobProg := range s.jobProgress {
		// Check if this job has NBD exports that match
		for _, export := range jobProg.NBD.Exports {
			if export.Name == exportName {
				return jobID, nil
			}
		}
	}

	return "", fmt.Errorf("no job found for NBD export: %s", exportName)
}

// InitializeJobWithNBDExport initializes job tracking with NBD export name mapping
func (s *ProgressService) InitializeJobWithNBDExport(ctx context.Context, jobID string, exportName string) error {
	// Use write lock for safe concurrent access
	s.mutex.Lock()
	defer s.mutex.Unlock()

	log.WithFields(log.Fields{
		"job_id":      jobID,
		"export_name": exportName,
	}).Debug("🔍 DEBUG: InitializeJobWithNBDExport called")

	// Initialize basic job progress if it doesn't exist
	if _, exists := s.jobProgress[jobID]; !exists {
		log.WithField("job_id", jobID).Debug("🔍 DEBUG: Creating new job entry")
		now := time.Now()
		s.jobProgress[jobID] = &progress.ReplicationProgress{
			JobID:     jobID,
			Status:    progress.StatusQueued,
			Stage:     progress.StageDiscover,
			StartedAt: now,
			UpdatedAt: now,
			NBD: progress.NBDInfo{
				Exports: []progress.NBDExport{},
			},
			Disks:     []progress.DiskProgress{},
			Aggregate: progress.AggregateProgress{},
		}
		log.WithFields(log.Fields{
			"job_id":   jobID,
			"map_size": len(s.jobProgress),
		}).Debug("🔍 DEBUG: Job entry created")
	} else {
		log.WithField("job_id", jobID).Debug("🔍 DEBUG: Job entry already exists")
	}

	// Add the NBD export mapping
	if exportName != "" {
		now := time.Now()
		export := progress.NBDExport{
			Name:      exportName,
			Connected: false,
			StartedAt: &now,
		}
		s.jobProgress[jobID].NBD.Exports = append(s.jobProgress[jobID].NBD.Exports, export)
	}

	s.jobProgress[jobID].UpdatedAt = time.Now()
	log.WithFields(log.Fields{
		"job_id":         jobID,
		"final_map_size": len(s.jobProgress),
	}).Debug("🔍 DEBUG: InitializeJobWithNBDExport completed")
	return nil
}
