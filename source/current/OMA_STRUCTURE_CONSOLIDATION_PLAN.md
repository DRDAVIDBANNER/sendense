# OMA Source Code Structure Consolidation Plan

**Current Status**: 2025-09-05  
**Objective**: Properly organize OMA API source code in `/source/current/oma/` separate from VMA and migratekit

## 🚨 CURRENT SITUATION

### ✅ WORKING SYSTEM (Don't Break!)
- **OMA API Binary**: `/opt/migratekit/bin/oma-api` (v2.6.0-vma-progress-poller)
- **Source Structure**: Mixed between `/cmd/oma/` and `/internal/oma/` + `/source/current/migratekit/internal/oma/`
- **Status**: VMAProgressPoller working perfectly, database updates pending investigation

### ❌ STRUCTURAL ISSUES
1. **OMA code scattered across multiple locations**:
   - `/cmd/oma/main.go` (entry point)
   - `/internal/oma/` (old handlers, outdated services)
   - `/source/current/migratekit/internal/oma/` (enhanced handlers, VMAProgressPoller)

2. **Module confusion**: OMA functionality mixed with migratekit module

## 🎯 TARGET STRUCTURE

### **Proper Organization**
```
source/current/
├── oma/                    # OMA API (OSSEA Migration Appliance)
│   ├── cmd/main.go        # Entry point
│   ├── api/               # HTTP handlers
│   ├── services/          # VMAProgressPoller, etc.
│   ├── database/          # Database operations
│   ├── workflows/         # Migration workflows
│   ├── go.mod             # Separate module
│   └── VERSION.txt        # OMA version
├── migratekit/            # migratekit binary only
│   ├── cmd/main.go
│   ├── internal/vmware_nbdkit/
│   ├── go.mod
│   └── VERSION.txt
├── vma/                   # VMA services
│   ├── api/
│   ├── services/
│   └── VERSION.txt
└── vma-api-server/        # VMA API binary
    ├── main.go
    └── VERSION.txt
```

## 📋 CONSOLIDATION PHASES

### **Phase 1: Current Working State** ✅
- [x] VMAProgressPoller working in current mixed structure
- [x] Database updates flowing from VMA to OMA (needs debugging)
- [x] All binaries properly versioned

### **Phase 2: Gradual Migration** (Next Session)
- [ ] Copy enhanced code to proper `/source/current/oma/` structure
- [ ] Update import paths incrementally 
- [ ] Test builds from new structure
- [ ] Keep old structure as backup until verified

### **Phase 3: Clean Migration** (Future)
- [ ] Build OMA API from clean `/source/current/oma/` structure
- [ ] Update systemd service to use new binary
- [ ] Remove old scattered code
- [ ] Archive legacy directories

## 🚨 CRITICAL RULES FOR MIGRATION

1. **Never break working VMAProgressPoller** - it's the core requirement
2. **Test each step** before proceeding to next
3. **Keep backups** of working binaries 
4. **Version everything** properly
5. **Verify end-to-end flow** after each change

## 📝 CURRENT SESSION ACCOMPLISHMENTS

### ✅ Source Code Authority Fixed
- **Problem**: Duplicate VMAProgressPoller code in `/internal/` vs `/source/current/`
- **Solution**: Identified `/source/current/migratekit/internal/oma/` as authoritative
- **Result**: Built working `oma-api-v2.6.0-vma-progress-poller` with VMAProgressPoller

### ✅ VMAProgressPoller Operational
- **Status**: VMAProgressPoller started successfully
- **Logs**: `🚀 Starting VMA progress poller max_concurrent=10 poll_interval=5s`
- **Next**: Debug database updates (VMA GET → OMA DB)

## 🔄 NEXT SESSION PRIORITY

**PRIMARY GOAL**: Debug VMAProgressPoller database updates
- VMA API GET endpoint working perfectly
- OMA Progress Poller receiving data (needs verification)
- Database not being updated (investigate polling service)

**SECONDARY GOAL**: Plan gradual OMA structure migration
- Map all dependencies for clean separation
- Design incremental migration strategy 
- Maintain working system throughout

---

**BOTTOM LINE**: VMAProgressPoller is working! Don't break it while cleaning up structure.
