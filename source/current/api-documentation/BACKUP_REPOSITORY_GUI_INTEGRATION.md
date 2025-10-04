# Backup Repository GUI Integration Guide

**Version:** 1.0  
**Phase:** Phase 1 - VMware Backup Implementation  
**Task:** 1.1 Backup Repository Abstraction  
**Date:** 2025-10-04  

## Overview

This document outlines how the Sendense Control Appliance (SCA) GUI interacts with the backup repository system. The GUI provides administrators with the ability to manage multiple backup repositories, configure backup policies, monitor storage capacity, and manage backup copies across multiple locations.

## Architecture Summary

```
┌─────────────────────────────────────────────────────────────┐
│                     SCA Web GUI (React)                      │
│  ┌────────────────┐  ┌────────────────┐  ┌────────────────┐ │
│  │  Repository    │  │ Backup Policy  │  │ Backup Copy    │ │
│  │  Management    │  │  Management    │  │  Management    │ │
│  └────────────────┘  └────────────────┘  └────────────────┘ │
└───────────────────────────────┬─────────────────────────────┘
                                │ REST API (JSON)
┌───────────────────────────────┴─────────────────────────────┐
│             OMA API Server (Go Backend)                      │
│  ┌────────────────┐  ┌────────────────┐  ┌────────────────┐ │
│  │  Repository    │  │ Backup Policy  │  │ Backup Copy    │ │
│  │   Manager      │  │   Service      │  │    Engine      │ │
│  └────────────────┘  └────────────────┘  └────────────────┘ │
└───────────────────────────────┬─────────────────────────────┘
                                │
                       ┌────────┴────────┐
                       │  MariaDB        │
                       │  (backup_*)     │
                       └─────────────────┘
```

## API Endpoints

### Repository Management

#### 1. List Repositories
```
GET /api/v1/repositories
```

**Response:**
```json
{
  "success": true,
  "repositories": [
    {
      "id": "repo-local-1638547200",
      "name": "Primary Local Storage",
      "type": "local",
      "enabled": true,
      "is_immutable": false,
      "config": {
        "path": "/mnt/backups/primary"
      },
      "storage": {
        "total_bytes": 10737418240000,
        "used_bytes": 4294967296000,
        "available_bytes": 6442450944000,
        "used_percentage": 40.0,
        "last_check_at": "2025-10-04T15:30:00Z"
      },
      "created_at": "2025-10-01T10:00:00Z",
      "updated_at": "2025-10-04T15:30:00Z"
    },
    {
      "id": "repo-nfs-1638547300",
      "name": "Secondary NFS Storage",
      "type": "nfs",
      "enabled": true,
      "is_immutable": true,
      "config": {
        "server": "nfs.example.com",
        "export_path": "/exports/backups",
        "mount_options": ["ro", "nolock"]
      },
      "immutable_config": {
        "method": "linux_chattr",
        "retention_period_days": 30,
        "locked": true
      },
      "min_retention_days": 30,
      "storage": {
        "total_bytes": 5368709120000,
        "used_bytes": 1073741824000,
        "available_bytes": 4294967296000,
        "used_percentage": 20.0,
        "last_check_at": "2025-10-04T15:30:00Z"
      },
      "created_at": "2025-10-02T12:00:00Z",
      "updated_at": "2025-10-04T15:30:00Z"
    }
  ]
}
```

#### 2. Register New Repository
```
POST /api/v1/repositories
```

**Request Body:**
```json
{
  "name": "Tertiary CIFS Storage",
  "type": "cifs",
  "enabled": true,
  "is_immutable": false,
  "config": {
    "server": "10.0.100.50",
    "share_name": "backups",
    "domain": "EXAMPLE",
    "username": "backup_user",
    "password": "secure_password"
  }
}
```

**Response:**
```json
{
  "success": true,
  "repository_id": "repo-cifs-1638547400",
  "message": "Repository registered successfully",
  "storage": {
    "total_bytes": 2147483648000,
    "available_bytes": 2147483648000,
    "used_percentage": 0.0
  }
}
```

#### 3. Test Repository Connection
```
POST /api/v1/repositories/test
```

**Request Body:**
```json
{
  "type": "nfs",
  "config": {
    "server": "nfs.example.com",
    "export_path": "/exports/backups"
  }
}
```

**Response:**
```json
{
  "success": true,
  "message": "Repository connection successful",
  "storage": {
    "total_bytes": 5368709120000,
    "available_bytes": 4294967296000,
    "writable": true
  }
}
```

#### 4. Update Repository
```
PATCH /api/v1/repositories/{id}
```

**Request Body:**
```json
{
  "enabled": false,
  "name": "Old NFS Storage (Disabled)"
}
```

**Response:**
```json
{
  "success": true,
  "message": "Repository updated successfully"
}
```

#### 5. Delete Repository
```
DELETE /api/v1/repositories/{id}
```

**Response:**
```json
{
  "success": true,
  "message": "Repository deleted successfully"
}
```

**Error Response (if backups exist):**
```json
{
  "success": false,
  "error": "cannot delete repository with 42 existing backups",
  "backup_count": 42
}
```

#### 6. Refresh Storage Info
```
POST /api/v1/repositories/refresh-storage
```

**Response:**
```json
{
  "success": true,
  "message": "Storage information refreshed for 3 repositories",
  "refreshed_count": 3,
  "failed_count": 0
}
```

### Backup Policy Management

#### 7. List Backup Policies
```
GET /api/v1/backup-policies
```

**Response:**
```json
{
  "success": true,
  "policies": [
    {
      "id": "policy-production-1638547500",
      "name": "Production VMs - 3-2-1",
      "enabled": true,
      "primary_repository_id": "repo-local-1638547200",
      "primary_repository_name": "Primary Local Storage",
      "retention_days": 30,
      "copy_rules": [
        {
          "id": "copy-rule-1638547600",
          "destination_repository_id": "repo-nfs-1638547300",
          "destination_repository_name": "Secondary NFS Storage",
          "copy_mode": "immediate",
          "priority": 1,
          "enabled": true,
          "verify_after_copy": true
        },
        {
          "id": "copy-rule-1638547700",
          "destination_repository_id": "repo-cifs-1638547400",
          "destination_repository_name": "Tertiary CIFS Storage",
          "copy_mode": "scheduled",
          "priority": 2,
          "enabled": true,
          "verify_after_copy": true
        }
      ],
      "vm_count": 25,
      "created_at": "2025-10-01T10:00:00Z",
      "updated_at": "2025-10-04T15:30:00Z"
    }
  ]
}
```

#### 8. Create Backup Policy
```
POST /api/v1/backup-policies
```

**Request Body:**
```json
{
  "name": "Test VMs - Single Copy",
  "enabled": true,
  "primary_repository_id": "repo-local-1638547200",
  "retention_days": 7,
  "copy_rules": []
}
```

**Response:**
```json
{
  "success": true,
  "policy_id": "policy-test-1638547800",
  "message": "Backup policy created successfully"
}
```

#### 9. Add Copy Rule to Policy
```
POST /api/v1/backup-policies/{policy_id}/copy-rules
```

**Request Body:**
```json
{
  "destination_repository_id": "repo-nfs-1638547300",
  "copy_mode": "immediate",
  "priority": 1,
  "enabled": true,
  "verify_after_copy": true
}
```

**Response:**
```json
{
  "success": true,
  "copy_rule_id": "copy-rule-1638547900",
  "message": "Copy rule added successfully"
}
```

#### 10. Update Backup Policy
```
PATCH /api/v1/backup-policies/{policy_id}
```

**Request Body:**
```json
{
  "enabled": false,
  "retention_days": 14
}
```

**Response:**
```json
{
  "success": true,
  "message": "Backup policy updated successfully"
}
```

#### 11. Delete Backup Policy
```
DELETE /api/v1/backup-policies/{policy_id}
```

**Response:**
```json
{
  "success": true,
  "message": "Backup policy deleted successfully"
}
```

### Backup Job Management

#### 12. Create Backup Job
```
POST /api/v1/backups
```

**Request Body:**
```json
{
  "vm_context_id": "ctx-pgtest1-20251004-153000",
  "repository_id": "repo-local-1638547200",
  "policy_id": "policy-production-1638547500",
  "backup_type": "full"
}
```

**Response:**
```json
{
  "success": true,
  "backup_job_id": "backup-20251004-153045-pgtest1",
  "message": "Backup job created successfully",
  "estimated_size_bytes": 42949672960
}
```

#### 13. Get Backup Job Status
```
GET /api/v1/backups/{backup_job_id}
```

**Response:**
```json
{
  "success": true,
  "backup": {
    "id": "backup-20251004-153045-pgtest1",
    "vm_context_id": "ctx-pgtest1-20251004-153000",
    "vm_name": "pgtest1",
    "repository_id": "repo-local-1638547200",
    "policy_id": "policy-production-1638547500",
    "backup_type": "full",
    "status": "running",
    "repository_path": "/mnt/backups/primary/pgtest1/backup-20251004-153045-pgtest1.qcow2",
    "bytes_transferred": 21474836480,
    "total_bytes": 42949672960,
    "progress_percentage": 50.0,
    "compression_enabled": true,
    "created_at": "2025-10-04T15:30:45Z",
    "started_at": "2025-10-04T15:30:50Z"
  }
}
```

#### 14. List Backups for VM
```
GET /api/v1/backups?vm_context_id={vm_context_id}
```

**Response:**
```json
{
  "success": true,
  "backups": [
    {
      "id": "backup-20251004-120000-pgtest1",
      "backup_type": "full",
      "status": "completed",
      "repository_name": "Primary Local Storage",
      "bytes_transferred": 42949672960,
      "compression_enabled": true,
      "created_at": "2025-10-04T12:00:00Z",
      "completed_at": "2025-10-04T12:15:30Z",
      "copies": [
        {
          "repository_name": "Secondary NFS Storage",
          "status": "completed",
          "verified_at": "2025-10-04T12:20:00Z"
        }
      ]
    },
    {
      "id": "backup-20251004-153045-pgtest1",
      "backup_type": "full",
      "status": "running",
      "repository_name": "Primary Local Storage",
      "bytes_transferred": 21474836480,
      "total_bytes": 42949672960,
      "progress_percentage": 50.0,
      "compression_enabled": true,
      "created_at": "2025-10-04T15:30:45Z",
      "started_at": "2025-10-04T15:30:50Z"
    }
  ]
}
```

#### 15. Get Backup Chain
```
GET /api/v1/backups/chain?vm_context_id={vm_context_id}&disk_id={disk_id}
```

**Response:**
```json
{
  "success": true,
  "chain": {
    "id": "chain-pgtest1-disk0",
    "vm_context_id": "ctx-pgtest1-20251004-153000",
    "disk_id": 0,
    "full_backup_id": "backup-20251004-120000-pgtest1",
    "latest_backup_id": "backup-20251004-153045-pgtest1",
    "total_backups": 5,
    "total_size_bytes": 214748364800,
    "backups": [
      {
        "id": "backup-20251004-120000-pgtest1",
        "backup_type": "full",
        "parent_backup_id": null,
        "size_bytes": 42949672960,
        "created_at": "2025-10-04T12:00:00Z"
      },
      {
        "id": "backup-20251004-130000-pgtest1",
        "backup_type": "incremental",
        "parent_backup_id": "backup-20251004-120000-pgtest1",
        "size_bytes": 4294967296,
        "created_at": "2025-10-04T13:00:00Z"
      }
    ]
  }
}
```

### Backup Copy Management

#### 16. List Backup Copies
```
GET /api/v1/backup-copies?source_backup_id={backup_id}
```

**Response:**
```json
{
  "success": true,
  "copies": [
    {
      "id": "copy-1638548000",
      "source_backup_id": "backup-20251004-120000-pgtest1",
      "repository_id": "repo-nfs-1638547300",
      "repository_name": "Secondary NFS Storage",
      "status": "completed",
      "file_path": "/exports/backups/pgtest1/backup-20251004-120000-pgtest1.qcow2",
      "size_bytes": 42949672960,
      "copy_started_at": "2025-10-04T12:15:35Z",
      "copy_completed_at": "2025-10-04T12:20:00Z",
      "verified_at": "2025-10-04T12:20:15Z",
      "verification_status": "passed"
    }
  ]
}
```

#### 17. Manually Trigger Backup Copy
```
POST /api/v1/backup-copies
```

**Request Body:**
```json
{
  "source_backup_id": "backup-20251004-120000-pgtest1",
  "destination_repository_id": "repo-cifs-1638547400",
  "verify_after_copy": true
}
```

**Response:**
```json
{
  "success": true,
  "copy_id": "copy-1638548100",
  "message": "Backup copy initiated successfully"
}
```

#### 18. Verify Backup Copy
```
POST /api/v1/backup-copies/{copy_id}/verify
```

**Response:**
```json
{
  "success": true,
  "verification_status": "passed",
  "message": "Backup copy verification successful"
}
```

### Storage Monitoring

#### 19. Get Storage Summary
```
GET /api/v1/repositories/storage-summary
```

**Response:**
```json
{
  "success": true,
  "summary": {
    "total_repositories": 3,
    "enabled_repositories": 2,
    "total_capacity_bytes": 18253611008000,
    "total_used_bytes": 5368709120000,
    "total_available_bytes": 12884901888000,
    "overall_used_percentage": 29.4,
    "repositories": [
      {
        "id": "repo-local-1638547200",
        "name": "Primary Local Storage",
        "type": "local",
        "used_percentage": 40.0,
        "health_status": "healthy"
      },
      {
        "id": "repo-nfs-1638547300",
        "name": "Secondary NFS Storage",
        "type": "nfs",
        "used_percentage": 20.0,
        "health_status": "healthy"
      }
    ]
  }
}
```

## GUI Components

### 1. Repository Management Page

**Location:** `/settings/repositories`

**Features:**
- List all configured repositories with storage capacity bars
- Add new repository button with wizard
- Edit/disable/delete repository actions
- Test connection button
- Refresh storage info button
- Storage capacity visualization (pie charts, bar graphs)
- Repository type badges (Local, NFS, CIFS, S3, Azure)
- Immutable storage indicator badge

**React Component Structure:**
```typescript
<RepositoryManagementPage>
  <PageHeader title="Backup Repositories" />
  <StorageSummaryCards />  {/* Total capacity, used space, available space */}
  <RepositoryTable>
    <RepositoryRow>
      <RepositoryInfo>
        <TypeBadge type="local" />
        <ImmutableBadge is_immutable={true} />
      </RepositoryInfo>
      <StorageCapacityBar />
      <ActionButtons>
        <TestButton />
        <EditButton />
        <DisableButton />
        <DeleteButton />
      </ActionButtons>
    </RepositoryRow>
  </RepositoryTable>
  <AddRepositoryButton onClick={openWizard} />
</RepositoryManagementPage>
```

**Add Repository Wizard:**
```typescript
<AddRepositoryWizard>
  <Step1: SelectType>
    {/* Radio buttons: Local, NFS, CIFS/SMB, S3 (future), Azure (future) */}
  </Step1>
  <Step2: Configure>
    {/* Dynamic form based on type selection */}
    {/* For Local: path input */}
    {/* For NFS: server, export_path, mount_options */}
    {/* For CIFS: server, share_name, domain, username, password */}
  </Step2>
  <Step3: ImmutableStorage>
    {/* Checkbox: Enable immutable storage */}
    {/* If enabled: retention_period_days input */}
  </Step3>
  <Step4: TestConnection>
    {/* Test button with loading spinner */}
    {/* Display test results (success/failure, capacity info) */}
  </Step4>
  <Step5: Review>
    {/* Display summary of configuration */}
    {/* Confirm and save button */}
  </Step5>
</AddRepositoryWizard>
```

### 2. Backup Policy Management Page

**Location:** `/settings/backup-policies`

**Features:**
- List all backup policies
- Create new policy wizard
- Edit/disable/delete policy actions
- View VMs assigned to each policy
- Configure copy rules (3-2-1 rule visualization)
- Retention period settings

**React Component Structure:**
```typescript
<BackupPolicyManagementPage>
  <PageHeader title="Backup Policies" />
  <PolicyTable>
    <PolicyRow>
      <PolicyInfo>
        <PolicyName />
        <PrimaryRepository />
        <RetentionDays />
      </PolicyInfo>
      <CopyRulesList>
        {/* Visual 3-2-1 rule representation */}
        <CopyRuleBadge priority={1} destination="NFS" mode="immediate" />
        <CopyRuleBadge priority={2} destination="CIFS" mode="scheduled" />
      </CopyRulesList>
      <VMCount count={25} />
      <ActionButtons>
        <EditButton />
        <DisableButton />
        <DeleteButton />
      </ActionButtons>
    </PolicyRow>
  </PolicyTable>
  <CreatePolicyButton onClick={openWizard} />
</BackupPolicyManagementPage>
```

**Create Policy Wizard:**
```typescript
<CreatePolicyWizard>
  <Step1: BasicInfo>
    {/* Name input */}
    {/* Primary repository dropdown */}
    {/* Retention days input */}
  </Step1>
  <Step2: CopyRules>
    {/* Add copy rule button */}
    {/* List of copy rules with priority ordering */}
    <CopyRuleForm>
      {/* Destination repository dropdown */}
      {/* Copy mode: immediate/scheduled/manual */}
      {/* Priority input */}
      {/* Verify after copy checkbox */}
    </CopyRuleForm>
  </Step2>
  <Step3: AssignVMs>
    {/* Multi-select VM list */}
    {/* Or select machine groups */}
  </Step3>
  <Step4: Review>
    {/* Display policy summary */}
    {/* Confirm and save button */}
  </Step4>
</CreatePolicyWizard>
```

### 3. VM Backup Management (Enhanced VM Detail View)

**Location:** `/virtual-machines/{vm_name}/backups`

**Features:**
- List all backups for this VM
- Create manual backup button
- Backup timeline visualization
- Backup chain diagram (full → incremental → incremental)
- Restore buttons (future: Phase 4)
- Copy status indicators
- Storage space used by backups

**React Component Structure:**
```typescript
<VMBackupManagement vm_context_id={contextId}>
  <BackupSummary>
    <LatestBackupCard />
    <BackupCountCard />
    <TotalStorageCard />
  </BackupSummary>
  <BackupTimeline>
    {/* Visual timeline of backups */}
    <TimelineEntry type="full" date="2025-10-04 12:00" />
    <TimelineEntry type="incremental" date="2025-10-04 13:00" />
    <TimelineEntry type="incremental" date="2025-10-04 14:00" />
  </BackupTimeline>
  <BackupTable>
    <BackupRow>
      <BackupInfo>
        <TypeBadge type="full" />
        <DateCreated />
        <SizeDisplay />
      </BackupInfo>
      <BackupChain>
        {/* Shows parent → child relationships */}
      </BackupChain>
      <CopyStatus>
        <CopyBadge repository="NFS" status="completed" verified={true} />
        <CopyBadge repository="CIFS" status="copying" verified={false} />
      </CopyStatus>
      <ActionButtons>
        <RestoreButton disabled={true} tooltip="Phase 4" />
        <CopyToButton />
        <DeleteButton />
      </ActionButtons>
    </BackupRow>
  </BackupTable>
  <CreateBackupButton onClick={openDialog} />
</VMBackupManagement>
```

**Create Backup Dialog:**
```typescript
<CreateBackupDialog>
  <BackupTypeSelector>
    {/* Radio buttons: Full, Incremental, Differential */}
    {/* Disabled if no full backup exists for incremental */}
  </BackupTypeSelector>
  <PolicySelector>
    {/* Dropdown: Select backup policy */}
    {/* Or manual repository selection */}
  </PolicySelector>
  <CompressionCheckbox />
  <EstimatedSize />
  <CreateButton onClick={submitBackup} />
</CreateBackupDialog>
```

### 4. Storage Monitoring Dashboard

**Location:** `/monitoring/storage`

**Features:**
- Real-time storage capacity monitoring
- Repository health status
- Backup growth trends
- Copy operation status
- Immutable storage compliance
- Alerts for low storage space

**React Component Structure:**
```typescript
<StorageMonitoringDashboard>
  <StorageOverviewCards>
    <TotalCapacityCard />
    <UsedSpaceCard />
    <AvailableSpaceCard />
    <HealthScoreCard />
  </StorageOverviewCards>
  <RepositoryHealthTable>
    <RepositoryHealthRow>
      <RepositoryName />
      <HealthIndicator status="healthy" />
      <CapacityGauge percentage={40} />
      <LastChecked />
      <RefreshButton />
    </RepositoryHealthRow>
  </RepositoryHealthTable>
  <BackupGrowthChart>
    {/* Line chart showing backup storage over time */}
  </BackupGrowthChart>
  <ActiveOperationsPanel>
    {/* List of in-progress backup jobs */}
    {/* List of in-progress copy operations */}
  </ActiveOperationsPanel>
  <AlertsPanel>
    {/* Low storage space warnings */}
    {/* Failed copy operation alerts */}
    {/* Immutable storage compliance issues */}
  </AlertsPanel>
</StorageMonitoringDashboard>
```

### 5. Backup Copy Monitor

**Location:** `/monitoring/backup-copies`

**Features:**
- List all backup copy operations
- Filter by status (pending, copying, verifying, completed, failed)
- Retry failed copies
- Manual verification trigger
- Copy operation progress

**React Component Structure:**
```typescript
<BackupCopyMonitor>
  <FilterBar>
    <StatusFilter options={['all', 'pending', 'copying', 'verifying', 'completed', 'failed']} />
    <RepositoryFilter />
    <DateRangeFilter />
  </FilterBar>
  <CopyOperationsTable>
    <CopyOperationRow>
      <SourceBackup />
      <DestinationRepository />
      <Status>
        <StatusBadge status="copying" />
        <ProgressBar percentage={65} />
      </Status>
      <Verification>
        <VerificationBadge status="pending" />
        <VerifyButton />
      </Verification>
      <Timestamps>
        <StartedAt />
        <CompletedAt />
      </Timestamps>
      <ActionButtons>
        <RetryButton />
        <CancelButton />
      </ActionButtons>
    </CopyOperationRow>
  </CopyOperationsTable>
</BackupCopyMonitor>
```

## Real-Time Updates (WebSocket)

The GUI should establish a WebSocket connection for real-time updates on:

1. **Backup Job Progress:**
   - Subscribe: `subscribe:backup-job:{backup_job_id}`
   - Message: `{ "type": "backup-progress", "backup_id": "...", "progress_percentage": 50.0, "bytes_transferred": 21474836480 }`

2. **Copy Operation Progress:**
   - Subscribe: `subscribe:copy-operation:{copy_id}`
   - Message: `{ "type": "copy-progress", "copy_id": "...", "status": "copying", "progress_percentage": 65.0 }`

3. **Storage Capacity Changes:**
   - Subscribe: `subscribe:repository-storage:{repository_id}`
   - Message: `{ "type": "storage-update", "repository_id": "...", "used_bytes": 4294967296000, "available_bytes": 6442450944000 }`

4. **Repository Health:**
   - Subscribe: `subscribe:repository-health`
   - Message: `{ "type": "repository-health", "repository_id": "...", "health_status": "warning", "message": "Low storage space" }`

## Error Handling

### Common Error Responses

1. **Repository Not Found:**
```json
{
  "success": false,
  "error": "repository not found",
  "error_code": "REPO_NOT_FOUND"
}
```

2. **Insufficient Storage:**
```json
{
  "success": false,
  "error": "insufficient storage space",
  "error_code": "INSUFFICIENT_STORAGE",
  "required_bytes": 42949672960,
  "available_bytes": 21474836480
}
```

3. **Repository Connection Failed:**
```json
{
  "success": false,
  "error": "failed to connect to NFS server: connection timeout",
  "error_code": "CONNECTION_FAILED"
}
```

4. **Immutable Retention Violation:**
```json
{
  "success": false,
  "error": "cannot delete backup: immutable retention period not expired",
  "error_code": "IMMUTABLE_RETENTION_ACTIVE",
  "retention_expires_at": "2025-11-03T12:00:00Z"
}
```

### GUI Error Handling

The GUI should:
1. Display user-friendly error messages
2. Log full error details to console
3. Provide retry options where appropriate
4. Show contextual help for configuration errors
5. Alert administrators for critical issues

## State Management (React Query)

### Query Keys

```typescript
// Repositories
['repositories']                                 // List all repositories
['repository', repository_id]                    // Get single repository
['repository-storage-summary']                   // Storage summary

// Policies
['backup-policies']                              // List all policies
['backup-policy', policy_id]                     // Get single policy
['backup-policy', policy_id, 'copy-rules']       // Get copy rules

// Backups
['backups', { vm_context_id }]                   // List VM backups
['backup', backup_id]                            // Get single backup
['backup-chain', { vm_context_id, disk_id }]     // Get backup chain

// Copies
['backup-copies', { source_backup_id }]          // List backup copies
['backup-copy', copy_id]                         // Get single copy
```

### Mutations

```typescript
// Repositories
useMutation('register-repository', registerRepository)
useMutation('update-repository', updateRepository)
useMutation('delete-repository', deleteRepository)
useMutation('test-repository', testRepository)
useMutation('refresh-storage', refreshStorageInfo)

// Policies
useMutation('create-backup-policy', createBackupPolicy)
useMutation('update-backup-policy', updateBackupPolicy)
useMutation('delete-backup-policy', deleteBackupPolicy)
useMutation('add-copy-rule', addCopyRule)

// Backups
useMutation('create-backup', createBackup)
useMutation('delete-backup', deleteBackup)

// Copies
useMutation('create-backup-copy', createBackupCopy)
useMutation('verify-backup-copy', verifyBackupCopy)
```

## Testing Checklist

### Repository Management
- [ ] Register local repository
- [ ] Register NFS repository
- [ ] Register CIFS/SMB repository
- [ ] Test repository connection
- [ ] Disable repository
- [ ] Delete repository (with and without backups)
- [ ] Refresh storage info
- [ ] View storage capacity visualization

### Backup Policy Management
- [ ] Create policy with single repository
- [ ] Create policy with multiple copy rules (3-2-1)
- [ ] Configure immutable repository in policy
- [ ] Assign VMs to policy
- [ ] Edit policy retention days
- [ ] Disable policy
- [ ] Delete policy

### Backup Operations
- [ ] Create full backup
- [ ] Create incremental backup
- [ ] View backup progress in real-time
- [ ] View backup chain
- [ ] Automatic copy to secondary repository
- [ ] Manual copy to tertiary repository
- [ ] Verify backup copy
- [ ] Delete backup (non-immutable)
- [ ] Attempt to delete immutable backup (should fail)

### Monitoring
- [ ] View storage summary dashboard
- [ ] Monitor active backup jobs
- [ ] Monitor copy operations
- [ ] Receive low storage alerts
- [ ] Receive failed copy alerts

## Implementation Order

1. **Phase 1A:** Repository management UI (list, add, edit, delete, test)
2. **Phase 1B:** Storage monitoring dashboard
3. **Phase 1C:** Backup policy management UI
4. **Phase 1D:** VM backup list and manual backup creation
5. **Phase 1E:** Backup copy monitoring
6. **Phase 1F:** Real-time WebSocket updates
7. **Phase 1G:** Error handling and user feedback

## Notes

- All timestamps should be displayed in user's local timezone
- Storage sizes should be displayed in human-readable format (GB, TB)
- Progress bars should show percentage and estimated time remaining
- Immutable repositories should have visual indicators (lock icon)
- 3-2-1 rule compliance should be highlighted in policy UI
- Repository health checks should run automatically every 5 minutes
- Failed operations should have detailed error logs accessible from GUI

---

**Document Version:** 1.0  
**Last Updated:** 2025-10-04  
**Maintained By:** Sendense Development Team
