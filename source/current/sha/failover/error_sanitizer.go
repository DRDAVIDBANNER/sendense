// Package failover provides error message sanitization for user-friendly display
// Converts technical implementation details to actionable, understandable messages
package failover

import (
	"fmt"
	"strings"
)

// SanitizedMessage provides user-friendly error messaging with actionable guidance
type SanitizedMessage struct {
	UserMessage     string   // Clean, user-friendly message for GUI display
	TechnicalDetail string   // Full technical details (admin/logs only)
	Category        string   // Error category: compatibility, network, storage, configuration, connectivity
	ActionableSteps []string // What user can do to resolve or work around
	Severity        string   // info, warning, error, critical
}

// SanitizeFailoverError converts technical failover errors to user-friendly messages
// This is the main entry point for error sanitization
func SanitizeFailoverError(stepName string, err error) SanitizedMessage {
	if err == nil {
		return SanitizedMessage{
			UserMessage:     "Operation completed successfully",
			TechnicalDetail: "",
			Category:        "success",
			Severity:        "info",
		}
	}

	errorStr := err.Error()

	// Route to appropriate sanitizer based on step name
	switch stepName {
	case "virtio-driver-injection", "driver-injection":
		return sanitizeDriverInjectionError(errorStr)
	case "vm-creation", "destination-vm-creation":
		return sanitizeVMCreationError(errorStr)
	case "network-configuration", "network-mapping":
		return sanitizeNetworkError(errorStr)
	case "volume-attachment", "volume-operations", "volume-detachment":
		return sanitizeVolumeError(errorStr)
	case "source-vm-power-off", "source-vm-power-on", "vm-startup-and-validation":
		return sanitizeVMPowerError(errorStr)
	case "cloudstack-snapshot-creation", "multi-volume-snapshot-creation":
		return sanitizeSnapshotError(errorStr)
	case "validation", "pre-flight-validation":
		return sanitizeValidationError(errorStr)
	default:
		return sanitizeGenericError(stepName, errorStr)
	}
}

// sanitizeDriverInjectionError sanitizes VirtIO/virt-v2v related errors
// Hides: virt-v2v, VirtIO, virtio-win.iso, device paths, script names
// Shows: User-friendly compatibility messages
func sanitizeDriverInjectionError(errMsg string) SanitizedMessage {
	lowerMsg := strings.ToLower(errMsg)

	// Pattern 1: virt-v2v tool errors
	if strings.Contains(lowerMsg, "virt-v2v") || strings.Contains(lowerMsg, "v2v") {
		if strings.Contains(lowerMsg, "exit code") || strings.Contains(lowerMsg, "failed") {
			return SanitizedMessage{
				UserMessage: "Driver injection failed - VM may not be compatible with automated conversion",
				TechnicalDetail: errMsg,
				Category: "compatibility",
				Severity: "error",
				ActionableSteps: []string{
					"Try live failover instead (no driver modification required)",
					"Verify VM is running Windows operating system",
					"Ensure VM disk is not corrupted",
					"Check VM is in a bootable state",
				},
			}
		}
		return SanitizedMessage{
			UserMessage: "Driver injection process error - automated tool encountered issue",
			TechnicalDetail: errMsg,
			Category: "compatibility",
			Severity: "error",
			ActionableSteps: []string{
				"Try live failover instead",
				"Contact support for manual driver installation",
			},
		}
	}

	// Pattern 2: VirtIO driver errors
	if strings.Contains(lowerMsg, "virtio") {
		if strings.Contains(lowerMsg, "not found") || strings.Contains(lowerMsg, "missing") {
			return SanitizedMessage{
				UserMessage: "Required driver package not available on system",
				TechnicalDetail: errMsg,
				Category: "configuration",
				Severity: "error",
				ActionableSteps: []string{
					"Contact administrator - driver package needs to be installed",
					"Try live failover as a workaround",
				},
			}
		}
		return SanitizedMessage{
			UserMessage: "KVM driver installation failed - compatibility issue",
			TechnicalDetail: errMsg,
			Category: "compatibility",
			Severity: "error",
			ActionableSteps: []string{
				"Try live failover (no driver modification)",
				"Verify VM is Windows-based",
			},
		}
	}

	// Pattern 3: Device/disk access errors
	if strings.Contains(lowerMsg, "/dev/") || strings.Contains(lowerMsg, "device") {
		return SanitizedMessage{
			UserMessage: "Storage access error during driver preparation",
			TechnicalDetail: errMsg,
			Category: "storage",
			Severity: "error",
			ActionableSteps: []string{
				"Verify storage volumes are attached",
				"Check disk is not in use by another process",
				"Try operation again",
			},
		}
	}

	// Pattern 4: Permission errors
	if strings.Contains(lowerMsg, "permission") || strings.Contains(lowerMsg, "denied") {
		return SanitizedMessage{
			UserMessage: "System permissions error during driver preparation",
			TechnicalDetail: errMsg,
			Category: "configuration",
			Severity: "error",
			ActionableSteps: []string{
				"Contact administrator - system permissions need adjustment",
			},
		}
	}

	// Default driver injection error
	return SanitizedMessage{
		UserMessage: "Driver preparation failed - unable to inject required drivers",
		TechnicalDetail: errMsg,
		Category: "compatibility",
		Severity: "error",
		ActionableSteps: []string{
			"Try live failover instead (no driver modification required)",
			"Verify VM is Windows-based and accessible",
		},
	}
}

// sanitizeVMCreationError sanitizes CloudStack/OSSEA VM creation errors
func sanitizeVMCreationError(errMsg string) SanitizedMessage {
	lowerMsg := strings.ToLower(errMsg)

	// Pattern 1: Network issues
	if strings.Contains(lowerMsg, "network") {
		if strings.Contains(lowerMsg, "not found") {
			return SanitizedMessage{
				UserMessage: "Network configuration error - specified network not available",
				TechnicalDetail: errMsg,
				Category: "network",
				Severity: "error",
				ActionableSteps: []string{
					"Verify network mapping is configured",
					"Check that target network exists in destination platform",
					"Update network mapping in settings",
				},
			}
		}
		return SanitizedMessage{
			UserMessage: "Network configuration error during VM creation",
			TechnicalDetail: errMsg,
			Category: "network",
			Severity: "error",
			ActionableSteps: []string{
				"Review network mapping configuration",
				"Verify target network is accessible",
			},
		}
	}

	// Pattern 2: Resource constraints
	if strings.Contains(lowerMsg, "insufficient") || strings.Contains(lowerMsg, "quota") || 
	   strings.Contains(lowerMsg, "capacity") {
		return SanitizedMessage{
			UserMessage: "Insufficient resources on destination platform",
			TechnicalDetail: errMsg,
			Category: "platform",
			Severity: "error",
			ActionableSteps: []string{
				"Check available CPU, memory, and storage on destination",
				"Free up resources or adjust VM specifications",
				"Contact administrator if quota limits need adjustment",
			},
		}
	}

	// Pattern 3: Template/offering errors
	if strings.Contains(lowerMsg, "template") || strings.Contains(lowerMsg, "offering") {
		return SanitizedMessage{
			UserMessage: "Platform configuration error - VM template or service offering issue",
			TechnicalDetail: errMsg,
			Category: "configuration",
			Severity: "error",
			ActionableSteps: []string{
				"Verify platform configuration is complete",
				"Check template and service offering are available",
				"Contact administrator to review platform settings",
			},
		}
	}

	// Pattern 4: API errors
	if strings.Contains(lowerMsg, "api") || strings.Contains(lowerMsg, "cloudstack") {
		return SanitizedMessage{
			UserMessage: "Platform communication error - could not create VM",
			TechnicalDetail: errMsg,
			Category: "platform",
			Severity: "error",
			ActionableSteps: []string{
				"Check platform connectivity",
				"Verify platform credentials are valid",
				"Try operation again",
			},
		}
	}

	// Default VM creation error
	return SanitizedMessage{
		UserMessage: "VM creation failed on destination platform",
		TechnicalDetail: errMsg,
		Category: "platform",
		Severity: "error",
		ActionableSteps: []string{
			"Verify platform is accessible and properly configured",
			"Check resource availability on destination",
			"Review platform logs for additional details",
		},
	}
}

// sanitizeNetworkError sanitizes network configuration errors
func sanitizeNetworkError(errMsg string) SanitizedMessage {
	lowerMsg := strings.ToLower(errMsg)

	// Pattern 1: Network not found
	if strings.Contains(lowerMsg, "not found") {
		return SanitizedMessage{
			UserMessage: "Network not found - target network is not available",
			TechnicalDetail: errMsg,
			Category: "network",
			Severity: "error",
			ActionableSteps: []string{
				"Verify network mapping is configured for this VM",
				"Check that target network exists on destination platform",
				"Update network mapping in VM settings",
			},
		}
	}

	// Pattern 2: Network mapping missing
	if strings.Contains(lowerMsg, "mapping") || strings.Contains(lowerMsg, "not mapped") {
		return SanitizedMessage{
			UserMessage: "Network mapping not configured for this VM",
			TechnicalDetail: errMsg,
			Category: "configuration",
			Severity: "error",
			ActionableSteps: []string{
				"Configure network mapping before attempting failover",
				"Go to VM settings → Network Mapping",
				"Map each source network to a destination network",
			},
		}
	}

	// Pattern 3: Network ID resolution
	if strings.Contains(lowerMsg, "network id") || strings.Contains(lowerMsg, "networkid") {
		return SanitizedMessage{
			UserMessage: "Network configuration error - cannot resolve network identifier",
			TechnicalDetail: errMsg,
			Category: "configuration",
			Severity: "error",
			ActionableSteps: []string{
				"Verify network mapping configuration",
				"Check that target network exists",
				"Reconfigure network mapping if needed",
			},
		}
	}

	// Default network error
	return SanitizedMessage{
		UserMessage: "Network configuration error during failover",
		TechnicalDetail: errMsg,
		Category: "network",
		Severity: "error",
		ActionableSteps: []string{
			"Review network mapping configuration",
			"Verify target networks are accessible",
			"Check platform network connectivity",
		},
	}
}

// sanitizeVolumeError sanitizes volume/storage related errors
func sanitizeVolumeError(errMsg string) SanitizedMessage {
	lowerMsg := strings.ToLower(errMsg)

	// Pattern 1: Volume attachment errors
	if strings.Contains(lowerMsg, "already attached") || strings.Contains(lowerMsg, "is attached") {
		return SanitizedMessage{
			UserMessage: "Storage volume is already in use",
			TechnicalDetail: errMsg,
			Category: "storage",
			Severity: "error",
			ActionableSteps: []string{
				"Wait for previous operation to complete",
				"Check if test VM is still running",
				"Try rollback operation to clean up",
			},
		}
	}

	// Pattern 2: Volume not found
	if strings.Contains(lowerMsg, "not found") || strings.Contains(lowerMsg, "does not exist") {
		return SanitizedMessage{
			UserMessage: "Storage volume not found",
			TechnicalDetail: errMsg,
			Category: "storage",
			Severity: "error",
			ActionableSteps: []string{
				"Verify VM has completed at least one replication",
				"Check storage volumes exist on destination platform",
				"Try running a new replication job",
			},
		}
	}

	// Pattern 3: Volume detachment errors
	if strings.Contains(lowerMsg, "detach") {
		return SanitizedMessage{
			UserMessage: "Storage detachment error - volume cannot be disconnected",
			TechnicalDetail: errMsg,
			Category: "storage",
			Severity: "error",
			ActionableSteps: []string{
				"Ensure VM is powered off",
				"Check volume is not locked by another process",
				"Try operation again after a few moments",
			},
		}
	}

	// Pattern 4: Volume daemon errors
	if strings.Contains(lowerMsg, "volume daemon") || strings.Contains(lowerMsg, "daemon") {
		return SanitizedMessage{
			UserMessage: "Storage management service error",
			TechnicalDetail: errMsg,
			Category: "platform",
			Severity: "error",
			ActionableSteps: []string{
				"Contact administrator - storage service may need attention",
				"Try operation again",
			},
		}
	}

	// Default volume error
	return SanitizedMessage{
		UserMessage: "Storage operation failed",
		TechnicalDetail: errMsg,
		Category: "storage",
		Severity: "error",
		ActionableSteps: []string{
			"Verify storage volumes are accessible",
			"Check destination platform storage health",
			"Try operation again",
		},
	}
}

// sanitizeVMPowerError sanitizes VM power management errors
func sanitizeVMPowerError(errMsg string) SanitizedMessage {
	lowerMsg := strings.ToLower(errMsg)

	// Pattern 1: SNA connectivity
	if strings.Contains(lowerMsg, "vma") && (strings.Contains(lowerMsg, "unreachable") || 
	   strings.Contains(lowerMsg, "connection") || strings.Contains(lowerMsg, "timeout")) {
		return SanitizedMessage{
			UserMessage: "Cannot connect to source environment",
			TechnicalDetail: errMsg,
			Category: "connectivity",
			Severity: "error",
			ActionableSteps: []string{
				"Verify source environment is online and accessible",
				"Check network connectivity between source and destination",
				"Try operation again once connectivity is restored",
			},
		}
	}

	// Pattern 2: VM not found
	if strings.Contains(lowerMsg, "not found") || strings.Contains(lowerMsg, "does not exist") {
		return SanitizedMessage{
			UserMessage: "Source VM not found",
			TechnicalDetail: errMsg,
			Category: "configuration",
			Severity: "error",
			ActionableSteps: []string{
				"Verify VM still exists in source environment",
				"Check VM has not been renamed or deleted",
				"Refresh VM discovery",
			},
		}
	}

	// Pattern 3: Power state errors
	if strings.Contains(lowerMsg, "power") || strings.Contains(lowerMsg, "already") {
		return SanitizedMessage{
			UserMessage: "VM power state change failed - VM may already be in requested state",
			TechnicalDetail: errMsg,
			Category: "platform",
			Severity: "warning",
			ActionableSteps: []string{
				"Verify current VM power state",
				"Operation may have already succeeded",
				"Try operation again if needed",
			},
		}
	}

	// Pattern 4: Permission errors
	if strings.Contains(lowerMsg, "permission") || strings.Contains(lowerMsg, "credential") {
		return SanitizedMessage{
			UserMessage: "Insufficient permissions to manage VM power state",
			TechnicalDetail: errMsg,
			Category: "configuration",
			Severity: "error",
			ActionableSteps: []string{
				"Verify source environment credentials are valid",
				"Check user has permissions to manage VM power",
				"Update credentials if needed",
			},
		}
	}

	// Default power error
	return SanitizedMessage{
		UserMessage: "VM power management operation failed",
		TechnicalDetail: errMsg,
		Category: "platform",
		Severity: "error",
		ActionableSteps: []string{
			"Verify VM is accessible",
			"Check VM is not locked or in maintenance",
			"Try operation again",
		},
	}
}

// sanitizeSnapshotError sanitizes snapshot creation/management errors
func sanitizeSnapshotError(errMsg string) SanitizedMessage {
	lowerMsg := strings.ToLower(errMsg)

	// Pattern 1: Snapshot creation failed
	if strings.Contains(lowerMsg, "create") || strings.Contains(lowerMsg, "creation") {
		return SanitizedMessage{
			UserMessage: "Backup snapshot creation failed",
			TechnicalDetail: errMsg,
			Category: "storage",
			Severity: "error",
			ActionableSteps: []string{
				"Check available storage space on destination platform",
				"Verify storage system is healthy",
				"Try operation again after freeing up space",
			},
		}
	}

	// Pattern 2: Snapshot not found
	if strings.Contains(lowerMsg, "not found") {
		return SanitizedMessage{
			UserMessage: "Backup snapshot not found",
			TechnicalDetail: errMsg,
			Category: "storage",
			Severity: "error",
			ActionableSteps: []string{
				"Snapshot may have been automatically cleaned up",
				"Try creating a new test failover",
			},
		}
	}

	// Pattern 3: Revert/rollback errors
	if strings.Contains(lowerMsg, "revert") || strings.Contains(lowerMsg, "rollback") {
		return SanitizedMessage{
			UserMessage: "Snapshot rollback failed - cannot restore to previous state",
			TechnicalDetail: errMsg,
			Category: "storage",
			Severity: "critical",
			ActionableSteps: []string{
				"Contact administrator immediately",
				"Do not attempt further operations on this VM",
				"Manual intervention may be required",
			},
		}
	}

	// Default snapshot error
	return SanitizedMessage{
		UserMessage: "Backup snapshot operation failed",
		TechnicalDetail: errMsg,
		Category: "storage",
		Severity: "error",
		ActionableSteps: []string{
			"Check storage system health",
			"Verify sufficient space available",
			"Try operation again",
		},
	}
}

// sanitizeValidationError sanitizes pre-flight validation errors
func sanitizeValidationError(errMsg string) SanitizedMessage {
	lowerMsg := strings.ToLower(errMsg)

	// Pattern 1: Replication not complete
	if strings.Contains(lowerMsg, "replication") || strings.Contains(lowerMsg, "not replicated") {
		return SanitizedMessage{
			UserMessage: "VM is not ready for failover - replication required first",
			TechnicalDetail: errMsg,
			Category: "configuration",
			Severity: "warning",
			ActionableSteps: []string{
				"Run a replication job before attempting failover",
				"Verify VM status is 'ready_for_failover'",
			},
		}
	}

	// Pattern 2: Network mapping missing
	if strings.Contains(lowerMsg, "network") && strings.Contains(lowerMsg, "mapping") {
		return SanitizedMessage{
			UserMessage: "Network mapping not configured - required for failover",
			TechnicalDetail: errMsg,
			Category: "configuration",
			Severity: "warning",
			ActionableSteps: []string{
				"Configure network mapping before failover",
				"Go to VM settings → Network Mapping",
				"Map each source network to destination network",
			},
		}
	}

	// Pattern 3: VM state invalid
	if strings.Contains(lowerMsg, "state") || strings.Contains(lowerMsg, "status") {
		return SanitizedMessage{
			UserMessage: "VM is not in valid state for this operation",
			TechnicalDetail: errMsg,
			Category: "configuration",
			Severity: "warning",
			ActionableSteps: []string{
				"Check current VM status",
				"Complete any pending operations first",
				"Verify VM is in 'ready_for_failover' state",
			},
		}
	}

	// Default validation error
	return SanitizedMessage{
		UserMessage: "Pre-flight validation failed - VM not ready for operation",
		TechnicalDetail: errMsg,
		Category: "configuration",
		Severity: "warning",
		ActionableSteps: []string{
			"Review VM configuration and status",
			"Ensure all prerequisites are met",
			"Check validation errors for specific issues",
		},
	}
}

// sanitizeGenericError provides generic sanitization for unclassified errors
func sanitizeGenericError(stepName string, errMsg string) SanitizedMessage {
	lowerMsg := strings.ToLower(errMsg)

	// Try to determine category from error message
	category := "unknown"
	severity := "error"
	actionableSteps := []string{
		"Review operation details for more information",
		"Try operation again",
		"Contact administrator if issue persists",
	}

	// Generic pattern matching
	switch {
	case strings.Contains(lowerMsg, "timeout"):
		category = "connectivity"
		actionableSteps = []string{
			"Check network connectivity",
			"Try operation again",
			"Increase timeout if issue persists",
		}
	case strings.Contains(lowerMsg, "not found"):
		category = "configuration"
		actionableSteps = []string{
			"Verify required resources exist",
			"Check configuration is complete",
		}
	case strings.Contains(lowerMsg, "permission") || strings.Contains(lowerMsg, "denied"):
		category = "configuration"
		actionableSteps = []string{
			"Contact administrator - permissions need adjustment",
		}
	}

	// Create user-friendly message from step name
	friendlyStepName := GetUserFriendlyStepName(stepName)
	userMessage := fmt.Sprintf("%s failed", friendlyStepName)
	if friendlyStepName == stepName {
		// Step name wasn't mapped, make it more generic
		userMessage = "Operation failed"
	}

	return SanitizedMessage{
		UserMessage:     userMessage,
		TechnicalDetail: errMsg,
		Category:        category,
		Severity:        severity,
		ActionableSteps: actionableSteps,
	}
}

// GetActionableSuggestions returns generic actionable suggestions based on error category
// Used when specific error pattern doesn't match
func GetActionableSuggestions(category string) []string {
	suggestions := map[string][]string{
		"compatibility": {
			"Try live failover instead (no driver modification)",
			"Verify VM operating system compatibility",
			"Check VM is not corrupted",
		},
		"network": {
			"Verify network mapping configuration",
			"Check target network is accessible",
			"Review network settings",
		},
		"storage": {
			"Verify storage volumes are accessible",
			"Check sufficient storage space available",
			"Review storage platform health",
		},
		"platform": {
			"Verify destination platform connectivity",
			"Check platform credentials are valid",
			"Review platform logs for details",
		},
		"connectivity": {
			"Check network connectivity",
			"Verify source environment is accessible",
			"Try operation again",
		},
		"configuration": {
			"Review configuration settings",
			"Verify all required fields are set",
			"Check prerequisites are met",
		},
	}

	if steps, exists := suggestions[category]; exists {
		return steps
	}

	// Default suggestions
	return []string{
		"Try operation again",
		"Contact administrator if issue persists",
	}
}


