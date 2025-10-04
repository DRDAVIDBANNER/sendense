# Scheduler Workflow Alignment Analysis

**Date**: September 19, 2025  
**Purpose**: Identify required changes to align scheduler with documented GUI workflow  
**Status**: 🚨 **CRITICAL GAPS IDENTIFIED**

---

## 🔍 **CURRENT SCHEDULER IMPLEMENTATION ANALYSIS**

### **❌ MAJOR WORKFLOW VIOLATION: Missing Fresh VM Discovery**

**Current Scheduler Process** (`scheduler_service.go:564-627`):
```go
// ❌ WRONG: Uses stale VM context data from database
vmInfo := models.VMInfo{
    ID:         vmCtx.VMwareVMID,        // From vm_replication_contexts table
    Name:       vmCtx.VMName,            // From vm_replication_contexts table  
    Path:       vmCtx.VMPath,            // From vm_replication_contexts table
    Datacenter: vmCtx.Datacenter,       // From vm_replication_contexts table
    CPUs:       valueOrDefault(vmCtx.CPUCount, 2),    // Potentially stale data!
    MemoryMB:   valueOrDefault(vmCtx.MemoryMB, 2048), // Potentially stale data!
    PowerState: stringPtrToString(vmCtx.PowerState),  // Potentially stale data!
    OSType:     stringPtrToString(vmCtx.OSType),      // Potentially stale data!
    Disks:      vmDisks,  // From vm_disks table - STALE DATA!
}

// ❌ CRITICAL: Directly calls Migration Engine with stale data
result, err := s.migrationEngine.StartMigration(ctx, migrationReq)
```

**✅ Required GUI Process** (from `GUI_REPLICATION_WORKFLOW.md`):
```typescript
// ✅ CORRECT: Always refresh VM data from vCenter before job creation
const discoveryResponse = await fetch('/api/discover', {
    body: JSON.stringify({
        vcenter: 'quad-vcenter-01.quadris.local',
        filter: selectedVM  // Get fresh VM specifications
    })
});

const discoveredVM = discoveryData.vms?.find((vm: any) => vm.name === selectedVM);

// ✅ Use fresh discovery data
const vmInfo = {
    cpus: discoveredVM.num_cpu || discoveredVM.cpus || 2,     // Fresh from vCenter
    memory_mb: discoveredVM.memory_mb || 4096,               // Fresh from vCenter
    disks: discoveredVM.disks,                               // Fresh disk specs
    networks: discoveredVM.networks                          // Fresh network config
};
```

---

## 🚨 **CRITICAL PROBLEMS WITH CURRENT IMPLEMENTATION**

### **1. Stale VM Specifications**
- **Issue**: Uses old CPU/memory/disk data from `vm_replication_contexts`
- **Risk**: Migration fails due to outdated VM specifications
- **Fix Required**: Always call VMA discovery API before job creation

### **2. Missing Disk Updates**  
- **Issue**: Uses `vmDisks` from `vm_disks` table (potentially stale)
- **Risk**: Wrong disk sizes, missing disks, outdated VMDK paths
- **Fix Required**: Get fresh disk specifications from vCenter

### **3. No Network Information**
- **Issue**: `// TODO: Add network information if available` (line 599)
- **Risk**: Network configuration not included in migration
- **Fix Required**: Include fresh network data from discovery

### **4. Bypasses Proven Workflow**
- **Issue**: Directly calls `migrationEngine.StartMigration()` 
- **Risk**: Misses field validation and transformation logic
- **Fix Required**: Use same API path as GUI (`/api/v1/replications`)

---

## 📋 **REQUIRED CHANGES FOR ALIGNMENT**

### **🔧 Change 1: Add VMA Discovery Integration**

**Current Missing**: No VMA discovery call  
**Required**: Add VMA discovery service integration

```go
// NEW: Add VMA discovery service to SchedulerService
type SchedulerService struct {
    // ... existing fields ...
    vmaAPIEndpoint string
    vmaClient      *http.Client
}

// NEW: Add fresh VM discovery method
func (s *SchedulerService) discoverVMFromVMA(ctx context.Context, vmName string) (*VMDiscoveryData, error) {
    discoveryRequest := VMDiscoveryRequest{
        VCenter:    "quad-vcenter-01.quadris.local",
        Username:   "administrator@vsphere.local", 
        Password:   "EmyGVoBFesGQc47-",
        Datacenter: "DatabanxDC",
        Filter:     vmName,
    }
    
    // Call VMA discovery API (same as GUI)
    resp, err := s.vmaClient.Post(s.vmaAPIEndpoint+"/api/v1/discover", ...)
    // Parse and validate response
    return discoveryData, nil
}
```

### **🔧 Change 2: Replace createReplicationJob Method**

**Current**: `createReplicationJob()` uses stale database data  
**Required**: Replace with fresh discovery + API call pattern

```go
// REPLACE: Current createReplicationJob method
func (s *SchedulerService) createReplicationJob(
    ctx context.Context,
    execution *database.ScheduleExecution,
    group *database.VMMachineGroup,
    vmCtx *database.VMReplicationContext,
    schedule *database.ReplicationSchedule,
) (string, error) {
    logger := s.jobTracker.Logger(ctx)
    
    // ✅ STEP 1: Fresh VM Discovery (CRITICAL ADDITION)
    discoveryData, err := s.discoverVMFromVMA(ctx, vmCtx.VMName)
    if err != nil {
        return "", fmt.Errorf("discovery failed: %w", err)
    }
    
    // Validate discovery data
    discoveredVM := discoveryData.VMs[0] // Should be filtered to single VM
    if len(discoveredVM.Disks) == 0 {
        return "", fmt.Errorf("VM %s has no disks configured", vmCtx.VMName)
    }
    
    // ✅ STEP 2: Use OMA API (same as GUI) instead of direct Migration Engine
    omaRequest := CreateMigrationRequest{
        SourceVM: models.VMInfo{
            // ✅ EXACT FIELD MAPPING (from GUI workflow)
            ID:         discoveredVM.ID,
            Name:       discoveredVM.Name,
            Path:       discoveredVM.Path,
            Datacenter: discoveredVM.Datacenter,
            VCenterHost: vmCtx.VCenterHost,
            CPUs:       discoveredVM.NumCPU,      // ✅ Fresh from vCenter
            MemoryMB:   discoveredVM.MemoryMB,   // ✅ Fresh from vCenter  
            PowerState: discoveredVM.PowerState, // ✅ Fresh from vCenter
            OSType:     discoveredVM.GuestOS,    // ✅ Fresh from vCenter
            Disks:      discoveredVM.Disks,      // ✅ CRITICAL: Fresh disk specs
            Networks:   discoveredVM.Networks,   // ✅ Fresh network config
        },
        OSSEAConfigID:   1,
        ReplicationType: schedule.ReplicationType,
        VCenterHost:     vmCtx.VCenterHost,
        Datacenter:      vmCtx.Datacenter,
    }
    
    // ✅ STEP 3: Call OMA API (same path as GUI)
    result, err := s.callOMAReplicationAPI(ctx, omaRequest)
    if err != nil {
        return "", fmt.Errorf("failed to call OMA replication API: %w", err)
    }
    
    return result.JobID, nil
}
```

### **🔧 Change 3: Add OMA API Client Method**

**Current Missing**: No OMA API client for replication  
**Required**: Add method to call same API as GUI

```go
// NEW: Add OMA API client method
func (s *SchedulerService) callOMAReplicationAPI(ctx context.Context, req CreateMigrationRequest) (*MigrationResult, error) {
    // Call same API endpoint as GUI: http://localhost:8082/api/v1/replications
    jsonData, err := json.Marshal(req)
    if err != nil {
        return nil, fmt.Errorf("failed to marshal request: %w", err)
    }
    
    httpReq, err := http.NewRequestWithContext(ctx, "POST", "http://localhost:8082/api/v1/replications", bytes.NewBuffer(jsonData))
    if err != nil {
        return nil, fmt.Errorf("failed to create request: %w", err)
    }
    
    httpReq.Header.Set("Content-Type", "application/json")
    httpReq.Header.Set("Authorization", "Bearer sess_longlived_dev_token_2025_2035_permanent")
    
    resp, err := s.httpClient.Do(httpReq)
    if err != nil {
        return nil, fmt.Errorf("failed to call OMA API: %w", err)
    }
    defer resp.Body.Close()
    
    var result MigrationResult
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, fmt.Errorf("failed to decode response: %w", err)
    }
    
    return &result, nil
}
```

### **🔧 Change 4: Update Service Constructor**

**Current**: Missing VMA integration  
**Required**: Add VMA API endpoint and HTTP client

```go
// UPDATE: NewSchedulerService constructor
func NewSchedulerService(
    repository *database.SchedulerRepository,
    replicationRepo *database.ReplicationJobRepository,
    jobTracker *joblog.Tracker,
    vmaAPIEndpoint string,  // ✅ ADD: VMA API endpoint
) *SchedulerService {
    return &SchedulerService{
        // ... existing fields ...
        vmaAPIEndpoint: vmaAPIEndpoint,
        vmaClient:      &http.Client{Timeout: 30 * time.Second},
    }
}
```

---

## 🎯 **IMPLEMENTATION PRIORITY ORDER**

### **Phase 1: Critical Workflow Fix** 
1. **Add VMA discovery integration** - Most critical missing piece
2. **Replace direct Migration Engine calls** - Use OMA API like GUI
3. **Update service constructor** - Add VMA endpoint configuration
4. **Test with existing schedules** - Verify alignment with GUI behavior

### **Phase 2: Enhancement**  
1. **Add network information support** - Complete VM specification coverage
2. **Add discovery error handling** - Timeout, retry, fallback logic
3. **Add discovery caching** - Optimize for bulk schedule executions
4. **Update documentation** - Reflect new aligned workflow

---

## 🚫 **WHAT NOT TO CHANGE**

**✅ Keep These (Already Correct)**:
- **Job tracking integration** - `joblog.StartJob()` is correct
- **Conflict detection** - Phantom and conflict detection logic is solid  
- **Schedule execution logic** - Cron scheduling and concurrency control works
- **Database schema** - VM contexts, schedules, executions are properly designed
- **Service integration** - Integration with machine groups and repositories is correct

---

## 📊 **IMPACT ASSESSMENT**

### **🔧 Code Changes Required**:
- **Files**: `scheduler_service.go` (1 file)
- **Methods**: 3 methods (add 2 new, replace 1 existing)
- **Constructor**: 1 parameter addition
- **Impact**: Medium (isolated to scheduler service)

### **🧪 Testing Required**:
- **Unit tests**: VMA discovery integration
- **Integration tests**: Scheduler job creation vs GUI job creation
- **End-to-end tests**: Schedule execution with fresh VM data

### **⚠️ Risk Level**: **LOW-MEDIUM**
- **Risk**: Changes are isolated to scheduler service
- **Mitigation**: Existing job tracking and conflict detection remain unchanged
- **Rollback**: Easy (revert to current database-based approach)

---

## 📚 **REFERENCES**

- **GUI Workflow Document**: `/home/pgrayson/migratekit-cloudstack/AI_Helper/GUI_REPLICATION_WORKFLOW.md`
- **Current Scheduler Code**: `source/current/oma/services/scheduler_service.go:564-627`
- **Target API Endpoint**: `http://localhost:8082/api/v1/replications` (same as GUI)
- **VMA Discovery API**: `http://localhost:9081/api/v1/discover` (same as GUI)

---

**🎯 SUMMARY**: The scheduler currently bypasses the proven GUI workflow by using stale database data instead of fresh vCenter discovery. **Critical fix required**: Add VMA discovery integration and use the same OMA API path as the GUI to ensure consistency and reliability.
