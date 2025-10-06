# Module 06: Hyper-V Source Connector

**Module ID:** MOD-06  
**Status:** 🟡 **PLANNED** (Phase 5B)  
**Priority:** High  
**Dependencies:** Module 01 (VMware Source pattern)  
**Owner:** Windows Platform Team

---

## 🎯 Module Purpose

Capture data from Microsoft Hyper-V environments using RCT (Resilient Change Tracking) for efficient incremental backups and replication.

**Key Capabilities:**
- Full VM backup from Hyper-V hosts
- Incremental backup using Hyper-V RCT
- Live VM replication from Hyper-V to any target platform
- Application-consistent snapshots via VSS integration
- Multi-disk VM support with VHD/VHDX handling

**Strategic Value:**
- **Windows Market:** Hyper-V widely deployed in Microsoft environments
- **Cross-Platform Escape:** Hyper-V → CloudStack/VMware migration path
- **RCT Technology:** 64KB granularity change tracking (finer than VMware CBT)

---

## 🔧 RCT (Resilient Change Tracking) Technology

### **RCT vs VMware CBT Comparison**

| Feature | VMware CBT | Hyper-V RCT |
|---------|------------|--------------|
| **Granularity** | 256KB blocks | 64KB blocks (finer) |
| **Performance Impact** | <1% | <1% |
| **Persistence** | Across reboots | Across reboots |
| **API Access** | VDDK | PowerShell cmdlets |
| **Reset Method** | Snapshot | Reset-VMRCT |
| **OS Support** | Any guest | Windows/Linux |
| **Live Migration** | vMotion aware | Live Migration aware |

### **RCT Implementation**

```go
type RCTManager struct {
    psClient     *powershell.Client
    rctDirectory string
}

func (rct *RCTManager) EnableRCT(vmName string) error {
    rctFile := filepath.Join(rct.rctDirectory, vmName + ".rct")
    
    psScript := fmt.Sprintf(`
        Enable-VMRCT -VMName "%s" -RCTFile "%s"
        if ($?) {
            Write-Output "SUCCESS: RCT enabled"
        } else {
            Write-Error "FAILED: $($Error[0].Exception.Message)"
        }
    `, vmName, rctFile)
    
    result, err := rct.psClient.Execute(psScript)
    if err != nil {
        return err
    }
    
    if result.ExitCode != 0 {
        return fmt.Errorf("RCT enable failed: %s", result.Stderr)
    }
    
    return nil
}

func (rct *RCTManager) GetChangedBlocks(vmName, sinceRCTId string) ([]ChangedBlock, error) {
    psScript := fmt.Sprintf(`
        $changes = Get-VMChangedBlocks -VMName "%s" -SinceRCT "%s"
        $changes | ForEach-Object {
            [PSCustomObject]@{
                Offset = $_.Offset
                Length = $_.Length  
                Type = "changed"
            }
        } | ConvertTo-Json -AsArray
    `, vmName, sinceRCTId)
    
    result, err := rct.psClient.Execute(psScript)
    if err != nil {
        return nil, err
    }
    
    var blocks []ChangedBlock
    err = json.Unmarshal([]byte(result.Stdout), &blocks)
    return blocks, err
}
```

**Strategic Value:** Hyper-V shops looking to escape Microsoft ecosystem can replicate to CloudStack/VMware.

---

**Module Owner:** Windows Engineering Team  
**Last Updated:** October 4, 2025

