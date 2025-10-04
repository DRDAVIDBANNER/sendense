package main

import (
	"log"

	"github.com/apache/cloudstack-go/cloudstack"
	"github.com/vexxhost/migratekit/internal/oma/database"
)

func main() {
	// Create database connection
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

	config := configs[0]
	log.Printf("Using OSSEA config: %s", config.Name)

	// Create CloudStack client directly
	cs := cloudstack.NewAsyncClient(config.APIURL, config.APIKey, config.SecretKey, false)

	// Test specific volumes from our database
	testVolumes := []struct {
		VolumeID string
		Name     string
	}{
		{"e915ef05-ddf5-48d5-8352-a01300609717", "PGWINTESTBIOS-OLD"},
		{"dd0e1f2f-1062-4c83-b011-cca29d21748b", "PGWINTESTBIOS-NEW"},
		{"00ff0e64-8619-433e-a4df-1ecaaf804010", "PhilB Test machine"},
	}

	for _, vol := range testVolumes {
		log.Printf("\n=== Testing Volume: %s ===", vol.Name)
		log.Printf("Volume ID: %s", vol.VolumeID)

		// Query CloudStack for this specific volume
		params := cs.Volume.NewListVolumesParams()
		params.SetId(vol.VolumeID)

		response, err := cs.Volume.ListVolumes(params)
		if err != nil {
			log.Printf("❌ Error querying CloudStack: %v", err)
			continue
		}

		if response.Count == 0 || len(response.Volumes) == 0 {
			log.Printf("❌ Volume not found in CloudStack")
			continue
		}

		csVol := response.Volumes[0]

		log.Printf("CloudStack Response:")
		log.Printf("  Volume Name: %s", csVol.Name)
		log.Printf("  Volume Size: %d bytes", csVol.Size)
		log.Printf("  Volume Type: %s", csVol.Type)
		log.Printf("  Volume State: %s", csVol.State)
		log.Printf("  VM ID: %s", csVol.Virtualmachineid)
		log.Printf("  **DEVICE ID: %d**", csVol.Deviceid)

		// Calculate device path based on Device ID
		if csVol.Deviceid >= 0 {
			deviceLetter := 'a' + rune(csVol.Deviceid)
			calculatedPath := "/dev/vd" + string(deviceLetter)
			log.Printf("  **Calculated Device Path: %s**", calculatedPath)
		}
	}

	log.Printf("\n=== Device ID Correlation Test Complete ===")
}
