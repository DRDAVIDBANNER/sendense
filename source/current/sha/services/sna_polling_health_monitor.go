// Package services provides SNA polling health monitoring
// Detects and recovers jobs that should be polling but aren't
package services

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/vexxhost/migratekit-sha/database"
)

// SNAPollingHealthMonitor continuously monitors for orphaned jobs that should be polling but aren't
type SNAPollingHealthMonitor struct {
	db                database.Connection
	snaClient         *SNAProgressClient
	snaProgressPoller *SNAProgressPoller
	checkInterval     time.Duration
	staleThreshold    time.Duration // How long without updates before checking
	stopChan          chan struct{}
	running           bool
}

// NewVMAPollingHealthMonitor creates a new health monitor
func NewVMAPollingHealthMonitor(
	db database.Connection,
	snaClient *SNAProgressClient,
	snaProgressPoller *SNAProgressPoller,
) *SNAPollingHealthMonitor {
	return &SNAPollingHealthMonitor{
		db:                db,
		snaClient:         snaClient,
		snaProgressPoller: snaProgressPoller,
		checkInterval:     2 * time.Minute,  // Check every 2 minutes
		staleThreshold:    30 * time.Second, // Flag if no update for 30 seconds
		stopChan:          make(chan struct{}),
		running:           false,
	}
}

// Start begins the health monitoring service
func (hm *SNAPollingHealthMonitor) Start(ctx context.Context) error {
	if hm.running {
		return fmt.Errorf("health monitor already running")
	}

	hm.running = true
	log.Println("🏥 Starting SNA polling health monitor")
	log.Printf("   Check interval: %v", hm.checkInterval)
	log.Printf("   Stale threshold: %v", hm.staleThreshold)

	go hm.monitoringLoop(ctx)

	return nil
}

// Stop gracefully stops the health monitoring service
func (hm *SNAPollingHealthMonitor) Stop() {
	if !hm.running {
		return
	}

	log.Println("🛑 Stopping SNA polling health monitor")
	close(hm.stopChan)
	hm.running = false
}

// monitoringLoop is the main monitoring loop
func (hm *SNAPollingHealthMonitor) monitoringLoop(ctx context.Context) {
	ticker := time.NewTicker(hm.checkInterval)
	defer ticker.Stop()

	log.Println("🏥 SNA polling health monitor started")

	for {
		select {
		case <-ctx.Done():
			log.Println("🏥 Health monitor stopped due to context cancellation")
			return
		case <-hm.stopChan:
			log.Println("🏥 Health monitor stopped")
			return
		case <-ticker.C:
			hm.checkForOrphanedJobs(ctx)
		}
	}
}

// checkForOrphanedJobs checks for jobs that should be polling but aren't
func (hm *SNAPollingHealthMonitor) checkForOrphanedJobs(ctx context.Context) {
	// Find jobs in "replicating" status that haven't been updated recently
	var staleJobs []database.ReplicationJob
	
	staleTime := time.Now().Add(-hm.staleThreshold)
	
	query := hm.db.GetGormDB().
		Where("status = ?", "replicating").
		Where("vma_last_poll_at IS NOT NULL").
		Where("vma_last_poll_at < ?", staleTime)
	
	if err := query.Find(&staleJobs).Error; err != nil {
		log.Printf("❌ Health monitor: Failed to query stale jobs: %v", err)
		return
	}

	if len(staleJobs) == 0 {
		log.Println("✅ Health monitor: No stale jobs detected - all polling active")
		return
	}

	log.Printf("🔍 Health monitor: Found %d jobs with stale polling", len(staleJobs))

	// Check each stale job
	for _, job := range staleJobs {
		stagnantMinutes := time.Since(job.UpdatedAt).Minutes()
		pollAgeSeconds := int64(0)
		if job.SNALastPollAt != nil {
			pollAgeSeconds = int64(time.Since(*job.SNALastPollAt).Seconds())
		}

		log.Printf("🔍 Health monitor: Checking stale job: %s (%s) - poll age: %ds, stagnant: %.1f min",
			job.ID, job.SourceVMName, pollAgeSeconds, stagnantMinutes)

		// Check if job is being polled
		if hm.snaProgressPoller != nil {
			activeJobs := hm.snaProgressPoller.GetPollingStatus()
			if jobList, ok := activeJobs["active_jobs"].([]map[string]interface{}); ok {
				isPolling := false
				for _, activeJob := range jobList {
					if activeJobID, ok := activeJob["job_id"].(string); ok && activeJobID == job.ID {
						isPolling = true
						break
					}
				}

				if isPolling {
					log.Printf("✅ Health monitor: Job %s is being polled (polling may be slow)", job.ID)
					continue // Job is in polling map, just slow
				}
			}
		}

		// Job is NOT being polled - this is the orphaned case
		log.Printf("🚨 Health monitor: Job %s NOT being polled (orphaned) - investigating",
			job.ID)

		// Validate with SNA and take action
		hm.recoverOrphanedJob(ctx, &job, stagnantMinutes)
	}
}

// recoverOrphanedJob handles a specific orphaned job detected by health monitor
func (hm *SNAPollingHealthMonitor) recoverOrphanedJob(
	ctx context.Context,
	job *database.ReplicationJob,
	stagnantMinutes float64,
) {
	// Create a temporary recovery instance to use existing logic
	recovery := &ProductionJobRecovery{
		db:                hm.db,
		snaClient:         hm.snaClient,
		snaProgressPoller: hm.snaProgressPoller,
		maxJobAge:         30 * time.Minute,
		recoveryEnabled:   true,
	}

	// Check SNA status
	snaStatus, err := recovery.checkVMAStatus(ctx, job)
	if err != nil && snaStatus == nil {
		log.Printf("❌ Health monitor: Failed to check SNA for job %s: %v", job.ID, err)
		return
	}

	// Make recovery decision
	log.Printf("🎯 Health monitor recovery decision for %s: SNA status=%s, stagnant=%.1f min",
		job.ID, snaStatus.Status, stagnantMinutes)

	if err := recovery.recoverJobWithVMAValidation(job, snaStatus, stagnantMinutes); err != nil {
		log.Printf("❌ Health monitor: Failed to recover job %s: %v", job.ID, err)
	} else {
		log.Printf("✅ Health monitor: Successfully handled orphaned job %s", job.ID)
	}
}

// GetStatus returns current health monitor status
func (hm *SNAPollingHealthMonitor) GetStatus() map[string]interface{} {
	return map[string]interface{}{
		"running":          hm.running,
		"check_interval":   hm.checkInterval.String(),
		"stale_threshold":  hm.staleThreshold.String(),
	}
}


