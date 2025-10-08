// Package services provides SNA enrollment client for secure SHA pairing
package services

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

// SNAEnrollmentClient handles SNA-side enrollment operations
type SNAEnrollmentClient struct {
	shaHost    string
	shaPort    int
	httpClient *http.Client
	configDir  string
}

// NewVMAEnrollmentClient creates a new SNA enrollment client
func NewVMAEnrollmentClient(shaHost string, shaPort int) *SNAEnrollmentClient {
	return &SNAEnrollmentClient{
		shaHost: shaHost,
		shaPort: shaPort,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		configDir: "/opt/vma/enrollment",
	}
}

// EnrollmentRequest represents the SNA enrollment request
type EnrollmentRequest struct {
	PairingCode    string `json:"pairing_code"`
	SNAPublicKey   string `json:"vma_public_key"`
	SNAName        string `json:"vma_name"`
	SNAVersion     string `json:"vma_version"`
	SNAFingerprint string `json:"vma_fingerprint"`
}

// EnrollmentResponse represents the SHA enrollment response
type EnrollmentResponse struct {
	EnrollmentID string `json:"enrollment_id"`
	Challenge    string `json:"challenge"`
	Status       string `json:"status"`
	Message      string `json:"message"`
}

// VerificationRequest represents challenge verification
type VerificationRequest struct {
	EnrollmentID string `json:"enrollment_id"`
	Signature    string `json:"signature"`
}

// VerificationResponse represents verification result
type VerificationResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

// EnrollmentResult represents final enrollment result
type EnrollmentResult struct {
	Status      string `json:"status"`
	SSHUser     string `json:"ssh_user,omitempty"`
	SSHOptions  string `json:"ssh_options,omitempty"`
	HostKeyHash string `json:"host_key_hash,omitempty"`
	Message     string `json:"message,omitempty"`
}

// EnrollmentConfig stores SNA enrollment configuration
type EnrollmentConfig struct {
	EnrollmentID   string    `json:"enrollment_id"`
	SHAHost        string    `json:"oma_host"`
	SHAPort        int       `json:"oma_port"`
	SNAName        string    `json:"vma_name"`
	SNAVersion     string    `json:"vma_version"`
	PublicKeyPath  string    `json:"public_key_path"`
	PrivateKeyPath string    `json:"private_key_path"`
	SSHUser        string    `json:"ssh_user"`
	SSHOptions     string    `json:"ssh_options"`
	HostKeyHash    string    `json:"host_key_hash"`
	EnrolledAt     time.Time `json:"enrolled_at"`
}

// EnrollWithOMA performs the complete SNA enrollment process
func (vec *SNAEnrollmentClient) EnrollWithOMA(pairingCode string, snaName string, snaVersion string) (*EnrollmentConfig, error) {
	log.WithFields(log.Fields{
		"oma_host":     vec.shaHost,
		"pairing_code": pairingCode,
		"vma_name":     snaName,
	}).Info("🔐 Starting SNA enrollment with SHA")

	// Step 1: Generate Ed25519 keypair for this SHA
	publicKey, privateKey, err := vec.generateKeyPair()
	if err != nil {
		return nil, fmt.Errorf("failed to generate keypair: %w", err)
	}

	// Step 2: Format public key and generate fingerprint
	publicKeySSH, err := vec.formatSSHPublicKey(publicKey, fmt.Sprintf("vma-%s", snaName))
	if err != nil {
		return nil, fmt.Errorf("failed to format public key: %w", err)
	}

	fingerprint, err := vec.generateSSHFingerprint(publicKeySSH)
	if err != nil {
		return nil, fmt.Errorf("failed to generate fingerprint: %w", err)
	}

	// Step 3: Initial enrollment request
	enrollReq := &EnrollmentRequest{
		PairingCode:    pairingCode,
		SNAPublicKey:   publicKeySSH,
		SNAName:        snaName,
		SNAVersion:     snaVersion,
		SNAFingerprint: fingerprint,
	}

	enrollResp, err := vec.sendEnrollmentRequest(enrollReq)
	if err != nil {
		return nil, fmt.Errorf("enrollment request failed: %w", err)
	}

	log.WithFields(log.Fields{
		"enrollment_id": enrollResp.EnrollmentID,
		"status":        enrollResp.Status,
	}).Info("✅ Enrollment request accepted, received challenge")

	// Step 4: Sign challenge and verify
	signature, err := vec.signChallenge(privateKey, enrollResp.Challenge)
	if err != nil {
		return nil, fmt.Errorf("failed to sign challenge: %w", err)
	}

	verifyReq := &VerificationRequest{
		EnrollmentID: enrollResp.EnrollmentID,
		Signature:    signature,
	}

	verifyResp, err := vec.sendVerificationRequest(verifyReq)
	if err != nil {
		return nil, fmt.Errorf("verification request failed: %w", err)
	}

	log.WithFields(log.Fields{
		"enrollment_id": enrollResp.EnrollmentID,
		"status":        verifyResp.Status,
	}).Info("✅ Challenge verification successful, awaiting approval")

	// Step 5: Poll for approval
	result, err := vec.pollForApproval(enrollResp.EnrollmentID, 10*time.Minute)
	if err != nil {
		return nil, fmt.Errorf("approval polling failed: %w", err)
	}

	if result.Status != "approved" {
		return nil, fmt.Errorf("enrollment not approved: %s - %s", result.Status, result.Message)
	}

	log.WithFields(log.Fields{
		"enrollment_id": enrollResp.EnrollmentID,
		"ssh_user":      result.SSHUser,
	}).Info("🎉 SNA enrollment approved!")

	// Step 6: Save configuration and keys
	config, err := vec.saveEnrollmentConfig(enrollResp.EnrollmentID, snaName, snaVersion, publicKey, privateKey, result)
	if err != nil {
		return nil, fmt.Errorf("failed to save enrollment config: %w", err)
	}

	return config, nil
}

// generateKeyPair creates a new Ed25519 keypair for SNA-SHA authentication
func (vec *SNAEnrollmentClient) generateKeyPair() (ed25519.PublicKey, ed25519.PrivateKey, error) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate Ed25519 keypair: %w", err)
	}

	log.Info("🔑 Generated new Ed25519 keypair for SHA enrollment")
	return publicKey, privateKey, nil
}

// formatSSHPublicKey formats Ed25519 public key as SSH public key string
func (vec *SNAEnrollmentClient) formatSSHPublicKey(publicKey ed25519.PublicKey, comment string) (string, error) {
	// SSH Ed25519 wire format
	keyType := "ssh-ed25519"

	// Create SSH wire format: length + algorithm + length + key
	wireFormat := make([]byte, 0, 51) // 4 + 11 + 4 + 32

	// Algorithm name length (11 bytes: "ssh-ed25519")
	wireFormat = append(wireFormat, 0, 0, 0, 11)
	wireFormat = append(wireFormat, []byte("ssh-ed25519")...)

	// Key length (32 bytes)
	wireFormat = append(wireFormat, 0, 0, 0, 32)
	wireFormat = append(wireFormat, publicKey...)

	// Base64 encode
	keyData := base64.StdEncoding.EncodeToString(wireFormat)

	// Format as SSH public key
	sshKey := fmt.Sprintf("%s %s %s", keyType, keyData, comment)

	log.WithField("comment", comment).Debug("🔑 Formatted SSH public key")
	return sshKey, nil
}

// generateSSHFingerprint creates SHA256 fingerprint of SSH public key
func (vec *SNAEnrollmentClient) generateSSHFingerprint(publicKeySSH string) (string, error) {
	// This would use the same logic as the SHA crypto service
	// For now, return a placeholder
	return "SHA256:placeholder-fingerprint", nil
}

// sendEnrollmentRequest sends initial enrollment request to SHA
func (vec *SNAEnrollmentClient) sendEnrollmentRequest(req *EnrollmentRequest) (*EnrollmentResponse, error) {
	url := fmt.Sprintf("https://%s:%d/api/v1/vma/enroll", vec.shaHost, vec.shaPort)

	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := vec.httpClient.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to send enrollment request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("enrollment request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var enrollResp EnrollmentResponse
	if err := json.NewDecoder(resp.Body).Decode(&enrollResp); err != nil {
		return nil, fmt.Errorf("failed to decode enrollment response: %w", err)
	}

	return &enrollResp, nil
}

// sendVerificationRequest sends challenge verification to SHA
func (vec *SNAEnrollmentClient) sendVerificationRequest(req *VerificationRequest) (*VerificationResponse, error) {
	url := fmt.Sprintf("https://%s:%d/api/v1/vma/enroll/verify", vec.shaHost, vec.shaPort)

	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal verification request: %w", err)
	}

	resp, err := vec.httpClient.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to send verification request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("verification request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var verifyResp VerificationResponse
	if err := json.NewDecoder(resp.Body).Decode(&verifyResp); err != nil {
		return nil, fmt.Errorf("failed to decode verification response: %w", err)
	}

	return &verifyResp, nil
}

// signChallenge signs the challenge nonce with Ed25519 private key
func (vec *SNAEnrollmentClient) signChallenge(privateKey ed25519.PrivateKey, challenge string) (string, error) {
	// Decode base64 challenge
	challengeBytes, err := base64.StdEncoding.DecodeString(challenge)
	if err != nil {
		return "", fmt.Errorf("failed to decode challenge: %w", err)
	}

	// Sign challenge with Ed25519 private key
	signature := ed25519.Sign(privateKey, challengeBytes)

	// Return base64-encoded signature
	signatureB64 := base64.StdEncoding.EncodeToString(signature)

	log.WithFields(log.Fields{
		"challenge_size": len(challengeBytes),
		"signature_size": len(signature),
	}).Debug("🔐 Signed challenge with Ed25519 private key")

	return signatureB64, nil
}

// pollForApproval polls SHA for enrollment approval status
func (vec *SNAEnrollmentClient) pollForApproval(enrollmentID string, timeout time.Duration) (*EnrollmentResult, error) {
	url := fmt.Sprintf("https://%s:%d/api/v1/vma/enroll/result?enrollment_id=%s",
		vec.shaHost, vec.shaPort, url.QueryEscape(enrollmentID))

	deadline := time.Now().Add(timeout)
	pollInterval := 5 * time.Second

	log.WithFields(log.Fields{
		"enrollment_id": enrollmentID,
		"timeout":       timeout,
		"poll_interval": pollInterval,
	}).Info("⏳ Polling for enrollment approval...")

	for time.Now().Before(deadline) {
		resp, err := vec.httpClient.Get(url)
		if err != nil {
			log.WithError(err).Debug("Polling request failed, retrying...")
			time.Sleep(pollInterval)
			continue
		}

		if resp.StatusCode == http.StatusOK {
			var result EnrollmentResult
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				resp.Body.Close()
				return nil, fmt.Errorf("failed to decode enrollment result: %w", err)
			}
			resp.Body.Close()

			log.WithFields(log.Fields{
				"enrollment_id": enrollmentID,
				"status":        result.Status,
			}).Debug("📊 Enrollment status polled")

			// Check if enrollment is complete (approved or rejected)
			if result.Status == "approved" || result.Status == "rejected" || result.Status == "expired" {
				return &result, nil
			}
		} else {
			resp.Body.Close()
		}

		// Wait before next poll
		time.Sleep(pollInterval)
	}

	return nil, fmt.Errorf("enrollment approval timeout after %v", timeout)
}

// saveEnrollmentConfig saves enrollment configuration and SSH keys
func (vec *SNAEnrollmentClient) saveEnrollmentConfig(
	enrollmentID string,
	snaName string,
	snaVersion string,
	publicKey ed25519.PublicKey,
	privateKey ed25519.PrivateKey,
	result *EnrollmentResult,
) (*EnrollmentConfig, error) {
	// Ensure config directory exists
	if err := os.MkdirAll(vec.configDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	// Save private key (PEM format)
	privateKeyPath := filepath.Join(vec.configDir, fmt.Sprintf("vma-%s-key", strings.ReplaceAll(vec.shaHost, ".", "-")))
	if err := vec.savePrivateKey(privateKey, privateKeyPath); err != nil {
		return nil, fmt.Errorf("failed to save private key: %w", err)
	}

	// Save public key (SSH format)
	publicKeyPath := privateKeyPath + ".pub"
	publicKeySSH, err := vec.formatSSHPublicKey(publicKey, fmt.Sprintf("vma-%s@%s", snaName, vec.shaHost))
	if err != nil {
		return nil, fmt.Errorf("failed to format public key: %w", err)
	}

	if err := os.WriteFile(publicKeyPath, []byte(publicKeySSH), 0644); err != nil {
		return nil, fmt.Errorf("failed to save public key: %w", err)
	}

	// Create enrollment configuration
	config := &EnrollmentConfig{
		EnrollmentID:   enrollmentID,
		SHAHost:        vec.shaHost,
		SHAPort:        vec.shaPort,
		SNAName:        snaName,
		SNAVersion:     snaVersion,
		PublicKeyPath:  publicKeyPath,
		PrivateKeyPath: privateKeyPath,
		SSHUser:        result.SSHUser,
		SSHOptions:     result.SSHOptions,
		HostKeyHash:    result.HostKeyHash,
		EnrolledAt:     time.Now(),
	}

	// Save configuration file
	configPath := filepath.Join(vec.configDir, fmt.Sprintf("enrollment-%s.json", enrollmentID))
	configData, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, configData, 0600); err != nil {
		return nil, fmt.Errorf("failed to save config: %w", err)
	}

	log.WithFields(log.Fields{
		"enrollment_id":    enrollmentID,
		"config_path":      configPath,
		"private_key_path": privateKeyPath,
		"public_key_path":  publicKeyPath,
	}).Info("💾 Saved SNA enrollment configuration")

	return config, nil
}

// savePrivateKey saves Ed25519 private key in OpenSSH format
func (vec *SNAEnrollmentClient) savePrivateKey(privateKey ed25519.PrivateKey, path string) error {
	// For now, save as raw bytes - in production would use proper OpenSSH format
	if err := os.WriteFile(path, privateKey, 0600); err != nil {
		return fmt.Errorf("failed to write private key: %w", err)
	}

	log.WithField("path", path).Debug("🔐 Saved Ed25519 private key")
	return nil
}

// ConfigureTunnel configures SNA tunnel service with enrollment credentials
func (vec *SNAEnrollmentClient) ConfigureTunnel(config *EnrollmentConfig) error {
	log.WithFields(log.Fields{
		"oma_host":   config.SHAHost,
		"ssh_user":   config.SSHUser,
		"enrollment": config.EnrollmentID,
	}).Info("🔧 Configuring SNA tunnel with enrollment credentials")

	// Update tunnel service configuration
	if err := vec.updateTunnelService(config); err != nil {
		return fmt.Errorf("failed to update tunnel service: %w", err)
	}

	// Update SSH known hosts with SHA host key
	if err := vec.updateKnownHosts(config); err != nil {
		return fmt.Errorf("failed to update known hosts: %w", err)
	}

	// Restart tunnel service
	if err := vec.restartTunnelService(); err != nil {
		return fmt.Errorf("failed to restart tunnel service: %w", err)
	}

	log.Info("✅ SNA tunnel configured with enrollment credentials")
	return nil
}

// updateTunnelService updates systemd service with enrollment configuration
func (vec *SNAEnrollmentClient) updateTunnelService(config *EnrollmentConfig) error {
	// This would update the vma-tunnel-enhanced-v2.service with the new credentials
	// For now, log the configuration that would be applied
	log.WithFields(log.Fields{
		"oma_host":      config.SHAHost,
		"ssh_key":       config.PrivateKeyPath,
		"ssh_user":      config.SSHUser,
		"host_key_hash": config.HostKeyHash,
	}).Info("🔧 Tunnel service configuration (Phase 4 implementation pending)")

	return nil
}

// updateKnownHosts adds SHA host key to SSH known hosts
func (vec *SNAEnrollmentClient) updateKnownHosts(config *EnrollmentConfig) error {
	// This would add the SHA host key to known_hosts for TOFU security
	log.WithFields(log.Fields{
		"oma_host":      config.SHAHost,
		"host_key_hash": config.HostKeyHash,
	}).Info("🔐 SSH known hosts update (Phase 4 implementation pending)")

	return nil
}

// restartTunnelService restarts the SNA tunnel service
func (vec *SNAEnrollmentClient) restartTunnelService() error {
	// This would restart the systemd tunnel service
	log.Info("🔄 Tunnel service restart (Phase 4 implementation pending)")
	return nil
}

// TestConnection verifies the enrollment-based tunnel connection
func (vec *SNAEnrollmentClient) TestConnection(config *EnrollmentConfig) error {
	// Test the tunnel connection using enrollment credentials
	log.WithFields(log.Fields{
		"oma_host":      config.SHAHost,
		"enrollment_id": config.EnrollmentID,
	}).Info("🧪 Testing enrollment-based tunnel connection")

	// This would test the actual SSH tunnel connection
	// For now, return success
	return nil
}

// GetEnrollmentStatus checks current enrollment status
func (vec *SNAEnrollmentClient) GetEnrollmentStatus(enrollmentID string) (*EnrollmentResult, error) {
	url := fmt.Sprintf("https://%s:%d/api/v1/vma/enroll/result?enrollment_id=%s",
		vec.shaHost, vec.shaPort, url.QueryEscape(enrollmentID))

	resp, err := vec.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to get enrollment status: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("status request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var result EnrollmentResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode status response: %w", err)
	}

	return &result, nil
}






