// Package vmware provides VM specification comparison and change detection services
package vmware

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/vmware/govmomi"

	"github.com/vexxhost/migratekit/internal/sha/models"
)

// VMSpecificationService handles VM specification collection and comparison
type VMSpecificationService struct {
	discovery *Discovery
	client    *govmomi.Client
}

// NewVMSpecificationService creates a new VM specification service
func NewVMSpecificationService(discovery *Discovery) *VMSpecificationService {
	return &VMSpecificationService{
		discovery: discovery,
		client:    discovery.GetClient(),
	}
}

// VMSpecificationDiff represents changes detected in VM configuration
type VMSpecificationDiff struct {
	HasChanges         bool                   `json:"has_changes"`
	VMID               string                 `json:"vm_id"`
	VMName             string                 `json:"vm_name"`
	CPUChanges         *FieldChange           `json:"cpu_changes,omitempty"`
	MemoryChanges      *FieldChange           `json:"memory_changes,omitempty"`
	NetworkChanges     []NetworkAdapterChange `json:"network_changes,omitempty"`
	PowerStateChange   *FieldChange           `json:"power_state_change,omitempty"`
	VMwareToolsChanges *FieldChange           `json:"vmware_tools_changes,omitempty"`
	DisplayNameChange  *FieldChange           `json:"display_name_change,omitempty"`
	AnnotationChange   *FieldChange           `json:"annotation_change,omitempty"`
	FolderPathChange   *FieldChange           `json:"folder_path_change,omitempty"`
	LastChecked        time.Time              `json:"last_checked"`
}

// FieldChange represents a change in a specific field
type FieldChange struct {
	Field    string      `json:"field"`
	OldValue interface{} `json:"old_value"`
	NewValue interface{} `json:"new_value"`
}

// NetworkAdapterChange represents changes in network adapter configuration
type NetworkAdapterChange struct {
	AdapterIndex int         `json:"adapter_index"`
	ChangeType   string      `json:"change_type"` // "modified", "added", "removed"
	Field        string      `json:"field,omitempty"`
	OldValue     interface{} `json:"old_value,omitempty"`
	NewValue     interface{} `json:"new_value,omitempty"`
}

// DetectVMSpecificationChanges compares current vCenter specs with stored VM data
func (s *VMSpecificationService) DetectVMSpecificationChanges(ctx context.Context, vmPath string, storedVMInfo *models.VMInfo) (*VMSpecificationDiff, error) {
	log.WithFields(log.Fields{
		"vm_path":     vmPath,
		"stored_name": storedVMInfo.Name,
	}).Info("Detecting VM specification changes")

	// Get current VM specs from vCenter
	currentVMInfo, err := s.discovery.GetVMDetails(ctx, vmPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get current VM specification: %w", err)
	}

	// Compare specifications
	diff := s.compareVMSpecifications(storedVMInfo, currentVMInfo)
	diff.LastChecked = time.Now().UTC()

	if diff.HasChanges {
		log.WithFields(log.Fields{
			"vm_id":          diff.VMID,
			"vm_name":        diff.VMName,
			"changes_count":  s.countChanges(diff),
			"has_cpu_change": diff.CPUChanges != nil,
			"has_mem_change": diff.MemoryChanges != nil,
			"has_net_change": len(diff.NetworkChanges) > 0,
		}).Warn("VM specification changes detected")
	} else {
		log.WithFields(log.Fields{
			"vm_id":   diff.VMID,
			"vm_name": diff.VMName,
		}).Debug("No VM specification changes detected")
	}

	return diff, nil
}

// compareVMSpecifications performs detailed comparison between stored and current VM specs
func (s *VMSpecificationService) compareVMSpecifications(stored, current *models.VMInfo) *VMSpecificationDiff {
	diff := &VMSpecificationDiff{
		VMID:       current.ID,
		VMName:     current.Name,
		HasChanges: false,
	}

	// Compare CPU configuration
	if stored.CPUs != current.CPUs {
		diff.CPUChanges = &FieldChange{
			Field:    "cpus",
			OldValue: stored.CPUs,
			NewValue: current.CPUs,
		}
		diff.HasChanges = true
	}

	// Compare memory configuration
	if stored.MemoryMB != current.MemoryMB {
		diff.MemoryChanges = &FieldChange{
			Field:    "memory_mb",
			OldValue: stored.MemoryMB,
			NewValue: current.MemoryMB,
		}
		diff.HasChanges = true
	}

	// Compare power state
	if stored.PowerState != current.PowerState {
		diff.PowerStateChange = &FieldChange{
			Field:    "power_state",
			OldValue: stored.PowerState,
			NewValue: current.PowerState,
		}
		diff.HasChanges = true
	}

	// Compare VMware Tools status
	if stored.VMwareToolsStatus != current.VMwareToolsStatus {
		diff.VMwareToolsChanges = &FieldChange{
			Field:    "vmware_tools_status",
			OldValue: stored.VMwareToolsStatus,
			NewValue: current.VMwareToolsStatus,
		}
		diff.HasChanges = true
	}

	// Compare display name
	if stored.DisplayName != current.DisplayName {
		diff.DisplayNameChange = &FieldChange{
			Field:    "display_name",
			OldValue: stored.DisplayName,
			NewValue: current.DisplayName,
		}
		diff.HasChanges = true
	}

	// Compare annotation
	if stored.Annotation != current.Annotation {
		diff.AnnotationChange = &FieldChange{
			Field:    "annotation",
			OldValue: stored.Annotation,
			NewValue: current.Annotation,
		}
		diff.HasChanges = true
	}

	// Compare folder path
	if stored.FolderPath != current.FolderPath {
		diff.FolderPathChange = &FieldChange{
			Field:    "folder_path",
			OldValue: stored.FolderPath,
			NewValue: current.FolderPath,
		}
		diff.HasChanges = true
	}

	// Compare network configurations
	networkChanges := s.compareNetworkConfigurations(stored.Networks, current.Networks)
	if len(networkChanges) > 0 {
		diff.NetworkChanges = networkChanges
		diff.HasChanges = true
	}

	return diff
}

// compareNetworkConfigurations compares network adapter configurations
func (s *VMSpecificationService) compareNetworkConfigurations(stored, current []models.NetworkInfo) []NetworkAdapterChange {
	var changes []NetworkAdapterChange

	// Create maps for easier comparison by MAC address
	storedNetMap := make(map[string]models.NetworkInfo)
	currentNetMap := make(map[string]models.NetworkInfo)

	for _, net := range stored {
		storedNetMap[net.MACAddress] = net
	}
	for _, net := range current {
		currentNetMap[net.MACAddress] = net
	}

	// Check for modified or removed adapters
	for macAddr, storedNet := range storedNetMap {
		if currentNet, exists := currentNetMap[macAddr]; exists {
			// Check for modifications
			adapterChanges := s.compareNetworkAdapter(storedNet, currentNet)
			changes = append(changes, adapterChanges...)
		} else {
			// Adapter removed
			changes = append(changes, NetworkAdapterChange{
				ChangeType: "removed",
				OldValue:   storedNet,
			})
		}
	}

	// Check for added adapters
	for macAddr, currentNet := range currentNetMap {
		if _, exists := storedNetMap[macAddr]; !exists {
			changes = append(changes, NetworkAdapterChange{
				ChangeType: "added",
				NewValue:   currentNet,
			})
		}
	}

	return changes
}

// compareNetworkAdapter compares individual network adapter properties
func (s *VMSpecificationService) compareNetworkAdapter(stored, current models.NetworkInfo) []NetworkAdapterChange {
	var changes []NetworkAdapterChange

	if stored.NetworkName != current.NetworkName {
		changes = append(changes, NetworkAdapterChange{
			ChangeType: "modified",
			Field:      "network_name",
			OldValue:   stored.NetworkName,
			NewValue:   current.NetworkName,
		})
	}

	if stored.AdapterType != current.AdapterType {
		changes = append(changes, NetworkAdapterChange{
			ChangeType: "modified",
			Field:      "adapter_type",
			OldValue:   stored.AdapterType,
			NewValue:   current.AdapterType,
		})
	}

	if stored.Connected != current.Connected {
		changes = append(changes, NetworkAdapterChange{
			ChangeType: "modified",
			Field:      "connected",
			OldValue:   stored.Connected,
			NewValue:   current.Connected,
		})
	}

	return changes
}

// countChanges counts the total number of changes in a diff
func (s *VMSpecificationService) countChanges(diff *VMSpecificationDiff) int {
	count := 0

	if diff.CPUChanges != nil {
		count++
	}
	if diff.MemoryChanges != nil {
		count++
	}
	if diff.PowerStateChange != nil {
		count++
	}
	if diff.VMwareToolsChanges != nil {
		count++
	}
	if diff.DisplayNameChange != nil {
		count++
	}
	if diff.AnnotationChange != nil {
		count++
	}
	if diff.FolderPathChange != nil {
		count++
	}

	count += len(diff.NetworkChanges)

	return count
}

// GetChangesSummary returns a human-readable summary of changes
func (s *VMSpecificationService) GetChangesSummary(diff *VMSpecificationDiff) string {
	if !diff.HasChanges {
		return "No changes detected"
	}

	summary := fmt.Sprintf("VM '%s' has %d specification change(s):\n", diff.VMName, s.countChanges(diff))

	if diff.CPUChanges != nil {
		summary += fmt.Sprintf("- CPU: %v → %v\n", diff.CPUChanges.OldValue, diff.CPUChanges.NewValue)
	}
	if diff.MemoryChanges != nil {
		summary += fmt.Sprintf("- Memory: %v MB → %v MB\n", diff.MemoryChanges.OldValue, diff.MemoryChanges.NewValue)
	}
	if diff.PowerStateChange != nil {
		summary += fmt.Sprintf("- Power State: %v → %v\n", diff.PowerStateChange.OldValue, diff.PowerStateChange.NewValue)
	}
	if diff.VMwareToolsChanges != nil {
		summary += fmt.Sprintf("- VMware Tools: %v → %v\n", diff.VMwareToolsChanges.OldValue, diff.VMwareToolsChanges.NewValue)
	}
	if diff.DisplayNameChange != nil {
		summary += fmt.Sprintf("- Display Name: %v → %v\n", diff.DisplayNameChange.OldValue, diff.DisplayNameChange.NewValue)
	}
	if diff.AnnotationChange != nil {
		summary += fmt.Sprintf("- Annotation: changed\n")
	}
	if diff.FolderPathChange != nil {
		summary += fmt.Sprintf("- Folder Path: %v → %v\n", diff.FolderPathChange.OldValue, diff.FolderPathChange.NewValue)
	}

	if len(diff.NetworkChanges) > 0 {
		summary += fmt.Sprintf("- Network Changes: %d adapter(s) modified\n", len(diff.NetworkChanges))
	}

	return summary
}

// SerializeChanges converts the diff to JSON for storage/logging
func (s *VMSpecificationService) SerializeChanges(diff *VMSpecificationDiff) (string, error) {
	jsonBytes, err := json.MarshalIndent(diff, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to serialize changes: %w", err)
	}
	return string(jsonBytes), nil
}




