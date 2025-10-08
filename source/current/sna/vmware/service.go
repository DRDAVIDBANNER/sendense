// Package vmware provides reusable VMware operations for both SNA client and API server
package vmware

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/vexxhost/migratekit-sha/models"
	"github.com/vexxhost/migratekit/source/current/sna/api"
	"github.com/vexxhost/migratekit/source/current/sna/cbt"
	"github.com/vexxhost/migratekit/source/current/sna/client"
)

// ServiceConfig holds configuration for the VMware service
type ServiceConfig struct {
	AutoCBTEnabled bool
}

// Service provides high-level VMware operations
type Service struct {
	// SHA client for API communication
	shaClient *client.Client
	// Configuration options
	config ServiceConfig
}

// NewService creates a new VMware service
func NewService(shaClient *client.Client) *Service {
	return &Service{
		shaClient: shaClient,
		config: ServiceConfig{
			AutoCBTEnabled: true, // Default to enabled
		},
	}
}

// NewServiceWithConfig creates a new VMware service with custom configuration
func NewServiceWithConfig(shaClient *client.Client, config ServiceConfig) *Service {
	return &Service{
		shaClient: shaClient,
		config:    config,
	}
}

// DiscoverVMsFromVCenter discovers VMs from a specific vCenter
func (s *Service) DiscoverVMsFromVCenter(ctx context.Context, vcenter, username, password, datacenter, filter string) (*models.VMInventoryRequest, error) {
	log.WithFields(log.Fields{
		"vcenter":    vcenter,
		"datacenter": datacenter,
		"filter":     filter,
	}).Info("Starting VM discovery from vCenter")

	// Create discovery client with provided credentials
	config := Config{
		Host:       vcenter,
		Username:   username,
		Password:   password,
		Datacenter: datacenter,
		Insecure:   true, // Use insecure for now (production should use proper certs)
	}

	discovery := NewDiscovery(config)

	// Connect to vCenter
	if err := discovery.Connect(ctx); err != nil {
		return nil, fmt.Errorf("failed to connect to vCenter: %w", err)
	}
	defer discovery.Disconnect()

	// Discover VMs
	inventory, err := discovery.DiscoverVMs(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to discover VMs: %w", err)
	}

	log.WithFields(log.Fields{
		"vm_count":     len(inventory.VMs),
		"vcenter_host": inventory.VCenter.Host,
		"datacenter":   inventory.VCenter.Datacenter,
	}).Info("VM discovery completed")

	// Apply filter if specified
	if filter != "" {
		inventory.VMs = s.filterVMs(inventory.VMs, filter)
		log.WithField("filtered_count", len(inventory.VMs)).Info("Applied VM filter")
	}

	return inventory, nil
}

// StartReplicationJob starts a replication job for specific VMs with NBD targets
func (s *Service) StartReplicationJob(ctx context.Context, jobID, vcenter, username, password string, vmPaths []string, nbdTargets []api.NBDTarget) error {
	log.WithFields(log.Fields{
		"job_id":   jobID,
		"vcenter":  vcenter,
		"vm_count": len(vmPaths),
	}).Info("Starting replication job")

	// Create discovery client to validate VMs exist
	config := Config{
		Host:       vcenter,
		Username:   username,
		Password:   password,
		Datacenter: "", // Will be derived from VM paths
		Insecure:   true,
	}

	discovery := NewDiscovery(config)

	// Connect to vCenter
	if err := discovery.Connect(ctx); err != nil {
		return fmt.Errorf("failed to connect to vCenter: %w", err)
	}
	defer discovery.Disconnect()

	// Debug: Log NBD targets received
	log.WithFields(log.Fields{
		"job_id":            jobID,
		"vm_count":          len(vmPaths),
		"nbd_targets_count": len(nbdTargets),
	}).Info("Processing replication job with NBD targets")

	// 🎯 CRITICAL FIX: Handle multi-disk VMs properly
	// Check if this is a multi-disk VM (single VM with multiple NBD targets)
	if len(vmPaths) == 1 && len(nbdTargets) > 1 {
		vmPath := vmPaths[0]
		log.WithFields(log.Fields{
			"job_id":            jobID,
			"vm_path":           vmPath,
			"nbd_targets_count": len(nbdTargets),
		}).Info("🔧 MULTI-DISK VM: Starting migratekit with multiple NBD targets")

		// For multi-disk VMs, start migratekit with ALL NBD targets
		if err := s.startMultiDiskReplication(jobID, vmPath, nbdTargets, vcenter, username, password, discovery); err != nil {
			log.WithError(err).WithField("vm_path", vmPath).Error("Failed to start multi-disk replication")
			return err
		}

		log.WithFields(log.Fields{
			"job_id":            jobID,
			"vm_path":           vmPath,
			"nbd_targets_count": len(nbdTargets),
		}).Info("✅ Multi-disk replication started successfully")
	} else {
		// Legacy single-disk logic: one NBD target per VM
		for i, vmPath := range vmPaths {
			log.WithFields(log.Fields{
				"job_id":  jobID,
				"vm_path": vmPath,
			}).Info("Starting single-disk replication for VM")

			// Find corresponding NBD target for this VM
			var targetDevice string
			if i < len(nbdTargets) {
				targetDevice = nbdTargets[i].DevicePath
			} else {
				log.WithField("vm_path", vmPath).Error("No NBD target provided for VM")
				continue
			}

			// Start migratekit directly with the NBD target device
			if err := s.startDirectReplication(jobID, vmPath, targetDevice, vcenter, username, password, discovery); err != nil {
				log.WithError(err).WithField("vm_path", vmPath).Error("Failed to start direct replication")
				continue
			}

			log.WithFields(log.Fields{
				"job_id":        jobID,
				"vm_path":       vmPath,
				"target_device": targetDevice,
			}).Info("Single-disk replication started successfully")
		}
	}

	return nil
}

// startMultiDiskReplication starts migratekit with multiple NBD targets for multi-disk VMs
func (s *Service) startMultiDiskReplication(jobID, vmPath string, nbdTargets []api.NBDTarget, vcenter, username, password string, discovery *Discovery) error {
	log.WithFields(log.Fields{
		"job_id":            jobID,
		"vm_path":           vmPath,
		"nbd_targets_count": len(nbdTargets),
		"vcenter":           vcenter,
	}).Info("🚀 Starting multi-disk migratekit with all NBD targets")

	// 🎯 CRITICAL: Start migratekit with ALL NBD targets as parameters
	// Build target list for migratekit --nbd-targets parameter
	var targetPairs []string
	for _, target := range nbdTargets {
		// 🎯 CRITICAL FIX: Use VMware disk key for direct correlation
		var targetID string
		if target.VMwareDiskKey != "" {
			// Use VMware disk key directly (e.g., "2000", "2001")
			targetID = target.VMwareDiskKey
		} else {
			// Fallback to vm_disk_id for backward compatibility
			targetID = fmt.Sprintf("%d", target.VMDiskID)
		}

		// Format: vmware_disk_key:nbd_target_url (e.g., "2000:nbd://...")
		targetPair := fmt.Sprintf("%s:%s", targetID, target.DevicePath)
		targetPairs = append(targetPairs, targetPair)

		log.WithFields(log.Fields{
			"job_id":          jobID,
			"vmware_disk_key": target.VMwareDiskKey,
			"vm_disk_id":      target.VMDiskID,
			"device_path":     target.DevicePath,
			"target_pair":     targetPair,
		}).Info("🎯 Added NBD target with VMware disk key for multi-disk migratekit")
	}

	// Join all targets with commas for --nbd-targets parameter
	ndbTargetsParam := strings.Join(targetPairs, ",")

	// Create migratekit command for multi-disk VM
	logFile := fmt.Sprintf("/tmp/migratekit-job-%s.log", jobID)
	cmd := exec.Command("/opt/vma/bin/migratekit",
		"migrate",
		"--vmware-endpoint", vcenter,
		"--vmware-username", username,
		"--vmware-password", password,
		"--vmware-path", vmPath,
		"--job-id", jobID,
		"--nbd-targets", ndbTargetsParam, // Pass all NBD targets for multi-disk support
		"--debug",
	)

	// Standard environment variables
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("CLOUDSTACK_API_URL=%s", "http://localhost:8082"),
		fmt.Sprintf("CLOUDSTACK_API_KEY=%s", "test-api-key"),
		fmt.Sprintf("CLOUDSTACK_SECRET_KEY=%s", "test-secret-key"),
		fmt.Sprintf("MIGRATEKIT_JOB_ID=%s", jobID), // 🎯 CRITICAL: Pass job ID for change ID storage
	)

	// Redirect output to log file
	logFileHandle, err := os.Create(logFile)
	if err != nil {
		return fmt.Errorf("failed to create log file %s: %w", logFile, err)
	}
	defer logFileHandle.Close()

	cmd.Stdout = logFileHandle
	cmd.Stderr = logFileHandle

	log.WithFields(log.Fields{
		"job_id":            jobID,
		"vm_path":           vmPath,
		"nbd_targets_param": ndbTargetsParam,
		"command":           cmd.String(),
		"log_file":          logFile,
	}).Info("🚀 Executing multi-disk migratekit command with all targets")

	// Start migratekit process
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start multi-disk migratekit: %w", err)
	}

	log.WithFields(log.Fields{
		"job_id":  jobID,
		"vm_path": vmPath,
		"pid":     cmd.Process.Pid,
	}).Info("✅ Multi-disk migratekit started successfully with all NBD targets")

	return nil
}

// startDirectReplication starts migratekit with dynamic stunnel for NBD connection
func (s *Service) startDirectReplication(jobID, vmPath, targetDevice, vcenter, username, password string, discovery *Discovery) error {
	log.WithFields(log.Fields{
		"job_id":        jobID,
		"vm_path":       vmPath,
		"target_device": targetDevice,
		"vcenter":       vcenter,
	}).Info("Starting migratekit with automatic NBD discovery")

	// NOTE: Using single stunnel architecture - SHA port 443 forwards to 10809
	// migratekit connects to localhost:10808, which goes through existing SNA->SHA tunnel
	// Extract export name from NBD URL for job-specific export targeting

	// 3. Extract export name from NBD target URL
	exportName, err := s.parseNBDExportName(targetDevice)
	if err != nil {
		return fmt.Errorf("failed to parse export name from NBD URL %s: %w", targetDevice, err)
	}

	log.WithFields(log.Fields{
		"job_id":      jobID,
		"target_url":  targetDevice,
		"export_name": exportName,
	}).Info("Extracted export name from NBD target URL")

	// 3.5. Verify NBD export is available before starting migration
	if err := s.verifyNBDExport(exportName); err != nil {
		return fmt.Errorf("NBD export verification failed for %s: %w", exportName, err)
	}

	log.WithField("export_name", exportName).Info("✅ NBD export verified and accessible")

	// 3.7. CRITICAL: Ensure CBT is enabled before migration - CBT disabled = migration failure
	if s.config.AutoCBTEnabled {
		// Reuse the existing authenticated discovery client to avoid session issues
		cbtManager := cbt.NewManager(vcenter, username, password)
		if err := cbtManager.EnsureCBTEnabledWithClient(context.Background(), vmPath, discovery.GetClient()); err != nil {
			return fmt.Errorf("CBT enablement check failed - migration cannot proceed: %w", err)
		}
		log.WithField("vm_path", vmPath).Info("✅ CBT enabled and verified - migration can proceed")
	} else {
		log.WithField("vm_path", vmPath).Warn("⚠️ CBT auto-enablement disabled - proceeding without CBT check")
	}

	// 4. Previous change ID will be retrieved by migratekit directly from SHA API
	// No need to create temp files - migratekit now uses database integration via API calls
	log.WithField("vm_path", vmPath).Info("🔄 migratekit will retrieve previous ChangeID from SHA database via API")

	// 5. Build migratekit command with job-specific export name
	// migratekit connects to localhost:10808 (via stunnel) and uses the specific export name
	cmd := exec.Command("/opt/vma/bin/migratekit", "migrate",
		"--vmware-endpoint", vcenter,
		"--vmware-username", username,
		"--vmware-password", password,
		"--vmware-path", vmPath,
		"--nbd-export-name", exportName, // Pass job-specific export name
		"--job-id", jobID, // 🎯 CRITICAL FIX: Pass job ID for progress tracking
		"--debug",
	)

	cmd.Dir = "/opt/vma"

	// Set required environment variables (do not rely on NBD_LOCAL_PORT)
	cmd.Env = append(os.Environ(),
		"CLOUDSTACK_API_URL=http://localhost:8082", // SHA API via tunnel
		"CLOUDSTACK_API_KEY=test-api-key",          // Placeholder for now
		"CLOUDSTACK_SECRET_KEY=test-secret-key",    // Placeholder for now
		fmt.Sprintf("MIGRATEKIT_JOB_ID=%s", jobID), // Pass job ID for ChangeID storage
	)

	// Create log file for migratekit output
	logFile := fmt.Sprintf("/tmp/migratekit-%s.log", jobID)
	logFileHandle, err := os.Create(logFile)
	if err != nil {
		log.WithError(err).Warn("Failed to create migratekit log file, using default logging")
	} else {
		cmd.Stdout = logFileHandle
		cmd.Stderr = logFileHandle
	}

	log.WithFields(log.Fields{
		"job_id":        jobID,
		"vm_path":       vmPath,
		"target_device": targetDevice,
		"command":       cmd.String(),
		"log_file":      logFile,
		"env_vars":      []string{"CLOUDSTACK_API_URL=http://localhost:8082", "CLOUDSTACK_API_KEY=test-api-key", "CLOUDSTACK_SECRET_KEY=test-secret-key"},
	}).Info("Executing migratekit command with output capture")

	// Start migratekit in background
	if err := cmd.Start(); err != nil {
		if logFileHandle != nil {
			logFileHandle.Close()
		}
		return fmt.Errorf("failed to start migratekit: %w", err)
	}

	log.WithFields(log.Fields{
		"job_id": jobID,
		"pid":    cmd.Process.Pid,
	}).Info("Migratekit started successfully with NBD target")

	// Start monitoring in background
	go s.monitorDirectMigration(jobID, cmd)

	return nil
}

// monitorDirectMigration monitors direct migratekit process
func (s *Service) monitorDirectMigration(jobID string, cmd *exec.Cmd) {
	// Wait for process to complete
	err := cmd.Wait()

	// Read and log migratekit output
	logFile := fmt.Sprintf("/tmp/migratekit-%s.log", jobID)
	var migrationOutput string
	if output, readErr := os.ReadFile(logFile); readErr == nil {
		migrationOutput = string(output)
		log.WithFields(log.Fields{
			"job_id": jobID,
			"error":  err,
			"output": migrationOutput,
		}).Info("Migratekit process completed with output")
	} else {
		log.WithFields(log.Fields{
			"job_id": jobID,
			"error":  err,
		}).Info("Migratekit process completed (no output captured)")
	}

	// Determine migration success/failure
	success := (err == nil)
	var finalStatus string
	if success {
		finalStatus = "completed"
	} else {
		finalStatus = "failed"
	}

	// ChangeID is now stored directly by migratekit via SHA API - no extraction needed
	// The migration completion notification will use the actual ChangeID from the database
	log.WithField("job_id", jobID).Info("📊 Migration completed - ChangeID stored by migratekit via SHA API")

	// Notify SHA of migration completion (ChangeID is stored separately by migratekit)
	s.notifyMigrationCompletion(jobID, finalStatus, migrationOutput)
}

// filterVMs filters VMs by name pattern and validates path consistency
func (s *Service) filterVMs(vms []models.VMInfo, filter string) []models.VMInfo {
	if filter == "" {
		return vms
	}

	filters := strings.Split(filter, ",")
	var filtered []models.VMInfo

	for _, vm := range vms {
		for _, f := range filters {
			f = strings.TrimSpace(f)
			if strings.Contains(strings.ToLower(vm.Name), strings.ToLower(f)) {
				// Check for path/name consistency issues
				pathParts := strings.Split(vm.Path, "/")
				vmToAdd := vm

				if len(pathParts) > 0 {
					lastPathComponent := pathParts[len(pathParts)-1]
					if lastPathComponent != vm.Name {
						log.WithFields(log.Fields{
							"vm_name":        vm.Name,
							"vm_path":        vm.Path,
							"path_component": lastPathComponent,
							"filter":         f,
						}).Warn("VM name/path mismatch in filtered results - potential vCenter data inconsistency")

						// For exact name matches with path mismatches, try to construct correct path
						if strings.EqualFold(vm.Name, f) {
							// Create a corrected path by replacing the last component with the VM name
							correctedPathParts := pathParts[:len(pathParts)-1]
							correctedPath := strings.Join(correctedPathParts, "/") + "/" + vm.Name

							log.WithFields(log.Fields{
								"original_path":  vm.Path,
								"corrected_path": correctedPath,
								"vm_name":        vm.Name,
							}).Info("Attempting path correction for exact name match")

							// Use corrected VM info
							vmToAdd.Path = correctedPath
						}
					}
				}

				filtered = append(filtered, vmToAdd)
				break
			}
		}
	}

	return filtered
}

// extractVMNameFromPath extracts VM name from a VM path
func (s *Service) extractVMNameFromPath(vmPath string) string {
	parts := strings.Split(vmPath, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return "unknown"
}

// parseNBDExportName extracts the export name from NBD URL
// Supports both formats: nbd://host:port/export and nbd://:port/export (hostless)
func (s *Service) parseNBDExportName(nbdURL string) (string, error) {
	// Parse nbd://[host]:port/export format
	if !strings.HasPrefix(nbdURL, "nbd://") {
		return "", fmt.Errorf("invalid NBD URL format: %s", nbdURL)
	}

	// Remove nbd:// prefix
	urlPart := strings.TrimPrefix(nbdURL, "nbd://")

	// Split by / to separate host:port from export name
	parts := strings.Split(urlPart, "/")
	if len(parts) < 2 {
		return "", fmt.Errorf("invalid NBD URL format, missing export name: %s", nbdURL)
	}

	// Return the export name (everything after the last /)
	exportName := parts[len(parts)-1]
	if exportName == "" {
		return "", fmt.Errorf("empty export name in NBD URL: %s", nbdURL)
	}

	return exportName, nil
}

// parseNBDPort extracts the port number from NBD URL (kept for compatibility)
// Supports both formats: nbd://host:port/export and nbd://:port/export (hostless)
func (s *Service) parseNBDPort(nbdURL string) (int, error) {
	// Parse nbd://[host]:port/export format
	if !strings.HasPrefix(nbdURL, "nbd://") {
		return 0, fmt.Errorf("invalid NBD URL format: %s", nbdURL)
	}

	// Remove nbd:// prefix
	urlPart := strings.TrimPrefix(nbdURL, "nbd://")

	// Split by / to separate host:port from export name
	parts := strings.Split(urlPart, "/")
	if len(parts) < 2 {
		return 0, fmt.Errorf("invalid NBD URL format, missing export name: %s", nbdURL)
	}

	// Extract port from host:port or :port (hostless)
	hostPort := parts[0]

	// Handle hostless format (e.g., ":10858")
	if strings.HasPrefix(hostPort, ":") {
		portStr := strings.TrimPrefix(hostPort, ":")
		port, err := strconv.Atoi(portStr)
		if err != nil {
			return 0, fmt.Errorf("invalid port in hostless NBD URL: %s", portStr)
		}
		return port, nil
	}

	// Handle standard format (e.g., "host:10858")
	hostPortParts := strings.Split(hostPort, ":")
	if len(hostPortParts) != 2 {
		return 0, fmt.Errorf("invalid host:port format in NBD URL: %s", hostPort)
	}

	port, err := strconv.Atoi(hostPortParts[1])
	if err != nil {
		return 0, fmt.Errorf("invalid port in NBD URL: %s", hostPortParts[1])
	}

	return port, nil
}

// NOTE: allocateLocalPort removed - using single stunnel architecture
// migratekit connects to localhost:10808 via existing tunnel to SHA:10809

// NOTE: generateJobStunnelConfigWithPort removed - using single stunnel architecture

// NOTE: generateJobStunnelConfig removed - using single stunnel architecture

// isLocalPortAvailable returns true if the local TCP port can be bound (i.e., not in use)
func (s *Service) isLocalPortAvailable(port int) bool {
	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return false
	}
	_ = listener.Close()
	return true
}

// NOTE: startJobStunnel removed - using single stunnel architecture

// notifyMigrationCompletion notifies SHA that migration has completed
func (s *Service) notifyMigrationCompletion(jobID, status, output string) {
	log.WithFields(log.Fields{
		"job_id": jobID,
		"status": status,
	}).Info("Notifying SHA of migration completion")

	// Call SHA API to update migration status
	if s.shaClient != nil {
		// Create job update with completion status (ChangeID is handled separately by migratekit)
		jobUpdate := &models.ReplicationJob{
			ID:        jobID,
			Status:    status,
			Progress:  100.0,
			UpdatedAt: time.Now(),
		}

		// Use SHA client to update job
		if err := s.shaClient.UpdateReplicationJob(jobUpdate); err != nil {
			log.WithError(err).WithField("job_id", jobID).Error("Failed to notify SHA of migration completion via client")
		} else {
			log.WithField("job_id", jobID).Info("✅ Successfully notified SHA of migration completion")
		}
	} else {
		log.WithField("job_id", jobID).Warn("No SHA client available, migration completion not reported")
	}
}

// REMOVED: getPreviousChangeIDFromOMA method - no longer needed
// migratekit now retrieves previous ChangeIDs directly from SHA API

// REMOVED: createChangeIDFile method - no longer needed
// ChangeIDs are now handled via SHA API calls from migratekit directly

// REMOVED: extractChangeIDFromOutput method - no longer needed
// ChangeIDs are now stored directly by migratekit via SHA API calls

// verifyNBDExport verifies that the specified NBD export is available and accessible
// Since listing is disabled (allowlist=false), we test direct connection to the export
func (s *Service) verifyNBDExport(exportName string) error {
	log.WithField("export_name", exportName).Info("Verifying NBD export availability")

	// Create context with timeout for verification
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Retry logic in case export is still being set up
	maxRetries := 3

	for attempt := 1; attempt <= maxRetries; attempt++ {
		log.WithFields(log.Fields{
			"export_name": exportName,
			"attempt":     attempt,
			"max_retries": maxRetries,
		}).Info("Attempting NBD export verification")

		// Test connectivity by attempting to connect to the specific export
		// Use a quick timeout and expect it to succeed in negotiation phase
		testCmd := exec.CommandContext(ctx, "timeout", "10", "nbd-client",
			"localhost", "10808", "/dev/null", "-N", exportName, "-b", "512", "-persist")

		output, err := testCmd.CombinedOutput()

		if err != nil {
			outputStr := string(output)

			// Check for specific error patterns indicating export doesn't exist
			if strings.Contains(outputStr, "Export unknown") ||
				strings.Contains(outputStr, "export not found") ||
				strings.Contains(outputStr, "unknown export") ||
				strings.Contains(outputStr, "No such file or directory") {
				log.WithFields(log.Fields{
					"export_name": exportName,
					"attempt":     attempt,
					"error":       err.Error(),
					"output":      outputStr,
				}).Warn("Export not found, retrying...")

				if attempt < maxRetries {
					time.Sleep(2 * time.Second) // Wait before retry
					continue
				}
				return fmt.Errorf("export '%s' not found after %d attempts: %w", exportName, maxRetries, err)
			}

			// For other errors (like "Invalid nbd device target"), that's actually success
			// because it means the NBD negotiation succeeded but /dev/null setup failed
			if strings.Contains(outputStr, "Invalid nbd device target") ||
				strings.Contains(outputStr, "size =") { // Negotiation shows export size
				log.WithFields(log.Fields{
					"export_name": exportName,
					"output":      outputStr,
				}).Debug("NBD connection negotiation successful (expected /dev/null error)")
				break
			}

			// Unknown error - log and retry
			log.WithFields(log.Fields{
				"export_name": exportName,
				"attempt":     attempt,
				"error":       err.Error(),
				"output":      outputStr,
			}).Warn("Unexpected NBD connection error, retrying...")

			if attempt < maxRetries {
				time.Sleep(2 * time.Second)
				continue
			}
			return fmt.Errorf("NBD export verification failed after %d attempts: %w", maxRetries, err)
		} else {
			// Unexpected success - shouldn't happen with /dev/null
			log.WithField("export_name", exportName).Debug("NBD connection completed successfully")
			break
		}
	}

	log.WithField("export_name", exportName).Info("✅ NBD export verification successful")
	return nil
}
