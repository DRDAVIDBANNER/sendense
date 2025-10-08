// Package services provides cryptographic operations for SNA enrollment system
package services

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"os/exec"
	"regexp"
	"strings"

	log "github.com/sirupsen/logrus"
)

// SNACryptoService handles cryptographic operations for SNA enrollment
type SNACryptoService struct{}

// NewVMACryptoService creates a new SNA crypto service
func NewVMACryptoService() *SNACryptoService {
	return &SNACryptoService{}
}

// GenerateChallenge creates a cryptographically secure challenge nonce
func (vcs *SNACryptoService) GenerateChallenge() (string, error) {
	// Generate 32-byte (256-bit) random nonce
	nonce := make([]byte, 32)
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("failed to generate random nonce: %w", err)
	}

	// Return base64-encoded nonce
	challenge := base64.StdEncoding.EncodeToString(nonce)

	log.WithField("challenge_length", len(challenge)).Debug("🔐 Generated cryptographic challenge")
	return challenge, nil
}

// VerifySignature verifies Ed25519 signature of challenge using SNA's public key
func (vcs *SNACryptoService) VerifySignature(publicKeySSH string, challenge string, signatureB64 string) (bool, error) {
	// Parse SSH public key to extract Ed25519 key
	publicKey, err := vcs.parseSSHEd25519PublicKey(publicKeySSH)
	if err != nil {
		return false, fmt.Errorf("failed to parse SSH public key: %w", err)
	}

	// Decode base64 challenge
	challengeBytes, err := base64.StdEncoding.DecodeString(challenge)
	if err != nil {
		return false, fmt.Errorf("failed to decode challenge: %w", err)
	}

	// Decode base64 signature
	signature, err := base64.StdEncoding.DecodeString(signatureB64)
	if err != nil {
		return false, fmt.Errorf("failed to decode signature: %w", err)
	}

	// Verify Ed25519 signature
	valid := ed25519.Verify(publicKey, challengeBytes, signature)

	log.WithFields(log.Fields{
		"signature_valid": valid,
		"challenge_size":  len(challengeBytes),
		"signature_size":  len(signature),
	}).Debug("🔐 Signature verification completed")

	return valid, nil
}

// GenerateSSHFingerprint creates SHA256 fingerprint of SSH public key for display
func (vcs *SNACryptoService) GenerateSSHFingerprint(publicKeySSH string) (string, error) {
	// Extract the base64 key portion from SSH format
	parts := strings.Fields(publicKeySSH)
	if len(parts) < 2 {
		return "", fmt.Errorf("invalid SSH public key format")
	}

	keyType := parts[0]
	keyData := parts[1]

	// Verify it's Ed25519
	if keyType != "ssh-ed25519" {
		return "", fmt.Errorf("unsupported key type: %s (only ssh-ed25519 supported)", keyType)
	}

	// Decode base64 key data
	keyBytes, err := base64.StdEncoding.DecodeString(keyData)
	if err != nil {
		return "", fmt.Errorf("failed to decode SSH key: %w", err)
	}

	// Generate SHA256 fingerprint
	hash := sha256.Sum256(keyBytes)
	fingerprint := "SHA256:" + base64.StdEncoding.EncodeToString(hash[:])

	log.WithFields(log.Fields{
		"key_type":    keyType,
		"fingerprint": fingerprint,
		"key_size":    len(keyBytes),
	}).Debug("🔑 Generated SSH key fingerprint")

	return fingerprint, nil
}

// parseSSHEd25519PublicKey extracts Ed25519 public key from SSH format
func (vcs *SNACryptoService) parseSSHEd25519PublicKey(publicKeySSH string) (ed25519.PublicKey, error) {
	// Parse SSH public key format: "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5... [comment]"
	parts := strings.Fields(publicKeySSH)
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid SSH public key format")
	}

	keyType := parts[0]
	keyData := parts[1]

	// Verify key type
	if keyType != "ssh-ed25519" {
		return nil, fmt.Errorf("unsupported key type: %s (only ssh-ed25519 supported)", keyType)
	}

	// Decode base64 key data
	keyBytes, err := base64.StdEncoding.DecodeString(keyData)
	if err != nil {
		return nil, fmt.Errorf("failed to decode SSH key data: %w", err)
	}

	// SSH Ed25519 format: 4 bytes length + "ssh-ed25519" + 4 bytes length + 32-byte key
	// Skip SSH wire format headers and extract the 32-byte Ed25519 key
	if len(keyBytes) < 51 { // Minimum SSH Ed25519 wire format size
		return nil, fmt.Errorf("SSH key data too short for Ed25519")
	}

	// Extract the actual Ed25519 public key (last 32 bytes)
	ed25519Key := keyBytes[len(keyBytes)-32:]

	if len(ed25519Key) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("invalid Ed25519 key size: %d (expected %d)", len(ed25519Key), ed25519.PublicKeySize)
	}

	return ed25519.PublicKey(ed25519Key), nil
}

// GetSSHHostKeyFingerprint retrieves the SHA server's SSH host key fingerprint
func (vcs *SNACryptoService) GetSSHHostKeyFingerprint() (string, error) {
	// Get Ed25519 host key fingerprint using ssh-keygen
	cmd := exec.Command("ssh-keygen", "-l", "-f", "/etc/ssh/ssh_host_ed25519_key.pub")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get SSH host key fingerprint: %w", err)
	}

	// Parse output: "256 SHA256:abc123... root@hostname (ED25519)"
	fingerprintRegex := regexp.MustCompile(`SHA256:([A-Za-z0-9+/=]+)`)
	matches := fingerprintRegex.FindStringSubmatch(string(output))
	if len(matches) < 2 {
		return "", fmt.Errorf("failed to parse SSH host key fingerprint from output: %s", string(output))
	}

	fingerprint := "SHA256:" + matches[1]

	log.WithField("host_key_fingerprint", fingerprint).Debug("🔑 Retrieved SSH host key fingerprint")
	return fingerprint, nil
}

// ValidateSSHPublicKey validates that a string is a valid SSH Ed25519 public key
func (vcs *SNACryptoService) ValidateSSHPublicKey(publicKeySSH string) error {
	// Basic format validation
	if !strings.HasPrefix(publicKeySSH, "ssh-ed25519 ") {
		return fmt.Errorf("SSH key must start with 'ssh-ed25519 '")
	}

	// Try to parse the key
	_, err := vcs.parseSSHEd25519PublicKey(publicKeySSH)
	if err != nil {
		return fmt.Errorf("invalid SSH Ed25519 public key: %w", err)
	}

	return nil
}

// GenerateKeyPair generates a new Ed25519 keypair (for SNA client use)
func (vcs *SNACryptoService) GenerateKeyPair() (ed25519.PublicKey, ed25519.PrivateKey, error) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate Ed25519 keypair: %w", err)
	}

	log.Debug("🔑 Generated new Ed25519 keypair")
	return publicKey, privateKey, nil
}

// SignChallenge signs a challenge with Ed25519 private key (for SNA client use)
func (vcs *SNACryptoService) SignChallenge(privateKey ed25519.PrivateKey, challenge string) (string, error) {
	// Decode challenge
	challengeBytes, err := base64.StdEncoding.DecodeString(challenge)
	if err != nil {
		return "", fmt.Errorf("failed to decode challenge: %w", err)
	}

	// Sign challenge
	signature := ed25519.Sign(privateKey, challengeBytes)

	// Return base64-encoded signature
	signatureB64 := base64.StdEncoding.EncodeToString(signature)

	log.WithFields(log.Fields{
		"challenge_size": len(challengeBytes),
		"signature_size": len(signature),
	}).Debug("🔐 Signed challenge with Ed25519 private key")

	return signatureB64, nil
}

// FormatSSHPublicKey formats Ed25519 public key as SSH public key string
func (vcs *SNACryptoService) FormatSSHPublicKey(publicKey ed25519.PublicKey, comment string) string {
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
	if comment != "" {
		return fmt.Sprintf("%s %s %s", keyType, keyData, comment)
	}
	return fmt.Sprintf("%s %s", keyType, keyData)
}






