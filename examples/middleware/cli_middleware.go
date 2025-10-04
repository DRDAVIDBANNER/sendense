package middleware

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"runtime/debug"
	"time"

	"github.com/vexxhost/migratekit/internal/joblog"
)

// CLIJobWrapper provides CLI command wrapping with job tracking
type CLIJobWrapper struct {
	tracker *joblog.Tracker
	config  *CLIJobConfig
}

// CLIJobConfig configures the CLI job wrapper
type CLIJobConfig struct {
	// JobType is the job type for CLI commands
	JobType string
	
	// Owner identifies who is running the CLI
	Owner *string
	
	// RecoverPanics enables panic recovery with job status updates
	RecoverPanics bool
	
	// LogLevel sets the logging level for CLI operations
	LogLevel slog.Level
}

// DefaultCLIJobConfig returns sensible defaults for CLI job tracking
func DefaultCLIJobConfig() *CLIJobConfig {
	owner := "cli-user"
	if user := os.Getenv("USER"); user != "" {
		owner = user
	}
	
	return &CLIJobConfig{
		JobType:       "cli_command",
		Owner:         &owner,
		RecoverPanics: true,
		LogLevel:      slog.LevelInfo,
	}
}

// NewCLIJobWrapper creates a new CLI job wrapper
func NewCLIJobWrapper(tracker *joblog.Tracker, config *CLIJobConfig) *CLIJobWrapper {
	if config == nil {
		config = DefaultCLIJobConfig()
	}
	
	return &CLIJobWrapper{
		tracker: tracker,
		config:  config,
	}
}

// ExecuteCommand wraps a CLI command with job tracking
func (cjw *CLIJobWrapper) ExecuteCommand(
	operation string,
	metadata map[string]any,
	fn func(ctx context.Context) error,
) error {
	ctx := context.Background()
	
	// Start job for the command
	jobStart := joblog.JobStart{
		JobType:   cjw.config.JobType,
		Operation: operation,
		Owner:     cjw.config.Owner,
		Metadata:  metadata,
	}
	
	ctx, jobID, err := cjw.tracker.StartJob(ctx, jobStart)
	if err != nil {
		return fmt.Errorf("failed to start job for command %s: %w", operation, err)
	}
	
	logger := cjw.tracker.Logger(ctx)
	logger.Info("CLI command started",
		slog.String("operation", operation),
		slog.String("job_id", jobID),
	)
	
	// Set up panic recovery if enabled
	var cmdErr error
	if cjw.config.RecoverPanics {
		defer func() {
			if r := recover(); r != nil {
				stack := debug.Stack()
				
				// Log the panic
				logger.Error("Panic in CLI command",
					slog.String("panic", fmt.Sprintf("%v", r)),
					slog.String("stack", string(stack)),
				)
				
				// Convert panic to error
				cmdErr = fmt.Errorf("command panic: %v", r)
				
				// End job with failure
				if endErr := cjw.tracker.EndJob(ctx, jobID, joblog.StatusFailed, cmdErr); endErr != nil {
					logger.Error("Failed to end job after panic",
						slog.String("error", endErr.Error()),
					)
				}
			}
		}()
	}
	
	// Execute the command
	start := time.Now()
	cmdErr = fn(ctx)
	duration := time.Since(start)
	
	// Determine job status
	var status joblog.Status
	if cmdErr != nil {
		status = joblog.StatusFailed
		logger.Error("CLI command failed",
			slog.String("operation", operation),
			slog.String("error", cmdErr.Error()),
			slog.Duration("duration", duration),
		)
	} else {
		status = joblog.StatusCompleted
		logger.Info("CLI command completed",
			slog.String("operation", operation),
			slog.Duration("duration", duration),
		)
	}
	
	// End the job
	if err := cjw.tracker.EndJob(ctx, jobID, status, cmdErr); err != nil {
		logger.Error("Failed to end job for CLI command",
			slog.String("error", err.Error()),
		)
	}
	
	return cmdErr
}

// ExecuteWithSteps wraps a CLI command that has multiple steps
func (cjw *CLIJobWrapper) ExecuteWithSteps(
	operation string,
	metadata map[string]any,
	steps []CLIStep,
) error {
	ctx := context.Background()
	
	// Start job for the command
	jobStart := joblog.JobStart{
		JobType:   cjw.config.JobType,
		Operation: operation,
		Owner:     cjw.config.Owner,
		Metadata:  metadata,
	}
	
	ctx, jobID, err := cjw.tracker.StartJob(ctx, jobStart)
	if err != nil {
		return fmt.Errorf("failed to start job for command %s: %w", operation, err)
	}
	
	logger := cjw.tracker.Logger(ctx)
	logger.Info("Multi-step CLI command started",
		slog.String("operation", operation),
		slog.Int("total_steps", len(steps)),
	)
	
	// Execute steps
	for i, step := range steps {
		err := cjw.tracker.RunStep(ctx, jobID, step.Name, func(stepCtx context.Context) error {
			stepLogger := cjw.tracker.Logger(stepCtx)
			stepLogger.Info("Executing step",
				slog.String("step_name", step.Name),
				slog.Int("step_number", i+1),
				slog.Int("total_steps", len(steps)),
			)
			
			start := time.Now()
			stepErr := step.Fn(stepCtx)
			duration := time.Since(start)
			
			if stepErr != nil {
				stepLogger.Error("Step failed",
					slog.String("step_name", step.Name),
					slog.String("error", stepErr.Error()),
					slog.Duration("duration", duration),
				)
				return stepErr
			}
			
			stepLogger.Info("Step completed",
				slog.String("step_name", step.Name),
				slog.Duration("duration", duration),
			)
			
			// Update job progress
			progress := uint8((i + 1) * 100 / len(steps))
			if progErr := cjw.tracker.MarkJobProgress(ctx, jobID, progress); progErr != nil {
				stepLogger.Warn("Failed to update progress",
					slog.String("error", progErr.Error()),
				)
			}
			
			return nil
		})
		
		if err != nil {
			// End job with failure
			if endErr := cjw.tracker.EndJob(ctx, jobID, joblog.StatusFailed, err); endErr != nil {
				logger.Error("Failed to end job after step failure",
					slog.String("error", endErr.Error()),
				)
			}
			return fmt.Errorf("step %s failed: %w", step.Name, err)
		}
	}
	
	// All steps completed successfully
	if err := cjw.tracker.EndJob(ctx, jobID, joblog.StatusCompleted, nil); err != nil {
		logger.Error("Failed to end successful job",
			slog.String("error", err.Error()),
		)
	}
	
	logger.Info("All steps completed successfully",
		slog.String("operation", operation),
		slog.Int("steps_completed", len(steps)),
	)
	
	return nil
}

// CLIStep represents a single step in a multi-step CLI operation
type CLIStep struct {
	Name string
	Fn   func(ctx context.Context) error
}

// Example CLI applications

// ExampleMigrationCLI demonstrates a migration CLI tool with job tracking
func ExampleMigrationCLI() {
	// Database setup
	dsn := os.Getenv("DB_DSN")
	if dsn == "" {
		dsn = "root:password@tcp(localhost:3306)/migratekit?parseTime=true"
	}
	
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		fmt.Printf("Failed to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()
	
	// Create tracker
	stdoutHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
	dbHandler := joblog.NewDBHandler(db, joblog.DefaultDBHandlerConfig())
	tracker := joblog.New(db, stdoutHandler, dbHandler)
	defer tracker.Close()
	
	// Create CLI wrapper
	wrapper := NewCLIJobWrapper(tracker, DefaultCLIJobConfig())
	
	// Parse command line arguments (simplified)
	if len(os.Args) < 2 {
		fmt.Println("Usage: migration-cli <command> [args...]")
		os.Exit(1)
	}
	
	command := os.Args[1]
	
	switch command {
	case "migrate":
		err = wrapper.ExecuteCommand(
			"migrate-vm",
			map[string]any{
				"vm_name":      getArg(2, ""),
				"source_type":  "vmware",
				"target_type":  "cloudstack",
				"command_args": os.Args[2:],
			},
			func(ctx context.Context) error {
				return migrateVM(ctx, tracker, getArg(2, ""))
			},
		)
		
	case "failover":
		err = wrapper.ExecuteWithSteps(
			"test-failover",
			map[string]any{
				"vm_name":      getArg(2, ""),
				"failover_type": "test",
				"command_args": os.Args[2:],
			},
			[]CLIStep{
				{
					Name: "validate-vm",
					Fn: func(ctx context.Context) error {
						return validateVM(ctx, tracker, getArg(2, ""))
					},
				},
				{
					Name: "create-snapshot",
					Fn: func(ctx context.Context) error {
						return createSnapshot(ctx, tracker, getArg(2, ""))
					},
				},
				{
					Name: "execute-failover",
					Fn: func(ctx context.Context) error {
						return executeFailover(ctx, tracker, getArg(2, ""))
					},
				},
			},
		)
		
	case "cleanup":
		err = wrapper.ExecuteCommand(
			"cleanup-test-failover",
			map[string]any{
				"vm_name":      getArg(2, ""),
				"command_args": os.Args[2:],
			},
			func(ctx context.Context) error {
				return cleanupTestFailover(ctx, tracker, getArg(2, ""))
			},
		)
		
	default:
		fmt.Printf("Unknown command: %s\n", command)
		os.Exit(1)
	}
	
	if err != nil {
		fmt.Printf("Command failed: %v\n", err)
		os.Exit(1)
	}
	
	fmt.Println("Command completed successfully")
}

// Example command implementations

func migrateVM(ctx context.Context, tracker *joblog.Tracker, vmName string) error {
	logger := tracker.Logger(ctx)
	
	if vmName == "" {
		return fmt.Errorf("VM name is required")
	}
	
	logger.Info("Starting VM migration", slog.String("vm_name", vmName))
	
	// Simulate migration work
	time.Sleep(2 * time.Second)
	
	logger.Info("VM migration completed", slog.String("vm_name", vmName))
	return nil
}

func validateVM(ctx context.Context, tracker *joblog.Tracker, vmName string) error {
	logger := tracker.Logger(ctx)
	
	if vmName == "" {
		return fmt.Errorf("VM name is required")
	}
	
	logger.Info("Validating VM for failover", slog.String("vm_name", vmName))
	
	// Simulate validation
	time.Sleep(1 * time.Second)
	
	// Simulate validation failure for demo
	if vmName == "failing-vm" {
		return fmt.Errorf("VM validation failed: VM is not ready for failover")
	}
	
	logger.Info("VM validation passed", slog.String("vm_name", vmName))
	return nil
}

func createSnapshot(ctx context.Context, tracker *joblog.Tracker, vmName string) error {
	logger := tracker.Logger(ctx)
	
	logger.Info("Creating snapshot for rollback protection", slog.String("vm_name", vmName))
	
	// Simulate snapshot creation
	time.Sleep(3 * time.Second)
	
	snapshotName := fmt.Sprintf("%s-failover-snapshot-%d", vmName, time.Now().Unix())
	logger.Info("Snapshot created successfully",
		slog.String("vm_name", vmName),
		slog.String("snapshot_name", snapshotName),
	)
	
	return nil
}

func executeFailover(ctx context.Context, tracker *joblog.Tracker, vmName string) error {
	logger := tracker.Logger(ctx)
	
	logger.Info("Executing test failover", slog.String("vm_name", vmName))
	
	// Simulate failover execution
	time.Sleep(4 * time.Second)
	
	logger.Info("Test failover completed successfully",
		slog.String("vm_name", vmName),
		slog.String("test_vm_id", fmt.Sprintf("test-%s-%d", vmName, time.Now().Unix())),
	)
	
	return nil
}

func cleanupTestFailover(ctx context.Context, tracker *joblog.Tracker, vmName string) error {
	logger := tracker.Logger(ctx)
	
	if vmName == "" {
		return fmt.Errorf("VM name is required")
	}
	
	logger.Info("Cleaning up test failover", slog.String("vm_name", vmName))
	
	// Simulate cleanup work
	time.Sleep(2 * time.Second)
	
	logger.Info("Test failover cleanup completed", slog.String("vm_name", vmName))
	return nil
}

// Helper functions

func getArg(index int, defaultValue string) string {
	if len(os.Args) > index {
		return os.Args[index]
	}
	return defaultValue
}

// OrchestrationCLI demonstrates parent-child job relationships in CLI
func OrchestrationCLI() {
	// Database setup
	dsn := os.Getenv("DB_DSN")
	if dsn == "" {
		dsn = "root:password@tcp(localhost:3306)/migratekit?parseTime=true"
	}
	
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		fmt.Printf("Failed to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()
	
	// Create tracker
	stdoutHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
	dbHandler := joblog.NewDBHandler(db, joblog.DefaultDBHandlerConfig())
	tracker := joblog.New(db, stdoutHandler, dbHandler)
	defer tracker.Close()
	
	// Create CLI wrapper
	wrapper := NewCLIJobWrapper(tracker, DefaultCLIJobConfig())
	
	// Start parent orchestration job
	ctx := context.Background()
	parentStart := joblog.JobStart{
		JobType:   "orchestration",
		Operation: "bulk-migration",
		Owner:     wrapper.config.Owner,
		Metadata: map[string]any{
			"vm_list": []string{"vm1", "vm2", "vm3"},
			"type":    "bulk_operation",
		},
	}
	
	parentCtx, parentJobID, err := tracker.StartJob(ctx, parentStart)
	if err != nil {
		fmt.Printf("Failed to start orchestration job: %v\n", err)
		os.Exit(1)
	}
	
	logger := tracker.Logger(parentCtx)
	logger.Info("Starting bulk migration orchestration",
		slog.String("parent_job_id", parentJobID),
	)
	
	// Execute child jobs for each VM
	vms := []string{"vm1", "vm2", "vm3"}
	for i, vmName := range vms {
		// Create child job
		childStart := joblog.JobStart{
			ParentJobID: &parentJobID,
			JobType:     "migration",
			Operation:   fmt.Sprintf("migrate-%s", vmName),
			Owner:       wrapper.config.Owner,
			Metadata: map[string]any{
				"vm_name":     vmName,
				"vm_index":    i + 1,
				"total_vms":   len(vms),
				"parent_job":  parentJobID,
			},
		}
		
		childCtx, childJobID, err := tracker.StartJob(parentCtx, childStart)
		if err != nil {
			logger.Error("Failed to start child job",
				slog.String("vm_name", vmName),
				slog.String("error", err.Error()),
			)
			continue
		}
		
		// Execute migration steps for this VM
		err = tracker.RunStep(childCtx, childJobID, "vm-migration", func(stepCtx context.Context) error {
			stepLogger := tracker.Logger(stepCtx)
			stepLogger.Info("Migrating VM",
				slog.String("vm_name", vmName),
				slog.String("child_job_id", childJobID),
			)
			
			// Simulate migration work
			time.Sleep(time.Duration(2+i) * time.Second)
			
			stepLogger.Info("VM migration completed",
				slog.String("vm_name", vmName),
			)
			
			return nil
		})
		
		// End child job
		var childStatus joblog.Status
		if err != nil {
			childStatus = joblog.StatusFailed
			logger.Error("Child job failed",
				slog.String("vm_name", vmName),
				slog.String("child_job_id", childJobID),
				slog.String("error", err.Error()),
			)
		} else {
			childStatus = joblog.StatusCompleted
			logger.Info("Child job completed",
				slog.String("vm_name", vmName),
				slog.String("child_job_id", childJobID),
			)
		}
		
		if endErr := tracker.EndJob(childCtx, childJobID, childStatus, err); endErr != nil {
			logger.Error("Failed to end child job",
				slog.String("child_job_id", childJobID),
				slog.String("error", endErr.Error()),
			)
		}
		
		// Update parent job progress
		progress := uint8((i + 1) * 100 / len(vms))
		if progErr := tracker.MarkJobProgress(parentCtx, parentJobID, progress); progErr != nil {
			logger.Warn("Failed to update parent job progress",
				slog.String("error", progErr.Error()),
			)
		}
	}
	
	// End parent job
	if err := tracker.EndJob(parentCtx, parentJobID, joblog.StatusCompleted, nil); err != nil {
		logger.Error("Failed to end parent job",
			slog.String("error", err.Error()),
		)
	}
	
	logger.Info("Bulk migration orchestration completed",
		slog.String("parent_job_id", parentJobID),
		slog.Int("vms_processed", len(vms)),
	)
}
