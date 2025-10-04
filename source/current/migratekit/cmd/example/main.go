// Example demonstrates the joblog package with end-to-end usage
package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"log/slog"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/vexxhost/migratekit/internal/joblog"
)

func main() {
	// Database connection
	dsn := os.Getenv("DB_DSN")
	if dsn == "" {
		dsn = "root:password@tcp(localhost:3306)/migratekit?parseTime=true"
		log.Printf("Using default DSN (set DB_DSN env var to override): %s", dsn)
	}

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	log.Println("✅ Database connection established")

	// Create handlers
	// 1. Stdout JSON handler for console output
	stdoutHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
		AddSource: true,
	})

	// 2. Database handler for structured logging
	dbConfig := &joblog.DBHandlerConfig{
		QueueSize:   1000,
		Level:       slog.LevelInfo,
		DropOldest:  true,
		WriterCount: 2,
	}
	dbHandler := joblog.NewDBHandler(db, dbConfig)
	defer dbHandler.Close()

	// Create tracker with fanout logging (stdout + database)
	tracker := joblog.New(db, stdoutHandler, dbHandler)
	defer tracker.Close()

	log.Println("✅ Job tracker initialized")

	// Example 1: Simple job with manual steps
	if err := runSimpleJobExample(tracker); err != nil {
		log.Printf("❌ Simple job example failed: %v", err)
	}

	// Example 2: Job with automatic step management
	if err := runAutoStepJobExample(tracker); err != nil {
		log.Printf("❌ Auto-step job example failed: %v", err)
	}

	// Example 3: Hierarchical jobs (parent-child)
	if err := runHierarchicalJobExample(tracker); err != nil {
		log.Printf("❌ Hierarchical job example failed: %v", err)
	}

	// Example 4: Error handling and recovery
	if err := runErrorHandlingExample(tracker); err != nil {
		log.Printf("❌ Error handling example failed: %v", err)
	}

	// Example 5: Panic recovery
	if err := runPanicRecoveryExample(tracker); err != nil {
		log.Printf("❌ Panic recovery example completed with expected error: %v", err)
	}

	log.Println("🎉 All examples completed")
}

// runSimpleJobExample demonstrates basic job and step tracking
func runSimpleJobExample(tracker *joblog.Tracker) error {
	ctx := context.Background()
	
	log.Println("\n📋 Example 1: Simple Job with Manual Steps")
	
	// Start a job
	jobStart := joblog.JobStart{
		JobType:   "migration",
		Operation: "vm-migrate-example",
		Owner:     stringPtr("example-user"),
		Metadata: map[string]any{
			"vm_name": "test-vm-001",
			"source":  "vmware",
			"target":  "cloudstack",
		},
	}
	
	ctx, jobID, err := tracker.StartJob(ctx, jobStart)
	if err != nil {
		return fmt.Errorf("failed to start job: %w", err)
	}
	
	log.Printf("Started job: %s", jobID)
	
	// Create manual steps
	steps := []struct {
		name     string
		duration time.Duration
		metadata map[string]any
	}{
		{
			name:     "vm-discovery",
			duration: 2 * time.Second,
			metadata: map[string]any{"vm_count": 1},
		},
		{
			name:     "snapshot-creation",
			duration: 3 * time.Second,
			metadata: map[string]any{"snapshot_size_gb": 100},
		},
		{
			name:     "data-replication",
			duration: 5 * time.Second,
			metadata: map[string]any{"bytes_transferred": 107374182400},
		},
	}
	
	// Execute steps manually
	for i, step := range steps {
		stepStart := joblog.StepStart{
			Name:     step.name,
			Metadata: step.metadata,
		}
		
		stepCtx, stepID, err := tracker.StartStep(ctx, jobID, stepStart)
		if err != nil {
			return fmt.Errorf("failed to start step %s: %w", step.name, err)
		}
		
		// Get logger with context
		logger := tracker.Logger(stepCtx)
		logger.Info("Executing step",
			slog.String("step_name", step.name),
			slog.Int("step_number", i+1),
			slog.Int("total_steps", len(steps)),
		)
		
		// Simulate work
		time.Sleep(step.duration)
		
		// Update job progress
		progress := uint8((i + 1) * 100 / len(steps))
		if err := tracker.MarkJobProgress(ctx, jobID, progress); err != nil {
			logger.Warn("Failed to update progress", slog.String("error", err.Error()))
		}
		
		// End step
		if err := tracker.EndStep(stepID, joblog.StatusCompleted, nil); err != nil {
			return fmt.Errorf("failed to end step %s: %w", step.name, err)
		}
		
		logger.Info("Step completed",
			slog.String("step_name", step.name),
			slog.Duration("duration", step.duration),
		)
	}
	
	// End job
	if err := tracker.EndJob(ctx, jobID, joblog.StatusCompleted, nil); err != nil {
		return fmt.Errorf("failed to end job: %w", err)
	}
	
	log.Printf("✅ Simple job completed: %s", jobID)
	
	// Show progress
	if progress, err := tracker.GetJobProgress(ctx, jobID); err == nil {
		log.Printf("   Progress: %d/%d steps (%.1f%%), Runtime: %ds",
			progress.CompletedSteps, progress.TotalSteps,
			progress.StepCompletion, progress.RuntimeSeconds)
	}
	
	return nil
}

// runAutoStepJobExample demonstrates the RunStep automatic step management
func runAutoStepJobExample(tracker *joblog.Tracker) error {
	ctx := context.Background()
	
	log.Println("\n🔄 Example 2: Job with Automatic Step Management")
	
	// Start a job
	jobStart := joblog.JobStart{
		JobType:   "failover",
		Operation: "enhanced-test-failover",
		Owner:     stringPtr("automation"),
		Metadata: map[string]any{
			"vm_id":      "vm-12345",
			"auto_cleanup": true,
		},
	}
	
	ctx, jobID, err := tracker.StartJob(ctx, jobStart)
	if err != nil {
		return fmt.Errorf("failed to start job: %w", err)
	}
	
	log.Printf("Started failover job: %s", jobID)
	
	// Step 1: Validation
	err = tracker.RunStep(ctx, jobID, "pre-failover-validation", func(stepCtx context.Context) error {
		logger := tracker.Logger(stepCtx)
		logger.Info("Validating VM state", slog.String("vm_id", "vm-12345"))
		
		time.Sleep(1 * time.Second)
		
		logger.Info("VM validation passed",
			slog.String("vm_state", "running"),
			slog.Int("disk_count", 2),
		)
		return nil
	})
	if err != nil {
		return fmt.Errorf("validation step failed: %w", err)
	}
	
	// Step 2: Snapshot creation
	err = tracker.RunStep(ctx, jobID, "linstor-snapshot", func(stepCtx context.Context) error {
		logger := tracker.Logger(stepCtx)
		logger.Info("Creating Linstor snapshot for rollback protection")
		
		time.Sleep(2 * time.Second)
		
		logger.Info("Snapshot created successfully",
			slog.String("snapshot_name", "test-failover-snapshot-001"),
			slog.String("volume_uuid", "vol-abcd1234"),
		)
		return nil
	})
	if err != nil {
		return fmt.Errorf("snapshot step failed: %w", err)
	}
	
	// Step 3: VirtIO injection (simulate potential failure)
	err = tracker.RunStep(ctx, jobID, "virtio-injection", func(stepCtx context.Context) error {
		logger := tracker.Logger(stepCtx)
		logger.Info("Injecting VirtIO drivers",
			slog.String("script_path", "/opt/migratekit/bin/inject-virtio-drivers.sh"),
		)
		
		time.Sleep(3 * time.Second)
		
		// Simulate success
		logger.Info("VirtIO drivers injected successfully",
			slog.String("device_path", "/dev/vdc"),
			slog.String("drivers", "viostor,netkvm,vioscsi"),
		)
		return nil
	})
	if err != nil {
		return fmt.Errorf("VirtIO injection failed: %w", err)
	}
	
	// End job
	if err := tracker.EndJob(ctx, jobID, joblog.StatusCompleted, nil); err != nil {
		return fmt.Errorf("failed to end job: %w", err)
	}
	
	log.Printf("✅ Auto-step job completed: %s", jobID)
	return nil
}

// runHierarchicalJobExample demonstrates parent-child job relationships
func runHierarchicalJobExample(tracker *joblog.Tracker) error {
	ctx := context.Background()
	
	log.Println("\n🌳 Example 3: Hierarchical Jobs (Parent-Child)")
	
	// Start parent job
	parentStart := joblog.JobStart{
		JobType:   "cleanup",
		Operation: "test-failover-cleanup",
		Owner:     stringPtr("cleanup-service"),
		Metadata: map[string]any{
			"cleanup_type": "full",
			"vm_name":      "pgtest1",
		},
	}
	
	parentCtx, parentJobID, err := tracker.StartJob(ctx, parentStart)
	if err != nil {
		return fmt.Errorf("failed to start parent job: %w", err)
	}
	
	log.Printf("Started parent cleanup job: %s", parentJobID)
	
	// Child job 1: VM cleanup
	vmCleanupStart := joblog.JobStart{
		ParentJobID: &parentJobID,
		JobType:     "cleanup",
		Operation:   "vm-cleanup",
		Owner:       stringPtr("cleanup-service"),
		Metadata: map[string]any{
			"vm_id": "vm-test-123",
		},
	}
	
	_, vmJobID, err := tracker.StartJob(parentCtx, vmCleanupStart)
	if err != nil {
		return fmt.Errorf("failed to start VM cleanup job: %w", err)
	}
	
	// Execute VM cleanup steps
	err = tracker.RunStep(parentCtx, vmJobID, "power-off-vm", func(stepCtx context.Context) error {
		logger := tracker.Logger(stepCtx)
		logger.Info("Powering off test VM")
		time.Sleep(1 * time.Second)
		logger.Info("VM powered off successfully")
		return nil
	})
	if err != nil {
		return fmt.Errorf("VM power-off failed: %w", err)
	}
	
	err = tracker.RunStep(parentCtx, vmJobID, "delete-vm", func(stepCtx context.Context) error {
		logger := tracker.Logger(stepCtx)
		logger.Info("Deleting test VM")
		time.Sleep(2 * time.Second)
		logger.Info("VM deleted successfully")
		return nil
	})
	if err != nil {
		return fmt.Errorf("VM deletion failed: %w", err)
	}
	
	if err := tracker.EndJob(parentCtx, vmJobID, joblog.StatusCompleted, nil); err != nil {
		return fmt.Errorf("failed to end VM cleanup job: %w", err)
	}
	
	// Child job 2: Volume cleanup
	volumeCleanupStart := joblog.JobStart{
		ParentJobID: &parentJobID,
		JobType:     "cleanup",
		Operation:   "volume-cleanup",
		Owner:       stringPtr("cleanup-service"),
		Metadata: map[string]any{
			"volume_count": 2,
		},
	}
	
	_, volumeJobID, err := tracker.StartJob(parentCtx, volumeCleanupStart)
	if err != nil {
		return fmt.Errorf("failed to start volume cleanup job: %w", err)
	}
	
	// Execute volume cleanup
	err = tracker.RunStep(parentCtx, volumeJobID, "reattach-volumes", func(stepCtx context.Context) error {
		logger := tracker.Logger(stepCtx)
		logger.Info("Reattaching volumes to OMA")
		time.Sleep(3 * time.Second)
		logger.Info("Volumes reattached successfully",
			slog.Int("volume_count", 2),
			slog.String("target", "OMA-vm"),
		)
		return nil
	})
	if err != nil {
		return fmt.Errorf("volume reattachment failed: %w", err)
	}
	
	if err := tracker.EndJob(parentCtx, volumeJobID, joblog.StatusCompleted, nil); err != nil {
		return fmt.Errorf("failed to end volume cleanup job: %w", err)
	}
	
	// Complete parent job
	if err := tracker.EndJob(parentCtx, parentJobID, joblog.StatusCompleted, nil); err != nil {
		return fmt.Errorf("failed to end parent job: %w", err)
	}
	
	log.Printf("✅ Hierarchical cleanup completed: parent=%s, children=[%s, %s]",
		parentJobID, vmJobID, volumeJobID)
	return nil
}

// runErrorHandlingExample demonstrates error handling and job failure
func runErrorHandlingExample(tracker *joblog.Tracker) error {
	ctx := context.Background()
	
	log.Println("\n❌ Example 4: Error Handling and Recovery")
	
	// Start a job that will fail
	jobStart := joblog.JobStart{
		JobType:   "migration",
		Operation: "failing-migration-example",
		Owner:     stringPtr("error-demo"),
		Metadata: map[string]any{
			"expected_failure": true,
		},
	}
	
	ctx, jobID, err := tracker.StartJob(ctx, jobStart)
	if err != nil {
		return fmt.Errorf("failed to start job: %w", err)
	}
	
	log.Printf("Started job (will fail): %s", jobID)
	
	// Step 1: Success
	err = tracker.RunStep(ctx, jobID, "preparation", func(stepCtx context.Context) error {
		logger := tracker.Logger(stepCtx)
		logger.Info("Preparing for migration")
		time.Sleep(1 * time.Second)
		logger.Info("Preparation completed")
		return nil
	})
	if err != nil {
		return fmt.Errorf("preparation step failed: %w", err)
	}
	
	// Step 2: Failure
	err = tracker.RunStep(ctx, jobID, "critical-operation", func(stepCtx context.Context) error {
		logger := tracker.Logger(stepCtx)
		logger.Info("Performing critical operation")
		time.Sleep(1 * time.Second)
		
		// Simulate an error
		logger.Error("Critical operation failed", slog.String("reason", "network timeout"))
		return fmt.Errorf("network timeout while connecting to source system")
	})
	
	// Expected error from step 2
	if err != nil {
		log.Printf("   Expected step failure: %v", err)
		
		// End job with failure
		if endErr := tracker.EndJob(ctx, jobID, joblog.StatusFailed, err); endErr != nil {
			return fmt.Errorf("failed to end job: %w", endErr)
		}
		
		log.Printf("✅ Error handling example completed (job failed as expected): %s", jobID)
		return nil
	}
	
	return fmt.Errorf("step should have failed but didn't")
}

// runPanicRecoveryExample demonstrates panic recovery
func runPanicRecoveryExample(tracker *joblog.Tracker) error {
	ctx := context.Background()
	
	log.Println("\n💥 Example 5: Panic Recovery")
	
	// Start a job that will panic
	jobStart := joblog.JobStart{
		JobType:   "test",
		Operation: "panic-recovery-example",
		Owner:     stringPtr("panic-demo"),
		Metadata: map[string]any{
			"expected_panic": true,
		},
	}
	
	ctx, jobID, err := tracker.StartJob(ctx, jobStart)
	if err != nil {
		return fmt.Errorf("failed to start job: %w", err)
	}
	
	log.Printf("Started job (will panic): %s", jobID)
	
	// This will panic and be recovered
	defer func() {
		if r := recover(); r != nil {
			log.Printf("   Recovered from panic: %v", r)
		}
	}()
	
	err = tracker.RunStep(ctx, jobID, "panic-step", func(stepCtx context.Context) error {
		logger := tracker.Logger(stepCtx)
		logger.Info("About to panic")
		time.Sleep(500 * time.Millisecond)
		
		// This will panic
		panic("simulated panic for demonstration")
	})
	
	// The error should contain the panic information
	if err != nil {
		log.Printf("   Step returned error after panic recovery: %v", err)
		
		// End job with failure
		if endErr := tracker.EndJob(ctx, jobID, joblog.StatusFailed, err); endErr != nil {
			return fmt.Errorf("failed to end job: %w", endErr)
		}
		
		log.Printf("✅ Panic recovery example completed: %s", jobID)
		return fmt.Errorf("expected panic occurred and was recovered")
	}
	
	return fmt.Errorf("step should have panicked but didn't")
}

// Helper function
func stringPtr(s string) *string {
	return &s
}
