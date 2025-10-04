package services

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/vexxhost/migratekit-oma/models"
)

// CentralLogger provides structured logging with job correlation and audit trails
type CentralLogger struct {
	db                 *gorm.DB
	jobTrackingService *JobTrackingService
	logger             *log.Logger
	logDir             string
	mutex              sync.RWMutex

	// Log rotation settings
	maxLogFileSize int64 // bytes
	maxLogFiles    int
	rotationHours  int
}

// LogEntry represents a structured log entry with correlation
type LogEntry struct {
	ID            string                 `json:"id"`
	Timestamp     time.Time              `json:"timestamp"`
	Level         string                 `json:"level"`
	Message       string                 `json:"message"`
	CorrelationID string                 `json:"correlation_id,omitempty"`
	JobID         string                 `json:"job_id,omitempty"`
	Component     string                 `json:"component"`
	Operation     string                 `json:"operation,omitempty"`
	Context       map[string]interface{} `json:"context,omitempty"`
	Error         *ErrorDetails          `json:"error,omitempty"`
	Duration      *time.Duration         `json:"duration,omitempty"`
}

// ErrorDetails provides structured error information
type ErrorDetails struct {
	Type       string                 `json:"type"`
	Message    string                 `json:"message"`
	Code       string                 `json:"code,omitempty"`
	Details    map[string]interface{} `json:"details,omitempty"`
	StackTrace string                 `json:"stack_trace,omitempty"`
}

// LoggerConfig configures the central logger
type LoggerConfig struct {
	LogDir         string `json:"log_dir"`
	MaxLogFileSize int64  `json:"max_log_file_size"` // bytes
	MaxLogFiles    int    `json:"max_log_files"`
	RotationHours  int    `json:"rotation_hours"`
}

// NewCentralLogger creates a new central logger instance
func NewCentralLogger(db *gorm.DB, jobTrackingService *JobTrackingService, config LoggerConfig) (*CentralLogger, error) {
	if config.LogDir == "" {
		config.LogDir = "/var/log/migratekit"
	}
	if config.MaxLogFileSize == 0 {
		config.MaxLogFileSize = 100 * 1024 * 1024 // 100MB default
	}
	if config.MaxLogFiles == 0 {
		config.MaxLogFiles = 10 // Keep 10 files by default
	}
	if config.RotationHours == 0 {
		config.RotationHours = 24 // Rotate daily by default
	}

	// Ensure log directory exists
	if err := os.MkdirAll(config.LogDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	// Create custom logger for file output
	logger := log.New()
	logger.SetFormatter(&log.JSONFormatter{
		TimestampFormat: time.RFC3339Nano,
	})

	cl := &CentralLogger{
		db:                 db,
		jobTrackingService: jobTrackingService,
		logger:             logger,
		logDir:             config.LogDir,
		maxLogFileSize:     config.MaxLogFileSize,
		maxLogFiles:        config.MaxLogFiles,
		rotationHours:      config.RotationHours,
	}

	// Set up log file output
	if err := cl.setupLogFile(); err != nil {
		return nil, fmt.Errorf("failed to setup log file: %w", err)
	}

	// Start log rotation goroutine
	go cl.logRotationWorker()

	return cl, nil
}

// LogWithCorrelation logs a message with job correlation
func (cl *CentralLogger) LogWithCorrelation(
	ctx context.Context,
	level log.Level,
	component, operation, message string,
	correlationID, jobID *string,
	context map[string]interface{},
	err error,
	duration *time.Duration,
) {
	entry := &LogEntry{
		ID:        uuid.New().String(),
		Timestamp: time.Now(),
		Level:     level.String(),
		Message:   message,
		Component: component,
		Operation: operation,
		Context:   context,
		Duration:  duration,
	}

	if correlationID != nil {
		entry.CorrelationID = *correlationID
	}

	if jobID != nil {
		entry.JobID = *jobID
	}

	if err != nil {
		entry.Error = &ErrorDetails{
			Type:    fmt.Sprintf("%T", err),
			Message: err.Error(),
		}

		// Add additional error context if available
		if context != nil {
			entry.Error.Details = context
		}
	}

	// Write to structured log file
	cl.writeToLogFile(entry)

	// Also write to database execution log if job ID is provided
	if jobID != nil {
		cl.writeToJobExecutionLog(ctx, *jobID, entry)
	}

	// Write to standard log output for console/systemd
	cl.writeToStandardLog(entry, err)
}

// LogInfo logs an info message with correlation
func (cl *CentralLogger) LogInfo(ctx context.Context, component, operation, message string, correlationID, jobID *string, context map[string]interface{}) {
	cl.LogWithCorrelation(ctx, log.InfoLevel, component, operation, message, correlationID, jobID, context, nil, nil)
}

// LogError logs an error message with correlation
func (cl *CentralLogger) LogError(ctx context.Context, component, operation, message string, correlationID, jobID *string, context map[string]interface{}, err error) {
	cl.LogWithCorrelation(ctx, log.ErrorLevel, component, operation, message, correlationID, jobID, context, err, nil)
}

// LogWarning logs a warning message with correlation
func (cl *CentralLogger) LogWarning(ctx context.Context, component, operation, message string, correlationID, jobID *string, context map[string]interface{}, err error) {
	cl.LogWithCorrelation(ctx, log.WarnLevel, component, operation, message, correlationID, jobID, context, err, nil)
}

// LogDebug logs a debug message with correlation
func (cl *CentralLogger) LogDebug(ctx context.Context, component, operation, message string, correlationID, jobID *string, context map[string]interface{}) {
	cl.LogWithCorrelation(ctx, log.DebugLevel, component, operation, message, correlationID, jobID, context, nil, nil)
}

// LogOperationStart logs the start of an operation with timing
func (cl *CentralLogger) LogOperationStart(ctx context.Context, component, operation, message string, correlationID, jobID *string, context map[string]interface{}) *OperationTimer {
	cl.LogInfo(ctx, component, operation, fmt.Sprintf("Starting: %s", message), correlationID, jobID, context)

	return &OperationTimer{
		logger:        cl,
		component:     component,
		operation:     operation,
		message:       message,
		correlationID: correlationID,
		jobID:         jobID,
		context:       context,
		startTime:     time.Now(),
	}
}

// OperationTimer tracks operation duration
type OperationTimer struct {
	logger        *CentralLogger
	component     string
	operation     string
	message       string
	correlationID *string
	jobID         *string
	context       map[string]interface{}
	startTime     time.Time
}

// Complete logs the completion of an operation with duration
func (ot *OperationTimer) Complete(ctx context.Context, err error) {
	duration := time.Since(ot.startTime)

	if err != nil {
		ot.logger.LogWithCorrelation(
			ctx,
			log.ErrorLevel,
			ot.component,
			ot.operation,
			fmt.Sprintf("Failed: %s", ot.message),
			ot.correlationID,
			ot.jobID,
			ot.context,
			err,
			&duration,
		)
	} else {
		ot.logger.LogWithCorrelation(
			ctx,
			log.InfoLevel,
			ot.component,
			ot.operation,
			fmt.Sprintf("Completed: %s", ot.message),
			ot.correlationID,
			ot.jobID,
			ot.context,
			nil,
			&duration,
		)
	}
}

// Private methods

func (cl *CentralLogger) writeToLogFile(entry *LogEntry) {
	cl.mutex.Lock()
	defer cl.mutex.Unlock()

	// Convert to JSON
	jsonData, err := json.Marshal(entry)
	if err != nil {
		log.WithError(err).Error("Failed to marshal log entry to JSON")
		return
	}

	// Write to current log file
	logFile := cl.getCurrentLogFile()
	file, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.WithError(err).Error("Failed to open log file")
		return
	}
	defer file.Close()

	// Write JSON line
	if _, err := fmt.Fprintln(file, string(jsonData)); err != nil {
		log.WithError(err).Error("Failed to write to log file")
	}

	// Check if log rotation is needed
	if cl.shouldRotateLog(logFile) {
		cl.rotateLogFile()
	}
}

func (cl *CentralLogger) writeToJobExecutionLog(ctx context.Context, jobID string, entry *LogEntry) {
	if cl.jobTrackingService == nil {
		return
	}

	// Convert context to JSON for storage
	var details models.JSON
	if entry.Context != nil {
		details = models.JSON(entry.Context)
	}

	// Create execution log entry
	logEntry := &models.CloudStackJobExecutionLog{
		JobTrackingID:       jobID,
		LogLevel:            entry.Level,
		Message:             entry.Message,
		Details:             details,
		OperationPhase:      &entry.Operation,
		CloudStackJobStatus: nil, // Will be set by job poller
	}

	// Don't fail the main operation if database logging fails
	if err := cl.db.WithContext(ctx).Create(logEntry).Error; err != nil {
		log.WithError(err).Warn("Failed to write to job execution log")
	}
}

func (cl *CentralLogger) writeToStandardLog(entry *LogEntry, err error) {
	fields := log.Fields{
		"component":      entry.Component,
		"operation":      entry.Operation,
		"correlation_id": entry.CorrelationID,
		"job_id":         entry.JobID,
	}

	if entry.Context != nil {
		for k, v := range entry.Context {
			fields[k] = v
		}
	}

	if entry.Duration != nil {
		fields["duration_ms"] = entry.Duration.Milliseconds()
	}

	logEntry := log.WithFields(fields)

	switch entry.Level {
	case "debug":
		logEntry.Debug(entry.Message)
	case "info":
		logEntry.Info(entry.Message)
	case "warning", "warn":
		if err != nil {
			logEntry.WithError(err).Warn(entry.Message)
		} else {
			logEntry.Warn(entry.Message)
		}
	case "error":
		if err != nil {
			logEntry.WithError(err).Error(entry.Message)
		} else {
			logEntry.Error(entry.Message)
		}
	default:
		logEntry.Info(entry.Message)
	}
}

func (cl *CentralLogger) getCurrentLogFile() string {
	timestamp := time.Now().Format("2006-01-02")
	return filepath.Join(cl.logDir, fmt.Sprintf("migratekit-%s.log", timestamp))
}

func (cl *CentralLogger) setupLogFile() error {
	logFile := cl.getCurrentLogFile()

	// Create the file if it doesn't exist
	file, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	return nil
}

func (cl *CentralLogger) shouldRotateLog(logFile string) bool {
	stat, err := os.Stat(logFile)
	if err != nil {
		return false
	}

	// Rotate if file size exceeds limit
	if stat.Size() > cl.maxLogFileSize {
		return true
	}

	// Rotate if file is older than rotation hours
	if time.Since(stat.ModTime()) > time.Duration(cl.rotationHours)*time.Hour {
		return true
	}

	return false
}

func (cl *CentralLogger) rotateLogFile() {
	// Create timestamped backup of current log
	currentLog := cl.getCurrentLogFile()
	if _, err := os.Stat(currentLog); os.IsNotExist(err) {
		return
	}

	timestamp := time.Now().Format("2006-01-02-15-04-05")
	backupLog := filepath.Join(cl.logDir, fmt.Sprintf("migratekit-%s.log", timestamp))

	if err := os.Rename(currentLog, backupLog); err != nil {
		log.WithError(err).Error("Failed to rotate log file")
		return
	}

	// Clean up old log files
	cl.cleanupOldLogs()

	// Create new log file
	cl.setupLogFile()
}

func (cl *CentralLogger) cleanupOldLogs() {
	files, err := filepath.Glob(filepath.Join(cl.logDir, "migratekit-*.log"))
	if err != nil {
		return
	}

	if len(files) <= cl.maxLogFiles {
		return
	}

	// Sort files by modification time (oldest first)
	type fileInfo struct {
		path    string
		modTime time.Time
	}

	var fileInfos []fileInfo
	for _, file := range files {
		stat, err := os.Stat(file)
		if err != nil {
			continue
		}
		fileInfos = append(fileInfos, fileInfo{
			path:    file,
			modTime: stat.ModTime(),
		})
	}

	// Sort by modification time
	for i := 0; i < len(fileInfos)-1; i++ {
		for j := i + 1; j < len(fileInfos); j++ {
			if fileInfos[i].modTime.After(fileInfos[j].modTime) {
				fileInfos[i], fileInfos[j] = fileInfos[j], fileInfos[i]
			}
		}
	}

	// Remove oldest files
	filesToRemove := len(fileInfos) - cl.maxLogFiles
	for i := 0; i < filesToRemove; i++ {
		if err := os.Remove(fileInfos[i].path); err != nil {
			log.WithError(err).Warn("Failed to remove old log file")
		}
	}
}

func (cl *CentralLogger) logRotationWorker() {
	ticker := time.NewTicker(1 * time.Hour) // Check every hour
	defer ticker.Stop()

	for range ticker.C {
		cl.mutex.Lock()
		if cl.shouldRotateLog(cl.getCurrentLogFile()) {
			cl.rotateLogFile()
		}
		cl.mutex.Unlock()
	}
}

// GetLogEntries retrieves structured log entries from files
func (cl *CentralLogger) GetLogEntries(correlationID string, limit int, since *time.Time) ([]LogEntry, error) {
	if limit == 0 {
		limit = 100
	}

	var entries []LogEntry
	files, err := filepath.Glob(filepath.Join(cl.logDir, "migratekit-*.log"))
	if err != nil {
		return nil, fmt.Errorf("failed to list log files: %w", err)
	}

	for _, file := range files {
		fileEntries, err := cl.parseLogFile(file, correlationID, since)
		if err != nil {
			log.WithError(err).Warn("Failed to parse log file")
			continue
		}
		entries = append(entries, fileEntries...)
	}

	// Sort by timestamp (newest first) and limit results
	for i := 0; i < len(entries)-1; i++ {
		for j := i + 1; j < len(entries); j++ {
			if entries[i].Timestamp.Before(entries[j].Timestamp) {
				entries[i], entries[j] = entries[j], entries[i]
			}
		}
	}

	if len(entries) > limit {
		entries = entries[:limit]
	}

	return entries, nil
}

func (cl *CentralLogger) parseLogFile(filename, correlationID string, since *time.Time) ([]LogEntry, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var entries []LogEntry
	decoder := json.NewDecoder(file)

	for decoder.More() {
		var entry LogEntry
		if err := decoder.Decode(&entry); err != nil {
			continue // Skip malformed entries
		}

		// Filter by correlation ID if specified
		if correlationID != "" && entry.CorrelationID != correlationID {
			continue
		}

		// Filter by time if specified
		if since != nil && entry.Timestamp.Before(*since) {
			continue
		}

		entries = append(entries, entry)
	}

	return entries, nil
}



