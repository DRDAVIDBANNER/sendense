# GUI Directory Structure Recommendation

**Date:** October 6, 2025  
**Purpose:** Directory structure for new Sendense Professional GUI implementation  
**Implementation Tool:** Grok Code Fast (AI coding assistant)

---

## 🎯 RECOMMENDED DIRECTORY STRUCTURE

### **Primary Recommendation: Development in Source Authority**

```
/home/oma_admin/sendense/source/current/sendense-gui/
├── README.md                           # Setup and development guide
├── package.json
├── next.config.ts
├── tailwind.config.ts  
├── tsconfig.json
├── .env.example                        # Environment variables template
├── .env.local                          # Local development config
├── .gitignore
│
├── public/                             # Static assets
│   ├── favicon.ico
│   ├── logo.svg
│   └── images/
│
├── src/
│   ├── app/                            # Next.js App Router
│   │   ├── layout.tsx                  # Root layout with sidebar
│   │   ├── page.tsx                    # Dashboard (redirect or home)
│   │   ├── globals.css                 # Global styles + Tailwind
│   │   │
│   │   ├── dashboard/                  # System overview
│   │   │   └── page.tsx
│   │   │
│   │   ├── protection-flows/           # Main feature (backup/replication)
│   │   │   ├── page.tsx
│   │   │   └── [flow-id]/
│   │   │       └── page.tsx
│   │   │
│   │   ├── protection-groups/          # VM organization
│   │   │   ├── page.tsx  
│   │   │   └── [group-id]/
│   │   │       └── page.tsx
│   │   │
│   │   ├── report-center/              # Analytics and reports
│   │   │   └── page.tsx
│   │   │
│   │   ├── settings/                   # Configuration
│   │   │   ├── sources/
│   │   │   │   └── page.tsx
│   │   │   └── destinations/
│   │   │       └── page.tsx
│   │   │
│   │   ├── users/                      # User management
│   │   │   └── page.tsx
│   │   │
│   │   └── support/                    # Help and documentation
│   │       └── page.tsx
│   │
│   ├── features/                       # Feature modules (following spec)
│   │   ├── dashboard/
│   │   │   ├── components/
│   │   │   │   ├── SystemHealthCards.tsx
│   │   │   │   ├── RecentActivity.tsx
│   │   │   │   └── PerformanceChart.tsx
│   │   │   ├── hooks/
│   │   │   │   └── useDashboardData.ts
│   │   │   └── types/
│   │   │       └── dashboard.types.ts
│   │   │
│   │   ├── protection-flows/           # Main feature
│   │   │   ├── components/
│   │   │   │   ├── FlowsTable/
│   │   │   │   │   ├── index.tsx
│   │   │   │   │   ├── FlowRow.tsx
│   │   │   │   │   ├── StatusCell.tsx
│   │   │   │   │   └── ActionsDropdown.tsx
│   │   │   │   ├── FlowDetailsPanel/
│   │   │   │   │   ├── index.tsx
│   │   │   │   │   ├── OverviewTab.tsx
│   │   │   │   │   ├── VolumesTab.tsx
│   │   │   │   │   └── HistoryTab.tsx
│   │   │   │   ├── JobLogPanel/
│   │   │   │   │   ├── index.tsx
│   │   │   │   │   ├── LogViewer.tsx
│   │   │   │   │   └── LogFilters.tsx
│   │   │   │   └── modals/
│   │   │   │       ├── CreateFlowModal.tsx
│   │   │   │       ├── EditFlowModal.tsx
│   │   │   │       └── DeleteConfirmModal.tsx
│   │   │   ├── hooks/
│   │   │   │   ├── useFlows.ts
│   │   │   │   ├── usePanelSize.ts
│   │   │   │   └── useRealTimeLogs.ts
│   │   │   └── types/
│   │   │       └── flows.types.ts
│   │   │
│   │   ├── protection-groups/
│   │   │   ├── components/
│   │   │   │   ├── GroupsList.tsx
│   │   │   │   ├── GroupCard.tsx
│   │   │   │   ├── CreateGroupModal.tsx
│   │   │   │   └── VMAssignment.tsx
│   │   │   ├── hooks/
│   │   │   │   └── useGroups.ts
│   │   │   └── types/
│   │   │       └── groups.types.ts
│   │   │
│   │   └── reports/
│   │       ├── components/
│   │       │   ├── KPISummary.tsx
│   │       │   ├── TrendCharts.tsx
│   │       │   └── DateRangePicker.tsx
│   │       ├── hooks/
│   │       │   └── useReports.ts
│   │       └── types/
│   │           └── reports.types.ts
│   │
│   ├── components/                     # Shared components
│   │   ├── ui/                         # shadcn/ui components
│   │   │   ├── button.tsx
│   │   │   ├── card.tsx
│   │   │   ├── dialog.tsx
│   │   │   ├── table.tsx
│   │   │   ├── tabs.tsx
│   │   │   └── progress.tsx
│   │   │
│   │   ├── layout/                     # Layout components
│   │   │   ├── Sidebar.tsx
│   │   │   ├── PageHeader.tsx
│   │   │   └── ThemeSwitcher.tsx
│   │   │
│   │   └── common/                     # Shared utility components
│   │       ├── StatusBadge.tsx
│   │       ├── LoadingSpinner.tsx
│   │       ├── EmptyState.tsx
│   │       ├── ErrorBoundary.tsx
│   │       └── DataTable.tsx
│   │
│   └── lib/                            # Utilities and configuration
│       ├── api/
│       │   ├── client.ts               # Main API client
│       │   ├── endpoints.ts            # API endpoint definitions
│       │   └── types.ts                # API response types
│       │
│       ├── hooks/                      # Shared React hooks
│       │   ├── useLocalStorage.ts
│       │   ├── useDebounce.ts
│       │   └── useWebSocket.ts
│       │
│       ├── utils/                      # Utility functions
│       │   ├── format.ts               # Data formatting
│       │   ├── validation.ts           # Form validation
│       │   └── constants.ts            # App constants
│       │
│       └── stores/                     # State management (if needed)
│           └── ui-state.ts
```

---

## 🏗️ ALTERNATIVE STRUCTURES

### **Option A: Development in Source Authority (RECOMMENDED)**
```
/home/oma_admin/sendense/source/current/sendense-gui/
```

**Benefits:**
- ✅ **Follows PROJECT_RULES:** All source code in `source/current/`
- ✅ **Version Control:** Part of main repository
- ✅ **Source Authority:** Canonical location for all code
- ✅ **Build Management:** Can be built and versioned with rest of platform

### **Option B: Deployment Directory (Current Pattern)**
```
/home/oma_admin/sendense/deployment/sha-appliance/gui-v2/
```

**Benefits:**
- ✅ **Deployment Ready:** Directly in deployment structure
- ✅ **Separate from Existing:** Won't conflict with current GUI
- ✅ **Production Path:** Can be deployed immediately

### **Option C: Separate Development Directory**
```
/home/oma_admin/sendense/gui-development/sendense-professional/
```

**Benefits:**
- ✅ **Clean Start:** No existing code conflicts
- ✅ **Development Focus:** Can iterate without deployment concerns

---

## 📋 RECOMMENDED APPROACH

### **PRIMARY RECOMMENDATION: Option A + Deployment Symlink**

**Development Location:**
```
/home/oma_admin/sendense/source/current/sendense-gui/
```

**Deployment Symlink:**
```bash
# Create symlink from deployment to source
ln -s /home/oma_admin/sendense/source/current/sendense-gui \
      /home/oma_admin/sendense/deployment/sha-appliance/gui-v2

# Or copy for production deployment
```

### **Why This Approach:**

**1. Follows PROJECT_RULES:**
- ✅ Source code in `source/current/` (canonical authority)
- ✅ Can be versioned with platform builds
- ✅ Consistent with other components (oma, volume-daemon, etc.)

**2. Development Efficiency:**
- ✅ Grok Code Fast can work in clean directory
- ✅ No conflicts with existing GUI
- ✅ Easy to reference other source components

**3. Deployment Flexibility:**
- ✅ Can symlink to deployment directory
- ✅ Can copy for production packaging
- ✅ Maintains separation during development

---

## 🚀 SETUP COMMANDS FOR GROK

### **Create Development Directory:**
```bash
cd /home/oma_admin/sendense/source/current/

# Create new GUI directory
mkdir sendense-gui
cd sendense-gui

# Initialize Next.js project
npx create-next-app@latest . --typescript --tailwind --app --no-src-dir

# Setup shadcn/ui
npx shadcn@latest init

# Install additional dependencies
npm install @tanstack/react-query lucide-react zustand date-fns recharts
npm install -D @types/node @types/react @types/react-dom

# Create feature-based structure
mkdir -p src/features/{dashboard,protection-flows,protection-groups,reports}
mkdir -p src/components/{ui,layout,common}
mkdir -p src/lib/{api,hooks,utils,stores}

# Initial git commit
git add .
git commit -m "Initial Sendense Professional GUI setup"
```

### **Environment Configuration:**
```bash
# Create .env.local for development
cat > .env.local << EOF
NEXT_PUBLIC_API_URL=http://localhost:8082
NEXT_PUBLIC_WS_URL=ws://localhost:8082/ws
NODE_ENV=development
EOF
```

---

## 🎨 PROJECT ORGANIZATION

### **Feature Module Structure (Following Spec):**

Each feature should be self-contained:
```
src/features/protection-flows/
├── components/
│   ├── FlowsTable/
│   ├── FlowDetailsPanel/
│   ├── JobLogPanel/
│   └── modals/
├── hooks/
│   ├── useFlows.ts
│   └── usePanelSize.ts  
├── types/
│   └── flows.types.ts
└── utils/
    └── flow.utils.ts
```

### **Shared Components:**
```
src/components/
├── ui/                     # shadcn/ui components (auto-generated)
├── layout/                 # Layout components
│   ├── Sidebar.tsx
│   ├── PageHeader.tsx
│   └── MainLayout.tsx
└── common/                 # Shared utility components
    ├── StatusBadge.tsx
    ├── LoadingSpinner.tsx
    └── EmptyState.tsx
```

---

## 🔧 GROK IMPLEMENTATION GUIDANCE

### **For Grok Code Fast:**

**1. Start Directory:**
```
/home/oma_admin/sendense/source/current/sendense-gui/
```

**2. Key Requirements:**
- **Component Size Limit:** <200 lines per file (as specified)
- **TypeScript Strict:** Zero `any` types allowed
- **shadcn/ui Components:** Use instead of custom components
- **Feature Architecture:** Self-contained feature modules
- **Accent Color:** #023E8A throughout
- **Dark Theme:** Default professional appearance

**3. Critical Files First:**
- `src/app/layout.tsx` (main layout with sidebar)
- `src/components/layout/Sidebar.tsx` (7 menu items)
- `src/app/protection-flows/page.tsx` (main feature)
- `src/features/protection-flows/components/FlowsTable/index.tsx`

**4. API Integration:**
- Base URL: `http://localhost:8082` (points to our sendense-hub)
- Use existing Task 5 backup endpoints
- Use existing Task 4 restore endpoints
- Follow existing API patterns

---

## 📂 DEPLOYMENT STRATEGY

### **Development → Deployment Flow:**

**1. Development:** 
```
source/current/sendense-gui/          # Development and source control
```

**2. Build:**
```bash
cd source/current/sendense-gui/
npm run build
```

**3. Deploy to SHA:**
```bash
# Copy built GUI to deployment directory
cp -r .next/ deployment/sha-appliance/gui-v2/
cp -r public/ deployment/sha-appliance/gui-v2/
cp package.json deployment/sha-appliance/gui-v2/
```

**4. Production Service:**
```bash
# Update systemd service to point to new GUI
sudo systemctl edit sendense-gui
# ExecStart=/usr/bin/npm start
# WorkingDirectory=/home/oma_admin/sendense/deployment/sha-appliance/gui-v2
```

---

## ✅ FINAL RECOMMENDATION

### **Create Development Directory:**
```bash
mkdir -p /home/oma_admin/sendense/source/current/sendense-gui
```

### **Benefits:**
- ✅ **Source Authority Compliance:** Follows PROJECT_RULES
- ✅ **Clean Development:** No conflicts with existing GUI
- ✅ **Version Control:** Part of main repository  
- ✅ **Deployment Ready:** Easy to copy to deployment directory
- ✅ **Grok Friendly:** Clean starting point for AI implementation

### **Next Steps for Grok:**
1. Create the directory structure above
2. Initialize Next.js 15 with TypeScript + Tailwind
3. Set up shadcn/ui with Sendense color palette
4. Implement feature-based architecture per the job sheet
5. Start with Protection Flows page (main feature)

**This structure will give Grok Code Fast a clean foundation to implement your professional GUI specification.** 🚀
