package vmware_nbdkit

import (
	"context"
	"fmt"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"libguestfs.org/libnbd"
)

// WorkerConfig contains configuration for a parallel copy worker
type WorkerConfig struct {
	WorkerID     int
	SourceSocket string // NBD socket for VMware connection
	TargetNBD    *libnbd.Libnbd
	Extents      []CoalescedExtent
	MaxRetries   int
	RetryDelay   time.Duration
}

// WorkerResult contains statistics from a completed worker
type WorkerResult struct {
	WorkerID        int
	ExtentsProcessed int
	BytesCopied     int64
	Duration        time.Duration
	ThroughputMBps  float64
	Errors          []error
}

// copyWorker processes assigned extents using dedicated NBD connection
func copyWorker(
	ctx context.Context,
	config WorkerConfig,
	progressChan chan<- int64,
	errorChan chan<- error,
	wg *sync.WaitGroup,
) {
	defer wg.Done()

	logger := log.WithField("worker_id", config.WorkerID)
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

	logger.Info("🚀 Worker started")

	var (
		bytesProcessed int64
		extentsProcessed int
	)

	// Process each assigned extent
	for _, extent := range config.Extents {
		// Check for context cancellation
		select {
		case <-ctx.Done():
			logger.Warn("Worker cancelled by context")
			return
		default:
		}

		// Copy this extent with retries
		err := copyExtentWithRetry(
			ctx,
			sourceNBD,
			config.TargetNBD,
			extent,
			config.MaxRetries,
			config.RetryDelay,
			logger,
		)

		if err != nil {
			errorChan <- fmt.Errorf("worker %d: failed to copy extent at offset %d: %w",
				config.WorkerID, extent.Offset, err)
			continue // Try next extent even if this one failed
		}

		// Update progress
		bytesProcessed += extent.Length
		extentsProcessed++

		// Send progress update (non-blocking)
		select {
		case progressChan <- extent.Length:
		default:
			// Channel full, skip this update
		}

		// Log progress every 10 extents
		if extentsProcessed%10 == 0 {
			elapsed := time.Since(startTime).Seconds()
			throughputMBps := float64(bytesProcessed) / elapsed / (1024 * 1024)

			logger.WithFields(log.Fields{
				"extents_processed": extentsProcessed,
				"bytes_processed":   bytesProcessed,
				"throughput_mbps":   fmt.Sprintf("%.2f", throughputMBps),
			}).Debug("📊 Worker progress")
		}
	}

	// Final statistics
	duration := time.Since(startTime)
	throughputMBps := float64(bytesProcessed) / duration.Seconds() / (1024 * 1024)

	logger.WithFields(log.Fields{
		"extents_processed": extentsProcessed,
		"bytes_copied":      bytesProcessed,
		"duration":          duration,
		"throughput_mbps":   fmt.Sprintf("%.2f", throughputMBps),
	}).Info("✅ Worker completed successfully")
}

// copyExtentWithRetry copies a single extent with exponential backoff retry logic
func copyExtentWithRetry(
	ctx context.Context,
	sourceNBD *libnbd.Libnbd,
	targetNBD *libnbd.Libnbd,
	extent CoalescedExtent,
	maxRetries int,
	initialDelay time.Duration,
	logger *log.Entry,
) error {
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		// Check for cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Try to copy the extent
		err := copyExtent(sourceNBD, targetNBD, extent)
		if err == nil {
			// Success!
			if attempt > 0 {
				logger.WithFields(log.Fields{
					"offset":   extent.Offset,
					"length":   extent.Length,
					"attempts": attempt + 1,
				}).Info("✅ Extent copy succeeded after retry")
			}
			return nil
		}

		lastErr = err

		// If this was the last attempt, give up
		if attempt == maxRetries {
			break
		}

		// Calculate backoff delay (exponential: 1s, 2s, 4s, ...)
		delay := initialDelay * time.Duration(1<<uint(attempt))

		logger.WithFields(log.Fields{
			"offset":       extent.Offset,
			"length":       extent.Length,
			"attempt":      attempt + 1,
			"max_retries":  maxRetries + 1,
			"retry_in":     delay,
		}).WithError(err).Warn("⚠️ Extent copy failed, retrying...")

		// Wait before retry
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}
	}

	return fmt.Errorf("extent copy failed after %d attempts: %w", maxRetries+1, lastErr)
}

// copyExtent copies a single extent from source to target NBD
// Handles large extents by splitting them into MaxChunkSize chunks to respect NBD limits
func copyExtent(sourceNBD *libnbd.Libnbd, targetNBD *libnbd.Libnbd, extent CoalescedExtent) error {
	// If extent is larger than MaxChunkSize, process it in chunks
	// This handles VMware CBT returning large extents or coalescing creating oversized chunks
	currentOffset := extent.Offset
	remainingLength := extent.Length
	
	for remainingLength > 0 {
		// Calculate chunk size (max 32 MB to respect NBD server limits)
		chunkSize := remainingLength
		if chunkSize > int64(MaxChunkSize) {
			chunkSize = int64(MaxChunkSize)
		}
		
		// Allocate buffer for this chunk
		buffer := make([]byte, chunkSize)

		// Read from source
		err := sourceNBD.Pread(buffer, uint64(currentOffset), nil)
		if err != nil {
			return fmt.Errorf("source read failed at offset %d: %w", currentOffset, err)
		}

		// Check if chunk is all zeros (sparse optimization)
		if isZeroBlock(buffer) {
			// Use NBD zero command for sparse blocks
			err = targetNBD.Zero(uint64(chunkSize), uint64(currentOffset), nil)
			if err != nil {
				// Fallback to regular write if Zero command fails
				err = targetNBD.Pwrite(buffer, uint64(currentOffset), nil)
				if err != nil {
					return fmt.Errorf("target zero/write fallback failed at offset %d: %w", currentOffset, err)
				}
			}
		} else {
			// Write actual data to target
			err = targetNBD.Pwrite(buffer, uint64(currentOffset), nil)
			if err != nil {
				return fmt.Errorf("target write failed at offset %d: %w", currentOffset, err)
			}
		}
		
		// Move to next chunk
		currentOffset += chunkSize
		remainingLength -= chunkSize
	}

	return nil
}

