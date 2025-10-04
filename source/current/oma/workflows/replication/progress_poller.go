package replication

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/vexxhost/migratekit-oma/joblog"
)

// VMAProgressResponse represents the JSON response from VMA /progress/{jobId} endpoint
type VMAProgressResponse struct {
	JobID     string            `json:"job_id"`
	Stage     string            `json:"stage"`     // ReplicationStage
	Status    string            `json:"status"`    // ReplicationStatus
	StartedAt time.Time         `json:"started_at"`
	UpdatedAt time.Time         `json:"updated_at"`

	Aggregate AggregateProgress `json:"aggregate"`
	CBT       CBTInfo           `json:"cbt"`
	NBD       NBDInfo           `json:"nbd"`
	Disks     []DiskProgress    `json:"disks"`
}

// AggregateProgress represents overall job progress
type AggregateProgress struct {
	TotalBytes       int64   `json:"total_bytes"`
	BytesTransferred int64   `json:"bytes_transferred"`
	ThroughputBPS    int64   `json:"throughput_bps"`
	Percent          float64 `json:"percent"`
}

// CBTInfo represents CBT-related information
type CBTInfo struct {
	Type             string `json:"type"`               // "full" or "incremental"
	PreviousChangeID string `json:"previous_change_id"`
	ChangeID         string `json:"change_id"`
}

// NBDExport represents a single NBD export
type NBDExport struct {
	Name      string     `json:"name"`
	Device    string     `json:"device"`
	Connected bool       `json:"connected"`
	StartedAt *time.Time `json:"started_at"`
}

// NBDInfo represents NBD export information
type NBDInfo struct {
	Exports []NBDExport `json:"exports"`
}

// DiskProgress represents individual disk replication progress
type DiskProgress struct {
	ID               string  `json:"id"`
	Label            string  `json:"label"`
	PlannedBytes     int64   `json:"planned_bytes"`
	BytesTransferred int64   `json:"bytes_transferred"`
	ThroughputBPS    int64   `json:"throughput_bps"`
	Percent          float64 `json:"percent"`
	Status           string  `json:"status"` // DiskStatus
}

// ProgressPoller polls VMA for progress updates and coordinates with progress updater
type ProgressPoller struct {
	vmaBaseURL        string
	httpClient        *http.Client
	progressUpdater   *ProgressUpdater
	tracker           *joblog.Tracker
	pollInterval      time.Duration
	timeoutDuration   time.Duration
	activeJobs        map[string]*PollerJob
}

// PollerJob represents an active job being polled
type PollerJob struct {
	JobID           string
	LastSuccessPoll time.Time
	CancelFunc      context.CancelFunc
	ContextDone     <-chan struct{}
}

// NewProgressPoller creates a new progress poller instance
func NewProgressPoller(vmaBaseURL string, progressUpdater *ProgressUpdater, tracker *joblog.Tracker) *ProgressPoller {
	return &ProgressPoller{
		vmaBaseURL:      vmaBaseURL,
		httpClient:      &http.Client{Timeout: 10 * time.Second},
		progressUpdater: progressUpdater,
		tracker:         tracker,
		pollInterval:    2 * time.Second,
		timeoutDuration: 5 * time.Minute,
		activeJobs:      make(map[string]*PollerJob),
	}
}

// StartPolling starts polling progress for a specific job
func (p *ProgressPoller) StartPolling(ctx context.Context, jobID string) error {
	// Check if job is already being polled
	if _, exists := p.activeJobs[jobID]; exists {
		return fmt.Errorf("job %s is already being polled", jobID)
	}

	// Create cancellable context for this job
	jobCtx, cancelFunc := context.WithCancel(ctx)

	// Create poller job
	pollerJob := &PollerJob{
		JobID:           jobID,
		LastSuccessPoll: time.Now(),
		CancelFunc:      cancelFunc,
		ContextDone:     jobCtx.Done(),
	}

	p.activeJobs[jobID] = pollerJob

	log.WithField("job_id", jobID).Info("🔄 Starting progress polling for replication job")

	// Start polling goroutine
	go p.pollJobProgress(jobCtx, pollerJob)

	return nil
}

// StopPolling stops polling for a specific job
func (p *ProgressPoller) StopPolling(jobID string) {
	if pollerJob, exists := p.activeJobs[jobID]; exists {
		pollerJob.CancelFunc()
		delete(p.activeJobs, jobID)
		log.WithField("job_id", jobID).Info("⏹️ Stopped progress polling for job")
	}
}

// pollJobProgress runs the polling loop for a specific job
func (p *ProgressPoller) pollJobProgress(ctx context.Context, pollerJob *PollerJob) {
	ticker := time.NewTicker(p.pollInterval)
	defer ticker.Stop()

	jobID := pollerJob.JobID
	consecutiveFailures := 0
	maxConsecutiveFailures := 3

	for {
		select {
		case <-ctx.Done():
			log.WithField("job_id", jobID).Debug("Polling context cancelled")
			return

		case <-ticker.C:
			// Fetch progress from VMA
			progress, err := p.fetchProgressFromVMA(ctx, jobID)
			if err != nil {
				consecutiveFailures++
				log.WithFields(log.Fields{
					"job_id":               jobID,
					"error":                err,
					"consecutive_failures": consecutiveFailures,
				}).Warn("Failed to fetch progress from VMA")

				// Check for communication timeout
				if time.Since(pollerJob.LastSuccessPoll) > p.timeoutDuration {
					log.WithFields(log.Fields{
						"job_id":         jobID,
						"timeout_duration": p.timeoutDuration,
						"last_success":   pollerJob.LastSuccessPoll,
					}).Error("VMA communication timeout exceeded")

					// Mark job as failed due to timeout
					if err := p.progressUpdater.MarkJobAsFailed(ctx, jobID, "VMA communication timeout (>5m)"); err != nil {
						log.WithError(err).Error("Failed to mark job as failed")
					}

					// End joblog tracking with failure
					if p.tracker != nil {
						p.tracker.EndJob(ctx, jobID, joblog.StatusFailed, fmt.Errorf("VMA communication timeout"))
					}

					// Stop polling for this job
					p.StopPolling(jobID)
					return
				}

				// If too many consecutive failures, reduce polling frequency temporarily
				if consecutiveFailures >= maxConsecutiveFailures {
					log.WithField("job_id", jobID).Warn("Too many consecutive failures, reducing poll frequency")
					time.Sleep(10 * time.Second) // Extra delay on persistent failures
				}
				continue
			}

			// Success - reset failure counter and update last success time
			consecutiveFailures = 0
			pollerJob.LastSuccessPoll = time.Now()

			// Process the progress update
			if err := p.processProgressUpdate(ctx, jobID, progress); err != nil {
				log.WithFields(log.Fields{
					"job_id": jobID,
					"error":  err,
				}).Error("Failed to process progress update")
				continue
			}

			// Check if job is completed
			if p.isJobCompleted(progress.Status) {
				log.WithFields(log.Fields{
					"job_id": jobID,
					"status": progress.Status,
					"stage":  progress.Stage,
				}).Info("✅ Job completed, stopping progress polling")

				// End joblog tracking
				if p.tracker != nil {
					status := joblog.StatusCompleted
					if progress.Status == "Failed" {
						status = joblog.StatusFailed
					}
					p.tracker.EndJob(ctx, jobID, status, nil)
				}

				// Stop polling for this job
				p.StopPolling(jobID)
				return
			}
		}
	}
}

// fetchProgressFromVMA fetches progress information from VMA endpoint
func (p *ProgressPoller) fetchProgressFromVMA(ctx context.Context, jobID string) (*VMAProgressResponse, error) {
	url := fmt.Sprintf("%s/api/v1/progress/%s", p.vmaBaseURL, jobID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("job not found on VMA: %s", jobID)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("VMA returned status %d", resp.StatusCode)
	}

	var progress VMAProgressResponse
	if err := json.NewDecoder(resp.Body).Decode(&progress); err != nil {
		return nil, fmt.Errorf("failed to decode VMA response: %w", err)
	}

	return &progress, nil
}

// processProgressUpdate processes a progress update from VMA
func (p *ProgressPoller) processProgressUpdate(ctx context.Context, jobID string, progress *VMAProgressResponse) error {
	// Use progress updater to update database and joblog
	return p.progressUpdater.UpdateFromVMAProgress(ctx, jobID, progress)
}

// isJobCompleted checks if a job status indicates completion
func (p *ProgressPoller) isJobCompleted(status string) bool {
	switch status {
	case "Succeeded", "Failed":
		return true
	default:
		return false
	}
}

// GetActiveJobs returns a list of currently active job IDs being polled
func (p *ProgressPoller) GetActiveJobs() []string {
	var jobIDs []string
	for jobID := range p.activeJobs {
		jobIDs = append(jobIDs, jobID)
	}
	return jobIDs
}

// GetJobStatus returns the polling status for a specific job
func (p *ProgressPoller) GetJobStatus(jobID string) (bool, time.Time) {
	if pollerJob, exists := p.activeJobs[jobID]; exists {
		return true, pollerJob.LastSuccessPoll
	}
	return false, time.Time{}
}
