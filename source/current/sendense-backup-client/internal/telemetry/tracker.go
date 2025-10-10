package telemetry

import (
	"context"
	"time"

	log "github.com/sirupsen/logrus"
)

// ProgressTracker implements hybrid cadence telemetry sending
// Sends updates based on time intervals AND progress milestones
type ProgressTracker struct {
	client           *Client
	jobID            string
	lastSentTime     time.Time
	lastSentProgress float64
	timeInterval     time.Duration
	progressInterval float64
}

// NewProgressTracker creates a new progress tracker
func NewProgressTracker(client *Client, jobID string) *ProgressTracker {
	return &ProgressTracker{
		client:           client,
		jobID:            jobID,
		lastSentTime:     time.Now(), // Initialize to now
		lastSentProgress: 0.0,
		timeInterval:     5 * time.Second, // Send every 5 seconds
		progressInterval: 10.0,            // Send every 10% progress
	}
}

// ShouldSend determines if telemetry should be sent based on hybrid cadence rules
// Returns true if ANY of these conditions are met:
// 1. Time-based: 5 seconds elapsed since last send
// 2. Progress-based: 10% progress made since last send
func (pt *ProgressTracker) ShouldSend(currentProgress float64) bool {
	now := time.Now()
	timeSinceLast := now.Sub(pt.lastSentTime)
	progressDelta := currentProgress - pt.lastSentProgress
	
	log.WithFields(log.Fields{
		"time_since_last_sec": timeSinceLast.Seconds(),
		"time_threshold_sec":  pt.timeInterval.Seconds(),
		"progress_delta":      progressDelta,
		"progress_threshold":  pt.progressInterval,
		"current_progress":    currentProgress,
		"last_sent_progress":  pt.lastSentProgress,
	}).Info("⏱️  Cadence evaluation")
	
	// Time-based: 5 seconds elapsed
	if timeSinceLast >= pt.timeInterval {
		log.WithFields(log.Fields{
			"elapsed":   timeSinceLast.Seconds(),
			"threshold": pt.timeInterval.Seconds(),
			"trigger":   "TIME",
		}).Info("✅ TIME threshold met - will send")
		return true
	}
	
	// Progress-based: 10% progress made
	if progressDelta >= pt.progressInterval {
		log.WithFields(log.Fields{
			"progress_delta": progressDelta,
			"threshold":      pt.progressInterval,
			"trigger":        "PROGRESS",
		}).Info("✅ PROGRESS threshold met - will send")
		return true
	}
	
	log.WithFields(log.Fields{
		"time_remaining_sec":     (pt.timeInterval - timeSinceLast).Seconds(),
		"progress_remaining_pct": pt.progressInterval - progressDelta,
	}).Debug("❌ No threshold met - will NOT send")
	
	return false
}

// SendIfNeeded sends telemetry if cadence conditions are met OR if job status changed
// Always sends for non-running states (completed, failed, etc.)
func (pt *ProgressTracker) SendIfNeeded(jobID string, update *TelemetryUpdate) error {
	// Always send for non-running states
	if update.Status != "running" {
		log.WithFields(log.Fields{
			"job_id": jobID,
			"status": update.Status,
		}).Debug("Sending telemetry for non-running state")
		
		err := pt.client.SendBackupUpdate(jobID, update)
		if err == nil {
			pt.lastSentTime = time.Now()
			pt.lastSentProgress = update.ProgressPercent
		}
		return err
	}
	
	// For running state, check cadence conditions
	shouldSend := pt.ShouldSend(update.ProgressPercent)
	
	log.WithFields(log.Fields{
		"job_id":          jobID,
		"progress":        update.ProgressPercent,
		"should_send":     shouldSend,
		"last_sent_time":  pt.lastSentTime.Format("15:04:05"),
		"time_since_last": time.Since(pt.lastSentTime).Seconds(),
	}).Info("🔍 Cadence check for telemetry send")
	
	if shouldSend {
		log.WithFields(log.Fields{
			"job_id":   jobID,
			"progress": update.ProgressPercent,
			"bytes":    update.BytesTransferred,
			"speed":    update.TransferSpeedBps,
		}).Info("📤 ATTEMPTING HTTP send to SHA")
		
		err := pt.client.SendBackupUpdate(jobID, update)
		if err == nil {
			pt.lastSentTime = time.Now()
			pt.lastSentProgress = update.ProgressPercent
			
			log.WithFields(log.Fields{
				"job_id":   jobID,
				"progress": update.ProgressPercent,
				"speed":    update.TransferSpeedBps,
			}).Info("✅ HTTP send SUCCESS - telemetry delivered to SHA")
		} else {
			log.WithError(err).Error("❌ HTTP send FAILED - telemetry not delivered")
		}
		return err
	}
	
	// Cadence conditions not met - skip sending
	log.WithFields(log.Fields{
		"job_id":   jobID,
		"progress": update.ProgressPercent,
		"reason":   "cadence conditions not met",
	}).Debug("⏭️  Skipping send - cadence not triggered")
	return nil
}

// UpdateProgress updates overall progress and sends if needed
// Used by progress aggregator to send real-time updates
func (pt *ProgressTracker) UpdateProgress(
	ctx context.Context,
	bytesTransferred int64,
	totalBytes int64,
	transferSpeedBps int64,
	etaSeconds int,
	currentPhase string,
) {
	// Calculate progress percent from bytes (don't hardcode 0!)
	progressPercent := 0.0
	if totalBytes > 0 {
		progressPercent = (float64(bytesTransferred) / float64(totalBytes)) * 100.0
	}
	
	// Build complete telemetry update with all required fields
	update := &TelemetryUpdate{
		JobID:            pt.jobID,                         // ✅ FIX: Populate job ID
		JobType:          "backup",                         // ✅ FIX: Set job type
		Status:           "running",
		CurrentPhase:     currentPhase,                     // ✅ FIX: Pass actual phase
		BytesTransferred: bytesTransferred,
		TotalBytes:       totalBytes,                       // ✅ FIX: Include total bytes
		TransferSpeedBps: transferSpeedBps,
		ETASeconds:       etaSeconds,
		ProgressPercent:  progressPercent,                  // ✅ FIX: Calculate from bytes!
		Timestamp:        time.Now().Format(time.RFC3339), // ✅ FIX: Add timestamp
		Disks:            []DiskTelemetry{},                // ✅ FIX: Initialize (per-disk coming later)
	}
	
	// Log before sending for debugging
	log.WithFields(log.Fields{
		"job_id":    pt.jobID,
		"bytes":     bytesTransferred,
		"total":     totalBytes,
		"percent":   progressPercent,
		"speed_bps": transferSpeedBps,
		"phase":     currentPhase,
	}).Info("🚀 Sending telemetry update to SHA")
	
	// Send via SendIfNeeded which handles cadence logic
	if err := pt.SendIfNeeded(pt.jobID, update); err != nil {
		log.WithError(err).Error("❌ FAILED to send telemetry update to SHA")
	}
}

// UpdateJobStatus updates job status and sends immediately (state change)
// Used for phase transitions, errors, completion
func (pt *ProgressTracker) UpdateJobStatus(ctx context.Context, status, currentPhase, errorMessage string) {
	// Build complete telemetry update for status change
	update := &TelemetryUpdate{
		JobID:        pt.jobID,                         // ✅ FIX: Include job ID
		JobType:      "backup",                         // ✅ FIX: Set job type
		Status:       status,
		CurrentPhase: currentPhase,
		Timestamp:    time.Now().Format(time.RFC3339), // ✅ FIX: Add timestamp
		Disks:        []DiskTelemetry{},                // ✅ FIX: Initialize
	}
	
	if errorMessage != "" {
		update.Error = &ErrorInfo{
			Message:   errorMessage,
			Timestamp: time.Now().Format(time.RFC3339),
		}
	}
	
	log.WithFields(log.Fields{
		"job_id": pt.jobID,
		"status": status,
		"phase":  currentPhase,
	}).Info("🚀 Sending job status telemetry to SHA")
	
	// Always send status changes (non-running state triggers immediate send)
	if err := pt.client.SendBackupUpdate(pt.jobID, update); err != nil {
		log.WithError(err).Error("❌ FAILED to send job status telemetry to SHA")
	}
}

