# GUI Add Job Logs Panel - Job Sheet

**Date:** October 9, 2025  
**Assignee:** Grok Code Fast  
**Priority:** HIGH  
**Type:** GUI Enhancement  
**Status:** 🟡 PENDING

---

## 🎯 OBJECTIVE

Add back the **Job Logs panel** on the right side of the Protection Flows page that was lost during the panel refactoring.

---

## 📸 DESIRED LAYOUT

```
┌─────────────────────────────────────────┬─────────────┐
│ MAIN CONTENT (70%)                      │ JOB LOGS    │
│ ┌─────────────────────────────────────┐ │ (30%)       │
│ │ TOP: Flows Table                    │ │             │
│ │ Default: 50vh, Min: 30vh            │ │ Live        │
│ └─────────────────────────────────────┘ │ Auto-scroll │
│ ═══════════════════════════════════════ │ ─────────── │
│ ┌─────────────────────────────────────┐ │ 11:00:01    │
│ │ BOTTOM: Flow Details                │ │ [INFO]      │
│ │ Default: 40vh, Min: 20vh            │ │ Starting... │
│ └─────────────────────────────────────┘ │             │
│   ↑ Resizable vertical divider          │ 11:00:02    │
│                                         │ [INFO]      │
│                                         │ Connecting  │
│                                         │             │
│                                         │ 11:00:05    │
│                                         │ [WARNING]   │
│                                         │ Network...  │
└─────────────────────────────────────────┴─────────────┘
  ↑ Resizable horizontal divider
```

---

## 🔧 IMPLEMENTATION

### **Nested PanelGroup Structure:**

```typescript
<PanelGroup direction="horizontal">
  {/* Left: Main Content (70%) */}
  <Panel defaultSize={70} minSize={50}>
    <PanelGroup direction="vertical">
      {/* Top: Flows Table */}
      <Panel defaultSize={50} minSize={30}>
        {/* Flows table */}
      </Panel>
      
      <PanelResizeHandle />
      
      {/* Bottom: Flow Details */}
      <Panel defaultSize={40} minSize={20}>
        {/* Flow details */}
      </Panel>
    </PanelGroup>
  </Panel>

  <PanelResizeHandle />
  
  {/* Right: Job Logs (30%) */}
  <Panel defaultSize={30} minSize={20}>
    <JobLogsPanel />
  </Panel>
</PanelGroup>
```

---

## 📋 JOB LOGS PANEL REQUIREMENTS

### **Header:**
- Title: "Job Logs"
- Badge: "Live" (green pulse animation)
- Buttons: "Auto-scroll" toggle, "Clear" button, Filter dropdown

### **Log Display:**
- Auto-scroll to bottom when new logs arrive
- Color-coded log levels:
  - `[INFO]` → Blue text
  - `[WARNING]` → Yellow/orange text
  - `[ERROR]` → Red text
  - `[SUCCESS]` → Green text
- Timestamp format: `HH:mm:ss`
- Monospace font (font-mono)
- Dark background (bg-gray-900)
- Scroll overflow (overflow-auto)

### **Features:**
- Filter by log level (All, Info, Warning, Error)
- Auto-scroll toggle (on by default)
- Clear logs button
- Max 1000 lines (auto-trim oldest)
- Click log line → Expand for full details

### **Mock Data (For Now):**
```typescript
const mockLogs = [
  { time: '11:00:01', level: 'INFO', message: 'Starting backup job for Backup Engine pgtest1' },
  { time: '11:00:02', level: 'INFO', message: 'Connecting to vCenter serv VMware API er' },
  { time: '11:00:03', level: 'INFO', message: 'Snapshot created successfu VMware API lly' },
  { time: '11:00:04', level: 'INFO', message: 'Transferring data: 25% c NBD Transfer omplete' },
  { time: '11:00:05', level: 'WARNING', message: 'Network latency detec NBD Transfer ted, adjusting buffer size' },
  { time: '11:00:06', level: 'INFO', message: 'Transferring data: 50% c NBD Transfer omplete' },
  { time: '11:00:07', level: 'INFO', message: 'Transferring data: 75% c NBD Transfer omplete' },
  { time: '11:00:08', level: 'INFO', message: 'Transferring data: 100% NBD Transfer complete' },
  { time: '11:00:09', level: 'INFO', message: 'Verifying backup in Validation Engine tegrity' },
  { time: '11:00:10', level: 'INFO', message: 'Backup completed succes Backup Engine sfully' },
]
```

---

## 🎨 DESIGN SPECIFICATIONS

### **Panel Sizes:**
```typescript
Horizontal Split:
├─ Main Content: 70% default, 50% min
└─ Job Logs: 30% default, 20% min

Vertical Split (within Main):
├─ Flows Table: 50% default, 30% min
└─ Flow Details: 40% default, 20% min
```

### **Job Logs Header:**
```typescript
<div className="flex items-center justify-between px-4 py-3 border-b border-gray-700 bg-gray-800/50">
  <div className="flex items-center gap-2">
    <h3 className="text-sm font-semibold text-white">Job Logs</h3>
    <span className="px-2 py-0.5 text-xs bg-green-500/20 text-green-400 rounded flex items-center gap-1">
      <span className="w-2 h-2 bg-green-500 rounded-full animate-pulse"></span>
      Live
    </span>
  </div>
  
  <div className="flex items-center gap-2">
    <Button size="xs" color="gray">
      <HiFilter className="h-3 w-3" />
      All
    </Button>
    <Button size="xs" color="gray">
      Auto-scroll
    </Button>
    <Button size="xs" color="gray">
      <HiX className="h-3 w-3" />
    </Button>
  </div>
</div>
```

### **Log Entry:**
```typescript
<div className="px-4 py-1 font-mono text-xs hover:bg-gray-800/30 cursor-pointer">
  <span className="text-gray-500">11:00:01</span>
  {' '}
  <span className="text-blue-400 font-semibold">[INFO]</span>
  {' '}
  <span className="text-gray-300">Starting backup job for</span>
  {' '}
  <span className="text-blue-300">Backup Engine</span>
  {' '}
  <span className="text-gray-300">pgtest1</span>
</div>
```

### **Color Coding:**
```typescript
const logLevelColors = {
  INFO: 'text-blue-400',
  WARNING: 'text-yellow-400',
  ERROR: 'text-red-400',
  SUCCESS: 'text-green-400',
}
```

---

## 📦 FILES TO CREATE/MODIFY

### **New File:**
```
components/features/protection-flows/JobLogsPanel.tsx (create)
```

### **Modified File:**
```
app/protection-flows/page.tsx (update PanelGroup structure)
```

---

## ✅ ACCEPTANCE CRITERIA

**Layout:**
- [x] Horizontal split: Main (70%) + Job Logs (30%)
- [x] Vertical split within Main: Table + Details
- [x] Both dividers resizable
- [x] Minimum sizes respected

**Job Logs Panel:**
- [x] Header with "Job Logs" title + "Live" badge
- [x] Auto-scroll toggle button
- [x] Clear logs button
- [x] Filter dropdown (All, Info, Warning, Error)
- [x] Color-coded log levels
- [x] Monospace font
- [x] Scrollable log area
- [x] Mock data displays correctly

**Design:**
- [x] Matches existing dark theme
- [x] Professional appearance
- [x] Smooth panel resizing
- [x] Responsive layout

**Code Quality:**
- [x] Production build succeeds
- [x] Zero TypeScript errors
- [x] Zero console warnings
- [x] Component <200 lines

---

## 🧪 TESTING CHECKLIST

```bash
# 1. Development mode
cd /home/oma_admin/sendense/source/current/sendense-gui
npm run dev

# 2. Test in browser
# Navigate to: http://localhost:3000/protection-flows

# 3. Verify layout
✓ Job Logs panel visible on right (30% width)
✓ Main content on left (70% width)
✓ Vertical divider between flows table and details
✓ Horizontal divider between main and logs

# 4. Test resizing
✓ Drag horizontal divider → Main/Logs resize
✓ Drag vertical divider → Table/Details resize
✓ Can't drag beyond min sizes

# 5. Test Job Logs
✓ "Live" badge shows and pulses (green)
✓ Mock logs display with timestamps
✓ Colors correct: INFO blue, WARNING yellow, ERROR red
✓ Auto-scroll to bottom works
✓ Clear button clears logs
✓ Filter dropdown changes visible logs

# 6. Production build
npm run build
✓ Build succeeds without errors
```

---

## 📝 COMMIT MESSAGE

```
feat: add job logs panel to protection flows

- Added JobLogsPanel component (right side, 30% width)
- Implemented nested PanelGroup (horizontal + vertical)
- Color-coded log levels (INFO, WARNING, ERROR, SUCCESS)
- Auto-scroll, clear, and filter functionality
- Mock data for development testing
- Resizable panels with minimum size constraints
- Professional dark theme styling
- Production build successful

Breaking changes: None
```

---

## 🎯 SUCCESS METRICS

**Layout Restored:**
- ✅ Job Logs panel back on right side
- ✅ Nested resizable panels working
- ✅ Original workflow preserved

**User Experience:**
- ✅ Live log streaming (mock for now)
- ✅ Auto-scroll keeps latest visible
- ✅ Easy to filter/clear logs
- ✅ Professional appearance

**Technical Quality:**
- ✅ Clean component architecture
- ✅ Production build passes
- ✅ No TypeScript errors
- ✅ Smooth panel resizing

---

**Status:** 🟡 PENDING  
**Assigned To:** Grok Code Fast  
**Expected Duration:** 1-2 hours  
**Complexity:** Medium  
**Priority:** HIGH

---

*This restores the Job Logs panel that was accidentally removed during the modal-to-panel refactoring.*

