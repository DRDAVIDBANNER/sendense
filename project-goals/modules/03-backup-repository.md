# Module 03: Backup Repository & Storage Backends

**Module ID:** MOD-03  
**Status:** 🟡 **PLANNED** (Phase 1 Critical Path)  
**Priority:** Critical  
**Dependencies:** None (Foundation module)  
**Owner:** Storage Engineering Team

---

## 🎯 Module Purpose

Universal storage abstraction layer supporting local disk, cloud storage (S3, Azure Blob, Google Cloud), and immutable storage for compliance.

**Key Capabilities:**
- **QCOW2 Backup Chains:** Full + incremental with backing files
- **S3-Compatible Storage:** AWS S3, Wasabi, Backblaze B2, MinIO
- **Azure Blob Storage:** Hot/Cool/Archive tiers with lifecycle policies
- **Immutable Storage:** S3 Object Lock, Azure Immutable Blob, WORM compliance
- **Local Storage:** High-performance local disk with compression/dedup
- **Encryption:** AES-256 at rest and in transit
- **Deduplication:** Block-level and file-level deduplication

---

## 🏗️ Storage Architecture

```
┌──────────────────────────────────────────────────────────────┐
│ BACKUP REPOSITORY ABSTRACTION LAYER                         │
├──────────────────────────────────────────────────────────────┤
│                                                              │
│  ┌────────────────────────────────────────────────────────┐ │
│  │              REPOSITORY INTERFACE                      │ │
│  │                                                        │ │
│  │  type BackupRepository interface {                     │ │
│  │      Write(backupID, data) error                       │ │
│  │      Read(backupID) (Reader, error)                    │ │
│  │      Delete(backupID) error                            │ │
│  │      List(filters) ([]Metadata, error)                 │ │
│  │      SetImmutable(backupID, duration) error            │ │
│  │  }                                                     │ │
│  └────────────────────────────────────────────────────────┘ │
│                          ↕ Pluggable Backends               │
│  ┌─────────┬─────────┬─────────┬─────────┬─────────┬──────┐ │
│  │ LOCAL   │   S3    │ AZURE   │ GOOGLE  │IMMUTABLE│HYBRID│ │ │
│  │ DISK    │COMPATIBLE│ BLOB   │ CLOUD   │ WORM    │MULTI │ │
│  │         │         │         │         │         │ TIER │ │
│  │ QCOW2   │   S3    │  Blob   │   GCS   │ Legal   │ Hot+ │ │
│  │ ZFS     │ Wasabi  │ Archive │ Bucket  │ Hold    │ Cold │ │
│  │ XFS     │   B2    │ Immut   │Nearline │ S3 Lock │      │ │
│  └─────────┴─────────┴─────────┴─────────┴─────────┴──────┘ │
│                          ↕ Encryption Layer                 │
│  ┌────────────────────────────────────────────────────────┐ │
│  │              STORAGE FEATURES                          │ │
│  │                                                        │ │
│  │  • AES-256 Encryption (per-customer keys)             │ │
│  │  • Compression (zstd, lz4, gzip)                      │ │
│  │  • Deduplication (block-level, file-level)            │ │
│  │  • Retention Policies (automatic cleanup)             │ │
│  │  • Lifecycle Management (hot→warm→cold→delete)        │ │
│  │  • Geo-replication (multi-region redundancy)          │ │
│  └────────────────────────────────────────────────────────┘ │
└──────────────────────────────────────────────────────────────┘
```

---

## 💾 Local Storage Implementation

### **QCOW2 Backup Chains**

```go
type LocalRepository struct {
    basePath    string // /var/lib/sendense/backups
    compression string // "zstd", "lz4", "none"
    encryption  EncryptionConfig
}

func (lr *LocalRepository) Write(backupID string, data io.Reader) error {
    // 1. Determine backup type from metadata
    backup, err := lr.parseBackupID(backupID)
    if err != nil {
        return err
    }
    
    // 2. Create backup file path
    backupPath := lr.getBackupPath(backup)
    
    // 3. Create QCOW2 file
    if backup.Type == "incremental" {
        // Create with backing file
        parentBackup := lr.getParentBackup(backup.VMUUID)
        err = lr.createQCOW2WithBacking(backupPath, parentBackup.Path)
    } else {
        // Create full backup QCOW2
        err = lr.createQCOW2(backupPath, backup.SizeBytes)
    }
    if err != nil {
        return err
    }
    
    // 4. Write data via qemu-nbd
    nbdDevice, err := lr.attachQCOW2(backupPath)
    if err != nil {
        return err
    }
    defer lr.detachQCOW2(nbdDevice)
    
    // 5. Stream data to NBD device
    return lr.streamToDevice(data, nbdDevice)
}

func (lr *LocalRepository) createQCOW2WithBacking(newFile, backingFile string) error {
    // Create incremental QCOW2 with parent backing file
    cmd := exec.Command("qemu-img", "create", 
        "-f", "qcow2", 
        "-b", backingFile,
        "-F", "qcow2",
        newFile)
    
    return cmd.Run()
}

func (lr *LocalRepository) Read(backupID string) (io.Reader, error) {
    // 1. Find backup file
    backupPath := lr.getBackupPathFromID(backupID)
    if !fileExists(backupPath) {
        return nil, fmt.Errorf("backup not found: %s", backupID)
    }
    
    // 2. Attach QCOW2 via qemu-nbd (read-only)
    nbdDevice, err := lr.attachQCOW2ReadOnly(backupPath)
    if err != nil {
        return nil, err
    }
    
    // 3. Return reader for NBD device
    return NewNBDReader(nbdDevice), nil
}
```

---

## ☁️ S3-Compatible Storage

### **S3 Repository Implementation**

```go
type S3Repository struct {
    client      *s3.Client
    bucket      string
    prefix      string
    region      string
    encryption  S3EncryptionConfig
    immutable   ImmutableConfig
}

func (s3r *S3Repository) Write(backupID string, data io.Reader) error {
    // 1. Create S3 object key
    key := s3r.generateObjectKey(backupID)
    
    // 2. Setup encryption
    var sseConfig *s3.PutObjectInput
    if s3r.encryption.Enabled {
        sseConfig = &s3.PutObjectInput{
            ServerSideEncryption: aws.String("AES256"),
            // or use KMS: ServerSideEncryption: aws.String("aws:kms")
        }
    }
    
    // 3. Setup immutable storage (if enabled)
    var objectLock *s3.PutObjectLegalHoldInput
    if s3r.immutable.Enabled {
        objectLock = &s3.PutObjectLegalHoldInput{
            Bucket: &s3r.bucket,
            Key:    &key,
            LegalHold: &types.ObjectLockLegalHold{
                Status: types.ObjectLockLegalHoldStatusOn,
            },
        }
    }
    
    // 4. Upload to S3
    uploader := manager.NewUploader(s3r.client)
    result, err := uploader.Upload(context.TODO(), &s3.PutObjectInput{
        Bucket:               &s3r.bucket,
        Key:                  &key,
        Body:                 data,
        ServerSideEncryption: sseConfig.ServerSideEncryption,
        Metadata: map[string]string{
            "backup-id":    backupID,
            "created-by":   "sendense",
            "vm-uuid":      s3r.extractVMUUID(backupID),
            "backup-type":  s3r.extractBackupType(backupID),
        },
    })
    if err != nil {
        return err
    }
    
    // 5. Apply object lock if enabled
    if objectLock != nil {
        _, err = s3r.client.PutObjectLegalHold(context.TODO(), objectLock)
        if err != nil {
            log.Warn("Failed to apply object lock: %v", err)
        }
    }
    
    log.Info("Backup uploaded to S3",
        "backup_id", backupID,
        "s3_key", key,
        "etag", result.ETag)
    
    return nil
}

func (s3r *S3Repository) SetImmutable(backupID string, duration time.Duration) error {
    key := s3r.generateObjectKey(backupID)
    
    // S3 Object Lock with retention period
    _, err := s3r.client.PutObjectRetention(context.TODO(), &s3.PutObjectRetentionInput{
        Bucket: &s3r.bucket,
        Key:    &key,
        Retention: &types.ObjectLockRetention{
            Mode:            types.ObjectLockRetentionModeCompliance, // WORM
            RetainUntilDate: aws.Time(time.Now().Add(duration)),
        },
    })
    
    return err
}
```

**S3-Compatible Providers:**
- **AWS S3:** Native S3 (expensive but feature-complete)
- **Wasabi:** $6/TB/month hot storage (cheaper than S3)
- **Backblaze B2:** $5/TB/month (cheapest option)
- **MinIO:** Self-hosted S3-compatible (on-prem option)
- **Google Cloud Storage:** Competitive with S3
- **Azure Blob Storage:** Microsoft-native option

---

## 🔐 Immutable Storage (Anti-Ransomware)

### **WORM (Write Once, Read Many) Support**

```go
type ImmutableRepository struct {
    underlying Repository      // Wrap any repository
    retention  RetentionPolicy
    compliance ComplianceConfig
}

type RetentionPolicy struct {
    MinRetentionDays int           `json:"min_retention_days"` // Legal minimum
    MaxRetentionDays int           `json:"max_retention_days"` // Compliance maximum
    LegalHoldSupport bool          `json:"legal_hold_support"` // For litigation
    GovernanceMode   bool          `json:"governance_mode"`    // vs compliance mode
}

func (ir *ImmutableRepository) Write(backupID string, data io.Reader) error {
    // 1. Write to underlying repository
    err := ir.underlying.Write(backupID, data)
    if err != nil {
        return err
    }
    
    // 2. Apply immutability
    retention := ir.calculateRetention(backupID)
    err = ir.underlying.SetImmutable(backupID, retention)
    if err != nil {
        return err
    }
    
    // 3. Log for compliance audit trail
    ir.logImmutableWrite(backupID, retention)
    
    return nil
}

func (ir *ImmutableRepository) Delete(backupID string) error {
    // Check if still under retention
    metadata, err := ir.underlying.GetMetadata(backupID)
    if err != nil {
        return err
    }
    
    if metadata.ImmutableUntil.After(time.Now()) {
        return fmt.Errorf("backup %s is immutable until %v (WORM protection)", 
            backupID, metadata.ImmutableUntil)
    }
    
    // Only allow deletion after retention period
    return ir.underlying.Delete(backupID)
}
```

**Immutable Storage Use Cases:**
- **Ransomware Protection:** Attackers can't delete backups
- **Compliance:** HIPAA, SOX, SEC require immutable records
- **Legal Hold:** Litigation support (preserve evidence)
- **Audit Requirements:** Immutable audit trails

---

## 🌐 Multi-Tier Storage Strategy

### **Hot, Warm, Cold Storage**

```go
type HybridRepository struct {
    hotStorage    Repository // Local SSD - recent backups
    warmStorage   Repository // S3 Standard - 30-day backups  
    coldStorage   Repository // S3 Glacier - long-term retention
    archiveStorage Repository // S3 Deep Archive - compliance retention
    
    lifecycle LifecyclePolicy
}

type LifecyclePolicy struct {
    HotToWarmDays    int `json:"hot_to_warm_days"`    // 7 days
    WarmToColdDays   int `json:"warm_to_cold_days"`   // 30 days
    ColdToArchiveDays int `json:"cold_to_archive_days"` // 90 days
    ArchiveRetentionYears int `json:"archive_years"`   // 7 years
}

func (hr *HybridRepository) Write(backupID string, data io.Reader) error {
    // Always write new backups to hot storage first
    return hr.hotStorage.Write(backupID, data)
}

func (hr *HybridRepository) processLifecycle() {
    // Run daily lifecycle management
    ticker := time.NewTicker(24 * time.Hour)
    
    for range ticker.C {
        backups, err := hr.hotStorage.List(RepositoryFilter{})
        if err != nil {
            continue
        }
        
        for _, backup := range backups {
            age := time.Since(backup.CreatedAt)
            
            switch {
            case age > time.Duration(hr.lifecycle.HotToWarmDays)*24*time.Hour:
                // Move to warm storage
                err := hr.moveBackup(backup.ID, hr.hotStorage, hr.warmStorage)
                if err != nil {
                    log.Error("Failed to move backup to warm storage: %v", err)
                }
                
            case age > time.Duration(hr.lifecycle.WarmToColdDays)*24*time.Hour:
                // Move to cold storage
                err := hr.moveBackup(backup.ID, hr.warmStorage, hr.coldStorage)
                if err != nil {
                    log.Error("Failed to move backup to cold storage: %v", err)
                }
                
            case age > time.Duration(hr.lifecycle.ColdToArchiveDays)*24*time.Hour:
                // Move to archive storage
                err := hr.moveBackup(backup.ID, hr.coldStorage, hr.archiveStorage)
                if err != nil {
                    log.Error("Failed to move backup to archive storage: %v", err)
                }
            }
        }
    }
}

func (hr *HybridRepository) moveBackup(backupID string, source, dest Repository) error {
    // 1. Read from source
    data, err := source.Read(backupID)
    if err != nil {
        return err
    }
    defer data.Close()
    
    // 2. Write to destination
    err = dest.Write(backupID, data)
    if err != nil {
        return err
    }
    
    // 3. Verify integrity
    err = hr.verifyBackupIntegrity(backupID, dest)
    if err != nil {
        dest.Delete(backupID) // Cleanup failed copy
        return err
    }
    
    // 4. Delete from source
    return source.Delete(backupID)
}
```

---

## 🔒 Encryption & Security

### **Per-Customer Encryption**

```go
type EncryptionManager struct {
    masterKey    []byte // AES-256 master key (per Control Plane)
    customerKeys map[string][]byte // Per-customer derived keys
    keyRotation  KeyRotationPolicy
}

func (em *EncryptionManager) getCustomerKey(customerID string) ([]byte, error) {
    // Check cache first
    if key, exists := em.customerKeys[customerID]; exists {
        return key, nil
    }
    
    // Derive customer-specific key from master key
    salt := fmt.Sprintf("sendense-customer-%s", customerID)
    key := pbkdf2.Key(em.masterKey, []byte(salt), 100000, 32, sha256.New)
    
    // Cache for performance
    em.customerKeys[customerID] = key
    
    return key, nil
}

func (em *EncryptionManager) EncryptBackup(customerID, backupID string, data io.Reader) (io.Reader, error) {
    // Get customer-specific key
    key, err := em.getCustomerKey(customerID)
    if err != nil {
        return nil, err
    }
    
    // Create AES-256-GCM cipher
    block, err := aes.NewCipher(key)
    if err != nil {
        return nil, err
    }
    
    aesgcm, err := cipher.NewGCM(block)
    if err != nil {
        return nil, err
    }
    
    // Generate random nonce
    nonce := make([]byte, aesgcm.NonceSize())
    if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
        return nil, err
    }
    
    // Return encrypted reader
    return NewEncryptedReader(data, aesgcm, nonce), nil
}
```

---

## 📊 Storage Cost Optimization

### **Intelligent Tiering**

```go
type CostOptimizer struct {
    repositories []Repository
    policies     []OptimizationPolicy
}

type OptimizationPolicy struct {
    Name        string
    Rules       []OptimizationRule
    Savings     EstimatedSavings
}

type OptimizationRule struct {
    Condition string // "age > 30d", "access_count < 3", "size > 1GB"
    Action    string // "move_to_cold", "compress", "dedupe"
    Target    string // Target repository or storage class
}

func (co *CostOptimizer) OptimizeStorage() {
    // Daily cost optimization
    for _, policy := range co.policies {
        backups := co.findBackupsMatchingPolicy(policy)
        
        for _, backup := range backups {
            for _, rule := range policy.Rules {
                if co.evaluateCondition(backup, rule.Condition) {
                    err := co.executeOptimization(backup, rule)
                    if err != nil {
                        log.Error("Optimization failed: %v", err)
                    } else {
                        log.Info("Applied optimization",
                            "backup_id", backup.ID,
                            "rule", rule.Action,
                            "estimated_savings", rule.EstimatedSavings)
                    }
                }
            }
        }
    }
}

// Example optimization policies
var defaultOptimizationPolicies = []OptimizationPolicy{
    {
        Name: "Move to Warm Storage",
        Rules: []OptimizationRule{
            {
                Condition: "age > 7d AND access_count < 3",
                Action:    "move_to_warm",
                Target:    "s3-ia", // S3 Infrequent Access
            },
        },
    },
    {
        Name: "Move to Cold Storage", 
        Rules: []OptimizationRule{
            {
                Condition: "age > 30d",
                Action:    "move_to_cold",
                Target:    "s3-glacier",
            },
        },
    },
    {
        Name: "Archive Old Backups",
        Rules: []OptimizationRule{
            {
                Condition: "age > 365d",
                Action:    "move_to_archive", 
                Target:    "s3-deep-archive",
            },
        },
    },
}
```

### **Cost Analytics Dashboard**

```
Storage Cost Analytics:
┌─────────────────────────────────────────────────────────┐
│              Monthly Storage Costs                      │
├─────────────────────────────────────────────────────────┤
│ Total Storage: 15.7 TB        Monthly Cost: $1,247     │
│ Growth Rate: +8.3% MoM        Projected: $1,351 next   │
│                                                         │
│ By Storage Tier:                                        │
│ 🔥 Hot (Local SSD):     2.3 TB  $230   (15%)          │
│ 🟡 Warm (S3 Standard):  5.1 TB  $512   (33%)          │
│ 🧊 Cold (S3 Glacier):   6.8 TB  $408   (33%)          │
│ 📦 Archive (Deep):       1.5 TB   $97   (8%)           │
│                                                         │
│ 💡 Optimization Opportunities:                          │
│ • Move 890 GB to cold storage → Save $67/month         │
│ • Enable compression on 2.1 TB → Save $84/month        │ │
│ • Archive 340 GB old backups → Save $23/month         │
│                                                         │
│ [Apply Optimizations] [Export Report] [Configure]      │
└─────────────────────────────────────────────────────────┘
```

---

## 🗃️ Deduplication Engine

### **Block-Level Deduplication**

```go
type DeduplicationEngine struct {
    hashIndex    map[string][]BlockLocation // SHA256 → block locations
    blockSize    int                        // 64KB typical
    threshold    float64                    // 0.8 (80% similarity)
    repository   Repository
}

func (de *DeduplicationEngine) Write(backupID string, data io.Reader) error {
    // 1. Split data into blocks
    blocks := de.splitIntoBlocks(data, de.blockSize)
    
    // 2. Calculate hash for each block
    var uniqueBlocks []Block
    var references []BlockReference
    
    for i, block := range blocks {
        hash := sha256.Sum256(block)
        hashStr := fmt.Sprintf("%x", hash)
        
        // 3. Check if block already exists
        if locations, exists := de.hashIndex[hashStr]; exists {
            // Block already stored, just reference it
            references = append(references, BlockReference{
                BackupID:     backupID,
                BlockIndex:   i,
                ReferencedHash: hashStr,
                Location:     locations[0], // Use first available location
            })
        } else {
            // New block, store it
            blockLocation := BlockLocation{
                Repository: de.repository,
                Path:       fmt.Sprintf("blocks/%s", hashStr),
            }
            
            err := de.repository.WriteBlock(blockLocation.Path, block)
            if err != nil {
                return err
            }
            
            uniqueBlocks = append(uniqueBlocks, Block{
                Hash:     hashStr,
                Data:     block,
                Location: blockLocation,
            })
            
            // Update hash index
            de.hashIndex[hashStr] = append(de.hashIndex[hashStr], blockLocation)
        }
    }
    
    // 4. Store backup metadata (block references)
    metadata := BackupMetadata{
        BackupID:       backupID,
        TotalBlocks:    len(blocks),
        UniqueBlocks:   len(uniqueBlocks),
        DedupeRatio:    float64(len(uniqueBlocks)) / float64(len(blocks)),
        BlockReferences: references,
    }
    
    return de.repository.WriteMetadata(backupID, metadata)
}

func (de *DeduplicationEngine) Read(backupID string) (io.Reader, error) {
    // 1. Load backup metadata
    metadata, err := de.repository.ReadMetadata(backupID)
    if err != nil {
        return nil, err
    }
    
    // 2. Reconstruct data from block references
    return NewDeduplicatedReader(metadata.BlockReferences, de.repository), nil
}
```

**Deduplication Benefits:**
- **Space Savings:** 60-90% typical reduction
- **Cost Savings:** Especially important for cloud storage
- **Transfer Efficiency:** Don't upload duplicate blocks
- **Global Dedup:** Across all VMs and customers (with privacy)

---

## 🎯 Repository Selection Logic

### **Intelligent Repository Selection**

```go
type RepositorySelector struct {
    repositories []Repository
    policies     []SelectionPolicy
    costModel    CostModel
}

func (rs *RepositorySelector) SelectOptimalRepository(backup BackupRequest) (Repository, error) {
    // Score each repository based on multiple factors
    scores := make(map[string]float64)
    
    for _, repo := range rs.repositories {
        score := 0.0
        
        // Factor 1: Performance requirements
        if backup.RequiresHighIOPS {
            if repo.Type() == "local" {
                score += 30 // Local storage wins for performance
            } else if repo.Type() == "s3" {
                score += 10 // S3 is slower
            }
        }
        
        // Factor 2: Cost optimization
        cost := rs.costModel.EstimateCost(backup.SizeGB, repo)
        if cost < backup.MaxAcceptableCost {
            score += 25
        }
        
        // Factor 3: Compliance requirements
        if backup.RequiresImmutable && repo.SupportsImmutable() {
            score += 20
        }
        
        // Factor 4: Geographic requirements
        if backup.DataResidency != "" {
            if repo.Region() == backup.DataResidency {
                score += 15
            }
        }
        
        // Factor 5: Redundancy requirements
        if backup.RequiresRedundancy && repo.Redundancy() >= backup.RequiredRedundancy {
            score += 10
        }
        
        scores[repo.ID()] = score
    }
    
    // Select highest scoring repository
    bestRepo, bestScore := rs.findBestRepository(scores)
    if bestScore < 50 { // Minimum acceptable score
        return nil, fmt.Errorf("no repository meets requirements for backup %s", backup.ID)
    }
    
    return bestRepo, nil
}
```

**Selection Criteria:**
- **Performance:** Local > S3 > Glacier for speed
- **Cost:** Glacier > S3 > Local for long-term storage
- **Compliance:** Immutable > Standard for legal requirements
- **Geography:** Region matching for data residency laws

---

This module is the foundation for everything - without solid storage, the whole platform falls apart. The multi-tier approach (local hot storage → S3 warm → Glacier cold) gives you cost optimization while maintaining performance.

Let me create the other missing modules next - Hyper-V source, AWS source, and the core restore engine module. Those were definitely mentioned in our discussion!
