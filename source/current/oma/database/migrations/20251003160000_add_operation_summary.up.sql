-- Migration: add_operation_summary
-- Created: 20251003160000
-- Purpose: Add operation summary field to vm_replication_contexts for persistent failover/rollback visibility
-- Minimal change: Single JSON column for user-friendly operation tracking

ALTER TABLE vm_replication_contexts
ADD COLUMN last_operation_summary JSON NULL COMMENT 'Summary of most recent operation (replication/failover/rollback) for GUI visibility';

