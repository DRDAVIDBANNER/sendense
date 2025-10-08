package vmware_nbdkit

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/vexxhost/migratekit/internal/progress"
	"github.com/vexxhost/migratekit/internal/target"
	vmware "github.com/vexxhost/migratekit/internal/vmware"
	"github.com/vmware/govmomi/vim25/types"
	"libguestfs.org/libnbd"
)

// OffsetRange represents a continuous range of disk to copy
type OffsetRange struct {
	Start  int64
	Length int64
}

// ParallelFullCopyToTarget performs full disk copy using parallel NBD workers
// This is optimized for initial/full migrations where the entire disk needs to be copied
func (s *NbdkitServer) ParallelFullCopyToTarget(ctx context.Context, t target.Target, path string, targetIsClean bool) error {
	logger := log.WithFields(log.Fields{
		"vm":   s.Servers.VirtualMachine.Name(),
		"disk": s.Disk.Backing.(types.BaseVirtualDeviceFileBackingInfo).GetVirtualDeviceFileBackingInfo().FileName,
	})

	logger.Info("🚀 Starting parallel full copy")

	// Send initial job type to SNA
	if snaProgressClient := ctx.Value("snaProgressClient"); snaProgressClient != nil {
		if vpc, ok := snaProgressClient.(*progress.SNAProgressClient); ok && vpc.IsEnabled() {
			vpc.SendUpdate(progress.SNAProgressUpdate{
				Stage:    "Transfer",
				Status:   "in_progress",
				SyncType: "initial",
				Percent:  0,
			})
		}
	}

	// Get total disk size
	diskSize := s.getActualDiskSize()
	logger.Infof("📊 Total disk size: %.2f GB (%d bytes)",
		float64(diskSize)/(1024*1024*1024), diskSize)

	// Calculate actual used space for accurate progress
	var totalBytes int64
	diskInfo, err := vmware.CalculateUsedSpace(ctx, s.Servers.VirtualMachine, s.Disk, s.Servers.SnapshotRef)
	if err != nil {
		logger.WithError(err).Warn("CBT calculation failed, using actual VMDK file size")
		totalBytes = diskSize
	} else {
		totalBytes = diskInfo.GetUsedBytes()
		logger.Infof("📊 Using CBT-calculated disk usage: %d GB used of %d GB total",
			diskInfo.UsedGB, diskSize/(1024*1024*1024))
	}

	// Connect to target NBD
	logger.Info("🔍 DEBUG: About to call connectToNBDTarget()")
	nbdTarget, err := s.connectToNBDTarget(ctx, path)
	logger.Info("🔍 DEBUG: connectToNBDTarget() returned to caller")
	
	if err != nil {
		return fmt.Errorf("failed to connect to target: %w", err)
	}
	defer nbdTarget.Close()

	logger.Info("🔍 DEBUG: About to call determineWorkerCount()")
	
	// Determine optimal worker count
	numWorkers := determineWorkerCount(100) // Full copy always uses max workers
	
	logger.Info("🔍 DEBUG: determineWorkerCount() returned")
	logger.Infof("🔧 Using %d parallel workers for full copy", numWorkers)

	// Divide disk into equal ranges for workers
	workerRanges := divideRangesAcrossWorkers(diskSize, numWorkers)

	// Setup progress aggregator
	var snaClient *progress.SNAProgressClient
	if vpc := ctx.Value("snaProgressClient"); vpc != nil {
		if client, ok := vpc.(*progress.SNAProgressClient); ok {
			snaClient = client
		}
	}

	progressAggregator := NewProgressAggregator(totalBytes, snaClient)
	progressChan := make(chan int64, 1000)
	errorChan := make(chan error, numWorkers)

	// Start progress aggregator
	aggregatorCtx, aggregatorCancel := context.WithCancel(ctx)
	defer aggregatorCancel()
	go progressAggregator.Run(aggregatorCtx, progressChan)

	// Launch worker pool for full copy
	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		if workerRanges[i].Length == 0 {
			continue
		}

		wg.Add(1)
		go fullCopyWorker(
			ctx,
			FullCopyWorkerConfig{
				WorkerID:     i,
				SourceSocket: s.Nbdkit.Socket(),
				TargetNBD:    nbdTarget,
				OffsetRange:  workerRanges[i],
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
		return fmt.Errorf("parallel full copy failed with %d worker errors", len(workerErrors))
	}

	// Send final 100% progress update
	if snaClient != nil && snaClient.IsEnabled() {
		progressAggregator.SendFinalUpdate()
		logger.Info("📊 Final progress update sent to SNA")
	}

	logger.WithFields(log.Fields{
		"bytes_copied": diskSize,
		"gb_copied":    diskSize / (1024 * 1024 * 1024),
	}).Info("✅ Parallel full copy completed successfully")

	return nil
}

// FullCopyWorkerConfig contains configuration for a parallel full copy worker
type FullCopyWorkerConfig struct {
	WorkerID     int
	SourceSocket string
	TargetNBD    *libnbd.Libnbd
	OffsetRange  OffsetRange
	MaxRetries   int
	RetryDelay   time.Duration
}

// fullCopyWorker processes a continuous disk range using dedicated NBD connection
func fullCopyWorker(
	ctx context.Context,
	config FullCopyWorkerConfig,
	progressChan chan<- int64,
	errorChan chan<- error,
	wg *sync.WaitGroup,
) {
	defer wg.Done()

	logger := log.WithFields(log.Fields{
		"worker_id": config.WorkerID,
		"start":     config.OffsetRange.Start,
		"end":       config.OffsetRange.Start + config.OffsetRange.Length,
		"size_mb":   config.OffsetRange.Length / (1024 * 1024),
	})
	startTime := time.Now()

	// Create dedicated NBD connection for this worker
	sourceNBD, err := libnbd.Create()
	if err != nil {
		errorChan <- fmt.Errorf("worker %d: failed to create NBD handle: %w", config.WorkerID, err)
		return
	}
	defer sourceNBD.Close()

	// Connect to VMware NBD source via socket
	err = sourceNBD.ConnectUnix(config.SourceSocket)
	if err != nil {
		errorChan <- fmt.Errorf("worker %d: failed to connect to source NBD: %w", config.WorkerID, err)
		return
	}

	logger.Info("🚀 Full copy worker started")

	var (
		bytesProcessed   int64
		chunksProcessed  int
		sparseSkipped    int64
		sparseBytesSaved int64
	)

	// Process assigned disk range in chunks
	currentOffset := config.OffsetRange.Start
	endOffset := config.OffsetRange.Start + config.OffsetRange.Length

	for currentOffset < endOffset {
		// Check for context cancellation
		select {
		case <-ctx.Done():
			logger.Warn("Worker cancelled by context")
			return
		default:
		}

		// Calculate chunk size (max 32MB)
		chunkSize := int64(MaxChunkSize)
		if currentOffset+chunkSize > endOffset {
			chunkSize = endOffset - currentOffset
		}

		// Copy this chunk with retries
		wasSparse, err := copyChunkWithRetry(
			ctx,
			sourceNBD,
			config.TargetNBD,
			currentOffset,
			chunkSize,
			config.MaxRetries,
			config.RetryDelay,
			logger,
		)

		if err != nil {
			errorChan <- fmt.Errorf("worker %d: failed to copy chunk at offset %d: %w",
				config.WorkerID, currentOffset, err)
			return // Fatal error, stop this worker
		}

		// Track sparse optimization
		if wasSparse {
			sparseSkipped++
			sparseBytesSaved += chunkSize
		}

		// Update progress
		bytesProcessed += chunkSize
		chunksProcessed++

		// Send progress update (non-blocking)
		select {
		case progressChan <- chunkSize:
		default:
			// Channel full, skip this update
		}

		// Log progress every 10 chunks
		if chunksProcessed%10 == 0 {
			elapsed := time.Since(startTime).Seconds()
			throughputMBps := float64(bytesProcessed) / elapsed / (1024 * 1024)

			logger.WithFields(log.Fields{
				"chunks_processed": chunksProcessed,
				"bytes_processed":  bytesProcessed,
				"mb_processed":     bytesProcessed / (1024 * 1024),
				"throughput_mbps":  fmt.Sprintf("%.2f", throughputMBps),
				"sparse_saved_mb":  sparseBytesSaved / (1024 * 1024),
			}).Debug("📊 Worker progress")
		}

		currentOffset += chunkSize
	}

	// Final statistics
	duration := time.Since(startTime)
	throughputMBps := float64(bytesProcessed) / duration.Seconds() / (1024 * 1024)

	logger.WithFields(log.Fields{
		"chunks_processed":  chunksProcessed,
		"bytes_copied":      bytesProcessed,
		"mb_copied":         bytesProcessed / (1024 * 1024),
		"sparse_skipped":    sparseSkipped,
		"sparse_saved_mb":   sparseBytesSaved / (1024 * 1024),
		"duration":          duration,
		"throughput_mbps":   fmt.Sprintf("%.2f", throughputMBps),
	}).Info("✅ Full copy worker completed successfully")
}

// copyChunkWithRetry copies a single chunk with retry logic and sparse optimization
func copyChunkWithRetry(
	ctx context.Context,
	sourceNBD *libnbd.Libnbd,
	targetNBD *libnbd.Libnbd,
	offset int64,
	chunkSize int64,
	maxRetries int,
	initialDelay time.Duration,
	logger *log.Entry,
) (wasSparse bool, err error) {
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		// Check for cancellation
		select {
		case <-ctx.Done():
			return false, ctx.Err()
		default:
		}

		// Try to copy the chunk
		sparse, err := copyChunk(sourceNBD, targetNBD, offset, chunkSize)
		if err == nil {
			// Success!
			if attempt > 0 {
				logger.WithFields(log.Fields{
					"offset":   offset,
					"size":     chunkSize,
					"attempts": attempt + 1,
				}).Info("✅ Chunk copy succeeded after retry")
			}
			return sparse, nil
		}

		lastErr = err

		// If this was the last attempt, give up
		if attempt == maxRetries {
			break
		}

		// Calculate backoff delay
		delay := initialDelay * time.Duration(1<<uint(attempt))

		logger.WithFields(log.Fields{
			"offset":      offset,
			"size":        chunkSize,
			"attempt":     attempt + 1,
			"max_retries": maxRetries + 1,
			"retry_in":    delay,
		}).WithError(err).Warn("⚠️ Chunk copy failed, retrying...")

		// Wait before retry
		select {
		case <-ctx.Done():
			return false, ctx.Err()
		case <-time.After(delay):
		}
	}

	return false, fmt.Errorf("chunk copy failed after %d attempts: %w", maxRetries+1, lastErr)
}

// copyChunk copies a single chunk from source to target with sparse optimization
func copyChunk(sourceNBD *libnbd.Libnbd, targetNBD *libnbd.Libnbd, offset int64, chunkSize int64) (wasSparse bool, err error) {
	// Use hierarchical sparse detection for large chunks
	var isZero bool
	var buffer []byte

	if chunkSize >= 4*1024*1024 {
		// Large chunk: use sampling first
		isZero = isZeroBlockSampled(sourceNBD, offset, int(chunkSize))
		if !isZero {
			// Not zero, read the actual data
			buffer = make([]byte, chunkSize)
			err = sourceNBD.Pread(buffer, uint64(offset), nil)
			if err != nil {
				return false, fmt.Errorf("source read failed: %w", err)
			}
		}
	} else {
		// Small chunk: read and check directly
		buffer = make([]byte, chunkSize)
		err = sourceNBD.Pread(buffer, uint64(offset), nil)
		if err != nil {
			return false, fmt.Errorf("source read failed: %w", err)
		}
		isZero = isZeroBlock(buffer)
	}

	if isZero {
		// Sparse block - use NBD zero command
		err = targetNBD.Zero(uint64(chunkSize), uint64(offset), nil)
		if err != nil {
			// Fallback to regular write if Zero command fails
			if buffer == nil {
				// We used sampling, need to read actual data for fallback
				buffer = make([]byte, chunkSize)
				err = sourceNBD.Pread(buffer, uint64(offset), nil)
				if err != nil {
					return false, fmt.Errorf("source read for zero fallback failed: %w", err)
				}
			}
			err = targetNBD.Pwrite(buffer, uint64(offset), nil)
			if err != nil {
				return false, fmt.Errorf("target write (zero fallback) failed: %w", err)
			}
		}
		return true, nil
	}

	// Non-zero data - write to target
	err = targetNBD.Pwrite(buffer, uint64(offset), nil)
	if err != nil {
		return false, fmt.Errorf("target write failed: %w", err)
	}

	return false, nil
}

// divideRangesAcrossWorkers divides disk into equal ranges for N workers
// All offsets and lengths are aligned to 512-byte boundaries for NBD compatibility
func divideRangesAcrossWorkers(diskSize int64, numWorkers int) []OffsetRange {
	if numWorkers <= 0 {
		numWorkers = 1
	}

	const alignment = 512 // NBD sector alignment requirement

	ranges := make([]OffsetRange, numWorkers)
	rangeSize := diskSize / int64(numWorkers)
	
	// Align range size down to 512-byte boundary
	rangeSize = (rangeSize / alignment) * alignment

	for i := 0; i < numWorkers; i++ {
		ranges[i] = OffsetRange{
			Start:  int64(i) * rangeSize,
			Length: rangeSize,
		}

		// Last worker gets any remaining bytes (up to disk end)
		if i == numWorkers-1 {
			ranges[i].Length = diskSize - ranges[i].Start
		}

		log.WithFields(log.Fields{
			"worker_id":  i,
			"start":      ranges[i].Start,
			"length":     ranges[i].Length,
			"size_mb":    ranges[i].Length / (1024 * 1024),
			"size_gb":    ranges[i].Length / (1024 * 1024 * 1024),
		}).Debug("📦 Worker range allocation")
	}

	return ranges
}

// connectToNBDTarget establishes NBD connection to the target
func (s *NbdkitServer) connectToNBDTarget(ctx context.Context, path string) (*libnbd.Libnbd, error) {
	if !strings.HasPrefix(path, "nbd://") {
		return nil, fmt.Errorf("parallel full copy only supports NBD targets (got: %s)", path)
	}

	// Create NBD target handle
	nbdTarget, err := libnbd.Create()
	if err != nil {
		return nil, fmt.Errorf("failed to create target NBD handle: %w", err)
	}

	// Parse NBD URL: nbd://host:port/export
	u, err := url.Parse(path)
	if err != nil {
		nbdTarget.Close()
		return nil, fmt.Errorf("failed to parse NBD URL: %w", err)
	}

	exportName := strings.TrimPrefix(u.Path, "/")
	err = nbdTarget.SetExportName(exportName)
	if err != nil {
		nbdTarget.Close()
		return nil, fmt.Errorf("failed to set export name: %w", err)
	}

	// Enable sparse optimization metadata context
	err = nbdTarget.AddMetaContext("base:allocation")
	if err != nil {
		log.WithError(err).Warn("Failed to add metadata context - sparse optimization disabled")
	} else {
		log.Info("✅ NBD metadata context enabled for sparse optimization")
	}

	log.Info("🔍 DEBUG: About to call ConnectTcp()")
	
	// Connect to target
	err = nbdTarget.ConnectTcp(u.Hostname(), u.Port())
	
	log.Info("🔍 DEBUG: ConnectTcp() returned successfully")
	
	if err != nil {
		log.WithError(err).Error("🔍 DEBUG: ConnectTcp() returned with error")
		nbdTarget.Close()
		return nil, fmt.Errorf("failed to connect to target NBD: %w", err)
	}

	log.Info("🔍 DEBUG: About to return nbdTarget from connectToNBDTarget()")
	return nbdTarget, nil
}

