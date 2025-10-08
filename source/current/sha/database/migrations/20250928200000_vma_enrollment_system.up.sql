-- Migration: VMA Enrollment System
-- Creates tables for secure VMA-OMA pairing with operator approval workflow
-- Date: September 28, 2025

-- Core enrollment tracking table
CREATE TABLE vma_enrollments (
    id VARCHAR(36) NOT NULL DEFAULT (UUID()),
    pairing_code VARCHAR(20) UNIQUE NOT NULL COMMENT 'Short-lived pairing code (AX7K-PJ3F-TH2Q format)',
    vma_public_key TEXT NOT NULL COMMENT 'Ed25519 public key from VMA',
    vma_name VARCHAR(255) COMMENT 'Human-readable VMA identifier',
    vma_version VARCHAR(100) COMMENT 'VMA software version',
    vma_fingerprint VARCHAR(255) COMMENT 'SSH key fingerprint for display',
    vma_ip_address VARCHAR(45) COMMENT 'Source IP address of enrollment request',
    challenge_nonce VARCHAR(64) COMMENT 'Cryptographic challenge for key verification',
    status ENUM('pending_verification', 'awaiting_approval', 'approved', 'rejected', 'expired') 
        NOT NULL DEFAULT 'pending_verification' COMMENT 'Enrollment workflow status',
    approved_by VARCHAR(255) COMMENT 'Admin user who approved this enrollment',
    approved_at TIMESTAMP NULL COMMENT 'When enrollment was approved',
    expires_at TIMESTAMP NOT NULL COMMENT 'When pairing code expires',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    PRIMARY KEY (id),
    INDEX idx_vma_enrollments_pairing_code (pairing_code),
    INDEX idx_vma_enrollments_status (status),
    INDEX idx_vma_enrollments_expires_at (expires_at),
    INDEX idx_vma_enrollments_created_at (created_at),
    INDEX idx_vma_enrollments_approved_by (approved_by)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci
COMMENT='VMA enrollment requests with operator approval workflow';

-- Audit trail for security and compliance
CREATE TABLE vma_connection_audit (
    id BIGINT AUTO_INCREMENT,
    enrollment_id VARCHAR(36) COMMENT 'Reference to vma_enrollments.id',
    event_type ENUM('enrollment', 'verification', 'approval', 'rejection', 'connection', 'disconnection', 'revocation')
        NOT NULL COMMENT 'Type of security event',
    vma_fingerprint VARCHAR(255) COMMENT 'SSH key fingerprint for correlation',
    source_ip VARCHAR(45) COMMENT 'Source IP address of event',
    user_agent VARCHAR(255) COMMENT 'User agent or client identifier',
    approved_by VARCHAR(255) COMMENT 'Admin user for approval/rejection events',
    event_details JSON COMMENT 'Additional event metadata',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    PRIMARY KEY (id),
    INDEX idx_vma_audit_enrollment_id (enrollment_id),
    INDEX idx_vma_audit_event_type (event_type),
    INDEX idx_vma_audit_created_at (created_at),
    INDEX idx_vma_audit_vma_fingerprint (vma_fingerprint),
    INDEX idx_vma_audit_source_ip (source_ip),
    FOREIGN KEY fk_vma_audit_enrollment (enrollment_id) 
        REFERENCES vma_enrollments(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci
COMMENT='Complete audit trail for VMA enrollment and connection events';

-- Active VMA connections tracking
CREATE TABLE vma_active_connections (
    id VARCHAR(36) NOT NULL DEFAULT (UUID()),
    enrollment_id VARCHAR(36) NOT NULL COMMENT 'Reference to approved enrollment',
    vma_name VARCHAR(255) NOT NULL COMMENT 'VMA identifier',
    vma_fingerprint VARCHAR(255) NOT NULL COMMENT 'SSH key fingerprint',
    ssh_user VARCHAR(50) NOT NULL DEFAULT 'vma_tunnel' COMMENT 'SSH user for tunnel connection',
    connection_status ENUM('connected', 'disconnected', 'revoked') 
        NOT NULL DEFAULT 'connected' COMMENT 'Current connection status',
    last_seen_at TIMESTAMP NULL COMMENT 'Last successful health check',
    connected_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    revoked_at TIMESTAMP NULL COMMENT 'When access was revoked',
    revoked_by VARCHAR(255) COMMENT 'Admin who revoked access',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    PRIMARY KEY (id),
    UNIQUE KEY unique_vma_connection (enrollment_id),
    INDEX idx_vma_connections_status (connection_status),
    INDEX idx_vma_connections_last_seen (last_seen_at),
    INDEX idx_vma_connections_fingerprint (vma_fingerprint),
    FOREIGN KEY fk_vma_connection_enrollment (enrollment_id) 
        REFERENCES vma_enrollments(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci
COMMENT='Active VMA tunnel connections for monitoring and management';

-- Pairing codes generation tracking (prevent replay)
CREATE TABLE vma_pairing_codes (
    id VARCHAR(36) NOT NULL DEFAULT (UUID()),
    pairing_code VARCHAR(20) UNIQUE NOT NULL COMMENT 'Generated pairing code',
    generated_by VARCHAR(255) NOT NULL COMMENT 'Admin who generated the code',
    used_by_enrollment_id VARCHAR(36) NULL COMMENT 'Which enrollment used this code',
    expires_at TIMESTAMP NOT NULL COMMENT 'Code expiry time',
    used_at TIMESTAMP NULL COMMENT 'When code was used',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    PRIMARY KEY (id),
    UNIQUE KEY unique_pairing_code (pairing_code),
    INDEX idx_pairing_codes_expires_at (expires_at),
    INDEX idx_pairing_codes_generated_by (generated_by),
    INDEX idx_pairing_codes_used_at (used_at),
    FOREIGN KEY fk_pairing_code_enrollment (used_by_enrollment_id) 
        REFERENCES vma_enrollments(id) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci
COMMENT='Pairing code generation and usage tracking for security audit';






