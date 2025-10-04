-- Migration: add_oma_vm_id
-- Created: 20250115140000
-- Add OMA VM ID field to ossea_configs table

ALTER TABLE ossea_configs 
ADD COLUMN oma_vm_id VARCHAR(255) COMMENT 'The VM ID of this OMA appliance in OSSEA';