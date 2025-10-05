# Job Sheet: Sendense Cockpit GUI - Aviation-Inspired Interface

**Date Created:** 2025-10-05  
**Status:** 🔴 **READY TO START**  
**Project Goal Link:** [project-goals/phases/phase-3-gui-redesign.md → Sendense Cockpit UI]  
**Duration:** 2-3 weeks (modular phases)  
**Priority:** High (Customer-facing differentiation + revenue enablement)

---

## 🎯 PROJECT GOALS INTEGRATION (MANDATORY)

### **Specific Project Goals Reference**
**Document:** `/sendense/project-goals/phases/phase-3-gui-redesign.md`  
**Task Section:** **Sendense Cockpit UI - Aviation-Inspired Interface**  
**Business Value:** Professional interface that justifies premium pricing and enables customer adoption  
**Success Criteria:** Cockpit interface that makes Veeam look like Fisher-Price toys

**Phase 3 Objectives (From Project Goals):**
- ✅ Cockpit-style dashboard with aviation-inspired design
- ✅ Real-time telemetry for all operations (descend/ascend/transcend)
- ✅ Multi-platform orchestration (6 platforms in single pane)
- ✅ Everything within reach (minimal clicks, fast operations)
- ✅ Enterprise professional feel (impress CIOs, not just IT staff)

---

## 🛩️ COCKPIT DESIGN FOUNDATION

### **Aviation Principle: "Everything Close to Hand"**

**Core Cockpit Requirements:**
- **Zero-Click Operations:** Primary controls (backup/restore/replicate) always visible
- **Live Instrument Panel:** System vitals displayed continuously  
- **Emergency Controls:** Pause/stop operations prominent and accessible
- **Context-Aware Interface:** Hover previews eliminate menu hunting
- **Professional Aesthetics:** Dark cockpit theme with aviation-inspired gauges

**Design System:**
```css
/* Cockpit Core Colors */
--cockpit-bg: #0B0C10;        /* Deep space black */
--cockpit-surface: #121418;    /* Panel background */
--cockpit-accent: #023E8A;     /* Professional blue */
--cockpit-text: #E5EAF0;       /* High contrast text */

/* Status Indicators (Aviation) */
--status-operational: #10B981;  /* Green - normal */
--status-caution: #F59E0B;      /* Amber - attention */
--status-warning: #EF4444;      /* Red - immediate action */
```

---

## 🔗 DEPENDENCY STATUS

### **Required Before Starting:**
- ✅ **Task 5:** Backup API Endpoints (POST /backup/start, GET /backup/list, etc.)
- ✅ **Task 4:** File-Level Restore API (POST /restore/mount, GET /files, etc.)
- ✅ **Existing Infrastructure:** VMA enrollment, VM discovery, replication APIs
- ✅ **Database Schema:** All tables operational for GUI data consumption

### **Enables These Features:**
- 🎯 **Customer Self-Service:** GUI-driven backup/restore operations
- 🎯 **Enterprise Sales:** Professional interface for C-level demonstrations
- 🎯 **MSP Platform:** Multi-tenant management interface
- 🎯 **Competitive Advantage:** Best-in-class UI that shames competitors

---

## 📋 MODULAR IMPLEMENTATION PHASES

### **Phase 1: Cockpit Foundation (Week 1)**

**Goal:** Establish cockpit shell with aviation-inspired design system

**Sub-Tasks:**

- [ ] **Next.js 14 Cockpit Project Setup**
  - **Directory:** `source/current/sendense-cockpit/`
  - **Tech Stack:** Next.js 14 + TypeScript + Tailwind + shadcn/ui
  - **Theme:** Dark cockpit color palette implementation
  - **Evidence:** Working Next.js app with cockpit theme

- [ ] **Cockpit Layout Components**
  - **Components:** CockpitLayout, ImmediateControls, SystemVitals, FlowGrid
  - **Responsive:** Desktop (full cockpit) → Mobile (condensed controls)
  - **Typography:** Inter font with proper aviation-style sizing
  - **Evidence:** Layout renders with proper cockpit aesthetics

- [ ] **Design System Implementation**
  - **Colors:** Complete cockpit palette with status indicators
  - **Components:** Aviation-style gauges, progress rings, status lights
  - **Icons:** Lucide React with consistent aviation feel
  - **Evidence:** Design system documented and operational

**Files to Create:**
```
source/current/sendense-cockpit/
├── components/cockpit/
│   ├── layout.tsx              # Main cockpit shell
│   ├── immediate-controls.tsx  # Primary action buttons
│   ├── system-vitals.tsx       # Live instrument panel
│   └── flow-grid.tsx           # Operations display area
├── lib/
│   ├── cockpit-theme.ts        # Aviation color system
│   └── types.ts                # TypeScript interfaces
└── styles/
    └── cockpit.css             # Cockpit-specific styles
```

### **Phase 2: Protection Flows Interface (Week 2)**

**Goal:** Build the Protection Flows section with real-time operation management

**Sub-Tasks:**

- [ ] **Flow Card Component (Aviation Instrument Style)**
  - **Design:** Progress gauges, telemetry graphs, status indicators
  - **Types:** Backup (descend), Restore (ascend), Replication (transcend)
  - **Controls:** Pause/stop/inspect buttons on every card
  - **Evidence:** Flow cards display real operations with live progress

- [ ] **Real-Time Integration**
  - **APIs:** Connect to Task 5 backup endpoints + existing replication APIs
  - **WebSocket:** Live progress updates (10-second intervals)
  - **Auto-Refresh:** React Query for optimistic updates
  - **Evidence:** Real-time progress visible without manual refresh

- [ ] **Immediate Control Integration**
  - **Backup Now:** Integrates with POST /api/v1/backup/start
  - **Restore:** Integrates with restore mount and file APIs  
  - **Replicate:** Integrates with existing replication endpoints
  - **Evidence:** One-click operations trigger backend workflows

**Files to Create:**
```
app/flows/
├── page.tsx                    # Protection Flows main page
└── components/
    ├── flow-card.tsx               # Individual operation cards
    ├── immediate-controls.tsx      # Primary action buttons
    ├── system-vitals.tsx          # Live system metrics
    ├── flow-queue.tsx             # Scheduled operations
    └── emergency-controls.tsx      # Pause/stop all operations
```

### **Phase 3: API Integration & Real-Time Data (Week 2-3)**

**Goal:** Connect cockpit to Sendense backend with live telemetry

**Sub-Tasks:**

- [ ] **Sendense API Client**
  - **Integration:** All backup APIs (Task 5) + restore APIs (Task 4)
  - **Real-Time:** WebSocket for live progress updates
  - **Error Handling:** Robust error boundaries with retry logic
  - **Evidence:** All backend operations controllable from cockpit

- [ ] **Live Telemetry System**
  - **Metrics:** System throughput, storage health, agent status
  - **Updates:** 5-second intervals for critical data
  - **Graphs:** Real-time throughput charts (last 60 seconds)
  - **Evidence:** Live data feeds update cockpit instruments

- [ ] **Context-Aware Interface**
  - **Smart Previews:** Hover shows ready VMs, recent backups
  - **Quick Select:** Most-used VMs pre-loaded for instant access
  - **Status Intelligence:** Automatic health warnings and alerts
  - **Evidence:** Context-aware interface reduces clicks for common operations

**Files to Create:**
```
lib/
├── api-client.ts               # Complete Sendense API client
├── websocket.ts                # Real-time data streaming
├── telemetry.ts                # Live system metrics
└── context-aware.ts            # Smart preview logic
```

---

## 🎨 TECHNICAL ARCHITECTURE

### **Cockpit Component Hierarchy**
```typescript
<CockpitLayout>
  <CockpitHeader>
    <AlertStrip />      // Top alerts and notifications
    <UserControls />    // User menu and global search
  </CockpitHeader>
  
  <CockpitMain>
    <ImmediateControls>
      <BackupNow />     // 🚀 Always visible
      <RestoreNow />    // 🔄 Always visible  
      <ReplicateNow />  // 🌉 Always visible
      <EmergencyZone>   // ⏸️ ⚡ Safety controls
        <PauseAll />
        <StopAll />
      </EmergencyZone>
    </ImmediateControls>
    
    <SystemVitals>
      <FleetGauge />    // VM protection status
      <ThroughputGauge />  // Current system load
      <StorageGauge />  // Storage health
      <ConnectivityGauge /> // Agent status
    </SystemVitals>
    
    <FlowGrid>
      <ActiveFlows>
        <FlowCard />    // Individual operations
      </ActiveFlows>
      <QueuedFlows />   // Scheduled operations
      <QuickAccess />   // Secondary functions
    </FlowGrid>
  </CockpitMain>
  
  <CockpitStatus>
    <SystemHealth />    // Overall system status
    <QuickActions />    // Secondary access buttons
  </CockpitStatus>
</CockpitLayout>
```

### **Real-Time Data Integration**
```typescript
// Live telemetry hooks
const useSystemVitals = () => {
  return useQuery(['system-vitals'], () => api.telemetry.getSystemHealth(), {
    refetchInterval: 5000,  // 5-second updates
    refetchIntervalInBackground: true
  });
};

// Flow operations integration (Task 5 APIs)
const useFlowOperations = () => {
  return {
    startBackup: (vm: string, type: 'full' | 'incremental') => 
      api.backup.start({ vm_name: vm, backup_type: type }),
    listBackups: () => api.backup.list(),
    getProgress: (flowId: string) => api.progress.get(flowId)
  };
};
```

---

## 🎯 SUCCESS VALIDATION

### **Completion Criteria (All Must Pass)**
- [ ] **Zero-Click Operations:** Primary actions (backup/restore/replicate) work without menu navigation
- [ ] **Live Instrument Panel:** System vitals update in real-time (5-second intervals)
- [ ] **Emergency Controls:** Pause All and Stop All operations accessible and functional
- [ ] **Context-Aware Interface:** Hover previews provide quick VM/backup selection
- [ ] **Professional Aesthetics:** Dark cockpit theme with aviation-style indicators
- [ ] **API Integration:** All Task 5 backup endpoints working from GUI
- [ ] **Mobile Responsive:** Cockpit principles maintained on tablet/mobile

### **Testing Evidence Required**
- [ ] Start backup via "BACKUP NOW" button successfully
- [ ] Real-time progress visible on flow cards without refresh
- [ ] Emergency stop halts all operations immediately
- [ ] Mobile cockpit maintains accessibility on tablet
- [ ] Hover previews show context-appropriate options
- [ ] Aviation-style status indicators work correctly

---

## 🚨 PROJECT RULES COMPLIANCE

### **Must Follow (No Exceptions):**
- ✅ **Source Authority:** All code in `source/current/sendense-cockpit/`
- ✅ **API Integration:** Use Task 5 backup APIs without modification
- ✅ **No Simulations:** Real backend integration with live data
- ✅ **Documentation Required:** Update GUI documentation with cockpit specs
- ✅ **Professional Standards:** Enterprise-grade code quality and error handling

### **Integration Constraints:**
- **API Endpoints:** Use existing backup/restore/replication APIs
- **Real-Time Data:** WebSocket + React Query for live updates
- **Authentication:** Integrate with existing bearer token system
- **Responsive Design:** Support desktop → mobile with cockpit principles

---

## 🎯 COMPETITIVE ADVANTAGE

### **What This Cockpit Achieves:**

**1. Professional Credibility**
- Aviation-inspired interface impresses C-level executives
- Dark cockpit theme suggests mission-critical operations
- Real-time telemetry demonstrates system sophistication

**2. Operational Efficiency**
- Everything critical within 0-1 clicks (aviation standard)
- Real-time status eliminates guesswork
- Context-aware interface reduces training requirements

**3. Market Differentiation**
- No backup vendor has a true "cockpit" interface
- Makes Veeam/Nakivo look outdated and clunky
- Professional aesthetics justify premium pricing ($10-100/VM)

---

## 🚀 IMPLEMENTATION RECOMMENDATION

### **Modular Approach:**
1. **Week 1:** Cockpit foundation + design system
2. **Week 2:** Protection Flows with real-time integration
3. **Week 3:** Polish, testing, and mobile optimization

### **Start With Protection Flows:**
Focus immediately on the Protection Flows section since that's where backup and replication jobs live - this directly enables customer self-service operations.

**Ready to create the detailed job sheet and start building this cockpit?** The foundation (Tasks 1-5) is solid, the wireframes nail the aviation principle, and this interface will give Sendense the professional edge needed to destroy Veeam. 🛩️
