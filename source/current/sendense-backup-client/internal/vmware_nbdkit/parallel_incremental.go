package vmware_nbdkit

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/vexxhost/migratekit/internal/progress"
	"github.com/vexxhost/migratekit/internal/target"
	"github.com/vexxhost/migratekit/internal/telemetry"
	"github.com/vmware/govmomi/vim25/methods"
	"github.com/vmware/govmomi/vim25/types"
	vmware "github.com/vexxhost/migratekit/internal/vmware"
	"libguestfs.org/libnbd"
)

const (
	// DefaultNumWorkers is the default number of parallel NBD workers per disk
	DefaultNumWorkers = 4

	// CoalesceGapThreshold is the maximum gap between extents to merge (1 MB)
	CoalesceGapThreshold = 1 * 1024 * 1024

	// MaxRetries is the number of retry attempts for failed chunks
	MaxRetries = 3

	// InitialRetryDelay is the base delay for exponential backoff
	InitialRetryDelay = 1 * time.Second
)

// ParallelIncrementalCopyToTarget performs incremental copy using parallel NBD workers
// This is a drop-in replacement for IncrementalCopyToTarget with better throughput
func (s *NbdkitServer) ParallelIncrementalCopyToTarget(ctx context.Context, t target.Target, path string) error {
	logger := log.WithFields(log.Fields{
		"vm":   s.Servers.VirtualMachine.Name(),
		"disk": s.Disk.Backing.(types.BaseVirtualDeviceFileBackingInfo).GetVirtualDeviceFileBackingInfo().FileName,
	})

	logger.Info("🚀 Starting parallel incremental copy")

	// Send initial job type to SNA
	if snaProgressClient := ctx.Value("snaProgressClient"); snaProgressClient != nil {
		if vpc, ok := snaProgressClient.(*progress.SNAProgressClient); ok && vpc.IsEnabled() {
			vpc.SendUpdate(progress.SNAProgressUpdate{
				Stage:    "Transfer",
				Status:   "in_progress",
				SyncType: "incremental",
				Percent:  0,
			})
		}
	}

	// Get current ChangeID from target
	currentChangeId, err := t.GetCurrentChangeID(ctx)
	if err != nil {
		return err
	}

	// Step 1: Query all changed disk areas from VMware
	extents, err := s.queryChangedDiskAreas(ctx, currentChangeId)
	if err != nil {
		return fmt.Errorf("failed to query changed disk areas: %w", err)
	}

	if len(extents) == 0 {
		logger.Info("No changed disk areas, skipping copy")
		
		// 🎯 CRITICAL: Send completion update to SNA even when nothing to copy
		// Without this, frontend shows job stuck at "Transfer" stage and times out
		if snaProgressClient := ctx.Value("snaProgressClient"); snaProgressClient != nil {
			if vpc, ok := snaProgressClient.(*progress.SNAProgressClient); ok && vpc.IsEnabled() {
				vpc.SendUpdate(progress.SNAProgressUpdate{
					Stage:            "Transfer",
					Status:           "completed",
					BytesTransferred: 0,
					TotalBytes:       0,
					Percent:          100,
					ThroughputBPS:    0,
				})
				logger.Info("📊 Sent completion update to SNA (zero changed blocks)")
			}
		}
		
		return nil
	}

	// Step 2: Calculate delta size for progress tracking
	deltaSize := s.calculateTotalDeltaSize(extents)
	logger.Infof("📊 Total delta size: %.2f MB (%d extents)",
		float64(deltaSize)/(1024*1024), len(extents))

	// Step 3: Coalesce extents to reduce request overhead
	coalescedExtents := coalesceExtents(extents, CoalesceGapThreshold, MaxChunkSize)
	totalBytes := calculateTotalBytes(coalescedExtents)

	logger.WithFields(log.Fields{
		"original_extents":  len(extents),
		"coalesced_extents": len(coalescedExtents),
		"total_bytes":       totalBytes,
		"total_mb":          totalBytes / (1024 * 1024),
	}).Info("🔗 Extent coalescing completed")

	// Step 4: Get NBD target connection
	nbdTarget, err := s.connectToTarget(ctx, t, path)
	if err != nil {
		return fmt.Errorf("failed to connect to target: %w", err)
	}
	defer nbdTarget.Close()

	// Step 5: Determine optimal worker count
	numWorkers := determineWorkerCount(len(coalescedExtents))
	logger.Infof("🔧 Using %d parallel workers", numWorkers)

	// Step 6: Split extents across workers
	workerExtents := splitExtentsAcrossWorkers(coalescedExtents, numWorkers)

	// Step 7: Setup progress aggregator
	var snaClient *progress.SNAProgressClient
	if vpc := ctx.Value("snaProgressClient"); vpc != nil {
		if client, ok := vpc.(*progress.SNAProgressClient); ok {
			snaClient = client
		}
	}

	progressAggregator := NewProgressAggregator(totalBytes, snaClient)
	
	// 🆕 NEW: Set telemetry tracker from context (SHA push-based real-time progress)
	if telemetryTracker := ctx.Value("telemetryTracker"); telemetryTracker != nil {
		if tracker, ok := telemetryTracker.(*telemetry.ProgressTracker); ok {
			progressAggregator.SetTelemetryTracker(tracker)
			logger.Info("🚀 SHA telemetry tracker initialized for incremental copy")
		}
	}
	
	progressChan := make(chan int64, 1000)    // Buffered channel for progress updates
	errorChan := make(chan error, numWorkers) // Buffered channel for errors

	// Start progress aggregator
	aggregatorCtx, aggregatorCancel := context.WithCancel(ctx)
	defer aggregatorCancel()
	go progressAggregator.Run(aggregatorCtx, progressChan)

	// Step 8: Launch worker pool
	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		if len(workerExtents[i]) == 0 {
			continue // Skip workers with no extents
		}

		wg.Add(1)
		go copyWorker(
			ctx,
			WorkerConfig{
				WorkerID:     i,
				SourceSocket: s.Nbdkit.Socket(),
				TargetNBD:    nbdTarget,
				Extents:      workerExtents[i],
				MaxRetries:   MaxRetries,
				RetryDelay:   InitialRetryDelay,
			},
			progressChan,
			errorChan,
			&wg,
		)
	}

	// Wait for all workers to complete
	wg.Wait()
	close(progressChan)
	close(errorChan)

	// Check for errors
	var workerErrors []error
	for err := range errorChan {
		workerErrors = append(workerErrors, err)
	}

	if len(workerErrors) > 0 {
		logger.WithField("error_count", len(workerErrors)).Error("❌ Workers reported errors")
		for _, err := range workerErrors {
			logger.WithError(err).Error("Worker error")
		}
		return fmt.Errorf("parallel copy failed with %d worker errors", len(workerErrors))
	}

	// Send final 100% progress update
	if snaClient != nil && snaClient.IsEnabled() {
		progressAggregator.SendFinalUpdate()
		logger.Info("📊 Final progress update sent to SNA")
	}

	logger.WithFields(log.Fields{
		"bytes_copied": totalBytes,
		"mb_copied":    totalBytes / (1024 * 1024),
	}).Info("✅ Parallel incremental copy completed successfully")

	return nil
}

// queryChangedDiskAreas queries all changed disk areas from VMware CBT
func (s *NbdkitServer) queryChangedDiskAreas(ctx context.Context, currentChangeId *vmware.ChangeID) ([]DiskExtent, error) {
	logger := log.WithField("change_id", currentChangeId.Value)
	logger.Info("🔍 Querying changed disk areas from VMware")

	var allExtents []DiskExtent
	startOffset := int64(0)
	diskSize := s.getActualDiskSize()

	for startOffset < diskSize {
		req := types.QueryChangedDiskAreas{
			This:        s.Servers.VirtualMachine.Reference(),
			Snapshot:    &s.Servers.SnapshotRef,
			DeviceKey:   s.Disk.Key,
			StartOffset: startOffset,
			ChangeId:    currentChangeId.Value,
		}

		res, err := methods.QueryChangedDiskAreas(ctx, s.Servers.VirtualMachine.Client(), &req)
		if err != nil {
			return nil, fmt.Errorf("QueryChangedDiskAreas failed at offset %d: %w", startOffset, err)
		}

		diskChangeInfo := res.Returnval

		// Collect changed areas
		for _, area := range diskChangeInfo.ChangedArea {
			allExtents = append(allExtents, DiskExtent{
				Offset: area.Start,
				Length: area.Length,
			})
		}

		// Move to next batch
		startOffset = diskChangeInfo.StartOffset + diskChangeInfo.Length

		if startOffset >= diskSize {
			break
		}
	}

	logger.WithField("extent_count", len(allExtents)).Info("📊 Changed disk areas queried")
	return allExtents, nil
}

// calculateTotalDeltaSize returns the sum of all extent lengths
func (s *NbdkitServer) calculateTotalDeltaSize(extents []DiskExtent) int64 {
	var total int64
	for _, extent := range extents {
		total += extent.Length
	}
	return total
}

// connectToTarget establishes NBD connection to the target
func (s *NbdkitServer) connectToTarget(ctx context.Context, t target.Target, path string) (*libnbd.Libnbd, error) {
	// Connect to target
	err := t.Connect(ctx)
	if err != nil {
		return nil, fmt.Errorf("target connect failed: %w", err)
	}

	// For NBD targets, get the NBD handle
	if nbdTarget, ok := t.(*target.NBDTarget); ok {
		nbdHandle := nbdTarget.GetNBDHandle()
		if nbdHandle == nil {
			return nil, fmt.Errorf("NBD target handle is nil")
		}
		return nbdHandle, nil
	}

	// For other targets, would need different connection logic
	return nil, fmt.Errorf("target type %T not supported for parallel copy", t)
}

// determineWorkerCount calculates optimal number of workers based on extent count
func determineWorkerCount(extentCount int) int {
	// Use fewer workers if we don't have many extents
	if extentCount < 10 {
		return 1
	} else if extentCount < 50 {
		return 2
	} else if extentCount < 200 {
		return 3
	}

	// Default to 4 workers for large extent counts
	return DefaultNumWorkers
}

// ParallelIncrementalCopyEnabled checks if parallel copy is enabled via environment variable
func ParallelIncrementalCopyEnabled() bool {
	val := os.Getenv("MIGRATEKIT_PARALLEL_NBD")
	return val == "true" || val == "1" || val == "enabled"
}

// IncrementalCopyToTargetAutoSelect automatically chooses between serial and parallel copy
// This provides a safe rollback path if parallel copy causes issues
func (s *NbdkitServer) IncrementalCopyToTargetAutoSelect(ctx context.Context, t target.Target, path string) error {
	logger := log.WithField("vm", s.Servers.VirtualMachine.Name())

	// Check if parallel copy is explicitly enabled
	if ParallelIncrementalCopyEnabled() {
		logger.Info("🚀 Parallel NBD copy enabled via MIGRATEKIT_PARALLEL_NBD")
		err := s.ParallelIncrementalCopyToTarget(ctx, t, path)
		if err != nil {
			logger.WithError(err).Warn("⚠️ Parallel copy failed, falling back to serial copy")
			return s.IncrementalCopyToTarget(ctx, t, path)
		}
		return nil
	}

	// Default: use serial copy (existing behavior)
	logger.Info("📝 Using serial incremental copy (default)")
	return s.IncrementalCopyToTarget(ctx, t, path)
}

