-- Add Foreign Key Constraints for Protection Flows
-- Date: 2025-10-09
-- Purpose: Add missing foreign key constraints with CASCADE DELETE

-- CRITICAL: fk_executions_flow already exists (CASCADE DELETE verified working)
-- This ensures execution records are automatically deleted when flows are deleted

-- NOTE: Schedule foreign keys cannot be added due to collation mismatch:
-- protection_flows: utf8mb4_unicode_ci
-- replication_schedules: utf8mb4_general_ci
-- These will be added in a future migration after collation alignment
