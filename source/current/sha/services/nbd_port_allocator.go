// Package services provides the NBD Port Allocator for dynamic port assignment
// Manages port allocation for concurrent qemu-nbd instances in the range 10100-10200
package services

import (
	"fmt"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

// NBDPortAllocator manages dynamic allocation of NBD ports for backup/replication jobs
type NBDPortAllocator struct {
	mu          sync.RWMutex
	minPort     int
	maxPort     int
	allocated   map[int]*PortAllocation // port -> allocation details
}

// PortAllocation tracks details about an allocated port
type PortAllocation struct {
	Port         int
	JobID        string
	AllocatedAt  time.Time
	VMName       string
	ExportName   string
}

// NewNBDPortAllocator creates a new port allocator for the specified range
// Default range: 10100-10200 (100 concurrent jobs)
func NewNBDPortAllocator(minPort, maxPort int) *NBDPortAllocator {
	allocator := &NBDPortAllocator{
		minPort:   minPort,
		maxPort:   maxPort,
		allocated: make(map[int]*PortAllocation),
	}
	
	log.WithFields(log.Fields{
		"min_port":     minPort,
		"max_port":     maxPort,
		"total_ports":  maxPort - minPort + 1,
	}).Info("📡 NBD Port Allocator initialized")
	
	return allocator
}

// Allocate assigns an available port to a job
// Returns the allocated port number or error if no ports available
func (a *NBDPortAllocator) Allocate(jobID, vmName, exportName string) (int, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	
	// Find first available port
	for port := a.minPort; port <= a.maxPort; port++ {
		if _, exists := a.allocated[port]; !exists {
			allocation := &PortAllocation{
				Port:        port,
				JobID:       jobID,
				AllocatedAt: time.Now(),
				VMName:      vmName,
				ExportName:  exportName,
			}
			
			a.allocated[port] = allocation
			
			log.WithFields(log.Fields{
				"port":        port,
				"job_id":      jobID,
				"vm_name":     vmName,
				"export_name": exportName,
				"allocated":   len(a.allocated),
				"available":   (a.maxPort - a.minPort + 1) - len(a.allocated),
			}).Info("✅ NBD port allocated")
			
			return port, nil
		}
	}
	
	// No ports available
	log.WithFields(log.Fields{
		"job_id":      jobID,
		"vm_name":     vmName,
		"min_port":    a.minPort,
		"max_port":    a.maxPort,
		"allocated":   len(a.allocated),
	}).Error("❌ No available NBD ports")
	
	return 0, fmt.Errorf("no available ports in range %d-%d (all %d ports allocated)", 
		a.minPort, a.maxPort, a.maxPort-a.minPort+1)
}

// Release frees a port for reuse
func (a *NBDPortAllocator) Release(port int) {
	a.mu.Lock()
	defer a.mu.Unlock()
	
	allocation, exists := a.allocated[port]
	if !exists {
		log.WithField("port", port).Warn("⚠️  Attempted to release unallocated port")
		return
	}
	
	duration := time.Since(allocation.AllocatedAt)
	delete(a.allocated, port)
	
	log.WithFields(log.Fields{
		"port":        port,
		"job_id":      allocation.JobID,
		"vm_name":     allocation.VMName,
		"duration":    duration.Round(time.Second),
		"allocated":   len(a.allocated),
		"available":   (a.maxPort - a.minPort + 1) - len(a.allocated),
	}).Info("🔓 NBD port released")
}

// ReleaseByJobID releases all ports allocated to a specific job
// Useful for cleanup when a job fails or completes
func (a *NBDPortAllocator) ReleaseByJobID(jobID string) int {
	a.mu.Lock()
	defer a.mu.Unlock()
	
	released := 0
	for port, allocation := range a.allocated {
		if allocation.JobID == jobID {
			delete(a.allocated, port)
			released++
			
			log.WithFields(log.Fields{
				"port":    port,
				"job_id":  jobID,
				"vm_name": allocation.VMName,
			}).Info("🔓 NBD port released by job ID")
		}
	}
	
	if released > 0 {
		log.WithFields(log.Fields{
			"job_id":      jobID,
			"released":    released,
			"allocated":   len(a.allocated),
			"available":   (a.maxPort - a.minPort + 1) - len(a.allocated),
		}).Info("✅ All ports for job released")
	}
	
	return released
}

// GetAllocation returns allocation details for a specific port
func (a *NBDPortAllocator) GetAllocation(port int) (*PortAllocation, bool) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	
	allocation, exists := a.allocated[port]
	if !exists {
		return nil, false
	}
	
	// Return a copy to prevent external modification
	copy := *allocation
	return &copy, true
}

// GetAllocated returns a copy of all current allocations
// Safe for external use (returns copies, not references)
func (a *NBDPortAllocator) GetAllocated() map[int]PortAllocation {
	a.mu.RLock()
	defer a.mu.RUnlock()
	
	result := make(map[int]PortAllocation)
	for port, allocation := range a.allocated {
		result[port] = *allocation // Copy value, not reference
	}
	return result
}

// GetAvailableCount returns the number of unallocated ports
func (a *NBDPortAllocator) GetAvailableCount() int {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return (a.maxPort - a.minPort + 1) - len(a.allocated)
}

// GetAllocatedCount returns the number of currently allocated ports
func (a *NBDPortAllocator) GetAllocatedCount() int {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return len(a.allocated)
}

// GetTotalPorts returns the total number of ports in the managed range
func (a *NBDPortAllocator) GetTotalPorts() int {
	return a.maxPort - a.minPort + 1
}

// GetMetrics returns comprehensive metrics about port allocation
func (a *NBDPortAllocator) GetMetrics() map[string]interface{} {
	a.mu.RLock()
	defer a.mu.RUnlock()
	
	total := a.maxPort - a.minPort + 1
	allocated := len(a.allocated)
	available := total - allocated
	utilizationPercent := float64(allocated) / float64(total) * 100
	
	return map[string]interface{}{
		"total_ports":          total,
		"allocated_ports":      allocated,
		"available_ports":      available,
		"utilization_percent":  utilizationPercent,
		"min_port":             a.minPort,
		"max_port":             a.maxPort,
	}
}

// IsPortAllocated checks if a specific port is currently allocated
func (a *NBDPortAllocator) IsPortAllocated(port int) bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	_, exists := a.allocated[port]
	return exists
}

// GetJobPorts returns all ports allocated to a specific job
func (a *NBDPortAllocator) GetJobPorts(jobID string) []int {
	a.mu.RLock()
	defer a.mu.RUnlock()
	
	var ports []int
	for port, allocation := range a.allocated {
		if allocation.JobID == jobID {
			ports = append(ports, port)
		}
	}
	return ports
}
