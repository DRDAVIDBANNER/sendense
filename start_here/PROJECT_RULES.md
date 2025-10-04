# Sendense Project Rules - MANDATORY COMPLIANCE

**Document Version:** 1.0  
**Last Updated:** October 4, 2025  
**Status:** 🔴 **MANDATORY - NO EXCEPTIONS**

---

## 🚨 ABSOLUTE PROJECT RULES - NEVER VIOLATE

### **1. NO BULLSHIT "PRODUCTION READY" CLAIMS**
- ❌ **FORBIDDEN:** Claiming code is "production ready" without complete testing
- ❌ **FORBIDDEN:** "This should work" or "probably production ready"
- ❌ **FORBIDDEN:** Deploying untested code to any environment
- ✅ **REQUIRED:** Explicit testing checklist completion before any "ready" claim
- ✅ **REQUIRED:** Documentation of all testing performed
- ✅ **REQUIRED:** Performance benchmarks and security review

### **2. NO SIMULATIONS OR PLACEHOLDER CODE**
- ❌ **FORBIDDEN:** Simulation logic, fake data, placeholder implementations
- ❌ **FORBIDDEN:** "TODO" comments in committed code
- ❌ **FORBIDDEN:** Mock endpoints that don't connect to real backend
- ❌ **FORBIDDEN:** Demo code or quick fixes
- ✅ **REQUIRED:** All code must be functional and complete
- ✅ **REQUIRED:** Real data connections and live functionality
- ✅ **REQUIRED:** Remove all placeholder logic before commit

### **3. SOURCE CODE AUTHORITY AND ORGANIZATION**
- ✅ **CANONICAL SOURCE:** `source/current/` is the ONLY authoritative code location
- ❌ **FORBIDDEN:** Code scattered outside `source/current/`
- ❌ **FORBIDDEN:** Binaries committed in source trees
- ✅ **REQUIRED:** All binaries in `source/builds/` or `dist/`
- ✅ **REQUIRED:** Version control for all builds with explicit version numbers
- ✅ **REQUIRED:** Build scripts and deployment procedures documented

### **4. DOCUMENTATION MANDATORY MAINTENANCE**
- ✅ **CRITICAL:** `/source/current/api-documentation/` MUST be updated with ALL API changes
- ✅ **REQUIRED:** CHANGELOG.md updated with every significant change
- ✅ **REQUIRED:** README.md in every module with current status
- ✅ **REQUIRED:** Architecture docs updated when workflows change
- ❌ **FORBIDDEN:** Code changes without corresponding documentation updates

### **5. NO DEVIATIONS FROM APPROVED PLAN**
- ✅ **REQUIRED:** Follow `project-goals/` roadmap and phase plans
- ❌ **FORBIDDEN:** Adding features not in approved roadmap
- ❌ **FORBIDDEN:** Architectural changes without approval
- ✅ **REQUIRED:** Document any necessary plan modifications
- ✅ **REQUIRED:** Stakeholder approval for plan changes

---

## 🔧 DEVELOPMENT STANDARDS

### **Code Quality Requirements**

**Go Code Standards:**
```go
// ✅ REQUIRED: All functions must have error handling
func BackupVM(vmID string) error {
    if vmID == "" {
        return fmt.Errorf("vmID cannot be empty")
    }
    
    // Implementation...
    
    return nil
}

// ❌ FORBIDDEN: Functions without error handling
func BackupVM(vmID string) {
    // This will panic if vmID is invalid - NOT ACCEPTABLE
}

// ✅ REQUIRED: Meaningful variable names
var (
    vmReplicationContext *VMContext
    backupRepository     Repository
    licenseValidator     *LicenseValidator
)

// ❌ FORBIDDEN: Cryptic or lazy naming
var (
    ctx *Context  // What kind of context?
    repo Repository // Which repository? 
    val *Validator // Validates what?
)
```

**TypeScript Standards:**
```typescript
// ✅ REQUIRED: Strict TypeScript interfaces
interface VMBackupRequest {
    vmID: string;
    vmName: string;
    platformType: PlatformType;
    repositoryID: string;
    backupType: 'full' | 'incremental';
}

// ❌ FORBIDDEN: any types or loose interfaces
interface BackupRequest {
    data: any; // Unacceptable - defines nothing
}

// ✅ REQUIRED: Error boundaries and proper error handling
const handleBackupSubmit = async (request: VMBackupRequest) => {
    try {
        const result = await api.backup.start(request);
        return result;
    } catch (error) {
        logger.error('Backup start failed', { error, request });
        throw new Error(`Backup failed: ${error.message}`);
    }
};
```

### **Database Standards**

**Schema Management:**
- ✅ **REQUIRED:** All schema changes via migration files
- ✅ **REQUIRED:** Migration files in `source/current/control-plane/database/migrations/`
- ✅ **REQUIRED:** Both `.up.sql` and `.down.sql` for every migration
- ❌ **FORBIDDEN:** Direct database modifications
- ❌ **FORBIDDEN:** Schema assumptions - always validate field names

**API Documentation:**
- ✅ **CRITICAL:** `/source/current/api-documentation/` updated with EVERY API change
- ✅ **REQUIRED:** OpenAPI/Swagger spec for all endpoints
- ✅ **REQUIRED:** Request/response examples for every endpoint
- ✅ **REQUIRED:** Error code documentation

---

## 📁 PROJECT STRUCTURE REQUIREMENTS

### **Directory Structure (Mandatory)**

```
sendense/
├── source/current/              # ✅ ONLY authoritative code location
│   ├── control-plane/           # Central orchestration
│   ├── capture-agent/           # Platform-specific agents  
│   ├── api-documentation/       # 🔴 CRITICAL: Must be current
│   └── VERSION.txt             # Current version number
├── source/builds/              # ✅ ALL binaries go here
│   ├── control-plane-v1.2.3   # Versioned binaries
│   ├── vmware-agent-v2.1.1    # Platform agents
│   └── CHANGELOG.md            # Build history
├── project-goals/              # ✅ Project roadmap and plans
│   ├── phases/                 # Implementation phases
│   ├── modules/                # Technical modules
│   └── architecture/           # System design
├── docs/                       # ✅ User-facing documentation
│   ├── admin-guide/            # Administrator documentation
│   ├── api-reference/          # API documentation
│   └── troubleshooting/        # Support documentation
└── tests/                      # ✅ ALL tests (unit, integration, e2e)
    ├── unit/                   # Unit tests
    ├── integration/            # Integration tests
    └── e2e/                    # End-to-end tests
```

### **File Naming Standards**

**Binaries:**
```bash
# ✅ REQUIRED: Explicit version numbers
control-plane-v1.2.3
vmware-capture-agent-v2.1.1-production
cloudstack-agent-v1.0.5-beta

# ❌ FORBIDDEN: Ambiguous naming
control-plane-latest
agent-final
backup-tool
main
```

**Documentation:**
```bash
# ✅ REQUIRED: Clear, descriptive names
API_REFERENCE.md
DEPLOYMENT_GUIDE.md
TROUBLESHOOTING_VMWARE.md
PHASE_1_COMPLETION_REPORT.md

# ❌ FORBIDDEN: Generic or cryptic names
README.md (unless in module root)
doc.md
notes.txt
temp-info.md
```

---

## 📝 CHANGE MANAGEMENT REQUIREMENTS

### **CHANGELOG.md Standards**

**Format (Mandatory):**
```markdown
# Sendense Changelog

## [Unreleased]
### Added
- New feature descriptions with issue references
### Changed  
- Modified functionality with impact assessment
### Fixed
- Bug fixes with root cause analysis
### Security
- Security improvements and vulnerability fixes

## [2.1.0] - 2025-10-04
### Added
- VMware backup repository support (#SEND-123)
- QCOW2 incremental backup chains (#SEND-124)
- Automatic backup validation (#SEND-125)

### Changed
- Improved CBT change tracking accuracy (#SEND-126)
- Enhanced error handling in backup workflows (#SEND-127)

### Fixed
- Fixed memory leak in NBD server (#SEND-128)
- Resolved concurrent backup job conflicts (#SEND-129)

### Performance
- 15% improvement in backup throughput
- Reduced memory usage by 200MB per backup job

### Security
- Updated SSH tunnel key rotation policy
- Enhanced license validation security
```

### **Git Commit Standards**

```bash
# ✅ REQUIRED: Descriptive commit messages with scope
feat(vmware): add CBT change tracking for incremental backups
fix(api): resolve backup job status update race condition  
docs(api): update backup endpoints documentation
test(integration): add CloudStack backup integration tests

# ❌ FORBIDDEN: Lazy or unclear commits
fix stuff
update
WIP
quick fix
```

### **Version Control Rules**

- ✅ **REQUIRED:** Feature branches for all changes (`feature/vmware-backup`, `fix/api-race-condition`)
- ✅ **REQUIRED:** Pull request review for all changes
- ✅ **REQUIRED:** All tests pass before merge
- ❌ **FORBIDDEN:** Direct commits to main branch
- ❌ **FORBIDDEN:** Force pushing to shared branches

---

## 🧪 TESTING REQUIREMENTS

### **Testing Standards (No Exceptions)**

**Test Coverage Requirements:**
- ✅ **MINIMUM:** 80% code coverage for all new code
- ✅ **REQUIRED:** Unit tests for all business logic
- ✅ **REQUIRED:** Integration tests for all API endpoints
- ✅ **REQUIRED:** End-to-end tests for critical workflows
- ❌ **FORBIDDEN:** Committing code without corresponding tests

**Test Categories:**
```bash
# Unit Tests (Fast, isolated)
go test ./src/control-plane/backup/... -v
npm test -- --coverage

# Integration Tests (Real backends)  
go test ./tests/integration/backup_test.go
npm run test:integration

# End-to-End Tests (Full workflows)
go test ./tests/e2e/vmware_backup_e2e_test.go
npm run test:e2e

# Performance Tests (Benchmarks)
go test ./tests/performance/ -bench=.
npm run test:performance
```

**Production Readiness Checklist:**
```markdown
Before claiming "production ready":
- [ ] All unit tests pass (100%)
- [ ] All integration tests pass (100%)  
- [ ] End-to-end tests pass (100%)
- [ ] Performance benchmarks meet targets
- [ ] Security review completed
- [ ] Load testing completed
- [ ] Error handling tested (fault injection)
- [ ] Rollback procedures tested
- [ ] Documentation complete and reviewed
- [ ] Deployment procedures tested
- [ ] Monitoring and alerting configured
- [ ] Support runbook created
```

---

## 🔐 SECURITY REQUIREMENTS

### **Security Standards (Mandatory)**

**Authentication & Authorization:**
- ✅ **REQUIRED:** All API endpoints require authentication
- ✅ **REQUIRED:** Role-based access control (RBAC) implementation
- ✅ **REQUIRED:** Secure credential storage (encrypted, rotated)
- ❌ **FORBIDDEN:** Hardcoded credentials or secrets
- ❌ **FORBIDDEN:** Plain text passwords anywhere

**Data Protection:**
- ✅ **REQUIRED:** Encryption in transit (TLS 1.3 minimum)
- ✅ **REQUIRED:** Encryption at rest (AES-256)
- ✅ **REQUIRED:** Customer data isolation (multi-tenant)
- ❌ **FORBIDDEN:** Plain text customer data
- ❌ **FORBIDDEN:** Shared encryption keys across customers

**Vulnerability Management:**
- ✅ **REQUIRED:** Dependency scanning (weekly)
- ✅ **REQUIRED:** Static code analysis (SonarQube or equivalent)
- ✅ **REQUIRED:** Security testing in CI/CD pipeline
- ✅ **REQUIRED:** Penetration testing before major releases

---

## 📊 PERFORMANCE REQUIREMENTS

### **Performance Standards (Non-Negotiable)**

**Backup Performance:**
- ✅ **MINIMUM:** 3.0 GiB/s throughput (proven baseline)
- ✅ **TARGET:** 3.2+ GiB/s sustained performance
- ✅ **REQUIRED:** Performance monitoring and alerting
- ❌ **FORBIDDEN:** Performance degradation without approval

**Application Performance:**
- ✅ **REQUIRED:** API response times <500ms (95th percentile)
- ✅ **REQUIRED:** UI load times <2 seconds initial
- ✅ **REQUIRED:** Real-time updates <1 second latency
- ✅ **REQUIRED:** System handles 50+ concurrent operations

**Scalability Requirements:**
- ✅ **REQUIRED:** Support 1000+ VMs per Control Plane
- ✅ **REQUIRED:** Support 100+ concurrent backup jobs
- ✅ **REQUIRED:** Horizontal scaling documentation
- ✅ **REQUIRED:** Resource usage monitoring

---

## 🗃️ API DOCUMENTATION REQUIREMENTS

### **API Documentation Standards**

**Location:** `/source/current/api-documentation/` (MANDATORY)

**Required Files:**
```
api-documentation/
├── API_REFERENCE.md         # Complete endpoint documentation
├── DB_SCHEMA.md             # 🔴 CRITICAL: Current database schema
├── AUTHENTICATION.md        # Auth and RBAC documentation
├── ERROR_CODES.md           # All error codes and meanings
├── CHANGELOG.md             # API version history
├── EXAMPLES/                # Request/response examples
│   ├── backup-operations.md
│   ├── restore-operations.md
│   └── platform-management.md
└── openapi.yaml            # Machine-readable API spec
```

**Update Requirements:**
- ✅ **MANDATORY:** Update API docs BEFORE merging API changes
- ✅ **MANDATORY:** Update DB_SCHEMA.md with every migration
- ✅ **MANDATORY:** Add examples for every new endpoint
- ✅ **MANDATORY:** Document all error conditions
- ❌ **FORBIDDEN:** Merging API changes without doc updates

### **API Documentation Standards**

**Endpoint Documentation Format:**
```markdown
## POST /api/v1/backup/start

**Description:** Start backup operation for specified VM

**Authentication:** Required (Bearer token)

**Request:**
```json
{
  "vm_id": "4205784a-098a-40f1-1f1e-a5cd2597fd59",
  "vm_name": "database-prod-01", 
  "platform": "vmware",
  "repository_id": "local-ssd-primary",
  "backup_type": "incremental",
  "change_id": "52 3c ec 11 9e 2c 4c 3d-87 4a c3 4e 85 f2 ea 95/446"
}
```

**Response (Success):**
```json
{
  "backup_job_id": "backup-db-prod-20251004120000",
  "status": "pending",
  "estimated_duration": "8 minutes",
  "nbd_endpoint": "localhost:10809/backup-export-abc123"
}
```

**Response (Error):**
```json
{
  "error": "vm_not_found",
  "message": "VM with ID 4205784a-098a-40f1-1f1e-a5cd2597fd59 not found",
  "error_code": "SEND-404-001",
  "details": {
    "searched_platforms": ["vmware", "cloudstack"],
    "suggestions": ["Check VM ID format", "Verify platform connection"]
  }
}
```

**Error Codes:**
- `SEND-404-001`: VM not found in any connected platform
- `SEND-403-002`: Insufficient permissions for VM access
- `SEND-429-003`: Backup job limit exceeded

**Example Usage:**
```bash
curl -X POST https://api.sendense.com/v1/backup/start \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d @backup-request.json
```
```

---

## 🏗️ ARCHITECTURE COMPLIANCE

### **Component Integration Rules**

**Volume Operations:**
- ✅ **MANDATORY:** ALL volume operations MUST use Volume Daemon
- ❌ **FORBIDDEN:** Direct CloudStack/platform SDK calls for volumes
- ✅ **REQUIRED:** Use `internal/common/volume_client.go` interface

**Logging and Job Tracking:**
- ✅ **MANDATORY:** ALL business logic MUST use `internal/joblog`
- ❌ **FORBIDDEN:** Direct `logrus`, `slog`, or `fmt.Printf` in operation logic
- ✅ **REQUIRED:** Pattern: `StartJob() → RunStep() → EndJob()`

**Database Access:**
- ✅ **MANDATORY:** ALL database queries via repository pattern
- ❌ **FORBIDDEN:** Direct SQL in business logic
- ✅ **REQUIRED:** Migration-based schema changes only
- ✅ **REQUIRED:** Validate field names against schema documentation

**Networking:**
- ✅ **MANDATORY:** ALL traffic via SSH tunnel port 443
- ❌ **FORBIDDEN:** Direct NBD port exposure
- ✅ **REQUIRED:** Ed25519 key authentication for tunnels

---

## 📋 QUALITY GATES

### **Merge Requirements (Automated)**

**Before any merge to main:**
```yaml
# .github/workflows/quality-gates.yml
quality_gates:
  - lint_check: pass          # No linting errors
  - unit_tests: pass          # 100% unit test pass rate
  - integration_tests: pass   # 100% integration test pass
  - coverage_check: >80%      # Minimum code coverage
  - security_scan: pass       # No high/critical vulnerabilities  
  - performance_check: pass   # No performance regressions
  - api_docs_updated: true    # API documentation current
  - changelog_updated: true   # CHANGELOG.md updated
```

### **Release Readiness (Manual Review)**

**Before any release:**
- [ ] **Security Review:** Penetration testing completed
- [ ] **Performance Review:** Benchmarks meet or exceed targets
- [ ] **Documentation Review:** All docs current and accurate
- [ ] **Architecture Review:** Compliance with project rules verified
- [ ] **Deployment Review:** Rollback procedures tested
- [ ] **Support Review:** Runbooks and escalation procedures ready

---

## 🚨 VIOLATION CONSEQUENCES

### **Rule Violations (Escalating Response)**

**First Violation:**
- Code review rejection with explanation
- Documentation of violation and correction
- Additional training if needed

**Second Violation:**
- Mandatory code review for all future changes
- Architecture team consultation required
- Performance improvement plan

**Third Violation:**
- Removal from project
- Code audit of all previous contributions
- Process improvement review

### **Critical Violations (Immediate Response)**

**Security Violations:**
- Hardcoded credentials, exposed secrets, disabled security
- **Response:** Immediate code revert, security audit, incident response

**Data Loss Risks:**
- Direct database modifications, untested migrations, data corruption risks
- **Response:** Immediate halt, backup verification, rollback procedures

**Production Deployment Violations:**
- Deploying untested code, skipping quality gates, unauthorized changes
- **Response:** Immediate rollback, incident investigation, process review

---

## 📚 MASTER AI ASSISTANT PROMPT

### **New Session Initialization (Mandatory)**

**When starting ANY new AI session on Sendense:**

1. **Read Project Context (Required Order):**
   ```bash
   1. /sendense/PROJECT_RULES.md (this file - FIRST)
   2. /sendense/project-goals/README.md (project overview)
   3. /sendense/project-goals/TERMINOLOGY.md (naming conventions)
   4. /sendense/project-goals/architecture/01-system-overview.md
   5. /sendense/source/current/api-documentation/DB_SCHEMA.md (CRITICAL)
   6. /sendense/source/current/VERSION.txt (current version)
   ```

2. **Understand Current State:**
   ```bash
   7. /sendense/project-goals/phases/phase-1-vmware-backup.md (current phase)
   8. /sendense/CHANGELOG.md (recent changes)
   9. /sendense/source/current/api-documentation/API_REFERENCE.md
   ```

3. **Before Making ANY Changes:**
   - Verify compliance with PROJECT_RULES.md
   - Check API documentation current status
   - Confirm changes align with approved project-goals
   - Validate against current database schema

**AI Assistant Behavior Rules:**
- ❌ **FORBIDDEN:** Making changes without reading project rules
- ❌ **FORBIDDEN:** Assuming field names or API structure
- ❌ **FORBIDDEN:** Creating simulation or placeholder code
- ❌ **FORBIDDEN:** Claiming code is "production ready" without evidence
- ✅ **REQUIRED:** Always validate against documented schema and APIs
- ✅ **REQUIRED:** Follow established patterns and conventions
- ✅ **REQUIRED:** Update documentation with all changes

---

## 💼 ENTERPRISE DEVELOPMENT STANDARDS

### **Professional Code Standards**

**Error Handling:**
```go
// ✅ REQUIRED: Comprehensive error handling
func StartBackup(ctx context.Context, vmID string) (*BackupJob, error) {
    // Validate inputs
    if vmID == "" {
        return nil, fmt.Errorf("vmID is required")
    }
    
    // Use structured logging
    logger := logging.FromContext(ctx)
    logger.Info("Starting backup", "vm_id", vmID)
    
    // Business logic with error context
    job, err := createBackupJob(ctx, vmID)
    if err != nil {
        logger.Error("Failed to create backup job", "vm_id", vmID, "error", err)
        return nil, fmt.Errorf("backup job creation failed for VM %s: %w", vmID, err)
    }
    
    // Success logging
    logger.Info("Backup job created successfully", 
        "vm_id", vmID, 
        "job_id", job.ID,
        "estimated_duration", job.EstimatedDuration)
    
    return job, nil
}
```

**Configuration Management:**
```go
// ✅ REQUIRED: Structured configuration
type BackupConfig struct {
    MaxConcurrentJobs    int           `yaml:"max_concurrent_jobs" validate:"min=1,max=50"`
    DefaultRepository    string        `yaml:"default_repository" validate:"required"`
    RetentionPolicyDays  int           `yaml:"retention_days" validate:"min=1,max=2555"`
    PerformanceTargets   PerformanceConfig `yaml:"performance"`
}

// ❌ FORBIDDEN: Magic numbers or hardcoded values
const maxJobs = 10 // What's the reasoning? Why 10?
```

**Resource Management:**
```go
// ✅ REQUIRED: Proper resource cleanup
func (b *BackupService) executeBackup(ctx context.Context, job *BackupJob) error {
    // Setup resources
    nbdConn, err := b.nbdClient.Connect(job.NBDEndpoint)
    if err != nil {
        return err
    }
    defer nbdConn.Close() // ✅ Always clean up
    
    tempFile, err := b.createTempFile(job.ID)
    if err != nil {
        return err
    }
    defer os.Remove(tempFile) // ✅ Clean up temp files
    
    // Business logic...
    
    return nil
}
```

---

## 🎯 SUCCESS METRICS AND ACCOUNTABILITY

### **Project Success Criteria**

**Technical Success:**
- Zero security vulnerabilities in production
- 99.9% uptime for all production services
- Performance targets met or exceeded
- Zero data loss incidents
- Complete test coverage maintained

**Process Success:**
- 100% compliance with project rules
- Zero unauthorized deviations from roadmap
- Documentation accuracy >99%
- Change management process followed 100%

**Business Success:**
- Customer satisfaction >4.5/5
- Performance competitive advantage maintained
- Feature delivery on schedule
- Zero customer-facing quality issues

### **Accountability Framework**

**Engineering Team:**
- Follow all technical standards
- Maintain documentation currency
- Complete testing requirements
- Security compliance

**Project Management:**
- Enforce process compliance
- Roadmap adherence tracking
- Quality gate verification
- Stakeholder communication

**Leadership:**
- Resource allocation
- Strategic direction
- Quality standards approval
- Customer success oversight

---

## 📞 ESCALATION AND SUPPORT

### **When to Escalate**

**Immediate Escalation Required:**
- Security vulnerabilities discovered
- Performance regression >20%
- Data loss or corruption risk
- Customer-facing outages
- Rule violations by team members

**Standard Escalation:**
- Roadmap deviation requests
- Architecture change proposals
- New technology adoption
- Resource requirement changes

### **Support Channels**

**Internal:**
- Architecture team: Complex design decisions
- Security team: Vulnerability response
- DevOps team: Infrastructure and deployment
- QA team: Testing strategy and execution

**External:**
- Customer success: User experience and adoption
- Sales engineering: Competitive positioning
- Professional services: Implementation support

---

## 🎯 PROJECT EXECUTION EXCELLENCE

### **Daily Practices**

**Every Developer, Every Day:**
- [ ] Pull latest changes and read any new documentation
- [ ] Run full test suite before starting work
- [ ] Update API documentation with any API changes
- [ ] Commit with descriptive messages and scope
- [ ] Verify no rule violations before requesting review

**Weekly Practices:**
- [ ] Architecture review for significant changes
- [ ] Performance benchmark verification
- [ ] Security scan review and remediation
- [ ] Documentation audit and updates
- [ ] Customer feedback review and incorporation

**Release Practices:**
- [ ] Complete quality gate verification
- [ ] Performance benchmark comparison
- [ ] Security review and sign-off
- [ ] Rollback procedure testing
- [ ] Support team training and handoff

---

**THIS IS NOT OPTIONAL. THESE RULES ENSURE SENDENSE BECOMES THE ENTERPRISE-GRADE PLATFORM THAT DESTROYS VEEAM, NOT ANOTHER SHITTY STARTUP THAT FAILS DUE TO POOR ENGINEERING DISCIPLINE.**

---

**Document Owner:** Engineering Leadership  
**Enforcement:** Mandatory for all team members  
**Review Cycle:** Monthly or on major violations  
**Last Updated:** October 4, 2025  
**Status:** 🔴 **ACTIVE - MANDATORY COMPLIANCE**
