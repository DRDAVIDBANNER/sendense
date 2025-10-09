# 🎯 GROK: Protection Flows GUI - Final Integration

Hey Grok! 👋

You've done fantastic work on the backend (Tasks 1-5 complete). Now we're at the final GUI integration phase, but we have TWO problems to solve:

---

## 🚨 Problem 1: Light Mode Support Broken (Your Fault 😅)

During your modal-to-panel refactoring, you hardcoded dark mode colors everywhere:
- `bg-gray-900` instead of `bg-background`
- `text-white` instead of `text-foreground`  
- `border-gray-700` instead of `border-border`

**This broke light mode support.** Every other page in the GUI supports light/dark mode switching. Protection Flows is the ONLY broken page.

---

## 🚨 Problem 2: Still Using Mock Data

The Protection Flows page is still using mock data. Backend API is ready and deployed at `http://localhost:8082/api/v1/protection-flows`, but GUI isn't using it yet.

---

## 📋 Your Task

Read the comprehensive job sheet at:
**`/home/oma_admin/sendense/job-sheets/GROK-TASK-6-gui-wiring-plus-theme-fix.md`**

This job sheet contains:
1. ✅ Complete explanation of the theme system
2. ✅ All files that need fixing
3. ✅ Before/after code examples for theme fixes
4. ✅ Complete API service layer code
5. ✅ Complete React Query hooks code
6. ✅ Component update patterns
7. ✅ Completion checklist
8. ✅ Testing instructions

---

## 🎯 Execution Order

### Step 1: Fix Theme Support (DO THIS FIRST) ⚡
1. Fix `app/protection-flows/page.tsx` - replace ALL hardcoded colors
2. Fix `components/features/protection-flows/FlowDetailsPanel.tsx` - replace ALL hardcoded colors
3. Fix `components/features/protection-flows/JobLogsDrawer.tsx` - replace ALL hardcoded colors
4. Test in light mode (remove `className="dark"` from `app/layout.tsx`)
5. Test in dark mode (restore `className="dark"`)

**Show me a git diff after this step.**

### Step 2: Wire API Integration 🔌

**⚠️ CRITICAL FIX REQUIRED:** The API file already exists but uses `http://localhost:8082` which bypasses the Next.js proxy!

**Fix in `src/features/protection-flows/api/protectionFlowsApi.ts` line 3:**
```typescript
// ❌ WRONG (current)
const API_BASE = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8082';

// ✅ CORRECT (change to this)
const API_BASE = '';  // Empty string uses Next.js rewrites proxy
```

**Why:** Next.js has a rewrite rule that proxies `/api/v1/*` to `http://localhost:8082/api/v1/*`. Using relative URLs avoids CORS issues and works in production.

**Then:**
1. Fix API_BASE in `protectionFlowsApi.ts` (line 3)
2. Create React Query hooks: `src/features/protection-flows/hooks/useProtectionFlows.ts`
3. Update `page.tsx` to use real API hooks
4. Update `FlowDetailsPanel.tsx` to use real execution data
5. Remove ALL mock data
6. Add loading states
7. Add error handling
8. Test CRUD operations

**Show me a git diff after this step.**

---

## ✅ Success Criteria

**Theme Fix:**
- [ ] No hardcoded `bg-gray-*` anywhere
- [ ] No hardcoded `text-white` anywhere  
- [ ] No hardcoded `border-gray-*` anywhere
- [ ] Page looks professional in BOTH light and dark mode

**API Wiring:**
- [ ] No mock data remaining
- [ ] All CRUD operations use real API
- [ ] Loading states during operations
- [ ] Error handling for failures
- [ ] Auto-refresh for live updates
- [ ] `npm run build` succeeds with no errors

---

## 📚 Key Reference

**Theme System Tokens (from globals.css):**
```css
/* Use these Tailwind classes: */
bg-background        /* Page background */
text-foreground      /* Primary text */
bg-card              /* Card/panel background */
text-card-foreground /* Card text */
bg-muted             /* Secondary background */
text-muted-foreground /* Secondary text */
border-border        /* Borders */
bg-primary           /* Primary action color */
text-primary         /* Primary text color */
```

**API URLs (CRITICAL):**
```typescript
// ❌ WRONG - Do NOT use absolute URL
const API_BASE = 'http://localhost:8082';

// ✅ CORRECT - Use empty string for Next.js proxy
const API_BASE = '';

// Then all calls use relative URLs:
axios.get('/api/v1/protection-flows')  // ✅ Proxied by Next.js
```

**Next.js Proxy Setup (already configured in next.config.ts):**
```typescript
async rewrites() {
  return [
    {
      source: '/api/v1/:path*',
      destination: 'http://localhost:8082/api/v1/:path*',
    },
  ];
}
```

---

## 🔥 Let's Go!

1. Read the full job sheet
2. Fix theme support FIRST (show diff)
3. Wire API integration SECOND (show diff)
4. Run completion checklist
5. Confirm all tests pass

You got this! 💪

