// Package vma provides VMA API client for CBT management
package vma

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	log "github.com/sirupsen/logrus"
)

// Client provides VMA API communication for CBT operations
type Client struct {
	baseURL string
	client  *http.Client
}

// EnableCBTRequest represents a CBT enablement request to VMA API
type EnableCBTRequest struct {
	VCenter    string `json:"vcenter"`
	Username   string `json:"username"`
	Password   string `json:"password"`
	Datacenter string `json:"datacenter"`
}

// CBTStatusResponse represents VMA API CBT status response
type CBTStatusResponse struct {
	Enabled    bool   `json:"enabled"`
	VMName     string `json:"vm_name"`
	PowerState string `json:"power_state"`
	VMPath     string `json:"vm_path"`
}

// NewClient creates a new VMA API client
func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		client: &http.Client{
			Timeout: 60 * time.Second, // CBT enablement can take time
		},
	}
}

// EnableCBT enables CBT on a VM via VMA API
func (c *Client) EnableCBT(ctx context.Context, vmPath, vcenter, username, password, datacenter string) error {
	log.WithFields(log.Fields{
		"vm_path": vmPath,
		"vcenter": vcenter,
	}).Info("🔧 Enabling CBT via VMA API")

	// Don't URL encode the VM path - VMA API expects raw paths
	// The route pattern {vm_path:.*} handles paths with slashes
	apiURL := fmt.Sprintf("%s/api/v1/vms%s/enable-cbt", c.baseURL, vmPath)

	// Prepare request body
	reqBody := EnableCBTRequest{
		VCenter:    vcenter,
		Username:   username,
		Password:   password,
		Datacenter: datacenter,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal CBT enable request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create CBT enable request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	log.WithFields(log.Fields{
		"url":     apiURL,
		"method":  "POST",
		"vm_path": vmPath,
	}).Debug("Sending CBT enable request to VMA API")

	// Execute request
	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to call VMA CBT enable API: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("VMA CBT enable API returned status %d", resp.StatusCode)
	}

	log.WithFields(log.Fields{
		"vm_path": vmPath,
		"vcenter": vcenter,
		"status":  resp.StatusCode,
	}).Info("✅ CBT enabled successfully via VMA API")

	return nil
}

// CheckCBTStatus checks if CBT is enabled on a VM via VMA API
func (c *Client) CheckCBTStatus(ctx context.Context, vmPath, vcenter, username, password string) (*CBTStatusResponse, error) {
	log.WithFields(log.Fields{
		"vm_path": vmPath,
		"vcenter": vcenter,
	}).Debug("🔍 Checking CBT status via VMA API")

	// Don't URL encode the VM path - VMA API expects raw paths
	apiURL := fmt.Sprintf("%s/api/v1/vms%s/cbt-status", c.baseURL, vmPath)

	// Add query parameters
	params := url.Values{}
	params.Add("vcenter", vcenter)
	params.Add("username", username)
	params.Add("password", password)

	fullURL := fmt.Sprintf("%s?%s", apiURL, params.Encode())

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "GET", fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create CBT status request: %w", err)
	}

	// Execute request
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call VMA CBT status API: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("VMA CBT status API returned status %d", resp.StatusCode)
	}

	// Parse response
	var statusResp CBTStatusResponse
	err = json.NewDecoder(resp.Body).Decode(&statusResp)
	if err != nil {
		return nil, fmt.Errorf("failed to decode VMA CBT status response: %w", err)
	}

	log.WithFields(log.Fields{
		"vm_path":     vmPath,
		"cbt_enabled": statusResp.Enabled,
		"vm_name":     statusResp.VMName,
		"power_state": statusResp.PowerState,
	}).Debug("✅ CBT status retrieved from VMA API")

	return &statusResp, nil
}
