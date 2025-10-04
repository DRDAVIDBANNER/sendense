// Package services provides VMA connection monitoring for enrollment system
package services

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/vexxhost/migratekit-oma/database"
	"github.com/vexxhost/migratekit-oma/models"
)

// VMAConnectionMonitor monitors active VMA tunnel connections
type VMAConnectionMonitor struct {
	enrollmentRepo  *database.VMAEnrollmentRepository
	auditRepo       *database.VMAAuditRepository
	monitorInterval time.Duration
	stopChan        chan struct{}
}

// NewVMAConnectionMonitor creates a new VMA connection monitor
func NewVMAConnectionMonitor(
	enrollmentRepo *database.VMAEnrollmentRepository,
	auditRepo *database.VMAAuditRepository,
) *VMAConnectionMonitor {
	return &VMAConnectionMonitor{
		enrollmentRepo:  enrollmentRepo,
		auditRepo:       auditRepo,
		monitorInterval: 60 * time.Second, // Check every minute
		stopChan:        make(chan struct{}),
	}
}

// Start begins connection monitoring
func (vcm *VMAConnectionMonitor) Start(ctx context.Context) error {
	log.Info("🔍 Starting VMA connection monitor")

	go vcm.monitorLoop(ctx)
	return nil
}

// Stop stops connection monitoring
func (vcm *VMAConnectionMonitor) Stop() {
	log.Info("🛑 Stopping VMA connection monitor")
	close(vcm.stopChan)
}

// monitorLoop runs the connection monitoring loop
func (vcm *VMAConnectionMonitor) monitorLoop(ctx context.Context) {
	ticker := time.NewTicker(vcm.monitorInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := vcm.checkConnections(ctx); err != nil {
				log.WithError(err).Error("Failed to check VMA connections")
			}
		case <-vcm.stopChan:
			return
		case <-ctx.Done():
			return
		}
	}
}

// checkConnections checks the status of all active VMA connections
func (vcm *VMAConnectionMonitor) checkConnections(ctx context.Context) error {
	// Get list of active SSH connections for vma_tunnel user
	activeConnections, err := vcm.getActiveSSHConnections()
	if err != nil {
		return fmt.Errorf("failed to get active SSH connections: %w", err)
	}

	// Get active VMA enrollments from database
	activeVMAs, err := vcm.enrollmentRepo.GetActiveConnections()
	if err != nil {
		return fmt.Errorf("failed to get active VMA connections: %w", err)
	}

	// Update connection status based on actual SSH connections
	for _, vma := range activeVMAs {
		isConnected := vcm.isVMAConnected(vma.VMAFingerprint, activeConnections)

		if isConnected {
			// Update last seen timestamp
			if err := vcm.enrollmentRepo.UpdateLastSeen(vma.EnrollmentID); err != nil {
				log.WithError(err).Warn("Failed to update VMA last seen timestamp")
			}
		} else {
			// Mark as disconnected if not seen for more than 5 minutes
			if vma.LastSeenAt != nil && time.Since(*vma.LastSeenAt) > 5*time.Minute {
				if err := vcm.markVMADisconnected(ctx, vma.EnrollmentID); err != nil {
					log.WithError(err).Warn("Failed to mark VMA as disconnected")
				}
			}
		}
	}

	log.WithFields(log.Fields{
		"active_ssh_connections": len(activeConnections),
		"active_vma_records":     len(activeVMAs),
	}).Debug("🔍 VMA connection monitoring completed")

	return nil
}

// getActiveSSHConnections returns list of active SSH connections
func (vcm *VMAConnectionMonitor) getActiveSSHConnections() ([]SSHConnection, error) {
	// Use 'who' command to get active SSH sessions
	cmd := exec.Command("who")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to execute 'who' command: %w", err)
	}

	var connections []SSHConnection
	lines := strings.Split(string(output), "\n")

	for _, line := range lines {
		if strings.Contains(line, "vma_tunnel") {
			// Parse connection info from 'who' output
			fields := strings.Fields(line)
			if len(fields) >= 5 {
				connection := SSHConnection{
					User:      fields[0],
					Terminal:  fields[1],
					LoginTime: strings.Join(fields[2:4], " "),
					SourceIP:  strings.Trim(fields[4], "()"),
				}
				connections = append(connections, connection)
			}
		}
	}

	return connections, nil
}

// isVMAConnected checks if a VMA is currently connected based on SSH sessions
func (vcm *VMAConnectionMonitor) isVMAConnected(vmaFingerprint string, activeConnections []SSHConnection) bool {
	// Extract IP from fingerprint (format: "ip:port")
	parts := strings.Split(vmaFingerprint, ":")
	if len(parts) < 1 {
		return false
	}
	vmaIP := parts[0]

	// Check if VMA IP has active SSH connection
	for _, conn := range activeConnections {
		if conn.SourceIP == vmaIP && conn.User == "vma_tunnel" {
			return true
		}
	}

	return false
}

// markVMADisconnected updates VMA connection status to disconnected
func (vcm *VMAConnectionMonitor) markVMADisconnected(ctx context.Context, enrollmentID string) error {
	if err := vcm.enrollmentRepo.UpdateConnectionStatus(enrollmentID, models.ConnectionStatusDisconnected); err != nil {
		return fmt.Errorf("failed to update connection status: %w", err)
	}

	// Log disconnection event
	vcm.auditRepo.LogEvent(&models.VMAConnectionAudit{
		EnrollmentID: &enrollmentID,
		EventType:    models.AuditEventDisconnection,
		EventDetails: func() *string {
			details := `{"reason":"connection_timeout","detected_by":"monitor"}`
			return &details
		}(),
	})

	log.WithField("enrollment_id", enrollmentID).Info("🔌 VMA marked as disconnected")
	return nil
}

// GetConnectionStatistics returns VMA connection statistics
func (vcm *VMAConnectionMonitor) GetConnectionStatistics(ctx context.Context) (*ConnectionStatistics, error) {
	// Get active SSH connections
	activeConnections, err := vcm.getActiveSSHConnections()
	if err != nil {
		return nil, fmt.Errorf("failed to get active connections: %w", err)
	}

	// Get database statistics
	activeVMAs, err := vcm.enrollmentRepo.GetActiveConnections()
	if err != nil {
		return nil, fmt.Errorf("failed to get active VMA records: %w", err)
	}

	stats := &ConnectionStatistics{
		ActiveSSHConnections: len(activeConnections),
		ActiveVMARecords:     len(activeVMAs),
		LastCheckTime:        time.Now(),
	}

	return stats, nil
}

// SSHConnection represents an active SSH connection
type SSHConnection struct {
	User      string `json:"user"`
	Terminal  string `json:"terminal"`
	LoginTime string `json:"login_time"`
	SourceIP  string `json:"source_ip"`
}

// ConnectionStatistics represents VMA connection monitoring statistics
type ConnectionStatistics struct {
	ActiveSSHConnections int       `json:"active_ssh_connections"`
	ActiveVMARecords     int       `json:"active_vma_records"`
	LastCheckTime        time.Time `json:"last_check_time"`
}
