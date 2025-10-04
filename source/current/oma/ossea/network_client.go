// Package ossea provides network discovery and management for OSSEA (CloudStack) failover functionality
package ossea

import (
	"fmt"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

// Network represents an OSSEA network
type Network struct {
	ID                string            `json:"id"`
	Name              string            `json:"name"`
	DisplayText       string            `json:"display_text"`
	State             string            `json:"state"`
	ZoneID            string            `json:"zone_id"`
	ZoneName          string            `json:"zone_name"`
	NetworkOffering   string            `json:"network_offering"`
	NetworkOfferingID string            `json:"network_offering_id"`
	NetworkType       string            `json:"network_type"`
	TrafficType       string            `json:"traffic_type"`
	Gateway           string            `json:"gateway"`
	Netmask           string            `json:"netmask"`
	CIDR              string            `json:"cidr"`
	VLAN              string            `json:"vlan"`
	BroadcastURI      string            `json:"broadcast_uri"`
	IsDefault         bool              `json:"is_default"`
	IsShared          bool              `json:"is_shared"`
	CanUseForDeploy   bool              `json:"can_use_for_deploy"`
	IsSystem          bool              `json:"is_system"`
	IsPersistent      bool              `json:"is_persistent"`
	RestartRequired   bool              `json:"restart_required"`
	SpecifyIPRanges   bool              `json:"specify_ip_ranges"`
	AclType           string            `json:"acl_type"`
	NetworkDomain     string            `json:"network_domain"`
	DNS1              string            `json:"dns1"`
	DNS2              string            `json:"dns2"`
	Tags              map[string]string `json:"tags"`
	Related           string            `json:"related"`
	Created           string            `json:"created"`
}

// NetworkOffering represents an OSSEA network offering
type NetworkOffering struct {
	ID                      string            `json:"id"`
	Name                    string            `json:"name"`
	DisplayText             string            `json:"display_text"`
	TrafficType             string            `json:"traffic_type"`
	IsDefault               bool              `json:"is_default"`
	SpecifyVLAN             bool              `json:"specify_vlan"`
	ConserveMode            bool              `json:"conserve_mode"`
	SpecifyIPRanges         bool              `json:"specify_ip_ranges"`
	Availability            string            `json:"availability"`
	NetworkRate             int               `json:"network_rate"`
	State                   string            `json:"state"`
	GuestIPType             string            `json:"guest_ip_type"`
	ServiceCapabilities     map[string]string `json:"service_capabilities"`
	ServiceOfferingID       string            `json:"service_offering_id"`
	MaxConnections          int               `json:"max_connections"`
	IsNetworkOfferingForVpc bool              `json:"is_network_offering_for_vpc"`
	SupportsStrechedL2      bool              `json:"supports_streched_l2"`
	SupportsPublicAccess    bool              `json:"supports_public_access"`
	Created                 string            `json:"created"`
}

// NetworkCapabilities represents network capability details
type NetworkCapabilities struct {
	CanReachInternet       bool     `json:"can_reach_internet"`
	SupportsStaticNAT      bool     `json:"supports_static_nat"`
	SupportsPortForwarding bool     `json:"supports_port_forwarding"`
	SupportsLoadBalancing  bool     `json:"supports_load_balancing"`
	SupportsFirewall       bool     `json:"supports_firewall"`
	SupportsVPN            bool     `json:"supports_vpn"`
	SupportedServices      []string `json:"supported_services"`
	SecurityGroupEnabled   bool     `json:"security_group_enabled"`
}

// NetworkValidationResult represents network mapping validation results
type NetworkValidationResult struct {
	IsValid               bool                `json:"is_valid"`
	ValidationErrors      []string            `json:"validation_errors"`
	CompatibilityWarnings []string            `json:"compatibility_warnings"`
	NetworkCapabilities   NetworkCapabilities `json:"network_capabilities"`
}

// NetworkClient provides network discovery and management operations
type NetworkClient struct {
	client *Client
}

// NewNetworkClient creates a new network client
func NewNetworkClient(client *Client) *NetworkClient {
	return &NetworkClient{
		client: client,
	}
}

// ListNetworks discovers all available OSSEA networks
func (nc *NetworkClient) ListNetworks() ([]Network, error) {
	log.Info("🔍 Discovering OSSEA networks...")

	if nc.client == nil || nc.client.cs == nil {
		return nil, fmt.Errorf("CloudStack client not initialized")
	}

	// List all networks in the zone
	p := nc.client.cs.Network.NewListNetworksParams()
	// Note: Zone filtering removed as client.config is not accessible
	// Zone filtering should be handled at the service layer if needed

	// Set list all parameter to get all networks
	p.SetListall(true)

	resp, err := nc.client.cs.Network.ListNetworks(p)
	if err != nil {
		log.WithError(err).Error("Failed to list OSSEA networks")
		return nil, fmt.Errorf("failed to list networks: %w", err)
	}

	var networks []Network
	for _, network := range resp.Networks {
		networks = append(networks, Network{
			ID:                network.Id,
			Name:              network.Name,
			DisplayText:       network.Displaytext,
			State:             network.State,
			ZoneID:            network.Zoneid,
			ZoneName:          network.Zonename,
			NetworkOffering:   network.Networkofferingname,
			NetworkOfferingID: network.Networkofferingid,
			NetworkType:       network.Type,
			TrafficType:       network.Traffictype,
			Gateway:           network.Gateway,
			Netmask:           network.Netmask,
			CIDR:              network.Cidr,
			VLAN:              network.Vlan,
			BroadcastURI:      network.Broadcasturi,
			IsDefault:         network.Isdefault,
			IsShared:          false, // Use safe default as field may not be available
			CanUseForDeploy:   network.Canusefordeploy,
			IsSystem:          network.Issystem,
			IsPersistent:      network.Ispersistent,
			RestartRequired:   network.Restartrequired,
			SpecifyIPRanges:   network.Specifyipranges,
			AclType:           network.Acltype,
			NetworkDomain:     network.Networkdomain,
			DNS1:              network.Dns1,
			DNS2:              network.Dns2,
			Tags:              make(map[string]string), // Initialize empty tags
			Related:           network.Related,
			Created:           "", // Created field not available in CloudStack SDK
		})
	}

	log.WithField("count", len(networks)).Info("✅ Discovered OSSEA networks")
	return networks, nil
}

// GetNetworkByID retrieves a specific network by ID
func (nc *NetworkClient) GetNetworkByID(networkID string) (*Network, error) {
	log.WithField("network_id", networkID).Debug("Getting OSSEA network details")

	if nc.client == nil || nc.client.cs == nil {
		return nil, fmt.Errorf("CloudStack client not initialized")
	}

	p := nc.client.cs.Network.NewListNetworksParams()
	p.SetId(networkID)

	resp, err := nc.client.cs.Network.ListNetworks(p)
	if err != nil {
		return nil, fmt.Errorf("failed to get network %s: %w", networkID, err)
	}

	if len(resp.Networks) == 0 {
		return nil, fmt.Errorf("network %s not found", networkID)
	}

	network := resp.Networks[0]
	return &Network{
		ID:                network.Id,
		Name:              network.Name,
		DisplayText:       network.Displaytext,
		State:             network.State,
		ZoneID:            network.Zoneid,
		ZoneName:          network.Zonename,
		NetworkOffering:   network.Networkofferingname,
		NetworkOfferingID: network.Networkofferingid,
		NetworkType:       network.Type,
		TrafficType:       network.Traffictype,
		Gateway:           network.Gateway,
		Netmask:           network.Netmask,
		CIDR:              network.Cidr,
		VLAN:              network.Vlan,
		BroadcastURI:      network.Broadcasturi,
		IsDefault:         network.Isdefault,
		IsShared:          false, // Use safe default as field may not be available
		CanUseForDeploy:   network.Canusefordeploy,
		IsSystem:          network.Issystem,
		IsPersistent:      network.Ispersistent,
		RestartRequired:   network.Restartrequired,
		SpecifyIPRanges:   network.Specifyipranges,
		AclType:           network.Acltype,
		NetworkDomain:     network.Networkdomain,
		DNS1:              network.Dns1,
		DNS2:              network.Dns2,
		Tags:              make(map[string]string), // Initialize empty tags
		Related:           network.Related,
		Created:           "", // Created field not available in CloudStack SDK
	}, nil
}

// GetNetworkByName retrieves a network by name
func (nc *NetworkClient) GetNetworkByName(networkName string) (*Network, error) {
	log.WithField("network_name", networkName).Debug("Finding OSSEA network by name")

	networks, err := nc.ListNetworks()
	if err != nil {
		return nil, fmt.Errorf("failed to list networks: %w", err)
	}

	for _, network := range networks {
		if network.Name == networkName {
			return &network, nil
		}
	}

	return nil, fmt.Errorf("network '%s' not found", networkName)
}

// ListNetworkOfferings discovers available network offerings
func (nc *NetworkClient) ListNetworkOfferings() ([]NetworkOffering, error) {
	log.Info("🔍 Discovering OSSEA network offerings...")

	if nc.client == nil || nc.client.cs == nil {
		return nil, fmt.Errorf("CloudStack client not initialized")
	}

	p := nc.client.cs.NetworkOffering.NewListNetworkOfferingsParams()
	p.SetState("Enabled") // Only get enabled offerings

	resp, err := nc.client.cs.NetworkOffering.ListNetworkOfferings(p)
	if err != nil {
		log.WithError(err).Error("Failed to list network offerings")
		return nil, fmt.Errorf("failed to list network offerings: %w", err)
	}

	var offerings []NetworkOffering
	for _, offering := range resp.NetworkOfferings {
		offerings = append(offerings, NetworkOffering{
			ID:                      offering.Id,
			Name:                    offering.Name,
			DisplayText:             offering.Displaytext,
			TrafficType:             offering.Traffictype,
			IsDefault:               offering.Isdefault,
			SpecifyVLAN:             offering.Specifyvlan,
			ConserveMode:            offering.Conservemode,
			SpecifyIPRanges:         offering.Specifyipranges,
			Availability:            offering.Availability,
			NetworkRate:             offering.Networkrate,
			State:                   offering.State,
			GuestIPType:             offering.Guestiptype,
			ServiceCapabilities:     make(map[string]string), // Initialize empty
			ServiceOfferingID:       offering.Serviceofferingid,
			MaxConnections:          offering.Maxconnections,
			IsNetworkOfferingForVpc: offering.Forvpc,
			SupportsStrechedL2:      offering.Supportsstrechedl2subnet,
			SupportsPublicAccess:    offering.Supportspublicaccess,
			Created:                 offering.Created,
		})
	}

	log.WithField("count", len(offerings)).Info("✅ Discovered network offerings")
	return offerings, nil
}

// ValidateNetworkMapping validates if a source network can be mapped to a destination network
func (nc *NetworkClient) ValidateNetworkMapping(sourceNetworkName, destinationNetworkID string) (*NetworkValidationResult, error) {
	log.WithFields(log.Fields{
		"source_network":      sourceNetworkName,
		"destination_network": destinationNetworkID,
	}).Info("🔍 Validating network mapping...")

	result := &NetworkValidationResult{
		IsValid:               true,
		ValidationErrors:      []string{},
		CompatibilityWarnings: []string{},
		NetworkCapabilities:   NetworkCapabilities{},
	}

	// Get destination network details
	destNetwork, err := nc.GetNetworkByID(destinationNetworkID)
	if err != nil {
		result.IsValid = false
		result.ValidationErrors = append(result.ValidationErrors, fmt.Sprintf("Destination network not found: %v", err))
		return result, nil
	}

	// Validate network state
	if destNetwork.State != "Implemented" && destNetwork.State != "Allocated" {
		result.ValidationErrors = append(result.ValidationErrors,
			fmt.Sprintf("Destination network '%s' is not in a usable state: %s", destNetwork.Name, destNetwork.State))
		result.IsValid = false
	}

	// Check if network can be used for deployment
	if !destNetwork.CanUseForDeploy {
		result.ValidationErrors = append(result.ValidationErrors,
			fmt.Sprintf("Destination network '%s' cannot be used for VM deployment", destNetwork.Name))
		result.IsValid = false
	}

	// Analyze network capabilities
	result.NetworkCapabilities = nc.analyzeNetworkCapabilities(destNetwork)

	// Add compatibility warnings
	if destNetwork.IsSystem {
		result.CompatibilityWarnings = append(result.CompatibilityWarnings,
			"Destination is a system network - verify management access requirements")
	}

	if destNetwork.RestartRequired {
		result.CompatibilityWarnings = append(result.CompatibilityWarnings,
			"Network may require restart for configuration changes")
	}

	if strings.Contains(sourceNetworkName, "prod") && strings.Contains(destNetwork.Name, "test") {
		result.CompatibilityWarnings = append(result.CompatibilityWarnings,
			"Mapping production network to test network - verify isolation requirements")
	}

	log.WithFields(log.Fields{
		"is_valid":    result.IsValid,
		"errors":      len(result.ValidationErrors),
		"warnings":    len(result.CompatibilityWarnings),
		"destination": destNetwork.Name,
	}).Info("✅ Network mapping validation completed")

	return result, nil
}

// analyzeNetworkCapabilities analyzes network capabilities based on network offering and configuration
func (nc *NetworkClient) analyzeNetworkCapabilities(network *Network) NetworkCapabilities {
	capabilities := NetworkCapabilities{
		SupportedServices: []string{},
	}

	// Basic capability analysis based on network type and offering
	if network.TrafficType == "Guest" {
		capabilities.CanReachInternet = true
		capabilities.SupportedServices = append(capabilities.SupportedServices, "Dhcp", "Dns")
	}

	if network.TrafficType == "Management" {
		capabilities.SupportsStaticNAT = true
		capabilities.SupportedServices = append(capabilities.SupportedServices, "Gateway")
	}

	// Analyze based on network offering capabilities
	if !network.IsSystem {
		capabilities.SupportsFirewall = true
		capabilities.SupportsPortForwarding = true
		capabilities.SupportedServices = append(capabilities.SupportedServices, "Firewall", "PortForwarding")
	}

	if network.IsShared {
		capabilities.SecurityGroupEnabled = true
		capabilities.SupportedServices = append(capabilities.SupportedServices, "SecurityGroup")
	}

	// L2 network capabilities
	if network.NetworkType == "L2" {
		capabilities.SupportsVPN = false
		capabilities.SupportsLoadBalancing = false
	} else {
		capabilities.SupportsVPN = true
		capabilities.SupportsLoadBalancing = true
		capabilities.SupportedServices = append(capabilities.SupportedServices, "Vpn", "Lb")
	}

	return capabilities
}

// GetNetworkCapabilities retrieves detailed capabilities for a network
func (nc *NetworkClient) GetNetworkCapabilities(networkID string) (*NetworkCapabilities, error) {
	network, err := nc.GetNetworkByID(networkID)
	if err != nil {
		return nil, fmt.Errorf("failed to get network for capability analysis: %w", err)
	}

	capabilities := nc.analyzeNetworkCapabilities(network)
	return &capabilities, nil
}

// ListTestNetworks finds networks suitable for test failovers (typically Layer 2 networks)
func (nc *NetworkClient) ListTestNetworks() ([]Network, error) {
	log.Info("🔍 Discovering test-suitable networks...")

	networks, err := nc.ListNetworks()
	if err != nil {
		return nil, fmt.Errorf("failed to list networks: %w", err)
	}

	var testNetworks []Network
	for _, network := range networks {
		// Consider networks suitable for testing based on:
		// 1. L2 networks (isolated)
		// 2. Networks with "test" in the name
		// 3. Non-production networks
		if nc.isTestSuitableNetwork(&network) {
			testNetworks = append(testNetworks, network)
		}
	}

	log.WithField("count", len(testNetworks)).Info("✅ Found test-suitable networks")
	return testNetworks, nil
}

// isTestSuitableNetwork determines if a network is suitable for test failovers
func (nc *NetworkClient) isTestSuitableNetwork(network *Network) bool {
	// L2 networks are ideal for test isolation
	if network.NetworkType == "L2" {
		return true
	}

	// Networks with test indicators in name
	testKeywords := []string{"test", "testing", "lab", "dev", "development", "staging"}
	networkNameLower := strings.ToLower(network.Name)
	for _, keyword := range testKeywords {
		if strings.Contains(networkNameLower, keyword) {
			return true
		}
	}

	// Isolated networks that are not system or default networks
	if !network.IsSystem && !network.IsDefault && network.CanUseForDeploy {
		return true
	}

	return false
}

// WaitForNetworkState waits for a network to reach the specified state
func (nc *NetworkClient) WaitForNetworkState(networkID, expectedState string, timeout time.Duration) error {
	log.WithFields(log.Fields{
		"network_id":     networkID,
		"expected_state": expectedState,
		"timeout":        timeout,
	}).Info("⏳ Waiting for network state...")

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		network, err := nc.GetNetworkByID(networkID)
		if err != nil {
			return fmt.Errorf("failed to check network state: %w", err)
		}

		if network.State == expectedState {
			log.WithFields(log.Fields{
				"network_id": networkID,
				"state":      network.State,
			}).Info("✅ Network reached expected state")
			return nil
		}

		log.WithFields(log.Fields{
			"network_id":     networkID,
			"current_state":  network.State,
			"expected_state": expectedState,
		}).Debug("Network state not ready, waiting...")

		time.Sleep(5 * time.Second)
	}

	return fmt.Errorf("timeout waiting for network %s to reach state %s", networkID, expectedState)
}
