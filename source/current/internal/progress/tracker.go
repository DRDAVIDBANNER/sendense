package progress

import (
	"fmt"
	"net/http"
	"sync"
	"time"
	"bytes"
	"encoding/json"
)

// ReplicationStage represents the current stage of replication
type ReplicationStage string

const (
	StageDiscover          ReplicationStage = "Discover"
	StageEnableCBT         ReplicationStage = "EnableCBT"
	StageQueryCBT          ReplicationStage = "QueryCBT"
	StageSnapshot          ReplicationStage = "Snapshot"
	StagePrepareVolumes    ReplicationStage = "PrepareVolumes"
	StageStartExports      ReplicationStage = "StartExports"
	StageTransfer          ReplicationStage = "Transfer"
	StageFinalize          ReplicationStage = "Finalize"
	StagePersistChangeIDs  ReplicationStage = "PersistChangeIDs"
)

// DataTracker tracks real data transfer with libnbd integration
type DataTracker struct {
	jobID           string
	totalPlannedBytes int64
	bytesTransferred  int64
	currentStage      ReplicationStage
	startTime         time.Time
	lastUpdateTime    time.Time
	transferRate      float64 // bytes per second
	progressEndpoint  string  // VMA progress endpoint URL
	
	mu              sync.RWMutex
	recentTransfers []transferSample
}

type transferSample struct {
	timestamp time.Time
	bytes     int64
}

// NewDataTracker creates a new progress tracker for libnbd operations
func NewDataTracker(jobID string, plannedBytes int64, progressEndpoint string) *DataTracker {
	return &DataTracker{
		jobID:             jobID,
		totalPlannedBytes: plannedBytes,
		currentStage:      StageDiscover,
		startTime:         time.Now(),
		lastUpdateTime:    time.Now(),
		progressEndpoint:  progressEndpoint,
		recentTransfers:   make([]transferSample, 0, 10),
	}
}

// OnDataTransfer is called for each libnbd read/write operation
func (dt *DataTracker) OnDataTransfer(bytes int64) {
	dt.mu.Lock()
	defer dt.mu.Unlock()
	
	dt.bytesTransferred += bytes
	dt.lastUpdateTime = time.Now()
	
	// Track recent transfers for rate calculation
	sample := transferSample{
		timestamp: dt.lastUpdateTime,
		bytes:     bytes,
	}
	dt.recentTransfers = append(dt.recentTransfers, sample)
	
	// Keep only last 10 samples for rolling average
	if len(dt.recentTransfers) > 10 {
		dt.recentTransfers = dt.recentTransfers[1:]
	}
	
	// Calculate transfer rate (last 5 seconds)
	dt.calculateTransferRate()
}

// SetStage updates the current replication stage
func (dt *DataTracker) SetStage(stage ReplicationStage) {
	dt.mu.Lock()
	defer dt.mu.Unlock()
	dt.currentStage = stage
	dt.lastUpdateTime = time.Now()
}

// calculateTransferRate computes current transfer rate
func (dt *DataTracker) calculateTransferRate() {
	if len(dt.recentTransfers) < 2 {
		return
	}
	
	now := time.Now()
	cutoff := now.Add(-5 * time.Second)
	
	var totalBytes int64
	var validSamples []transferSample
	
	for _, sample := range dt.recentTransfers {
		if sample.timestamp.After(cutoff) {
			validSamples = append(validSamples, sample)
			totalBytes += sample.bytes
		}
	}
	
	if len(validSamples) > 0 {
		duration := now.Sub(validSamples[0].timestamp).Seconds()
		if duration > 0 {
			dt.transferRate = float64(totalBytes) / duration
		}
	}
}

// GetProgress returns current progress information
func (dt *DataTracker) GetProgress() ProgressSnapshot {
	dt.mu.RLock()
	defer dt.mu.RUnlock()
	
	var percentComplete float64
	if dt.totalPlannedBytes > 0 {
		percentComplete = float64(dt.bytesTransferred) / float64(dt.totalPlannedBytes) * 100.0
	}
	
	return ProgressSnapshot{
		JobID:             dt.jobID,
		Stage:             dt.currentStage,
		TotalBytes:        dt.totalPlannedBytes,
		BytesTransferred:  dt.bytesTransferred,
		PercentComplete:   percentComplete,
		TransferRate:      dt.transferRate,
		StartedAt:         dt.startTime,
		UpdatedAt:         dt.lastUpdateTime,
	}
}

// UpdateVMAEndpoint sends progress update to VMA progress service
func (dt *DataTracker) UpdateVMAEndpoint() error {
	if dt.progressEndpoint == "" {
		return nil // No endpoint configured
	}
	
	progress := dt.GetProgress()
	
	updateData := map[string]interface{}{
		"stage":             string(progress.Stage),
		"bytes_transferred": progress.BytesTransferred,
		"percent_complete":  progress.PercentComplete,
		"transfer_rate":     progress.TransferRate,
		"updated_at":        progress.UpdatedAt.Format(time.RFC3339),
	}
	
	jsonData, err := json.Marshal(updateData)
	if err != nil {
		return fmt.Errorf("failed to marshal progress data: %v", err)
	}
	
	url := fmt.Sprintf("%s/%s/update", dt.progressEndpoint, dt.jobID)
	
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to update VMA progress: %v", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("VMA progress update failed with status: %d", resp.StatusCode)
	}
	
	return nil
}

// ProgressSnapshot represents a point-in-time progress state
type ProgressSnapshot struct {
	JobID             string            `json:"job_id"`
	Stage             ReplicationStage  `json:"stage"`
	TotalBytes        int64             `json:"total_bytes"`
	BytesTransferred  int64             `json:"bytes_transferred"`
	PercentComplete   float64           `json:"percent_complete"`
	TransferRate      float64           `json:"transfer_rate"`
	StartedAt         time.Time         `json:"started_at"`
	UpdatedAt         time.Time         `json:"updated_at"`
}
