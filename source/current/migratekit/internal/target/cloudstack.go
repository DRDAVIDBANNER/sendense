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
	"strconv"
	"strings"

	"github.com/vexxhost/migratekit/internal/cloudstack"
	"github.com/vexxhost/migratekit/internal/vmware"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/types"
	"libguestfs.org/libnbd"
)

type CloudStack struct {
	VirtualMachine *object.VirtualMachine
	Disk           *types.VirtualDisk
	ClientSet      *cloudstack.ClientSet
	nbdHandle      *libnbd.Libnbd
	nbdHost        string
	nbdPort        string
	nbdExportName  string
	SSHTarget      string
}

type CloudStackVolumeCreateOpts struct {
	AvailabilityZone string
	VolumeType       string
	BusType          string
}

func NewCloudStack(ctx context.Context, vm *object.VirtualMachine, disk *types.VirtualDisk) (*CloudStack, error) {
	clientSet, err := cloudstack.NewClientSet(ctx)
	if err != nil {
		return nil, err
	}

	return &CloudStack{
		VirtualMachine: vm,
		Disk:           disk,
		ClientSet:      clientSet,
	}, nil
}

func (t *CloudStack) GetDisk() *types.VirtualDisk {
	return t.Disk
}

func (t *CloudStack) Connect(ctx context.Context) error {
	log.Println("🔥🔥🔥 CloudStack Connect() called - NBD OVER TLS MODE! 🔥🔥🔥")

	// Check for local test mode
	if localPath := os.Getenv("LOCAL_TEST_DEVICE"); localPath != "" {
		log.Printf("🏠 LOCAL TEST MODE: NBD not supported for local devices, falling back to file: %s", localPath)
		return fmt.Errorf("LOCAL_TEST_DEVICE not supported with NBD mode - use SSH streaming mode")
	}

	// Configure NBD connection via local TLS tunnel
	t.nbdHost = "127.0.0.1" // Local stunnel client
	if port := os.Getenv("NBD_LOCAL_PORT"); port != "" {
		t.nbdPort = port
	} else {
		t.nbdPort = "10808"
	}

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

	// Connect via stunnel TLS tunnel (plain NBD to localhost)
	log.Printf("🔐 Step 4: Connecting to local stunnel tunnel %s:%s", t.nbdHost, t.nbdPort)
	log.Printf("🚀 Flow: libnbd → localhost:%s → stunnel → TLS:443 → CloudStack", t.nbdPort)

	err = handle.ConnectTcp(t.nbdHost, t.nbdPort)
	if err != nil {
		handle.Close()
		log.Printf("❌ ConnectTcp failed: %v", err)
		return fmt.Errorf("failed to connect via TLS tunnel: %v", err)
	}
	log.Printf("🎉 Connection established via TLS tunnel!")

	t.nbdHandle = handle
	log.Printf("✅ CloudStack NBD connection ready with TLS encryption: %s:%s/%s", t.nbdHost, t.nbdPort, t.nbdExportName)
	return nil
}

func (t *CloudStack) GetPath(ctx context.Context) (string, error) {
	if t.nbdHandle == nil {
		return "", fmt.Errorf("CloudStack target not connected - call Connect() first")
	}

	// Return special NBD identifier with dynamic export name that the incremental copy can recognize
	nbdPath := fmt.Sprintf("nbd://%s:%s/%s", t.nbdHost, t.nbdPort, t.nbdExportName)
	log.Printf("🚀 CloudStack GetPath() returning NBD: %s", nbdPath)
	return nbdPath, nil
}

// GetNBDHandle returns the NBD handle for positioned writes
func (t *CloudStack) GetNBDHandle() *libnbd.Libnbd {
	return t.nbdHandle
}

func (t *CloudStack) Disconnect(ctx context.Context) error {
	log.Println("🧹 CloudStack Disconnect() - Cleaning up NBD connection")

	// Close NBD handle
	if t.nbdHandle != nil {
		err := t.nbdHandle.Close()
		if err != nil {
			log.Printf("⚠️ Warning: Failed to close NBD handle: %v", err)
		}
		t.nbdHandle = nil
		log.Printf("🔌 Closed NBD connection to %s:%s", t.nbdHost, t.nbdPort)
	}

	log.Println("✅ CloudStack NBD cleanup completed")
	return nil
}

func (t *CloudStack) Exists(ctx context.Context) (bool, error) {
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

func (t *CloudStack) GetCurrentChangeID(ctx context.Context) (*vmware.ChangeID, error) {
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

func (t *CloudStack) WriteChangeID(ctx context.Context, changeId *vmware.ChangeID) error {
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

// getChangeIDFilePath returns the path where ChangeID is stored on the CloudStack appliance
// DEPRECATED: This method is no longer used - ChangeIDs are now stored in SHA database
func (t *CloudStack) getChangeIDFilePath() string {
	// Create a unique file path based on VM name and disk key
	vmName := t.VirtualMachine.Name()
	diskKey := strconv.Itoa(int(t.Disk.Key))
	return fmt.Sprintf("/tmp/migratekit_changeid_%s_disk_%s", vmName, diskKey)
}

func (t *CloudStack) CreateImageFromVolume(ctx context.Context) error {
	log.Println("🚧 CloudStack CreateImageFromVolume() - stub implementation")
	return nil
}

// CloudStackDiskLabel creates a label for the disk
func CloudStackDiskLabel(vm *object.VirtualMachine, disk *types.VirtualDisk) string {
	return vm.Name() + "-disk-" + string(rune(disk.Key))
}

// getCurrentDiskID calculates the disk ID for the current disk
func (t *CloudStack) getCurrentDiskID() string {
	if t.Disk == nil || t.Disk.Key == 0 {
		log.Printf("⚠️ No disk key available, falling back to default disk-2000")
		return "disk-2000" // Backward compatibility fallback
	}
	diskID := fmt.Sprintf("disk-%d", t.Disk.Key)
	log.Printf("🎯 Calculated disk ID for change ID storage: %s (VMware disk.Key: %d)", diskID, t.Disk.Key)
	return diskID
}

// determineNBDExportForDisk determines the correct NBD export for the current disk
func (t *CloudStack) determineNBDExportForDisk(ctx context.Context) (string, error) {
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
func (t *CloudStack) parseMultiDiskNBDTargets(ctx context.Context, nbdTargetsStr string) (string, error) {
	if t.Disk == nil {
		return "", fmt.Errorf("no disk context available for multi-disk NBD target selection")
	}

	// Calculate current disk ID from VMware disk key
	currentDiskID := t.getCurrentDiskID()

	log.Printf("🎯 Multi-disk mode: Looking for NBD target for disk %s (VMware key: %d)", currentDiskID, t.Disk.Key)

	// Parse NBD targets: "2000:nbd://...,2001:nbd://..." (VMware disk keys)
	targetPairs := strings.Split(nbdTargetsStr, ",")

	// Create VMware disk key → export_name mapping for direct correlation
	targetMap := make(map[string]string)
	for _, pair := range targetPairs {
		parts := strings.SplitN(pair, ":", 2)
		if len(parts) != 2 {
			log.Printf("⚠️ Invalid NBD target format: %s", pair)
			continue
		}

		diskKey := parts[0]
		nbdURL := parts[1]

		// Extract export name from NBD URL (nbd://host:port/export_name)
		parsedURL, err := url.Parse(nbdURL)
		if err != nil {
			log.Printf("⚠️ Failed to parse NBD URL: %s", nbdURL)
			continue
		}

		exportName := strings.TrimPrefix(parsedURL.Path, "/")
		targetMap[diskKey] = exportName

		log.Printf("🔍 Mapped NBD target: VMware disk key %s → export=%s", diskKey, exportName)
	}

	// 🎯 DIRECT CORRELATION: Use VMware disk key for exact matching
	vmwareDiskKey := fmt.Sprintf("%d", t.Disk.Key)

	if exportName, exists := targetMap[vmwareDiskKey]; exists {
		log.Printf("✅ DIRECT MATCH: VMware disk %s (key:%d) → export=%s", currentDiskID, t.Disk.Key, exportName)
		return exportName, nil
	} else {
		log.Printf("❌ No NBD target found for VMware disk key %s (disk %s)", vmwareDiskKey, currentDiskID)
		log.Printf("🔍 Available targets: %v", targetMap)
		return "", fmt.Errorf("no NBD target found for VMware disk key %s (disk %s)", vmwareDiskKey, currentDiskID)
	}

	return "", fmt.Errorf("no matching NBD target found for disk %s in targets: %s", currentDiskID, nbdTargetsStr)
}

// getChangeIDFromOMA retrieves ChangeID from SHA database via API
func (t *CloudStack) getChangeIDFromOMA(vmPath string) (string, error) {
	// Call SHA API to get previous ChangeID
	shaURL := os.Getenv("CLOUDSTACK_API_URL")
	if shaURL == "" {
		shaURL = "http://localhost:8082" // Default for SNA tunnel
	}

	// NEW: Calculate disk ID for this specific disk
	diskID := t.getCurrentDiskID() // Use our existing method!

	// Encode parameters
	encodedVMPath := url.QueryEscape(vmPath)
	encodedDiskID := url.QueryEscape(diskID)

	// NEW: Include disk_id parameter for multi-disk support
	apiURL := fmt.Sprintf("%s/api/v1/replications/changeid?vm_path=%s&disk_id=%s",
		shaURL, encodedVMPath, encodedDiskID)

	log.Printf("📡 Getting ChangeID from SHA API for disk %s: %s", diskID, apiURL)

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
		log.Printf("📋 Found previous ChangeID for disk %s: %s", diskID, changeID)
	} else {
		log.Printf("📋 No previous ChangeID found for disk %s", diskID)
	}

	return changeID, nil
}

// storeChangeIDInOMA stores ChangeID in SHA database via API
func (t *CloudStack) storeChangeIDInOMA(jobID, changeID string) error {
	// Call SHA API to store ChangeID
	shaURL := os.Getenv("CLOUDSTACK_API_URL")
	if shaURL == "" {
		shaURL = "http://localhost:8082" // Default for SNA tunnel
	}

	apiURL := fmt.Sprintf("%s/api/v1/replications/%s/changeid", shaURL, jobID)

	// CRITICAL FIX: Calculate correct disk ID from VMware disk.Key
	diskID := t.getCurrentDiskID()

	// Create request payload with dynamic disk ID
	payload := map[string]string{
		"change_id": changeID,
		"disk_id":   diskID, // Dynamic disk ID based on VMware disk.Key
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal request data: %w", err)
	}

	log.Printf("📡 Storing ChangeID in SHA API for disk %s: %s", diskID, apiURL)
	log.Printf("🔄 Change ID storage details - Job: %s, Disk: %s, ChangeID: %s", jobID, diskID, changeID)

	resp, err := http.Post(apiURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to call SHA API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("SHA API returned status %d: %s", resp.StatusCode, string(body))
	}

	log.Printf("✅ Successfully stored ChangeID %s in database for job %s", changeID, jobID)
	return nil
}
