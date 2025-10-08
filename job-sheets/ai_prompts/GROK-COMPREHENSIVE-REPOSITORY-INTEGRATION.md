# GROK JOB SHEET: Repository Management GUI Integration

**Date:** October 6, 2025  
**Project:** Sendense - Repository Management API Integration  
**Goal:** Wire up Repositories GUI page to backend API (replace all mock data)  
**Reference:** `/sendense/source/current/api-documentation/BACKUP_REPOSITORY_GUI_INTEGRATION.md`

---

## 🎯 OVERVIEW

You need to integrate the Repository Management GUI (`app/repositories/page.tsx`) with the existing backend API. The GUI page and components are already built with mock data - your job is to **replace all mock implementations with real API calls**.

**Backend API Endpoints (Already Built):**
- `GET /api/v1/repositories` - List all repositories
- `POST /api/v1/repositories` - Create new repository
- `POST /api/v1/repositories/test` - Test repository configuration
- `PATCH /api/v1/repositories/{id}` - Update repository
- `DELETE /api/v1/repositories/{id}` - Delete repository  
- `POST /api/v1/repositories/refresh-storage` - Refresh storage info for all repos

---

## 📁 FILES TO MODIFY

### **1. Main Page**
- **File:** `/sendense/source/current/sendense-gui/app/repositories/page.tsx`
- **Current:** Uses `mockRepositories` array with fake data
- **Action:** Replace all mock API calls with real `fetch()` calls to backend

### **2. AddRepositoryModal**
- **File:** `/sendense/source/current/sendense-gui/components/features/repositories/AddRepositoryModal.tsx`
- **Current:** Simulates test connection with timeout
- **Action:** Call real `/api/v1/repositories/test` endpoint

### **3. RepositoryCard** (Minor Updates)
- **File:** `/sendense/source/current/sendense-gui/components/features/repositories/RepositoryCard.tsx`
- **Current:** Displays repository data correctly
- **Action:** No changes needed (component already handles data properly)

---

## 🔀 BACKEND API RESPONSE STRUCTURE

### **GET /api/v1/repositories - List Repositories**

**Backend Response:**
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
    }
  ]
}
```

**GUI Expected Format (Current):**
```typescript
interface Repository {
  id: string;
  name: string;
  type: 'local' | 's3' | 'nfs' | 'cifs' | 'azure';
  status: 'online' | 'offline' | 'warning';  // Derived from enabled
  capacity: {
    total: number;    // Convert from bytes to GB
    used: number;     // Convert from bytes to GB
    available: number;  // Convert from bytes to GB
    unit: 'GB'
  };
  description?: string;   // From config if present
  lastTested?: string;    // storage.last_check_at
  location?: string;      // Derived from config (path, host, etc.)
}
```

**CRITICAL FIELD MAPPINGS:**
- Backend `enabled` field → GUI `status` ('online' if enabled, 'offline' if not)
- Backend `storage.total_bytes` → GUI `capacity.total` (divide by 1,073,741,824 for GB)
- Backend `storage.used_bytes` → GUI `capacity.used` (divide by 1,073,741,824 for GB)
- Backend `storage.available_bytes` → GUI `capacity.available` (divide by 1,073,741,824 for GB)
- Backend `storage.last_check_at` → GUI `lastTested`
- Backend `storage.used_percentage` → Use for warning status (>85% = 'warning', <85% = 'online')
- Backend `config` object → Extract location string based on type

---

## 🔧 IMPLEMENTATION REQUIREMENTS

### **TASK 1: Update `page.tsx` - Load Repositories**

Replace the `loadRepositories` function:

```typescript
const loadRepositories = async () => {
  setIsLoading(true);
  try {
    const response = await fetch('/api/v1/repositories');
    const data = await response.json();
    
    if (!data.success) {
      throw new Error(data.error || 'Failed to load repositories');
    }
    
    // Transform backend data to GUI format
    const transformedRepos = data.repositories.map(transformRepository);
    setRepositories(transformedRepos);
  } catch (error) {
    console.error('Failed to load repositories:', error);
    // Show error toast or notification to user
  } finally {
    setIsLoading(false);
  }
};
```

**Add helper function `transformRepository`:**

```typescript
const transformRepository = (backendRepo: any): Repository => {
  // Convert bytes to GB
  const bytesToGB = (bytes: number) => Math.round(bytes / 1073741824);
  
  // Determine status based on enabled and usage percentage
  let status: 'online' | 'offline' | 'warning' = 'offline';
  if (backendRepo.enabled) {
    const usagePercent = backendRepo.storage?.used_percentage || 0;
    status = usagePercent > 85 ? 'warning' : 'online';
  }
  
  // Extract location from config based on type
  const getLocation = () => {
    const config = backendRepo.config || {};
    switch (backendRepo.type) {
      case 'local':
        return config.path || '';
      case 'nfs':
        return `${config.server || 'unknown'}:${config.export_path || ''}`;
      case 'cifs':
        return `\\\\${config.server || 'unknown'}\\${config.share_name || ''}`;
      case 's3':
        return `${config.bucket || 'unknown'} (${config.region || 'us-east-1'})`;
      case 'azure':
        return `${config.account_name || 'unknown'}/${config.container || ''}`;
      default:
        return 'Unknown';
    }
  };
  
  return {
    id: backendRepo.id,
    name: backendRepo.name,
    type: backendRepo.type,
    status,
    capacity: {
      total: bytesToGB(backendRepo.storage?.total_bytes || 0),
      used: bytesToGB(backendRepo.storage?.used_bytes || 0),
      available: bytesToGB(backendRepo.storage?.available_bytes || 0),
      unit: 'GB'
    },
    description: backendRepo.config?.description || undefined,
    lastTested: backendRepo.storage?.last_check_at || undefined,
    location: getLocation()
  };
};
```

---

### **TASK 2: Update `page.tsx` - Create Repository**

Replace the `handleCreateRepository` function:

```typescript
const handleCreateRepository = async (repositoryData: Omit<Repository, 'id' | 'status' | 'lastTested'>) => {
  try {
    // Build backend config object from repository data
    const buildConfig = () => {
      const config: any = {};
      
      // Extract location into proper config fields
      switch (repositoryData.type) {
        case 'local':
          config.path = repositoryData.location || '';
          break;
        case 'nfs':
          // Parse "server:/export/path" format
          const nfsParts = (repositoryData.location || '').split(':');
          config.server = nfsParts[0] || '';
          config.export_path = nfsParts[1] || '';
          break;
        case 'cifs':
          // Parse "\\server\share" format
          const cifsParts = (repositoryData.location || '').replace(/\\\\/g, '').split('\\');
          config.server = cifsParts[0] || '';
          config.share_name = cifsParts[1] || '';
          break;
        case 's3':
          // Parse "bucket (region)" format
          const s3Match = (repositoryData.location || '').match(/(.+?)\s*\((.+?)\)/);
          config.bucket = s3Match ? s3Match[1] : repositoryData.location;
          config.region = s3Match ? s3Match[2] : 'us-east-1';
          break;
        case 'azure':
          // Parse "account/container" format
          const azureParts = (repositoryData.location || '').split('/');
          config.account_name = azureParts[0] || '';
          config.container = azureParts[1] || '';
          break;
      }
      
      if (repositoryData.description) {
        config.description = repositoryData.description;
      }
      
      return config;
    };
    
    const requestBody = {
      name: repositoryData.name,
      type: repositoryData.type,
      enabled: true,
      is_immutable: false,
      config: buildConfig()
    };
    
    const response = await fetch('/api/v1/repositories', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(requestBody)
    });
    
    const data = await response.json();
    
    if (!data.success) {
      throw new Error(data.error || 'Failed to create repository');
    }
    
    // Reload repositories to get fresh data including the new one
    await loadRepositories();
  } catch (error) {
    console.error('Failed to create repository:', error);
    throw error; // Re-throw so modal can show error
  }
};
```

---

### **TASK 3: Update `page.tsx` - Delete Repository**

Replace the `handleDeleteRepository` function:

```typescript
const handleDeleteRepository = async (repository: Repository) => {
  if (!confirm(`Are you sure you want to delete repository "${repository.name}"?`)) {
    return;
  }

  try {
    const response = await fetch(`/api/v1/repositories/${repository.id}`, {
      method: 'DELETE'
    });

    const data = await response.json();

    if (!data.success) {
      // Backend returns specific error for repos with backups
      if (data.backup_count) {
        alert(`Cannot delete repository: ${data.backup_count} backups exist. Delete backups first.`);
      } else {
        throw new Error(data.error || 'Failed to delete repository');
      }
      return;
    }

    // Remove from local state
    setRepositories(prev => prev.filter(r => r.id !== repository.id));
  } catch (error) {
    console.error('Failed to delete repository:', error);
    alert('Failed to delete repository. See console for details.');
  }
};
```

---

### **TASK 4: Update `page.tsx` - Test Repository**

Replace the `handleTestRepository` function:

```typescript
const handleTestRepository = async (repository: Repository) => {
  try {
    // Backend test endpoint requires repository ID
    const response = await fetch(`/api/v1/repositories/${repository.id}/test`, {
      method: 'POST'
    });

    const data = await response.json();

    if (!data.success) {
      alert(`Connection test failed: ${data.error || 'Unknown error'}`);
      return;
    }

    // Update last tested timestamp on success
    setRepositories(prev => prev.map(r =>
      r.id === repository.id
        ? { ...r, lastTested: new Date().toISOString() }
        : r
    ));

    alert(`Connection test successful for "${repository.name}"`);
  } catch (error) {
    console.error('Failed to test repository:', error);
    alert('Connection test failed. See console for details.');
  }
};
```

---

### **TASK 5: Update `page.tsx` - Refresh Storage**

Update the `handleRefresh` function to call refresh endpoint:

```typescript
const handleRefresh = async () => {
  setIsLoading(true);
  try {
    // Call backend refresh endpoint to update storage info for all repos
    const response = await fetch('/api/v1/repositories/refresh-storage', {
      method: 'POST'
    });

    const data = await response.json();

    if (!data.success) {
      throw new Error(data.error || 'Failed to refresh storage');
    }

    console.log(`Refreshed storage for ${data.refreshed_count} repositories`);
    if (data.failed_count > 0) {
      console.warn(`${data.failed_count} repositories failed to refresh`);
    }

    // Reload repositories to get updated storage info
    await loadRepositories();
  } catch (error) {
    console.error('Failed to refresh storage:', error);
    alert('Failed to refresh storage. See console for details.');
  } finally {
    setIsLoading(false);
  }
};
```

---

### **TASK 6: Update `AddRepositoryModal.tsx` - Test Connection**

The modal needs to be updated to properly collect config data and test BEFORE saving.

**Update the modal's `handleTestConnection` function:**

```typescript
const handleTestConnection = async () => {
  if (!selectedType) return;

  setIsTesting(true);
  setTestResult(null);

  try {
    // Build config object based on selected type and form data
    const buildTestConfig = () => {
      const config: any = {};
      
      switch (selectedType.id) {
        case 'local':
          config.path = formData.path;
          break;
        case 'nfs':
          config.server = formData.host;
          config.export_path = formData.path;
          if (formData.mountOptions) {
            config.mount_options = formData.mountOptions.split(',').map(o => o.trim());
          }
          break;
        case 'cifs':
          config.server = formData.host;
          config.share_name = formData.share;
          config.username = formData.username;
          config.password = formData.password;
          if (formData.domain) {
            config.domain = formData.domain;
          }
          break;
        case 's3':
          config.bucket = formData.bucket;
          config.region = formData.region;
          config.access_key = formData.accessKey;
          config.secret_key = formData.secretKey;
          if (formData.endpoint) {
            config.endpoint = formData.endpoint;
          }
          break;
        case 'azure':
          config.account_name = formData.accountName;
          config.container = formData.container;
          config.account_key = formData.accountKey;
          break;
      }
      
      return config;
    };

    const requestBody = {
      type: selectedType.id,
      config: buildTestConfig()
    };

    const response = await fetch('/api/v1/repositories/test', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(requestBody)
    });

    const data = await response.json();

    if (data.success) {
      setTestResult('success');
      
      // Optionally update capacity estimate if backend returns it
      if (data.storage) {
        console.log('Storage info:', {
          total: Math.round(data.storage.total_bytes / 1073741824),
          available: Math.round(data.storage.available_bytes / 1073741824),
          writable: data.storage.writable
        });
      }
    } else {
      setTestResult('error');
      console.error('Test failed:', data.error || data.message);
    }
  } catch (error) {
    console.error('Test connection error:', error);
    setTestResult('error');
  } finally {
    setIsTesting(false);
  }
};
```

**Update the modal's `handleCreate` function:**

The current implementation in `AddRepositoryModal.tsx` has a mock capacity object. Update it to work with the transformed data from the parent component:

```typescript
const handleCreate = async () => {
  if (!selectedType || !formData.name) return;

  setIsCreating(true);
  try {
    // Build location string for parent component
    const getLocationString = () => {
      switch (selectedType.id) {
        case 'local': 
          return formData.path;
        case 's3': 
          return `${formData.bucket} (${formData.region})`;
        case 'nfs': 
          return `${formData.host}:${formData.path}`;
        case 'cifs': 
          return `\\\\${formData.host}\\${formData.share}`;
        case 'azure': 
          return `${formData.accountName}/${formData.container}`;
        default: 
          return '';
      }
    };

    // Pass repository data to parent's onCreate handler
    // Parent will transform this into backend format and make API call
    await onCreate({
      name: formData.name,
      type: selectedType.id,
      capacity: { total: 0, used: 0, available: 0, unit: 'GB' }, // Will be filled by backend
      description: formData.description,
      location: getLocationString()
    });

    handleClose();
  } catch (error) {
    console.error('Failed to create repository:', error);
    alert('Failed to create repository. See console for details.');
  } finally {
    setIsCreating(false);
  }
};
```

---

### **TASK 7: Handle Edit Flow (BONUS)**

The current GUI has `editingRepository` prop but doesn't fully implement updates. Implement PATCH support:

**Add `handleUpdateRepository` function in `page.tsx`:**

```typescript
const handleUpdateRepository = async (
  repository: Repository,
  updates: Partial<Omit<Repository, 'id'>>
) => {
  try {
    // Build backend update payload (only send changed fields)
    const payload: any = {};
    
    if (updates.name !== undefined) payload.name = updates.name;
    if (updates.description !== undefined) {
      payload.config = { ...payload.config, description: updates.description };
    }
    // Note: For now, only name and description are updatable
    // Config changes should require delete + recreate for safety
    
    const response = await fetch(`/api/v1/repositories/${repository.id}`, {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload)
    });
    
    const data = await response.json();
    
    if (!data.success) {
      throw new Error(data.error || 'Failed to update repository');
    }
    
    // Reload to get fresh data
    await loadRepositories();
  } catch (error) {
    console.error('Failed to update repository:', error);
    throw error;
  }
};
```

Then update the `handleEditRepository` function to support updates properly, or for safety, just show current config as read-only in edit mode.

---

## 🎨 UI/UX ENHANCEMENTS

### **1. Add Loading Skeletons**

While `isLoading` is true, show skeleton cards instead of empty state:

```typescript
{isLoading ? (
  <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
    {[1, 2, 3].map(i => (
      <Card key={i} className="animate-pulse">
        <CardHeader>
          <div className="h-6 bg-muted rounded w-3/4" />
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="h-4 bg-muted rounded w-1/2" />
          <div className="h-2 bg-muted rounded w-full" />
          <div className="h-4 bg-muted rounded w-2/3" />
        </CardContent>
      </Card>
    ))}
  </div>
) : repositories.length === 0 ? (
  // ... empty state
) : (
  // ... repository cards
)}
```

### **2. Add Error State**

Add error state to show when API calls fail:

```typescript
const [error, setError] = useState<string | null>(null);

// In loadRepositories catch block:
setError('Failed to load repositories. Please try again.');

// In JSX, show error banner if error exists
{error && (
  <div className="bg-red-500/10 border border-red-500/20 rounded-lg p-4 mb-4">
    <p className="text-red-400">{error}</p>
    <Button 
      variant="outline" 
      size="sm" 
      onClick={() => { setError(null); loadRepositories(); }}
      className="mt-2"
    >
      Retry
    </Button>
  </div>
)}
```

### **3. Add Success Toasts**

Use a toast notification system for success messages:
- "Repository created successfully"
- "Repository deleted successfully"
- "Connection test passed"
- "Storage refreshed successfully"

(Optional: Use shadcn/ui toast component if available)

---

## 📊 DATA VALIDATION

### **Critical Validations:**

1. **Capacity Calculations:**
   - Always divide bytes by `1,073,741,824` for GB conversion
   - Round to whole numbers for display: `Math.round(bytes / 1073741824)`
   - Handle `null` or `undefined` storage data gracefully

2. **Status Determination:**
   - `enabled: true` + `used_percentage < 85` = `'online'`
   - `enabled: true` + `used_percentage >= 85` = `'warning'`
   - `enabled: false` = `'offline'`

3. **Location Extraction:**
   - Each repository type has different config structure
   - Handle missing fields gracefully with fallbacks

4. **Error Handling:**
   - Backend returns `{ success: false, error: "message" }` on errors
   - Always check `data.success` before processing
   - Display user-friendly error messages

---

## 🧪 TESTING CHECKLIST

After implementation, test these scenarios:

### **Core Functionality:**
- [ ] Page loads and displays repositories from backend
- [ ] Summary cards show correct counts (Total, Online, Warning, Offline)
- [ ] Total capacity bar shows correct percentage
- [ ] Each repository card displays correct info (type badge, capacity, status)

### **Create Repository:**
- [ ] Can create Local repository
- [ ] Can create NFS repository (requires NFS server to test properly)
- [ ] Can create CIFS repository (requires file server to test properly)
- [ ] Can create S3 repository (requires AWS credentials)
- [ ] Can create Azure repository (requires Azure credentials)
- [ ] Test connection works and shows success/failure before creating
- [ ] Created repository appears in list immediately
- [ ] Capacity info is correct after creation

### **Delete Repository:**
- [ ] Can delete repository with no backups
- [ ] Cannot delete repository with existing backups (shows error message)
- [ ] Confirmation dialog appears before deletion
- [ ] Deleted repository disappears from list

### **Test Repository:**
- [ ] Test connection works for each repository type
- [ ] Shows loading state during test
- [ ] Shows success/failure message
- [ ] Updates "Last tested" timestamp on success

### **Refresh Storage:**
- [ ] Refresh button updates all repository storage info
- [ ] Shows loading state during refresh
- [ ] Updates capacity bars and percentages
- [ ] Handles errors gracefully if some repos fail to refresh

### **Error Handling:**
- [ ] Shows error if backend is unreachable
- [ ] Shows error if API returns error response
- [ ] Error messages are user-friendly
- [ ] Retry mechanism works

### **UI/UX:**
- [ ] Loading skeletons show while fetching data
- [ ] Status badges show correct colors (green/yellow/red)
- [ ] Capacity bars show correct colors based on usage
- [ ] Dropdown menus work (Edit, Delete, Test Connection)
- [ ] Modal closes properly after creation
- [ ] Page refreshes data after operations

---

## 🚨 CRITICAL REQUIREMENTS

### **MUST DO:**
1. ✅ **Replace ALL mock data** - No hardcoded repositories
2. ✅ **Use real API endpoints** - `/api/v1/repositories` prefix
3. ✅ **Transform backend data correctly** - Bytes to GB, status derivation
4. ✅ **Handle errors properly** - Show user-friendly messages
5. ✅ **Test before create** - Use `/api/v1/repositories/test` endpoint
6. ✅ **Validate responses** - Check `data.success` field
7. ✅ **Update UI state** - Reload data after create/delete operations

### **DO NOT:**
- ❌ Leave any mock data in place
- ❌ Hardcode capacity values
- ❌ Skip error handling
- ❌ Create repositories without testing connection first
- ❌ Assume field names match between backend and GUI
- ❌ Use placeholder/simulation code

---

## 📝 FINAL NOTES

**Backend API is already built and operational.** This is purely a frontend integration task - wire up the GUI to existing endpoints.

**Data transformation is critical.** The backend uses `bytes`, GUI uses `GB`. The backend uses `enabled`, GUI uses `status`. Make sure mappings are correct.

**Test thoroughly.** Each repository type has different config structures. Make sure Local, NFS, and CIFS work properly (S3/Azure may not be testable without credentials).

**Ask questions if stuck.** Better to clarify than to guess and break things.

---

## 🎯 EXPECTED OUTCOME

After completion:
1. ✅ Repositories page loads real data from backend
2. ✅ Can create new repositories (Local, NFS, CIFS)
3. ✅ Can test repository connections before creating
4. ✅ Can delete repositories (with safety check for backups)
5. ✅ Can test existing repository connections
6. ✅ Can refresh storage information
7. ✅ Summary cards show accurate statistics
8. ✅ Capacity bars show correct percentages
9. ✅ Status badges reflect actual repository health
10. ✅ All error cases handled gracefully

**No mock data should remain in the codebase.**

---

**END OF GROK JOB SHEET**

**When you complete this work, report back with:**
1. List of files modified
2. Any issues encountered
3. Any deviations from this spec (with justification)
4. Screenshots of working functionality

Good luck! 🚀

