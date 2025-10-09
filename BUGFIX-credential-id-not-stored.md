# CRITICAL BUG FIX: credential_id Not Stored When Adding VMs to Management

## **Impact:**
- VMs added via GUI "Add to Management" have `credential_id = NULL` in database
- Backups fail with: `"VM context has no credential_id set"`
- Multi-vCenter environments won't work (VMs don't know which vCenter they belong to)

## **Affected:**
- pgtest2: Added today, has `credential_id = NULL` (manually fixed)
- Any VM added via `/api/v1/discovery/add-vms` or `/api/v1/discovery/discover-vms` with `create_context=true`

---

## **Root Cause Analysis:**

### **Complete Data Flow:**

1. **GUI → Backend Handler** (`enhanced_discovery.go:349`)
   - GUI sends: `{ credential_id: 35, vm_names: ["pgtest2"] }`
   - Handler receives: `request.CredentialID = 35` ✅

2. **Handler → Service** (`enhanced_discovery.go:411-417`)
   ```go
   discoveryRequest := services.DiscoveryRequest{
       VCenter:    vcenter,     // ✅ Stored
       Username:   username,    // ✅ Stored
       Password:   password,    // ✅ Stored
       Datacenter: datacenter,  // ✅ Stored
       Filter:     "",
       // ❌ MISSING: CredentialID
   }
   ```
   **BUG:** `credential_id` is NOT added to `DiscoveryRequest`

3. **Service → processDiscoveredVMs** (`enhanced_discovery_service.go:222`)
   - `processDiscoveredVMs` never receives `credential_id`

4. **processDiscoveredVMs → createVMContext** (`enhanced_discovery_service.go:286`)
   - `createVMContext` never receives `credential_id`

5. **createVMContext → Database** (`enhanced_discovery_service.go:344-363`)
   ```go
   vmContext := database.VMReplicationContext{
       ContextID:        contextID,
       VMName:           vm.Name,
       VCenterHost:      vcenter.Host,
       Datacenter:       vcenter.Datacenter,
       // ❌ MISSING: CredentialID field not set
       // ...
   }
   ```
   **BUG:** `VMReplicationContext` struct never sets `CredentialID` field

6. **Database** → `vm_replication_contexts` table
   - `credential_id` column defaults to `NULL`
   - Result: VM has no link to its vCenter credentials

---

## **THE COMPLETE FIX:**

### **Step 1: Add CredentialID to DiscoveryRequest struct**

**File:** `source/current/sha/services/enhanced_discovery_service.go`

**Location:** Line 38

**Change:**
```go
type DiscoveryRequest struct {
	VCenter      string `json:"vcenter" binding:"required"`
	Username     string `json:"username" binding:"required"`
	Password     string `json:"password" binding:"required"`
	Datacenter   string `json:"datacenter" binding:"required"`
	Filter       string `json:"filter,omitempty"` // Optional VM name filter
	CredentialID *int   `json:"credential_id,omitempty"` // 🆕 NEW: Link to vmware_credentials table
}
```

---

### **Step 2: Pass CredentialID from Handler to Service**

**File:** `source/current/sha/api/handlers/enhanced_discovery.go`

**Location:** Lines 411-417

**Change:**
```go
// Convert to service request format for internal processing
discoveryRequest := services.DiscoveryRequest{
	VCenter:      vcenter,
	Username:     username,
	Password:     password,
	Datacenter:   datacenter,
	Filter:       "", // No filter for specific VM add
	CredentialID: request.CredentialID, // 🆕 NEW: Pass credential_id to service
}
```

---

### **Step 3: Pass CredentialID to createVMContext**

**File:** `source/current/sha/services/enhanced_discovery_service.go`

**Location 1:** Line 222 (processDiscoveredVMs function)

Add parameter to function signature:

```go
func (eds *EnhancedDiscoveryService) processDiscoveredVMs(
	ctx context.Context, 
	snaResponse *SNADiscoveryResponse,
	selectedVMNames []string, 
	result *BulkAddResult,
	credentialID *int, // 🆕 NEW: Pass credential_id through
) error {
```

**Location 2:** Line 286 (call to createVMContext)

```go
contextID, err := eds.createVMContext(ctx, vm, vcenterInfo, credentialID) // 🆕 NEW: Pass credential_id
```

**Location 3:** Line 222 (call to processDiscoveredVMs)

```go
return eds.processDiscoveredVMs(ctx, snaResponse, selectedVMNames, result, discoveryRequest.CredentialID)
```

---

### **Step 4: Update createVMContext to Store CredentialID**

**File:** `source/current/sha/services/enhanced_discovery_service.go`

**Location 1:** Line 322 (function signature)

```go
func (eds *EnhancedDiscoveryService) createVMContext(
	ctx context.Context, 
	vm SNAVMInfo,
	vcenter struct{ Host, Datacenter string },
	credentialID *int, // 🆕 NEW: Receive credential_id
) (string, error) {
```

**Location 2:** Line 344-363 (VMReplicationContext struct)

```go
// Create VM context
vmContext := database.VMReplicationContext{
	ContextID:        contextID,
	VMName:           vm.Name,
	VMwareVMID:       vm.ID,
	VMPath:           vm.Path,
	VCenterHost:      vcenter.Host,
	Datacenter:       vcenter.Datacenter,
	CredentialID:     credentialID, // 🆕 NEW: Store credential_id link
	CurrentStatus:    "discovered",
	OSSEAConfigID:    &osseaConfigID,
	CPUCount:         &vm.NumCPU,
	MemoryMB:         &vm.MemoryMB,
	OSType:           &osType,
	PowerState:       &vm.PowerState,
	VMToolsVersion:   &vm.VMXVersion,
	AutoAdded:        true,
	SchedulerEnabled: true,
	CreatedAt:        time.Now(),
	UpdatedAt:        time.Now(),
	LastStatusChange: time.Now(),
}
```

**Location 3:** Line 370 (add logging)

```go
log.Info("Created VM context",
	"context_id", contextID,
	"vm_name", vm.Name,
	"vm_path", vm.Path,
	"vcenter", vcenter.Host,
	"credential_id", credentialID, // 🆕 NEW: Log credential_id for debugging
	"auto_added", true,
	"disk_count", len(vm.Disks))
```

---

### **Step 5: Fix DiscoverVMs endpoint (same issue)**

**File:** `source/current/sha/api/handlers/enhanced_discovery.go`

**Location:** Line 218 (in DiscoverVMs handler)

**Change:**
```go
// Discover VMs from SNA (using resolved credentials)
discoveryReq := services.DiscoveryRequest{
	VCenter:      vcenter,
	Username:     username,
	Password:     password,
	Datacenter:   datacenter,
	Filter:       request.Filter,
	CredentialID: request.CredentialID, // 🆕 NEW: Pass credential_id
}
```

---

## **Testing After Fix:**

### **Test 1: Add VM with credential_id**
```bash
# Add PhilB Test machine again
curl -X POST http://localhost:8082/api/v1/discovery/add-vms \
  -H "Content-Type: application/json" \
  -d '{
    "credential_id": 35,
    "vm_names": ["PhilB Test machine"]
  }'

# Verify credential_id was stored
mysql -u oma_user -p'oma_password' -D migratekit_oma -e \
  "SELECT vm_name, credential_id, vcenter_host FROM vm_replication_contexts WHERE vm_name = 'PhilB Test machine';"

# Expected result:
# vm_name              credential_id  vcenter_host
# PhilB Test machine   35             quad-vcenter-01.quadris.local
```

### **Test 2: Run backup flow**
```bash
# Create protection flow for PhilB group
# Click "Run Now" in GUI

# Check logs - should NOT see "VM context has no credential_id set"
sudo journalctl -u sendense-hub.service --since "1 minute ago" | grep credential

# Backup should start successfully
```

---

## **Database Verification:**

```sql
-- Check all VMs and their credential assignments
SELECT 
    c.vm_name,
    c.credential_id,
    c.vcenter_host,
    v.credential_name,
    v.vcenter_host as cred_vcenter
FROM vm_replication_contexts c
LEFT JOIN vmware_credentials v ON c.credential_id = v.id
ORDER BY c.created_at DESC
LIMIT 10;

-- Expected: All VMs should have credential_id populated
-- Any NULL values = bug still exists
```

---

## **Rollout Plan:**

1. ✅ **Manual fix applied:** pgtest2 set to credential_id = 35
2. 🔧 **Code fix:** Apply all 5 steps above
3. 🏗️  **Build:** `go build -o /tmp/sendense-hub ./cmd/main.go`
4. 🚀 **Deploy:** Stop service, replace binary, restart
5. ✅ **Test:** Add PhilB Test machine, verify credential_id stored
6. ✅ **Test:** Run protection flow, verify backup works
7. 📝 **Document:** Update CHANGELOG.md

---

## **Prevention:**

### **Add Database Constraint (Optional):**
```sql
-- Make credential_id NOT NULL after fixing existing VMs
-- ALTER TABLE vm_replication_contexts MODIFY credential_id INT NOT NULL;

-- Add index for faster lookups
CREATE INDEX idx_credential_id ON vm_replication_contexts(credential_id);
```

### **Add Validation in Code:**
In backup handlers, add early validation:
```go
if vmContext.CredentialID == nil || *vmContext.CredentialID == 0 {
    return fmt.Errorf("VM context has no credential_id set - cannot perform backup")
}
```

---

## **Files Modified:**

1. `source/current/sha/services/enhanced_discovery_service.go` (3 changes)
2. `source/current/sha/api/handlers/enhanced_discovery.go` (2 changes)

**Total:** 5 code changes across 2 files

