// Package handlers provides VM group assignment API endpoints for scheduler system
package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"

	"github.com/vexxhost/migratekit-oma/database"
	"github.com/vexxhost/migratekit-oma/joblog"
	"github.com/vexxhost/migratekit-oma/services"
)

// VMGroupAssignmentHandler handles VM group assignment API endpoints
type VMGroupAssignmentHandler struct {
	machineGroupService *services.MachineGroupService
	schedulerRepo       *database.SchedulerRepository
	vmContextRepo       *database.VMReplicationContextRepository
	tracker             *joblog.Tracker
}

// NewVMGroupAssignmentHandler creates a new VM group assignment handler
func NewVMGroupAssignmentHandler(machineGroupService *services.MachineGroupService,
	schedulerRepo *database.SchedulerRepository,
	vmContextRepo *database.VMReplicationContextRepository,
	tracker *joblog.Tracker) *VMGroupAssignmentHandler {
	return &VMGroupAssignmentHandler{
		machineGroupService: machineGroupService,
		schedulerRepo:       schedulerRepo,
		vmContextRepo:       vmContextRepo,
		tracker:             tracker,
	}
}

// AssignVMRequest represents a request to assign a single VM to a group
type AssignVMRequest struct {
	VMContextID string `json:"vm_context_id" binding:"required"`
	Priority    int    `json:"priority" binding:"min=0"`
	Enabled     bool   `json:"enabled"`
}

// BulkAssignRequest represents a request to assign multiple VMs to a group
type BulkAssignRequest struct {
	VMContextIDs []string `json:"vm_context_ids" binding:"required,min=1"`
	Priority     int      `json:"priority" binding:"min=0"`
	Enabled      bool     `json:"enabled"`
}

// UpdateMembershipRequest represents a request to update VM membership
type UpdateMembershipRequest struct {
	Priority *int  `json:"priority,omitempty" binding:"omitempty,min=0"`
	Enabled  *bool `json:"enabled,omitempty"`
}

// CrossGroupMoveRequest represents a request to move VMs between groups
type CrossGroupMoveRequest struct {
	SourceGroupID    string   `json:"source_group_id" binding:"required"`
	TargetGroupID    string   `json:"target_group_id" binding:"required"`
	VMContextIDs     []string `json:"vm_context_ids" binding:"required,min=1"`
	Priority         int      `json:"priority" binding:"min=0"`
	Enabled          bool     `json:"enabled"`
	ValidateCapacity bool     `json:"validate_capacity"`
}

// AssignmentResponse represents the response from VM assignment operations
type AssignmentResponse struct {
	Success        bool                          `json:"success"`
	Message        string                        `json:"message"`
	AssignedVM     *VMAssignmentInfo             `json:"assigned_vm,omitempty"`
	BulkResult     *services.BulkOperationResult `json:"bulk_result,omitempty"`
	GroupCapacity  *GroupCapacityInfo            `json:"group_capacity,omitempty"`
	ProcessingTime time.Duration                 `json:"processing_time"`
}

// VMAssignmentInfo represents information about an assigned VM
type VMAssignmentInfo struct {
	VMContextID string    `json:"vm_context_id"`
	VMName      string    `json:"vm_name"`
	GroupID     string    `json:"group_id"`
	GroupName   string    `json:"group_name"`
	Priority    int       `json:"priority"`
	Enabled     bool      `json:"enabled"`
	AssignedAt  time.Time `json:"assigned_at"`
}

// GroupCapacityInfo represents group capacity validation information
type GroupCapacityInfo struct {
	GroupID          string `json:"group_id"`
	GroupName        string `json:"group_name"`
	CurrentMembers   int    `json:"current_members"`
	MaxConcurrentVMs int    `json:"max_concurrent_vms"`
	Available        int    `json:"available"`
	AtCapacity       bool   `json:"at_capacity"`
}

// VMMembershipResponse represents a VM membership in a group
type VMMembershipResponse struct {
	VMContextID string    `json:"vm_context_id"`
	VMName      string    `json:"vm_name"`
	VMwareVMID  string    `json:"vmware_vm_id"`
	Priority    int       `json:"priority"`
	Enabled     bool      `json:"enabled"`
	AddedAt     time.Time `json:"added_at"`
	AddedBy     string    `json:"added_by"`
}

// GroupVMListResponse represents a list of VMs in a group
type GroupVMListResponse struct {
	GroupID     string                 `json:"group_id"`
	GroupName   string                 `json:"group_name"`
	TotalVMs    int                    `json:"total_vms"`
	VMs         []VMMembershipResponse `json:"vms"`
	RetrievedAt time.Time              `json:"retrieved_at"`
}

// AssignToGroupRequest represents a request to assign a VM to a group via VM context
type AssignToGroupRequest struct {
	GroupID  string `json:"group_id" binding:"required"`
	Priority int    `json:"priority" binding:"min=0,max=100"`
	Enabled  bool   `json:"enabled"`
	AddedBy  string `json:"added_by,omitempty"`
}

// VMAssignmentResponse represents the response from assigning a VM to a group
type VMAssignmentResponse struct {
	Success     bool      `json:"success"`
	Message     string    `json:"message"`
	VMContextID string    `json:"vm_context_id"`
	GroupID     string    `json:"group_id"`
	Priority    int       `json:"priority"`
	Enabled     bool      `json:"enabled"`
	AddedAt     time.Time `json:"added_at"`
	AddedBy     string    `json:"added_by"`
}

// AssignVMToGroup assigns a single VM to a machine group
// POST /api/v1/groups/{group_id}/vms
func (h *VMGroupAssignmentHandler) AssignVMToGroup(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	groupID := vars["id"] // Changed from "group_id" to "id" to match route

	if groupID == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "Group ID is required")
		return
	}

	var request AssignVMRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	log.WithFields(log.Fields{
		"group_id":      groupID,
		"vm_context_id": request.VMContextID,
		"priority":      request.Priority,
		"enabled":       request.Enabled,
	}).Info("Assigning VM to group")

	ctx := r.Context()
	start := time.Now()

	// Validate group exists and get capacity info
	group, err := h.machineGroupService.GetGroup(ctx, groupID)
	if err != nil {
		log.WithError(err).Error("Failed to get group for VM assignment")
		h.writeErrorResponse(w, http.StatusNotFound, "Group not found: "+err.Error())
		return
	}

	// Get VM context to validate it exists (using context_id)
	vmContext, err := h.schedulerRepo.GetVMContextByID(request.VMContextID)
	if err != nil {
		log.WithError(err).Error("Failed to get VM context for assignment")
		h.writeErrorResponse(w, http.StatusNotFound, "VM context not found: "+err.Error())
		return
	}

	// Check group capacity
	capacityInfo := &GroupCapacityInfo{
		GroupID:          group.Group.ID,
		GroupName:        group.Group.Name,
		CurrentMembers:   group.TotalVMs,
		MaxConcurrentVMs: group.Group.MaxConcurrentVMs,
		Available:        group.Group.MaxConcurrentVMs - group.TotalVMs,
		AtCapacity:       group.TotalVMs >= group.Group.MaxConcurrentVMs,
	}

	if capacityInfo.AtCapacity {
		response := AssignmentResponse{
			Success:        false,
			Message:        "Group is at capacity",
			GroupCapacity:  capacityInfo,
			ProcessingTime: time.Since(start),
		}
		h.writeJSONResponse(w, http.StatusConflict, response)
		return
	}

	// Assign VM to group
	membershipReq := &services.VMMembershipRequest{
		VMContextID: request.VMContextID,
		Priority:    request.Priority,
		Enabled:     request.Enabled,
	}

	membership, err := h.machineGroupService.AddVMToGroup(ctx, groupID, membershipReq)
	if err != nil {
		log.WithError(err).Error("Failed to assign VM to group")
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to assign VM to group: "+err.Error())
		return
	}

	// Update capacity info after assignment
	capacityInfo.CurrentMembers++
	capacityInfo.Available--
	capacityInfo.AtCapacity = capacityInfo.CurrentMembers >= capacityInfo.MaxConcurrentVMs

	assignmentInfo := &VMAssignmentInfo{
		VMContextID: request.VMContextID,
		VMName:      vmContext.VMName,
		GroupID:     groupID,
		GroupName:   group.Group.Name,
		Priority:    membership.Priority,
		Enabled:     membership.Enabled,
		AssignedAt:  membership.AddedAt,
	}

	response := AssignmentResponse{
		Success:        true,
		Message:        "VM successfully assigned to group",
		AssignedVM:     assignmentInfo,
		GroupCapacity:  capacityInfo,
		ProcessingTime: time.Since(start),
	}

	log.WithFields(log.Fields{
		"group_id":        groupID,
		"vm_context_id":   request.VMContextID,
		"vm_name":         vmContext.VMName,
		"processing_time": response.ProcessingTime,
		"new_capacity":    capacityInfo.Available,
	}).Info("VM assignment completed successfully")

	h.writeJSONResponse(w, http.StatusCreated, response)
}

// BulkAssignVMsToGroup assigns multiple VMs to a machine group
// POST /api/v1/groups/{group_id}/vms/bulk
func (h *VMGroupAssignmentHandler) BulkAssignVMsToGroup(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	groupID := vars["group_id"]

	if groupID == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "Group ID is required")
		return
	}

	var request BulkAssignRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	if len(request.VMContextIDs) == 0 {
		h.writeErrorResponse(w, http.StatusBadRequest, "At least one VM context ID is required")
		return
	}

	log.WithFields(log.Fields{
		"group_id": groupID,
		"vm_count": len(request.VMContextIDs),
		"priority": request.Priority,
		"enabled":  request.Enabled,
	}).Info("Bulk assigning VMs to group")

	ctx := r.Context()
	start := time.Now()

	// Validate group exists and get capacity info
	group, err := h.machineGroupService.GetGroup(ctx, groupID)
	if err != nil {
		log.WithError(err).Error("Failed to get group for bulk VM assignment")
		h.writeErrorResponse(w, http.StatusNotFound, "Group not found: "+err.Error())
		return
	}

	// Check group capacity
	availableSlots := group.Group.MaxConcurrentVMs - group.TotalVMs
	if len(request.VMContextIDs) > availableSlots {
		capacityInfo := &GroupCapacityInfo{
			GroupID:          group.Group.ID,
			GroupName:        group.Group.Name,
			CurrentMembers:   group.TotalVMs,
			MaxConcurrentVMs: group.Group.MaxConcurrentVMs,
			Available:        availableSlots,
			AtCapacity:       availableSlots <= 0,
		}

		response := AssignmentResponse{
			Success:        false,
			Message:        fmt.Sprintf("Not enough capacity: %d VMs requested, %d slots available", len(request.VMContextIDs), availableSlots),
			GroupCapacity:  capacityInfo,
			ProcessingTime: time.Since(start),
		}
		h.writeJSONResponse(w, http.StatusConflict, response)
		return
	}

	// Perform bulk assignment
	bulkReq := &services.BulkMembershipRequest{
		VMContextIDs: request.VMContextIDs,
		Priority:     request.Priority,
		Enabled:      request.Enabled,
	}

	bulkResult, err := h.machineGroupService.BulkAddVMs(ctx, groupID, bulkReq)
	if err != nil {
		log.WithError(err).Error("Failed to bulk assign VMs to group")
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to bulk assign VMs: "+err.Error())
		return
	}

	// Update capacity info after assignment
	capacityInfo := &GroupCapacityInfo{
		GroupID:          group.Group.ID,
		GroupName:        group.Group.Name,
		CurrentMembers:   group.TotalVMs + bulkResult.Successful,
		MaxConcurrentVMs: group.Group.MaxConcurrentVMs,
		Available:        availableSlots - bulkResult.Successful,
		AtCapacity:       (availableSlots - bulkResult.Successful) <= 0,
	}

	response := AssignmentResponse{
		Success:        bulkResult.Successful > 0,
		Message:        fmt.Sprintf("Bulk assignment completed: %d successful, %d failed", bulkResult.Successful, bulkResult.Failed),
		BulkResult:     bulkResult,
		GroupCapacity:  capacityInfo,
		ProcessingTime: time.Since(start),
	}

	log.WithFields(log.Fields{
		"group_id":        groupID,
		"requested":       len(request.VMContextIDs),
		"successful":      bulkResult.Successful,
		"failed":          bulkResult.Failed,
		"processing_time": response.ProcessingTime,
	}).Info("Bulk VM assignment completed")

	h.writeJSONResponse(w, http.StatusOK, response)
}

// RemoveVMFromGroup removes a VM from a machine group
// DELETE /api/v1/groups/{group_id}/vms/{vm_context_id}
func (h *VMGroupAssignmentHandler) RemoveVMFromGroup(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	groupID := vars["id"]       // Changed from "group_id" to "id" to match route
	vmContextID := vars["vmId"] // Changed from "vm_context_id" to "vmId" to match route

	if groupID == "" || vmContextID == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "Group ID and VM context ID are required")
		return
	}

	log.WithFields(log.Fields{
		"group_id":      groupID,
		"vm_context_id": vmContextID,
	}).Info("Removing VM from group")

	ctx := r.Context()
	start := time.Now()

	// Remove VM from group
	err := h.machineGroupService.RemoveVMFromGroup(ctx, groupID, vmContextID)
	if err != nil {
		log.WithError(err).Error("Failed to remove VM from group")
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to remove VM from group: "+err.Error())
		return
	}

	response := AssignmentResponse{
		Success:        true,
		Message:        "VM successfully removed from group",
		ProcessingTime: time.Since(start),
	}

	log.WithFields(log.Fields{
		"group_id":        groupID,
		"vm_context_id":   vmContextID,
		"processing_time": response.ProcessingTime,
	}).Info("VM removal completed successfully")

	h.writeJSONResponse(w, http.StatusOK, response)
}

// UpdateVMMembership updates VM membership settings within a group
// PUT /api/v1/groups/{group_id}/vms/{vm_context_id}
func (h *VMGroupAssignmentHandler) UpdateVMMembership(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	groupID := vars["group_id"]
	vmContextID := vars["vm_context_id"]

	if groupID == "" || vmContextID == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "Group ID and VM context ID are required")
		return
	}

	var request UpdateMembershipRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	log.WithFields(log.Fields{
		"group_id":      groupID,
		"vm_context_id": vmContextID,
		"priority":      request.Priority,
		"enabled":       request.Enabled,
	}).Info("Updating VM membership")

	ctx := r.Context()
	start := time.Now()

	// Update VM membership
	err := h.machineGroupService.UpdateVMMembership(ctx, groupID, vmContextID, request.Priority, request.Enabled)
	if err != nil {
		log.WithError(err).Error("Failed to update VM membership")
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to update VM membership: "+err.Error())
		return
	}

	response := AssignmentResponse{
		Success:        true,
		Message:        "VM membership updated successfully",
		ProcessingTime: time.Since(start),
	}

	log.WithFields(log.Fields{
		"group_id":        groupID,
		"vm_context_id":   vmContextID,
		"processing_time": response.ProcessingTime,
	}).Info("VM membership update completed successfully")

	h.writeJSONResponse(w, http.StatusOK, response)
}

// MoveVMsBetweenGroups moves VMs from one group to another
// POST /api/v1/groups/move-vms
func (h *VMGroupAssignmentHandler) MoveVMsBetweenGroups(w http.ResponseWriter, r *http.Request) {
	var request CrossGroupMoveRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	if len(request.VMContextIDs) == 0 {
		h.writeErrorResponse(w, http.StatusBadRequest, "At least one VM context ID is required")
		return
	}

	log.WithFields(log.Fields{
		"source_group_id":   request.SourceGroupID,
		"target_group_id":   request.TargetGroupID,
		"vm_count":          len(request.VMContextIDs),
		"validate_capacity": request.ValidateCapacity,
	}).Info("Moving VMs between groups")

	ctx := r.Context()
	start := time.Now()

	// Validate capacity if requested
	if request.ValidateCapacity {
		targetGroup, err := h.machineGroupService.GetGroup(ctx, request.TargetGroupID)
		if err != nil {
			log.WithError(err).Error("Failed to get target group for capacity validation")
			h.writeErrorResponse(w, http.StatusNotFound, "Target group not found: "+err.Error())
			return
		}

		availableSlots := targetGroup.Group.MaxConcurrentVMs - targetGroup.TotalVMs
		if len(request.VMContextIDs) > availableSlots {
			capacityInfo := &GroupCapacityInfo{
				GroupID:          targetGroup.Group.ID,
				GroupName:        targetGroup.Group.Name,
				CurrentMembers:   targetGroup.TotalVMs,
				MaxConcurrentVMs: targetGroup.Group.MaxConcurrentVMs,
				Available:        availableSlots,
				AtCapacity:       availableSlots <= 0,
			}

			response := AssignmentResponse{
				Success:        false,
				Message:        fmt.Sprintf("Target group capacity exceeded: %d VMs requested, %d slots available", len(request.VMContextIDs), availableSlots),
				GroupCapacity:  capacityInfo,
				ProcessingTime: time.Since(start),
			}
			h.writeJSONResponse(w, http.StatusConflict, response)
			return
		}
	}

	// Remove VMs from source group
	removeResult, err := h.machineGroupService.BulkRemoveVMs(ctx, request.SourceGroupID, request.VMContextIDs)
	if err != nil {
		log.WithError(err).Error("Failed to remove VMs from source group")
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to remove VMs from source group: "+err.Error())
		return
	}

	// Add VMs to target group (only the ones successfully removed)
	if removeResult.Successful > 0 {
		// Get successfully removed VM context IDs
		successfullyRemovedIDs := make([]string, 0, removeResult.Successful)
		for _, vmID := range request.VMContextIDs {
			// In a real implementation, you'd track which ones were successfully removed
			// For now, assume all requested VMs were successfully removed if Successful > 0
			successfullyRemovedIDs = append(successfullyRemovedIDs, vmID)
			if len(successfullyRemovedIDs) >= removeResult.Successful {
				break
			}
		}

		addReq := &services.BulkMembershipRequest{
			VMContextIDs: successfullyRemovedIDs,
			Priority:     request.Priority,
			Enabled:      request.Enabled,
		}

		addResult, err := h.machineGroupService.BulkAddVMs(ctx, request.TargetGroupID, addReq)
		if err != nil {
			log.WithError(err).Error("Failed to add VMs to target group")
			// VMs are now orphaned - should rollback or handle this scenario
			h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to add VMs to target group: "+err.Error())
			return
		}

		combinedResult := &services.BulkOperationResult{
			TotalRequested: len(request.VMContextIDs),
			Successful:     addResult.Successful,
			Failed:         removeResult.Failed + addResult.Failed,
			SuccessfulIDs:  addResult.SuccessfulIDs,
			FailedIDs:      append(removeResult.FailedIDs, addResult.FailedIDs...),
			ErrorMessages:  append(removeResult.ErrorMessages, addResult.ErrorMessages...),
			Duration:       time.Since(start),
		}

		response := AssignmentResponse{
			Success:        addResult.Successful > 0,
			Message:        fmt.Sprintf("Cross-group move completed: %d moved successfully, %d failed", addResult.Successful, combinedResult.Failed),
			BulkResult:     combinedResult,
			ProcessingTime: time.Since(start),
		}

		log.WithFields(log.Fields{
			"source_group_id": request.SourceGroupID,
			"target_group_id": request.TargetGroupID,
			"moved":           addResult.Successful,
			"failed":          combinedResult.Failed,
			"processing_time": response.ProcessingTime,
		}).Info("Cross-group VM move completed")

		h.writeJSONResponse(w, http.StatusOK, response)
	} else {
		response := AssignmentResponse{
			Success:        false,
			Message:        "Failed to remove any VMs from source group",
			BulkResult:     removeResult,
			ProcessingTime: time.Since(start),
		}
		h.writeJSONResponse(w, http.StatusInternalServerError, response)
	}
}

// GetGroupCapacity returns current capacity information for a group
// GET /api/v1/groups/{group_id}/capacity
func (h *VMGroupAssignmentHandler) GetGroupCapacity(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	groupID := vars["group_id"]

	if groupID == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "Group ID is required")
		return
	}

	log.WithField("group_id", groupID).Info("Getting group capacity information")

	ctx := r.Context()
	start := time.Now()

	// Get group information
	group, err := h.machineGroupService.GetGroup(ctx, groupID)
	if err != nil {
		log.WithError(err).Error("Failed to get group for capacity check")
		h.writeErrorResponse(w, http.StatusNotFound, "Group not found: "+err.Error())
		return
	}

	capacityInfo := &GroupCapacityInfo{
		GroupID:          group.Group.ID,
		GroupName:        group.Group.Name,
		CurrentMembers:   group.TotalVMs,
		MaxConcurrentVMs: group.Group.MaxConcurrentVMs,
		Available:        group.Group.MaxConcurrentVMs - group.TotalVMs,
		AtCapacity:       group.TotalVMs >= group.Group.MaxConcurrentVMs,
	}

	response := AssignmentResponse{
		Success:        true,
		Message:        "Group capacity information retrieved successfully",
		GroupCapacity:  capacityInfo,
		ProcessingTime: time.Since(start),
	}

	log.WithFields(log.Fields{
		"group_id":        groupID,
		"current_members": capacityInfo.CurrentMembers,
		"max_concurrent":  capacityInfo.MaxConcurrentVMs,
		"available":       capacityInfo.Available,
		"at_capacity":     capacityInfo.AtCapacity,
	}).Info("Group capacity information retrieved")

	h.writeJSONResponse(w, http.StatusOK, response)
}

// ListGroupVMs lists all VMs in a machine group
// GET /api/v1/machine-groups/{id}/vms
func (h *VMGroupAssignmentHandler) ListGroupVMs(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	groupID := vars["id"]

	if groupID == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "Group ID is required")
		return
	}

	// Get group with memberships
	group, err := h.schedulerRepo.GetGroupByID(groupID, "Memberships", "Memberships.VMContext")
	if err != nil {
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to get group: "+err.Error())
		return
	}
	if group == nil {
		h.writeErrorResponse(w, http.StatusNotFound, "Group not found")
		return
	}

	// Convert memberships to response format
	vms := make([]VMMembershipResponse, len(group.Memberships))
	for i, membership := range group.Memberships {
		vms[i] = VMMembershipResponse{
			VMContextID: membership.VMContextID,
			VMName:      membership.VMContext.VMName,
			VMwareVMID:  membership.VMContext.VMwareVMID,
			Priority:    membership.Priority,
			Enabled:     membership.Enabled,
			AddedAt:     membership.AddedAt,
			AddedBy:     membership.AddedBy,
		}
	}

	response := GroupVMListResponse{
		GroupID:     groupID,
		GroupName:   group.Name,
		TotalVMs:    len(vms),
		VMs:         vms,
		RetrievedAt: time.Now().UTC(),
	}

	h.writeJSONResponse(w, http.StatusOK, response)
}

// AssignVMToGroupByContext assigns a VM to a group using VM context ID
// PUT /api/v1/vm-contexts/{id}/group
func (h *VMGroupAssignmentHandler) AssignVMToGroupByContext(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	vmContextID := vars["id"]

	if vmContextID == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "VM Context ID is required")
		return
	}

	var request AssignToGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	if request.GroupID == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "Group ID is required")
		return
	}

	// Verify VM context exists
	vmContext, err := h.vmContextRepo.GetVMContextWithFullDetails(vmContextID)
	if err != nil {
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to get VM context: "+err.Error())
		return
	}
	if vmContext == nil {
		h.writeErrorResponse(w, http.StatusNotFound, "VM context not found")
		return
	}

	// Verify group exists
	group, err := h.schedulerRepo.GetGroupByID(request.GroupID)
	if err != nil {
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to get group: "+err.Error())
		return
	}
	if group == nil {
		h.writeErrorResponse(w, http.StatusNotFound, "Group not found")
		return
	}

	// Create membership request
	membershipReq := &services.VMMembershipRequest{
		VMContextID: vmContextID,
		Priority:    request.Priority,
		Enabled:     request.Enabled,
	}

	// Add VM to group
	membership, err := h.machineGroupService.AddVMToGroup(r.Context(), request.GroupID, membershipReq)
	if err != nil {
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to assign VM to group: "+err.Error())
		return
	}

	// Use the AddedBy from request if provided, otherwise use the value from the created membership
	addedBy := membership.AddedBy
	if request.AddedBy != "" {
		addedBy = request.AddedBy
	}

	response := VMAssignmentResponse{
		Success:     true,
		Message:     "VM successfully assigned to group",
		VMContextID: vmContextID,
		GroupID:     request.GroupID,
		Priority:    membership.Priority,
		Enabled:     membership.Enabled,
		AddedAt:     membership.AddedAt,
		AddedBy:     addedBy,
	}

	h.writeJSONResponse(w, http.StatusOK, response)
}

// writeJSONResponse writes a JSON response
func (h *VMGroupAssignmentHandler) writeJSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.WithError(err).Error("Failed to write JSON response")
	}
}

// writeErrorResponse writes an error response
func (h *VMGroupAssignmentHandler) writeErrorResponse(w http.ResponseWriter, statusCode int, message string) {
	response := map[string]interface{}{
		"error":     message,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}
	h.writeJSONResponse(w, statusCode, response)
}
