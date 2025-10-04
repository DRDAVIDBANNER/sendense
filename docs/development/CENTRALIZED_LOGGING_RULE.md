# MANDATORY: Centralized Logging Rule

**Created**: 2025-01-22  
**Status**: **MANDATORY RULE** - No Exceptions  
**Scope**: ALL component updates and operations  
**Priority**: **CRITICAL** - Must be followed in all development

---

## 🚨 **ABSOLUTE PROJECT RULE**

### **MANDATORY REQUIREMENT**
**ALL component updates, operations, and logic changes MUST use the centralized logging system.**

**Location**: `/internal/common/logging/`

**No Exceptions**: This rule applies to **ALL** systems without exception.

---

## 📋 **SCOPE OF APPLICATION**

### **Systems That MUST Use Centralized Logging**
- ✅ **Failover Operations** (live, test, cleanup)
- ✅ **Migration Workflows** (replication, sync, transfer)
- ✅ **Volume Operations** (create, attach, detach, delete)
- ✅ **Network Operations** (mapping, resolution, configuration)
- ✅ **Database Operations** (job tracking, status updates)
- ✅ **Cleanup Services** (test failover cleanup, orphan cleanup)
- ✅ **Validation Services** (pre-failover, network, volume)
- ✅ **API Handlers** (request/response logging)
- ✅ **Background Workers** (polling, monitoring, async operations)
- ✅ **Integration Services** (VMA communication, CloudStack API)

### **Operations That MUST Be Logged**
- Operation start/end with duration
- Step-by-step progress tracking
- Error conditions with full context
- Performance metrics and timing
- State transitions and status changes
- API calls and responses (sanitized)
- Database operations and results
- External service interactions

---

## 🔧 **TECHNICAL REQUIREMENTS**

### **Mandatory Interface Implementation**
All operations MUST implement the `OperationLogger` interface:

```go
type OperationLogger interface {
    StartOperation(operation, jobID string) LogContext
    LogStep(step, message string, fields map[string]interface{})
    LogError(step, message string, err error)
    LogDuration(step string, start time.Time)
    EndOperation(status string, summary map[string]interface{})
}
```

### **Required Logging Elements**
1. **Correlation IDs** - Every operation must have traceable correlation ID
2. **Structured Fields** - All log entries must use structured fields
3. **Step Tracking** - Major operations broken into logged steps
4. **Error Context** - Errors must include full context and stack trace
5. **Performance Metrics** - Duration tracking for all operations
6. **Operation Metadata** - Job IDs, VM IDs, request parameters

### **Prohibited Logging Practices**
❌ **NEVER ALLOWED**:
- Direct `logrus` logging in operation logic
- Unstructured log messages
- Logging without correlation IDs
- Silent failures without logging
- Performance-sensitive operations without timing
- Error conditions without context

---

## 📖 **USAGE PATTERNS**

### **Basic Operation Logging**
```go
// Required import
import "github.com/vexxhost/migratekit/internal/common/logging"

// Start operation with correlation
logger := logging.NewOperationLogger("test-failover", jobID)
opCtx := logger.StartOperation("enhanced-test-failover", jobID)

// Log major steps
stepCtx := opCtx.StartStep("linstor-snapshot")
stepCtx.Info("Creating Linstor snapshot for rollback protection", logrus.Fields{
    "vm_id": vmID,
    "volume_uuid": volumeUUID,
})

// Log errors with context
if err != nil {
    stepCtx.Error("Snapshot creation failed", err, logrus.Fields{
        "linstor_config": config.Name,
        "api_url": config.APIURL,
    })
    return fmt.Errorf("snapshot creation failed: %w", err)
}

// Log successful completion
stepCtx.Success("Snapshot created successfully", logrus.Fields{
    "snapshot_name": snapshotName,
    "duration_ms": time.Since(start).Milliseconds(),
})
stepCtx.EndStep("completed")

// End operation
opCtx.EndOperation("completed", logrus.Fields{
    "total_duration": time.Since(operationStart),
    "steps_completed": 6,
})
```

### **Error Handling with Logging**
```go
// Proper error logging with context
if err != nil {
    logger.LogError("vm-creation", "Failed to create test VM", err, logrus.Fields{
        "vm_name": vmName,
        "service_offering": serviceOfferingID,
        "network_mappings": networkMappings,
        "cloudstack_error_code": extractErrorCode(err),
    })
    
    // Update job status with error context
    jobCtx.FailOperation("vm-creation-failed", map[string]interface{}{
        "error_message": err.Error(),
        "failure_step": "vm-creation",
        "retry_possible": isRetryableError(err),
    })
    
    return fmt.Errorf("VM creation failed: %w", err)
}
```

### **Performance Monitoring**
```go
// Required for all operations
operationStart := time.Now()
stepStart := time.Now()

// ... operation logic ...

// Log step duration
logger.LogDuration("linstor-snapshot", stepStart)

// Log total operation duration  
logger.LogDuration("complete-operation", operationStart)
```

---

## 🔍 **CORRELATION AND TRACING**

### **Correlation ID Requirements**
- Every operation MUST have a unique correlation ID
- Correlation IDs MUST be propagated to all sub-operations
- Parent/child operation relationships MUST be tracked
- Cross-service calls MUST include correlation ID

### **Tracing Requirements**
```go
// Parent operation
parentCtx := logger.StartOperation("test-failover", jobID)

// Child operations inherit correlation
childCtx := parentCtx.StartChildOperation("volume-operations")
childCtx.LogStep("detach-volume", "Detaching volume from OMA")

// External service calls include correlation
volumeClient.DetachVolume(ctx, volumeID, 
    WithCorrelationID(parentCtx.CorrelationID))
```

---

## 📊 **MONITORING AND METRICS**

### **Required Metrics Collection**
- Operation success/failure rates
- Operation duration percentiles 
- Error frequency by type
- Step completion times
- Resource utilization during operations

### **Log Aggregation Requirements**
- All logs MUST be structured for parsing
- Correlation IDs MUST enable end-to-end tracing
- Error logs MUST include classification
- Performance logs MUST include timing data

---

## ✅ **COMPLIANCE VALIDATION**

### **Development Checklist**
Before any component update:
- [ ] Centralized logging interface implemented
- [ ] All major steps logged with context
- [ ] Error conditions properly logged
- [ ] Performance metrics collected
- [ ] Correlation IDs properly propagated
- [ ] No direct logrus usage in operation logic

### **Code Review Requirements**
- Centralized logging usage MUST be verified
- Correlation ID propagation MUST be confirmed
- Error logging context MUST be comprehensive
- Performance logging MUST be present

### **Testing Requirements**
- Log output MUST be verified in tests
- Correlation ID flow MUST be tested
- Error logging MUST be tested
- Performance metrics MUST be validated

---

## 🚨 **ENFORCEMENT**

### **Mandatory Compliance**
- This rule is **NON-NEGOTIABLE**
- All future code MUST comply
- All refactoring MUST add centralized logging
- Code reviews MUST verify compliance

### **AI Assistant Compliance**
- AI assistants MUST follow this rule in ALL code generation
- AI assistants MUST update existing code to use centralized logging
- AI assistants MUST refuse to create operation logic without centralized logging
- AI assistants MUST document centralized logging usage

---

## 📚 **DOCUMENTATION REQUIREMENTS**

### **Implementation Documentation**
- Every component using centralized logging MUST document its usage
- Log field specifications MUST be documented
- Error handling patterns MUST be documented
- Performance metrics MUST be documented

### **Integration Guides**
- New components MUST include centralized logging integration guide
- Existing components MUST be updated with logging documentation
- Troubleshooting guides MUST reference log correlation

---

**🎯 BOTTOM LINE**: Centralized logging is NOT optional. It is a fundamental requirement for ALL operation logic in the MigrateKit OSSEA project. No exceptions, no compromises, no direct logrus usage in operation code.

