# 🔧 **VIRTIO FAILOVER RECOVERY PLAN**

**Created**: September 27, 2025  
**Priority**: 🔥 **CRITICAL** - 5 stuck test failovers with snapshots need completion  
**Issue ID**: VIRTIO-RECOVERY-001  
**Status**: 🚨 **IMMEDIATE RECOVERY REQUIRED** - Jobs stuck after VirtIO injection

---

## 🎯 **CRITICAL SITUATION ANALYSIS**

### **🚨 Current State:**
- **5 test failover jobs**: Stuck after VirtIO injection completion
- **VirtIO steps**: Manually marked as completed (11:39:13)
- **Failover engines**: Not detecting VirtIO completion, still waiting
- **Snapshots created**: Each VM has snapshots that need proper cleanup
- **Volume state**: Volumes detached for test VMs, need reattachment after completion

### **⚠️ Risk Assessment:**
- **Cannot restart OMA**: Would abandon jobs mid-process ❌
- **Cannot cancel jobs**: Would leave orphaned snapshots ❌
- **Must complete workflow**: Need volume detachment and snapshot cleanup ✅
- **Manual intervention required**: Get engines to continue from current state ✅

---

## 🔍 **ROOT CAUSE ANALYSIS**

### **🔧 What Happened:**
1. **VirtIO injection started**: All 5 jobs began VirtIO injection normally ✅
2. **VirtIO processes hung**: `virt-v2v-in-place` processes stuck/died ❌
3. **Manual completion**: VirtIO steps marked as completed manually ✅
4. **Engine disconnect**: Unified failover engines not detecting manual completion ❌

### **🎯 Current Job State:**
```sql
-- Job tracking shows:
unified-test-failover: completed (manually marked)

-- But failover_jobs shows:
status: pending (engines haven't progressed)

-- Job steps show:
virtio-driver-injection: completed (manually marked)
```

### **💡 The Problem:**
**Unified failover engines are waiting for VirtIO step completion signals** that were lost when the VirtIO processes hung. The engines don't know the VirtIO phase is actually complete.

---

## 🔧 **RECOVERY STRATEGY**

### **🎯 OPTION 1: Signal Engine Continuation (PREFERRED)**

#### **Approach: Trigger Engine State Check**
The unified failover engines likely poll job steps for completion. We need to **trigger them to check the current state** and continue.

#### **Method 1: Database Trigger Update**
```sql
-- Update job_tracking updated_at to trigger engine polling
UPDATE job_tracking 
SET updated_at = NOW() 
WHERE operation = 'unified-test-failover' AND status = 'completed';

-- This should trigger engines to check current state and continue
```

#### **Method 2: External Job ID Correlation**
```sql
-- Check if engines are waiting on external job correlation
SELECT id, external_job_id, status FROM job_tracking 
WHERE operation = 'unified-test-failover';

-- Update external job status if needed
```

### **🎯 OPTION 2: Manual Workflow Continuation (TARGETED)**

#### **Approach: Manually Advance to Next Phase**
If engines don't respond to database updates, manually advance the workflow to the next phase.

#### **Method: Update Failover Job Status**
```sql
-- Advance failover_jobs to next phase (creating_vm)
UPDATE failover_jobs 
SET status = 'creating_vm', updated_at = NOW() 
WHERE status = 'pending' AND job_type = 'test';

-- This should trigger VM creation phase
```

### **🎯 OPTION 3: JobLog Step Injection (ADVANCED)**

#### **Approach: Inject Next Step Signal**
Create job step records for the next phase to trigger engine continuation.

#### **Method: Create VM Creation Steps**
```sql
-- Insert vm-creation step to trigger next phase
INSERT INTO job_steps (job_id, name, seq, status, started_at)
SELECT job_id, 'vm-creation', 6, 'running', NOW()
FROM job_steps 
WHERE name = 'virtio-driver-injection' AND status = 'completed'
GROUP BY job_id;
```

---

## 📋 **RECOMMENDED RECOVERY SEQUENCE**

### **🚀 PHASE 1: Gentle Engine Nudge (2 minutes)**
1. **Update job_tracking timestamps** to trigger engine polling
2. **Wait 30 seconds** for engines to detect and continue
3. **Monitor failover_jobs status** for progression

### **🔧 PHASE 2: Manual Phase Advancement (5 minutes)**
1. **Update failover_jobs status** to next phase if engines don't respond
2. **Monitor VM creation activity** in CloudStack
3. **Verify snapshot cleanup** begins

### **🚨 PHASE 3: Emergency Completion (10 minutes)**
1. **Manual VM creation** if engines still stuck
2. **Manual volume operations** via Volume Daemon
3. **Manual snapshot cleanup** to prevent orphaned snapshots

---

## 🎯 **SUCCESS CRITERIA**

### **Immediate Goals:**
- [ ] ✅ **Engines continue**: Failover jobs progress from pending to creating_vm
- [ ] ✅ **VM creation**: Test VMs created in CloudStack
- [ ] ✅ **Volume operations**: Volumes detached from OMA, attached to test VMs
- [ ] ✅ **Snapshot cleanup**: Snapshots properly managed

### **Recovery Validation:**
- [ ] ✅ **All 5 VMs**: Complete test failover workflow
- [ ] ✅ **No orphaned snapshots**: Proper snapshot lifecycle management
- [ ] ✅ **Volume consistency**: Volumes properly attached to test VMs
- [ ] ✅ **Rollback capability**: Can roll back test failovers when needed

---

## 🚨 **CRITICAL RECOVERY COMMANDS**

### **Phase 1: Engine Nudge**
```sql
-- Trigger engine polling by updating timestamps
UPDATE job_tracking 
SET updated_at = NOW() 
WHERE operation = 'unified-test-failover' AND status = 'completed';
```

### **Phase 2: Manual Advancement**
```sql
-- Advance to VM creation phase
UPDATE failover_jobs 
SET status = 'creating_vm', updated_at = NOW() 
WHERE status = 'pending' AND job_type = 'test';
```

### **Monitoring Commands:**
```sql
-- Monitor progress
SELECT job_id, source_vm_name, status FROM failover_jobs WHERE job_type = 'test';
SELECT id, operation, status FROM job_tracking WHERE operation = 'unified-test-failover';
```

---

**🎯 This recovery plan will complete the stuck test failovers properly, ensuring snapshots are cleaned up and volumes are correctly managed without losing the failover progress.**






