# START HERE - Sendense Project Orientation

**Essential Reading for ALL Team Members and AI Assistants**

**Last Updated:** October 4, 2025  
**Status:** 🔴 **MANDATORY - READ BEFORE ANY WORK**

---

## 🎯 SENDENSE PROJECT OVERVIEW

**Vision:** Universal backup and replication platform that breaks vendor lock-in and destroys Veeam with modern architecture and unique cross-platform capabilities.

**Unique Advantage:** ONLY vendor offering VMware → CloudStack migration with near-live replication.

**Business Model:** 3-tier pricing (Backup $10/VM, Enterprise $25/VM, Replication $100/VM) + MSP platform with 50% margins.

---

## 📚 MANDATORY READING ORDER (CRITICAL)

### **STEP 1: Project Rules (READ FIRST - NO EXCEPTIONS)**
```
start_here/PROJECT_RULES.md
```
🔴 **CRITICAL RULES:**
- NO "production ready" claims without complete testing
- NO simulations or placeholder code
- ALL code in `source/current/` directory ONLY
- ALL API changes MUST update `source/current/api-documentation/`
- NO deviations from approved `project-goals/` roadmap

### **STEP 2: AI Assistant Context Loading**
```
start_here/MASTER_AI_PROMPT.md
```
🤖 **FOR AI ASSISTANTS:**
- Context loading procedure for new sessions
- Common mistakes to avoid
- Emergency procedures and escalation
- Auto-reload trigger instructions

### **STEP 3: System Understanding (CRITICAL)**
```
start_here/LEGACY-SYSTEM-CLARIFICATION.md  # Current vs deprecated code (CRITICAL)
start_here/GOVERNANCE-IMPROVEMENTS.md      # Framework improvements and AI feedback
start_here/PHASE_1_CONTEXT_HELPER.md       # Phase 1 quick reference (backups)
job-sheets/CURRENT-ACTIVE-WORK.md          # Active work tracking
```

### **STEP 4: Project Structure and Governance**
```
start_here/BINARY_MANAGEMENT.md            # Binary organization and versioning
start_here/CHANGELOG.md                    # Change tracking standards  
start_here/PROJECT_GOVERNANCE_SUMMARY.md   # Complete framework overview
```

---

## 🎯 PROJECT GOALS INTEGRATION

### **Complete Roadmap Location**
```
project-goals/                     # Master project roadmap (24 documents)
├── README.md                     # Project overview and structure
├── TERMINOLOGY.md                # descend/ascend/transcend naming
├── architecture/                 # System design (2 documents)
├── editions/                     # Product tiers and competitive analysis
├── modules/                      # Technical modules (11 documents)
└── phases/                       # Implementation phases (7 documents)
```

### **JOB SHEET REQUIREMENT (MANDATORY)**

**Every Development Task Must:**
1. **Link to Project Goals:** Reference specific phase/module from `project-goals/`
2. **Create Job Sheet:** Detailed task breakdown with acceptance criteria
3. **Track Progress:** Mark off tasks as completed with evidence
4. **Update Documentation:** API docs, schema docs, changelogs with every change

**Job Sheet Format:**
```markdown
# Job Sheet: [Task Name]

**Project Goal Reference:** /project-goals/phases/phase-1-vmware-backup.md (Task 2.3)
**Assigned:** [Developer Name]
**Started:** 2025-10-04
**Target Completion:** 2025-10-11

## Task Breakdown
- [ ] Subtask 1 with acceptance criteria
- [ ] Subtask 2 with measurable outcome
- [ ] Subtask 3 with testing requirements

## Completion Criteria
- [ ] All tests pass (unit, integration, e2e)
- [ ] API documentation updated (if applicable)
- [ ] Performance benchmarks maintained
- [ ] Security review completed
- [ ] Code review approved

## Documentation Updates Required
- [ ] Update /source/current/api-documentation/API_REFERENCE.md
- [ ] Update /source/current/api-documentation/DB_SCHEMA.md
- [ ] Update CHANGELOG.md with changes
- [ ] Update deployment scripts if needed

## Evidence of Completion
- Link to passing CI/CD build
- Link to performance test results
- Link to security scan results
- Screenshots of working functionality
```

---

## 🏗️ UPDATED COMPONENT TERMINOLOGY

### **New Appliance Names (Effective Immediately)**

| Old Name | New Name | Abbreviation | Purpose |
|----------|----------|--------------|---------|
| **VMA** | **Sendense Node Appliance** | **SNA** | Source-side data capture |
| **OMA** | **Sendense Hub Appliance** | **SHA** | Central orchestration |
| **MSP Control** | **Sendense Control Appliance** | **SCA** | Multi-tenant cloud control |

### **Updated Data Flow**
```
VMware vCenter
     ↓
Sendense Node Appliance (SNA)     # Captures data from sources
     ↓ (SSH tunnel, port 443)
Sendense Hub Appliance (SHA)      # Orchestrates operations
     ↓ (API management)
Sendense Control Appliance (SCA)  # Multi-tenant MSP control
```

### **Code References to Update**
```bash
# Find and replace across codebase (when updating)
Old → New:
VMA → SNA (Sendense Node Appliance)
OMA → SHA (Sendense Hub Appliance)
"VMware Migration Appliance" → "Sendense Node Appliance"
"OSSEA Migration Appliance" → "Sendense Hub Appliance"
```

---

## 📦 BINARY DEPLOYMENT DIRECTORIES

### **Deployment Structure (Updated)**

```
sendense/
├── deployment/
│   ├── sha-appliance/              # Sendense Hub Appliance (on-prem)
│   │   ├── binaries/
│   │   │   ├── sendense-hub-v3.0.1-linux-amd64-20251004-abc123ef
│   │   │   ├── volume-daemon-v1.2.1-linux-amd64-20251004-def456ab
│   │   │   └── CHECKSUMS.sha256
│   │   ├── configs/
│   │   │   ├── sendense-hub.service
│   │   │   ├── volume-daemon.service
│   │   │   └── config-templates/
│   │   ├── database/
│   │   │   ├── production-schema.sql
│   │   │   └── migrations/
│   │   ├── scripts/
│   │   │   ├── deploy-sha.sh
│   │   │   ├── upgrade-sha.sh
│   │   │   └── rollback-sha.sh
│   │   └── gui/
│   │       ├── sendense-cockpit-v1.2.0.tar.gz
│   │       └── nginx.conf
│   ├── sna-appliance/              # Sendense Node Appliance (source-side)
│   │   ├── binaries/
│   │   │   ├── sendense-node-vmware-v2.1.5-linux-amd64-20251004-ghi789cd
│   │   │   ├── sendense-node-cloudstack-v1.0.3-linux-amd64-20251004-jkl012ef
│   │   │   └── CHECKSUMS.sha256
│   │   ├── configs/
│   │   │   ├── sendense-node.service
│   │   │   ├── ssh-tunnel.service
│   │   │   └── platform-configs/
│   │   ├── scripts/
│   │   │   ├── deploy-sna.sh
│   │   │   ├── setup-vmware-node.sh
│   │   │   ├── setup-cloudstack-node.sh
│   │   │   └── setup-ssh-tunnel.sh
│   │   └── dependencies/
│   │       ├── vddk-libs/          # VMware VDDK (if licensed)
│   │       └── system-packages.list
│   └── sca-appliance/              # Sendense Control Appliance (cloud MSP)
│       ├── binaries/
│       │   ├── sendense-control-v1.0.0-linux-amd64-20251004-mno345gh
│       │   ├── sendense-msp-api-v1.0.0-linux-amd64-20251004-pqr678ij
│       │   └── CHECKSUMS.sha256
│       ├── configs/
│       │   ├── kubernetes/         # K8s deployment manifests
│       │   ├── docker/             # Docker configurations
│       │   └── cloud-configs/      # AWS/Azure deployment configs
│       ├── scripts/
│       │   ├── deploy-sca-aws.sh
│       │   ├── deploy-sca-azure.sh
│       │   └── deploy-sca-k8s.sh
│       └── licensing/
│           ├── license-server-config.yaml
│           └── rsa-key-templates/
```

---

## 🔄 AUTOMATIC DEPLOYMENT SCRIPT UPDATES

### **Deployment Script Maintenance Rules**

**Location:** Each appliance has deployment scripts that MUST be kept current

**SHA Deployment Scripts:** `deployment/sha-appliance/scripts/`
- `deploy-sha.sh` - Fresh SHA deployment
- `upgrade-sha.sh` - Upgrade existing SHA
- `rollback-sha.sh` - Rollback to previous version

**SNA Deployment Scripts:** `deployment/sna-appliance/scripts/`  
- `deploy-sna.sh` - Fresh SNA deployment
- `setup-vmware-node.sh` - VMware-specific SNA setup
- `setup-cloudstack-node.sh` - CloudStack-specific SNA setup

**SCA Deployment Scripts:** `deployment/sca-appliance/scripts/`
- `deploy-sca-aws.sh` - AWS cloud deployment
- `deploy-sca-k8s.sh` - Kubernetes deployment

**Update Requirements:**
- ✅ **MANDATORY:** Update deployment scripts with every binary release
- ✅ **MANDATORY:** Test deployment scripts with every release
- ✅ **MANDATORY:** Version deployment scripts with appliance versions
- ❌ **FORBIDDEN:** Deployment scripts that don't match current binaries

---

## 🤖 AI CONTEXT MANAGEMENT (AUTO-RELOAD)

### **Context Exhaustion Detection**

**Auto-Reload Trigger System:**
```markdown
## AI ASSISTANT CONTEXT MANAGEMENT

**When you detect context approaching limits:**

### AUTOMATIC RELOAD PROCEDURE (REQUIRED)

1. **Detect Context Exhaustion:**
   - If you feel context memory becoming unclear
   - If you start making assumptions about database fields
   - If you can't remember project rules or recent decisions
   - If conversation history becomes too long to maintain accuracy

2. **Immediate Actions (MANDATORY):**
   ```
   CONTEXT RELOAD REQUIRED - RESTARTING SESSION
   
   Reason: [Context exhaustion/memory unclear/assumptions being made]
   
   NEXT AI SESSION MUST:
   1. Read start_here/MASTER_AI_PROMPT.md FIRST
   2. Follow mandatory reading order
   3. Validate current project state
   4. Continue work from documented state
   
   DO NOT:
   - Make database field assumptions
   - Create duplicate endpoints
   - Deviate from project goals
   - Skip context loading procedure
   ```

3. **Handoff Information:**
   - Document exactly what you were working on
   - Reference specific project goals task
   - Note any discoveries or decisions made
   - Provide clear next steps for new session

### CONTEXT RELOAD TRIGGERS:

**Automatic Triggers:**
- Can't recall specific project rules
- Making assumptions about API endpoints
- Unsure about database schema field names
- Forgetting appliance terminology (SNA/SHA/SCA)
- Unclear about current project phase or goals

**Manual Triggers:**
- User says "reload context" or "start fresh session"
- Major scope change or new requirements
- After completing major phase/milestone
- Beginning new complex task or investigation
```

### **Context Handoff Template**

```markdown
# AI SESSION HANDOFF

**Session End Reason:** Context exhaustion detected
**Date/Time:** 2025-10-04 15:30:00 UTC
**Duration:** 2 hours 15 minutes

## Work Completed This Session
- [x] Task 1: Created project governance framework
- [x] Task 2: Updated appliance terminology (VMA→SNA, OMA→SHA)  
- [x] Task 3: Established binary management rules
- [ ] Task 4: Update deployment scripts (IN PROGRESS)

## Current Project State
- **Active Phase:** Phase 1 - VMware Backups (Week 2 of 6)
- **Last Binary Version:** SHA v3.0.1, SNA-VMware v2.1.5
- **API Documentation Status:** Current as of Oct 4 15:00
- **Outstanding Issues:** None critical

## Next Session Must Do
1. **Read start_here/MASTER_AI_PROMPT.md** (mandatory context loading)
2. **Continue Task 4:** Update deployment scripts with new terminology
3. **Validate:** API documentation still current
4. **Reference:** project-goals/phases/phase-1-vmware-backup.md (Task 7)

## Decisions Made This Session
- Adopted SNA/SHA/SCA terminology across platform
- Established start_here/ directory for governance
- Created job sheet linking requirement to project goals

## Next AI Assistant Instructions
- DO NOT make database field assumptions
- DO NOT create duplicate endpoints  
- DO NOT skip mandatory reading order
- DO follow established patterns and project rules
```

---

## 📋 JOB SHEET SYSTEM

### **Job Sheet Storage & Organization**

**Active Job Sheets:** `job-sheets/YYYY-MM-DD-[task-description].md`
**Completed Job Sheets:** Move to `job-sheets/archive/YYYY/MM/` when complete
**Current Work:** `job-sheets/CURRENT-ACTIVE-WORK.md` (link to active sheet)

**Job Sheet Lifecycle:**
1. **Create:** `job-sheets/2025-10-04-vmware-backup.md` (active work)
2. **Reference:** Update `job-sheets/CURRENT-ACTIVE-WORK.md` to point to active sheet
3. **Complete:** Move to `job-sheets/archive/2025/10/` when task complete
4. **Future Reference:** Completed sheets accessible for context in new sessions

### **Job Sheet Creation Template**

```markdown
# Job Sheet: [Specific Task Name]

**Project Goal Reference:** /project-goals/phases/[phase-name].md → Task [X.Y]
**Job Sheet Location:** job-sheets/YYYY-MM-DD-[task-description].md  
**Archive Location:** job-sheets/archive/YYYY/MM/ (when complete)
**Assigned:** [Developer/AI Session]
**Priority:** Critical/High/Medium/Low
**Started:** 2025-10-04
**Target Completion:** 2025-10-11
**Estimated Effort:** [X hours/days]

## Task Link to Project Goals
**Specific Reference:**
- **Phase:** [Phase 1: VMware Backups]
- **Task Number:** [Task 2.3: NBD File Export Handler]  
- **Acceptance Criteria:** [As defined in project goals]
- **Business Value:** [Revenue impact, competitive advantage, etc.]

## Task Breakdown (Checkboxes Required)
- [ ] **Subtask 1:** [Specific deliverable with measurable outcome]
- [ ] **Subtask 2:** [Specific deliverable with testing requirement]
- [ ] **Subtask 3:** [Specific deliverable with documentation update]

## Technical Requirements
- [ ] **Code Quality:** Follow PROJECT_RULES.md standards
- [ ] **Testing:** Unit tests (80%+ coverage), integration tests, e2e tests
- [ ] **Security:** Security scan passing, no vulnerabilities
- [ ] **Performance:** Meet or exceed benchmark targets
- [ ] **Documentation:** API docs, schema docs, changelog updated

## Documentation Updates Required
- [ ] **API Reference:** /source/current/api-documentation/API_REFERENCE.md
- [ ] **Database Schema:** /source/current/api-documentation/DB_SCHEMA.md
- [ ] **Changelog:** CHANGELOG.md with semantic versioning
- [ ] **Build Manifest:** Update binary manifests if needed
- [ ] **Deployment Scripts:** Update if deployment process changes

## Dependencies
- **Blocks:** [Other tasks that must complete first]
- **Blocked By:** [Tasks waiting for this completion]
- **External:** [External dependencies or decisions needed]

## Success Criteria (Must All Be Met)
- [ ] **Functional:** Feature works as specified in project goals
- [ ] **Performance:** Meets or exceeds benchmark requirements  
- [ ] **Security:** Passes security review and vulnerability scans
- [ ] **Documentation:** All required docs updated and accurate
- [ ] **Testing:** All test categories pass
- [ ] **Integration:** Works with existing platform components
- [ ] **Deployment:** Deployment scripts updated and tested

## Evidence of Completion (Required)
- **CI/CD Build:** [Link to passing build]
- **Test Results:** [Link to test reports]
- **Performance:** [Link to benchmark results]  
- **Security:** [Link to security scan results]
- **Code Review:** [Link to approved code review]
- **Documentation:** [Links to updated documentation]

## Project Goals Task Completion
**Mark this task complete in project goals when done:**
```bash
# Update project goals document
vi project-goals/phases/phase-1-vmware-backup.md
# Find Task X.Y and mark as [x] COMPLETED with date and evidence
```

## Handoff to Next Task
- **Next Task:** [Reference to next project goals task]
- **Dependencies Satisfied:** [What this completion enables]
- **Knowledge Transfer:** [Important learnings for next task]

---

**Job Sheet Owner:** [Primary Developer]  
**Reviewer:** [Lead Developer/Architect]  
**Project Goals Link:** [Specific phase and task reference]  
**Completion Status:** 🔴 IN PROGRESS / ✅ COMPLETED
```

---

## 🔄 APPLIANCE TERMINOLOGY UPDATES

### **Updated Component Names (Effective Immediately)**

```
Legacy Naming (DEPRECATED):
❌ VMA (VMware Migration Appliance)
❌ OMA (OSSEA Migration Appliance)  
❌ MSP Control Plane

New Naming (ACTIVE):
✅ SNA (Sendense Node Appliance)     # Source-side capture
✅ SHA (Sendense Hub Appliance)      # On-prem orchestration
✅ SCA (Sendense Control Appliance)  # Cloud MSP control
```

### **Data Flow with New Terminology**
```
VMware vCenter
     ↓ (vSphere API)
Sendense Node Appliance (SNA)       # Captures from VMware
     ↓ (SSH tunnel, NBD stream)
Sendense Hub Appliance (SHA)        # Orchestrates backup/restore/replication
     ↓ (Management API)
Sendense Control Appliance (SCA)    # MSP multi-tenant control
```

### **File Path Updates Required**
```bash
# Directory structure updates needed
source/current/
├── hub-appliance/           # Renamed from oma/
├── node-appliance/          # Renamed from vma/
├── control-appliance/       # New for MSP platform
└── api-documentation/       # Update with new terminology

deployment/
├── sha-appliance/           # Hub appliance deployment
├── sna-appliance/           # Node appliance deployment  
└── sca-appliance/           # Control appliance deployment
```

---

## 🎯 CONTEXT RELOAD AUTOMATION

### **AI Assistant Auto-Reload System**

**Auto-Reload Trigger Script:**
```markdown
## CONTEXT RELOAD DETECTION (For AI Assistants)

**Check These Signals Every 10 Messages:**

### Critical Reload Triggers (Immediate)
- [ ] Can't remember specific database field names
- [ ] Making assumptions about API endpoint structure
- [ ] Forgetting project rules (simulations, binary locations, etc.)
- [ ] Unclear about current project phase or active tasks
- [ ] Using deprecated terminology (VMA/OMA instead of SNA/SHA/SCA)

### Warning Reload Triggers (Soon)
- [ ] Conversation becoming too long to track effectively
- [ ] Multiple context switches between different topics
- [ ] Uncertainty about recent decisions or changes
- [ ] Forgetting recent API or schema changes

### Auto-Reload Procedure (When Triggered)
```markdown
🔄 CONTEXT RELOAD INITIATED

**Trigger:** [Specific reason - database assumption/API uncertainty/etc.]

**Preservation Note:**
Current task: [What you were working on]
Progress: [What was completed]
Next step: [What needs to happen next]
Reference: [Specific project goals link]

**Next AI Session Instructions:**
1. READ start_here/MASTER_AI_PROMPT.md (mandatory first step)
2. FOLLOW complete reading order (all required documents)
3. CONTINUE from documented project state
4. DO NOT make assumptions about database fields or APIs
5. VALIDATE all information against current documentation

**Handoff Complete - NEW SESSION REQUIRED**
```

### Manual Reload Command
If user types: "reload context" or "restart session"
Response: Execute auto-reload procedure immediately
```

---

## ⚙️ AUTOMATED PROJECT MAINTENANCE

### **Daily Automated Checks**

```bash
#!/bin/bash
# daily-project-health-check.sh

echo "🔍 Daily Sendense Project Health Check - $(date)"

# 1. Verify no binaries in source code
if find source/current/ -type f -executable -size +1M | grep -q .; then
    echo "❌ VIOLATION: Binaries found in source code"
    find source/current/ -type f -executable -size +1M
    exit 1
fi

# 2. Check API documentation currency
if git diff HEAD~1 --name-only | grep -E "(handlers|routes|api)" > /dev/null; then
    if ! git diff HEAD~1 --name-only | grep "api-documentation" > /dev/null; then
        echo "❌ VIOLATION: API changes without documentation updates"
        exit 1
    fi
fi

# 3. Validate deployment script currency
for script in deployment/*/scripts/*.sh; do
    if [[ ! -x "$script" ]]; then
        echo "⚠️ WARNING: Deployment script not executable: $script"
    fi
done

# 4. Check terminology compliance
if grep -r "VMA\|OMA" source/current/ --exclude-dir=.git > /dev/null; then
    echo "⚠️ WARNING: Old terminology found (VMA/OMA), should be SNA/SHA/SCA"
    grep -r "VMA\|OMA" source/current/ --exclude-dir=.git
fi

# 5. Verify project goals task tracking
if [[ ! -f "current-active-job-sheet.md" ]]; then
    echo "⚠️ WARNING: No active job sheet found - all work must link to project goals"
fi

echo "✅ Daily health check completed"
```

### **Weekly Project Status Report**

```bash
# weekly-project-report.sh
#!/bin/bash

echo "📊 Sendense Weekly Project Report - Week of $(date +%Y-%m-%d)"

# Project Goals Progress
echo "## Project Goals Progress"
find project-goals/phases/ -name "*.md" -exec grep -l "✅" {} \; | wc -l
echo "Phases with completed tasks: $(find project-goals/phases/ -name "*.md" -exec grep -l "✅" {} \; | wc -l)/7"

# Build Quality
echo "## Build Quality"
echo "Total binaries: $(find source/builds/ -name "*-v*" -executable | wc -l)"
echo "Security scanned: $(find source/builds/ -name "*.security-scan" | wc -l)"
echo "Missing scans: $(find source/builds/ -name "*-v*" -executable ! -name "*.security-scan" | wc -l)"

# Documentation Currency
echo "## Documentation Status"
echo "API endpoints documented: $(grep -c "^## " source/current/api-documentation/API_REFERENCE.md)"
echo "Database tables documented: $(grep -c "^### " source/current/api-documentation/DB_SCHEMA.md)"

# Rule Compliance
echo "## Rule Compliance"
if find source/current/ -name "*.go" -exec grep -l "fmt.Printf\|log.Printf" {} \; > /dev/null; then
    echo "⚠️ Direct logging found (should use JobLog)"
fi

if grep -r "production ready" source/current/ > /dev/null; then
    echo "⚠️ Premature production ready claims found"
fi

echo "✅ Weekly report completed"
```

---

## 🎯 IMMEDIATE ACTION ITEMS

### **Team Implementation Tasks**

**Immediate (This Week):**
1. **Team Training:** All developers read and acknowledge start_here/PROJECT_RULES.md
2. **Terminology Update:** Global find/replace VMA→SNA, OMA→SHA in all code
3. **Binary Cleanup:** Move any scattered binaries to proper deployment directories  
4. **API Documentation Audit:** Verify /source/current/api-documentation/ is current
5. **Deployment Script Updates:** Update all scripts with new terminology and binary names

**Setup (Next Week):**
1. **Automated Checks:** Implement pre-commit hooks and CI/CD quality gates
2. **Context Management:** Train team on AI assistant context reload procedures
3. **Job Sheets:** Create job sheets for any active development work
4. **Process Validation:** Test full development workflow with new standards

### **Active Job Linking Example**

**If working on VMware backups:**
```markdown
# Job Sheet: Implement QCOW2 Backup Storage

**Project Goal Reference:** /project-goals/phases/phase-1-vmware-backup.md → Task 1.2
**Business Value:** Enables Backup Edition ($10/VM tier)
**Current Phase:** Phase 1 - VMware Backups (Week 1 of 6)

[Complete job sheet following template above]
```

---

## 📞 ESCALATION PROCEDURES

### **When to Escalate Immediately**

**Rule Violations:**
- Binaries committed to source/current/
- API changes without documentation updates
- "Production ready" claims without testing evidence
- Architecture violations (direct volume calls, etc.)

**Context Issues:**
- AI assistant making database field assumptions
- AI assistant creating endpoints not in roadmap
- AI assistant forgetting project rules mid-session
- Multiple AI sessions working on same task without coordination

**Project Deviations:**
- Tasks not linked to approved project goals
- Architecture changes without approval
- Scope creep or unauthorized feature additions

---

**THIS FRAMEWORK ENSURES SENDENSE DEVELOPMENT REMAINS PROFESSIONAL, DISCIPLINED, AND ALIGNED WITH BUSINESS OBJECTIVES**

**NO MORE AMATEUR HOUR - WE'RE BUILDING AN ENTERPRISE PLATFORM TO DESTROY VEEAM**

---

**Document Owner:** Project Leadership  
**Scope:** All Sendense development and AI assistance  
**Compliance:** Mandatory for all team members and AI sessions  
**Review:** Weekly compliance, monthly process improvement  
**Last Updated:** October 4, 2025  
**Status:** 🔴 **ACTIVE - IMMEDIATE COMPLIANCE REQUIRED**
