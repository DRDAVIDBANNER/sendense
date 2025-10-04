package database

import (
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// =============================================================================
// SCHEDULER REPOSITORY - VM-Centric Operations using context_id
// =============================================================================
// ALL VM operations in this repository use vm_replication_contexts.context_id
// as the primary VM identifier, following project rules for VM-centric architecture

// SchedulerRepository handles all scheduler-related database operations
// CRITICAL: All VM references use context_id, never vm_name or other identifiers
type SchedulerRepository struct {
	db *gorm.DB
}

// NewSchedulerRepository creates a new scheduler repository
func NewSchedulerRepository(conn Connection) *SchedulerRepository {
	return &SchedulerRepository{
		db: conn.GetGormDB(),
	}
}

// =============================================================================
// SCHEDULE MANAGEMENT - CRUD operations for ReplicationSchedule
// =============================================================================

// CreateSchedule creates a new replication schedule
func (r *SchedulerRepository) CreateSchedule(schedule *ReplicationSchedule) error {
	if r.db == nil {
		return fmt.Errorf("database not available")
	}

	log.WithFields(log.Fields{
		"schedule_name": schedule.Name,
		"cron_expr":     schedule.CronExpression,
		"type":          schedule.ScheduleType,
	}).Info("Creating replication schedule")

	if err := r.db.Create(schedule).Error; err != nil {
		log.WithError(err).WithField("schedule_name", schedule.Name).Error("Failed to create schedule")
		return fmt.Errorf("failed to create schedule: %w", err)
	}

	log.WithField("schedule_id", schedule.ID).Info("Successfully created schedule")
	return nil
}

// GetScheduleByID retrieves a schedule by ID with optional preloading
func (r *SchedulerRepository) GetScheduleByID(scheduleID string, preload ...string) (*ReplicationSchedule, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database not available")
	}

	var schedule ReplicationSchedule
	query := r.db

	// Apply preloading if requested
	for _, relation := range preload {
		query = query.Preload(relation)
	}

	if err := query.Where("id = ?", scheduleID).First(&schedule).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("schedule not found: %s", scheduleID)
		}
		return nil, fmt.Errorf("failed to get schedule: %w", err)
	}

	return &schedule, nil
}

// ListSchedules retrieves all schedules with optional filtering
func (r *SchedulerRepository) ListSchedules(enabledOnly bool, preload ...string) ([]ReplicationSchedule, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database not available")
	}

	var schedules []ReplicationSchedule
	query := r.db

	// Apply preloading if requested
	for _, relation := range preload {
		query = query.Preload(relation)
	}

	// Filter by enabled status if requested
	if enabledOnly {
		query = query.Where("enabled = ?", true)
	}

	if err := query.Order("name ASC").Find(&schedules).Error; err != nil {
		return nil, fmt.Errorf("failed to list schedules: %w", err)
	}

	log.WithField("count", len(schedules)).Debug("Listed schedules")
	return schedules, nil
}

// UpdateSchedule updates an existing schedule
func (r *SchedulerRepository) UpdateSchedule(scheduleID string, updates map[string]interface{}) error {
	if r.db == nil {
		return fmt.Errorf("database not available")
	}

	log.WithFields(log.Fields{
		"schedule_id": scheduleID,
		"updates":     len(updates),
	}).Info("Updating schedule")

	result := r.db.Model(&ReplicationSchedule{}).Where("id = ?", scheduleID).Updates(updates)
	if result.Error != nil {
		return fmt.Errorf("failed to update schedule: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("schedule not found: %s", scheduleID)
	}

	return nil
}

// DeleteSchedule deletes a schedule (will cascade to related records)
func (r *SchedulerRepository) DeleteSchedule(scheduleID string) error {
	if r.db == nil {
		return fmt.Errorf("database not available")
	}

	log.WithField("schedule_id", scheduleID).Info("Deleting schedule")

	result := r.db.Delete(&ReplicationSchedule{}, "id = ?", scheduleID)
	if result.Error != nil {
		return fmt.Errorf("failed to delete schedule: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("schedule not found: %s", scheduleID)
	}

	log.WithField("schedule_id", scheduleID).Info("Successfully deleted schedule")
	return nil
}

// =============================================================================
// MACHINE GROUP MANAGEMENT - VM grouping using context_id
// =============================================================================

// CreateMachineGroup creates a new VM machine group
func (r *SchedulerRepository) CreateMachineGroup(group *VMMachineGroup) error {
	if r.db == nil {
		return fmt.Errorf("database not available")
	}

	log.WithFields(log.Fields{
		"group_name":  group.Name,
		"schedule_id": group.ScheduleID,
	}).Info("Creating machine group")

	if err := r.db.Create(group).Error; err != nil {
		log.WithError(err).WithField("group_name", group.Name).Error("Failed to create machine group")
		return fmt.Errorf("failed to create machine group: %w", err)
	}

	log.WithField("group_id", group.ID).Info("Successfully created machine group")
	return nil
}

// GetMachineGroupByID retrieves a machine group by ID with optional preloading
func (r *SchedulerRepository) GetMachineGroupByID(groupID string, preload ...string) (*VMMachineGroup, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database not available")
	}

	var group VMMachineGroup
	query := r.db

	// Apply preloading if requested
	for _, relation := range preload {
		query = query.Preload(relation)
	}

	if err := query.Where("id = ?", groupID).First(&group).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("machine group not found: %s", groupID)
		}
		return nil, fmt.Errorf("failed to get machine group: %w", err)
	}

	return &group, nil
}

// ListMachineGroups retrieves all machine groups
func (r *SchedulerRepository) ListMachineGroups(preload ...string) ([]VMMachineGroup, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database not available")
	}

	var groups []VMMachineGroup
	query := r.db

	// Apply preloading if requested
	for _, relation := range preload {
		query = query.Preload(relation)
	}

	if err := query.Order("name ASC").Find(&groups).Error; err != nil {
		return nil, fmt.Errorf("failed to list machine groups: %w", err)
	}

	log.WithField("count", len(groups)).Debug("Listed machine groups")
	return groups, nil
}

// UpdateMachineGroup updates an existing machine group
func (r *SchedulerRepository) UpdateMachineGroup(groupID string, updates map[string]interface{}) error {
	if r.db == nil {
		return fmt.Errorf("database not available")
	}

	log.WithFields(log.Fields{
		"group_id": groupID,
		"updates":  len(updates),
	}).Info("Updating machine group")

	result := r.db.Model(&VMMachineGroup{}).Where("id = ?", groupID).Updates(updates)
	if result.Error != nil {
		return fmt.Errorf("failed to update machine group: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("machine group not found: %s", groupID)
	}

	return nil
}

// DeleteMachineGroup deletes a machine group (will cascade to memberships)
func (r *SchedulerRepository) DeleteMachineGroup(groupID string) error {
	if r.db == nil {
		return fmt.Errorf("database not available")
	}

	log.WithField("group_id", groupID).Info("Deleting machine group")

	result := r.db.Delete(&VMMachineGroup{}, "id = ?", groupID)
	if result.Error != nil {
		return fmt.Errorf("failed to delete machine group: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("machine group not found: %s", groupID)
	}

	log.WithField("group_id", groupID).Info("Successfully deleted machine group")
	return nil
}

// =============================================================================
// VM GROUP MEMBERSHIP - Managing VMs in groups using context_id
// =============================================================================

// AddVMToGroup adds a VM to a machine group using context_id
func (r *SchedulerRepository) AddVMToGroup(groupID, vmContextID string, priority int, enabled bool) (*VMGroupMembership, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database not available")
	}

	log.WithFields(log.Fields{
		"group_id":      groupID,
		"vm_context_id": vmContextID,
		"priority":      priority,
		"enabled":       enabled,
	}).Info("Adding VM to machine group")

	// Verify VM context exists
	var vmContext VMReplicationContext
	if err := r.db.Where("context_id = ?", vmContextID).First(&vmContext).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("VM context not found: %s", vmContextID)
		}
		return nil, fmt.Errorf("failed to verify VM context: %w", err)
	}

	// Verify group exists
	var group VMMachineGroup
	if err := r.db.Where("id = ?", groupID).First(&group).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("machine group not found: %s", groupID)
		}
		return nil, fmt.Errorf("failed to verify machine group: %w", err)
	}

	// Create membership
	membership := &VMGroupMembership{
		GroupID:     groupID,
		VMContextID: vmContextID,
		Priority:    priority,
		Enabled:     enabled,
	}

	if err := r.db.Create(membership).Error; err != nil {
		log.WithError(err).WithFields(log.Fields{
			"group_id":      groupID,
			"vm_context_id": vmContextID,
		}).Error("Failed to add VM to group")
		return nil, fmt.Errorf("failed to add VM to group: %w", err)
	}

	log.WithFields(log.Fields{
		"membership_id": membership.ID,
		"vm_name":       vmContext.VMName,
		"group_name":    group.Name,
	}).Info("Successfully added VM to group")

	return membership, nil
}

// RemoveVMFromGroup removes a VM from a machine group using context_id
func (r *SchedulerRepository) RemoveVMFromGroup(groupID, vmContextID string) error {
	if r.db == nil {
		return fmt.Errorf("database not available")
	}

	log.WithFields(log.Fields{
		"group_id":      groupID,
		"vm_context_id": vmContextID,
	}).Info("Removing VM from machine group")

	result := r.db.Where("group_id = ? AND vm_context_id = ?", groupID, vmContextID).Delete(&VMGroupMembership{})
	if result.Error != nil {
		return fmt.Errorf("failed to remove VM from group: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("VM membership not found in group")
	}

	log.WithFields(log.Fields{
		"group_id":      groupID,
		"vm_context_id": vmContextID,
	}).Info("Successfully removed VM from group")

	return nil
}

// GetGroupMemberships retrieves all VMs in a group with their contexts
func (r *SchedulerRepository) GetGroupMemberships(groupID string, enabledOnly bool) ([]VMGroupMembership, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database not available")
	}

	var memberships []VMGroupMembership
	query := r.db.Preload("VMContext").Where("group_id = ?", groupID)

	if enabledOnly {
		query = query.Where("enabled = ?", true)
	}

	if err := query.Order("priority ASC, added_at ASC").Find(&memberships).Error; err != nil {
		return nil, fmt.Errorf("failed to get group memberships: %w", err)
	}

	log.WithFields(log.Fields{
		"group_id": groupID,
		"count":    len(memberships),
	}).Debug("Retrieved group memberships")

	return memberships, nil
}

// GetVMGroupMemberships retrieves all groups a VM belongs to using context_id
func (r *SchedulerRepository) GetVMGroupMemberships(vmContextID string) ([]VMGroupMembership, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database not available")
	}

	var memberships []VMGroupMembership
	if err := r.db.Preload("Group").Preload("Group.Schedule").
		Where("vm_context_id = ?", vmContextID).
		Find(&memberships).Error; err != nil {
		return nil, fmt.Errorf("failed to get VM group memberships: %w", err)
	}

	log.WithFields(log.Fields{
		"vm_context_id": vmContextID,
		"count":         len(memberships),
	}).Debug("Retrieved VM group memberships")

	return memberships, nil
}

// UpdateVMGroupMembership updates VM membership settings
func (r *SchedulerRepository) UpdateVMGroupMembership(groupID, vmContextID string, updates map[string]interface{}) error {
	if r.db == nil {
		return fmt.Errorf("database not available")
	}

	log.WithFields(log.Fields{
		"group_id":      groupID,
		"vm_context_id": vmContextID,
		"updates":       len(updates),
	}).Info("Updating VM group membership")

	result := r.db.Model(&VMGroupMembership{}).
		Where("group_id = ? AND vm_context_id = ?", groupID, vmContextID).
		Updates(updates)

	if result.Error != nil {
		return fmt.Errorf("failed to update VM group membership: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("VM membership not found in group")
	}

	return nil
}

// =============================================================================
// SCHEDULE EXECUTION TRACKING - Track scheduler runs
// =============================================================================

// CreateScheduleExecution creates a new schedule execution record
func (r *SchedulerRepository) CreateScheduleExecution(execution *ScheduleExecution) error {
	if r.db == nil {
		return fmt.Errorf("database not available")
	}

	log.WithFields(log.Fields{
		"schedule_id":  execution.ScheduleID,
		"group_id":     execution.GroupID,
		"scheduled_at": execution.ScheduledAt,
	}).Info("Creating schedule execution")

	if err := r.db.Create(execution).Error; err != nil {
		log.WithError(err).WithField("schedule_id", execution.ScheduleID).Error("Failed to create schedule execution")
		return fmt.Errorf("failed to create schedule execution: %w", err)
	}

	log.WithField("execution_id", execution.ID).Info("Successfully created schedule execution")
	return nil
}

// UpdateScheduleExecution updates execution status and metrics
func (r *SchedulerRepository) UpdateScheduleExecution(executionID string, updates map[string]interface{}) error {
	if r.db == nil {
		return fmt.Errorf("database not available")
	}

	log.WithFields(log.Fields{
		"execution_id": executionID,
		"updates":      len(updates),
	}).Info("Updating schedule execution")

	result := r.db.Model(&ScheduleExecution{}).Where("id = ?", executionID).Updates(updates)
	if result.Error != nil {
		return fmt.Errorf("failed to update schedule execution: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("schedule execution not found: %s", executionID)
	}

	return nil
}

// GetScheduleExecution retrieves a schedule execution by ID
func (r *SchedulerRepository) GetScheduleExecution(executionID string, preload ...string) (*ScheduleExecution, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database not available")
	}

	var execution ScheduleExecution
	query := r.db

	// Apply preloading if requested
	for _, relation := range preload {
		query = query.Preload(relation)
	}

	if err := query.Where("id = ?", executionID).First(&execution).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("schedule execution not found: %s", executionID)
		}
		return nil, fmt.Errorf("failed to get schedule execution: %w", err)
	}

	return &execution, nil
}

// GetScheduleExecutions retrieves executions for a schedule with pagination
func (r *SchedulerRepository) GetScheduleExecutions(scheduleID string, limit, offset int) ([]ScheduleExecution, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database not available")
	}

	var executions []ScheduleExecution
	query := r.db.Where("schedule_id = ?", scheduleID).
		Order("scheduled_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	if err := query.Find(&executions).Error; err != nil {
		return nil, fmt.Errorf("failed to get schedule executions: %w", err)
	}

	log.WithFields(log.Fields{
		"schedule_id": scheduleID,
		"count":       len(executions),
	}).Debug("Retrieved schedule executions")

	return executions, nil
}

// =============================================================================
// VM CONTEXT ENHANCEMENT - Add discovery without jobs
// =============================================================================

// AddVMFromDiscovery creates a VM context from discovery without creating a replication job
// CRITICAL: This is the key method for adding VMs to OMA without immediate replication
func (r *SchedulerRepository) AddVMFromDiscovery(vmName, vmwareVMID, vmPath, vcenterHost, datacenter string) (*VMReplicationContext, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database not available")
	}

	log.WithFields(log.Fields{
		"vm_name":      vmName,
		"vmware_vm_id": vmwareVMID,
		"vm_path":      vmPath,
		"vcenter_host": vcenterHost,
		"datacenter":   datacenter,
	}).Info("Adding VM from discovery without replication job")

	// Check if VM context already exists
	var existingContext VMReplicationContext
	err := r.db.Where("vm_name = ? AND vcenter_host = ?", vmName, vcenterHost).First(&existingContext).Error
	if err == nil {
		log.WithField("context_id", existingContext.ContextID).Info("VM context already exists")
		return &existingContext, nil
	}
	if err != gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("failed to check existing VM context: %w", err)
	}

	// Create new VM context
	vmContext := &VMReplicationContext{
		VMName:           vmName,
		VMwareVMID:       vmwareVMID,
		VMPath:           vmPath,
		VCenterHost:      vcenterHost,
		Datacenter:       datacenter,
		CurrentStatus:    "discovered",
		AutoAdded:        true, // Mark as added via discovery
		SchedulerEnabled: true, // Enable for scheduling by default
	}

	if err := r.db.Create(vmContext).Error; err != nil {
		log.WithError(err).WithField("vm_name", vmName).Error("Failed to create VM context from discovery")
		return nil, fmt.Errorf("failed to create VM context: %w", err)
	}

	log.WithFields(log.Fields{
		"context_id": vmContext.ContextID,
		"vm_name":    vmName,
		"auto_added": true,
	}).Info("Successfully added VM from discovery")

	return vmContext, nil
}

// GetUngroupedVMs retrieves VMs that are not in any machine group
func (r *SchedulerRepository) GetUngroupedVMs() ([]VMReplicationContext, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database not available")
	}

	var ungroupedVMs []VMReplicationContext

	// Use LEFT JOIN to find VMs not in any group
	if err := r.db.
		Where("context_id NOT IN (SELECT DISTINCT vm_context_id FROM vm_group_memberships)").
		Order("vm_name ASC").
		Find(&ungroupedVMs).Error; err != nil {
		return nil, fmt.Errorf("failed to get ungrouped VMs: %w", err)
	}

	log.WithField("count", len(ungroupedVMs)).Debug("Retrieved ungrouped VMs")
	return ungroupedVMs, nil
}

// =============================================================================
// SCHEDULER MONITORING QUERIES - Analytics and status
// =============================================================================

// GetActiveSchedules retrieves enabled schedules with next execution info
func (r *SchedulerRepository) GetActiveSchedules() ([]ReplicationSchedule, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database not available")
	}

	var schedules []ReplicationSchedule
	if err := r.db.
		Preload("Groups").
		Preload("Groups.Memberships", "enabled = ?", true).
		Where("enabled = ?", true).
		Order("name ASC").
		Find(&schedules).Error; err != nil {
		return nil, fmt.Errorf("failed to get active schedules: %w", err)
	}

	log.WithField("count", len(schedules)).Debug("Retrieved active schedules")
	return schedules, nil
}

// GetScheduleStats retrieves execution statistics for a schedule
func (r *SchedulerRepository) GetScheduleStats(scheduleID string, days int) (map[string]interface{}, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database not available")
	}

	// Get execution statistics for the last N days
	var stats struct {
		TotalExecutions      int     `json:"total_executions"`
		SuccessfulExecutions int     `json:"successful_executions"`
		FailedExecutions     int     `json:"failed_executions"`
		SkippedExecutions    int     `json:"skipped_executions"`
		AvgExecutionTime     float64 `json:"avg_execution_time_seconds"`
		TotalJobsCreated     int     `json:"total_jobs_created"`
		TotalJobsCompleted   int     `json:"total_jobs_completed"`
		TotalJobsFailed      int     `json:"total_jobs_failed"`
	}

	cutoffDate := time.Now().AddDate(0, 0, -days)

	if err := r.db.Model(&ScheduleExecution{}).
		Select(`
			COUNT(*) as total_executions,
			SUM(CASE WHEN status = 'completed' THEN 1 ELSE 0 END) as successful_executions,
			SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END) as failed_executions,
			SUM(CASE WHEN status = 'skipped' THEN 1 ELSE 0 END) as skipped_executions,
			AVG(execution_duration_seconds) as avg_execution_time,
			SUM(jobs_created) as total_jobs_created,
			SUM(jobs_completed) as total_jobs_completed,
			SUM(jobs_failed) as total_jobs_failed
		`).
		Where("schedule_id = ? AND scheduled_at >= ?", scheduleID, cutoffDate).
		Scan(&stats).Error; err != nil {
		return nil, fmt.Errorf("failed to get schedule stats: %w", err)
	}

	result := map[string]interface{}{
		"total_executions":      stats.TotalExecutions,
		"successful_executions": stats.SuccessfulExecutions,
		"failed_executions":     stats.FailedExecutions,
		"skipped_executions":    stats.SkippedExecutions,
		"avg_execution_time":    stats.AvgExecutionTime,
		"total_jobs_created":    stats.TotalJobsCreated,
		"total_jobs_completed":  stats.TotalJobsCompleted,
		"total_jobs_failed":     stats.TotalJobsFailed,
		"success_rate":          0.0,
		"days":                  days,
	}

	// Calculate success rate
	if stats.TotalExecutions > 0 {
		result["success_rate"] = float64(stats.SuccessfulExecutions) / float64(stats.TotalExecutions) * 100
	}

	log.WithFields(log.Fields{
		"schedule_id": scheduleID,
		"days":        days,
		"executions":  stats.TotalExecutions,
	}).Debug("Retrieved schedule statistics")

	return result, nil
}

// GetGormDB exposes the underlying GORM database connection
// Used for complex queries that need direct database access
func (r *SchedulerRepository) GetGormDB() *gorm.DB {
	return r.db
}

// =============================================================================
// ADDITIONAL METHODS FOR JOB CONFLICT DETECTION
// =============================================================================

// GetVMContextByID retrieves a VM replication context by context_id
func (r *SchedulerRepository) GetVMContextByID(contextID string) (*VMReplicationContext, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database not available")
	}

	var vmCtx VMReplicationContext
	err := r.db.Where("context_id = ?", contextID).First(&vmCtx).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get VM context by ID: %w", err)
	}

	log.WithField("context_id", contextID).Debug("Retrieved VM context by ID")
	return &vmCtx, nil
}

// GetGroupByID retrieves a machine group by ID with optional preloading
func (r *SchedulerRepository) GetGroupByID(groupID string, preload ...string) (*VMMachineGroup, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database not available")
	}

	query := r.db
	for _, p := range preload {
		query = query.Preload(p)
	}

	var group VMMachineGroup
	err := query.Where("id = ?", groupID).First(&group).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get group by ID: %w", err)
	}

	log.WithFields(log.Fields{
		"group_id": groupID,
		"preload":  preload,
	}).Debug("Retrieved group by ID")

	return &group, nil
}

// =============================================================================
// MACHINE GROUP MANAGEMENT METHODS
// =============================================================================

// CreateGroup creates a new machine group
func (r *SchedulerRepository) CreateGroup(group *VMMachineGroup) error {
	if r.db == nil {
		return fmt.Errorf("database not available")
	}

	err := r.db.Create(group).Error
	if err != nil {
		return fmt.Errorf("failed to create group: %w", err)
	}

	log.WithFields(log.Fields{
		"group_id": group.ID,
		"name":     group.Name,
	}).Info("Created machine group")

	return nil
}

// UpdateGroup updates an existing machine group
func (r *SchedulerRepository) UpdateGroup(groupID string, updates map[string]interface{}) error {
	if r.db == nil {
		return fmt.Errorf("database not available")
	}

	err := r.db.Model(&VMMachineGroup{}).Where("id = ?", groupID).Updates(updates).Error
	if err != nil {
		return fmt.Errorf("failed to update group: %w", err)
	}

	log.WithFields(log.Fields{
		"group_id": groupID,
		"fields":   len(updates),
	}).Debug("Updated machine group")

	return nil
}

// DeleteGroup deletes a machine group
func (r *SchedulerRepository) DeleteGroup(groupID string) error {
	if r.db == nil {
		return fmt.Errorf("database not available")
	}

	err := r.db.Delete(&VMMachineGroup{}, "id = ?", groupID).Error
	if err != nil {
		return fmt.Errorf("failed to delete group: %w", err)
	}

	log.WithField("group_id", groupID).Info("Deleted machine group")
	return nil
}

// ListGroups lists machine groups with optional schedule filtering
func (r *SchedulerRepository) ListGroups(scheduleID *string) ([]VMMachineGroup, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database not available")
	}

	query := r.db.Preload("Schedule")
	if scheduleID != nil {
		query = query.Where("schedule_id = ?", *scheduleID)
	}

	var groups []VMMachineGroup
	err := query.Order("name ASC").Find(&groups).Error
	if err != nil {
		return nil, fmt.Errorf("failed to list groups: %w", err)
	}

	log.WithFields(log.Fields{
		"count":       len(groups),
		"schedule_id": scheduleID,
	}).Debug("Listed machine groups")

	return groups, nil
}

// CreateMembership creates a new VM group membership
func (r *SchedulerRepository) CreateMembership(membership *VMGroupMembership) error {
	if r.db == nil {
		return fmt.Errorf("database not available")
	}

	err := r.db.Create(membership).Error
	if err != nil {
		return fmt.Errorf("failed to create membership: %w", err)
	}

	log.WithFields(log.Fields{
		"membership_id": membership.ID,
		"group_id":      membership.GroupID,
		"vm_context_id": membership.VMContextID,
	}).Info("Created VM group membership")

	return nil
}

// UpdateMembership updates VM group membership
func (r *SchedulerRepository) UpdateMembership(groupID, vmContextID string, updates map[string]interface{}) error {
	if r.db == nil {
		return fmt.Errorf("database not available")
	}

	err := r.db.Model(&VMGroupMembership{}).
		Where("group_id = ? AND vm_context_id = ?", groupID, vmContextID).
		Updates(updates).Error
	if err != nil {
		return fmt.Errorf("failed to update membership: %w", err)
	}

	log.WithFields(log.Fields{
		"group_id":      groupID,
		"vm_context_id": vmContextID,
		"fields":        len(updates),
	}).Debug("Updated VM group membership")

	return nil
}

// DeleteMembership deletes VM group membership
func (r *SchedulerRepository) DeleteMembership(groupID, vmContextID string) error {
	if r.db == nil {
		return fmt.Errorf("database not available")
	}

	err := r.db.Where("group_id = ? AND vm_context_id = ?", groupID, vmContextID).
		Delete(&VMGroupMembership{}).Error
	if err != nil {
		return fmt.Errorf("failed to delete membership: %w", err)
	}

	log.WithFields(log.Fields{
		"group_id":      groupID,
		"vm_context_id": vmContextID,
	}).Info("Deleted VM group membership")

	return nil
}

// GetGroupExecutions retrieves recent executions for a group
func (r *SchedulerRepository) GetGroupExecutions(groupID string, limit int) ([]ScheduleExecution, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database not available")
	}

	var executions []ScheduleExecution
	query := r.db.Where("group_id = ?", groupID).
		Order("scheduled_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	err := query.Find(&executions).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get group executions: %w", err)
	}

	log.WithFields(log.Fields{
		"group_id": groupID,
		"count":    len(executions),
		"limit":    limit,
	}).Debug("Retrieved group executions")

	return executions, nil
}

// GetVMContextsWithoutGroups returns VM contexts that are not assigned to any machine group
func (r *SchedulerRepository) GetVMContextsWithoutGroups() ([]VMReplicationContext, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database not available")
	}

	var contexts []VMReplicationContext

	// Find VM contexts that don't have any group memberships
	err := r.db.Where(`context_id NOT IN (
		SELECT DISTINCT vm_context_id 
		FROM vm_group_memberships 
		WHERE vm_context_id IS NOT NULL
	)`).Find(&contexts).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get VM contexts without groups: %w", err)
	}

	log.WithField("count", len(contexts)).Debug("Retrieved VM contexts without groups")
	return contexts, nil
}
