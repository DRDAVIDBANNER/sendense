-- Migration: add_oma_vm_id (rollback)
-- Created: 20250115140000
-- Remove OMA VM ID field from ossea_configs table

ALTER TABLE ossea_configs 
DROP COLUMN oma_vm_id;