# GUI Development Rules - Quick Reference

## 🚨 **CRITICAL RULES - NEVER BREAK**

### **TypeScript & Code Quality**
- ✅ **100% TypeScript** - NO `any` types, use strict mode
- ✅ **Explicit interfaces** for all props and data structures
- ✅ **Components max 200 lines** - single responsibility principle
- ✅ **React.memo** for performance optimization
- ✅ **useCallback/useMemo** for expensive operations

### **Component Structure**
- ✅ **File naming**: PascalCase for components, camelCase for hooks
- ✅ **Component pattern**: Props interface → React.memo → hooks → handlers → early returns → JSX
- ✅ **Error boundaries** wrap all major component sections
- ✅ **Loading states** for ALL async operations
- ✅ **Test files** for every component

### **API & Data**
- ✅ **Custom hooks only** - NO direct API calls in components
- ✅ **React Query** for all server state management
- ✅ **5-second polling** for active job progress
- ✅ **30-second polling** for VM list updates
- ✅ **Error handling** with toast notifications

### **Styling & UI**
- ✅ **Tailwind CSS only** - NO custom CSS unless absolutely necessary
- ✅ **Flowbite React** components for consistency
- ✅ **Dark mode support** throughout
- ✅ **Responsive design** - mobile first approach
- ✅ **WCAG 2.1 AA** accessibility compliance

## 📋 **Code Patterns**

### **Component Template**
```typescript
interface ComponentProps {
  data: SpecificType;
  onAction: (param: string) => void;
  loading?: boolean;
}

export const Component = React.memo(({ data, onAction, loading = false }: ComponentProps) => {
  const handleClick = useCallback((id: string) => onAction(id), [onAction]);
  
  if (loading) return <LoadingSkeleton />;
  
  return <div className="bg-white dark:bg-gray-800">{/* JSX */}</div>;
});
```

### **Custom Hook Template**
```typescript
export function useFeature(param: string) {
  return useQuery({
    queryKey: ['feature', param],
    queryFn: () => api.getFeature(param),
    enabled: !!param,
    refetchInterval: 5000,
    onError: (error) => toast.error(`Failed: ${error.message}`)
  });
}
```

### **Error Boundary Usage**
```typescript
<ErrorBoundary fallback={ErrorFallback}>
  <ComponentThatMightFail />
</ErrorBoundary>
```

## 🎯 **File Structure Rules**

```
src/
├── app/                    # Next.js pages (App Router)
├── components/
│   ├── ui/                # Base components
│   ├── layout/            # Layout components
│   ├── vm/                # VM-specific
│   ├── jobs/              # Job-related
│   └── common/            # Utilities
├── lib/                   # API client, types, utils
├── hooks/                 # Custom React hooks
├── store/                 # Zustand stores
└── styles/                # Global styles
```

## 🔧 **Development Checklist**

### **Before Writing Code**
- [ ] Read existing components for similar functionality
- [ ] Plan component props interface
- [ ] Identify required custom hooks
- [ ] Design loading and error states

### **Implementation**
- [ ] Create TypeScript interfaces first
- [ ] Implement loading/error states
- [ ] Add error boundaries
- [ ] Make responsive
- [ ] Write unit tests
- [ ] Test accessibility

### **Code Review**
- [ ] TypeScript strict mode passes
- [ ] No `any` types
- [ ] Error boundaries present
- [ ] Loading states work
- [ ] Responsive design
- [ ] Unit tests pass
- [ ] Accessibility tested

## 🎨 **Common Class Patterns**

```typescript
// Cards
const cardClasses = "bg-white dark:bg-gray-800 rounded-lg shadow-sm border border-gray-200 dark:border-gray-700 p-6";

// Status badges
const statusClasses = {
  success: "bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-300",
  error: "bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-300"
};

// Buttons
const buttonClasses = "bg-blue-600 hover:bg-blue-700 text-white font-medium rounded-lg text-sm px-5 py-2.5";
```

## 🚨 **Common Mistakes to Avoid**

- ❌ Using `any` types
- ❌ Direct API calls in components
- ❌ Missing loading states
- ❌ No error boundaries
- ❌ Ignoring accessibility
- ❌ Not using React.memo
- ❌ Custom CSS instead of Tailwind
- ❌ Missing TypeScript interfaces
- ❌ No unit tests
- ❌ Breaking file structure

## 🎯 **Success Metrics**

- **Performance**: Page loads < 2 seconds
- **Accessibility**: WCAG 2.1 AA compliant
- **Quality**: 100% TypeScript strict mode
- **Testing**: >80% test coverage
- **Maintainability**: Components < 200 lines

---

**Remember: Production enterprise application - quality first! 🚀**
