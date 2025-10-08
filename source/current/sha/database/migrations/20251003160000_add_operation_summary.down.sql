-- Migration: add_operation_summary (DOWN)
-- Removes operation summary field from vm_replication_contexts

DROP INDEX IF EXISTS idx_vm_contexts_last_op_time ON vm_replication_contexts;

ALTER TABLE vm_replication_contexts
DROP COLUMN IF EXISTS last_operation_summary;


