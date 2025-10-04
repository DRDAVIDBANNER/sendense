# Module 10: Azure VM Source Connector

**Module ID:** MOD-10  
**Status:** 🟡 **PLANNED** (Phase 5D)  
**Priority:** High  
**Dependencies:** Module 01 (Architecture pattern)  
**Owner:** Cloud Platform Team

---

## 🎯 Module Purpose

Capture data from Azure VMs using Managed Disk snapshots and incremental snapshot tracking for efficient backups and replication.

**Key Capabilities:**
- Full VM backup from Azure VMs
- Incremental backup using Azure incremental snapshots
- Live VM replication from Azure to any target platform
- Cross-region backup and replication
- Azure Resource Manager integration

**Strategic Value:**
- **Azure Escape:** Azure → On-prem migration path
- **Cost Optimization:** Replicate Azure VMs to cheaper platforms
- **Multi-Cloud:** Azure → AWS migration capability

---

## 🔧 Azure Managed Disk Change Tracking

```go
type AzureChangeTracker struct {
    computeClient  *armcompute.VirtualMachinesClient
    diskClient     *armcompute.DisksClient
    snapshotClient *armcompute.SnapshotsClient
}

func (act *AzureChangeTracker) CreateIncrementalSnapshot(diskID, baseSnapshotID string) (*Snapshot, error) {
    snapshotName := fmt.Sprintf("sendense-incr-%d", time.Now().Unix())
    
    snapshot := armcompute.Snapshot{
        Location: to.Ptr(act.region),
        Properties: &armcompute.SnapshotProperties{
            CreationData: &armcompute.CreationData{
                CreateOption:     to.Ptr(armcompute.DiskCreateOptionIncremental),
                SourceResourceID: to.Ptr(diskID),
                SourceUniqueID:   to.Ptr(baseSnapshotID),
            },
            Incremental: to.Ptr(true),
        },
    }
    
    poller, err := act.snapshotClient.BeginCreateOrUpdate(
        context.TODO(),
        act.resourceGroup,
        snapshotName,
        snapshot,
        nil)
    
    if err != nil {
        return nil, err
    }
    
    result, err := poller.PollUntilDone(context.TODO(), nil)
    if err != nil {
        return nil, err
    }
    
    return &Snapshot{
        ID:           *result.ID,
        Name:         *result.Name,
        CreationTime: *result.Properties.TimeCreated,
        SourceDisk:   diskID,
        Incremental:  true,
    }, nil
}
```

**Strategic Value:** Azure customers can escape Microsoft cloud lock-in by replicating to on-prem CloudStack.

---

**Module Owner:** Azure Engineering Team  
**Last Updated:** October 4, 2025
