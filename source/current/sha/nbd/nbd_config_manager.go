// Package nbd provides NBD server configuration management for SHA
// Following Volume Daemon's proven config.d + SIGHUP architecture
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

// NBDConfigManager handles atomic NBD server configuration file operations
// Following the proven Volume Daemon pattern: config.d + SIGHUP reload
type NBDConfigManager struct {
	configPath string    // Base NBD config file path
	confDir    string    // Individual export config directory
	port       int       // NBD server port
	mutex      sync.RWMutex
}

// NBDExport represents an NBD export configuration
// Supports both block devices and QCOW2 files
type NBDExport struct {
	Name       string            `json:"name"`        // Export name (must be unique, <64 chars)
	ExportPath string            `json:"export_path"` // Block device OR file path
	ReadOnly   bool              `json:"read_only"`   // Read-only access
	IsFile     bool              `json:"is_file"`     // true for QCOW2 files, false for block devices
	Metadata   map[string]string `json:"metadata"`    // Additional metadata
}

// NewNBDConfigManager creates a new NBD configuration manager
func NewNBDConfigManager(configPath, confDir string, port int) *NBDConfigManager {
	cm := &NBDConfigManager{
		configPath: configPath,
		confDir:    confDir,
		port:       port,
	}

	// Ensure base configuration exists on initialization
	if err := cm.EnsureBaseConfiguration(); err != nil {
		log.WithError(err).Warn("Failed to ensure base NBD configuration on startup")
	}

	return cm
}

// EnsureBaseConfiguration ensures the NBD config file exists with required base setup
func (cm *NBDConfigManager) EnsureBaseConfiguration() error {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	// Check if config file exists
	if _, err := os.Stat(cm.configPath); os.IsNotExist(err) {
		log.WithField("config_path", cm.configPath).Info("🔧 Creating NBD base configuration file")

		// Create the base configuration
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

	if !strings.Contains(currentConfig, fmt.Sprintf("includedir = %s", cm.confDir)) {
		log.Warn("NBD config missing includedir directive - will add")
		needsUpdate = true
	}

	if needsUpdate {
		log.Info("🔧 Updating NBD configuration to include required sections")

		// Backup current config
		backupPath := cm.configPath + ".backup-ensure-" + time.Now().Format("20060102-150405")
		if err := cm.copyFile(cm.configPath, backupPath); err != nil {
			log.WithError(err).Warn("Failed to create backup during base config ensure")
		}

		// Generate complete base config
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
func (cm *NBDConfigManager) ensureConfDir() error {
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
func (cm *NBDConfigManager) generateBaseConfig() string {
	config := fmt.Sprintf(`[generic]
port = %d
allowlist = false`, cm.port)

	// Add includedir if conf.d is configured
	if cm.confDir != "" {
		config += fmt.Sprintf("\nincludedir = %s", cm.confDir)
	}

	// Note: No dummy export needed - conf.d exports will be loaded

	return config + "\n"
}

// ensureRequiredSections ensures the config has [generic] section and includedir
func (cm *NBDConfigManager) ensureRequiredSections(currentConfig string) string {
	lines := strings.Split(currentConfig, "\n")
	result := make([]string, 0, len(lines)+10) // Extra space for new sections

	hasGeneric := false
	hasIncludedir := false

	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)

		// Check for required sections
		if trimmedLine == "[generic]" {
			hasGeneric = true
		} else if strings.HasPrefix(trimmedLine, "includedir = ") {
			hasIncludedir = true
		}

		result = append(result, line)
	}

	config := strings.Join(result, "\n")

	// Add [generic] section if missing
	if !hasGeneric {
		genericSection := fmt.Sprintf(`[generic]
port = %d
allowlist = false
`, cm.port)
		if cm.confDir != "" {
			genericSection += fmt.Sprintf("includedir = %s\n", cm.confDir)
			hasIncludedir = true // We just added it
		}
		genericSection += "\n"

		config = genericSection + config
	}

	// Add includedir if missing (and [generic] exists)
	if !hasIncludedir && cm.confDir != "" {
		// Find [generic] section and add includedir
		lines := strings.Split(config, "\n")
		for i, line := range lines {
			if strings.TrimSpace(line) == "[generic]" {
				// Insert includedir after [generic] section
				// Find the end of [generic] section
				insertIndex := i + 1
				for insertIndex < len(lines) {
					nextLine := strings.TrimSpace(lines[insertIndex])
					if nextLine == "" || strings.HasPrefix(nextLine, "[") {
						break
					}
					insertIndex++
				}
				// Insert includedir
				includeDir := fmt.Sprintf("includedir = %s", cm.confDir)
				lines = append(lines[:insertIndex], append([]string{includeDir}, lines[insertIndex:]...)...)
				break
			}
		}
		config = strings.Join(lines, "\n")
	}

	return config
}

// AddExport adds a new NBD export by creating an individual file in conf.d directory
func (cm *NBDConfigManager) AddExport(export *NBDExport) error {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	log.WithFields(log.Fields{
		"export_name": export.Name,
		"export_path": export.ExportPath,
		"read_only":   export.ReadOnly,
		"is_file":     export.IsFile,
		"conf_dir":    cm.confDir,
	}).Info("🔗 Adding NBD export to conf.d directory")

	// Create export file path in conf.d directory
	exportFilePath := filepath.Join(cm.confDir, export.Name+".conf")

	// Check if export file already exists
	if _, err := os.Stat(exportFilePath); err == nil {
		return fmt.Errorf("export '%s' already exists", export.Name)
	}

	// Create export configuration content
	// Extract size from metadata (NBD uses 'size' not 'filesize')
	filesize := export.Metadata["file_size"]
	
	exportConfig := fmt.Sprintf(`[%s]
exportname = %s
readonly = %t
multifile = false
copyonwrite = false
`, export.Name, export.ExportPath, export.ReadOnly)
	
	// Add size parameter for file exports (NBD server uses 'size' for virtual disk size)
	if filesize != "" {
		exportConfig += fmt.Sprintf("size = %s\n", filesize)
	}

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
func (cm *NBDConfigManager) RemoveExport(exportName string) error {
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

// ListExports returns all current NBD exports from conf.d directory
func (cm *NBDConfigManager) ListExports() ([]*NBDExport, error) {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	// Read all .conf files from conf.d directory
	files, err := os.ReadDir(cm.confDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []*NBDExport{}, nil // No exports yet
		}
		return nil, fmt.Errorf("failed to read conf.d directory: %w", err)
	}

	var exports []*NBDExport

	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".conf") {
			continue
		}

		// Parse export configuration
		exportName := strings.TrimSuffix(file.Name(), ".conf")
		exportPath := filepath.Join(cm.confDir, file.Name())

		// Read and parse the config file
		export, err := cm.parseExportFile(exportPath, exportName)
		if err != nil {
			log.WithError(err).WithField("file", exportPath).Warn("Failed to parse export file")
			continue
		}

		exports = append(exports, export)
	}

	return exports, nil
}

// parseExportFile parses an individual export configuration file
func (cm *NBDConfigManager) parseExportFile(filePath, exportName string) (*NBDExport, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read export file: %w", err)
	}

	export := &NBDExport{
		Name:     exportName,
		Metadata: make(map[string]string),
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "[") {
			continue
		}

		if strings.Contains(line, "=") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])

				switch key {
				case "exportname":
					export.ExportPath = value
					// Determine if it's a file or block device
					export.IsFile = !strings.HasPrefix(value, "/dev/")
				case "readonly":
					export.ReadOnly = value == "true"
				default:
					export.Metadata[key] = value
				}
			}
		}
	}

	return export, nil
}

// ExportExists checks if an export exists in the conf.d directory
func (cm *NBDConfigManager) ExportExists(exportName string) (bool, error) {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	exportFilePath := filepath.Join(cm.confDir, exportName+".conf")
	_, err := os.Stat(exportFilePath)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to check export file: %w", err)
	}

	return true, nil
}

// reloadNBDServer sends SIGHUP to the NBD server to reload configuration
func (cm *NBDConfigManager) reloadNBDServer() error {
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

// GetNBDServerPID returns the PID of the running NBD server
func (cm *NBDConfigManager) GetNBDServerPID() (int, error) {
	// First try reading systemd PID file (for systemd service)
	pidFile := "/run/nbd-server.pid"
	if pidData, err := os.ReadFile(pidFile); err == nil {
		var pid int
		if _, err := fmt.Sscanf(strings.TrimSpace(string(pidData)), "%d", &pid); err == nil {
			// Verify the process exists by checking /proc/PID
			if _, err := os.Stat(fmt.Sprintf("/proc/%d", pid)); err == nil {
				log.WithField("pid", pid).Debug("✅ NBD server PID verified via /proc filesystem")
				return pid, nil
			}
		}
	}

	// Fallback to pgrep for manual processes
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

func (cm *NBDConfigManager) readConfig() (string, error) {
	data, err := os.ReadFile(cm.configPath)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (cm *NBDConfigManager) writeConfigAtomic(content string) error {
	// Write to temporary file first
	tempFile := cm.configPath + ".tmp"

	if err := os.WriteFile(tempFile, []byte(content), 0644); err != nil {
		return err
	}

	// Atomic move
	return os.Rename(tempFile, cm.configPath)
}

func (cm *NBDConfigManager) copyFile(src, dst string) error {
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
