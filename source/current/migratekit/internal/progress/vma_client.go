package progress

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	log "github.com/sirupsen/logrus"
)

// VMAProgressClient sends progress updates to VMA API
type VMAProgressClient struct {
	jobID      string
	baseURL    string
	httpClient *http.Client
	enabled    bool
}

// VMAProgressUpdate represents progress data sent to VMA
type VMAProgressUpdate struct {
	Stage            string  `json:"stage"`
	Status           string  `json:"status,omitempty"`
	BytesTransferred int64   `json:"bytes_transferred"`
	TotalBytes       int64   `json:"total_bytes,omitempty"`
	ThroughputBPS    int64   `json:"throughput_bps"`
	Percent          float64 `json:"percent,omitempty"`
	DiskID           string  `json:"disk_id,omitempty"`
	SyncType         string  `json:"sync_type,omitempty"`     // 🎯 FIX: Sync type (full/incremental)
	ErrorMessage     string  `json:"error_message,omitempty"` // Error details for failed operations
}

// NewVMAProgressClient creates a new VMA progress client
// Reads MIGRATEKIT_PROGRESS_JOB_ID environment variable to enable progress tracking
func NewVMAProgressClient() *VMAProgressClient {
	jobID := os.Getenv("MIGRATEKIT_PROGRESS_JOB_ID")
	if jobID == "" {
		log.Debug("MIGRATEKIT_PROGRESS_JOB_ID not set - progress tracking disabled")
		return &VMAProgressClient{enabled: false}
	}

	// VMA API URL - use localhost:8081 (VMA runs on same machine as migratekit)
	baseURL := "http://localhost:8081"

	log.WithFields(log.Fields{
		"job_id":  jobID,
		"vma_url": baseURL,
	}).Info("🎯 VMA progress tracking enabled")

	return &VMAProgressClient{
		jobID:   jobID,
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 5 * time.Second, // Short timeout to avoid blocking data transfer
		},
		enabled: true,
	}
}

// SendUpdate sends a progress update to VMA API
func (vpc *VMAProgressClient) SendUpdate(update VMAProgressUpdate) error {
	if !vpc.enabled {
		return nil // Progress tracking disabled
	}

	url := fmt.Sprintf("%s/api/v1/progress/%s/update", vpc.baseURL, vpc.jobID)

	updateJSON, err := json.Marshal(update)
	if err != nil {
		log.WithError(err).Warn("Failed to marshal progress update")
		return nil // Don't fail the migration for progress tracking errors
	}

	log.WithFields(log.Fields{
		"job_id":            vpc.jobID,
		"stage":             update.Stage,
		"bytes_transferred": update.BytesTransferred,
		"throughput_bps":    update.ThroughputBPS,
		"percent":           update.Percent,
	}).Debug("Sending progress update to VMA")

	resp, err := vpc.httpClient.Post(url, "application/json", bytes.NewReader(updateJSON))
	if err != nil {
		log.WithError(err).Warn("Failed to send progress update to VMA - continuing migration")
		return nil // Don't fail the migration for progress tracking errors
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.WithFields(log.Fields{
			"status_code": resp.StatusCode,
			"job_id":      vpc.jobID,
		}).Warn("VMA progress API returned non-200 status - continuing migration")
		return nil // Don't fail the migration for progress tracking errors
	}

	log.WithField("job_id", vpc.jobID).Debug("Progress update sent successfully to VMA")
	return nil
}

// IsEnabled returns true if progress tracking is enabled
func (vpc *VMAProgressClient) IsEnabled() bool {
	return vpc.enabled
}

// GetJobID returns the current job ID
func (vpc *VMAProgressClient) GetJobID() string {
	return vpc.jobID
}

// SendStageUpdate sends a stage progress update with percentage
func (vpc *VMAProgressClient) SendStageUpdate(stage string, percent float64) error {
	return vpc.SendUpdate(VMAProgressUpdate{
		Stage:   stage,
		Status:  "in_progress",
		Percent: percent,
	})
}

// SendErrorUpdate sends an error update with error message
func (vpc *VMAProgressClient) SendErrorUpdate(stage string, errorMsg string) error {
	return vpc.SendUpdate(VMAProgressUpdate{
		Stage:        stage,
		Status:       "failed",
		ErrorMessage: errorMsg,
	})
}

// SendCompletedUpdate sends a completion update for a stage
func (vpc *VMAProgressClient) SendCompletedUpdate(stage string, percent float64) error {
	return vpc.SendUpdate(VMAProgressUpdate{
		Stage:   stage,
		Status:  "completed",
		Percent: percent,
	})
}
