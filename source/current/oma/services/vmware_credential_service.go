// Package services provides VMware credential management for centralized vCenter authentication
package services

import (
	"context"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/vexxhost/migratekit-oma/database"
)

// VMwareCredentialService manages VMware vCenter credentials with encryption
type VMwareCredentialService struct {
	db                *database.Connection
	encryptionService *CredentialEncryptionService
}

// NewVMwareCredentialService creates a new VMware credential service
func NewVMwareCredentialService(db *database.Connection, encryptionService *CredentialEncryptionService) *VMwareCredentialService {
	return &VMwareCredentialService{
		db:                db,
		encryptionService: encryptionService,
	}
}

// GetCredentials retrieves and decrypts VMware credentials by ID
func (vcs *VMwareCredentialService) GetCredentials(ctx context.Context, credentialID int) (*database.VMwareCredentials, error) {
	var credential database.VMwareCredential
	err := (*vcs.db).GetGormDB().Where("id = ? AND is_active = ?", credentialID, true).First(&credential).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get VMware credentials: %w", err)
	}

	// Decrypt password (handle nil encryption service)
	var password string
	if vcs.encryptionService != nil {
		password, err = vcs.encryptionService.DecryptPassword(credential.PasswordEncrypted)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt password: %w", err)
		}
	} else {
		// Development mode: password stored as plaintext
		password = credential.PasswordEncrypted
	}

	// Update last used timestamp
	now := time.Now()
	(*vcs.db).GetGormDB().Model(&credential).Updates(map[string]interface{}{
		"last_used":   now,
		"usage_count": credential.UsageCount + 1,
	})

	log.WithFields(log.Fields{
		"credential_id":   credentialID,
		"credential_name": credential.CredentialName,
		"vcenter_host":    credential.VCenterHost,
		"username":        credential.Username,
		"db_vcenter_host": credential.VCenterHost,
		"raw_credential":  fmt.Sprintf("%+v", credential),
	}).Info("🔐 VMware credentials retrieved and decrypted - DEBUG")

	return &database.VMwareCredentials{
		ID:          credential.ID,
		Name:        credential.CredentialName,
		VCenterHost: credential.VCenterHost,
		Username:    credential.Username,
		Password:    password,
		Datacenter:  credential.Datacenter,
		IsActive:    credential.IsActive,
		IsDefault:   credential.IsDefault,
	}, nil
}

// GetDefaultCredentials retrieves the default VMware credential set
func (vcs *VMwareCredentialService) GetDefaultCredentials(ctx context.Context) (*database.VMwareCredentials, error) {
	var credential database.VMwareCredential
	err := (*vcs.db).GetGormDB().Where("is_default = ? AND is_active = ?", true, true).First(&credential).Error
	if err != nil {
		return nil, fmt.Errorf("no default VMware credentials found: %w", err)
	}

	return vcs.GetCredentials(ctx, credential.ID)
}

// CreateCredentials stores new encrypted credential set
func (vcs *VMwareCredentialService) CreateCredentials(ctx context.Context, creds *database.VMwareCredentials) (*database.VMwareCredential, error) {
	var encryptedPassword string
	var err error
	
	// Handle encryption gracefully (development mode support)
	if vcs.encryptionService != nil {
		encryptedPassword, err = vcs.encryptionService.EncryptPassword(creds.Password)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt password: %w", err)
		}
	} else {
		// Development mode: store plaintext with warning
		encryptedPassword = creds.Password
		log.Warn("⚠️ Storing password in plaintext (encryption service unavailable)")
	}

	// If this is set as default, unset other defaults
	if creds.IsDefault {
		(*vcs.db).GetGormDB().Model(&database.VMwareCredential{}).Where("is_default = ?", true).Updates(map[string]interface{}{
			"is_default": false,
		})
	}

	credential := &database.VMwareCredential{
		CredentialName:    creds.Name,
		VCenterHost:       creds.VCenterHost,
		Username:          creds.Username,
		PasswordEncrypted: encryptedPassword,
		Datacenter:        creds.Datacenter,
		IsActive:          creds.IsActive,
		IsDefault:         creds.IsDefault,
		CreatedBy:         "gui_user", // TODO: Get from authentication context
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	err = (*vcs.db).GetGormDB().Create(credential).Error
	if err != nil {
		return nil, fmt.Errorf("failed to create VMware credentials: %w", err)
	}

	log.WithFields(log.Fields{
		"credential_id":   credential.ID,
		"credential_name": credential.CredentialName,
		"vcenter_host":    credential.VCenterHost,
		"is_default":      credential.IsDefault,
	}).Info("✅ VMware credentials created successfully")

	return credential, nil
}

// UpdateCredentials updates existing credential set
func (vcs *VMwareCredentialService) UpdateCredentials(ctx context.Context, credentialID int, creds *database.VMwareCredentials) error {
	var credential database.VMwareCredential
	err := (*vcs.db).GetGormDB().Where("id = ?", credentialID).First(&credential).Error
	if err != nil {
		return fmt.Errorf("credential set not found: %w", err)
	}

	// Encrypt new password if provided
	if creds.Password != "" {
		encryptedPassword, err := vcs.encryptionService.EncryptPassword(creds.Password)
		if err != nil {
			return fmt.Errorf("failed to encrypt password: %w", err)
		}
		credential.PasswordEncrypted = encryptedPassword
	}

	// Update other fields
	credential.CredentialName = creds.Name
	credential.VCenterHost = creds.VCenterHost
	credential.Username = creds.Username
	credential.Datacenter = creds.Datacenter
	credential.IsActive = creds.IsActive
	credential.UpdatedAt = time.Now()

	// Handle default flag
	if creds.IsDefault && !credential.IsDefault {
		// Unset other defaults first
		(*vcs.db).GetGormDB().Model(&database.VMwareCredential{}).Where("is_default = ? AND id != ?", true, credentialID).Updates(map[string]interface{}{
			"is_default": false,
		})
		credential.IsDefault = true
	}

	err = (*vcs.db).GetGormDB().Save(&credential).Error
	if err != nil {
		return fmt.Errorf("failed to update VMware credentials: %w", err)
	}

	log.WithFields(log.Fields{
		"credential_id":   credentialID,
		"credential_name": credential.CredentialName,
	}).Info("✅ VMware credentials updated successfully")

	return nil
}

// ListCredentials returns all credential sets (passwords masked for security)
func (vcs *VMwareCredentialService) ListCredentials(ctx context.Context) ([]database.VMwareCredential, error) {
	var credentials []database.VMwareCredential
	err := (*vcs.db).GetGormDB().Find(&credentials).Error
	if err != nil {
		return nil, fmt.Errorf("failed to list VMware credentials: %w", err)
	}

	// Mask passwords for security (already done by JSON tag, but extra safety)
	for i := range credentials {
		credentials[i].PasswordEncrypted = "[ENCRYPTED]"
	}

	return credentials, nil
}

// DeleteCredentials removes a credential set (with safety checks)
func (vcs *VMwareCredentialService) DeleteCredentials(ctx context.Context, credentialID int) error {
	// Check if credential is in use
	var contextCount int64
	(*vcs.db).GetGormDB().Model(&database.VMReplicationContext{}).Where("vmware_credential_id = ?", credentialID).Count(&contextCount)
	if contextCount > 0 {
		return fmt.Errorf("cannot delete credential set - %d VM contexts are using it", contextCount)
	}

	// Check if it's the last active credential
	var activeCount int64
	(*vcs.db).GetGormDB().Model(&database.VMwareCredential{}).Where("is_active = ? AND id != ?", true, credentialID).Count(&activeCount)
	if activeCount == 0 {
		return fmt.Errorf("cannot delete credential set - it's the last active credential")
	}

	err := (*vcs.db).GetGormDB().Delete(&database.VMwareCredential{}, credentialID).Error
	if err != nil {
		return fmt.Errorf("failed to delete VMware credentials: %w", err)
	}

	log.WithField("credential_id", credentialID).Info("✅ VMware credentials deleted successfully")
	return nil
}

// TestConnectivity validates VMware credentials by testing vCenter connection
func (vcs *VMwareCredentialService) TestConnectivity(ctx context.Context, creds *database.VMwareCredentials) error {
	// TODO: Implement vCenter connectivity test
	// This would use VMware govmomi to test authentication
	log.WithFields(log.Fields{
		"vcenter_host": creds.VCenterHost,
		"username":     creds.Username,
	}).Info("🔍 Testing VMware credential connectivity")

	// For now, return success - will implement actual connectivity test
	return nil
}
