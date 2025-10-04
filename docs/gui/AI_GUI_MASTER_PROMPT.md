# AI GUI Master Prompt - MigrateKit Migration GUI

## 🎯 **Project Context**

You are working on the **MigrateKit OSSEA Migration GUI** - a production-ready Next.js application for managing VMware to CloudStack migrations. This is a professional, enterprise-grade interface that must follow strict best practices.

## 🏗️ **Architecture Overview**

### **Technology Stack**
- **Framework**: Next.js 15.4.5 (App Router)
- **Language**: TypeScript (100% - NO `any` types)
- **Styling**: Tailwind CSS + Flowbite React components
- **State Management**: Zustand + React Query
- **UI Framework**: Reavyr-inspired three-panel layout
- **Target**: Production deployment on OMA appliance

### **Three-Panel Layout Structure**
```
┌─ Left Navigation ─────────┬─ Main Content Area ──────────────────┬─ Right Context Panel ─┐
│ 🏠 Dashboard              │ Dynamic content based on navigation  │ Selected VM info      │
│ 🔍 Discovery              │ - VM Table (primary)                 │ Progress & status     │
│ 💻 Virtual Machines       │ - VM Detail Tabs                     │ Quick actions         │
│ 📋 Replication Jobs       │ - Job Lists                          │ Recent activity       │
│ 🔄 Failover               │ - Configuration forms                │ System health         │
│ 🌐 Network Mapping        │                                      │                       │
│ 📝 Logs                   │                                      │                       │
│ ⚙️ Settings               │                                      │                       │
└───────────────────────────┴──────────────────────────────────────┴───────────────────────┘
```

## 📊 **Data Integration**

### **VM Context API** (Primary Data Source)
```typescript
// Main endpoints
GET /api/v1/vm-contexts              // List all VMs with status
GET /api/v1/vm-contexts/{vm_name}    // Complete VM details

// Response structure
interface VMContextDetails {
  context: VMReplicationContext;     // VM metadata and status
  current_job: ReplicationJob;       // Live job progress
  job_history: ReplicationJob[];     // Last 10 jobs
  disks: VMDisk[];                   // VM disk configuration
  cbt_history: CBTHistory[];         // Change tracking
}
```

### **Real-Time Updates**
- **5-second polling** for active job progress
- **30-second polling** for VM list updates
- **Live progress bars** with ETA and transfer speed
- **Status indicators** with color coding

## 🎨 **UI/UX Requirements**

### **Design Principles**
1. **Professional**: Enterprise-grade interface, consistent with Reavyr aesthetics
2. **Responsive**: Works on desktop, tablet, mobile
3. **Accessible**: WCAG 2.1 AA compliance
4. **Performance**: Fast loading, smooth interactions
5. **User-Friendly**: Intuitive navigation, clear feedback

### **Component Standards**
- **Flowbite React** components for consistency
- **Tailwind CSS** utility classes (no custom CSS unless necessary)
- **Dark mode support** throughout
- **Loading states** for all async operations
- **Error boundaries** with recovery options

## 🔧 **Development Rules**

### **🚨 CRITICAL RULES (NEVER BREAK)**

1. **TypeScript Only**: 100% TypeScript, NO `any` types, strict mode
2. **Component Modularity**: Components max 200 lines, single responsibility
3. **No Direct API Calls**: Use custom hooks and React Query
4. **Error Handling**: Every component wrapped in error boundaries
5. **Loading States**: Show loading feedback for ALL async operations
6. **File Structure**: Follow exact structure in NEXTJS_BEST_PRACTICES.md
7. **Props Interfaces**: All props must have explicit TypeScript interfaces
8. **Performance**: Use React.memo, useCallback, useMemo appropriately
9. **Accessibility**: All interactive elements must be keyboard accessible
10. **Testing**: Write tests for all new components

### **Code Quality Standards**
```typescript
// ✅ ALWAYS DO THIS
interface ComponentProps {
  data: SpecificType;
  onAction: (param: string) => void;
  loading?: boolean;
  error?: string;
}

const Component = React.memo(({ data, onAction, loading = false }: ComponentProps) => {
  const handleClick = useCallback((id: string) => {
    onAction(id);
  }, [onAction]);

  if (loading) return <LoadingSpinner />;
  
  return (
    <div className="bg-white dark:bg-gray-800 rounded-lg shadow-sm">
      {/* Implementation */}
    </div>
  );
});

// ❌ NEVER DO THIS
const Component = ({ data, onClick }: any) => {
  return <div>{data.map(item => <span onClick={() => onClick(item.id)}>{item.name}</span>)}</div>;
};
```

## 📁 **File Organization**

### **Component Creation Pattern**
```typescript
// 1. Create component file: src/components/[category]/ComponentName.tsx
// 2. Create types file: src/lib/types/[category].ts (if new types needed)
// 3. Create hook file: src/hooks/use[Feature].ts (if custom logic needed)
// 4. Create test file: __tests__/components/[category]/ComponentName.test.tsx
// 5. Update exports: src/components/index.ts
```

### **Required File Structure**
```
src/
├── app/                     # Next.js App Router pages
├── components/              # Reusable components
│   ├── ui/                 # Base components (Button, Card, etc.)
│   ├── layout/             # Layout components
│   ├── vm/                 # VM-specific components
│   ├── jobs/               # Job-related components
│   ├── forms/              # Form components
│   └── common/             # Common utilities
├── lib/                    # Utilities and types
├── hooks/                  # Custom React hooks
├── store/                  # Zustand stores
└── styles/                 # Global styles
```

## 🔌 **API Integration Patterns**

### **Custom Hooks Pattern** (MANDATORY)
```typescript
// hooks/useVMContext.ts
export function useVMContext(vmName: string) {
  return useQuery({
    queryKey: ['vmContext', vmName],
    queryFn: () => api.getVMContext(vmName),
    enabled: !!vmName,
    refetchInterval: vmName ? 5000 : false, // Real-time for selected VM
    staleTime: 2000,
    retry: 3,
    onError: (error) => {
      toast.error(`Failed to load VM context: ${error.message}`);
    }
  });
}

// Usage in components
function VMDetailPage({ vmName }: { vmName: string }) {
  const { data: vmContext, loading, error, refetch } = useVMContext(vmName);
  
  if (loading) return <VMDetailSkeleton />;
  if (error) return <ErrorMessage error={error} onRetry={refetch} />;
  if (!vmContext) return <NotFound message="VM not found" />;
  
  return <VMDetailTabs vmContext={vmContext} />;
}
```

### **State Management Pattern**
```typescript
// store/vmStore.ts
interface VMStore {
  selectedVM: string | null;
  setSelectedVM: (vmName: string | null) => void;
  // More state...
}

export const useVMStore = create<VMStore>((set) => ({
  selectedVM: null,
  setSelectedVM: (vmName) => set({ selectedVM: vmName }),
}));
```

## 🎯 **Component Implementation Guidelines**

### **Required Component Structure**
```typescript
// Every component MUST follow this pattern:

interface ComponentNameProps {
  // Explicit prop types
}

export const ComponentName = React.memo(({ prop1, prop2 }: ComponentNameProps) => {
  // 1. Hooks (useState, useEffect, custom hooks)
  // 2. Handlers (useCallback wrapped)
  // 3. Early returns (loading, error, empty states)
  // 4. Main JSX

  const handleAction = useCallback((param: string) => {
    // Handler implementation
  }, [dependencies]);

  if (loading) return <LoadingSkeleton />;
  if (error) return <ErrorMessage error={error} />;

  return (
    <div className="component-container">
      {/* Implementation */}
    </div>
  );
});

ComponentName.displayName = 'ComponentName';
```

### **Navigation Implementation**
```typescript
// Left Navigation Structure
const navigationItems = [
  { id: 'dashboard', label: 'Dashboard', icon: HomeIcon, href: '/dashboard' },
  { id: 'discovery', label: 'Discovery', icon: SearchIcon, href: '/discovery' },
  { id: 'virtual-machines', label: 'Virtual Machines', icon: ServerIcon, href: '/virtual-machines' },
  { id: 'replication-jobs', label: 'Replication Jobs', icon: ClipboardIcon, href: '/replication-jobs' },
  { id: 'failover', label: 'Failover', icon: ArrowPathIcon, href: '/failover' },
  { id: 'network-mapping', label: 'Network Mapping', icon: GlobeAltIcon, href: '/network-mapping' },
  { id: 'logs', label: 'Logs', icon: DocumentTextIcon, href: '/logs' },
  { id: 'settings', label: 'Settings', icon: CogIcon, href: '/settings' }
];
```

## 🚨 **Error Handling Requirements**

### **Mandatory Error Patterns**
```typescript
// 1. Error Boundaries (wrap all major sections)
<ErrorBoundary fallback={ErrorFallback}>
  <ComponentThatMightFail />
</ErrorBoundary>

// 2. API Error Handling
const { data, error, isLoading } = useQuery({
  queryKey: ['data'],
  queryFn: fetchData,
  onError: (error) => {
    console.error('API Error:', error);
    toast.error(`Failed to load data: ${error.message}`);
  }
});

// 3. Form Validation
const form = useForm({
  resolver: zodResolver(validationSchema),
  onError: (errors) => {
    Object.values(errors).forEach(error => {
      toast.error(error.message);
    });
  }
});
```

## 📱 **Responsive Design Requirements**

### **Breakpoint Strategy**
```typescript
// Tailwind breakpoints
// sm: 640px  - Mobile landscape
// md: 768px  - Tablet
// lg: 1024px - Desktop
// xl: 1280px - Large desktop

// Layout behavior:
// Mobile (< 768px): Stack panels, bottom navigation
// Tablet (768px - 1023px): Collapsible sidebar
// Desktop (≥ 1024px): Full three-panel layout
```

## 🧪 **Testing Requirements**

### **Required Tests**
```typescript
// 1. Component rendering
test('renders VM table with data', () => {
  render(<VMTable vms={mockVMs} />);
  expect(screen.getByText('test-vm')).toBeInTheDocument();
});

// 2. User interactions
test('calls onVMSelect when VM clicked', () => {
  const onVMSelect = jest.fn();
  render(<VMTable vms={mockVMs} onVMSelect={onVMSelect} />);
  fireEvent.click(screen.getByText('test-vm'));
  expect(onVMSelect).toHaveBeenCalledWith('test-vm');
});

// 3. Loading states
test('shows loading skeleton when loading', () => {
  render(<VMTable loading={true} />);
  expect(screen.getByTestId('vm-table-skeleton')).toBeInTheDocument();
});

// 4. Error states
test('shows error message on error', () => {
  render(<VMTable error="Failed to load" />);
  expect(screen.getByText('Failed to load')).toBeInTheDocument();
});
```

## 🎨 **Styling Guidelines**

### **Tailwind Class Patterns**
```typescript
// Card containers
const cardClasses = "bg-white dark:bg-gray-800 rounded-lg shadow-sm border border-gray-200 dark:border-gray-700 p-6";

// Status badges
const statusClasses = {
  success: "bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-300",
  warning: "bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-300",
  error: "bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-300",
  info: "bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-300"
};

// Buttons
const buttonClasses = "bg-blue-600 hover:bg-blue-700 disabled:bg-gray-300 text-white font-medium rounded-lg text-sm px-5 py-2.5 transition-colors";
```

## 🔄 **Real-Time Update Strategy**

### **Polling Intervals**
- **Active job progress**: 5 seconds
- **VM list updates**: 30 seconds
- **System health**: 60 seconds
- **User interactions**: Immediate

### **Update Patterns**
```typescript
// Real-time progress for selected VM
const { data: vmContext } = useQuery({
  queryKey: ['vmContext', selectedVM],
  queryFn: () => api.getVMContext(selectedVM),
  enabled: !!selectedVM,
  refetchInterval: 5000,
  staleTime: 2000
});

// Background updates for all VMs
const { data: vmList } = useQuery({
  queryKey: ['vmContexts'],
  queryFn: () => api.getVMContexts(),
  refetchInterval: 30000,
  staleTime: 15000
});
```

## 📋 **Development Workflow**

### **Before Writing Any Code**
1. **Read** the VM Context API documentation
2. **Check** existing components for similar functionality
3. **Plan** component structure and props interface
4. **Identify** required custom hooks
5. **Design** error and loading states

### **Implementation Steps**
1. **Create** TypeScript interfaces first
2. **Build** component with loading/error states
3. **Add** proper error boundaries
4. **Implement** responsive design
5. **Write** unit tests
6. **Test** accessibility
7. **Document** complex logic

### **Code Review Checklist**
- [ ] TypeScript strict mode passes
- [ ] No `any` types used
- [ ] Props have explicit interfaces
- [ ] Error boundaries in place
- [ ] Loading states implemented
- [ ] Responsive design working
- [ ] Accessibility tested
- [ ] Unit tests written
- [ ] Performance optimized (memo, callback)
- [ ] Follows file structure

## 🎯 **Success Criteria**

### **User Experience**
- **Fast**: Page loads < 2 seconds
- **Responsive**: Works on all devices
- **Intuitive**: Users can complete tasks without training
- **Reliable**: No crashes, graceful error handling

### **Code Quality**
- **Maintainable**: Easy to understand and modify
- **Testable**: High test coverage
- **Performant**: Smooth interactions
- **Accessible**: WCAG 2.1 AA compliant

### **Production Ready**
- **Secure**: No security vulnerabilities
- **Scalable**: Handles growth
- **Monitored**: Error tracking and analytics
- **Documented**: Complete documentation

---

## 🚨 **Critical Reminders**

1. **ALWAYS** check existing components before creating new ones
2. **NEVER** use `any` types - prefer `unknown` or proper interfaces
3. **ALWAYS** implement loading and error states
4. **NEVER** make direct API calls in components - use custom hooks
5. **ALWAYS** use React.memo for performance optimization
6. **NEVER** ignore accessibility - test with keyboard navigation
7. **ALWAYS** follow the three-panel Reavyr-inspired layout
8. **NEVER** break the established file structure
9. **ALWAYS** write tests for new components
10. **NEVER** deploy without proper error handling

**Remember: This is a production enterprise application. Quality and maintainability are paramount! 🚀**
