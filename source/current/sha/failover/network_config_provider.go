// Package failover provides network configuration resolution for unified failover operations
package failover

import (
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/vexxhost/migratekit-sha/database"
)

// NetworkConfigProvider resolves network mappings for failover operations
// This component bridges the dual network mapping system with VM creation
type NetworkConfigProvider struct {
	networkMappingRepo *database.NetworkMappingRepository
	defaultNetworkID   string // Fallback network ID from OSSEA config
}

// NewNetworkConfigProvider creates a new network configuration provider
func NewNetworkConfigProvider(
	networkMappingRepo *database.NetworkMappingRepository,
	defaultNetworkID string,
) *NetworkConfigProvider {
	return &NetworkConfigProvider{
		networkMappingRepo: networkMappingRepo,
		defaultNetworkID:   defaultNetworkID,
	}
}

// GetNetworkIDForFailover resolves the appropriate network ID for VM creation
// This is the core method that applies the dual network mapping system
func (ncp *NetworkConfigProvider) GetNetworkIDForFailover(
	contextID string,
	failoverType FailoverType,
	vmwareNetworkName string,
) (string, error) {
	log.WithFields(log.Fields{
		"context_id":          contextID,
		"failover_type":       failoverType,
		"vmware_network_name": vmwareNetworkName,
	}).Debug("🌐 Resolving network ID for failover operation")

	// Step 1: Get all network mappings for this VM context
	mappings, err := ncp.networkMappingRepo.GetByContextID(contextID)
	if err != nil {
		log.WithError(err).WithField("context_id", contextID).
			Debug("No network mappings found, using default network")
		return ncp.defaultNetworkID, nil
	}

	if len(mappings) == 0 {
		log.WithField("context_id", contextID).
			Debug("No network mappings configured, using default network")
		return ncp.defaultNetworkID, nil
	}

	// Step 2: Find mapping for the specific VMware network
	var targetMapping *database.NetworkMapping
	for _, mapping := range mappings {
		if mapping.SourceNetworkName == vmwareNetworkName {
			// Check if this mapping matches the failover type
			if ncp.mappingMatchesFailoverType(mapping, failoverType) {
				targetMapping = &mapping
				break
			}
		}
	}

	// Step 3: If no specific mapping found, try to find any mapping for this failover type
	if targetMapping == nil {
		for _, mapping := range mappings {
			if ncp.mappingMatchesFailoverType(mapping, failoverType) {
				log.WithFields(log.Fields{
					"vmware_network":      vmwareNetworkName,
					"fallback_network":    mapping.SourceNetworkName,
					"destination_network": mapping.DestinationNetworkID,
				}).Debug("Using fallback network mapping")
				targetMapping = &mapping
				break
			}
		}
	}

	// Step 4: Apply the mapping or use default
	if targetMapping != nil {
		log.WithFields(log.Fields{
			"vmware_network":      vmwareNetworkName,
			"destination_network": targetMapping.DestinationNetworkID,
			"destination_name":    targetMapping.DestinationNetworkName,
			"is_test_network":     targetMapping.IsTestNetwork,
			"failover_type":       failoverType,
		}).Info("✅ Applied network mapping for failover")
		return targetMapping.DestinationNetworkID, nil
	}

	// Step 5: No suitable mapping found, use default with warning
	log.WithFields(log.Fields{
		"context_id":         contextID,
		"vmware_network":     vmwareNetworkName,
		"failover_type":      failoverType,
		"available_mappings": len(mappings),
		"default_network_id": ncp.defaultNetworkID,
	}).Warn("⚠️ No suitable network mapping found, using default OSSEA network")

	return ncp.defaultNetworkID, nil
}

// mappingMatchesFailoverType determines if a network mapping is appropriate for the failover type
func (ncp *NetworkConfigProvider) mappingMatchesFailoverType(
	mapping database.NetworkMapping,
	failoverType FailoverType,
) bool {
	switch failoverType {
	case FailoverTypeLive:
		// Live failover should use production networks (is_test_network = false)
		return !mapping.IsTestNetwork
	case FailoverTypeTest:
		// Test failover should use test networks (is_test_network = true)
		return mapping.IsTestNetwork
	default:
		log.WithField("failover_type", failoverType).
			Error("Unknown failover type in network mapping")
		return false
	}
}

// GetNetworkMappingSummary provides a summary of available network mappings for debugging
func (ncp *NetworkConfigProvider) GetNetworkMappingSummary(contextID string) (map[string]interface{}, error) {
	mappings, err := ncp.networkMappingRepo.GetByContextID(contextID)
	if err != nil {
		return nil, fmt.Errorf("failed to get network mappings: %w", err)
	}

	summary := map[string]interface{}{
		"context_id":       contextID,
		"total_mappings":   len(mappings),
		"production_count": 0,
		"test_count":       0,
		"networks":         make([]map[string]interface{}, 0),
	}

	productionCount := 0
	testCount := 0

	for _, mapping := range mappings {
		networkInfo := map[string]interface{}{
			"source_network":      mapping.SourceNetworkName,
			"destination_network": mapping.DestinationNetworkName,
			"destination_id":      mapping.DestinationNetworkID,
			"is_test_network":     mapping.IsTestNetwork,
		}
		summary["networks"] = append(summary["networks"].([]map[string]interface{}), networkInfo)

		if mapping.IsTestNetwork {
			testCount++
		} else {
			productionCount++
		}
	}

	summary["production_count"] = productionCount
	summary["test_count"] = testCount

	return summary, nil
}

// ValidateNetworkConfiguration checks if the VM has adequate network mappings for both failover types
func (ncp *NetworkConfigProvider) ValidateNetworkConfiguration(contextID string) (*NetworkValidationResult, error) {
	mappings, err := ncp.networkMappingRepo.GetByContextID(contextID)
	if err != nil {
		return nil, fmt.Errorf("failed to get network mappings: %w", err)
	}

	result := &NetworkValidationResult{
		ContextID:             contextID,
		TotalMappings:         len(mappings),
		HasProductionMappings: false,
		HasTestMappings:       false,
		MissingMappings:       make([]string, 0),
		Recommendations:       make([]string, 0),
	}

	// Analyze existing mappings
	productionNetworks := make(map[string]bool)
	testNetworks := make(map[string]bool)

	for _, mapping := range mappings {
		if mapping.IsTestNetwork {
			testNetworks[mapping.SourceNetworkName] = true
			result.HasTestMappings = true
		} else {
			productionNetworks[mapping.SourceNetworkName] = true
			result.HasProductionMappings = true
		}
	}

	// Check for missing dual mappings
	allSourceNetworks := make(map[string]bool)
	for _, mapping := range mappings {
		allSourceNetworks[mapping.SourceNetworkName] = true
	}

	for sourceNetwork := range allSourceNetworks {
		if !productionNetworks[sourceNetwork] {
			result.MissingMappings = append(result.MissingMappings,
				fmt.Sprintf("Production mapping for %s", sourceNetwork))
		}
		if !testNetworks[sourceNetwork] {
			result.MissingMappings = append(result.MissingMappings,
				fmt.Sprintf("Test mapping for %s", sourceNetwork))
		}
	}

	// Generate recommendations
	if len(mappings) == 0 {
		result.Recommendations = append(result.Recommendations,
			"Configure network mappings for this VM to enable proper failover operations")
	} else if !result.HasProductionMappings {
		result.Recommendations = append(result.Recommendations,
			"Add production network mappings to enable live failover")
	} else if !result.HasTestMappings {
		result.Recommendations = append(result.Recommendations,
			"Add test network mappings to enable test failover")
	} else if len(result.MissingMappings) > 0 {
		result.Recommendations = append(result.Recommendations,
			"Complete dual network mappings for all VMware networks")
	}

	result.IsComplete = len(result.MissingMappings) == 0 &&
		result.HasProductionMappings && result.HasTestMappings

	return result, nil
}

// NetworkValidationResult represents the validation status of network configuration
type NetworkValidationResult struct {
	ContextID             string   `json:"context_id"`
	TotalMappings         int      `json:"total_mappings"`
	HasProductionMappings bool     `json:"has_production_mappings"`
	HasTestMappings       bool     `json:"has_test_mappings"`
	IsComplete            bool     `json:"is_complete"`
	MissingMappings       []string `json:"missing_mappings"`
	Recommendations       []string `json:"recommendations"`
}

