package telemetry

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"
)

// Client sends telemetry updates to SHA
type Client struct {
	shaURL     string
	httpClient *http.Client
}

// NewClient creates a new telemetry client
func NewClient(shaURL string) *Client {
	return &Client{
		shaURL: shaURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// SendBackupUpdate sends a backup telemetry update to SHA
// POST /api/v1/telemetry/backup/{job_id}
func (c *Client) SendBackupUpdate(jobID string, update *TelemetryUpdate) error {
	url := fmt.Sprintf("%s/api/v1/telemetry/backup/%s", c.shaURL, jobID)
	
	log.WithFields(log.Fields{
		"job_id":   jobID,
		"url":      url,
		"progress": update.ProgressPercent,
		"bytes":    update.BytesTransferred,
	}).Info("🌐 Starting HTTP POST to SHA")
	
	// Marshal update to JSON
	jsonData, err := json.Marshal(update)
	if err != nil {
		log.WithError(err).Error("❌ JSON marshal failed")
		return fmt.Errorf("failed to marshal telemetry: %w", err)
	}
	
	log.WithFields(log.Fields{
		"json_size": len(jsonData),
		"job_id":    jobID,
	}).Debug("✅ JSON marshaled successfully")
	
	// Create HTTP request
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		log.WithError(err).Error("❌ HTTP request creation failed")
		return fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("Content-Type", "application/json")
	
	log.WithField("job_id", jobID).Info("🚀 Executing HTTP Do() - request leaving SBC")
	
	// Send request
	startTime := time.Now()
	resp, err := c.httpClient.Do(req)
	duration := time.Since(startTime)
	
	if err != nil {
		log.WithFields(log.Fields{
			"job_id":   jobID,
			"error":    err.Error(),
			"duration": duration.Milliseconds(),
		}).Error("❌ HTTP Do() failed - network error")
		return fmt.Errorf("failed to send telemetry: %w", err)
	}
	defer resp.Body.Close()
	
	log.WithFields(log.Fields{
		"job_id":      jobID,
		"status_code": resp.StatusCode,
		"duration_ms": duration.Milliseconds(),
	}).Info("📥 HTTP response received from SHA")
	
	// Check response
	if resp.StatusCode >= 400 {
		log.WithFields(log.Fields{
			"job_id":      jobID,
			"status_code": resp.StatusCode,
		}).Error("❌ SHA rejected telemetry - HTTP error status")
		return fmt.Errorf("telemetry rejected: status %d", resp.StatusCode)
	}
	
	log.WithFields(log.Fields{
		"job_id":      jobID,
		"progress":    update.ProgressPercent,
		"phase":       update.CurrentPhase,
		"status_code": resp.StatusCode,
	}).Info("✅ Telemetry accepted by SHA - HTTP 200 OK")
	
	return nil
}

