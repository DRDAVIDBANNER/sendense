package main

import (
	"context"
	"log"
	"time"

	"github.com/vexxhost/migratekit-volume-daemon/device"
)

func main() {
	log.Println("🧪 Testing Device Monitor...")

	// Create device monitor
	monitor, err := device.NewMonitor()
	if err != nil {
		log.Fatalf("Failed to create device monitor: %v", err)
	}

	// Start monitoring
	ctx := context.Background()
	if err := monitor.StartMonitoring(ctx); err != nil {
		log.Fatalf("Failed to start device monitor: %v", err)
	}

	log.Println("✅ Device monitor started successfully")

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
		log.Println("💚 Device monitor is healthy")
	} else {
		log.Println("❌ Device monitor is not healthy")
	}

	log.Println("👀 Monitoring for device events for 30 seconds...")
	log.Println("   (Try attaching/detaching volumes from CloudStack to see events)")

	// Monitor events for 30 seconds
	timeout := 30 * time.Second
	deadline := time.Now().Add(timeout)

	eventCount := 0
	for time.Now().Before(deadline) {
		eventCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		event, err := monitor.WaitForDevice(eventCtx, 2*time.Second)
		cancel()

		if err != nil {
			// No event, continue
			continue
		}

		eventCount++
		log.Printf("🎯 Device Event #%d:", eventCount)
		log.Printf("   Type: %s", event.Type)
		log.Printf("   Path: %s", event.DevicePath)
		log.Printf("   Time: %s", event.Timestamp.Format(time.RFC3339))

		if event.DeviceInfo != nil {
			log.Printf("   Size: %d bytes", event.DeviceInfo.Size)
			log.Printf("   Controller: %s", event.DeviceInfo.Controller)
		}
	}

	if eventCount == 0 {
		log.Println("📝 No device events detected during monitoring period")
		log.Println("   This is normal if no volumes were attached/detached")
	} else {
		log.Printf("✅ Detected %d device events", eventCount)
	}

	// Stop monitoring
	if err := monitor.StopMonitoring(ctx); err != nil {
		log.Printf("Failed to stop device monitor: %v", err)
	} else {
		log.Println("🛑 Device monitor stopped successfully")
	}

	log.Println("🎉 Device monitor test completed!")
}
