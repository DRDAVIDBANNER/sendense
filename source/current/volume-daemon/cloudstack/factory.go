package cloudstack

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"
	log "github.com/sirupsen/logrus"
	
	"github.com/vexxhost/migratekit-volume-daemon/service"
)

// OSSEAConfig represents the database structure for CloudStack configuration
type OSSEAConfig struct {
	ID        int    `db:"id"`
	Name      string `db:"name"`
	APIURL    string `db:"api_url"`
	APIKey    string `db:"api_key"`
	SecretKey string `db:"secret_key"`
	Domain    string `db:"domain"`
	Zone      string `db:"zone"`
	IsActive  bool   `db:"is_active"`
}

// Factory creates CloudStack clients from database configuration
type Factory struct {
	db *sqlx.DB
}

// NewFactory creates a new CloudStack client factory
func NewFactory(db *sqlx.DB) *Factory {
	return &Factory{db: db}
}

// CreateClient creates a CloudStack client using the active configuration from database
func (f *Factory) CreateClient(ctx context.Context) (service.CloudStackClient, error) {
	config, err := f.getActiveConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get active CloudStack config: %w", err)
	}

	clientConfig := CloudStackConfig{
		APIURL:    config.APIURL,
		APIKey:    config.APIKey,
		SecretKey: config.SecretKey,
		Domain:    config.Domain,
		Zone:      config.Zone,
	}

	log.WithFields(log.Fields{
		"config_name": config.Name,
		"zone":        config.Zone,
		"domain":      config.Domain,
	}).Info("Creating CloudStack client from database configuration")

	return NewClient(clientConfig), nil
}

// CreateClientByName creates a CloudStack client using a specific named configuration
func (f *Factory) CreateClientByName(ctx context.Context, name string) (service.CloudStackClient, error) {
	config, err := f.getConfigByName(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("failed to get CloudStack config '%s': %w", name, err)
	}

	clientConfig := CloudStackConfig{
		APIURL:    config.APIURL,
		APIKey:    config.APIKey,
		SecretKey: config.SecretKey,
		Domain:    config.Domain,
		Zone:      config.Zone,
	}

	log.WithFields(log.Fields{
		"config_name": config.Name,
		"zone":        config.Zone,
		"domain":      config.Domain,
	}).Info("Creating CloudStack client from named configuration")

	return NewClient(clientConfig), nil
}

// GetActiveConfig retrieves the active CloudStack configuration
func (f *Factory) GetActiveConfig(ctx context.Context) (*OSSEAConfig, error) {
	return f.getActiveConfig(ctx)
}

// ListConfigs lists all available CloudStack configurations
func (f *Factory) ListConfigs(ctx context.Context) ([]OSSEAConfig, error) {
	query := `
		SELECT id, name, api_url, api_key, secret_key, domain, zone, is_active
		FROM ossea_configs 
		ORDER BY name
	`

	var configs []OSSEAConfig
	err := f.db.SelectContext(ctx, &configs, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list CloudStack configs: %w", err)
	}

	log.WithFields(log.Fields{
		"config_count": len(configs),
	}).Debug("Listed CloudStack configurations")

	return configs, nil
}

// getActiveConfig retrieves the active CloudStack configuration from database
func (f *Factory) getActiveConfig(ctx context.Context) (*OSSEAConfig, error) {
	query := `
		SELECT id, name, api_url, api_key, secret_key, domain, zone, is_active
		FROM ossea_configs 
		WHERE is_active = true
		LIMIT 1
	`

	var config OSSEAConfig
	err := f.db.GetContext(ctx, &config, query)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("no active CloudStack configuration found")
		}
		return nil, fmt.Errorf("failed to query active CloudStack config: %w", err)
	}

	log.WithFields(log.Fields{
		"config_id":   config.ID,
		"config_name": config.Name,
		"zone":        config.Zone,
	}).Debug("Retrieved active CloudStack configuration")

	return &config, nil
}

// getConfigByName retrieves a specific CloudStack configuration by name
func (f *Factory) getConfigByName(ctx context.Context, name string) (*OSSEAConfig, error) {
	query := `
		SELECT id, name, api_url, api_key, secret_key, domain, zone, is_active
		FROM ossea_configs 
		WHERE name = ?
		LIMIT 1
	`

	var config OSSEAConfig
	err := f.db.GetContext(ctx, &config, query, name)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("CloudStack configuration '%s' not found", name)
		}
		return nil, fmt.Errorf("failed to query CloudStack config '%s': %w", name, err)
	}

	log.WithFields(log.Fields{
		"config_id":   config.ID,
		"config_name": config.Name,
		"zone":        config.Zone,
		"is_active":   config.IsActive,
	}).Debug("Retrieved CloudStack configuration by name")

	return &config, nil
}

// TestConnection tests the connection to CloudStack using the active configuration
func (f *Factory) TestConnection(ctx context.Context) error {
	client, err := f.CreateClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to create CloudStack client: %w", err)
	}

	return client.Ping(ctx)
}
