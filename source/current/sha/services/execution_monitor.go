package services

import (
	"context"
	"time"

	"github.com/vexxhost/migratekit-sha/database"
	log "github.com/sirupsen/logrus"
)

// ExecutionMonitor watches running flow executions and updates their status
// when all associated backup jobs complete
type ExecutionMonitor struct {
	flowRepo *database.FlowRepository
	db       database.Connection
	ticker   *time.Ticker
	stopChan chan struct{}
}

// NewExecutionMonitor creates a new execution monitor
func NewExecutionMonitor(flowRepo *database.FlowRepository, db database.Connection) *ExecutionMonitor {
	return &ExecutionMonitor{
		flowRepo: flowRepo,
		db:       db,
		stopChan: make(chan struct{}),
	}
}

// Start begins monitoring executions every 10 seconds
func (em *ExecutionMonitor) Start() {
	em.ticker = time.NewTicker(10 * time.Second)
	
	log.Info("🔍 Execution Monitor started - checking every 10 seconds")
	
	go func() {
		// Run immediately on start
		em.checkRunningExecutions()
		
		// Then run every 10 seconds
		for {
			select {
			case <-em.ticker.C:
				em.checkRunningExecutions()
			case <-em.stopChan:
				log.Info("Execution Monitor stopped")
				return
			}
		}
	}()
}

// Stop stops the monitor
func (em *ExecutionMonitor) Stop() {
	if em.ticker != nil {
		em.ticker.Stop()
	}
	close(em.stopChan)
}

// checkRunningExecutions finds all running executions and checks if their jobs are complete
func (em *ExecutionMonitor) checkRunningExecutions() {
	ctx := context.Background()
	
	// Find all executions with status="running"
	var runningExecutions []database.ProtectionFlowExecution
	if err := em.db.GetGormDB().Where("status = ?", "running").Find(&runningExecutions).Error; err != nil {
		log.WithError(err).Error("Failed to query running executions")
		return
	}
	
	if len(runningExecutions) == 0 {
		return // No running executions
	}
	
	log.WithField("count", len(runningExecutions)).Debug("Checking running executions")
	
	for _, execution := range runningExecutions {
		em.checkExecution(ctx, &execution)
	}
}

// checkExecution checks a single execution and updates its status if jobs are complete
func (em *ExecutionMonitor) checkExecution(ctx context.Context, execution *database.ProtectionFlowExecution) {
	// Get the backup job IDs from created_job_ids JSON field
	if execution.CreatedJobIDs == nil || *execution.CreatedJobIDs == "" {
		log.WithField("execution_id", execution.ID).Warn("Execution has no created_job_ids")
		return
	}
	
	// Parse JSON array of job IDs
	var jobIDs []string
	if err := em.db.GetGormDB().Raw("SELECT JSON_UNQUOTE(JSON_EXTRACT(?, CONCAT('$[', idx, ']'))) FROM "+
		"(SELECT 0 AS idx UNION SELECT 1 UNION SELECT 2 UNION SELECT 3 UNION SELECT 4) AS numbers "+
		"WHERE JSON_EXTRACT(?, CONCAT('$[', idx, ']')) IS NOT NULL",
		*execution.CreatedJobIDs, *execution.CreatedJobIDs).Scan(&jobIDs).Error; err != nil {
		log.WithError(err).WithField("execution_id", execution.ID).Error("Failed to parse created_job_ids")
		return
	}
	
	if len(jobIDs) == 0 {
		return
	}
	
	// Check status of all backup jobs
	var jobStatuses []struct {
		ID     string
		Status string
	}
	
	if err := em.db.GetGormDB().
		Table("backup_jobs").
		Select("id, status").
		Where("id IN ?", jobIDs).
		Scan(&jobStatuses).Error; err != nil {
		log.WithError(err).WithField("execution_id", execution.ID).Error("Failed to query backup job statuses")
		return
	}
	
	// Count completed and failed jobs
	var completed, failed int
	for _, job := range jobStatuses {
		switch job.Status {
		case "completed":
			completed++
		case "failed":
			failed++
		}
	}
	
	totalJobs := len(jobStatuses)
	
	// Check if all jobs are done (completed or failed)
	if completed+failed < totalJobs {
		// Still running
		log.WithFields(log.Fields{
			"execution_id": execution.ID,
			"total_jobs":   totalJobs,
			"completed":    completed,
			"failed":       failed,
			"running":      totalJobs - completed - failed,
		}).Debug("Execution still has running jobs")
		return
	}
	
	// All jobs are done - update execution status
	finalStatus := "success"
	if failed > 0 {
		if completed == 0 {
			finalStatus = "error" // All failed
		} else {
			finalStatus = "warning" // Some failed
		}
	}
	
	now := time.Now()
	executionTime := int(now.Sub(*execution.StartedAt).Seconds())
	
	log.WithFields(log.Fields{
		"execution_id":     execution.ID,
		"final_status":     finalStatus,
		"jobs_completed":   completed,
		"jobs_failed":      failed,
		"execution_time":   executionTime,
	}).Info("✅ Execution complete - updating status")
	
	// Update execution record
	if err := em.flowRepo.UpdateExecutionStatus(ctx, execution.ID, finalStatus, map[string]interface{}{
		"jobs_completed":         completed,
		"jobs_failed":            failed,
		"completed_at":           now,
		"execution_time_seconds": executionTime,
	}); err != nil {
		log.WithError(err).WithField("execution_id", execution.ID).Error("Failed to update execution status")
		return
	}
	
	// Update flow statistics
	flow, err := em.flowRepo.GetFlowByID(ctx, execution.FlowID)
	if err != nil {
		log.WithError(err).WithField("flow_id", execution.FlowID).Error("Failed to get flow for statistics update")
		return
	}
	
	successIncrement := 0
	failureIncrement := 0
	if finalStatus == "success" {
		successIncrement = 1
	} else if finalStatus == "error" {
		failureIncrement = 1
	}
	
	if err := em.flowRepo.UpdateFlowStatistics(ctx, execution.FlowID, database.FlowStatistics{
		LastExecutionID:      &execution.ID,
		LastExecutionStatus:  finalStatus,
		LastExecutionTime:    &now,
		TotalExecutions:      flow.TotalExecutions, // Don't increment (already done on start)
		SuccessfulExecutions: flow.SuccessfulExecutions + successIncrement,
		FailedExecutions:     flow.FailedExecutions + failureIncrement,
	}); err != nil {
		log.WithError(err).WithField("flow_id", execution.FlowID).Error("Failed to update flow statistics")
	}
	
	log.WithFields(log.Fields{
		"execution_id": execution.ID,
		"flow_id":      execution.FlowID,
		"status":       finalStatus,
	}).Info("🎉 Execution monitoring complete - flow updated")
}

