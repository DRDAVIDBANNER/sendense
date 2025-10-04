// Package nbd provides NBD server configuration management for the Volume Daemon
package nbd

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

// ConfigManager handles atomic NBD server configuration file operations
type ConfigManager struct {
	configPath string
	confDir    string
	mutex      sync.RWMutex
}

// BaseConfig represents the required base NBD server configuration
type BaseConfig struct {
	Port      int    `json:"port"`
	AllowList bool   `json:"allowlist"`
	ConfDir   string `json:"includedir"`
}

// Export represents an NBD export configuration
type Export struct {
	Name       string            `json:"name"`
	DevicePath string            `json:"device_path"`
	ReadOnly   bool              `json:"read_only"`
	Metadata   map[string]string `json:"metadata"`
}

// NewConfigManager creates a new NBD configuration manager
func NewConfigManager(configPath, confDir string) *ConfigManager {
	cm := &ConfigManager{
		configPath: configPath,
		confDir:    confDir,
	}

	// Ensure base configuration exists on initialization
	if err := cm.EnsureBaseConfiguration(); err != nil {
		log.WithError(err).Warn("Failed to ensure base NBD configuration on startup")
	}

	return cm
}

// EnsureBaseConfiguration ensures the NBD config file exists with required base setup
func (cm *ConfigManager) EnsureBaseConfiguration() error {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	// Check if config file exists
	if _, err := os.Stat(cm.configPath); os.IsNotExist(err) {
		log.WithField("config_path", cm.configPath).Info("🔧 Creating NBD base configuration file")

		// Create the base configuration with dummy export
		baseConfig := cm.generateBaseConfig()
		if err := cm.writeConfigAtomic(baseConfig); err != nil {
			return fmt.Errorf("failed to create base NBD configuration: %w", err)
		}

		log.WithField("config_path", cm.configPath).Info("✅ Created NBD base configuration")
		return nil
	}

	// Config exists, ensure it has required sections
	currentConfig, err := cm.readConfig()
	if err != nil {
		return fmt.Errorf("failed to read existing config: %w", err)
	}

	// Check for required sections
	needsUpdate := false

	if !strings.Contains(currentConfig, "[generic]") {
		log.Warn("NBD config missing [generic] section - will add")
		needsUpdate = true
	}

	if !strings.Contains(currentConfig, "[dummy]") {
		log.Warn("NBD config missing [dummy] export - will add")
		needsUpdate = true
	}

	if needsUpdate {
		log.Info("🔧 Updating NBD configuration to include required sections")

		// Backup current config
		backupPath := cm.configPath + ".backup-ensure-" + time.Now().Format("20060102-150405")
		if err := cm.copyFile(cm.configPath, backupPath); err != nil {
			log.WithError(err).Warn("Failed to create backup during base config ensure")
		}

		// Generate complete base config with existing exports
		updatedConfig := cm.ensureRequiredSections(currentConfig)

		if err := cm.writeConfigAtomic(updatedConfig); err != nil {
			// Try to restore backup
			if restoreErr := cm.copyFile(backupPath, cm.configPath); restoreErr != nil {
				log.WithError(restoreErr).Error("Failed to restore backup after base config update failure")
			}
			return fmt.Errorf("failed to update NBD base configuration: %w", err)
		}

		// Clean up backup on success
		os.Remove(backupPath)

		log.Info("✅ Updated NBD configuration with required sections")
	}

	// Ensure conf.d directory exists
	if err := cm.ensureConfDir(); err != nil {
		log.WithError(err).Warn("Failed to ensure conf.d directory exists")
	}

	return nil
}

// ensureConfDir creates the conf.d directory if it doesn't exist
func (cm *ConfigManager) ensureConfDir() error {
	if cm.confDir == "" {
		return nil // No conf.d directory configured
	}

	if _, err := os.Stat(cm.confDir); os.IsNotExist(err) {
		log.WithField("conf_dir", cm.confDir).Info("🔧 Creating NBD conf.d directory")

		if err := os.MkdirAll(cm.confDir, 0755); err != nil {
			return fmt.Errorf("failed to create conf.d directory: %w", err)
		}

		log.WithField("conf_dir", cm.confDir).Info("✅ Created NBD conf.d directory")
	}

	return nil
}

// generateBaseConfig creates the minimal required NBD configuration
func (cm *ConfigManager) generateBaseConfig() string {
	config := `[generic]
port = 10809
allowlist = false`

	// Add includedir if conf.d is configured
	if cm.confDir != "" {
		config += fmt.Sprintf("\nincludedir = %s", cm.confDir)
	}

	// Add required dummy export
	config += `

# Dummy export required for NBD server to start
[dummy]
exportname = /dev/null
readonly = true
multifile = false
copyonwrite = false
`

	return config
}

// ensureRequiredSections ensures the config has [generic] and [dummy] sections
func (cm *ConfigManager) ensureRequiredSections(currentConfig string) string {
	lines := strings.Split(currentConfig, "\n")
	result := make([]string, 0, len(lines)+20) // Extra space for new sections

	hasGeneric := false
	hasDummy := false

	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)

		// Check for required sections
		if trimmedLine == "[generic]" {
			hasGeneric = true
		} else if trimmedLine == "[dummy]" {
			hasDummy = true
		}

		result = append(result, line)
	}

	config := strings.Join(result, "\n")

	// Add [generic] section if missing
	if !hasGeneric {
		genericSection := `[generic]
port = 10809
allowlist = false`
		if cm.confDir != "" {
			genericSection += fmt.Sprintf("\nincludedir = %s", cm.confDir)
		}
		genericSection += "\n\n"

		config = genericSection + config
	}

	// Add [dummy] section if missing
	if !hasDummy {
		dummySection := `
# Dummy export required for NBD server to start
[dummy]
exportname = /dev/null
readonly = true
multifile = false
copyonwrite = false
`
		config += dummySection
	}

	return config
}

// AddExport adds a new NBD export by creating an individual file in conf.d directory
func (cm *ConfigManager) AddExport(export *Export) error {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	log.WithFields(log.Fields{
		"export_name": export.Name,
		"device_path": export.DevicePath,
		"read_only":   export.ReadOnly,
		"conf_dir":    cm.confDir,
	}).Info("🔗 Adding NBD export to conf.d directory")

	// Create export file path in conf.d directory
	exportFilePath := filepath.Join(cm.confDir, export.Name+".conf")

	// Check if export file already exists
	if _, err := os.Stat(exportFilePath); err == nil {
		return fmt.Errorf("export '%s' already exists", export.Name)
	}

	// Create export configuration content
	exportConfig := fmt.Sprintf(`[%s]
exportname = %s
readonly = %t
multifile = false
copyonwrite = false
`, export.Name, export.DevicePath, export.ReadOnly)

	// Write export configuration to individual file
	log.WithFields(log.Fields{
		"export_file_path": exportFilePath,
		"config_content":   exportConfig,
		"file_mode":        "0644",
	}).Debug("About to write NBD export config file")

	if err := os.WriteFile(exportFilePath, []byte(exportConfig), 0644); err != nil {
		log.WithError(err).WithFields(log.Fields{
			"export_file_path": exportFilePath,
			"error_type":       fmt.Sprintf("%T", err),
		}).Error("Failed to write NBD export config file")
		return fmt.Errorf("failed to write export config file: %w", err)
	}

	log.WithField("export_file_path", exportFilePath).Info("✅ Successfully wrote NBD export config file")

	// Send SIGHUP to NBD server to reload configuration
	if err := cm.reloadNBDServer(); err != nil {
		log.WithError(err).Warn("Failed to reload NBD server after adding export - export created but may need manual reload")
		// Don't remove the file - the export was created successfully
		// Manual NBD server restart may be needed
	}

	log.WithFields(log.Fields{
		"export_name": export.Name,
		"export_file": exportFilePath,
	}).Info("✅ NBD export added successfully to conf.d")

	return nil
}

// RemoveExport removes an NBD export by deleting its individual file from conf.d directory
func (cm *ConfigManager) RemoveExport(exportName string) error {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	log.WithFields(log.Fields{
		"export_name": exportName,
		"conf_dir":    cm.confDir,
	}).Info("🗑️ Removing NBD export from conf.d directory")

	// Create export file path in conf.d directory
	exportFilePath := filepath.Join(cm.confDir, exportName+".conf")

	// Check if export file exists
	if _, err := os.Stat(exportFilePath); os.IsNotExist(err) {
		log.WithField("export_name", exportName).Warn("Export file not found - already removed")
		return nil // Not an error - idempotent operation
	}

	// Remove the export file
	if err := os.Remove(exportFilePath); err != nil {
		return fmt.Errorf("failed to remove export config file: %w", err)
	}

	// Send SIGHUP to NBD server to reload configuration
	if err := cm.reloadNBDServer(); err != nil {
		log.WithError(err).Warn("Failed to reload NBD server after removing export")
		return fmt.Errorf("failed to reload NBD server: %w", err)
	}

	log.WithFields(log.Fields{
		"export_name": exportName,
		"export_file": exportFilePath,
	}).Info("✅ NBD export removed successfully from conf.d")

	return nil
}

// ListExports returns all current NBD exports
func (cm *ConfigManager) ListExports() ([]*Export, error) {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	config, err := cm.readConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	return cm.parseExportsFromConfig(config), nil
}

// ExportExists checks if an export exists in the configuration
func (cm *ConfigManager) ExportExists(exportName string) (bool, error) {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	config, err := cm.readConfig()
	if err != nil {
		return false, fmt.Errorf("failed to read config: %w", err)
	}

	return cm.exportExists(config, exportName), nil
}

// ValidateConfig checks if the NBD configuration is valid
func (cm *ConfigManager) ValidateConfig() error {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	// Check if config file exists
	if _, err := os.Stat(cm.configPath); os.IsNotExist(err) {
		return fmt.Errorf("NBD config file does not exist: %s", cm.configPath)
	}

	// Try to parse the config
	config, err := cm.readConfig()
	if err != nil {
		return fmt.Errorf("invalid NBD config: %w", err)
	}

	// Check for [generic] section
	if !strings.Contains(config, "[generic]") {
		return fmt.Errorf("NBD config missing required [generic] section")
	}

	// Check that all export device paths exist (skip dummy exports)
	exports := cm.parseExportsFromConfig(config)
	for _, export := range exports {
		// parseExportsFromConfig already skips dummy exports, but double-check
		if export.Name != "dummy" && export.DevicePath != "/dev/null" {
			if _, err := os.Stat(export.DevicePath); os.IsNotExist(err) {
				log.WithFields(log.Fields{
					"export_name": export.Name,
					"device_path": export.DevicePath,
				}).Warn("NBD export device path does not exist")
			}
		}
	}

	return nil
}

// GetNBDServerPID returns the PID of the running NBD server
func (cm *ConfigManager) GetNBDServerPID() (int, error) {
	// First try reading systemd PID file (for systemd service)
	pidFile := "/run/nbd-server.pid"
	if pidData, err := os.ReadFile(pidFile); err == nil {
		var pid int
		if _, err := fmt.Sscanf(strings.TrimSpace(string(pidData)), "%d", &pid); err == nil {
			// Verify the process exists by checking /proc/PID (no permission needed)
			if _, err := os.Stat(fmt.Sprintf("/proc/%d", pid)); err == nil {
				log.WithField("pid", pid).Debug("✅ NBD server PID verified via /proc filesystem")
				return pid, nil
			} else {
				log.WithFields(log.Fields{
					"pid":   pid,
					"error": err,
				}).Debug("PID file exists but process not running")
			}
		}
	}

	// Fallback to pgrep for manual processes (backward compatibility)
	cmd := exec.Command("pgrep", "-f", "nbd-server.*"+filepath.Base(cm.configPath))
	output, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("NBD server not running or not found")
	}

	var pid int
	if _, err := fmt.Sscanf(strings.TrimSpace(string(output)), "%d", &pid); err != nil {
		return 0, fmt.Errorf("failed to parse NBD server PID: %w", err)
	}

	return pid, nil
}

// Private helper methods

func (cm *ConfigManager) readConfig() (string, error) {
	data, err := os.ReadFile(cm.configPath)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (cm *ConfigManager) writeConfigAtomic(content string) error {
	// Write to temporary file first
	tempFile := cm.configPath + ".tmp"

	if err := os.WriteFile(tempFile, []byte(content), 0644); err != nil {
		return err
	}

	// Atomic move
	return os.Rename(tempFile, cm.configPath)
}

func (cm *ConfigManager) copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}

func (cm *ConfigManager) exportExists(config, exportName string) bool {
	return strings.Contains(config, "["+exportName+"]")
}

func (cm *ConfigManager) addExportToConfig(config string, export *Export) string {
	// Build export section
	exportSection := fmt.Sprintf("\n[%s]\n", export.Name)
	exportSection += fmt.Sprintf("exportname = %s\n", export.DevicePath)
	exportSection += fmt.Sprintf("readonly = %t\n", export.ReadOnly)
	exportSection += "multifile = false\n"
	exportSection += "copyonwrite = false\n"

	// Add metadata as comments for reference
	if len(export.Metadata) > 0 {
		exportSection += "# Metadata:\n"
		for key, value := range export.Metadata {
			exportSection += fmt.Sprintf("# %s = %s\n", key, value)
		}
	}

	return config + exportSection
}

func (cm *ConfigManager) removeExportFromConfig(config, exportName string) string {
	lines := strings.Split(config, "\n")
	result := make([]string, 0, len(lines))

	inExportSection := false
	exportSectionName := "[" + exportName + "]"

	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)

		// Check if we're entering the export section to remove
		if trimmedLine == exportSectionName {
			inExportSection = true
			continue
		}

		// Check if we're entering a different section
		if strings.HasPrefix(trimmedLine, "[") && strings.HasSuffix(trimmedLine, "]") && trimmedLine != exportSectionName {
			inExportSection = false
		}

		// Only include lines that are not in the export section to remove
		if !inExportSection {
			result = append(result, line)
		}
	}

	return strings.Join(result, "\n")
}

func (cm *ConfigManager) parseExportsFromConfig(config string) []*Export {
	var exports []*Export
	lines := strings.Split(config, "\n")

	var currentExport *Export

	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)

		// Check for section header
		if strings.HasPrefix(trimmedLine, "[") && strings.HasSuffix(trimmedLine, "]") {
			// Save previous export if exists
			if currentExport != nil {
				exports = append(exports, currentExport)
			}

			// Start new export (skip [generic] and [dummy] sections)
			sectionName := trimmedLine[1 : len(trimmedLine)-1]
			if sectionName != "generic" && sectionName != "dummy" {
				currentExport = &Export{
					Name:     sectionName,
					Metadata: make(map[string]string),
				}
			} else {
				currentExport = nil
			}
			continue
		}

		// Parse export properties
		if currentExport != nil && strings.Contains(trimmedLine, "=") {
			parts := strings.SplitN(trimmedLine, "=", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])

				switch key {
				case "exportname":
					currentExport.DevicePath = value
				case "readonly":
					currentExport.ReadOnly = value == "true"
				default:
					// Store other properties as metadata
					currentExport.Metadata[key] = value
				}
			}
		}
	}

	// Don't forget the last export
	if currentExport != nil {
		exports = append(exports, currentExport)
	}

	return exports
}

func (cm *ConfigManager) reloadNBDServer() error {
	pid, err := cm.GetNBDServerPID()
	if err != nil {
		return fmt.Errorf("failed to get NBD server PID: %w", err)
	}

	// Send SIGHUP to reload configuration (use sudo for permission)
	cmd := exec.Command("sudo", "kill", "-HUP", fmt.Sprintf("%d", pid))
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to send SIGHUP to NBD server (PID %d): %w", pid, err)
	}

	log.WithField("pid", pid).Info("✅ Sent SIGHUP to NBD server for configuration reload")
	return nil
}
