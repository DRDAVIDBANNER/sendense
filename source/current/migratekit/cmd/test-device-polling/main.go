package main

import (
	"io/ioutil"
	"log"
	"strings"
	"time"
)

func main() {
	log.Println("🔍 Testing device polling detection...")
	log.Println("👀 Monitoring /sys/block for device changes every 2 seconds for 60 seconds")
	log.Println("📝 Try adding/removing volumes now...")

	previousDevices := make(map[string]bool)

	// Initial scan
	devices := scanVirtioDevices()
	for _, device := range devices {
		previousDevices[device] = true
		log.Printf("📋 Initial device: %s", device)
	}
	log.Printf("🎯 Found %d initial virtio devices", len(devices))

	// Poll for changes
	for i := 0; i < 30; i++ { // 30 iterations * 2 seconds = 60 seconds
		time.Sleep(2 * time.Second)

		currentDevices := scanVirtioDevices()
		currentMap := make(map[string]bool)

		for _, device := range currentDevices {
			currentMap[device] = true
		}

		// Check for new devices
		for device := range currentMap {
			if !previousDevices[device] {
				log.Printf("✅ NEW DEVICE DETECTED: %s", device)
			}
		}

		// Check for removed devices
		for device := range previousDevices {
			if !currentMap[device] {
				log.Printf("❌ DEVICE REMOVED: %s", device)
			}
		}

		// Update previous state
		previousDevices = currentMap

		if i%5 == 0 { // Status update every 10 seconds
			log.Printf("📊 Status check %d: %d devices present", i/5+1, len(currentDevices))
		}
	}

	log.Println("🏁 Polling test completed!")
}

func scanVirtioDevices() []string {
	entries, err := ioutil.ReadDir("/sys/block")
	if err != nil {
		log.Printf("Failed to read /sys/block: %v", err)
		return nil
	}

	var devices []string
	for _, entry := range entries {
		deviceName := entry.Name()
		if strings.HasPrefix(deviceName, "vd") {
			devices = append(devices, deviceName)
		}
	}

	return devices
}
