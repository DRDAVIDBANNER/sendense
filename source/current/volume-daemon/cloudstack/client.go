package cloudstack

import (
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/apache/cloudstack-go/cloudstack"
	log "github.com/sirupsen/logrus"

	"github.com/vexxhost/migratekit-volume-daemon/models"
	"github.com/vexxhost/migratekit-volume-daemon/service"
)

// CloudStackConfig represents CloudStack connection configuration
type CloudStackConfig struct {
	APIURL    string `json:"api_url"`
	APIKey    string `json:"api_key"`
	SecretKey string `json:"secret_key"`
	Domain    string `json:"domain"`
	Zone      string `json:"zone"`
}

// Client implements the CloudStackClient interface for volume operations
type Client struct {
	cs     *cloudstack.CloudStackClient
	config CloudStackConfig
}

// NewClient creates a new CloudStack client for volume operations
func NewClient(config CloudStackConfig) service.CloudStackClient {
	// Ensure API URL has the correct path
	apiURL := config.APIURL
	if !strings.HasSuffix(apiURL, "/client/api") {
		apiURL = strings.TrimSuffix(apiURL, "/") + "/client/api"
	}

	log.WithFields(log.Fields{
		"api_url": apiURL,
		"zone":    config.Zone,
		"domain":  config.Domain,
	}).Debug("Creating CloudStack client for volume management")

	cs := cloudstack.NewAsyncClient(apiURL, config.APIKey, config.SecretKey, false)
	cs.HTTPGETOnly = true // Use GET requests for better compatibility

	return &Client{
		cs:     cs,
		config: config,
	}
}

// CreateVolume creates a new volume in CloudStack
func (c *Client) CreateVolume(ctx context.Context, req models.CreateVolumeRequest) (string, error) {
	log.WithFields(log.Fields{
		"name":             req.Name,
		"size":             req.Size,
		"disk_offering_id": req.DiskOfferingID,
		"zone_id":          req.ZoneID,
	}).Info("Creating CloudStack volume")

	// Create volume parameters
	params := c.cs.Volume.NewCreateVolumeParams()
	params.SetName(req.Name)

	// Size in GB (CloudStack expects GB, not bytes)
	sizeGB := req.Size / (1024 * 1024 * 1024)
	if sizeGB == 0 {
		sizeGB = 1 // Minimum 1GB
	}
	params.SetSize(sizeGB)

	if req.DiskOfferingID != "" {
		params.SetDiskofferingid(req.DiskOfferingID)
	}

	if req.ZoneID != "" {
		params.SetZoneid(req.ZoneID)
	} else {
		// Resolve zone name to zone ID
		zoneID, err := c.getZoneID(c.config.Zone)
		if err != nil {
			return "", fmt.Errorf("failed to resolve zone '%s' to zone ID: %w", c.config.Zone, err)
		}
		params.SetZoneid(zoneID)
	}

	// Create the volume
	resp, err := c.cs.Volume.CreateVolume(params)
	if err != nil {
		return "", fmt.Errorf("failed to create volume: %w", err)
	}

	log.WithFields(log.Fields{
		"volume_id": resp.Id,
		"name":      resp.Name,
		"size":      resp.Size,
		"state":     resp.State,
	}).Info("CloudStack volume created successfully")

	return resp.Id, nil
}

// AttachVolume attaches a volume to a VM in CloudStack
func (c *Client) AttachVolume(ctx context.Context, volumeID, vmID string) error {
	log.WithFields(log.Fields{
		"volume_id": volumeID,
		"vm_id":     vmID,
	}).Info("Attaching CloudStack volume to VM")

	params := c.cs.Volume.NewAttachVolumeParams(volumeID, vmID)

	_, err := c.cs.Volume.AttachVolume(params)
	if err != nil {
		return fmt.Errorf("failed to attach volume %s to VM %s: %w", volumeID, vmID, err)
	}

	log.WithFields(log.Fields{
		"volume_id": volumeID,
		"vm_id":     vmID,
	}).Info("CloudStack volume attached successfully")

	return nil
}

// AttachVolumeAsRoot attaches a volume to a VM as root disk (device ID 0)
func (c *Client) AttachVolumeAsRoot(ctx context.Context, volumeID, vmID string) error {
	log.WithFields(log.Fields{
		"volume_id": volumeID,
		"vm_id":     vmID,
		"device_id": 0,
	}).Info("Attaching CloudStack volume to VM as root disk")

	params := c.cs.Volume.NewAttachVolumeParams(volumeID, vmID)
	params.SetDeviceid(0) // Explicitly set device ID 0 for root disk

	_, err := c.cs.Volume.AttachVolume(params)
	if err != nil {
		return fmt.Errorf("failed to attach volume %s to VM %s as root: %w", volumeID, vmID, err)
	}

	log.WithFields(log.Fields{
		"volume_id": volumeID,
		"vm_id":     vmID,
		"device_id": 0,
	}).Info("CloudStack volume attached as root disk successfully")

	return nil
}

// GetVMPowerState gets the current power state of a VM
func (c *Client) GetVMPowerState(ctx context.Context, vmID string) (string, error) {
	log.WithField("vm_id", vmID).Debug("Getting VM power state from CloudStack")

	params := c.cs.VirtualMachine.NewListVirtualMachinesParams()
	params.SetId(vmID)

	resp, err := c.cs.VirtualMachine.ListVirtualMachines(params)
	if err != nil {
		// Check if this is the systemic CloudStack SDK ostypeid parsing issue
		if strings.Contains(err.Error(), "ostypeid") &&
			strings.Contains(err.Error(), "cannot unmarshal string") {
			log.WithError(err).Warn("🚨 CloudStack SDK ostypeid parsing error on GetVMPowerState - attempting direct API workaround")

			// Use direct CloudStack API to get VM state
			return c.getVMPowerStateDirectAPI(ctx, vmID)
		}
		return "", fmt.Errorf("failed to get VM %s: %w", vmID, err)
	}

	if len(resp.VirtualMachines) == 0 {
		return "", fmt.Errorf("VM %s not found", vmID)
	}

	vm := resp.VirtualMachines[0]
	log.WithFields(log.Fields{
		"vm_id": vmID,
		"state": vm.State,
	}).Debug("Retrieved VM power state")

	return vm.State, nil
}

// ValidateVMPoweredOff ensures a VM is powered off before proceeding with cleanup
func (c *Client) ValidateVMPoweredOff(ctx context.Context, vmID string) error {
	log.WithField("vm_id", vmID).Debug("Validating VM is powered off")

	state, err := c.GetVMPowerState(ctx, vmID)
	if err != nil {
		return fmt.Errorf("failed to get VM power state: %w", err)
	}

	if state != "Stopped" {
		return fmt.Errorf("VM %s is not powered off (current state: %s)", vmID, state)
	}

	log.WithField("vm_id", vmID).Info("✅ VM validated as powered off")
	return nil
}

// PowerOffVM forcefully powers off a VM
func (c *Client) PowerOffVM(ctx context.Context, vmID string) error {
	log.WithField("vm_id", vmID).Info("Powering off VM")

	// First check if VM is already stopped
	state, err := c.GetVMPowerState(ctx, vmID)
	if err != nil {
		return fmt.Errorf("failed to check VM state before power off: %w", err)
	}

	if state == "Stopped" {
		log.WithField("vm_id", vmID).Info("VM already powered off")
		return nil
	}

	// Stop the VM
	params := c.cs.VirtualMachine.NewStopVirtualMachineParams(vmID)
	params.SetForced(true) // Force stop for cleanup scenarios

	_, err = c.cs.VirtualMachine.StopVirtualMachine(params)
	if err != nil {
		return fmt.Errorf("failed to power off VM %s: %w", vmID, err)
	}

	log.WithFields(log.Fields{
		"vm_id":     vmID,
		"forced":    true,
		"operation": "cleanup",
	}).Info("✅ VM powered off successfully")

	return nil
}

// DeleteVM deletes a VM from CloudStack
func (c *Client) DeleteVM(ctx context.Context, vmID string) error {
	log.WithField("vm_id", vmID).Info("Deleting VM from CloudStack")

	// Ensure VM is stopped before deletion
	err := c.ValidateVMPoweredOff(ctx, vmID)
	if err != nil {
		log.WithField("vm_id", vmID).Warn("VM not stopped, attempting forced power off before deletion")
		if powerErr := c.PowerOffVM(ctx, vmID); powerErr != nil {
			return fmt.Errorf("failed to power off VM before deletion: %w", powerErr)
		}

		// Wait a moment for state to stabilize
		time.Sleep(2 * time.Second)
	}

	// Delete the VM
	params := c.cs.VirtualMachine.NewDestroyVirtualMachineParams(vmID)
	params.SetExpunge(true) // Expunge immediately for cleanup

	_, err = c.cs.VirtualMachine.DestroyVirtualMachine(params)
	if err != nil {
		return fmt.Errorf("failed to delete VM %s: %w", vmID, err)
	}

	log.WithFields(log.Fields{
		"vm_id":     vmID,
		"expunged":  true,
		"operation": "cleanup",
	}).Info("✅ VM deleted successfully")

	return nil
}

// DetachVolume detaches a volume from its VM in CloudStack
func (c *Client) DetachVolume(ctx context.Context, volumeID string) error {
	log.WithFields(log.Fields{
		"volume_id": volumeID,
	}).Info("Detaching CloudStack volume")

	params := c.cs.Volume.NewDetachVolumeParams()
	params.SetId(volumeID)

	_, err := c.cs.Volume.DetachVolume(params)
	if err != nil {
		return fmt.Errorf("failed to detach volume %s: %w", volumeID, err)
	}

	log.WithFields(log.Fields{
		"volume_id": volumeID,
	}).Info("CloudStack volume detached successfully")

	return nil
}

// DeleteVolume deletes a volume from CloudStack
func (c *Client) DeleteVolume(ctx context.Context, volumeID string) error {
	log.WithFields(log.Fields{
		"volume_id": volumeID,
	}).Info("Deleting CloudStack volume")

	params := c.cs.Volume.NewDeleteVolumeParams(volumeID)

	_, err := c.cs.Volume.DeleteVolume(params)
	if err != nil {
		return fmt.Errorf("failed to delete volume %s: %w", volumeID, err)
	}

	log.WithFields(log.Fields{
		"volume_id": volumeID,
	}).Info("CloudStack volume deleted successfully")

	return nil
}

// GetVolume retrieves volume information from CloudStack
func (c *Client) GetVolume(ctx context.Context, volumeID string) (map[string]interface{}, error) {
	log.WithFields(log.Fields{
		"volume_id": volumeID,
	}).Debug("Retrieving CloudStack volume details")

	params := c.cs.Volume.NewListVolumesParams()
	params.SetId(volumeID)

	resp, err := c.cs.Volume.ListVolumes(params)
	if err != nil {
		return nil, fmt.Errorf("failed to get volume %s: %w", volumeID, err)
	}

	if resp.Count == 0 {
		return nil, fmt.Errorf("volume %s not found", volumeID)
	}

	volume := resp.Volumes[0]

	// Convert to generic map for flexibility
	result := map[string]interface{}{
		"id":               volume.Id,
		"name":             volume.Name,
		"size":             volume.Size,
		"state":            volume.State,
		"type":             volume.Type,
		"zoneid":           volume.Zoneid,
		"zonename":         volume.Zonename,
		"diskofferingid":   volume.Diskofferingid,
		"diskofferingname": volume.Diskofferingname,
		"created":          volume.Created,
		"attached":         volume.Attached,
		"deviceid":         volume.Deviceid,
		"virtualmachineid": volume.Virtualmachineid,
		"storagetype":      volume.Storagetype,
		"provisioningtype": volume.Provisioningtype,
	}

	log.WithFields(log.Fields{
		"volume_id": volumeID,
		"name":      volume.Name,
		"state":     volume.State,
		"size":      volume.Size,
	}).Debug("CloudStack volume details retrieved")

	return result, nil
}

// ListVolumes lists volumes for a VM in CloudStack
func (c *Client) ListVolumes(ctx context.Context, vmID string) ([]map[string]interface{}, error) {
	log.WithFields(log.Fields{
		"vm_id": vmID,
	}).Debug("Listing CloudStack volumes for VM")

	params := c.cs.Volume.NewListVolumesParams()
	if vmID != "" {
		params.SetVirtualmachineid(vmID)
	}

	resp, err := c.cs.Volume.ListVolumes(params)
	if err != nil {
		return nil, fmt.Errorf("failed to list volumes for VM %s: %w", vmID, err)
	}

	var result []map[string]interface{}
	for _, volume := range resp.Volumes {
		volumeData := map[string]interface{}{
			"id":               volume.Id,
			"name":             volume.Name,
			"size":             volume.Size,
			"state":            volume.State,
			"type":             volume.Type,
			"zoneid":           volume.Zoneid,
			"zonename":         volume.Zonename,
			"diskofferingid":   volume.Diskofferingid,
			"diskofferingname": volume.Diskofferingname,
			"created":          volume.Created,
			"attached":         volume.Attached,
			"deviceid":         volume.Deviceid,
			"virtualmachineid": volume.Virtualmachineid,
			"storagetype":      volume.Storagetype,
			"provisioningtype": volume.Provisioningtype,
		}
		result = append(result, volumeData)
	}

	log.WithFields(log.Fields{
		"vm_id":        vmID,
		"volume_count": len(result),
	}).Debug("CloudStack volumes listed")

	return result, nil
}

// Ping tests connectivity to CloudStack API
func (c *Client) Ping(ctx context.Context) error {
	log.Debug("Testing CloudStack API connectivity")

	// Use a simple API call to test connectivity
	params := c.cs.Zone.NewListZonesParams()
	params.SetAvailable(true)

	resp, err := c.cs.Zone.ListZones(params)
	if err != nil {
		return fmt.Errorf("CloudStack API ping failed: %w", err)
	}

	log.WithFields(log.Fields{
		"zone_count":    resp.Count,
		"response_time": "< 1s", // We could add timing if needed
	}).Debug("CloudStack API connectivity test successful")

	return nil
}

// WaitForVolumeState waits for a volume to reach a specific state
func (c *Client) WaitForVolumeState(ctx context.Context, volumeID, targetState string, timeout time.Duration) error {
	log.WithFields(log.Fields{
		"volume_id":    volumeID,
		"target_state": targetState,
		"timeout":      timeout,
	}).Info("Waiting for CloudStack volume state")

	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		volumeData, err := c.GetVolume(ctx, volumeID)
		if err != nil {
			return fmt.Errorf("failed to check volume state: %w", err)
		}

		currentState, ok := volumeData["state"].(string)
		if !ok {
			return fmt.Errorf("invalid volume state format")
		}

		if currentState == targetState {
			log.WithFields(log.Fields{
				"volume_id": volumeID,
				"state":     currentState,
			}).Info("CloudStack volume reached target state")
			return nil
		}

		if currentState == "Error" || currentState == "Failed" {
			return fmt.Errorf("volume %s entered error state: %s", volumeID, currentState)
		}

		log.WithFields(log.Fields{
			"volume_id":     volumeID,
			"current_state": currentState,
			"target_state":  targetState,
		}).Debug("Volume state check - waiting...")

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(5 * time.Second):
			// Continue polling
		}
	}

	return fmt.Errorf("timeout waiting for volume %s to reach state %s", volumeID, targetState)
}

// getVMPowerStateDirectAPI uses direct CloudStack management API to get VM power state
// This bypasses the Go SDK's JSON parsing issues with ostypeid
func (c *Client) getVMPowerStateDirectAPI(ctx context.Context, vmID string) (string, error) {
	log.WithField("vm_id", vmID).Info("🔍 Using direct CloudStack API to get VM power state")

	// Get CloudStack endpoint and credentials from stored client config
	baseURL := c.config.APIURL
	apiKey := c.config.APIKey
	secretKey := c.config.SecretKey

	// Prepare CloudStack API parameters for listVirtualMachines
	params := url.Values{}
	params.Set("command", "listVirtualMachines")
	params.Set("id", vmID)
	params.Set("response", "json")
	params.Set("apiKey", apiKey)

	// Sort parameters for signature
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Build query string for signature generation
	var queryString strings.Builder
	for i, k := range keys {
		if i > 0 {
			queryString.WriteString("&")
		}
		queryString.WriteString(k)
		queryString.WriteString("=")
		queryString.WriteString(url.QueryEscape(params.Get(k)))
	}

	// Generate signature
	mac := hmac.New(sha1.New, []byte(secretKey))
	mac.Write([]byte(strings.ToLower(queryString.String())))
	signature := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	// Add signature to params
	params.Set("signature", signature)

	// Make the HTTP request
	requestURL := fmt.Sprintf("%s?%s", baseURL, params.Encode())
	log.WithField("vm_id", vmID).Debug("Making direct CloudStack API call to get VM")

	resp, err := http.Get(requestURL)
	if err != nil {
		return "", fmt.Errorf("failed to call CloudStack get VM API: %w", err)
	}
	defer resp.Body.Close()

	// Parse the response
	var response struct {
		ListVirtualMachinesResponse struct {
			VirtualMachine []struct {
				ID    string `json:"id"`
				Name  string `json:"name"`
				State string `json:"state"`
			} `json:"virtualmachine"`
		} `json:"listvirtualmachinesresponse"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", fmt.Errorf("failed to parse CloudStack API response: %w", err)
	}

	// Check if VM was found
	if len(response.ListVirtualMachinesResponse.VirtualMachine) == 0 {
		return "", fmt.Errorf("VM %s not found", vmID)
	}

	vm := response.ListVirtualMachinesResponse.VirtualMachine[0]

	log.WithFields(log.Fields{
		"vm_id":    vm.ID,
		"vm_name":  vm.Name,
		"vm_state": vm.State,
	}).Info("✅ Retrieved VM power state using direct CloudStack API")

	return vm.State, nil
}

// getZoneID resolves a zone name to zone ID
func (c *Client) getZoneID(zoneName string) (string, error) {
	log.WithField("zone_name", zoneName).Debug("Resolving zone name to zone ID")

	params := c.cs.Zone.NewListZonesParams()
	resp, err := c.cs.Zone.ListZones(params)
	if err != nil {
		return "", fmt.Errorf("failed to list zones: %w", err)
	}

	for _, zone := range resp.Zones {
		if zone.Name == zoneName {
			log.WithFields(log.Fields{
				"zone_name": zoneName,
				"zone_id":   zone.Id,
			}).Debug("Zone name resolved to zone ID")
			return zone.Id, nil
		}
	}

	return "", fmt.Errorf("zone not found: %s", zoneName)
}
