package services

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

// SNAProgressClient handles communication with SNA v1.5.0 progress tracking API
type SNAProgressClient struct {
	baseURL    string
	httpClient *http.Client
}

// SNAProgressResponse represents the actual SNA progress API response structure
type SNAProgressResponse struct {
	JobID            string  `json:"job_id"`
	Status           string  `json:"status"`
	SyncType         string  `json:"sync_type"`
	Phase            string  `json:"phase"`
	Percentage       float64 `json:"percentage"`
	CurrentOperation string  `json:"current_operation"`
	BytesTransferred int64   `json:"bytes_transferred"`
	TotalBytes       int64   `json:"total_bytes"`

	// Throughput data
	Throughput struct {
		CurrentMBps float64 `json:"current_mbps"`
		AverageMBps float64 `json:"average_mbps"`
		PeakMBps    float64 `json:"peak_mbps"`
		LastUpdate  string  `json:"last_update"`
	} `json:"throughput"`

	// Timing information
	Timing struct {
		StartTime      string `json:"start_time"`
		LastUpdate     string `json:"last_update"`
		ElapsedMs      int64  `json:"elapsed_ms"`
		ETASeconds     int    `json:"eta_seconds"`
		PhaseStart     string `json:"phase_start"`
		PhaseElapsedMs int64  `json:"phase_elapsed_ms"`
	} `json:"timing"`

	// VM Information
	VMInfo struct {
		Name          string `json:"name"`
		Path          string `json:"path"`
		DiskSizeGB    int64  `json:"disk_size_gb"`
		DiskSizeBytes int64  `json:"disk_size_bytes"`
		CBTEnabled    bool   `json:"cbt_enabled"`
	} `json:"vm_info"`

	// Phase progression
	Phases []struct {
		Name       string `json:"name"`
		Status     string `json:"status"`
		StartTime  string `json:"start_time"`
		EndTime    string `json:"end_time"`
		DurationMs int64  `json:"duration_ms"`
	} `json:"phases"`

	// Error information
	Errors    []interface{} `json:"errors"`
	LastError interface{}   `json:"last_error"`
}

// SNAProgressError represents SNA API errors
type SNAProgressError struct {
	StatusCode int
	Message    string
	JobID      string
}

func (e *SNAProgressError) Error() string {
	return fmt.Sprintf("SNA progress API error (status %d) for job %s: %s", e.StatusCode, e.JobID, e.Message)
}

// NewVMAProgressClient creates a new SNA progress client
func NewVMAProgressClient(baseURL string) *SNAProgressClient {
	return &SNAProgressClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// GetProgress retrieves progress information for a specific job
func (vpc *SNAProgressClient) GetProgress(jobID string) (*SNAProgressResponse, error) {
	url := fmt.Sprintf("%s/api/v1/progress/%s", vpc.baseURL, jobID)

	log.WithFields(log.Fields{
		"job_id": jobID,
		"url":    url,
	}).Debug("Polling SNA progress API")

	resp, err := vpc.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to call SNA progress API: %w", err)
	}
	defer resp.Body.Close()

	// Handle different status codes
	switch resp.StatusCode {
	case http.StatusOK:
		// Success - parse response
		break
	case http.StatusNotFound:
		return nil, &SNAProgressError{
			StatusCode: resp.StatusCode,
			Message:    "job not found or progress tracking not available",
			JobID:      jobID,
		}
	default:
		body, _ := io.ReadAll(resp.Body)
		return nil, &SNAProgressError{
			StatusCode: resp.StatusCode,
			Message:    string(body),
			JobID:      jobID,
		}
	}

	var progressResp SNAProgressResponse
	if err := json.NewDecoder(resp.Body).Decode(&progressResp); err != nil {
		return nil, fmt.Errorf("failed to decode SNA progress response: %w", err)
	}

	log.WithFields(log.Fields{
		"job_id":           jobID,
		"progress_percent": progressResp.Percentage,
		"current_phase":    progressResp.Phase,
		"throughput_mbps":  progressResp.Throughput.CurrentMBps,
		"sync_type":        progressResp.SyncType,
	}).Debug("Successfully retrieved SNA progress")

	return &progressResp, nil
}

// GetBasicStatus retrieves basic job status (backward compatible endpoint)
func (vpc *SNAProgressClient) GetBasicStatus(jobID string) (map[string]interface{}, error) {
	url := fmt.Sprintf("%s/api/v1/status/%s", vpc.baseURL, jobID)

	log.WithFields(log.Fields{
		"job_id": jobID,
		"url":    url,
	}).Debug("Polling SNA status API")

	resp, err := vpc.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to call SNA status API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, &SNAProgressError{
			StatusCode: resp.StatusCode,
			Message:    string(body),
			JobID:      jobID,
		}
	}

	var statusResp map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&statusResp); err != nil {
		return nil, fmt.Errorf("failed to decode SNA status response: %w", err)
	}

	return statusResp, nil
}

// IsHealthy checks if SNA API is responsive
func (vpc *SNAProgressClient) IsHealthy() bool {
	url := fmt.Sprintf("%s/api/v1/health", vpc.baseURL)

	resp, err := vpc.httpClient.Get(url)
	if err != nil {
		log.WithError(err).Debug("SNA health check failed")
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

// MapVMAPhaseToStatus converts SNA phases to SHA status values
func MapVMAPhaseToStatus(snaPhase string) string {
	switch snaPhase {
	case "Initializing":
		return "initializing"
	case "Snapshot Creation":
		return "snapshotting"
	case "Copying Data":
		return "replicating"
	case "Cleanup":
		return "finalizing"
	case "Completed":
		return "completed"
	case "Error":
		return "failed"
	default:
		return "replicating" // Default fallback
	}
}

// ConvertThroughputToBps converts MB/s to bytes per second
func ConvertThroughputToBps(throughputMBps float64) int64 {
	return int64(throughputMBps * 1048576) // MB/s * 1024 * 1024
}

// SNAProgressUpdate represents progress update data sent to SNA
type SNAProgressUpdate struct {
	Stage            string  `json:"stage"`
	Status           string  `json:"status,omitempty"`
	BytesTransferred int64   `json:"bytes_transferred"`
	TotalBytes       int64   `json:"total_bytes,omitempty"`
	ThroughputBPS    int64   `json:"throughput_bps"`
	Percent          float64 `json:"percent,omitempty"`
	DiskID           string  `json:"disk_id,omitempty"`
}

// UpdateProgress sends progress update to SNA v1.5.0 progress API
func (vpc *SNAProgressClient) UpdateProgress(jobID string, update SNAProgressUpdate) error {
	url := fmt.Sprintf("%s/api/v1/progress/%s/update", vpc.baseURL, jobID)

	updateJSON, err := json.Marshal(update)
	if err != nil {
		return fmt.Errorf("failed to marshal progress update: %w", err)
	}

	log.WithFields(log.Fields{
		"job_id":            jobID,
		"stage":             update.Stage,
		"bytes_transferred": update.BytesTransferred,
		"throughput_bps":    update.ThroughputBPS,
		"percent":           update.Percent,
	}).Debug("Sending progress update to SNA")

	resp, err := vpc.httpClient.Post(url, "application/json", strings.NewReader(string(updateJSON)))
	if err != nil {
		return fmt.Errorf("failed to send progress update to SNA: %w", err)
	}
	defer resp.Body.Close()

	// Handle response
	switch resp.StatusCode {
	case http.StatusOK:
		log.WithField("job_id", jobID).Debug("Progress update sent successfully to SNA")
		return nil
	default:
		body, _ := io.ReadAll(resp.Body)
		return &SNAProgressError{
			StatusCode: resp.StatusCode,
			Message:    string(body),
			JobID:      jobID,
		}
	}
}
