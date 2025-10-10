package database

import (
	"context"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// =============================================================================
// PROTECTION FLOW REPOSITORY - Flow orchestration operations
// =============================================================================
// Handles all protection flow CRUD operations, execution tracking, and statistics

// FlowRepository handles all protection flow-related database operations
type FlowRepository struct {
	db *gorm.DB
}

// NewFlowRepository creates a new flow repository
func NewFlowRepository(conn Connection) *FlowRepository {
	return &FlowRepository{
		db: conn.GetGormDB(),
	}
}

// FlowFilters represents filtering options for flow queries
type FlowFilters struct {
	FlowType   *string
	TargetType *string
	Enabled    *bool
	ScheduleID *string
	Limit      int
	Offset     int
}

// FlowStatistics represents flow execution statistics
type FlowStatistics struct {
	LastExecutionID     *string
	LastExecutionStatus string
	LastExecutionTime   *time.Time
	TotalExecutions     int
	SuccessfulExecutions int
	FailedExecutions    int
}

// =============================================================================
// FLOW CRUD OPERATIONS
// =============================================================================

// CreateFlow creates a new protection flow
func (r *FlowRepository) CreateFlow(ctx context.Context, flow *ProtectionFlow) error {
	if r.db == nil {
		return fmt.Errorf("database not available")
	}

	log.WithFields(log.Fields{
		"flow_name":   flow.Name,
		"flow_type":   flow.FlowType,
		"target_type": flow.TargetType,
	}).Info("Creating protection flow")

	if err := r.db.Create(flow).Error; err != nil {
		log.WithError(err).WithField("flow_name", flow.Name).Error("Failed to create flow")
		return fmt.Errorf("failed to create flow: %w", err)
	}

	log.WithField("flow_id", flow.ID).Info("Protection flow created successfully")
	return nil
}

// GetFlowByID retrieves a flow by ID with optional relationships
func (r *FlowRepository) GetFlowByID(ctx context.Context, id string) (*ProtectionFlow, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database not available")
	}

	var flow ProtectionFlow
	query := r.db.Where("id = ?", id)

	// Load relationships
	query = query.Preload("Schedule").Preload("Repository").Preload("Policy")

	if err := query.First(&flow).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("flow not found: %s", id)
		}
		log.WithError(err).WithField("flow_id", id).Error("Failed to get flow")
		return nil, fmt.Errorf("failed to get flow: %w", err)
	}

	return &flow, nil
}

// GetFlowByName retrieves a flow by name
func (r *FlowRepository) GetFlowByName(ctx context.Context, name string) (*ProtectionFlow, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database not available")
	}

	var flow ProtectionFlow
	query := r.db.Where("name = ?", name)
	query = query.Preload("Schedule").Preload("Repository").Preload("Policy")

	if err := query.First(&flow).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("flow not found: %s", name)
		}
		log.WithError(err).WithField("flow_name", name).Error("Failed to get flow by name")
		return nil, fmt.Errorf("failed to get flow by name: %w", err)
	}

	return &flow, nil
}

// ListFlows retrieves flows with optional filtering
func (r *FlowRepository) ListFlows(ctx context.Context, filters FlowFilters) ([]*ProtectionFlow, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database not available")
	}

	var flows []*ProtectionFlow
	query := r.db.Model(&ProtectionFlow{})

	// Apply filters
	if filters.FlowType != nil {
		query = query.Where("flow_type = ?", *filters.FlowType)
	}
	if filters.TargetType != nil {
		query = query.Where("target_type = ?", *filters.TargetType)
	}
	if filters.Enabled != nil {
		query = query.Where("enabled = ?", *filters.Enabled)
	}
	if filters.ScheduleID != nil {
		query = query.Where("schedule_id = ?", *filters.ScheduleID)
	}

	// Apply pagination
	if filters.Limit > 0 {
		query = query.Limit(filters.Limit)
	}
	if filters.Offset > 0 {
		query = query.Offset(filters.Offset)
	}

	// Order by creation time (newest first)
	query = query.Order("created_at DESC")

	// Load relationships
	query = query.Preload("Schedule").Preload("Repository").Preload("Policy")

	if err := query.Find(&flows).Error; err != nil {
		log.WithError(err).Error("Failed to list flows")
		return nil, fmt.Errorf("failed to list flows: %w", err)
	}

	return flows, nil
}

// UpdateFlow updates a flow with the provided updates map
func (r *FlowRepository) UpdateFlow(ctx context.Context, id string, updates map[string]interface{}) error {
	if r.db == nil {
		return fmt.Errorf("database not available")
	}

	log.WithFields(log.Fields{
		"flow_id":  id,
		"updates":  updates,
	}).Info("Updating protection flow")

	updates["updated_at"] = time.Now()

	result := r.db.Model(&ProtectionFlow{}).Where("id = ?", id).Updates(updates)
	if result.Error != nil {
		log.WithError(result.Error).WithField("flow_id", id).Error("Failed to update flow")
		return fmt.Errorf("failed to update flow: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("flow not found: %s", id)
	}

	return nil
}

// DeleteFlow deletes a flow by ID
func (r *FlowRepository) DeleteFlow(ctx context.Context, id string) error {
	if r.db == nil {
		return fmt.Errorf("database not available")
	}

	log.WithField("flow_id", id).Info("Deleting protection flow")

	result := r.db.Where("id = ?", id).Delete(&ProtectionFlow{})
	if result.Error != nil {
		log.WithError(result.Error).WithField("flow_id", id).Error("Failed to delete flow")
		return fmt.Errorf("failed to delete flow: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("flow not found: %s", id)
	}

	log.WithField("flow_id", id).Info("Protection flow deleted successfully")
	return nil
}

// =============================================================================
// FLOW CONTROL OPERATIONS
// =============================================================================

// EnableFlow enables a flow
func (r *FlowRepository) EnableFlow(ctx context.Context, id string) error {
	return r.UpdateFlow(ctx, id, map[string]interface{}{
		"enabled": true,
	})
}

// DisableFlow disables a flow
func (r *FlowRepository) DisableFlow(ctx context.Context, id string) error {
	return r.UpdateFlow(ctx, id, map[string]interface{}{
		"enabled": false,
	})
}

// BulkEnableFlows enables multiple flows
func (r *FlowRepository) BulkEnableFlows(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}

	log.WithField("count", len(ids)).Info("Bulk enabling flows")

	result := r.db.Model(&ProtectionFlow{}).Where("id IN ?", ids).Update("enabled", true)
	if result.Error != nil {
		log.WithError(result.Error).Error("Failed to bulk enable flows")
		return fmt.Errorf("failed to bulk enable flows: %w", result.Error)
	}

	log.WithFields(log.Fields{
		"count":         len(ids),
		"rows_affected": result.RowsAffected,
	}).Info("Flows bulk enabled successfully")

	return nil
}

// BulkDisableFlows disables multiple flows
func (r *FlowRepository) BulkDisableFlows(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}

	log.WithField("count", len(ids)).Info("Bulk disabling flows")

	result := r.db.Model(&ProtectionFlow{}).Where("id IN ?", ids).Update("enabled", false)
	if result.Error != nil {
		log.WithError(result.Error).Error("Failed to bulk disable flows")
		return fmt.Errorf("failed to bulk disable flows: %w", result.Error)
	}

	log.WithFields(log.Fields{
		"count":         len(ids),
		"rows_affected": result.RowsAffected,
	}).Info("Flows bulk disabled successfully")

	return nil
}

// =============================================================================
// STATISTICS AND STATUS UPDATES
// =============================================================================

// UpdateFlowStatistics updates flow execution statistics
func (r *FlowRepository) UpdateFlowStatistics(ctx context.Context, id string, stats FlowStatistics) error {
	updates := map[string]interface{}{
		"last_execution_id":      stats.LastExecutionID,
		"last_execution_status":  stats.LastExecutionStatus,
		"last_execution_time":    stats.LastExecutionTime,
		"total_executions":       stats.TotalExecutions,
		"successful_executions":  stats.SuccessfulExecutions,
		"failed_executions":      stats.FailedExecutions,
	}

	return r.UpdateFlow(ctx, id, updates)
}

// =============================================================================
// EXECUTION TRACKING
// =============================================================================

// CreateExecution creates a new flow execution record
func (r *FlowRepository) CreateExecution(ctx context.Context, execution *ProtectionFlowExecution) error {
	if r.db == nil {
		return fmt.Errorf("database not available")
	}

	log.WithFields(log.Fields{
		"flow_id":        execution.FlowID,
		"execution_type": execution.ExecutionType,
	}).Info("Creating flow execution record")

	if err := r.db.Create(execution).Error; err != nil {
		log.WithError(err).WithField("flow_id", execution.FlowID).Error("Failed to create execution")
		return fmt.Errorf("failed to create execution: %w", err)
	}

	log.WithField("execution_id", execution.ID).Info("Flow execution record created successfully")
	return nil
}

// GetExecution retrieves an execution by ID
func (r *FlowRepository) GetExecution(ctx context.Context, id string) (*ProtectionFlowExecution, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database not available")
	}

	var execution ProtectionFlowExecution
	if err := r.db.Preload("Flow").Where("id = ?", id).First(&execution).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("execution not found: %s", id)
		}
		log.WithError(err).WithField("execution_id", id).Error("Failed to get execution")
		return nil, fmt.Errorf("failed to get execution: %w", err)
	}

	return &execution, nil
}

// ListExecutions retrieves executions for a flow with optional limit
func (r *FlowRepository) ListExecutions(ctx context.Context, flowID string, limit int) ([]*ProtectionFlowExecution, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database not available")
	}

	var executions []*ProtectionFlowExecution
	query := r.db.Where("flow_id = ?", flowID).Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	if err := query.Find(&executions).Error; err != nil {
		log.WithError(err).WithField("flow_id", flowID).Error("Failed to list executions")
		return nil, fmt.Errorf("failed to list executions: %w", err)
	}

	return executions, nil
}

// UpdateExecutionStatus updates an execution status and related fields
func (r *FlowRepository) UpdateExecutionStatus(ctx context.Context, id string, status string, updates map[string]interface{}) error {
	if r.db == nil {
		return fmt.Errorf("database not available")
	}

	log.WithFields(log.Fields{
		"execution_id": id,
		"status":       status,
	}).Info("Updating execution status")

	updates["status"] = status

	result := r.db.Model(&ProtectionFlowExecution{}).Where("id = ?", id).Updates(updates)
	if result.Error != nil {
		log.WithError(result.Error).WithField("execution_id", id).Error("Failed to update execution status")
		return fmt.Errorf("failed to update execution status: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("execution not found: %s", id)
	}

	return nil
}

// =============================================================================
// QUERY METHODS
// =============================================================================

// GetFlowsByTarget retrieves flows targeting a specific VM or group
func (r *FlowRepository) GetFlowsByTarget(ctx context.Context, targetType, targetID string) ([]*ProtectionFlow, error) {
	return r.ListFlows(ctx, FlowFilters{
		TargetType: &targetType,
		Limit:      100, // Reasonable limit
	})
}

// GetFlowsBySchedule retrieves flows using a specific schedule
func (r *FlowRepository) GetFlowsBySchedule(ctx context.Context, scheduleID string) ([]*ProtectionFlow, error) {
	return r.ListFlows(ctx, FlowFilters{
		ScheduleID: &scheduleID,
		Limit:      100,
	})
}

// GetEnabledFlows retrieves all enabled flows of a specific type
func (r *FlowRepository) GetEnabledFlows(ctx context.Context, flowType string) ([]*ProtectionFlow, error) {
	enabled := true
	return r.ListFlows(ctx, FlowFilters{
		FlowType: &flowType,
		Enabled:  &enabled,
		Limit:    1000, // Large limit for enabled flows
	})
}

// GetFlowsWithNextRun retrieves flows with next execution before the specified time
func (r *FlowRepository) GetFlowsWithNextRun(ctx context.Context, before time.Time) ([]*ProtectionFlow, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database not available")
	}

	var flows []*ProtectionFlow
	query := r.db.Where("enabled = ? AND next_execution_time IS NOT NULL AND next_execution_time <= ?", true, before)
	query = query.Order("next_execution_time ASC")
	query = query.Preload("Schedule").Preload("Repository").Preload("Policy")

	if err := query.Find(&flows).Error; err != nil {
		log.WithError(err).Error("Failed to get flows with next run")
		return nil, fmt.Errorf("failed to get flows with next run: %w", err)
	}

	return flows, nil
}

