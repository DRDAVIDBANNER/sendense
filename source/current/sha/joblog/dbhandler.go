package joblog

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// DBHandler implements slog.Handler and writes log records to the database asynchronously
type DBHandler struct {
	db          *sql.DB
	level       slog.Level
	attrs       []slog.Attr
	groups      []string
	ch          chan *LogRecord
	stopCh      chan struct{}
	stopped     bool
	mu          sync.RWMutex
	wg          sync.WaitGroup
	queueSize   int
	dropOldest  bool
}

// DBHandlerConfig provides configuration options for the DBHandler
type DBHandlerConfig struct {
	// QueueSize is the size of the internal buffer for log records
	QueueSize int
	
	// Level is the minimum log level to handle
	Level slog.Level
	
	// DropOldest determines behavior when queue is full
	// If true, drop oldest records; if false, block until space is available
	DropOldest bool
	
	// WriterCount is the number of concurrent database writers
	WriterCount int
}

// DefaultDBHandlerConfig returns sensible default configuration
func DefaultDBHandlerConfig() *DBHandlerConfig {
	return &DBHandlerConfig{
		QueueSize:   10000,
		Level:       slog.LevelInfo,
		DropOldest:  true,
		WriterCount: 2,
	}
}

// NewDBHandler creates a new database handler for structured logging
func NewDBHandler(db *sql.DB, config *DBHandlerConfig) *DBHandler {
	if config == nil {
		config = DefaultDBHandlerConfig()
	}
	
	if config.WriterCount < 1 {
		config.WriterCount = 1
	}
	
	handler := &DBHandler{
		db:         db,
		level:      config.Level,
		ch:         make(chan *LogRecord, config.QueueSize),
		stopCh:     make(chan struct{}),
		queueSize:  config.QueueSize,
		dropOldest: config.DropOldest,
	}
	
	// Start background writers
	for i := 0; i < config.WriterCount; i++ {
		handler.wg.Add(1)
		go handler.writer()
	}
	
	return handler
}

// Enabled reports whether the handler handles records at the given level
func (h *DBHandler) Enabled(ctx context.Context, level slog.Level) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return level >= h.level && !h.stopped
}

// Handle processes a log record
func (h *DBHandler) Handle(ctx context.Context, record slog.Record) error {
	h.mu.RLock()
	if h.stopped {
		h.mu.RUnlock()
		return nil
	}
	h.mu.RUnlock()
	
	// Extract job, step, and external job IDs from context
	jobID, _ := JobIDFromCtx(ctx)
	stepID, hasStepID := StepIDFromCtx(ctx)
	externalJobID, hasExternalJobID := ExternalJobIDFromCtx(ctx)
	
	// Convert attributes to JSON
	attrs := make(map[string]any)
	
	// Add handler attributes
	for _, attr := range h.attrs {
		attrs[attr.Key] = attr.Value.Any()
	}
	
	// Add record attributes
	record.Attrs(func(a slog.Attr) bool {
		attrs[a.Key] = a.Value.Any()
		return true
	})
	
	// Marshal attributes to JSON
	var attrsJSON *string
	if len(attrs) > 0 {
		if jsonBytes, err := json.Marshal(attrs); err == nil {
			jsonStr := string(jsonBytes)
			attrsJSON = &jsonStr
		}
	}
	
	// Create log record
	logRecord := &LogRecord{
		JobID:   stringPtr(jobID),
		Level:   levelToString(record.Level),
		Message: record.Message,
		Attrs:   attrsJSON,
		Ts:      record.Time,
	}
	
	if hasStepID {
		logRecord.StepID = &stepID
	}
	
	if hasExternalJobID {
		logRecord.ExternalJobID = &externalJobID
	}
	
	// Try to enqueue the record
	select {
	case h.ch <- logRecord:
		// Successfully enqueued
	default:
		// Queue is full
		if h.dropOldest {
			// Drop oldest record and add new one
			select {
			case <-h.ch: // Remove oldest
			default:
			}
			select {
			case h.ch <- logRecord:
			default:
				// Still full, drop this record
			}
		} else {
			// Block until space is available
			select {
			case h.ch <- logRecord:
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}
	
	return nil
}

// WithAttrs returns a new handler with the given attributes added
func (h *DBHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	h.mu.RLock()
	newAttrs := make([]slog.Attr, len(h.attrs)+len(attrs))
	copy(newAttrs, h.attrs)
	copy(newAttrs[len(h.attrs):], attrs)
	newGroups := make([]string, len(h.groups))
	copy(newGroups, h.groups)
	h.mu.RUnlock()
	
	return &DBHandler{
		db:         h.db,
		level:      h.level,
		attrs:      newAttrs,
		groups:     newGroups,
		ch:         h.ch,
		stopCh:     h.stopCh,
		queueSize:  h.queueSize,
		dropOldest: h.dropOldest,
	}
}

// WithGroup returns a new handler with the given group added
func (h *DBHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	
	h.mu.RLock()
	newGroups := make([]string, len(h.groups)+1)
	copy(newGroups, h.groups)
	newGroups[len(h.groups)] = name
	newAttrs := make([]slog.Attr, len(h.attrs))
	copy(newAttrs, h.attrs)
	h.mu.RUnlock()
	
	return &DBHandler{
		db:         h.db,
		level:      h.level,
		attrs:      newAttrs,
		groups:     newGroups,
		ch:         h.ch,
		stopCh:     h.stopCh,
		queueSize:  h.queueSize,
		dropOldest: h.dropOldest,
	}
}

// Close stops the handler and waits for all pending writes to complete
func (h *DBHandler) Close() error {
	h.mu.Lock()
	if h.stopped {
		h.mu.Unlock()
		return nil
	}
	h.stopped = true
	h.mu.Unlock()
	
	close(h.stopCh)
	close(h.ch)
	h.wg.Wait()
	return nil
}

// writer runs in a goroutine and writes log records to the database
func (h *DBHandler) writer() {
	defer h.wg.Done()
	
	// Prepare the SQL statement
	stmt, err := h.db.Prepare(`
		INSERT INTO log_events (job_id, step_id, level, message, attrs, ts, external_job_id)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		// If we can't prepare the statement, we can't write logs
		// Log to stderr as a fallback
		fmt.Printf("ERROR: Failed to prepare log insert statement: %v\n", err)
		return
	}
	defer stmt.Close()
	
	// Batch size for bulk inserts
	const batchSize = 100
	batch := make([]*LogRecord, 0, batchSize)
	ticker := time.NewTicker(1 * time.Second) // Flush every second
	defer ticker.Stop()
	
	for {
		select {
		case record, ok := <-h.ch:
			if !ok {
				// Channel closed, flush remaining records and exit
				if len(batch) > 0 {
					h.writeBatch(stmt, batch)
				}
				return
			}
			
			batch = append(batch, record)
			
			// Write batch if it's full
			if len(batch) >= batchSize {
				h.writeBatch(stmt, batch)
				batch = batch[:0] // Reset slice but keep capacity
			}
			
		case <-ticker.C:
			// Periodic flush
			if len(batch) > 0 {
				h.writeBatch(stmt, batch)
				batch = batch[:0]
			}
			
		case <-h.stopCh:
			// Shutdown signal, flush remaining records and exit
			if len(batch) > 0 {
				h.writeBatch(stmt, batch)
			}
			return
		}
	}
}

// writeBatch writes a batch of log records to the database
func (h *DBHandler) writeBatch(stmt *sql.Stmt, batch []*LogRecord) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	// Start transaction for batch insert
	tx, err := h.db.BeginTx(ctx, nil)
	if err != nil {
		fmt.Printf("ERROR: Failed to start transaction for log batch: %v\n", err)
		return
	}
	defer tx.Rollback()
	
	txStmt := tx.StmtContext(ctx, stmt)
	
	// Insert all records in the batch
	for _, record := range batch {
		_, err := txStmt.ExecContext(ctx,
			record.JobID,
			record.StepID,
			record.Level,
			record.Message,
			record.Attrs,
			record.Ts,
			record.ExternalJobID,
		)
		if err != nil {
			fmt.Printf("ERROR: Failed to insert log record: %v\n", err)
			// Continue with other records in the batch
		}
	}
	
	// Commit the transaction
	if err := tx.Commit(); err != nil {
		fmt.Printf("ERROR: Failed to commit log batch transaction: %v\n", err)
	}
}

// GetQueueSize returns the current size of the log queue
func (h *DBHandler) GetQueueSize() int {
	return len(h.ch)
}

// GetQueueCapacity returns the maximum capacity of the log queue
func (h *DBHandler) GetQueueCapacity() int {
	return h.queueSize
}

// IsQueueFull returns true if the log queue is at capacity
func (h *DBHandler) IsQueueFull() bool {
	return len(h.ch) >= h.queueSize
}

// Helper functions

func stringPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func levelToString(level slog.Level) string {
	switch level {
	case slog.LevelDebug:
		return "DEBUG"
	case slog.LevelInfo:
		return "INFO"
	case slog.LevelWarn:
		return "WARN"
	case slog.LevelError:
		return "ERROR"
	default:
		// Handle custom levels
		if level < slog.LevelInfo {
			return "DEBUG"
		} else if level < slog.LevelWarn {
			return "INFO"
		} else if level < slog.LevelError {
			return "WARN"
		} else {
			return "ERROR"
		}
	}
}

// FanoutHandler combines multiple handlers into one
type FanoutHandler struct {
	handlers []slog.Handler
}

// NewFanoutHandler creates a handler that sends records to multiple handlers
func NewFanoutHandler(handlers ...slog.Handler) *FanoutHandler {
	return &FanoutHandler{
		handlers: handlers,
	}
}

// Enabled reports whether any of the handlers handle records at the given level
func (f *FanoutHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, h := range f.handlers {
		if h.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

// Handle sends the record to all handlers
func (f *FanoutHandler) Handle(ctx context.Context, record slog.Record) error {
	var errs []error
	for _, h := range f.handlers {
		if h.Enabled(ctx, record.Level) {
			if err := h.Handle(ctx, record); err != nil {
				errs = append(errs, err)
			}
		}
	}
	
	if len(errs) > 0 {
		// Return first error, but all handlers were attempted
		return errs[0]
	}
	return nil
}

// WithAttrs returns a new fanout handler with the given attributes added to all handlers
func (f *FanoutHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	newHandlers := make([]slog.Handler, len(f.handlers))
	for i, h := range f.handlers {
		newHandlers[i] = h.WithAttrs(attrs)
	}
	return &FanoutHandler{handlers: newHandlers}
}

// WithGroup returns a new fanout handler with the given group added to all handlers
func (f *FanoutHandler) WithGroup(name string) slog.Handler {
	newHandlers := make([]slog.Handler, len(f.handlers))
	for i, h := range f.handlers {
		newHandlers[i] = h.WithGroup(name)
	}
	return &FanoutHandler{handlers: newHandlers}
}
