# VM-Centric GUI Redesign Plan

## 🎯 **Overview**

Transform the current discovery-focused GUI into a VM-centric migration management interface that leverages the new VM Context API for comprehensive VM lifecycle management.

## 📊 **Current State Analysis**

### **Current GUI Structure** (Discovery-Focused)
```
┌─────────────────────────────────────────┐
│                Header                   │
├─────────────────────────────────────────┤
│  VM Discovery    Active VMs   Jobs      │ <- Summary Cards
├─────────────────────────────────────────┤
│                                         │
│        Large VM Discovery Table         │ <- Primary Focus
│     (95 VMs with Migrate buttons)       │
│                                         │
├─────────────────────────────────────────┤
│    Migrations(0) | Failovers(0) | Logs  │ <- Bottom Tabs
└─────────────────────────────────────────┘
```

### **Issues with Current Design**
- **Space Inefficient**: Large table dominates screen for one-time discovery
- **Discovery-Centric**: Everything starts from discovery rather than VM management
- **Limited Context**: No historical view or VM-specific context
- **Scattered Actions**: Migrate, failover, cleanup actions separated
- **No Job History**: Limited visibility into VM migration history

## 🎨 **Target VM-Centric Design**

### **New Interface Layout** (Inspired by Reavyr)
```
┌─ Left Navigation ─────────┬─ Main Content Area ──────────────────┬─ Right Panel ─────┐
│ 🏠 Dashboard              │ 📊 Virtual Machines Table            │ 🔍 VM: pgtest1    │
│ 🔍 Discovery              │ ┌─────────────────────────────────────┐ │ Status: ● Rep     │
│ 💻 Virtual Machines       │ │ VM Name   │ Status  │ Jobs │ Action│ │ 49.3% @ 11.46Mbps │
│ 📋 Replication Jobs       │ │ pgtest1   │ ● Rep   │  2   │ [...]│ │                   │
│ 🔄 Failover               │ │ pgtest2   │ ● Rep   │  1   │ [...]│ │ 📋 Recent Jobs    │
│ 🌐 Network Mapping        │ │ pgtest3   │ ○ Ready │  0   │ [...]│ │ ● job-xxx Rep     │
│ 📝 Logs                   │ └─────────────────────────────────────┘ │ ✅ job-yyy Done   │
│ ⚙️ Settings               │                                       │ ❌ job-zzz Fail   │
│                           │ 📄 pgtest1 Detail View               │                   │
│                           │ ┌─────────────────────────────────────┐ │ 🔧 Quick Actions  │
│                           │ │ Overview│Jobs│Network│Details│CBT  │ │ [🗺️ Map Network] │
│                           │ │ ┌─Current Job─┬─VM Specs─┬─Disks─┐ │ │ [🚀 Replicate]   │
│                           │ │ │●Replicating │2CPU 8GB  │102GB  │ │ │ [⚡ Live Fail]   │
│                           │ │ │49.3% @11Mbps│Windows   │vSAN   │ │ │ [🧪 Test Fail]   │
│                           │ │ │ETA: 27min   │PoweredOn │VLAN253│ │ │ [🧹 Cleanup]     │
│                           │ └─────────────────────────────────────┘ │                   │
└───────────────────────────┴───────────────────────────────────────┴───────────────────┘
```

## 🏗️ **Component Architecture** (Reavyr-Inspired)

### **1. Left Navigation Panel**
```typescript
interface LeftNavigation {
  activeSection: 'dashboard' | 'discovery' | 'vms' | 'jobs' | 'failover' | 'network' | 'logs' | 'settings';
  onSectionChange: (section: string) => void;
}

// Navigation Sections:
- 🏠 Dashboard: Overview and summaries
- 🔍 Discovery: VM discovery from vCenter  
- 💻 Virtual Machines: VM management (primary)
- 📋 Replication Jobs: Job-centric view
- 🔄 Failover: Failover management
- 🌐 Network Mapping: Network configuration
- 📝 Logs: System logs and troubleshooting
- ⚙️ Settings: Configuration
```

### **2. Main Content Area**
```typescript
interface MainContentArea {
  section: string;
  selectedVM?: string;
  onVMSelect: (vmName: string) => void;
}

// Content Variations:
- VM Table View: List all VMs with status/actions
- VM Detail View: Tabbed interface (Overview|Jobs|Network|Details|CBT)
- Job List View: All replication jobs
- Dashboard View: System overview
```

### **3. Right Panel (Context Panel)**
```typescript
interface RightContextPanel {
  selectedVM: string | null;
  vmContext: VMContextDetails;
  quickActions: boolean;
}

// Panel Sections:
- VM Quick Info: Status, progress, ETA
- Recent Jobs: Job history with status
- Quick Actions: Context-aware action buttons
- System Status: Active jobs count, health
```

### **4. VM Detail Tabs (Main Area)**
```typescript
interface VMDetailTabs {
  activeTab: 'overview' | 'jobs' | 'network' | 'details' | 'cbt';
  vmContext: VMContextDetails;
}

// Tab Content:
- Overview: Current job, progress, quick stats
- Jobs: Complete job history with details
- Network: Network mapping configuration
- Details: VM specifications, disks, config
- CBT: Change block tracking history
```

## 🔄 **User Workflow Redesign** (Reavyr Navigation Pattern)

### **Primary Workflow: VM Management**
```
1. Navigation: 💻 Virtual Machines (Primary Section)
   └── View: VM table with status, job count, actions
   └── Select: Click VM row → Right panel updates + Detail view

2. VM Context (Right Panel Always Visible)
   └── Quick Info: Current status, progress, ETA
   └── Recent Jobs: Last 3-4 jobs with status
   └── Quick Actions: Context-aware buttons

3. VM Detail Tabs (Main Area)
   └── Overview: Live progress, current operation
   └── Jobs: Complete job history
   └── Network: Network mapping configuration  
   └── Details: VM specs, disks, configuration
   └── CBT: Change tracking history

4. Action Workflows:
   └── Network Mapping: 🌐 Network Mapping section OR VM→Network tab
   └── Start Replication: Right panel "Replicate" button
   └── Failover: 🔄 Failover section OR Right panel buttons
   └── Cleanup: Post-failover actions in context
```

### **Secondary Workflows**
```
🏠 Dashboard: System overview, active jobs, health
🔍 Discovery: One-time VM discovery, refresh operations  
📋 Replication Jobs: Job-centric view (all VMs)
🔄 Failover: Failover-specific management
📝 Logs: Troubleshooting and system logs
⚙️ Settings: System configuration
```

## 📱 **Responsive Design Strategy**

### **Desktop (≥1200px)**
- Full sidebar + main panel layout
- All components visible simultaneously
- Optimal for operations monitoring

### **Tablet (768px - 1199px)**
- Collapsible sidebar
- Main panel adapts to available space
- Touch-friendly action buttons

### **Mobile (≤767px)**
- Bottom navigation for VM selection
- Full-screen VM context view
- Swipe between panels

## 🔌 **API Integration Plan**

### **Current APIs** (Keep)
```typescript
// VMA Discovery (via tunnel)
POST /api/discover → VMA /api/v1/discover

// Job Management (existing)
POST /api/replicate → OMA /api/v1/replications
GET /api/jobs → OMA /api/v1/replications
```

### **New VM Context APIs** (Implement)
```typescript
// VM Context Management
GET /api/vm-contexts → OMA /api/v1/vm-contexts
GET /api/vm-contexts/:name → OMA /api/v1/vm-contexts/:name

// Network Mapping
POST /api/network-mapping → OMA /api/v1/network-mappings
GET /api/network-mapping/:vm → OMA /api/v1/network-mappings/:vm

// Failover Management
POST /api/failover/live → OMA /api/v1/failover/live
POST /api/failover/test → OMA /api/v1/failover/test
DELETE /api/failover/test/:job → OMA /api/v1/failover/test/:job
```

## 🎯 **Implementation Phases**

### **Phase 1: VM Context Integration** (Week 1)
**Goal**: Replace discovery-focused with VM-centric data flow

**Tasks**:
- [ ] Create `VMContextSidebar` component
- [ ] Implement VM Context API integration
- [ ] Replace main dashboard with VM context view
- [ ] Add real-time progress updates

**Files to Modify**:
- `src/app/page.tsx` - Main dashboard redesign
- `src/app/api/vm-contexts/route.ts` - New API route
- `src/components/VMContextSidebar.tsx` - New component
- `src/components/VMContextPanel.tsx` - New component

### **Phase 2: Action Integration** (Week 2)
**Goal**: Integrate all VM actions in context

**Tasks**:
- [ ] Add network mapping modal
- [ ] Implement replication start from context
- [ ] Add failover action buttons
- [ ] Integrate cleanup workflows

**Files to Create**:
- `src/components/NetworkMappingModal.tsx`
- `src/components/ReplicationControls.tsx`
- `src/components/FailoverControls.tsx`
- `src/app/api/network-mapping/route.ts`

### **Phase 3: Historical View** (Week 3)
**Goal**: Rich historical and detailed views

**Tasks**:
- [ ] Job history panel with timeline
- [ ] VM details with specifications
- [ ] CBT history visualization
- [ ] Error history and troubleshooting

**Files to Create**:
- `src/components/JobHistoryPanel.tsx`
- `src/components/VMDetailsPanel.tsx`
- `src/components/CBTHistoryChart.tsx`

### **Phase 4: Polish & Performance** (Week 4)
**Goal**: Production-ready interface

**Tasks**:
- [ ] Responsive design implementation
- [ ] Performance optimization
- [ ] Real-time WebSocket integration
- [ ] Advanced error handling

## 📊 **Data Flow Architecture**

### **State Management**
```typescript
interface AppState {
  // Discovery state (minimal)
  discoveredVMs: string[];
  discoveryLoading: boolean;
  
  // VM Context state (primary)
  selectedVM: string | null;
  vmContexts: Record<string, VMContextDetails>;
  
  // UI state
  sidebarCollapsed: boolean;
  activePanel: 'progress' | 'history' | 'details';
  
  // Real-time state
  progressUpdates: Record<string, ProgressUpdate>;
  
  // Action state
  networkMapping: Record<string, NetworkMapping>;
  failoverStates: Record<string, FailoverState>;
}
```

### **API Polling Strategy**
```typescript
// Real-time Updates (5-second intervals)
- VM Context progress for active jobs
- Job status for running operations
- Failover progress for active failovers

// Periodic Updates (30-second intervals)  
- VM Context summary for all VMs
- System health status
- Alert notifications

// On-demand Updates
- Discovery refresh
- Network mapping updates
- Manual action triggers
```

## 🎨 **UI/UX Enhancements**

### **Visual Improvements**
- **Progress Visualization**: Rich progress bars with ETA and speed
- **Status Indicators**: Color-coded VM and job status
- **Action Buttons**: Context-aware action availability
- **Real-time Updates**: Live progress without page refresh

### **User Experience**
- **One-Click Actions**: All VM actions accessible from context
- **Contextual Help**: Inline help for complex operations
- **Error Recovery**: Clear error messages with suggested actions
- **Keyboard Shortcuts**: Power user keyboard navigation

### **Professional Polish**
- **Loading States**: Smooth loading animations
- **Confirmation Dialogs**: Safe operation confirmations
- **Success Feedback**: Clear success/failure notifications
- **Responsive Design**: Works on all device sizes

## 🔧 **Technical Requirements**

### **Frontend Stack** (Keep Current)
- **Next.js 15.4.5**: React framework
- **TypeScript**: Type safety
- **Tailwind CSS**: Styling framework  
- **Flowbite React**: UI components
- **React Icons**: Icon library

### **New Dependencies**
```json
{
  "react-query": "^3.39.0",     // API state management
  "recharts": "^2.8.0",         // Progress charts
  "date-fns": "^2.30.0",        // Date formatting
  "react-hot-toast": "^2.4.1"   // Toast notifications
}
```

### **Performance Targets**
- **Initial Load**: <2 seconds
- **VM Selection**: <500ms
- **Progress Updates**: <200ms
- **Action Response**: <1 second

## 🚀 **Migration Strategy**

### **Backward Compatibility**
- Keep existing API routes during transition
- Gradual component replacement
- Feature flag for new interface
- Rollback capability

### **Deployment Plan**
1. **Development**: New interface on separate branch
2. **Testing**: Side-by-side testing with current interface
3. **Staging**: Feature flag deployment
4. **Production**: Gradual rollout with monitoring

## 📈 **Success Metrics**

### **User Experience**
- Reduced clicks to start migration (target: 3 clicks)
- Faster job status visibility (target: immediate)
- Improved error recovery (target: self-service)

### **Operational Efficiency**
- Reduced discovery refresh frequency
- Better VM context awareness
- Improved troubleshooting capability

### **Technical Performance**
- Faster page load times
- Reduced API calls
- Better real-time updates

---

## 🎯 **Next Steps**

1. **Review & Approval**: User feedback on proposed design
2. **Architecture Validation**: Technical review of component design
3. **Timeline Confirmation**: Resource allocation and scheduling
4. **Implementation Start**: Begin Phase 1 development

**Ready to proceed with VM-centric GUI transformation! 🚀**
