// Package failover provides user-friendly step name mapping
// Converts internal technical step names to GUI-friendly display names
package failover

import (
	"fmt"
	"strings"
)

// GetUserFriendlyStepName converts internal step names to user-friendly display names
// Hides implementation details while providing clear progress indication
func GetUserFriendlyStepName(internalStep string) string {
	stepNames := map[string]string{
		// Validation and preparation steps
		"validation":              "Pre-flight Validation",
		"pre-flight-validation":   "Pre-flight Validation",
		"failover-job-creation":   "Initializing Failover Operation",
		
		// Source VM operations
		"source-vm-power-off":     "Powering Off Source VM",
		"source-vm-power-on":      "Powering On Source VM",
		"source-vm-discovery":     "Discovering Source VM",
		
		// Synchronization steps
		"final-sync":              "Final Data Synchronization",
		"incremental-replication": "Incremental Data Transfer",
		"data-sync":               "Data Synchronization",
		
		// Snapshot operations
		"multi-volume-snapshot-creation": "Creating Backup Snapshots",
		"cloudstack-snapshot-creation":   "Creating Backup Snapshot",
		"cloudstack-volume-snapshot-creation": "Creating Storage Backup",
		"linstor-snapshot-creation":      "Creating Backup Snapshots",
		"snapshot-creation":              "Creating Backup",
		
		// Volume operations
		"volume-mode-switch-oma":      "Preparing Storage Volumes",
		"volume-mode-switch-failover": "Configuring Storage Mode",
		"volume-attachment":           "Attaching Storage Volumes",
		"volume-detachment":           "Detaching Storage Volumes",
		"volume-reattachment-to-oma":  "Restoring Storage Configuration",
		"volume-operations":           "Storage Operations",
		
		// Driver injection (SANITIZED - no virt-v2v/VirtIO mention)
		"virtio-driver-injection":     "Preparing Drivers for Compatibility",
		"driver-injection":            "Installing Required Drivers",
		"virtio-injection":            "Driver Preparation",
		
		// VM creation and configuration
		"vm-creation":                 "Creating Destination VM",
		"destination-vm-creation":     "Creating Destination VM",
		"vm-startup-and-validation":   "Starting and Validating VM",
		"vm-startup":                  "Starting VM",
		"vm-validation":               "Validating VM",
		
		// Network operations
		"network-configuration":       "Configuring Network Adapters",
		"network-mapping":             "Mapping Network Configuration",
		"network-setup":               "Network Setup",
		
		// Cleanup and rollback operations
		"test-vm-shutdown":            "Shutting Down Test VM",
		"test-vm-deletion":            "Removing Test VM",
		"vm-deletion":                 "Removing VM",
		"cloudstack-snapshot-rollback": "Rolling Back to Backup",
		"cloudstack-snapshot-deletion": "Cleaning Up Backups",
		"snapshot-cleanup":            "Cleaning Up Backups",
		"multi-volume-cleanup":        "Cleaning Up Storage Snapshots",
		
		// Status and finalization
		"status-update":               "Updating VM Status",
		"failover-job-status-update":  "Updating Operation Status",
		"vm-context-status-update":    "Finalizing VM State",
		"completion":                  "Completing Operation",
		
		// System operations
		"ossea-client-initialization": "Initializing Platform Connection",
		"failover-job-retrieval":      "Retrieving Operation Details",
		"cleanup":                     "Cleanup Operations",
		"finalization":                "Finalizing",
	}

	if friendlyName, exists := stepNames[internalStep]; exists {
		return friendlyName
	}

	// Fallback: Try to make it more readable
	// Convert kebab-case to Title Case
	words := strings.Split(internalStep, "-")
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(word[:1]) + word[1:]
		}
	}
	
	return strings.Join(words, " ")
}

// GetStepCategory returns the category of a step for grouping/display
func GetStepCategory(internalStep string) string {
	categories := map[string]string{
		// Setup phase
		"validation":              "setup",
		"pre-flight-validation":   "setup",
		"failover-job-creation":   "setup",
		"ossea-client-initialization": "setup",
		"failover-job-retrieval":  "setup",
		
		// Preparation phase
		"source-vm-power-off":     "preparation",
		"final-sync":              "preparation",
		"multi-volume-snapshot-creation": "preparation",
		"cloudstack-snapshot-creation": "preparation",
		
		// Execution phase  
		"virtio-driver-injection": "execution",
		"vm-creation":             "execution",
		"volume-attachment":       "execution",
		"network-configuration":   "execution",
		"vm-startup-and-validation": "execution",
		
		// Cleanup phase
		"test-vm-shutdown":        "cleanup",
		"test-vm-deletion":        "cleanup",
		"volume-detachment":       "cleanup",
		"snapshot-cleanup":        "cleanup",
		"multi-volume-cleanup":    "cleanup",
		"cloudstack-snapshot-rollback": "cleanup",
		"cloudstack-snapshot-deletion": "cleanup",
		"volume-reattachment-to-oma": "cleanup",
		
		// Finalization phase
		"status-update":           "finalization",
		"vm-context-status-update": "finalization",
		"completion":              "finalization",
	}

	if category, exists := categories[internalStep]; exists {
		return category
	}
	return "execution" // Default to execution phase
}

// GetStepDescription provides detailed description for a step
func GetStepDescription(internalStep string) string {
	descriptions := map[string]string{
		"validation": "Checking VM is ready for failover operation",
		"pre-flight-validation": "Validating all prerequisites are met",
		
		"source-vm-power-off": "Shutting down source VM to ensure data consistency",
		"source-vm-power-on": "Restarting source VM after failover",
		
		"final-sync": "Synchronizing final changes before failover",
		
		"multi-volume-snapshot-creation": "Creating backup snapshots of all VM disks for rollback protection",
		"cloudstack-snapshot-creation": "Creating backup snapshot for rollback protection",
		
		"virtio-driver-injection": "Preparing VM for KVM virtualization platform",
		"driver-injection": "Installing drivers required for destination platform",
		
		"vm-creation": "Creating VM on destination platform with specified configuration",
		"volume-attachment": "Connecting storage volumes to destination VM",
		"network-configuration": "Configuring network adapters for destination environment",
		"vm-startup-and-validation": "Starting VM and verifying it boots correctly",
		
		"test-vm-shutdown": "Shutting down test VM to prepare for cleanup",
		"test-vm-deletion": "Removing test VM from destination platform",
		"volume-detachment": "Disconnecting storage volumes from VM",
		"cloudstack-snapshot-rollback": "Restoring volumes to pre-failover state",
		"cloudstack-snapshot-deletion": "Removing temporary backup snapshots",
		"multi-volume-cleanup": "Cleaning up all backup snapshots",
		
		"status-update": "Updating VM status in system",
		"completion": "Finalizing operation and cleaning up resources",
	}

	if description, exists := descriptions[internalStep]; exists {
		return description
	}
	return "Performing operation step"
}

// GetStepIcon returns an emoji/icon for visual step representation
func GetStepIcon(internalStep string, status string) string {
	// Status-based icons
	switch status {
	case "completed", "success":
		return "✅"
	case "failed", "error":
		return "❌"
	case "running", "in_progress":
		return "⏳"
	case "pending", "waiting":
		return "🔲"
	case "skipped":
		return "⏭️"
	default:
		// Step-based icons (when status is unknown)
		stepIcons := map[string]string{
			"validation":              "🔍",
			"source-vm-power-off":     "🔌",
			"source-vm-power-on":      "⚡",
			"final-sync":              "🔄",
			"snapshot-creation":       "📸",
			"virtio-driver-injection": "💾",
			"vm-creation":             "🖥️",
			"volume-attachment":       "🔗",
			"network-configuration":   "🌐",
			"vm-startup":              "🚀",
			"cleanup":                 "🧹",
			"completion":              "✨",
		}
		
		// Check for partial matches
		for key, icon := range stepIcons {
			if strings.Contains(internalStep, key) {
				return icon
			}
		}
		
		return "📋" // Default icon
	}
}

// FormatStepForDisplay returns a complete formatted step for GUI display
func FormatStepForDisplay(internalStep string, status string, errorMsg string) map[string]interface{} {
	formatted := map[string]interface{}{
		"name":        internalStep,
		"display_name": GetUserFriendlyStepName(internalStep),
		"description":  GetStepDescription(internalStep),
		"category":     GetStepCategory(internalStep),
		"icon":         GetStepIcon(internalStep, status),
		"status":       status,
	}

	// Add sanitized error if present
	if errorMsg != "" && status == "failed" {
		sanitized := SanitizeFailoverError(internalStep, fmt.Errorf("%s", errorMsg))
		formatted["error_message"] = sanitized.UserMessage
		formatted["error_category"] = sanitized.Category
		formatted["actionable_steps"] = sanitized.ActionableSteps
		formatted["severity"] = sanitized.Severity
	}

	return formatted
}

