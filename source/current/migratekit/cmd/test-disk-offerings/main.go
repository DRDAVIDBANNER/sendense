package main

import (
	"context"
	"log"
	"strings"

	"github.com/apache/cloudstack-go/cloudstack"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

type OSSEAConfig struct {
	ID        int    `db:"id"`
	Name      string `db:"name"`
	APIURL    string `db:"api_url"`
	APIKey    string `db:"api_key"`
	SecretKey string `db:"secret_key"`
	Domain    string `db:"domain"`
	Zone      string `db:"zone"`
}

func main() {
	log.Println("🔍 Listing CloudStack disk offerings and zones...")

	// Connect to database
	dsn := "oma_user:oma_password@tcp(localhost:3306)/migratekit_oma?parseTime=true"
	db, err := sqlx.Connect("mysql", dsn)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Get CloudStack config
	query := `SELECT id, name, api_url, api_key, secret_key, domain, zone FROM ossea_configs WHERE is_active = 1 LIMIT 1`
	var config OSSEAConfig
	err = db.GetContext(context.Background(), &config, query)
	if err != nil {
		log.Fatalf("Failed to get CloudStack config: %v", err)
	}

	log.Printf("Using config: %s", config.Name)

	// Create CloudStack client directly
	apiURL := config.APIURL
	if !strings.HasSuffix(apiURL, "/client/api") {
		apiURL = strings.TrimSuffix(apiURL, "/") + "/client/api"
	}

	cs := cloudstack.NewAsyncClient(apiURL, config.APIKey, config.SecretKey, false)
	cs.HTTPGETOnly = true

	// List zones
	log.Println("\n📍 Available zones:")
	zoneParams := cs.Zone.NewListZonesParams()
	zoneResp, err := cs.Zone.ListZones(zoneParams)
	if err != nil {
		log.Printf("Failed to list zones: %v", err)
	} else {
		for _, zone := range zoneResp.Zones {
			log.Printf("  - Zone: %s (ID: %s)", zone.Name, zone.Id)
			if zone.Name == config.Zone {
				log.Printf("    ✅ This matches the configured zone")
			}
		}
	}

	// List disk offerings
	log.Println("\n💾 Available disk offerings:")
	diskParams := cs.DiskOffering.NewListDiskOfferingsParams()
	diskResp, err := cs.DiskOffering.ListDiskOfferings(diskParams)
	if err != nil {
		log.Printf("Failed to list disk offerings: %v", err)
	} else {
		for _, offering := range diskResp.DiskOfferings {
			log.Printf("  - Disk Offering: %s (ID: %s)", offering.Name, offering.Id)
			if offering.Name == "Custom OSSEA" {
				log.Printf("    ✅ This is the Custom OSSEA offering we need!")
				log.Printf("    📝 Use disk_offering_id: %s", offering.Id)
			}
		}
	}

	log.Println("\n💡 Use the zone ID and Custom OSSEA disk offering ID for volume creation")
}
