package models

import "time"

// APIResponse represents a standard API response wrapper
type APIResponse struct {
	Success   bool        `json:"success"`
	Data      interface{} `json:"data,omitempty"`
	Error     string      `json:"error,omitempty"`
	Code      string      `json:"code,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
}

// AuthRequest represents authentication request from VMA
type AuthRequest struct {
	ApplianceID string `json:"appliance_id" binding:"required"`
	Token       string `json:"token" binding:"required"`
	Version     string `json:"version" binding:"required"`
}

// AuthResponse represents authentication response to VMA
type AuthResponse struct {
	Success      bool      `json:"success"`
	SessionToken string    `json:"session_token"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// VMInventoryRequest represents VM inventory submission from VMA
type VMInventoryRequest struct {
	ApplianceID string      `json:"appliance_id" binding:"required"`
	Datacenter  string      `json:"datacenter" binding:"required"`
	VMs         []VMInfo    `json:"vms" binding:"required"`
	VCenter     VCenterInfo `json:"vcenter"`
	Timestamp   time.Time   `json:"timestamp"`
}

// JobStatusRequest represents job status update from VMA
type JobStatusRequest struct {
	JobID            string  `json:"job_id" binding:"required"`
	Status           string  `json:"status" binding:"required"`
	Progress         float64 `json:"progress"`
	CurrentOperation string  `json:"current_operation"`
	BytesTransferred int64   `json:"bytes_transferred"`
	TransferSpeedBps int64   `json:"transfer_speed_bps"`
	ErrorMessage     string  `json:"error_message"`
}
