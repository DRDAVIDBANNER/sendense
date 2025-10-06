# Job Sheet: Sendense Professional GUI - Clean Enterprise Design

**Project Goal Reference:** `/project-goals/phases/phase-3-gui-redesign.md` → Full GUI Implementation  
**Job Sheet Location:** `job-sheets/2025-10-06-sendense-professional-gui.md`  
**Archive Location:** `job-sheets/archive/2025/10/` (when complete)  
**Assigned:** AI Assistant (Next Session)  
**Priority:** Critical  
**Started:** 2025-10-06  
**Target Completion:** 2025-11-10 (4-6 weeks)  
**Estimated Effort:** 200-300 hours (2-3 developers)

---

## 🎯 Task Link to Project Goals

**Specific Reference:**
- **Phase:** Phase 3: GUI Redesign
- **Task:** Complete professional GUI replacement
- **Acceptance Criteria:** Clean React-based interface matching design specifications
- **Business Value:** Professional appearance for enterprise sales, improved UX reduces training costs

---

## 📋 Task Overview

**Goal:** Build a clean, professional Next.js GUI for Sendense that replaces the current interface with a modular, maintainable design inspired by enterprise backup management best practices.

**Key Requirements:**
1. **Clean design** - No aviation metaphors, no gradients, no emojis
2. **Enterprise-standard layout** - Protection Flows page uses industry-standard pattern (table + details + logs)
3. **Modular architecture** - Feature-based folders, no files >200 lines
4. **shadcn/ui + Lucide** - Consistent component library
5. **Accent color #023E8A** - Professional blue throughout
6. **Dark theme default** - Modern, professional appearance

---

## 🏗️ Project Structure

**Location:** `/home/oma_admin/sendense/deployment/sha-appliance/gui-v2/`

**Directory Structure:**
```
gui-v2/
├── src/
│   ├── app/                          # Next.js App Router
│   │   ├── dashboard/page.tsx
│   │   ├── protection-flows/page.tsx
│   │   ├── protection-groups/page.tsx
│   │   ├── report-center/page.tsx
│   │   ├── settings/
│   │   │   ├── sources/page.tsx
│   │   │   └── destinations/page.tsx
│   │   ├── users/page.tsx
│   │   ├── support/page.tsx
│   │   ├── layout.tsx
│   │   └── globals.css
│   │
│   ├── features/                     # Feature modules
│   │   ├── dashboard/
│   │   ├── protection-flows/         # Main feature
│   │   ├── protection-groups/
│   │   ├── reports/
│   │   └── settings/
│   │
│   ├── components/                   # Shared components
│   │   ├── ui/                       # shadcn components
│   │   ├── layout/
│   │   └── common/
│   │
│   └── lib/
│       ├── api/
│       ├── hooks/
│       ├── utils/
│       └── constants/
│
├── public/
├── package.json
├── next.config.ts
├── tailwind.config.ts
└── tsconfig.json
```

---

## ✅ Task Breakdown (7 Phases)

### **Phase 1: Foundation Setup** (Week 1)

- [ ] 1.1. Initialize Next.js 15 project
  ```bash
  cd /home/oma_admin/sendense/deployment/sha-appliance/
  npx create-next-app@latest gui-v2 --typescript --tailwind --app --no-src-dir
  cd gui-v2
  ```

- [ ] 1.2. Install dependencies
  ```bash
  npm install @radix-ui/react-dialog @radix-ui/react-dropdown-menu @radix-ui/react-tabs
  npm install @tanstack/react-query lucide-react zustand date-fns recharts
  npm install -D @types/node @types/react @types/react-dom
  ```

- [ ] 1.3. Set up shadcn/ui
  ```bash
  npx shadcn@latest init
  # Select: Default style, Zinc as base color, CSS variables
  ```

- [ ] 1.4. Install shadcn components
  ```bash
  npx shadcn@latest add button card dialog dropdown-menu input label
  npx shadcn@latest add progress table tabs badge
  ```

- [ ] 1.5. Configure Tailwind CSS
  - Update `tailwind.config.ts` with Sendense color palette
  - Add custom colors: `sendense-bg`, `sendense-accent` (#023E8A), etc.
  - Configure dark mode: `darkMode: 'class'`

- [ ] 1.6. Create `globals.css` with design system
  ```css
  :root {
    --sendense-bg: #0a0e17;
    --sendense-surface: #12172a;
    --sendense-accent: #023E8A;
    --sendense-text: #e4e7eb;
    /* ... rest of color palette */
  }
  ```

- [ ] 1.7. Create base layout with sidebar (`src/app/layout.tsx`)
  - Dark theme by default
  - Sidebar with 7 menu items
  - Theme toggle (optional light mode)

**Acceptance Criteria:**
- [ ] Next.js app runs on `http://10.245.246.134:3000`
- [ ] Dark theme applied correctly
- [ ] Sidebar navigation functional
- [ ] All 7 menu sections accessible (placeholder pages)

---

### **Phase 2: Shared Components** (Week 1)

- [ ] 2.1. Create `<Sidebar>` component
  - Fixed 256px width
  - Logo at top
  - 7 menu items with Lucide icons
  - Active state highlighting
  - Theme toggle at bottom

- [ ] 2.2. Create `<PageHeader>` component
  - Title + breadcrumbs
  - Action buttons (right side)
  - Consistent across all pages

- [ ] 2.3. Create `<StatusBadge>` component
  - Props: `status: 'success' | 'warning' | 'error' | 'pending' | 'running'`
  - Consistent colors
  - Icon + label

- [ ] 2.4. Create `<LoadingSpinner>` component
  - Centered spinner
  - Optional message

- [ ] 2.5. Create `<EmptyState>` component
  - Icon + title + description
  - Optional action button

- [ ] 2.6. Create `<ErrorBoundary>` component
  - Catch React errors
  - Show friendly error message
  - Reload button

- [ ] 2.7. Set up API client (`src/lib/api/client.ts`)
  ```typescript
  const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || 'http://10.245.246.134:8080';
  
  export const apiClient = {
    get: (endpoint: string) => fetch(`${API_BASE_URL}${endpoint}`),
    post: (endpoint: string, data: any) => fetch(`${API_BASE_URL}${endpoint}`, {...}),
    // ... etc
  };
  ```

- [ ] 2.8. Set up React Query provider

**Acceptance Criteria:**
- [ ] All shared components render correctly
- [ ] Sidebar navigation works
- [ ] API client structure ready
- [ ] TypeScript types defined

---

### **Phase 3: Protection Flows Page** (Week 2)

**Goal:** Build main feature matching Enterprise Catalogs layout

#### **3.1. FlowsTable Component**

- [ ] 3.1.1. Create `src/features/protection-flows/components/FlowsTable/index.tsx`
  ```tsx
  interface Flow {
    id: string;
    name: string;
    type: 'backup' | 'replication';
    status: 'success' | 'running' | 'warning' | 'error';
    lastRun: string;
    nextRun: string;
  }
  
  interface FlowsTableProps {
    flows: Flow[];
    onSelectFlow: (flow: Flow) => void;
    selectedFlowId?: string;
  }
  ```

- [ ] 3.1.2. Create `FlowRow.tsx`
  - Clickable row (highlights selected)
  - Status badge
  - Actions dropdown (Edit, Delete, Run Now)

- [ ] 3.1.3. Create `StatusCell.tsx`
  - Displays status badge
  - Shows last run time

- [ ] 3.1.4. Create `ActionsDropdown.tsx`
  - Three-dot menu
  - Actions: Edit, Delete, Run Now, View History

- [ ] 3.1.5. Add sortable columns
  - Click column header to sort
  - Ascending/descending indicators

#### **3.2. FlowDetailsPanel Component** (Draggable Up/Down - Enterprise Pattern)

- [ ] 3.2.1. Create `src/features/protection-flows/components/FlowDetailsPanel/index.tsx`
  - Shows below the table
  - **Horizontal drag divider** at top (cursor: row-resize)
  - **Draggable up/down** to resize height
  - Minimum height: 100px (collapsed)
  - Maximum height: 60% of viewport
  - Default height: 400px
  - **Persist height to localStorage**

- [ ] 3.2.2. Create tabbed interface
  - Tabs: Overview, Volumes, History
  - Use shadcn Tabs component

- [ ] 3.2.3. Create `OverviewTab.tsx`
  - Source/destination info
  - Schedule details
  - Last run status

- [ ] 3.2.4. Create `VolumesTab.tsx`
  - List of volumes in flow
  - Size, status per volume

- [ ] 3.2.5. Create `HistoryTab.tsx`
  - Last 10 runs
  - Status, duration, timestamp

- [ ] 3.2.6. Implement drag-to-resize behavior
  ```tsx
  const [height, setHeight] = useState(
    () => parseInt(localStorage.getItem('detailsPanelHeight') || '400')
  );
  
  const handleDragStart = (e: React.MouseEvent) => {
    const startY = e.clientY;
    const startHeight = height;
    
    const handleMouseMove = (moveEvent: MouseEvent) => {
      const deltaY = startY - moveEvent.clientY;
      const newHeight = Math.max(100, Math.min(
        window.innerHeight * 0.6,
        startHeight + deltaY
      ));
      setHeight(newHeight);
    };
    
    const handleMouseUp = () => {
      localStorage.setItem('detailsPanelHeight', height.toString());
      document.removeEventListener('mousemove', handleMouseMove);
      document.removeEventListener('mouseup', handleMouseUp);
    };
    
    document.addEventListener('mousemove', handleMouseMove);
    document.addEventListener('mouseup', handleMouseUp);
  };
  
  // Horizontal divider
  <div 
    className="h-1 bg-border cursor-row-resize hover:bg-primary transition-colors"
    onMouseDown={handleDragStart}
  />
  ```

#### **3.3. JobLogPanel Component** (Collapsible & Draggable - Enterprise Pattern)

- [ ] 3.3.1. Create `src/features/protection-flows/components/JobLogPanel/index.tsx`
  - **Collapsible panel** (pops out from right side)
  - **Chevron button** to toggle expand/collapse
  - **Vertical drag divider** on left edge (cursor: col-resize)
  - Minimum width: 48px (collapsed, just chevron visible)
  - Maximum width: 600px
  - Default width: 420px
  - **Persist state to localStorage** (expanded/collapsed)
  - **Persist width to localStorage**

- [ ] 3.3.2. Create `LogViewer.tsx`
  - Scrollable log output
  - Monospace font
  - Color-coded log levels (Error: red, Warning: yellow, Info: blue)

- [ ] 3.3.3. Create `LogFilters.tsx`
  - Filter by level: All, Info, Warning, Error
  - Auto-scroll toggle checkbox

- [ ] 3.3.4. Add real-time log streaming
  - WebSocket or polling every 2s
  - Auto-scroll to bottom when new logs arrive
  
- [ ] 3.3.5. Implement collapse/expand behavior
  ```tsx
  const [isExpanded, setIsExpanded] = useState(
    () => localStorage.getItem('jobPanelExpanded') === 'true'
  );
  
  // Toggle button
  <button onClick={() => {
    setIsExpanded(!isExpanded);
    localStorage.setItem('jobPanelExpanded', (!isExpanded).toString());
  }}>
    {isExpanded ? <ChevronRight /> : <ChevronLeft />}
  </button>
  ```

#### **3.4. Flow Modals**

- [ ] 3.4.1. Create `CreateFlowModal.tsx`
  - Multi-step form
  - Source selection
  - Destination selection
  - Schedule configuration

- [ ] 3.4.2. Create `EditFlowModal.tsx`
  - Pre-populated form
  - Same fields as create

- [ ] 3.4.3. Create `DeleteConfirmModal.tsx`
  - Warning message
  - Confirm button (red)

#### **3.5. Page Layout** (Enterprise Standard)

- [ ] 3.5.1. Create `src/app/protection-flows/page.tsx`
  ```tsx
  export default function ProtectionFlowsPage() {
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
    
    return (
      <div className="flex h-screen flex-col">
        <PageHeader title="Protection Flows" actions={<CreateButton />} />
        
        <div className="flex flex-1 overflow-hidden">
          {/* Left: Table + Details */}
          <div className="flex-1 flex flex-col min-h-0">
            {/* Flows Table - grows to fill available space */}
            <div 
              className="overflow-auto"
              style={{ height: `calc(100% - ${detailsPanelHeight}px - 4px)` }}
            >
              <FlowsTable />
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
              <FlowDetailsPanel />
            </div>
          </div>
          
          {/* Vertical Draggable Divider (only visible when panel expanded) */}
          {isJobPanelExpanded && (
            <div
              className="w-1 bg-border cursor-col-resize hover:bg-primary transition-colors"
              onMouseDown={handleVerticalDragStart}
            />
          )}
          
          {/* Right: Job Logs Panel - collapsible */}
          <div 
            className="relative"
            style={{ width: isJobPanelExpanded ? jobPanelWidth : 48 }}
          >
            {/* Collapse/Expand Toggle Button */}
            <button
              onClick={() => {
                setIsJobPanelExpanded(!isJobPanelExpanded);
                localStorage.setItem('jobPanelExpanded', (!isJobPanelExpanded).toString());
              }}
              className="absolute left-0 top-4 z-10 p-2 bg-surface border rounded-r"
            >
              {isJobPanelExpanded ? <ChevronRight /> : <ChevronLeft />}
            </button>
            
            {/* Job Log Panel Content */}
            {isJobPanelExpanded && <JobLogPanel />}
          </div>
        </div>
      </div>
    );
  }
  ```

- [ ] 3.5.2. Implement horizontal drag handler (details panel up/down)
  ```tsx
  const handleHorizontalDragStart = (e: React.MouseEvent) => {
    const startY = e.clientY;
    const startHeight = detailsPanelHeight;
    
    const handleMouseMove = (moveEvent: MouseEvent) => {
      const deltaY = startY - moveEvent.clientY; // Inverted
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
  ```

- [ ] 3.5.3. Implement vertical drag handler (job panel left/right)
  ```tsx
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
  ```

- [ ] 3.5.4. Verify professional enterprise UX behavior
  - Details panel drags up/down smoothly
  - Job panel collapses to 48px (just chevron)
  - Job panel expands on click
  - Job panel drags left/right when expanded
  - All sizes persist to localStorage
  - Hover on dividers shows primary color
  - Cursor changes appropriately (row-resize, col-resize)

**Acceptance Criteria:**
- [ ] Protection Flows page uses enterprise-standard layout
- [ ] Table displays flows with sorting
- [ ] Details panel shows selected flow info
- [ ] Log panel shows real-time logs
- [ ] All panels resizable
- [ ] Modals functional (create, edit, delete)

---

### **Phase 4: Dashboard Page** (Week 2-3)

- [ ] 4.1. Create `src/app/dashboard/page.tsx`

- [ ] 4.2. Create system health cards
  - All Systems OK card
  - VMs Protected count
  - Active Operations count
  - Storage usage

- [ ] 4.3. Create recent activity feed
  - Last 10 activities
  - Status icons
  - Timestamps

- [ ] 4.4. Create performance graph
  - Throughput over 24h
  - Use Recharts LineChart

- [ ] 4.5. Add auto-refresh (every 30s)

**Acceptance Criteria:**
- [ ] Dashboard shows system overview
- [ ] Real-time metrics displayed
- [ ] Performance graph renders
- [ ] Auto-refresh works

---

### **Phase 5: Protection Groups Page** (Week 3)

- [ ] 5.1. Create `src/app/protection-groups/page.tsx`

- [ ] 5.2. Create group list component
  - Card per group
  - Shows VM count, schedule, policy

- [ ] 5.3. Create group creation modal
  - Name, description
  - Schedule picker
  - Policy configuration

- [ ] 5.4. Create VM assignment interface
  - Select VMs from list
  - Add to group
  - Remove from group

- [ ] 5.5. Add schedule configuration
  - Cron expression builder
  - Or simple daily/weekly picker

**Acceptance Criteria:**
- [ ] Groups listed correctly
- [ ] Create group modal functional
- [ ] VM assignment works
- [ ] Schedule configuration intuitive

---

### **Phase 6: Report Center Page** (Week 3)

- [ ] 6.1. Create `src/app/report-center/page.tsx`

- [ ] 6.2. Create KPI summary cards
  - Success rate
  - Total backups
  - Average duration
  - Storage growth

- [ ] 6.3. Create trend charts
  - Backup success over time (Recharts)
  - Storage growth over time
  - Top 10 VMs by size

- [ ] 6.4. Add date range picker
  - Last 7 days, 30 days, custom

- [ ] 6.5. Add export functionality
  - Export as CSV
  - Export as PDF (future)

**Acceptance Criteria:**
- [ ] Report Center shows KPIs
- [ ] Charts render correctly
- [ ] Date range filtering works
- [ ] Export to CSV functional

---

### **Phase 7: Settings & Users Pages** (Week 4)

#### **7.1. Settings Pages**

- [ ] 7.1.1. Create `src/app/settings/sources/page.tsx`
  - List of vCenter connections
  - Connection status (🟢/🔴)
  - Test connection button
  - Add/Edit/Remove

- [ ] 7.1.2. Create `src/app/settings/destinations/page.tsx`
  - List of CloudStack/storage destinations
  - Connection status
  - Available storage display
  - Add/Edit/Remove

#### **7.2. Users Page**

- [ ] 7.2.1. Create `src/app/users/page.tsx`
  - User list table
  - Columns: Name, Email, Role, Status

- [ ] 7.2.2. Create user creation modal
  - Name, email, password
  - Role selection

- [ ] 7.2.3. Create role management
  - Admin, Operator, Viewer roles
  - Permission matrix

#### **7.3. Support Page**

- [ ] 7.3.1. Create `src/app/support/page.tsx`
  - Documentation links
  - Contact information
  - System information display
  - Download logs button

**Acceptance Criteria:**
- [ ] Settings pages functional
- [ ] Users page functional
- [ ] Support page complete

---

### **Phase 8: Polish & Production** (Week 5-6)

#### **8.1. Loading States**

- [ ] 8.1.1. Add loading spinners to all async operations
- [ ] 8.1.2. Add skeleton loaders for tables
- [ ] 8.1.3. Add progress bars for long operations

#### **8.2. Error Handling**

- [ ] 8.2.1. Add error boundaries to all pages
- [ ] 8.2.2. Add toast notifications (success/error)
- [ ] 8.2.3. Add retry mechanisms for failed requests

#### **8.3. Accessibility**

- [ ] 8.3.1. Add ARIA labels to all interactive elements
- [ ] 8.3.2. Ensure keyboard navigation works
- [ ] 8.3.3. Test with screen reader

#### **8.4. Performance**

- [ ] 8.4.1. Optimize bundle size
- [ ] 8.4.2. Add React.memo to expensive components
- [ ] 8.4.3. Implement code splitting

#### **8.5. Responsive Design**

- [ ] 8.5.1. Test on desktop (1920x1080, 1366x768)
- [ ] 8.5.2. Test on laptop (1280x800)
- [ ] 8.5.3. Basic tablet support (iPad)

#### **8.6. Documentation**

- [ ] 8.6.1. Create README.md with setup instructions
- [ ] 8.6.2. Document component API
- [ ] 8.6.3. Create deployment guide

#### **8.7. Production Build**

- [ ] 8.7.1. Test production build locally
  ```bash
  npm run build
  npm run start
  ```

- [ ] 8.7.2. Configure environment variables
  ```
  NEXT_PUBLIC_API_URL=http://10.245.246.134:8080
  NEXT_PUBLIC_WS_URL=ws://10.245.246.134:8080/ws
  ```

- [ ] 8.7.3. Create systemd service file
  ```ini
  [Unit]
  Description=Sendense GUI
  After=network.target

  [Service]
  Type=simple
  User=oma_admin
  WorkingDirectory=/home/oma_admin/sendense/deployment/sha-appliance/gui-v2
  ExecStart=/usr/bin/npm start
  Restart=always

  [Install]
  WantedBy=multi-user.target
  ```

- [ ] 8.7.4. Deploy to production
  ```bash
  sudo systemctl enable sendense-gui
  sudo systemctl start sendense-gui
  ```

**Acceptance Criteria:**
- [ ] All pages have loading states
- [ ] Error handling robust
- [ ] Accessibility score >90 (Lighthouse)
- [ ] Performance score >90 (Lighthouse)
- [ ] Production deployment successful
- [ ] Documentation complete

---

## 🎯 Technical Requirements

### **Code Quality**

- [ ] **TypeScript:** Strict mode, zero `any` types
- [ ] **Component Size:** No files >200 lines
- [ ] **Naming:** Consistent (PascalCase components, camelCase functions)
- [ ] **Comments:** JSDoc comments for complex functions
- [ ] **ESLint:** Zero errors, zero warnings

### **Testing** (Optional but Recommended)

- [ ] Unit tests for utilities
- [ ] Component tests for key components
- [ ] E2E tests for critical flows

### **Security**

- [ ] No secrets in code
- [ ] Environment variables for config
- [ ] XSS prevention (React handles most)
- [ ] CSRF tokens if needed

### **Performance**

- [ ] Initial load <2 seconds
- [ ] Lighthouse score >90
- [ ] Bundle size <500KB (gzipped)

---

## 📚 Documentation Updates Required

- [ ] Update `/source/current/api-documentation/API_REFERENCE.md` if API changes
- [ ] Create `GUI_COMPONENT_LIBRARY.md` documenting all shared components
- [ ] Update `CHANGELOG.md` with GUI version
- [ ] Create deployment documentation

---

## 🔗 Dependencies

**Blocks:** None (can start immediately)  
**Blocked By:** None (backend API already functional)  
**External:** None

---

## ✅ Success Criteria (Must All Be Met)

- [ ] **Functional:** All 7 pages functional with real data
- [ ] **Design:** Matches design specification (clean, professional, no gradients/emojis)
- [ ] **Layout:** Protection Flows page matches Enterprise Catalogs layout exactly
- [ ] **Performance:** Lighthouse score >90 across all pages
- [ ] **Code Quality:** TypeScript strict, no files >200 lines, modular architecture
- [ ] **Responsive:** Works on desktop (1920x1080, 1366x768, 1280x800)
- [ ] **Documentation:** README, component docs, deployment guide complete
- [ ] **Deployment:** Running in production on http://10.245.246.134:3000

---

## 📸 Evidence of Completion (Required)

- [ ] Screenshots of all 7 pages
- [ ] Lighthouse report (all scores >90)
- [ ] Component library documentation
- [ ] Production URL accessible
- [ ] Git commit with clean history

---

## 🎯 Project Goals Task Completion

**Mark this task complete in project goals when done:**
```bash
# Update project goals document
vi /home/oma_admin/sendense/project-goals/phases/phase-3-gui-redesign.md
# Change status from 🟡 PLANNED to ✅ COMPLETED
# Add completion date and evidence
```

---

## 🚀 Handoff to Next Task

**Next Task:** Phase 4 - Cross-Platform Restore  
**Dependencies Satisfied:** Professional GUI ready for restore features  
**Knowledge Transfer:** Component library reusable for future features

---

## 💡 Implementation Tips

### **Start with Phase 1 Foundation**

```bash
# Connect to Sendense server
ssh oma_admin@10.245.246.134

# Navigate to deployment directory
cd /home/oma_admin/sendense/deployment/sha-appliance/

# Create new GUI project
npx create-next-app@latest gui-v2 --typescript --tailwind --app

# Follow the task breakdown step by step
```

### **Use shadcn/ui Consistently**

```bash
# Install all needed components at once
npx shadcn@latest add button card dialog dropdown-menu input label
npx shadcn@latest add progress table tabs badge
```

### **Enterprise Layout Reference**

- Use industry-standard three-panel layout pattern
- Implement resizable dividers for professional UX
- Clean table design with proper sorting
- Tabbed details panel for organized information
- Professional log panel with filtering

### **Keep Components Small**

```
Bad:  ProtectionFlowsPage.tsx (1,500 lines)
Good: ProtectionFlowsPage.tsx (100 lines)
      + FlowsTable/index.tsx (150 lines)
      + FlowDetailsPanel/index.tsx (120 lines)
      + JobLogPanel/index.tsx (130 lines)
```

### **Test Frequently**

```bash
# Run dev server and test after each component
npm run dev
# Visit http://10.245.246.134:3000
```

---

**Job Sheet Owner:** AI Assistant (Next Session)  
**Reviewer:** Project Lead  
**Project Goals Link:** `/project-goals/phases/phase-3-gui-redesign.md`  
**Completion Status:** 🔴 **NOT STARTED** → Update as you progress

---

## 🎯 IMPORTANT REMINDERS

1. **Read `start_here/PROJECT_RULES.md` FIRST** before starting any work
2. **Follow modular architecture** - No files >200 lines
3. **Use shadcn/ui** - Don't create custom components when shadcn has them
4. **Match Enterprise layout** - Protection Flows must match Catalogs page exactly
5. **Accent color #023E8A** - Use consistently throughout
6. **No gradients, no emojis** - Clean professional design only
7. **TypeScript strict mode** - Zero `any` types allowed
8. **Test on production URL** - http://10.245.246.134:3000

---

**READY TO START - ALL REQUIREMENTS DOCUMENTED**

**Next AI Session: Read this job sheet, then begin Phase 1 implementation!**
