package vmware_nbdkit

import (
	"context"
	"os"
	"sync/atomic"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/vexxhost/migratekit/internal/progress"
	"github.com/vexxhost/migratekit/internal/telemetry"
)

// ProgressAggregator collects progress from multiple workers and sends SNA updates + SHA telemetry
type ProgressAggregator struct {
	totalBytes           int64
	bytesTransferred     atomic.Int64
	startTime            time.Time
	lastUpdateTime       time.Time
	lastProgressPercent  float64
	snaProgressClient    *progress.SNAProgressClient
	telemetryTracker     *telemetry.ProgressTracker // 🆕 NEW: SHA telemetry
	jobID                string                       // 🆕 NEW: Job ID for telemetry
	updateInterval       time.Duration
	progressPercentDelta float64 // Minimum percent change to trigger update
}

// NewProgressAggregator creates a new progress aggregator with SNA + SHA telemetry support
func NewProgressAggregator(totalBytes int64, snaClient *progress.SNAProgressClient) *ProgressAggregator {
	// 🆕 NEW: Get telemetry tracker from context if available
	// Note: This will be set from context in the caller (parallel_full_copy.go)
	jobID := os.Getenv("MIGRATEKIT_JOB_ID")
	
	return &ProgressAggregator{
		totalBytes:           totalBytes,
		startTime:            time.Now(),
		lastUpdateTime:       time.Now(),
		snaProgressClient:    snaClient,
		telemetryTracker:     nil, // Will be set via SetTelemetryTracker
		jobID:                jobID,
		updateInterval:       2 * time.Second, // Send updates every 2 seconds
		progressPercentDelta: 1.0,              // Or when progress changes by 1%
	}
}

// SetTelemetryTracker sets the telemetry tracker (called from context)
func (pa *ProgressAggregator) SetTelemetryTracker(tracker *telemetry.ProgressTracker) {
	pa.telemetryTracker = tracker
}

// Run starts the progress aggregator goroutine
func (pa *ProgressAggregator) Run(ctx context.Context, progressChan <-chan int64) {
	logger := log.WithField("component", "progress_aggregator")
	ticker := time.NewTicker(100 * time.Millisecond) // Check every 100ms
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.Info("Progress aggregator stopped")
			return

		case bytesChunk := <-progressChan:
			// Accumulate progress from workers (atomic)
			pa.bytesTransferred.Add(bytesChunk)

		case <-ticker.C:
			// Check if we should send SNA update
			pa.maybeUpdateVMA(logger)
		}
	}
}

// maybeUpdateVMA sends SNA progress update + SHA telemetry if enough time/progress has passed
func (pa *ProgressAggregator) maybeUpdateVMA(logger *log.Entry) {
	currentBytes := pa.bytesTransferred.Load()
	currentPercent := float64(currentBytes) / float64(pa.totalBytes) * 100
	timeSinceUpdate := time.Since(pa.lastUpdateTime)

	// Send update if:
	// 1. Enough time has passed (2 seconds)
	// 2. Progress changed by at least 1%
	shouldUpdate := timeSinceUpdate >= pa.updateInterval ||
		currentPercent >= pa.lastProgressPercent+pa.progressPercentDelta

	if !shouldUpdate {
		return
	}

	// Calculate throughput (bytes per second) and ETA
	elapsed := time.Since(pa.startTime).Seconds()
	var throughputBPS int64
	var etaSeconds int
	if elapsed > 0 {
		throughputBPS = int64(float64(currentBytes) / elapsed)
		if throughputBPS > 0 {
			remainingBytes := pa.totalBytes - currentBytes
			etaSeconds = int(float64(remainingBytes) / float64(throughputBPS))
		}
	}

	// 1️⃣ Send SNA progress update (backward compatibility during transition)
	if pa.snaProgressClient != nil && pa.snaProgressClient.IsEnabled() {
		err := pa.snaProgressClient.SendUpdate(progress.SNAProgressUpdate{
			Stage:            "Transfer",
			Status:           "in_progress",
			BytesTransferred: currentBytes,
			TotalBytes:       pa.totalBytes,
			Percent:          currentPercent,
			ThroughputBPS:    throughputBPS,
		})

		if err != nil {
			logger.WithError(err).Warn("Failed to send SNA progress update")
		} else {
			logger.WithFields(log.Fields{
				"bytes_transferred": currentBytes,
				"percent":           currentPercent,
				"throughput_bps":    throughputBPS,
				"throughput_mbps":   float64(throughputBPS) / (1024 * 1024),
			}).Debug("📊 SNA progress update sent")
		}
	}

	// 2️⃣ 🆕 NEW: Send SHA telemetry (push-based real-time tracking)
	if pa.telemetryTracker != nil && pa.jobID != "" {
		pa.telemetryTracker.UpdateProgress(
			context.Background(),
			currentBytes,
			pa.totalBytes,  // ✅ FIX: Pass total bytes (was missing!)
			throughputBPS,
			etaSeconds,
			"transferring", // ✅ FIX: Pass current phase
		)
		
		logger.WithFields(log.Fields{
			"job_id":            pa.jobID,
			"bytes_transferred": currentBytes,
			"total_bytes":       pa.totalBytes,
			"percent":           currentPercent,
			"throughput_bps":    throughputBPS,
			"eta_seconds":       etaSeconds,
		}).Debug("🚀 SHA telemetry update sent")
	}

	pa.lastUpdateTime = time.Now()
	pa.lastProgressPercent = currentPercent
}

// SendFinalUpdate sends the final 100% completion update to SNA + SHA telemetry
func (pa *ProgressAggregator) SendFinalUpdate() error {
	// 1️⃣ Send SNA completion update (backward compatibility)
	if pa.snaProgressClient != nil && pa.snaProgressClient.IsEnabled() {
		err := pa.snaProgressClient.SendUpdate(progress.SNAProgressUpdate{
			Stage:            "Transfer",
			Status:           "completed",
			BytesTransferred: pa.totalBytes,
			TotalBytes:       pa.totalBytes,
			Percent:          100,
			ThroughputBPS:    0,
		})
		if err != nil {
			log.WithError(err).Warn("Failed to send SNA final update")
		}
	}

	// 2️⃣ 🆕 NEW: Send SHA telemetry completion
	if pa.telemetryTracker != nil && pa.jobID != "" {
		// Send final progress update with 100%
		pa.telemetryTracker.UpdateProgress(
			context.Background(),
			pa.totalBytes,  // All bytes transferred
			pa.totalBytes,
			0, // No more throughput
			0, // No more ETA
			"completed",
		)
		
		// Then send completion status
		pa.telemetryTracker.UpdateJobStatus(context.Background(), "completed", "completed", "")
		log.WithField("job_id", pa.jobID).Info("🚀 SHA telemetry completion sent")
	}

	return nil
}

// GetBytesTransferred returns current bytes transferred (thread-safe)
func (pa *ProgressAggregator) GetBytesTransferred() int64 {
	return pa.bytesTransferred.Load()
}

