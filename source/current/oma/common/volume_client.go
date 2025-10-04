package common

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

// VolumeClient provides a simple interface to the Volume Management Daemon
type VolumeClient struct {
	baseURL string
	client  *http.Client
}

// CreateVolumeRequest represents a volume creation request
type CreateVolumeRequest struct {
	Name           string            `json:"name"`
	Size           int64             `json:"size"`
	DiskOfferingID string            `json:"disk_offering_id"`
	ZoneID         string            `json:"zone_id"`
	Metadata       map[string]string `json:"metadata,omitempty"`
}

// AttachVolumeRequest represents a volume attachment request
type AttachVolumeRequest struct {
	VMID string `json:"vm_id"`
}

// AttachVolumeWithContextRequest represents an enhanced volume attachment request with persistent naming
type AttachVolumeWithContextRequest struct {
	VMID                  string `json:"vm_id"`
	VMName                string `json:"vm_name,omitempty"`
	DiskID                string `json:"disk_id,omitempty"`
	RequestPersistentName bool   `json:"request_persistent_name,omitempty"`
}

// VolumeOperation represents a volume operation response
type VolumeOperation struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Status      string                 `json:"status"`
	VolumeID    string                 `json:"volume_id"`
	VMID        *string                `json:"vm_id"`
	Request     map[string]interface{} `json:"request"`
	Response    map[string]interface{} `json:"response"`
	Error       string                 `json:"error"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	CompletedAt *time.Time             `json:"completed_at"`
}

// DeviceMapping represents a volume-to-device mapping
type DeviceMapping struct {
	ID              string    `json:"id"`
	VolumeID        string    `json:"volume_id"`
	VMID            string    `json:"vm_id"`
	DevicePath      string    `json:"device_path"`
	CloudStackState string    `json:"cloudstack_state"`
	LinuxState      string    `json:"linux_state"`
	Size            int64     `json:"size"`
	LastSync        time.Time `json:"last_sync"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// CreateNBDExportRequest represents an NBD export creation request
type CreateNBDExportRequest struct {
	VolumeID   string `json:"volume_id"`
	VMName     string `json:"vm_name"`
	VMID       string `json:"vm_id"`
	DiskNumber int    `json:"disk_number"`
}

// NBDExportInfo represents NBD export information
type NBDExportInfo struct {
	ID         string            `json:"id"`
	VolumeID   string            `json:"volume_id"`
	ExportName string            `json:"export_name"`
	DevicePath string            `json:"device_path"`
	Port       int               `json:"port"`
	Status     string            `json:"status"`
	CreatedAt  time.Time         `json:"created_at"`
	UpdatedAt  time.Time         `json:"updated_at"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

// NewVolumeClient creates a new volume client
func NewVolumeClient(baseURL string) *VolumeClient {
	if baseURL == "" {
		baseURL = "http://localhost:8090"
	}

	return &VolumeClient{
		baseURL: baseURL,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// CreateVolume creates a new volume via the daemon
func (vc *VolumeClient) CreateVolume(ctx context.Context, req CreateVolumeRequest) (*VolumeOperation, error) {
	return vc.doVolumeOperation(ctx, "POST", "/api/v1/volumes", req)
}

// AttachVolume attaches a volume to a VM via the daemon
func (vc *VolumeClient) AttachVolume(ctx context.Context, volumeID, vmID string) (*VolumeOperation, error) {
	req := AttachVolumeRequest{VMID: vmID}
	endpoint := fmt.Sprintf("/api/v1/volumes/%s/attach", volumeID)
	return vc.doVolumeOperation(ctx, "POST", endpoint, req)
}

// AttachVolumeAsRoot attaches a volume to a VM as root disk (device ID 0) via the daemon
func (vc *VolumeClient) AttachVolumeAsRoot(ctx context.Context, volumeID, vmID string) (*VolumeOperation, error) {
	log.WithFields(log.Fields{
		"volume_id": volumeID,
		"vm_id":     vmID,
		"device_id": 0,
	}).Debug("Attaching volume as root disk via Volume Daemon")

	req := AttachVolumeRequest{VMID: vmID}
	endpoint := fmt.Sprintf("/api/v1/volumes/%s/attach-root", volumeID)
	return vc.doVolumeOperation(ctx, "POST", endpoint, req)
}

// DetachVolume detaches a volume via the daemon
func (vc *VolumeClient) DetachVolume(ctx context.Context, volumeID string) (*VolumeOperation, error) {
	endpoint := fmt.Sprintf("/api/v1/volumes/%s/detach", volumeID)
	return vc.doVolumeOperation(ctx, "POST", endpoint, nil)
}

// CleanupTestFailover performs complete test failover cleanup via the daemon
func (vc *VolumeClient) CleanupTestFailover(ctx context.Context, testVMID, volumeID, omaVMID string, deleteVM bool) (*VolumeOperation, error) {
	log.WithFields(log.Fields{
		"test_vm_id": testVMID,
		"volume_id":  volumeID,
		"oma_vm_id":  omaVMID,
		"delete_vm":  deleteVM,
	}).Debug("Starting test failover cleanup via Volume Daemon")

	req := map[string]interface{}{
		"test_vm_id":  testVMID,
		"volume_id":   volumeID,
		"oma_vm_id":   omaVMID,
		"delete_vm":   deleteVM,
		"force_clean": false,
	}

	endpoint := "/api/v1/cleanup/test-failover"
	return vc.doVolumeOperation(ctx, "POST", endpoint, req)
}

// DeleteVolume deletes a volume via the daemon
func (vc *VolumeClient) DeleteVolume(ctx context.Context, volumeID string) (*VolumeOperation, error) {
	endpoint := fmt.Sprintf("/api/v1/volumes/%s", volumeID)
	return vc.doVolumeOperation(ctx, "DELETE", endpoint, nil)
}

// GetOperation gets the status of an operation
func (vc *VolumeClient) GetOperation(ctx context.Context, operationID string) (*VolumeOperation, error) {
	endpoint := fmt.Sprintf("/api/v1/operations/%s", operationID)

	resp, err := vc.doRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, vc.handleErrorResponse(resp)
	}

	var operation VolumeOperation
	if err := json.NewDecoder(resp.Body).Decode(&operation); err != nil {
		return nil, fmt.Errorf("failed to decode operation: %w", err)
	}

	return &operation, nil
}

// GetVolumeDevice gets the device mapping for a volume
func (vc *VolumeClient) GetVolumeDevice(ctx context.Context, volumeID string) (*DeviceMapping, error) {
	endpoint := fmt.Sprintf("/api/v1/volumes/%s/device", volumeID)

	resp, err := vc.doRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, vc.handleErrorResponse(resp)
	}

	var mapping DeviceMapping
	if err := json.NewDecoder(resp.Body).Decode(&mapping); err != nil {
		return nil, fmt.Errorf("failed to decode device mapping: %w", err)
	}

	return &mapping, nil
}

// VolumeMapping represents a volume attached to a VM
type VolumeMapping struct {
	VolumeID   string `json:"volume_id"`
	VMID       string `json:"vm_id"`
	DevicePath string `json:"device_path"`
	AttachedAt string `json:"attached_at"`
}

// ListVolumes lists all volumes attached to a specific VM
func (vc *VolumeClient) ListVolumes(ctx context.Context, vmID string) ([]VolumeMapping, error) {
	log.WithField("vm_id", vmID).Debug("Listing volumes for VM via Volume Daemon")

	endpoint := fmt.Sprintf("/api/v1/vms/%s/volumes", vmID)
	resp, err := vc.doRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list volumes for VM %s: %w", vmID, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		// VM has no volumes attached
		return []VolumeMapping{}, nil
	}

	if resp.StatusCode != http.StatusOK {
		return nil, vc.handleErrorResponse(resp)
	}

	var volumes []VolumeMapping
	if err := json.NewDecoder(resp.Body).Decode(&volumes); err != nil {
		return nil, fmt.Errorf("failed to decode volume list: %w", err)
	}

	log.WithFields(log.Fields{
		"vm_id":        vmID,
		"volume_count": len(volumes),
	}).Debug("Successfully retrieved VM volumes from daemon")

	return volumes, nil
}

// WaitForCompletion waits for an operation to complete
func (vc *VolumeClient) WaitForCompletion(ctx context.Context, operationID string) (*VolumeOperation, error) {
	return vc.WaitForCompletionWithTimeout(ctx, operationID, 5*time.Minute)
}

// WaitForCompletionWithTimeout waits for an operation to complete with a timeout
func (vc *VolumeClient) WaitForCompletionWithTimeout(ctx context.Context, operationID string, timeout time.Duration) (*VolumeOperation, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	log.WithFields(log.Fields{
		"operation_id": operationID,
		"timeout":      timeout,
	}).Info("Waiting for volume operation completion")

	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("operation timeout or cancelled: %w", ctx.Err())
		case <-ticker.C:
			op, err := vc.GetOperation(ctx, operationID)
			if err != nil {
				log.WithError(err).Warn("Failed to get operation status, retrying")
				continue
			}

			log.WithFields(log.Fields{
				"operation_id": operationID,
				"status":       op.Status,
			}).Debug("Operation status check")

			switch op.Status {
			case "completed":
				log.WithFields(log.Fields{
					"operation_id": operationID,
					"duration":     time.Since(op.CreatedAt),
				}).Info("Volume operation completed successfully")
				return op, nil
			case "failed":
				return op, fmt.Errorf("operation failed: %s", op.Error)
			case "cancelled":
				return op, fmt.Errorf("operation was cancelled")
			}
			// Continue waiting for pending/executing
		}
	}
}

// HealthCheck checks if the volume daemon is healthy
func (vc *VolumeClient) HealthCheck(ctx context.Context) error {
	resp, err := vc.doRequest(ctx, "GET", "/health", nil)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("volume daemon unhealthy: status %d", resp.StatusCode)
	}

	return nil
}

// Helper methods

func (vc *VolumeClient) doVolumeOperation(ctx context.Context, method, endpoint string, body interface{}) (*VolumeOperation, error) {
	resp, err := vc.doRequest(ctx, method, endpoint, body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 201 {
		return nil, vc.handleErrorResponse(resp)
	}

	var operation VolumeOperation
	if err := json.NewDecoder(resp.Body).Decode(&operation); err != nil {
		return nil, fmt.Errorf("failed to decode operation response: %w", err)
	}

	return &operation, nil
}

func (vc *VolumeClient) doRequest(ctx context.Context, method, endpoint string, body interface{}) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewBuffer(jsonData)
	}

	url := vc.baseURL + endpoint
	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	log.WithFields(log.Fields{
		"method":   method,
		"endpoint": endpoint,
	}).Debug("Making volume daemon API request")

	resp, err := vc.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}

	return resp, nil
}

func (vc *VolumeClient) handleErrorResponse(resp *http.Response) error {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("volume operation failed with status %d", resp.StatusCode)
	}

	var errorResp struct {
		Error   string      `json:"error"`
		Code    string      `json:"code"`
		Details interface{} `json:"details"`
	}

	if err := json.Unmarshal(body, &errorResp); err != nil {
		return fmt.Errorf("volume operation failed with status %d: %s", resp.StatusCode, string(body))
	}

	return fmt.Errorf("volume operation failed (%s): %s", errorResp.Code, errorResp.Error)
}

// CreateNBDExport creates a new NBD export for a volume via the Volume Daemon
func (vc *VolumeClient) CreateNBDExport(ctx context.Context, req CreateNBDExportRequest) (*NBDExportInfo, error) {
	log.WithFields(log.Fields{
		"volume_id":   req.VolumeID,
		"vm_name":     req.VMName,
		"vm_id":       req.VMID,
		"disk_number": req.DiskNumber,
	}).Debug("Creating NBD export via Volume Daemon")

	resp, err := vc.doRequest(ctx, "POST", "/api/v1/exports", req)
	if err != nil {
		return nil, fmt.Errorf("failed to create NBD export: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, vc.handleErrorResponse(resp)
	}

	var exportInfo NBDExportInfo
	if err := json.NewDecoder(resp.Body).Decode(&exportInfo); err != nil {
		return nil, fmt.Errorf("failed to parse NBD export response: %w", err)
	}

	log.WithFields(log.Fields{
		"export_name": exportInfo.ExportName,
		"device_path": exportInfo.DevicePath,
		"port":        exportInfo.Port,
	}).Info("✅ NBD export created via Volume Daemon")

	return &exportInfo, nil
}

// DeleteNBDExport deletes an NBD export for a volume via the Volume Daemon
func (vc *VolumeClient) DeleteNBDExport(ctx context.Context, volumeID string) error {
	log.WithField("volume_id", volumeID).Debug("Deleting NBD export via Volume Daemon")

	resp, err := vc.doRequest(ctx, "DELETE", fmt.Sprintf("/api/v1/exports/%s", volumeID), nil)
	if err != nil {
		return fmt.Errorf("failed to delete NBD export: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return vc.handleErrorResponse(resp)
	}

	log.WithField("volume_id", volumeID).Info("✅ NBD export deleted via Volume Daemon")
	return nil
}

// GetNBDExport retrieves NBD export information for a volume via the Volume Daemon
func (vc *VolumeClient) GetNBDExport(ctx context.Context, volumeID string) (*NBDExportInfo, error) {
	log.WithField("volume_id", volumeID).Debug("Getting NBD export info via Volume Daemon")

	resp, err := vc.doRequest(ctx, "GET", fmt.Sprintf("/api/v1/exports/%s", volumeID), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get NBD export: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("NBD export not found for volume: %s", volumeID)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, vc.handleErrorResponse(resp)
	}

	var exportInfo NBDExportInfo
	if err := json.NewDecoder(resp.Body).Decode(&exportInfo); err != nil {
		return nil, fmt.Errorf("failed to parse NBD export response: %w", err)
	}

	return &exportInfo, nil
}

// ListNBDExports lists NBD exports with optional filtering via the Volume Daemon
func (vc *VolumeClient) ListNBDExports(ctx context.Context, vmName *string, status *string) ([]*NBDExportInfo, error) {
	endpoint := "/api/v1/exports"

	// Build query parameters
	params := make([]string, 0)
	if vmName != nil {
		params = append(params, fmt.Sprintf("vm_name=%s", *vmName))
	}
	if status != nil {
		params = append(params, fmt.Sprintf("status=%s", *status))
	}

	if len(params) > 0 {
		endpoint += "?" + strings.Join(params, "&")
	}

	log.WithFields(log.Fields{
		"vm_name": vmName,
		"status":  status,
	}).Debug("Listing NBD exports via Volume Daemon")

	resp, err := vc.doRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list NBD exports: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, vc.handleErrorResponse(resp)
	}

	var exports []*NBDExportInfo
	if err := json.NewDecoder(resp.Body).Decode(&exports); err != nil {
		return nil, fmt.Errorf("failed to parse NBD exports response: %w", err)
	}

	return exports, nil
}
