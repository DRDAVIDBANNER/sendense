package services

import (
	"context"
	"fmt"
	"time"

	"github.com/vexxhost/migratekit/source/current/sna/progress"
)

// ProgressService provides replication progress tracking functionality
type ProgressService struct {
	// In-memory storage for active job progress
	// In a production system, this might be backed by a database or cache
	jobProgress map[string]*progress.ReplicationProgress
}

// NewProgressService creates a new progress service instance
func NewProgressService() *ProgressService {
	return &ProgressService{
		jobProgress: make(map[string]*progress.ReplicationProgress),
	}
}

// GetJobProgress retrieves progress information for a specific job
func (s *ProgressService) GetJobProgress(ctx context.Context, jobID string) (*progress.ReplicationProgress, error) {
	// Look up job progress from in-memory store
	jobProg, exists := s.jobProgress[jobID]
	if !exists {
		return nil, fmt.Errorf("job not found: %s", jobID)
	}

	// Update the timestamp to current time
	jobProg.UpdatedAt = time.Now()

	return jobProg, nil
}

// UpdateJobProgress updates progress information for a job
func (s *ProgressService) UpdateJobProgress(ctx context.Context, jobID string, update *progress.ReplicationProgress) error {
	update.UpdatedAt = time.Now()
	s.jobProgress[jobID] = update
	return nil
}

// StartJobTracking initializes progress tracking for a new job
func (s *ProgressService) StartJobTracking(ctx context.Context, jobID string) error {
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
	var totalThroughput int64

	for _, disk := range jobProg.Disks {
		totalBytesTransferred += disk.BytesTransferred
		totalThroughput += disk.ThroughputBPS
	}

	jobProg.Aggregate.BytesTransferred = totalBytesTransferred
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

// ProgressUpdateRequest represents a progress update from migratekit
type ProgressUpdateRequest struct {
	Stage            progress.ReplicationStage
	Status           progress.ReplicationStatus
	BytesTransferred int64
	TotalBytes       int64
	ThroughputBPS    int64
	Percent          float64
	DiskID           string
}

// UpdateJobProgressFromMigratekit updates progress information from migratekit callbacks
func (s *ProgressService) UpdateJobProgressFromMigratekit(ctx context.Context, jobID string, update *ProgressUpdateRequest) error {
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

	// Update aggregate progress
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

	// Update disk-specific progress if disk_id provided
	if update.DiskID != "" {
		for i := range jobProg.Disks {
			if jobProg.Disks[i].ID == update.DiskID {
				jobProg.Disks[i].BytesTransferred = update.BytesTransferred
				jobProg.Disks[i].ThroughputBPS = update.ThroughputBPS
				if jobProg.Disks[i].PlannedBytes > 0 {
					jobProg.Disks[i].Percent = float64(jobProg.Disks[i].BytesTransferred) / float64(jobProg.Disks[i].PlannedBytes) * 100.0
				}
				break
			}
		}
	}

	jobProg.UpdatedAt = time.Now()
	return nil
}
