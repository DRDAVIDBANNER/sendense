// Package nbd provides high-level NBD service management
package nbd

import (
	"fmt"

	log "github.com/sirupsen/logrus"
)

// Service provides high-level NBD management operations
type Service struct {
	configManager *ConfigManager
	serverManager *ServerManager
}

// NewService creates a new NBD service
func NewService() *Service {
	return &Service{
		configManager: NewConfigManager(),
		serverManager: NewServerManager(),
	}
}

// CreateAndStartExport creates NBD configuration and starts server for a volume
func (s *Service) CreateAndStartExport(jobID, devicePath string) (*ExportInfo, error) {
	log.WithFields(log.Fields{
		"job_id":      jobID,
		"device_path": devicePath,
	}).Info("Creating and starting NBD export")

	// Generate NBD configuration
	config, err := s.configManager.GenerateConfig(jobID, devicePath)
	if err != nil {
		return nil, fmt.Errorf("failed to generate NBD config: %w", err)
	}

	// Start NBD server
	server, err := s.serverManager.StartServer(config)
	if err != nil {
		// Cleanup config if server start fails
		s.configManager.RemoveConfig(config)
		return nil, fmt.Errorf("failed to start NBD server: %w", err)
	}

	// Create export info
	exportInfo := &ExportInfo{
		JobID:      jobID,
		Port:       config.Port,
		ExportName: fmt.Sprintf("migration-%s", jobID), // Unique export name per job for database
		DevicePath: devicePath,
		Status:     server.Status,
		PID:        server.PID,
		ConfigPath: config.ConfigPath,
	}

	log.WithFields(log.Fields{
		"job_id":      jobID,
		"port":        config.Port,
		"export_name": exportInfo.ExportName, // Use the simple export name
		"pid":         server.PID,
	}).Info("✅ NBD export created and started")

	return exportInfo, nil
}

// StopAndCleanupExport stops NBD server and removes configuration
func (s *Service) StopAndCleanupExport(port int) error {
	log.WithField("port", port).Info("Stopping and cleaning up NBD export")

	// Stop and cleanup server (includes config removal)
	if err := s.serverManager.CleanupServer(port); err != nil {
		return fmt.Errorf("failed to cleanup NBD server: %w", err)
	}

	log.WithField("port", port).Info("✅ NBD export stopped and cleaned up")
	return nil
}

// GetExportStatus returns status information for an NBD export
func (s *Service) GetExportStatus(port int) (*ExportInfo, error) {
	server, err := s.serverManager.GetServerStatus(port)
	if err != nil {
		return nil, err
	}

	exportInfo := &ExportInfo{
		JobID:      server.Config.JobID,
		Port:       server.Config.Port,
		ExportName: server.Config.ExportName,
		DevicePath: server.Config.DevicePath,
		Status:     server.Status,
		PID:        server.PID,
		ConfigPath: server.Config.ConfigPath,
	}

	return exportInfo, nil
}

// ListAllExports returns status for all managed NBD exports
func (s *Service) ListAllExports() ([]*ExportInfo, error) {
	servers := s.serverManager.ListServers()
	exports := make([]*ExportInfo, 0, len(servers))

	for _, server := range servers {
		exportInfo := &ExportInfo{
			JobID:      server.Config.JobID,
			Port:       server.Config.Port,
			ExportName: server.Config.ExportName,
			DevicePath: server.Config.DevicePath,
			Status:     server.Status,
			PID:        server.PID,
			ConfigPath: server.Config.ConfigPath,
		}
		exports = append(exports, exportInfo)
	}

	return exports, nil
}

// CheckExportHealth verifies NBD export is healthy and accessible
func (s *Service) CheckExportHealth(port int) error {
	// Check server status
	server, err := s.serverManager.GetServerStatus(port)
	if err != nil {
		return fmt.Errorf("server status check failed: %w", err)
	}

	if server.Status != "running" {
		return fmt.Errorf("NBD server not running (status: %s)", server.Status)
	}

	// Check port health
	if err := s.serverManager.CheckPortHealth(port); err != nil {
		return fmt.Errorf("port health check failed: %w", err)
	}

	log.WithField("port", port).Debug("✅ NBD export health check passed")
	return nil
}

// RestartExport restarts an NBD export
func (s *Service) RestartExport(port int) error {
	log.WithField("port", port).Info("Restarting NBD export")

	if err := s.serverManager.RestartServer(port); err != nil {
		return fmt.Errorf("failed to restart NBD server: %w", err)
	}

	log.WithField("port", port).Info("✅ NBD export restarted")
	return nil
}

// ExportInfo represents NBD export information
type ExportInfo struct {
	JobID      string `json:"job_id"`      // Associated migration job
	Port       int    `json:"port"`        // NBD server port
	ExportName string `json:"export_name"` // NBD export name
	DevicePath string `json:"device_path"` // Block device path
	Status     string `json:"status"`      // running, stopped, error
	PID        int    `json:"pid"`         // Process ID (0 if not running)
	ConfigPath string `json:"config_path"` // Configuration file path
}

