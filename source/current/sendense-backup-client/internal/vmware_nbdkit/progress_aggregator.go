package vmware_nbdkit

import (
	"context"
	"sync/atomic"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/vexxhost/migratekit/internal/progress"
)

// ProgressAggregator collects progress from multiple workers and sends SNA updates
type ProgressAggregator struct {
	totalBytes           int64
	bytesTransferred     atomic.Int64
	startTime            time.Time
	lastUpdateTime       time.Time
	lastProgressPercent  float64
	snaProgressClient    *progress.SNAProgressClient
	updateInterval       time.Duration
	progressPercentDelta float64 // Minimum percent change to trigger update
}

// NewProgressAggregator creates a new progress aggregator
func NewProgressAggregator(totalBytes int64, snaClient *progress.SNAProgressClient) *ProgressAggregator {
	return &ProgressAggregator{
		totalBytes:           totalBytes,
		startTime:            time.Now(),
		lastUpdateTime:       time.Now(),
		snaProgressClient:    snaClient,
		updateInterval:       2 * time.Second, // Send SNA updates every 2 seconds
		progressPercentDelta: 1.0,              // Or when progress changes by 1%
	}
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

// maybeUpdateVMA sends SNA progress update if enough time/progress has passed
func (pa *ProgressAggregator) maybeUpdateVMA(logger *log.Entry) {
	if pa.snaProgressClient == nil || !pa.snaProgressClient.IsEnabled() {
		return
	}

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

	// Calculate throughput (bytes per second)
	elapsed := time.Since(pa.startTime).Seconds()
	var throughputBPS int64
	if elapsed > 0 {
		throughputBPS = int64(float64(currentBytes) / elapsed)
	}

	// Send SNA progress update
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

	pa.lastUpdateTime = time.Now()
	pa.lastProgressPercent = currentPercent
}

// SendFinalUpdate sends the final 100% completion update to SNA
func (pa *ProgressAggregator) SendFinalUpdate() error {
	if pa.snaProgressClient == nil || !pa.snaProgressClient.IsEnabled() {
		return nil
	}

	return pa.snaProgressClient.SendUpdate(progress.SNAProgressUpdate{
		Stage:            "Transfer",
		Status:           "completed",
		BytesTransferred: pa.totalBytes,
		TotalBytes:       pa.totalBytes,
		Percent:          100,
		ThroughputBPS:    0,
	})
}

// GetBytesTransferred returns current bytes transferred (thread-safe)
func (pa *ProgressAggregator) GetBytesTransferred() int64 {
	return pa.bytesTransferred.Load()
}

