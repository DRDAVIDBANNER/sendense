# Job Recovery System Implementation Plan

## Overview
Comprehensive job recovery system to handle orphaned and stuck replication jobs, providing enterprise-grade operational reliability.

## Problem Statement
Current system lacks automatic recovery when:
- OMA API restarts (jobs stuck in 'replicating' status)
- VMA API restarts (loses track of migratekit processes)
- migratekit processes die (jobs remain 'replicating' indefinitely)
- Network communication fails (polling stops permanently)

## Components

### 1. OMA Startup Job Recovery
**Location**: `services/job_recovery_simple.go`
**Purpose**: Detect and recover orphaned jobs during OMA API startup

**Key Functions**:
```go
func RecoverOrphanedJobs(ctx context.Context) error
func recoverJob(job *database.ReplicationJob) error  
func GetOrphanedJobStatus() ([]map[string]interface{}, error)
```

**Recovery Logic**:
- Find jobs with status='replicating' older than 30 minutes
- Mark as 'failed' with reason "startup recovery"
- Update VM context to 'ready_for_failover'
- Enable new replication operations

### 2. Manual Recovery CLI Tool
**Location**: `cmd/job-recovery/main.go`
**Purpose**: Manual job recovery for operators

**Usage**:
```bash
# Scan for orphaned jobs
./job-recovery -action=scan

# Recover orphaned jobs (dry run)
./job-recovery -action=recover -dry-run=true

# Actually recover orphaned jobs  
./job-recovery -action=recover

# Show status
./job-recovery -action=status
```

### 3. VMA Process Health Monitor
**Location**: `vma/services/process_health_monitor.go`
**Purpose**: Track migratekit process health and detect failures

**Key Features**:
- Track all migratekit processes with PID monitoring
- Detect when processes die unexpectedly
- Notify OMA API about job failures
- Automatic cleanup of dead process tracking

### 4. Enhanced VMA Progress Poller
**Enhancement**: Add restart recovery to existing VMAProgressPoller
**Purpose**: Resume polling for active jobs after VMA restart

**Key Enhancement**:
```go
func RecoverActiveJobsOnStartup(ctx context.Context) error {
    // Query database for jobs with status='replicating'
    // Check if VMA API knows about each job
    // Resume polling or mark as failed
}
```

## Integration Points

### OMA API Startup
```go
// Add to OMA API initialization
func InitializeOMAAPI() {
    // ... existing initialization ...
    
    // NEW: Job recovery on startup
    jobRecovery := services.NewSimpleJobRecovery(db)
    if err := jobRecovery.RecoverOrphanedJobs(context.Background()); err != nil {
        log.WithError(err).Warn("Job recovery failed during startup")
    }
}
```

### VMA API Startup  
```go
// Add to VMA API initialization
func InitializeVMAAPI() {
    // ... existing initialization ...
    
    // NEW: Process health monitoring
    processMonitor := services.NewProcessHealthMonitor(omaNotifier)
    if err := processMonitor.Start(context.Background()); err != nil {
        log.WithError(err).Warn("Process health monitoring failed to start")
    }
}
```

### Migratekit Process Tracking
```go
// Enhance VMA migratekit startup
func StartMigratekit(jobID, command string) (int, error) {
    cmd := exec.Command(command)
    if err := cmd.Start(); err != nil {
        return 0, err
    }
    
    // NEW: Track process for health monitoring
    processMonitor.TrackMigratekitProcess(jobID, cmd.Process.Pid, command)
    
    return cmd.Process.Pid, nil
}
```

## Deployment Strategy

### Phase 1: Manual Recovery Tool
1. Build job-recovery CLI tool
2. Test with current stuck jobs
3. Validate recovery functionality

### Phase 2: OMA Startup Recovery
1. Deploy enhanced OMA API with startup job recovery
2. Test OMA restart scenarios
3. Validate automatic orphaned job detection

### Phase 3: VMA Process Monitoring
1. Deploy enhanced VMA API with process health monitoring
2. Test migratekit process failure scenarios
3. Validate automatic job failure notification

### Phase 4: Enhanced Progress Poller
1. Deploy resilient VMA progress poller
2. Test restart recovery scenarios
3. Validate polling resumption

## Benefits

### Operational Reliability
- ✅ **Automatic orphaned job detection** during service restarts
- ✅ **Process death detection** and notification
- ✅ **Job status recovery** without manual intervention
- ✅ **VM context cleanup** to enable new operations

### Enterprise Readiness
- ✅ **Service restart resilience** - no stuck jobs after restarts
- ✅ **Process failure handling** - automatic detection and recovery
- ✅ **Operational visibility** - clear status and recovery reporting
- ✅ **Manual override capabilities** - operator tools for edge cases

### Production Deployment
- ✅ **Non-disruptive enhancement** - doesn't affect running jobs
- ✅ **Backward compatible** - preserves existing workflows
- ✅ **Incremental deployment** - can be deployed component by component
- ✅ **Monitoring integration** - works with existing logging and monitoring

## Testing Scenarios

### Test 1: OMA API Restart Recovery
1. Start replication job
2. Restart OMA API mid-job
3. Verify job is detected as orphaned and recovered
4. Verify VM context updated to allow new operations

### Test 2: VMA API Restart Recovery  
1. Start replication job
2. Restart VMA API mid-job
3. Verify process health monitor detects failure
4. Verify OMA notified and job marked as failed

### Test 3: migratekit Process Death
1. Start replication job
2. Kill migratekit process manually
3. Verify VMA process monitor detects death
4. Verify automatic job failure notification

This comprehensive job recovery system provides enterprise-grade operational reliability for the MigrateKit OSSEA platform.








