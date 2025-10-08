package main

import (
	"context"
	"database/sql"
	"log"

	"github.com/jmoiron/sqlx"
	_ "github.com/go-sql-driver/mysql"
	
	"github.com/vexxhost/migratekit-volume-daemon/cloudstack"
)

func main() {
	log.Println("🔍 Testing CloudStack zones...")

	// Connect to database
	dsn := "oma_user:oma_password@tcp(localhost:3306)/migratekit_oma?parseTime=true"
	db, err := sqlx.Connect("mysql", dsn)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Create CloudStack client
	factory := cloudstack.NewFactory(db)
	client, err := factory.CreateClient(context.Background())
	if err != nil {
		log.Fatalf("Failed to create CloudStack client: %v", err)
	}

	// Query zones using the CloudStack client's underlying connection
	log.Println("📝 Note: Using direct CloudStack SDK call since our client doesn't expose zone listing yet")
	
	// For now, let's check what zone info we can get from the configs
	query := `
		SELECT id, name, zone, api_url, domain
		FROM ossea_configs 
		WHERE is_active = true
		LIMIT 1
	`

	var config struct {
		ID     int    `db:"id"`
		Name   string `db:"name"`
		Zone   string `db:"zone"`
		APIURL string `db:"api_url"`
		Domain string `db:"domain"`
	}

	err = db.GetContext(context.Background(), &config, query)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Fatalf("No active CloudStack configuration found")
		}
		log.Fatalf("Failed to query CloudStack config: %v", err)
	}

	log.Printf("Active config: %s", config.Name)
	log.Printf("Zone configured: %s", config.Zone)
	log.Printf("Domain: %s", config.Domain)
	log.Printf("API URL: %s", config.APIURL)

	log.Println("💡 The zone value should be a zone ID (UUID), not a zone name")
	log.Println("💡 You may need to update the ossea_configs table with the correct zone ID")
}
