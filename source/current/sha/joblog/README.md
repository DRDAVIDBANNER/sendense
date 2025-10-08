# JobLog - Unified Job Tracking and Structured Logging

A comprehensive, modular logging and job tracking package for Go applications, designed for the MigrateKit VMware→CloudStack migration project. Built on Go 1.21+ `log/slog` with asynchronous database logging and hierarchical job tracking.

## 🚀 Features

- **Unified Job & Step Tracking**: Complete lifecycle management with hierarchical parent-child relationships
- **Structured Logging**: Built on `log/slog` with automatic job/step context injection
- **Asynchronous Database Logging**: High-performance buffered writes with backpressure handling
- **Panic Recovery**: Automatic panic handling with proper job/step status updates
- **Context Propagation**: Seamless job and step ID propagation through Go contexts
- **Progress Tracking**: Real-time progress updates and completion monitoring
- **Fan-out Logging**: Send logs to multiple destinations (stdout, database, files)
- **Middleware Support**: Ready-to-use HTTP and CLI integration examples

## 📦 Installation

```bash
go get github.com/vexxhost/migratekit/internal/joblog
```

## 🗄️ Database Setup

First, run the database migration to create the required tables:

```sql
-- Apply the migration script
mysql -u root -p migratekit < internal/db/migrate.sql
```

This creates:
- Enhanced `job_tracking` table with progress tracking
- `job_steps` table for detailed step tracking  
- `log_events` table for structured logging
- Performance views for monitoring

## 🏁 Quick Start

### Basic Usage

```go
package main

import (
    "context"
    "database/sql"
    "log/slog"
    "os"
    
    _ "github.com/go-sql-driver/mysql"
    "github.com/vexxhost/migratekit/internal/joblog"
)

func main() {
    // Database connection
    db, err := sql.Open("mysql", "root:password@tcp(localhost:3306)/migratekit?parseTime=true")
    if err != nil {
        panic(err)
    }
    defer db.Close()
    
    // Create handlers
    stdoutHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
    dbHandler := joblog.NewDBHandler(db, joblog.DefaultDBHandlerConfig())
    
    // Create tracker with fan-out logging
    tracker := joblog.New(db, stdoutHandler, dbHandler)
    defer tracker.Close()
    
    // Start a job
    ctx := context.Background()
    jobStart := joblog.JobStart{
        JobType:   "migration",
        Operation: "vm-migrate",
        Owner:     stringPtr("user123"),
        Metadata: map[string]any{
            "vm_name": "test-vm",
            "source":  "vmware",
        },
    }
    
    ctx, jobID, err := tracker.StartJob(ctx, jobStart)
    if err != nil {
        panic(err)
    }
    
    // Execute work with automatic step management
    err = tracker.RunStep(ctx, jobID, "data-replication", func(stepCtx context.Context) error {
        logger := tracker.Logger(stepCtx)
        logger.Info("Starting data replication", 
            slog.String("vm_name", "test-vm"),
            slog.Int64("bytes_to_copy", 107374182400),
        )
        
        // Simulate work
        time.Sleep(2 * time.Second)
        
        logger.Info("Data replication completed",
            slog.Float64("throughput_gbps", 3.2),
        )
        return nil
    })
    
    // End job
    status := joblog.StatusCompleted
    if err != nil {
        status = joblog.StatusFailed
    }
    tracker.EndJob(ctx, jobID, status, err)
}

func stringPtr(s string) *string { return &s }
```

### Manual Step Management

```go
// Create steps manually for more control
stepStart := joblog.StepStart{
    Name: "vm-validation",
    Metadata: map[string]any{
        "vm_id": "vm-12345",
    },
}

stepCtx, stepID, err := tracker.StartStep(ctx, jobID, stepStart)
if err != nil {
    return err
}

// Get logger with automatic context
logger := tracker.Logger(stepCtx)
logger.Info("Validating VM configuration")

// ... do work ...

// End step with status
err = tracker.EndStep(stepID, joblog.StatusCompleted, nil)
```

## 🏗️ Architecture

### Core Components

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│     Tracker     │───▶│   slog.Logger    │───▶│  FanoutHandler  │
│   (Lifecycle)   │    │   (Structured)   │    │   (Multi-out)   │
└─────────────────┘    └──────────────────┘    └─────────────────┘
                                                         │
                                                         ▼
                                               ┌─────────────────┐
                                               │   StdoutHandler │
                                               │   DBHandler     │
                                               │   FileHandler   │
                                               └─────────────────┘
```

### Database Schema

```
job_tracking (enhanced)
├── id (varchar(64) PK)
├── parent_job_id (varchar(64))
├── job_type (enum)
├── operation (varchar(100))
├── status (enum)
├── percent_complete (tinyint) ✨ NEW
├── owner (varchar(100)) ✨ NEW
├── canceled_at (datetime(6)) ✨ NEW
└── ... (existing fields)

job_steps ✨ NEW
├── id (bigint PK)
├── job_id (varchar(64) FK)
├── name (varchar(200))
├── seq (int)
├── status (enum)
├── started_at (datetime(6))
├── completed_at (datetime(6))
├── error_message (longtext)
└── metadata (json)

log_events ✨ NEW
├── id (bigint PK)
├── job_id (varchar(64) FK)
├── step_id (bigint FK)
├── level (enum)
├── message (text)
├── attrs (json)
└── ts (datetime(6))
```

## 🔧 Configuration

### Database Handler Configuration

```go
config := &joblog.DBHandlerConfig{
    QueueSize:   10000,        // Buffer size for async writes
    Level:       slog.LevelInfo, // Minimum log level
    DropOldest:  true,         // Drop vs block on full queue
    WriterCount: 2,            // Concurrent DB writers
}

dbHandler := joblog.NewDBHandler(db, config)
```

### Handler Options

- **QueueSize**: Buffer size for log records (default: 10,000)
- **Level**: Minimum log level to handle (default: Info)
- **DropOldest**: Drop oldest records when queue is full (default: true)
- **WriterCount**: Number of concurrent database writers (default: 2)

## 📊 Status and Progress

### Job Status Flow

```
pending → running → completed
                 ↘ failed
                 ↘ cancelled
```

### Step Status Flow

```
running → completed
       ↘ failed  
       ↘ skipped
```

### Progress Tracking

```go
// Manual progress updates
tracker.MarkJobProgress(ctx, jobID, 75) // 75% complete

// Get progress information
progress, err := tracker.GetJobProgress(ctx, jobID)
if err == nil {
    fmt.Printf("Progress: %d/%d steps (%.1f%%), Runtime: %ds\n",
        progress.CompletedSteps, progress.TotalSteps,
        progress.StepCompletion, progress.RuntimeSeconds)
}
```

## 🔄 Hierarchical Jobs

### Parent-Child Relationships

```go
// Start parent job
parentStart := joblog.JobStart{
    JobType:   "orchestration",
    Operation: "bulk-migration",
    Owner:     stringPtr("automation"),
}
parentCtx, parentJobID, _ := tracker.StartJob(ctx, parentStart)

// Start child job
childStart := joblog.JobStart{
    ParentJobID: &parentJobID,  // Link to parent
    JobType:     "migration",
    Operation:   "migrate-vm1",
    Owner:       stringPtr("automation"),
}
childCtx, childJobID, _ := tracker.StartJob(parentCtx, childStart)

// Child job inherits parent context
logger := tracker.Logger(childCtx) // Includes both job IDs
```

## 🛡️ Error Handling and Recovery

### Panic Recovery

```go
// Automatic panic recovery in RunStep
err := tracker.RunStep(ctx, jobID, "risky-operation", func(stepCtx context.Context) error {
    // This will panic
    panic("something went wrong")
})
// err will contain the panic information
// Step and job status automatically updated to "failed"
```

### Manual Recovery

```go
defer joblog.RecoverAsFailure(ctx, tracker, jobID, &stepID)

// Your risky code here
// Panics are automatically converted to failed status
```

### Error Enrichment

```go
enrichedErr := joblog.ErrorWithContext(ctx, err, "vm-creation", map[string]any{
    "vm_name": "test-vm",
    "zone_id": "zone-1",
})
// Returns: original error with context: [job_id=123, step_id=456, operation=vm-creation, vm_name=test-vm, zone_id=zone-1]
```

## 🌐 Middleware Integration

### HTTP Middleware

```go
// Create HTTP middleware
jobMiddleware := middleware.NewHTTPJobMiddleware(tracker, middleware.DefaultHTTPJobConfig())

// Apply to your router
handler := jobMiddleware.Middleware(mux)

// Each request gets automatic job tracking
// Response includes X-Job-ID header
```

### CLI Wrapper

```go
// Create CLI wrapper
wrapper := middleware.NewCLIJobWrapper(tracker, middleware.DefaultCLIJobConfig())

// Wrap command execution
err := wrapper.ExecuteCommand(
    "migrate-vm",
    map[string]any{"vm_name": "test-vm"},
    func(ctx context.Context) error {
        // Your CLI logic here
        return migrateVM(ctx, tracker)
    },
)
```

## 🔍 Monitoring and Querying

### Database Views

The migration creates helpful views for monitoring:

```sql
-- Active jobs with progress
SELECT * FROM active_jobs;

-- Job progress summary
SELECT * FROM job_progress WHERE job_id = 'your-job-id';
```

### Progress Monitoring

```go
// Get detailed progress
progress, err := tracker.GetJobProgress(ctx, jobID)
if err == nil {
    fmt.Printf("Job: %s\n", progress.JobID)
    fmt.Printf("Steps: %d/%d completed\n", progress.CompletedSteps, progress.TotalSteps)
    fmt.Printf("Progress: %.1f%%\n", progress.StepCompletion)
    fmt.Printf("Runtime: %ds\n", progress.RuntimeSeconds)
}
```

### Log Querying

```go
// Query logs with filters
filter := &joblog.LogFilter{
    JobID:  &jobID,
    Level:  stringPtr("ERROR"),
    Since:  &since,
    Limit:  100,
}

// Custom database queries for complex analysis
```

## 🧪 Testing

### Unit Tests

```go
func TestJobLifecycle(t *testing.T) {
    // Use in-memory database for tests
    db := setupTestDB(t)
    
    tracker := joblog.New(db, slog.NewTextHandler(os.Stdout, nil))
    defer tracker.Close()
    
    ctx := context.Background()
    
    // Test job creation
    jobStart := joblog.JobStart{
        JobType:   "test",
        Operation: "test-operation",
    }
    
    ctx, jobID, err := tracker.StartJob(ctx, jobStart)
    require.NoError(t, err)
    assert.NotEmpty(t, jobID)
    
    // Test step creation
    err = tracker.RunStep(ctx, jobID, "test-step", func(stepCtx context.Context) error {
        logger := tracker.Logger(stepCtx)
        logger.Info("Test step executed")
        return nil
    })
    require.NoError(t, err)
    
    // Test job completion
    err = tracker.EndJob(ctx, jobID, joblog.StatusCompleted, nil)
    require.NoError(t, err)
    
    // Verify job was created and completed
    job, err := tracker.GetJob(ctx, jobID)
    require.NoError(t, err)
    assert.Equal(t, joblog.StatusCompleted, job.Status)
}
```

### Mock Testing

```go
// Use sqlmock for database testing
db, mock, err := sqlmock.New()
require.NoError(t, err)

// Set up expected database calls
mock.ExpectExec("INSERT INTO job_tracking").
    WillReturnResult(sqlmock.NewResult(1, 1))

tracker := joblog.New(db, slog.NewTextHandler(os.Stdout, nil))
// ... test code ...

assert.NoError(t, mock.ExpectationsWereMet())
```

## ⚡ Performance Considerations

### Queue Management

```go
// Monitor queue size
handler := joblog.NewDBHandler(db, config)
fmt.Printf("Queue size: %d/%d\n", handler.GetQueueSize(), handler.GetQueueCapacity())

if handler.IsQueueFull() {
    // Take action if queue is full
}
```

### Batch Processing

The DB handler automatically batches writes for efficiency:
- **Batch Size**: 100 records per transaction
- **Flush Interval**: 1 second maximum
- **Concurrent Writers**: Configurable (default: 2)

### Memory Usage

- Log records are queued in memory before database writes
- Configure `QueueSize` based on your memory constraints
- Use `DropOldest: true` to prevent memory exhaustion

## 🛠️ Advanced Usage

### Custom Handlers

```go
// Create custom log handler
type CustomHandler struct {
    // Your implementation
}

func (h *CustomHandler) Enabled(ctx context.Context, level slog.Level) bool { /* ... */ }
func (h *CustomHandler) Handle(ctx context.Context, record slog.Record) error { /* ... */ }
func (h *CustomHandler) WithAttrs(attrs []slog.Attr) slog.Handler { /* ... */ }
func (h *CustomHandler) WithGroup(name string) slog.Handler { /* ... */ }

// Use with tracker
tracker := joblog.New(db, stdoutHandler, dbHandler, customHandler)
```

### Structured Metadata

```go
// Complex metadata structures
metadata := map[string]any{
    "vm_config": map[string]any{
        "cpu_count": 4,
        "memory_gb": 16,
        "disks": []map[string]any{
            {"size_gb": 100, "type": "ssd"},
            {"size_gb": 500, "type": "hdd"},
        },
    },
    "network": map[string]any{
        "interfaces": []string{"eth0", "eth1"},
        "vlans": []int{100, 200},
    },
}

jobStart := joblog.JobStart{
    JobType:   "migration",
    Operation: "complex-vm-migrate",
    Metadata:  metadata, // Automatically serialized to JSON
}
```

### Context Propagation

```go
// Pass context through service calls
func migrateVM(ctx context.Context) error {
    // Start child operation
    return volumeService.AttachVolume(ctx, volumeID, vmID)
}

func (vs *VolumeService) AttachVolume(ctx context.Context, volumeID, vmID string) error {
    // Get logger with inherited job/step context
    logger := tracker.Logger(ctx)
    logger.Info("Attaching volume",
        slog.String("volume_id", volumeID),
        slog.String("vm_id", vmID),
    )
    // Job and step IDs automatically included in logs
    return nil
}
```

## 🔧 Troubleshooting

### Common Issues

**Database Connection Errors**
```bash
# Check database connectivity
mysql -u root -p -e "SELECT 1"

# Verify migration was applied
mysql -u root -p migratekit -e "SHOW TABLES LIKE 'job_%'"
```

**Queue Full Warnings**
```go
// Monitor queue health
if handler.IsQueueFull() {
    log.Warn("Log queue is full, consider increasing QueueSize or adding more writers")
}
```

**Memory Usage**
```go
// Reduce memory usage
config := &joblog.DBHandlerConfig{
    QueueSize:   1000,  // Smaller queue
    DropOldest:  true,  // Don't block
    WriterCount: 1,     // Fewer goroutines
}
```

### Debug Mode

```go
// Enable debug logging to see internal operations
debugHandler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
    Level: slog.LevelDebug,
})

tracker := joblog.New(db, debugHandler, dbHandler)
```

## 📝 Migration from Existing Systems

### From Old Job Tracking

```go
// Old system
type OldJobTracker struct {
    // Legacy implementation
}

// New system
func migrateToNewSystem(oldTracker *OldJobTracker, newTracker *joblog.Tracker) {
    // Migrate existing jobs
    oldJobs := oldTracker.GetActiveJobs()
    for _, oldJob := range oldJobs {
        jobStart := joblog.JobStart{
            JobType:   oldJob.Type,
            Operation: oldJob.Operation,
            Metadata:  oldJob.Metadata,
        }
        
        // Create new job
        ctx, jobID, err := newTracker.StartJob(context.Background(), jobStart)
        if err != nil {
            continue
        }
        
        // Map old job ID to new job ID
        mappingTable[oldJob.ID] = jobID
    }
}
```

### Integration Points

Replace existing logging calls:

```go
// Old logging
log.WithFields(log.Fields{
    "vm_id": vmID,
    "operation": "migrate",
}).Info("Starting migration")

// New logging
logger := tracker.Logger(ctx)
logger.Info("Starting migration",
    slog.String("vm_id", vmID),
    slog.String("operation", "migrate"),
)
// Job and step context automatically included
```

## 📋 Best Practices

### DO ✅

- **Always use context propagation** for job and step tracking
- **End all jobs and steps** with appropriate status
- **Use structured logging** with meaningful attributes
- **Handle panics** with recovery mechanisms
- **Monitor queue health** in production
- **Use hierarchical jobs** for complex operations

### DON'T ❌

- **Don't forget to close** the tracker and handlers
- **Don't use blocking operations** in log handlers
- **Don't log sensitive information** (passwords, keys)
- **Don't create jobs for simple operations** (single function calls)
- **Don't ignore errors** from job/step operations

### Production Deployment

```go
// Production configuration
config := &joblog.DBHandlerConfig{
    QueueSize:   50000,          // Large buffer
    Level:       slog.LevelInfo, // Info and above
    DropOldest:  true,           // Don't block
    WriterCount: 4,              // More writers
}

// Multiple handlers for redundancy
stdoutHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
fileHandler := slog.NewJSONHandler(logFile, &slog.HandlerOptions{Level: slog.LevelWarn})
dbHandler := joblog.NewDBHandler(db, config)

tracker := joblog.New(db, stdoutHandler, fileHandler, dbHandler)

// Graceful shutdown
c := make(chan os.Signal, 1)
signal.Notify(c, os.Interrupt, syscall.SIGTERM)
go func() {
    <-c
    log.Info("Shutting down...")
    tracker.Close()
    os.Exit(0)
}()
```

## 📚 Examples

See the complete examples in:
- `cmd/example/main.go` - End-to-end usage examples
- `examples/middleware/http_middleware.go` - HTTP integration
- `examples/middleware/cli_middleware.go` - CLI integration

## 🤝 Contributing

1. Follow Go best practices and conventions
2. Add tests for new functionality
3. Update documentation for API changes
4. Ensure database migrations are backward compatible

## 📄 License

This package is part of the MigrateKit project and follows the same licensing terms.
