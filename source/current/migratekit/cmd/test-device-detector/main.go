package main

import (
	"log"

	"github.com/vexxhost/migratekit/internal/oma/database"
	"github.com/vexxhost/migratekit/internal/oma/device"

	"github.com/apache/cloudstack-go/cloudstack"
)

func main() {
	// Create database connection using MariaDB
	dbConfig := &database.MariaDBConfig{
		Host:     "localhost",
		Port:     3306,
		Username: "oma_user",
		Password: "oma_password",
		Database: "migratekit_oma",
	}

	db, err := database.NewMariaDBConnection(dbConfig)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Get OSSEA configuration
	osseaRepo := database.NewOSSEAConfigRepository(db)
	configs, err := osseaRepo.GetAll()
	if err != nil {
		log.Fatalf("Failed to get OSSEA configs: %v", err)
	}

	if len(configs) == 0 {
		log.Fatalf("No OSSEA configurations found")
	}

	config := configs[0]
	log.Printf("Using OSSEA config: %s at %s", config.Name, config.APIURL)

	// Create CloudStack client directly
	cs := cloudstack.NewAsyncClient(config.APIURL, config.APIKey, config.SecretKey, false)

	// Create device detector
	detector := device.NewDeviceDetector(cs)

	// Get volumes from database to test
	var volumes []database.OSSEAVolume
	if err := db.GetGormDB().Find(&volumes).Error; err != nil {
		log.Fatalf("Failed to get volumes from database: %v", err)
	}

	log.Printf("Testing device detection on %d volumes from database:", len(volumes))

	for i, vol := range volumes {
		if i >= 3 { // Test only first 3 volumes
			break
		}

		log.Printf("\n--- Testing Volume %d ---", i+1)
		log.Printf("Volume ID: %s", vol.VolumeID)
		log.Printf("Volume Name: %s", vol.VolumeName)
		log.Printf("Database Device Path: %s", vol.DevicePath)

		// Get volume info from CloudStack API
		info, err := detector.GetVolumeDeviceInfo(vol.VolumeID)
		if err != nil {
			log.Printf("❌ Error getting volume info: %v", err)
			continue
		}

		log.Printf("CloudStack Status: %s", info.Status)
		log.Printf("CloudStack VM ID: %s", info.VMID)
		log.Printf("CloudStack Detected Device Path: %s", info.DevicePath)

		// Compare with database
		if info.DevicePath != vol.DevicePath {
			log.Printf("⚠️  MISMATCH DETECTED!")
			log.Printf("  Database says: %s", vol.DevicePath)
			log.Printf("  CloudStack says: %s", info.DevicePath)
		} else {
			log.Printf("✅ Device paths match")
		}

		// Test direct device path detection if volume is attached
		if info.VMID != "" {
			devicePath, err := detector.GetActualDevicePath(vol.VolumeID, info.VMID)
			if err != nil {
				log.Printf("❌ Error getting actual device path: %v", err)
			} else {
				log.Printf("Direct Detection Result: %s", devicePath)
			}
		}
	}

	log.Printf("\n=== Device Detection Test Complete ===")
}
