# Machine Backup Details Modal - Complete Implementation with Charts

**Date:** October 10, 2025  
**Priority:** HIGH  
**Complexity:** HIGH  
**Estimated Time:** 6-8 hours

---

## 🎯 Objective

Complete the Machine Backup Details Modal with professional charts and telemetry history, transforming it from a simple data table into a comprehensive backup monitoring dashboard comparable to Veeam.

### What We're Building:

**3-Tab Modal System:**
1. **Summary Tab** - VM info, KPIs, backup history table (ALREADY IMPLEMENTED)
2. **Performance Tab** - NEW - Transfer speed chart, size distribution, job timeline
3. **Analytics Tab** - NEW - Success rate donut, backup trends, detailed job analysis

**Backend Requirements:**
- Telemetry history snapshots system
- Historical data storage and retrieval
- Chart data API endpoints

---

## 📊 Current State vs. Target State

### Current State (What Exists)
✅ **Backend:**
- `backup_jobs` table with telemetry fields (current state only)
- `backup_disks` table with per-disk data
- API returns complete backup data with telemetry
- `/api/v1/backups?vm_name={name}&repository_id={repo}` working

✅ **Frontend:**
- `MachineDetailsModal` component exists
- Summary view with VM specs, KPIs, backup table
- Opens when clicking machine in flows table
- Displays backup type, size, duration, status

### Target State (What We Need)
🎯 **Backend:**
- NEW: `job_telemetry_snapshots` table for historical performance data
- NEW: Rolled-up performance metrics in `backup_jobs` (JSON field)
- NEW: API endpoint `/api/v1/telemetry/history/{job_id}` 
- NEW: Completion handler to roll up snapshots into summary

🎯 **Frontend:**
- NEW: Tabs component in modal (Summary, Performance, Analytics)
- NEW: Performance charts (speed histogram, timeline)
- NEW: Analytics charts (success donut, trends, averages)
- NEW: Job selector dropdown to switch between backups
- ENHANCE: Better modal sizing and layout

---

## 🗄️ Phase 1: Database Schema - Telemetry History System

### Problem Statement
Currently, telemetry data is **single-row** that gets overwritten with each update. We only see the final state (100% complete, last speed). We need **historical snapshots** to build performance charts showing speed fluctuations over time.

### Solution: Hybrid Storage System

**During Backup (Real-Time):**
- Store snapshot every 30 seconds in `job_telemetry_snapshots` table
- Enables live performance charts while job running

**On Backup Completion:**
- Roll up snapshots into summary JSON in `backup_jobs.performance_metrics`
- Delete individual snapshots to keep database lean
- Preserves chart data forever without bloat

---

### New Table: `job_telemetry_snapshots`

```sql
CREATE TABLE job_telemetry_snapshots (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    job_id VARCHAR(255) NOT NULL COMMENT 'References backup_jobs.id',
    job_type VARCHAR(50) NOT NULL COMMENT 'backup, restore, replication',
    timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    -- Snapshot metrics (point-in-time values)
    bytes_transferred BIGINT DEFAULT 0 COMMENT 'Bytes transferred at this moment',
    total_bytes BIGINT DEFAULT 0 COMMENT 'Total bytes to transfer',
    transfer_speed_bps BIGINT DEFAULT 0 COMMENT 'Transfer speed at this moment (bytes/sec)',
    progress_percent DECIMAL(5,2) DEFAULT 0.00 COMMENT 'Progress percentage 0-100',
    current_phase VARCHAR(100) COMMENT 'snapshot, transferring, finalizing, etc.',
    
    -- Optional: Per-disk breakdown
    disk_index INT COMMENT 'For multi-disk VMs, which disk (0, 1, 2...)',
    worker_count INT COMMENT 'Number of parallel workers at this moment',
    
    -- Indexes for fast queries
    INDEX idx_job_id (job_id),
    INDEX idx_timestamp (timestamp),
    INDEX idx_job_timestamp (job_id, timestamp),
    
    -- Foreign key (optional - can enable CASCADE DELETE)
    FOREIGN KEY (job_id) REFERENCES backup_jobs(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Historical telemetry snapshots for performance charts';
```

**Storage Estimates:**
- 5-minute backup: 10 snapshots × 150 bytes = 1.5 KB (deleted after completion)
- 1-hour backup: 120 snapshots × 150 bytes = 18 KB (deleted after completion)
- Final rolled-up JSON: 2-5 KB per job (kept forever)

**1000 completed backups = 2-5 MB total** (trivial storage cost)

---

### Alter Table: `backup_jobs` - Add Performance Metrics

```sql
ALTER TABLE backup_jobs 
ADD COLUMN performance_metrics JSON COMMENT 'Rolled-up telemetry data for charts' AFTER last_telemetry_at;
```

**Example JSON Structure:**
```json
{
  "duration_seconds": 62,
  "snapshots_recorded": 3,
  "peak_speed_bps": 336392246,
  "avg_speed_bps": 280000000,
  "min_speed_bps": 200000000,
  "peak_speed_mbps": 320.7,
  "avg_speed_mbps": 267.0,
  "chart_data": [
    {"t": 0, "s": 300000000, "p": 0, "phase": "snapshot"},
    {"t": 30, "s": 336392246, "p": 50, "phase": "transferring"},
    {"t": 62, "s": 320000000, "p": 100, "phase": "completed"}
  ]
}
```

**Field Definitions:**
- `t` - Seconds since start
- `s` - Speed in bytes per second
- `p` - Progress percent (0-100)
- `phase` - Current operation phase

---

### Migration File

**File:** `source/current/sha/database/migrations/20251010_telemetry_history.sql`

```sql
-- Migration: Telemetry History System for Performance Charts
-- Date: 2025-10-10

-- Create telemetry snapshots table
CREATE TABLE IF NOT EXISTS job_telemetry_snapshots (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    job_id VARCHAR(255) NOT NULL,
    job_type VARCHAR(50) NOT NULL,
    timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    bytes_transferred BIGINT DEFAULT 0,
    total_bytes BIGINT DEFAULT 0,
    transfer_speed_bps BIGINT DEFAULT 0,
    progress_percent DECIMAL(5,2) DEFAULT 0.00,
    current_phase VARCHAR(100),
    
    disk_index INT,
    worker_count INT,
    
    INDEX idx_job_id (job_id),
    INDEX idx_timestamp (timestamp),
    INDEX idx_job_timestamp (job_id, timestamp),
    
    FOREIGN KEY (job_id) REFERENCES backup_jobs(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Add performance_metrics JSON column to backup_jobs
ALTER TABLE backup_jobs 
ADD COLUMN IF NOT EXISTS performance_metrics JSON COMMENT 'Rolled-up telemetry for charts';

-- Create index for efficient JSON queries
CREATE INDEX idx_performance_metrics ON backup_jobs((CAST(performance_metrics->>'$.avg_speed_bps' AS UNSIGNED)));
```

---

## 🔧 Phase 2: Backend Implementation - SHA Telemetry Service

### Task 2.1: Update Telemetry Service to Store Snapshots

**File:** `source/current/sha/services/telemetry_service.go`

**Add snapshot storage to `ProcessTelemetryUpdate`:**

```go
func (ts *TelemetryService) ProcessTelemetryUpdate(
	ctx context.Context,
	jobType string,
	jobID string,
	update *TelemetryUpdate,
) error {
	now := time.Now()
	
	// ... existing backup_jobs update logic ...
	
	// 🆕 NEW: Store snapshot for historical chart data (only if job is running)
	if update.Status == "running" || update.Status == "" {
		snapshot := map[string]interface{}{
			"job_id":              jobID,
			"job_type":            jobType,
			"timestamp":           now,
			"bytes_transferred":   update.BytesTransferred,
			"total_bytes":         update.TotalBytes,
			"transfer_speed_bps":  update.TransferSpeedBps,
			"progress_percent":    update.ProgressPercent,
			"current_phase":       update.CurrentPhase,
		}
		
		// Insert snapshot (fire-and-forget, don't fail job on snapshot error)
		go func() {
			if err := ts.db.GetGormDB().Table("job_telemetry_snapshots").Create(snapshot).Error; err != nil {
				log.WithError(err).WithField("job_id", jobID).Warn("Failed to store telemetry snapshot")
			}
		}()
	}
	
	return nil
}
```

---

### Task 2.2: Create Telemetry Completion Handler

**New File:** `source/current/sha/services/telemetry_completion_handler.go`

```go
package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"
	log "github.com/sirupsen/logrus"
)

// TelemetryCompletionHandler handles rolling up snapshots when job completes
type TelemetryCompletionHandler struct {
	db *gorm.DB
}

// NewTelemetryCompletionHandler creates a new completion handler
func NewTelemetryCompletionHandler(db *gorm.DB) *TelemetryCompletionHandler {
	return &TelemetryCompletionHandler{
		db: db,
	}
}

// SnapshotData represents a telemetry snapshot
type SnapshotData struct {
	ID                int64     `gorm:"column:id"`
	JobID             string    `gorm:"column:job_id"`
	Timestamp         time.Time `gorm:"column:timestamp"`
	BytesTransferred  int64     `gorm:"column:bytes_transferred"`
	TotalBytes        int64     `gorm:"column:total_bytes"`
	TransferSpeedBps  int64     `gorm:"column:transfer_speed_bps"`
	ProgressPercent   float64   `gorm:"column:progress_percent"`
	CurrentPhase      string    `gorm:"column:current_phase"`
}

// PerformanceMetrics represents the rolled-up performance data
type PerformanceMetrics struct {
	DurationSeconds   int                `json:"duration_seconds"`
	SnapshotsRecorded int                `json:"snapshots_recorded"`
	PeakSpeedBps      int64              `json:"peak_speed_bps"`
	AvgSpeedBps       int64              `json:"avg_speed_bps"`
	MinSpeedBps       int64              `json:"min_speed_bps"`
	PeakSpeedMbps     float64            `json:"peak_speed_mbps"`
	AvgSpeedMbps      float64            `json:"avg_speed_mbps"`
	ChartData         []ChartDataPoint   `json:"chart_data"`
}

// ChartDataPoint represents a point on the performance chart
type ChartDataPoint struct {
	T     int     `json:"t"`     // Seconds since start
	S     int64   `json:"s"`     // Speed in bytes/sec
	P     float64 `json:"p"`     // Progress percent
	Phase string  `json:"phase"` // Current phase
}

// RollUpTelemetry rolls up snapshots into performance metrics JSON
func (h *TelemetryCompletionHandler) RollUpTelemetry(ctx context.Context, jobID string) error {
	log.WithField("job_id", jobID).Info("🎯 Rolling up telemetry snapshots to performance metrics")
	
	// 1. Get all snapshots for this job
	var snapshots []SnapshotData
	if err := h.db.Table("job_telemetry_snapshots").
		Where("job_id = ?", jobID).
		Order("timestamp ASC").
		Find(&snapshots).Error; err != nil {
		return fmt.Errorf("failed to fetch snapshots: %w", err)
	}
	
	if len(snapshots) == 0 {
		log.WithField("job_id", jobID).Debug("No snapshots found - job may have completed too quickly")
		return nil // Not an error - some jobs complete in <30 seconds
	}
	
	log.WithFields(log.Fields{
		"job_id":    jobID,
		"snapshots": len(snapshots),
	}).Info("📊 Processing telemetry snapshots")
	
	// 2. Calculate metrics
	startTime := snapshots[0].Timestamp
	endTime := snapshots[len(snapshots)-1].Timestamp
	durationSeconds := int(endTime.Sub(startTime).Seconds())
	
	var peakSpeed, minSpeed, totalSpeed int64
	minSpeed = snapshots[0].TransferSpeedBps
	chartData := make([]ChartDataPoint, 0, len(snapshots))
	
	for _, snap := range snapshots {
		if snap.TransferSpeedBps > peakSpeed {
			peakSpeed = snap.TransferSpeedBps
		}
		if snap.TransferSpeedBps < minSpeed && snap.TransferSpeedBps > 0 {
			minSpeed = snap.TransferSpeedBps
		}
		totalSpeed += snap.TransferSpeedBps
		
		// Build chart data point
		secondsSinceStart := int(snap.Timestamp.Sub(startTime).Seconds())
		chartData = append(chartData, ChartDataPoint{
			T:     secondsSinceStart,
			S:     snap.TransferSpeedBps,
			P:     snap.ProgressPercent,
			Phase: snap.CurrentPhase,
		})
	}
	
	avgSpeed := totalSpeed / int64(len(snapshots))
	
	// Convert to Mbps for display
	peakSpeedMbps := float64(peakSpeed) / (1024 * 1024)
	avgSpeedMbps := float64(avgSpeed) / (1024 * 1024)
	
	// 3. Build performance metrics JSON
	metrics := PerformanceMetrics{
		DurationSeconds:   durationSeconds,
		SnapshotsRecorded: len(snapshots),
		PeakSpeedBps:      peakSpeed,
		AvgSpeedBps:       avgSpeed,
		MinSpeedBps:       minSpeed,
		PeakSpeedMbps:     peakSpeedMbps,
		AvgSpeedMbps:      avgSpeedMbps,
		ChartData:         chartData,
	}
	
	metricsJSON, err := json.Marshal(metrics)
	if err != nil {
		return fmt.Errorf("failed to marshal metrics: %w", err)
	}
	
	// 4. Update backup_jobs with performance_metrics JSON
	if err := h.db.Table("backup_jobs").
		Where("id = ?", jobID).
		Update("performance_metrics", string(metricsJSON)).Error; err != nil {
		return fmt.Errorf("failed to update performance_metrics: %w", err)
	}
	
	log.WithFields(log.Fields{
		"job_id":          jobID,
		"peak_speed_mbps": fmt.Sprintf("%.1f", peakSpeedMbps),
		"avg_speed_mbps":  fmt.Sprintf("%.1f", avgSpeedMbps),
		"duration":        durationSeconds,
	}).Info("✅ Performance metrics rolled up successfully")
	
	// 5. Delete snapshots to keep database lean
	if err := h.db.Table("job_telemetry_snapshots").
		Where("job_id = ?", jobID).
		Delete(nil).Error; err != nil {
		log.WithError(err).WithField("job_id", jobID).Warn("Failed to delete snapshots (non-critical)")
	} else {
		log.WithField("job_id", jobID).Info("🧹 Cleaned up telemetry snapshots")
	}
	
	return nil
}
```

---

### Task 2.3: Integrate Completion Handler into Backup Workflow

**File:** `source/current/sha/workflows/backup.go`

**In `CompleteBackup` method, add after status update:**

```go
// After updating backup_jobs status to "completed"...

// 🆕 NEW: Roll up telemetry snapshots to performance metrics
if be.telemetryHandler != nil {
	go func(backupID string) {
		ctx := context.Background()
		if err := be.telemetryHandler.RollUpTelemetry(ctx, backupID); err != nil {
			log.WithError(err).WithField("backup_id", backupID).Warn("Failed to roll up telemetry (non-critical)")
		}
	}(backupID)
}
```

**Add field to BackupEngine struct:**

```go
type BackupEngine struct {
	// ... existing fields ...
	telemetryHandler *services.TelemetryCompletionHandler // 🆕 NEW
}
```

**Initialize in constructor or main.go:**

```go
telemetryHandler := services.NewTelemetryCompletionHandler(db.GetGormDB())
backupEngine := &BackupEngine{
	// ... existing fields ...
	telemetryHandler: telemetryHandler,
}
```

---

### Task 2.4: Create Telemetry History API Endpoint

**File:** `source/current/sha/api/handlers/telemetry_handlers.go`

**Add new handler method:**

```go
// GetTelemetryHistory handles GET /api/v1/telemetry/history/{job_id}
// Returns historical performance data for charts
func (th *TelemetryHandler) GetTelemetryHistory(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	jobID := vars["job_id"]
	
	if jobID == "" {
		http.Error(w, "Missing job_id parameter", http.StatusBadRequest)
		return
	}
	
	log.WithField("job_id", jobID).Info("📊 Fetching telemetry history for charts")
	
	// Check if job is completed (has performance_metrics)
	var job struct {
		ID                  string          `gorm:"column:id"`
		Status              string          `gorm:"column:status"`
		PerformanceMetrics  json.RawMessage `gorm:"column:performance_metrics"`
		BytesTransferred    int64           `gorm:"column:bytes_transferred"`
		TotalBytes          int64           `gorm:"column:total_bytes"`
		TransferSpeedBps    int64           `gorm:"column:transfer_speed_bps"`
		ProgressPercent     float64         `gorm:"column:progress_percent"`
		CurrentPhase        string          `gorm:"column:current_phase"`
		LastTelemetryAt     *time.Time      `gorm:"column:last_telemetry_at"`
	}
	
	if err := th.db.Table("backup_jobs").Where("id = ?", jobID).First(&job).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			http.Error(w, "Job not found", http.StatusNotFound)
			return
		}
		log.WithError(err).Error("Failed to fetch job")
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	
	// If job is completed, return rolled-up metrics from JSON
	if job.Status == "completed" && job.PerformanceMetrics != nil {
		var metrics map[string]interface{}
		if err := json.Unmarshal(job.PerformanceMetrics, &metrics); err != nil {
			log.WithError(err).Error("Failed to parse performance_metrics JSON")
			http.Error(w, "Invalid performance data", http.StatusInternalServerError)
			return
		}
		
		response := map[string]interface{}{
			"job_id":              jobID,
			"status":              job.Status,
			"data_source":         "rolled_up",
			"performance_metrics": metrics,
		}
		
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
		return
	}
	
	// If job is running, return live snapshots
	if job.Status == "running" {
		type Snapshot struct {
			Timestamp        time.Time `gorm:"column:timestamp"`
			BytesTransferred int64     `gorm:"column:bytes_transferred"`
			TransferSpeedBps int64     `gorm:"column:transfer_speed_bps"`
			ProgressPercent  float64   `gorm:"column:progress_percent"`
			CurrentPhase     string    `gorm:"column:current_phase"`
		}
		
		var snapshots []Snapshot
		if err := th.db.Table("job_telemetry_snapshots").
			Where("job_id = ?", jobID).
			Order("timestamp ASC").
			Find(&snapshots).Error; err != nil {
			log.WithError(err).Error("Failed to fetch snapshots")
			http.Error(w, "Failed to fetch snapshots", http.StatusInternalServerError)
			return
		}
		
		// Build chart data from live snapshots
		chartData := make([]map[string]interface{}, 0, len(snapshots))
		if len(snapshots) > 0 {
			startTime := snapshots[0].Timestamp
			for _, snap := range snapshots {
				secondsSinceStart := int(snap.Timestamp.Sub(startTime).Seconds())
				chartData = append(chartData, map[string]interface{}{
					"t":     secondsSinceStart,
					"s":     snap.TransferSpeedBps,
					"p":     snap.ProgressPercent,
					"phase": snap.CurrentPhase,
				})
			}
		}
		
		response := map[string]interface{}{
			"job_id":      jobID,
			"status":      job.Status,
			"data_source": "live_snapshots",
			"current_state": map[string]interface{}{
				"bytes_transferred":  job.BytesTransferred,
				"total_bytes":        job.TotalBytes,
				"transfer_speed_bps": job.TransferSpeedBps,
				"progress_percent":   job.ProgressPercent,
				"current_phase":      job.CurrentPhase,
			},
			"chart_data": chartData,
		}
		
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
		return
	}
	
	// Job exists but has no telemetry data
	response := map[string]interface{}{
		"job_id":      jobID,
		"status":      job.Status,
		"data_source": "none",
		"message":     "No telemetry data available for this job",
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}
```

**Register route in `RegisterRoutes`:**

```go
// GET /api/v1/telemetry/history/{job_id} - Get historical telemetry for charts
r.HandleFunc("/telemetry/history/{job_id}", th.GetTelemetryHistory).Methods("GET")
```

---

## 🎨 Phase 3: Frontend Implementation - Modal with Charts

### Task 3.1: Install Chart Library

```bash
npm install recharts
npm install --save-dev @types/recharts
```

**Why Recharts?**
- React-native with hooks support
- Beautiful default styling
- Responsive and performant
- TypeScript support
- Wide component library (Line, Bar, Pie, Area charts)

---

### Task 3.2: Create Telemetry History Hook

**New File:** `src/features/protection-flows/hooks/useTelemetryHistory.ts`

```typescript
import { useQuery } from '@tanstack/react-query';

const API_BASE = '';

interface ChartDataPoint {
  t: number;      // Seconds since start
  s: number;      // Speed in bytes/sec
  p: number;      // Progress percent
  phase: string;  // Current phase
}

interface PerformanceMetrics {
  duration_seconds: number;
  snapshots_recorded: number;
  peak_speed_bps: number;
  avg_speed_bps: number;
  min_speed_bps: number;
  peak_speed_mbps: number;
  avg_speed_mbps: number;
  chart_data: ChartDataPoint[];
}

interface TelemetryHistoryResponse {
  job_id: string;
  status: string;
  data_source: 'rolled_up' | 'live_snapshots' | 'none';
  performance_metrics?: PerformanceMetrics;
  current_state?: {
    bytes_transferred: number;
    total_bytes: number;
    transfer_speed_bps: number;
    progress_percent: number;
    current_phase: string;
  };
  chart_data?: ChartDataPoint[];
  message?: string;
}

export function useTelemetryHistory(jobId: string | null) {
  return useQuery({
    queryKey: ['telemetry-history', jobId],
    queryFn: async (): Promise<TelemetryHistoryResponse | null> => {
      if (!jobId) return null;
      
      const response = await fetch(`${API_BASE}/api/v1/telemetry/history/${jobId}`);
      
      if (!response.ok) {
        throw new Error(`Failed to fetch telemetry history: ${response.statusText}`);
      }
      
      const data = await response.json();
      return data;
    },
    enabled: !!jobId,
    staleTime: 30000, // 30 seconds for completed jobs
    refetchInterval: (data) => {
      // Poll every 3 seconds if job is running, otherwise don't poll
      return data?.status === 'running' ? 3000 : false;
    },
  });
}
```

---

### Task 3.3: Create Chart Components

**New File:** `src/features/protection-flows/components/charts/SpeedHistogram.tsx`

```typescript
"use client";

import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer } from 'recharts';

interface ChartDataPoint {
  t: number;
  s: number;
  p: number;
  phase: string;
}

interface SpeedHistogramProps {
  data: ChartDataPoint[];
  className?: string;
}

export function SpeedHistogram({ data, className }: SpeedHistogramProps) {
  // Transform data for chart display
  const chartData = data.map(point => ({
    time: `${Math.floor(point.t / 60)}:${(point.t % 60).toString().padStart(2, '0')}`,
    speedMbps: (point.s / (1024 * 1024)).toFixed(1),
    progress: point.p.toFixed(1),
    phase: point.phase,
  }));
  
  return (
    <div className={className}>
      <ResponsiveContainer width="100%" height={300}>
        <LineChart data={chartData}>
          <CartesianGrid strokeDasharray="3 3" className="stroke-muted" />
          <XAxis 
            dataKey="time" 
            label={{ value: 'Time (mm:ss)', position: 'insideBottom', offset: -5 }}
            className="text-sm"
          />
          <YAxis 
            label={{ value: 'Speed (MB/s)', angle: -90, position: 'insideLeft' }}
            className="text-sm"
          />
          <Tooltip 
            contentStyle={{ 
              backgroundColor: 'hsl(var(--background))',
              border: '1px solid hsl(var(--border))',
              borderRadius: '0.5rem'
            }}
            labelStyle={{ color: 'hsl(var(--foreground))' }}
          />
          <Legend />
          <Line 
            type="monotone" 
            dataKey="speedMbps" 
            name="Transfer Speed"
            stroke="hsl(var(--primary))" 
            strokeWidth={2}
            dot={{ r: 3 }}
            activeDot={{ r: 5 }}
          />
        </LineChart>
      </ResponsiveContainer>
    </div>
  );
}
```

---

**New File:** `src/features/protection-flows/components/charts/SuccessDonut.tsx`

```typescript
"use client";

import { PieChart, Pie, Cell, ResponsiveContainer, Legend, Tooltip } from 'recharts';

interface SuccessDonutProps {
  successCount: number;
  failedCount: number;
  className?: string;
}

const COLORS = {
  success: 'hsl(142, 76%, 36%)',  // Green
  failed: 'hsl(0, 84%, 60%)',     // Red
};

export function SuccessDonut({ successCount, failedCount, className }: SuccessDonutProps) {
  const data = [
    { name: 'Successful', value: successCount, color: COLORS.success },
    { name: 'Failed', value: failedCount, color: COLORS.failed },
  ];
  
  const total = successCount + failedCount;
  const successRate = total > 0 ? ((successCount / total) * 100).toFixed(1) : '0';
  
  return (
    <div className={className}>
      <ResponsiveContainer width="100%" height={250}>
        <PieChart>
          <Pie
            data={data}
            cx="50%"
            cy="50%"
            innerRadius={60}
            outerRadius={80}
            paddingAngle={5}
            dataKey="value"
          >
            {data.map((entry, index) => (
              <Cell key={`cell-${index}`} fill={entry.color} />
            ))}
          </Pie>
          <Tooltip 
            contentStyle={{ 
              backgroundColor: 'hsl(var(--background))',
              border: '1px solid hsl(var(--border))',
              borderRadius: '0.5rem'
            }}
          />
          <Legend />
          <text
            x="50%"
            y="50%"
            textAnchor="middle"
            dominantBaseline="middle"
            className="text-2xl font-bold fill-foreground"
          >
            {successRate}%
          </text>
        </PieChart>
      </ResponsiveContainer>
    </div>
  );
}
```

---

### Task 3.4: Update Modal with Tabs

**File:** `components/features/protection-flows/MachineDetailsModal.tsx`

**Add imports:**

```typescript
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { SpeedHistogram } from "@/src/features/protection-flows/components/charts/SpeedHistogram";
import { SuccessDonut } from "@/src/features/protection-flows/components/charts/SuccessDonut";
import { useTelemetryHistory } from "@/src/features/protection-flows/hooks/useTelemetryHistory";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
```

**Add state for selected backup:**

```typescript
const [selectedBackupId, setSelectedBackupId] = useState<string | null>(null);

// Fetch telemetry history for selected backup
const { data: telemetryData } = useTelemetryHistory(selectedBackupId);
```

**Update modal content to use tabs:**

```typescript
<DialogContent className="max-w-5xl max-h-[90vh] overflow-hidden">
  <DialogHeader>
    <DialogTitle className="flex items-center gap-2 text-xl">
      <Server className="h-5 w-5" />
      {machine.vm_name}
    </DialogTitle>
  </DialogHeader>

  <Tabs defaultValue="summary" className="w-full">
    <TabsList className="grid w-full grid-cols-3">
      <TabsTrigger value="summary">Summary</TabsTrigger>
      <TabsTrigger value="performance">Performance</TabsTrigger>
      <TabsTrigger value="analytics">Analytics</TabsTrigger>
    </TabsList>
    
    {/* Tab 1: Summary (existing content) */}
    <TabsContent value="summary" className="space-y-4">
      {/* Existing VM summary, KPIs, backup table... */}
    </TabsContent>
    
    {/* Tab 2: Performance */}
    <TabsContent value="performance" className="space-y-4">
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <CardTitle>Job Performance</CardTitle>
            {backups && backups.length > 0 && (
              <Select
                value={selectedBackupId || backups[0].backup_id}
                onValueChange={setSelectedBackupId}
              >
                <SelectTrigger className="w-[250px]">
                  <SelectValue placeholder="Select backup job" />
                </SelectTrigger>
                <SelectContent>
                  {backups.map((backup: any) => (
                    <SelectItem key={backup.backup_id} value={backup.backup_id}>
                      {formatTimestamp(backup.created_at)} - {backup.type}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            )}
          </div>
        </CardHeader>
        <CardContent>
          {telemetryData?.data_source === 'rolled_up' && telemetryData.performance_metrics ? (
            <div className="space-y-6">
              {/* Performance Stats */}
              <div className="grid grid-cols-3 gap-4">
                <div className="text-center">
                  <div className="text-sm text-muted-foreground">Peak Speed</div>
                  <div className="text-2xl font-bold">
                    {telemetryData.performance_metrics.peak_speed_mbps.toFixed(1)} MB/s
                  </div>
                </div>
                <div className="text-center">
                  <div className="text-sm text-muted-foreground">Avg Speed</div>
                  <div className="text-2xl font-bold">
                    {telemetryData.performance_metrics.avg_speed_mbps.toFixed(1)} MB/s
                  </div>
                </div>
                <div className="text-center">
                  <div className="text-sm text-muted-foreground">Duration</div>
                  <div className="text-2xl font-bold">
                    {formatDuration(telemetryData.performance_metrics.duration_seconds * 1000)}
                  </div>
                </div>
              </div>
              
              {/* Speed Chart */}
              <SpeedHistogram 
                data={telemetryData.performance_metrics.chart_data}
                className="mt-6"
              />
            </div>
          ) : telemetryData?.data_source === 'live_snapshots' ? (
            <div className="space-y-6">
              <div className="text-center py-4">
                <div className="text-sm text-muted-foreground">Job In Progress</div>
                <div className="text-lg font-medium">Real-time performance data</div>
              </div>
              {telemetryData.chart_data && (
                <SpeedHistogram 
                  data={telemetryData.chart_data}
                  className="mt-6"
                />
              )}
            </div>
          ) : (
            <div className="text-center py-12 text-muted-foreground">
              <p>No performance data available for this backup</p>
              <p className="text-sm mt-2">Select a backup to view performance charts</p>
            </div>
          )}
        </CardContent>
      </Card>
    </TabsContent>
    
    {/* Tab 3: Analytics */}
    <TabsContent value="analytics" className="space-y-4">
      <div className="grid grid-cols-2 gap-4">
        <Card>
          <CardHeader>
            <CardTitle>Success Rate</CardTitle>
          </CardHeader>
          <CardContent>
            <SuccessDonut 
              successCount={completedBackups.length}
              failedCount={backups?.filter((b: any) => b.status === 'failed').length || 0}
            />
          </CardContent>
        </Card>
        
        <Card>
          <CardHeader>
            <CardTitle>Backup Trends</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="space-y-4">
              <div>
                <div className="text-sm text-muted-foreground">Total Backups</div>
                <div className="text-3xl font-bold">{totalBackups}</div>
              </div>
              <div>
                <div className="text-sm text-muted-foreground">Success Rate</div>
                <div className="text-3xl font-bold text-green-600">{successRate}</div>
              </div>
              <div>
                <div className="text-sm text-muted-foreground">Avg Size</div>
                <div className="text-2xl font-bold">{avgSizeFormatted}</div>
              </div>
              <div>
                <div className="text-sm text-muted-foreground">Avg Duration</div>
                <div className="text-2xl font-bold">{avgDurationFormatted}</div>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>
    </TabsContent>
  </Tabs>
</DialogContent>
```

---

## 🧪 Phase 4: Testing & Verification

### Backend Testing

1. **Database Migration**
```bash
# Run migration
mysql -u oma_user -poma_password migratekit_oma < source/current/sha/database/migrations/20251010_telemetry_history.sql

# Verify tables
mysql -u oma_user -poma_password migratekit_oma -e "SHOW TABLES LIKE '%telemetry%'"
```

2. **Test Snapshot Storage**
```bash
# Start a backup
curl -X POST http://localhost:8082/api/v1/backups -H "Content-Type: application/json" -d '{"vm_name":"pgtest1","backup_type":"incremental","repository_id":"repo-local-1759780872"}'

# Check snapshots (during backup)
mysql -u oma_user -poma_password migratekit_oma -e "SELECT COUNT(*) FROM job_telemetry_snapshots"
```

3. **Test Completion Rollup**
```bash
# After backup completes, check performance_metrics
mysql -u oma_user -poma_password migratekit_oma -e "SELECT id, performance_metrics FROM backup_jobs WHERE id='backup-pgtest1-XXXXX' \G"

# Verify snapshots deleted
mysql -u oma_user -poma_password migratekit_oma -e "SELECT COUNT(*) FROM job_telemetry_snapshots WHERE job_id='backup-pgtest1-XXXXX'"
```

4. **Test API Endpoint**
```bash
# Get telemetry history for completed job
curl -s "http://localhost:8082/api/v1/telemetry/history/backup-pgtest1-XXXXX" | jq .

# Should return rolled-up metrics with chart_data array
```

---

### Frontend Testing

1. **Tab Navigation**
   - [ ] Open modal, verify 3 tabs visible
   - [ ] Click each tab, verify content displays
   - [ ] Verify tabs remember state when switching

2. **Performance Tab**
   - [ ] Select different backups from dropdown
   - [ ] Verify chart updates with new data
   - [ ] Check peak/avg speed stats display
   - [ ] Verify chart axes labeled correctly
   - [ ] Hover over chart line, verify tooltip

3. **Analytics Tab**
   - [ ] Verify donut chart shows correct success/fail ratio
   - [ ] Check percentage in center of donut
   - [ ] Verify trend stats display correctly

4. **Real-Time Updates**
   - [ ] Start a backup
   - [ ] Open modal while backup running
   - [ ] Go to Performance tab
   - [ ] Verify chart updates every 3 seconds
   - [ ] Check progress indicator moves

---

## 📋 Implementation Checklist

### Backend (SHA)
- [ ] Run database migration (`20251010_telemetry_history.sql`)
- [ ] Update `telemetry_service.go` to store snapshots
- [ ] Create `telemetry_completion_handler.go`
- [ ] Update `backup.go` to call rollup on completion
- [ ] Add `GetTelemetryHistory` API endpoint
- [ ] Register new route in handlers
- [ ] Build and deploy `sendense-hub-vX.X.X`

### Frontend (GUI)
- [ ] Install `recharts` npm package
- [ ] Create `useTelemetryHistory.ts` hook
- [ ] Create `SpeedHistogram.tsx` chart component
- [ ] Create `SuccessDonut.tsx` chart component
- [ ] Update `MachineDetailsModal.tsx` with tabs
- [ ] Add job selector dropdown
- [ ] Wire up telemetry data to charts
- [ ] Test all functionality

### Testing
- [ ] Test snapshot storage during backup
- [ ] Test rollup on completion
- [ ] Test API endpoint returns correct data
- [ ] Test real-time chart updates
- [ ] Test completed job charts
- [ ] Test tab navigation
- [ ] Test job selector dropdown

---

## 🎯 Success Criteria

1. ✅ Telemetry snapshots stored every 30s during backup
2. ✅ Snapshots rolled up to JSON on completion
3. ✅ Snapshots deleted after rollup (database stays lean)
4. ✅ API returns historical chart data
5. ✅ Modal has 3 tabs (Summary, Performance, Analytics)
6. ✅ Speed histogram displays correctly
7. ✅ Donut chart shows success rate
8. ✅ Job selector switches between backups
9. ✅ Real-time updates while backup running
10. ✅ No performance impact on backup speed

---

## 📐 Data Flow Diagram

```
Backup Start
    ↓
SBC sends telemetry every 30s
    ↓
SHA: ProcessTelemetryUpdate()
    ├─→ Updates backup_jobs (current state)
    └─→ Inserts job_telemetry_snapshots (history)
    ↓
Backup Completes
    ↓
SHA: RollUpTelemetry()
    ├─→ Fetches all snapshots
    ├─→ Calculates peak/avg/min speed
    ├─→ Builds chart_data array
    ├─→ Saves to backup_jobs.performance_metrics (JSON)
    └─→ Deletes snapshots
    ↓
Frontend: useTelemetryHistory()
    ├─→ GET /api/v1/telemetry/history/{job_id}
    └─→ Receives chart data
    ↓
Display Charts
```

---

## 💡 Optional Enhancements (Future)

1. **More Chart Types**
   - Bar chart for size comparison
   - Area chart for cumulative data
   - Timeline view of backup schedule

2. **Advanced Analytics**
   - Week-over-week comparison
   - Anomaly detection (unusual speeds)
   - Predictive failure detection

3. **Export Functionality**
   - Export chart as PNG
   - Download telemetry data as CSV
   - Generate PDF report

4. **Zoom & Pan**
   - Interactive chart zoom
   - Pan timeline for long backups
   - Detail view on hover

---

## 📝 Notes

- **Storage is efficient**: Snapshots deleted after rollup, only JSON kept
- **Performance overhead is minimal**: Async snapshot inserts, no backup slowdown
- **Real-time works**: Live snapshots for running jobs, rolled-up for completed
- **Backward compatible**: Old backups without metrics still display in table
- **Chart library**: Recharts is React-native, well-maintained, TypeScript-friendly

---

**END OF JOB SHEET**

**Estimated Total Time:** 6-8 hours
- Backend: 3-4 hours
- Frontend: 2-3 hours  
- Testing: 1 hour

Ready for Grok to implement! 🚀

