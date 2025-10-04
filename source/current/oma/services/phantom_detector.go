package services

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/vexxhost/migratekit-oma/database"
	"github.com/vexxhost/migratekit-oma/joblog"
)

// =============================================================================
// PHANTOM JOB DETECTOR - Multi-Factor Detection Logic
// =============================================================================
// IMPROVED APPROACH: Uses VMA API validation + progress stagnation + impossible states
// Replaces simple time-based detection which was flawed for long-running migrations

// PhantomJobDetector provides intelligent phantom job detection
type PhantomJobDetector struct {
	repository     *database.ReplicationJobRepository
	jobTracker     *joblog.Tracker
	vmaAPIEndpoint string
	httpClient     *http.Client
}

// VMAJobStatus represents the status of a job from VMA perspective
type VMAJobStatus string

const (
	VMAJobFound    VMAJobStatus = "found"     // VMA knows about the job
	VMAJobNotFound VMAJobStatus = "not_found" // VMA doesn't know about job = phantom
	VMAJobNoData   VMAJobStatus = "no_data"   // VMA unreachable or no data
	VMAJobError    VMAJobStatus = "error"     // API error occurred
)

// VMAJobResponse represents VMA API response for job status
type VMAJobResponse struct {
	JobID        string    `json:"job_id"`
	Status       string    `json:"status"`
	LastUpdate   time.Time `json:"last_update"`
	ProgressData *struct {
		Percent          float64 `json:"percent"`
		BytesTransferred int64   `json:"bytes_transferred"`
		CurrentOperation string  `json:"current_operation"`
	} `json:"progress_data,omitempty"`
}

// PhantomDetectionResult provides detailed analysis of job status
type PhantomDetectionResult struct {
	JobID            string          `json:"job_id"`
	IsPhantom        bool            `json:"is_phantom"`
	Reason           string          `json:"reason"`
	VMAStatus        VMAJobStatus    `json:"vma_status"`
	ProgressStagnant bool            `json:"progress_stagnant"`
	StateImpossible  bool            `json:"state_impossible"`
	LastUpdate       time.Time       `json:"last_update"`
	TimeSinceUpdate  time.Duration   `json:"time_since_update"`
	VMAData          *VMAJobResponse `json:"vma_data,omitempty"`
}

// PhantomScanSummary provides results of scanning for phantom jobs
type PhantomScanSummary struct {
	ScannedJobs   int                      `json:"scanned_jobs"`
	PhantomJobs   int                      `json:"phantom_jobs"`
	ValidJobs     int                      `json:"valid_jobs"`
	ErrorJobs     int                      `json:"error_jobs"`
	PhantomJobIDs []string                 `json:"phantom_job_ids"`
	Details       []PhantomDetectionResult `json:"details"`
	ScanDuration  time.Duration            `json:"scan_duration"`
}

// NewPhantomJobDetector creates a new phantom job detector
func NewPhantomJobDetector(
	replicationRepo *database.ReplicationJobRepository,
	jobTracker *joblog.Tracker,
	vmaAPIEndpoint string,
) *PhantomJobDetector {
	return &PhantomJobDetector{
		repository:     replicationRepo,
		jobTracker:     jobTracker,
		vmaAPIEndpoint: vmaAPIEndpoint,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// DetectPhantomJob analyzes a single job using multi-factor detection
func (p *PhantomJobDetector) DetectPhantomJob(ctx context.Context, job *database.ReplicationJob) (*PhantomDetectionResult, error) {
	logger := p.jobTracker.Logger(ctx)

	result := &PhantomDetectionResult{
		JobID:           job.ID,
		IsPhantom:       false,
		LastUpdate:      job.UpdatedAt,
		TimeSinceUpdate: time.Since(job.UpdatedAt),
	}

	logger.Info("Analyzing job for phantom detection",
		"job_id", job.ID,
		"status", job.Status,
		"last_update", job.UpdatedAt,
	)

	// TIER 1: VMA API Validation (Most Reliable)
	vmaStatus, vmaData, err := p.checkVMAJobStatus(ctx, job.ID)
	result.VMAStatus = vmaStatus
	result.VMAData = vmaData

	if err != nil {
		logger.Error("VMA API check failed", "error", err, "job_id", job.ID)
		result.VMAStatus = VMAJobError
	}

	// If VMA doesn't know about the job, it's definitely phantom
	if vmaStatus == VMAJobNotFound {
		result.IsPhantom = true
		result.Reason = "VMA API reports job not found - definitive phantom"
		logger.Warn("PHANTOM DETECTED: VMA doesn't know about job", "job_id", job.ID)
		return result, nil
	}

	// TIER 2: Progress Stagnation Detection
	progressStagnant := p.checkProgressStagnation(job, vmaStatus)
	result.ProgressStagnant = progressStagnant

	if progressStagnant {
		result.IsPhantom = true
		result.Reason = "Progress stagnation: no updates >2 hours AND no VMA data"
		logger.Warn("PHANTOM DETECTED: Progress stagnation",
			"job_id", job.ID,
			"last_update", job.UpdatedAt,
			"vma_status", vmaStatus,
		)
		return result, nil
	}

	// TIER 3: Impossible State Detection
	stateImpossible := p.checkImpossibleState(job)
	result.StateImpossible = stateImpossible

	if stateImpossible {
		result.IsPhantom = true
		result.Reason = "Impossible state: claims replicating but zero progress >30min"
		logger.Warn("PHANTOM DETECTED: Impossible state",
			"job_id", job.ID,
			"status", job.Status,
			"progress", job.ProgressPercent,
			"started_at", job.StartedAt,
		)
		return result, nil
	}

	// Job appears legitimate
	result.Reason = "Job appears legitimate - all checks passed"
	logger.Info("Job validation passed - appears legitimate", "job_id", job.ID)
	return result, nil
}

// ScanForPhantomJobs scans all active jobs for phantom detection
func (p *PhantomJobDetector) ScanForPhantomJobs(ctx context.Context) (*PhantomScanSummary, error) {
	startTime := time.Now()

	// Start job tracking for scan operation
	owner := "phantom-detector"
	ctx, scanJobID, err := p.jobTracker.StartJob(ctx, joblog.JobStart{
		JobType:   "phantom-detection",
		Operation: "scan-phantom-jobs",
		Owner:     &owner,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to start phantom scan job tracking: %w", err)
	}

	logger := p.jobTracker.Logger(ctx)
	logger.Info("🔍 Starting phantom job scan")

	summary := &PhantomScanSummary{
		PhantomJobIDs: make([]string, 0),
		Details:       make([]PhantomDetectionResult, 0),
	}

	var activeJobs []database.ReplicationJob
	err = p.jobTracker.RunStep(ctx, scanJobID, "fetch-active-jobs", func(ctx context.Context) error {
		// Get all jobs that claim to be active
		activeJobs, err = p.repository.GetJobsByStatus([]string{"replicating", "provisioning", "pending"})
		if err != nil {
			return fmt.Errorf("failed to fetch active jobs: %w", err)
		}

		logger.Info("Fetched active jobs for scanning", "count", len(activeJobs))
		return nil
	})

	if err != nil {
		p.jobTracker.EndJob(ctx, scanJobID, joblog.StatusFailed, err)
		return nil, err
	}

	summary.ScannedJobs = len(activeJobs)

	// Analyze each job
	err = p.jobTracker.RunStep(ctx, scanJobID, "analyze-jobs", func(ctx context.Context) error {
		for _, job := range activeJobs {
			result, err := p.DetectPhantomJob(ctx, &job)
			if err != nil {
				logger.Error("Failed to analyze job", "error", err, "job_id", job.ID)
				summary.ErrorJobs++
				continue
			}

			summary.Details = append(summary.Details, *result)

			if result.IsPhantom {
				summary.PhantomJobs++
				summary.PhantomJobIDs = append(summary.PhantomJobIDs, job.ID)
			} else {
				summary.ValidJobs++
			}
		}

		return nil
	})

	if err != nil {
		p.jobTracker.EndJob(ctx, scanJobID, joblog.StatusFailed, err)
		return nil, err
	}

	summary.ScanDuration = time.Since(startTime)

	logger.Info("Phantom job scan completed",
		"scanned", summary.ScannedJobs,
		"phantom", summary.PhantomJobs,
		"valid", summary.ValidJobs,
		"errors", summary.ErrorJobs,
		"duration", summary.ScanDuration,
	)

	p.jobTracker.EndJob(ctx, scanJobID, joblog.StatusCompleted, nil)
	return summary, nil
}

// MarkPhantomJobsAsFailed marks detected phantom jobs as failed and updates VM contexts
func (p *PhantomJobDetector) MarkPhantomJobsAsFailed(ctx context.Context, phantomJobIDs []string) error {
	if len(phantomJobIDs) == 0 {
		return nil
	}

	// Start job tracking for cleanup operation
	owner := "phantom-detector"
	ctx, cleanupJobID, err := p.jobTracker.StartJob(ctx, joblog.JobStart{
		JobType:   "phantom-detection",
		Operation: "mark-phantom-jobs-failed",
		Owner:     &owner,
		Metadata: map[string]interface{}{
			"phantom_job_count": len(phantomJobIDs),
			"phantom_job_ids":   phantomJobIDs,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to start phantom cleanup job tracking: %w", err)
	}

	logger := p.jobTracker.Logger(ctx)
	logger.Info("🧹 Marking phantom jobs as failed", "count", len(phantomJobIDs))

	err = p.jobTracker.RunStep(ctx, cleanupJobID, "mark-jobs-failed", func(ctx context.Context) error {
		for _, jobID := range phantomJobIDs {
			// Update job status to failed
			updates := map[string]interface{}{
				"status":        "failed",
				"error_message": "Job marked as phantom by automated detection",
				"completed_at":  time.Now(),
				"updated_at":    time.Now(),
			}

			if err := p.repository.UpdateJobFields(jobID, updates); err != nil {
				logger.Error("Failed to mark phantom job as failed",
					"error", err,
					"job_id", jobID,
				)
				continue
			}

			// Trigger VM context update
			if err := p.repository.UpdateVMContextAfterJobCompletion(jobID); err != nil {
				logger.Error("Failed to update VM context after phantom cleanup",
					"error", err,
					"job_id", jobID,
				)
			}

			logger.Info("Marked phantom job as failed", "job_id", jobID)
		}

		return nil
	})

	if err != nil {
		p.jobTracker.EndJob(ctx, cleanupJobID, joblog.StatusFailed, err)
		return err
	}

	p.jobTracker.EndJob(ctx, cleanupJobID, joblog.StatusCompleted, nil)
	return nil
}

// checkVMAJobStatus queries VMA API to verify job existence
func (p *PhantomJobDetector) checkVMAJobStatus(ctx context.Context, jobID string) (VMAJobStatus, *VMAJobResponse, error) {
	if p.vmaAPIEndpoint == "" {
		return VMAJobNoData, nil, fmt.Errorf("VMA API endpoint not configured")
	}

	// Build VMA API URL for job status
	url := fmt.Sprintf("%s/api/v1/jobs/%s/status", p.vmaAPIEndpoint, jobID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return VMAJobError, nil, fmt.Errorf("failed to create VMA request: %w", err)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return VMAJobNoData, nil, fmt.Errorf("VMA API request failed: %w", err)
	}
	defer resp.Body.Close()

	// Handle different response codes
	switch resp.StatusCode {
	case http.StatusOK:
		var vmaResp VMAJobResponse
		if err := json.NewDecoder(resp.Body).Decode(&vmaResp); err != nil {
			return VMAJobError, nil, fmt.Errorf("failed to decode VMA response: %w", err)
		}
		return VMAJobFound, &vmaResp, nil

	case http.StatusNotFound:
		// VMA doesn't know about this job = phantom
		return VMAJobNotFound, nil, nil

	default:
		return VMAJobError, nil, fmt.Errorf("VMA API returned status %d", resp.StatusCode)
	}
}

// checkProgressStagnation detects stagnant progress combined with no VMA data
func (p *PhantomJobDetector) checkProgressStagnation(job *database.ReplicationJob, vmaStatus VMAJobStatus) bool {
	// Only consider stagnant if no updates >2 hours AND no VMA data
	return time.Since(job.UpdatedAt) > 2*time.Hour && vmaStatus == VMAJobNoData
}

// checkImpossibleState detects logically impossible job states
func (p *PhantomJobDetector) checkImpossibleState(job *database.ReplicationJob) bool {
	// Job claims to be replicating but has made zero progress for >30 minutes
	if job.Status == "replicating" &&
		job.ProgressPercent == 0 &&
		job.StartedAt != nil &&
		time.Since(*job.StartedAt) > 30*time.Minute {
		return true
	}

	// Add more impossible state checks here as needed
	return false
}
