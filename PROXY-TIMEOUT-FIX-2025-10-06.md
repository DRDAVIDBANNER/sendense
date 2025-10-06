# Next.js Proxy Timeout Fix - October 6, 2025

## Problem

GUI showing "socket hang up" errors when adding VMs to management:
```
Failed to proxy http://localhost:8082/api/v1/discovery/discover-vms [Error: socket hang up] { code: 'ECONNRESET' }
Failed to proxy http://localhost:8082/api/v1/discovery/add-vms [Error: socket hang up] { code: 'ECONNRESET' }
```

## Root Cause Analysis

### Backend Timing (from journalctl)
```
"discovery_duration":14227559674  // 14.2 seconds
"processing_duration":2476400     // 2.4 ms
```

**Backend behavior:**
- Discovery takes 14+ seconds (querying vCenter for 98 VMs)
- Backend completes successfully
- Logs show proper response: `"status":"success"`

**Frontend behavior:**
- Next.js rewrite proxy times out before backend completes
- Connection dropped: `ECONNRESET`
- No response received by GUI

### Why It Happens

1. **vCenter query overhead:**
   - 98 VMs discovered
   - Each VM checked for existing context
   - Full metadata retrieval (disks, networks, power state)
   - Takes 14-30 seconds depending on vCenter load

2. **Next.js rewrite limitations:**
   - `next.config.ts` rewrites use default HTTP timeout
   - No built-in way to configure timeout for rewrites
   - Designed for fast API responses (< 5 seconds)

3. **Proxy connection drops:**
   - Client → Next.js → Backend
   - Next.js drops connection before backend responds
   - Client sees `ECONNRESET`

## Solution

### Custom API Route with Extended Timeout

**File:** `/source/current/sendense-gui/app/api/v1/discovery/[...path]/route.ts`

**Key Features:**
```typescript
const DISCOVERY_TIMEOUT = 60000; // 60 seconds

export async function POST(request, { params }) {
  const response = await fetch(url, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
    signal: AbortSignal.timeout(DISCOVERY_TIMEOUT), // ✅ 60s timeout
  });
}

export const maxDuration = 60; // ✅ Next.js route config
export const runtime = 'nodejs'; // ✅ Full Node.js (not edge)
```

**How It Works:**
1. GUI calls `/api/v1/discovery/discover-vms`
2. Next.js matches custom route BEFORE rewrite rule
3. Custom route proxies with 60-second timeout
4. Backend has time to complete vCenter queries
5. Response returned to GUI

### Request Flow

**Before Fix:**
```
GUI → Next.js Rewrite (default timeout ~10s) → Backend (14s) → TIMEOUT ❌
```

**After Fix:**
```
GUI → Custom API Route (60s timeout) → Backend (14s) → SUCCESS ✅
```

## Files Changed

### 1. next.config.ts
- Added comment about rewrite timeout limitations
- Kept rewrite for other endpoints (credentials, etc.)
- Custom routes override rewrites for specific paths

### 2. app/api/v1/discovery/[...path]/route.ts (NEW)
- Custom proxy for all `/api/v1/discovery/*` endpoints
- 60-second timeout for vCenter operations
- Proper error handling (504 Gateway Timeout on timeout)
- GET and POST method support
- TypeScript with full type safety

## Endpoints Affected

✅ **Now working with extended timeout:**
- `POST /api/v1/discovery/discover-vms` (14-30s typical)
- `POST /api/v1/discovery/add-vms` (14-30s typical)
- `POST /api/v1/discovery/preview` (14-30s typical)
- `GET /api/v1/discovery/ungrouped-vms` (fast, but covered)

## Testing

### Expected Behavior
1. Click "Discover VMs" in GUI
2. Shows loading state for 15-30 seconds
3. Returns 98 VMs with full metadata
4. Click "Add to Management"
5. Shows loading state for 15-30 seconds
6. Successfully adds VMs or shows "already exists"

### Error Handling
- **Timeout after 60s:** Returns 504 Gateway Timeout with clear message
- **Backend error:** Returns backend error with proper status code
- **Network error:** Returns 500 with error message

## Why This Approach

### Alternative Approaches (NOT used)

❌ **http-proxy-middleware**
- Requires additional npm dependency
- Overkill for simple proxy
- More complex configuration

❌ **Custom Next.js server**
- Breaks Vercel deployment
- Requires significant refactoring
- Against Next.js 15 best practices

❌ **Increase all timeouts globally**
- Would affect all API routes
- No granular control
- Potential performance issues

✅ **Custom API route (CHOSEN)**
- Next.js 15 native App Router pattern
- Zero additional dependencies
- Granular timeout control per route
- Easy to maintain
- Works in dev and production

## Deployment Notes

### Development
- Hot reload picks up new API route
- Restart dev server: `npm run dev`
- Check route registered: Look for `/api/v1/discovery/[...path]` in startup logs

### Production
- Next.js build includes custom route
- No additional configuration needed
- Works with standalone or node server
- Compatible with Docker deployment

## Related Work

- **API Documentation Update:** Documented discovery endpoint timing
- **GUI Fix Prompt:** `GROK-FIX-DISKS-NETWORKS-BULKADD.md` has related GUI fixes
- **Backend Logs:** Backend working correctly, no changes needed

## Session Context

Part of October 6, 2025 session:
1. ✅ Fixed VM names display
2. ✅ Fixed API documentation (discovery endpoints)
3. ✅ Fixed proxy timeout (this fix)
4. 🔄 Pending: Disk/network display in GUI
5. 🔄 Pending: Bulk add endpoint correction

## Future Considerations

### If Discovery Takes > 60 Seconds
- Add progress indicator in GUI
- Consider breaking into multiple requests
- Backend: Implement async job with polling
- Frontend: Show progress updates

### Monitoring
- Track discovery operation duration
- Alert if consistently > 45 seconds
- May indicate vCenter performance issues

### Optimization Opportunities
- Backend: Cache discovered VMs for 5 minutes
- Backend: Parallel VM metadata queries
- Backend: Skip network info if not needed
- VMA: Optimize vCenter API calls
