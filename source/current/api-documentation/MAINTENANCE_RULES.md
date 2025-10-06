# API Documentation Maintenance Rules

**Document Version:** 1.0  
**Last Updated:** October 4, 2025  
**Status:** 🔴 **MANDATORY COMPLIANCE**

---

## 🚨 CRITICAL REQUIREMENTS

### **API Documentation MUST Be Current (NO EXCEPTIONS)**

**The Rule:**
- `/source/current/api-documentation/` MUST be updated with EVERY API change
- API changes WITHOUT documentation updates are FORBIDDEN
- Documentation updates MUST happen BEFORE merging API changes
- Outdated API documentation is considered a critical project violation

**Why This Matters:**
- Prevents integration failures between frontend and backend
- Enables accurate customer documentation
- Ensures MSP platform partners have correct information
- Maintains professional enterprise image
- Prevents costly debugging sessions due to API mismatches

---

## 📁 REQUIRED FILES (Must Be Current)

### **Core Documentation Files**

```
api-documentation/
├── API_REFERENCE.md         # 🔴 CRITICAL: Complete endpoint reference
├── DB_SCHEMA.md             # 🔴 CRITICAL: Current database schema 
├── ERROR_CODES.md           # All error codes and meanings
├── AUTHENTICATION.md        # Auth/RBAC documentation
├── CHANGELOG.md             # API version history
├── EXAMPLES/                # Request/response examples
│   ├── backup-operations.md # Backup flow examples
│   ├── restore-operations.md # Restore flow examples
│   ├── platform-management.md # Platform config examples
│   └── msp-operations.md    # MSP-specific examples
├── schemas/
│   ├── openapi.yaml         # Machine-readable API spec
│   ├── request-schemas.json # Request validation schemas
│   └── response-schemas.json # Response validation schemas
└── MAINTENANCE_RULES.md     # This file
```

### **Legacy Files (Transition)**
```
Current Legacy Files (Migrate to new structure):
├── OMA.md                   # → Merge into API_REFERENCE.md
├── VMA.md                   # → Merge into API_REFERENCE.md  
├── API_DB_MAPPING.md        # → Merge into DB_SCHEMA.md
└── README.md                # → Update with new structure
```

---

## ✅ UPDATE REQUIREMENTS

### **When Documentation MUST Be Updated**

**1. API Endpoint Changes (CRITICAL)**
- New endpoint added → Document in API_REFERENCE.md + openapi.yaml
- Endpoint modified → Update documentation + examples
- Endpoint deprecated → Mark deprecated with timeline
- Endpoint removed → Move to deprecated section with date

**2. Database Schema Changes (CRITICAL)** 
- New table → Update DB_SCHEMA.md with full table definition
- Modified table → Update schema + migration reference
- New field → Document field purpose and constraints
- Field renamed/removed → Update schema + migration notes

**3. Request/Response Changes (MANDATORY)**
- New request fields → Update schemas + examples
- Changed response format → Update schemas + examples  
- New error codes → Update ERROR_CODES.md
- Authentication changes → Update AUTHENTICATION.md

**4. Feature Additions (REQUIRED)**
- New platform support → Add platform-specific examples
- New operation types → Document workflow and API calls
- New configuration options → Update examples and schemas

### **Documentation Quality Standards**

**Endpoint Documentation Format:**
```markdown
## POST /api/v1/backup/start

**Description:** Start backup operation for VM to specified repository

**Authentication:** Required (Bearer token)
**RBAC:** Requires `backup:create` permission
**Rate Limit:** 10 requests/minute per user

**Path Parameters:** None

**Request Body:**
```json
{
  "vm_id": "string (required) - VM identifier",
  "vm_name": "string (required) - VM display name", 
  "platform": "string (required) - Source platform: vmware|cloudstack|hyperv|aws|azure|nutanix",
  "repository_id": "string (required) - Target repository identifier",
  "backup_type": "string (required) - full|incremental|differential",
  "consistency": "string (optional) - application|crash (default: application)",
  "priority": "string (optional) - critical|normal|low (default: normal)",
  "change_id": "string (optional) - Previous change ID for incremental"
}
```

**Response (200 Success):**
```json
{
  "backup_job_id": "backup-db-prod-20251004120000",
  "status": "pending",
  "created_at": "2025-10-04T12:00:00Z",
  "estimated_duration_minutes": 8,
  "estimated_size_gb": 12.3,
  "nbd_endpoint": "localhost:10809/backup-export-abc123",
  "flow_type": "descend"
}
```

**Response (400 Bad Request):**
```json
{
  "error": "validation_failed",
  "error_code": "SEND-400-001", 
  "message": "Invalid backup request parameters",
  "details": {
    "field_errors": {
      "platform": "Unsupported platform: xenserver",
      "backup_type": "incremental requires change_id parameter"
    },
    "supported_platforms": ["vmware", "cloudstack", "hyperv", "aws", "azure", "nutanix"]
  }
}
```

**Error Codes:**
- `SEND-400-001`: Invalid request parameters
- `SEND-404-002`: VM not found
- `SEND-409-003`: Backup already in progress for VM
- `SEND-429-004`: Rate limit exceeded
- `SEND-500-005`: Internal server error

**Example cURL:**
```bash
curl -X POST https://api.sendense.com/v1/backup/start \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIs..." \
  -H "Content-Type: application/json" \
  -d '{
    "vm_id": "4205784a-098a-40f1-1f1e-a5cd2597fd59",
    "vm_name": "database-prod-01",
    "platform": "vmware", 
    "repository_id": "local-ssd-primary",
    "backup_type": "incremental",
    "change_id": "52 3c ec 11 9e 2c 4c 3d-87 4a c3 4e 85 f2 ea 95/446"
  }'
```

**Related Endpoints:**
- `GET /api/v1/backup/{job_id}` - Get backup status
- `DELETE /api/v1/backup/{job_id}` - Cancel backup  
- `GET /api/v1/backup/history` - Backup job history
```

---

## 🔍 DATABASE SCHEMA MAINTENANCE

### **DB_SCHEMA.md Requirements**

**Must Include:**
```markdown
# Database Schema - Sendense Platform

**Last Updated:** 2025-10-04  
**Schema Version:** 3.1.0  
**Migration Status:** All migrations applied  

## Table Definitions

### vm_replication_contexts (Master Table)
**Purpose:** Central VM context and lifecycle management
**Primary Key:** context_id (VARCHAR(64))
**Foreign Keys:** 
- current_job_id → replication_jobs(id) ON DELETE SET NULL
- last_successful_job_id → replication_jobs(id) ON DELETE SET NULL

**Fields:**
| Field Name | Type | Constraints | Purpose |
|------------|------|-------------|---------|
| context_id | VARCHAR(64) | PRIMARY KEY | Unique VM context identifier |
| vm_name | VARCHAR(255) | NOT NULL | VM display name |
| vmware_vm_id | VARCHAR(191) | UNIQUE | VMware VM identifier |
| current_status | ENUM(...) | NOT NULL | VM lifecycle status |
| created_at | TIMESTAMP | DEFAULT CURRENT_TIMESTAMP | Record creation |

**Indexes:**
- PRIMARY KEY (context_id)
- UNIQUE KEY unique_vm_vcenter (vmware_vm_id, vcenter_host)
- INDEX idx_status (current_status)
- INDEX idx_created (created_at)

**CASCADE DELETE Relationships:**
- replication_jobs → vm_context_id (CASCADE DELETE)
- vm_disks → vm_context_id (CASCADE DELETE)
- backup_jobs → vm_context_id (CASCADE DELETE)
- [... list all dependent tables]

### backup_jobs (NEW - Sendense Extension)
**Purpose:** Track backup operations (descend flows)
[Complete table definition...]

### restore_jobs (NEW - Sendense Extension) 
**Purpose:** Track restore operations (ascend flows)
[Complete table definition...]

## Migration History
- 20250115120000_initial_schema.up.sql - Base MigrateKit schema
- 20251004000001_add_backup_tables.up.sql - Sendense backup extensions
- [... all migrations listed with descriptions]

## Schema Validation
Last validated: 2025-10-04 12:00:00 UTC
Validation status: All constraints valid, no orphaned records
```

**Update Triggers (MANDATORY):**
- New migration file created → Update DB_SCHEMA.md immediately
- Table modified → Document changes with reasoning
- Constraint added/removed → Update constraint documentation
- Index added → Document index purpose and queries it optimizes

---

## 🌐 API VERSIONING STRATEGY

### **Version Management**

**API Version Format:** `/api/v{major}.{minor}/endpoint`
- **v1.0:** Current MigrateKit OSSEA APIs
- **v1.1:** Sendense backup extensions  
- **v2.0:** Major breaking changes (cross-platform support)
- **v3.0:** MSP platform APIs

**Backward Compatibility:**
- Maintain v1.x endpoints during transition
- Deprecation notices 90 days before removal
- Migration guides for major version changes
- Versioned documentation for each API version

### **Breaking Change Management**

**Process for Breaking Changes:**
1. **90-day notice** in API response headers
2. **New version** introduced alongside old
3. **Migration guide** published
4. **Client library updates** provided
5. **Customer communication** plan executed
6. **Old version removal** after transition period

---

## 🔄 DOCUMENTATION WORKFLOW

### **Development Workflow Integration**

**Branch Protection:**
```yaml
# Required checks before merge
required_status_checks:
  - api-docs-updated      # Automated check for API doc changes
  - schema-docs-current   # Validate DB schema documentation
  - examples-tested       # API examples work
  - openapi-valid         # OpenAPI spec validates
```

**Pre-Commit Hooks:**
```bash
#!/bin/bash
# Check if API files changed without doc updates

if git diff --cached --name-only | grep -E "(api|handlers|routes)" > /dev/null; then
    if ! git diff --cached --name-only | grep "api-documentation" > /dev/null; then
        echo "ERROR: API changes detected without documentation updates"
        echo "Please update /source/current/api-documentation/ before committing"
        exit 1
    fi
fi
```

**Automated Validation:**
```bash
# Daily automated checks
check-api-docs-currency:
  - Compare API routes in code vs documented endpoints
  - Validate all endpoints have documentation
  - Check all error codes documented
  - Verify examples work against actual API
  - Report discrepancies for immediate fixing
```

---

## 🎯 COMPLIANCE MONITORING

### **Documentation Quality Metrics**

**Tracked Metrics:**
- **Coverage:** % of endpoints documented (Target: 100%)
- **Currency:** Days since last API change without doc update (Target: 0)
- **Accuracy:** % of documented examples that work (Target: 100%)
- **Completeness:** % of endpoints with error codes documented (Target: 100%)

**Quality Gates:**
- All API changes require documentation review
- No merge without documentation update
- Quarterly documentation audit
- Customer-facing documentation review

### **Automated Monitoring**

**Documentation Drift Detection:**
```bash
# Weekly automated report
documentation-health-check:
  checks:
    - undocumented-endpoints: scan for new routes without docs
    - outdated-examples: test all cURL examples 
    - missing-error-codes: scan for new error patterns
    - schema-drift: compare database vs documentation
    - broken-links: verify all internal links work
    
  report:
    format: html + json
    recipients: [engineering-lead, documentation-team]
    escalation: critical-drift-threshold-exceeded
```

---

## 📚 DOCUMENTATION TEMPLATES

### **New Endpoint Template**

```markdown
## {METHOD} {PATH}

**Description:** [One sentence description of what this endpoint does]

**Authentication:** [Required/Optional] ([Type of auth])
**RBAC:** [Required permissions]
**Rate Limit:** [Limit per user/IP]

**Path Parameters:**
- `param_name` (type, required/optional) - Description

**Query Parameters:**
- `param_name` (type, required/optional) - Description

**Request Body:** [Schema with validation rules]

**Response:** [Success response schema]

**Error Responses:** [All possible error responses]

**Error Codes:** [All error codes this endpoint can return]

**Example:** [Working cURL example]

**Related Endpoints:** [Links to related operations]

**Changelog:**
- v1.1.0: Added support for new parameter
- v1.0.0: Initial implementation
```

### **Database Table Template**

```markdown
### {table_name}

**Purpose:** [What this table stores and why]
**Primary Key:** [Field name and type]
**Created:** [Date when table was added]
**Last Modified:** [Date of last schema change]

**Fields:**
| Field | Type | Constraints | Purpose | Added |
|-------|------|-------------|---------|-------|
| id | VARCHAR(64) | PRIMARY KEY | Unique identifier | v1.0.0 |
| [etc...] | ... | ... | ... | ... |

**Foreign Keys:**
- field_name → target_table(target_field) [CASCADE/SET NULL] - [Purpose]

**Indexes:**
- index_name (field_list) - [Query optimization purpose]

**Constraints:**
- constraint_name: [Description and purpose]

**Related Tables:**
- [Tables this depends on]
- [Tables that depend on this]

**Migration Files:**
- 20251004120000_create_{table_name}.up.sql
- [... other related migrations]
```

---

## 🎯 ENFORCEMENT MECHANISMS

### **Automated Checks**

**Pre-Merge Validation:**
```bash
# Required checks (automated)
validate-api-documentation:
  - check-endpoint-coverage: every route handler documented
  - validate-examples: all cURL examples return expected results
  - check-error-codes: all error returns documented
  - validate-schema: OpenAPI spec matches actual endpoints
  - test-authentication: all auth requirements documented correctly
```

**Post-Merge Validation:**
```bash
# Daily validation (automated)
documentation-currency-check:
  - compare-routes: actual routes vs documented routes
  - test-examples: run all documented examples
  - validate-responses: actual responses match documented schemas
  - check-deprecations: verify deprecated endpoints still work
```

### **Manual Review Process**

**Documentation Review Checklist:**
- [ ] All new endpoints documented with examples
- [ ] All modified endpoints updated
- [ ] Error codes complete and tested
- [ ] Examples work against actual API
- [ ] Breaking changes clearly marked
- [ ] Migration guides provided (if needed)
- [ ] Related endpoints cross-referenced
- [ ] OpenAPI spec updated and validates

---

## 🚀 SENDENSE-SPECIFIC REQUIREMENTS

### **Multi-Platform Documentation**

**Platform-Specific Examples:**
```markdown
# Backup Examples by Platform

## VMware Backup
```bash
# Full VMware VM backup
curl -X POST /api/v1/backup/start \
  -d '{
    "vm_id": "vm-123",
    "platform": "vmware",
    "backup_type": "full"
  }'
```

## CloudStack Backup  
```bash
# Incremental CloudStack VM backup
curl -X POST /api/v1/backup/start \
  -d '{
    "vm_id": "cs-456", 
    "platform": "cloudstack",
    "backup_type": "incremental"
  }'
```
```

### **Operation Type Documentation**

**Flow Type Examples:**
- **descend:** VM → Repository (backup operations)
- **ascend:** Repository → VM (restore operations)  
- **transcend:** Platform A → Platform B (replication operations)

```markdown
## Flow Operations

### descend (Backup Flows)
All backup operations that move data from VM to repository.
Endpoints: /api/v1/backup/*, /api/v1/descend/*

### ascend (Restore Flows)  
All restore operations that move data from repository to VM.
Endpoints: /api/v1/restore/*, /api/v1/ascend/*

### transcend (Replication Flows)
All replication operations that move data between platforms.
Endpoints: /api/v1/replication/*, /api/v1/transcend/*
```

---

## 📊 METRICS AND REPORTING

### **Documentation Health Metrics**

**Weekly Report:**
```
API Documentation Health Report - Week of Oct 4, 2025

📊 COVERAGE METRICS:
✅ Documented Endpoints: 127/127 (100%)
✅ Working Examples: 124/127 (97.6%) 
⚠️ Missing Error Codes: 3/127 (2.4%)
✅ Schema Currency: 0 days drift

🔍 ISSUES FOUND:
- 3 endpoints missing error code documentation
- 2 examples need updating for new response format
- 1 deprecated endpoint still receiving traffic

📋 ACTION ITEMS:
- Update error codes for backup/restore endpoints (Priority: Medium)
- Refresh examples for platform detection endpoints (Priority: Low) 
- Plan deprecation timeline for legacy replication endpoint (Priority: High)

📈 TRENDS:
- Documentation coverage improved 2.4% this week
- Example accuracy maintained at >95%
- Zero critical documentation issues
```

### **Customer Impact Tracking**

**Documentation-Related Issues:**
- Track customer support tickets caused by unclear documentation
- Monitor API integration failures due to doc inaccuracy
- Measure developer onboarding time with current docs
- Survey customer satisfaction with API documentation

---

## 🔗 INTEGRATION WITH DEVELOPMENT PROCESS

### **Code Review Requirements**

**API Change Reviews Must Include:**
- [ ] API documentation updated
- [ ] Examples tested and working
- [ ] Error codes documented
- [ ] OpenAPI spec updated
- [ ] Breaking changes identified
- [ ] Migration guide provided (if needed)

**Database Change Reviews Must Include:**
- [ ] DB_SCHEMA.md updated
- [ ] Migration documented
- [ ] Foreign key relationships documented
- [ ] Index purposes explained
- [ ] Performance impact assessed

---

**COMPLIANCE WITH THESE RULES IS MANDATORY**

**VIOLATIONS WILL RESULT IN IMMEDIATE CODE REVIEW REJECTION**

**KEEPING API DOCUMENTATION CURRENT IS NOT OPTIONAL - IT'S CRITICAL FOR PROJECT SUCCESS**

---

**Document Owner:** API Documentation Team  
**Enforcement:** Mandatory for all API changes  
**Review Cycle:** Weekly validation, quarterly audit  
**Last Updated:** October 4, 2025  
**Status:** 🔴 **MANDATORY COMPLIANCE**

