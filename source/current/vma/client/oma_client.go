// Package client provides the VMA client for communicating with OMA API
// This handles VM inventory submission and replication job management with dynamic port allocation
package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/vexxhost/migratekit-oma/models"
)

// Config holds the configuration for OMA client
type Config struct {
	// OMA API endpoint
	BaseURL string `json:"base_url"`
	// VMA authentication token
	AuthToken string `json:"auth_token"`
	// VMA appliance ID
	ApplianceID string `json:"appliance_id"`
	// HTTP client timeout
	Timeout time.Duration `json:"timeout"`
}

// DefaultConfig returns default OMA client configuration
func DefaultConfig() Config {
	return Config{
		BaseURL:     "http://localhost:8082",
		ApplianceID: "vma-001",
		Timeout:     30 * time.Second,
	}
}

// Client provides OMA API communication capabilities with automatic token renewal
type Client struct {
	config       Config
	httpClient   *http.Client
	sessionToken string
	tokenExpiry  time.Time
	authMutex    sync.RWMutex
}

// NewClient creates a new OMA API client
func NewClient(config Config) *Client {
	return &Client{
		config: config,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
	}
}

// Authenticate authenticates with OMA and gets session token
func (c *Client) Authenticate() error {
	c.authMutex.Lock()
	defer c.authMutex.Unlock()

	authReq := models.AuthRequest{
		ApplianceID: c.config.ApplianceID,
		Token:       c.config.AuthToken,
		Version:     "1.0.0",
	}

	resp, err := c.makeRequest("POST", "/api/v1/auth/login", authReq)
	if err != nil {
		return fmt.Errorf("authentication request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("authentication failed with status %d", resp.StatusCode)
	}

	var authResp models.AuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		return fmt.Errorf("failed to decode auth response: %w", err)
	}

	c.sessionToken = authResp.SessionToken
	c.tokenExpiry = authResp.ExpiresAt
	log.WithFields(log.Fields{
		"expires_at":     authResp.ExpiresAt,
		"valid_duration": time.Until(authResp.ExpiresAt),
	}).Info("Successfully authenticated with OMA")

	return nil
}

// isTokenExpired checks if the current token is expired or will expire soon
func (c *Client) isTokenExpired() bool {
	c.authMutex.RLock()
	defer c.authMutex.RUnlock()

	// Consider token expired if it expires within the next 5 minutes
	return time.Now().Add(5 * time.Minute).After(c.tokenExpiry)
}

// ensureAuthenticated ensures we have a valid token, refreshing if necessary
func (c *Client) ensureAuthenticated() error {
	if c.sessionToken == "" || c.isTokenExpired() {
		log.Info("Token is expired or missing, refreshing authentication")
		return c.Authenticate()
	}
	return nil
}

// SendVMInventory sends VM inventory to OMA
func (c *Client) SendVMInventory(inventory *models.VMInventoryRequest) error {
	resp, err := c.makeAuthenticatedRequest("POST", "/api/v1/vms/inventory", inventory)
	if err != nil {
		return fmt.Errorf("failed to send VM inventory: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("VM inventory submission failed with status %d", resp.StatusCode)
	}

	log.WithField("vm_count", len(inventory.VMs)).Info("Successfully sent VM inventory to OMA")
	return nil
}

// CreateReplicationJob creates a new replication job and returns allocated port info
func (c *Client) CreateReplicationJob(job *models.ReplicationJob) (*models.ReplicationJob, error) {
	resp, err := c.makeAuthenticatedRequest("POST", "/api/v1/replications", job)
	if err != nil {
		return nil, fmt.Errorf("failed to create replication job: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("replication job creation failed with status %d: %s", resp.StatusCode, string(body))
	}

	var apiResp models.APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode job response: %w", err)
	}

	// Extract the replication job from the response
	jobData, err := json.Marshal(apiResp.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal job data: %w", err)
	}

	var createdJob models.ReplicationJob
	if err := json.Unmarshal(jobData, &createdJob); err != nil {
		return nil, fmt.Errorf("failed to unmarshal job data: %w", err)
	}

	log.WithFields(log.Fields{
		"job_id":      createdJob.ID,
		"nbd_port":    createdJob.NBDPort,
		"export_name": createdJob.NBDExportName,
		"device":      createdJob.TargetDevice,
	}).Info("Successfully created replication job with allocated port")

	return &createdJob, nil
}

// UpdateReplicationJob updates a replication job status/progress
func (c *Client) UpdateReplicationJob(job *models.ReplicationJob) error {
	url := fmt.Sprintf("/api/v1/replications/%s", job.ID)
	resp, err := c.makeAuthenticatedRequest("PUT", url, job)
	if err != nil {
		return fmt.Errorf("failed to update replication job: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("replication job update failed with status %d", resp.StatusCode)
	}

	log.WithFields(log.Fields{
		"job_id":   job.ID,
		"status":   job.Status,
		"progress": job.Progress,
	}).Info("Successfully updated replication job")

	return nil
}

// DeleteReplicationJob deletes a replication job and releases allocated resources
func (c *Client) DeleteReplicationJob(jobID string) error {
	url := fmt.Sprintf("/api/v1/replications/%s", jobID)
	resp, err := c.makeAuthenticatedRequest("DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to delete replication job: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("replication job deletion failed with status %d", resp.StatusCode)
	}

	log.WithField("job_id", jobID).Info("Successfully deleted replication job")
	return nil
}

// makeRequest makes an HTTP request without authentication
func (c *Client) makeRequest(method, path string, body interface{}) (*http.Response, error) {
	url := c.config.BaseURL + path

	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(jsonData)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return c.httpClient.Do(req)
}

// makeAuthenticatedRequest makes an HTTP request with authentication header and automatic token renewal
func (c *Client) makeAuthenticatedRequest(method, path string, body interface{}) (*http.Response, error) {
	// Ensure we have a valid token before making the request
	if err := c.ensureAuthenticated(); err != nil {
		return nil, fmt.Errorf("failed to ensure authentication: %w", err)
	}

	// Make the request with current token
	resp, err := c.doAuthenticatedRequest(method, path, body)
	if err != nil {
		return nil, err
	}

	// If we get a 401, try to refresh the token and retry once
	if resp.StatusCode == http.StatusUnauthorized {
		resp.Body.Close()
		log.Warn("Received 401 Unauthorized, attempting to refresh token and retry")

		// Force refresh the token
		if err := c.Authenticate(); err != nil {
			return nil, fmt.Errorf("failed to refresh token after 401: %w", err)
		}

		// Retry the request with the new token
		return c.doAuthenticatedRequest(method, path, body)
	}

	return resp, nil
}

// doAuthenticatedRequest performs the actual HTTP request with current token
func (c *Client) doAuthenticatedRequest(method, path string, body interface{}) (*http.Response, error) {
	c.authMutex.RLock()
	token := c.sessionToken
	c.authMutex.RUnlock()

	if token == "" {
		return nil, fmt.Errorf("not authenticated - no session token available")
	}

	url := c.config.BaseURL + path

	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(jsonData)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add authentication header
	req.Header.Set("Authorization", "Bearer "+token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return c.httpClient.Do(req)
}

// GetPreviousChangeID retrieves the previous change ID for a VM from OMA
func (c *Client) GetPreviousChangeID(vmPath string) (string, error) {
	url := fmt.Sprintf("/api/v1/replications/changeid?vm_path=%s", vmPath)
	resp, err := c.makeAuthenticatedRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to get previous change ID: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("OMA API returned status %d", resp.StatusCode)
	}

	var response map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return response["change_id"], nil
}
