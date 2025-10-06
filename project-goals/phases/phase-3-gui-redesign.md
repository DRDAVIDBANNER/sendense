# Phase 3: Sendense Professional GUI - Clean Enterprise Design

**Phase ID:** PHASE-03  
**Status:** 🟢 **READY TO IMPLEMENT**  
**Priority:** Critical (User Experience & Professional Appearance)  
**Timeline:** 4-6 weeks  
**Team Size:** 2-3 frontend developers  
**Dependencies:** Backend API functional (Phase 1 preferred)

---

## 🎯 Phase Objectives

**Primary Goal:** Build a clean, professional interface that makes competitors look outdated

**Success Criteria:**
- ✅ **Clean modern design** inspired by Reavyr's best qualities
- ✅ **Protection Flows page** matching Reavyr Catalogs layout (table + details + logs)
- ✅ **Intuitive navigation** with 7 clear menu sections
- ✅ **Real-time updates** for all protection operations
- ✅ **Professional appearance** that justifies premium pricing
- ✅ **Fully responsive** (desktop focus, mobile friendly)

**Strategic Value:**
- **Enterprise Sales:** Professional interface for CIO-level demos
- **User Retention:** Intuitive design reduces training needs
- **Competitive Edge:** Modern React-based UI vs legacy competitors
- **Platform Growth:** Modular architecture allows rapid feature addition

---

## 🎨 Sendense Design System

### **Design Philosophy: "Clean, Professional, Functional"**

**Core Principles:**
- **Dark theme by default** (modern, professional, easy on eyes)
- **Clean typography** (Inter font, clear hierarchy)
- **Consistent spacing** (Tailwind spacing scale)
- **No unnecessary decoration** (no gradients, no emojis)
- **Function over form** (but beautiful in execution)

### **Color Palette**

```css
/* Core Colors */
--sendense-bg: #0a0e17;          /* Deep background */
--sendense-surface: #12172a;     /* Card/panel background */
--sendense-accent: #023E8A;      /* Primary blue accent */
--sendense-accent-hover: #012E6A; /* Darker accent for hover */
--sendense-text: #e4e7eb;        /* Primary text */
--sendense-text-muted: #94a3b8;  /* Secondary text */
--sendense-border: #2d3748;      /* Border color */

/* Status Colors */
--sendense-success: #10b981;     /* Success/healthy */
--sendense-warning: #f59e0b;     /* Warning/attention */
--sendense-error: #ef4444;       /* Error/critical */
--sendense-info: #3b82f6;        /* Info/running */
```

### **Typography**

```
Font Family: Inter (Google Fonts)
Headings: 600-700 weight
Body: 400-500 weight
Code/Monospace: IBM Plex Mono
```

---

## 🗺️ Navigation Structure

### **Primary Menu (7 Sections)**

```
SENDENSE

├─ 📊 Dashboard        - System overview, health, realtime monitoring
├─ 🛡️ Protection Flows  - Backup/Replication Jobs (Reavyr Catalogs layout)
├─ 📁 Protection Groups - Schedules, VM groupings, assignments
├─ 📈 Report Center     - KPI reports, custom dashboards, filters
├─ ⚙️ Settings          - Sources (vCenter), Destinations (CloudStack)
├─ 👥 Users             - User/group/permissions management
└─ 🆘 Support           - Help, documentation, support access
```

### **Layout Pattern** (Consistent Across All Pages)

```
┌─────────────────────────────────────────────────────────────┐
│ [Sidebar - 256px]  │  [Main Content Area - Flex]           │
│                    │                                        │
│ Logo               │  [Page Header]                         │
│                    │  Page Title, Actions, Breadcrumbs     │
│ 📊 Dashboard       │                                        │
│ 🛡️ Protection Flows│  ────────────────────────────────────  │
│ 📁 Protection...   │                                        │
│ 📈 Report Center   │  [Page Content]                        │
│ ⚙️ Settings        │  Dynamic content based on page        │
│ 👥 Users           │                                        │
│ 🆘 Support         │                                        │
│                    │                                        │
│ [Theme Toggle]     │                                        │
└─────────────────────────────────────────────────────────────┘
```

---

## 📦 Project Structure (Modular Architecture)

### **Directory Structure**

```
sendense-gui/
├── src/
│   ├── app/                          # Next.js App Router
│   │   ├── dashboard/
│   │   │   └── page.tsx              # Dashboard page
│   │   ├── protection-flows/
│   │   │   ├── page.tsx              # Main flows page (Reavyr-style)
│   │   │   └── [flowId]/
│   │   │       └── page.tsx          # Flow details page
│   │   ├── protection-groups/
│   │   │   └── page.tsx              # Groups & schedules
│   │   ├── report-center/
│   │   │   └── page.tsx              # Reports & dashboards
│   │   ├── settings/
│   │   │   ├── page.tsx              # Settings home
│   │   │   ├── sources/
│   │   │   │   └── page.tsx          # Source configuration
│   │   │   └── destinations/
│   │   │       └── page.tsx          # Destination configuration
│   │   ├── users/
│   │   │   └── page.tsx              # User management
│   │   ├── support/
│   │   │   └── page.tsx              # Support page
│   │   ├── layout.tsx                # Root layout with sidebar
│   │   └── globals.css               # Global styles
│   │
│   ├── features/                     # Feature-based modules
│   │   ├── dashboard/
│   │   │   ├── components/
│   │   │   │   ├── SystemHealthCard.tsx
│   │   │   │   ├── MetricsGrid.tsx
│   │   │   │   └── RealtimeMonitor.tsx
│   │   │   ├── hooks/
│   │   │   │   └── useDashboardMetrics.ts
│   │   │   └── types/
│   │   │       └── index.ts
│   │   │
│   │   ├── protection-flows/         # Main feature (Reavyr-style)
│   │   │   ├── components/
│   │   │   │   ├── FlowsTable/
│   │   │   │   │   ├── index.tsx     # Main table component
│   │   │   │   │   ├── FlowRow.tsx   # Individual row
│   │   │   │   │   ├── StatusCell.tsx
│   │   │   │   │   └── ActionsDropdown.tsx
│   │   │   │   ├── FlowDetailsPanel/
│   │   │   │   │   ├── index.tsx     # Details panel
│   │   │   │   │   ├── OverviewTab.tsx
│   │   │   │   │   ├── VolumesTab.tsx
│   │   │   │   │   └── HistoryTab.tsx
│   │   │   │   ├── JobLogPanel/
│   │   │   │   │   ├── index.tsx     # Log panel (right side)
│   │   │   │   │   ├── LogViewer.tsx
│   │   │   │   │   └── LogFilters.tsx
│   │   │   │   └── modals/
│   │   │   │       ├── CreateFlowModal.tsx
│   │   │   │       ├── EditFlowModal.tsx
│   │   │   │       └── DeleteConfirmModal.tsx
│   │   │   ├── hooks/
│   │   │   │   ├── useProtectionFlows.ts
│   │   │   │   ├── useFlowActions.ts
│   │   │   │   └── useJobLogs.ts
│   │   │   ├── stores/
│   │   │   │   └── flowsStore.ts
│   │   │   └── types/
│   │   │       └── index.ts
│   │   │
│   │   ├── protection-groups/
│   │   │   ├── components/
│   │   │   ├── hooks/
│   │   │   └── types/
│   │   │
│   │   ├── reports/
│   │   │   ├── components/
│   │   │   ├── hooks/
│   │   │   └── types/
│   │   │
│   │   └── settings/
│   │       ├── components/
│   │       ├── hooks/
│   │       └── types/
│   │
│   ├── components/                   # Shared components
│   │   ├── ui/                       # shadcn components
│   │   │   ├── button.tsx
│   │   │   ├── card.tsx
│   │   │   ├── dialog.tsx
│   │   │   ├── table.tsx
│   │   │   ├── tabs.tsx
│   │   │   └── ...
│   │   ├── layout/
│   │   │   ├── Sidebar.tsx
│   │   │   ├── Header.tsx
│   │   │   └── PageHeader.tsx
│   │   └── common/
│   │       ├── StatusBadge.tsx
│   │       ├── LoadingSpinner.tsx
│   │       ├── EmptyState.tsx
│   │       └── ErrorBoundary.tsx
│   │
│   ├── lib/
│   │   ├── api/                      # API client
│   │   │   ├── client.ts
│   │   │   ├── endpoints.ts
│   │   │   └── types.ts
│   │   ├── hooks/                    # Global hooks
│   │   │   ├── useAuth.ts
│   │   │   └── useTheme.ts
│   │   ├── utils/
│   │   │   ├── cn.ts                 # Classname utility
│   │   │   ├── formatters.ts
│   │   │   └── validators.ts
│   │   └── constants/
│   │       ├── colors.ts
│   │       └── routes.ts
│   │
│   └── styles/
│       └── globals.css
│
├── public/
│   └── sendense-logo.svg
├── .env.local
├── next.config.ts
├── tailwind.config.ts
├── tsconfig.json
└── package.json
```

---

## 🛡️ Protection Flows Page (Reavyr Catalogs Layout)

### **Layout Design** (Matching Reavyr's Best Pattern)

```
┌─────────────────────────────────────────────────────────────────────┐
│ Protection Flows                    [+ Create Flow] [⟳ Refresh]    │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│ ┌────────── Flows Table ──────────┬─── Job Logs ───┐               │
│ │ Name     Type    Status  Last │ Running Job:   │               │
│ │ ───────────────────────────────│ VM-Backup-01   │               │
│ │ DB-Backup Backup  🟢 2h   │                    │               │
│ │ Web-Repl  Repl    🟢 1h   │ [===75%====]     │               │
│ │ File-Back Backup  🟡 5m   │                    │               │
│ │                           │ Logs:              │               │
│ │ [Select row for details]  │ [Log viewer here]  │               │
│ ├───────────────────────────┘                    │               │
│ │                                                 │               │
│ │ ──── Horizontal Divider (Draggable) ────       │               │
│ │                                                 │               │
│ │ ┌──── Details Panel ─────────────────┐         │               │
│ │ │ VM-Backup-01 Details               │         │               │
│ │ │                                    │         │               │
│ │ │ [Overview] [Volumes] [History]     │         │               │
│ │ │                                    │         │               │
│ │ │ Source: vcenter01/db-server        │         │               │
│ │ │ Destination: cloudstack01/backup   │         │               │
│ │ │ Schedule: Daily at 2 AM            │         │               │
│ │ │ Last Run: Success (2h ago)         │         │               │
│ │ └────────────────────────────────────┘         │               │
│ └─────────────────────────────────────────────────┘               │
└─────────────────────────────────────────────────────────────────────┘
```

### **Key Features** (Exactly Like Reavyr)

1. **Flows Table** (Top Section)
   - Sortable columns (Name, Type, Status, Last Run, Next Run)
   - Status badges (Success/Running/Warning/Error)
   - Actions dropdown per row
   - Selection highlights row and shows details below

2. **Details Panel** (Bottom Section - Draggable Up/Down)
   - **Horizontal drag divider** at top (grab and drag up/down to resize)
   - Minimum height: 100px (collapsed)
   - Maximum height: 60% of viewport
   - Default height: 400px
   - Tabs: Overview, Volumes, History
   - Shows selected flow's configuration
   - Quick actions (Edit, Delete, Run Now)
   - **Persists size** to localStorage

3. **Job Log Panel** (Right Section - Collapsible & Draggable)
   - **Pops out from right side** (like Reavyr)
   - **Chevron button** to collapse/expand panel
   - **Vertical drag divider** on left edge (grab and drag left/right to resize)
   - Minimum width: 48px (collapsed with just chevron visible)
   - Maximum width: 600px
   - Default width: 420px
   - Real-time log streaming
   - Log level filtering (All, Info, Warning, Error)
   - Auto-scroll option
   - **Persists state** (collapsed/expanded) to localStorage
   - **Persists width** to localStorage

### **Component Breakdown**

```tsx
// Main page structure (Reavyr-style with draggable panels)
<ProtectionFlowsPage>
  <PageHeader 
    title="Protection Flows"
    actions={<CreateFlowButton />}
  />
  
  <div className="flex h-full overflow-hidden">
    {/* Left side: Table + Details */}
    <div className="flex-1 flex flex-col min-h-0">
      {/* Flows Table - grows to fill available space */}
      <div 
        className="overflow-auto"
        style={{ height: `calc(100% - ${detailsPanelHeight}px - 4px)` }}
      >
        <FlowsTable 
          flows={flows}
          onSelectFlow={setSelectedFlow}
          selectedFlowId={selectedFlow?.id}
        />
      </div>
      
      {/* Horizontal Draggable Divider */}
      <div
        className="h-1 bg-border cursor-row-resize hover:bg-primary transition-colors"
        onMouseDown={handleHorizontalDragStart}
      />
      
      {/* Details Panel - resizable height */}
      <div 
        className="overflow-auto"
        style={{ height: detailsPanelHeight }}
      >
        <FlowDetailsPanel flow={selectedFlow} />
      </div>
    </div>
    
    {/* Vertical Draggable Divider (only visible when panel expanded) */}
    {isJobPanelExpanded && (
      <div
        className="w-1 bg-border cursor-col-resize hover:bg-primary transition-colors"
        onMouseDown={handleVerticalDragStart}
      />
    )}
    
    {/* Right side: Job Logs Panel - collapsible */}
    <div 
      className="relative"
      style={{ width: isJobPanelExpanded ? jobPanelWidth : 48 }}
    >
      {/* Collapse/Expand Toggle Button */}
      <button
        onClick={() => setIsJobPanelExpanded(!isJobPanelExpanded)}
        className="absolute left-0 top-4 z-10 p-2 bg-surface border rounded-r"
      >
        {isJobPanelExpanded ? <ChevronRight /> : <ChevronLeft />}
      </button>
      
      {/* Job Log Panel Content */}
      {isJobPanelExpanded && (
        <JobLogPanel 
          jobId={activeJobId}
          width={jobPanelWidth}
        />
      )}
    </div>
  </div>
</ProtectionFlowsPage>
```

### **Draggable Panel Implementation**

```tsx
// State for panel sizes (persisted to localStorage)
const [detailsPanelHeight, setDetailsPanelHeight] = useState(
  () => parseInt(localStorage.getItem('detailsPanelHeight') || '400')
);
const [jobPanelWidth, setJobPanelWidth] = useState(
  () => parseInt(localStorage.getItem('jobPanelWidth') || '420')
);
const [isJobPanelExpanded, setIsJobPanelExpanded] = useState(
  () => localStorage.getItem('jobPanelExpanded') === 'true'
);

// Horizontal drag handler (for details panel up/down)
const handleHorizontalDragStart = (e: React.MouseEvent) => {
  const startY = e.clientY;
  const startHeight = detailsPanelHeight;
  
  const handleMouseMove = (moveEvent: MouseEvent) => {
    const deltaY = startY - moveEvent.clientY; // Inverted because we're dragging up
    const newHeight = Math.max(100, Math.min(
      window.innerHeight * 0.6,
      startHeight + deltaY
    ));
    setDetailsPanelHeight(newHeight);
  };
  
  const handleMouseUp = () => {
    localStorage.setItem('detailsPanelHeight', detailsPanelHeight.toString());
    document.removeEventListener('mousemove', handleMouseMove);
    document.removeEventListener('mouseup', handleMouseUp);
  };
  
  document.addEventListener('mousemove', handleMouseMove);
  document.addEventListener('mouseup', handleMouseUp);
};

// Vertical drag handler (for job log panel left/right)
const handleVerticalDragStart = (e: React.MouseEvent) => {
  const startX = e.clientX;
  const startWidth = jobPanelWidth;
  
  const handleMouseMove = (moveEvent: MouseEvent) => {
    const deltaX = startX - moveEvent.clientX;
    const newWidth = Math.max(48, Math.min(600, startWidth + deltaX));
    setJobPanelWidth(newWidth);
  };
  
  const handleMouseUp = () => {
    localStorage.setItem('jobPanelWidth', jobPanelWidth.toString());
    document.removeEventListener('mousemove', handleMouseMove);
    document.removeEventListener('mouseup', handleMouseUp);
  };
  
  document.addEventListener('mousemove', handleMouseMove);
  document.addEventListener('mouseup', handleMouseUp);
};

// Persist expanded state when toggled
useEffect(() => {
  localStorage.setItem('jobPanelExpanded', isJobPanelExpanded.toString());
}, [isJobPanelExpanded]);
```

---

## 📊 Dashboard Page

### **Layout**

```
┌─────────────────────────────────────────────────────────────┐
│ Dashboard                                     Last: 5s ago   │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│ ┌─ System Health ────┬─ Active Operations ─┬─ Storage ───┐ │
│ │ 🟢 All Systems OK  │ 12 Running          │ 2.3TB / 5TB │ │
│ │ 247 VMs Protected  │ 4 Queued            │ 46% Used    │ │
│ └───────────────────┴────────────────────┴──────────────┘ │
│                                                             │
│ ┌─ Recent Activity ───────────────────────────────────────┐ │
│ │ ✅ DB-Backup-01 completed (2m ago)                      │ │
│ │ ⚡ Web-Replication running (75% complete)               │ │
│ │ ⚠️ File-Backup-03 attention needed (network issue)      │ │
│ └────────────────────────────────────────────────────────┘ │
│                                                             │
│ ┌─ Performance ─────────────────────────────────────────┐  │
│ │ [Throughput Graph - Last 24h]                         │  │
│ │                                                        │  │
│ └────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

---

## 📁 Protection Groups Page

### **Layout**

```
┌─────────────────────────────────────────────────────────────┐
│ Protection Groups                 [+ Create Group] [Import]  │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│ ┌─ Production Servers ─────────────────────────────────┐    │
│ │ 45 VMs | Schedule: Daily 2 AM | Policy: 30d retention │    │
│ │                                                       │    │
│ │ VMs: database-01, database-02, web-01, web-02...     │    │
│ │                                                       │    │
│ │ [Edit] [Add VMs] [Run Now] [View History]            │    │
│ └────────────────────────────────────────────────────────┘    │
│                                                             │
│ ┌─ Development Servers ──────────────────────────────┐      │
│ │ 23 VMs | Schedule: Weekly Sun 1 AM | Policy: 14d    │      │
│ │                                                      │      │
│ │ [Edit] [Add VMs] [View History]                     │      │
│ └────────────────────────────────────────────────────────┘      │
└─────────────────────────────────────────────────────────────┘
```

---

## 📈 Report Center Page

### **Layout**

```
┌─────────────────────────────────────────────────────────────┐
│ Report Center           [Date Range ▼] [Group ▼] [Export]  │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│ ┌─ KPI Summary ────────────────────────────────────────┐    │
│ │ Success Rate: 98.5% | Total Backups: 1,234         │    │
│ │ Avg Duration: 45m   | Storage Growth: +12% (30d)    │    │
│ └──────────────────────────────────────────────────────┘    │
│                                                             │
│ ┌─ Backup Success Trend ───────────────────────────────┐    │
│ │ [Line Graph - 30 days]                              │    │
│ └──────────────────────────────────────────────────────┘    │
│                                                             │
│ ┌─ Top 10 VMs by Size ─────────────────────────────────┐    │
│ │ [Bar Chart]                                          │    │
│ └──────────────────────────────────────────────────────┘    │
│                                                             │
│ [Save as Custom Dashboard] [Schedule Email] [Share URL]    │
└─────────────────────────────────────────────────────────────┘
```

---

## ⚙️ Settings Pages

### **Sources (vCenter Configuration)**

```
┌─────────────────────────────────────────────────────────────┐
│ Settings > Sources                         [+ Add Source]   │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│ ┌─ VMware vCenter ──────────────────────────────────────┐   │
│ │ vcenter.company.com               🟢 Connected        │   │
│ │                                                       │   │
│ │ Hostname: vcenter.company.com                         │   │
│ │ Username: backup@vsphere.local                        │   │
│ │ VMs Discovered: 165                                   │   │
│ │ Last Sync: 5m ago                                     │   │
│ │                                                       │   │
│ │ [Test Connection] [Edit] [Sync Now] [Remove]         │   │
│ └────────────────────────────────────────────────────────┘   │
│                                                             │
│ [Add vCenter] [Add Hyper-V] [Add AWS] [Add Azure]          │
└─────────────────────────────────────────────────────────────┘
```

### **Destinations (CloudStack/Storage Configuration)**

```
┌─────────────────────────────────────────────────────────────┐
│ Settings > Destinations                [+ Add Destination]  │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│ ┌─ CloudStack Primary ──────────────────────────────────┐   │
│ │ cloudstack.company.com            🟢 Connected        │   │
│ │                                                       │   │
│ │ API URL: https://cloudstack.company.com/api           │   │
│ │ Zone: zone01                                          │   │
│ │ Available Storage: 2.7TB                              │   │
│ │ Last Check: 2m ago                                    │   │
│ │                                                       │   │
│ │ [Test Connection] [Edit] [Refresh] [Remove]          │   │
│ └────────────────────────────────────────────────────────┘   │
│                                                             │
│ [Add CloudStack] [Add S3] [Add Azure Blob] [Add NFS]       │
└─────────────────────────────────────────────────────────────┘
```

---

## 👥 Users Page

### **Layout**

```
┌─────────────────────────────────────────────────────────────┐
│ Users & Permissions                      [+ Add User]       │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│ ┌─ Users ──────────────────────────────────────────────┐    │
│ │ Name          Email              Role       Status   │    │
│ │ ────────────────────────────────────────────────────│    │
│ │ John Admin    jadmin@co.com      Admin      Active  │    │
│ │ Jane Operator joperator@co.com   Operator   Active  │    │
│ │ Bob Viewer    bviewer@co.com     Viewer     Active  │    │
│ └────────────────────────────────────────────────────────┘    │
│                                                             │
│ ┌─ Roles & Permissions ─────────────────────────────────┐    │
│ │ Admin: Full access                                   │    │
│ │ Operator: Create/manage flows, view reports         │    │
│ │ Viewer: Read-only access                             │    │
│ │                                                      │    │
│ │ [Manage Roles] [Create Custom Role]                  │    │
│ └────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────┘
```

---

## 🆘 Support Page

### **Layout**

```
┌─────────────────────────────────────────────────────────────┐
│ Support                                                     │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│ ┌─ Documentation ───────────────────────────────────────┐   │
│ │ 📚 Getting Started Guide                             │   │
│ │ 📖 User Manual                                       │   │
│ │ 🔧 API Documentation                                 │   │
│ └────────────────────────────────────────────────────────┘   │
│                                                             │
│ ┌─ Contact Support ─────────────────────────────────────┐   │
│ │ Email: support@sendense.com                          │   │
│ │ Phone: +1 (555) 123-4567                             │   │
│ │ Hours: Mon-Fri 9AM-5PM EST                           │   │
│ │                                                      │   │
│ │ [Open Support Ticket] [View Ticket History]          │   │
│ └────────────────────────────────────────────────────────┘   │
│                                                             │
│ ┌─ System Information ──────────────────────────────────┐   │
│ │ Version: 1.0.0                                       │   │
│ │ Build: 2025-10-04                                    │   │
│ │ License: Enterprise (247 VMs)                        │   │
│ │                                                      │   │
│ │ [Download Logs] [System Diagnostics]                 │   │
│ └────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
```

---

## 🎯 Implementation Phases

### **Phase 1: Foundation** (Week 1)

**Goal:** Set up project structure and design system

**Tasks:**
- [ ] Initialize Next.js 15 project with App Router
- [ ] Configure TypeScript (strict mode)
- [ ] Install and configure Tailwind CSS
- [ ] Install shadcn/ui components
- [ ] Install Lucide React icons
- [ ] Set up design system (colors, typography)
- [ ] Create base layout with sidebar
- [ ] Implement theme system (dark default)

**Deliverables:**
- Working Next.js app with sidebar navigation
- All 7 menu sections accessible (placeholder pages)
- Design system documented and applied
- Dark theme implemented

---

### **Phase 2: Core Components** (Week 1)

**Goal:** Build shared component library

**Tasks:**
- [ ] Implement `<Sidebar>` component
- [ ] Implement `<PageHeader>` component
- [ ] Implement `<StatusBadge>` component
- [ ] Implement `<LoadingSpinner>` component
- [ ] Implement `<EmptyState>` component
- [ ] Implement `<ErrorBoundary>` component
- [ ] Create API client structure
- [ ] Set up React Query

**Deliverables:**
- Shared component library ready
- API client scaffolded
- Type definitions created

---

### **Phase 3: Protection Flows Page** (Week 2)

**Goal:** Build main feature matching Reavyr Catalogs layout

**Tasks:**
- [ ] Create `FlowsTable` component
  - [ ] Sortable columns
  - [ ] Status badges
  - [ ] Actions dropdown
  - [ ] Row selection
- [ ] Create `FlowDetailsPanel` component
  - [ ] Overview tab
  - [ ] Volumes tab
  - [ ] History tab
- [ ] Create `JobLogPanel` component
  - [ ] Real-time log streaming
  - [ ] Log filtering
  - [ ] Auto-scroll
- [ ] Implement resizable panels
- [ ] Create flow modals (Create, Edit, Delete)

**Deliverables:**
- Complete Protection Flows page
- Reavyr-style three-panel layout
- All CRUD operations functional

---

### **Phase 4: Dashboard & Reports** (Week 2-3)

**Goal:** Implement monitoring and reporting pages

**Tasks:**
- [ ] Build Dashboard page
  - [ ] System health cards
  - [ ] Recent activity feed
  - [ ] Performance graphs
- [ ] Build Report Center page
  - [ ] KPI summary
  - [ ] Custom date ranges
  - [ ] Export functionality

**Deliverables:**
- Dashboard with real-time metrics
- Report Center with KPIs and exports

---

### **Phase 5: Protection Groups** (Week 3)

**Goal:** Implement VM grouping and scheduling

**Tasks:**
- [ ] Build Protection Groups list
- [ ] Create group creation modal
- [ ] Implement VM assignment interface
- [ ] Add schedule configuration

**Deliverables:**
- Complete Protection Groups page
- Group and schedule management functional

---

### **Phase 6: Settings & Users** (Week 4)

**Goal:** Complete configuration pages

**Tasks:**
- [ ] Build Settings pages
  - [ ] Sources configuration
  - [ ] Destinations configuration
- [ ] Build Users page
  - [ ] User list
  - [ ] Role management
  - [ ] Permissions system

**Deliverables:**
- Complete Settings section
- Complete Users management

---

### **Phase 7: Polish & Production** (Week 5-6)

**Goal:** Finalize for production deployment

**Tasks:**
- [ ] Add loading states everywhere
- [ ] Add error handling everywhere
- [ ] Implement toast notifications
- [ ] Add keyboard shortcuts
- [ ] Responsive design testing
- [ ] Accessibility audit (ARIA labels)
- [ ] Performance optimization
- [ ] Production build testing

**Deliverables:**
- Production-ready GUI
- Documentation complete
- Deployment scripts ready

---

## 📦 Tech Stack

```json
{
  "dependencies": {
    "next": "15.4.5",
    "react": "19.1.0",
    "react-dom": "19.1.0",
    "typescript": "^5.0.0",
    "@radix-ui/react-dialog": "latest",
    "@radix-ui/react-dropdown-menu": "latest",
    "@radix-ui/react-tabs": "latest",
    "@radix-ui/react-progress": "latest",
    "@tanstack/react-query": "^5.0.0",
    "tailwindcss": "^3.4.0",
    "lucide-react": "latest",
    "recharts": "^2.8.0",
    "zustand": "^5.0.0",
    "date-fns": "^4.0.0"
  },
  "devDependencies": {
    "@types/node": "^20",
    "@types/react": "^19",
    "@types/react-dom": "^19",
    "eslint": "^9",
    "eslint-config-next": "15.4.5",
    "autoprefixer": "^10.0.0",
    "postcss": "^8.0.0"
  }
}
```

---

## 🎯 Success Metrics

**User Experience:**
- ✅ Task completion <3 clicks for common operations
- ✅ Page load time <2 seconds
- ✅ Real-time updates <500ms latency
- ✅ Zero training required (intuitive design)

**Technical:**
- ✅ TypeScript strict mode with zero `any` types
- ✅ All components <200 lines
- ✅ Lighthouse score >90
- ✅ Zero console errors/warnings

**Business:**
- ✅ Professional appearance for enterprise demos
- ✅ Feature parity with competitors
- ✅ Modular for rapid feature addition
- ✅ Maintainable by any React developer

---

## 🚀 Deployment

**Build Command:**
```bash
cd sendense-gui
npm run build
npm run start
```

**Production URL:**
```
http://10.245.246.134:3000
```

**Environment Variables:**
```
NEXT_PUBLIC_API_URL=http://10.245.246.134:8080
NEXT_PUBLIC_WS_URL=ws://10.245.246.134:8080/ws
```

---

## 📝 Notes

**Key Differences from Current GUI:**
- ✅ **Cleaner design** (no aviation metaphors, no cockpit theme)
- ✅ **Modular architecture** (feature-based, no 3,500 line files)
- ✅ **shadcn/ui** (replacing Flowbite)
- ✅ **Lucide icons** (replacing Heroicons)
- ✅ **Accent color** (#023E8A instead of mixed colors)
- ✅ **Consistent patterns** (one modal system, one table system)

**Key Similarities to Reavyr:**
- ✅ **Protection Flows layout** (matches Reavyr Catalogs exactly)
- ✅ **Three-panel design** (table + details + logs)
- ✅ **Dark theme** (professional, easy on eyes)
- ✅ **Clean typography** (no decoration, function-first)

---

**Phase Owner:** Frontend Engineering Team  
**Last Updated:** October 6, 2025  
**Status:** 🟢 **READY TO IMPLEMENT**
