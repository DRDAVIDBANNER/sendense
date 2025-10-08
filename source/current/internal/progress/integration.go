package progress

import (
	"fmt"
	"log"
	"os"
	"time"

	"libguestfs.org/libnbd"
)

// LibNBDProgressWrapper wraps libnbd operations with progress tracking
type LibNBDProgressWrapper struct {
	handle  *libnbd.Libnbd
	tracker *DataTracker
}

// NewLibNBDProgressWrapper creates a progress-aware libnbd wrapper
func NewLibNBDProgressWrapper(handle *libnbd.Libnbd, tracker *DataTracker) *LibNBDProgressWrapper {
	return &LibNBDProgressWrapper{
		handle:  handle,
		tracker: tracker,
	}
}

// Pread wraps libnbd.Pread with progress tracking
func (w *LibNBDProgressWrapper) Pread(buf []byte, offset uint64, flags *libnbd.PreadOptargs) error {
	err := w.handle.Pread(buf, offset, flags)
	if err != nil {
		return err
	}

	// Track actual bytes read from source
	w.tracker.OnDataTransfer(int64(len(buf)))
	return nil
}

// Pwrite wraps libnbd.Pwrite with progress tracking
func (w *LibNBDProgressWrapper) Pwrite(buf []byte, offset uint64, flags *libnbd.PwriteOptargs) error {
	err := w.handle.Pwrite(buf, offset, flags)
	if err != nil {
		return err
	}

	// Track actual bytes written to target
	w.tracker.OnDataTransfer(int64(len(buf)))
	return nil
}

// Proxy all other libnbd methods through to the wrapped handle
func (w *LibNBDProgressWrapper) ConnectUri(uri string) error {
	return w.handle.ConnectUri(uri)
}

func (w *LibNBDProgressWrapper) Close() error {
	return w.handle.Close()
}

func (w *LibNBDProgressWrapper) GetSize() (uint64, error) {
	return w.handle.GetSize()
}

func (w *LibNBDProgressWrapper) CanZero() (bool, error) {
	return w.handle.CanZero()
}

func (w *LibNBDProgressWrapper) CanFastZero() (bool, error) {
	return w.handle.CanFastZero()
}

func (w *LibNBDProgressWrapper) IsReadOnly() (bool, error) {
	return w.handle.IsReadOnly()
}

// SNAProgressNotifier handles communication with SNA progress endpoint
type SNAProgressNotifier struct {
	tracker    *DataTracker
	notifyFunc func() error
}

// NewVMAProgressNotifier creates a notifier that updates SNA every few seconds
func NewVMAProgressNotifier(tracker *DataTracker) *SNAProgressNotifier {
	return &SNAProgressNotifier{
		tracker: tracker,
		notifyFunc: func() error {
			return tracker.UpdateVMAEndpoint()
		},
	}
}

// StartPeriodicUpdates sends progress updates to SNA every 2 seconds
func (vpn *SNAProgressNotifier) StartPeriodicUpdates() {
	go func() {
		for {
			err := vpn.notifyFunc()
			if err != nil {
				// Log error but don't stop - progress tracking shouldn't break migration
				log.Printf("Warning: Failed to update SNA progress: %v", err)
			}
			// Update every 2 seconds per SHA polling design
			time.Sleep(2 * time.Second)
		}
	}()
}

// CalculatePlannedBytes determines total expected transfer size
func CalculatePlannedBytes(changeAreas []interface{}) int64 {
	// This will be called with VMware CBT change areas to calculate
	// the actual planned transfer size based on changed blocks
	var totalBytes int64

	// Note: Implementation depends on VMware CBT structure
	// For now, return a placeholder that can be updated when integrating
	// with actual VMware QueryChangedDiskAreas results

	return totalBytes
}

// GetJobIDFromContext extracts job ID from migration context
func GetJobIDFromContext() string {
	// 🎯 PRIORITY ORDER FOR JOB ID SELECTION:
	// 1) Command line --job-id flag (highest priority for progress tracking)
	// 2) Environment variable MIGRATEKIT_JOB_ID (for CBT compatibility + legacy)
	// 3) Auto-generated fallback (timestamp-based)

	// Priority 1: Check for command line job ID in environment
	// (This will be set by main.go when --job-id flag is provided)
	if cmdJobID := os.Getenv("MIGRATEKIT_PROGRESS_JOB_ID"); cmdJobID != "" {
		return cmdJobID
	}

	// Priority 2: Environment variable (CBT compatibility + legacy support)
	if envJobID := os.Getenv("MIGRATEKIT_JOB_ID"); envJobID != "" {
		return envJobID
	}

	// Priority 3: Auto-generated fallback (for backward compatibility)
	return fmt.Sprintf("job-%d", time.Now().Unix())
}

// GetVMAProgressEndpoint returns the SNA progress endpoint URL
func GetVMAProgressEndpoint() string {
	// Default to localhost since migratekit runs on SNA
	endpoint := os.Getenv("SNA_PROGRESS_ENDPOINT")
	if endpoint == "" {
		endpoint = "http://localhost:8081/api/v1/progress"
	}
	return endpoint
}
