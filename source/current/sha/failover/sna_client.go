// Package failover provides SNA client for power management operations
package failover

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// SNAClientImpl implements SNAClient interface for power management operations
type SNAClientImpl struct {
	snaHost  string // SNA tunnel endpoint (e.g., "http://localhost:9081")
	vcenter  string // vCenter hostname/IP
	username string // vCenter username
	password string // vCenter password
	timeout  time.Duration
}

// PowerManagementRequest represents a VM power management request to SNA
type PowerManagementRequest struct {
	VMID     string `json:"vm_id"`
	VCenter  string `json:"vcenter"`
	Username string `json:"username"`
	Password string `json:"password"`
	Force    bool   `json:"force,omitempty"`
	Timeout  int    `json:"timeout,omitempty"`
}

// PowerManagementResponse represents the response from SNA power management operations
type PowerManagementResponse struct {
	Success         bool   `json:"success"`
	VMID            string `json:"vm_id"`
	PreviousState   string `json:"previous_state"`
	NewState        string `json:"new_state"`
	Operation       string `json:"operation"`
	ShutdownMethod  string `json:"shutdown_method,omitempty"`
	DurationSeconds int    `json:"duration_seconds"`
	Timestamp       string `json:"timestamp"`
	Message         string `json:"message,omitempty"`
}

// PowerStateResponse represents the response for power state queries
type PowerStateResponse struct {
	VMID        string `json:"vm_id"`
	PowerState  string `json:"power_state"`
	ToolsStatus string `json:"tools_status"`
	Timestamp   string `json:"timestamp"`
}

// NewVMAClient creates a new SNA client for power management
func NewVMAClient(snaHost, vcenter, username, password string) *SNAClientImpl {
	return &SNAClientImpl{
		snaHost:  snaHost,
		vcenter:  vcenter,
		username: username,
		password: password,
		timeout:  60 * time.Second, // Default 60 second timeout
	}
}

// NewVMAClientForFailover creates a SNA client for failover operations with VM context credentials
// vCenter credentials are obtained from the VM context and passed to SNA API calls
func NewVMAClientForFailover() *SNAClientImpl {
	return &SNAClientImpl{
		snaHost:  "http://localhost:9081", // SNA tunnel endpoint
		vcenter:  "",                      // Will be set per-operation from VM context
		username: "",                      // Will be set per-operation from VM context
		password: "",                      // Will be set per-operation from VM context
		timeout:  60 * time.Second,
	}
}

// PowerOnSourceVM powers on a source VM via SNA
func (vmc *SNAClientImpl) PowerOnSourceVM(ctx context.Context, vmwareVMID, vcenter, username, password string) error {
	// Create request payload
	request := PowerManagementRequest{
		VMID:     vmwareVMID,
		VCenter:  vcenter,
		Username: username,
		Password: password,
		Timeout:  180, // 3 minute timeout for power-on
	}

	requestBody, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal power-on request: %w", err)
	}

	// Create HTTP request
	url := fmt.Sprintf("%s/api/v1/vm/%s/power-on", vmc.snaHost, vmwareVMID)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(requestBody))
	if err != nil {
		return fmt.Errorf("failed to create power-on request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Make HTTP call with timeout
	client := &http.Client{Timeout: vmc.timeout}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("SNA power-on request failed: %w", err)
	}
	defer resp.Body.Close()

	// Check HTTP status
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("SNA power-on failed with status %d", resp.StatusCode)
	}

	// Parse response
	var response PowerManagementResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return fmt.Errorf("failed to decode power-on response: %w", err)
	}

	// Verify success
	if !response.Success {
		return fmt.Errorf("SNA power-on failed: %s", response.Message)
	}

	// Verify VM is actually powered on
	if response.NewState != "poweredOn" {
		return fmt.Errorf("VM not properly powered on: current state is %s", response.NewState)
	}

	return nil
}

// PowerOffSourceVM powers off a source VM via SNA
func (vmc *SNAClientImpl) PowerOffSourceVM(ctx context.Context, vmwareVMID, vcenter, username, password string) error {
	// Create request payload
	request := PowerManagementRequest{
		VMID:     vmwareVMID,
		VCenter:  vcenter,
		Username: username,
		Password: password,
		Force:    false, // Try graceful shutdown first
		Timeout:  300,   // 5 minute timeout for graceful shutdown
	}

	requestBody, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal power-off request: %w", err)
	}

	// Create HTTP request
	url := fmt.Sprintf("%s/api/v1/vm/%s/power-off", vmc.snaHost, vmwareVMID)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(requestBody))
	if err != nil {
		return fmt.Errorf("failed to create power-off request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Make HTTP call with timeout
	client := &http.Client{Timeout: vmc.timeout}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("SNA power-off request failed: %w", err)
	}
	defer resp.Body.Close()

	// Check HTTP status
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("SNA power-off failed with status %d", resp.StatusCode)
	}

	// Parse response
	var response PowerManagementResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return fmt.Errorf("failed to decode power-off response: %w", err)
	}

	// Verify success
	if !response.Success {
		return fmt.Errorf("SNA power-off failed: %s", response.Message)
	}

	// For graceful shutdown, wait for VM to actually power down
	// The SNA API returns success when the shutdown command is issued, not when VM is off
	if response.NewState != "poweredOff" {
		// Wait up to 2 minutes for graceful shutdown to complete
		maxWaitTime := 120 * time.Second
		pollInterval := 5 * time.Second
		startTime := time.Now()

		for time.Since(startTime) < maxWaitTime {
			time.Sleep(pollInterval)

			// Check current power state
			currentState, err := vmc.GetVMPowerState(ctx, vmwareVMID, vcenter, username, password)
			if err != nil {
				// If we can't check state, continue waiting
				continue
			}

			if currentState == "poweredOff" {
				return nil // Successfully powered off
			}
		}

		// Timeout reached - VM didn't power off in time
		return fmt.Errorf("VM graceful shutdown timeout: VM still %s after %v", response.NewState, maxWaitTime)
	}

	return nil
}

// GetVMPowerState gets the current power state of a VM via SNA
func (vmc *SNAClientImpl) GetVMPowerState(ctx context.Context, vmwareVMID, vcenter, username, password string) (string, error) {
	// Build query parameters
	params := url.Values{}
	params.Add("vcenter", vcenter)
	params.Add("username", username)
	params.Add("password", password)

	// Create HTTP request
	url := fmt.Sprintf("%s/api/v1/vm/%s/power-state?%s", vmc.snaHost, vmwareVMID, params.Encode())
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "unknown", fmt.Errorf("failed to create power-state request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Make HTTP call with timeout
	client := &http.Client{Timeout: vmc.timeout}
	resp, err := client.Do(req)
	if err != nil {
		return "unknown", fmt.Errorf("SNA power-state request failed: %w", err)
	}
	defer resp.Body.Close()

	// Check HTTP status
	if resp.StatusCode != http.StatusOK {
		return "unknown", fmt.Errorf("SNA power-state query failed with status %d", resp.StatusCode)
	}

	// Parse response
	var response PowerStateResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "unknown", fmt.Errorf("failed to decode power-state response: %w", err)
	}

	return response.PowerState, nil
}

// NullVMAClient implements SNAClient interface with no-op operations
type NullVMAClient struct{}

// NewNullVMAClient creates a null SNA client for fallback
func NewNullVMAClient() *NullVMAClient {
	return &NullVMAClient{}
}

// PowerOnSourceVM is a no-op for null client
func (n *NullVMAClient) PowerOnSourceVM(ctx context.Context, vmwareVMID, vcenter, username, password string) error {
	return fmt.Errorf("SNA client not available - power management disabled")
}

// PowerOffSourceVM is a no-op for null client
func (n *NullVMAClient) PowerOffSourceVM(ctx context.Context, vmwareVMID, vcenter, username, password string) error {
	return fmt.Errorf("SNA client not available - power management disabled")
}

// GetVMPowerState is a no-op for null client
func (n *NullVMAClient) GetVMPowerState(ctx context.Context, vmwareVMID, vcenter, username, password string) (string, error) {
	return "unknown", fmt.Errorf("SNA client not available - power state unavailable")
}
