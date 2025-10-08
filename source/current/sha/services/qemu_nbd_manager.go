// Package services provides the qemu-nbd Process Manager
// Manages lifecycle of qemu-nbd instances for NBD exports
package services

import (
	"fmt"
	"os/exec"
	"strconv"
	"sync"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"
)

// QemuNBDManager manages running qemu-nbd processes
// Tracks process lifecycle, health monitoring, and cleanup
type QemuNBDManager struct {
	processes     map[int]*QemuNBDProcess // port -> process details
	portAllocator *NBDPortAllocator       // Optional: for automatic port release
	mu            sync.RWMutex
}

// QemuNBDProcess represents a running qemu-nbd instance
type QemuNBDProcess struct {
	Port          int           // NBD server port
	ExportName    string        // NBD export name
	FilePath      string        // Path to QCOW2 file
	PID           int           // Process ID
	StartTime     time.Time     // When the process started
	JobID         string        // Associated job ID
	VMName        string        // VM name
	DiskID        int           // Disk number
	Cmd           *exec.Cmd     // Command reference (for cleanup)
}

// NewQemuNBDManager creates a new qemu-nbd process manager
// portAllocator is optional - if provided, ports will be automatically released on Stop()
func NewQemuNBDManager(portAllocator *NBDPortAllocator) *QemuNBDManager {
	manager := &QemuNBDManager{
		processes:     make(map[int]*QemuNBDProcess),
		portAllocator: portAllocator,
	}
	
	if portAllocator != nil {
		log.Info("🖥️  qemu-nbd Process Manager initialized with automatic port release")
	} else {
		log.Info("🖥️  qemu-nbd Process Manager initialized (manual port management)")
	}
	
	return manager
}

// Start launches a new qemu-nbd instance
// Returns the PID of the started process
func (m *QemuNBDManager) Start(port int, exportName, filePath, jobID, vmName string, diskID int) (*QemuNBDProcess, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// Check if port is already in use
	if existing, exists := m.processes[port]; exists {
		log.WithFields(log.Fields{
			"port":         port,
			"existing_pid": existing.PID,
			"existing_job": existing.JobID,
		}).Warn("⚠️  Port already in use by existing qemu-nbd process")
		return nil, fmt.Errorf("port %d already in use by PID %d (job %s)", port, existing.PID, existing.JobID)
	}
	
	// Build qemu-nbd command
	// --shared=10: Allow up to 10 concurrent connections (migratekit needs 2: metadata + data)
	// -f qcow2: QCOW2 format
	// -x exportName: NBD export name
	// -p port: Listen port
	// -b 0.0.0.0: Bind to all interfaces (accessible via tunnel)
	// -t: Enable write-through cache
	cmd := exec.Command("qemu-nbd",
		"-f", "qcow2",
		"-x", exportName,
		"-p", strconv.Itoa(port),
		"-b", "0.0.0.0",
		"--shared", "10",
		"-t",
		filePath,
	)
	
	// Start the process
	if err := cmd.Start(); err != nil {
		log.WithFields(log.Fields{
			"port":        port,
			"export_name": exportName,
			"file_path":   filePath,
			"job_id":      jobID,
			"error":       err,
		}).Error("❌ Failed to start qemu-nbd process")
		return nil, fmt.Errorf("failed to start qemu-nbd: %w", err)
	}
	
	pid := cmd.Process.Pid
	
	// Create process tracking record
	process := &QemuNBDProcess{
		Port:       port,
		ExportName: exportName,
		FilePath:   filePath,
		PID:        pid,
		StartTime:  time.Now(),
		JobID:      jobID,
		VMName:     vmName,
		DiskID:     diskID,
		Cmd:        cmd,
	}
	
	m.processes[port] = process
	
	log.WithFields(log.Fields{
		"port":        port,
		"pid":         pid,
		"export_name": exportName,
		"job_id":      jobID,
		"vm_name":     vmName,
		"disk_id":     diskID,
		"file_path":   filePath,
	}).Info("✅ qemu-nbd process started successfully")
	
	// Start background monitoring for this process
	go m.monitorProcess(port, pid)
	
	return process, nil
}

// Stop terminates a qemu-nbd process
func (m *QemuNBDManager) Stop(port int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	process, exists := m.processes[port]
	if !exists {
		log.WithField("port", port).Warn("⚠️  No qemu-nbd process found on port")
		return fmt.Errorf("no qemu-nbd process on port %d", port)
	}
	
	// Try graceful SIGTERM first
	if err := process.Cmd.Process.Signal(syscall.SIGTERM); err != nil {
		log.WithFields(log.Fields{
			"port": port,
			"pid":  process.PID,
		}).Warn("⚠️  Failed to send SIGTERM, trying SIGKILL")
		
		// Force kill if SIGTERM fails
		if killErr := process.Cmd.Process.Kill(); killErr != nil {
			log.WithError(killErr).Error("❌ Failed to kill qemu-nbd process")
			return fmt.Errorf("failed to kill qemu-nbd PID %d: %w", process.PID, killErr)
		}
	}
	
	// Wait for process to exit (with timeout)
	done := make(chan error, 1)
	go func() {
		done <- process.Cmd.Wait()
	}()
	
	select {
	case <-done:
		// Process exited successfully
	case <-time.After(5 * time.Second):
		log.WithField("pid", process.PID).Warn("⚠️  Process did not exit within timeout, forcing kill")
		process.Cmd.Process.Kill()
		<-done // Wait for forced kill to complete
	}
	
	// Give kernel time to release file locks (100ms should be plenty)
	time.Sleep(100 * time.Millisecond)
	
	duration := time.Since(process.StartTime)
	
	// Release NBD port if portAllocator available
	if m.portAllocator != nil {
		m.portAllocator.Release(port)
		log.WithField("port", port).Debug("✅ NBD port released via port allocator")
	}
	
	// Remove from tracking
	delete(m.processes, port)
	
	log.WithFields(log.Fields{
		"port":      port,
		"pid":       process.PID,
		"job_id":    process.JobID,
		"vm_name":   process.VMName,
		"uptime":    duration.Round(time.Second),
	}).Info("🛑 qemu-nbd process stopped and cleaned up")
	
	return nil
}

// StopByJobID stops all qemu-nbd processes for a specific job
func (m *QemuNBDManager) StopByJobID(jobID string) int {
	m.mu.Lock()
	portsToStop := []int{}
	for port, process := range m.processes {
		if process.JobID == jobID {
			portsToStop = append(portsToStop, port)
		}
	}
	m.mu.Unlock()
	
	// Stop processes outside the lock
	stopped := 0
	for _, port := range portsToStop {
		if err := m.Stop(port); err != nil {
			log.WithError(err).Errorf("Failed to stop qemu-nbd on port %d", port)
		} else {
			stopped++
		}
	}
	
	if stopped > 0 {
		log.WithFields(log.Fields{
			"job_id":  jobID,
			"stopped": stopped,
		}).Info("✅ Stopped all qemu-nbd processes for job")
	}
	
	return stopped
}

// GetStatus returns status of a qemu-nbd process on a specific port
func (m *QemuNBDManager) GetStatus(port int) (*QemuNBDProcess, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	process, exists := m.processes[port]
	if !exists {
		return nil, fmt.Errorf("no qemu-nbd process on port %d", port)
	}
	
	// Return a copy to prevent external modification
	copy := *process
	copy.Cmd = nil // Don't expose internal cmd reference
	return &copy, nil
}

// GetAllProcesses returns status of all running qemu-nbd processes
func (m *QemuNBDManager) GetAllProcesses() map[int]QemuNBDProcess {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	result := make(map[int]QemuNBDProcess)
	for port, process := range m.processes {
		copy := *process
		copy.Cmd = nil // Don't expose internal cmd reference
		result[port] = copy
	}
	return result
}

// GetProcessCount returns the number of running qemu-nbd processes
func (m *QemuNBDManager) GetProcessCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.processes)
}

// IsPortActive checks if a qemu-nbd process is running on a port
func (m *QemuNBDManager) IsPortActive(port int) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, exists := m.processes[port]
	return exists
}

// monitorProcess watches a qemu-nbd process and logs if it crashes
func (m *QemuNBDManager) monitorProcess(port, pid int) {
	// Wait for process to exit
	m.mu.RLock()
	process, exists := m.processes[port]
	m.mu.RUnlock()
	
	if !exists {
		return
	}
	
	err := process.Cmd.Wait()
	
	// Check if process is still tracked (might have been stopped normally)
	m.mu.RLock()
	_, stillTracked := m.processes[port]
	m.mu.RUnlock()
	
	if stillTracked {
		// Process crashed or exited unexpectedly
		uptime := time.Since(process.StartTime)
		
		log.WithFields(log.Fields{
			"port":      port,
			"pid":       pid,
			"job_id":    process.JobID,
			"vm_name":   process.VMName,
			"uptime":    uptime.Round(time.Second),
			"exit_err":  err,
		}).Error("💥 qemu-nbd process died unexpectedly")
		
		// Clean up tracking
		m.mu.Lock()
		delete(m.processes, port)
		m.mu.Unlock()
	}
}

// GetMetrics returns comprehensive metrics about qemu-nbd processes
func (m *QemuNBDManager) GetMetrics() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	totalProcesses := len(m.processes)
	
	// Calculate average uptime
	var totalUptime time.Duration
	for _, process := range m.processes {
		totalUptime += time.Since(process.StartTime)
	}
	
	avgUptime := time.Duration(0)
	if totalProcesses > 0 {
		avgUptime = totalUptime / time.Duration(totalProcesses)
	}
	
	return map[string]interface{}{
		"total_processes": totalProcesses,
		"average_uptime_seconds": avgUptime.Seconds(),
	}
}
