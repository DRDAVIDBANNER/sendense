# Quick Reference: Machine Modal Data Flow

**Feature:** Machine Backup Details Modal  
**Date:** October 10, 2025

---

## 📊 **DATA FLOW**

```
User clicks VM row
    ↓
FlowMachinesTable.onMachineClick(machine)
    ↓
FlowDetailsPanel sets state
    ↓
MachineDetailsModal opens
    ↓
useMachineBackups hook fires
    ↓
API: GET /api/v1/backups?vm_name={machine.vm_name}&repository_id={flow.repository_id}
    ↓
Backend: backup_handlers.go ListBackups() queries database
    ↓
Returns: { backups: [...], total: N }
    ↓
Modal calculates KPIs client-side
    ↓
Displays: Summary + KPIs + Backup List
```

---

## 🎯 **KEY DATA POINTS**

### **Input (from FlowMachineInfo):**
```typescript
{
  context_id: "ctx-pgtest1-...",
  vm_name: "pgtest1",
  cpu_count: 2,
  memory_mb: 4096,
  os_type: "Ubuntu Linux",
  power_state: "poweredOn",
  disks: [
    { disk_id: "2000", size_gb: 102 },
    { disk_id: "2001", size_gb: 5 }
  ],
  backup_stats: {
    backup_count: 12,
    total_size_bytes: 1234567890,
    last_backup_at: "2025-10-10T02:00:00Z"
  }
}
```

### **API Response (BackupResponse[]):**
```json
{
  "backups": [
    {
      "backup_id": "backup-pgtest1-1728518400",
      "vm_name": "pgtest1",
      "backup_type": "full",
      "status": "completed",
      "bytes_transferred": 109521739776,
      "total_bytes": 109521739776,
      "disks_count": 2,
      "created_at": "2025-10-10T02:00:00Z",
      "started_at": "2025-10-10T02:00:15Z",
      "completed_at": "2025-10-10T02:45:23Z",
      "error_message": null
    },
    {
      "backup_id": "backup-pgtest1-1728432000",
      "vm_name": "pgtest1",
      "backup_type": "incremental",
      "status": "completed",
      "bytes_transferred": 4503599627,
      "total_bytes": 109521739776,
      "disks_count": 2,
      "created_at": "2025-10-09T02:00:00Z",
      "started_at": "2025-10-09T02:00:12Z",
      "completed_at": "2025-10-09T02:08:34Z",
      "error_message": null
    },
    {
      "backup_id": "backup-pgtest1-1728345600",
      "vm_name": "pgtest1",
      "backup_type": "incremental",
      "status": "failed",
      "bytes_transferred": 0,
      "total_bytes": 109521739776,
      "disks_count": 2,
      "created_at": "2025-10-08T02:00:00Z",
      "started_at": "2025-10-08T02:00:10Z",
      "completed_at": null,
      "error_message": "qemu-nbd process exited unexpectedly"
    }
  ],
  "total": 3
}
```

---

## 🧮 **KPI CALCULATIONS**

### **1. Total Backups**
```typescript
backups.length
```

### **2. Success Rate**
```typescript
const completed = backups.filter(b => b.status === 'completed').length;
const successRate = (completed / backups.length * 100).toFixed(0) + '%';
```

### **3. Average Size**
```typescript
const completedBackups = backups.filter(b => b.status === 'completed');
const totalBytes = completedBackups.reduce((sum, b) => sum + b.bytes_transferred, 0);
const avgSize = totalBytes / completedBackups.length;
// Format: formatBytes(avgSize)
```

### **4. Average Duration**
```typescript
const completedBackups = backups.filter(b => b.status === 'completed');
const totalSeconds = completedBackups.reduce((sum, b) => {
  const start = new Date(b.started_at).getTime();
  const end = new Date(b.completed_at).getTime();
  return sum + ((end - start) / 1000);
}, 0);
const avgDuration = totalSeconds / completedBackups.length;
// Format: formatDuration(avgDuration)
```

---

## 🎨 **DISPLAY FORMATTING**

### **Size (bytes_transferred):**
```
109521739776 → "102.0 GB"
4503599627   → "4.2 GB"
0            → "0 B"
```

### **Duration (completed_at - started_at):**
```
2708 seconds → "45m"
502 seconds  → "8m"
3661 seconds → "1h 1m"
```

### **Date (created_at):**
```
"2025-10-10T02:00:00Z" → "Oct 10, 02:00"
```

### **Type Badge:**
```
"full"        → Blue badge "Full"
"incremental" → Green badge "Incremental"
```

### **Status Badge:**
```
"completed" → Green ✅ "Success"
"failed"    → Red ❌ "Failed"
"running"   → Blue 🔄 "Running"
```

---

## 🚨 **GOTCHAS & EDGE CASES**

### **1. No Backups**
```typescript
if (backups.length === 0) {
  // Show empty state: "No backups found for this machine"
}
```

### **2. Failed Backup (no completed_at)**
```typescript
if (backup.status === 'failed' && !backup.completed_at) {
  // Duration = N/A
  // Show error_message
}
```

### **3. Running Backup (no completed_at)**
```typescript
if (backup.status === 'running' && !backup.completed_at) {
  // Show spinner
  // Duration = elapsed time since started_at
}
```

### **4. Zero bytes_transferred**
```typescript
if (backup.bytes_transferred === 0) {
  // Format as "0 B" not "—"
  // Could be failed backup or just started
}
```

### **5. Only Failed Backups**
```typescript
const completedBackups = backups.filter(b => b.status === 'completed');
if (completedBackups.length === 0) {
  // Avg Size = "N/A"
  // Avg Duration = "N/A"
  // Success Rate = "0%"
}
```

### **6. Multiple Repositories**
**IMPORTANT:** Filter by `repository_id` to only show backups from THIS flow's repository.

```typescript
// API call MUST include repository_id from flow
`/api/v1/backups?vm_name=${vmName}&repository_id=${repositoryId}`
```

---

## 🔍 **DEBUGGING**

### **API Not Returning Data:**
```bash
# Test API directly
curl "http://localhost:8082/api/v1/backups?vm_name=pgtest1&repository_id=repo-local-1"
```

### **Check Database:**
```sql
-- Count backups for VM
SELECT COUNT(*) FROM backup_jobs 
WHERE vm_name = 'pgtest1' 
  AND repository_id = 'repo-local-1' 
  AND id NOT LIKE '%-disk%'
  AND status = 'completed';

-- List backups
SELECT 
  id, 
  backup_type, 
  status, 
  bytes_transferred, 
  created_at, 
  completed_at
FROM backup_jobs 
WHERE vm_name = 'pgtest1' 
  AND repository_id = 'repo-local-1'
  AND id NOT LIKE '%-disk%'
ORDER BY created_at DESC;
```

### **Console Logging:**
```typescript
// In modal component
console.log('Machine:', machine);
console.log('Repository ID:', repositoryId);
console.log('Fetched backups:', backups);
console.log('Calculated KPIs:', kpis);
```

### **Common Issues:**
1. **Modal doesn't open:** Check `isOpen` state and click handler
2. **No data shown:** Check API response in Network tab
3. **Wrong calculations:** Filter `status === 'completed'` first
4. **Repository filter not working:** Verify `repository_id` is passed correctly
5. **Date formatting error:** Check for null/undefined `completed_at`

---

## 📝 **TYPE DEFINITIONS**

```typescript
// Backend response
interface BackupResponse {
  backup_id: string;
  vm_name: string;
  backup_type: 'full' | 'incremental' | 'differential';
  status: 'pending' | 'running' | 'completed' | 'failed' | 'cancelled';
  bytes_transferred: number;
  total_bytes: number;
  disks_count: number;
  created_at: string;
  started_at: string | null;
  completed_at: string | null;
  error_message: string | null;
}

// KPIs
interface BackupKPIs {
  total: number;
  successRate: string;  // "92%"
  avgSize: string;       // "42.3 GB"
  avgDuration: string;   // "18m"
}
```

---

## ✅ **VALIDATION CHECKLIST**

Before marking complete, verify:

- [ ] Click any VM row → modal opens instantly
- [ ] VM summary shows correct: CPU, Memory, Disks, OS, Power
- [ ] Total backups count matches API response
- [ ] Success rate calculates correctly (completed / total * 100)
- [ ] Average size shows only completed backups
- [ ] Average duration shows only completed backups
- [ ] Backup list shows newest first
- [ ] Type badges: Full=blue, Incremental=green
- [ ] Status badges: Success=green, Failed=red
- [ ] Failed backups show error messages
- [ ] Sizes formatted correctly (GB/TB)
- [ ] Durations formatted correctly (Xh Xm)
- [ ] Dates formatted correctly (MMM DD, HH:MM)
- [ ] Modal closes cleanly
- [ ] No console errors
- [ ] Repository filtering works (only shows backups from this repo)

---

**End of Reference**

