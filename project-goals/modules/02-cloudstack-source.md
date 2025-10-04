# Module 02: CloudStack Source Connector

**Module ID:** MOD-02  
**Status:** 🟡 **PLANNED** (Phase 2)  
**Priority:** High  
**Dependencies:** Module 01 (VMware Source)  
**Owner:** Platform Engineering Team

---

## 🎯 Module Purpose

Capture data from CloudStack/KVM environments using libvirt dirty bitmaps for efficient incremental backups and replication.

**Key Capabilities:**
- Full VM backup from CloudStack management
- Incremental backup using QEMU dirty bitmaps
- Live VM replication from CloudStack to any target platform
- Application-consistent snapshots via CloudStack guest tools
- Multi-disk CloudStack VM support

---

## 🏗️ Architecture

```
┌──────────────────────────────────────────────────────────────┐
│ CLOUDSTACK SOURCE CONNECTOR ARCHITECTURE                    │
├──────────────────────────────────────────────────────────────┤
│                                                              │
│  CloudStack Management Server                               │
│  ┌─────────────────────────────────────────────────────┐   │
│  │ CloudStack API                                      │   │
│  │ ├─ VM Management (start, stop, snapshot)           │   │
│  │ ├─ Host Discovery (find KVM hosts)                 │   │
│  │ ├─ Volume Management (attach, detach)              │   │
│  │ └─ Network Configuration                            │   │
│  └─────────────────────────────────────────────────────┘   │
│                        ↓ CloudStack API                     │
│  ┌─────────────────────────────────────────────────────┐   │
│  │ KVM Hypervisor Hosts                               │   │
│  │ ├─ Host 1: [VM1, VM2, VM3]                        │   │
│  │ ├─ Host 2: [VM4, VM5, VM6]                        │   │
│  │ └─ Shared Storage (NFS, Ceph, Local)              │   │
│  └─────────────────────────────────────────────────────┘   │
│                        ↓ libvirt + SSH                      │
│  ┌─────────────────────────────────────────────────────┐   │
│  │ Sendense Capture Agent (CloudStack)                │   │
│  │ ├─ libvirt Client (qemu:///system)                │   │
│  │ ├─ Dirty Bitmap Manager (QMP/Monitor)              │   │
│  │ ├─ qemu-nbd Server (Export VM disks)               │   │
│  │ ├─ CloudStack API Client (VM control)              │   │
│  │ └─ SSH Tunnel Client (Secure transport)            │   │
│  └─────────────────────────────────────────────────────┘   │
│                        ↓ NBD Stream (SSH Tunnel Port 443)   │
│  ┌─────────────────────────────────────────────────────┐   │
│  │ Sendense Control Plane                              │   │
│  │ ├─ NBD Server (Receive streams)                     │   │
│  │ ├─ Backup Repository (QCOW2, S3, etc.)             │   │
│  │ └─ Target Connectors (VMware, AWS, etc.)           │   │
│  └─────────────────────────────────────────────────────┘   │
└──────────────────────────────────────────────────────────────┘
```

---

## 🔧 Technical Implementation

### **Dirty Bitmap Integration (QEMU/KVM)**

**Core Implementation:**
```go
// Location: source/current/capture-agent/cloudstack/dirty_bitmap.go
type DirtyBitmapManager struct {
    libvirtConn *libvirt.Connect
    qmpClients  map[string]*qmp.SocketMonitor
}

func (dbm *DirtyBitmapManager) EnableDirtyBitmap(vmUUID string, diskTarget string) error {
    // 1. Connect to VM's QMP socket
    qmpSocket := fmt.Sprintf("/var/lib/libvirt/qemu/%s.monitor", vmUUID)
    monitor, err := qmp.NewSocketMonitor("unix", qmpSocket, 2*time.Second)
    if err != nil {
        return err
    }
    
    // 2. Add dirty bitmap to disk
    bitmapName := fmt.Sprintf("sendense-bitmap-%d", time.Now().Unix())
    cmd := fmt.Sprintf(`{
        "execute": "block-dirty-bitmap-add",
        "arguments": {
            "node": "%s",
            "name": "%s",
            "persistent": true,
            "disabled": false
        }
    }`, diskTarget, bitmapName)
    
    response, err := monitor.Run([]byte(cmd))
    if err != nil {
        return err
    }
    
    dbm.qmpClients[vmUUID] = monitor
    return nil
}

func (dbm *DirtyBitmapManager) GetChangedBlocks(vmUUID, bitmapName string) ([]ChangedBlock, error) {
    monitor := dbm.qmpClients[vmUUID]
    
    // Query dirty bitmap for changed blocks
    cmd := fmt.Sprintf(`{
        "execute": "query-block-dirty-bitmap",
        "arguments": {
            "node": "drive-virtio-disk0",
            "name": "%s"
        }
    }`, bitmapName)
    
    response, err := monitor.Run([]byte(cmd))
    if err != nil {
        return nil, err
    }
    
    // Parse response to get changed block ranges
    return parseChangedBlocks(response), nil
}
```

**libvirt Integration:**
```go
// Alternative approach using libvirt-go
import "github.com/libvirt/libvirt-go"

func (dbm *DirtyBitmapManager) EnableDirtyBitmapLibvirt(vmName, diskTarget string) error {
    // 1. Connect to libvirt
    conn, err := libvirt.NewConnect("qemu:///system")
    if err != nil {
        return err
    }
    defer conn.Close()
    
    // 2. Get domain (VM)
    dom, err := conn.LookupDomainByName(vmName)
    if err != nil {
        return err
    }
    defer dom.Free()
    
    // 3. Add dirty bitmap
    bitmapName := "sendense-incremental"
    err = dom.BlockDirtyBitmapAdd(
        diskTarget,                                    // "vda", "vdb", etc.
        bitmapName,                                    // Bitmap name
        libvirt.DOMAIN_BLOCK_DIRTY_BITMAP_PERSISTENT, // Persist across reboots
    )
    
    return err
}
```

---

## 💾 CloudStack VM Discovery

### **VM Inventory Integration**

**CloudStack API Discovery:**
```go
type CloudStackSource struct {
    apiClient *cloudstack.CloudStackClient
    kvmHosts  map[string]KVMHost
}

func (cs *CloudStackSource) DiscoverVMs() ([]CloudStackVM, error) {
    // 1. Get all VMs from CloudStack API
    vms, err := cs.apiClient.ListVirtualMachines(&cloudstack.ListVirtualMachinesParams{})
    if err != nil {
        return nil, err
    }
    
    var discoveredVMs []CloudStackVM
    for _, vm := range vms.VirtualMachines {
        // 2. Get detailed VM information
        vmDetail, err := cs.apiClient.GetVirtualMachine(&cloudstack.GetVirtualMachineParams{
            ID: vm.ID,
        })
        if err != nil {
            continue
        }
        
        // 3. Find which KVM host runs this VM
        kvmHost, err := cs.findVMHost(vm.ID)
        if err != nil {
            continue
        }
        
        // 4. Get disk information
        volumes, err := cs.apiClient.ListVolumes(&cloudstack.ListVolumesParams{
            VirtualMachineID: vm.ID,
        })
        
        discoveredVM := CloudStackVM{
            UUID:        vm.ID,
            Name:        vm.Name,
            State:       vm.State,
            KVMHost:     kvmHost,
            CPUs:        vm.CPUNumber,
            Memory:      vm.Memory,
            Volumes:     volumes.Volumes,
            Networks:    vmDetail.Nic,
        }
        
        discoveredVMs = append(discoveredVMs, discoveredVM)
    }
    
    return discoveredVMs, nil
}
```

**KVM Host Discovery:**
```go
func (cs *CloudStackSource) findVMHost(vmID string) (KVMHost, error) {
    // CloudStack API provides host information
    vm, err := cs.apiClient.GetVirtualMachine(&cloudstack.GetVirtualMachineParams{
        ID: vmID,
    })
    if err != nil {
        return KVMHost{}, err
    }
    
    // Get host details
    host, err := cs.apiClient.GetHost(&cloudstack.GetHostParams{
        ID: vm.HostID,
    })
    if err != nil {
        return KVMHost{}, err
    }
    
    return KVMHost{
        ID:          host.ID,
        Name:        host.Name,
        IPAddress:   host.IPAddress,
        Type:        host.Type,        // Should be "Routing" for KVM
        Hypervisor:  host.Hypervisor, // Should be "KVM"
    }, nil
}
```

---

## 🎯 Capture Agent (CloudStack Variant)

### **Agent Architecture**

**Agent Components:**
```
Sendense Capture Agent (CloudStack)
├── CloudStack API Client          # VM discovery, control
├── libvirt Connection            # Local KVM management
├── Dirty Bitmap Manager          # QEMU change tracking
├── qemu-nbd Server              # Disk export
├── SSH Tunnel Client            # Secure transport to Control Plane
├── Local API Server             # Control Plane commands
└── Agent Health Monitor         # Status reporting
```

**Agent Deployment:**
- **Location:** KVM hypervisor hosts
- **Installation:** systemd service
- **Configuration:** YAML config file
- **Management:** Control Plane API

### **Multi-VM Support**

**Concurrent Operations:**
```go
type KVMHostAgent struct {
    maxConcurrentBackups int
    activeJobs          map[string]*BackupJob
    jobQueue           chan BackupJob
}

func (agent *KVMHostAgent) StartBackup(vmUUID string) error {
    // 1. Check if we can handle another job
    if len(agent.activeJobs) >= agent.maxConcurrentBackups {
        // Queue the job
        agent.jobQueue <- BackupJob{VMUUID: vmUUID}
        return nil
    }
    
    // 2. Start backup immediately
    go agent.executeBackup(vmUUID)
    return nil
}

func (agent *KVMHostAgent) executeBackup(vmUUID string) {
    // 1. Enable dirty bitmap
    err := agent.enableDirtyBitmap(vmUUID)
    if err != nil {
        agent.reportError(vmUUID, err)
        return
    }
    
    // 2. Export disk via qemu-nbd
    diskPath := agent.getVMDiskPath(vmUUID)
    nbdPort := agent.allocateNBDPort()
    
    cmd := exec.Command("qemu-nbd", "-t", diskPath, "-p", fmt.Sprint(nbdPort))
    err = cmd.Start()
    if err != nil {
        agent.reportError(vmUUID, err)
        return
    }
    
    // 3. Notify Control Plane that export is ready
    agent.notifyExportReady(vmUUID, nbdPort)
    
    // 4. Monitor progress and cleanup
    agent.monitorBackupProgress(vmUUID, cmd)
}
```

---

## 🌟 CloudStack Integration Advantages

### **Native QCOW2 Support**

**Advantage:** CloudStack/KVM uses QCOW2 natively
- **No Format Conversion:** Backup QCOW2 → Store QCOW2 (efficient)
- **Backing File Support:** Perfect for incremental chains
- **Snapshot Integration:** CloudStack snapshots work naturally with QCOW2
- **Compression:** Native QCOW2 compression support

**Comparison with VMware:**
| Feature | VMware (VMDK) | CloudStack (QCOW2) |
|---------|---------------|-------------------|
| **Native Format** | VMDK | QCOW2 |
| **Incremental Support** | CBT | Dirty Bitmaps |
| **Compression** | External | Native |
| **Backing Files** | No | Yes |
| **Snapshot Integration** | Separate mechanism | Native |

### **libvirt Ecosystem**

**Advantages:**
- **Open Standards:** libvirt is open-source, well-documented
- **Rich API:** More granular control than proprietary APIs
- **Tool Ecosystem:** qemu-img, qemu-nbd, virsh, etc.
- **Community Support:** Large open-source community

**Example Operations:**
```bash
# Query VM state via virsh
virsh list --all

# Get VM disk information
virsh domblklist database-prod-01

# Create dirty bitmap
virsh qemu-monitor-command database-prod-01 --hmp \
  "block-dirty-bitmap-add drive-virtio-disk0 sendense-bitmap"

# Query dirty bitmap
virsh qemu-monitor-command database-prod-01 --hmp \
  "info block-dirty-bitmap drive-virtio-disk0"

# Export disk via NBD
qemu-nbd -t /var/lib/libvirt/images/database-prod-01.qcow2 -p 10808
```

---

## 🔄 Change Tracking: Dirty Bitmaps vs CBT

### **Dirty Bitmap Workflow**

```
Initial State (Full Backup):
┌─────────────────────────────────────┐
│ VM Disk: 100 GB QCOW2               │
│ Bitmap: Created but empty            │
└─────────────────────────────────────┘

VM Operations (writes to disk):
┌─────────────────────────────────────┐
│ Block 0-999:    [Clean]              │
│ Block 1000-1999: [DIRTY] ← Write     │
│ Block 2000-2999: [Clean]             │
│ Block 3000-3999: [DIRTY] ← Write     │
│ Block 4000+:    [Clean]              │
│                                     │
│ Bitmap tracks exactly what changed!  │
└─────────────────────────────────────┘

Incremental Backup:
┌─────────────────────────────────────┐
│ 1. Query bitmap for dirty blocks     │
│ 2. Read only blocks 1000-1999,      │
│    3000-3999 (2GB vs 100GB full)    │
│ 3. Stream to backup repository       │
│ 4. Clear bitmap after success        │ │
│ Result: 98% data reduction!          │
└─────────────────────────────────────┘
```

### **CBT vs Dirty Bitmap Comparison**

| Feature | VMware CBT | QEMU Dirty Bitmap |
|---------|------------|-------------------|
| **Granularity** | 256KB-1MB | 64KB-1MB (configurable) |
| **Performance Impact** | <1% | <1% |
| **Persistence** | Across reboots | Optional (can persist) |
| **API Access** | VDDK only | libvirt + QMP |
| **Reset Method** | Snapshot creation | Clear bitmap command |
| **Multi-disk** | Per-disk CBT | Per-disk bitmap |
| **Maturity** | Very mature | Stable (QEMU 4.0+) |

**Dirty Bitmap Advantages:**
- More granular control (64KB vs 256KB)
- Direct API access (no proprietary SDK)
- Flexible configuration
- Open-source transparency

---

## 📋 Implementation Tasks

### **Task 1: CloudStack API Integration**

**Goal:** Control CloudStack VMs for backup orchestration

```go
type CloudStackClient struct {
    apiURL    string
    apiKey    string
    secretKey string
    client    *http.Client
}

func (cs *CloudStackClient) CreateVMSnapshot(vmID string) (*VMSnapshot, error) {
    params := map[string]string{
        "command":            "createVMSnapshot",
        "virtualmachineid":   vmID,
        "snapshotmemory":     "false", // Disk-only snapshot
        "description":        "Sendense backup consistency point",
        "quiescevm":          "true",  // Application-consistent if guest tools
    }
    
    response, err := cs.makeAPICall(params)
    if err != nil {
        return nil, err
    }
    
    // CloudStack returns async job ID
    jobResult, err := cs.waitForAsyncJob(response.JobID)
    if err != nil {
        return nil, err
    }
    
    return &VMSnapshot{
        ID:           jobResult.VMSnapshot.ID,
        Name:         jobResult.VMSnapshot.Name,
        Created:      jobResult.VMSnapshot.Created,
        VirtualMachineID: vmID,
    }, nil
}

func (cs *CloudStackClient) FindKVMHost(vmID string) (*KVMHost, error) {
    // Get VM details to find host
    vm, err := cs.GetVirtualMachine(vmID)
    if err != nil {
        return nil, err
    }
    
    // Get host information
    host, err := cs.GetHost(vm.HostID)
    if err != nil {
        return nil, err
    }
    
    return &KVMHost{
        ID:         host.ID,
        Name:       host.Name,
        IPAddress:  host.IPAddress,
        Type:       host.Type,
        SSHAccess:  agent.canSSHToHost(host.IPAddress),
    }, nil
}
```

### **Task 2: KVM Host Agent**

**Goal:** Deploy agent on KVM hosts for libvirt access

```go
// Agent runs as systemd service on KVM host
type KVMHostAgent struct {
    config      AgentConfig
    libvirtConn *libvirt.Connect
    qmpSockets  map[string]*qmp.SocketMonitor
    nbdExports  map[string]*NBDExport
    sshTunnel   *SSHTunnelClient
}

func (agent *KVMHostAgent) Initialize() error {
    // 1. Connect to local libvirt
    conn, err := libvirt.NewConnect("qemu:///system")
    if err != nil {
        return fmt.Errorf("failed to connect to libvirt: %w", err)
    }
    agent.libvirtConn = conn
    
    // 2. Setup SSH tunnel to Control Plane
    tunnel, err := NewSSHTunnelClient(agent.config.ControlPlaneHost)
    if err != nil {
        return err
    }
    agent.sshTunnel = tunnel
    
    // 3. Start local API server for Control Plane commands
    go agent.startAPIServer()
    
    // 4. Register with Control Plane
    return agent.registerWithControlPlane()
}

func (agent *KVMHostAgent) StartVMBackup(vmUUID string, incrementalFrom string) error {
    // 1. Find VM in libvirt
    dom, err := agent.libvirtConn.LookupDomainByUUIDString(vmUUID)
    if err != nil {
        return err
    }
    
    // 2. Get VM disk information
    diskInfo, err := agent.getVMDiskInfo(dom)
    if err != nil {
        return err
    }
    
    // 3. Setup dirty bitmap (if incremental)
    if incrementalFrom != "" {
        err = agent.setupDirtyBitmap(vmUUID, diskInfo.Target)
        if err != nil {
            return err
        }
    }
    
    // 4. Export disk via qemu-nbd
    nbdPort := agent.allocateNBDPort()
    nbdExport, err := agent.exportDiskViaNBD(diskInfo.Path, nbdPort)
    if err != nil {
        return err
    }
    
    // 5. Notify Control Plane
    return agent.notifyBackupReady(vmUUID, nbdPort)
}
```

### **Task 3: Multi-Host Deployment**

**Goal:** Deploy and manage agents across multiple KVM hosts

```go
type CloudStackAgentManager struct {
    cloudstackClient *CloudStackClient
    hostAgents      map[string]*HostAgentConnection
    sshManager      *SSHManager
}

func (manager *CloudStackAgentManager) DeployAgentsToAllHosts() error {
    // 1. Discover all KVM hosts from CloudStack
    hosts, err := manager.cloudstackClient.ListHosts(&cloudstack.ListHostsParams{
        Type: "Routing", // KVM hosts
    })
    if err != nil {
        return err
    }
    
    // 2. Deploy agent to each host
    for _, host := range hosts.Hosts {
        if host.Hypervisor != "KVM" {
            continue // Skip non-KVM hosts
        }
        
        err := manager.deployAgentToHost(host)
        if err != nil {
            log.Errorf("Failed to deploy to host %s: %v", host.Name, err)
            continue
        }
        
        // 3. Establish connection to agent
        connection, err := manager.connectToAgent(host)
        if err != nil {
            log.Errorf("Failed to connect to agent on %s: %v", host.Name, err)
            continue
        }
        
        manager.hostAgents[host.ID] = connection
    }
    
    return nil
}

func (manager *CloudStackAgentManager) deployAgentToHost(host cloudstack.Host) error {
    // 1. SSH to KVM host
    sshClient, err := manager.sshManager.Connect(host.IPAddress)
    if err != nil {
        return err
    }
    defer sshClient.Close()
    
    // 2. Copy agent binary
    agentBinary := "/opt/sendense/bin/sendense-capture-agent"
    err = sshClient.CopyFile("./sendense-capture-agent", agentBinary)
    if err != nil {
        return err
    }
    
    // 3. Install systemd service
    serviceFile := generateSystemdService(host)
    err = sshClient.WriteFile("/etc/systemd/system/sendense-capture.service", serviceFile)
    if err != nil {
        return err
    }
    
    // 4. Enable and start service
    commands := []string{
        "systemctl daemon-reload",
        "systemctl enable sendense-capture",
        "systemctl start sendense-capture",
    }
    
    for _, cmd := range commands {
        err = sshClient.Execute(cmd)
        if err != nil {
            return err
        }
    }
    
    return nil
}
```

---

## 🎯 Performance Characteristics

### **Expected Performance**

**Throughput:**
- **Target:** 3+ GiB/s (matching VMware module)
- **Factors:** KVM host CPU, storage backend, network bandwidth
- **Optimization:** Multiple NBD connections per VM

**Incremental Efficiency:**
- **Target:** 95%+ data reduction vs full backup
- **Depends On:** VM change rate, dirty bitmap granularity
- **Tuning:** Bitmap block size (64KB-1MB)

**Resource Usage:**
- **Agent CPU:** <5% of KVM host CPU
- **Agent Memory:** 256 MB RAM per agent instance
- **Network:** Configurable bandwidth limiting
- **Storage:** Temporary space for NBD exports

### **Performance Tuning**

**Dirty Bitmap Configuration:**
```yaml
# KVM agent configuration
kvm_agent:
  dirty_bitmap:
    granularity: 65536        # 64KB blocks (balance efficiency vs overhead)
    persistent: true          # Survive VM reboots
    max_bitmaps_per_vm: 3    # Limit bitmap count
    
  nbd_export:
    block_size: 1048576      # 1MB NBD block size
    max_concurrent_exports: 10 # Concurrent disk exports
    port_range: "10800-10900" # NBD port allocation
    
  performance:
    max_concurrent_vms: 10    # Concurrent VM backups per host
    bandwidth_limit_mbps: 1000 # Optional throttling
    backup_window: "22:00-06:00" # Preferred hours
```

---

## 🛠️ Troubleshooting

### **Common Issues**

**Issue 1: libvirt Connection Failed**
```bash
# Check libvirt service
systemctl status libvirtd

# Test libvirt connection
virsh list --all

# Check permissions
id sendense-user
groups sendense-user  # Should include 'libvirt' group
```

**Issue 2: Dirty Bitmap Not Working**
```bash
# Check QEMU version (need 4.0+)
qemu-system-x86_64 --version

# Test dirty bitmap manually
virsh qemu-monitor-command vm-name --hmp \
  "block-dirty-bitmap-add drive-virtio-disk0 test-bitmap"

# Check for errors
journalctl -u sendense-capture-agent | grep bitmap
```

**Issue 3: qemu-nbd Export Failed**
```bash
# Check available NBD devices
lsmod | grep nbd
modprobe nbd max_part=8

# Test manual export
qemu-nbd -t /var/lib/libvirt/images/vm.qcow2 -p 10808

# Check port conflicts
ss -tlnp | grep 10808
```

---

## 📚 Documentation

### **Admin Guide: CloudStack Integration**
1. **Prerequisites:** CloudStack version requirements, KVM host access
2. **Installation:** Agent deployment across KVM cluster
3. **Configuration:** CloudStack API credentials, network setup
4. **Monitoring:** Agent health, performance metrics
5. **Troubleshooting:** Common issues and resolution

### **API Reference**
```bash
# CloudStack VM discovery
GET /api/v1/cloudstack/vms

# Start CloudStack VM backup
POST /api/v1/backup/cloudstack
{
  "vm_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "backup_type": "incremental",
  "consistency": "application"
}

# KVM host agent status
GET /api/v1/agents/cloudstack
```

---

## 🎯 Success Criteria

### **Module Complete When:**
- ✅ CloudStack VMs can be discovered via API
- ✅ Agent deployed on all KVM hosts in cluster
- ✅ Dirty bitmap tracking operational
- ✅ Full backup completes successfully
- ✅ Incremental backup achieves 95%+ data reduction
- ✅ Performance: 3+ GiB/s throughput
- ✅ Multi-disk VM support working
- ✅ Application-consistent snapshots via CloudStack

### **Quality Gates:**
- ✅ Tested with Apache CloudStack 4.18+
- ✅ Tested with Ubuntu and RHEL KVM hosts
- ✅ Performance benchmarked under load
- ✅ Security audit passed (agent permissions)
- ✅ Documentation complete

---

## 🌟 Strategic Value

### **Bidirectional CloudStack Support**

**Phase 2 (This Module):**
- CloudStack → Backup Repository (descend)
- CloudStack → Backup Repository → Same CloudStack (ascend)

**Phase 4 (Cross-Platform Restore):**
- CloudStack → Backup Repository → VMware (ascend)
- CloudStack → Backup Repository → AWS/Azure (ascend)

**Phase 5 (Multi-Platform Replication):**
- CloudStack → VMware (transcend) - Real-time replication
- CloudStack → AWS/Azure (transcend) - Real-time replication

### **Competitive Advantage**

**Market Gap:**
- **Veeam:** No CloudStack support
- **Bacula/Bareos:** Complex, no modern GUI
- **Native CloudStack:** Basic functionality only

**Sendense Position:**
- Only modern solution for CloudStack backup
- Enterprise features (encryption, compliance, GUI)
- Cross-platform capabilities (CloudStack ↔ Any platform)
- MSP-friendly (multi-tenant, white-label)

---

## 🔗 Related Modules

- **Module 01:** VMware Source (comparison and integration)
- **Module 03:** Backup Repository (storage backend)
- **Module 04:** Restore Engine (cross-platform targets)
- **Module 05:** Replication Engine (continuous sync)

---

## 📈 Future Enhancements

**CloudStack Roadmap:**
1. **Advanced CBT:** Native CloudStack CBT (if developed by Apache)
2. **Storage Integration:** Direct Ceph/NFS backup (bypass QCOW2)
3. **Kubernetes:** CloudStack with Kubernetes support
4. **Multi-Zone:** Cross-zone replication and backup
5. **Edge Computing:** CloudStack edge deployments

---

**Module Owner:** CloudStack Engineering Team  
**Implementation Phase:** Phase 2  
**Last Updated:** October 4, 2025  
**Status:** 🟡 Planned - High Strategic Value (Bidirectional CloudStack)
