API Documentation

Scope: OMA API and VMA API endpoints currently implemented in `sendense/source/current`. Each endpoint lists path, method, handler, description, known callsites, and classification:

- Key: actively used by core flows (replication, discovery, failover, progress, credentials)
- Auxiliary: used for health/debug/admin, not on critical path
- Potentially legacy: present but no in-repo callsites or superseded; may be GUI-only or slated for consolidation
- Legacy (avoid): duplicates older engine paths or unimplemented expectations; keep for backward compatibility only

Files:
- OMA endpoints: `./OMA.md`
- VMA endpoints: `./VMA.md`

Cross-links/References:
- OMA routes defined in `oma/api/server.go`; handlers under `oma/api/handlers/*`
- VMA routes defined in `vma/api/server.go`; progress routes in `vma/api/progress_handler.go`
- Known callsites include: `oma/services/*`, `oma/failover/*`, `oma/workflows/*`, `migratekit/internal/*`, `vma/client/*`, `vma/services/*`

Notes:
- Canonical source per project rules is under `sendense/source/current/`.
- Classifications are based on code references in this repo; externally-consumed GUI calls may not appear here but could still be active in production.

## ⚠️ Common API Gotchas & Best Practices

### Discovery Endpoints Confusion
**Problem:** Multiple discovery endpoints exist with similar names but different capabilities.

**Best Practice:**
- ✅ **Use `/discovery/add-vms`** for GUI bulk add operations
  - Supports `credential_id` for saved credentials
  - Accepts `vm_names` array
  - Returns detailed per-VM success/failure
- ❌ **Don't use `/discovery/bulk-add`** from GUI
  - Legacy endpoint requiring manual credentials
  - Does NOT support `credential_id`
  - Requires explicit vcenter/username/password in every request

**Why It Matters:**
- GUI sends `credential_id` which bulk-add doesn't support → 400 Bad Request
- add-vms properly handles saved credentials from vmware_credentials table
- Proper audit trail with `added_by` field

**Detailed Documentation:** See `OMA.md` Discovery section for complete request/response schemas

### Disk and Network Metadata
**API Response Includes:**
- `disks`: Array of `VMADiskInfo` with `size_gb`, `capacity_bytes`, `datastore`
- `networks`: Array of `VMANetworkInfo` with `network_name`, `mac_address`

**GUI Display Requirements:**
- Show disk count and total capacity: "💾 3 disks (500 GB total)"
- Show network count and names: "🌐 2 networks: VM Network, Production"
- Frontend interfaces must include `disks?: any[]` and `networks?: any[]` fields

### Database Schema Safety
**Always validate field names against:**
1. `DB_SCHEMA.md` for table definitions
2. `API_DB_MAPPING.md` for endpoint impacts
3. Backend handler structs for exact JSON field names

**Common Mismatches:**
- Backend: `credential_name` → Frontend might expect: `name`
- Backend: `num_cpu` → Frontend might expect: `cpu_count`
- Backend: `discovered_vms` → Frontend might expect: `vms`

### Credential Management
**Saved Credentials Workflow:**
1. Create credentials via `POST /vmware-credentials` (stores encrypted in database)
2. Use `credential_id` in discovery endpoints (auto-decrypts and loads)
3. Credential usage tracked: `last_used`, `usage_count` fields updated

**Manual Credentials Workflow:**
- Provide vcenter/username/password/datacenter directly
- No database storage, no reuse
- Use only for one-off operations or testing

