-- Rollback Migration 007: Drop Audit Tables and Functions
-- Description: Safely removes audit logging infrastructure
-- Created: 2025-06-12
-- Version: 1.0.0

-- Drop trigger first
DROP TRIGGER IF EXISTS task_audit_trigger ON tasks;

-- Drop functions
DROP FUNCTION IF EXISTS log_task_audit() CASCADE;
DROP FUNCTION IF EXISTS get_audit_statistics(TIMESTAMP WITH TIME ZONE, TIMESTAMP WITH TIME ZONE) CASCADE;
DROP FUNCTION IF EXISTS cleanup_old_audit_entries(INTEGER) CASCADE;
DROP FUNCTION IF EXISTS create_audit_partition(DATE, DATE) CASCADE;

-- Drop view
DROP VIEW IF EXISTS audit_summary CASCADE;

-- Drop main audit table
DROP TABLE IF EXISTS task_audit_log CASCADE;

-- Drop audit action enum
DROP TYPE IF EXISTS audit_action CASCADE;

-- Reset audit configuration (if still needed)
-- SELECT set_config('app.audit_enabled', 'false', false);
-- SELECT set_config('app.current_user_id', 'system', false);

-- Log rollback completion
-- Note: This would normally go in migration_records but that table may not exist yet
-- INSERT INTO migration_records (version, name, checksum, applied_at, success, error_msg, duration_ms, backup_path, metadata)
-- VALUES ('007_rollback', 'Audit Tables Rollback', 'rollback_007', NOW(), true, '', 0, '', '{"operation": "rollback", "target": "007"}');