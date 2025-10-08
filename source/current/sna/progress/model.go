package progress

import (
	"fmt"
	"os"
	"time"

	"github.com/k0kubun/go-ansi"
	"github.com/schollz/progressbar/v3"
	"github.com/vmware/govmomi/vim25/progress"
)

var theme = progressbar.Theme{
	Saucer:        "[green]=[reset]",
	SaucerHead:    "[green]>[reset]",
	SaucerPadding: " ",
	BarStart:      "[",
	BarEnd:        "]",
}

func DataProgressBar(desc string, size int64) *progressbar.ProgressBar {
	return progressbar.NewOptions64(size,
		progressbar.OptionSetWriter(ansi.NewAnsiStdout()),
		progressbar.OptionUseANSICodes(true),
		progressbar.OptionOnCompletion(func() {
			fmt.Fprint(os.Stderr, "\n")
		}),
		progressbar.OptionEnableColorCodes(true),
		progressbar.OptionShowBytes(true),
		progressbar.OptionShowCount(),
		progressbar.OptionUseIECUnits(true),
		progressbar.OptionFullWidth(),
		progressbar.OptionSetDescription(desc),
		progressbar.OptionSetTheme(theme),
	)
}

func PercentageProgressBar(task string) *progressbar.ProgressBar {
	return progressbar.NewOptions64(100,
		progressbar.OptionSetWriter(ansi.NewAnsiStdout()),
		progressbar.OptionUseANSICodes(true),
		progressbar.OptionOnCompletion(func() {
			fmt.Fprint(os.Stderr, "\n")
		}),
		progressbar.OptionEnableColorCodes(true),
		progressbar.OptionShowCount(),
		progressbar.OptionFullWidth(),
		progressbar.OptionSetDescription(task),
		progressbar.OptionSetTheme(theme),
	)
}

type VMwareProgressBar struct {
	bar *progressbar.ProgressBar
	ch  chan progress.Report
}

func NewVMwareProgressBar(task string) *VMwareProgressBar {
	bar := PercentageProgressBar(task)

	return &VMwareProgressBar{
		bar: bar,
		ch:  make(chan progress.Report),
	}
}

func (p *VMwareProgressBar) Sink() chan<- progress.Report {
	return p.ch
}

func (u *VMwareProgressBar) Loop(done <-chan struct{}) {
	for {
		select {
		case <-done:
			return
		case report, ok := <-u.ch:
			if !ok {
				return
			}
			if err := report.Error(); err != nil {
				return
			}

			pct := int(report.Percentage())
			u.bar.Set(pct)
			if detail := report.Detail(); detail != "" {
				u.bar.Describe(report.Detail())
			}
		}
	}
}

// ================================================================
// Replication Progress Tracking Models for SNA → SHA Integration
// ================================================================

// ReplicationStage represents the current stage of replication
type ReplicationStage string

const (
	StageDiscover          ReplicationStage = "Discover"
	StageEnableCBT         ReplicationStage = "EnableCBT"
	StageQueryCBT          ReplicationStage = "QueryCBT"
	StageSnapshot          ReplicationStage = "Snapshot"
	StagePrepareVolumes    ReplicationStage = "PrepareVolumes"
	StageStartExports      ReplicationStage = "StartExports"
	StageTransfer          ReplicationStage = "Transfer"
	StageFinalize          ReplicationStage = "Finalize"
	StagePersistChangeIDs  ReplicationStage = "PersistChangeIDs"
)

// ReplicationStatus represents the current status of replication
type ReplicationStatus string

const (
	StatusQueued     ReplicationStatus = "Queued"
	StatusPreparing  ReplicationStatus = "Preparing"
	StatusSnapshot   ReplicationStatus = "Snapshotting"
	StatusStreaming  ReplicationStatus = "Streaming"
	StatusFinalizing ReplicationStatus = "Finalizing"
	StatusSucceeded  ReplicationStatus = "Succeeded"
	StatusFailed     ReplicationStatus = "Failed"
)

// DiskStatus represents the current status of individual disk replication
type DiskStatus string

const (
	DiskStatusQueued     DiskStatus = "Queued"
	DiskStatusSnapshot   DiskStatus = "Snapshotting"
	DiskStatusStreaming  DiskStatus = "Streaming"
	DiskStatusCompleted  DiskStatus = "Completed"
	DiskStatusFailed     DiskStatus = "Failed"
)

// CBTType represents the type of CBT operation
type CBTType string

const (
	CBTTypeFull        CBTType = "full"
	CBTTypeIncremental CBTType = "incremental"
)

// AggregateProgress represents overall job progress
type AggregateProgress struct {
	TotalBytes       int64   `json:"total_bytes"`
	BytesTransferred int64   `json:"bytes_transferred"`
	ThroughputBPS    int64   `json:"throughput_bps"`
	Percent          float64 `json:"percent"`
}

// CBTInfo represents CBT-related information
type CBTInfo struct {
	Type              CBTType `json:"type"`
	PreviousChangeID  string  `json:"previous_change_id"`
	ChangeID          string  `json:"change_id"`
}

// NBDExport represents a single NBD export
type NBDExport struct {
	Name      string     `json:"name"`
	Device    string     `json:"device"`
	Connected bool       `json:"connected"`
	StartedAt *time.Time `json:"started_at"`
}

// NBDInfo represents NBD export information
type NBDInfo struct {
	Exports []NBDExport `json:"exports"`
}

// DiskProgress represents individual disk replication progress
type DiskProgress struct {
	ID               string     `json:"id"`
	Label            string     `json:"label"`
	PlannedBytes     int64      `json:"planned_bytes"`
	BytesTransferred int64      `json:"bytes_transferred"`
	ThroughputBPS    int64      `json:"throughput_bps"`
	Percent          float64    `json:"percent"`
	Status           DiskStatus `json:"status"`
}

// ReplicationProgress represents the complete progress information for a replication job
type ReplicationProgress struct {
	JobID     string            `json:"job_id"`
	Stage     ReplicationStage  `json:"stage"`
	Status    ReplicationStatus `json:"status"`
	StartedAt time.Time         `json:"started_at"`
	UpdatedAt time.Time         `json:"updated_at"`

	Aggregate AggregateProgress `json:"aggregate"`
	CBT       CBTInfo           `json:"cbt"`
	NBD       NBDInfo           `json:"nbd"`
	Disks     []DiskProgress    `json:"disks"`
}
