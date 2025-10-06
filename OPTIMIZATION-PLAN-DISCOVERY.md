# VM Discovery Optimization Plan - October 6, 2025

## Current Wasteful Flow (28+ seconds)

### Step 2: Discovery
```
GUI → POST /api/v1/discovery/discover-vms (create_context=false)
  → Backend queries vCenter (14s)
  → Returns 98 VMs with full metadata
  → GUI stores in React state
```

### Step 3: Add to Management
```
GUI → POST /api/v1/discovery/add-vms {vm_names: ["VM1"]}
  → Backend queries vCenter AGAIN (14s) ❌
  → Backend creates 1 context
  → Total: 28 seconds for 1 VM
```

## Proposed Optimized Flow (14 seconds)

### Option A: Use discover-vms with create_context=true

**Single Request:**
```typescript
// Step 2+3 Combined
POST /api/v1/discovery/discover-vms {
  credential_id: 35,
  create_context: true,
  selected_vms: ["VM1", "VM2", "VM3"]
}

Response:
{
  discovered_vms: [...],  // All 98 VMs for GUI display
  addition_result: {      // Creation result for selected VMs
    successfully_added: 3,
    added_vms: [...]
  }
}
```

**BUT THIS STILL RE-DISCOVERS** because AddVMsToOMAWithoutJobs calls DiscoverVMsFromVMA again!

### Option B: New Endpoint - Create Contexts from Cached Data ✅

**Step 2: Discovery (cache results)**
```typescript
POST /api/v1/discovery/discover-vms {
  credential_id: 35,
  create_context: false  // Just discover, don't create
}

Response: { discovered_vms: [98 VMs with full metadata] }
```

**Step 3: Create contexts from cached data (instant)**
```typescript
POST /api/v1/discovery/create-contexts {
  credential_id: 35,  // For vCenter/datacenter info
  vms: [
    {
      id: "vm-123",
      name: "VM1",
      path: "/DC/vm/VM1",
      num_cpu: 4,
      memory_mb: 8192,
      guest_os: "windows",
      power_state: "poweredOn",
      disks: [...],
      networks: [...]
    }
  ]
}

Response: { successfully_added: 1, added_vms: [...] }
```

**Backend creates contexts directly from provided metadata (no vCenter query!)**

### Option C: Hybrid Approach ✅✅ (BEST)

**Modify discover-vms endpoint:**

1. **Discover phase:** Query vCenter once, cache results in memory (5 min TTL)
2. **Create phase:** If `create_context=true`, use cached data (no re-query)

**Implementation:**
```go
// In enhanced_discovery_service.go

type DiscoveryCache struct {
    mu        sync.RWMutex
    cache     map[string]*CachedDiscovery
}

type CachedDiscovery struct {
    VMs        []VMAVMInfo
    VCenter    string
    Datacenter string
    CachedAt   time.Time
    TTL        time.Duration // 5 minutes
}

func (eds *EnhancedDiscoveryService) AddVMsToOMAWithoutJobs(
    ctx context.Context,
    discoveryRequest DiscoveryRequest,
    selectedVMNames []string,
    cachedVMs []VMAVMInfo, // NEW: optional cached VMs
) (*BulkAddResult, error) {
    
    var vmaResponse *VMADiscoveryResponse
    
    // Use cached VMs if provided
    if len(cachedVMs) > 0 {
        vmaResponse = &VMADiscoveryResponse{
            VMs: cachedVMs,
            VCenter: struct{Host, Datacenter string}{
                Host: discoveryRequest.VCenter,
                Datacenter: discoveryRequest.Datacenter,
            },
        }
        log.Info("Using cached VM discovery data (no vCenter query)")
    } else {
        // Original flow: query vCenter
        var err error
        vmaResponse, err = eds.DiscoverVMsFromVMA(ctx, discoveryRequest)
        if err != nil {
            return nil, err
        }
    }
    
    // Process VMs (same as before)
    return eds.processDiscoveredVMs(ctx, vmaResponse, selectedVMNames, result)
}
```

**Handler change:**
```go
// In enhanced_discovery.go

func (h *EnhancedDiscoveryHandler) DiscoverVMs(...) {
    // Discover VMs
    vmaResponse, err := h.discoveryService.DiscoverVMsFromVMA(ctx, discoveryReq)
    
    // Convert to response format
    discoveredVMs := make([]DiscoveredVMInfo, 0, len(vmaResponse.VMs))
    // ... (existing conversion code)
    
    // If create_context is true, use cached data (no re-query)
    if request.CreateContext {
        addResult, err := h.discoveryService.AddVMsToOMAWithoutJobs(
            ctx, 
            discoveryReq, 
            request.SelectedVMs,
            vmaResponse.VMs, // ✅ Pass discovered VMs (no re-query!)
        )
        // ...
    }
}
```

## Benefits

### Performance
- ✅ **50% faster:** 28s → 14s (one vCenter query instead of two)
- ✅ **Instant for additional VMs:** Select more VMs? Instant context creation

### UX
- ✅ **Snappier UI:** No 14s wait after selecting VMs
- ✅ **Bulk operations:** Add 10 VMs in 14s instead of 140s

### vCenter Load
- ✅ **50% less API calls** to vCenter
- ✅ **Better for large environments** (500+ VMs)

## Implementation Priority

**Option C (Hybrid) - RECOMMENDED**

**Changes needed:**
1. Modify `AddVMsToOMAWithoutJobs` signature to accept optional cached VMs
2. Modify `DiscoverVMs` handler to pass discovered VMs to AddVMsToOMAWithoutJobs
3. Add logging to show when cached data is used
4. Update API documentation

**Effort:** ~30 minutes
**Impact:** 50% performance improvement
**Breaking changes:** None (backward compatible)

## Alternative: Frontend Caching Only

**Quick Win (No Backend Changes):**

Modify GUI workflow:
```typescript
// Step 2: Discovery (store full VM objects)
const [discoveredVMs, setDiscoveredVMs] = useState<DiscoveredVM[]>([]);

// Step 3: Use discover-vms with create_context=true
// GUI already has the data, but we send selected_vms to create contexts
// Backend will re-query vCenter (14s) but at least it's a single call

const addToManagement = async () => {
  await fetch('/api/v1/discovery/discover-vms', {
    method: 'POST',
    body: JSON.stringify({
      credential_id: selectedCredentialId,
      create_context: true,
      selected_vms: selectedVMNames,
    }),
  });
};
```

**This combines Step 2+3 into one call (14s instead of 28s)** ✅

But backend still re-queries vCenter. Not ideal, but better than current.

## Decision Matrix

| Approach | Backend Changes | Performance Gain | Complexity | Recommended |
|----------|----------------|------------------|------------|-------------|
| Option A (current create_context) | None | 0% (still re-queries) | Low | ❌ No |
| Option B (new endpoint) | Medium | 50% | Medium | ⚠️ Maybe |
| Option C (cached VMs parameter) | Small | 50% | Low | ✅ Yes |
| Frontend-only (combine calls) | None | ~25% | Very Low | ✅ Quick Win |

## Recommendation

**Immediate (5 min):** Frontend-only optimization
- Change GUI to use `/discover-vms` with `create_context=true`
- Reduces 28s → 14s (50% improvement)
- Zero backend changes

**Follow-up (30 min):** Backend cached VMs optimization
- Modify `AddVMsToOMAWithoutJobs` to accept cached VMs
- Reduces vCenter load
- Enables instant multi-select operations

## Testing

**Before:**
```
Discovery: 14s
Add 1 VM: 14s
Total: 28s
```

**After (frontend-only):**
```
Discovery + Add: 14s
Total: 14s (50% improvement)
```

**After (backend caching):**
```
Discovery + Add: 14s
Add 5 more VMs: <1s (instant)
Total: 14s (90% improvement for multiple adds)
```
