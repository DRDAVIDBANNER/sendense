# Job Sheet: Sendense Professional GUI Implementation

**Date Created:** 2025-10-06  
**Status:** 🔴 **READY TO START**  
**Project Goal Link:** [project-goals/phases/phase-3-gui-redesign.md → Sendense Professional GUI]  
**Duration:** 4-6 weeks  
**Priority:** Critical (Enterprise customer-facing interface)  
**Implementation Tool:** Grok Code Fast (AI coding assistant)

---

## 🎯 PROJECT GOALS INTEGRATION (MANDATORY)

### **Specific Project Goals Reference**
**Document:** `/sendense/project-goals/phases/phase-3-gui-redesign.md`  
**Phase:** Phase 3 - GUI Redesign  
**Business Value:** Professional interface for enterprise sales, competitive advantage vs Veeam  
**Success Criteria:** Clean, enterprise-grade interface that justifies premium pricing

**Phase 3 Objectives:**
- ✅ **Professional Design:** Clean, modern interface without aviation metaphors
- ✅ **Enterprise Layout:** Three-panel layout (table + details + logs)
- ✅ **Multi-Platform Orchestration:** 6 platforms in single interface
- ✅ **Everything Within Reach:** Minimal clicks, fast operations
- ✅ **Real-Time Telemetry:** Live updates for all operations

---

## 🔗 DEPENDENCY STATUS

### **Required Before Starting:**
- ✅ **Task 5:** Backup API Endpoints (POST /backup/start, GET /backup/list, etc.) - OPERATIONAL
- ✅ **Task 4:** File-Level Restore API (POST /restore/mount, GET /files, etc.) - OPERATIONAL
- ✅ **Existing APIs:** VM management, replication, failover endpoints - OPERATIONAL
- ✅ **Backend Infrastructure:** All APIs tested and documented

### **Foundation Ready:**
- ✅ **API Documentation:** Complete endpoint documentation in `api-documentation/`
- ✅ **Database Schema:** All tables operational for GUI data consumption
- ✅ **Real-Time Updates:** WebSocket infrastructure available
- ✅ **Authentication:** Bearer token system operational

### **Enables These Features:**
- 🎯 **Enterprise Customer Adoption:** Professional interface for C-level demonstrations
- 🎯 **Self-Service Operations:** Customer-driven backup/restore workflows
- 🎯 **Competitive Advantage:** Superior interface vs Veeam, Nakivo, competitors
- 🎯 **Revenue Enablement:** GUI required for $10-100/VM tier customer adoption

---

## 📋 JOB BREAKDOWN (Modular Implementation)

### **Phase 1: Foundation Setup (Week 1)**

- [ ] **Initialize Next.js 15 Project**
  - **Location:** `/home/oma_admin/sendense/source/current/sendense-gui/`
  - **Command:** `npx create-next-app@latest . --typescript --tailwind --app --no-src-dir`
  - **Evidence:** Working Next.js app with TypeScript + Tailwind

- [ ] **Install Dependencies & shadcn/ui**
  - **Commands:** `npm install` + `npx shadcn@latest init`
  - **Additional:** React Query, Lucide icons, date-fns, recharts
  - **Evidence:** All dependencies installed, shadcn/ui configured

- [ ] **Configure Sendense Design System**
  - **File:** `tailwind.config.ts` with Sendense colors
  - **Accent:** #023E8A professional blue throughout
  - **Theme:** Dark mode default, optional light mode
  - **Evidence:** Design system applied, professional appearance

- [ ] **Create Feature-Based Structure**
  - **Directories:** `src/features/`, `src/components/`, `src/lib/`
  - **Pattern:** Feature modules with components/hooks/types per feature
  - **Evidence:** Modular architecture established

### **Phase 2: Core Layout & Sidebar (Week 1)**

- [ ] **Create Main Layout Component**
  - **File:** `src/app/layout.tsx`
  - **Features:** Root layout with sidebar, dark theme
  - **Integration:** Theme provider, React Query provider
  - **Evidence:** Layout renders with proper styling

- [ ] **Create Sidebar Navigation**
  - **File:** `src/components/layout/Sidebar.tsx`
  - **Items:** 7 menu items (Dashboard, Protection Flows, Groups, Reports, etc.)
  - **Design:** 256px width, professional styling, active states
  - **Evidence:** Navigation functional with all routes

- [ ] **Create Shared Components**
  - **Files:** StatusBadge, LoadingSpinner, EmptyState, PageHeader
  - **Standard:** <200 lines per file, TypeScript strict
  - **Evidence:** Reusable components operational

### **Phase 3: Protection Flows (Main Feature) (Week 2-3)**

- [ ] **Create FlowsTable Component**
  - **File:** `src/features/protection-flows/components/FlowsTable/index.tsx`
  - **Features:** Sortable table, flow selection, status display
  - **Integration:** API endpoints for backup/replication jobs
  - **Evidence:** Table displays real flow data

- [ ] **Create FlowDetailsPanel (Draggable)**
  - **File:** `src/features/protection-flows/components/FlowDetailsPanel/index.tsx`
  - **Features:** Horizontal drag divider, tabs (Overview/Volumes/History)
  - **State:** localStorage persistence for height
  - **Evidence:** Panel resizes smoothly, state persists

- [ ] **Create JobLogPanel (Collapsible)**
  - **File:** `src/features/protection-flows/components/JobLogPanel/index.tsx`
  - **Features:** Vertical drag divider, collapsible (48px → 420px)
  - **State:** localStorage persistence for width and expansion
  - **Evidence:** Panel collapses/expands, logs display in real-time

- [ ] **Create Flow Management Modals**
  - **Files:** CreateFlowModal, EditFlowModal, DeleteConfirmModal
  - **Integration:** Task 5 backup APIs for flow creation
  - **Evidence:** Can create, edit, and delete flows via modals

### **Phase 4: Additional Pages (Week 3-4)**

- [ ] **Dashboard Page**
  - **File:** `src/app/dashboard/page.tsx`
  - **Features:** System health cards, recent activity, performance graphs
  - **Evidence:** Dashboard shows real system metrics

- [ ] **Protection Groups Page**
  - **File:** `src/app/protection-groups/page.tsx`
  - **Features:** VM grouping, policy assignment, bulk operations
  - **Evidence:** Group management functional

- [ ] **Report Center Page**
  - **File:** `src/app/report-center/page.tsx`
  - **Features:** KPI summaries, trend charts, data export
  - **Evidence:** Reports display with real data

### **Phase 5: Settings & User Management (Week 4)**

- [ ] **Settings Pages**
  - **Files:** `src/app/settings/sources/page.tsx`, `destinations/page.tsx`
  - **Features:** Platform connections, repository management
  - **Evidence:** Configuration interfaces functional

- [ ] **Users Management**
  - **File:** `src/app/users/page.tsx`
  - **Features:** User list, role management, permissions
  - **Evidence:** User management operational

### **Phase 6: Polish & Production (Week 5-6)**

- [ ] **Performance Optimization**
  - **Tasks:** Bundle optimization, React.memo, code splitting
  - **Target:** Lighthouse score >90
  - **Evidence:** Performance benchmarks met

- [ ] **Error Handling & Accessibility**
  - **Tasks:** Error boundaries, toast notifications, ARIA labels
  - **Target:** Accessibility score >90
  - **Evidence:** Robust error handling, accessible interface

- [ ] **Production Build & Deployment**
  - **Tasks:** Production build, systemd service, deployment scripts
  - **Target:** Running on production server
  - **Evidence:** GUI accessible at production URL

---

## 🎨 TECHNICAL ARCHITECTURE

### **Component Architecture (Following Spec)**
```typescript
// Feature-based organization
src/features/protection-flows/
├── components/
│   ├── FlowsTable/index.tsx           # <200 lines
│   ├── FlowDetailsPanel/index.tsx     # <200 lines
│   └── JobLogPanel/index.tsx          # <200 lines
├── hooks/
│   ├── useFlows.ts
│   └── usePanelSize.ts
└── types/
    └── flows.types.ts
```

### **API Integration Pattern**
```typescript
// Extend for all Sendense APIs
const apiClient = {
  // Task 5: Backup APIs
  backup: {
    start: (req: StartBackupRequest) => post('/api/v1/backup/start', req),
    list: (filters?) => get('/api/v1/backup/list', filters),
    details: (id: string) => get(`/api/v1/backup/${id}`),
    delete: (id: string) => del(`/api/v1/backup/${id}`),
    chain: (vm: string) => get(`/api/v1/backup/chain?vm_name=${vm}`)
  },
  
  // Task 4: Restore APIs  
  restore: {
    mount: (backupId: string) => post('/api/v1/restore/mount', {backup_id: backupId}),
    files: (mountId: string, path?: string) => get(`/api/v1/restore/${mountId}/files?path=${path}`),
    download: (mountId: string, path: string) => get(`/api/v1/restore/${mountId}/download?path=${path}`),
    unmount: (mountId: string) => del(`/api/v1/restore/${mountId}`)
  },
  
  // Existing APIs
  vms: { /* existing endpoints */ },
  replication: { /* existing endpoints */ }
};
```

---

## 🚨 PROJECT RULES COMPLIANCE

### **Must Follow (No Exceptions):**
- ✅ **Source Authority:** All code in `source/current/sendense-gui/`
- ✅ **Component Size Limit:** <200 lines per file (specified)
- ✅ **TypeScript Strict:** Zero `any` types allowed
- ✅ **API Integration:** Use Task 4 + Task 5 endpoints without modification
- ✅ **No Simulations:** Real backend integration with live data
- ✅ **Documentation Updates:** Update API docs, CHANGELOG, project status
- ✅ **shadcn/ui Required:** Use component library, don't create custom components

### **Design Requirements:**
- ✅ **Professional Aesthetics:** Clean, enterprise-grade appearance
- ✅ **No Emojis in UI:** Professional consistency throughout
- ✅ **Accent Color:** #023E8A professional blue
- ✅ **Dark Theme:** Default professional appearance
- ✅ **Enterprise Layout:** Three-panel pattern for Protection Flows

### **Quality Standards:**
- ✅ **Performance:** Lighthouse score >90
- ✅ **Accessibility:** ARIA labels, keyboard navigation
- ✅ **Error Handling:** Robust error boundaries and user feedback
- ✅ **Real-Time Updates:** Live data without manual refresh

---

## ✅ SUCCESS VALIDATION

### **Completion Criteria (All Must Pass)**
- [ ] **All 7 Pages Functional:** Dashboard, Protection Flows, Groups, Reports, Settings, Users, Support
- [ ] **Enterprise Layout:** Protection Flows uses three-panel layout with draggable dividers
- [ ] **API Integration:** All backend endpoints accessible via GUI
- [ ] **Real-Time Updates:** Live data feeds for flows, logs, system health
- [ ] **Professional Design:** Clean interface suitable for enterprise demonstrations
- [ ] **Performance Standards:** Lighthouse scores >90 across all pages
- [ ] **Component Architecture:** Feature-based modules, <200 lines per file

### **Testing Evidence Required**
- [ ] **Navigation:** All 7 menu items functional
- [ ] **Protection Flows:** Table, details panel, log panel all working
- [ ] **Flow Creation:** Can start backup/replication via GUI
- [ ] **File Restore:** Can browse and download files from backups
- [ ] **Real-Time Updates:** Live progress visible without refresh
- [ ] **Panel Dragging:** Details and log panels resize smoothly
- [ ] **Responsive Design:** Works on desktop (1920x1080, 1366x768)

---

## 📚 DOCUMENTATION UPDATES REQUIRED

### **Must Update (PROJECT_RULES Compliance):**
- [ ] **API Documentation:** Update `/source/current/api-documentation/GUI_INTEGRATION.md`
- [ ] **Component Library:** Create `/source/current/api-documentation/GUI_COMPONENTS.md`
- [ ] **CHANGELOG:** Update `/start_here/CHANGELOG.md` with GUI version
- [ ] **Project Goals:** Mark Phase 3 complete in `project-goals/phases/phase-3-gui-redesign.md`
- [ ] **Deployment Guide:** Create deployment instructions for production

### **Expected Deliverables (For Documentation):**
- [ ] **Component Inventory:** List of all components with descriptions
- [ ] **API Integration Summary:** Which endpoints are used where
- [ ] **Feature Coverage:** What functionality is available
- [ ] **Performance Results:** Lighthouse scores and optimization details
- [ ] **Deployment Instructions:** How to build and deploy to production

---

## 🎯 SUCCESS METRICS

### **Technical Success**
- [ ] **Code Quality:** Zero TypeScript errors, zero ESLint warnings
- [ ] **Performance:** Lighthouse scores >90 (performance, accessibility, best practices)
- [ ] **Architecture:** Feature-based modules, component size <200 lines
- [ ] **Integration:** All 15+ API endpoints working from GUI

### **Business Success**
- [ ] **Professional Appearance:** Suitable for enterprise customer demonstrations
- [ ] **User Experience:** Intuitive workflow requiring minimal training
- [ ] **Competitive Advantage:** Superior interface compared to Veeam/Nakivo
- [ ] **Revenue Enablement:** GUI supports $10-100/VM pricing tier adoption

---

## 🔗 READY FOR IMPLEMENTATION

### **Foundation Ready**
- ✅ **Backend APIs:** Tasks 1-5 complete with operational endpoints
- ✅ **Database Schema:** All tables and relationships ready
- ✅ **Real-Time Infrastructure:** WebSocket support available
- ✅ **Authentication:** Bearer token system operational

### **Implementation Path Clear**
- ✅ **Directory Structure:** Recommended location and organization
- ✅ **Tech Stack:** Next.js 15 + shadcn/ui + TypeScript specified
- ✅ **Component Patterns:** Three-panel layout with draggable dividers
- ✅ **Feature Scope:** 7 pages with Protection Flows as primary focus

---

**Job Owner:** AI Implementation (Grok Code Fast)  
**Reviewer:** Project Overseer + Quality Assurance  
**Status:** 🔴 Ready for AI Implementation  
**Expected Completion:** 4-6 weeks  
**Success Metric:** Professional GUI operational in production
