package vmware_nbdkit

import (
	"fmt"

	log "github.com/sirupsen/logrus"
)

// CoalescedExtent represents a merged group of adjacent or near-adjacent extents
type CoalescedExtent struct {
	Offset        int64 // Starting offset in bytes
	Length        int64 // Total length in bytes
	OriginalCount int   // Number of original extents merged into this one
}

// coalesceExtents merges adjacent or near-adjacent extents into larger chunks
// to reduce the number of NBD operations and improve throughput.
//
// Parameters:
//   - extents: slice of disk change extents from VMware CBT
//   - maxGap: maximum gap between extents to merge (e.g. 1 MB)
//   - maxChunkSize: maximum size of coalesced extent (e.g. 32 MB)
//
// Returns: slice of coalesced extents ready for parallel processing
func coalesceExtents(extents []DiskExtent, maxGap int64, maxChunkSize int64) []CoalescedExtent {
	if len(extents) == 0 {
		return nil
	}

	logger := log.WithFields(log.Fields{
		"total_extents": len(extents),
		"max_gap":       maxGap,
		"max_chunk":     maxChunkSize,
	})

	var coalesced []CoalescedExtent
	current := CoalescedExtent{
		Offset:        extents[0].Offset,
		Length:        extents[0].Length,
		OriginalCount: 1,
	}

	for i := 1; i < len(extents); i++ {
		extent := extents[i]
		currentEnd := current.Offset + current.Length
		gap := extent.Offset - currentEnd

		// Check if we can merge this extent with the current coalesced extent
		canMerge := (gap <= maxGap) && (current.Length+gap+extent.Length <= maxChunkSize)

		if canMerge {
			// Merge: extend current extent to include gap + new extent
			current.Length = extent.Offset + extent.Length - current.Offset
			current.OriginalCount++
		} else {
			// Cannot merge: save current and start new coalesced extent
			coalesced = append(coalesced, current)
			current = CoalescedExtent{
				Offset:        extent.Offset,
				Length:        extent.Length,
				OriginalCount: 1,
			}
		}
	}

	// Don't forget the last extent
	coalesced = append(coalesced, current)

	// Calculate statistics
	var totalOriginalBytes, totalCoalescedBytes int64
	for i := range extents {
		totalOriginalBytes += extents[i].Length
	}
	for i := range coalesced {
		totalCoalescedBytes += coalesced[i].Length
	}

	overheadRatio := float64(totalCoalescedBytes-totalOriginalBytes) / float64(totalOriginalBytes) * 100

	logger.WithFields(log.Fields{
		"original_extents":  len(extents),
		"coalesced_extents": len(coalesced),
		"reduction":         fmt.Sprintf("%.1f%%", float64(len(extents)-len(coalesced))/float64(len(extents))*100),
		"overhead_ratio":    fmt.Sprintf("%.2f%%", overheadRatio),
	}).Info("🔗 Extent coalescing completed")

	return coalesced
}

// splitExtentsAcrossWorkers distributes coalesced extents evenly across N workers
// using round-robin distribution to balance load.
//
// Parameters:
//   - extents: coalesced extents ready for parallel processing
//   - numWorkers: number of worker goroutines (typically 2-4)
//
// Returns: slice of extent slices, one per worker
func splitExtentsAcrossWorkers(extents []CoalescedExtent, numWorkers int) [][]CoalescedExtent {
	if numWorkers <= 0 {
		numWorkers = 1
	}

	// Initialize worker slices
	workerExtents := make([][]CoalescedExtent, numWorkers)
	for i := range workerExtents {
		workerExtents[i] = make([]CoalescedExtent, 0, len(extents)/numWorkers+1)
	}

	// Round-robin distribution
	for i, extent := range extents {
		workerID := i % numWorkers
		workerExtents[workerID] = append(workerExtents[workerID], extent)
	}

	// Log distribution statistics
	for i, extents := range workerExtents {
		var totalBytes int64
		for _, extent := range extents {
			totalBytes += extent.Length
		}

		log.WithFields(log.Fields{
			"worker_id":     i,
			"extent_count":  len(extents),
			"total_bytes":   totalBytes,
			"total_mb":      totalBytes / (1024 * 1024),
		}).Debug("📦 Worker extent allocation")
	}

	return workerExtents
}

// DiskExtent represents a changed disk area from VMware CBT
type DiskExtent struct {
	Offset int64
	Length int64
}

// calculateTotalBytes returns the sum of all extent lengths
func calculateTotalBytes(extents []CoalescedExtent) int64 {
	var total int64
	for _, extent := range extents {
		total += extent.Length
	}
	return total
}

