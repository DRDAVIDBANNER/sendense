# Next.js Best Practices - MigrateKit GUI

## 🎯 **Core Principles**

### **1. Production-Ready Architecture**
- **Type Safety**: 100% TypeScript, no `any` types
- **Component Modularity**: Small, focused, reusable components
- **Clear Separation**: Strict separation of concerns
- **Maintainability**: Code that's easy to read, understand, and modify
- **Performance**: Optimized for production deployment

### **2. Professional Standards**
- **Consistent Patterns**: Follow established patterns throughout
- **Error Handling**: Comprehensive error boundaries and user feedback
- **Loading States**: Professional loading indicators for all async operations
- **Accessibility**: WCAG 2.1 AA compliance
- **Responsive Design**: Mobile-first, works on all screen sizes

## 📁 **Project Structure** (Next.js 15.4.5 App Router)

```
~/migration-dashboard/
├── src/
│   ├── app/                          # App Router (Next.js 15+)
│   │   ├── globals.css               # Global styles
│   │   ├── layout.tsx                # Root layout
│   │   ├── page.tsx                  # Dashboard page
│   │   ├── loading.tsx               # Global loading UI
│   │   ├── error.tsx                 # Global error UI
│   │   ├── not-found.tsx             # 404 page
│   │   │
│   │   ├── dashboard/                # Dashboard section
│   │   │   └── page.tsx
│   │   ├── discovery/                # Discovery section
│   │   │   └── page.tsx
│   │   ├── virtual-machines/         # VM management (primary)
│   │   │   ├── page.tsx              # VM list
│   │   │   └── [vmName]/             # Dynamic VM detail
│   │   │       ├── page.tsx          # VM detail page
│   │   │       ├── loading.tsx       # VM detail loading
│   │   │       └── error.tsx         # VM detail error
│   │   ├── replication-jobs/         # Job management
│   │   │   └── page.tsx
│   │   ├── failover/                 # Failover management
│   │   │   └── page.tsx
│   │   ├── network-mapping/          # Network configuration
│   │   │   └── page.tsx
│   │   ├── logs/                     # System logs
│   │   │   └── page.tsx
│   │   ├── settings/                 # Configuration
│   │   │   └── page.tsx
│   │   │
│   │   └── api/                      # API routes (Next.js server functions)
│   │       ├── vm-contexts/
│   │       │   ├── route.ts          # GET /api/vm-contexts
│   │       │   └── [vmName]/
│   │       │       └── route.ts      # GET /api/vm-contexts/[vmName]
│   │       ├── discover/
│   │       │   └── route.ts          # POST /api/discover
│   │       ├── replicate/
│   │       │   └── route.ts          # POST /api/replicate
│   │       ├── failover/
│   │       │   └── route.ts          # POST /api/failover
│   │       └── network-mapping/
│   │           └── route.ts          # POST /api/network-mapping
│   │
│   ├── components/                   # Reusable UI components
│   │   ├── ui/                       # Base UI components (shadcn/ui style)
│   │   │   ├── button.tsx
│   │   │   ├── card.tsx
│   │   │   ├── table.tsx
│   │   │   ├── tabs.tsx
│   │   │   ├── badge.tsx
│   │   │   ├── progress.tsx
│   │   │   └── modal.tsx
│   │   │
│   │   ├── layout/                   # Layout components
│   │   │   ├── LeftNavigation.tsx    # Main navigation
│   │   │   ├── RightContextPanel.tsx # Context panel
│   │   │   ├── Header.tsx            # Top header
│   │   │   └── Footer.tsx            # Footer/status bar
│   │   │
│   │   ├── vm/                       # VM-specific components
│   │   │   ├── VMTable.tsx           # VM list table
│   │   │   ├── VMDetailTabs.tsx      # VM detail tabs
│   │   │   ├── VMProgressCard.tsx    # Progress display
│   │   │   ├── VMStatusBadge.tsx     # Status indicator
│   │   │   └── VMQuickActions.tsx    # Action buttons
│   │   │
│   │   ├── jobs/                     # Job-related components
│   │   │   ├── JobHistoryList.tsx
│   │   │   ├── JobProgressBar.tsx
│   │   │   └── JobStatusBadge.tsx
│   │   │
│   │   ├── forms/                    # Form components
│   │   │   ├── NetworkMappingForm.tsx
│   │   │   ├── ReplicationForm.tsx
│   │   │   └── FailoverForm.tsx
│   │   │
│   │   └── common/                   # Common components
│   │       ├── LoadingSpinner.tsx
│   │       ├── ErrorBoundary.tsx
│   │       ├── ConfirmDialog.tsx
│   │       └── Toast.tsx
│   │
│   ├── lib/                          # Utility libraries
│   │   ├── api.ts                    # API client functions
│   │   ├── types.ts                  # TypeScript type definitions
│   │   ├── utils.ts                  # Utility functions
│   │   ├── constants.ts              # App constants
│   │   ├── validations.ts            # Form validation schemas
│   │   └── formatters.ts             # Data formatting utilities
│   │
│   ├── hooks/                        # Custom React hooks
│   │   ├── useVMContext.ts           # VM context data management
│   │   ├── useRealTimeUpdates.ts     # Real-time polling
│   │   ├── useLocalStorage.ts        # Local storage management
│   │   └── useToast.ts               # Toast notifications
│   │
│   ├── store/                        # State management (Zustand)
│   │   ├── vmStore.ts                # VM state
│   │   ├── jobStore.ts               # Job state
│   │   ├── uiStore.ts                # UI state
│   │   └── index.ts                  # Store exports
│   │
│   └── styles/                       # Styling
│       ├── globals.css               # Global styles
│       └── components.css            # Component-specific styles
│
├── public/                           # Static assets
│   ├── icons/
│   ├── images/
│   └── favicon.ico
│
├── docs/                             # Component documentation
│   └── components/
│       ├── README.md
│       └── component-library.md
│
├── .eslintrc.js                      # ESLint configuration
├── .prettierrc                       # Prettier configuration
├── next.config.js                    # Next.js configuration
├── tailwind.config.ts                # Tailwind CSS configuration
├── tsconfig.json                     # TypeScript configuration
├── package.json                      # Dependencies
└── README.md                         # Project documentation
```

## 🏗️ **Component Architecture Patterns**

### **1. Component Composition Pattern**
```typescript
// ✅ GOOD: Composable components
<VMDetailPage>
  <VMDetailTabs defaultTab="overview">
    <VMOverviewTab vmContext={vmContext} />
    <VMJobsTab jobs={jobs} />
    <VMNetworkTab networkConfig={networkConfig} />
    <VMDetailsTab vmSpecs={vmSpecs} />
    <VMCBTTab cbtHistory={cbtHistory} />
  </VMDetailTabs>
</VMDetailPage>

// ❌ BAD: Monolithic component
<VMDetailPageWithEverythingInOne />
```

### **2. Props Interface Pattern**
```typescript
// ✅ GOOD: Clear, typed interfaces
interface VMTableProps {
  vms: VMContextSummary[];
  selectedVM?: string;
  onVMSelect: (vmName: string) => void;
  onRefresh: () => void;
  loading?: boolean;
  error?: string;
}

// ❌ BAD: Unclear props
interface VMTableProps {
  data: any;
  onClick: Function;
  loading: boolean;
}
```

### **3. Custom Hooks Pattern**
```typescript
// ✅ GOOD: Reusable logic in hooks
function useVMContext(vmName: string) {
  const [data, setData] = useState<VMContextDetails | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  
  // Implementation
  
  return { data, loading, error, refetch };
}

// Usage
function VMDetailPage({ vmName }: { vmName: string }) {
  const { data: vmContext, loading, error, refetch } = useVMContext(vmName);
  
  if (loading) return <LoadingSpinner />;
  if (error) return <ErrorMessage error={error} onRetry={refetch} />;
  if (!vmContext) return <NotFound />;
  
  return <VMDetailTabs vmContext={vmContext} />;
}
```

## 🎨 **Styling Guidelines**

### **1. Tailwind CSS + Flowbite Strategy**
```typescript
// ✅ GOOD: Consistent utility classes
const cardClasses = "bg-white dark:bg-gray-800 rounded-lg shadow-sm border border-gray-200 dark:border-gray-700 p-6";
const buttonClasses = "bg-blue-600 hover:bg-blue-700 text-white font-medium rounded-lg text-sm px-5 py-2.5";

// ✅ GOOD: Component variants
const badgeVariants = {
  success: "bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-300",
  warning: "bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-300",
  error: "bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-300",
  info: "bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-300"
};
```

### **2. CSS Variables for Theming**
```css
/* globals.css */
:root {
  --color-primary: #3b82f6;
  --color-primary-hover: #2563eb;
  --color-success: #10b981;
  --color-warning: #f59e0b;
  --color-error: #ef4444;
  --color-text: #111827;
  --color-text-secondary: #6b7280;
  --border-radius: 0.5rem;
  --shadow-sm: 0 1px 2px 0 rgb(0 0 0 / 0.05);
  --shadow-md: 0 4px 6px -1px rgb(0 0 0 / 0.1);
}

[data-theme="dark"] {
  --color-text: #f9fafb;
  --color-text-secondary: #d1d5db;
}
```

## 📊 **State Management Strategy**

### **1. Zustand Store Pattern**
```typescript
// store/vmStore.ts
interface VMStore {
  // State
  vms: Record<string, VMContextDetails>;
  selectedVM: string | null;
  loading: boolean;
  error: string | null;
  
  // Actions
  setSelectedVM: (vmName: string | null) => void;
  fetchVMContext: (vmName: string) => Promise<void>;
  updateVMContext: (vmName: string, context: Partial<VMContextDetails>) => void;
  refreshAllVMs: () => Promise<void>;
}

export const useVMStore = create<VMStore>((set, get) => ({
  vms: {},
  selectedVM: null,
  loading: false,
  error: null,
  
  setSelectedVM: (vmName) => set({ selectedVM: vmName }),
  
  fetchVMContext: async (vmName) => {
    set({ loading: true, error: null });
    try {
      const context = await api.getVMContext(vmName);
      set((state) => ({
        vms: { ...state.vms, [vmName]: context },
        loading: false
      }));
    } catch (error) {
      set({ error: error.message, loading: false });
    }
  },
  
  // More actions...
}));
```

### **2. React Query Integration**
```typescript
// hooks/useVMContext.ts
import { useQuery } from '@tanstack/react-query';

export function useVMContext(vmName: string) {
  return useQuery({
    queryKey: ['vmContext', vmName],
    queryFn: () => api.getVMContext(vmName),
    enabled: !!vmName,
    refetchInterval: 5000, // Real-time updates
    staleTime: 2000,
  });
}

export function useVMList() {
  return useQuery({
    queryKey: ['vmContexts'],
    queryFn: () => api.getVMContexts(),
    refetchInterval: 30000, // Less frequent for list
  });
}
```

## 🔧 **API Integration Patterns**

### **1. Type-Safe API Client**
```typescript
// lib/api.ts
import { VMContextDetails, VMContextSummary } from './types';

class APIClient {
  private baseURL = process.env.NODE_ENV === 'production' 
    ? 'http://localhost:8082' 
    : 'http://localhost:8082';

  async getVMContexts(): Promise<{ vm_contexts: VMContextSummary[]; count: number }> {
    const response = await fetch(`${this.baseURL}/api/v1/vm-contexts`, {
      headers: { 'Authorization': `Bearer ${this.getToken()}` }
    });
    
    if (!response.ok) {
      throw new Error(`Failed to fetch VM contexts: ${response.statusText}`);
    }
    
    return response.json();
  }

  async getVMContext(vmName: string): Promise<VMContextDetails> {
    const response = await fetch(`${this.baseURL}/api/v1/vm-contexts/${vmName}`, {
      headers: { 'Authorization': `Bearer ${this.getToken()}` }
    });
    
    if (!response.ok) {
      if (response.status === 404) {
        throw new Error(`VM context not found: ${vmName}`);
      }
      throw new Error(`Failed to fetch VM context: ${response.statusText}`);
    }
    
    return response.json();
  }

  private getToken(): string {
    // Token management logic
    return localStorage.getItem('auth_token') || '';
  }
}

export const api = new APIClient();
```

### **2. Next.js API Routes Pattern**
```typescript
// app/api/vm-contexts/route.ts
import { NextRequest, NextResponse } from 'next/server';
import { api } from '@/lib/api';

export async function GET(request: NextRequest) {
  try {
    const data = await api.getVMContexts();
    return NextResponse.json(data);
  } catch (error) {
    console.error('API Error:', error);
    return NextResponse.json(
      { error: 'Failed to fetch VM contexts' },
      { status: 500 }
    );
  }
}
```

## 🚨 **Error Handling Patterns**

### **1. Error Boundary Component**
```typescript
// components/common/ErrorBoundary.tsx
'use client';

interface ErrorBoundaryProps {
  children: React.ReactNode;
  fallback?: React.ComponentType<{ error: Error; reset: () => void }>;
}

export function ErrorBoundary({ children, fallback: Fallback }: ErrorBoundaryProps) {
  return (
    <React.ErrorBoundary
      fallback={Fallback || DefaultErrorFallback}
      onError={(error, errorInfo) => {
        console.error('Error Boundary:', error, errorInfo);
        // Send to error tracking service
      }}
    >
      {children}
    </React.ErrorBoundary>
  );
}

function DefaultErrorFallback({ error, reset }: { error: Error; reset: () => void }) {
  return (
    <div className="min-h-[400px] flex items-center justify-center">
      <Card className="p-6 max-w-md">
        <h3 className="text-lg font-semibold text-red-600 mb-2">Something went wrong</h3>
        <p className="text-gray-600 mb-4">{error.message}</p>
        <Button onClick={reset} variant="outline">
          Try again
        </Button>
      </Card>
    </div>
  );
}
```

## ⚡ **Performance Guidelines**

### **1. Component Optimization**
```typescript
// ✅ GOOD: Memoized components
const VMTableRow = React.memo(({ vm, onSelect }: VMTableRowProps) => {
  return (
    <tr onClick={() => onSelect(vm.vm_name)} className="hover:bg-gray-50">
      <td>{vm.vm_name}</td>
      <td><VMStatusBadge status={vm.current_status} /></td>
      <td>{vm.total_jobs_run}</td>
    </tr>
  );
});

// ✅ GOOD: Optimized callbacks
const VMTable = ({ vms }: VMTableProps) => {
  const handleVMSelect = useCallback((vmName: string) => {
    // Selection logic
  }, []);

  return (
    <table>
      {vms.map(vm => (
        <VMTableRow key={vm.vm_name} vm={vm} onSelect={handleVMSelect} />
      ))}
    </table>
  );
};
```

### **2. Loading Patterns**
```typescript
// ✅ GOOD: Skeleton loading
function VMTableSkeleton() {
  return (
    <div className="space-y-4">
      {Array.from({ length: 5 }).map((_, i) => (
        <div key={i} className="animate-pulse flex space-x-4">
          <div className="h-4 bg-gray-200 rounded w-1/4"></div>
          <div className="h-4 bg-gray-200 rounded w-1/6"></div>
          <div className="h-4 bg-gray-200 rounded w-1/8"></div>
        </div>
      ))}
    </div>
  );
}

// Usage
function VMTable() {
  const { data: vms, loading } = useVMList();
  
  if (loading) return <VMTableSkeleton />;
  
  return <ActualVMTable vms={vms} />;
}
```

## 🧪 **Testing Strategy**

### **1. Component Testing**
```typescript
// __tests__/components/VMTable.test.tsx
import { render, screen, fireEvent } from '@testing-library/react';
import { VMTable } from '@/components/vm/VMTable';

const mockVMs = [
  { vm_name: 'test-vm-1', current_status: 'replicating', total_jobs_run: 2 },
  { vm_name: 'test-vm-2', current_status: 'ready', total_jobs_run: 0 }
];

describe('VMTable', () => {
  it('renders VM list correctly', () => {
    render(<VMTable vms={mockVMs} onVMSelect={jest.fn()} />);
    
    expect(screen.getByText('test-vm-1')).toBeInTheDocument();
    expect(screen.getByText('test-vm-2')).toBeInTheDocument();
  });

  it('calls onVMSelect when VM is clicked', () => {
    const onVMSelect = jest.fn();
    render(<VMTable vms={mockVMs} onVMSelect={onVMSelect} />);
    
    fireEvent.click(screen.getByText('test-vm-1'));
    expect(onVMSelect).toHaveBeenCalledWith('test-vm-1');
  });
});
```

## 📱 **Responsive Design Rules**

### **1. Mobile-First Approach**
```typescript
// ✅ GOOD: Responsive component
function VMTable({ vms }: VMTableProps) {
  return (
    <div className="overflow-x-auto">
      <table className="min-w-full divide-y divide-gray-200">
        <thead className="bg-gray-50">
          <tr>
            <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
              VM Name
            </th>
            <th className="hidden sm:table-cell px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
              Status
            </th>
            <th className="hidden md:table-cell px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
              Jobs
            </th>
          </tr>
        </thead>
      </table>
    </div>
  );
}
```

---

## 🎯 **Summary Rules**

1. **Always use TypeScript** - No `any` types
2. **Component modularity** - Small, focused components
3. **Custom hooks** - Extract reusable logic
4. **Error boundaries** - Wrap components properly
5. **Loading states** - Always show user feedback
6. **Performance** - Memoize when needed
7. **Accessibility** - WCAG 2.1 AA compliance
8. **Testing** - Unit tests for all components
9. **Responsive** - Mobile-first design
10. **Documentation** - Document complex components

**This foundation ensures we build a production-ready, maintainable GUI! 🚀**
