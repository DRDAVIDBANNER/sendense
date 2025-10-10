package telemetry

// TelemetryUpdate represents a telemetry update to send to SHA
type TelemetryUpdate struct {
	JobID            string          `json:"job_id"`
	JobType          string          `json:"job_type"` // "backup", "replication", "restore"
	Status           string          `json:"status"`   // "running", "completed", "failed"
	CurrentPhase     string          `json:"current_phase"` // "snapshot", "transferring", "finalizing"
	BytesTransferred int64           `json:"bytes_transferred"`
	TotalBytes       int64           `json:"total_bytes"`
	TransferSpeedBps int64           `json:"transfer_speed_bps"`
	ETASeconds       int             `json:"eta_seconds"`
	ProgressPercent  float64         `json:"progress_percent"`
	Disks            []DiskTelemetry `json:"disks"`
	Error            *ErrorInfo      `json:"error,omitempty"`
	Timestamp        string          `json:"timestamp"`
}

// DiskTelemetry represents per-disk progress
type DiskTelemetry struct {
	DiskIndex        int     `json:"disk_index"`
	BytesTransferred int64   `json:"bytes_transferred"`
	TotalBytes       int64   `json:"total_bytes"`
	Status           string  `json:"status"` // "pending", "transferring", "completed"
	ProgressPercent  float64 `json:"progress_percent"`
}

// ErrorInfo represents error information
type ErrorInfo struct {
	Message   string `json:"message"`
	Code      string `json:"code,omitempty"`
	Timestamp string `json:"timestamp"`
}

