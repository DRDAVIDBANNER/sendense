# Phase 3: Sendense Cockpit UI - Aviation-Inspired Interface

**Phase ID:** PHASE-03  
**Status:** ✅ **COMPLETED** (100% Complete)  
**Priority:** Critical (User Experience Differentiator)  
**Timeline:** 8-10 weeks (4-6 weeks remaining)  
**Team Size:** AI Implementation (Grok Code Fast) + Review  
**Dependencies:** ✅ Phase 1 Complete (Backup infrastructure operational)

**Implementation Complete + Enhanced (October 6, 2025):**
- ✅ **All 8 Phases Complete:** Professional enterprise GUI fully implemented
- ✅ **Production Build:** Successfully compiles (15/15 pages, optimized bundles)
- ✅ **Professional Design:** Enterprise-grade interface with Sendense branding (#023E8A)
- ✅ **All 9 Pages Functional:** Dashboard, Protection Flows, Groups, Reports, Settings, Users, Support, Appliances, Repositories
- ✅ **Enhanced Features:** Appliance fleet management, flow operational controls, repository management
- ✅ **Development & Production:** Both environments operational and tested
- ✅ **Documentation Complete:** Deployment guides, component docs, troubleshooting guides
- 📊 **Status:** 100% complete and production-ready with enterprise enhancements

**Major Enhancements Completed (October 6, 2025):**

**✅ Appliance Fleet Management (IMPLEMENTED):**
- **Purpose:** Manage distributed Sendense Node Appliances (SNA) and Hub Appliances (SHA)
- **Features:** Site organization, health monitoring, approval workflow, appliance-scoped VM discovery
- **Implementation:** Complete interface with site management and health dashboard
- **Value:** Enterprise/MSP multi-site deployment management capability

**✅ Flow Control & Operations (IMPLEMENTED):**
- **Purpose:** Transform GUI from view-only to full operational control platform
- **Features:** Expanded flow modals (654-line FlowDetailsModal), backup/restore operations, failover controls
- **Controls:** Replication (replicate now, failover, test failover, rollback, cleanup), Backup (backup now, multi-step restore workflow)
- **Implementation:** Complete operational interface with conditional actions and license integration
- **Value:** Complete customer operational autonomy, professional disaster recovery capabilities

**✅ Repository Management (IMPLEMENTED):**
- **Purpose:** Complete storage infrastructure management via professional GUI
- **Features:** Multi-type repository support (Local, S3, NFS, CIFS, Azure), health monitoring, capacity tracking
- **Implementation:** Complete interface (611-line AddRepositoryModal, 184-line RepositoryCard)
- **Integration:** Ready for Phase 1 repository API endpoints
- **Value:** Complete customer self-service storage management capability

---

## 🎯 Phase Objectives

**Primary Goal:** Build a cockpit-style interface that makes Veeam and Nakivo look like Fisher-Price toys

**Success Criteria:**
- ✅ **Cockpit-style dashboard** with aviation-inspired design
- ✅ **Real-time telemetry** for all operations (descend/ascend/transcend)
- ✅ **Multi-platform orchestration** (6 platforms in single pane)
- ✅ **Everything within reach** (minimal clicks, fast operations)
- ✅ **Enterprise professional feel** (impress CIOs, not just IT staff)
- ✅ **Mobile cockpit** (responsive for tablets and phones)

**Strategic Value:**
- **Competitive Advantage:** Best-in-class GUI that shames competitors
- **Enterprise Sales:** Professional interface that justifies premium pricing
- **User Retention:** Intuitive interface reduces churn
- **Platform Differentiation:** Modern backup interface with distributed appliance management
- **Enterprise/MSP Ready:** Appliance fleet management for multi-site deployments

---

## 🏗️ Sendense Cockpit Architecture

```
┌────────────────────────────────────────────────────────────────┐
│ SENDENSE COCKPIT UI ARCHITECTURE (Aviation-Inspired)          │
├────────────────────────────────────────────────────────────────┤
│                                                                │
│  ┌──────────────────────────────────────────────────────────┐ │
│  │              COCKPIT INTERFACE LAYER                     │ │
│  │                                                          │ │
│  │  🛩️ Aviation-Inspired Design:                            │ │
│  │  • Dark cockpit theme (#0B0C10 background)              │ │
│  │  • Accent #023E8A (professional blue)                   │ │
│  │  • Real-time gauges and indicators                      │ │
│  │  • Everything within reach (minimal navigation)         │ │
│  │  • Glass morphism effects (subtle depth)                │ │
│  │                                                          │ │
│  │  🔧 Tech Stack:                                          │ │
│  │  • Next.js 14+ (App Router)                            │ │
│  │  • React 18 (Server Components)                        │ │
│  │  • Tailwind CSS + shadcn/ui                            │ │
│  │  • Framer Motion (smooth animations)                    │ │
│  │  • Recharts (real-time graphs)                          │ │
│  │  • Socket.io (live telemetry)                           │ │
│  └──────────────────────────────────────────────────────────┘ │
│                          ↕ GraphQL + WebSocket                │
│  ┌──────────────────────────────────────────────────────────┐ │
│  │              SENDENSE BACKEND API                        │ │
│  │                                                          │ │
│  │  🎯 Multi-Platform Operations:                           │ │
│  │  • descend (backup operations)                          │ │
│  │  • ascend (restore operations)                          │ │
│  │  • transcend (replication operations)                   │ │
│  │                                                          │ │
│  │  🌐 Platform Connectors:                                │ │
│  │  • VMware (✅), CloudStack (✅), Hyper-V, AWS, Azure    │ │
│  │  • Nutanix, Physical Servers                            │ │
│  │                                                          │ │
│  │  💾 Repository Management:                               │ │
│  │  • Local (QCOW2), S3, Azure Blob, Immutable            │ │
│  │  • Backup validation, Performance benchmarking         │ │
│  └──────────────────────────────────────────────────────────┘ │
└────────────────────────────────────────────────────────────────┘
```

---

## 🎨 Sendense Cockpit Design System

### **Design Philosophy: "Mission Control for Data"**

**Aviation-Inspired Professional Interface:**
- **Dark Cockpit:** Professional, high-contrast, easy on eyes during long sessions
- **Real-Time Telemetry:** Live gauges, indicators, status lights
- **Everything Within Reach:** Critical functions accessible without menu diving
- **Glass Morphism:** Subtle depth and layering (cockpit panel feel)
- **Minimal Chrome:** Focus on data, not decoration

### **Sendense Color Palette (Cockpit Theme)**

```
Core Cockpit Colors:
├─ Background: #0B0C10 (Deep space black - primary background)
├─ Surface: #121418 (Panel background - cards, sidebars)
├─ Accent: #023E8A (Professional blue - primary actions)
├─ Text: #E5EAF0 (High contrast text)
└─ Maintenance: #014C97 (Accent variant for maintenance states)

Status Indicators (Aviation-Style):
├─ Operational: #10B981 (Green - systems normal)
├─ Caution: #F59E0B (Amber - attention required) 
├─ Warning: #EF4444 (Red - immediate action)
├─ Info: #3B82F6 (Blue - informational)
└─ Offline: #64748B (Gray - inactive/disabled)

Platform Identity Colors (Subtle accents):
├─ VMware: #00A8E4 (Official VMware blue)
├─ CloudStack: #FF8C00 (Apache orange)
├─ Hyper-V: #0078D4 (Microsoft blue)
├─ AWS: #FF9900 (AWS orange)
├─ Azure: #0078D4 (Microsoft blue)
└─ Nutanix: #024DA1 (Nutanix blue)
```

### **Typography & Iconography**
- **Font:** Inter (cockpit readability, professional)
- **Icons:** Lucide React (minimal, consistent)
- **Gauges:** Custom SVG components (aviation-inspired)
- **Status Lights:** CSS-based indicators with subtle animations

---

## 🛩️ Sendense Cockpit Navigation (Aviation-Inspired)

### **Primary Navigation (Always Visible)**

```
┌─────────────────────────────────────────────────────────────┐
│ SENDENSE COCKPIT - MAIN CONSOLE                            │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  Primary Flight Controls (Top Bar):                        │
│  ├─ 🎯 COMMAND (main dashboard)                            │
│  ├─ 🌊 FLOWS (backup/replication operations)               │
│  ├─ 🗂️  ASSETS (protected VMs across all platforms)        │
│  ├─ 🔄 RECOVERY (restore and failover)                     │
│  └─ 📊 TELEMETRY (system health and performance)           │
│                                                             │
│  Secondary Controls (Context Bar):                         │
│  ├─ 💾 Repositories (storage management)                   │
│  ├─ 🌐 Platforms (source/target systems)                   │
│  ├─ 📅 Schedules (backup/replication scheduling)           │
│  ├─ 🎛️  Policies (retention, encryption, compliance)       │
│  └─ ⚙️  Systems (settings, users, licensing)               │
│                                                             │
│  Quick Actions (Always Accessible):                        │
│  ├─ ⚡ Emergency Stop (stop all operations)                │
│  ├─ 🚨 Alerts (real-time notifications)                    │
│  ├─ 🔍 Global Search (find any VM, job, or setting)        │
│  └─ 👤 User Menu (profile, logout, help)                   │
└─────────────────────────────────────────────────────────────┘
```

### **Cockpit Layout Strategy**

```
Aviation-Inspired Layout:

Primary Display (Center):     Main operational view
Instrument Panel (Left):      Key metrics, status indicators  
Navigation Panel (Right):     Context-sensitive actions
Status Bar (Bottom):          System health, connectivity, version
Alert Panel (Top-Right):      Critical notifications, warnings

Responsive Adaptation:
Desktop (>1280px):   Full cockpit (all panels visible)
Laptop (1024px):     Collapsible side panels
Tablet (768px):      Overlay panels with gesture controls
Mobile (480px):      Single-panel focus with bottom navigation
```

---

## 📋 Sendense Cockpit Implementation Plan

### **Phase 1: Foundation Setup** (Week 1)

**Goal:** Establish cockpit foundation with Sendense design system

**Sub-Tasks:**
1.1. **Next.js 14+ Cockpit Project**
   - Initialize with App Router (not Pages Router)
   - TypeScript strict mode configuration
   - TailwindCSS + shadcn/ui integration
   - Lucide React icon library
   - Inter font via Google Fonts

1.2. **Sendense Cockpit Design System**
   ```css
   /* Cockpit color palette */
   --sendense-bg: #0B0C10;      /* Deep space black */
   --sendense-surface: #121418;  /* Panel background */
   --sendense-accent: #023E8A;   /* Professional blue */
   --sendense-text: #E5EAF0;     /* High contrast */
   --sendense-maintenance: #014C97; /* Maintenance mode */
   ```

1.3. **Layout Components (Cockpit Style)**
   - `<CockpitLayout>` - Aviation-inspired layout wrapper
   - `<InstrumentPanel>` - Left metrics panel
   - `<PrimaryDisplay>` - Center main view
   - `<ContextPanel>` - Right action panel
   - `<StatusBar>` - Bottom system status
   - `<AlertStrip>` - Top notification bar

**Files to Create:**
```
sendense-cockpit/
├── components/ui/           # shadcn/ui components
├── components/cockpit/
│   ├── layout.tsx          # Main cockpit layout
│   ├── instrument-panel.tsx # Left metrics/gauges
│   ├── primary-display.tsx  # Center operational view
│   ├── context-panel.tsx    # Right context actions
│   ├── status-bar.tsx       # Bottom system status
│   └── alert-strip.tsx      # Top alert notifications
├── lib/
│   ├── api.ts              # Backend API integration
│   └── cockpit-theme.ts    # Cockpit styling system
└── styles/
    └── cockpit.css         # Cockpit-specific styles
```

**Acceptance Criteria:**
- [ ] Cockpit theme renders correctly (dark, professional)
- [ ] Aviation-inspired layout responsive
- [ ] All navigation elements accessible
- [ ] Design system documented
- [ ] Real-time data placeholders working

---

### **Phase 2: API Integration Layer** (Week 1)

**Goal:** Connect cockpit to Sendense backend with real-time data

**Sub-Tasks:**
2.1. **Sendense API Client** (adapting your original `/lib/api.ts`)
   ```typescript
   // Enhanced API client for full Sendense platform
   const sendenseAPI = {
     // Multi-platform operations
     flows: {
       descend: (vmID: string, repoID: string) => post('/api/v1/backup/start'),
       ascend: (backupID: string, targetPlatform: string) => post('/api/v1/restore/start'),
       transcend: (vmID: string, targetPlatform: string) => post('/api/v1/replication/start'),
       getActive: () => get('/api/v1/flows/active'),
       getHistory: () => get('/api/v1/flows/history'),
     },
     
     // Multi-platform assets
     assets: {
       getByPlatform: (platform: string) => get(`/api/v1/vms?platform=${platform}`),
       getAllPlatforms: () => get('/api/v1/platforms'),
       getVMDetails: (vmID: string) => get(`/api/v1/vms/${vmID}`),
     },
     
     // Repository management
     repositories: {
       getAll: () => get('/api/v1/repositories'),
       testConnection: (repoID: string) => post(`/api/v1/repositories/${repoID}/test`),
       getMetrics: (repoID: string) => get(`/api/v1/repositories/${repoID}/metrics`),
     },
     
     // Real-time telemetry
     telemetry: {
       getSystemHealth: () => get('/api/v1/telemetry/health'),
       getPerformanceMetrics: () => get('/api/v1/telemetry/performance'),
       getCaptureAgents: () => get('/api/v1/agents'),
     }
   };
   ```

2.2. **Real-Time Data Layer**
   - WebSocket integration for live flow updates
   - Server-Sent Events for telemetry streaming
   - React Query for optimistic updates
   - Error boundaries with retry logic

2.3. **TypeScript Interfaces**
   ```typescript
   interface SendenseFlow {
     id: string;
     type: 'descend' | 'ascend' | 'transcend';
     source: PlatformVM;
     target: PlatformTarget | Repository;
     status: FlowStatus;
     progress: ProgressMetrics;
     telemetry: TelemetryData;
   }
   
   interface PlatformVM {
     platform: 'vmware' | 'cloudstack' | 'hyperv' | 'aws' | 'azure' | 'nutanix';
     id: string;
     name: string;
     specs: VMSpecifications;
     health: HealthStatus;
   }
   ```

**Files to Create:**
```
lib/
├── api.ts                  # Main API client (adapted from your plan)
├── types.ts                # TypeScript interfaces
├── websocket.ts            # Real-time data streaming
├── constants.ts            # Platform colors, statuses, etc.
└── utils.ts                # Helper functions
```

**Acceptance Criteria:**
- [ ] All backend endpoints accessible via typed API
- [ ] Real-time updates working (WebSocket + SSE)
- [ ] Error handling robust
- [ ] TypeScript strict mode clean

---

### **Phase 3: COMMAND Dashboard** (Week 2)

**Goal:** Mission control center showing system-wide status

**Features (Adapting your flows dashboard to broader scope):**

3.1. **System Overview (Cockpit Style)**
```
┌─────────────────────────────────────────────────────────────┐
│ SENDENSE COMMAND CENTER                   🚨 2 ALERTS      │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  ┌─ FLEET STATUS ────────────┐  ┌─ OPERATIONS STATUS ───┐  │
│  │ 🟢 247 VMs Protected      │  │ 🟢 12 Active Flows    │  │
│  │ 🟡 3 Attention Required   │  │ ⚡ 4 Queued           │  │ │
│  │ 🔴 1 Critical Issue       │  │ ⏸️  2 Paused          │  │
│  │                           │  │ ✅ 156 Today          │  │
│  └───────────────────────────┘  └────────────────────────┘  │
│                                                             │
│  ┌─ PLATFORM DISTRIBUTION ─────────────────────────────┐    │
│  │ VMware     ████████████████ 67% (165 VMs)           │    │
│  │ CloudStack ████████ 25% (62 VMs)                   │    │
│  │ Hyper-V    ████ 12% (30 VMs)                       │    │
│  │ AWS EC2    ██ 8% (20 VMs)                          │    │ │
│  └─────────────────────────────────────────────────────┘    │
│                                                             │
│  ┌─ THROUGHPUT GAUGE ──┐  ┌─ STORAGE EFFICIENCY ──────┐    │
│  │      3.2 GiB/s       │  │ Dedup Ratio: 6.2:1      │    │
│  │   ████████████▓▓▓   │  │ Compression: 2.1:1       │    │
│  │   Current Load: 78%  │  │ Total Savings: 87%       │    │
│  └─────────────────────┘  └──────────────────────────┘    │
│                                                             │
│  [INITIATE FLOW] [EMERGENCY STOP] [VIEW TELEMETRY]         │
└─────────────────────────────────────────────────────────────┘
```

3.2. **Live Activity Feed (Real-time)**
```tsx
interface FlowCard {
  id: string;
  type: 'descend' | 'ascend' | 'transcend';
  vm: PlatformVM;
  status: 'active' | 'queued' | 'paused' | 'completed' | 'failed';
  progress: number;
  throughput: number; // GiB/s
  eta: string;
}

// Your original FlowCard concept but expanded for Sendense
<FlowCard 
  type="transcend"
  source="vmware://database-prod-01" 
  target="cloudstack://replica-db"
  status="active"
  progress={73}
  throughput={2.8}
  actions={['pause', 'inspect', 'emergency-stop']}
/>
```

**Files to Create:**
```
app/command/
├── page.tsx                # COMMAND dashboard (main cockpit)
└── components/
    ├── system-overview.tsx     # Fleet and operations status
    ├── platform-distribution.tsx # Multi-platform VM chart
    ├── throughput-gauge.tsx    # Real-time performance gauge
    ├── activity-feed.tsx       # Live operations feed
    └── flow-card.tsx          # Individual operation cards (your design)
```

---

### **Phase 4: FLOWS Console** (Week 3)

**Goal:** Real-time operation management (your core flows concept expanded)

**Features:**

4.1. **Flow Types (descend/ascend/transcend)**
```
Flow Management Console:
┌─────────────────────────────────────────────────────────────┐
│ ACTIVE FLOWS (12)              [PAUSE ALL] [EMERGENCY STOP] │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│ 📥 DESCEND: VMware → S3 Repository                         │
│ database-prod-01 ████████████████████▓▓▓▓ 83% (2.1 GiB/s) │
│ ETA: 4m 23s | 12.3GB / 14.8GB                             │
│ [⏸️ Pause] [🔍 Inspect] [⏹️ Stop]                            │
│                                                             │
│ 🌉 TRANSCEND: VMware → CloudStack                          │
│ exchange-server ██████████████▓▓▓▓▓▓▓▓ 67% (1.8 GiB/s)    │
│ ETA: 8m 12s | CBT incremental sync                        │
│ [⏸️ Pause] [🔍 Inspect] [⏹️ Stop]                            │
│                                                             │
│ 📤 ASCEND: S3 Backup → AWS EC2                             │
│ web-cluster-02 ██████████████████▓▓ 91% (Converting...)    │
│ ETA: 2m 45s | Cross-platform restore                      │
│ [⏸️ Pause] [🔍 Inspect] [⏹️ Stop]                            │
│                                                             │
│ ⏳ QUEUED FLOWS (4)                                        │ │
│ • file-server-01 (descend → Local)                        │
│ • app-server-02 (transcend → Azure)                       │ │
│ [📋 Queue Management] [⚡ Priority Override]                │
└─────────────────────────────────────────────────────────────┘
```

4.2. **Flow Inspection Modal (Your GlassyModal concept)**
```tsx
<FlowInspectionModal>
  <FlowHeader 
    type="transcend"
    source="vmware://database-prod-01"
    target="cloudstack://replica-db"
  />
  <TelemetryGraphs>
    <ThroughputGraph timeRange="60s" /> {/* Your original concept */}
    <LatencyGraph />
    <ErrorRateGraph />
  </TelemetryGraphs>
  <FlowLogs stream={true} />
  <FlowActions>
    <Button variant="destructive">Emergency Stop</Button>
    <Button variant="secondary">Pause</Button>
    <Button variant="primary">Adjust Priority</Button>
  </FlowActions>
</FlowInspectionModal>
```

**Files to Create:**
```
app/flows/
├── page.tsx                # FLOWS console (your original concept)
└── components/
    ├── flow-card.tsx          # Individual flow cards (your design)
    ├── throughput-graph.tsx   # Your original ThroughputGraph
    ├── flow-modal.tsx         # Your original GlassyModal
    ├── queue-manager.tsx      # Flow queue management
    └── emergency-controls.tsx  # Emergency stop/pause all
```

---

### **Phase 5: ASSETS Management** (Week 4-5)

**Goal:** Multi-platform VM inventory with cockpit-style interface

**Features:**

5.1. **Multi-Platform Asset Grid**
```
┌─────────────────────────────────────────────────────────────┐
│ PROTECTED ASSETS (247 VMs)         [PLATFORM ▼] [STATUS ▼] │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│ 🏢 VMware Infrastructure (165 VMs)                         │
│ ┌────────────────────────────────────────────────────┐    │
│ │ 🖥️ database-prod-01    🟢 Active  Last: 2h ago      │    │ │
│ │   8CPU | 32GB | 500GB  Backup: ✅ Replication: ✅   │    │
│ │   [Backup Now] [Restore] [Replicate]                │    │
│ │                                                      │    │
│ │ 🖥️ exchange-server     🟡 Attention  Last: 4h ago    │    │ │
│ │   16CPU | 64GB | 1TB   Backup: ⚠️ Replication: ✅    │    │
│ │   [Investigate] [Force Backup] [Settings]            │    │
│ └────────────────────────────────────────────────────┘    │
│                                                             │
│ 🌐 CloudStack Infrastructure (62 VMs)                      │
│ ┌────────────────────────────────────────────────────┐    │
│ │ 🖥️ web-cluster-01      🟢 Active  Last: 1h ago      │    │
│ │   4CPU | 16GB | 200GB  Backup: ✅ Replication: ❌   │    │ │
│ │   [Enable Replication] [Backup] [Migrate]            │    │
│ └────────────────────────────────────────────────────┘    │
│                                                             │
│ [BULK ACTIONS] [ADD PLATFORM] [IMPORT VMS]                 │
└─────────────────────────────────────────────────────────────┘
```

5.2. **Asset Health Monitoring**
```tsx
<AssetHealthPanel>
  <PlatformStatus 
    platform="vmware"
    vms={165}
    health="operational"
    lastSync="2m ago"
  />
  <BackupCoverage 
    protected={247}
    unprotected={12}
    coverage={95.4}
  />
  <ReplicationStatus
    active={23}
    healthy={21}
    degraded={2}
  />
</AssetHealthPanel>
```

**Files to Create:**
```
app/assets/
├── page.tsx                # ASSETS main page
├── [platform]/page.tsx     # Platform-specific views
└── components/
    ├── platform-grid.tsx      # Multi-platform VM grid
    ├── asset-card.tsx         # Individual VM cards
    ├── health-panel.tsx       # Asset health monitoring
    ├── platform-selector.tsx  # Platform filtering
    └── bulk-actions.tsx       # Bulk operations interface
```

---

### **Phase 6: RECOVERY Center** (Week 5-6)

**Goal:** Cross-platform restore and failover interface

**Features:**

6.1. **Recovery Mission Control**
```
┌─────────────────────────────────────────────────────────────┐
│ RECOVERY CENTER                           🚨 DISASTER MODE  │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│ ⚡ EMERGENCY ACTIONS                                        │
│ [🔥 SITE FAILOVER] [⚡ BULK RESTORE] [🔄 TEST RECOVERY]    │
│                                                             │
│ 📋 RECOVERY OPTIONS:                                        │
│                                                             │
│ ┌─ Cross-Platform Restore ─────────────────────────────┐    │
│ │ Source: VMware backup → Target: CloudStack           │    │
│ │ database-prod-01 (Oct 4, 11:00 PM backup)            │    │ │
│ │                                                       │    │
│ │ Compatibility: ✅ Supported                           │    │
│ │ Resources: ✅ Target adequate (8CPU, 32GB available)  │    │
│ │ Network: ✅ Mapped to Production VLAN               │    │
│ │ Drivers: ✅ VirtIO injection ready                   │    │
│ │                                                       │    │
│ │ Estimated Time: 12 minutes                            │    │
│ │ [🚀 START RECOVERY] [⚙️ Advanced Options]             │    │
│ └───────────────────────────────────────────────────────┘    │
│                                                             │
│ ┌─ File-Level Recovery ──────────────────────────────┐      │
│ │ Browse backup: web-server-01 (Oct 4, 2:00 AM)      │      │
│ │ 📁 /var/www/html/                                   │      │
│ │ ├─ 📄 index.php (4.2 KB) ☑                         │      │
│ │ ├─ 📄 config.php (1.8 KB) ☑                        │      │
│ │ └─ 📁 assets/                                       │      │
│ │                                                      │      │
│ │ [📥 Download Selected] [🔄 Restore to Server]       │      │
│ └──────────────────────────────────────────────────────┘      │ │
└─────────────────────────────────────────────────────────────┘
```

6.2. **Recovery Wizard (Cross-Platform)**
```tsx
<RecoveryWizard>
  <Step1_BackupSelection 
    backups={availableBackups}
    showCompatibility={true}
  />
  <Step2_TargetPlatform
    compatibleTargets={['vmware', 'cloudstack', 'aws']}
    resourceValidation={true}
  />
  <Step3_Configuration
    driverInjection={true}
    networkMapping={true}
    performanceEstimation={true}
  />
  <Step4_Execution
    realTimeProgress={true}
    stepByStep={true}
  />
</RecoveryWizard>
```

**Files to Create:**
```
app/recovery/
├── page.tsx                # RECOVERY center
├── wizard/page.tsx         # Cross-platform restore wizard
├── files/page.tsx          # File-level restore
└── components/
    ├── recovery-dashboard.tsx  # Main recovery interface
    ├── cross-platform-wizard/ # Multi-step restore wizard
    ├── file-browser.tsx       # Backup file browser
    └── emergency-controls.tsx  # Disaster response controls
```

---

### **Phase 7: TELEMETRY Monitoring** (Week 6-7) 

**Goal:** Real-time system health and performance monitoring

**Features (Expanding your reports module):**

7.1. **Live System Telemetry**
```
┌─────────────────────────────────────────────────────────────┐
│ SYSTEM TELEMETRY                        LAST UPDATE: 2.3s  │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  ┌─ CAPTURE AGENT STATUS ─────────────────────────────┐    │
│  │ VMware-ESX01    🟢 Online   3.1 GiB/s  2 active    │    │
│  │ CloudStack-01   🟢 Online   2.7 GiB/s  1 active    │    │ │
│  │ Hyper-V-01     🟡 Degraded  1.2 GiB/s  High CPU    │    │
│  │ AWS-Agent-01   🔴 Offline   0.0 GiB/s  Connection   │    │
│  └─────────────────────────────────────────────────────┘    │
│                                                             │
│  ┌─ REPOSITORY HEALTH ───────────────────────────────┐     │
│  │ Local-SSD      🟢 1.2TB / 2.0TB (60%)             │     │
│  │ AWS-S3         🟢 5.7TB / ∞ (Unlimited)           │     │
│  │ Azure-Blob     🟡 890GB / 1TB (89% - Near full)   │     │
│  │ Immutable-S3   🟢 2.3TB (WORM compliance active)  │     │
│  └─────────────────────────────────────────────────────┘     │
│                                                             │
│  ┌─ PERFORMANCE METRICS (24h) ───────────────────────┐      │
│  │  3.5GB/s ┤                                       │      │
│  │  3.0GB/s ┤ ▄▄▄▄                     ▄▄▄▄         │      │
│  │  2.5GB/s ┤     ▄▄▄▄             ▄▄▄▄    ▄▄▄      │      │
│  │  2.0GB/s ┤          ▄▄▄      ▄▄▄▄         ▄▄▄    │      │
│  │   0.0GB/s └───────────────────────────────────── │      │
│  │          00:00   06:00   12:00   18:00   00:00   │      │
│  └─────────────────────────────────────────────────────┘      │ │
└─────────────────────────────────────────────────────────────┘
```

**Files to Create:**
```
app/telemetry/
├── page.tsx                # TELEMETRY dashboard
└── components/
    ├── system-gauges.tsx      # Live system metrics
    ├── agent-status.tsx       # Capture agent monitoring
    ├── repository-health.tsx  # Storage backend status
    ├── performance-charts.tsx # Throughput/latency graphs
    └── alert-center.tsx       # Alert management
```

---

### **Phase 8: Platform Management** (Week 7-8)

**Goal:** Repository, platform, and system configuration

**Features:**

8.1. **Repository Management (Cockpit Style)**
```
┌─────────────────────────────────────────────────────────────┐
│ STORAGE REPOSITORIES                        [ADD REPOSITORY] │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│ 💾 Local-SSD-Primary                           [PRIMARY]    │
│    /var/lib/sendense/backups/                              │
│    📊 1.2TB used / 2.0TB (60%) | 47 VMs                   │
│    🟢 Healthy | Last backup: 2m ago                       │
│    [Configure] [Test] [Set Primary] [Maintenance Mode]     │
│                                                             │
│ ☁️ AWS-S3-Production                            [ACTIVE]    │
│    s3://company-backups/sendense/                          │
│    📊 5.7TB used / ∞ (Unlimited) | 23 VMs                │
│    🟢 Healthy | Immutable: ✅ Object Lock                 │
│    [Configure] [Test] [Cost Analysis] [Lifecycle]         │
│                                                             │
│ [STORAGE OPTIMIZER] [COST CALCULATOR] [BACKUP VALIDATOR]   │
└─────────────────────────────────────────────────────────────┘
```

8.2. **Platform Connection Manager**
```
┌─────────────────────────────────────────────────────────────┐
│ PLATFORM CONNECTIONS                      [ADD PLATFORM]    │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│ [VMware] vcenter.company.com                   🟢 Connected │
│          165 VMs discovered | CBT: ✅ | Last sync: 1m ago  │
│          [Test] [Rediscover] [Agent Status] [Configure]     │
│                                                             │
│ [CloudStack] cloudstack.company.com           🟢 Connected  │
│             62 VMs discovered | Agent: ✅ | Last sync: 3m │  │
│             [Test] [Deploy Agent] [KVM Hosts] [Configure]  │
│                                                             │
│ [Hyper-V] hyperv-cluster.company.com         🟡 Degraded  │
│          30 VMs discovered | RCT: ⚠️ | Last sync: 15m   │
│          [Investigate] [RCT Status] [Agent Health]         │
│                                                             │
│ [BULK DISCOVERY] [AGENT DEPLOYMENT] [HEALTH CHECK]         │
└─────────────────────────────────────────────────────────────┘
```

**Files to Create:**
```
app/platforms/
├── page.tsx                # Platform management
├── repositories/page.tsx   # Repository management  
├── settings/page.tsx       # System settings
└── components/
    ├── repository-manager.tsx  # Storage backend config
    ├── platform-connector.tsx # Platform connection setup
    ├── agent-deployer.tsx     # Capture Agent deployment
    └── system-settings.tsx    # Global configuration
```

---

## 🎯 Cockpit Component Library

### **Core Cockpit Components (Your Concepts Adapted)**

**1. FlowCard (Your Original Design Enhanced)**
```tsx
interface FlowCardProps {
  flow: SendenseFlow;
  onPause: () => void;
  onInspect: () => void;
  onStop: () => void;
}

const FlowCard: React.FC<FlowCardProps> = ({ flow }) => {
  const flowTypeConfig = {
    descend: { icon: ArrowDown, color: 'text-blue-400', label: 'BACKUP' },
    ascend: { icon: ArrowUp, color: 'text-green-400', label: 'RESTORE' },
    transcend: { icon: ArrowLeftRight, color: 'text-purple-400', label: 'REPLICATE' }
  };

  const config = flowTypeConfig[flow.type];

  return (
    <Card className="bg-sendense-surface border-sendense-accent/20">
      <CardHeader className="flex flex-row items-center space-y-0 pb-3">
        <config.icon className={`h-5 w-5 ${config.color}`} />
        <div className="ml-3 flex-1">
          <h3 className="font-medium text-sendense-text">{config.label}</h3>
          <p className="text-sm text-sendense-text/60">
            {flow.source.platform} → {flow.target.platform}
          </p>
        </div>
        <Badge variant="secondary">{flow.status}</Badge>
      </CardHeader>
      
      <CardContent>
        <div className="space-y-3">
          <ThroughputGraph 
            data={flow.telemetry.throughputHistory}
            current={flow.telemetry.currentThroughput}
          />
          
          <ProgressBar 
            value={flow.progress.percent}
            className="h-2"
          />
          
          <div className="flex justify-between text-sm text-sendense-text/60">
            <span>{flow.progress.transferred} / {flow.progress.total}</span>
            <span>ETA: {flow.progress.eta}</span>
          </div>
        </div>
      </CardContent>
      
      <CardActions>
        <Button size="sm" variant="secondary" onClick={onPause}>
          {flow.status === 'paused' ? 'Resume' : 'Pause'}
        </Button>
        <Button size="sm" variant="outline" onClick={onInspect}>
          Inspect
        </Button>
        <Button size="sm" variant="destructive" onClick={onStop}>
          Stop
        </Button>
      </CardActions>
    </Card>
  );
};
```

**2. ThroughputGraph (Your Original Concept)**
```tsx
const ThroughputGraph: React.FC<ThroughputGraphProps> = ({ data, current }) => (
  <div className="h-24">
    <ResponsiveContainer width="100%" height="100%">
      <LineChart data={data}>
        <Line 
          type="monotone" 
          dataKey="throughput" 
          stroke="#023E8A" 
          strokeWidth={2}
          dot={false}
          animationDuration={600}
          animationEasing="ease-out"
        />
        <XAxis hide />
        <YAxis hide />
      </LineChart>
    </ResponsiveContainer>
    
    <div className="flex justify-between mt-1 text-xs text-sendense-text/60">
      <span>Last 60s</span>
      <span className="font-mono">{current} GiB/s</span>
    </div>
  </div>
);
```

**3. GlassyModal (Your Original Concept)**
```tsx
const GlassyModal: React.FC<GlassyModalProps> = ({ children, isOpen, onClose }) => (
  <Dialog open={isOpen} onOpenChange={onClose}>
    <DialogContent className="
      bg-sendense-surface/80 
      backdrop-blur-xl 
      border-sendense-accent/20
      max-w-4xl
    ">
      <div className="glass-morphism">
        {children}
      </div>
    </DialogContent>
  </Dialog>
);
```

---

## 🎯 Technical Implementation (Your Foundation)

### **Your Excellent Tech Stack (Preserved)**

```json
{
  "dependencies": {
    "next": "^14.0.0",
    "react": "^18.0.0", 
    "typescript": "^5.0.0",
    "@radix-ui/react-*": "latest",
    "tailwindcss": "^3.3.0",
    "framer-motion": "^10.0.0",
    "recharts": "^2.8.0",
    "socket.io-client": "^4.7.0",
    "lucide-react": "latest"
  }
}
```

### **Real-Time Integration (Your Concept Enhanced)**

```typescript
// WebSocket integration for live telemetry
const useLiveTelemetry = () => {
  const [telemetry, setTelemetry] = useState<TelemetryData>();
  
  useEffect(() => {
    const socket = io('/ws/telemetry');
    
    socket.on('flow_progress', (data: FlowProgress) => {
      setTelemetry(prev => ({
        ...prev,
        flows: updateFlowProgress(prev.flows, data)
      }));
    });
    
    socket.on('system_health', (data: SystemHealth) => {
      setTelemetry(prev => ({
        ...prev,
        health: data
      }));
    });
    
    return () => socket.disconnect();
  }, []);
  
  return telemetry;
};

// Your auto-refresh concept for flows
const useFlowRefresh = () => {
  return useQuery(['flows'], sendenseAPI.flows.getActive, {
    refetchInterval: 10000, // 10s refresh
    refetchIntervalInBackground: true
  });
};
```

---

## 🎯 Success Metrics

### **User Experience Metrics**
- ✅ **Task completion 60% faster** than current GUI
- ✅ **Zero-training operation** (intuitive cockpit design)
- ✅ **Mobile usability >90%** (Lighthouse mobile score)
- ✅ **Enterprise satisfaction >4.5/5** (C-level approval)

### **Technical Metrics** 
- ✅ **Initial load <2 seconds** (optimized Next.js)
- ✅ **Real-time updates <500ms** latency
- ✅ **99.9% uptime** for cockpit interface
- ✅ **Cross-platform awareness** (show all 6 platforms clearly)

### **Competitive Metrics**
- ✅ **"Holy shit" demos** (prospects amazed vs Veeam)
- ✅ **UI mentioned in sales wins** (differentiating factor)
- ✅ **User retention >95%** (sticky professional interface)

---

## 💻 Development Timeline (Adapted)

| Week | Phase | Deliverable | Based On Your Plan |
|------|--------|------------|-------------------|
| **Week 1** | Foundation + API | Cockpit shell + backend integration | Your Phase 1 + 2 |
| **Week 2** | COMMAND Center | Mission control dashboard | Your Phase 3 (flows) |
| **Week 3** | FLOWS Console | Real-time operation management | Your Phase 3 enhanced |
| **Week 4** | ASSETS Management | Multi-platform VM inventory | Your Phase 5 expanded |
| **Week 5** | RECOVERY Center | Cross-platform restore interface | New (restore wizards) |
| **Week 6** | TELEMETRY | Real-time monitoring | Your Phase 4 enhanced |
| **Week 7** | Platform Management | Repos, platforms, settings | Your Phase 6 expanded |
| **Week 8** | Polish & Production | Animations, packaging, QA | Your Phase 7 + 8 |

---

## 🚀 Key Adaptations from Your Plan

### **What I Preserved (Your Excellence)**
- ✅ **Next.js 14 + TypeScript** (solid foundation)
- ✅ **Cockpit theme concept** (aviation-inspired) 
- ✅ **FlowCard design** (perfect for operations)
- ✅ **ThroughputGraph** (real-time telemetry)
- ✅ **GlassyModal** (professional inspection modals)
- ✅ **8-phase timeline** (well-structured approach)
- ✅ **Real-time updates** (WebSocket + polling)

### **What I Expanded (For Full Platform)**
- 🔥 **Navigation:** 5 primary + 5 secondary (vs 4 simple pages)
- 🔥 **Multi-Platform:** 6 platforms (vs 2 in original)
- 🔥 **Operation Types:** descend/ascend/transcend (vs simple flows)
- 🔥 **Cross-Platform:** Restore wizards, compatibility matrices
- 🔥 **Enterprise:** MSP features, compliance, validation
- 🔥 **Scope:** Complete backup platform (vs migration tool)

### **Your Foundation + Our Vision = Killer Combination**

Your technical choices are spot-on:
- **Next.js 14:** Perfect for our real-time needs
- **Cockpit theme:** Professional, aviation-inspired (makes competitors look amateur)
- **shadcn/ui:** Consistent, accessible components
- **Real-time telemetry:** Critical for backup/replication monitoring

Plus our expanded scope:
- **Multi-platform orchestration:** Manage VMware + CloudStack + Hyper-V + AWS + Azure + Nutanix
- **Three operation types:** descend (backup), ascend (restore), transcend (replication)
- **Enterprise features:** Cross-platform restore, application-aware recovery, compliance
- **MSP capabilities:** Multi-tenant, white-label, billing integration

---

## 🎯 Ready to Build

**Your plan adapted for full Sendense platform:**
- ✅ **Foundation solid** (Next.js 14, cockpit theme, real-time)
- ✅ **Navigation expanded** (5 primary, 5 secondary sections)
- ✅ **Feature scope complete** (backup/restore/replication/MSP)
- ✅ **Timeline realistic** (8 weeks for full cockpit)

**Next step:** Start implementing Phase 1 (Foundation) with your excellent tech choices, expanded for our multi-platform architecture.

Want to **start building the cockpit**, or want to **refine any part** of this adapted plan first?

---

**Phase Owner:** Frontend Engineering Team (Following Your Cockpit Vision)  
**Last Updated:** October 4, 2025  
**Status:** 🔴 Ready to Start - Cockpit Architecture Defined