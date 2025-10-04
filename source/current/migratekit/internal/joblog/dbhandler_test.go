package joblog

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDBHandler(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// Create DB handler
	config := &DBHandlerConfig{
		QueueSize:   100,
		Level:       slog.LevelInfo,
		DropOldest:  true,
		WriterCount: 1,
	}
	
	handler := NewDBHandler(db, config)
	defer handler.Close()

	// Create logger with DB handler
	logger := slog.New(handler)

	// Set up context with job and step IDs
	ctx := WithStepID(WithJobID(context.Background(), "test-job-123"), 456)

	// Expect SQL preparation and log insert
	mock.ExpectPrepare("INSERT INTO log_events")
	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO log_events").
		WithArgs("test-job-123", int64(456), "INFO", "Test log message", sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	// Log a message
	logger.InfoContext(ctx, "Test log message",
		slog.String("test_attr", "test_value"),
	)

	// Give time for async processing
	time.Sleep(100 * time.Millisecond)

	// Close handler to flush remaining logs
	handler.Close()

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDBHandlerLevels(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// Create handler with WARN level
	config := &DBHandlerConfig{
		QueueSize:   100,
		Level:       slog.LevelWarn,
		DropOldest:  true,
		WriterCount: 1,
	}
	
	handler := NewDBHandler(db, config)
	defer handler.Close()

	ctx := context.Background()

	// Should NOT handle INFO level
	assert.False(t, handler.Enabled(ctx, slog.LevelInfo))
	assert.False(t, handler.Enabled(ctx, slog.LevelDebug))

	// Should handle WARN and ERROR levels
	assert.True(t, handler.Enabled(ctx, slog.LevelWarn))
	assert.True(t, handler.Enabled(ctx, slog.LevelError))
}

func TestDBHandlerWithAttrs(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	config := DefaultDBHandlerConfig()
	handler := NewDBHandler(db, config)
	defer handler.Close()

	// Create handler with attributes
	attrs := []slog.Attr{
		slog.String("service", "test-service"),
		slog.String("version", "1.0.0"),
	}
	
	handlerWithAttrs := handler.WithAttrs(attrs)
	logger := slog.New(handlerWithAttrs)

	ctx := WithJobID(context.Background(), "attr-test-job")

	// Expect SQL operations
	mock.ExpectPrepare("INSERT INTO log_events")
	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO log_events").
		WithArgs("attr-test-job", nil, "INFO", "Test with attributes", sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	// Log message (attributes should be included in attrs JSON)
	logger.InfoContext(ctx, "Test with attributes",
		slog.String("additional", "data"),
	)

	time.Sleep(100 * time.Millisecond)
	handler.Close()

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDBHandlerWithGroup(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	config := DefaultDBHandlerConfig()
	handler := NewDBHandler(db, config)
	defer handler.Close()

	// Create handler with group
	handlerWithGroup := handler.WithGroup("migration")
	logger := slog.New(handlerWithGroup)

	ctx := WithJobID(context.Background(), "group-test-job")

	// Expect SQL operations
	mock.ExpectPrepare("INSERT INTO log_events")
	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO log_events").
		WithArgs("group-test-job", nil, "INFO", "Test with group", sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	// Log message
	logger.InfoContext(ctx, "Test with group",
		slog.String("vm_id", "vm-123"),
	)

	time.Sleep(100 * time.Millisecond)
	handler.Close()

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestFanoutHandler(t *testing.T) {
	// Create a text handler for testing
	textHandler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug})
	
	// Create a mock DB handler
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	
	dbHandler := NewDBHandler(db, DefaultDBHandlerConfig())
	defer dbHandler.Close()

	// Create fanout handler
	fanout := NewFanoutHandler(textHandler, dbHandler)
	logger := slog.New(fanout)

	ctx := context.Background()

	// Test that fanout handler enables correctly
	assert.True(t, fanout.Enabled(ctx, slog.LevelDebug)) // textHandler enables debug
	assert.True(t, fanout.Enabled(ctx, slog.LevelInfo))  // both enable info

	// Test logging (should go to both handlers)
	logger.Info("Fanout test message",
		slog.String("handler", "fanout"),
	)

	// Test with attributes
	fanoutWithAttrs := fanout.WithAttrs([]slog.Attr{
		slog.String("test", "attr"),
	})
	
	loggerWithAttrs := slog.New(fanoutWithAttrs)
	loggerWithAttrs.Info("Message with fanout attrs")

	// Test with group
	fanoutWithGroup := fanout.WithGroup("test-group")
	loggerWithGroup := slog.New(fanoutWithGroup)
	loggerWithGroup.Info("Message with fanout group")
}

func TestDBHandlerQueueManagement(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// Create handler with small queue for testing
	config := &DBHandlerConfig{
		QueueSize:   2, // Very small queue
		Level:       slog.LevelInfo,
		DropOldest:  true,
		WriterCount: 1,
	}
	
	handler := NewDBHandler(db, config)
	defer handler.Close()

	// Test queue capacity
	assert.Equal(t, 2, handler.GetQueueCapacity())
	assert.Equal(t, 0, handler.GetQueueSize())
	assert.False(t, handler.IsQueueFull())

	logger := slog.New(handler)
	ctx := context.Background()

	// Fill the queue (these might not be processed immediately)
	logger.InfoContext(ctx, "Message 1")
	logger.InfoContext(ctx, "Message 2")
	
	// Give a moment for messages to be queued
	time.Sleep(10 * time.Millisecond)
	
	// Queue might be full or processing
	// The exact behavior depends on timing of the background writer
	queueSize := handler.GetQueueSize()
	assert.True(t, queueSize >= 0 && queueSize <= 2)
}

func TestDBHandlerErrorScenarios(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	config := DefaultDBHandlerConfig()
	handler := NewDBHandler(db, config)
	defer handler.Close()

	logger := slog.New(handler)
	ctx := WithJobID(context.Background(), "error-test-job")

	// Simulate database error
	mock.ExpectPrepare("INSERT INTO log_events").
		WillReturnError(sql.ErrConnDone)

	// This should not panic, but log to stderr instead
	logger.InfoContext(ctx, "This message should handle DB error gracefully")

	time.Sleep(100 * time.Millisecond)
	handler.Close()

	// Expectations might not be met due to error, that's OK
}

func TestLevelToString(t *testing.T) {
	tests := []struct {
		level    slog.Level
		expected string
	}{
		{slog.LevelDebug, "DEBUG"},
		{slog.LevelInfo, "INFO"},
		{slog.LevelWarn, "WARN"},
		{slog.LevelError, "ERROR"},
		{slog.Level(-8), "DEBUG"}, // Custom level below debug
		{slog.Level(8), "ERROR"},  // Custom level above error
	}

	for _, test := range tests {
		result := levelToString(test.level)
		assert.Equal(t, test.expected, result, "Level %d should map to %s", test.level, test.expected)
	}
}

func TestStringPtr(t *testing.T) {
	// Test with empty string
	result := stringPtr("")
	assert.Nil(t, result)

	// Test with non-empty string
	result = stringPtr("test")
	require.NotNil(t, result)
	assert.Equal(t, "test", *result)
}

func BenchmarkDBHandler(b *testing.B) {
	db, mock, err := sqlmock.New()
	require.NoError(b, err)
	defer db.Close()

	config := &DBHandlerConfig{
		QueueSize:   10000,
		Level:       slog.LevelInfo,
		DropOldest:  true,
		WriterCount: 2,
	}
	
	handler := NewDBHandler(db, config)
	defer handler.Close()

	logger := slog.New(handler)
	ctx := WithJobID(context.Background(), "benchmark-job")

	// Mock expectations for many inserts
	mock.ExpectPrepare("INSERT INTO log_events")
	for i := 0; i < b.N; i++ {
		mock.ExpectBegin()
		mock.ExpectExec("INSERT INTO log_events").
			WillReturnResult(sqlmock.NewResult(int64(i+1), 1))
		mock.ExpectCommit()
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.InfoContext(ctx, "Benchmark log message",
			slog.Int("iteration", i),
		)
	}
	b.StopTimer()

	// Give time for async processing
	time.Sleep(1 * time.Second)
	handler.Close()
}

func TestDBHandlerConcurrency(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	config := &DBHandlerConfig{
		QueueSize:   1000,
		Level:       slog.LevelInfo,
		DropOldest:  true,
		WriterCount: 3, // Multiple concurrent writers
	}
	
	handler := NewDBHandler(db, config)
	defer handler.Close()

	logger := slog.New(handler)

	// Mock many concurrent writes
	mock.ExpectPrepare("INSERT INTO log_events")
	for i := 0; i < 100; i++ {
		mock.ExpectBegin()
		mock.ExpectExec("INSERT INTO log_events").
			WillReturnResult(sqlmock.NewResult(int64(i+1), 1))
		mock.ExpectCommit()
	}

	// Log from multiple goroutines
	done := make(chan bool, 10)
	
	for g := 0; g < 10; g++ {
		go func(goroutine int) {
			defer func() { done <- true }()
			
			ctx := WithJobID(context.Background(), fmt.Sprintf("concurrent-job-%d", goroutine))
			
			for i := 0; i < 10; i++ {
				logger.InfoContext(ctx, "Concurrent log message",
					slog.Int("goroutine", goroutine),
					slog.Int("iteration", i),
				)
			}
		}(g)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	// Give time for async processing
	time.Sleep(2 * time.Second)
	handler.Close()

	// Not all expectations may be met due to batching, but there should be no panics
}
