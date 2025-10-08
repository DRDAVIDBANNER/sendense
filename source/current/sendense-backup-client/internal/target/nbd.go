package target

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/vexxhost/migratekit/internal/vmware"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/types"
	"libguestfs.org/libnbd"
)

type NBDTarget struct {
	VirtualMachine *object.VirtualMachine
	Disk           *types.VirtualDisk
	nbdHandle      *libnbd.Libnbd
	nbdHost        string
	nbdPort        string
	nbdExportName  string
	SSHTarget      string
}

type NBDVolumeCreateOpts struct {
	AvailabilityZone string
	VolumeType       string
	BusType          string
}

func NewNBDTarget(ctx context.Context, vm *object.VirtualMachine, disk *types.VirtualDisk) (*NBDTarget, error) {
	return &NBDTarget{
		VirtualMachine: vm,
		Disk:           disk,
	}, nil
}

func (t *NBDTarget) GetDisk() *types.VirtualDisk {
	return t.Disk
}

func (t *NBDTarget) Connect(ctx context.Context) error {
	log.Println("🔥 NBD Target Connect() called - establishing NBD connection")

	// Check for local test mode
	if localPath := os.Getenv("LOCAL_TEST_DEVICE"); localPath != "" {
		log.Printf("🏠 LOCAL TEST MODE: NBD not supported for local devices, fallback to file: %s", localPath)
		return fmt.Errorf("LOCAL_TEST_DEVICE not supported with NBD mode - use SSH streaming mode")
	}

	// Get NBD connection parameters from context (set via command-line flags)
	t.nbdHost = "127.0.0.1" // Default
	t.nbdPort = "10808"     // Default
	
	// Override with context values if provided
	if host := ctx.Value("nbdHost"); host != nil && host.(string) != "" {
		t.nbdHost = host.(string)
	}
	if port := ctx.Value("nbdPort"); port != nil && port.(int) != 0 {
		t.nbdPort = strconv.Itoa(port.(int))
	}
	
	log.Printf("🎯 Using NBD connection parameters: host=%s port=%s", t.nbdHost, t.nbdPort)

	// Create NBD handle
	handle, err := libnbd.Create()
	if err != nil {
		return fmt.Errorf("failed to create NBD handle: %v", err)
	}

	// Enable debug logging to see TLS handshake details
	err = handle.SetDebug(true)
	if err != nil {
		log.Printf("⚠️  Failed to enable debug logging: %v", err)
	} else {
		log.Printf("🔍 NBD debug logging enabled for TLS troubleshooting")
	}

	// 🎯 MULTI-DISK FIX: Determine correct NBD export for this disk
	exportName, err := t.determineNBDExportForDisk(ctx)
	if err != nil {
		handle.Close()
		return fmt.Errorf("failed to determine NBD export for disk: %v", err)
	}

	t.nbdExportName = exportName // Store for use in GetPath()
	err = handle.SetExportName(exportName)
	if err != nil {
		handle.Close()
		return fmt.Errorf("failed to set export name: %v", err)
	}

	log.Printf("🎯 Using NBD export for this disk: %s", exportName)

	// Use plain NBD - TLS handled by stunnel client tunnel
	log.Printf("🔐 Using plain NBD via stunnel TLS tunnel")
	err = handle.SetTls(libnbd.TLS_DISABLE)
	if err != nil {
		handle.Close()
		return fmt.Errorf("failed to disable TLS: %v", err)
	}
	log.Printf("✅ Plain NBD configured (TLS via stunnel tunnel)")

	// 🚀 PHASE 2: Enable structured replies for better performance and error handling
	err = handle.SetRequestStructuredReplies(true)
	if err != nil {
		log.Printf("⚠️ Structured replies not supported by server, using legacy mode: %v", err)
	} else {
		log.Printf("✅ NBD structured replies enabled for better performance")
	}

	// Connect to NBD server
	log.Printf("🔐 Step 4: Connecting to NBD server at %s:%s", t.nbdHost, t.nbdPort)
	log.Printf("🚀 Flow: libnbd → %s:%s (NBD server)", t.nbdHost, t.nbdPort)

	err = handle.ConnectTcp(t.nbdHost, t.nbdPort)
	if err != nil {
		handle.Close()
		log.Printf("❌ ConnectTcp failed: %v", err)
		return fmt.Errorf("failed to connect to NBD server: %v", err)
	}
	log.Printf("🎉 NBD connection established!")

	t.nbdHandle = handle
	log.Printf("✅ NBD connection ready: %s:%s/%s", t.nbdHost, t.nbdPort, t.nbdExportName)
	return nil
}

func (t *NBDTarget) GetPath(ctx context.Context) (string, error) {
	if t.nbdHandle == nil {
		return "", fmt.Errorf("NBD target not connected - call Connect() first")
	}

	// Return NBD URL with dynamic export name
	nbdPath := fmt.Sprintf("nbd://%s:%s/%s", t.nbdHost, t.nbdPort, t.nbdExportName)
	log.Printf("🚀 GetPath() returning NBD URL: %s", nbdPath)
	return nbdPath, nil
}

// GetNBDHandle returns the NBD handle for positioned writes
func (t *NBDTarget) GetNBDHandle() *libnbd.Libnbd {
	return t.nbdHandle
}

func (t *NBDTarget) Disconnect(ctx context.Context) error {
	log.Println("🧹 NBD Target Disconnect() - Cleaning up NBD connection")

	// Close NBD handle
	if t.nbdHandle != nil {
		err := t.nbdHandle.Close()
		if err != nil {
			log.Printf("⚠️ Warning: Failed to close NBD handle: %v", err)
		}
		t.nbdHandle = nil
		log.Printf("🔌 Closed NBD connection to %s:%s", t.nbdHost, t.nbdPort)
	}

	log.Println("✅ NBD connection cleanup completed")
	return nil
}

func (t *NBDTarget) Exists(ctx context.Context) (bool, error) {
	// Check if we have a stored ChangeID in SHA database via API
	vmPath := t.VirtualMachine.InventoryPath

	changeID, err := t.getChangeIDFromOMA(vmPath)
	if err != nil {
		log.Printf("⚠️ Failed to check ChangeID from SHA API: %v", err)
		return false, nil // Assume target doesn't exist on API error
	}

	if changeID != "" {
		log.Println("📋 Found existing ChangeID in database - target exists, can try incremental")
		return true, nil // ChangeID exists, can try incremental
	} else {
		log.Println("📋 No ChangeID found in database - target doesn't exist, full copy needed")
		return false, nil // No ChangeID, need full copy
	}
}

func (t *NBDTarget) GetCurrentChangeID(ctx context.Context) (*vmware.ChangeID, error) {
	// Get ChangeID from SHA database via API
	vmPath := t.VirtualMachine.InventoryPath

	changeIDStr, err := t.getChangeIDFromOMA(vmPath)
	if err != nil {
		log.Printf("⚠️ Failed to read ChangeID from SHA API: %v", err)
		return &vmware.ChangeID{}, nil // Return empty for first sync
	}

	if changeIDStr == "" {
		log.Println("📋 No previous ChangeID found in database - will perform full sync")
		return &vmware.ChangeID{}, nil
	}

	log.Printf("📋 Found previous ChangeID in database: %s", changeIDStr)
	return vmware.ParseChangeID(changeIDStr)
}

func (t *NBDTarget) WriteChangeID(ctx context.Context, changeId *vmware.ChangeID) error {
	if changeId == nil || changeId.Value == "" {
		log.Println("📋 Skipping empty ChangeID write")
		return nil
	}

	// Get job ID from environment variable (set by SNA service)
	jobID := os.Getenv("MIGRATEKIT_JOB_ID")
	if jobID == "" {
		log.Println("⚠️ No MIGRATEKIT_JOB_ID environment variable set, cannot store ChangeID")
		return nil // Don't fail the migration for this
	}

	// Store ChangeID in SHA database via API
	err := t.storeChangeIDInOMA(jobID, changeId.Value)
	if err != nil {
		return fmt.Errorf("failed to write ChangeID to SHA database: %w", err)
	}

	log.Printf("📋 Stored ChangeID in database: %s", changeId.Value)
	return nil
}

// getChangeIDFilePath returns the path where ChangeID is stored
// DEPRECATED: This method is no longer used - ChangeIDs are now stored in SHA database
func (t *NBDTarget) getChangeIDFilePath() string {
	// Create a unique file path based on VM name and disk key
	vmName := t.VirtualMachine.Name()
	diskKey := strconv.Itoa(int(t.Disk.Key))
	return fmt.Sprintf("/tmp/migratekit_changeid_%s_disk_%s", vmName, diskKey)
}

func (t *NBDTarget) CreateImageFromVolume(ctx context.Context) error {
	log.Println("🚧 CreateImageFromVolume() - stub implementation")
	return nil
}

// NBDDiskLabel creates a label for the disk
func NBDDiskLabel(vm *object.VirtualMachine, disk *types.VirtualDisk) string {
	return vm.Name() + "-disk-" + string(rune(disk.Key))
}

// getCurrentDiskID calculates the disk ID for the current disk
func (t *NBDTarget) getCurrentDiskID() string {
	if t.Disk == nil || t.Disk.Key == 0 {
		log.Printf("⚠️ No disk key available, falling back to default disk-2000")
		return "disk-2000" // Backward compatibility fallback
	}
	diskID := fmt.Sprintf("disk-%d", t.Disk.Key)
	log.Printf("🎯 Calculated disk ID for change ID storage: %s (VMware disk.Key: %d)", diskID, t.Disk.Key)
	return diskID
}

// getDiskIndexFromJobID extracts the numeric disk index from the backup job ID or NBD export name
// For multi-disk backups, extracts from: "backup-pgtest1-disk0-..." or "pgtest1-disk0"
// This extracts the "0" from "disk0"
func (t *NBDTarget) getDiskIndexFromJobID(jobID string) int {
	// First try: Extract from job ID pattern (backup-...-disk0-...)
	re := regexp.MustCompile(`-disk(\d+)-`)
	matches := re.FindStringSubmatch(jobID)
	
	if len(matches) > 1 {
		diskIndex, err := strconv.Atoi(matches[1])
		if err == nil {
			log.Printf("🎯 Extracted disk index %d from job ID: %s", diskIndex, jobID)
			return diskIndex
		}
	}
	
	// Second try: Extract from NBD export name (pgtest1-disk0)
	if t.nbdExportName != "" {
		re2 := regexp.MustCompile(`disk(\d+)$`)
		matches2 := re2.FindStringSubmatch(t.nbdExportName)
		if len(matches2) > 1 {
			diskIndex, err := strconv.Atoi(matches2[1])
			if err == nil {
				log.Printf("🎯 Extracted disk index %d from NBD export name: %s", diskIndex, t.nbdExportName)
				return diskIndex
			}
		}
	}
	
	// Fallback: default to disk 0 for single-disk VMs
	log.Printf("⚠️ Could not extract disk index from job ID %s or export %s, defaulting to 0", jobID, t.nbdExportName)
	return 0
}

// determineNBDExportForDisk determines the correct NBD export for the current disk
func (t *NBDTarget) determineNBDExportForDisk(ctx context.Context) (string, error) {
	// Check if multi-disk targets are provided
	nbdTargetsStr := ctx.Value("nbdTargets")
	if nbdTargetsStr != nil && nbdTargetsStr.(string) != "" {
		// Parse multi-disk NBD targets: "vm_disk_id:nbd_url,vm_disk_id:nbd_url"
		return t.parseMultiDiskNBDTargets(ctx, nbdTargetsStr.(string))
	}

	// Fallback to single-disk mode
	exportName := ctx.Value("nbdExportName").(string)
	if exportName == "" {
		exportName = "migration"
	}

	log.Printf("🔄 Using single-disk NBD export: %s", exportName)
	return exportName, nil
}

// parseMultiDiskNBDTargets parses NBD targets and returns the correct export for this disk
func (t *NBDTarget) parseMultiDiskNBDTargets(ctx context.Context, nbdTargetsStr string) (string, error) {
	if t.Disk == nil {
		return "", fmt.Errorf("no disk context available for multi-disk NBD target selection")
	}

	// Calculate current disk ID from VMware disk key
	currentDiskID := t.getCurrentDiskID()

	log.Printf("🎯 Multi-disk mode: Looking for NBD target for disk %s (VMware key: %d)", currentDiskID, t.Disk.Key)

	// Parse NBD targets: "2000:nbd://...,2001:nbd://..." (VMware disk keys)
	targetPairs := strings.Split(nbdTargetsStr, ",")

	// Create VMware disk key → NBD target mapping (export + host + port)
	type NBDTargetInfo struct {
		ExportName string
		Host       string
		Port       string
	}
	targetMap := make(map[string]NBDTargetInfo)
	
	for _, pair := range targetPairs {
		parts := strings.SplitN(pair, ":", 2)
		if len(parts) != 2 {
			log.Printf("⚠️ Invalid NBD target format: %s", pair)
			continue
		}

		diskKey := parts[0]
		nbdURL := parts[1]

		// Extract host, port, and export name from NBD URL (nbd://host:port/export_name)
		parsedURL, err := url.Parse(nbdURL)
		if err != nil {
			log.Printf("⚠️ Failed to parse NBD URL: %s", nbdURL)
			continue
		}

		exportName := strings.TrimPrefix(parsedURL.Path, "/")
		host := parsedURL.Hostname()
		port := parsedURL.Port()
		
		if host == "" {
			host = "127.0.0.1"
		}
		if port == "" {
			port = "10809"
		}
		
		targetMap[diskKey] = NBDTargetInfo{
			ExportName: exportName,
			Host:       host,
			Port:       port,
		}

		log.Printf("🔍 Mapped NBD target: VMware disk key %s → %s:%s/%s", diskKey, host, port, exportName)
	}

	// 🎯 DIRECT CORRELATION: Use VMware disk key for exact matching
	vmwareDiskKey := fmt.Sprintf("%d", t.Disk.Key)

	if targetInfo, exists := targetMap[vmwareDiskKey]; exists {
		// Update NBD connection parameters for this disk
		t.nbdHost = targetInfo.Host
		t.nbdPort = targetInfo.Port
		log.Printf("✅ DIRECT MATCH: VMware disk %s (key:%d) → %s:%s/%s", currentDiskID, t.Disk.Key, t.nbdHost, t.nbdPort, targetInfo.ExportName)
		return targetInfo.ExportName, nil
	} else {
		log.Printf("❌ No NBD target found for VMware disk key %s (disk %s)", vmwareDiskKey, currentDiskID)
		log.Printf("🔍 Available targets: %v", targetMap)
		return "", fmt.Errorf("no NBD target found for VMware disk key %s (disk %s)", vmwareDiskKey, currentDiskID)
	}

	return "", fmt.Errorf("no matching NBD target found for disk %s in targets: %s", currentDiskID, nbdTargetsStr)
}

// getChangeIDFromOMA retrieves ChangeID from SHA database via API
func (t *NBDTarget) getChangeIDFromOMA(vmPath string) (string, error) {
	// Call SHA API to get previous ChangeID
	shaURL := os.Getenv("SHA_API_URL")
	if shaURL == "" {
		shaURL = "http://localhost:8082" // Default for SNA tunnel
	}

	// Determine if this is a backup or replication
	jobID := os.Getenv("MIGRATEKIT_JOB_ID")
	isBackup := strings.HasPrefix(jobID, "backup-")
	
	var apiURL string
	
	if isBackup {
		// ✅ NEW: Use backup-specific endpoint
		// Extract VM name from vmPath (/DatabanxDC/vm/pgtest1 → pgtest1)
		parts := strings.Split(vmPath, "/")
		vmName := parts[len(parts)-1]
		
		// Get disk index from job ID (e.g., "backup-pgtest1-disk0-..." → 0)
		diskIndex := t.getDiskIndexFromJobID(jobID)
		
		apiURL = fmt.Sprintf("%s/api/v1/backups/changeid?vm_name=%s&disk_id=%d",
			shaURL, url.QueryEscape(vmName), diskIndex)
		
		log.Printf("📡 Getting ChangeID from BACKUP API for VM %s disk %d", vmName, diskIndex)
	} else {
		// Replication - use existing logic
		diskID := t.getCurrentDiskID()
		encodedVMPath := url.QueryEscape(vmPath)
		encodedDiskID := url.QueryEscape(diskID)
		
		apiURL = fmt.Sprintf("%s/api/v1/replications/changeid?vm_path=%s&disk_id=%s",
			shaURL, encodedVMPath, encodedDiskID)
		
		log.Printf("📡 Getting ChangeID from REPLICATION API for disk %s", diskID)
	}

	resp, err := http.Get(apiURL)
	if err != nil {
		return "", fmt.Errorf("failed to call SHA API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("SHA API returned status %d: %s", resp.StatusCode, string(body))
	}

	var response map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", fmt.Errorf("failed to decode SHA API response: %w", err)
	}

	changeID := response["change_id"]
	if changeID != "" {
		log.Printf("📋 Found previous ChangeID: %s", changeID)
	} else {
		log.Printf("📋 No previous ChangeID found")
	}

	return changeID, nil
}

// storeChangeIDInOMA stores ChangeID in SHA database via API
// Automatically detects job type (backup vs replication) and calls appropriate endpoint
func (t *NBDTarget) storeChangeIDInOMA(jobID, changeID string) error {
	// Call SHA API to store ChangeID
	shaURL := os.Getenv("SHA_API_URL")
	if shaURL == "" {
		shaURL = "http://localhost:8082" // Default for SNA tunnel
	}

	// Determine API endpoint based on job ID prefix
	var apiURL string
	var payload map[string]interface{}
	
	if strings.HasPrefix(jobID, "backup-") {
		// Backup job - use backup completion endpoint
		apiURL = fmt.Sprintf("%s/api/v1/backups/%s/complete", shaURL, jobID)
		
		// Extract disk index from job ID (e.g., "backup-pgtest1-disk0-..." → 0)
		diskIndex := t.getDiskIndexFromJobID(jobID)
		
		// Backup API accepts change_id with disk_id for multi-disk support
		payload = map[string]interface{}{
			"change_id":         changeID,
			"disk_id":           diskIndex, // ✅ NEW: numeric disk index for multi-disk VMs
			"bytes_transferred": 0,         // Client doesn't track total bytes
		}
		
		log.Printf("📡 Storing ChangeID via BACKUP completion API for disk %d", diskIndex)
	} else {
		// Replication job - use replication endpoint
		apiURL = fmt.Sprintf("%s/api/v1/replications/%s/changeid", shaURL, jobID)
		
		// Calculate disk ID for replication (per-disk tracking)
		diskID := t.getCurrentDiskID()
		
		// Replication API requires disk_id for per-disk change tracking
		payload = map[string]interface{}{
			"change_id": changeID,
			"disk_id":   diskID,
		}
		
		log.Printf("📡 Storing ChangeID via REPLICATION API for disk %s", diskID)
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal request data: %w", err)
	}

	log.Printf("🔄 API URL: %s", apiURL)
	log.Printf("🔄 Job ID: %s", jobID)
	log.Printf("🔄 ChangeID: %s", changeID)

	resp, err := http.Post(apiURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to call SHA API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("SHA API returned status %d: %s", resp.StatusCode, string(body))
	}

	log.Printf("✅ Successfully stored ChangeID in SHA database")
	return nil
}
