-- Migration: Drop Protection Flows Tables
-- Date: 2025-10-09
-- Purpose: Reverse migration for protection flows tables

-- Drop executions table first (due to foreign key constraint)
DROP TABLE IF EXISTS protection_flow_executions;

-- Drop main flows table
DROP TABLE IF EXISTS protection_flows;

