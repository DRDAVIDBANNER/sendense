// Package services provides SNA VM information service for network mapping
package services

import (
	"fmt"
	"strings"

	"github.com/vexxhost/migratekit-sha/models"

	log "github.com/sirupsen/logrus"
)

// SNAVMInfoService provides VM information from SNA via control API
type SNAVMInfoService struct {
	snaClient SNAControlClient
}

// SNAControlClient interface for SNA control API communication
type SNAControlClient interface {
	GetVMInfo(vmID string) (*models.VMInfo, error)
	DiscoverVMs(filter string) ([]models.VMInfo, error)
}

// NewVMAVMInfoService creates a new SNA VM info service
func NewVMAVMInfoService(snaClient SNAControlClient) *SNAVMInfoService {
	return &SNAVMInfoService{
		snaClient: snaClient,
	}
}

// GetVMNetworkConfiguration retrieves VM network configuration from SNA
func (vvis *SNAVMInfoService) GetVMNetworkConfiguration(vmID string) ([]SourceNetworkInfo, error) {
	log.WithField("vm_id", vmID).Debug("Getting VM network configuration from SNA")

	// Get VM info from SNA
	vmInfo, err := vvis.snaClient.GetVMInfo(vmID)
	if err != nil {
		return nil, fmt.Errorf("failed to get VM info from SNA: %w", err)
	}

	// Convert network information to SourceNetworkInfo
	var sourceNetworks []SourceNetworkInfo
	for _, networkInfo := range vmInfo.Networks {
		sourceNetwork := SourceNetworkInfo{
			NetworkName:     networkInfo.NetworkName,
			AdapterType:     networkInfo.AdapterType,
			MACAddress:      networkInfo.MACAddress,
			ConnectionState: getConnectionState(networkInfo.Connected),
			VLANID:          "",           // VLAN ID not available in current NetworkInfo struct
			IPConfiguration: "configured", // Default since IP address not available
		}

		sourceNetworks = append(sourceNetworks, sourceNetwork)
	}

	log.WithFields(log.Fields{
		"vm_id":         vmID,
		"network_count": len(sourceNetworks),
	}).Debug("✅ Retrieved VM network configuration from SNA")

	return sourceNetworks, nil
}

// ValidateVMExists validates that a VM exists in SNA
func (vvis *SNAVMInfoService) ValidateVMExists(vmID string) error {
	log.WithField("vm_id", vmID).Debug("Validating VM exists in SNA")

	// Try to get VM info
	vmInfo, err := vvis.snaClient.GetVMInfo(vmID)
	if err != nil {
		return fmt.Errorf("VM %s not found in SNA: %w", vmID, err)
	}

	if vmInfo == nil {
		return fmt.Errorf("VM %s not found in SNA", vmID)
	}

	// Additional validation checks
	if vmInfo.ID == "" {
		return fmt.Errorf("VM %s has empty ID in SNA", vmID)
	}

	if vmInfo.Name == "" {
		return fmt.Errorf("VM %s has empty name in SNA", vmID)
	}

	log.WithFields(log.Fields{
		"vm_id":       vmID,
		"vm_name":     vmInfo.Name,
		"power_state": vmInfo.PowerState,
	}).Debug("✅ VM validation successful")

	return nil
}

// GetCompleteVMInfo retrieves complete VM information including specifications
func (vvis *SNAVMInfoService) GetCompleteVMInfo(vmID string) (*models.VMInfo, error) {
	log.WithField("vm_id", vmID).Info("Getting complete VM information from SNA")

	vmInfo, err := vvis.snaClient.GetVMInfo(vmID)
	if err != nil {
		return nil, fmt.Errorf("failed to get VM info: %w", err)
	}

	log.WithFields(log.Fields{
		"vm_id":         vmID,
		"vm_name":       vmInfo.Name,
		"cpu_count":     vmInfo.CPUs,
		"memory_mb":     vmInfo.MemoryMB,
		"disk_count":    len(vmInfo.Disks),
		"network_count": len(vmInfo.Networks),
	}).Info("✅ Retrieved complete VM information")

	return vmInfo, nil
}

// GetVMNetworkSummary provides a summary of VM network configuration
func (vvis *SNAVMInfoService) GetVMNetworkSummary(vmID string) (map[string]interface{}, error) {
	log.WithField("vm_id", vmID).Debug("Getting VM network summary")

	sourceNetworks, err := vvis.GetVMNetworkConfiguration(vmID)
	if err != nil {
		return nil, fmt.Errorf("failed to get network configuration: %w", err)
	}

	summary := map[string]interface{}{
		"vm_id":             vmID,
		"total_networks":    len(sourceNetworks),
		"network_types":     make(map[string]int),
		"connection_states": make(map[string]int),
		"has_vlan_networks": false,
		"networks":          []map[string]interface{}{},
	}

	// Analyze network configuration
	for _, network := range sourceNetworks {
		// Count network types
		if count, exists := summary["network_types"].(map[string]int)[network.AdapterType]; exists {
			summary["network_types"].(map[string]int)[network.AdapterType] = count + 1
		} else {
			summary["network_types"].(map[string]int)[network.AdapterType] = 1
		}

		// Count connection states
		if count, exists := summary["connection_states"].(map[string]int)[network.ConnectionState]; exists {
			summary["connection_states"].(map[string]int)[network.ConnectionState] = count + 1
		} else {
			summary["connection_states"].(map[string]int)[network.ConnectionState] = 1
		}

		// Check for VLAN networks (VLAN info not currently available)
		if false { // Placeholder until VLAN info is available
			summary["has_vlan_networks"] = true
		}

		// Add network details
		networkDetails := map[string]interface{}{
			"name":             network.NetworkName,
			"type":             network.AdapterType,
			"mac_address":      network.MACAddress,
			"connection_state": network.ConnectionState,
			"vlan_id":          "", // VLAN ID not available
			"ip_configuration": network.IPConfiguration,
		}
		summary["networks"] = append(summary["networks"].([]map[string]interface{}), networkDetails)
	}

	log.WithFields(log.Fields{
		"vm_id":          vmID,
		"total_networks": summary["total_networks"],
		"network_types":  len(summary["network_types"].(map[string]int)),
		"has_vlans":      summary["has_vlan_networks"],
	}).Debug("✅ Generated VM network summary")

	return summary, nil
}

// DiscoverVMsWithNetworkInfo discovers VMs and includes network information
func (vvis *SNAVMInfoService) DiscoverVMsWithNetworkInfo(filter string) ([]models.VMInfo, error) {
	log.WithField("filter", filter).Info("Discovering VMs with network information from SNA")

	vms, err := vvis.snaClient.DiscoverVMs(filter)
	if err != nil {
		return nil, fmt.Errorf("failed to discover VMs from SNA: %w", err)
	}

	log.WithFields(log.Fields{
		"filter":   filter,
		"vm_count": len(vms),
	}).Info("✅ Discovered VMs with network information")

	return vms, nil
}

// ValidateNetworkConnectivity validates VM network connectivity requirements
func (vvis *SNAVMInfoService) ValidateNetworkConnectivity(vmID string) (map[string]interface{}, error) {
	log.WithField("vm_id", vmID).Info("Validating VM network connectivity")

	sourceNetworks, err := vvis.GetVMNetworkConfiguration(vmID)
	if err != nil {
		return nil, fmt.Errorf("failed to get network configuration: %w", err)
	}

	validation := map[string]interface{}{
		"vm_id":                 vmID,
		"total_networks":        len(sourceNetworks),
		"connected_networks":    0,
		"disconnected_networks": 0,
		"network_issues":        []string{},
		"connectivity_warnings": []string{},
		"network_requirements":  []string{},
		"is_connectivity_ready": true,
	}

	// Analyze each network
	for _, network := range sourceNetworks {
		if network.ConnectionState == "connected" {
			validation["connected_networks"] = validation["connected_networks"].(int) + 1
		} else {
			validation["disconnected_networks"] = validation["disconnected_networks"].(int) + 1
			validation["network_issues"] = append(validation["network_issues"].([]string),
				fmt.Sprintf("Network '%s' is not connected", network.NetworkName))
		}

		// Check for network naming issues
		if strings.Contains(strings.ToLower(network.NetworkName), "unknown") {
			validation["connectivity_warnings"] = append(validation["connectivity_warnings"].([]string),
				fmt.Sprintf("Network '%s' has unknown name - may need manual mapping", network.NetworkName))
		}

		// Check for VLAN requirements (VLAN info not currently available)
		if false { // Placeholder until VLAN info is available
			validation["network_requirements"] = append(validation["network_requirements"].([]string),
				fmt.Sprintf("Network '%s' requires VLAN %s support", network.NetworkName, network.VLANID))
		}

		// Check for static IP configuration
		if strings.Contains(strings.ToLower(network.IPConfiguration), "static") {
			validation["network_requirements"] = append(validation["network_requirements"].([]string),
				fmt.Sprintf("Network '%s' uses static IP configuration", network.NetworkName))
		}
	}

	// Overall connectivity assessment
	if validation["disconnected_networks"].(int) > 0 {
		validation["is_connectivity_ready"] = false
		validation["network_issues"] = append(validation["network_issues"].([]string),
			"VM has disconnected network adapters")
	}

	if len(sourceNetworks) == 0 {
		validation["is_connectivity_ready"] = false
		validation["network_issues"] = append(validation["network_issues"].([]string),
			"VM has no network adapters configured")
	}

	log.WithFields(log.Fields{
		"vm_id":              vmID,
		"total_networks":     validation["total_networks"],
		"connected":          validation["connected_networks"],
		"disconnected":       validation["disconnected_networks"],
		"connectivity_ready": validation["is_connectivity_ready"],
		"issues":             len(validation["network_issues"].([]string)),
		"warnings":           len(validation["connectivity_warnings"].([]string)),
	}).Info("✅ Network connectivity validation completed")

	return validation, nil
}

// Helper functions

// getConnectionState converts connection status to standard state
func getConnectionState(connected bool) string {
	if connected {
		return "connected"
	}
	return "disconnected"
}

// getIPConfiguration analyzes IP configuration type
func getIPConfiguration(ipAddress string) string {
	if ipAddress == "" {
		return "none"
	}

	// Simple heuristic to determine configuration type
	if strings.Contains(ipAddress, "169.254.") {
		return "apipa"
	}

	if strings.Contains(ipAddress, "127.") {
		return "loopback"
	}

	if strings.Contains(ipAddress, "192.168.") ||
		strings.Contains(ipAddress, "10.") ||
		strings.HasPrefix(ipAddress, "172.") {
		return "private_dhcp_or_static"
	}

	return "configured"
}

// GetVMNetworkCapabilities analyzes VM network adapter capabilities
func (vvis *SNAVMInfoService) GetVMNetworkCapabilities(vmID string) (map[string]interface{}, error) {
	log.WithField("vm_id", vmID).Debug("Analyzing VM network capabilities")

	vmInfo, err := vvis.GetCompleteVMInfo(vmID)
	if err != nil {
		return nil, fmt.Errorf("failed to get VM info: %w", err)
	}

	capabilities := map[string]interface{}{
		"vm_id":                  vmID,
		"supports_multiple_nics": len(vmInfo.Networks) > 1,
		"nic_types":              make(map[string]int),
		"has_vlan_support":       false,
		"has_advanced_features":  false,
		"max_nics_supported":     8, // Default assumption for most VMs
		"network_features":       []string{},
	}

	// Analyze network adapter types and capabilities
	for _, network := range vmInfo.Networks {
		// Count NIC types
		if count, exists := capabilities["nic_types"].(map[string]int)[network.Type]; exists {
			capabilities["nic_types"].(map[string]int)[network.Type] = count + 1
		} else {
			capabilities["nic_types"].(map[string]int)[network.Type] = 1
		}

		// Check for VLAN support (VLAN info not currently available)
		if false { // Placeholder until VLAN info is available
			capabilities["has_vlan_support"] = true
			capabilities["network_features"] = append(capabilities["network_features"].([]string), "VLAN")
		}

		// Check for advanced adapter types
		if strings.Contains(strings.ToLower(network.Type), "vmxnet") {
			capabilities["has_advanced_features"] = true
			capabilities["network_features"] = append(capabilities["network_features"].([]string), "VMXNet")
		}

		if strings.Contains(strings.ToLower(network.Type), "sr-iov") {
			capabilities["has_advanced_features"] = true
			capabilities["network_features"] = append(capabilities["network_features"].([]string), "SR-IOV")
		}
	}

	// Remove duplicate features
	capabilities["network_features"] = removeDuplicateStrings(capabilities["network_features"].([]string))

	log.WithFields(log.Fields{
		"vm_id":             vmID,
		"multiple_nics":     capabilities["supports_multiple_nics"],
		"nic_types":         len(capabilities["nic_types"].(map[string]int)),
		"vlan_support":      capabilities["has_vlan_support"],
		"advanced_features": capabilities["has_advanced_features"],
	}).Debug("✅ VM network capabilities analyzed")

	return capabilities, nil
}

// removeDuplicateStrings removes duplicate strings from slice
func removeDuplicateStrings(slice []string) []string {
	keys := make(map[string]bool)
	var result []string

	for _, item := range slice {
		if !keys[item] {
			keys[item] = true
			result = append(result, item)
		}
	}

	return result
}
