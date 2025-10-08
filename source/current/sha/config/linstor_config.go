package config

import (
	"fmt"
	"os"
	"strconv"

	log "github.com/sirupsen/logrus"
	"github.com/vexxhost/migratekit-sha/database"
)

// LinstorConfigManager manages Linstor API configurations
type LinstorConfigManager struct {
	db database.Connection
}

// NewLinstorConfigManager creates a new Linstor configuration manager
func NewLinstorConfigManager(db database.Connection) *LinstorConfigManager {
	return &LinstorConfigManager{
		db: db,
	}
}

// LinstorConfigInput represents input for creating/updating Linstor config
type LinstorConfigInput struct {
	Name                     string `json:"name" yaml:"name" binding:"required"`
	APIURL                   string `json:"api_url" yaml:"api_url" binding:"required"`
	APIPort                  int    `json:"api_port" yaml:"api_port"`
	APIProtocol              string `json:"api_protocol" yaml:"api_protocol"`
	APIKey                   string `json:"api_key" yaml:"api_key"`
	APISecret                string `json:"api_secret" yaml:"api_secret"`
	ConnectionTimeoutSeconds int    `json:"connection_timeout_seconds" yaml:"connection_timeout_seconds"`
	RetryAttempts            int    `json:"retry_attempts" yaml:"retry_attempts"`
	Description              string `json:"description" yaml:"description"`
}

// CreateLinstorConfig creates a new Linstor configuration
func (cm *LinstorConfigManager) CreateLinstorConfig(input *LinstorConfigInput) (*database.LinstorConfig, error) {
	log.WithFields(log.Fields{
		"name":    input.Name,
		"api_url": input.APIURL,
		"api_port": input.APIPort,
	}).Info("🔧 Creating Linstor configuration")

	// Validate configuration by testing connection
	if err := cm.validateLinstorConnection(input); err != nil {
		return nil, fmt.Errorf("Linstor connection validation failed: %w", err)
	}

	// Set defaults
	if input.APIPort == 0 {
		input.APIPort = 3370
	}
	if input.APIProtocol == "" {
		input.APIProtocol = "http"
	}
	if input.ConnectionTimeoutSeconds == 0 {
		input.ConnectionTimeoutSeconds = 30
	}
	if input.RetryAttempts == 0 {
		input.RetryAttempts = 3
	}

	// Create database record
	config := &database.LinstorConfig{
		Name:                     input.Name,
		APIURL:                   input.APIURL,
		APIPort:                  input.APIPort,
		APIProtocol:              input.APIProtocol,
		APIKey:                   input.APIKey,
		APISecret:                input.APISecret,
		ConnectionTimeoutSeconds: input.ConnectionTimeoutSeconds,
		RetryAttempts:            input.RetryAttempts,
		Description:              input.Description,
		IsActive:                 true,
	}

	if err := cm.db.GetGormDB().Create(config).Error; err != nil {
		return nil, fmt.Errorf("failed to save Linstor config: %w", err)
	}

	log.WithFields(log.Fields{
		"config_id": config.ID,
		"name":      config.Name,
	}).Info("✅ Linstor configuration created")

	return config, nil
}

// GetLinstorConfig retrieves a Linstor configuration by ID
func (cm *LinstorConfigManager) GetLinstorConfig(configID int) (*database.LinstorConfig, error) {
	var config database.LinstorConfig
	err := cm.db.GetGormDB().Where("id = ? AND is_active = true", configID).First(&config).Error
	if err != nil {
		return nil, fmt.Errorf("Linstor config not found: %w", err)
	}
	return &config, nil
}

// GetLinstorConfigByName retrieves a Linstor configuration by name
func (cm *LinstorConfigManager) GetLinstorConfigByName(name string) (*database.LinstorConfig, error) {
	var config database.LinstorConfig
	err := cm.db.GetGormDB().Where("name = ? AND is_active = true", name).First(&config).Error
	if err != nil {
		return nil, fmt.Errorf("Linstor config '%s' not found: %w", name, err)
	}
	return &config, nil
}

// ListLinstorConfigs lists all active Linstor configurations
func (cm *LinstorConfigManager) ListLinstorConfigs() ([]database.LinstorConfig, error) {
	var configs []database.LinstorConfig
	err := cm.db.GetGormDB().Where("is_active = true").Order("created_at DESC").Find(&configs).Error
	if err != nil {
		return nil, fmt.Errorf("failed to list Linstor configs: %w", err)
	}
	return configs, nil
}

// UpdateLinstorConfig updates an existing Linstor configuration
func (cm *LinstorConfigManager) UpdateLinstorConfig(configID int, input *LinstorConfigInput) (*database.LinstorConfig, error) {
	// Get existing config
	existing, err := cm.GetLinstorConfig(configID)
	if err != nil {
		return nil, err
	}

	log.WithFields(log.Fields{
		"config_id":   configID,
		"config_name": input.Name,
	}).Info("🔧 Updating Linstor configuration")

	// Validate configuration by testing connection
	if err := cm.validateLinstorConnection(input); err != nil {
		return nil, fmt.Errorf("Linstor connection validation failed: %w", err)
	}

	// Update fields
	existing.Name = input.Name
	existing.APIURL = input.APIURL
	existing.APIPort = input.APIPort
	existing.APIProtocol = input.APIProtocol
	existing.APIKey = input.APIKey
	existing.APISecret = input.APISecret
	existing.ConnectionTimeoutSeconds = input.ConnectionTimeoutSeconds
	existing.RetryAttempts = input.RetryAttempts
	existing.Description = input.Description

	if err := cm.db.GetGormDB().Save(existing).Error; err != nil {
		return nil, fmt.Errorf("failed to update Linstor config: %w", err)
	}

	log.WithField("config_id", configID).Info("✅ Linstor configuration updated")
	return existing, nil
}

// DeleteLinstorConfig marks a Linstor configuration as inactive (soft delete)
func (cm *LinstorConfigManager) DeleteLinstorConfig(configID int) error {
	log.WithField("config_id", configID).Info("🗑️ Deleting Linstor configuration")

	result := cm.db.GetGormDB().Model(&database.LinstorConfig{}).Where("id = ?", configID).Update("is_active", false)
	if result.Error != nil {
		return fmt.Errorf("failed to delete Linstor config: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("Linstor config with ID %d not found", configID)
	}

	log.WithField("config_id", configID).Info("✅ Linstor configuration deleted")
	return nil
}

// GetDefaultLinstorConfig returns the first active Linstor configuration
func (cm *LinstorConfigManager) GetDefaultLinstorConfig() (*database.LinstorConfig, error) {
	var config database.LinstorConfig
	err := cm.db.GetGormDB().Where("is_active = true").First(&config).Error
	if err != nil {
		return nil, fmt.Errorf("no default Linstor config found: %w", err)
	}
	return &config, nil
}

// LoadFromEnvironment loads Linstor configuration from environment variables
func (cm *LinstorConfigManager) LoadFromEnvironment(name string) (*database.LinstorConfig, error) {
	log.WithField("name", name).Info("🌍 Loading Linstor config from environment")

	apiURL := os.Getenv("LINSTOR_API_URL")
	if apiURL == "" {
		return nil, fmt.Errorf("LINSTOR_API_URL environment variable is required")
	}

	apiPort := 3370
	if portStr := os.Getenv("LINSTOR_API_PORT"); portStr != "" {
		if p, err := strconv.Atoi(portStr); err == nil {
			apiPort = p
		}
	}

	input := &LinstorConfigInput{
		Name:        name,
		APIURL:      apiURL,
		APIPort:     apiPort,
		APIProtocol: getEnvWithDefault("LINSTOR_API_PROTOCOL", "http"),
		APIKey:      os.Getenv("LINSTOR_API_KEY"),
		APISecret:   os.Getenv("LINSTOR_API_SECRET"),
		Description: "Configuration loaded from environment variables",
	}

	if timeoutStr := os.Getenv("LINSTOR_CONNECTION_TIMEOUT"); timeoutStr != "" {
		if t, err := strconv.Atoi(timeoutStr); err == nil {
			input.ConnectionTimeoutSeconds = t
		}
	}

	if retriesStr := os.Getenv("LINSTOR_RETRY_ATTEMPTS"); retriesStr != "" {
		if r, err := strconv.Atoi(retriesStr); err == nil {
			input.RetryAttempts = r
		}
	}

	return cm.CreateLinstorConfig(input)
}

// validateLinstorConnection validates connection to Linstor API
func (cm *LinstorConfigManager) validateLinstorConnection(input *LinstorConfigInput) error {
	// TODO: Implement actual Linstor API connection test
	// For now, just basic URL validation
	if input.APIURL == "" {
		return fmt.Errorf("API URL is required")
	}
	if input.APIPort <= 0 || input.APIPort > 65535 {
		return fmt.Errorf("API port must be between 1 and 65535")
	}

	log.WithFields(log.Fields{
		"api_url":  input.APIURL,
		"api_port": input.APIPort,
	}).Info("🔍 Validating Linstor connection")

	// TODO: Add actual HTTP call to Linstor API /v1/version or similar
	// client := &http.Client{Timeout: 10 * time.Second}
	// resp, err := client.Get(fmt.Sprintf("%s:%d/v1/version", input.APIURL, input.APIPort))

	return nil
}

// Helper function
func getEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
