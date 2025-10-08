// Package services provides SNA SSH key management for enrollment system
package services

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

// SNASSHManager handles SSH key management for SNA tunnel access
type SNASSHManager struct {
	snaUser            string
	snaUserHome        string
	authorizedKeysPath string
	backupPath         string
}

// SNASSHKey represents an SSH key entry
type SNASSHKey struct {
	Fingerprint  string    `json:"fingerprint"`
	PublicKey    string    `json:"public_key"`
	Restrictions string    `json:"restrictions"`
	AddedAt      time.Time `json:"added_at"`
}

// NewVMASSHManager creates a new SNA SSH manager
func NewVMASSHManager() (*SNASSHManager, error) {
	const snaUserName = "vma_tunnel"

	// Check if vma_tunnel user exists
	snaUser, err := user.Lookup(snaUserName)
	if err != nil {
		// User doesn't exist, create it
		if err := createVMATunnelUser(snaUserName); err != nil {
			return nil, fmt.Errorf("failed to create vma_tunnel user: %w", err)
		}

		// Re-lookup after creation
		snaUser, err = user.Lookup(snaUserName)
		if err != nil {
			return nil, fmt.Errorf("failed to lookup vma_tunnel user after creation: %w", err)
		}
	}

	snaUserHome := snaUser.HomeDir
	authorizedKeysPath := filepath.Join(snaUserHome, ".ssh", "authorized_keys")
	backupPath := filepath.Join(snaUserHome, ".ssh", "authorized_keys.backup")

	manager := &SNASSHManager{
		snaUser:            snaUserName,
		snaUserHome:        snaUserHome,
		authorizedKeysPath: authorizedKeysPath,
		backupPath:         backupPath,
	}

	// Ensure SSH directory exists with proper permissions
	if err := manager.ensureSSHDirectory(); err != nil {
		return nil, fmt.Errorf("failed to setup SSH directory: %w", err)
	}

	log.WithFields(log.Fields{
		"vma_user":      snaUserName,
		"home_dir":      snaUserHome,
		"ssh_keys_path": authorizedKeysPath,
	}).Info("🔑 SNA SSH Manager initialized")

	return manager, nil
}

// createVMATunnelUser creates the vma_tunnel system user
func createVMATunnelUser(username string) error {
	log.WithField("username", username).Info("🏗️ Creating vma_tunnel system user")

	// Create system user with home directory using sudo
	cmd := exec.Command("sudo", "useradd",
		"-r",                        // System user
		"-m",                        // Create home directory
		"-d", "/var/lib/vma_tunnel", // Home directory
		"-s", "/bin/false", // No shell access
		"-c", "SNA SSH Tunnel User", // Comment
		username)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create user %s: %w", username, err)
	}

	log.WithField("username", username).Info("✅ SNA tunnel user created successfully")
	return nil
}

// ensureSSHDirectory creates SSH directory with proper permissions using sudo
func (vsm *SNASSHManager) ensureSSHDirectory() error {
	sshDir := filepath.Join(vsm.snaUserHome, ".ssh")

	// Create .ssh directory using sudo if it doesn't exist
	if _, err := os.Stat(sshDir); os.IsNotExist(err) {
		cmd := exec.Command("sudo", "mkdir", "-p", sshDir)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to create SSH directory with sudo: %w", err)
		}
	}

	// Set proper ownership using sudo (vma_tunnel:vma_tunnel)
	cmd := exec.Command("sudo", "chown", "vma_tunnel", sshDir)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to set SSH directory ownership: %w", err)
	}

	// Set proper permissions using sudo
	cmd = exec.Command("sudo", "chmod", "700", sshDir)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to set SSH directory permissions: %w", err)
	}

	log.WithField("ssh_dir", sshDir).Debug("🔒 SSH directory configured with proper permissions using sudo")
	return nil
}

// AddVMAKey adds a SNA SSH public key with tunnel restrictions
func (vsm *SNASSHManager) AddVMAKey(publicKey, fingerprint string) error {
	log.WithFields(log.Fields{
		"fingerprint":     fingerprint[:16] + "...", // Truncate for logging
		"key_type":        "ed25519",
		"authorized_keys": vsm.authorizedKeysPath,
	}).Info("🔑 Adding SNA SSH key")

	// Validate SSH key format
	if !strings.HasPrefix(publicKey, "ssh-ed25519 ") {
		return fmt.Errorf("invalid SSH key format - only Ed25519 keys supported")
	}

	// Create backup of existing authorized_keys
	if err := vsm.backupAuthorizedKeys(); err != nil {
		log.WithError(err).Warn("Failed to backup authorized_keys file")
	}

	// Build SSH key entry with restrictions
	restrictions := `command="/usr/local/sbin/oma_tunnel_wrapper.sh",restrict,permitopen="127.0.0.1:10809",permitopen="127.0.0.1:8081"`
	keyEntry := fmt.Sprintf("%s %s # SNA enrollment key - %s\n", restrictions, publicKey, fingerprint)

	// Read existing keys
	existingKeys, err := vsm.readAuthorizedKeys()
	if err != nil {
		log.WithError(err).Debug("No existing authorized_keys file")
		existingKeys = ""
	}

	// Check if key already exists
	if strings.Contains(existingKeys, fingerprint) {
		log.WithField("fingerprint", fingerprint[:16]+"...").Warn("SSH key already exists")
		return nil // Not an error, just skip
	}

	// Write updated keys atomically
	tempFile := vsm.authorizedKeysPath + ".tmp"
	file, err := os.OpenFile(tempFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("failed to create temporary authorized_keys file: %w", err)
	}
	defer file.Close()

	// Write existing keys + new key
	if _, err := file.WriteString(existingKeys); err != nil {
		return fmt.Errorf("failed to write existing keys: %w", err)
	}
	if _, err := file.WriteString(keyEntry); err != nil {
		return fmt.Errorf("failed to write new key: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tempFile, vsm.authorizedKeysPath); err != nil {
		return fmt.Errorf("failed to update authorized_keys file: %w", err)
	}

	// Set proper ownership and permissions using sudo
	cmd := exec.Command("sudo", "chown", "vma_tunnel", vsm.authorizedKeysPath)
	if err := cmd.Run(); err != nil {
		log.WithError(err).Warn("Failed to set authorized_keys ownership")
	}

	if err := os.Chmod(vsm.authorizedKeysPath, 0600); err != nil {
		log.WithError(err).Warn("Failed to set authorized_keys permissions")
	}

	log.WithFields(log.Fields{
		"fingerprint":     fingerprint[:16] + "...",
		"authorized_keys": vsm.authorizedKeysPath,
		"restrictions":    "tunnel access only",
	}).Info("✅ SNA SSH key added successfully")

	return nil
}

// RemoveVMAKey removes a SNA SSH key by fingerprint
func (vsm *SNASSHManager) RemoveVMAKey(fingerprint string) error {
	log.WithFields(log.Fields{
		"fingerprint":     fingerprint[:16] + "...",
		"authorized_keys": vsm.authorizedKeysPath,
	}).Info("🗑️ Removing SNA SSH key")

	// Create backup
	if err := vsm.backupAuthorizedKeys(); err != nil {
		log.WithError(err).Warn("Failed to backup authorized_keys before removal")
	}

	// Read existing keys
	existingKeys, err := vsm.readAuthorizedKeys()
	if err != nil {
		return fmt.Errorf("failed to read authorized_keys: %w", err)
	}

	// Filter out the key to remove
	var filteredKeys []string
	scanner := bufio.NewScanner(strings.NewReader(existingKeys))
	keyRemoved := false

	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, fingerprint) {
			log.WithField("removed_line", line[:50]+"...").Debug("🗑️ Removing SSH key line")
			keyRemoved = true
			continue // Skip this line
		}
		if line != "" { // Skip empty lines
			filteredKeys = append(filteredKeys, line)
		}
	}

	if !keyRemoved {
		log.WithField("fingerprint", fingerprint[:16]+"...").Warn("SSH key not found for removal")
		return nil // Not an error, just skip
	}

	// Write filtered keys atomically
	tempFile := vsm.authorizedKeysPath + ".tmp"
	file, err := os.OpenFile(tempFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("failed to create temporary authorized_keys file: %w", err)
	}
	defer file.Close()

	for _, keyLine := range filteredKeys {
		if _, err := file.WriteString(keyLine + "\n"); err != nil {
			return fmt.Errorf("failed to write filtered keys: %w", err)
		}
	}

	// Atomic rename
	if err := os.Rename(tempFile, vsm.authorizedKeysPath); err != nil {
		return fmt.Errorf("failed to update authorized_keys file: %w", err)
	}

	log.WithField("fingerprint", fingerprint[:16]+"...").Info("✅ SNA SSH key removed successfully")
	return nil
}

// readAuthorizedKeys reads the current authorized_keys file
func (vsm *SNASSHManager) readAuthorizedKeys() (string, error) {
	content, err := os.ReadFile(vsm.authorizedKeysPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil // File doesn't exist, return empty
		}
		return "", fmt.Errorf("failed to read authorized_keys: %w", err)
	}
	return string(content), nil
}

// backupAuthorizedKeys creates a backup of the authorized_keys file
func (vsm *SNASSHManager) backupAuthorizedKeys() error {
	if _, err := os.Stat(vsm.authorizedKeysPath); os.IsNotExist(err) {
		return nil // No file to backup
	}

	content, err := os.ReadFile(vsm.authorizedKeysPath)
	if err != nil {
		return fmt.Errorf("failed to read authorized_keys for backup: %w", err)
	}

	if err := os.WriteFile(vsm.backupPath, content, 0600); err != nil {
		return fmt.Errorf("failed to create authorized_keys backup: %w", err)
	}

	log.WithField("backup_path", vsm.backupPath).Debug("💾 Created authorized_keys backup")
	return nil
}

// ListInstalledKeys returns list of currently installed SNA keys
func (vsm *SNASSHManager) ListInstalledKeys() ([]SNASSHKey, error) {
	content, err := vsm.readAuthorizedKeys()
	if err != nil {
		return nil, fmt.Errorf("failed to read authorized_keys: %w", err)
	}

	var keys []SNASSHKey
	scanner := bufio.NewScanner(strings.NewReader(content))

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse SSH key line
		if strings.Contains(line, "SNA enrollment key") {
			parts := strings.Split(line, " ")
			if len(parts) >= 3 {
				// Extract fingerprint from comment
				fingerprint := "unknown"
				if commentIdx := strings.Index(line, "# SNA enrollment key - "); commentIdx != -1 {
					fingerprint = line[commentIdx+len("# SNA enrollment key - "):]
				}

				keys = append(keys, SNASSHKey{
					Fingerprint:  fingerprint,
					PublicKey:    strings.Join(parts[len(parts)-3:len(parts)-1], " "), // Get key part
					Restrictions: strings.Join(parts[:len(parts)-3], " "),             // Get restrictions
					AddedAt:      time.Now(),                                          // Would need metadata file for actual time
				})
			}
		}
	}

	log.WithField("key_count", len(keys)).Debug("📋 Listed installed SNA SSH keys")
	return keys, nil
}

// CreateTunnelWrapperScript creates the tunnel wrapper script if it doesn't exist
func (vsm *SNASSHManager) CreateTunnelWrapperScript() error {
	wrapperPath := "/usr/local/sbin/oma_tunnel_wrapper.sh"

	// Check if wrapper already exists
	if _, err := os.Stat(wrapperPath); err == nil {
		log.WithField("wrapper_path", wrapperPath).Debug("🔧 Tunnel wrapper script already exists")
		return nil
	}

	wrapperContent := `#!/bin/bash
# SHA Tunnel Wrapper Script for SNA SSH Connections
# Logs SNA connections and allows tunnel forwarding

# Log connection
if [ -n "$SSH_CLIENT" ]; then
    echo "$(date): SNA tunnel connection from $SSH_CLIENT" >> /var/log/vma-connections.log
fi

# Log the command being executed
if [ $# -gt 0 ]; then
    echo "$(date): SNA tunnel command: $*" >> /var/log/vma-connections.log
fi

# Allow SSH tunnel forwarding by executing the command
exec "$@"
`

	// Write wrapper script
	if err := os.WriteFile(wrapperPath, []byte(wrapperContent), 0755); err != nil {
		return fmt.Errorf("failed to create tunnel wrapper script: %w", err)
	}

	log.WithField("wrapper_path", wrapperPath).Info("✅ Created tunnel wrapper script")
	return nil
}

// GetVMATunnelUserInfo returns information about the vma_tunnel user
func (vsm *SNASSHManager) GetVMATunnelUserInfo() (map[string]interface{}, error) {
	snaUser, err := user.Lookup(vsm.snaUser)
	if err != nil {
		return nil, fmt.Errorf("vma_tunnel user not found: %w", err)
	}

	// Count installed keys
	keys, err := vsm.ListInstalledKeys()
	if err != nil {
		log.WithError(err).Warn("Failed to count installed keys")
		keys = []SNASSHKey{} // Continue with empty list
	}

	info := map[string]interface{}{
		"username":         snaUser.Username,
		"uid":              snaUser.Uid,
		"gid":              snaUser.Gid,
		"home_dir":         snaUser.HomeDir,
		"installed_keys":   len(keys),
		"authorized_keys":  vsm.authorizedKeysPath,
		"ssh_restrictions": "tunnel access only",
	}

	return info, nil
}
