package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"
)

func main() {
	log.Println("🔍 Debug: Device Scanning Logic")

	// Read /sys/block directory
	entries, err := ioutil.ReadDir("/sys/block")
	if err != nil {
		log.Fatalf("Failed to read /sys/block: %v", err)
	}

	log.Printf("Found %d entries in /sys/block", len(entries))

	virtioCount := 0
	for _, entry := range entries {
		deviceName := entry.Name()
		log.Printf("  Entry: %s (IsDir: %v)", deviceName, entry.IsDir())

		if !entry.IsDir() {
			log.Printf("    Skipping %s: not a directory", deviceName)
			continue
		}

		if !strings.HasPrefix(deviceName, "vd") {
			log.Printf("    Skipping %s: not a virtio device", deviceName)
			continue
		}

		virtioCount++
		log.Printf("    ✅ Virtio device found: %s", deviceName)

		// Try to read size
		sizePath := fmt.Sprintf("/sys/block/%s/size", deviceName)
		sizeData, err := ioutil.ReadFile(sizePath)
		if err != nil {
			log.Printf("    ❌ Failed to read size from %s: %v", sizePath, err)
			continue
		}

		sectors, err := strconv.ParseInt(strings.TrimSpace(string(sizeData)), 10, 64)
		if err != nil {
			log.Printf("    ❌ Failed to parse size %s: %v", string(sizeData), err)
			continue
		}

		sizeBytes := sectors * 512
		log.Printf("    Size: %d sectors = %d bytes", sectors, sizeBytes)

		// Try to read device symlink
		deviceLink := fmt.Sprintf("/sys/block/%s/device", deviceName)
		target, err := os.Readlink(deviceLink)
		if err != nil {
			log.Printf("    ❌ Failed to read device link %s: %v", deviceLink, err)
			continue
		}

		log.Printf("    Device link: %s", target)

		// Extract virtio controller
		parts := strings.Split(target, "/")
		controller := "unknown"
		for _, part := range parts {
			if strings.HasPrefix(part, "virtio") {
				controller = part
				break
			}
		}

		log.Printf("    Controller: %s", controller)
		log.Printf("    ✅ Device %s successfully processed", deviceName)
	}

	log.Printf("🎯 Summary: Found %d virtio devices", virtioCount)

	if virtioCount == 0 {
		log.Println("❌ No virtio devices detected - this explains why the monitor showed 0 devices")
	} else {
		log.Println("✅ Virtio devices detected - monitor should have found them")
	}
}
