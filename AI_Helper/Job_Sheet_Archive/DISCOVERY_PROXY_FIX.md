# Discovery System Proxy Fix

**Date**: October 4, 2025  
**Issue**: GUI calling OMA API directly instead of using Next.js proxy routes  
**Status**: ✅ **FIXED**

---

## 🐛 **PROBLEM**

After deploying the discovery system repair, the GUI showed "Failed to load VMware credentials" error with HTTP 404 responses.

**Root Cause**: The fix incorrectly had the GUI calling `http://localhost:8082` (OMA API) directly instead of using Next.js proxy routes at `/api/v1/*`.

---

## ✅ **FIX APPLIED**

### **1. Corrected DiscoveryView.tsx API Calls**

**Changed FROM** (direct OMA API calls):
```typescript
fetch('http://localhost:8082/api/v1/vmware-credentials', { ... })
fetch('http://localhost:8082/api/v1/discovery/discover-vms', { ... })
fetch('http://localhost:8082/api/v1/discovery/add-vms', { ... })
```

**Changed TO** (Next.js proxy routes):
```typescript
fetch('/api/v1/vmware-credentials')
fetch('/api/discovery/discover-vms', { ... })
fetch('/api/discovery/add-vms', { ... })
```

### **2. Created Missing Proxy Routes**

**Created**: `/migration-dashboard/src/app/api/discovery/discover-vms/route.ts`
```typescript
export async function POST(request: NextRequest) {
  const body = await request.json();
  const omaResponse = await fetch('http://localhost:8082/api/v1/discovery/discover-vms', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'Authorization': 'Bearer sess_longlived_dev_token_2025_2035_permanent'
    },
    body: JSON.stringify(body),
  });
  return NextResponse.json(await omaResponse.json());
}
```

**Created**: `/migration-dashboard/src/app/api/discovery/add-vms/route.ts`
```typescript
export async function POST(request: NextRequest) {
  const body = await request.json();
  const omaResponse = await fetch('http://localhost:8082/api/v1/discovery/add-vms', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'Authorization': 'Bearer sess_longlived_dev_token_2025_2035_permanent'
    },
    body: JSON.stringify(body),
  });
  return NextResponse.json(await omaResponse.json());
}
```

---

## 🧪 **VERIFICATION TESTS**

### **Test 1: VMware Credentials Proxy**
```bash
curl http://localhost:3001/api/v1/vmware-credentials
```
**Result**: ✅ Returns 1 credential (HTTP 200)

### **Test 2: Discovery Proxy**
```bash
curl -X POST http://localhost:3001/api/discovery/discover-vms \
  -d '{"credential_id": 2, "filter": "pgtest", "create_context": false}'
```
**Result**: ✅ Returns 3 VMs with full data (HTTP 200)

### **Test 3: Add VMs Proxy**
```bash
curl -X POST http://localhost:3001/api/discovery/add-vms \
  -d '{"credential_id": 2, "vm_names": ["pgtest3"], "added_by": "test"}'
```
**Result**: ✅ Successfully adds VM to management (HTTP 200)

---

## 📋 **FILES MODIFIED**

1. **DiscoveryView.tsx** - Changed to use proxy routes
2. **Created**: `api/discovery/discover-vms/route.ts` - New proxy
3. **Created**: `api/discovery/add-vms/route.ts` - New proxy

---

## 🎯 **WHY USE PROXY ROUTES?**

### **GUI Architecture Pattern**:
```
Browser → Next.js GUI (port 3001) → Proxy Routes → OMA API (port 8082)
```

**Reasons**:
1. **CORS**: Avoids cross-origin issues
2. **Authentication**: Proxy adds auth tokens automatically
3. **Security**: OMA API not exposed directly to browser
4. **Consistency**: All GUI API calls use same pattern
5. **Deployment**: Works regardless of hostname/port config

### **Existing Proxy Routes**:
- ✅ `/api/v1/vmware-credentials` → Already existed
- ✅ `/api/v1/vmware-credentials/[id]` → Already existed
- ✅ `/api/replicate` → Already existed
- ✅ `/api/vm-contexts` → Already existed
- ✅ NEW: `/api/discovery/discover-vms` → Created
- ✅ NEW: `/api/discovery/add-vms` → Created

---

## ✅ **STATUS**

**Production**: 10.246.5.124:3001  
**GUI**: Redeployed with proxy route fixes  
**Credentials API**: Working via `/api/v1/vmware-credentials`  
**Discovery API**: Working via `/api/discovery/discover-vms`  
**Add VMs API**: Working via `/api/discovery/add-vms`  

**All functionality now operational on production! ✅**

---

**Fixed By**: AI Assistant  
**Date**: October 4, 2025  
**Time to Fix**: 10 minutes

