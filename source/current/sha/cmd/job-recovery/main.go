// Job Recovery CLI Tool for MigrateKit OSSEA
// Provides manual job recovery capabilities for operational reliability
package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	var (
		dbHost = flag.String("db-host", "localhost", "Database host")
		dbPort = flag.String("db-port", "3306", "Database port")
		dbName = flag.String("db-name", "migratekit_oma", "Database name")
		dbUser = flag.String("db-user", "oma_user", "Database user")
		dbPass = flag.String("db-pass", "oma_password", "Database password")
		action = flag.String("action", "scan", "Action: scan, recover, status")
		maxAge = flag.Int("max-age", 30, "Maximum job age in minutes for recovery")
		dryRun = flag.Bool("dry-run", false, "Show what would be done without making changes")
	)
	flag.Parse()

	// Initialize database connection
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		*dbUser, *dbPass, *dbHost, *dbPort, *dbName)

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	switch *action {
	case "scan":
		fmt.Println("🔍 Scanning for orphaned jobs...")
		if err := scanOrphanedJobs(db, *maxAge); err != nil {
			log.Fatalf("Scan failed: %v", err)
		}

	case "recover":
		if *dryRun {
			fmt.Println("🧪 DRY RUN: Would recover orphaned jobs (no changes made)")
			if err := scanOrphanedJobs(db, *maxAge); err != nil {
				log.Fatalf("Dry run scan failed: %v", err)
			}
		} else {
			fmt.Println("🔄 Recovering orphaned jobs...")
			if err := recoverOrphanedJobs(db, *maxAge); err != nil {
				log.Fatalf("Recovery failed: %v", err)
			}
		}

	case "status":
		fmt.Println("📊 Job recovery status...")
		if err := showJobStatus(db, *maxAge); err != nil {
			log.Fatalf("Status check failed: %v", err)
		}

	default:
		fmt.Printf("Unknown action: %s\n", *action)
		fmt.Println("Available actions: scan, recover, status")
		os.Exit(1)
	}
}

type OrphanedJob struct {
	ID              string  `gorm:"column:id"`
	SourceVMName    string  `gorm:"column:source_vm_name"`
	Status          string  `gorm:"column:status"`
	CreatedAt       string  `gorm:"column:created_at"`
	UpdatedAt       string  `gorm:"column:updated_at"`
	ProgressPercent float64 `gorm:"column:progress_percent"`
	VMContextID     string  `gorm:"column:vm_context_id"`
}

func scanOrphanedJobs(db *gorm.DB, maxAge int) error {
	var jobs []OrphanedJob
	query := `
		SELECT id, source_vm_name, status, created_at, updated_at, progress_percent, vm_context_id
		FROM replication_jobs 
		WHERE status = 'replicating' 
		AND updated_at < NOW() - INTERVAL ? MINUTE
		ORDER BY updated_at ASC
	`

	err := db.Raw(query, maxAge).Scan(&jobs).Error
	if err != nil {
		return fmt.Errorf("failed to query orphaned jobs: %w", err)
	}

	if len(jobs) == 0 {
		fmt.Println("✅ No orphaned jobs found")
		return nil
	}

	fmt.Printf("Found %d potentially orphaned jobs:\n\n", len(jobs))

	for _, job := range jobs {
		fmt.Printf("Job ID: %s\n", job.ID)
		fmt.Printf("VM Name: %s\n", job.SourceVMName)
		fmt.Printf("Status: %s\n", job.Status)
		fmt.Printf("Created: %s\n", job.CreatedAt)
		fmt.Printf("Last Update: %s\n", job.UpdatedAt)
		fmt.Printf("Progress: %.1f%%\n", job.ProgressPercent)
		fmt.Printf("VM Context: %s\n", job.VMContextID)
		fmt.Println("---")
	}

	return nil
}

func recoverOrphanedJobs(db *gorm.DB, maxAge int) error {
	var jobs []OrphanedJob
	query := `
		SELECT id, source_vm_name, status, vm_context_id
		FROM replication_jobs 
		WHERE status = 'replicating' 
		AND updated_at < NOW() - INTERVAL ? MINUTE
	`

	err := db.Raw(query, maxAge).Scan(&jobs).Error
	if err != nil {
		return fmt.Errorf("failed to query jobs for recovery: %w", err)
	}

	if len(jobs) == 0 {
		fmt.Println("✅ No jobs need recovery")
		return nil
	}

	fmt.Printf("Recovering %d orphaned jobs...\n\n", len(jobs))

	for _, job := range jobs {
		fmt.Printf("Recovering job: %s (%s)\n", job.ID, job.SourceVMName)

		// Mark job as failed
		updates := map[string]interface{}{
			"status":        "failed",
			"error_message": "Job recovery: Process orphaned during service restart",
			"completed_at":  "NOW()",
			"updated_at":    "NOW()",
		}

		err := db.Model(&OrphanedJob{}).Where("id = ?", job.ID).Updates(updates).Error
		if err != nil {
			fmt.Printf("❌ Failed to recover job %s: %v\n", job.ID, err)
			continue
		}

		// Update VM context if available
		if job.VMContextID != "" {
			err = db.Exec("UPDATE vm_replication_contexts SET current_status = ?, current_job_id = ?, updated_at = NOW() WHERE context_id = ?",
				"ready_for_failover", nil, job.VMContextID).Error
			if err != nil {
				fmt.Printf("⚠️ Failed to update VM context for %s: %v\n", job.VMContextID, err)
			}
		}

		fmt.Printf("✅ Job %s recovered successfully\n", job.ID)
	}

	return nil
}

func showJobStatus(db *gorm.DB, maxAge int) error {
	var jobs []OrphanedJob
	query := `
		SELECT id, source_vm_name, status, created_at, updated_at, progress_percent
		FROM replication_jobs 
		WHERE status = 'replicating' 
		ORDER BY updated_at ASC
	`

	err := db.Raw(query).Scan(&jobs).Error
	if err != nil {
		return fmt.Errorf("failed to query job status: %w", err)
	}

	fmt.Printf("Job Recovery Status Report\n")
	fmt.Printf("=========================\n")
	fmt.Printf("Total active replication jobs: %d\n", len(jobs))

	orphanedCount := 0
	for _, job := range jobs {
		// Check if job is older than maxAge
		var ageMinutes int
		db.Raw("SELECT TIMESTAMPDIFF(MINUTE, updated_at, NOW()) FROM replication_jobs WHERE id = ?", job.ID).Scan(&ageMinutes)

		if ageMinutes > maxAge {
			orphanedCount++
		}
	}

	fmt.Printf("Potentially orphaned jobs: %d\n", orphanedCount)

	if orphanedCount > 0 {
		fmt.Printf("\nRecommended action:\n")
		fmt.Printf("  ./job-recovery -action=recover\n")
	} else {
		fmt.Printf("\n✅ System is healthy - no orphaned jobs detected\n")
	}

	return nil
}
