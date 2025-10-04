# 🔐 VMware Credentials Integration Guide

**Version:** v6.15.0-security-cleanup  
**Date:** October 3, 2025  

---

## Overview

The GUI now integrates with the VMware credentials database system, allowing users to:
- **Save credentials once** in Settings → VMware
- **Reuse credentials** automatically across all discovery forms
- **Avoid re-entering** credentials for every discovery operation
- **Manage multiple vCenters** easily

---

## How It Works

### 1. **Credential Storage** (Settings → VMware)

Users add credentials via the VMware Credentials Manager:
```
Credential Name: "Production vCenter"
vCenter Host: vcenter.company.com
Username: administrator@vsphere.local
Password: ••••••••
Datacenter: Datacenter1
```

Credentials are stored in the `vmware_credentials` table with encrypted passwords.

### 2. **Discovery Form** (Discovery Page)

The discovery form now has a dropdown selector:

```
┌─────────────────────────────────────────────────────┐
│ vCenter Credentials                                  │
│ ┌───────────────────────────────────────────────┐  │
│ │ Production vCenter (vcenter.company.com) ⭐   │  │  <-- Dropdown
│ │ DR vCenter (vcenter-dr.company.com)           │  │
│ │ ──────────────                                │  │
│ │ Manual Entry                                  │  │
│ └───────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────┐
│ [vcenter.company.com]  [admin@vsphere] [••••••••]  │  <-- Auto-filled & disabled
└─────────────────────────────────────────────────────┘
```

**Features:**
- **Auto-selection:** Default credential (⭐) is selected automatically
- **Auto-populate:** Fields populate from database when credential is selected
- **Disabled fields:** Cannot edit when using saved credentials (prevents mistakes)
- **Manual override:** Select "Manual Entry" to enter credentials manually

### 3. **API Communication**

#### When Using Saved Credentials:
```javascript
// GUI sends credential ID (not password!)
POST /api/discover
{
  "credential_id": 1,
  "filter": "optional-vm-name"
}

// OMA API fetches credentials from database
// Decrypts password automatically
// Makes vCenter call with full credentials
```

#### When Using Manual Entry:
```javascript
// GUI sends full credentials
POST /api/discover
{
  "vcenter": "vcenter.company.com",
  "username": "admin@vsphere.local",
  "password": "password123",
  "datacenter": "Datacenter1",
  "filter": "optional-vm-name"
}
```

---

## User Workflow

### First-Time Setup

1. Navigate to **Settings → VMware**
2. Click "Add Credential"
3. Fill in vCenter details
4. Check "Set as Default" ✅
5. Click "Create"

### Daily Usage

1. Go to **Discovery** page
2. Dropdown automatically shows default credential
3. Click "Discover" (no typing needed!)
4. VMs appear in list

**Time saved:** ~30 seconds per discovery operation

---

## Code Changes (v6.15.0)

### Frontend Changes

**File:** `gui/src/components/discovery/DiscoveryView.tsx`

```typescript
// Added credential management state
const [savedCredentials, setSavedCredentials] = useState<VMwareCredential[]>([]);
const [selectedCredentialId, setSelectedCredentialId] = useState<number | null>(null);

// Load credentials on mount
useEffect(() => {
  const loadCredentials = async () => {
    const response = await fetch('/api/v1/vmware-credentials');
    const data = await response.json();
    setSavedCredentials(data.credentials || []);
    
    // Auto-select default
    const defaultCred = data.credentials?.find(c => c.is_default);
    if (defaultCred) {
      setSelectedCredentialId(defaultCred.id);
      setVcenterHost(defaultCred.vcenter_host);
      // ... populate other fields
    }
  };
  loadCredentials();
}, []);

// Send credential ID or full credentials
const discoverVMs = async () => {
  const requestBody = selectedCredentialId
    ? { credential_id: selectedCredentialId, filter }
    : { vcenter, username, password, datacenter, filter };
    
  const response = await fetch('/api/discover', {
    method: 'POST',
    body: JSON.stringify(requestBody)
  });
};
```

### Backend Requirements

**OMA API** must support both modes:

```go
// Pseudo-code
func DiscoverVMs(req Request) {
    var creds VMwareCredentials
    
    if req.CredentialID != 0 {
        // Load from database
        creds = database.GetCredentialByID(req.CredentialID)
        creds.Password = decrypt(creds.PasswordEncrypted)
    } else {
        // Use provided credentials
        creds = VMwareCredentials{
            Host: req.Vcenter,
            Username: req.Username,
            Password: req.Password,
            Datacenter: req.Datacenter,
        }
    }
    
    // Make vCenter call
    vms := vmware.DiscoverVMs(creds, req.Filter)
    return vms
}
```

---

## Security Benefits

### Before (v6.14.0 and earlier)
❌ **Hardcoded credentials** in source code  
❌ **Credentials visible** in browser DevTools  
❌ **No credential management**  
❌ **Repeated manual entry** required  

### After (v6.15.0)
✅ **No hardcoded credentials**  
✅ **Encrypted storage** in database  
✅ **Credential ID sent** (not passwords)  
✅ **Automatic reuse** of saved credentials  
✅ **Centralized management**  

---

## User Benefits

| Feature | Before | After |
|---------|--------|-------|
| Credential Entry | Every discovery | Once in settings |
| Time per Discovery | ~30 seconds typing | 1 click |
| Multiple vCenters | Re-type each time | Select from dropdown |
| Typos | Common | Prevented (pre-validated) |
| Password Visibility | Visible in form | Hidden (using saved) |

---

## Admin Features

### Credential Management

**Location:** Settings → VMware

**Capabilities:**
- ✅ Add/Edit/Delete credentials
- ✅ Set default credential (auto-selected)
- ✅ Test connection before saving
- ✅ Track usage statistics
- ✅ Mark credentials as active/inactive

**Database Fields:**
```sql
CREATE TABLE vmware_credentials (
  id INT PRIMARY KEY AUTO_INCREMENT,
  credential_name VARCHAR(255) NOT NULL,
  vcenter_host VARCHAR(255) NOT NULL,
  username VARCHAR(255) NOT NULL,
  password_encrypted TEXT NOT NULL,
  datacenter VARCHAR(255) NOT NULL,
  is_active BOOLEAN DEFAULT TRUE,
  is_default BOOLEAN DEFAULT FALSE,
  created_at TIMESTAMP,
  updated_at TIMESTAMP,
  last_used TIMESTAMP,
  usage_count INT DEFAULT 0
);
```

---

## Testing Checklist

### GUI Testing
- [ ] Load discovery page → default credential selected
- [ ] Switch between credentials → fields update
- [ ] Select "Manual Entry" → fields become editable
- [ ] Click Discover with saved credential → works
- [ ] Click Discover with manual entry → works
- [ ] No saved credentials → shows helpful message with link

### Backend Testing
- [ ] Send credential_id → OMA fetches from DB
- [ ] Send manual credentials → OMA uses provided values
- [ ] Invalid credential_id → returns error
- [ ] Encrypted password → decrypted correctly

### Security Testing
- [ ] Browser DevTools → no plaintext passwords visible
- [ ] Network traffic → credential_id sent (not password)
- [ ] Database → passwords encrypted
- [ ] Failed auth → credential marked in logs

---

## Migration Path

### For Existing Users

1. **Add credentials** in Settings → VMware
2. **Test connection** before saving
3. **Set as default** if primary vCenter
4. **Use discovery page** normally (auto-populated)

### For Fresh Deployments

Credentials start empty:
1. First visit → "Manual Entry" selected
2. Helpful message → link to Settings → VMware
3. Add credentials → return to discovery
4. Dropdown now shows saved credentials

---

## API Endpoints

### Frontend → GUI Backend
```
GET  /api/v1/vmware-credentials          # List all credentials
GET  /api/v1/vmware-credentials/default  # Get default credential
POST /api/v1/vmware-credentials          # Create credential
PUT  /api/v1/vmware-credentials/:id      # Update credential
DELETE /api/v1/vmware-credentials/:id    # Delete credential
POST /api/v1/vmware-credentials/:id/test # Test connection
POST /api/v1/vmware-credentials/:id/set-default # Set as default
```

### GUI Backend → OMA API
```
POST /api/v1/discover  # Discovery with credential_id or manual
```

---

## Future Enhancements

### Planned Features
1. **Credential rotation** (auto-detect password changes)
2. **Multi-user support** (per-user credentials)
3. **Credential groups** (dev, staging, prod)
4. **Auto-discovery** of all vCenters from credential list
5. **Credential health check** (periodic validation)
6. **Audit logging** (who used which credential when)

### Nice-to-Have
- Import/export credentials (encrypted)
- Credential templates
- Integration with external secret managers (Vault, etc.)

---

## Troubleshooting

### "No saved credentials" message
**Cause:** No credentials in database  
**Fix:** Go to Settings → VMware → Add Credential

### Fields are disabled/greyed out
**Cause:** Saved credential is selected  
**Fix:** Select "Manual Entry" from dropdown to edit fields

### Discovery fails with "Invalid credentials"
**Cause:** Saved credential is outdated  
**Fix:** Settings → VMware → Edit credential → Update password

### Dropdown doesn't show my credential
**Cause:** Credential marked as inactive  
**Fix:** Settings → VMware → Edit → Check "Active"

---

## Summary

The credential integration system provides:
- ✅ **Secure storage** of vCenter credentials
- ✅ **Convenient reuse** across all forms
- ✅ **Time savings** (1 click vs 30 seconds typing)
- ✅ **Multi-vCenter support** with easy switching
- ✅ **Production-ready** security practices

**Result:** Users save credentials once, never type them again!



