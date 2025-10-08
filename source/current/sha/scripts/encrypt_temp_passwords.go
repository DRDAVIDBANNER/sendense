// Simple script to encrypt temporary plaintext passwords in VMware credentials
package main

import (
	"fmt"
	"log"

	"github.com/vexxhost/migratekit-sha/database"
	"github.com/vexxhost/migratekit-sha/services"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	// Connect to database
	dsn := "oma_user:oma_password@tcp(localhost:3306)/migratekit_oma?charset=utf8mb4&parseTime=True&loc=Local"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// Initialize encryption service
	encryptionService, err := services.NewCredentialEncryptionService()
	if err != nil {
		log.Fatal("Failed to initialize encryption service:", err)
	}

	// Get all credentials with temporary passwords
	var credentials []database.VMwareCredential
	err = db.Where("password_encrypted LIKE ?", "TEMP_%").Find(&credentials).Error
	if err != nil {
		log.Fatal("Failed to query credentials:", err)
	}

	fmt.Printf("Found %d credentials with temporary passwords\n", len(credentials))

	for _, cred := range credentials {
		// Extract plaintext password (remove TEMP_PLAINTEXT_ prefix)
		plaintextPassword := cred.PasswordEncrypted[15:] // Remove "TEMP_PLAINTEXT_" prefix

		// Encrypt the password
		encryptedPassword, err := encryptionService.EncryptPassword(plaintextPassword)
		if err != nil {
			log.Printf("Failed to encrypt password for credential %d: %v", cred.ID, err)
			continue
		}

		// Update database with encrypted password
		err = db.Model(&cred).Update("password_encrypted", encryptedPassword).Error
		if err != nil {
			log.Printf("Failed to update credential %d: %v", cred.ID, err)
			continue
		}

		fmt.Printf("✅ Encrypted password for credential: %s\n", cred.CredentialName)
	}

	fmt.Println("Password encryption migration completed!")
}
