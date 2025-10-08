# Sendense API Reference

**Version:** v2.18.0  
**Last Updated:** October 8, 2025  
**Project:** Sendense Universal Backup Platform

---

## 📚 **Documentation Index**

### **Core API Documentation**
- **[OMA.md](./OMA.md)** - Sendense Hub Appliance (SHA) API endpoints
  - VM inventory, replication, failover, backup management
  - Repository management, backup policies, backup jobs
  - Progress tracking, network mapping, VMA enrollment
  
- **[VMA.md](./VMA.md)** - Sendense Node Appliance (SNA) API endpoints
  - VMware inventory discovery, progress reporting
  - VMA health monitoring, enrollment system

### **Database Documentation**
- **[DB_SCHEMA.md](./DB_SCHEMA.md)** - Complete database schema
  - All 36 tables with field definitions
  - Foreign key relationships and constraints
  - Index definitions and performance notes

- **[API_DB_MAPPING.md](./API_DB_MAPPING.md)** - API-to-database mapping
  - Which endpoints modify which tables
  - Data flow documentation
  - Transaction boundaries

### **Additional Documentation**
- **[MAINTENANCE_RULES.md](./MAINTENANCE_RULES.md)** - Documentation maintenance requirements
- **[BACKUP_REPOSITORY_GUI_INTEGRATION.md](./BACKUP_REPOSITORY_GUI_INTEGRATION.md)** - GUI integration guide

---

## 🎯 **Quick Start**

### **Most Important Endpoints**

**VM Operations:**
- `POST /api/v1/replications` - Start VM replication
- `GET /api/v1/vms` - List discovered VMs
- `POST /api/v1/failover/live` - Perform live failover
- `POST /api/v1/failover/test` - Perform test failover

**Backup Operations:**
- `POST /api/v1/repositories` - Create backup repository
- `POST /api/v1/backups` - Create VM backup
- `GET /api/v1/backups` - List backups
- `GET /api/v1/backups/chain` - Get backup chain for recovery

**Monitoring:**
- `GET /api/v1/health` - Service health
- `GET /api/v1/progress/{job_id}` - Job progress
- `GET /api/v1/repositories/{id}/storage` - Storage capacity

---

## 📋 **API Documentation Standards**

As per PROJECT_RULES, all endpoint documentation includes:
- ✅ HTTP method and path
- ✅ Request/response types
- ✅ Handler function location
- ✅ Authentication requirements
- ✅ Known callsites (internal consumers)
- ✅ Classification (Key/Auxiliary/Legacy)

---

## 🔐 **Authentication**

All `/api/v1/*` endpoints (except public VMA enrollment) require:
- **Type:** Bearer token
- **Header:** `Authorization: Bearer <token>`
- **Login:** `POST /api/v1/auth/login`

---

## 🏗️ **Architecture Notes**

### **Terminology Mapping (Code Navigation)**
For module paths, treat SHA (Hub) as `oma/*` and SNA (Node) as `vma/*`.  
Code directories named OMA/VMA; SHA/SNA are conceptual names used in docs.

### **Source Authority**
- **Canonical source:** `/sendense/source/current/`
- **OMA API routes:** `oma/api/server.go`
- **VMA API routes:** `vma/api/server.go`
- **Handlers:** `oma/api/handlers/*` and `vma/api/progress_handler.go`

### **Callsite References**
Known internal consumers:
- `oma/services/*` - Core services
- `oma/failover/*` - Failover engines
- `oma/workflows/*` - Migration workflows
- `vma/client/*` - VMA-to-OMA client
- `vma/services/*` - VMA services
- `migratekit/internal/*` - Migration engine

---

## 📝 **Maintenance Requirements**

**CRITICAL (PROJECT_RULES compliance):**
- ✅ Update this documentation with **EVERY API change**
- ✅ Update CHANGELOG.md for all significant changes
- ✅ Keep DB_SCHEMA.md synchronized with migrations
- ✅ Document all request/response schemas
- ✅ Add callsite references for new endpoints
- ❌ **FORBIDDEN:** Merging API changes without doc updates

---

## 📖 **Additional Resources**

- **Project Rules:** `/sendense/start_here/PROJECT_RULES.md`
- **Master Prompt:** `/sendense/start_here/MASTER_AI_PROMPT.md`
- **CHANGELOG:** `/sendense/start_here/CHANGELOG.md`
- **Phase 1 Goals:** `/sendense/project-goals/phases/phase-1-vmware-backup.md`

---

**For detailed endpoint documentation, see [OMA.md](./OMA.md) and [VMA.md](./VMA.md)**

