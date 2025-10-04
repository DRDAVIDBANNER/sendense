// Package nbd provides NBD server lifecycle management
package nbd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/vexxhost/migratekit-oma/common"
	"github.com/vexxhost/migratekit-oma/database"
)

// Server represents a running NBD server instance
type Server struct {
	Config  *Config     `json:"config"` // NBD configuration
	Process *os.Process `json:"-"`      // Running process (not serialized)
	PID     int         `json:"pid"`    // Process ID
	Status  string      `json:"status"` // running, stopped, error
}

// ServerManager handles NBD server lifecycle operations
type ServerManager struct {
	configManager *ConfigManager
	servers       map[int]*Server // Port -> Server mapping
}

// NewServerManager creates a new NBD server manager
func NewServerManager() *ServerManager {
	return &ServerManager{
		configManager: NewConfigManager(),
		servers:       make(map[int]*Server),
	}
}

// StartServer starts an NBD server for the given configuration
func (sm *ServerManager) StartServer(config *Config) (*Server, error) {
	log.WithFields(log.Fields{
		"port":        config.Port,
		"export_name": config.ExportName,
		"device_path": config.DevicePath,
		"config_path": config.ConfigPath,
	}).Info("Starting NBD server")

	// Check if NBD server is already running on this port (single port architecture)
	if sm.isNBDServerRunning(config.Port) {
		log.WithFields(log.Fields{
			"port":        config.Port,
			"export_name": config.ExportName,
		}).Info("NBD server already running on port, adding new export via SIGHUP reload")

		return sm.addExportToRunningServer(nil, config)
	}

	// Validate configuration exists
	if err := sm.configManager.ValidateConfig(config); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	// Ensure base config exists and contains the export for this job
	basePath := "/etc/nbd-server/config-base"

	// Read existing base config if present; otherwise, create with required generic header
	var existingContent string
	if _, err := os.Stat(basePath); err == nil {
		contentBytes, readErr := os.ReadFile(basePath)
		if readErr != nil {
			return nil, fmt.Errorf("failed to read base NBD config: %w", readErr)
		}
		existingContent = string(contentBytes)
	} else {
		existingContent = "[generic]\nport = 10809\nallowlist = false\n"
	}

	// Append export section for this job if not already present
	exportSection := fmt.Sprintf("\n[%s]\nexportname = %s\nreadonly = false\nmultifile = false\ncopyonwrite = false\n",
		config.ExportName, config.DevicePath)
	updatedContent := existingContent + exportSection

	// Write updated base config via helper (handles permissions)
	tempFile := fmt.Sprintf("/opt/migratekit/nbd-config-update-%s-initial", config.ExportName)
	if err := os.WriteFile(tempFile, []byte(updatedContent), 0644); err != nil {
		return nil, fmt.Errorf("failed to write temp base config: %w", err)
	}
	updateCmd := exec.Command("sudo", "/usr/local/bin/oma-nbd-helper", "update-config", basePath, tempFile)
	if output, err := updateCmd.CombinedOutput(); err != nil {
		os.Remove(tempFile)
		log.WithFields(log.Fields{"error": err.Error(), "output": string(output)}).Error("Failed to update base NBD config")
		return nil, fmt.Errorf("failed to update base NBD config via helper: %w", err)
	}
	os.Remove(tempFile)

	// Start NBD server process using the NBD helper script and base config
	cmd := exec.Command("sudo", "/usr/local/bin/oma-nbd-helper", "start-server", basePath)

	// Set process attributes
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true, // Create new process group for clean shutdown
	}

	log.WithFields(log.Fields{
		"command":     cmd.String(),
		"config_path": config.ConfigPath,
		"port":        config.Port,
	}).Info("Executing NBD server command")

	// Start the process
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start NBD server: %w", err)
	}

	// Create server instance
	server := &Server{
		Config:  config,
		Process: cmd.Process,
		PID:     cmd.Process.Pid,
		Status:  "running",
	}

	// Store server reference
	sm.servers[config.Port] = server

	log.WithFields(log.Fields{
		"pid":         server.PID,
		"port":        config.Port,
		"export_name": config.ExportName,
	}).Info("✅ NBD server started successfully")

	// Start monitoring in background
	go sm.monitorServer(server)

	return server, nil
}

// StopServer stops an NBD server
func (sm *ServerManager) StopServer(port int) error {
	log.WithField("port", port).Info("Stopping NBD server")

	server, exists := sm.servers[port]
	if !exists {
		return fmt.Errorf("no NBD server found on port %d", port)
	}

	if server.Status != "running" {
		log.WithField("port", port).Info("NBD server already stopped")
		return nil
	}

	// For NBD servers started with sudo, we need to find and kill the actual nbd-server process
	// First try to kill via the process we tracked
	if server.Process != nil {
		if err := server.Process.Signal(syscall.SIGTERM); err != nil {
			log.WithError(err).WithField("pid", server.PID).Warn("Failed to send TERM signal to tracked process")
		}
	}

	// Also try to kill the actual nbd-server process by port
	if err := sm.killNBDServerByPort(server.Config.Port); err != nil {
		log.WithError(err).WithField("port", server.Config.Port).Warn("Failed to kill NBD server by port")
		return fmt.Errorf("failed to stop NBD server on port %d: %w", server.Config.Port, err)
	}

	// Wait for process to exit (with timeout)
	if server.Process != nil {
		done := make(chan error, 1)
		go func() {
			_, err := server.Process.Wait()
			done <- err
		}()

		select {
		case err := <-done:
			if err != nil {
				log.WithError(err).WithField("pid", server.PID).Warn("Process wait returned error")
			}
		case <-time.After(5 * time.Second):
			log.WithField("pid", server.PID).Warn("Process did not exit within timeout, force killing")
			server.Process.Kill()
		}
	}

	// Update server status
	server.Status = "stopped"
	server.Process = nil

	log.WithFields(log.Fields{
		"port": port,
		"pid":  server.PID,
	}).Info("✅ NBD server stopped")

	return nil
}

// GetServerStatus returns the status of an NBD server
func (sm *ServerManager) GetServerStatus(port int) (*Server, error) {
	server, exists := sm.servers[port]
	if !exists {
		return nil, fmt.Errorf("no NBD server found on port %d", port)
	}

	// For NBD daemons, check if port is still accessible instead of process
	if server.Status == "running" {
		if err := sm.CheckPortHealth(port); err != nil {
			// Daemon no longer accessible
			server.Status = "stopped"
			server.Process = nil
		}
	}

	return server, nil
}

// ListServers returns all managed NBD servers
func (sm *ServerManager) ListServers() map[int]*Server {
	// Update status for all servers
	for port := range sm.servers {
		sm.GetServerStatus(port) // Updates status
	}
	return sm.servers
}

// CheckPortHealth verifies NBD server is listening on the expected port
func (sm *ServerManager) CheckPortHealth(port int) error {
	log.WithField("port", port).Debug("Checking NBD server port health")

	// Use ss command to check if port is listening
	cmd := exec.Command("ss", "-tln", "sport", fmt.Sprintf("= :%d", port))
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to check port %d: %w", port, err)
	}

	if !strings.Contains(string(output), fmt.Sprintf(":%d", port)) {
		return fmt.Errorf("NBD server not listening on port %d", port)
	}

	log.WithField("port", port).Debug("✅ NBD server port health check passed")
	return nil
}

// monitorServer monitors NBD server process in background
func (sm *ServerManager) monitorServer(server *Server) {
	log.WithFields(log.Fields{
		"pid":  server.PID,
		"port": server.Config.Port,
	}).Info("Starting NBD server monitoring")

	if server.Process == nil {
		return
	}

	// Wait for initial process to exit (NBD server forks/daemonizes)
	processState, _ := server.Process.Wait()

	log.WithFields(log.Fields{
		"pid":       server.PID,
		"port":      server.Config.Port,
		"exit_code": processState.ExitCode(),
	}).Debug("Initial NBD server process exited (normal for daemon)")

	// For NBD servers, the initial process exits after forking daemon
	// Check if daemon is actually running by checking port
	time.Sleep(1 * time.Second) // Give daemon time to start

	if err := sm.CheckPortHealth(server.Config.Port); err != nil {
		server.Status = "error"
		log.WithFields(log.Fields{
			"port":  server.Config.Port,
			"error": err,
		}).Error("NBD daemon failed to start - port not accessible")
	} else {
		// Daemon is running successfully
		log.WithField("port", server.Config.Port).Info("✅ NBD daemon started successfully")
		server.Status = "running"
		// Note: server.Process is now nil since the original process exited
		// But the daemon is running and accessible
	}

	server.Process = nil
}

// CleanupServer removes server from tracking and cleans up configuration
func (sm *ServerManager) CleanupServer(port int) error {
	log.WithField("port", port).Info("Cleaning up NBD server")

	// Stop server if running
	if err := sm.StopServer(port); err != nil {
		log.WithError(err).WithField("port", port).Warn("Failed to stop server during cleanup")
	}

	// Get server reference
	server, exists := sm.servers[port]
	if !exists {
		return fmt.Errorf("no server found on port %d", port)
	}

	// Remove configuration file
	if err := sm.configManager.RemoveConfig(server.Config); err != nil {
		log.WithError(err).WithField("port", port).Warn("Failed to remove config during cleanup")
	}

	// Remove from tracking
	delete(sm.servers, port)

	log.WithField("port", port).Info("✅ NBD server cleanup completed")
	return nil
}

// RestartServer restarts an NBD server
func (sm *ServerManager) RestartServer(port int) error {
	log.WithField("port", port).Info("Restarting NBD server")

	server, exists := sm.servers[port]
	if !exists {
		return fmt.Errorf("no NBD server found on port %d", port)
	}

	// Stop existing server
	if err := sm.StopServer(port); err != nil {
		return fmt.Errorf("failed to stop server for restart: %w", err)
	}

	// Wait a moment for cleanup
	time.Sleep(1 * time.Second)

	// Start server again
	_, err := sm.StartServer(server.Config)
	if err != nil {
		return fmt.Errorf("failed to start server after restart: %w", err)
	}

	log.WithField("port", port).Info("✅ NBD server restarted successfully")
	return nil
}

// killNBDServerByPort finds and kills NBD server process by port using the NBD helper script
func (sm *ServerManager) killNBDServerByPort(port int) error {
	// Find the NBD server process using the config file pattern
	configFile := fmt.Sprintf("config-dynamic-%d", port)

	// Use the NBD helper script to kill NBD server process by config file pattern
	cmd := exec.Command("sudo", "/usr/local/bin/oma-nbd-helper", "kill-server", configFile)
	if err := cmd.Run(); err != nil {
		// kill-server returns error if no matching process found, which is not always an error
		log.WithField("port", port).Debug("NBD helper kill-server returned error (may be normal if process already stopped)")
	}

	log.WithField("port", port).Debug("Attempted to kill NBD server by port")
	return nil
}

// addExportToRunningServer appends a new export to existing NBD server and reloads via SIGHUP
func (sm *ServerManager) addExportToRunningServer(existingServer *Server, newConfig *Config) (*Server, error) {
	log.WithFields(log.Fields{
		"new_export":  newConfig.ExportName,
		"port":        newConfig.Port,
		"device_path": newConfig.DevicePath,
	}).Info("Adding new export to existing NBD server")

	// 1. Append new export section to the main NBD config file
	// Switched to config-base to avoid conflicts with system-managed files
	mainConfigPath := "/etc/nbd-server/config-base"

	// Read existing config
	existingContent, err := os.ReadFile(mainConfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read existing NBD config: %w", err)
	}

	// Append new export section
	newExportSection := fmt.Sprintf("\n[%s]\nexportname = %s\nreadonly = false\nmultifile = false\ncopyonwrite = false\n",
		newConfig.ExportName, newConfig.DevicePath)

	updatedContent := string(existingContent) + newExportSection

	// Write updated config using oma-nbd-helper (OMA user has sudo access to this script)
	// Use /opt/migratekit for temp files (systemd PrivateTmp=yes isolates /tmp)
	tempFile := fmt.Sprintf("/opt/migratekit/nbd-config-update-%s", newConfig.ExportName)
	log.WithFields(log.Fields{
		"temp_file":      tempFile,
		"main_config":    mainConfigPath,
		"content_length": len(updatedContent),
	}).Debug("Writing temp config file for NBD update")

	if err := os.WriteFile(tempFile, []byte(updatedContent), 0644); err != nil {
		return nil, fmt.Errorf("failed to write temp config: %w", err)
	}

	updateCmd := exec.Command("sudo", "/usr/local/bin/oma-nbd-helper", "update-config", mainConfigPath, tempFile)
	output, err := updateCmd.CombinedOutput()
	if err != nil {
		os.Remove(tempFile) // Cleanup temp file on error
		log.WithFields(log.Fields{
			"error":  err.Error(),
			"output": string(output),
			"cmd":    updateCmd.String(),
		}).Error("Helper command failed")
		return nil, fmt.Errorf("failed to update NBD config via helper: %w", err)
	}

	log.WithField("helper_output", string(output)).Debug("Helper command succeeded")

	os.Remove(tempFile) // Cleanup temp file after successful update

	// 2. Send SIGHUP to reload config without interrupting existing connections
	// Pass config-base to helper so it can locate the correct daemon
	sighupCmd := exec.Command("sudo", "/usr/local/bin/oma-nbd-helper", "sighup-nbd", "/etc/nbd-server/config-base")
	sighupOutput, err := sighupCmd.CombinedOutput()
	if err != nil {
		// CRITICAL: Do NOT restart the server as this kills existing connections
		// Log detailed diagnostic information for troubleshooting
		log.WithFields(log.Fields{
			"error":       err.Error(),
			"output":      string(sighupOutput),
			"export_name": newConfig.ExportName,
			"device_path": newConfig.DevicePath,
			"helper_cmd":  "sighup-nbd",
			"config_file": "/etc/nbd-server/config-base",
		}).Error("⚠️  SIGHUP failed - new export may not be available until manual server restart")

		// Return error but don't kill existing connections
		return nil, fmt.Errorf("failed to reload NBD config via SIGHUP (existing connections preserved): %w", err)
	}

	log.WithFields(log.Fields{
		"output": string(sighupOutput),
	}).Info("✅ SIGHUP sent to NBD server via oma-nbd-helper for config reload")

	log.WithFields(log.Fields{
		"export_name": newConfig.ExportName,
		"port":        newConfig.Port,
		"config_path": mainConfigPath,
	}).Info("✅ Export added to NBD server via SIGHUP reload")

	// Create a server object representing the updated NBD server
	updatedServer := &Server{
		Config:  newConfig,
		Process: nil, // NBD daemon manages its own process
		PID:     0,   // Will be determined by the daemon
		Status:  "running",
	}

	// Track this configuration in our server map
	sm.servers[newConfig.Port] = updatedServer

	return updatedServer, nil
}

// isNBDServerRunning checks if an NBD server is running on the specified port
func (sm *ServerManager) isNBDServerRunning(port int) bool {
	// Check if port is listening using ss command
	cmd := exec.Command("ss", "-tln", "sport", fmt.Sprintf("= :%d", port))
	output, err := cmd.Output()
	if err != nil {
		log.WithError(err).WithField("port", port).Debug("Failed to check if NBD server is running")
		return false
	}

	isRunning := strings.Contains(string(output), fmt.Sprintf(":%d", port))
	log.WithFields(log.Fields{
		"port":       port,
		"is_running": isRunning,
	}).Debug("NBD server status checked")

	return isRunning
}

// LEGACY FUNCTION REMOVED: findOrCreateVMExportMapping
// Volume Daemon now provides real device paths directly - no need for database allocation

// verifyExportExists checks if an NBD export actually exists in the running server
func verifyExportExists(exportName string) bool {
	// Use nbd-client to list exports and check if our export is available
	cmd := exec.Command("nbd-client", "-l", "localhost", "10809")
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.WithError(err).Debug("Failed to list NBD exports for verification")
		return false
	}

	// Check if the export name appears in the output
	exportList := string(output)
	lines := strings.Split(exportList, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == exportName {
			return true
		}
	}

	log.WithFields(log.Fields{
		"export_name":       exportName,
		"available_exports": strings.Replace(exportList, "\n", ", ", -1),
	}).Debug("Export not found in NBD server")
	return false
}

// AddDynamicExportWithVolume adds an export using Volume Daemon for device correlation
func AddDynamicExportWithVolume(jobID, vmName, vmID, volumeID string, diskUnitNumber int, repo *database.VMExportMappingRepository) (*ExportInfo, bool, error) {
	const sharedNBDPort = 10808
	log.WithFields(log.Fields{
		"job_id":           jobID,
		"vm_name":          vmName,
		"vm_id":            vmID,
		"volume_id":        volumeID,
		"disk_unit_number": diskUnitNumber,
	}).Info("🔗 Adding dynamic NBD export with Volume Daemon device correlation")

	// Get REAL device path from Volume Management Daemon
	volumeClient := common.NewVolumeClient("http://localhost:8090")

	mapping, err := volumeClient.GetVolumeDevice(context.Background(), volumeID)
	if err != nil {
		log.WithError(err).WithFields(log.Fields{
			"volume_id": volumeID,
			"vm_id":     vmID,
		}).Error("Failed to get device mapping from Volume Daemon - cannot create NBD export without real device path")
		return nil, false, fmt.Errorf("failed to get device mapping from Volume Daemon for volume %s: %w", volumeID, err)
	}

	if mapping.DevicePath == "" {
		log.WithFields(log.Fields{
			"volume_id": volumeID,
			"vm_id":     vmID,
		}).Error("Volume has no device mapping in Volume Daemon - cannot create NBD export")
		return nil, false, fmt.Errorf("volume %s has no device mapping in Volume Daemon", volumeID)
	}

	// Use REAL device path from daemon
	devicePath := mapping.DevicePath

	log.WithFields(log.Fields{
		"job_id":      jobID,
		"volume_id":   volumeID,
		"device_path": devicePath,
		"vm_id":       vmID,
	}).Info("📍 Using REAL device path from Volume Management Daemon")

	// Generate export name
	exportName := fmt.Sprintf("migration-vm-%s-disk%d", vmID, diskUnitNumber)

	// Create export info
	exportInfo := &ExportInfo{
		ExportName: exportName,
		Port:       sharedNBDPort,
		DevicePath: devicePath,
	}

	// Create new mapping with daemon-verified device path
	newMapping := &database.VMExportMapping{
		VMID:           vmID,
		DiskUnitNumber: diskUnitNumber,
		ExportName:     exportName,
		DevicePath:     devicePath,
		Status:         "active",
		CreatedAt:      time.Now(),
	}

	if err := repo.Create(newMapping); err != nil {
		return nil, false, fmt.Errorf("failed to create VM export mapping: %w", err)
	}

	// Add export to the shared NBD server
	if err := addExportToSharedServer(exportName, vmName, vmID, devicePath); err != nil {
		return nil, false, fmt.Errorf("failed to add export to shared NBD server: %w", err)
	}

	log.WithFields(log.Fields{
		"job_id":      jobID,
		"export_name": exportName,
		"port":        sharedNBDPort,
		"device_path": devicePath,
		"volume_id":   volumeID,
	}).Info("✅ Dynamic export added with daemon-verified device path")

	return exportInfo, true, nil
}

// AddDynamicExportWithDevicePath creates NBD export using known device path (no Volume Daemon query needed)
func AddDynamicExportWithDevicePath(jobID, vmName, vmID, devicePath string, diskUnitNumber int, repo *database.VMExportMappingRepository) (*ExportInfo, bool, error) {
	const sharedNBDPort = 10808
	log.WithFields(log.Fields{
		"job_id":           jobID,
		"vm_name":          vmName,
		"vm_id":            vmID,
		"device_path":      devicePath,
		"disk_unit_number": diskUnitNumber,
	}).Info("🔗 Creating NBD export with known device path")

	// Generate export name
	exportName := fmt.Sprintf("migration-vm-%s-disk%d", vmID, diskUnitNumber)

	// Create export info
	exportInfo := &ExportInfo{
		JobID:      jobID,
		Port:       sharedNBDPort,
		ExportName: exportName,
		DevicePath: devicePath,
		Status:     "running",
		PID:        0,
		ConfigPath: "/etc/nbd-server/config-base",
	}

	// Add export to the shared NBD server
	if err := addExportToSharedServer(exportName, vmName, vmID, devicePath); err != nil {
		return nil, false, fmt.Errorf("failed to add export to shared NBD server: %w", err)
	}

	log.WithFields(log.Fields{
		"job_id":      jobID,
		"export_name": exportName,
		"port":        sharedNBDPort,
		"device_path": devicePath,
	}).Info("✅ NBD export created with known device path")

	return exportInfo, true, nil
}

// addExportToSharedServer adds a new export to the shared NBD server using the existing helper script
// Updated to be VM-aware for better conflict resolution
func addExportToSharedServer(exportName, vmName, vmID, devicePath string) error {
	log.WithFields(log.Fields{
		"export_name": exportName,
		"vm_name":     vmName,
		"vm_id":       vmID,
		"device_path": devicePath,
	}).Info("Adding export to shared NBD server using oma-nbd-helper")

	// Conflict resolution is now handled in the repository layer
	// The mapping table ensures devices are allocated properly and conflicts are minimal

	// Use the existing oma-nbd-helper script which already has proper permissions
	// This calls the same logic that was working before in addExportToRunningServer

	// 1. Create the export section content
	exportSection := fmt.Sprintf(`
[%s]
exportname = %s
readonly = false
multifile = false
copyonwrite = false
`, exportName, devicePath)

	// 2. Append new export section to the main NBD config file using the helper with sudo
	appendCmd := exec.Command("sudo", "/usr/local/bin/oma-nbd-helper", "append-config", "/etc/nbd-server/config-base")
	appendCmd.Stdin = strings.NewReader(exportSection)

	if output, err := appendCmd.CombinedOutput(); err != nil {
		log.WithFields(log.Fields{
			"error":  err.Error(),
			"output": string(output),
		}).Error("Failed to append export to config using helper")
		return fmt.Errorf("failed to append export to config: %w", err)
	}

	log.WithFields(log.Fields{
		"export_name": exportName,
		"device_path": devicePath,
	}).Info("✅ Export section added to config using helper")

	// PHASE 2: Check NBD server status and handle appropriately
	if !isNBDServerRunning() {
		log.Info("🚀 NBD server not running, starting it with updated config")
		startCmd := exec.Command("sudo", "/usr/local/bin/oma-nbd-helper",
			"start-server", "/etc/nbd-server/config-base")
		if startOutput, startErr := startCmd.CombinedOutput(); startErr != nil {
			log.WithFields(log.Fields{
				"error":  startErr.Error(),
				"output": string(startOutput),
			}).Error("Failed to start NBD server")
			return fmt.Errorf("failed to start NBD server: %w", startErr)
		} else {
			log.WithFields(log.Fields{
				"export_name": exportName,
				"output":      string(startOutput),
			}).Info("✅ NBD server started with new export")
		}
	} else {
		// 2. Send SIGHUP to reload config without interrupting existing connections
		log.Info("📡 NBD server running, sending SIGHUP to reload config")
		sighupCmd := exec.Command("sudo", "/usr/local/bin/oma-nbd-helper", "sighup-nbd", "/etc/nbd-server/config-base")
		sighupOutput, err := sighupCmd.CombinedOutput()
		if err != nil {
			// CRITICAL: Do NOT restart the server as this kills existing connections
			// Log detailed diagnostic information for troubleshooting
			log.WithFields(log.Fields{
				"error":       err.Error(),
				"output":      string(sighupOutput),
				"export_name": exportName,
				"device_path": devicePath,
				"helper_cmd":  "sighup-nbd",
				"config_file": "/etc/nbd-server/config-base",
			}).Error("⚠️  SIGHUP failed - export may not be available until manual server restart")

			// Check if NBD server is still running
			checkCmd := exec.Command("sudo", "ps", "aux")
			if checkOutput, checkErr := checkCmd.CombinedOutput(); checkErr == nil {
				log.WithField("ps_output", string(checkOutput)).Debug("Process list for NBD server diagnostics")
			}

			// Return error but don't kill existing connections
			return fmt.Errorf("failed to reload NBD config via SIGHUP (existing connections preserved): %w", err)
		} else {
			log.WithFields(log.Fields{
				"export_name": exportName,
				"output":      string(sighupOutput),
			}).Info("✅ NBD config reloaded via SIGHUP")
		}
	}

	log.WithField("export_name", exportName).Info("✅ Export successfully added to shared NBD server")
	return nil
}

// findExistingExports parses the NBD config to find exports using the specified device path
func findExistingExports(devicePath string) ([]string, error) {
	log.WithField("device_path", devicePath).Debug("Finding existing exports for device path")

	configPath := "/etc/nbd-server/config-base"
	file, err := os.Open(configPath)
	if err != nil {
		// If config doesn't exist, no existing exports
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to open NBD config: %w", err)
	}
	defer file.Close()

	var exports []string
	var currentSection string
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Check for section headers [section-name]
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			currentSection = strings.Trim(line, "[]")
			continue
		}

		// Check for exportname lines in current section
		if strings.HasPrefix(line, "exportname = ") && currentSection != "" && currentSection != "generic" {
			exportDevice := strings.TrimSpace(strings.TrimPrefix(line, "exportname = "))
			if exportDevice == devicePath {
				exports = append(exports, currentSection)
				log.WithFields(log.Fields{
					"export_name": currentSection,
					"device_path": exportDevice,
				}).Debug("Found existing export using device path")
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading NBD config: %w", err)
	}

	log.WithFields(log.Fields{
		"device_path":    devicePath,
		"existing_count": len(exports),
		"exports":        exports,
	}).Debug("Found existing exports for device path")

	return exports, nil
}

// isNBDServerRunning checks if the NBD server process is currently running
func isNBDServerRunning() bool {
	cmd := exec.Command("pgrep", "-f", "nbd-server -C /etc/nbd-server/config-base")
	err := cmd.Run()
	running := err == nil

	log.WithField("running", running).Debug("NBD server status check")
	return running
}
