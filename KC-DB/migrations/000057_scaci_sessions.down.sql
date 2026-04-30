-- Migration: 000057_scaci_sessions (DOWN)
-- Description: Remove SCACI session management tables
-- Author: KiloCenter Development Team
-- Date: 2025-01-30

-- Drop triggers first
DROP TRIGGER IF EXISTS trigger_scaci_op_log_updated_at ON scaci_operation_log;
DROP TRIGGER IF EXISTS trigger_scaci_sessions_updated_at ON scaci_sessions;

-- Drop indexes
DROP INDEX IF EXISTS idx_scaci_op_log_command;
DROP INDEX IF EXISTS idx_scaci_op_log_state;
DROP INDEX IF EXISTS idx_scaci_op_log_op_id;
DROP INDEX IF EXISTS idx_scaci_op_log_tenant;
DROP INDEX IF EXISTS idx_scaci_op_log_session;

DROP INDEX IF EXISTS idx_scaci_sessions_unique_active;
DROP INDEX IF EXISTS idx_scaci_sessions_status;
DROP INDEX IF EXISTS idx_scaci_sessions_sc_uuid;
DROP INDEX IF EXISTS idx_scaci_sessions_ac_uuid;
DROP INDEX IF EXISTS idx_scaci_sessions_ac_eui;
DROP INDEX IF EXISTS idx_scaci_sessions_tenant;

-- Drop tables (CASCADE will remove foreign key constraints)
DROP TABLE IF EXISTS scaci_operation_log CASCADE;
DROP TABLE IF EXISTS scaci_sessions CASCADE;

-- Note: We do NOT drop the update_updated_at_column() function as it may be
-- used by other tables (e.g., basestation_sessions, mioty_messages)