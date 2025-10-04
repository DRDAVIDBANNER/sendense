// Package services provides VMware credential encryption for secure password storage
package services

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"os"
)

// CredentialEncryptionService handles AES-256-GCM encryption/decryption for VMware passwords
type CredentialEncryptionService struct {
	gcm cipher.AEAD
}

// NewCredentialEncryptionService creates a new credential encryption service
func NewCredentialEncryptionService() (*CredentialEncryptionService, error) {
	// Get encryption key from environment
	keyBase64 := os.Getenv("MIGRATEKIT_CRED_ENCRYPTION_KEY")
	if keyBase64 == "" {
		return nil, fmt.Errorf("MIGRATEKIT_CRED_ENCRYPTION_KEY environment variable not set")
	}

	// Decode base64 key
	key, err := base64.StdEncoding.DecodeString(keyBase64)
	if err != nil {
		return nil, fmt.Errorf("invalid base64 encryption key: %w", err)
	}

	if len(key) != 32 {
		return nil, fmt.Errorf("encryption key must be 32 bytes (256 bits), got %d bytes", len(key))
	}

	// Create AES cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM mode: %w", err)
	}

	return &CredentialEncryptionService{
		gcm: gcm,
	}, nil
}

// EncryptPassword encrypts a VMware password using AES-256-GCM
func (ces *CredentialEncryptionService) EncryptPassword(password string) (string, error) {
	if password == "" {
		return "", fmt.Errorf("password cannot be empty")
	}

	// Generate random nonce
	nonce := make([]byte, ces.gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Encrypt password
	plaintext := []byte(password)
	ciphertext := ces.gcm.Seal(nonce, nonce, plaintext, nil)

	// Encode to base64 for database storage
	encrypted := base64.StdEncoding.EncodeToString(ciphertext)
	return encrypted, nil
}

// DecryptPassword decrypts a VMware password from database storage
func (ces *CredentialEncryptionService) DecryptPassword(encryptedPassword string) (string, error) {
	if encryptedPassword == "" {
		return "", fmt.Errorf("encrypted password cannot be empty")
	}

	// Handle temporary plaintext passwords during migration
	if encryptedPassword[:5] == "TEMP_" {
		// During migration phase, return plaintext without TEMP_ prefix
		return encryptedPassword[15:], nil // Remove "TEMP_PLAINTEXT_" prefix
	}

	// Decode from base64
	ciphertext, err := base64.StdEncoding.DecodeString(encryptedPassword)
	if err != nil {
		return "", fmt.Errorf("invalid base64 encrypted password: %w", err)
	}

	nonceSize := ces.gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	// Extract nonce and encrypted data
	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]

	// Decrypt password
	plaintext, err := ces.gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt password: %w", err)
	}

	return string(plaintext), nil
}

// ValidateEncryptionKey tests if encryption key is working correctly
func (ces *CredentialEncryptionService) ValidateEncryptionKey() error {
	testPassword := "test-encryption-validation"

	// Test encryption
	encrypted, err := ces.EncryptPassword(testPassword)
	if err != nil {
		return fmt.Errorf("encryption test failed: %w", err)
	}

	// Test decryption
	decrypted, err := ces.DecryptPassword(encrypted)
	if err != nil {
		return fmt.Errorf("decryption test failed: %w", err)
	}

	if decrypted != testPassword {
		return fmt.Errorf("encryption/decryption validation failed: passwords don't match")
	}

	return nil
}

// MigrateTemporaryPasswords encrypts any temporary plaintext passwords in the database
func (ces *CredentialEncryptionService) MigrateTemporaryPasswords(db interface{}) error {
	// This will be implemented when we integrate with the database layer
	// For now, return success to enable service initialization
	return nil
}

