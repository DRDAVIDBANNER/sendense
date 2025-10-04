# GUI Replication Workflow - Definitive Implementation Guide

**Date**: September 19, 2025  
**Status**: ✅ **VERIFIED WORKING** - Tested with pgtest1 and pgtest2  
**Purpose**: Document the exact workflow used by GUI for replication job creation for scheduler implementation

---

## 🎯 **OVERVIEW**

This document describes the **exact workflow** used by the VM-centric GUI to start replication jobs, including the fresh VM discovery process and migration engine API integration. **The scheduler MUST use this same process** to ensure consistency and reliability.

---

## 📋 **COMPLETE WORKFLOW STEPS**

### **🔍 STEP 1: Fresh VM Discovery**

**Location**: `src/components/layout/RightContextPanel.tsx:76-107`

**Purpose**: Always get fresh VM specifications from vCenter before starting replication

**Process**:
```typescript
// 1.1: Create discovery request with timeout protection
const controller = new AbortController();
const timeoutId = setTimeout(() => controller.abort(), 15000); // 15 second timeout

// 1.2: Call VMA discovery API through frontend proxy
const discoveryResponse = await fetch('/api/discover', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({
    vcenter: vmContext.context.vcenter_host || 'quad-vcenter-01.quadris.local',
    username: 'administrator@vsphere.local',
    password: 'EmyGVoBFesGQc47-',
    datacenter: vmContext.context.datacenter || 'DatabanxDC',
    filter: selectedVM  // Specific VM name filter
  }),
  signal: controller.signal
});

// 1.3: Parse discovery response and validate
const discoveryData = await discoveryResponse.json();
const discoveredVM = discoveryData.vms?.find((vm: any) => vm.name === selectedVM);
```

**API Flow**:
- **Frontend**: `/api/discover` (proxy)
- **VMA Tunnel**: `http://localhost:9081/api/v1/discover` 
- **vCenter**: Direct VMware API calls

**Discovery Data Structure** (from VMA):
```json
{
  "vms": [{
    "id": "4205784a-098a-40f1-1f1e-a5cd2597fd59",
    "name": "pgtest2", 
    "path": "/DatabanxDC/vm/pgtest2",
    "datacenter": "DatabanxDC",
    "num_cpu": 2,           // ✅ CPU count from vCenter
    "cpus": 2,              // Alternative field name
    "memory_mb": 4096,      // ✅ Memory from vCenter
    "power_state": "poweredOn",
    "guest_os": "windows",  // ✅ OS type from vCenter
    "vmx_version": "vmx-15",
    "disks": [              // ✅ CRITICAL: Disk specifications
      {
        "id": "disk-2000",
        "path": "[vsanDatastore] c31d8a68-9a66-0818-668e-246e966f3564/PG-MIGRATIONDEV_1.vmdk",
        "size_gb": 110,
        "capacity_bytes": 118111600640,
        "datastore": "vsanDatastore",
        "label": "Hard disk 1",
        "provisioning_type": "thin"
      }
    ],
    "networks": [           // ✅ Network configuration
      {
        "name": "",
        "type": "",
        "connected": true,
        "mac_address": "00:50:56:85:bf:a2",
        "label": "Network adapter 1", 
        "network_name": "QUAD_VLAN_1_STD",
        "adapter_type": "e1000e"
      }
    ]
  }]
}
```

**Validation Requirements**:
- VM must be found in discovery results
- VM must have at least one disk configured
- Disk specifications must be complete

---

### **🚀 STEP 2: Migration Engine API Call**

**Location**: `src/components/layout/RightContextPanel.tsx:113-139`

**Purpose**: Start replication using fresh discovery data through Migration Engine

**Process**:
```typescript
// 2.1: Transform discovery data to Migration Engine format
const response = await fetch('/api/replicate', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({
    source_vm: {
      // ✅ EXACT FIELD MAPPING (CRITICAL FOR COMPATIBILITY)
      id: discoveredVM.id,                                    // VMware VM ID
      name: discoveredVM.name,                                 // VM name
      path: discoveredVM.path,                                 // VM path in vCenter
      vm_id: discoveredVM.id,                                  // Duplicate for compatibility
      vm_name: discoveredVM.name,                              // Duplicate for compatibility  
      vm_path: discoveredVM.path,                              // Duplicate for compatibility
      datacenter: discoveredVM.datacenter,                    // vCenter datacenter
      vcenter_host: vmContext.context.vcenter_host || 'quad-vcenter-01.quadris.local',
      cpus: discoveredVM.num_cpu || discoveredVM.cpus || 2,   // ✅ FIXED: Must be 'cpus' not 'cpu_count'
      memory_mb: discoveredVM.memory_mb || 4096,              // Memory in MB
      power_state: discoveredVM.power_state || "poweredOn",   // Power state
      os_type: discoveredVM.guest_os || "otherGuest",         // OS type
      vmx_version: discoveredVM.vmx_version,                  // VMX version
      disks: discoveredVM.disks,                              // ✅ CRITICAL: Complete disk array
      networks: discoveredVM.networks                         // ✅ Network configuration
    },
    replication_type: 'initial',                              // Always start with 'initial'
    vcenter_host: vmContext.context.vcenter_host || 'quad-vcenter-01.quadris.local',
    datacenter: vmContext.context.datacenter || 'DatabanxDC'
  })
});
```

**API Flow**:
- **Frontend**: `/api/replicate` (proxy)  
- **OMA Backend**: `http://localhost:8082/api/v1/replications`
- **Migration Engine**: `workflows.MigrationEngine.StartMigration()`

---

### **🔄 STEP 3: API Proxy Processing**

**Location**: `src/app/api/replicate/route.ts:8-19`

**Purpose**: Transform frontend request to OMA Migration Engine format

**Process**:
```typescript
// 3.1: Transform to OMA format with defaults
const omaRequest = {
  source_vm: body.source_vm,                                 // Complete VM data from discovery
  ossea_config_id: body.ossea_config_id || 1,               // Default to first OSSEA config  
  replication_type: body.replication_type || "initial",     // Default to full initial sync
  target_network: body.target_network || "",                // Optional
  vcenter_host: body.vcenter_host || "quad-vcenter-01.quadris.local",
  datacenter: body.datacenter || "DatabanxDC", 
  change_id: body.change_id || "",                           // Optional for incremental
  previous_change_id: body.previous_change_id || "",        // Optional for incremental
  snapshot_id: body.snapshot_id || ""                       // Optional
};

// 3.2: Call OMA Migration Engine API
const omaResponse = await fetch('http://localhost:8082/api/v1/replications', {
  method: 'POST',
  headers: {
    'Content-Type': 'application/json',
    'Authorization': 'Bearer sess_longlived_dev_token_2025_2035_permanent'
  },
  body: JSON.stringify(omaRequest),
});
```

---

### **⚙️ STEP 4: OMA Backend Processing**

**Location**: `source/current/oma/api/handlers/replication.go:139-217`

**Purpose**: Validate request and start Migration Engine workflow

**Process**:
```go
// 4.1: Parse and validate request
var req CreateMigrationRequest
json.NewDecoder(r.Body).Decode(&req)

// 4.2: Validate required fields
if req.SourceVM.ID == "" || req.SourceVM.Name == "" || req.SourceVM.Path == "" {
    // Return validation error
}

// 4.3: Generate unique job ID
jobID := "job-" + time.Now().Format("20060102-150405")

// 4.4: Create Migration Engine request  
migrationReq := &workflows.MigrationRequest{
    SourceVM:         req.SourceVM,      // Complete VM data from discovery
    VCenterHost:      req.VCenterHost,
    Datacenter:       req.Datacenter, 
    JobID:            jobID,
    OSSEAConfigID:    req.OSSEAConfigID,
    ReplicationType:  req.ReplicationType,
    TargetNetwork:    req.TargetNetwork,
    ChangeID:         req.ChangeID,
    PreviousChangeID: req.PreviousChangeID,
    SnapshotID:       req.SnapshotID,
}

// 4.5: Determine replication type (initial vs incremental)
if migrationReq.ReplicationType == "" {
    replicationType, previousChangeID, err := h.determineReplicationType(req.SourceVM.Path)
    // Sets to "incremental" if previous successful migration exists
    // Sets to "initial" for first migration or if no previous found
}

// 4.6: Start Migration Engine workflow
result, err := h.migrationEngine.StartMigration(ctx, migrationReq)
```

---

### **🏗️ STEP 5: Migration Engine Workflow**

**Location**: `source/current/oma/workflows/migration.go:115-242`

**Purpose**: Execute complete migration workflow with VM context integration

**Process**:
```go
// 5.1: Create replication job record with VM context
err := m.createReplicationJob(ctx, req)

// 5.2: Analyze VM disks and store specifications  
err := m.analyzeAndRecordVMDisks(req)

// 5.3: Provision OSSEA volumes for each disk
createdVolumes, err := m.provisionOSSEAVolumes(req)

// 5.4: Mount volumes and correlate device paths
mountedVolumes, err := m.mountVolumes(req, createdVolumes)

// 5.5: Create NBD exports for migration
exports, err := m.createNBDExports(req, mountedVolumes)

// 5.6: Start VMA replication process
err := m.startVMAReplication(req, exports)

// 5.7: Initialize VMA progress polling
err := m.vmaProgressPoller.StartPolling(req.JobID)
```

**Key Integration Points**:
- **VM Context Creation**: `createReplicationJob()` creates/updates `vm_replication_contexts`
- **Specification Storage**: `analyzeAndRecordVMDisks()` stores VM specs from discovery data
- **Context Updates**: `updateVMContextWithSpecs()` updates context with fresh VM specifications

---

## 🔧 **CRITICAL IMPLEMENTATION DETAILS**

### **⚠️ Required Field Mappings**

**Frontend Discovery → Backend VMInfo**:
```typescript
// ✅ CORRECT MAPPING (Fixed September 19, 2025)
{
  cpus: discoveredVM.num_cpu || discoveredVM.cpus || 2,        // Backend expects 'cpus'
  memory_mb: discoveredVM.memory_mb || 4096,                  // Backend expects 'memory_mb'
  os_type: discoveredVM.guest_os || "otherGuest",             // Backend expects 'os_type'
  power_state: discoveredVM.power_state || "poweredOn",      // Backend expects 'power_state'
  disks: discoveredVM.disks,                                  // Backend expects 'disks' array
  networks: discoveredVM.networks                             // Backend expects 'networks' array
}

// ❌ PREVIOUS ERROR (Before Fix)
{
  cpu_count: discoveredVM.num_cpu,  // Wrong! Backend doesn't recognize this
}
```

### **🔄 Replication Type Logic**

The backend automatically determines replication type:
```go
// Backend logic in determineReplicationType()
if previousSuccessfulMigration {
    return "incremental", previousChangeID  // Use CBT incremental sync
} else {
    return "initial", ""                    // Full migration required
}
```

### **📊 VM Context Integration**

**Database Updates** (Automatic):
1. **Job Creation**: Creates/finds VM context by `vm_name + vcenter_host`
2. **Specification Update**: Updates context with fresh discovery data (CPU, memory, OS, etc.)
3. **Job Linking**: Links job to context via `vm_context_id` 
4. **Statistics Update**: Updates context job counters and status

**Tables Affected**:
- `replication_jobs` (job record with `vm_context_id`)
- `vm_replication_contexts` (VM specifications and statistics)
- `vm_disks` (disk specifications and VM metadata)
- `ossea_volumes` (target volume provisioning)
- `device_mappings` (volume-to-device correlation)
- `nbd_exports` (migration export configuration)

---

## 🎯 **SCHEDULER IMPLEMENTATION REQUIREMENTS**

### **✅ Exact Process to Follow**

**For scheduler implementation, replicate this exact workflow**:

1. **Fresh Discovery**: Always call VMA discovery API for latest VM specifications
2. **Data Transformation**: Use exact field mappings documented above  
3. **Migration Engine**: Call OMA `/api/v1/replications` with complete VM data
4. **Context Integration**: Rely on Migration Engine for VM context creation/updates
5. **Progress Tracking**: Use existing VMA progress polling system

### **🚫 What NOT to Do**

- ❌ **Skip discovery**: Never use stale VM context data for replication
- ❌ **Direct job creation**: Never bypass Migration Engine workflow
- ❌ **Manual context updates**: Let Migration Engine handle VM context integration
- ❌ **Field name mismatches**: Use exact field names documented above

### **📝 Scheduler Implementation Template**

```go
// Scheduler should follow this pattern:

// 1. Fresh VM Discovery
discoveryData, err := discoverVMFromVMA(vmName, vmaAPIEndpoint)
if err != nil {
    return fmt.Errorf("discovery failed: %w", err)
}

// 2. Transform to Migration Request  
migrationRequest := &CreateMigrationRequest{
    SourceVM: models.VMInfo{
        ID:         discoveryData.ID,
        Name:       discoveryData.Name,
        Path:       discoveryData.Path,
        Datacenter: discoveryData.Datacenter,
        CPUs:       discoveryData.NumCPU,     // ✅ Note: 'CPUs' not 'cpu_count'
        MemoryMB:   discoveryData.MemoryMB,
        PowerState: discoveryData.PowerState,
        OSType:     discoveryData.GuestOS,
        Disks:      discoveryData.Disks,      // ✅ CRITICAL: Include disk data
        Networks:   discoveryData.Networks,
    },
    OSSEAConfigID:   1,
    ReplicationType: "",  // Let backend determine initial vs incremental
    VCenterHost:     "quad-vcenter-01.quadris.local",
    Datacenter:      "DatabanxDC",
}

// 3. Call Migration Engine
result, err := callOMAMigrationAPI(migrationRequest)
```

---

## 📚 **REFERENCES**

### **Source Files**:
- **GUI Workflow**: `src/components/layout/RightContextPanel.tsx:57-159`
- **API Proxy**: `src/app/api/replicate/route.ts`
- **Backend Handler**: `source/current/oma/api/handlers/replication.go:139-217`
- **Migration Engine**: `source/current/oma/workflows/migration.go:115-242`
- **VM Context Integration**: `source/current/oma/database/repository.go:1333-1486`

### **API Endpoints**:
- **VMA Discovery**: `http://localhost:9081/api/v1/discover`
- **OMA Migration**: `http://localhost:8082/api/v1/replications`  
- **Frontend Proxy**: `http://localhost:3000/api/replicate`

### **Database Schema**:
- **Jobs**: `replication_jobs` table with `vm_context_id` linking
- **Contexts**: `vm_replication_contexts` table with VM specifications
- **Specifications**: `vm_disks` table with detailed VM metadata

---

**🎯 SUMMARY**: This workflow ensures **fresh VM discovery → proper field mapping → Migration Engine integration → VM context updates**. The scheduler MUST follow this exact process for consistency and reliability.
