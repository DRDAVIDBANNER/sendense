# GROK PROMPT: Job Logs Collapsible Drawer

**Task:** Convert always-visible Job Logs panel to collapsible slide-out drawer  
**Priority:** HIGH  
**Expected Duration:** 1-2 hours

---

## 🎯 MISSION

Replace the current always-visible Job Logs panel with a **collapsible drawer** that:
- Starts hidden (collapsed)
- Slides out from right edge when clicked
- Is resizable by dragging left edge
- Persists state and width in localStorage
- Has smooth animations (60fps)

---

## 📸 BEFORE vs AFTER

### **BEFORE (Current - Always Visible):**
```
┌─────────────────────────┬─────────────┐
│ Main Content (70%)      │ Job Logs    │ ← Always visible
│                         │ (30%)       │    Takes space
└─────────────────────────┴─────────────┘
```

### **AFTER (Collapsible Drawer):**

**Collapsed (Default):**
```
┌───────────────────────────────────────────┐
│ Backup & Replication Jobs      [+] [📋]  │ ← Toggle button
│ (Main content uses full width)            │
└───────────────────────────────────────────┘
```

**Expanded (Slides Out):**
```
┌─────────────────────────┬─────────────┐
│ Main Content            │║ Job Logs ✕ │ ← Slides in
│ (Adjusts width)         │║ Live 🟢    │    Resizable
│                         │║ [logs...]  │
└─────────────────────────┴─────────────┘
                          ↑ Drag to resize
```

---

## 🔧 IMPLEMENTATION STEPS

### **Step 1: Create JobLogsDrawer Component**

**File:** `components/features/protection-flows/JobLogsDrawer.tsx`

See the complete implementation in the job sheet at:
`/home/oma_admin/sendense/job-sheets/2025-10-09-gui-job-logs-collapsible-drawer.md`

**Key Features:**
- `isOpen` prop controls visibility
- Fixed positioning (right: 0, top: 0, bottom: 0)
- Resizable via drag handle on left edge
- Width range: 300px (min) to 600px (max)
- localStorage persistence for width
- Smooth animation: `transition: width 300ms ease-in-out`
- Same log display logic as before

---

### **Step 2: Update Protection Flows Page**

**File:** `app/protection-flows/page.tsx`

**Changes:**
1. Remove nested horizontal PanelGroup
2. Keep only vertical PanelGroup (table + details)
3. Add toggle button in header (next to "Create Flow")
4. Add JobLogsDrawer at bottom (outside PanelGroup)
5. Add state management for isLogsOpen
6. Add localStorage persistence
7. Add Ctrl+L keyboard shortcut

**See complete code in job sheet.**

---

### **Step 3: Update Exports**

**File:** `components/features/protection-flows/index.tsx`

Add:
```typescript
export { JobLogsDrawer } from './JobLogsDrawer'
```

---

### **Step 4: Delete Old Component**

**Action:** DELETE this file completely:
```
components/features/protection-flows/JobLogsPanel.tsx
```

Verify no imports remain:
```bash
grep -r "JobLogsPanel" components/ app/
# Should return zero results
```

---

## 🎨 DESIGN SPECIFICATIONS

### **Toggle Button (Top Right Header):**
```typescript
<button
  onClick={() => setIsLogsOpen(!isLogsOpen)}
  className={`p-2 rounded-lg transition-colors ${
    isLogsOpen
      ? 'bg-blue-500/20 text-blue-400'
      : 'bg-gray-700 text-gray-400 hover:bg-gray-600'
  }`}
  title="Job Logs (Ctrl+L)"
>
  <HiClipboardList className="h-5 w-5" />
</button>
```

### **Drawer Container:**
```typescript
<div
  className="fixed top-0 bottom-0 right-0 bg-gray-900 border-l border-gray-700 flex flex-col z-50 shadow-2xl"
  style={{ 
    width: `${width}px`,
    transition: isResizing ? 'none' : 'width 300ms ease-in-out'
  }}
>
```

### **Resize Handle:**
```typescript
<div
  onMouseDown={handleMouseDown}
  className={`fixed top-0 bottom-0 w-1 bg-gray-700 hover:bg-blue-500 cursor-ew-resize z-40 transition-colors ${
    isResizing ? 'bg-blue-500' : ''
  }`}
  style={{ right: width }}
/>
```

### **Animation Details:**
```typescript
Slide In/Out: width property animates (300ms ease-in-out)
No Transition While Resizing: transition: 'none' when isResizing = true
Smooth at 60fps: Use CSS transitions, not JavaScript animation
```

---

## ✅ ACCEPTANCE CRITERIA

**Visual:**
- [ ] Toggle button visible in top right (next to Create Flow)
- [ ] Button icon: HiClipboardList
- [ ] Button blue when drawer open, gray when closed
- [ ] Drawer hidden by default
- [ ] Drawer slides smoothly from right (no jank)
- [ ] Main content adjusts width when drawer opens

**Interaction:**
- [ ] Click button → Drawer slides in
- [ ] Click button again → Drawer slides out
- [ ] Click X in drawer → Drawer closes
- [ ] Ctrl+L keyboard shortcut toggles drawer
- [ ] Hover left edge → ew-resize cursor
- [ ] Drag left edge → Resize drawer (300-600px)
- [ ] Release drag → Width persists

**Persistence:**
- [ ] Open drawer, reload → Still open
- [ ] Resize drawer, reload → Width remembered
- [ ] Close drawer, reload → Still closed

**Code Quality:**
- [ ] JobLogsPanel.tsx deleted completely
- [ ] No imports of JobLogsPanel remain
- [ ] Production build succeeds (`npm run build`)
- [ ] Zero TypeScript errors
- [ ] Zero console warnings
- [ ] Smooth 60fps animation

---

## 🧪 TESTING CHECKLIST

```bash
# 1. Start dev server
cd /home/oma_admin/sendense/source/current/sendense-gui
npm run dev

# 2. Navigate to page
http://localhost:3000/protection-flows

# 3. Test collapsed state
✓ No drawer visible
✓ Toggle button visible (top right)
✓ Main content full width

# 4. Test open animation
✓ Click button → Smooth slide in (300ms)
✓ Main content narrows
✓ Button turns blue

# 5. Test close animation
✓ Click X → Smooth slide out (300ms)
✓ Main content expands back
✓ Button turns gray

# 6. Test keyboard shortcut
✓ Press Ctrl+L → Drawer toggles
✓ Works both open and closed

# 7. Test resize
✓ Open drawer
✓ Hover left edge → Cursor changes (ew-resize)
✓ Drag left → Drawer grows (watch max 600px)
✓ Drag right → Drawer shrinks (watch min 300px)
✓ Edge turns blue while dragging

# 8. Test persistence
✓ Open drawer → Reload page → Still open
✓ Resize to 500px → Reload → Still 500px
✓ Close drawer → Reload → Still closed

# 9. Test logs functionality
✓ Auto-scroll works
✓ Filter dropdown works
✓ Clear button works
✓ Log colors correct

# 10. Production build
npm run build
✓ Completes successfully
✓ All pages generate
```

---

## ⚠️ CRITICAL IMPLEMENTATION DETAILS

### **1. Resize Logic (IMPORTANT):**
```typescript
const handleMouseMove = (e: MouseEvent) => {
  if (!isResizing) return
  
  // Calculate width from right edge
  const newWidth = window.innerWidth - e.clientX
  
  // Enforce min/max
  if (newWidth >= 300 && newWidth <= 600) {
    setWidth(newWidth)
  }
}
```

### **2. State Persistence:**
```typescript
// Load on mount
useEffect(() => {
  const savedState = localStorage.getItem('jobLogsOpen')
  const savedWidth = localStorage.getItem('jobLogsWidth')
  if (savedState) setIsLogsOpen(JSON.parse(savedState))
  if (savedWidth) setWidth(parseInt(savedWidth))
}, [])

// Save on change
useEffect(() => {
  localStorage.setItem('jobLogsOpen', JSON.stringify(isLogsOpen))
}, [isLogsOpen])

useEffect(() => {
  localStorage.setItem('jobLogsWidth', width.toString())
}, [width])
```

### **3. Keyboard Shortcut:**
```typescript
useEffect(() => {
  const handleKeyDown = (e: KeyboardEvent) => {
    if (e.ctrlKey && e.key === 'l') {
      e.preventDefault()
      setIsLogsOpen(prev => !prev)
    }
  }

  window.addEventListener('keydown', handleKeyDown)
  return () => window.removeEventListener('keydown', handleKeyDown)
}, [])
```

### **4. No Transition While Dragging:**
```typescript
style={{ 
  width: `${width}px`,
  transition: isResizing ? 'none' : 'width 300ms ease-in-out'
}}
```

---

## 📝 EXPECTED COMMIT

```
refactor: convert job logs to collapsible drawer

- Replaced always-visible JobLogsPanel with JobLogsDrawer
- Toggle button in top right header (HiClipboardList)
- Smooth slide animation (300ms ease-in-out)
- Resizable by dragging left edge (300-600px)
- State persistence in localStorage (open/closed + width)
- Keyboard shortcut: Ctrl+L
- No overlay (pushes content left)
- Production build successful

Files:
- NEW: components/features/protection-flows/JobLogsDrawer.tsx
- MODIFIED: app/protection-flows/page.tsx
- MODIFIED: components/features/protection-flows/index.tsx
- DELETED: components/features/protection-flows/JobLogsPanel.tsx

Breaking changes: None
```

---

## 🎯 SUCCESS METRICS

**Space Efficiency:**
- Collapsed: 100% screen width available
- Expanded: User chooses width (300-600px)

**User Experience:**
- Smooth 60fps animations
- Instant toggle response
- Intuitive resize drag
- Keyboard shortcut power user

**Code Quality:**
- Production build passes
- Zero TypeScript errors
- Clean component architecture
- Proper cleanup (delete old file)

---

## 💡 PRO TIPS

1. **Test resize thoroughly** - It's the trickiest part
2. **Verify localStorage** - Open DevTools → Application → Local Storage
3. **Watch for jank** - Animation should be butter smooth
4. **Delete old file** - `JobLogsPanel.tsx` must go
5. **Build before commit** - `npm run build` must succeed

---

**Priority:** HIGH  
**Complexity:** Medium-High  
**Estimated Time:** 1-2 hours

Good luck! This is professional production code - make it smooth and efficient! 🚀

