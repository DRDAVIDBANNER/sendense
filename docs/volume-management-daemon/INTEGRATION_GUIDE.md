# Volume Management Daemon - Integration Guide

**How to integrate existing services with the Volume Management Daemon**

## Table of Contents

1. [Integration Overview](#integration-overview)
2. [Migration Process](#migration-process)
3. [API Client Implementation](#api-client-implementation)
4. [Service Integration Examples](#service-integration-examples)
5. [Error Handling](#error-handling)
6. [Testing Integration](#testing-integration)
7. [Rollback Procedures](#rollback-procedures)
8. [Best Practices](#best-practices)

---

## Integration Overview

The Volume Management Daemon centralizes all CloudStack volume operations to eliminate database corruption and ensure consistency. Existing services must be updated to use the daemon's REST API instead of making direct CloudStack API calls.

### Services Requiring Integration

1. **Migration Engine** (`internal/oma/workflows/migration.go`)
2. **Failover System** (`internal/oma/failover/`)  
3. **NBD Manager** (`internal/oma/nbd/server.go`)
4. **Any service performing volume operations**

### Integration Benefits

- ✅ **Eliminated database corruption**
- ✅ **Real-time device correlation**
- ✅ **Atomic transaction management**
- ✅ **Complete operation auditing**
- ✅ **Simplified error handling**

---

## Migration Process

### Phase 1: Preparation

#### 1. Audit Current Volume Operations

**Identify all CloudStack volume calls**:
```bash
# Find direct CloudStack SDK usage
grep -r "cloudstack.*Volume" internal/ --include="*.go"
grep -r "attachVolume\|detachVolume\|createVolume\|deleteVolume" internal/ --include="*.go"

# Find database volume operations
grep -r "ossea_volumes\|vm_export_mappings" internal/ --include="*.go"
```

**Create integration checklist**:
```
□ Migration workflow volume creation
□ Migration workflow volume attachment  
□ Failover volume operations
□ NBD export volume management
□ Volume cleanup operations
□ Test volume operations
```

#### 2. Install Volume Daemon

**Deploy the daemon**:
```bash
# Build and install
go build -o /usr/local/bin/volume-daemon cmd/volume-daemon/main.go

# Install service
sudo systemctl enable volume-daemon
sudo systemctl start volume-daemon

# Verify operation
curl -f http://localhost:8090/health
```

#### 3. Update Database Schema

**Run schema updates**:
```bash
mysql -u oma_user -poma_password migratekit_oma < internal/volume/database/schema.sql
```

### Phase 2: Service Integration

#### 1. Create Volume Client Library

**Create shared client** (`internal/common/volume_client.go`):
```go
package common

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    "time"
)

type VolumeClient struct {
    baseURL string
    client  *http.Client
}

func NewVolumeClient(baseURL string) *VolumeClient {
    return &VolumeClient{
        baseURL: baseURL,
        client: &http.Client{
            Timeout: 30 * time.Second,
        },
    }
}

func (vc *VolumeClient) CreateVolume(ctx context.Context, req CreateVolumeRequest) (*VolumeOperation, error) {
    jsonData, err := json.Marshal(req)
    if err != nil {
        return nil, fmt.Errorf("failed to marshal request: %w", err)
    }

    resp, err := vc.client.Post(
        vc.baseURL+"/api/v1/volumes",
        "application/json",
        bytes.NewBuffer(jsonData),
    )
    if err != nil {
        return nil, fmt.Errorf("failed to create volume: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != 201 {
        return nil, fmt.Errorf("volume creation failed with status %d", resp.StatusCode)
    }

    var operation VolumeOperation
    if err := json.NewDecoder(resp.Body).Decode(&operation); err != nil {
        return nil, fmt.Errorf("failed to decode response: %w", err)
    }

    return &operation, nil
}

func (vc *VolumeClient) AttachVolume(ctx context.Context, volumeID, vmID string) (*VolumeOperation, error) {
    req := AttachVolumeRequest{VMID: vmID}
    jsonData, err := json.Marshal(req)
    if err != nil {
        return nil, fmt.Errorf("failed to marshal request: %w", err)
    }

    url := fmt.Sprintf("%s/api/v1/volumes/%s/attach", vc.baseURL, volumeID)
    resp, err := vc.client.Post(url, "application/json", bytes.NewBuffer(jsonData))
    if err != nil {
        return nil, fmt.Errorf("failed to attach volume: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != 201 {
        return nil, fmt.Errorf("volume attachment failed with status %d", resp.StatusCode)
    }

    var operation VolumeOperation
    if err := json.NewDecoder(resp.Body).Decode(&operation); err != nil {
        return nil, fmt.Errorf("failed to decode response: %w", err)
    }

    return &operation, nil
}

func (vc *VolumeClient) WaitForCompletion(ctx context.Context, operationID string) (*VolumeOperation, error) {
    ticker := time.NewTicker(2 * time.Second)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return nil, ctx.Err()
        case <-ticker.C:
            op, err := vc.GetOperation(ctx, operationID)
            if err != nil {
                return nil, err
            }

            switch op.Status {
            case "completed":
                return op, nil
            case "failed":
                return op, fmt.Errorf("operation failed: %s", op.Error)
            case "cancelled":
                return op, fmt.Errorf("operation cancelled")
            }
            // Continue waiting for pending/executing
        }
    }
}

func (vc *VolumeClient) GetOperation(ctx context.Context, operationID string) (*VolumeOperation, error) {
    url := fmt.Sprintf("%s/api/v1/operations/%s", vc.baseURL, operationID)
    resp, err := vc.client.Get(url)
    if err != nil {
        return nil, fmt.Errorf("failed to get operation: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != 200 {
        return nil, fmt.Errorf("failed to get operation status %d", resp.StatusCode)
    }

    var operation VolumeOperation
    if err := json.NewDecoder(resp.Body).Decode(&operation); err != nil {
        return nil, fmt.Errorf("failed to decode response: %w", err)
    }

    return &operation, nil
}

// Additional methods: DetachVolume, DeleteVolume, GetVolumeDevice, etc.
```

#### 2. Update Migration Workflow

**Update** `internal/oma/workflows/migration.go`:

**Before** (direct CloudStack calls):
```go
func (w *MigrationWorkflow) attachOSSEAVolumes(ctx context.Context) error {
    for i, volume := range w.job.Volumes {
        // Direct CloudStack API call
        params := w.osseaClient.Volume.NewAttachVolumeParams(volume.ID, w.job.OSSEAVMID)
        _, err := w.osseaClient.Volume.AttachVolume(params)
        if err != nil {
            return fmt.Errorf("failed to attach volume: %w", err)
        }

        // Assume device path
        devicePath := fmt.Sprintf("/dev/vd%c", 'b'+i)
        
        // Store in database
        volume.DevicePath = devicePath
        // ... database update
    }
    return nil
}
```

**After** (Volume Daemon integration):
```go
func (w *MigrationWorkflow) attachOSSEAVolumes(ctx context.Context) error {
    volumeClient := common.NewVolumeClient("http://localhost:8090")
    
    for _, volume := range w.job.Volumes {
        // Use Volume Daemon
        operation, err := volumeClient.AttachVolume(ctx, volume.ID, w.job.OSSEAVMID)
        if err != nil {
            return fmt.Errorf("failed to start volume attachment: %w", err)
        }

        // Wait for completion with device correlation
        completedOp, err := volumeClient.WaitForCompletion(ctx, operation.ID)
        if err != nil {
            return fmt.Errorf("volume attachment failed: %w", err)
        }

        // Get actual device path from daemon
        devicePath := completedOp.Response["device_path"].(string)
        
        // Update volume with real device path
        volume.DevicePath = devicePath
        log.WithFields(log.Fields{
            "volume_id":   volume.ID,
            "device_path": devicePath,
            "operation_id": operation.ID,
        }).Info("Volume attached with device correlation")
    }
    return nil
}
```

#### 3. Update Failover System

**Update** `internal/oma/failover/test_failover.go`:

**Before**:
```go
func (tfe *TestFailoverEngine) detachVolumeFromOMA(ctx context.Context, volumeID string) error {
    params := tfe.osseaClient.Volume.NewDetachVolumeParams(volumeID)
    _, err := tfe.osseaClient.Volume.DetachVolume(params)
    return err
}
```

**After**:
```go
func (tfe *TestFailoverEngine) detachVolumeFromOMA(ctx context.Context, volumeID string) error {
    volumeClient := common.NewVolumeClient("http://localhost:8090")
    
    operation, err := volumeClient.DetachVolume(ctx, volumeID)
    if err != nil {
        return fmt.Errorf("failed to start volume detachment: %w", err)
    }

    _, err = volumeClient.WaitForCompletion(ctx, operation.ID)
    if err != nil {
        return fmt.Errorf("volume detachment failed: %w", err)
    }

    log.WithFields(log.Fields{
        "volume_id":    volumeID,
        "operation_id": operation.ID,
    }).Info("Volume detached via daemon")
    
    return nil
}
```

#### 4. Update NBD Server Integration

**Update** `internal/oma/nbd/server.go`:

**Before** (database-driven device paths):
```go
func (s *Server) AddDynamicExport(vmName, volumeID, devicePath string) error {
    // Direct database manipulation
    mapping := &VMExportMapping{
        VMName:     vmName,
        VolumeID:   volumeID,
        DevicePath: devicePath,
        ExportName: fmt.Sprintf("migration-vm-%s", vmName),
    }
    return s.repo.CreateMapping(mapping)
}
```

**After** (Volume Daemon correlation):
```go
func (s *Server) AddDynamicExport(vmName, volumeID string) error {
    volumeClient := common.NewVolumeClient("http://localhost:8090")
    
    // Get current device mapping from daemon
    deviceMapping, err := volumeClient.GetVolumeDevice(context.Background(), volumeID)
    if err != nil {
        return fmt.Errorf("failed to get device mapping: %w", err)
    }

    if deviceMapping.DevicePath == "" {
        return fmt.Errorf("volume %s has no device mapping", volumeID)
    }

    mapping := &VMExportMapping{
        VMName:     vmName,
        VolumeID:   volumeID,
        DevicePath: deviceMapping.DevicePath, // Real device path from daemon
        ExportName: fmt.Sprintf("migration-vm-%s", vmName),
    }
    
    log.WithFields(log.Fields{
        "volume_id":   volumeID,
        "device_path": deviceMapping.DevicePath,
        "vm_name":     vmName,
    }).Info("Creating NBD export with daemon-verified device path")
    
    return s.repo.CreateMapping(mapping)
}
```

---

## API Client Implementation

### Complete Client Library

**File**: `internal/common/volume_client.go`

```go
package common

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "net/url"
    "time"
    
    log "github.com/sirupsen/logrus"
)

// Client configuration
type VolumeClientConfig struct {
    BaseURL string
    Timeout time.Duration
    Retries int
}

// Default configuration
func DefaultVolumeClientConfig() VolumeClientConfig {
    return VolumeClientConfig{
        BaseURL: "http://localhost:8090",
        Timeout: 30 * time.Second,
        Retries: 3,
    }
}

// Volume client
type VolumeClient struct {
    config VolumeClientConfig
    client *http.Client
}

// Request/Response types
type CreateVolumeRequest struct {
    Name           string            `json:"name"`
    Size           int64             `json:"size"`
    DiskOfferingID string            `json:"disk_offering_id"`
    ZoneID         string            `json:"zone_id"`
    Metadata       map[string]string `json:"metadata,omitempty"`
}

type AttachVolumeRequest struct {
    VMID string `json:"vm_id"`
}

type VolumeOperation struct {
    ID          string                 `json:"id"`
    Type        string                 `json:"type"`
    Status      string                 `json:"status"`
    VolumeID    string                 `json:"volume_id"`
    VMID        *string                `json:"vm_id"`
    Request     map[string]interface{} `json:"request"`
    Response    map[string]interface{} `json:"response"`
    Error       string                 `json:"error"`
    CreatedAt   time.Time              `json:"created_at"`
    UpdatedAt   time.Time              `json:"updated_at"`
    CompletedAt *time.Time             `json:"completed_at"`
}

type DeviceMapping struct {
    ID              string    `json:"id"`
    VolumeID        string    `json:"volume_id"`
    VMID            string    `json:"vm_id"`
    DevicePath      string    `json:"device_path"`
    CloudStackState string    `json:"cloudstack_state"`
    LinuxState      string    `json:"linux_state"`
    Size            int64     `json:"size"`
    LastSync        time.Time `json:"last_sync"`
    CreatedAt       time.Time `json:"created_at"`
    UpdatedAt       time.Time `json:"updated_at"`
}

// Constructor
func NewVolumeClient(config VolumeClientConfig) *VolumeClient {
    return &VolumeClient{
        config: config,
        client: &http.Client{
            Timeout: config.Timeout,
        },
    }
}

// Convenience constructor with defaults
func NewDefaultVolumeClient() *VolumeClient {
    return NewVolumeClient(DefaultVolumeClientConfig())
}

// Core operations
func (vc *VolumeClient) CreateVolume(ctx context.Context, req CreateVolumeRequest) (*VolumeOperation, error) {
    return vc.doVolumeOperation(ctx, "POST", "/api/v1/volumes", req)
}

func (vc *VolumeClient) AttachVolume(ctx context.Context, volumeID, vmID string) (*VolumeOperation, error) {
    req := AttachVolumeRequest{VMID: vmID}
    endpoint := fmt.Sprintf("/api/v1/volumes/%s/attach", volumeID)
    return vc.doVolumeOperation(ctx, "POST", endpoint, req)
}

func (vc *VolumeClient) DetachVolume(ctx context.Context, volumeID string) (*VolumeOperation, error) {
    endpoint := fmt.Sprintf("/api/v1/volumes/%s/detach", volumeID)
    return vc.doVolumeOperation(ctx, "POST", endpoint, nil)
}

func (vc *VolumeClient) DeleteVolume(ctx context.Context, volumeID string) (*VolumeOperation, error) {
    endpoint := fmt.Sprintf("/api/v1/volumes/%s", volumeID)
    return vc.doVolumeOperation(ctx, "DELETE", endpoint, nil)
}

// Status operations
func (vc *VolumeClient) GetOperation(ctx context.Context, operationID string) (*VolumeOperation, error) {
    endpoint := fmt.Sprintf("/api/v1/operations/%s", operationID)
    
    resp, err := vc.doRequest(ctx, "GET", endpoint, nil)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    var operation VolumeOperation
    if err := json.NewDecoder(resp.Body).Decode(&operation); err != nil {
        return nil, fmt.Errorf("failed to decode operation: %w", err)
    }

    return &operation, nil
}

func (vc *VolumeClient) GetVolumeDevice(ctx context.Context, volumeID string) (*DeviceMapping, error) {
    endpoint := fmt.Sprintf("/api/v1/volumes/%s/device", volumeID)
    
    resp, err := vc.doRequest(ctx, "GET", endpoint, nil)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    var mapping DeviceMapping
    if err := json.NewDecoder(resp.Body).Decode(&mapping); err != nil {
        return nil, fmt.Errorf("failed to decode device mapping: %w", err)
    }

    return &mapping, nil
}

// Wait for operation completion
func (vc *VolumeClient) WaitForCompletion(ctx context.Context, operationID string) (*VolumeOperation, error) {
    return vc.WaitForCompletionWithTimeout(ctx, operationID, 5*time.Minute)
}

func (vc *VolumeClient) WaitForCompletionWithTimeout(ctx context.Context, operationID string, timeout time.Duration) (*VolumeOperation, error) {
    ctx, cancel := context.WithTimeout(ctx, timeout)
    defer cancel()

    ticker := time.NewTicker(2 * time.Second)
    defer ticker.Stop()

    log.WithFields(log.Fields{
        "operation_id": operationID,
        "timeout":      timeout,
    }).Info("Waiting for volume operation completion")

    for {
        select {
        case <-ctx.Done():
            return nil, fmt.Errorf("operation timeout or cancelled: %w", ctx.Err())
        case <-ticker.C:
            op, err := vc.GetOperation(ctx, operationID)
            if err != nil {
                log.WithError(err).Warn("Failed to get operation status, retrying")
                continue
            }

            log.WithFields(log.Fields{
                "operation_id": operationID,
                "status":       op.Status,
            }).Debug("Operation status check")

            switch op.Status {
            case "completed":
                log.WithFields(log.Fields{
                    "operation_id": operationID,
                    "duration":     time.Since(op.CreatedAt),
                }).Info("Volume operation completed successfully")
                return op, nil
            case "failed":
                return op, fmt.Errorf("operation failed: %s", op.Error)
            case "cancelled":
                return op, fmt.Errorf("operation was cancelled")
            }
            // Continue waiting for pending/executing
        }
    }
}

// Helper methods
func (vc *VolumeClient) doVolumeOperation(ctx context.Context, method, endpoint string, body interface{}) (*VolumeOperation, error) {
    resp, err := vc.doRequest(ctx, method, endpoint, body)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    if resp.StatusCode != 201 {
        return nil, vc.handleErrorResponse(resp)
    }

    var operation VolumeOperation
    if err := json.NewDecoder(resp.Body).Decode(&operation); err != nil {
        return nil, fmt.Errorf("failed to decode operation response: %w", err)
    }

    return &operation, nil
}

func (vc *VolumeClient) doRequest(ctx context.Context, method, endpoint string, body interface{}) (*http.Response, error) {
    var bodyReader io.Reader
    if body != nil {
        jsonData, err := json.Marshal(body)
        if err != nil {
            return nil, fmt.Errorf("failed to marshal request body: %w", err)
        }
        bodyReader = bytes.NewBuffer(jsonData)
    }

    url := vc.config.BaseURL + endpoint
    req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
    if err != nil {
        return nil, fmt.Errorf("failed to create request: %w", err)
    }

    if body != nil {
        req.Header.Set("Content-Type", "application/json")
    }

    log.WithFields(log.Fields{
        "method":   method,
        "endpoint": endpoint,
    }).Debug("Making volume daemon API request")

    resp, err := vc.client.Do(req)
    if err != nil {
        return nil, fmt.Errorf("failed to execute request: %w", err)
    }

    return resp, nil
}

func (vc *VolumeClient) handleErrorResponse(resp *http.Response) error {
    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return fmt.Errorf("volume operation failed with status %d", resp.StatusCode)
    }

    var errorResp struct {
        Error   string      `json:"error"`
        Code    string      `json:"code"`
        Details interface{} `json:"details"`
    }

    if err := json.Unmarshal(body, &errorResp); err != nil {
        return fmt.Errorf("volume operation failed with status %d: %s", resp.StatusCode, string(body))
    }

    return fmt.Errorf("volume operation failed (%s): %s", errorResp.Code, errorResp.Error)
}

// Health check
func (vc *VolumeClient) HealthCheck(ctx context.Context) error {
    resp, err := vc.doRequest(ctx, "GET", "/health", nil)
    if err != nil {
        return fmt.Errorf("health check failed: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != 200 {
        return fmt.Errorf("volume daemon unhealthy: status %d", resp.StatusCode)
    }

    return nil
}
```

---

## Service Integration Examples

### 1. Migration Service Integration

**File**: `internal/oma/workflows/migration_volume_operations.go`

```go
package workflows

import (
    "context"
    "fmt"
    "time"

    log "github.com/sirupsen/logrus"
    "github.com/vexxhost/migratekit/internal/common"
)

// VolumeManager handles volume operations for migration workflows
type VolumeManager struct {
    client *common.VolumeClient
    config VolumeConfig
}

type VolumeConfig struct {
    DiskOfferingID string
    ZoneID         string
    DefaultSize    int64
    Timeout        time.Duration
}

func NewVolumeManager(config VolumeConfig) *VolumeManager {
    return &VolumeManager{
        client: common.NewDefaultVolumeClient(),
        config: config,
    }
}

func (vm *VolumeManager) CreateMigrationVolume(ctx context.Context, vmName string, size int64) (*common.VolumeOperation, error) {
    req := common.CreateVolumeRequest{
        Name:           fmt.Sprintf("migration-%s-%d", vmName, time.Now().Unix()),
        Size:           size,
        DiskOfferingID: vm.config.DiskOfferingID,
        ZoneID:         vm.config.ZoneID,
        Metadata: map[string]string{
            "purpose":  "migration",
            "vm_name":  vmName,
            "created":  time.Now().Format(time.RFC3339),
        },
    }

    log.WithFields(log.Fields{
        "vm_name": vmName,
        "size":    size,
        "request": req,
    }).Info("Creating migration volume")

    operation, err := vm.client.CreateVolume(ctx, req)
    if err != nil {
        return nil, fmt.Errorf("failed to create migration volume: %w", err)
    }

    completedOp, err := vm.client.WaitForCompletionWithTimeout(ctx, operation.ID, vm.config.Timeout)
    if err != nil {
        return nil, fmt.Errorf("migration volume creation failed: %w", err)
    }

    log.WithFields(log.Fields{
        "operation_id": operation.ID,
        "volume_id":    completedOp.VolumeID,
        "vm_name":      vmName,
    }).Info("Migration volume created successfully")

    return completedOp, nil
}

func (vm *VolumeManager) AttachVolumeToOMA(ctx context.Context, volumeID, omaVMID string) (string, error) {
    log.WithFields(log.Fields{
        "volume_id": volumeID,
        "vm_id":     omaVMID,
    }).Info("Attaching volume to OMA")

    operation, err := vm.client.AttachVolume(ctx, volumeID, omaVMID)
    if err != nil {
        return "", fmt.Errorf("failed to attach volume: %w", err)
    }

    completedOp, err := vm.client.WaitForCompletionWithTimeout(ctx, operation.ID, vm.config.Timeout)
    if err != nil {
        return "", fmt.Errorf("volume attachment failed: %w", err)
    }

    devicePath, ok := completedOp.Response["device_path"].(string)
    if !ok || devicePath == "" {
        return "", fmt.Errorf("no device path returned from volume attachment")
    }

    log.WithFields(log.Fields{
        "operation_id": operation.ID,
        "volume_id":    volumeID,
        "device_path":  devicePath,
    }).Info("Volume attached to OMA with device correlation")

    return devicePath, nil
}

// Integration with existing migration workflow
func (w *MigrationWorkflow) createAndAttachOSSEAVolumes(ctx context.Context) error {
    volumeManager := NewVolumeManager(VolumeConfig{
        DiskOfferingID: w.config.DiskOfferingID,
        ZoneID:         w.config.ZoneID,
        DefaultSize:    w.config.DefaultVolumeSize,
        Timeout:        5 * time.Minute,
    })

    for i, disk := range w.job.Disks {
        // Create volume
        operation, err := volumeManager.CreateMigrationVolume(ctx, w.job.VMName, disk.Size)
        if err != nil {
            return fmt.Errorf("failed to create volume for disk %d: %w", i, err)
        }

        volumeID := operation.VolumeID
        
        // Attach to OMA
        devicePath, err := volumeManager.AttachVolumeToOMA(ctx, volumeID, w.job.OSSEAVMID)
        if err != nil {
            return fmt.Errorf("failed to attach volume %s: %w", volumeID, err)
        }

        // Update job with real volume information
        w.job.Volumes = append(w.job.Volumes, VolumeInfo{
            Index:      i,
            VolumeID:   volumeID,
            DevicePath: devicePath,
            Size:       disk.Size,
            CreatedAt:  time.Now(),
        })

        log.WithFields(log.Fields{
            "disk_index":   i,
            "volume_id":    volumeID,
            "device_path":  devicePath,
            "size":         disk.Size,
        }).Info("Volume created and attached for migration")
    }

    return nil
}
```

### 2. Failover Service Integration

**File**: `internal/oma/failover/volume_operations.go`

```go
package failover

import (
    "context"
    "fmt"
    "time"

    log "github.com/sirupsen/logrus"
    "github.com/vexxhost/migratekit/internal/common"
)

// FailoverVolumeManager handles volume operations during failover
type FailoverVolumeManager struct {
    client *common.VolumeClient
    logger *log.Entry
}

func NewFailoverVolumeManager(jobID string) *FailoverVolumeManager {
    return &FailoverVolumeManager{
        client: common.NewDefaultVolumeClient(),
        logger: log.WithField("failover_job", jobID),
    }
}

func (fvm *FailoverVolumeManager) DetachVolumeFromOMA(ctx context.Context, volumeID string) error {
    fvm.logger.WithField("volume_id", volumeID).Info("Detaching volume from OMA")

    operation, err := fvm.client.DetachVolume(ctx, volumeID)
    if err != nil {
        return fmt.Errorf("failed to start volume detachment: %w", err)
    }

    _, err = fvm.client.WaitForCompletionWithTimeout(ctx, operation.ID, 2*time.Minute)
    if err != nil {
        return fmt.Errorf("volume detachment failed: %w", err)
    }

    fvm.logger.WithFields(log.Fields{
        "volume_id":    volumeID,
        "operation_id": operation.ID,
    }).Info("Volume detached from OMA successfully")

    return nil
}

func (fvm *FailoverVolumeManager) AttachVolumeToTestVM(ctx context.Context, volumeID, testVMID string) error {
    fvm.logger.WithFields(log.Fields{
        "volume_id":  volumeID,
        "test_vm_id": testVMID,
    }).Info("Attaching volume to test VM")

    operation, err := fvm.client.AttachVolume(ctx, volumeID, testVMID)
    if err != nil {
        return fmt.Errorf("failed to start volume attachment: %w", err)
    }

    completedOp, err := fvm.client.WaitForCompletionWithTimeout(ctx, operation.ID, 2*time.Minute)
    if err != nil {
        return fmt.Errorf("volume attachment failed: %w", err)
    }

    devicePath, ok := completedOp.Response["device_path"].(string)
    if !ok {
        devicePath = "unknown"
    }

    fvm.logger.WithFields(log.Fields{
        "volume_id":    volumeID,
        "test_vm_id":   testVMID,
        "device_path":  devicePath,
        "operation_id": operation.ID,
    }).Info("Volume attached to test VM successfully")

    return nil
}

func (fvm *FailoverVolumeManager) ReattachVolumeToOMA(ctx context.Context, volumeID, omaVMID string) (string, error) {
    fvm.logger.WithFields(log.Fields{
        "volume_id": volumeID,
        "oma_vm_id": omaVMID,
    }).Info("Reattaching volume to OMA")

    operation, err := fvm.client.AttachVolume(ctx, volumeID, omaVMID)
    if err != nil {
        return "", fmt.Errorf("failed to start volume reattachment: %w", err)
    }

    completedOp, err := fvm.client.WaitForCompletionWithTimeout(ctx, operation.ID, 2*time.Minute)
    if err != nil {
        return "", fmt.Errorf("volume reattachment failed: %w", err)
    }

    devicePath, ok := completedOp.Response["device_path"].(string)
    if !ok || devicePath == "" {
        return "", fmt.Errorf("no device path returned from volume reattachment")
    }

    fvm.logger.WithFields(log.Fields{
        "volume_id":    volumeID,
        "device_path":  devicePath,
        "operation_id": operation.ID,
    }).Info("Volume reattached to OMA successfully")

    return devicePath, nil
}

// Integration with existing test failover
func (tfe *TestFailoverEngine) executeVolumeDetachStep(ctx context.Context) error {
    volumeManager := NewFailoverVolumeManager(tfe.job.ID)
    
    for _, volumeID := range tfe.job.VolumeIDs {
        if err := volumeManager.DetachVolumeFromOMA(ctx, volumeID); err != nil {
            return fmt.Errorf("failed to detach volume %s: %w", volumeID, err)
        }
    }
    
    return nil
}

func (tfe *TestFailoverEngine) executeVolumeAttachStep(ctx context.Context) error {
    volumeManager := NewFailoverVolumeManager(tfe.job.ID)
    
    for _, volumeID := range tfe.job.VolumeIDs {
        if err := volumeManager.AttachVolumeToTestVM(ctx, volumeID, tfe.job.TestVMID); err != nil {
            return fmt.Errorf("failed to attach volume %s to test VM: %w", volumeID, err)
        }
    }
    
    return nil
}
```

---

## Error Handling

### Centralized Error Handling

**File**: `internal/common/volume_errors.go`

```go
package common

import (
    "fmt"
    "strings"
)

// Error types
type VolumeError struct {
    Type    string
    Message string
    Code    string
    Details map[string]interface{}
}

func (ve *VolumeError) Error() string {
    return fmt.Sprintf("%s: %s", ve.Type, ve.Message)
}

// Error classification
func ClassifyVolumeError(err error) *VolumeError {
    if err == nil {
        return nil
    }

    errStr := err.Error()
    
    switch {
    case strings.Contains(errStr, "CloudStack API error 431"):
        return &VolumeError{
            Type:    "InvalidParameter",
            Message: "CloudStack rejected request parameters",
            Code:    "CLOUDSTACK_INVALID_PARAMETER",
            Details: map[string]interface{}{"original_error": errStr},
        }
    case strings.Contains(errStr, "CloudStack API error 432"):
        return &VolumeError{
            Type:    "ResourceLimit",
            Message: "CloudStack resource limit exceeded", 
            Code:    "CLOUDSTACK_RESOURCE_LIMIT",
            Details: map[string]interface{}{"original_error": errStr},
        }
    case strings.Contains(errStr, "timeout"):
        return &VolumeError{
            Type:    "Timeout",
            Message: "Operation timed out",
            Code:    "OPERATION_TIMEOUT",
            Details: map[string]interface{}{"original_error": errStr},
        }
    case strings.Contains(errStr, "device"):
        return &VolumeError{
            Type:    "DeviceCorrelation",
            Message: "Failed to correlate volume with device",
            Code:    "DEVICE_CORRELATION_FAILED",
            Details: map[string]interface{}{"original_error": errStr},
        }
    default:
        return &VolumeError{
            Type:    "Unknown",
            Message: errStr,
            Code:    "UNKNOWN_ERROR",
            Details: map[string]interface{}{"original_error": errStr},
        }
    }
}

// Retry logic
func IsRetryableError(err error) bool {
    classified := ClassifyVolumeError(err)
    if classified == nil {
        return false
    }

    // Retryable error types
    retryable := []string{
        "OPERATION_TIMEOUT",
        "CLOUDSTACK_RESOURCE_LIMIT", // May be temporary
        "DEVICE_CORRELATION_FAILED", // May succeed on retry
    }

    for _, code := range retryable {
        if classified.Code == code {
            return true
        }
    }

    return false
}

// Error recovery suggestions
func GetRecoverySuggestion(err error) string {
    classified := ClassifyVolumeError(err)
    if classified == nil {
        return "No recovery suggestion available"
    }

    switch classified.Code {
    case "CLOUDSTACK_INVALID_PARAMETER":
        return "Check zone IDs, disk offering IDs, and VM IDs in configuration"
    case "CLOUDSTACK_RESOURCE_LIMIT":
        return "Clean up unused volumes or request quota increase"
    case "OPERATION_TIMEOUT":
        return "Check CloudStack performance and increase timeout if needed"
    case "DEVICE_CORRELATION_FAILED":
        return "Verify device monitor is running and volumes are properly attached"
    default:
        return "Check logs for detailed error information"
    }
}
```

### Integration Error Handling

```go
func (w *MigrationWorkflow) createVolumeWithRetry(ctx context.Context, req common.CreateVolumeRequest) (*common.VolumeOperation, error) {
    const maxRetries = 3
    
    for attempt := 1; attempt <= maxRetries; attempt++ {
        operation, err := w.volumeClient.CreateVolume(ctx, req)
        if err == nil {
            return operation, nil
        }

        // Classify error
        classified := common.ClassifyVolumeError(err)
        
        log.WithFields(log.Fields{
            "attempt":     attempt,
            "error_type":  classified.Type,
            "error_code":  classified.Code,
            "suggestion":  common.GetRecoverySuggestion(err),
        }).Warn("Volume creation failed")

        // Don't retry non-retryable errors
        if !common.IsRetryableError(err) {
            return nil, fmt.Errorf("non-retryable error: %w", err)
        }

        // Don't retry on last attempt
        if attempt == maxRetries {
            return nil, fmt.Errorf("max retries exceeded: %w", err)
        }

        // Exponential backoff
        backoff := time.Duration(attempt*attempt) * time.Second
        log.WithField("backoff", backoff).Info("Retrying after backoff")
        time.Sleep(backoff)
    }

    return nil, fmt.Errorf("unreachable code")
}
```

---

## Testing Integration

### Integration Test Framework

**File**: `internal/common/volume_client_test.go`

```go
package common

import (
    "context"
    "testing"
    "time"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestVolumeClientIntegration(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test")
    }

    // Require daemon to be running
    client := NewDefaultVolumeClient()
    
    ctx := context.Background()
    err := client.HealthCheck(ctx)
    require.NoError(t, err, "Volume daemon must be running for integration tests")

    t.Run("CreateVolume", func(t *testing.T) {
        req := CreateVolumeRequest{
            Name:           "test-volume-" + time.Now().Format("20060102-150405"),
            Size:           1 * 1024 * 1024 * 1024, // 1GB
            DiskOfferingID: "c813c642-d946-49e1-9289-c616dd70206a",
            ZoneID:         "057e86db-c726-4d8c-ab1f-75c5f55d1881",
            Metadata: map[string]string{
                "test": "true",
            },
        }

        operation, err := client.CreateVolume(ctx, req)
        require.NoError(t, err)
        assert.Equal(t, "create", operation.Type)
        assert.Equal(t, "pending", operation.Status)

        // Wait for completion
        completed, err := client.WaitForCompletionWithTimeout(ctx, operation.ID, 2*time.Minute)
        require.NoError(t, err)
        assert.Equal(t, "completed", completed.Status)
        assert.NotEmpty(t, completed.VolumeID)

        // Clean up
        deleteOp, err := client.DeleteVolume(ctx, completed.VolumeID)
        require.NoError(t, err)
        
        _, err = client.WaitForCompletionWithTimeout(ctx, deleteOp.ID, 1*time.Minute)
        require.NoError(t, err)
    })

    t.Run("AttachDetachVolume", func(t *testing.T) {
        // This test requires a known VM ID
        testVMID := "vm-test-12345" // Replace with actual test VM
        
        // Create volume first
        createReq := CreateVolumeRequest{
            Name:           "test-attach-" + time.Now().Format("20060102-150405"),
            Size:           1 * 1024 * 1024 * 1024,
            DiskOfferingID: "c813c642-d946-49e1-9289-c616dd70206a",
            ZoneID:         "057e86db-c726-4d8c-ab1f-75c5f55d1881",
        }

        createOp, err := client.CreateVolume(ctx, createReq)
        require.NoError(t, err)

        completed, err := client.WaitForCompletionWithTimeout(ctx, createOp.ID, 2*time.Minute)
        require.NoError(t, err)
        volumeID := completed.VolumeID

        // Attach volume
        attachOp, err := client.AttachVolume(ctx, volumeID, testVMID)
        require.NoError(t, err)

        attachCompleted, err := client.WaitForCompletionWithTimeout(ctx, attachOp.ID, 2*time.Minute)
        require.NoError(t, err)
        assert.Equal(t, "completed", attachCompleted.Status)

        // Verify device mapping
        mapping, err := client.GetVolumeDevice(ctx, volumeID)
        require.NoError(t, err)
        assert.NotEmpty(t, mapping.DevicePath)
        assert.Equal(t, volumeID, mapping.VolumeID)

        // Detach volume
        detachOp, err := client.DetachVolume(ctx, volumeID)
        require.NoError(t, err)

        _, err = client.WaitForCompletionWithTimeout(ctx, detachOp.ID, 2*time.Minute)
        require.NoError(t, err)

        // Clean up
        deleteOp, err := client.DeleteVolume(ctx, volumeID)
        require.NoError(t, err)
        
        _, err = client.WaitForCompletionWithTimeout(ctx, deleteOp.ID, 1*time.Minute)
        require.NoError(t, err)
    })
}

func TestVolumeClientErrorHandling(t *testing.T) {
    client := NewDefaultVolumeClient()
    ctx := context.Background()

    t.Run("InvalidParameters", func(t *testing.T) {
        req := CreateVolumeRequest{
            Name:           "test-invalid",
            Size:           100, // Too small
            DiskOfferingID: "invalid-id",
            ZoneID:         "invalid-zone",
        }

        _, err := client.CreateVolume(ctx, req)
        assert.Error(t, err)
        
        classified := ClassifyVolumeError(err)
        assert.Equal(t, "InvalidParameter", classified.Type)
    })

    t.Run("NonExistentOperation", func(t *testing.T) {
        _, err := client.GetOperation(ctx, "op-nonexistent")
        assert.Error(t, err)
    })
}

// Benchmark tests
func BenchmarkVolumeOperations(b *testing.B) {
    if testing.Short() {
        b.Skip("Skipping benchmark test")
    }

    client := NewDefaultVolumeClient()
    ctx := context.Background()

    b.Run("CreateVolume", func(b *testing.B) {
        for i := 0; i < b.N; i++ {
            req := CreateVolumeRequest{
                Name:           fmt.Sprintf("bench-volume-%d", i),
                Size:           1 * 1024 * 1024 * 1024,
                DiskOfferingID: "c813c642-d946-49e1-9289-c616dd70206a",
                ZoneID:         "057e86db-c726-4d8c-ab1f-75c5f55d1881",
            }

            operation, err := client.CreateVolume(ctx, req)
            if err != nil {
                b.Fatalf("Failed to create volume: %v", err)
            }

            // Wait for completion (not included in timing)
            b.StopTimer()
            _, err = client.WaitForCompletionWithTimeout(ctx, operation.ID, 2*time.Minute)
            if err != nil {
                b.Fatalf("Volume creation failed: %v", err)
            }
            b.StartTimer()
        }
    })
}
```

### Service Integration Tests

**File**: `internal/oma/workflows/migration_integration_test.go`

```go
package workflows

import (
    "context"
    "testing"
    "time"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    "github.com/vexxhost/migratekit/internal/common"
)

func TestMigrationVolumeIntegration(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test")
    }

    volumeManager := NewVolumeManager(VolumeConfig{
        DiskOfferingID: "c813c642-d946-49e1-9289-c616dd70206a",
        ZoneID:         "057e86db-c726-4d8c-ab1f-75c5f55d1881",
        DefaultSize:    1 * 1024 * 1024 * 1024,
        Timeout:        5 * time.Minute,
    })

    ctx := context.Background()

    t.Run("CreateMigrationVolume", func(t *testing.T) {
        vmName := "test-vm-" + time.Now().Format("20060102-150405")
        size := int64(2 * 1024 * 1024 * 1024) // 2GB

        operation, err := volumeManager.CreateMigrationVolume(ctx, vmName, size)
        require.NoError(t, err)
        assert.Equal(t, "completed", operation.Status)
        assert.NotEmpty(t, operation.VolumeID)

        // Verify metadata
        assert.Equal(t, vmName, operation.Request["metadata"].(map[string]interface{})["vm_name"])
        assert.Equal(t, "migration", operation.Request["metadata"].(map[string]interface{})["purpose"])

        // Clean up
        client := common.NewDefaultVolumeClient()
        deleteOp, err := client.DeleteVolume(ctx, operation.VolumeID)
        require.NoError(t, err)
        
        _, err = client.WaitForCompletionWithTimeout(ctx, deleteOp.ID, 2*time.Minute)
        require.NoError(t, err)
    })

    t.Run("AttachVolumeToOMA", func(t *testing.T) {
        // This test requires a known OMA VM ID
        omaVMID := "vm-oma-test-12345" // Replace with actual OMA VM
        
        // Create volume first
        vmName := "test-attach-" + time.Now().Format("20060102-150405")
        operation, err := volumeManager.CreateMigrationVolume(ctx, vmName, 1*1024*1024*1024)
        require.NoError(t, err)
        volumeID := operation.VolumeID

        // Attach to OMA
        devicePath, err := volumeManager.AttachVolumeToOMA(ctx, volumeID, omaVMID)
        require.NoError(t, err)
        assert.NotEmpty(t, devicePath)
        assert.Contains(t, devicePath, "/dev/vd")

        // Verify device mapping
        client := common.NewDefaultVolumeClient()
        mapping, err := client.GetVolumeDevice(ctx, volumeID)
        require.NoError(t, err)
        assert.Equal(t, devicePath, mapping.DevicePath)
        assert.Equal(t, "attached", mapping.CloudStackState)

        // Clean up - detach first
        detachOp, err := client.DetachVolume(ctx, volumeID)
        require.NoError(t, err)
        _, err = client.WaitForCompletionWithTimeout(ctx, detachOp.ID, 2*time.Minute)
        require.NoError(t, err)

        // Then delete
        deleteOp, err := client.DeleteVolume(ctx, volumeID)
        require.NoError(t, err)
        _, err = client.WaitForCompletionWithTimeout(ctx, deleteOp.ID, 2*time.Minute)
        require.NoError(t, err)
    })
}
```

---

## Rollback Procedures

### Integration Rollback Plan

If integration fails, you can rollback to direct CloudStack calls:

#### 1. Service Rollback

**Create rollback flags**:
```go
// In service configuration
type ServiceConfig struct {
    UseVolumeDaemon bool `env:"USE_VOLUME_DAEMON" default:"false"`
    // ... other config
}

// In service implementation
func (w *MigrationWorkflow) attachVolumes(ctx context.Context) error {
    if w.config.UseVolumeDaemon {
        return w.attachVolumesViaDaemon(ctx)
    }
    return w.attachVolumesDirect(ctx) // Original implementation
}
```

#### 2. Database Rollback

**Preserve original tables**:
```sql
-- Before integration, backup existing data
CREATE TABLE vm_export_mappings_backup AS SELECT * FROM vm_export_mappings;
CREATE TABLE ossea_volumes_backup AS SELECT * FROM ossea_volumes;

-- Rollback procedure
DROP TABLE vm_export_mappings;
DROP TABLE ossea_volumes;
RENAME TABLE vm_export_mappings_backup TO vm_export_mappings;
RENAME TABLE ossea_volumes_backup TO ossea_volumes;
```

#### 3. Configuration Rollback

**Environment variable toggles**:
```bash
# Disable volume daemon integration
export USE_VOLUME_DAEMON=false

# Restart services
systemctl restart oma-api
systemctl restart migration-worker
systemctl restart failover-engine
```

### Emergency Procedures

**If volume daemon fails during operation**:

1. **Switch to maintenance mode**:
```bash
# Stop new operations
systemctl stop oma-api

# Set maintenance flag
echo "maintenance" > /tmp/migration-maintenance-mode
```

2. **Complete pending operations manually**:
```bash
# List pending operations
mysql -u oma_user -poma_password migratekit_oma -e "
SELECT id, type, volume_id, vm_id 
FROM volume_operations 
WHERE status IN ('pending', 'executing');"

# Complete operations via CloudStack API directly
# (Use existing direct CloudStack tools)
```

3. **Resume with direct CloudStack**:
```bash
# Set rollback flag
export USE_VOLUME_DAEMON=false

# Restart services
systemctl start oma-api
rm /tmp/migration-maintenance-mode
```

---

## Best Practices

### 1. Gradual Integration

**Phase the integration**:
```
Phase 1: New migrations only
Phase 2: Failover operations  
Phase 3: All volume operations
Phase 4: Remove direct CloudStack code
```

### 2. Monitoring Integration

**Add integration monitoring**:
```go
func (vc *VolumeClient) doRequest(ctx context.Context, method, endpoint string, body interface{}) (*http.Response, error) {
    start := time.Now()
    defer func() {
        duration := time.Since(start)
        
        // Metrics
        volumeDaemonRequestDuration.WithLabelValues(method, endpoint).Observe(duration.Seconds())
        volumeDaemonRequestTotal.WithLabelValues(method, endpoint).Inc()
        
        // Logging
        log.WithFields(log.Fields{
            "method":      method,
            "endpoint":    endpoint,
            "duration_ms": duration.Milliseconds(),
        }).Debug("Volume daemon API call completed")
    }()
    
    // ... rest of implementation
}
```

### 3. Error Context

**Provide rich error context**:
```go
func (w *MigrationWorkflow) createVolume(ctx context.Context, diskIndex int, size int64) error {
    operation, err := w.volumeClient.CreateVolume(ctx, req)
    if err != nil {
        // Add context before returning
        return fmt.Errorf("failed to create volume for disk %d (size: %d, vm: %s): %w", 
            diskIndex, size, w.job.VMName, err)
    }
    
    // ... rest of implementation
}
```

### 4. Health Checks

**Verify daemon health before operations**:
```go
func (w *MigrationWorkflow) Start(ctx context.Context) error {
    // Health check before starting
    if err := w.volumeClient.HealthCheck(ctx); err != nil {
        return fmt.Errorf("volume daemon unhealthy, cannot start migration: %w", err)
    }
    
    // Continue with migration
    return w.execute(ctx)
}
```

### 5. Operation Timeouts

**Use appropriate timeouts for different operations**:
```go
var operationTimeouts = map[string]time.Duration{
    "create": 5 * time.Minute,  // Volume creation
    "attach": 2 * time.Minute,  // Volume attachment
    "detach": 1 * time.Minute,  // Volume detachment
    "delete": 2 * time.Minute,  // Volume deletion
}

func (vc *VolumeClient) WaitForOperation(ctx context.Context, operationID, operationType string) (*VolumeOperation, error) {
    timeout, ok := operationTimeouts[operationType]
    if !ok {
        timeout = 5 * time.Minute // Default
    }
    
    return vc.WaitForCompletionWithTimeout(ctx, operationID, timeout)
}
```

This integration guide provides a comprehensive approach to migrating existing services to use the Volume Management Daemon while maintaining system stability and providing rollback options.
