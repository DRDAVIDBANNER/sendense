// Package nbd provides NBD server configuration management for volume exports
package nbd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	log "github.com/sirupsen/logrus"
)

// Config represents NBD server configuration for a volume export
type Config struct {
	Port       int    `json:"port"`        // NBD server port
	ExportName string `json:"export_name"` // Export name for NBD client access
	DevicePath string `json:"device_path"` // Block device path (e.g., /dev/vdb)
	JobID      string `json:"job_id"`      // Associated migration job ID
	ConfigPath string `json:"config_path"` // Generated config file path
}

// ConfigManager handles NBD server configuration generation
type ConfigManager struct {
	configDir    string // Directory for NBD config files (/etc/nbd-server/)
	portRange    PortRange
	templatePath string
}

// PortRange defines the range of ports available for NBD servers
type PortRange struct {
	Start int // Starting port (e.g., 10800)
	End   int // Ending port (e.g., 11000)
}

// NewConfigManager creates a new NBD configuration manager
func NewConfigManager() *ConfigManager {
	return &ConfigManager{
		configDir: "/opt/migratekit/nbd-configs", // Use oma-owned directory
		portRange: PortRange{
			Start: 10800,
			End:   11000,
		},
		templatePath: "/opt/migratekit/nbd-configs/config-template",
	}
}

// NewConfigManagerWithDir creates a new NBD configuration manager with custom directory
func NewConfigManagerWithDir(configDir string) *ConfigManager {
	return &ConfigManager{
		configDir: configDir,
		portRange: PortRange{
			Start: 10800,
			End:   11000,
		},
		templatePath: "/etc/nbd-server/config-template",
	}
}

// GenerateConfig creates NBD server configuration for a volume
func (cm *ConfigManager) GenerateConfig(jobID, devicePath string) (*Config, error) {
	log.WithFields(log.Fields{
		"job_id":      jobID,
		"device_path": devicePath,
	}).Info("Generating NBD server configuration")

	// Allocate dynamic port
	port, err := cm.allocatePort()
	if err != nil {
		return nil, fmt.Errorf("failed to allocate port: %w", err)
	}

	// Use unique export name per job for single-port architecture
	// Job ID is already unique, no need for timestamp
	exportName := fmt.Sprintf("migration-%s", jobID)

	// Create configuration
	config := &Config{
		Port:       port,
		ExportName: exportName,
		DevicePath: devicePath,
		JobID:      jobID,
		ConfigPath: filepath.Join(cm.configDir, fmt.Sprintf("config-dynamic-%d", port)),
	}

	// Generate configuration file
	if err := cm.writeConfigFile(config); err != nil {
		return nil, fmt.Errorf("failed to write config file: %w", err)
	}

	log.WithFields(log.Fields{
		"job_id":      jobID,
		"port":        port,
		"export_name": exportName,
		"config_path": config.ConfigPath,
	}).Info("✅ NBD server configuration generated")

	return config, nil
}

// allocatePort returns the single NBD port (10809) for all jobs
// Multiple jobs are handled via unique export names on the same port
func (cm *ConfigManager) allocatePort() (int, error) {
	// Always use port 10808 for hybrid tunnel architecture
	// This matches VMA stunnel client: localhost:10808 -> OMA stunnel -> NBD:10809
	const singleNBDPort = 10808

	log.WithField("port", singleNBDPort).Debug("Using single NBD port for all jobs")
	return singleNBDPort, nil
}

// isPortAvailable checks if a port is available for binding
func (cm *ConfigManager) isPortAvailable(port int) bool {
	// Use ss command to check if port is in use
	cmd := exec.Command("ss", "-tln", "sport", fmt.Sprintf("= :%d", port))
	output, err := cmd.Output()
	if err != nil {
		log.WithError(err).WithField("port", port).Warn("Failed to check port availability")
		return false
	}

	// If port is found in output, it's in use
	return !strings.Contains(string(output), fmt.Sprintf(":%d", port))
}

// writeConfigFile generates the NBD server configuration file using sudo
func (cm *ConfigManager) writeConfigFile(config *Config) error {
	// NBD server configuration template (base + dynamic export)
	// Generic section must not include user/group restrictions; allowlist disabled
	configTemplate := `[generic]
port = {{.Port}}
allowlist = false

[{{.ExportName}}]
exportname = {{.DevicePath}}
readonly = false
multifile = false
copyonwrite = false
`

	// Parse template
	tmpl, err := template.New("nbd-config").Parse(configTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse config template: %w", err)
	}

	// Execute template to string buffer
	var buf strings.Builder
	if err := tmpl.Execute(&buf, config); err != nil {
		return fmt.Errorf("failed to execute config template: %w", err)
	}

	// Write config file directly (oma user owns the config directory)
	configContent := buf.String()
	if err := os.WriteFile(config.ConfigPath, []byte(configContent), 0644); err != nil {
		return fmt.Errorf("failed to write config file %s: %w", config.ConfigPath, err)
	}

	log.WithFields(log.Fields{
		"config_path": config.ConfigPath,
		"port":        config.Port,
		"device_path": config.DevicePath,
	}).Info("📝 NBD configuration file created")

	return nil
}

// Note: writeSudoFile function removed - using direct file operations instead

// RemoveConfig removes NBD server configuration file
func (cm *ConfigManager) RemoveConfig(config *Config) error {
	log.WithFields(log.Fields{
		"config_path": config.ConfigPath,
		"job_id":      config.JobID,
		"port":        config.Port,
	}).Info("Removing NBD server configuration")

	// Remove config file directly (oma user owns the config directory)
	if err := os.Remove(config.ConfigPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove config file %s: %w", config.ConfigPath, err)
	}

	log.WithField("config_path", config.ConfigPath).Info("🗑️ NBD configuration file removed")
	return nil
}

// ValidateConfig checks if configuration is valid and device exists
func (cm *ConfigManager) ValidateConfig(config *Config) error {
	// Check if device exists
	if _, err := os.Stat(config.DevicePath); err != nil {
		return fmt.Errorf("device %s does not exist: %w", config.DevicePath, err)
	}

	// Check if port is in valid range
	if config.Port < cm.portRange.Start || config.Port > cm.portRange.End {
		return fmt.Errorf("port %d is outside valid range %d-%d", config.Port, cm.portRange.Start, cm.portRange.End)
	}

	log.WithFields(log.Fields{
		"port":        config.Port,
		"device_path": config.DevicePath,
		"export_name": config.ExportName,
	}).Debug("✅ NBD configuration validation passed")

	return nil
}
