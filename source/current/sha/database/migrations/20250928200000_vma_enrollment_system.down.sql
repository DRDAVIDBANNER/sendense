-- Migration rollback: VMA Enrollment System
-- Removes VMA enrollment tables
-- Date: September 28, 2025

-- Drop tables in reverse dependency order
DROP TABLE IF EXISTS vma_pairing_codes;
DROP TABLE IF EXISTS vma_active_connections;
DROP TABLE IF EXISTS vma_connection_audit;
DROP TABLE IF EXISTS vma_enrollments;






