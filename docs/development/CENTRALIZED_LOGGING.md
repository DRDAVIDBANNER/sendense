# Centralized Logging System

**Created**: 2025-01-22  
**Status**: **PRODUCTION READY**  
**Priority**: **MANDATORY** - Required for all operations  
**Location**: `/internal/common/logging/`

---

## 🎯 **OVERVIEW**

The centralized logging system provides consistent, structured logging for all MigrateKit OSSEA operations. It ensures proper correlation tracking, step-by-step operation monitoring, and comprehensive error context preservation.

**MANDATORY RULE**: ALL component updates and operations MUST use this centralized logging system.

---

## 🏗️ **ARCHITECTURE**

### **Core Components**

#### **OperationLogger Interface**
Main entry point for all operation logging:
```go
type OperationLogger interface {
    StartOperation(operation, jobID string) OperationContext
    GetCorrelationID() string
    GetJobID() string
}
```

#### **OperationContext Interface**
Provides context for entire operations:
```go
type OperationContext interface {
    LogStep(step, message string, fields log.Fields)
    LogError(step, message string, err error, fields log.Fields)
    LogDuration(step string, start time.Time)
    LogSuccess(step, message string, fields log.Fields)
    StartStep(stepName string) StepContext
    EndOperation(status string, summary log.Fields)
    CreateChildContext(childOperation string) OperationContext
}
```

#### **StepContext Interface**
Provides detailed context for individual steps:
```go
type StepContext interface {
    Info(message string, fields log.Fields)
    Warn(message string, fields log.Fields)
    Error(message string, err error, fields log.Fields)
    Success(message string, fields log.Fields)
    EndStep(status string, fields log.Fields)
    LogDuration(start time.Time)
}
```

---

## 🔧 **IMPLEMENTATION GUIDE**

### **Basic Usage Pattern**

#### **1. Initialize Operation Logger**
```go
import "github.com/vexxhost/migratekit/internal/common/logging"

// For new operations
logger := logging.NewOperationLogger(jobID)

// For child operations (inherits correlation ID)
logger := logging.NewOperationLoggerWithCorrelation(jobID, parentCorrelationID)
```

#### **2. Start Operation**
```go
opCtx := logger.StartOperation("enhanced-test-failover", jobID)
defer opCtx.EndOperation("completed", log.Fields{
    "total_steps": 6,
    "vm_id": vmID,
})
```

#### **3. Log Major Steps**
```go
// Simple step logging
opCtx.LogStep("linstor-snapshot", "Creating snapshot for rollback protection", log.Fields{
    "vm_id": vmID,
    "volume_uuid": volumeUUID,
})

// Detailed step with context
stepCtx := opCtx.StartStep("virtio-injection")
stepCtx.Info("Starting VirtIO driver injection", log.Fields{
    "device_path": devicePath,
    "script_path": "/opt/migratekit/bin/inject-virtio-drivers.sh",
})

// ... operation logic ...

stepCtx.Success("VirtIO drivers injected successfully", log.Fields{
    "injection_duration": duration.String(),
    "drivers_installed": ["viostor", "netkvm", "vioscsi"],
})
stepCtx.EndStep("completed", nil)
```

#### **4. Error Handling**
```go
if err != nil {
    opCtx.LogError("vm-creation", "Failed to create test VM", err, log.Fields{
        "vm_name": vmName,
        "service_offering": serviceOfferingID,
        "cloudstack_error": extractErrorCode(err),
    })
    
    opCtx.EndOperation("failed", log.Fields{
        "failure_step": "vm-creation",
        "error_type": "cloudstack_api_error",
    })
    
    return fmt.Errorf("VM creation failed: %w", err)
}
```

#### **5. Performance Monitoring**
```go
// Step timing
stepStart := time.Now()
// ... step logic ...
opCtx.LogDuration("linstor-snapshot", stepStart)

// Or with step context
stepCtx := opCtx.StartStep("volume-attachment")
stepStart := time.Now()
// ... attachment logic ...
stepCtx.LogDuration(stepStart)
stepCtx.EndStep("completed", nil)
```

---

## 📊 **LOGGING PATTERNS**

### **Enhanced Test Failover Example**
```go
func (etfe *EnhancedTestFailoverEngine) ExecuteEnhancedTestFailover(
    ctx context.Context,
    request *EnhancedTestFailoverRequest,
) (*EnhancedTestFailoverResult, error) {
    
    // Initialize centralized logging
    logger := logging.NewOperationLogger(request.FailoverJobID)
    opCtx := logger.StartOperation("enhanced-test-failover", request.FailoverJobID)
    
    opCtx.LogStep("initialization", "Starting enhanced test failover", log.Fields{
        "vm_id": request.VMID,
        "vm_name": request.VMName,
        "auto_cleanup": request.AutoCleanup,
        "skip_validation": request.SkipValidation,
        "skip_snapshot": request.SkipSnapshot,
        "skip_virtio": request.SkipVirtIOInjection,
    })
    
    // Execute steps with detailed logging
    for i, step := range steps {
        stepCtx := opCtx.StartStep(step.Name)
        stepCtx.Info(step.Description, log.Fields{
            "step_number": i + 1,
            "total_steps": len(steps),
        })
        
        stepStart := time.Now()
        
        switch step.Name {
        case "linstor-snapshot":
            snapshotName, err := etfe.executeSnapshotStep(ctx, stepCtx, request)
            if err != nil {
                stepCtx.Error("Snapshot creation failed", err, log.Fields{
                    "linstor_config": request.LinstorConfigID,
                })
                stepCtx.EndStep("failed", nil)
                opCtx.EndOperation("failed", log.Fields{
                    "failure_step": step.Name,
                })
                return nil, fmt.Errorf("snapshot step failed: %w", err)
            }
            
            stepCtx.Success("Snapshot created successfully", log.Fields{
                "snapshot_name": snapshotName,
            })
            
        case "virtio-injection":
            status, err := etfe.executeVirtIOStep(ctx, stepCtx, request)
            if err != nil {
                stepCtx.Error("VirtIO injection failed", err, log.Fields{
                    "device_path": devicePath,
                })
                stepCtx.EndStep("failed", nil)
                return nil, fmt.Errorf("VirtIO injection failed: %w", err)
            }
            
            stepCtx.Success("VirtIO drivers injected", log.Fields{
                "injection_status": status,
            })
        }
        
        stepCtx.LogDuration(stepStart)
        stepCtx.EndStep("completed", nil)
    }
    
    opCtx.EndOperation("completed", log.Fields{
        "destination_vm_id": result.DestinationVMID,
        "snapshot_name": result.LinstorSnapshotName,
        "total_steps_completed": len(steps),
    })
    
    return result, nil
}
```

### **Child Operation Example**
```go
func (etfe *EnhancedTestFailoverEngine) executeSnapshotStep(
    ctx context.Context,
    parentCtx logging.OperationContext,
    request *EnhancedTestFailoverRequest,
) (string, error) {
    
    // Create child context for snapshot operations
    childCtx := parentCtx.CreateChildContext("linstor-snapshot-creation")
    
    childCtx.LogStep("volume-discovery", "Finding volume UUID for VM", log.Fields{
        "vm_id": request.VMID,
    })
    
    volumeUUID, err := etfe.getVolumeUUIDForVM(request.VMID)
    if err != nil {
        childCtx.LogError("volume-discovery", "Failed to find volume UUID", err, log.Fields{
            "vm_id": request.VMID,
        })
        childCtx.EndOperation("failed", nil)
        return "", fmt.Errorf("volume discovery failed: %w", err)
    }
    
    childCtx.LogSuccess("volume-discovery", "Found volume UUID", log.Fields{
        "volume_uuid": volumeUUID,
    })
    
    // ... continue with snapshot creation ...
    
    childCtx.EndOperation("completed", log.Fields{
        "snapshot_name": snapshotName,
        "volume_uuid": volumeUUID,
    })
    
    return snapshotName, nil
}
```

---

## 🔗 **CORRELATION TRACKING**

### **Correlation ID Flow**
```
Operation Start (generate UUID)
├── Step 1 (inherit correlation ID)
├── Step 2 (inherit correlation ID)
│   ├── Child Operation A (inherit correlation ID)
│   └── Child Operation B (inherit correlation ID)
├── External Service Call (pass correlation ID)
└── Operation End (correlation ID in summary)
```

### **Cross-Service Correlation**
```go
// In failover engine
opCtx := logger.StartOperation("test-failover", jobID)

// Pass correlation to Volume Daemon
volumeClient.DetachVolume(
    logging.WithContext(ctx, opCtx.GetCorrelationID()),
    volumeID,
)

// Pass correlation to cleanup service
cleanupService.ExecuteCleanup(
    logging.WithContext(ctx, opCtx.GetCorrelationID()),
    vmName,
)
```

---

## 📋 **STRUCTURED FIELDS**

### **Required Fields**
Every log entry includes:
- `correlation_id` - Unique operation correlation ID
- `job_id` - Job identifier
- `operation` - High-level operation name
- `step` - Current step name
- `event` - Event type (operation_start, step_info, step_error, etc.)

### **Common Field Examples**
```go
// VM-related fields
log.Fields{
    "vm_id": "4205a841-0265-f4bd-39a6-39fd92196f53",
    "vm_name": "PGWINTESTBIOS",
    "vm_state": "running",
}

// Volume-related fields
log.Fields{
    "volume_uuid": "babe9a3f-2746-4bfa-86cb-36830e0cd399",
    "volume_id": "12345",
    "device_path": "/dev/vdc",
    "volume_size_gb": 110,
}

// Error fields
log.Fields{
    "error_type": "cloudstack_api_error",
    "error_code": "431",
    "retry_possible": true,
    "recovery_suggestion": "Check CloudStack API credentials",
}

// Performance fields
log.Fields{
    "duration_ms": 5420,
    "duration_str": "5.42s",
    "operation_size_gb": 110,
    "throughput_mbps": 204.8,
}
```

---

## 🚨 **ERROR HANDLING PATTERNS**

### **Comprehensive Error Logging**
```go
if err != nil {
    // Extract error details
    errorDetails := log.Fields{
        "error_type": classifyError(err),
        "retry_possible": isRetryableError(err),
        "cloudstack_error_code": extractCloudStackError(err),
        "underlying_error": getUnderlyingError(err),
    }
    
    // Log with full context
    stepCtx.Error("Operation failed with detailed context", err, errorDetails)
    
    // Update operation status
    stepCtx.EndStep("failed", errorDetails)
    
    // Propagate error
    return fmt.Errorf("operation failed: %w", err)
}
```

### **Recovery and Retry Logging**
```go
for attempt := 1; attempt <= maxRetries; attempt++ {
    attemptCtx := opCtx.StartStep(fmt.Sprintf("attempt-%d", attempt))
    
    err := executeOperation()
    if err == nil {
        attemptCtx.Success("Operation succeeded", log.Fields{
            "attempt_number": attempt,
        })
        attemptCtx.EndStep("completed", nil)
        break
    }
    
    if isRetryableError(err) && attempt < maxRetries {
        attemptCtx.Warn("Operation failed, retrying", log.Fields{
            "attempt_number": attempt,
            "max_retries": maxRetries,
            "retry_delay": retryDelay.String(),
            "error": err.Error(),
        })
        attemptCtx.EndStep("retrying", nil)
        time.Sleep(retryDelay)
        continue
    }
    
    // Final failure
    attemptCtx.Error("Operation failed permanently", err, log.Fields{
        "attempt_number": attempt,
        "max_retries": maxRetries,
    })
    attemptCtx.EndStep("failed", nil)
    return fmt.Errorf("operation failed after %d attempts: %w", attempt, err)
}
```

---

## 📊 **MONITORING AND METRICS**

### **Log Aggregation Queries**

#### **Operation Success Rate**
```json
{
  "query": {
    "bool": {
      "must": [
        {"term": {"event": "operation_end"}},
        {"range": {"@timestamp": {"gte": "now-1h"}}}
      ]
    }
  },
  "aggs": {
    "success_rate": {
      "terms": {"field": "status"},
      "aggs": {
        "by_operation": {
          "terms": {"field": "operation"}
        }
      }
    }
  }
}
```

#### **Performance Metrics**
```json
{
  "query": {
    "bool": {
      "must": [
        {"term": {"event": "operation_end"}},
        {"term": {"status": "completed"}}
      ]
    }
  },
  "aggs": {
    "performance": {
      "terms": {"field": "operation"},
      "aggs": {
        "avg_duration": {"avg": {"field": "total_duration_ms"}},
        "p95_duration": {"percentiles": {"field": "total_duration_ms", "percents": [95]}},
        "p99_duration": {"percentiles": {"field": "total_duration_ms", "percents": [99]}}
      }
    }
  }
}
```

#### **Error Analysis**
```json
{
  "query": {
    "bool": {
      "must": [
        {"term": {"event": "step_error"}},
        {"range": {"@timestamp": {"gte": "now-24h"}}}
      ]
    }
  },
  "aggs": {
    "error_patterns": {
      "terms": {"field": "error_type"},
      "aggs": {
        "by_step": {"terms": {"field": "step"}},
        "by_operation": {"terms": {"field": "operation"}}
      }
    }
  }
}
```

---

## 🔧 **INTEGRATION EXAMPLES**

### **Volume Daemon Integration**
```go
// In Volume Daemon operations
func (vs *VolumeService) AttachVolume(ctx context.Context, volumeID, vmID string) error {
    correlationID := logging.GetCorrelationFromContext(ctx)
    logger := logging.NewOperationLoggerWithCorrelation("volume-attach", correlationID)
    opCtx := logger.StartOperation("volume-attach", volumeID)
    
    opCtx.LogStep("cloudstack-api", "Calling CloudStack attach volume", log.Fields{
        "volume_id": volumeID,
        "vm_id": vmID,
    })
    
    // ... CloudStack API call ...
    
    opCtx.LogStep("device-correlation", "Correlating device path", log.Fields{
        "expected_device": expectedDevice,
    })
    
    // ... device correlation logic ...
    
    opCtx.EndOperation("completed", log.Fields{
        "device_path": actualDevicePath,
        "attach_duration_ms": duration.Milliseconds(),
    })
    
    return nil
}
```

### **Cleanup Service Integration**
```go
// In cleanup service
func (cs *CleanupService) ExecuteTestFailoverCleanup(ctx context.Context, vmName string) error {
    correlationID := logging.GetCorrelationFromContext(ctx)
    logger := logging.NewOperationLoggerWithCorrelation("cleanup", correlationID)
    opCtx := logger.StartOperation("test-failover-cleanup", vmName)
    
    // Create child contexts for major cleanup steps
    vmCleanupCtx := opCtx.CreateChildContext("vm-cleanup")
    volumeCleanupCtx := opCtx.CreateChildContext("volume-cleanup")
    snapshotCleanupCtx := opCtx.CreateChildContext("snapshot-cleanup")
    
    // ... cleanup logic with detailed logging ...
    
    opCtx.EndOperation("completed", log.Fields{
        "vm_deleted": true,
        "volumes_reattached": volumeCount,
        "snapshots_rolled_back": snapshotCount,
    })
    
    return nil
}
```

---

## 📚 **TROUBLESHOOTING**

### **Common Issues**

#### **Missing Correlation IDs**
**Symptom**: Logs not correlating across services
**Solution**: Ensure correlation ID is passed in context:
```go
// Correct: Pass correlation in context
ctx = logging.WithContext(ctx, opCtx.GetCorrelationID())
service.CallExternalAPI(ctx, params)

// Incorrect: No correlation passed
service.CallExternalAPI(context.Background(), params)
```

#### **Performance Impact**
**Symptom**: Logging affecting operation performance
**Solution**: Use structured fields efficiently:
```go
// Efficient: Pre-build fields
baseFields := log.Fields{
    "vm_id": vmID,
    "operation": operation,
}

// Add specific fields as needed
fields := baseFields
fields["step_specific"] = value
stepCtx.Info("Message", fields)
```

#### **Log Volume Management**
**Symptom**: Too many log entries
**Solution**: Use appropriate log levels:
```go
// Use Info for major steps
stepCtx.Info("Major operation step", fields)

// Use Success for completion
stepCtx.Success("Operation completed", fields)

// Avoid Debug in production critical path
// Debug logs should be used sparingly
```

---

## 🎯 **BEST PRACTICES**

### **DO**
- ✅ Always use centralized logging for operations
- ✅ Include correlation IDs in all service calls
- ✅ Log major steps with timing information
- ✅ Use structured fields for searchability
- ✅ Include error context and recovery information
- ✅ End operations with summary information

### **DON'T**
- ❌ Use direct logrus logging in operation code
- ❌ Log sensitive information (passwords, keys)
- ❌ Create logs without correlation IDs
- ❌ Skip error logging for debugging
- ❌ Log without structured fields
- ❌ Forget to end operations and steps

### **Performance Considerations**
- Log fields are evaluated at logging time
- Use string formatting only when necessary
- Pre-build common field sets
- Avoid expensive field calculations in hot paths
- Consider log level impact on performance

---

**🎯 BOTTOM LINE**: Centralized logging provides the foundation for operational visibility, debugging, and monitoring in MigrateKit OSSEA. Proper usage ensures maintainable, traceable, and debuggable operations across all system components.

