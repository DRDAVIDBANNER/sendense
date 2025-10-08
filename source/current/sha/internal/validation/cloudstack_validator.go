package validation

import (
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"sort"
	"strings"

	"github.com/vexxhost/migratekit-sha/ossea"
	log "github.com/sirupsen/logrus"
)

// CloudStackValidator handles CloudStack prerequisite validation
type CloudStackValidator struct {
	client *ossea.Client
}

// NewCloudStackValidator creates a new validator instance
func NewCloudStackValidator(client *ossea.Client) *CloudStackValidator {
	return &CloudStackValidator{
		client: client,
	}
}

// ValidationResult contains the results of all CloudStack validations
type ValidationResult struct {
	SHAVMDetection   *ValidationCheck `json:"oma_vm_detection"`
	ComputeOffering  *ValidationCheck `json:"compute_offering"`
	AccountMatch     *ValidationCheck `json:"account_match"`
	NetworkSelection *ValidationCheck `json:"network_selection"`
	OverallStatus    string           `json:"overall_status"` // "pass", "warning", "fail"
}

// ValidationCheck represents the result of a single validation
type ValidationCheck struct {
	Status  string                 `json:"status"`  // "pass", "warning", "fail", "skipped"
	Message string                 `json:"message"` // User-friendly message
	Details map[string]interface{} `json:"details,omitempty"`
}

// SHAVMInfo contains information about the detected SHA VM
type SHAVMInfo struct {
	VMID       string `json:"vm_id"`
	VMName     string `json:"vm_name"`
	MACAddress string `json:"mac_address"`
	IPAddress  string `json:"ip_address"`
	Account    string `json:"account"`
}

// NetworkInfo contains CloudStack network information
type NetworkInfo struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	ZoneID    string `json:"zone_id"`
	ZoneName  string `json:"zone_name"`
	State     string `json:"state"`
	IsDefault bool   `json:"is_default"`
}

// DetectOMAVMID attempts to find the SHA VM ID by matching MAC addresses
func (v *CloudStackValidator) DetectOMAVMID(ctx context.Context) (*SHAVMInfo, error) {
	log.Info("🔍 Detecting SHA VM ID by MAC address")

	// Get local MAC addresses
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("failed to get network interfaces: %w", err)
	}

	// Collect MAC addresses from non-loopback, active interfaces
	localMACs := make(map[string]string) // mac -> interface name
	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp != 0 && iface.Flags&net.FlagLoopback == 0 {
			mac := iface.HardwareAddr.String()
			if mac != "" {
				localMACs[mac] = iface.Name
				log.WithFields(log.Fields{
					"interface": iface.Name,
					"mac":       mac,
				}).Debug("Found local MAC address")
			}
		}
	}

	if len(localMACs) == 0 {
		return nil, fmt.Errorf("no network interfaces found")
	}

	// List all VMs in CloudStack
	vms, err := v.client.ListVMs(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list CloudStack VMs: %w", err)
	}

	log.WithField("vm_count", len(vms)).Debug("Searching for SHA VM by MAC address")

	// Match MAC addresses
	for _, vm := range vms {
		// Check all NICs on the VM
		for _, nic := range vm.NICs {
			if ifaceName, found := localMACs[nic.MACAddress]; found {
				log.WithFields(log.Fields{
					"vm_id":        vm.ID,
					"vm_name":      vm.DisplayName,
					"mac_address":  nic.MACAddress,
					"interface":    ifaceName,
					"account":      vm.Account,
				}).Info("✅ Found SHA VM by MAC address match")

				return &SHAVMInfo{
					VMID:       vm.ID,
					VMName:     vm.DisplayName,
					MACAddress: nic.MACAddress,
					IPAddress:  nic.IPAddress,
					Account:    vm.Account,
				}, nil
			}
		}
	}

	// No match found
	return nil, fmt.Errorf("no CloudStack VM found matching local MAC addresses")
}

// ValidateComputeOffering checks if the compute offering supports custom specifications
func (v *CloudStackValidator) ValidateComputeOffering(ctx context.Context, offeringID string) error {
	log.WithField("offering_id", offeringID).Info("🔍 Validating compute offering")

	if offeringID == "" {
		return fmt.Errorf("compute offering ID is required")
	}

	// Get all service offerings
	offerings, err := v.client.ListServiceOfferings()
	if err != nil {
		return fmt.Errorf("failed to list service offerings: %w", err)
	}

	// Find the specified offering
	var targetOffering *ossea.ServiceOffering
	for i, offering := range offerings {
		if offering.ID == offeringID {
			targetOffering = &offerings[i]
			break
		}
	}

	if targetOffering == nil {
		return fmt.Errorf("compute offering '%s' not found in CloudStack", offeringID)
	}

	// Check if it's customized
	if !targetOffering.IsCustomized {
		return fmt.Errorf("compute offering '%s' does not support custom VM specifications (iscustomized=false). A customizable offering is required to match source VM configurations", targetOffering.DisplayText)
	}

	log.WithFields(log.Fields{
		"offering_id":   targetOffering.ID,
		"offering_name": targetOffering.DisplayText,
		"is_customized": targetOffering.IsCustomized,
	}).Info("✅ Compute offering is valid (supports custom specifications)")

	return nil
}

// ValidateAccountMatch verifies the API key account matches the SHA VM account
func (v *CloudStackValidator) ValidateAccountMatch(ctx context.Context, shaVMID string) error {
	log.Info("🔍 Validating API key account matches SHA VM account")

	if shaVMID == "" {
		return fmt.Errorf("SHA VM ID is required for account validation")
	}

	// Get the account that owns the API key by calling listAccounts
	apiKeyAccount, err := v.getAPIKeyAccount(ctx)
	if err != nil {
		return fmt.Errorf("failed to get API key account: %w", err)
	}

	// Get SHA VM details
	vms, err := v.client.ListVMs(ctx)
	if err != nil {
		return fmt.Errorf("failed to list VMs: %w", err)
	}

	var shaVM *ossea.VirtualMachine
	for _, vm := range vms {
		if vm.ID == shaVMID {
			shaVM = vm
			break
		}
	}

	if shaVM == nil {
		return fmt.Errorf("SHA VM with ID '%s' not found in CloudStack", shaVMID)
	}

	// Compare accounts
	if shaVM.Account != apiKeyAccount {
		return fmt.Errorf("API key account '%s' does not match SHA VM account '%s'. Please use an API key from the same CloudStack account that owns the SHA VM", apiKeyAccount, shaVM.Account)
	}

	log.WithFields(log.Fields{
		"api_key_account": apiKeyAccount,
		"oma_vm_account":  shaVM.Account,
		"oma_vm_id":       shaVMID,
	}).Info("✅ Account validation passed")

	return nil
}

// getAPIKeyAccount retrieves the account that owns the API key
func (v *CloudStackValidator) getAPIKeyAccount(ctx context.Context) (string, error) {
	// Use direct API call to get account info
	params := url.Values{}
	params.Set("command", "listAccounts")
	params.Set("response", "json")
	params.Set("apiKey", v.client.GetAPIKey())

	// Sort and sign
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var queryString strings.Builder
	for i, k := range keys {
		if i > 0 {
			queryString.WriteString("&")
		}
		queryString.WriteString(url.QueryEscape(k))
		queryString.WriteString("=")
		queryString.WriteString(url.QueryEscape(params.Get(k)))
	}

	mac := hmac.New(sha1.New, []byte(v.client.GetSecretKey()))
	mac.Write([]byte(strings.ToLower(queryString.String())))
	signature := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	params.Set("signature", signature)

	requestURL := fmt.Sprintf("%s?%s", v.client.GetAPIURL(), params.Encode())
	resp, err := http.Get(requestURL)
	if err != nil {
		return "", fmt.Errorf("failed to call CloudStack API: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read API response: %w", err)
	}

	var accountResp struct {
		ListAccountsResponse struct {
			Account []struct {
				Name string `json:"name"`
			} `json:"account"`
		} `json:"listaccountsresponse"`
	}

	if err := json.Unmarshal(body, &accountResp); err != nil {
		return "", fmt.Errorf("failed to parse account response: %w", err)
	}

	if len(accountResp.ListAccountsResponse.Account) == 0 {
		return "", fmt.Errorf("no account information returned from CloudStack")
	}

	return accountResp.ListAccountsResponse.Account[0].Name, nil
}

// ListAvailableNetworks retrieves all networks accessible to the API key account
func (v *CloudStackValidator) ListAvailableNetworks(ctx context.Context) ([]NetworkInfo, error) {
	log.Info("🔍 Listing available CloudStack networks")

	networks, err := v.client.ListNetworks()
	if err != nil {
		return nil, fmt.Errorf("failed to list networks: %w", err)
	}

	result := make([]NetworkInfo, 0, len(networks))
	for _, net := range networks {
		result = append(result, NetworkInfo{
			ID:       net.ID,
			Name:     net.Name,
			ZoneID:   net.ZoneID,
			ZoneName: net.ZoneName,
			State:    net.State,
		})
	}

	log.WithField("network_count", len(result)).Info("✅ Listed available networks")
	return result, nil
}

// ValidateNetworkExists checks if the specified network ID exists
func (v *CloudStackValidator) ValidateNetworkExists(ctx context.Context, networkID string) error {
	log.WithField("network_id", networkID).Info("🔍 Validating network exists")

	if networkID == "" {
		return fmt.Errorf("network ID is required")
	}

	networks, err := v.ListAvailableNetworks(ctx)
	if err != nil {
		return err
	}

	for _, net := range networks {
		if net.ID == networkID {
			log.WithFields(log.Fields{
				"network_id":   net.ID,
				"network_name": net.Name,
				"zone":         net.ZoneName,
			}).Info("✅ Network validation passed")
			return nil
		}
	}

	return fmt.Errorf("network with ID '%s' not found in CloudStack", networkID)
}

// ValidateAll runs all CloudStack prerequisite validations
func (v *CloudStackValidator) ValidateAll(ctx context.Context, shaVMID, computeOfferingID, networkID string) *ValidationResult {
	log.Info("🔍 Running comprehensive CloudStack validation")

	result := &ValidationResult{
		SHAVMDetection:   &ValidationCheck{Status: "skipped"},
		ComputeOffering:  &ValidationCheck{Status: "skipped"},
		AccountMatch:     &ValidationCheck{Status: "skipped"},
		NetworkSelection: &ValidationCheck{Status: "skipped"},
		OverallStatus:    "pass",
	}

	// 1. SHA VM Detection (only if not provided)
	if shaVMID == "" {
		shaInfo, err := v.DetectOMAVMID(ctx)
		if err != nil {
			result.SHAVMDetection = &ValidationCheck{
				Status:  "warning",
				Message: "Could not auto-detect SHA VM ID. Please enter it manually.",
				Details: map[string]interface{}{
					"error": err.Error(),
				},
			}
			result.OverallStatus = "warning"
		} else {
			result.SHAVMDetection = &ValidationCheck{
				Status:  "pass",
				Message: fmt.Sprintf("SHA VM detected: %s", shaInfo.VMName),
				Details: map[string]interface{}{
					"vm_id":       shaInfo.VMID,
					"vm_name":     shaInfo.VMName,
					"mac_address": shaInfo.MACAddress,
					"ip_address":  shaInfo.IPAddress,
					"account":     shaInfo.Account,
				},
			}
			shaVMID = shaInfo.VMID // Use detected ID for subsequent validations
		}
	} else {
		result.SHAVMDetection = &ValidationCheck{
			Status:  "pass",
			Message: "SHA VM ID provided manually",
			Details: map[string]interface{}{
				"vm_id": shaVMID,
			},
		}
	}

	// 2. Compute Offering Validation
	if computeOfferingID != "" {
		if err := v.ValidateComputeOffering(ctx, computeOfferingID); err != nil {
			result.ComputeOffering = &ValidationCheck{
				Status:  "fail",
				Message: "Compute offering does not support custom VM specifications",
				Details: map[string]interface{}{
					"error":       err.Error(),
					"offering_id": computeOfferingID,
				},
			}
			result.OverallStatus = "fail"
		} else {
			result.ComputeOffering = &ValidationCheck{
				Status:  "pass",
				Message: "Compute offering supports custom specifications",
				Details: map[string]interface{}{
					"offering_id": computeOfferingID,
				},
			}
		}
	} else {
		result.ComputeOffering = &ValidationCheck{
			Status:  "warning",
			Message: "No compute offering specified",
		}
		if result.OverallStatus != "fail" {
			result.OverallStatus = "warning"
		}
	}

	// 3. Account Match Validation (requires SHA VM ID)
	if shaVMID != "" {
		if err := v.ValidateAccountMatch(ctx, shaVMID); err != nil {
			result.AccountMatch = &ValidationCheck{
				Status:  "fail",
				Message: "API key account does not match SHA VM account",
				Details: map[string]interface{}{
					"error": err.Error(),
				},
			}
			result.OverallStatus = "fail"
		} else {
			result.AccountMatch = &ValidationCheck{
				Status:  "pass",
				Message: "API key account matches SHA VM account",
			}
		}
	} else {
		result.AccountMatch = &ValidationCheck{
			Status:  "skipped",
			Message: "SHA VM ID required for account validation",
		}
	}

	// 4. Network Selection Validation
	if networkID != "" {
		if err := v.ValidateNetworkExists(ctx, networkID); err != nil {
			result.NetworkSelection = &ValidationCheck{
				Status:  "fail",
				Message: "Selected network not found in CloudStack",
				Details: map[string]interface{}{
					"error":      err.Error(),
					"network_id": networkID,
				},
			}
			result.OverallStatus = "fail"
		} else {
			result.NetworkSelection = &ValidationCheck{
				Status:  "pass",
				Message: "Network selection is valid",
				Details: map[string]interface{}{
					"network_id": networkID,
				},
			}
		}
	} else {
		result.NetworkSelection = &ValidationCheck{
			Status:  "warning",
			Message: "No network selected",
		}
		if result.OverallStatus != "fail" {
			result.OverallStatus = "warning"
		}
	}

	log.WithField("overall_status", result.OverallStatus).Info("✅ CloudStack validation complete")
	return result
}


