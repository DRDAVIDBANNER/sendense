-- Rollback Foreign Key Constraints for Protection Flows
-- Date: 2025-10-09
-- Purpose: Remove foreign key constraints added in 20251009120001

-- NOTE: Only fk_executions_flow was actually added in this migration
-- Schedule FKs were not added due to collation mismatch

ALTER TABLE protection_flow_executions
    DROP FOREIGN KEY IF EXISTS fk_executions_flow;
