# Module 07: AWS EC2 Source Connector

**Module ID:** MOD-07  
**Status:** 🟡 **PLANNED** (Phase 5C)  
**Priority:** High  
**Dependencies:** Module 01 (Architecture pattern)  
**Owner:** Cloud Platform Team

---

## 🎯 Module Purpose

Capture data from AWS EC2 instances using EBS snapshots and changed block tracking for efficient incremental backups and replication.

**Key Capabilities:**
- Full VM backup from AWS EC2 instances
- Incremental backup using EBS Changed Block Tracking API
- Live VM replication from AWS to any target platform
- Cross-region backup and replication
- Multi-volume EC2 instance support

**Strategic Value:**
- **Cloud Escape:** AWS → On-prem migration path (avoid cloud lock-in)
- **Cost Optimization:** Replicate to cheaper platforms for DR
- **Compliance:** Backup AWS data to on-prem for data sovereignty

---

## 🔧 EBS Changed Block Tracking

### **AWS Native Change Tracking**

```go
type EBSChangeTracker struct {
    ec2Client *ec2.Client
    region    string
}

func (ect *EBSChangeTracker) GetChangedBlocks(volumeID, baseSnapshot, currentSnapshot string) ([]ChangedBlock, error) {
    // Use AWS EBS ListChangedBlocks API
    input := &ec2.ListChangedBlocksInput{
        FirstSnapshotId:  &baseSnapshot,
        SecondSnapshotId: &currentSnapshot,
        MaxResults:       aws.Int32(10000),
    }
    
    result, err := ect.ec2Client.ListChangedBlocks(context.TODO(), input)
    if err != nil {
        return nil, err
    }
    
    var changedBlocks []ChangedBlock
    for _, block := range result.ChangedBlocks {
        changedBlocks = append(changedBlocks, ChangedBlock{
            Offset: *block.BlockIndex * 512 * 1024, // EBS uses 512KB blocks
            Length: 512 * 1024,
            Type:   "changed",
        })
    }
    
    return changedBlocks, nil
}

func (ect *EBSChangeTracker) CreateSnapshot(volumeID string) (*Snapshot, error) {
    input := &ec2.CreateSnapshotInput{
        VolumeId:    &volumeID,
        Description: aws.String("Sendense incremental backup snapshot"),
        TagSpecifications: []types.TagSpecification{
            {
                ResourceType: types.ResourceTypeSnapshot,
                Tags: []types.Tag{
                    {Key: aws.String("CreatedBy"), Value: aws.String("Sendense")},
                    {Key: aws.String("Purpose"), Value: aws.String("Backup")},
                },
            },
        },
    }
    
    result, err := ect.ec2Client.CreateSnapshot(context.TODO(), input)
    if err != nil {
        return nil, err
    }
    
    return &Snapshot{
        ID:       *result.SnapshotId,
        VolumeID: volumeID,
        State:    string(result.State),
        StartTime: *result.StartTime,
    }, nil
}
```

**Strategic Value:** Enable cloud exit strategy - backup AWS to on-prem, replicate AWS to CloudStack for cost savings.

---

**Module Owner:** Cloud Engineering Team  
**Last Updated:** October 4, 2025
