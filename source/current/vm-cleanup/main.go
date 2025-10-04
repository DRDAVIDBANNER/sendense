package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

const (
	volumeDaemonURL    = "http://localhost:8090"
	dbConnectionString = "oma_user:oma_password@tcp(localhost:3306)/migratekit_oma"
)

type VMContext struct {
	ContextID     string `json:"context_id"`
	VMName        string `json:"vm_name"`
	CurrentStatus string `json:"current_status"`
}

type Volume struct {
	VolumeID   string `json:"volume_id"`
	VolumeName string `json:"volume_name"`
}

type Operation struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Status   string `json:"status"`
	VolumeID string `json:"volume_id"`
	Error    string `json:"error,omitempty"`
}

type Config struct {
	VMName  string
	DryRun  bool
	Force   bool
	Verbose bool
}

func main() {
	var config Config

	flag.StringVar(&config.VMName, "vm", "", "VM name to cleanup (required)")
	flag.BoolVar(&config.DryRun, "dry-run", false, "Show what would be deleted without actually deleting")
	flag.BoolVar(&config.Force, "force", false, "Skip confirmation prompt")
	flag.BoolVar(&config.Verbose, "verbose", false, "Enable verbose logging")
	flag.Parse()

	if config.VMName == "" {
		fmt.Println("❌ Error: VM name is required")
		fmt.Println("Usage: vm-cleanup -vm <vm_name> [-dry-run] [-force] [-verbose]")
		fmt.Println("")
		fmt.Println("Examples:")
		fmt.Println("  vm-cleanup -vm pgtest1 -dry-run")
		fmt.Println("  vm-cleanup -vm PGWINTESTBIOS -force")
		os.Exit(1)
	}

	if err := cleanup(config); err != nil {
		log.Fatalf("❌ Cleanup failed: %v", err)
	}
}

func cleanup(config Config) error {
	ctx := context.Background()

	// Connect to database
	db, err := sql.Open("mysql", dbConnectionString)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	// Step 1: Find VM context
	if config.Verbose {
		fmt.Printf("🔍 Looking up VM context for '%s'...\n", config.VMName)
	}

	vmContext, err := findVMContext(db, config.VMName)
	if err != nil {
		return fmt.Errorf("failed to find VM context: %w", err)
	}

	fmt.Printf("📋 Found VM: %s (Context: %s, Status: %s)\n",
		vmContext.VMName, vmContext.ContextID, vmContext.CurrentStatus)

	// Step 2: Find volumes
	if config.Verbose {
		fmt.Println("🔍 Looking up associated volumes...")
	}

	volumes, err := findVolumes(db, vmContext.ContextID)
	if err != nil {
		return fmt.Errorf("failed to find volumes: %w", err)
	}

	if len(volumes) == 0 {
		fmt.Println("📦 No volumes found for this VM")
	} else {
		fmt.Printf("📦 Found %d volume(s):\n", len(volumes))
		for _, vol := range volumes {
			fmt.Printf("   - %s (%s)\n", vol.VolumeName, vol.VolumeID)
		}
	}

	// Safety check: warn if VM is in active state
	if isActiveState(vmContext.CurrentStatus) {
		fmt.Printf("⚠️  WARNING: VM is in active state '%s'\n", vmContext.CurrentStatus)
		if !config.Force {
			fmt.Println("   Use -force to cleanup anyway, or stop the VM first")
			return fmt.Errorf("VM is in active state, aborting")
		}
	}

	// Step 3: Confirmation (unless dry-run or force)
	if !config.DryRun && !config.Force {
		fmt.Printf("\n⚠️  This will PERMANENTLY DELETE:\n")
		fmt.Printf("   - VM Context: %s\n", vmContext.ContextID)
		fmt.Printf("   - All replication jobs and history\n")
		fmt.Printf("   - All disk and volume mappings\n")
		fmt.Printf("   - All CBT history\n")
		if len(volumes) > 0 {
			fmt.Printf("   - %d CloudStack volume(s) will be detached first\n", len(volumes))
		}
		fmt.Printf("\nType 'yes' to confirm: ")

		var response string
		fmt.Scanln(&response)
		if strings.ToLower(response) != "yes" {
			fmt.Println("❌ Cleanup cancelled")
			return nil
		}
	}

	if config.DryRun {
		fmt.Println("\n🧪 DRY RUN MODE - No changes will be made")
		fmt.Println("Would perform the following actions:")
		for _, vol := range volumes {
			fmt.Printf("   1. Detach volume: %s\n", vol.VolumeID)
		}
		fmt.Printf("   2. Cascade delete VM context: %s\n", vmContext.ContextID)
		return nil
	}

	// Step 4: Detach volumes
	for _, vol := range volumes {
		if err := detachVolume(ctx, vol, config.Verbose); err != nil {
			return fmt.Errorf("failed to detach volume %s: %w", vol.VolumeID, err)
		}
	}

	// Step 5: Cascade delete VM context
	if config.Verbose {
		fmt.Printf("🗑️ Performing cascade delete of VM context '%s'...\n", vmContext.ContextID)
	}

	if err := deleteVMContext(db, vmContext.ContextID); err != nil {
		return fmt.Errorf("failed to delete VM context: %w", err)
	}

	// Step 6: Verify cleanup
	if config.Verbose {
		fmt.Println("✅ Verifying cleanup completion...")
	}

	if err := verifyCleanup(db, config.VMName, vmContext.ContextID); err != nil {
		return fmt.Errorf("cleanup verification failed: %w", err)
	}

	fmt.Printf("✅ Cleanup completed successfully for VM '%s'\n", config.VMName)
	return nil
}

func findVMContext(db *sql.DB, vmName string) (*VMContext, error) {
	query := `SELECT context_id, vm_name, current_status 
			  FROM vm_replication_contexts 
			  WHERE vm_name = ?`

	var ctx VMContext
	err := db.QueryRow(query, vmName).Scan(&ctx.ContextID, &ctx.VMName, &ctx.CurrentStatus)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("VM '%s' not found in system", vmName)
	}
	if err != nil {
		return nil, err
	}

	return &ctx, nil
}

func findVolumes(db *sql.DB, contextID string) ([]Volume, error) {
	query := `SELECT volume_id, volume_name 
			  FROM ossea_volumes 
			  WHERE vm_context_id = ?`

	rows, err := db.Query(query, contextID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var volumes []Volume
	for rows.Next() {
		var vol Volume
		if err := rows.Scan(&vol.VolumeID, &vol.VolumeName); err != nil {
			return nil, err
		}
		volumes = append(volumes, vol)
	}

	return volumes, rows.Err()
}

func isActiveState(status string) bool {
	activeStates := []string{"replicating", "discovering", "provisioning", "mounting"}
	for _, state := range activeStates {
		if status == state {
			return true
		}
	}
	return false
}

func detachVolume(ctx context.Context, vol Volume, verbose bool) error {
	if verbose {
		fmt.Printf("📤 Detaching volume: %s (%s)...\n", vol.VolumeName, vol.VolumeID)
	}

	// Start detach operation
	url := fmt.Sprintf("%s/api/v1/volumes/%s/detach", volumeDaemonURL, vol.VolumeID)
	resp, err := http.Post(url, "application/json", nil)
	if err != nil {
		return fmt.Errorf("failed to call volume daemon: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("volume daemon returned status %d", resp.StatusCode)
	}

	var operation Operation
	if err := json.NewDecoder(resp.Body).Decode(&operation); err != nil {
		return fmt.Errorf("failed to decode operation response: %w", err)
	}

	if verbose {
		fmt.Printf("⏳ Waiting for detach operation %s to complete...\n", operation.ID)
	}

	// Wait for completion
	for {
		time.Sleep(2 * time.Second)

		status, err := getOperationStatus(operation.ID)
		if err != nil {
			return fmt.Errorf("failed to check operation status: %w", err)
		}

		if status.Status == "completed" {
			if verbose {
				fmt.Printf("✅ Volume %s detached successfully\n", vol.VolumeID)
			}
			break
		}

		if status.Status == "failed" {
			return fmt.Errorf("detach operation failed: %s", status.Error)
		}

		if verbose {
			fmt.Printf("   Status: %s...\n", status.Status)
		}
	}

	return nil
}

func getOperationStatus(operationID string) (*Operation, error) {
	url := fmt.Sprintf("%s/api/v1/operations/%s", volumeDaemonURL, operationID)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var operation Operation
	if err := json.NewDecoder(resp.Body).Decode(&operation); err != nil {
		return nil, err
	}

	return &operation, nil
}

func deleteVMContext(db *sql.DB, contextID string) error {
	query := `DELETE FROM vm_replication_contexts WHERE context_id = ?`

	result, err := db.Exec(query, contextID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("no VM context found with ID: %s", contextID)
	}

	return nil
}

func verifyCleanup(db *sql.DB, vmName, contextID string) error {
	// Check VM context deleted
	var count int
	query := `SELECT COUNT(*) FROM vm_replication_contexts WHERE vm_name = ?`
	if err := db.QueryRow(query, vmName).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return fmt.Errorf("VM context still exists after deletion")
	}

	// Check related records cleaned up (they should be gone due to CASCADE DELETE)
	tables := []string{"vm_disks", "ossea_volumes"}
	for _, table := range tables {
		query := fmt.Sprintf(`SELECT COUNT(*) FROM %s WHERE vm_context_id = ?`, table)
		if err := db.QueryRow(query, contextID).Scan(&count); err != nil {
			return err
		}
		if count > 0 {
			return fmt.Errorf("found %d orphaned records in %s table", count, table)
		}
	}

	return nil
}

