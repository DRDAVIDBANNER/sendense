package main

import (
	"log"

	"github.com/apache/cloudstack-go/cloudstack"
	"github.com/vexxhost/migratekit/internal/sha/database"
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
	cs := cloudstack.NewAsyncClient(config.APIURL, config.APIKey, config.SecretKey, false)

	// Look for the new 5GB volume
	params := cs.Volume.NewListVolumesParams()
	response, err := cs.Volume.ListVolumes(params)
	if err != nil {
		log.Fatalf("Failed to list volumes: %v", err)
	}

	log.Printf("=== All CloudStack Volumes ===")
	for _, vol := range response.Volumes {
		sizeGB := float64(vol.Size) / (1024 * 1024 * 1024)
		log.Printf("Volume: %s", vol.Name)
		log.Printf("  ID: %s", vol.Id)
		log.Printf("  Size: %.1f GB (%d bytes)", sizeGB, vol.Size)
		log.Printf("  Type: %s", vol.Type)
		log.Printf("  Deviceid: %d", vol.Deviceid)
		log.Printf("  VM ID: %s", vol.Virtualmachineid)
		log.Printf("  State: %s", vol.State)

		// Check if this matches our new 5GB device
		if sizeGB >= 4.5 && sizeGB <= 5.5 {
			log.Printf("  *** POTENTIAL MATCH FOR NEW 5GB DEVICE ***")
		}
		log.Printf("")
	}
}
