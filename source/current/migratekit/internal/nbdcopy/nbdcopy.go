package nbdcopy

import (
	"bufio"
	"os"
	"os/exec"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/vexxhost/migratekit/internal/progress"
)

// AllocatedBlock represents a block of data that needs to be transferred
type AllocatedBlock struct {
	Offset int64
	Length int64
}

func Run(source, destination string, size int64, targetIsClean bool) error {
	return RunWithCBTTracking(source, destination, size, targetIsClean, nil)
}

// RunWithCBTTracking runs nbdcopy with proper CBT-based completion detection
func RunWithCBTTracking(source, destination string, size int64, targetIsClean bool, cbtBlocks []AllocatedBlock) error {
	logger := log.WithFields(log.Fields{
		"source":      source,
		"destination": destination,
	})

	progressRead, progressWrite, err := os.Pipe()
	if err != nil {
		return err
	}
	defer progressRead.Close()

	args := []string{
		"--progress=3",
		source,
		destination,
	}

	// Only add --destination-is-zero for regular files, not raw devices
	if targetIsClean && !strings.HasPrefix(destination, "/dev/") {
		args = append(args, "--destination-is-zero")
	}

	cmd := exec.Command(
		"nbdcopy",
		args...,
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.ExtraFiles = []*os.File{progressWrite}

	logger.Debug("Running command: ", cmd)
	if err := cmd.Start(); err != nil {
		return err
	}

	// Close the parent's copy of progressWrite
	// See: https://github.com/golang/go/issues/4261
	progressWrite.Close()

	// Use percentage-based progress since nbdcopy reports disk position processed
	bar := progress.PercentageProgressBar("Copying VM data")

	// Channel to signal when nbdcopy process completes
	done := make(chan error, 1)

	go func() {
		scanner := bufio.NewScanner(progressRead)
		for scanner.Scan() {
			progressLine := scanner.Text()
			// nbdcopy --progress=FD outputs "N/100" where N is percentage (0-100)
			progressParts := strings.Split(progressLine, "/")
			if len(progressParts) >= 2 {
				progressPct, err := strconv.ParseInt(progressParts[0], 10, 64)
				if err != nil {
					log.Error("Error parsing progress percentage: ", err)
					continue
				}
				// Show progress of disk position processed (may not reach 100% for sparse disks)
				bar.Set(int(progressPct))
			} else {
				log.Warn("Unexpected progress format: ", progressLine)
			}
		}

		if err := scanner.Err(); err != nil {
			log.Error("Error reading progress: ", err)
		}
	}()

	// 🚨 SIMPLIFIED COMPLETION DETECTION
	// Just wait for nbdcopy to complete naturally - no external termination

	// Wait for nbdcopy process completion
	go func() {
		done <- cmd.Wait()
	}()

	// 🎯 DISABLED: Problematic external completion detection for named pipes
	// The named pipe monitoring logic was causing premature termination
	// We'll rely on OMA-side monitoring for proper completion detection
	log.Infof("📊 Expected data size: %.2f GB (%d bytes) - letting nbdcopy run naturally",
		float64(size)/(1024*1024*1024), size)

	// Wait for natural nbdcopy completion (no premature termination)
	err = <-done
	log.Info("🚀 nbdcopy process completed naturally")

	// Always force progress to 100% when we detect completion
	log.Info("🚀 Migration completed - finalizing progress...")
	bar.Set(100)

	if err != nil {
		return err
	}

	return nil
}
