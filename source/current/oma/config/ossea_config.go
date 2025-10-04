package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/vexxhost/migratekit-oma/database"
	"github.com/vexxhost/migratekit-oma/ossea"
	"gopkg.in/yaml.v2"
)

// OSSEAConfigManager manages OSSEA connection configurations
type OSSEAConfigManager struct {
	db database.Connection
}

// NewOSSEAConfigManager creates a new configuration manager
func NewOSSEAConfigManager(db database.Connection) *OSSEAConfigManager {
	return &OSSEAConfigManager{
		db: db,
	}
}

// OSSEAConfigInput represents input for creating/updating OSSEA config
type OSSEAConfigInput struct {
	Name              string `json:"name" yaml:"name" binding:"required"`
	APIURL            string `json:"api_url" yaml:"api_url" binding:"required"`
	APIKey            string `json:"api_key" yaml:"api_key" binding:"required"`
	SecretKey         string `json:"secret_key" yaml:"secret_key" binding:"required"`
	Domain            string `json:"domain" yaml:"domain"`
	Zone              string `json:"zone" yaml:"zone" binding:"required"`
	TemplateID        string `json:"template_id" yaml:"template_id"`
	NetworkID         string `json:"network_id" yaml:"network_id"`
	ServiceOfferingID string `json:"service_offering_id" yaml:"service_offering_id"`
	DiskOfferingID    string `json:"disk_offering_id" yaml:"disk_offering_id"`
	OMAVMID           string `json:"oma_vm_id" yaml:"oma_vm_id"` // OMA VM ID in OSSEA
}

// CreateOSSEAConfig creates a new OSSEA configuration
func (cm *OSSEAConfigManager) CreateOSSEAConfig(input *OSSEAConfigInput) (*database.OSSEAConfig, error) {
	log.WithFields(log.Fields{
		"name":    input.Name,
		"api_url": input.APIURL,
		"zone":    input.Zone,
	}).Info("🔧 Creating OSSEA configuration")

	// Validate configuration by testing connection
	if err := cm.validateOSSEAConnection(input); err != nil {
		return nil, fmt.Errorf("OSSEA connection validation failed: %w", err)
	}

	// Create database record
	config := &database.OSSEAConfig{
		Name:              input.Name,
		APIURL:            input.APIURL,
		APIKey:            input.APIKey,
		SecretKey:         input.SecretKey,
		Domain:            input.Domain,
		Zone:              input.Zone,
		TemplateID:        input.TemplateID,
		NetworkID:         input.NetworkID,
		ServiceOfferingID: input.ServiceOfferingID,
		DiskOfferingID:    input.DiskOfferingID,
		OMAVMID:           input.OMAVMID,
		IsActive:          true,
	}

	if err := cm.db.GetGormDB().Create(config).Error; err != nil {
		return nil, fmt.Errorf("failed to save OSSEA config: %w", err)
	}

	log.WithFields(log.Fields{
		"config_id": config.ID,
		"name":      config.Name,
	}).Info("✅ OSSEA configuration created")

	return config, nil
}

// GetOSSEAConfig retrieves an OSSEA configuration by ID
func (cm *OSSEAConfigManager) GetOSSEAConfig(configID int) (*database.OSSEAConfig, error) {
	var config database.OSSEAConfig
	err := cm.db.GetGormDB().Where("id = ? AND is_active = true", configID).First(&config).Error
	if err != nil {
		return nil, fmt.Errorf("OSSEA config not found: %w", err)
	}
	return &config, nil
}

// GetOSSEAConfigByName retrieves an OSSEA configuration by name
func (cm *OSSEAConfigManager) GetOSSEAConfigByName(name string) (*database.OSSEAConfig, error) {
	var config database.OSSEAConfig
	err := cm.db.GetGormDB().Where("name = ? AND is_active = true", name).First(&config).Error
	if err != nil {
		return nil, fmt.Errorf("OSSEA config '%s' not found: %w", name, err)
	}
	return &config, nil
}

// ListOSSEAConfigs lists all active OSSEA configurations
func (cm *OSSEAConfigManager) ListOSSEAConfigs() ([]database.OSSEAConfig, error) {
	var configs []database.OSSEAConfig
	err := cm.db.GetGormDB().Where("is_active = true").Find(&configs).Error
	if err != nil {
		return nil, fmt.Errorf("failed to list OSSEA configs: %w", err)
	}
	return configs, nil
}

// UpdateOSSEAConfig updates an existing OSSEA configuration
func (cm *OSSEAConfigManager) UpdateOSSEAConfig(configID int, input *OSSEAConfigInput) (*database.OSSEAConfig, error) {
	log.WithFields(log.Fields{
		"config_id": configID,
		"name":      input.Name,
	}).Info("🔧 Updating OSSEA configuration")

	// Get existing config
	config, err := cm.GetOSSEAConfig(configID)
	if err != nil {
		return nil, err
	}

	// Validate new configuration
	if err := cm.validateOSSEAConnection(input); err != nil {
		return nil, fmt.Errorf("OSSEA connection validation failed: %w", err)
	}

	// Update fields
	config.Name = input.Name
	config.APIURL = input.APIURL
	config.APIKey = input.APIKey
	config.SecretKey = input.SecretKey
	config.Domain = input.Domain
	config.Zone = input.Zone
	config.TemplateID = input.TemplateID
	config.NetworkID = input.NetworkID
	config.ServiceOfferingID = input.ServiceOfferingID
	config.DiskOfferingID = input.DiskOfferingID
	config.OMAVMID = input.OMAVMID

	if err := cm.db.GetGormDB().Save(config).Error; err != nil {
		return nil, fmt.Errorf("failed to update OSSEA config: %w", err)
	}

	log.WithField("config_id", configID).Info("✅ OSSEA configuration updated")
	return config, nil
}

// DeleteOSSEAConfig soft deletes an OSSEA configuration
func (cm *OSSEAConfigManager) DeleteOSSEAConfig(configID int) error {
	log.WithField("config_id", configID).Info("🗑️ Deleting OSSEA configuration")

	err := cm.db.GetGormDB().Model(&database.OSSEAConfig{}).
		Where("id = ?", configID).
		Update("is_active", false).Error

	if err != nil {
		return fmt.Errorf("failed to delete OSSEA config: %w", err)
	}

	log.WithField("config_id", configID).Info("✅ OSSEA configuration deleted")
	return nil
}

// CreateOSSEAClient creates an OSSEA client from configuration
func (cm *OSSEAConfigManager) CreateOSSEAClient(configID int) (*ossea.Client, error) {
	config, err := cm.GetOSSEAConfig(configID)
	if err != nil {
		return nil, err
	}

	client := ossea.NewClient(
		config.APIURL,
		config.APIKey,
		config.SecretKey,
		config.Domain,
		config.Zone,
	)

	return client, nil
}

// CreateOSSEAClientByName creates an OSSEA client from configuration by name
func (cm *OSSEAConfigManager) CreateOSSEAClientByName(name string) (*ossea.Client, error) {
	config, err := cm.GetOSSEAConfigByName(name)
	if err != nil {
		return nil, err
	}

	client := ossea.NewClient(
		config.APIURL,
		config.APIKey,
		config.SecretKey,
		config.Domain,
		config.Zone,
	)

	return client, nil
}

// TestOSSEAConnection tests the connection to OSSEA using a configuration
func (cm *OSSEAConfigManager) TestOSSEAConnection(configID int) error {
	config, err := cm.GetOSSEAConfig(configID)
	if err != nil {
		return err
	}

	input := &OSSEAConfigInput{
		APIURL:    config.APIURL,
		APIKey:    config.APIKey,
		SecretKey: config.SecretKey,
		Domain:    config.Domain,
		Zone:      config.Zone,
	}

	return cm.validateOSSEAConnection(input)
}

// LoadOSSEAConfigFromFile loads OSSEA configuration from a YAML file
func (cm *OSSEAConfigManager) LoadOSSEAConfigFromFile(filename string) (*database.OSSEAConfig, error) {
	log.WithField("filename", filename).Info("📁 Loading OSSEA config from file")

	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var input OSSEAConfigInput
	if err := yaml.Unmarshal(data, &input); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return cm.CreateOSSEAConfig(&input)
}

// SaveOSSEAConfigToFile saves an OSSEA configuration to a YAML file
func (cm *OSSEAConfigManager) SaveOSSEAConfigToFile(configID int, filename string) error {
	config, err := cm.GetOSSEAConfig(configID)
	if err != nil {
		return err
	}

	input := &OSSEAConfigInput{
		Name:              config.Name,
		APIURL:            config.APIURL,
		APIKey:            config.APIKey,
		SecretKey:         config.SecretKey,
		Domain:            config.Domain,
		Zone:              config.Zone,
		TemplateID:        config.TemplateID,
		NetworkID:         config.NetworkID,
		ServiceOfferingID: config.ServiceOfferingID,
		DiskOfferingID:    config.DiskOfferingID,
		OMAVMID:           config.OMAVMID,
	}

	data, err := yaml.Marshal(input)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(filename, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	log.WithFields(log.Fields{
		"config_id": configID,
		"filename":  filename,
	}).Info("✅ OSSEA config saved to file")

	return nil
}

// LoadFromEnvironment loads OSSEA configuration from environment variables
func (cm *OSSEAConfigManager) LoadFromEnvironment(name string) (*database.OSSEAConfig, error) {
	log.WithField("name", name).Info("🌍 Loading OSSEA config from environment")

	input := &OSSEAConfigInput{
		Name:              name,
		APIURL:            os.Getenv("OSSEA_API_URL"),
		APIKey:            os.Getenv("OSSEA_API_KEY"),
		SecretKey:         os.Getenv("OSSEA_SECRET_KEY"),
		Domain:            os.Getenv("OSSEA_DOMAIN"),
		Zone:              os.Getenv("OSSEA_ZONE"),
		TemplateID:        os.Getenv("OSSEA_TEMPLATE_ID"),
		NetworkID:         os.Getenv("OSSEA_NETWORK_ID"),
		ServiceOfferingID: os.Getenv("OSSEA_SERVICE_OFFERING_ID"),
		DiskOfferingID:    os.Getenv("OSSEA_DISK_OFFERING_ID"),
		OMAVMID:           os.Getenv("OSSEA_OMA_VM_ID"),
	}

	// Validate required environment variables
	if input.APIURL == "" {
		return nil, fmt.Errorf("OSSEA_API_URL environment variable is required")
	}
	if input.APIKey == "" {
		return nil, fmt.Errorf("OSSEA_API_KEY environment variable is required")
	}
	if input.SecretKey == "" {
		return nil, fmt.Errorf("OSSEA_SECRET_KEY environment variable is required")
	}
	if input.Zone == "" {
		return nil, fmt.Errorf("OSSEA_ZONE environment variable is required")
	}

	return cm.CreateOSSEAConfig(input)
}

// GetDefaultOSSEAConfig returns the first active OSSEA configuration
func (cm *OSSEAConfigManager) GetDefaultOSSEAConfig() (*database.OSSEAConfig, error) {
	var config database.OSSEAConfig
	err := cm.db.GetGormDB().Where("is_active = true").First(&config).Error
	if err != nil {
		return nil, fmt.Errorf("no default OSSEA config found: %w", err)
	}
	return &config, nil
}

// Helper methods

// validateOSSEAConnection validates connection to OSSEA
func (cm *OSSEAConfigManager) validateOSSEAConnection(input *OSSEAConfigInput) error {
	log.WithField("api_url", input.APIURL).Debug("Validating OSSEA connection")

	// Create client
	client := ossea.NewClient(
		input.APIURL,
		input.APIKey,
		input.SecretKey,
		input.Domain,
		input.Zone,
	)

	// Test connection by listing zones
	_, err := client.ListVolumes(nil)
	if err != nil {
		return fmt.Errorf("failed to connect to OSSEA: %w", err)
	}

	log.WithField("api_url", input.APIURL).Debug("✅ OSSEA connection validated")
	return nil
}

// Environment variable helpers

// GetEnvOrDefault gets environment variable with default value
func GetEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// GetEnvAsInt gets environment variable as integer
func GetEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// GetEnvAsBool gets environment variable as boolean
func GetEnvAsBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		return strings.ToLower(value) == "true"
	}
	return defaultValue
}
