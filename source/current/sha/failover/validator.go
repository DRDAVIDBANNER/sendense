// Package failover provides VM failover validation functionality
package failover

import (
	"fmt"
	"time"

	"github.com/vexxhost/migratekit-sha/database"
	"github.com/vexxhost/migratekit-sha/services"

	log "github.com/sirupsen/logrus"
)

// ValidationResult represents the result of pre-failover validation
type ValidationResult struct {
	IsValid            bool                   `json:"is_valid"`
	ValidationErrors   []string               `json:"validation_errors"`
	ValidationWarnings []string               `json:"validation_warnings"`
	RequiredActions    []string               `json:"required_actions"`
	ValidationDetails  map[string]interface{} `json:"validation_details"`
	ValidatedAt        time.Time              `json:"validated_at"`
	ReadinessScore     float64                `json:"readiness_score"`
	EstimatedDuration  string                 `json:"estimated_duration"`
}

// FailoverReadinessCheck represents an individual validation check
type FailoverReadinessCheck struct {
	CheckName     string        `json:"check_name"`
	CheckType     string        `json:"check_type"` // critical, warning, info
	Status        string        `json:"status"`     // pass, fail, warning, skip
	Message       string        `json:"message"`
	Details       interface{}   `json:"details"`
	ExecutionTime time.Duration `json:"execution_time"`
}

// VMSyncStatus represents VM synchronization status
type VMSyncStatus struct {
	HasValidChangeID   bool      `json:"has_valid_change_id"`
	LastChangeID       string    `json:"last_change_id"`
	LastSyncTime       time.Time `json:"last_sync_time"`
	SyncJobsActive     int       `json:"sync_jobs_active"`
	SyncCompletionRate float64   `json:"sync_completion_rate"`
	IsSyncUpToDate     bool      `json:"is_sync_up_to_date"`
	TotalSyncedBytes   int64     `json:"total_synced_bytes"`
}

// NetworkMappingStatus represents network mapping validation status
type NetworkMappingStatus struct {
	TotalSourceNetworks   int      `json:"total_source_networks"`
	MappedNetworks        int      `json:"mapped_networks"`
	UnmappedNetworks      []string `json:"unmapped_networks"`
	InvalidMappings       []string `json:"invalid_mappings"`
	TestNetworkConfigured bool     `json:"test_network_configured"`
	AllNetworksMapped     bool     `json:"all_networks_mapped"`
}

// VolumeReadinessStatus represents volume state validation
type VolumeReadinessStatus struct {
	TotalVolumes           int      `json:"total_volumes"`
	ReadyVolumes           int      `json:"ready_volumes"`
	VolumeIssues           []string `json:"volume_issues"`
	HasOSSEAVolumes        bool     `json:"has_ossea_volumes"`
	VolumeIntegrityChecked bool     `json:"volume_integrity_checked"`
	AllVolumesReady        bool     `json:"all_volumes_ready"`
}

// PreFailoverValidator provides comprehensive validation before failover operations
type PreFailoverValidator struct {
	db                 database.Connection
	vmDiskRepo         *database.VMDiskRepository
	failoverJobRepo    *database.FailoverJobRepository
	networkMappingRepo *database.NetworkMappingRepository
	vmInfoService      services.VMInfoProvider
}

// NewPreFailoverValidator creates a new pre-failover validator
func NewPreFailoverValidator(
	db database.Connection,
	vmInfoService services.VMInfoProvider,
	networkMappingService *services.NetworkMappingService,
) *PreFailoverValidator {
	return &PreFailoverValidator{
		db:                 db,
		vmDiskRepo:         database.NewVMDiskRepository(db),
		failoverJobRepo:    database.NewFailoverJobRepository(db),
		networkMappingRepo: database.NewNetworkMappingRepository(db),
		vmInfoService:      vmInfoService,
	}
}

// ValidateFailoverReadiness performs comprehensive pre-failover validation
func (pfv *PreFailoverValidator) ValidateFailoverReadiness(vmID string, failoverType string) (*ValidationResult, error) {
	log.WithFields(log.Fields{
		"vm_id":         vmID,
		"failover_type": failoverType,
	}).Info("🔍 Starting comprehensive failover readiness validation")

	startTime := time.Now()
	result := &ValidationResult{
		IsValid:            true,
		ValidationErrors:   []string{},
		ValidationWarnings: []string{},
		RequiredActions:    []string{},
		ValidationDetails:  make(map[string]interface{}),
		ValidatedAt:        startTime,
	}

	var checks []FailoverReadinessCheck

	// 1. VM Existence and State Validation
	vmCheck := pfv.validateVMExistence(vmID)
	checks = append(checks, vmCheck)
	if vmCheck.Status == "fail" {
		result.IsValid = false
		result.ValidationErrors = append(result.ValidationErrors, vmCheck.Message)
	}

	// 2. Sync Status Validation (check for valid ChangeID)
	syncCheck := pfv.validateSyncStatus(vmID)
	checks = append(checks, syncCheck)
	if syncCheck.Status == "fail" {
		result.IsValid = false
		result.ValidationErrors = append(result.ValidationErrors, syncCheck.Message)
	} else if syncCheck.Status == "warning" {
		result.ValidationWarnings = append(result.ValidationWarnings, syncCheck.Message)
	}

	// 3. Network Mappings Validation
	networkCheck := pfv.validateNetworkMappings(vmID, failoverType)
	checks = append(checks, networkCheck)
	if networkCheck.Status == "fail" {
		result.IsValid = false
		result.ValidationErrors = append(result.ValidationErrors, networkCheck.Message)
	} else if networkCheck.Status == "warning" {
		result.ValidationWarnings = append(result.ValidationWarnings, networkCheck.Message)
	}

	result.ValidationDetails["checks"] = checks
	result.ValidationDetails["execution_time"] = time.Since(startTime)

	readinessScore := float64(0)
	passedChecks := 0
	for _, check := range checks {
		if check.Status == "pass" {
			passedChecks++
		}
	}
	if len(checks) > 0 {
		readinessScore = (float64(passedChecks) / float64(len(checks))) * 100
	}
	result.ValidationDetails["readiness_score"] = readinessScore
	result.ReadinessScore = readinessScore

	log.WithFields(log.Fields{
		"vm_id":           vmID,
		"failover_type":   failoverType,
		"is_valid":        result.IsValid,
		"errors":          len(result.ValidationErrors),
		"warnings":        len(result.ValidationWarnings),
		"readiness_score": readinessScore,
		"execution_time":  time.Since(startTime),
	}).Info("✅ Failover readiness validation completed")

	return result, nil
}

// validateVMExistence validates that the VM exists and is accessible
func (pfv *PreFailoverValidator) validateVMExistence(vmID string) FailoverReadinessCheck {
	startTime := time.Now()
	check := FailoverReadinessCheck{
		CheckName: "VM Existence",
		CheckType: "critical",
	}

	err := pfv.vmInfoService.ValidateVMExists(vmID)
	if err != nil {
		check.Status = "fail"
		check.Message = fmt.Sprintf("VM %s not found or inaccessible: %v", vmID, err)
	} else {
		check.Status = "pass"
		check.Message = "VM exists and is accessible"
	}

	check.ExecutionTime = time.Since(startTime)
	return check
}

// validateSyncStatus checks if VM has valid ChangeID from successful migration
func (pfv *PreFailoverValidator) validateSyncStatus(vmID string) FailoverReadinessCheck {
	startTime := time.Now()
	check := FailoverReadinessCheck{
		CheckName: "Sync Status",
		CheckType: "critical",
	}

	// Check cbt_history for any valid ChangeID for this VM (regardless of job status)
	var cbtRecord database.CBTHistory
	err := pfv.db.GetGormDB().
		Table("cbt_history").
		Select("cbt_history.*").
		Joins("JOIN replication_jobs ON cbt_history.job_id = replication_jobs.id").
		Where("replication_jobs.source_vm_id = ? AND cbt_history.sync_success = ?", vmID, true).
		Where("cbt_history.change_id IS NOT NULL AND cbt_history.change_id != ''").
		Order("cbt_history.created_at DESC").
		First(&cbtRecord).Error

	if err != nil {
		check.Status = "fail"
		check.Message = "No valid ChangeID found - VM must be synced at least once"
	} else {
		check.Status = "pass"
		check.Message = fmt.Sprintf("VM has valid ChangeID from successful sync: %s", cbtRecord.ChangeID[:20]+"...")
		check.Details = map[string]interface{}{
			"job_id":    cbtRecord.JobID,
			"change_id": cbtRecord.ChangeID,
			"sync_time": cbtRecord.CreatedAt,
		}
	}

	check.ExecutionTime = time.Since(startTime)
	return check
}

// validateNetworkMappings ensures all VM networks are properly mapped
func (pfv *PreFailoverValidator) validateNetworkMappings(vmID string, failoverType string) FailoverReadinessCheck {
	startTime := time.Now()
	check := FailoverReadinessCheck{
		CheckName: "Network Mappings",
		CheckType: "critical",
	}

	log.WithFields(log.Fields{
		"vm_id":         vmID,
		"failover_type": failoverType,
	}).Info("🔍 Validating network mappings")

	// Get VM network configuration
	sourceNetworks, err := pfv.vmInfoService.GetVMNetworkConfiguration(vmID)
	if err != nil {
		check.Status = "fail"
		check.Message = fmt.Sprintf("Failed to get VM network configuration: %v", err)
		check.ExecutionTime = time.Since(startTime)
		return check
	}

	// Get existing network mappings
	mappings, err := pfv.networkMappingRepo.GetByVMID(vmID)
	if err != nil {
		check.Status = "fail"
		check.Message = fmt.Sprintf("Failed to get network mappings: %v", err)
		check.ExecutionTime = time.Since(startTime)
		return check
	}

	// Validate mappings
	unmappedNetworks := []string{}
	invalidMappings := []string{}
	mappedNetworks := 0

	for _, sourceNet := range sourceNetworks {
		mapped := false
		for _, mapping := range mappings {
			if mapping.SourceNetworkName == sourceNet.NetworkName {
				mapped = true
				mappedNetworks++

				// For test failover, ensure test network is configured
				if failoverType == "test" && !mapping.IsTestNetwork {
					invalidMappings = append(invalidMappings,
						fmt.Sprintf("Network %s not configured for test failover", sourceNet.NetworkName))
				}
				break
			}
		}
		if !mapped {
			unmappedNetworks = append(unmappedNetworks, sourceNet.NetworkName)
		}
	}

	// Determine status
	allMapped := len(unmappedNetworks) == 0
	noInvalidMappings := len(invalidMappings) == 0

	if !allMapped || !noInvalidMappings {
		check.Status = "fail"
		errorParts := []string{}
		if !allMapped {
			errorParts = append(errorParts, fmt.Sprintf("Unmapped networks: %v", unmappedNetworks))
		}
		if !noInvalidMappings {
			errorParts = append(errorParts, fmt.Sprintf("Invalid mappings: %v", invalidMappings))
		}
		check.Message = fmt.Sprintf("Network mapping validation failed: %s", fmt.Sprintf("[%s]", fmt.Sprintf("%v", errorParts)))
	} else {
		check.Status = "pass"
		check.Message = "All networks properly mapped"
	}

	check.Details = map[string]interface{}{
		"source_networks":   len(sourceNetworks),
		"mapped_networks":   mappedNetworks,
		"unmapped_networks": len(unmappedNetworks),
		"all_mapped":        allMapped,
		"failover_type":     failoverType,
	}

	log.WithFields(log.Fields{
		"vm_id":             vmID,
		"failover_type":     failoverType,
		"source_networks":   len(sourceNetworks),
		"mapped_networks":   mappedNetworks,
		"unmapped_networks": len(unmappedNetworks),
		"invalid_mappings":  len(invalidMappings),
		"all_mapped":        allMapped,
	}).Info("✅ Network mappings validation completed")

	check.ExecutionTime = time.Since(startTime)
	return check
}
