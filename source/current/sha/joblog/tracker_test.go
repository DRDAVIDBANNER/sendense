package joblog

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJobLifecycle(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// Create tracker with text handler for tests
	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug})
	tracker := New(db, handler)
	defer tracker.Close()

	ctx := context.Background()

	// Test job creation
	jobStart := JobStart{
		JobType:   "test",
		Operation: "test-operation",
		Owner:     stringPtr("test-user"),
		Metadata: map[string]any{
			"test_key": "test_value",
		},
	}

	// Mock job creation
	mock.ExpectExec("INSERT INTO job_tracking").
		WithArgs(sqlmock.AnyArg(), nil, "test", "test-operation", StatusRunning, sqlmock.AnyArg(), "test-user", sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	ctx, jobID, err := tracker.StartJob(ctx, jobStart)
	require.NoError(t, err)
	assert.NotEmpty(t, jobID)

	// Test job completion
	mock.ExpectExec("UPDATE job_tracking").
		WithArgs(StatusCompleted, sqlmock.AnyArg(), nil, nil, sqlmock.AnyArg(), jobID).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = tracker.EndJob(ctx, jobID, StatusCompleted, nil)
	require.NoError(t, err)

	// Verify all expectations were met
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestStepLifecycle(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug})
	tracker := New(db, handler)
	defer tracker.Close()

	ctx := context.Background()
	jobID := "test-job-123"

	// Mock next sequence query
	mock.ExpectQuery("SELECT COALESCE\\(MAX\\(seq\\), 0\\) \\+ 1").
		WithArgs(jobID).
		WillReturnRows(sqlmock.NewRows([]string{"seq"}).AddRow(1))

	// Mock step creation
	mock.ExpectExec("INSERT INTO job_steps").
		WithArgs(jobID, "test-step", 1, StatusRunning, sqlmock.AnyArg(), nil).
		WillReturnResult(sqlmock.NewResult(123, 1))

	stepStart := StepStart{
		Name: "test-step",
	}

	stepCtx, stepID, err := tracker.StartStep(ctx, jobID, stepStart)
	require.NoError(t, err)
	assert.Equal(t, int64(123), stepID)

	// Verify step ID is in context
	ctxStepID, ok := StepIDFromCtx(stepCtx)
	assert.True(t, ok)
	assert.Equal(t, int64(123), ctxStepID)

	// Test step completion
	mock.ExpectExec("UPDATE job_steps").
		WithArgs(StatusCompleted, sqlmock.AnyArg(), nil, stepID).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// Mock query for step details (for logging)
	mock.ExpectQuery("SELECT job_id, name FROM job_steps").
		WithArgs(stepID).
		WillReturnRows(sqlmock.NewRows([]string{"job_id", "name"}).AddRow(jobID, "test-step"))

	err = tracker.EndStep(stepID, StatusCompleted, nil)
	require.NoError(t, err)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRunStep(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug})
	tracker := New(db, handler)
	defer tracker.Close()

	ctx := context.Background()
	jobID := "test-job-456"

	// Mock step creation
	mock.ExpectQuery("SELECT COALESCE\\(MAX\\(seq\\), 0\\) \\+ 1").
		WithArgs(jobID).
		WillReturnRows(sqlmock.NewRows([]string{"seq"}).AddRow(1))

	mock.ExpectExec("INSERT INTO job_steps").
		WithArgs(jobID, "auto-step", 1, StatusRunning, sqlmock.AnyArg(), nil).
		WillReturnResult(sqlmock.NewResult(456, 1))

	// Mock step completion
	mock.ExpectExec("UPDATE job_steps").
		WithArgs(StatusCompleted, sqlmock.AnyArg(), nil, int64(456)).
		WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectQuery("SELECT job_id, name FROM job_steps").
		WithArgs(int64(456)).
		WillReturnRows(sqlmock.NewRows([]string{"job_id", "name"}).AddRow(jobID, "auto-step"))

	// Test successful step execution
	executed := false
	err = tracker.RunStep(ctx, jobID, "auto-step", func(stepCtx context.Context) error {
		executed = true

		// Verify context has step ID
		stepID, ok := StepIDFromCtx(stepCtx)
		assert.True(t, ok)
		assert.Equal(t, int64(456), stepID)

		return nil
	})

	require.NoError(t, err)
	assert.True(t, executed)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRunStepWithError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug})
	tracker := New(db, handler)
	defer tracker.Close()

	ctx := context.Background()
	jobID := "test-job-789"

	// Mock step creation
	mock.ExpectQuery("SELECT COALESCE\\(MAX\\(seq\\), 0\\) \\+ 1").
		WithArgs(jobID).
		WillReturnRows(sqlmock.NewRows([]string{"seq"}).AddRow(1))

	mock.ExpectExec("INSERT INTO job_steps").
		WithArgs(jobID, "failing-step", 1, StatusRunning, sqlmock.AnyArg(), nil).
		WillReturnResult(sqlmock.NewResult(789, 1))

	// Mock step failure
	mock.ExpectExec("UPDATE job_steps").
		WithArgs(StatusFailed, sqlmock.AnyArg(), sqlmock.AnyArg(), int64(789)).
		WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectQuery("SELECT job_id, name FROM job_steps").
		WithArgs(int64(789)).
		WillReturnRows(sqlmock.NewRows([]string{"job_id", "name"}).AddRow(jobID, "failing-step"))

	// Test failing step execution
	testError := fmt.Errorf("step failed")
	err = tracker.RunStep(ctx, jobID, "failing-step", func(stepCtx context.Context) error {
		return testError
	})

	require.Error(t, err)
	assert.Equal(t, testError, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRunStepWithPanic(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug})
	tracker := New(db, handler)
	defer tracker.Close()

	ctx := context.Background()
	jobID := "test-job-panic"

	// Mock step creation
	mock.ExpectQuery("SELECT COALESCE\\(MAX\\(seq\\), 0\\) \\+ 1").
		WithArgs(jobID).
		WillReturnRows(sqlmock.NewRows([]string{"seq"}).AddRow(1))

	mock.ExpectExec("INSERT INTO job_steps").
		WithArgs(jobID, "panic-step", 1, StatusRunning, sqlmock.AnyArg(), nil).
		WillReturnResult(sqlmock.NewResult(999, 1))

	// Mock step failure due to panic
	mock.ExpectExec("UPDATE job_steps").
		WithArgs(StatusFailed, sqlmock.AnyArg(), sqlmock.AnyArg(), int64(999)).
		WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectQuery("SELECT job_id, name FROM job_steps").
		WithArgs(int64(999)).
		WillReturnRows(sqlmock.NewRows([]string{"job_id", "name"}).AddRow(jobID, "panic-step"))

	// Test panic recovery
	err = tracker.RunStep(ctx, jobID, "panic-step", func(stepCtx context.Context) error {
		panic("test panic")
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "panic")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestJobProgress(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug})
	tracker := New(db, handler)
	defer tracker.Close()

	ctx := context.Background()
	jobID := "progress-job-123"

	// Test progress update
	mock.ExpectExec("UPDATE job_tracking").
		WithArgs(uint8(50), sqlmock.AnyArg(), jobID).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = tracker.MarkJobProgress(ctx, jobID, 50)
	require.NoError(t, err)

	// Test invalid progress
	err = tracker.MarkJobProgress(ctx, jobID, 150)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid percentage")

	// Test get progress
	now := time.Now()
	mock.ExpectQuery("SELECT").
		WithArgs(jobID).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "total_steps", "completed_steps", "failed_steps", "running_steps", "skipped_steps",
			"percent_complete", "started_at", "last_activity", "runtime_seconds",
		}).AddRow(
			jobID, 4, 2, 0, 1, 0, 50, now, now, 120,
		))

	progress, err := tracker.GetJobProgress(ctx, jobID)
	require.NoError(t, err)
	assert.Equal(t, jobID, progress.JobID)
	assert.Equal(t, 4, progress.TotalSteps)
	assert.Equal(t, 2, progress.CompletedSteps)
	assert.Equal(t, uint8(50), *progress.ManualCompletion)
	assert.Equal(t, float64(50), progress.StepCompletion) // 2/4 * 100

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestContextPropagation(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug})
	tracker := New(db, handler)
	defer tracker.Close()

	ctx := context.Background()

	// Test job ID context
	jobID := "context-job-123"
	ctxWithJob := WithJobID(ctx, jobID)

	retrievedJobID, ok := JobIDFromCtx(ctxWithJob)
	assert.True(t, ok)
	assert.Equal(t, jobID, retrievedJobID)

	// Test step ID context
	stepID := int64(456)
	ctxWithStep := WithStepID(ctxWithJob, stepID)

	retrievedStepID, ok := StepIDFromCtx(ctxWithStep)
	assert.True(t, ok)
	assert.Equal(t, stepID, retrievedStepID)

	// Test logger context
	logger := tracker.Logger(ctxWithStep)
	assert.NotNil(t, logger)

	// Test missing context
	_, ok = JobIDFromCtx(ctx)
	assert.False(t, ok)

	_, ok = StepIDFromCtx(ctx)
	assert.False(t, ok)
}

func TestJobValidation(t *testing.T) {
	// Test valid job start
	validJob := JobStart{
		JobType:   "test",
		Operation: "test-op",
	}
	assert.NoError(t, validJob.Validate())

	// Test invalid job type
	invalidJobType := JobStart{
		JobType:   "",
		Operation: "test-op",
	}
	assert.Error(t, invalidJobType.Validate())

	// Test invalid operation
	invalidOperation := JobStart{
		JobType:   "test",
		Operation: "",
	}
	assert.Error(t, invalidOperation.Validate())
}

func TestStepValidation(t *testing.T) {
	// Test valid step start
	validStep := StepStart{
		Name: "test-step",
	}
	assert.NoError(t, validStep.Validate())

	// Test invalid step name
	invalidStep := StepStart{
		Name: "",
	}
	assert.Error(t, invalidStep.Validate())
}

func TestStatusMethods(t *testing.T) {
	// Test terminal statuses
	assert.True(t, StatusCompleted.IsTerminal())
	assert.True(t, StatusFailed.IsTerminal())
	assert.True(t, StatusCancelled.IsTerminal())
	assert.False(t, StatusPending.IsTerminal())
	assert.False(t, StatusRunning.IsTerminal())

	// Test active statuses
	assert.True(t, StatusPending.IsActive())
	assert.True(t, StatusRunning.IsActive())
	assert.False(t, StatusCompleted.IsActive())
	assert.False(t, StatusFailed.IsActive())
	assert.False(t, StatusCancelled.IsActive())

	// Test string conversion
	assert.Equal(t, "completed", StatusCompleted.String())
	assert.Equal(t, "failed", StatusFailed.String())
}

func TestMetadataSerialization(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug})
	tracker := New(db, handler)
	defer tracker.Close()

	ctx := context.Background()

	// Complex metadata
	metadata := map[string]any{
		"vm_config": map[string]any{
			"cpu_count": 4,
			"memory_gb": 16,
			"networks":  []string{"vlan100", "vlan200"},
		},
		"timestamps": map[string]any{
			"started_at": time.Now().Format(time.RFC3339),
		},
		"flags": []bool{true, false, true},
	}

	jobStart := JobStart{
		JobType:   "test",
		Operation: "metadata-test",
		Metadata:  metadata,
	}

	// Mock job creation with metadata validation
	mock.ExpectExec("INSERT INTO job_tracking").
		WithArgs(
			sqlmock.AnyArg(), // job ID
			nil,              // parent job ID
			"test",           // job type
			"metadata-test",  // operation
			StatusRunning,    // status
			&anyJSONMatcher{}, // metadata as JSON
			nil,              // owner
			sqlmock.AnyArg(), // started_at
			sqlmock.AnyArg(), // created_at
			sqlmock.AnyArg(), // updated_at
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	_, _, err = tracker.StartJob(ctx, jobStart)
	require.NoError(t, err)

	assert.NoError(t, mock.ExpectationsWereMet())
}

// Custom matcher for JSON fields
type anyJSONMatcher struct{}

func (a *anyJSONMatcher) Match(v driver.Value) bool {
	if v == nil {
		return false
	}
	
	str, ok := v.(string)
	if !ok {
		return false
	}
	
	// Check if it's valid JSON
	var js json.RawMessage
	return json.Unmarshal([]byte(str), &js) == nil
}

func (a *anyJSONMatcher) String() string {
	return "any valid JSON"
}

func TestHierarchicalJobs(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug})
	tracker := New(db, handler)
	defer tracker.Close()

	ctx := context.Background()

	// Create parent job
	parentStart := JobStart{
		JobType:   "orchestration",
		Operation: "bulk-operation",
	}

	mock.ExpectExec("INSERT INTO job_tracking").
		WithArgs(sqlmock.AnyArg(), nil, "orchestration", "bulk-operation", StatusRunning, nil, nil, sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	parentCtx, parentJobID, err := tracker.StartJob(ctx, parentStart)
	require.NoError(t, err)

	// Create child job
	childStart := JobStart{
		ParentJobID: &parentJobID,
		JobType:     "processing",
		Operation:   "child-operation",
	}

	mock.ExpectExec("INSERT INTO job_tracking").
		WithArgs(sqlmock.AnyArg(), &parentJobID, "processing", "child-operation", StatusRunning, nil, nil, sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(2, 1))

	_, childJobID, err := tracker.StartJob(parentCtx, childStart)
	require.NoError(t, err)

	assert.NotEqual(t, parentJobID, childJobID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetJob(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug})
	tracker := New(db, handler)
	defer tracker.Close()

	ctx := context.Background()
	jobID := "get-job-test"

	// Mock successful job retrieval
	now := time.Now()
	mock.ExpectQuery("SELECT id, parent_job_id, job_type").
		WithArgs(jobID).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "parent_job_id", "job_type", "operation", "status", "percent_complete",
			"cloudstack_job_id", "external_job_id", "metadata", "error_message", "owner",
			"started_at", "completed_at", "canceled_at", "created_at", "updated_at",
		}).AddRow(
			jobID, nil, "test", "test-operation", StatusCompleted, uint8(100),
			nil, nil, nil, nil, "test-user",
			now, &now, nil, now, now,
		))

	job, err := tracker.GetJob(ctx, jobID)
	require.NoError(t, err)
	assert.Equal(t, jobID, job.ID)
	assert.Equal(t, "test", job.JobType)
	assert.Equal(t, "test-operation", job.Operation)
	assert.Equal(t, StatusCompleted, job.Status)

	// Test job not found
	mock.ExpectQuery("SELECT id, parent_job_id, job_type").
		WithArgs("nonexistent").
		WillReturnError(sql.ErrNoRows)

	_, err = tracker.GetJob(ctx, "nonexistent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "job not found")

	assert.NoError(t, mock.ExpectationsWereMet())
}

// Helper function (using the one from dbhandler.go to avoid duplication)

// Benchmark tests
func BenchmarkJobCreation(b *testing.B) {
	db, mock, err := sqlmock.New()
	require.NoError(b, err)
	defer db.Close()

	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError})
	tracker := New(db, handler)
	defer tracker.Close()

	// Set up mock expectations for all iterations
	for i := 0; i < b.N; i++ {
		mock.ExpectExec("INSERT INTO job_tracking").
			WillReturnResult(sqlmock.NewResult(int64(i+1), 1))
	}

	ctx := context.Background()
	jobStart := JobStart{
		JobType:   "benchmark",
		Operation: "performance-test",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, err := tracker.StartJob(ctx, jobStart)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkStepExecution(b *testing.B) {
	db, mock, err := sqlmock.New()
	require.NoError(b, err)
	defer db.Close()

	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError})
	tracker := New(db, handler)
	defer tracker.Close()

	// Set up mock expectations for all iterations
	for i := 0; i < b.N; i++ {
		mock.ExpectQuery("SELECT COALESCE\\(MAX\\(seq\\), 0\\) \\+ 1").
			WillReturnRows(sqlmock.NewRows([]string{"seq"}).AddRow(1))
		mock.ExpectExec("INSERT INTO job_steps").
			WillReturnResult(sqlmock.NewResult(int64(i+1), 1))
		mock.ExpectExec("UPDATE job_steps").
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectQuery("SELECT job_id, name FROM job_steps").
			WillReturnRows(sqlmock.NewRows([]string{"job_id", "name"}).AddRow("test-job", "bench-step"))
	}

	ctx := context.Background()
	jobID := "benchmark-job"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := tracker.RunStep(ctx, jobID, "bench-step", func(stepCtx context.Context) error {
			return nil
		})
		if err != nil {
			b.Fatal(err)
		}
	}
}
