# CloudStack Validation - Real Requirements

**Created**: October 3, 2025  
**Status**: 📋 **PLANNING PHASE** - Testing approaches before implementation

---

## 🎯 **ACTUAL PROBLEMS TO SOLVE**

Based on real deployment issues, here's what we **actually** need:

---

## 1️⃣ **OMA VM ID Auto-Detection** 

### **Current Problem**:
- Manual OMA VM ID entry is error-prone
- Users don't know how to find their VM ID in CloudStack
- Wrong VM ID causes volume attachment failures

### **Proposed Solution**:
Auto-detect OMA VM ID using MAC address lookup in CloudStack

### **Implementation Plan**:

**Step 1: Get OMA's MAC Address**
```bash
# Primary interface MAC
ip link show $(ip route show default | awk '/default/ {print $5}') | grep "link/ether" | awk '{print $2}'
```

**Step 2: Query CloudStack for VM with that MAC**
```python
def find_oma_vm_by_mac(cloudstack_client, target_mac):
    vms = cloudstack_client.listVirtualMachines()
    for vm in vms:
        for nic in vm.get('nic', []):
            if nic.get('macaddress', '').lower() == target_mac.lower():
                return vm['id']
    return None
```

**Step 3: Validation**
```python
if oma_vm_id := auto_detect_oma_vm():
    validate_vm_exists(oma_vm_id)
    validate_vm_running(oma_vm_id)
    save_to_config(oma_vm_id)
else:
    # Fallback to manual entry
    require_manual_oma_vm_id()
```

**Blocking Rules**:
- ❌ **BLOCK all replication operations** until valid OMA VM ID is set
- ❌ **BLOCK volume operations** without OMA VM ID
- ✅ Allow manual override if auto-detection fails

### **Testing Required**:
- [ ] Does CloudStack API return MAC addresses in VM NIC info?
- [ ] Which interface MAC to use (primary vs all)?
- [ ] What if multiple VMs share same MAC (unlikely but possible)?
- [ ] Fallback behavior if auto-detection fails?

**Test Script**: `scripts/test-cloudstack-prerequisites.sh` (section 1)

---

## 2️⃣ **Network Selection - User Choice Only**

### **Current Problem**:
Auto-selecting network removes user control over critical topology decisions

### **Solution**:
User must explicitly select network, no auto-selection

### **Implementation Plan**:

**Discovery**:
```python
networks = cloudstack.listNetworks()
return [{
    'id': net['id'],
    'name': net['name'],
    'zone': net['zonename'],
    'state': net['state'],
    'cidr': net.get('cidr', 'N/A')
} for net in networks]
```

**Validation**:
```python
if not config.network_id:
    raise ValidationError("Network must be selected by user")
```

**NO Auto-Fix**: User must make conscious choice

### **GUI Behavior**:
- Show all available networks in dropdown
- Require user to select one
- Show network details (zone, CIDR) to help decision
- No default selection (force explicit choice)

---

## 3️⃣ **Compute Offering - Custom CPU/Memory/Disk Required**

### **Current Problem**:
Migrations need flexible resource allocation:
- Variable CPU counts (from source VMs)
- Variable memory sizes (from source VMs)  
- Variable root disk sizes (from source VMs)

Fixed offerings don't work!

### **Requirements**:
Service offering MUST have:
1. ✅ Customizable CPU (`cpunumber` can be set at VM creation)
2. ✅ Customizable Memory (`memory` can be set at VM creation)
3. ✅ Custom Root Disk Size = 0 or customizable

### **Implementation Plan**:

**Discovery & Filtering**:
```python
def find_suitable_compute_offerings(cloudstack):
    offerings = cloudstack.listServiceOfferings()
    suitable = []
    
    for offering in offerings:
        # Check for customization support
        is_customized = offering.get('customized', False)
        root_disk_size = offering.get('rootdisksize', -1)
        
        # Offering is suitable if:
        # 1. customized=true (allows CPU/memory/disk customization)
        # OR
        # 2. rootdisksize=0 (no fixed root disk, can specify at creation)
        
        if is_customized or root_disk_size == 0:
            suitable.append({
                'id': offering['id'],
                'name': offering['displaytext'],
                'customized': is_customized,
                'root_disk_size': root_disk_size,
                'default_cpu': offering.get('cpunumber'),
                'default_memory': offering.get('memory')
            })
    
    return suitable
```

**Validation**:
```python
def validate_compute_offering(cloudstack, offering_id):
    offering = cloudstack.getServiceOffering(offering_id)
    
    if not offering.get('customized') and offering.get('rootdisksize', -1) != 0:
        raise ValidationError(
            f"Service offering '{offering['name']}' has fixed root disk size "
            f"({offering['rootdisksize']} GB). Migrations require offerings "
            f"with customizable root disk (rootdisksize=0 or customized=true)."
        )
```

**Auto-Creation** (if admin permissions):
```python
def create_migration_compute_offering(cloudstack):
    try:
        offering = cloudstack.createServiceOffering({
            'name': 'Migration-Custom',
            'displaytext': 'Customizable CPU/Memory/Disk for VM Migrations',
            'customized': True,  # Allows all customization
            # No cpunumber, memory, rootdisksize = fully customizable
        })
        return offering['id']
    except PermissionError:
        raise ValidationError(
            "No suitable compute offering found and insufficient permissions "
            "to create one. Please contact CloudStack administrator to create "
            "a compute offering with customizable CPU, memory, and root disk."
        )
```

**Blocking Rules**:
- ❌ **BLOCK VM creation** if selected offering has fixed root disk
- ❌ **BLOCK configuration save** if no suitable offerings exist
- ⚠️ **OFFER creation** if admin permissions detected

### **Testing Required**:
- [ ] Which field indicates custom root disk? (`customized`, `rootdisksize`?)
- [ ] Can we create offerings with API key we have?
- [ ] What happens if we try to create VM with wrong offering type?
- [ ] CloudStack error messages for fixed vs custom disk scenarios?

**Test Script**: `scripts/test-cloudstack-prerequisites.sh` (section 2)

---

## 4️⃣ **Disk Offering - For Data Disks**

### **Current Problem**:
Need disk offering for creating data volumes (non-root disks)

### **Requirements**:
Disk offering for data volumes - can be any offering, prefer custom size

### **Implementation Plan**:

**Discovery**:
```python
def find_disk_offerings(cloudstack):
    offerings = cloudstack.listDiskOfferings()
    return [{
        'id': off['id'],
        'name': off['displaytext'],
        'size_gb': off.get('disksize', 0),
        'customized': off.get('customized', False),
        'storage_type': off.get('storagetype', 'shared')
    } for off in offerings]
```

**Validation**:
```python
if not config.disk_offering_id:
    raise ValidationError("Disk offering must be selected")

# Verify exists and accessible
offering = cloudstack.getDiskOffering(config.disk_offering_id)
if not offering:
    raise ValidationError(f"Disk offering {config.disk_offering_id} not found")
```

**User Choice**: Let user pick any disk offering (they know their storage needs)

---

## 5️⃣ **API Key Account Validation** 🔐

### **Current Problem**:
API key from wrong account causes volume attachment failures across accounts

### **Requirements**:
API key MUST belong to same account that owns OMA VM

### **Implementation Plan**:

**Step 1: Determine API Key Account**
```python
def get_api_key_account(cloudstack):
    # Option A: Check current session
    try:
        # CloudStack may have whoami or current user endpoint
        user = cloudstack.getCurrentUser()
        return user['account']
    except:
        pass
    
    # Option B: List accounts and infer from API key permissions
    accounts = cloudstack.listAccounts()
    # If only one account visible, likely the API key's account
    if len(accounts) == 1:
        return accounts[0]['name']
    
    # Option C: Try an operation that returns account info
    vms = cloudstack.listVirtualMachines(listall=False)
    if vms:
        # VMs returned without listall=true are in our account
        return vms[0]['account']
```

**Step 2: Get OMA VM Account**
```python
def get_oma_vm_account(cloudstack, oma_vm_id):
    vm = cloudstack.getVirtualMachine(oma_vm_id)
    return vm['account']
```

**Step 3: Validate Match**
```python
def validate_account_match(cloudstack, oma_vm_id):
    api_account = get_api_key_account(cloudstack)
    oma_account = get_oma_vm_account(cloudstack, oma_vm_id)
    
    if api_account != oma_account:
        raise ValidationError(
            f"Account mismatch: API key belongs to '{api_account}' "
            f"but OMA VM is owned by '{oma_account}'. "
            f"Volume operations will fail across accounts. "
            f"Please use API keys from the OMA VM owner account ({oma_account})."
        )
```

**Blocking Rule**:
- ❌ **HARD BLOCK** - No exceptions
- ❌ **BLOCK all operations** until account matches

### **Testing Required**:
- [ ] How to reliably determine API key's account?
- [ ] Does CloudStack have a "current user" or "whoami" endpoint?
- [ ] What error occurs when attaching volume across accounts?

**Test Script**: `scripts/test-cloudstack-prerequisites.sh` (sections 4, 5, 7)

---

## 6️⃣ **GUI Credential Management** 💻

### **Current Requirements**:
1. Store API key/secret encrypted (don't re-enter every time)
2. "Test Connection" button - validates without saving
3. "Discover Resources" button - populates dropdowns
4. Show current database values when loading config page

### **Implementation Plan**:

**Encryption**:
- Use existing `MIGRATEKIT_CRED_ENCRYPTION_KEY` environment variable
- Same encryption service as VMware credentials

**GUI Flow**:

```javascript
// On page load
async function loadCurrentConfig() {
    const config = await fetch('/api/v1/ossea/config');
    
    // Pre-fill form
    setApiUrl(config.api_url);
    setApiKey(config.api_key);  // Show masked: "abc...xyz"
    setSecretKey('••••••••');  // Always masked
    
    // Pre-select in dropdowns
    setSelectedZone(config.zone);
    setSelectedTemplate(config.template_id);
    setSelectedNetwork(config.network_id);
    setSelectedComputeOffering(config.service_offering_id);
    setSelectedDiskOffering(config.disk_offering_id);
    setOmaVmId(config.oma_vm_id);
}

// Test Connection (no save)
async function testConnection() {
    const result = await fetch('/api/v1/ossea/test-connection', {
        method: 'POST',
        body: JSON.stringify({
            api_url: apiUrl,
            api_key: apiKey,
            secret_key: secretKey
        })
    });
    
    if (result.ok) {
        showSuccess("✅ Connection successful!");
        enableDiscoverButton();
    } else {
        showError(result.error);
    }
}

// Discover Resources (no save)
async function discoverResources() {
    setLoading(true);
    
    const resources = await fetch('/api/v1/ossea/discover-resources', {
        method: 'POST',
        body: JSON.stringify({
            api_url: apiUrl,
            api_key: apiKey,
            secret_key: secretKey
        })
    });
    
    // Populate dropdowns
    setZones(resources.zones);
    setNetworks(resources.networks);
    setComputeOfferings(resources.compute_offerings);
    setDiskOfferings(resources.disk_offerings);
    
    // Highlight current selections if they exist
    if (currentConfig.zone) {
        highlightInDropdown('zone', currentConfig.zone);
    }
    
    setLoading(false);
}

// Save Configuration
async function saveConfiguration() {
    // Validate first
    if (!selectedNetwork) {
        showError("Network must be selected");
        return;
    }
    
    if (!isComputeOfferingSuitable(selectedComputeOffering)) {
        showError("Compute offering must support custom root disk");
        return;
    }
    
    // Save
    await fetch('/api/v1/ossea/config', {
        method: 'POST',
        body: JSON.stringify({
            api_url: apiUrl,
            api_key: apiKey,
            secret_key: secretKey,
            zone: selectedZone,
            template_id: selectedTemplate,
            network_id: selectedNetwork,
            service_offering_id: selectedComputeOffering,
            disk_offering_id: selectedDiskOffering,
            oma_vm_id: omaVmId
        })
    });
    
    showSuccess("Configuration saved!");
}
```

---

## 🧪 **TESTING WORKFLOW**

### **Phase 1: Understand CloudStack Behavior**

Run the test script to see what CloudStack actually returns:

```bash
./scripts/test-cloudstack-prerequisites.sh \
    http://10.245.241.101:8080 \
    YOUR_API_KEY \
    YOUR_SECRET_KEY
```

**Answer these questions**:
1. Can we find VMs by MAC address?
2. Which field indicates custom root disk support?
3. How do we determine API key's account?
4. What fields are available in VM/offering responses?

### **Phase 2: Design Validation Logic**

Based on test results, design:
- Validation checks
- Error messages
- Auto-detection logic
- Fallback strategies

### **Phase 3: Implement & Test**

Build minimal validation focused on real issues:
- OMA VM ID detection/validation
- Network selection requirement
- Compute offering validation
- Account matching
- GUI credential flow

### **Phase 4: Integrate & Validate**

Test with real CloudStack deployments

---

## 📝 **NEXT STEPS**

1. **Run test script** on your CloudStack
2. **Review output** and answer the questions
3. **Share findings** so we can design real solution
4. **Build focused validation** based on actual CloudStack behavior
5. **Test on real deployment** to ensure it catches real issues

---

## 🎯 **SUCCESS CRITERIA**

A successful validation system will:

- ✅ Auto-detect OMA VM ID via MAC (or require manual with validation)
- ✅ Force user to consciously select network
- ✅ Validate compute offering supports custom CPU/memory/disk
- ✅ Hard block if API key account doesn't match OMA VM account
- ✅ Block all operations until valid configuration exists
- ✅ Store credentials encrypted using existing key
- ✅ Show current config values in GUI
- ✅ Provide clear error messages with fix guidance

---

**Status**: 📋 **TESTING PHASE**  
**Next**: Run test script and review CloudStack API responses



