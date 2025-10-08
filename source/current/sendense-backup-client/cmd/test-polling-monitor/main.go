package main

import (
	"context"
	"log"
	"time"

	"github.com/vexxhost/migratekit-volume-daemon/device"
)

func main() {
	log.Println("🧪 Testing Polling-Based Device Monitor...")

	// Create polling device monitor
	monitor, err := device.NewPollingMonitor()
	if err != nil {
		log.Fatalf("Failed to create polling monitor: %v", err)
	}

	// Start monitoring
	ctx := context.Background()
	if err := monitor.StartMonitoring(ctx); err != nil {
		log.Fatalf("Failed to start polling monitor: %v", err)
	}

	log.Println("✅ Polling device monitor started successfully")

	// Get current devices
	devices, err := monitor.GetDevices(ctx)
	if err != nil {
		log.Printf("Failed to get devices: %v", err)
	} else {
		log.Printf("📋 Current devices: %d", len(devices))
		for _, dev := range devices {
			log.Printf("  - %s: %d bytes, controller: %s", dev.Path, dev.Size, dev.Controller)
		}
	}

	// Check health
	if monitor.IsHealthy(ctx) {
		log.Println("💚 Polling device monitor is healthy")
	} else {
		log.Println("❌ Polling device monitor is not healthy")
	}

	log.Println("👀 Monitoring for device events for 60 seconds...")
	log.Println("   🔧 Try attaching/detaching volumes from CloudStack now!")

	// Monitor events for 60 seconds
	timeout := 60 * time.Second
	deadline := time.Now().Add(timeout)

	eventCount := 0
	for time.Now().Before(deadline) {
		eventCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
		event, err := monitor.WaitForDevice(eventCtx, 3*time.Second)
		cancel()

		if err != nil {
			// No event, continue
			continue
		}

		eventCount++
		log.Printf("🎯 Device Event #%d:", eventCount)
		log.Printf("   Type: %s", event.Type)
		log.Printf("   Path: %s", event.DevicePath)
		log.Printf("   Time: %s", event.Timestamp.Format("15:04:05.000"))

		if event.DeviceInfo != nil {
			log.Printf("   Size: %d bytes", event.DeviceInfo.Size)
			log.Printf("   Controller: %s", event.DeviceInfo.Controller)
		}
		log.Printf("   📊 Source: %s", event.DeviceInfo.Metadata["source"])
	}

	if eventCount == 0 {
		log.Println("📝 No device events detected during monitoring period")
		log.Println("   This could mean no volumes were attached/detached")
	} else {
		log.Printf("✅ Successfully detected %d device events via polling!", eventCount)
	}

	// Stop monitoring
	if err := monitor.StopMonitoring(ctx); err != nil {
		log.Printf("Failed to stop polling monitor: %v", err)
	} else {
		log.Println("🛑 Polling device monitor stopped successfully")
	}

	log.Println("🎉 Polling device monitor test completed!")
}
