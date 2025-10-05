# Sendense Cockpit - Everything Close to Hand

**Date:** October 5, 2025  
**Design Principle:** Aviation Cockpit - Critical controls within immediate reach

---

## 🛩️ COCKPIT PRINCIPLE APPLIED

**"Everything Close to Hand" means:**
- **No hunting through menus** for critical operations
- **System status always visible** (like aircraft instruments)
- **Emergency controls prominent** (pause/stop operations)
- **Most common tasks** accessible with zero clicks
- **Context-aware interfaces** (hover previews, smart defaults)

---

## 🖥️ PROTECTION FLOWS COCKPIT WIREFRAME

### **Main Interface Layout (1920x1080)**

```
┌─────────────────────────────────────────────────────────────────────────────┐
│ SENDENSE COCKPIT                     🚨 2 ALERTS  👤 admin  🔍 [pgtest2   ] │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│ ⚡ FLIGHT CONTROLS (Always Visible - Zero Clicks) ─────────────────────────  │
│                                                                             │
│ [🚀 BACKUP NOW] [🔄 RESTORE] [🌉 REPLICATE] [⏸️ PAUSE ALL] [⚡ EMERGENCY STOP] │
│ └─ Primary operations require no menu navigation ──────────────────────────  │
│                                                                             │
│ 🎛️  INSTRUMENT PANEL (Live System Vitals) ─────────────────────────────────  │
│ ┌─────────────────────────────────────────────────────────────────────────┐ │
│ │ Fleet: 🟢 247 │ Speed: 🟢 3.2 GiB/s │ Storage: 🟡 89% │ Agents: 🟢 4/4  │ │
│ │     Protected │      Current Load   │     Local SSD   │   All Connected │ │
│ └─────────────────────────────────────────────────────────────────────────┘ │
│                                                                             │
│ 🌊 PROTECTION FLOWS (Grid View - Everything Visible) ──────────────────────  │
│                                                                             │
│ ┌─ ACTIVE FLOWS (4) ─────────────────────────────────────────────────────┐ │
│ │                                                                         │ │
│ │ ┌─ database-prod-01 ────────────┐ ┌─ exchange-server ───────────────┐   │ │
│ │ │ 📥 VMware → Local             │ │ 🌉 VMware → CloudStack          │   │ │
│ │ │ ████████████████▓▓▓▓ 83%      │ │ ████████████████████▓ 96%      │   │ │
│ │ │ 2.1 GiB/s  •  ETA: 4m 23s    │ │ 1.8 GiB/s  •  ETA: 45s        │   │ │
│ │ │ [⏸️] [🔍] [⏹️]                │ │ [⏸️] [🔍] [🧪] [🚀]             │   │ │
│ │ └───────────────────────────────┘ └─────────────────────────────────┘   │ │
│ │                                                                         │ │
│ │ ┌─ file-server-02 ──────────────┐ ┌─ web-cluster-01 ─────────────────┐   │ │
│ │ │ 📥 VMware → S3                │ │ 🔄 S3 → CloudStack              │   │ │
│ │ │ ██████▓▓▓▓▓▓▓▓▓▓▓ 34%         │ │ ██████████████████▓▓▓ 91%        │   │ │
│ │ │ 1.2 GiB/s  •  ETA: 18m        │ │ 2.7 GiB/s  •  ETA: 2m          │   │ │
│ │ │ [⏸️] [🔍] [⏹️] [⚙️]            │ │ [⏸️] [🔍] [⏹️]                   │   │ │
│ │ └───────────────────────────────┘ └─────────────────────────────────┘   │ │
│ └─────────────────────────────────────────────────────────────────────────┘ │
│                                                                             │
│ 🚀 INSTANT ACTIONS (Context-Aware) ────────────────────────────────────────  │
│                                                                             │
│ ┌─ READY FOR BACKUP ───┐ ┌─ READY FOR RESTORE ──┐ ┌─ REPLICATION STATUS ──┐ │
│ │ 🖥️  pgtest2           │ │ 📦 Last: 2h ago      │ │ 🌉 2 Active pairs     │ │
│ │ 🖥️  app-server-02     │ │ 📦 web-srv: 4h ago   │ │ 🌉 1 Healthy         │ │
│ │ 🖥️  database-dev      │ │ 📦 db-prod: 6h ago   │ │ 🌉 1 Attention       │ │
│ │                       │ │                      │ │                       │ │
│ │ [BACKUP SELECTED]     │ │ [RESTORE SELECTED]   │ │ [MANAGE PAIRS]        │ │
│ └───────────────────────┘ └──────────────────────┘ └───────────────────────┘ │
│                                                                             │
│ 📊 QUICK ACCESS (Bottom Action Bar) ───────────────────────────────────────  │
│                                                                             │
│ [🗂️ ALL VMs] [💾 Storage] [🌐 Platforms] [📊 Analytics] [⚙️ Settings]      │
│ └─ Secondary functions accessible in one click ────────────────────────────  │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 🎯 COCKPIT DESIGN PRINCIPLES

### **Aviation UX Applied to Data Protection:**

**1. Critical Controls Always Visible**
```
🚀 BACKUP NOW    ← Most common operation (always visible)
🔄 RESTORE       ← Emergency operation (always visible)
🌉 REPLICATE     ← Premium feature (always visible)
⏸️ PAUSE ALL     ← Safety control (always visible)
⚡ STOP ALL      ← Emergency stop (red, prominent, always visible)
```

**2. Instrument Panel (Live Status)**
```
🟢 Fleet Status     ← How many VMs protected/unprotected
🟢 System Load      ← Current throughput and performance  
🟡 Resource Health  ← Storage/network/agent status
🟢 Connectivity     ← Platform and agent connection status
```

**3. Context-Aware Operation**
```
Hover "BACKUP NOW":
┌─ READY FOR BACKUP ─────────────┐
│ 🖥️  pgtest2 (VMware)           │
│ 🖥️  database-dev (VMware)      │ 
│ ☁️  web-01 (CloudStack)        │
│ [Select VM] [Backup All Ready] │
└─────────────────────────────────┘

Hover "RESTORE":
┌─ RECENT BACKUPS ───────────────┐
│ 📦 database-prod (2h ago)      │
│ 📦 web-server (4h ago)         │
│ 📦 file-server (6h ago)        │
│ [Browse All] [Emergency Restore] │
└─────────────────────────────────┘
```

**4. No Deep Navigation**
- Primary operations: 0 clicks (always visible)
- Secondary operations: 1 click maximum
- Settings/config: 1 click from bottom bar
- Everything else: Search function

---

## 🎛️ PROTECTION FLOWS - DETAILED WIREFRAME

### **Flow Management Area (Main Content)**

```
┌─ PROTECTION FLOWS COCKPIT ──────────────────────────────────────────────────┐
│                                                                             │
│ 🚀 START FLOW: [🖥️ pgtest2 ▼] [📥 Backup ▼] [💾 Local ▼] [🚀 GO]          │
│ └─ One-line flow creation (VM + Operation + Destination) ───────────────────  │
│                                                                             │
│ 🎛️  ACTIVE OPERATIONS (All visible, no scrolling) ─────────────────────────  │
│                                                                             │
│ ┌─ FLOW 1: BACKUP ───────────────────────────────────────────────────────┐ │
│ │ 📥 database-prod-01 → Local SSD                        🟢 RUNNING      │ │
│ │                                                                         │ │
│ │ ████████████████████▓▓▓▓▓ 83%    2.1 GiB/s    ETA: 4m 23s            │ │
│ │                                                                         │ │
│ │ [⏸️ Pause] [🔍 Details] [⏹️ Stop] [📊 Telemetry]                        │ │
│ └─────────────────────────────────────────────────────────────────────────┘ │
│                                                                             │
│ ┌─ FLOW 2: REPLICATION ──────────────────────────────────────────────────┐ │
│ │ 🌉 exchange-server → CloudStack                         🟢 SYNCING      │ │
│ │                                                                         │ │
│ │ ████████████████████████▓ 96%    1.8 GiB/s    ETA: 45s               │ │
│ │ Incremental sync (2.8GB changed blocks via CBT)                       │ │
│ │                                                                         │ │
│ │ [⏸️ Pause] [🔍 Details] [🧪 Test Failover] [🚀 Live Failover]          │ │
│ └─────────────────────────────────────────────────────────────────────────┘ │
│                                                                             │
│ ┌─ FLOW 3: BACKUP ───────────────────────────────────────────────────────┐ │
│ │ 📥 file-server-02 → AWS S3                              🟡 THROTTLED   │ │
│ │                                                                         │ │
│ │ ██████▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓ 34%    1.2 GiB/s    ETA: 18m 45s               │ │
│ │ S3 rate limiting detected - reducing speed to stay within quota       │ │
│ │                                                                         │ │
│ │ [▶️ Resume] [🔍 Details] [⚙️ S3 Config] [⏹️ Stop]                       │ │
│ └─────────────────────────────────────────────────────────────────────────┘ │
│                                                                             │
│ ⏳ QUEUE & SCHEDULE (Immediate Visibility) ────────────────────────────────  │
│                                                                             │
│ Next: app-server-02 → Local (18:00 UTC) [▶️ Start Now] [⏰ Reschedule]     │
│ Then: backup-test-vm → S3 (20:00 UTC)   [📋 Queue Mgmt] [⚙️ Policies]     │
│                                                                             │
│ 🔥 EMERGENCY ACCESS (Always Visible) ──────────────────────────────────────  │
│                                                                             │
│ [⚡ STOP ALL FLOWS] [🚨 DISASTER MODE] [📞 SUPPORT] [📋 INCIDENT LOG]      │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 🎯 COCKPIT INTERACTION PATTERNS

### **"Zero-Click" Information**
```
Always Visible (No Interaction Required):
├─ System throughput: 3.2 GiB/s
├─ Protected fleet: 247 VMs  
├─ Storage health: 89% used
├─ Agent status: 4/4 online
├─ Active flows: 4 running
├─ Queue status: 2 pending
├─ Failure alerts: 0 critical
└─ Next scheduled: 18:00 UTC
```

### **"One-Click" Operations**
```
Primary Actions (Single Click):
├─ 🚀 BACKUP NOW → Hover shows ready VMs → Click to start
├─ 🔄 RESTORE → Hover shows recent backups → Click to start
├─ 🌉 REPLICATE → Hover shows replication candidates → Click to start
├─ ⏸️ PAUSE ALL → Immediate pause (confirmation for safety)
└─ ⚡ STOP ALL → Emergency stop (red button, confirmation required)

Secondary Access (Single Click):
├─ 🗂️ ALL VMs → Complete VM inventory 
├─ 💾 Storage → Repository management
├─ 🌐 Platforms → Agent and platform status
├─ 📊 Analytics → Performance reports  
└─ ⚙️ Settings → System configuration
```

### **"Context-Aware" Smart Interface**

**Smart VM Selection (Hover Preview):**
```
Hover [🚀 BACKUP NOW]:
┌─ READY FOR BACKUP ─────────────────────┐
│ 🖥️  pgtest2        Last: 6h ago  Ready │
│ 🖥️  database-dev   Last: 12h ago Ready │
│ ☁️  web-cluster-01 Last: 24h ago Ready │
│ 🏢 file-server-01  Last: 3h ago  Ready │
│                                         │
│ [BACKUP SELECTED] [BACKUP ALL READY]   │
└─────────────────────────────────────────┘
```

**Smart Restore Selection (Hover Preview):**
```
Hover [🔄 RESTORE]:
┌─ RECENT BACKUPS ───────────────────────┐
│ 📦 database-prod   2h ago   45GB  VMware │
│ 📦 web-server-01   4h ago   12GB  S3     │
│ 📦 exchange-srv    6h ago   89GB  Local  │
│ 📦 file-server     8h ago   156GB AWS    │
│                                          │
│ [RESTORE SELECTED] [BROWSE BACKUPS]     │
└──────────────────────────────────────────┘
```

---

## 🚀 FLOW CARD DESIGN (Aviation Cockpit Style)

### **Active Flow Card (Cockpit Instrument)**

```
┌─ FLOW INSTRUMENT ─────────────────────────────────────────────────────┐
│                                                                       │
│ 📥 DESCEND: database-prod-01 → Local SSD              🟢 OPERATIONAL   │
│ VMware • Production • 8CPU/32GB • Critical System                     │
│                                                                       │
│ ┌─ PROGRESS GAUGE ──────────┐ ┌─ TELEMETRY FEED ────────────────────┐ │
│ │                           │ │    Current: 2.1 GiB/s               │ │
│ │     83%                   │ │    Peak: 3.1 GiB/s                  │ │
│ │  ████████████████▓▓▓▓     │ │    Average: 2.0 GiB/s               │ │
│ │                           │ │    ▄▃▅▇█▇▅▃▄▃▄▅▇█▇▅ Live graph     │ │
│ │  12.3GB / 14.8GB         │ │    Efficiency: 94%                   │ │
│ │  ETA: 4m 23s             │ │    Compression: 2.1x                │ │
│ └───────────────────────────┘ └──────────────────────────────────────┘ │
│                                                                       │
│ 🎛️  COCKPIT CONTROLS (Everything Close to Hand):                      │
│ [⏸️ Pause] [🔍 Inspect] [⏹️ Stop] [📊 Telemetry] [⚙️ Config] [📋 Log] │
│                                                                       │
│ 🚨 Flow Health: 🟢 Source ✅ 🟢 Network ✅ 🟢 Storage ✅ 🟢 Target ✅   │
└───────────────────────────────────────────────────────────────────────┘
```

### **Replication Flow Card (Premium Feature)**

```
┌─ REPLICATION INSTRUMENT ──────────────────────────────────────────────┐
│                                                                       │
│ 🌉 TRANSCEND: exchange-server → CloudStack            🟢 SYNCHRONIZED  │
│ VMware → CloudStack • Mission Critical • 16CPU/64GB • $100/VM        │
│                                                                       │
│ ┌─ SYNC STATUS ─────────────────┐ ┌─ REPLICATION HEALTH ─────────────┐ │
│ │                               │ │ 🟢 CBT Tracking: ✅              │ │
│ │     96% Complete              │ │ 🟢 Target VM: Healthy           │ │
│ │  ████████████████████████▓    │ │ 🟢 Network: 1.8 GiB/s          │ │
│ │                               │ │ 🟢 RPO: 12 minutes              │ │
│ │  2.8GB / 2.9GB (CBT deltas)   │ │ 🟢 RTO: 5 minutes               │ │
│ │  Last Sync: 12m ago           │ │ Next Sync: 3m                  │ │
│ └───────────────────────────────┘ └─────────────────────────────────┘ │
│                                                                       │
│ 🎛️  REPLICATION CONTROLS:                                             │
│ [⏸️ Pause] [🔍 Inspect] [🧪 Test Failover] [🚀 Live Failover] [📊 RTO] │
│                                                                       │
│ 💰 Premium Feature: $100/VM • SLA: 15min RPO • 5min RTO              │
└───────────────────────────────────────────────────────────────────────┘
```

---

## ⚡ EMERGENCY COCKPIT MODE

### **Disaster Response Interface**

```
🚨 EMERGENCY COCKPIT ACTIVATED (Red Alert Mode)

┌─ DISASTER RESPONSE CONTROLS ────────────────────────────────────────────┐
│                                                                         │
│ ⚡ IMMEDIATE ACTIONS (Prominent, No Confirmation Needed):               │
│                                                                         │
│ [🛑 STOP ALL FLOWS]  [⏸️ PAUSE EVERYTHING]  [🔄 START MASS RESTORE]     │
│                                                                         │
│ 🚨 CRITICAL OPERATIONS (One-Click Disaster Recovery):                  │
│                                                                         │
│ [🏢 SITE FAILOVER]   [🌉 ACTIVATE DR]   [📞 CALL SUPPORT]              │
│                                                                         │
│ 🎛️  EMERGENCY TELEMETRY:                                                │
│ ┌─────────────────────────────────────────────────────────────────────┐ │
│ │ Systems: 🔴 1 Critical  🟡 3 Warnings  🟢 243 Operational          │ │
│ │ RPO Status: 🟢 All <15min  🟡 2 Degraded  🔴 1 Stale (>1h)         │ │  
│ │ Restore Capacity: 🟢 Ready  🟢 Storage OK  🟢 Target Platforms OK   │ │
│ └─────────────────────────────────────────────────────────────────────┘ │
│                                                                         │
│ [📋 INCIDENT REPORT] [📞 ESCALATE] [🔄 EXIT EMERGENCY MODE]             │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## 📱 MOBILE COCKPIT (Responsive Design)

### **Mobile Interface (Everything Still Close to Hand)**

```
┌─ MOBILE COCKPIT (480px width) ──────┐
│                                     │
│ SENDENSE 🚨2  👤  🔍               │
│                                     │
│ ⚡ FLIGHT CONTROLS ─────────────────  │
│ [🚀 BACKUP] [🔄 RESTORE] [⚡ STOP]  │
│                                     │
│ 🎛️  VITALS ─────────────────────────  │
│ 🟢 247 VMs  🟢 3.2GB/s  🟡 89%     │
│                                     │
│ 🌊 ACTIVE FLOWS ───────────────────  │
│                                     │
│ ┌─ database-prod-01 ─────────────┐  │
│ │ 📥 VMware → Local  🟢 83%      │  │
│ │ 2.1 GiB/s • 4m 23s left       │  │
│ │ [⏸️] [🔍] [⏹️]                  │  │
│ └─────────────────────────────────┘  │
│                                     │
│ ┌─ exchange-server ──────────────┐  │
│ │ 🌉 VMware → CS  🟢 96%         │  │
│ │ 1.8 GiB/s • 45s left           │  │
│ │ [⏸️] [🔍] [🧪] [🚀]             │  │
│ └─────────────────────────────────┘  │
│                                     │
│ ┌─ QUEUE (2) ────────────────────┐  │
│ │ app-server-02 (18:00)          │  │
│ │ [▶️ Start] [⏰ Reschedule]       │  │
│ └─────────────────────────────────┘  │
│                                     │
│ [🗂️VMs] [💾Store] [📊Stats] [⚙️Set] │
└─────────────────────────────────────┘
```

**Mobile Cockpit Principles:**
- Critical controls remain prominent (backup/restore/stop)
- System vitals condensed but visible
- Flow cards stacked vertically
- Actions remain accessible (no buried menus)

---

## ✅ IMPLEMENTATION PRIORITY

### **Phase 1: Core Cockpit (Week 1)**

**Essential "Close to Hand" Components:**
```typescript
// Immediate control bar (always visible)
<ImmediateControls>
  <BackupNow />      // 🚀 BACKUP NOW
  <RestoreNow />     // 🔄 RESTORE  
  <ReplicateNow />   // 🌉 REPLICATE
  <PauseAll />       // ⏸️ PAUSE ALL
  <EmergencyStop />  // ⚡ STOP ALL
</ImmediateControls>

// System vitals instrument panel
<SystemVitals>
  <FleetStatus />     // 247 VMs protected
  <Throughput />      // 3.2 GiB/s current
  <StorageHealth />   // 89% used warning
  <AgentStatus />     // 4/4 agents online
</SystemVitals>

// Flow operations grid
<FlowGrid>
  <ActiveFlows />     // Running operations (all visible)
  <QueuedFlows />     // Scheduled operations
  <QuickStats />      // Today's summary
</FlowGrid>
```

### **Success Criteria:**
- [ ] All primary operations (backup/restore/replicate) accessible with 0 clicks
- [ ] System status visible without hovering or clicking
- [ ] Emergency controls (pause/stop) always prominent
- [ ] Flow status and progress visible at all times
- [ ] Context-aware hover previews for quick decisions
- [ ] Mobile version maintains "close to hand" principle

---

## 🚀 READY TO BUILD THE COCKPIT?

This wireframe nails the aviation principle: **everything operators need is immediately visible and within reach**. No menu hunting, no hidden controls, no multi-click workflows for critical operations.

**The result:** Backup operators can manage complex multi-platform protection workflows as efficiently as pilots manage aircraft - everything critical is within immediate reach.

Want me to create the modular job sheet to start building this cockpit interface?
