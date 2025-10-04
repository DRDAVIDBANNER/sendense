package validation

import (
	"context"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/vexxhost/migratekit-oma/database"
	"github.com/vexxhost/migratekit-oma/ossea"
)

// CloudStackAutoFixer automatically fixes or provisions missing CloudStack prerequisites
type CloudStackAutoFixer struct {
	client *ossea.Client
	db     database.Connection
}

// NewCloudStackAutoFixer creates a new auto-fixer
func NewCloudStackAutoFixer(client *ossea.Client, db database.Connection) *CloudStackAutoFixer {
	return &CloudStackAutoFixer{
		client: client,
		db:     db,
	}
}

// AutoFixResult represents the result of an auto-fix operation
type AutoFixResult struct {
	FixName     string `json:"fix_name"`
	Attempted   bool   `json:"attempted"`
	Successful  bool   `json:"successful"`
	Message     string `json:"message"`
	Details     string `json:"details,omitempty"`
	ResourceID  string `json:"resource_id,omitempty"`  // ID of created/fixed resource
}

// AutoFixReport contains all auto-fix results
type AutoFixReport struct {
	Timestamp       time.Time       `json:"timestamp"`
	FixesAttempted  int            `json:"fixes_attempted"`
	FixesSuccessful int            `json:"fixes_successful"`
	FixesFailed     int            `json:"fixes_failed"`
	Results         []AutoFixResult `json:"results"`
}

// AutoFixConfiguration attempts to automatically fix CloudStack configuration issues
func (f *CloudStackAutoFixer) AutoFixConfiguration(ctx context.Context, validationReport *ValidationReport, config *CloudStackConfig) (*AutoFixReport, error) {
	log.Info("🔧 Starting CloudStack configuration auto-fix")
	
	report := &AutoFixReport{
		Timestamp: time.Now(),
		Results:   make([]AutoFixResult, 0),
	}
	
	// Only attempt auto-fixes for issues marked as auto-fixable
	for _, result := range validationReport.Results {
		if !result.Passed && result.AutoFixable {
			report.FixesAttempted++
			
			switch result.CheckName {
			case "Zone Specified":
				fixResult := f.autoSelectZone(ctx, config)
				report.Results = append(report.Results, fixResult)
				if fixResult.Successful {
					report.FixesSuccessful++
				} else {
					report.FixesFailed++
				}
				
			case "Template Specified":
				fixResult := f.autoSelectTemplate(ctx, config)
				report.Results = append(report.Results, fixResult)
				if fixResult.Successful {
					report.FixesSuccessful++
				} else {
					report.FixesFailed++
				}
				
			case "Network Specified":
				fixResult := f.autoSelectNetwork(ctx, config)
				report.Results = append(report.Results, fixResult)
				if fixResult.Successful {
					report.FixesSuccessful++
				} else {
					report.FixesFailed++
				}
				
			case "Service Offering Specified":
				fixResult := f.autoSelectServiceOffering(ctx, config)
				report.Results = append(report.Results, fixResult)
				if fixResult.Successful {
					report.FixesSuccessful++
				} else {
					report.FixesFailed++
				}
				
			case "Disk Offering Specified":
				fixResult := f.autoSelectDiskOffering(ctx, config)
				report.Results = append(report.Results, fixResult)
				if fixResult.Successful {
					report.FixesSuccessful++
				} else {
					report.FixesFailed++
				}
				
			case "OMA VM ID Specified":
				fixResult := f.autoDetectOMAVM(ctx, config)
				report.Results = append(report.Results, fixResult)
				if fixResult.Successful {
					report.FixesSuccessful++
				} else {
					report.FixesFailed++
				}
			
			default:
				// Skip non-auto-fixable issues
				log.WithField("check", result.CheckName).Debug("Skipping non-auto-fixable check")
			}
		}
	}
	
	log.WithFields(log.Fields{
		"attempted":  report.FixesAttempted,
		"successful": report.FixesSuccessful,
		"failed":     report.FixesFailed,
	}).Info("✅ CloudStack configuration auto-fix complete")
	
	return report, nil
}

// Auto-select first available zone
func (f *CloudStackAutoFixer) autoSelectZone(ctx context.Context, config *CloudStackConfig) AutoFixResult {
	zones, err := f.client.ListZones()
	if err != nil {
		return AutoFixResult{
			FixName:    "Auto-select Zone",
			Attempted:  true,
			Successful: false,
			Message:    "Failed to list zones",
			Details:    fmt.Sprintf("Error: %v", err),
		}
	}
	
	if len(zones) == 0 {
		return AutoFixResult{
			FixName:    "Auto-select Zone",
			Attempted:  true,
			Successful: false,
			Message:    "No zones available",
			Details:    "CloudStack deployment has no zones configured",
		}
	}
	
	// Select first zone
	selectedZone := zones[0]
	config.Zone = selectedZone.ID
	
	return AutoFixResult{
		FixName:    "Auto-select Zone",
		Attempted:  true,
		Successful: true,
		Message:    fmt.Sprintf("Auto-selected zone: %s", selectedZone.Name),
		ResourceID: selectedZone.ID,
	}
}

// Auto-select first ready template
func (f *CloudStackAutoFixer) autoSelectTemplate(ctx context.Context, config *CloudStackConfig) AutoFixResult {
	templates, err := f.client.ListTemplates("featured")
	if err != nil {
		return AutoFixResult{
			FixName:    "Auto-select Template",
			Attempted:  true,
			Successful: false,
			Message:    "Failed to list templates",
			Details:    fmt.Sprintf("Error: %v", err),
		}
	}
	
	// Find first ready template
	for _, template := range templates {
		if template.IsReady {
			config.TemplateID = template.ID
			return AutoFixResult{
				FixName:    "Auto-select Template",
				Attempted:  true,
				Successful: true,
				Message:    fmt.Sprintf("Auto-selected template: %s (%s)", template.DisplayText, template.OSTypeName),
				ResourceID: template.ID,
			}
		}
	}
	
	return AutoFixResult{
		FixName:    "Auto-select Template",
		Attempted:  true,
		Successful: false,
		Message:    "No ready templates available",
		Details:    fmt.Sprintf("Found %d templates, but none are ready", len(templates)),
	}
}

// Auto-select first available network in zone
func (f *CloudStackAutoFixer) autoSelectNetwork(ctx context.Context, config *CloudStackConfig) AutoFixResult {
	networks, err := f.client.ListNetworks()
	if err != nil {
		return AutoFixResult{
			FixName:    "Auto-select Network",
			Attempted:  true,
			Successful: false,
			Message:    "Failed to list networks",
			Details:    fmt.Sprintf("Error: %v", err),
		}
	}
	
	// Find first network in selected zone that's ready
	for _, network := range networks {
		if (network.State == "Implemented" || network.State == "Allocated") {
			// If zone is specified, match zone
			if config.Zone == "" || network.ZoneID == config.Zone {
				config.NetworkID = network.ID
				return AutoFixResult{
					FixName:    "Auto-select Network",
					Attempted:  true,
					Successful: true,
					Message:    fmt.Sprintf("Auto-selected network: %s in zone %s", network.Name, network.ZoneName),
					ResourceID: network.ID,
				}
			}
		}
	}
	
	return AutoFixResult{
		FixName:    "Auto-select Network",
		Attempted:  true,
		Successful: false,
		Message:    "No suitable network found",
		Details:    fmt.Sprintf("Found %d networks, but none are in ready state in selected zone", len(networks)),
	}
}

// Auto-select service offering with reasonable defaults (2+ CPU, 4GB+ RAM)
func (f *CloudStackAutoFixer) autoSelectServiceOffering(ctx context.Context, config *CloudStackConfig) AutoFixResult {
	offerings, err := f.client.ListServiceOfferings()
	if err != nil {
		return AutoFixResult{
			FixName:    "Auto-select Service Offering",
			Attempted:  true,
			Successful: false,
			Message:    "Failed to list service offerings",
			Details:    fmt.Sprintf("Error: %v", err),
		}
	}
	
	// Find first offering with reasonable specs
	for _, offering := range offerings {
		if offering.CPUNumber >= 2 && offering.Memory >= 4096 {
			config.ServiceOfferingID = offering.ID
			return AutoFixResult{
				FixName:    "Auto-select Service Offering",
				Attempted:  true,
				Successful: true,
				Message:    fmt.Sprintf("Auto-selected service offering: %s (%d CPU, %d MB RAM)", offering.DisplayText, offering.CPUNumber, offering.Memory),
				ResourceID: offering.ID,
			}
		}
	}
	
	// If no offering meets minimum specs, select first available
	if len(offerings) > 0 {
		offering := offerings[0]
		config.ServiceOfferingID = offering.ID
		return AutoFixResult{
			FixName:    "Auto-select Service Offering",
			Attempted:  true,
			Successful: true,
			Message:    fmt.Sprintf("Auto-selected service offering (minimal): %s (%d CPU, %d MB RAM)", offering.DisplayText, offering.CPUNumber, offering.Memory),
			Details:    "Warning: Selected offering is below recommended minimum (2 CPU, 4096 MB RAM)",
			ResourceID: offering.ID,
		}
	}
	
	return AutoFixResult{
		FixName:    "Auto-select Service Offering",
		Attempted:  true,
		Successful: false,
		Message:    "No service offerings available",
	}
}

// Auto-select disk offering (prefer custom size offerings)
func (f *CloudStackAutoFixer) autoSelectDiskOffering(ctx context.Context, config *CloudStackConfig) AutoFixResult {
	offerings, err := f.client.ListDiskOfferings()
	if err != nil {
		return AutoFixResult{
			FixName:    "Auto-select Disk Offering",
			Attempted:  true,
			Successful: false,
			Message:    "Failed to list disk offerings",
			Details:    fmt.Sprintf("Error: %v", err),
		}
	}
	
	// Prefer custom size disk offerings (DiskSize == 0 means custom)
	for _, offering := range offerings {
		if offering.DiskSize == 0 {
			config.DiskOfferingID = offering.ID
			return AutoFixResult{
				FixName:    "Auto-select Disk Offering",
				Attempted:  true,
				Successful: true,
				Message:    fmt.Sprintf("Auto-selected disk offering: %s (custom size)", offering.DisplayText),
				ResourceID: offering.ID,
			}
		}
	}
	
	// If no custom offering, select first available
	if len(offerings) > 0 {
		offering := offerings[0]
		config.DiskOfferingID = offering.ID
		return AutoFixResult{
			FixName:    "Auto-select Disk Offering",
			Attempted:  true,
			Successful: true,
			Message:    fmt.Sprintf("Auto-selected disk offering: %s (%d GB)", offering.DisplayText, offering.DiskSize),
			ResourceID: offering.ID,
		}
	}
	
	return AutoFixResult{
		FixName:    "Auto-select Disk Offering",
		Attempted:  true,
		Successful: false,
		Message:    "No disk offerings available",
	}
}

// Auto-detect OMA VM ID by searching for current VM
func (f *CloudStackAutoFixer) autoDetectOMAVM(ctx context.Context, config *CloudStackConfig) AutoFixResult {
	// This would require additional logic to detect the current VM
	// For now, return a helpful message
	return AutoFixResult{
		FixName:    "Auto-detect OMA VM",
		Attempted:  true,
		Successful: false,
		Message:    "Cannot auto-detect OMA VM ID",
		Details:    "OMA VM ID must be manually specified - check CloudStack for this VM's ID",
	}
}

// SaveFixedConfiguration saves the auto-fixed configuration to database
func (f *CloudStackAutoFixer) SaveFixedConfiguration(ctx context.Context, config *CloudStackConfig) error {
	log.Info("💾 Saving auto-fixed CloudStack configuration to database")
	
	// Create or update OSSEA config in database
	osseaConfig := &database.OSSEAConfig{
		Name:              "production-ossea",
		APIURL:            config.APIURL,
		// Note: API keys not in CloudStackConfig struct, must be passed separately
		Zone:              config.Zone,
		TemplateID:        config.TemplateID,
		NetworkID:         config.NetworkID,
		ServiceOfferingID: config.ServiceOfferingID,
		DiskOfferingID:    config.DiskOfferingID,
		OMAVMID:           config.OMAVMID,
		IsActive:          true,
	}
	
	repo := database.NewOSSEAConfigRepository(f.db)
	
	// Try to get existing config
	existingConfig, err := repo.GetByName("production-ossea")
	if err == nil {
		// Update existing
		return repo.Update(existingConfig.ID, osseaConfig)
	}
	
	// Create new
	return repo.Create(osseaConfig)
}


