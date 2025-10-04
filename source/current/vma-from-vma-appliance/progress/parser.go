// Package progress provides real-time migration progress tracking and error reporting
package progress

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

// JobProgress represents comprehensive migration progress including errors and sync type
type JobProgress struct {
	JobID            string          `json:"job_id"`
	Status           string          `json:"status"` // "starting", "running", "completed", "failed", "error"
	SyncType         string          `json:"sync_type"` // "initial", "incremental", "unknown"
	Phase            string          `json:"phase"` // "initializing", "snapshot_creation", "copying_data", "cleanup", "error"
	Percentage       float64         `json:"percentage"`
	CurrentOperation string          `json:"current_operation"`
	
	// Data transfer metrics
	BytesTransferred int64           `json:"bytes_transferred"`
	TotalBytes       int64           `json:"total_bytes"`
	
	// Throughput metrics
	Throughput       ThroughputData  `json:"throughput"`
	
	// Timing information
	Timing           TimingData      `json:"timing"`
	
	// VM information
	VMInfo           VMProgressInfo  `json:"vm_info"`
	
	// Error tracking
	Errors           []ErrorInfo     `json:"errors,omitempty"`
	LastError        *ErrorInfo      `json:"last_error,omitempty"`
	
	// Phase tracking
	Phases           []PhaseInfo     `json:"phases"`
}

// ThroughputData contains throughput metrics
type ThroughputData struct {
	CurrentMBps  float64 `json:"current_mbps"`
	AverageMBps  float64 `json:"average_mbps"`
	PeakMBps     float64 `json:"peak_mbps"`
	LastUpdate   time.Time `json:"last_update"`
}

// TimingData contains timing information
type TimingData struct {
	StartTime     time.Time `json:"start_time"`
	LastUpdate    time.Time `json:"last_update"`
	ElapsedMs     int64     `json:"elapsed_ms"`
	ETASeconds    int       `json:"eta_seconds"`
	PhaseStart    time.Time `json:"phase_start"`
	PhaseElapsed  int64     `json:"phase_elapsed_ms"`
}

// VMProgressInfo contains VM-specific progress information
type VMProgressInfo struct {
	Name         string `json:"name"`
	Path         string `json:"path"`
	DiskSizeGB   float64 `json:"disk_size_gb"`
	DiskSizeBytes int64  `json:"disk_size_bytes"`
	ExportName   string `json:"export_name,omitempty"`
	CBTEnabled   bool   `json:"cbt_enabled"`
	PreviousChangeID string `json:"previous_change_id,omitempty"`
}

// ErrorInfo contains detailed error information
type ErrorInfo struct {
	Timestamp   time.Time `json:"timestamp"`
	Phase       string    `json:"phase"`
	ErrorType   string    `json:"error_type"` // "connection", "authentication", "permission", "disk", "network", "system"
	Message     string    `json:"message"`
	Details     string    `json:"details,omitempty"`
	Recoverable bool      `json:"recoverable"`
}

// PhaseInfo tracks individual migration phases
type PhaseInfo struct {
	Name        string    `json:"name"`
	Status      string    `json:"status"` // "pending", "running", "completed", "failed"
	StartTime   time.Time `json:"start_time,omitempty"`
	EndTime     time.Time `json:"end_time,omitempty"`
	DurationMs  int64     `json:"duration_ms"`
	Percentage  float64   `json:"percentage,omitempty"`
}

// ProgressParser parses migratekit log files in real-time
type ProgressParser struct {
	jobID       string
	logPath     string
	progress    *JobProgress
	file        *os.File
	scanner     *bufio.Scanner
	
	// Regex patterns for parsing
	progressRegex   *regexp.Regexp
	errorRegex      *regexp.Regexp
	cbtRegex        *regexp.Regexp
	snapshotRegex   *regexp.Regexp
	sizeRegex       *regexp.Regexp
	changeIDRegex   *regexp.Regexp
	
	// Throughput calculation
	lastProgressTime  time.Time
	lastProgressBytes int64
	throughputHistory []float64
	
	// NBDCopy pipe integration
	nbdProgressChan   chan float64
	latestNBDProgress float64
	usingNBDProgress  bool
	mu                sync.RWMutex
}

// NewProgressParser creates a new progress parser for a job
func NewProgressParser(jobID string, logPath string) *ProgressParser {
	parser := &ProgressParser{
		jobID:             jobID,
		logPath:           logPath,
		nbdProgressChan:   make(chan float64, 10), // Buffered channel for NBD progress
		latestNBDProgress: 0.0,
		usingNBDProgress:  false,
		progress: &JobProgress{
			JobID:    jobID,
			Status:   "starting",
			SyncType: "unknown",
			Phase:    "initializing",
			Timing: TimingData{
				StartTime:  time.Now(),
				LastUpdate: time.Now(),
			},
			Phases: []PhaseInfo{
				{Name: "initializing", Status: "running", StartTime: time.Now()},
				{Name: "snapshot_creation", Status: "pending"},
				{Name: "copying_data", Status: "pending"},
				{Name: "cleanup", Status: "pending"},
			},
		},
	}
	
	// Compile regex patterns
	parser.compilePatterns()
	
	// Start NBD progress monitor
	go parser.monitorNBDProgress()
	
	return parser
}

// compilePatterns compiles all regex patterns for log parsing
func (p *ProgressParser) compilePatterns() {
	// Progress pattern: "Copying VM data   45% [===] (45/100) [2m15s:3m10s]"
	p.progressRegex = regexp.MustCompile(`Copying VM data\s+(\d+)%.*?\[(\d+)m(\d+)s:(\d+)m(\d+)s\]`)
	
	// Error patterns
	p.errorRegex = regexp.MustCompile(`level=error|ERROR|FATAL|failed|Failed|Error:|error:`)
	
	// CBT and sync type detection
	p.cbtRegex = regexp.MustCompile(`CBT.*enabled|Change tracking.*enabled|incremental|full.*copy.*needed|No ChangeID found`)
	
	// Snapshot creation
	p.snapshotRegex = regexp.MustCompile(`Creating snapshot.*(\d+)%`)
	
	// Data size detection
	p.sizeRegex = regexp.MustCompile(`Expected data size:\s*([\d.]+)\s*GB.*?(\d+)\s*bytes`)
	
	// ChangeID detection
	p.changeIDRegex = regexp.MustCompile(`Stored ChangeID.*:\s*([a-f0-9\s\-/]+)`)
}

// StartParsing starts real-time parsing of the log file
func (p *ProgressParser) StartParsing() error {
	// Wait for log file to be created (with timeout)
	timeout := time.After(30 * time.Second)
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-timeout:
			p.addError("system", "Log file creation timeout", "Migratekit log file not created within 30 seconds", false)
			p.progress.Status = "failed"
			return fmt.Errorf("log file not created: %s", p.logPath)
		case <-ticker.C:
			if _, err := os.Stat(p.logPath); err == nil {
				goto FileExists
			}
		}
	}
	
FileExists:
	var err error
	p.file, err = os.Open(p.logPath)
	if err != nil {
		p.addError("system", "Failed to open log file", err.Error(), false)
		return err
	}
	
	p.scanner = bufio.NewScanner(p.file)
	p.progress.Status = "running"
	
	// Start parsing in background
	go p.parseLoop()
	
	return nil
}

// parseLoop continuously parses the log file
func (p *ProgressParser) parseLoop() {
	defer func() {
		if p.file != nil {
			p.file.Close()
		}
	}()
	
	for p.scanner.Scan() {
		line := p.scanner.Text()
		p.parseLine(line)
	}
	
	// Check if file ended normally or due to error
	if err := p.scanner.Err(); err != nil {
		p.addError("system", "Log parsing error", err.Error(), false)
	}
	
	// If we reach here, migration likely completed
	if p.progress.Status == "running" {
		p.progress.Status = "completed"
		p.updatePhase("cleanup", "completed")
	}
}

// parseLine parses a single log line and updates progress
func (p *ProgressParser) parseLine(line string) {
	now := time.Now()
	p.progress.Timing.LastUpdate = now
	
	// Detect sync type
	if strings.Contains(line, "full copy needed") || strings.Contains(line, "No ChangeID found") {
		p.progress.SyncType = "initial"
		p.progress.VMInfo.PreviousChangeID = ""
	} else if strings.Contains(line, "incremental") || (strings.Contains(line, "ChangeID") && !strings.Contains(line, "No ChangeID")) {
		p.progress.SyncType = "incremental"
	}
	
	// Detect CBT status
	if strings.Contains(line, "CBT) is enabled") || strings.Contains(line, "Change tracking") {
		p.progress.VMInfo.CBTEnabled = true
	}
	
	// Extract VM information
	if matches := p.sizeRegex.FindStringSubmatch(line); len(matches) >= 3 {
		if sizeGB, err := strconv.ParseFloat(matches[1], 64); err == nil {
			p.progress.VMInfo.DiskSizeGB = sizeGB
		}
		if sizeBytes, err := strconv.ParseInt(matches[2], 10, 64); err == nil {
			p.progress.VMInfo.DiskSizeBytes = sizeBytes
			p.progress.TotalBytes = sizeBytes
		}
	}
	
	// Parse progress percentage
	if matches := p.progressRegex.FindStringSubmatch(line); len(matches) >= 6 {
		if percentage, err := strconv.ParseFloat(matches[1], 64); err == nil {
			p.updateProgress(percentage, now)
			p.updatePhase("copying_data", "running")
		}
		
		// Calculate ETA from time parts
		if elapsedMin, err1 := strconv.Atoi(matches[2]); err1 == nil {
			if elapsedSec, err2 := strconv.Atoi(matches[3]); err2 == nil {
				if etaMin, err3 := strconv.Atoi(matches[4]); err3 == nil {
					if etaSec, err4 := strconv.Atoi(matches[5]); err4 == nil {
						p.progress.Timing.ElapsedMs = int64((elapsedMin*60 + elapsedSec) * 1000)
						p.progress.Timing.ETASeconds = etaMin*60 + etaSec
					}
				}
			}
		}
	}
	
	// Parse snapshot creation progress
	if matches := p.snapshotRegex.FindStringSubmatch(line); len(matches) >= 2 {
		if percentage, err := strconv.ParseFloat(matches[1], 64); err == nil {
			if percentage == 100 {
				p.updatePhase("snapshot_creation", "completed")
				p.updatePhase("copying_data", "running")
			} else {
				p.updatePhase("snapshot_creation", "running")
			}
		}
	}
	
	// Detect errors
	if p.errorRegex.MatchString(line) {
		errorType := p.classifyError(line)
		p.addError(errorType, "Migration error", line, p.isRecoverableError(line))
		
		// Some errors are fatal
		if p.isFatalError(line) {
			p.progress.Status = "failed"
		}
	}
	
	// Detect ChangeID extraction (completion)
	if matches := p.changeIDRegex.FindStringSubmatch(line); len(matches) >= 2 {
		p.progress.Status = "completed"
		p.updatePhase("cleanup", "completed")
	}
}

// updateProgress updates progress percentage and calculates throughput
func (p *ProgressParser) updateProgress(percentage float64, timestamp time.Time) {
	p.progress.Percentage = percentage
	p.progress.CurrentOperation = "Copying VM data"
	
	// Calculate bytes transferred
	if p.progress.TotalBytes > 0 {
		p.progress.BytesTransferred = int64(float64(p.progress.TotalBytes) * percentage / 100.0)
	}
	
	// Calculate throughput
	if !p.lastProgressTime.IsZero() && p.lastProgressBytes > 0 {
		timeDelta := timestamp.Sub(p.lastProgressTime).Seconds()
		bytesDelta := p.progress.BytesTransferred - p.lastProgressBytes
		
		if timeDelta > 0 && bytesDelta > 0 {
			currentMBps := float64(bytesDelta) / (1024 * 1024) / timeDelta
			p.progress.Throughput.CurrentMBps = currentMBps
			
			// Track throughput history for averages
			p.throughputHistory = append(p.throughputHistory, currentMBps)
			if len(p.throughputHistory) > 60 { // Keep last 60 measurements
				p.throughputHistory = p.throughputHistory[1:]
			}
			
			// Calculate average and peak
			var sum, peak float64
			for _, mbps := range p.throughputHistory {
				sum += mbps
				if mbps > peak {
					peak = mbps
				}
			}
			p.progress.Throughput.AverageMBps = sum / float64(len(p.throughputHistory))
			p.progress.Throughput.PeakMBps = peak
			p.progress.Throughput.LastUpdate = timestamp
		}
	}
	
	p.lastProgressTime = timestamp
	p.lastProgressBytes = p.progress.BytesTransferred
}

// updatePhase updates the status of a migration phase
func (p *ProgressParser) updatePhase(phaseName, status string) {
	now := time.Now()
	
	for i, phase := range p.progress.Phases {
		if phase.Name == phaseName {
			if phase.Status != status {
				p.progress.Phases[i].Status = status
				
				if status == "running" && phase.StartTime.IsZero() {
					p.progress.Phases[i].StartTime = now
					p.progress.Phase = phaseName
					p.progress.Timing.PhaseStart = now
				} else if status == "completed" && !phase.EndTime.IsZero() == false {
					p.progress.Phases[i].EndTime = now
					if !phase.StartTime.IsZero() {
						p.progress.Phases[i].DurationMs = now.Sub(phase.StartTime).Milliseconds()
					}
				}
			}
			break
		}
	}
}

// addError adds an error to the progress tracking
func (p *ProgressParser) addError(errorType, message, details string, recoverable bool) {
	errorInfo := ErrorInfo{
		Timestamp:   time.Now(),
		Phase:       p.progress.Phase,
		ErrorType:   errorType,
		Message:     message,
		Details:     details,
		Recoverable: recoverable,
	}
	
	p.progress.Errors = append(p.progress.Errors, errorInfo)
	p.progress.LastError = &errorInfo
	
	log.WithFields(log.Fields{
		"job_id":      p.jobID,
		"error_type":  errorType,
		"message":     message,
		"recoverable": recoverable,
	}).Warn("Migration error detected")
}

// classifyError classifies error type based on the error message
func (p *ProgressParser) classifyError(line string) string {
	line = strings.ToLower(line)
	
	if strings.Contains(line, "connection") || strings.Contains(line, "connect") {
		return "connection"
	} else if strings.Contains(line, "auth") || strings.Contains(line, "login") || strings.Contains(line, "credential") {
		return "authentication"
	} else if strings.Contains(line, "permission") || strings.Contains(line, "access denied") {
		return "permission"
	} else if strings.Contains(line, "disk") || strings.Contains(line, "vmdk") || strings.Contains(line, "snapshot") {
		return "disk"
	} else if strings.Contains(line, "network") || strings.Contains(line, "nbd") || strings.Contains(line, "tunnel") {
		return "network"
	} else {
		return "system"
	}
}

// isRecoverableError determines if an error is recoverable
func (p *ProgressParser) isRecoverableError(line string) bool {
	line = strings.ToLower(line)
	
	// Recoverable errors
	if strings.Contains(line, "timeout") || strings.Contains(line, "retry") || strings.Contains(line, "temporary") {
		return true
	}
	
	// Non-recoverable errors
	if strings.Contains(line, "fatal") || strings.Contains(line, "permission denied") || strings.Contains(line, "authentication failed") {
		return false
	}
	
	return true // Default to recoverable
}

// isFatalError determines if an error should stop the migration
func (p *ProgressParser) isFatalError(line string) bool {
	line = strings.ToLower(line)
	return strings.Contains(line, "fatal") || strings.Contains(line, "failed to start")
}

// updatePhaseStatus updates the phase status in the phases array
func (p *ProgressParser) updatePhaseStatus(phaseName, status string) {
	for i := range p.progress.Phases {
		if p.progress.Phases[i].Name == phaseName {
			p.progress.Phases[i].Status = status
			if status == "running" && p.progress.Phases[i].StartTime.IsZero() {
				p.progress.Phases[i].StartTime = time.Now()
			} else if status == "completed" && p.progress.Phases[i].EndTime.IsZero() {
				p.progress.Phases[i].EndTime = time.Now()
				if !p.progress.Phases[i].StartTime.IsZero() {
					p.progress.Phases[i].DurationMs = p.progress.Phases[i].EndTime.Sub(p.progress.Phases[i].StartTime).Milliseconds()
				}
			}
			break
		}
	}
}

// parseLogFile reads and parses the log file to update progress state
func (p *ProgressParser) parseLogFile() {
	file, err := os.Open(p.logPath)
	if err != nil {
		// Log file might not exist yet, keep current state
		return
	}
	defer file.Close()
	
	// Reset basic state for fresh parsing (preserve phase timing data)
	hasExistingPhases := len(p.progress.Phases) > 0
	
	p.progress.Status = "starting"
	p.progress.Phase = "initializing"
	p.progress.Percentage = 0.0
	p.progress.CurrentOperation = "Starting migration"
	p.progress.BytesTransferred = 0
	p.progress.TotalBytes = 0
	p.progress.Errors = []ErrorInfo{}
	p.progress.LastError = nil
	
	// Initialize phases if not already done
	if !hasExistingPhases {
		p.progress.Phases = []PhaseInfo{
			{Name: "initializing", Status: "running", StartTime: time.Now()},
			{Name: "snapshot_creation", Status: "pending"},
			{Name: "copying_data", Status: "pending"},
			{Name: "cleanup", Status: "pending"},
		}
	}
	
	// Regex patterns (improved for migratekit log format)
	progressRegex := regexp.MustCompile(`\((\d+)/100\)`)
	sizeRegex := regexp.MustCompile(`(\d+\.?\d*)\s*GB`)
	totalSizeRegex := regexp.MustCompile(`Expected data size: (\d+\.?\d*) GB \((\d+) bytes\)`)
	
	scanner := bufio.NewScanner(file)
	hasErrors := false
	
	for scanner.Scan() {
		line := scanner.Text()
		
		// Split line on carriage returns (same as test script)
		virtualLines := strings.Split(line, "\r")
		
		for _, virtualLine := range virtualLines {
			virtualLine = strings.TrimSpace(virtualLine)
			if virtualLine == "" {
				continue
			}
			
			// Error detection (improved to avoid hex dump false positives)
			lowerLine := strings.ToLower(virtualLine)
			// Skip hex dump lines (format: "0030: 1e 00 00...")
			isHexDump := regexp.MustCompile(`^[0-9a-f]{4}:\s+[0-9a-f\s]+\|.*\|$`).MatchString(lowerLine)
			if !isHexDump && (strings.Contains(lowerLine, "error") || strings.Contains(lowerLine, "failed") || 
			   strings.Contains(lowerLine, "failure") || strings.Contains(lowerLine, "unable to")) {
				hasErrors = true
				p.progress.Status = "failed"
				errorInfo := ErrorInfo{
					Message:   virtualLine,
					Timestamp: time.Now(),
					ErrorType: "system",
					Phase:     p.progress.Phase,
				}
				p.progress.Errors = append(p.progress.Errors, errorInfo)
				p.progress.LastError = &errorInfo
				continue
			}
			
			// Extract total data size
			if matches := totalSizeRegex.FindStringSubmatch(virtualLine); len(matches) >= 3 {
				if totalBytes, err := strconv.ParseInt(matches[2], 10, 64); err == nil {
					p.progress.TotalBytes = totalBytes
				}
			}
			
			// Job state transitions (same as test script)
			if strings.Contains(virtualLine, "Starting") && strings.Contains(virtualLine, "copy") {
				if p.progress.Status == "starting" {
					p.progress.Status = "running"
				}
			}
			
			// Phase detection and progress tracking (same as test script)
			if strings.Contains(virtualLine, "Creating snapshot") {
				if p.progress.Phase != "snapshot_creation" {
					// Mark previous phase as completed if transitioning
					if p.progress.Phase == "initializing" {
						p.updatePhaseStatus("initializing", "completed")
					}
					
					p.progress.Phase = "snapshot_creation"
					p.progress.Status = "running"
					p.progress.CurrentOperation = "Creating VM snapshot"
					p.updatePhaseStatus("snapshot_creation", "running")
				}
				
				// Get snapshot progress
				if matches := progressRegex.FindStringSubmatch(virtualLine); len(matches) >= 2 {
					if pct, err := strconv.ParseFloat(matches[1], 64); err == nil {
						p.progress.Percentage = pct * 0.1 // Snapshot is ~10% of overall
						p.progress.Timing.LastUpdate = time.Now()
						
						// Mark snapshot as completed when it reaches 100%
						if pct == 100 {
							p.updatePhaseStatus("snapshot_creation", "completed")
						}
					}
				}
			} else if strings.Contains(virtualLine, "Starting copy") || strings.Contains(virtualLine, "Copying") || 
			          strings.Contains(virtualLine, "nbd_pwrite") || strings.Contains(virtualLine, "nbdcopy") {
				if p.progress.Phase != "copying_data" {
					// Mark previous phases as completed if transitioning
					if p.progress.Phase == "initializing" {
						p.updatePhaseStatus("initializing", "completed")
					}
					if p.progress.Phase == "snapshot_creation" {
						p.updatePhaseStatus("snapshot_creation", "completed")
					}
					
					p.progress.Phase = "copying_data"
					p.progress.Status = "running"
					p.progress.CurrentOperation = "Copying VM data"
					p.updatePhaseStatus("copying_data", "running")
				}
				
				// Get copying progress
				if matches := progressRegex.FindStringSubmatch(virtualLine); len(matches) >= 2 {
					if pct, err := strconv.ParseFloat(matches[1], 64); err == nil {
						// Copying is 85% of overall (10% snapshot + 85% copying + 5% cleanup)
						p.progress.Percentage = 10.0 + (pct * 0.85)
						p.progress.Timing.LastUpdate = time.Now()
						
						// Calculate bytes transferred from percentage and total
						if p.progress.TotalBytes > 0 {
							p.progress.BytesTransferred = int64(float64(p.progress.TotalBytes) * (pct / 100.0))
						}
						
						// Extract operation details for current_operation
						if strings.Contains(virtualLine, "MB/s") || strings.Contains(virtualLine, "GB/s") {
							p.progress.CurrentOperation = fmt.Sprintf("Copying VM data: %.0f%% - %s", pct, virtualLine)
						} else {
							p.progress.CurrentOperation = fmt.Sprintf("Copying VM data: %.0f%%", pct)
						}
						
						// Mark copying as completed when it reaches 100%
						if pct == 100 {
							p.updatePhaseStatus("copying_data", "completed")
						}
					}
				}
				
				// Parse data sizes (same as test script)
				if matches := sizeRegex.FindAllStringSubmatch(virtualLine, -1); len(matches) >= 2 {
					if transferred, err1 := strconv.ParseFloat(matches[0][1], 64); err1 == nil {
						if total, err2 := strconv.ParseFloat(matches[1][1], 64); err2 == nil {
							p.progress.BytesTransferred = int64(transferred * 1024 * 1024 * 1024) // GB to bytes
							p.progress.TotalBytes = int64(total * 1024 * 1024 * 1024)
						}
					}
				}
			} else if strings.Contains(virtualLine, "completed") || strings.Contains(virtualLine, "Migration completed") {
				// Mark all previous phases as completed
				p.updatePhaseStatus("initializing", "completed")
				p.updatePhaseStatus("snapshot_creation", "completed")
				p.updatePhaseStatus("copying_data", "completed")
				p.updatePhaseStatus("cleanup", "completed")
				
				p.progress.Phase = "completed"
				p.progress.Status = "completed"
				p.progress.Percentage = 100.0
				p.progress.CurrentOperation = "Migration completed successfully"
			}
		}
	}
	
	// Set final status based on parsing results
	if hasErrors {
		p.progress.Status = "failed"
	} else if p.progress.Percentage >= 100.0 {
		p.progress.Status = "completed"
	} else if p.progress.Phase != "initializing" {
		p.progress.Status = "running"
	}
}

// GetProgress returns the current progress state
func (p *ProgressParser) GetProgress() *JobProgress {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	// Parse the log file to get current status
	p.parseLogFile()
	
	// Use NBD progress if available and we're in copying phase
	if p.usingNBDProgress && p.progress.Phase == "copying_data" {
		p.progress.Percentage = p.latestNBDProgress
		p.progress.CurrentOperation = fmt.Sprintf("Copying data via NBD: %.1f%% complete", p.latestNBDProgress)
	}
	
	// Update overall timing
	if !p.progress.Timing.StartTime.IsZero() {
		p.progress.Timing.ElapsedMs = time.Since(p.progress.Timing.StartTime).Milliseconds()
	}
	if !p.progress.Timing.PhaseStart.IsZero() {
		p.progress.Timing.PhaseElapsed = time.Since(p.progress.Timing.PhaseStart).Milliseconds()
	}
	
	return p.progress
}

// Close closes the parser and cleans up resources
func (p *ProgressParser) Close() error {
	if p.nbdProgressChan != nil {
		close(p.nbdProgressChan)
	}
	if p.file != nil {
		return p.file.Close()
	}
	return nil
}

// NBDProgressData represents the structured progress data from migratekit JSON file
type NBDProgressData struct {
	Percentage       float64   `json:"percentage"`
	BytesTransferred int64     `json:"bytes_transferred"`
	TotalBytes       int64     `json:"total_bytes"`
	ThroughputMBps   float64   `json:"throughput_mbps"`
	LastUpdate       time.Time `json:"last_update"`
	Status           string    `json:"status"`
}

// ReadProgressFromFile reads progress data from the structured JSON file (more reliable than log parsing)
func ReadProgressFromFile(jobID string) (*JobProgress, error) {
	progressFile := fmt.Sprintf("/tmp/migratekit-progress-%s.json", jobID)
	
	// Check if file exists
	if _, err := os.Stat(progressFile); os.IsNotExist(err) {
		return nil, fmt.Errorf("progress file not found: %s", progressFile)
	}
	
	// Read progress file
	data, err := os.ReadFile(progressFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read progress file: %w", err)
	}
	
	// Parse JSON data
	var nbdProgress NBDProgressData
	if err := json.Unmarshal(data, &nbdProgress); err != nil {
		return nil, fmt.Errorf("failed to parse progress JSON: %w", err)
	}
	
	// Convert to JobProgress format
	now := time.Now()
	progress := &JobProgress{
		JobID:            jobID,
		Status:           "running",
		SyncType:         "unknown", // Will be detected from logs if needed
		Phase:            "copying_data",
		Percentage:       nbdProgress.Percentage,
		CurrentOperation: "Copying VM data",
		BytesTransferred: nbdProgress.BytesTransferred,
		TotalBytes:       nbdProgress.TotalBytes,
		Throughput: ThroughputData{
			CurrentMBps:  nbdProgress.ThroughputMBps,
			AverageMBps:  nbdProgress.ThroughputMBps, // For now, use current as average
			PeakMBps:     nbdProgress.ThroughputMBps, // Will be tracked over time
			LastUpdate:   nbdProgress.LastUpdate,
		},
		Timing: TimingData{
			LastUpdate:    now,
			ElapsedMs:     now.Sub(nbdProgress.LastUpdate).Milliseconds(),
		},
		VMInfo: VMProgressInfo{
			DiskSizeBytes: nbdProgress.TotalBytes,
			DiskSizeGB:    float64(nbdProgress.TotalBytes) / (1024 * 1024 * 1024),
		},
		Phases: []PhaseInfo{
			{Name: "initializing", Status: "completed"},
			{Name: "snapshot_creation", Status: "completed"},
			{Name: "copying_data", Status: "running", Percentage: nbdProgress.Percentage},
			{Name: "cleanup", Status: "pending"},
		},
	}
	
	// Mark as completed if at 100%
	if nbdProgress.Percentage >= 100 {
		progress.Status = "completed"
		progress.Phase = "cleanup"
		progress.Phases[2].Status = "completed"
		progress.Phases[3].Status = "completed"
	}
	
	return progress, nil
}

// UpdateNBDProgress updates the NBD progress percentage from pipe data
func (p *ProgressParser) UpdateNBDProgress(percentage float64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	p.latestNBDProgress = percentage
	p.usingNBDProgress = true
	
	// Send to channel for monitoring goroutine
	select {
	case p.nbdProgressChan <- percentage:
		// Successfully sent
	default:
		// Channel full, skip this update
	}
}

// monitorNBDProgress monitors NBD progress updates in background
func (p *ProgressParser) monitorNBDProgress() {
	for percentage := range p.nbdProgressChan {
		p.mu.Lock()
		
		// Update progress only if we're in copying phase
		if p.progress.Phase == "copying_data" {
			p.progress.Percentage = percentage
			p.progress.Timing.LastUpdate = time.Now()
			
			// Calculate throughput if we have total bytes
			if p.progress.TotalBytes > 0 {
				p.calculateThroughputFromNBD(percentage)
			}
		}
		
		p.mu.Unlock()
	}
}

// calculateThroughputFromNBD calculates throughput from NBD percentage
func (p *ProgressParser) calculateThroughputFromNBD(percentage float64) {
	now := time.Now()
	currentBytes := int64(float64(p.progress.TotalBytes) * percentage / 100.0)
	
	if !p.lastProgressTime.IsZero() {
		timeDiff := now.Sub(p.lastProgressTime).Seconds()
		bytesDiff := currentBytes - p.lastProgressBytes
		
		if timeDiff > 0 && bytesDiff >= 0 {
			currentMBps := float64(bytesDiff) / (1024 * 1024) / timeDiff
			
			// Update throughput history
			p.throughputHistory = append(p.throughputHistory, currentMBps)
			if len(p.throughputHistory) > 10 {
				p.throughputHistory = p.throughputHistory[1:]
			}
			
			// Calculate average
			var total float64
			for _, mbps := range p.throughputHistory {
				total += mbps
			}
			avgMBps := total / float64(len(p.throughputHistory))
			
			p.progress.Throughput = ThroughputData{
				CurrentMBps: currentMBps,
				AverageMBps: avgMBps,
			}
			
			p.progress.BytesTransferred = currentBytes
		}
	}
	
	p.lastProgressTime = now
	p.lastProgressBytes = currentBytes
}


