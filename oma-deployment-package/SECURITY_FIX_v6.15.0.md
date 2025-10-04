# 🔐 Security Fix: Hardcoded Credentials Removal

**Version:** v6.15.0-security-cleanup  
**Date:** October 3, 2025  
**Severity:** HIGH  
**Status:** RESOLVED ✅

---

## Issue Summary

Production GUI deployment package contained hardcoded vCenter credentials that were visible to end users, even in private browsing sessions. These credentials were development/testing defaults that inadvertently made it into the production codebase.

### Exposed Credentials
- **vCenter Host:** `quad-vcenter-01.quadris.local`
- **Username:** `administrator@vsphere.local`
- **Password:** `EmyGVoBFesGQc47-`
- **Datacenter:** `DatabanxDC`

---

## Impact

**Risk Level:** HIGH
- Credentials were visible in browser developer tools
- No browser cache involved - hardcoded in JavaScript
- Affected all fresh OMA deployments
- Could allow unauthorized access to internal vCenter infrastructure

**Affected Versions:**
- All GUI versions prior to v6.15.0

---

## Resolution

### Files Modified (11 total)

#### 1. Primary Form Defaults
**File:** `gui/src/components/discovery/DiscoveryView.tsx`
```typescript
// BEFORE (INSECURE)
const [vcenterHost, setVcenterHost] = useState('quad-vcenter-01.quadris.local');
const [username, setUsername] = useState('administrator@vsphere.local');
const [password, setPassword] = useState('EmyGVoBFesGQc47-');
const [datacenter, setDatacenter] = useState('DatabanxDC');

// AFTER (SECURE)
const [vcenterHost, setVcenterHost] = useState('');
const [username, setUsername] = useState('');
const [password, setPassword] = useState('');
const [datacenter, setDatacenter] = useState('');
```

#### 2. Replication Workflow
**File:** `gui/src/components/layout/RightContextPanel.tsx`
- Removed hardcoded credentials from discovery API calls (2 occurrences)
- Changed all fallback values from hardcoded credentials to empty strings

#### 3. API Route Defaults
**File:** `gui/src/app/api/replicate/route.ts`
```typescript
// BEFORE
vcenter_host: body.vcenter_host || "quad-vcenter-01.quadris.local"
datacenter: body.datacenter || "DatabanxDC"

// AFTER
vcenter_host: body.vcenter_host || ""
datacenter: body.datacenter || ""
```

#### 4-5. Test/Mock Data
**Files:**
- `gui/src/app/failover/page.tsx`
- `gui/src/app/network-mapping/page.tsx` (2 occurrences)

Removed all hardcoded credentials from test discovery calls.

#### 6-9. Network API Routes
**Files:**
- `gui/src/app/api/networks/topology/route.ts`
- `gui/src/app/api/networks/bulk-mapping-preview/route.ts`
- `gui/src/app/api/networks/bulk-mapping/route.ts`
- `gui/src/app/api/networks/recommendations/route.ts`

All network-related API routes cleaned of hardcoded credentials.

---

## Verification

### Automated Checks
```bash
# Scan for hardcoded credentials
grep -r "quad-vcenter\|EmyGVoBFesGQc47\|DatabanxDC" gui/src/
# Result: 0 matches found ✅

# Rebuild GUI
cd gui && npm run build
# Result: Success ✅

# Deploy to test OMA
systemctl restart migration-gui
# Result: Service active ✅
```

### Manual Testing
1. ✅ Discovery page loads with empty fields
2. ✅ No credentials visible in browser DevTools
3. ✅ Users must manually enter credentials
4. ✅ All form validations still work correctly

---

## Deployment Instructions

### For Fresh Deployments
The updated deployment script (v6.15.0) automatically:
1. Deploys cleaned GUI source code
2. Runs `npm run build` with secure defaults
3. Starts GUI service with no exposed credentials

```bash
./scripts/deploy-real-production-oma.sh <TARGET_IP>
```

### For Existing OMA Instances (Patch)
To patch existing deployments:

```bash
# 1. Copy cleaned GUI source
rsync -av /home/pgrayson/oma-deployment-package/gui/src/ target:/opt/migratekit/gui/src/

# 2. Rebuild on target
ssh target 'cd /opt/migratekit/gui && npm run build'

# 3. Restart GUI service
ssh target 'systemctl restart migration-gui'

# 4. Verify
curl http://target:3001/discovery | grep -o '<title>[^<]*</title>'
```

---

## Best Practices Implemented

1. ✅ **No Default Credentials:** All form fields start empty
2. ✅ **No Fallback Credentials:** API routes use empty strings, not defaults
3. ✅ **No Test Data in Production:** Removed all hardcoded test credentials
4. ✅ **Context-Driven:** Credentials come from user input or stored context only

---

## Security Recommendations

### For Operators
1. **Change Exposed Password:** If `EmyGVoBFesGQc47-` was a real password, rotate it immediately
2. **Audit Access:** Check vCenter access logs for unauthorized access from OMA IPs
3. **Update All Deployments:** Apply this patch to all production OMA instances

### For Development
1. **Never commit credentials:** Use `.env` files (gitignored) for local testing
2. **Code review focus:** Always check for hardcoded credentials in PRs
3. **Automated scanning:** Add pre-commit hooks to detect credentials

---

## Timeline

- **Discovered:** October 3, 2025 (by user observation)
- **Analysis:** Identified 11 files with hardcoded credentials
- **Fix Applied:** October 3, 2025 (same day)
- **Verified:** Test deployment successful with empty forms
- **Version Bump:** v6.14.0 → v6.15.0-security-cleanup

---

## Related Changes

This security fix is bundled with:
- GUI auto-build automation (v6.14.0)
- Database migration system
- Deployment script improvements

---

## Contact

For questions or concerns about this security fix:
- **Team:** MigrateKit OSSEA Team
- **Documentation:** See `/home/pgrayson/oma-deployment-package/SECURITY_FIX_v6.15.0.md`

---

**Status:** ✅ RESOLVED - All hardcoded credentials removed from codebase.


