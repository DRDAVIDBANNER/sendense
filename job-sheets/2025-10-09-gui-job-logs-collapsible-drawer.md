# GUI Job Logs Collapsible Drawer - Job Sheet

**Date:** October 9, 2025  
**Assignee:** Grok Code Fast  
**Priority:** HIGH  
**Type:** GUI Refactoring  
**Status:** 🟡 PENDING

---

## 🎯 OBJECTIVE

Convert the Job Logs panel from always-visible to a **collapsible drawer** that slides out from the right when needed.

---

## 📸 DESIRED BEHAVIOR

### **Collapsed State (Default):**
```
┌─────────────────────────────────────────────────┐
│ Backup & Replication Jobs               [+] [📋]│ ← Tab button
├─────────────────────────────────────────────────┤
│ Name | Type | Status | Last Run | Actions      │
│ Critical DB Backup  | ...                       │
│ Daily VM Backup     | ...                       │
└─────────────────────────────────────────────────┘
═══════════════════════════════════════════════════
┌─────────────────────────────────────────────────┐
│ [Flow Details Panel]                            │
│ • Machines (3 VMs)                              │
│ • Jobs & Progress                               │
│ • Performance Charts                            │
└─────────────────────────────────────────────────┘
```

### **Expanded State (Slides Out):**
```
┌─────────────────────────────────┬───────────────┐
│ Backup & Replication Jobs   [+] │ Job Logs [✕]  │
├─────────────────────────────────┼───────────────┤
│ Name | Type | Status | ...      │ Live 🟢       │
│ Critical DB Backup  | ...       │ ───────────── │
│ Daily VM Backup     | ...       │ 11:00:01      │
└─────────────────────────────────┤ [INFO]        │
═══════════════════════════════════ Starting...   │
┌─────────────────────────────────┤               │
│ [Flow Details Panel]            │ 11:00:02      │
│ • Machines (3 VMs)              │ [WARNING]     │
│ • Jobs & Progress               │ Network...    │
└─────────────────────────────────┴───────────────┘
  ↑ Content adjusts            ↑ Resizable drawer
```

---

## 🔧 IMPLEMENTATION REQUIREMENTS

### **1. Collapsible Tab Button (Top Right):**
- Location: Top right corner of page header
- Icon: HiClipboardList or HiDocumentText
- Badge: "Live" indicator when logs active
- Tooltip: "Job Logs (Ctrl+L)"
- Click: Toggle drawer open/closed
- Keyboard: Ctrl+L to toggle

### **2. Drawer Panel:**
- Slides from right edge
- Default width: 400px (25% of screen)
- Min width: 300px
- Max width: 600px (40% of screen)
- Resizable: Drag left edge to resize
- Animation: Smooth slide (300ms ease-in-out)
- Overlay or Push: Push content left (no overlay)

### **3. Drawer Header:**
- Title: "Job Logs"
- Live badge: Green pulse animation
- Close button: X icon (top right)
- Controls: Filter dropdown, Auto-scroll toggle, Clear
- Sticky: Header stays at top on scroll

### **4. State Management:**
- Persist state: localStorage ('jobLogsOpen': boolean, 'jobLogsWidth': number)
- Default: Closed on page load
- Remember: Width preference when reopened

---

## 📝 IMPLEMENTATION CODE

### **File 1: Create Drawer Component**

**File:** `components/features/protection-flows/JobLogsDrawer.tsx`

```typescript
'use client'

import React, { useState, useEffect, useRef } from 'react'
import { Button } from 'flowbite-react'
import { HiX, HiFilter } from 'react-icons/hi'

interface LogEntry {
  time: string
  level: 'INFO' | 'WARNING' | 'ERROR' | 'SUCCESS'
  message: string
  component?: string
}

const mockLogs: LogEntry[] = [
  { time: '11:00:01', level: 'INFO', message: 'Starting backup job for', component: 'Backup Engine' },
  { time: '11:00:02', level: 'INFO', message: 'Connecting to vCenter serv', component: 'VMware API' },
  { time: '11:00:03', level: 'INFO', message: 'Snapshot created successfu', component: 'VMware API' },
  { time: '11:00:04', level: 'INFO', message: 'Transferring data: 25% c', component: 'NBD Transfer' },
  { time: '11:00:05', level: 'WARNING', message: 'Network latency detected, adjusting buffer size', component: 'NBD Transfer' },
  { time: '11:00:06', level: 'INFO', message: 'Transferring data: 50% c', component: 'NBD Transfer' },
  { time: '11:00:07', level: 'INFO', message: 'Transferring data: 75% c', component: 'NBD Transfer' },
  { time: '11:00:08', level: 'INFO', message: 'Transferring data: 100%', component: 'NBD Transfer' },
  { time: '11:00:09', level: 'INFO', message: 'Verifying backup integrity', component: 'Validation Engine' },
  { time: '11:00:10', level: 'INFO', message: 'Backup completed successfully', component: 'Backup Engine' },
]

const logLevelColors = {
  INFO: 'text-blue-400',
  WARNING: 'text-yellow-400',
  ERROR: 'text-red-400',
  SUCCESS: 'text-green-400',
}

interface JobLogsDrawerProps {
  isOpen: boolean
  onClose: () => void
}

export function JobLogsDrawer({ isOpen, onClose }: JobLogsDrawerProps) {
  const [logs, setLogs] = useState<LogEntry[]>(mockLogs)
  const [autoScroll, setAutoScroll] = useState(true)
  const [filter, setFilter] = useState<'All' | 'INFO' | 'WARNING' | 'ERROR'>('All')
  const [width, setWidth] = useState(400)
  const [isResizing, setIsResizing] = useState(false)
  const scrollRef = useRef<HTMLDivElement>(null)
  const drawerRef = useRef<HTMLDivElement>(null)

  // Load saved width from localStorage
  useEffect(() => {
    const savedWidth = localStorage.getItem('jobLogsWidth')
    if (savedWidth) {
      setWidth(parseInt(savedWidth))
    }
  }, [])

  // Save width to localStorage
  useEffect(() => {
    localStorage.setItem('jobLogsWidth', width.toString())
  }, [width])

  // Auto-scroll to bottom
  useEffect(() => {
    if (autoScroll && scrollRef.current) {
      scrollRef.current.scrollTop = scrollRef.current.scrollHeight
    }
  }, [logs, autoScroll])

  // Handle resize
  const handleMouseDown = (e: React.MouseEvent) => {
    e.preventDefault()
    setIsResizing(true)
  }

  useEffect(() => {
    const handleMouseMove = (e: MouseEvent) => {
      if (!isResizing) return
      
      const newWidth = window.innerWidth - e.clientX
      if (newWidth >= 300 && newWidth <= 600) {
        setWidth(newWidth)
      }
    }

    const handleMouseUp = () => {
      setIsResizing(false)
    }

    if (isResizing) {
      document.addEventListener('mousemove', handleMouseMove)
      document.addEventListener('mouseup', handleMouseUp)
    }

    return () => {
      document.removeEventListener('mousemove', handleMouseMove)
      document.removeEventListener('mouseup', handleMouseUp)
    }
  }, [isResizing])

  const filteredLogs = filter === 'All' 
    ? logs 
    : logs.filter(log => log.level === filter)

  const handleClear = () => {
    setLogs([])
  }

  if (!isOpen) return null

  return (
    <>
      {/* Resize Handle */}
      <div
        onMouseDown={handleMouseDown}
        className={`fixed top-0 bottom-0 w-1 bg-gray-700 hover:bg-blue-500 cursor-ew-resize z-40 transition-colors ${
          isResizing ? 'bg-blue-500' : ''
        }`}
        style={{ right: width }}
      />

      {/* Drawer */}
      <div
        ref={drawerRef}
        className="fixed top-0 bottom-0 right-0 bg-gray-900 border-l border-gray-700 flex flex-col z-50 shadow-2xl"
        style={{ 
          width: `${width}px`,
          transition: isResizing ? 'none' : 'width 300ms ease-in-out'
        }}
      >
        {/* Header */}
        <div className="flex items-center justify-between px-4 py-3 border-b border-gray-700 bg-gray-800/50 shrink-0">
          <div className="flex items-center gap-2">
            <h3 className="text-sm font-semibold text-white">Job Logs</h3>
            <span className="px-2 py-0.5 text-xs bg-green-500/20 text-green-400 rounded flex items-center gap-1">
              <span className="w-2 h-2 bg-green-500 rounded-full animate-pulse"></span>
              Live
            </span>
          </div>
          
          <div className="flex items-center gap-2">
            <select
              value={filter}
              onChange={(e) => setFilter(e.target.value as any)}
              className="text-xs bg-gray-700 border-gray-600 text-white rounded px-2 py-1"
            >
              <option value="All">All</option>
              <option value="INFO">Info</option>
              <option value="WARNING">Warning</option>
              <option value="ERROR">Error</option>
            </select>
            
            <button
              onClick={() => setAutoScroll(!autoScroll)}
              className={`text-xs px-2 py-1 rounded ${
                autoScroll 
                  ? 'bg-blue-500/20 text-blue-400' 
                  : 'bg-gray-700 text-gray-400'
              }`}
            >
              Auto
            </button>
            
            <button
              onClick={handleClear}
              className="text-xs px-2 py-1 rounded bg-gray-700 text-gray-400 hover:bg-gray-600"
              title="Clear logs"
            >
              Clear
            </button>
            
            <button
              onClick={onClose}
              className="text-gray-400 hover:text-white p-1"
              title="Close (Ctrl+L)"
            >
              <HiX className="h-4 w-4" />
            </button>
          </div>
        </div>

        {/* Logs Display */}
        <div 
          ref={scrollRef}
          className="flex-1 overflow-auto p-2 space-y-0.5"
        >
          {filteredLogs.length === 0 ? (
            <div className="flex items-center justify-center h-full text-gray-500 text-sm">
              No logs to display
            </div>
          ) : (
            filteredLogs.map((log, index) => (
              <div
                key={index}
                className="px-2 py-1 font-mono text-xs hover:bg-gray-800/30 cursor-pointer rounded"
              >
                <span className="text-gray-500">{log.time}</span>
                {' '}
                <span className={`font-semibold ${logLevelColors[log.level]}`}>
                  [{log.level}]
                </span>
                {' '}
                <span className="text-gray-300">{log.message}</span>
                {log.component && (
                  <>
                    {' '}
                    <span className="text-blue-300">{log.component}</span>
                  </>
                )}
              </div>
            ))
          )}
        </div>
      </div>
    </>
  )
}
```

---

### **File 2: Update Protection Flows Page**

**File:** `app/protection-flows/page.tsx`

```typescript
'use client'

import React, { useState, useEffect } from 'react'
import { Panel, PanelGroup, PanelResizeHandle } from 'react-resizable-panels'
import { FlowsTable } from '@/components/features/protection-flows/FlowsTable'
import { FlowDetailsPanel } from '@/components/features/protection-flows/FlowDetailsPanel'
import { JobLogsDrawer } from '@/components/features/protection-flows/JobLogsDrawer'
import { Flow } from '@/lib/types'
import { Button } from 'flowbite-react'
import { HiPlus, HiClipboardList } from 'react-icons/hi'

export default function ProtectionFlowsPage() {
  const [selectedFlow, setSelectedFlow] = useState<Flow | null>(null)
  const [isLogsOpen, setIsLogsOpen] = useState(false)

  // Load saved state from localStorage
  useEffect(() => {
    const savedState = localStorage.getItem('jobLogsOpen')
    if (savedState) {
      setIsLogsOpen(JSON.parse(savedState))
    }
  }, [])

  // Save state to localStorage
  useEffect(() => {
    localStorage.setItem('jobLogsOpen', JSON.stringify(isLogsOpen))
  }, [isLogsOpen])

  // Keyboard shortcut: Ctrl+L
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

  return (
    <div className="h-screen bg-gray-900 relative">
      {/* Main Content */}
      <PanelGroup direction="vertical">
        {/* Top Panel: Flows Table */}
        <Panel defaultSize={50} minSize={30}>
          <div className="flex flex-col h-full bg-gray-900">
            {/* Compact header with Job Logs toggle */}
            <div className="flex items-center justify-between px-4 py-3 border-b border-gray-700 shrink-0">
              <div>
                <h2 className="text-lg font-semibold text-white">
                  Backup & Replication Jobs
                </h2>
                <p className="text-xs text-gray-400">
                  Manage and monitor your protection flows across all environments
                </p>
              </div>
              
              <div className="flex items-center gap-2">
                <Button color="blue" size="sm">
                  <HiPlus className="mr-2 h-4 w-4" />
                  Create Flow
                </Button>
                
                {/* Job Logs Toggle Button */}
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
              </div>
            </div>
            
            {/* Table */}
            <div className="flex-1 overflow-auto">
              <FlowsTable onFlowSelect={setSelectedFlow} selectedFlow={selectedFlow} />
            </div>
          </div>
        </Panel>
        
        {/* Vertical Resize Handle */}
        <PanelResizeHandle className="h-1 bg-gray-700 hover:bg-blue-500 transition-colors cursor-ns-resize" />
        
        {/* Bottom Panel: Flow Details */}
        <Panel defaultSize={40} minSize={20}>
          <div className="h-full bg-gray-900 border-t border-gray-700 overflow-auto">
            {selectedFlow ? (
              <FlowDetailsPanel flow={selectedFlow} />
            ) : (
              <div className="flex items-center justify-center h-full">
                <p className="text-gray-400 text-center">
                  Select a flow to view details
                </p>
              </div>
            )}
          </div>
        </Panel>
      </PanelGroup>

      {/* Job Logs Drawer (Slides from right) */}
      <JobLogsDrawer isOpen={isLogsOpen} onClose={() => setIsLogsOpen(false)} />
    </div>
  )
}
```

---

### **File 3: Export Drawer Component**

**File:** `components/features/protection-flows/index.tsx`

Add export:
```typescript
export { JobLogsDrawer } from './JobLogsDrawer'
```

---

### **File 4: Delete Old JobLogsPanel**

**Action:** DELETE this file:
```
components/features/protection-flows/JobLogsPanel.tsx
```

---

## ✅ ACCEPTANCE CRITERIA

**Collapsed State:**
- [ ] Job Logs button visible in top right header
- [ ] Icon: HiClipboardList
- [ ] Button highlighted when drawer open
- [ ] Tooltip shows "Job Logs (Ctrl+L)"
- [ ] Main content uses full width

**Expanded State:**
- [ ] Drawer slides in from right (300ms smooth animation)
- [ ] Default width: 400px
- [ ] Main content adjusts (gets narrower)
- [ ] No overlay/backdrop (push content mode)

**Resizing:**
- [ ] Drag left edge to resize drawer
- [ ] Min width: 300px
- [ ] Max width: 600px
- [ ] Cursor changes to ew-resize on hover
- [ ] Blue indicator when resizing
- [ ] Width persists in localStorage

**Functionality:**
- [ ] Click button → Toggle open/close
- [ ] Ctrl+L keyboard shortcut works
- [ ] Close button (X) in drawer header
- [ ] Auto-scroll, filter, clear all work
- [ ] State persists across page reloads

**Code Quality:**
- [ ] Production build succeeds
- [ ] Zero TypeScript errors
- [ ] Zero console warnings
- [ ] Smooth 60fps animation
- [ ] JobLogsPanel.tsx deleted

---

## 🎨 DESIGN SPECIFICATIONS

### **Toggle Button:**
```typescript
Position: Top right header, next to "Create Flow"
Size: p-2 (8px padding)
Icon: HiClipboardList (5x5)
Background: 
  - Closed: bg-gray-700 hover:bg-gray-600
  - Open: bg-blue-500/20 text-blue-400
Transition: transition-colors
```

### **Drawer:**
```typescript
Position: fixed, right: 0, top: 0, bottom: 0
Width: 400px default, 300-600px range
Background: bg-gray-900
Border: border-l border-gray-700
Shadow: shadow-2xl
Z-index: z-50
Animation: width 300ms ease-in-out
```

### **Resize Handle:**
```typescript
Position: fixed, left edge of drawer
Width: w-1 (4px)
Background: bg-gray-700 hover:bg-blue-500
Cursor: cursor-ew-resize
Z-index: z-40
Active: bg-blue-500 (when dragging)
```

---

## 🧪 TESTING CHECKLIST

```bash
# 1. Development mode
cd /home/oma_admin/sendense/source/current/sendense-gui
npm run dev

# 2. Navigate to Protection Flows
http://localhost:3000/protection-flows

# 3. Test collapsed state
✓ Job Logs button visible (top right)
✓ Main content uses full width
✓ No drawer visible

# 4. Test open/close
✓ Click button → Drawer slides in from right
✓ Main content gets narrower
✓ Click X in drawer → Drawer slides out
✓ Click button again → Drawer slides back in
✓ Press Ctrl+L → Drawer toggles

# 5. Test resizing
✓ Hover left edge of drawer → ew-resize cursor
✓ Drag left → Drawer grows (max 600px)
✓ Drag right → Drawer shrinks (min 300px)
✓ Release → Width saved

# 6. Test persistence
✓ Open drawer, reload page → Drawer opens
✓ Resize drawer, reload page → Width remembered
✓ Close drawer, reload page → Drawer closed

# 7. Test functionality
✓ Logs display with colors
✓ Auto-scroll works
✓ Filter dropdown works
✓ Clear button works
✓ Live badge pulses

# 8. Production build
npm run build
✓ Build succeeds
```

---

## 📝 COMMIT MESSAGE

```
refactor: convert job logs to collapsible drawer

- Replaced always-visible panel with slide-out drawer
- Toggle button in top right header (HiClipboardList icon)
- Resizable drawer (300-600px, drag left edge)
- Smooth slide animation (300ms ease-in-out)
- Keyboard shortcut: Ctrl+L to toggle
- State persistence (localStorage)
- Width persistence (localStorage)
- No overlay (push content mode)
- Production build successful

Files:
- NEW: components/features/protection-flows/JobLogsDrawer.tsx
- MODIFIED: app/protection-flows/page.tsx (toggle button + drawer)
- MODIFIED: components/features/protection-flows/index.tsx (export)
- DELETED: components/features/protection-flows/JobLogsPanel.tsx

Breaking changes: None
```

---

## ⚠️ CRITICAL RULES

1. **NO OVERLAY:** Drawer pushes content, doesn't overlay
2. **SMOOTH ANIMATION:** 300ms ease-in-out transition
3. **STATE PERSISTENCE:** localStorage for open/closed + width
4. **KEYBOARD SHORTCUT:** Ctrl+L must work
5. **DELETE OLD FILE:** Remove JobLogsPanel.tsx completely
6. **BUILD MUST SUCCEED:** Test `npm run build` before finishing

---

**Status:** 🟡 PENDING  
**Priority:** HIGH  
**Estimated:** 1-2 hours  
**Complexity:** Medium-High

---

*This converts the always-visible Job Logs panel into a professional collapsible drawer that slides out when needed, saving valuable screen space.*

