# GROK Code Fast: vCenter Credentials API Integration

**Project:** Sendense Professional GUI - Complete vCenter Credentials Workflow  
**Task:** Replace mock data in Settings → Sources page with real VMware credentials API integration  
**Implementation Tool:** Grok Code Fast  
**Duration:** 1-2 hours (quick fix)  
**Location:** `/home/oma_admin/sendense/source/current/sendense-gui/`

---

## 🎯 CRITICAL CONTEXT

**Current Status:** The VMware backup GUI integration is **95% COMPLETE**. The VMDiscoveryModal is perfectly implemented and ready to use real vCenter credentials, but the Settings → Sources page still uses mock data instead of the real API.

### **The Problem:**
1. ✅ **VMDiscoveryModal** correctly fetches from `/api/v1/vmware-credentials` (lines 66-80)
2. ❌ **Settings → Sources page** uses mock data (lines 45-85) instead of real API
3. ❌ **Users cannot persist credentials** - when they add vCenter credentials, they're not saved
4. ❌ **Broken workflow** - VMDiscoveryModal has no real credentials to discover with

### **The Solution:**
Replace ALL mock data in `app/settings/sources/page.tsx` with real API integration using the existing VMware credentials endpoints.

---

## 🚨 ABSOLUTE PROJECT RULES (NEVER VIOLATE)

### **1. PRESERVE EXISTING FUNCTIONALITY**
- ❌ **FORBIDDEN:** Breaking ANY existing functionality
- ✅ **REQUIRED:** Maintain exact visual appearance and UX
- ✅ **REQUIRED:** Test that Settings → Sources → VMDiscoveryModal workflow works end-to-end

### **2. API INTEGRATION REQUIREMENTS**
- ✅ **MANDATORY:** Use real API endpoints (documented below) - NO mock data
- ✅ **REQUIRED:** Replace mock data with real API calls
- ✅ **REQUIRED:** Maintain all existing visual styling and interaction patterns

---

## 📋 SPECIFIC INTEGRATION TASK

### **Primary Goal:** 
Replace mock data in Settings → Sources page with real VMware credentials API integration so users can add vCenter credentials that the VMDiscoveryModal can use for VM discovery.

### **Expected Workflow After Fix:**
```
Settings → Sources → [+ Add vCenter] → Save credentials to database
                                     ↓
Protection Groups → [+ Add VMs] → Select from REAL saved credentials → Discovery works
```

---

## 🔧 IMPLEMENTATION REQUIREMENTS

### **File to Modify:** `app/settings/sources/page.tsx`

#### **Step 1: Replace Mock Data with Real API (CRITICAL)**

**REMOVE these lines (45-85):**
```typescript
// DELETE ALL MOCK DATA - Lines 45-85
const mockSources: VCenterSource[] = [
  // ... all mock data
];

// DELETE this line (85):
const [sources, setSources] = useState<VCenterSource[]>(mockSources);
```

**REPLACE with Real API Integration:**
```typescript
// Real API integration
const [sources, setSources] = useState<VCenterSource[]>([]);
const [isLoading, setIsLoading] = useState(false);

// Load real credentials on page mount
useEffect(() => {
  const loadVCenterSources = async () => {
    setIsLoading(true);
    try {
      const response = await fetch('/api/v1/vmware-credentials');
      if (response.ok) {
        const credentials = await response.json();
        // Transform API response to match existing interface
        const transformedSources = credentials.map((cred: any) => ({
          id: cred.id,
          name: cred.name,
          host: cred.vcenter_host,
          port: 443, // Default vCenter port
          username: cred.username,
          status: 'connected', // You can enhance this with real connection testing
          lastConnected: cred.updated_at || new Date().toISOString(),
          version: 'Unknown', // Can be enhanced with real version detection
          datacenterCount: 0, // Can be enhanced with real counts
          vmCount: 0 // Can be enhanced with real counts
        }));
        setSources(transformedSources);
      } else {
        console.error('Failed to load VMware credentials');
      }
    } catch (error) {
      console.error('Error loading VMware credentials:', error);
    } finally {
      setIsLoading(false);
    }
  };

  loadVCenterSources();
}, []);
```

#### **Step 2: Update Add/Edit Credential Functions**

**Find and Update handleSave function:**
```typescript
const handleSave = async () => {
  try {
    const credentialData = {
      name: formData.name,
      vcenter_host: formData.host,
      username: formData.username,
      password: formData.password,
      port: parseInt(formData.port) || 443
    };

    let response;
    if (editingSource) {
      // Update existing credential
      response = await fetch(`/api/v1/vmware-credentials/${editingSource.id}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(credentialData)
      });
    } else {
      // Create new credential
      response = await fetch('/api/v1/vmware-credentials', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(credentialData)
      });
    }

    if (response.ok) {
      // Refresh the credentials list
      window.location.reload(); // Simple refresh - can be enhanced
      handleCloseModal();
    } else {
      const error = await response.json();
      console.error('Failed to save credential:', error);
      // Add user feedback here
    }
  } catch (error) {
    console.error('Error saving credential:', error);
    // Add user feedback here
  }
};
```

#### **Step 3: Update Delete Function**
```typescript
const handleDelete = async (sourceId: string) => {
  try {
    const response = await fetch(`/api/v1/vmware-credentials/${sourceId}`, {
      method: 'DELETE'
    });

    if (response.ok) {
      // Remove from local state
      setSources(prev => prev.filter(s => s.id !== sourceId));
    } else {
      console.error('Failed to delete credential');
    }
  } catch (error) {
    console.error('Error deleting credential:', error);
  }
};
```

#### **Step 4: Update Test Connection (Optional Enhancement)**
```typescript
const handleTestConnection = async (sourceId: string) => {
  try {
    const response = await fetch(`/api/v1/vmware-credentials/${sourceId}/test`, {
      method: 'POST'
    });
    
    if (response.ok) {
      const result = await response.json();
      // Update UI to show connection test result
      console.log('Connection test result:', result);
    }
  } catch (error) {
    console.error('Connection test failed:', error);
  }
};
```

---

## 🔌 API ENDPOINTS REFERENCE

### **VMware Credentials API (All Available):**
```bash
# List all credentials
GET /api/v1/vmware-credentials
Response: [{ id, name, vcenter_host, username, created_at, updated_at }]

# Create new credential  
POST /api/v1/vmware-credentials
Body: { name, vcenter_host, username, password, port? }

# Update credential
PUT /api/v1/vmware-credentials/{id}  
Body: { name, vcenter_host, username, password, port? }

# Delete credential
DELETE /api/v1/vmware-credentials/{id}

# Test connection (optional)
POST /api/v1/vmware-credentials/{id}/test
```

---

## 🎨 MAINTAIN EXACT VISUAL DESIGN

### **Keep All Existing Styling:**
- **Card layouts**: Maintain existing professional card design
- **Status badges**: Keep existing status styling (connected/error/etc.)
- **Buttons**: Use existing button variants and styling
- **Modal**: Keep existing modal styling and form layouts
- **Loading states**: Add professional loading indicators

### **Add Loading States:**
```typescript
// Add loading state to the main content
{isLoading ? (
  <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-3">
    {[...Array(3)].map((_, i) => (
      <Card key={i} className="animate-pulse">
        <CardContent className="p-6">
          <div className="h-4 bg-muted rounded w-3/4 mb-2"></div>
          <div className="h-3 bg-muted rounded w-1/2"></div>
        </CardContent>
      </Card>
    ))}
  </div>
) : (
  // Existing sources grid
  <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-3">
    {sources.map((source) => (
      // Existing card content
    ))}
  </div>
)}
```

---

## ✅ SUCCESS CRITERIA

### **Functional Requirements:**
- [ ] **Settings → Sources** loads real vCenter credentials from API
- [ ] **Add vCenter** saves new credentials to database via API
- [ ] **Edit vCenter** updates existing credentials via API  
- [ ] **Delete vCenter** removes credentials from database via API
- [ ] **VMDiscoveryModal Integration** can now find and use real saved credentials
- [ ] **End-to-End Workflow** works: Add creds in Settings → Use in Protection Groups

### **Quality Requirements:**
- [ ] **No Visual Changes** - Exact same appearance and UX
- [ ] **No Mock Data** - All mock data removed from sources page
- [ ] **Error Handling** - Proper error states for API failures
- [ ] **Loading States** - Professional loading indicators
- [ ] **Production Build** - Continues to work (15/15 pages)

---

## 🎯 TESTING REQUIREMENTS

### **End-to-End Workflow Test:**
1. **Settings → Sources** → Add new vCenter credentials → Save
2. **Protection Groups** → + Add VMs → Should see saved credentials in dropdown
3. **VM Discovery** → Select credentials → Discovery should work with real vCenter

### **API Integration Test:**
- Verify credentials persist after page refresh
- Test edit and delete operations
- Ensure VMDiscoveryModal can access saved credentials

---

## 🚨 CRITICAL SUCCESS FACTOR

**The Goal:** Complete the vCenter credentials workflow so users can:
1. **Add vCenter credentials** in Settings → Sources (saved to database)
2. **Use those credentials** in Protection Groups → + Add VMs for VM discovery
3. **Have end-to-end working workflow** from credential management to VM discovery

**Expected Outcome:** Professional Settings → Sources page with real API integration that enables the complete VMware backup workflow.

---

**CURRENT STATUS:** VMware GUI integration is 95% complete - just need this final API integration!
**FILES TO MODIFY:** Only `app/settings/sources/page.tsx` (replace mock data with API calls)
**MAINTAIN:** All existing visual design and UX patterns
**SUCCESS:** Complete vCenter credentials workflow enabling real VM discovery
