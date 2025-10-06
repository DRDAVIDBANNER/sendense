# Module 05: Nutanix Source Connector

**Module ID:** MOD-05  
**Status:** 🟡 **PLANNED** (Phase 5)  
**Priority:** Medium  
**Dependencies:** Module 01 (VMware), Module 02 (CloudStack)  
**Owner:** Platform Engineering Team

---

## 🎯 Module Purpose

Capture data from Nutanix AHV (Acropolis Hypervisor) environments using Nutanix native snapshots and change tracking for efficient incremental backups and replication.

**Key Capabilities:**
- Full VM backup from Nutanix Prism Central/Element
- Incremental backup using Nutanix CBT or snapshot diffs
- Live VM replication from Nutanix to any target platform
- Application-consistent snapshots via Nutanix Guest Tools (NGT)
- Multi-disk VM support with storage tiering awareness

---

## 🏗️ Architecture

```
┌──────────────────────────────────────────────────────────────┐
│ NUTANIX SOURCE CONNECTOR ARCHITECTURE                       │
├──────────────────────────────────────────────────────────────┤
│                                                              │
│  Nutanix Infrastructure                                      │
│  ┌─────────────────────────────────────────────────────┐   │
│  │ Prism Central (Multi-Cluster Management)           │   │
│  │ ├─ Cluster 1 (Production)                          │   │
│  │ ├─ Cluster 2 (Development)                         │   │
│  │ └─ Cluster 3 (DR Site)                             │   │
│  └─────────────────────────────────────────────────────┘   │
│                        ↓ Prism REST API v3                  │
│  ┌─────────────────────────────────────────────────────┐   │
│  │ Nutanix Cluster (AHV Nodes)                        │   │
│  │ ├─ Node 1: [VM1, VM2] + CVM1                      │   │
│  │ ├─ Node 2: [VM3, VM4] + CVM2                      │   │
│  │ ├─ Node 3: [VM5, VM6] + CVM3                      │   │ │
│  │ └─ Distributed Storage (DSF)                       │   │
│  └─────────────────────────────────────────────────────┘   │
│                        ↓ SSH + Prism API                    │
│  ┌─────────────────────────────────────────────────────┐   │
│  │ Sendense Capture Agent (Nutanix)                   │   │
│  │ ├─ Prism API Client (VM management)                │   │
│  │ ├─ Snapshot Manager (incremental diffs)            │   │
│  │ ├─ NGT Integration (application consistency)        │   │
│  │ ├─ qemu-nbd Server (export VM disks)               │   │
│  │ └─ SSH Tunnel Client (secure transport)            │   │
│  └─────────────────────────────────────────────────────┘   │
│                        ↓ NBD Stream (SSH Tunnel Port 443)   │
│  ┌─────────────────────────────────────────────────────┐   │
│  │ Sendense Control Plane                              │   │
│  │ ├─ NBD Server (receive streams)                     │   │
│  │ ├─ Backup Repository (QCOW2, S3, etc.)             │   │
│  │ └─ Target Connectors (VMware, CloudStack, etc.)    │   │
│  └─────────────────────────────────────────────────────┘   │
└──────────────────────────────────────────────────────────────┘
```

---

## 🔧 Technical Implementation

### **Nutanix Prism API Integration**

**VM Discovery:**
```go
type NutanixSource struct {
    prismClient   *nutanix.PrismClient
    clusterUUID   string
    authToken     string
}

func (ns *NutanixSource) DiscoverVMs() ([]NutanixVM, error) {
    // 1. Connect to Prism Central/Element
    client := nutanix.NewPrismClient(&nutanix.Config{
        Endpoint: ns.config.PrismEndpoint, // e.g., "10.0.1.100:9440"
        Username: ns.config.Username,
        Password: ns.config.Password,
        Insecure: false, // Validate SSL certificates
    })
    
    // 2. Get all VMs in cluster
    vms, err := client.GetVMs(&nutanix.GetVMsRequest{
        Length: 500, // Max VMs per request
        Offset: 0,
    })
    if err != nil {
        return nil, err
    }
    
    var discoveredVMs []NutanixVM
    for _, vm := range vms.Entities {
        // 3. Get detailed VM information
        vmDetail, err := client.GetVM(vm.UUID)
        if err != nil {
            continue
        }
        
        // 4. Get disk information
        disks, err := ns.getVMDisks(vm.UUID)
        if err != nil {
            continue
        }
        
        nutanixVM := NutanixVM{
            UUID:        vm.UUID,
            Name:        vm.Name,
            PowerState:  vm.PowerState,
            CPUs:        vm.NumVCPUs,
            Memory:      vm.MemorySizeMB,
            Disks:       disks,
            HostUUID:    vm.HostUUID,
            ContainerUUID: vm.ContainerUUID,
            NGTEnabled:  vm.NutanixGuestTools.Enabled,
        }
        
        discoveredVMs = append(discoveredVMs, nutanixVM)
    }
    
    return discoveredVMs, nil
}
```

### **Nutanix Snapshot Management**

**Application-Consistent Snapshots:**
```go
func (ns *NutanixSource) CreateApplicationConsistentSnapshot(vmUUID string) (*Snapshot, error) {
    // 1. Check if NGT (Nutanix Guest Tools) is installed
    vm, err := ns.prismClient.GetVM(vmUUID)
    if err != nil {
        return nil, err
    }
    
    var snapshotType string
    if vm.NutanixGuestTools.Enabled {
        snapshotType = "APPLICATION_CONSISTENT" // Uses NGT for quiescing
    } else {
        snapshotType = "CRASH_CONSISTENT"      // Just disk snapshot
    }
    
    // 2. Create snapshot via Prism API
    snapshotSpec := nutanix.SnapshotSpec{
        SnapshotSpecs: []nutanix.SnapshotSpecDetails{
            {
                VMUUID:      vmUUID,
                SnapshotType: snapshotType,
            },
        },
        Name:        fmt.Sprintf("sendense-backup-%d", time.Now().Unix()),
        Description: "Sendense backup consistency point",
    }
    
    task, err := ns.prismClient.CreateSnapshot(&snapshotSpec)
    if err != nil {
        return nil, err
    }
    
    // 3. Wait for snapshot completion
    snapshot, err := ns.prismClient.WaitForTask(task.TaskUUID)
    if err != nil {
        return nil, err
    }
    
    return &Snapshot{
        UUID:        snapshot.UUID,
        Name:        snapshot.Name,
        Created:     snapshot.CreatedTime,
        VMUUID:      vmUUID,
        IsAppConsistent: snapshotType == "APPLICATION_CONSISTENT",
    }, nil
}
```

### **Incremental Change Detection**

**Option 1: Snapshot Diffs (Nutanix Native)**
```go
func (ns *NutanixSource) GetChangedBlocksViaSnapshots(vmUUID, baseSnapshot, currentSnapshot string) ([]ChangedBlock, error) {
    // Nutanix can provide block diffs between snapshots
    diff, err := ns.prismClient.GetSnapshotDiff(&nutanix.SnapshotDiffRequest{
        BaseSnapshot:    baseSnapshot,
        CurrentSnapshot: currentSnapshot,
        BlockSize:       65536, // 64KB blocks
    })
    if err != nil {
        return nil, err
    }
    
    var changedBlocks []ChangedBlock
    for _, block := range diff.ChangedBlocks {
        changedBlocks = append(changedBlocks, ChangedBlock{
            Offset:   block.Offset,
            Length:   block.Length,
            Checksum: block.Checksum, // Nutanix provides checksums
        })
    }
    
    return changedBlocks, nil
}
```

**Option 2: CBT Integration (If Available)**
```go
func (ns *NutanixSource) EnableCBT(vmUUID string) error {
    // Check if Nutanix supports CBT in AOS version
    aosVersion, err := ns.prismClient.GetAOSVersion()
    if err != nil {
        return err
    }
    
    // CBT support varies by AOS version
    if aosVersion.Major >= 6 && aosVersion.Minor >= 1 {
        // AOS 6.1+ has experimental CBT support
        return ns.enableNutanixCBT(vmUUID)
    }
    
    // Fallback to snapshot-based change tracking
    log.Info("CBT not available, using snapshot-based change detection")
    return ns.enableSnapshotBasedTracking(vmUUID)
}
```

---

## 💾 Nutanix-Specific Features

### **Storage Tiers Integration**

**Nutanix Advantage:** Storage tiering awareness
```go
type NutanixDisk struct {
    UUID            string
    Size            int64
    ContainerUUID   string
    StorageTier     string // "SSD", "DAS-SATA", "CLOUD"
    CompressionEnabled bool
    DeduplicationEnabled bool
}

func (ns *NutanixSource) OptimizeBackupForStorageTier(disk NutanixDisk) BackupConfig {
    config := BackupConfig{}
    
    switch disk.StorageTier {
    case "SSD":
        // Fast tier - can handle high IOPS
        config.Parallelism = 8
        config.BlockSize = 1024 * 1024 // 1MB blocks
        
    case "DAS-SATA":
        // Medium tier - balance performance and impact
        config.Parallelism = 4
        config.BlockSize = 512 * 1024  // 512KB blocks
        
    case "CLOUD":
        // Cloud tier - minimize impact, larger blocks
        config.Parallelism = 2
        config.BlockSize = 4096 * 1024 // 4MB blocks
    }
    
    // Honor Nutanix compression/dedup
    if disk.CompressionEnabled {
        config.EnableCompression = false // Already compressed
    }
    
    return config
}
```

### **Distributed Storage (DSF) Awareness**

**Multi-Node Access:**
```go
func (ns *NutanixSource) FindOptimalAccessPoint(vmUUID string) (*AccessPoint, error) {
    // 1. Get VM location
    vm, err := ns.prismClient.GetVM(vmUUID)
    if err != nil {
        return nil, err
    }
    
    // 2. Find which node has VM's disk locally
    for _, disk := range vm.VMDisks {
        // Nutanix DSF can tell us which node has the data locally
        replica, err := ns.prismClient.GetDiskReplica(disk.UUID)
        if err != nil {
            continue
        }
        
        // 3. Prefer local access for performance
        if replica.IsPrimary {
            return &AccessPoint{
                NodeUUID:    replica.NodeUUID,
                NodeIP:      replica.NodeIP,
                LocalAccess: true,
                DiskPath:    replica.LocalPath,
            }, nil
        }
    }
    
    // 4. Fallback to any available node
    return &AccessPoint{
        NodeUUID:    vm.HostUUID,
        LocalAccess: false,
    }, nil
}
```

---

## 🎯 Capture Agent (Nutanix Variant)

### **Agent Deployment Options**

**Option 1: CVM Agent (Recommended)**
- **Location:** Nutanix Controller VM (CVM)
- **Advantages:** Direct access to storage layer, native integration
- **Challenges:** Resource sharing with Nutanix services

**Option 2: External Agent**
- **Location:** Separate VM with Prism API access
- **Advantages:** No impact on Nutanix infrastructure
- **Challenges:** Network overhead, API rate limits

**Option 3: Container Agent (Modern)**
- **Location:** Kubernetes on Nutanix (Karbon)
- **Advantages:** Cloud-native, scalable
- **Challenges:** Requires Karbon deployment

### **Agent Implementation**

```go
type NutanixCaptureAgent struct {
    prismClient     *nutanix.PrismClient
    snapshotManager *SnapshotManager
    qemuNBD         *QEMUNBDServer
    sshTunnel       *SSHTunnelClient
    config          AgentConfig
}

func (agent *NutanixCaptureAgent) StartVMBackup(vmUUID string, isIncremental bool) error {
    // 1. Create application-consistent snapshot
    snapshot, err := agent.createConsistentSnapshot(vmUUID)
    if err != nil {
        return err
    }
    defer agent.cleanupSnapshot(snapshot.UUID)
    
    // 2. Determine what data to transfer
    var blocks []ChangedBlock
    if isIncremental {
        // Get changed blocks since last backup
        lastSnapshot := agent.getLastBackupSnapshot(vmUUID)
        blocks, err = agent.getChangedBlocks(lastSnapshot.UUID, snapshot.UUID)
        if err != nil {
            // Fallback to full backup
            blocks = nil
        }
    }
    
    // 3. Export disk data via NBD
    diskPath := agent.getSnapshotDiskPath(snapshot.UUID)
    nbdPort := agent.allocateNBDPort()
    
    export, err := agent.qemuNBD.ExportDisk(diskPath, nbdPort)
    if err != nil {
        return err
    }
    defer export.Close()
    
    // 4. Notify Control Plane
    endpoint := fmt.Sprintf("%s:%d", agent.config.LocalIP, nbdPort)
    return agent.notifyControlPlane(vmUUID, endpoint, blocks)
}
```

---

## 🌟 Nutanix-Specific Advantages

### **Native QCOW2 Support**
- **Nutanix AHV uses QCOW2 natively** (like CloudStack)
- **No format conversion needed** for backup storage
- **Efficient incremental chains** using QCOW2 backing files
- **Native compression and deduplication** integration

### **Application Integration**

**Nutanix Guest Tools (NGT):**
```go
func (agent *NutanixCaptureAgent) checkNGTStatus(vmUUID string) (*NGTStatus, error) {
    vm, err := agent.prismClient.GetVM(vmUUID)
    if err != nil {
        return nil, err
    }
    
    ngtStatus := &NGTStatus{
        Installed:           vm.NutanixGuestTools.Enabled,
        Version:            vm.NutanixGuestTools.Version,
        VSSEnabled:         vm.NutanixGuestTools.Applications.VSS,
        FileSystemFreezeEnabled: vm.NutanixGuestTools.Applications.FileSystemFreeze,
        AppConsistentSupport: vm.NutanixGuestTools.Capabilities.AppConsistent,
    }
    
    return ngtStatus, nil
}

func (agent *NutanixCaptureAgent) createAppConsistentSnapshot(vmUUID string) (*Snapshot, error) {
    ngtStatus, err := agent.checkNGTStatus(vmUUID)
    if err != nil {
        return nil, err
    }
    
    snapshotSpec := nutanix.SnapshotSpec{
        VMUUID: vmUUID,
        Name:   fmt.Sprintf("sendense-backup-%d", time.Now().Unix()),
    }
    
    if ngtStatus.AppConsistentSupport {
        // Use NGT for application-consistent snapshot
        snapshotSpec.SnapshotType = "APPLICATION_CONSISTENT"
        snapshotSpec.Quiesce = true
    } else {
        // Fall back to crash-consistent
        snapshotSpec.SnapshotType = "CRASH_CONSISTENT"
    }
    
    task, err := agent.prismClient.CreateSnapshot(snapshotSpec)
    if err != nil {
        return nil, err
    }
    
    return agent.prismClient.WaitForSnapshotTask(task.TaskUUID)
}
```

### **Nutanix Storage Efficiency**

**Leverage Native Features:**
```go
func (agent *NutanixCaptureAgent) optimizeForNutanixStorage(vmUUID string) BackupOptimization {
    // Get storage container information
    vm, _ := agent.prismClient.GetVM(vmUUID)
    container, _ := agent.prismClient.GetStorageContainer(vm.ContainerUUID)
    
    optimization := BackupOptimization{}
    
    // Don't double-compress if Nutanix compression enabled
    if container.CompressionEnabled {
        optimization.DisableCompression = true
        log.Info("Nutanix compression detected, disabling backup compression")
    }
    
    // Account for Nutanix deduplication
    if container.DeduplicationEnabled {
        optimization.ExpectedRatio = 0.7 // Nutanix already deduplicated
    } else {
        optimization.ExpectedRatio = 0.9 // Normal compression ratio
    }
    
    // Storage tier optimization
    switch container.StorageTier {
    case "SSD":
        optimization.PreferredBlockSize = 1024 * 1024 // 1MB for SSD
        optimization.MaxParallelism = 8
    case "HYBRID":
        optimization.PreferredBlockSize = 512 * 1024  // 512KB for hybrid
        optimization.MaxParallelism = 4
    }
    
    return optimization
}
```

---

## 🔄 Change Tracking Methods

### **Method 1: Snapshot-Based (Primary)**

**Nutanix Native Approach:**
```go
func (agent *NutanixCaptureAgent) performIncrementalBackup(vmUUID string) error {
    // 1. Get last backup's snapshot ID
    lastSnapshot := agent.getLastBackupSnapshot(vmUUID)
    
    // 2. Create new snapshot
    currentSnapshot, err := agent.createConsistentSnapshot(vmUUID)
    if err != nil {
        return err
    }
    
    // 3. Get changed blocks between snapshots
    changedBlocks, err := agent.getSnapshotDiff(lastSnapshot.UUID, currentSnapshot.UUID)
    if err != nil {
        // Fallback to full backup
        log.Warn("Snapshot diff failed, performing full backup")
        return agent.performFullBackup(vmUUID)
    }
    
    // 4. Export only changed blocks
    return agent.exportChangedBlocks(vmUUID, changedBlocks)
}

func (agent *NutanixCaptureAgent) getSnapshotDiff(oldSnapshot, newSnapshot string) ([]ChangedBlock, error) {
    // Use Nutanix API to get block differences
    diff, err := agent.prismClient.GetSnapshotBlockDiff(&nutanix.SnapshotDiffRequest{
        BaseSnapshotUUID: oldSnapshot,
        TargetSnapshotUUID: newSnapshot,
        BlockSize: 65536, // 64KB blocks
    })
    if err != nil {
        return nil, err
    }
    
    var blocks []ChangedBlock
    for _, block := range diff.ChangedBlocks {
        blocks = append(blocks, ChangedBlock{
            Offset: block.OffsetBytes,
            Length: block.LengthBytes,
            Type:   "changed", // or "zero" for sparse handling
        })
    }
    
    return blocks, nil
}
```

### **Method 2: CBT Integration (Future)**

**AOS 6.1+ CBT Support:**
```go
func (agent *NutanixCaptureAgent) enableNutanixCBT(vmUUID string) error {
    // Check AOS version for CBT support
    version, err := agent.prismClient.GetAOSVersion()
    if err != nil {
        return err
    }
    
    if version.SupportsNativeCBT() {
        // Use Nutanix native CBT
        return agent.prismClient.EnableCBT(&nutanix.EnableCBTRequest{
            VMUUID: vmUUID,
            TrackingGranularity: 65536, // 64KB
        })
    }
    
    // Fallback to snapshot-based tracking
    return agent.enableSnapshotTracking(vmUUID)
}
```

---

## 🚀 Performance Characteristics

### **Expected Performance**

**Throughput:**
- **Target:** 2.5-3.0 GiB/s (slightly lower than VMware due to snapshot overhead)
- **Factors:** Nutanix node CPU, DSF performance, network bandwidth
- **Optimization:** Direct CVM access, local disk reads

**Incremental Efficiency:**
- **Target:** 90-95% data reduction (slightly lower than pure CBT)
- **Method:** Snapshot differencing + sparse block detection
- **Frequency:** 5-15 minute incremental cycles

**Resource Impact:**
- **CVM Impact:** <5% CPU (if agent runs on CVM)
- **Storage Impact:** Temporary snapshots (<10% overhead)
- **Network Impact:** Minimal (controlled backup windows)

### **Nutanix Performance Optimization**

**Storage Pool Awareness:**
```go
func (agent *NutanixCaptureAgent) optimizeForStoragePool(vmUUID string) {
    vm, _ := agent.prismClient.GetVM(vmUUID)
    
    // Check storage pool type
    for _, disk := range vm.VMDisks {
        pool, _ := agent.prismClient.GetStoragePool(disk.StoragePoolUUID)
        
        switch pool.StorageTier {
        case "SSD":
            // Optimize for high IOPS
            agent.config.ReadBlockSize = 1024 * 1024  // 1MB
            agent.config.Parallelism = 8
            
        case "HYBRID":
            // Balance between performance and impact
            agent.config.ReadBlockSize = 512 * 1024   // 512KB
            agent.config.Parallelism = 4
            
        case "ARCHIVE":
            // Minimize impact on archive tier
            agent.config.ReadBlockSize = 256 * 1024   // 256KB
            agent.config.Parallelism = 2
        }
    }
}
```

---

## 🔒 Security Integration

### **Nutanix Security Features**

**Data Encryption:**
- **At Rest:** Nutanix native encryption (if enabled)
- **In Transit:** SSH tunnel encryption (Sendense standard)
- **Key Management:** Integration with Nutanix key management

**Authentication:**
```go
type NutanixSecurityManager struct {
    prismClient *nutanix.PrismClient
    credentials SecureCredentials
}

func (nsm *NutanixSecurityManager) authenticateWithPrism() error {
    // Support multiple authentication methods
    switch nsm.credentials.Type {
    case "local":
        return nsm.prismClient.AuthenticateLocal(
            nsm.credentials.Username,
            nsm.credentials.Password,
        )
        
    case "ad":
        return nsm.prismClient.AuthenticateAD(
            nsm.credentials.Domain,
            nsm.credentials.Username,
            nsm.credentials.Password,
        )
        
    case "saml":
        return nsm.prismClient.AuthenticateSAML(
            nsm.credentials.SAMLAssertion,
        )
    }
    
    return fmt.Errorf("unsupported auth type: %s", nsm.credentials.Type)
}
```

**Access Control:**
```yaml
# Required Prism permissions for Sendense
nutanix_permissions:
  - "Virtual Machine Administrator"  # VM operations
  - "Cluster Viewer"                # Cluster information
  - "Storage Administrator"         # Snapshot operations
  
# Minimal permissions (if granular RBAC available)
granular_permissions:
  vm_operations:
    - "VM.Read"
    - "VM.Snapshot.Create"
    - "VM.Snapshot.Delete"
  storage_operations:
    - "StoragePool.Read"
    - "Volume.Read"
    - "Snapshot.Create"
    - "Snapshot.Read"
    - "Snapshot.Delete"
```

---

## 🛠️ Deployment Strategy

### **Agent Deployment (CVM)**

**Installation on Controller VM:**
```bash
# 1. SSH to Nutanix CVM
ssh nutanix@cvm-ip

# 2. Download Sendense agent
curl -O https://releases.sendense.com/nutanix-capture-agent-v1.0.0
chmod +x nutanix-capture-agent-v1.0.0

# 3. Install as systemd service
sudo cp nutanix-capture-agent-v1.0.0 /usr/local/bin/sendense-capture-agent
sudo cp sendense-nutanix.service /etc/systemd/system/

# 4. Configure agent
sudo vi /opt/sendense/config/nutanix-agent.yaml

# 5. Start service
sudo systemctl enable sendense-nutanix
sudo systemctl start sendense-nutanix
```

**Configuration File:**
```yaml
# nutanix-agent.yaml
nutanix:
  prism_central: "prism-central.company.com:9440"
  prism_element: "prism-element.company.com:9440"
  username: "sendense-backup@company.com"
  password: "${NUTANIX_PASSWORD}"
  cluster_uuid: "12345678-1234-5678-9012-123456789012"
  
backup:
  method: "snapshot_diff"          # or "cbt" if supported
  consistency: "application"       # Use NGT for app consistency
  snapshot_retention_hours: 24     # Keep snapshots 24h for incrementals
  
performance:
  max_concurrent_vms: 5           # Concurrent VM backups per CVM
  snapshot_timeout_minutes: 30    # Timeout for snapshot operations
  bandwidth_limit_mbps: 500       # Optional throttling
  
tunnel:
  control_plane_host: "control.sendense.com"
  tunnel_port: 443
  ssh_key_path: "/opt/sendense/ssh/nutanix_tunnel_key"
```

---

## 📊 Nutanix Version Support

### **AOS (Acropolis Operating System) Compatibility**

| AOS Version | Snapshot API | CBT Support | NGT Support | Sendense Status |
|-------------|-------------|-------------|-------------|----------------|
| **AOS 6.5+** | ✅ Full | ⚠️ Experimental | ✅ Full | 🎯 **Target** |
| **AOS 6.1-6.4** | ✅ Full | ❌ No | ✅ Full | ✅ **Supported** |
| **AOS 5.20+** | ✅ Basic | ❌ No | ⚠️ Limited | ⚠️ **Limited** |
| **AOS 5.15-5.19** | ⚠️ Limited | ❌ No | ❌ No | ❌ **Not Supported** |

### **Prism API Versions**

| Prism Version | API v3 | Snapshot Diff | Recommended |
|---------------|--------|---------------|-------------|
| **Prism 6.5+** | ✅ Full | ✅ Native | 🎯 **Target** |
| **Prism 6.1-6.4** | ✅ Full | ⚠️ Basic | ✅ **Supported** |
| **Prism 5.20+** | ⚠️ Limited | ❌ No | ❌ **Not Supported** |

---

## 🎯 Integration with Sendense Platform

### **Backup Integration (Phase 2)**
```go
// How Nutanix integrates with backup repository
func BackupNutanixVM(vmUUID string, repository BackupRepository) error {
    // 1. Discover VM via Prism API
    vm, err := nutanixSource.GetVMSpecs(vmUUID)
    if err != nil {
        return err
    }
    
    // 2. Create backup target (QCOW2 - native format)
    backupID := generateBackupID(vmUUID)
    backupFile := repository.CreateQCOW2Backup(backupID, vm.TotalDiskSize)
    
    // 3. Stream Nutanix data to backup (via snapshots)
    return nutanixSource.StreamToTarget(vmUUID, backupFile)
}
```

### **Replication Integration (Phase 5)**
```go
// Nutanix → Any platform replication
func ReplicateNutanixToVMware(vmUUID string, vmwareTarget VMwareTarget) error {
    // 1. Setup incremental tracking
    err := nutanixSource.EnableChangeTracking(vmUUID)
    if err != nil {
        return err
    }
    
    // 2. Create VMware replica VM
    replicaVM, err := vmwareTarget.CreateReplicaVM(vm.Specs)
    if err != nil {
        return err
    }
    
    // 3. Setup replication job
    job := ReplicationJob{
        SourcePlatform: "nutanix",
        TargetPlatform: "vmware",
        SourceVMUUID:   vmUUID,
        TargetVMUUID:   replicaVM.UUID,
        Method:         "snapshot_diff", // or "cbt" if available
    }
    
    return startReplication(job)
}
```

---

## 🌐 Competitive Positioning

### **Nutanix Backup Market**

**Current Solutions:**
- **Nutanix Mine:** Native but limited (backup only, no replication)
- **Veeam:** Limited Nutanix support, no cross-platform
- **Cohesity:** Good Nutanix support but expensive
- **Rubrik:** Decent support but locked to Rubrik hardware

**Sendense Advantages:**
- **Cross-Platform:** Nutanix → Any target platform
- **Modern UI:** Better than most Nutanix-focused tools  
- **Open Architecture:** Not locked to specific hardware
- **Cost Effective:** Lower price point than Cohesity/Rubrik

### **Unique Value Propositions**

1. **Nutanix → CloudStack:** Unique combination (no other vendor)
2. **Nutanix → VMware:** Escape Nutanix lock-in if needed
3. **Nutanix → Cloud:** AWS/Azure migration path from Nutanix
4. **Storage Efficiency:** Leverage Nutanix compression/dedup

---

## 🔧 Troubleshooting

### **Common Issues**

**Issue 1: Prism API Authentication**
```bash
# Test Prism connectivity
curl -k "https://prism-central.company.com:9440/api/nutanix/v3/clusters" \
  -H "Authorization: Basic $(echo -n 'user:pass' | base64)"

# Should return cluster information
```

**Issue 2: CVM Resource Contention**
```bash
# Check CVM resource usage
ssh nutanix@cvm-ip "top -bn1 | head -20"

# Monitor agent impact
systemctl status sendense-nutanix
journalctl -u sendense-nutanix --since "1 hour ago"
```

**Issue 3: Snapshot Operations Slow**
```bash
# Check storage pool performance
ncli storagepool list
ncli container list

# Check for storage issues
nutanix@cvm$ allssh "curator_cli get_cluster_stats"
```

---

## 🎯 Success Criteria

### **Module Complete When:**
- ✅ Nutanix VMs discoverable via Prism API
- ✅ Agent deployed successfully on CVM or dedicated VM
- ✅ Snapshot-based incremental backup operational
- ✅ Application-consistent snapshots via NGT
- ✅ Multi-disk Nutanix VMs supported
- ✅ Integration with backup repository
- ✅ Performance: 2.5+ GiB/s throughput

### **Quality Gates:**
- ✅ Tested with AOS 6.1+ clusters
- ✅ Tested with various storage configurations (SSD, hybrid)
- ✅ NGT integration verified
- ✅ Security review passed
- ✅ Documentation complete

---

## 🌟 Strategic Value

### **Nutanix Market Opportunity**

**Market Size:**
- Nutanix installed base: 18,000+ customers worldwide
- Average environment: 100-500 VMs
- Growing cloud migration needs (Nutanix → AWS/Azure)

**Revenue Potential:**
- **Backup Tier:** Nutanix customers need backup → $10/VM
- **Enterprise Tier:** Cross-platform restore → $25/VM
- **Replication Tier:** Nutanix → Cloud migration → $100/VM

**Competitive Moat:**
- Few vendors focus on Nutanix
- Cross-platform capability unique
- Modern interface vs legacy tools
- Cost advantage vs Cohesity/Rubrik

---

## 🔗 Related Modules

- **Module 01:** VMware Source (architecture pattern)
- **Module 02:** CloudStack Source (libvirt similarities)
- **Module 03:** Backup Repository (QCOW2 storage)
- **Module 04:** Restore Engine (Nutanix as target)

---

## 📈 Future Enhancements

**Roadmap:**
1. **AOS 6.5+ Features:** Native CBT, enhanced snapshots
2. **Karbon Integration:** Kubernetes workload backup
3. **Multi-Cluster:** Cross-cluster replication
4. **Edge Computing:** Nutanix Edge backup
5. **Xi Cloud:** Nutanix cloud service integration

---

**Module Owner:** Nutanix Engineering Team  
**Implementation Phase:** Phase 5  
**Last Updated:** October 4, 2025  
**Status:** 🟡 Planned - Growing Market Opportunity

