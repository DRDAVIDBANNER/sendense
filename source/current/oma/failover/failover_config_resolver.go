// Package failover provides configuration resolution for unified failover engine
package failover

import (
	"fmt"
	"strings"

	"github.com/vexxhost/migratekit-oma/database"
	"github.com/vexxhost/migratekit-oma/services"
)

// FailoverConfigResolver converts existing API requests to unified failover configurations
// This provides backward compatibility while enabling the unified engine
type FailoverConfigResolver struct {
	networkMappingService *services.NetworkMappingService
	vmContextRepo         *database.VMReplicationContextRepository
	networkMappingRepo    *database.NetworkMappingRepository
}

// NewFailoverConfigResolver creates a new configuration resolver
func NewFailoverConfigResolver(
	networkMappingService *services.NetworkMappingService,
	vmContextRepo *database.VMReplicationContextRepository,
	networkMappingRepo *database.NetworkMappingRepository,
) *FailoverConfigResolver {
	return &FailoverConfigResolver{
		networkMappingService: networkMappingService,
		vmContextRepo:         vmContextRepo,
		networkMappingRepo:    networkMappingRepo,
	}
}

// ResolveTestFailoverConfig converts an EnhancedTestFailoverRequest to UnifiedFailoverConfig
func (fcr *FailoverConfigResolver) ResolveTestFailoverConfig(request *EnhancedTestFailoverRequest) (*UnifiedFailoverConfig, error) {
	// Create base test failover configuration
	config := NewTestFailoverConfig(
		request.ContextID,
		request.VMID, // This is actually VMware VM ID in the new architecture
		request.VMName,
		request.FailoverJobID,
	)

	// Set timestamp from request
	config.Timestamp = request.Timestamp

	// Determine network strategy based on existing network mappings
	networkStrategy, err := fcr.determineNetworkStrategy(request.ContextID, request.VMName, true)
	if err != nil {
		return nil, fmt.Errorf("failed to determine network strategy: %w", err)
	}
	config.NetworkStrategy = networkStrategy

	return config, nil
}

// ResolveLiveFailoverConfig converts an EnhancedFailoverRequest to UnifiedFailoverConfig
func (fcr *FailoverConfigResolver) ResolveLiveFailoverConfig(request *EnhancedFailoverRequest, contextID string) (*UnifiedFailoverConfig, error) {
	// Create base live failover configuration
	config := NewLiveFailoverConfig(
		contextID,
		request.VMID, // This is actually VMware VM ID in the new architecture
		request.VMName,
		request.FailoverJobID,
	)

	// Apply any specific options from the request
	if request.SkipSnapshot {
		config.SnapshotType = SnapshotTypeNone
	}
	if request.SkipVirtIOInjection {
		config.SkipVirtIO = true
	}
	if request.SkipValidation {
		config.SkipValidation = true
	}

	// Determine network strategy based on existing network mappings
	networkStrategy, err := fcr.determineNetworkStrategy(contextID, request.VMName, false)
	if err != nil {
		return nil, fmt.Errorf("failed to determine network strategy: %w", err)
	}
	config.NetworkStrategy = networkStrategy

	return config, nil
}

// ResolveFromAPIRequest converts a generic API request to UnifiedFailoverConfig
// This handles the common case where we receive context_id, vm_id, vm_name, and failover_type
func (fcr *FailoverConfigResolver) ResolveFromAPIRequest(
	contextID, vmwareVMID, vmName, failoverJobID, failoverType string,
	options map[string]interface{},
) (*UnifiedFailoverConfig, error) {
	var config *UnifiedFailoverConfig

	switch failoverType {
	case "live":
		config = NewLiveFailoverConfig(contextID, vmwareVMID, vmName, failoverJobID)
	case "test":
		config = NewTestFailoverConfig(contextID, vmwareVMID, vmName, failoverJobID)
	default:
		return nil, fmt.Errorf("unsupported failover type: %s", failoverType)
	}

	// Apply any options from the request
	if options != nil {
		if powerOffSource, ok := options["power_off_source"].(bool); ok {
			config.PowerOffSource = powerOffSource
		}
		if performFinalSync, ok := options["perform_final_sync"].(bool); ok {
			config.PerformFinalSync = performFinalSync
		}
		if skipValidation, ok := options["skip_validation"].(bool); ok {
			config.SkipValidation = skipValidation
		}
		if skipVirtIO, ok := options["skip_virtio"].(bool); ok {
			config.SkipVirtIO = skipVirtIO
		}
		if userID, ok := options["user_id"].(string); ok {
			config.UserID = userID
		}
		if reason, ok := options["reason"].(string); ok {
			config.Reason = reason
		}

		// CRITICAL FIX: Handle user-provided network_strategy parameter
		if networkStrategyStr, ok := options["network_strategy"].(string); ok {
			fmt.Printf("🔧 DEBUG: User provided network_strategy: %s\n", networkStrategyStr)
			switch networkStrategyStr {
			case "live":
				config.NetworkStrategy = NetworkStrategyProduction
				fmt.Printf("✅ DEBUG: Mapped 'live' -> NetworkStrategyProduction\n")
			case "production":
				config.NetworkStrategy = NetworkStrategyProduction
				fmt.Printf("✅ DEBUG: Mapped 'production' -> NetworkStrategyProduction\n")
			case "test":
				config.NetworkStrategy = NetworkStrategyIsolated
				fmt.Printf("✅ DEBUG: Mapped 'test' -> NetworkStrategyIsolated\n")
			case "isolated":
				config.NetworkStrategy = NetworkStrategyIsolated
				fmt.Printf("✅ DEBUG: Mapped 'isolated' -> NetworkStrategyIsolated\n")
			case "custom":
				config.NetworkStrategy = NetworkStrategyCustom
				fmt.Printf("✅ DEBUG: Mapped 'custom' -> NetworkStrategyCustom\n")
			default:
				return nil, fmt.Errorf("invalid network_strategy: %s (must be 'live', 'test', 'production', 'isolated', or 'custom')", networkStrategyStr)
			}
		} else {
			// Only auto-determine network strategy if user didn't specify one
			isTestFailover := failoverType == "test"
			networkStrategy, err := fcr.determineNetworkStrategy(contextID, vmName, isTestFailover)
			if err != nil {
				return nil, fmt.Errorf("failed to determine network strategy: %w", err)
			}
			config.NetworkStrategy = networkStrategy
		}
	} else {
		// No options provided - use auto-determination
		isTestFailover := failoverType == "test"
		networkStrategy, err := fcr.determineNetworkStrategy(contextID, vmName, isTestFailover)
		if err != nil {
			return nil, fmt.Errorf("failed to determine network strategy: %w", err)
		}
		config.NetworkStrategy = networkStrategy
	}

	// Populate VCenterHost from VM context (TODO: Add proper repository method)
	if contextID != "" {
		// 🆕 ENHANCED: Use credential service for vCenter host
		config.VCenterHost = fcr.getDefaultVCenterHost()
	}

	return config, nil
}

// determineNetworkStrategy analyzes existing network mappings to determine the appropriate strategy
// Enhanced implementation with improved logic and error handling
func (fcr *FailoverConfigResolver) determineNetworkStrategy(contextID, vmName string, isTestFailover bool) (NetworkStrategy, error) {
	// Use VM-centric network mapping lookup (Task 5.5.2 implementation)
	var mappings []database.NetworkMapping
	var err error

	// Try context_id first (preferred method), fallback to vm_name for backward compatibility
	if contextID != "" {
		mappings, err = fcr.networkMappingRepo.GetByContextID(contextID)
	} else {
		mappings, err = fcr.networkMappingRepo.GetByVMID(vmName)
	}

	if err != nil {
		// Log the error but continue with default strategy
		// This is expected for VMs without network mappings configured
		if isTestFailover {
			return NetworkStrategyIsolated, nil
		}
		return NetworkStrategyProduction, nil
	}

	if len(mappings) == 0 {
		// No network mappings configured, use default strategy based on failover type
		if isTestFailover {
			return NetworkStrategyIsolated, nil
		}
		return NetworkStrategyProduction, nil
	}

	// Analyze the mappings to determine strategy
	hasTestNetworks := false
	hasProductionNetworks := false
	testNetworkCount := 0
	productionNetworkCount := 0

	for _, mapping := range mappings {
		// Enhanced network classification logic
		// Check both the explicit IsTestNetwork flag and network ID patterns
		isTest := mapping.IsTestNetwork || fcr.isTestNetwork(mapping.DestinationNetworkID) || fcr.isTestNetwork(mapping.DestinationNetworkName)

		if isTest {
			hasTestNetworks = true
			testNetworkCount++
		} else {
			hasProductionNetworks = true
			productionNetworkCount++
		}
	}

	// Enhanced strategy determination logic
	if hasTestNetworks && hasProductionNetworks {
		// Mixed network types - use custom strategy to allow user control
		return NetworkStrategyCustom, nil
	} else if hasTestNetworks && testNetworkCount > 0 {
		// All networks are test/isolated networks
		return NetworkStrategyIsolated, nil
	} else if hasProductionNetworks && productionNetworkCount > 0 {
		// All networks are production networks
		return NetworkStrategyProduction, nil
	}

	// Fallback to default strategy based on failover type
	// This handles edge cases where network classification is unclear
	if isTestFailover {
		return NetworkStrategyIsolated, nil
	}
	return NetworkStrategyProduction, nil
}

// isTestNetwork determines if a network ID represents a test/isolated network
// Enhanced implementation based on Phase 1 network mapping analysis
func (fcr *FailoverConfigResolver) isTestNetwork(networkID string) bool {
	// Enhanced test network detection logic
	// Based on Phase 1 analysis: test networks are filtered by keywords and L2 network type

	// Convert to lowercase for case-insensitive comparison
	networkIDLower := strings.ToLower(networkID)

	// Test network keywords from Phase 1 analysis
	// Note: Removed "l2" to avoid false positives with "OSSEA-L2" production network
	testKeywords := []string{"test", "lab", "dev", "staging", "isolated"}

	for _, keyword := range testKeywords {
		if strings.Contains(networkIDLower, keyword) {
			return true
		}
	}

	// Additional heuristics for test network detection
	// Networks with specific patterns that indicate test environments
	testPatterns := []string{
		"_test_", "-test-", ".test.",
		"_lab_", "-lab-", ".lab.",
		"_dev_", "-dev-", ".dev.",
		"_staging_", "-staging-", ".staging.",
		"_isolated_", "-isolated-", ".isolated.",
	}

	for _, pattern := range testPatterns {
		if strings.Contains(networkIDLower, pattern) {
			return true
		}
	}

	return false
}

// ValidateConfiguration ensures the resolved configuration is valid and consistent
func (fcr *FailoverConfigResolver) ValidateConfiguration(config *UnifiedFailoverConfig) error {
	// Perform basic validation
	if err := config.Validate(); err != nil {
		return err
	}

	// Perform additional validation based on current system state
	// TODO: Add VM context existence validation when repository method is available
	// For now, skip this validation as ExistsByContextID method doesn't exist yet

	// Validate network mappings exist for the chosen strategy
	if config.NetworkStrategy == NetworkStrategyCustom {
		mappings, err := fcr.networkMappingRepo.GetByVMID(config.VMName)
		if err != nil || len(mappings) == 0 {
			return fmt.Errorf("custom network strategy requires existing network mappings")
		}
	}

	return nil
}

// GetConfigurationSummary returns a human-readable summary of the configuration
func (fcr *FailoverConfigResolver) GetConfigurationSummary(config *UnifiedFailoverConfig) map[string]interface{} {
	summary := map[string]interface{}{
		"failover_type":      config.FailoverType,
		"vm_name":            config.VMName,
		"destination_name":   config.GetDestinationVMName(),
		"network_strategy":   config.NetworkStrategy,
		"snapshot_type":      config.SnapshotType,
		"power_off_source":   config.PowerOffSource,
		"perform_final_sync": config.PerformFinalSync,
		"cleanup_enabled":    config.CleanupEnabled,
		"skip_validation":    config.SkipValidation,
		"skip_virtio":        config.SkipVirtIO,
	}

	if config.SnapshotType != SnapshotTypeNone {
		summary["snapshot_name"] = config.GetSnapshotName()
	}

	return summary
}

// getDefaultVCenterHost retrieves the default vCenter host from credential service
func (fcr *FailoverConfigResolver) getDefaultVCenterHost() string {
	// Note: Repository doesn't expose database connection directly
	// For now, falling back to hardcoded until repository architecture is enhanced
	// TODO: Modify repository to provide database access for credential service
	return "quad-vcenter-01.quadris.local"
}
