-- Rollback Migration 001: Drop Enhanced Tasks Table and Dependencies
-- Description: Safely removes the enhanced tasks table and all related structures
-- Created: 2025-06-12
-- Version: 1.0.0

-- Drop dependent objects first (foreign key constraints)
ALTER TABLE tasks DROP CONSTRAINT IF EXISTS fk_tasks_prd;
ALTER TABLE tasks DROP CONSTRAINT IF EXISTS fk_tasks_source_prd;
ALTER TABLE tasks DROP CONSTRAINT IF EXISTS fk_tasks_pattern;
ALTER TABLE tasks DROP CONSTRAINT IF EXISTS fk_tasks_template;

-- Drop dependent tables in correct order
DROP TABLE IF EXISTS task_quality_issues CASCADE;
DROP TABLE IF EXISTS task_effort_breakdown CASCADE;
DROP TABLE IF EXISTS task_templates CASCADE;
DROP TABLE IF EXISTS task_patterns CASCADE;

-- Drop main tasks table
DROP TABLE IF EXISTS tasks CASCADE;

-- Drop PRDs table if it was created by this migration
DROP TABLE IF EXISTS prds CASCADE;

-- Drop functions and triggers
DROP FUNCTION IF EXISTS update_task_search_vector() CASCADE;
DROP FUNCTION IF EXISTS update_updated_at_column() CASCADE;

-- Drop ENUM types
DROP TYPE IF EXISTS impact_level CASCADE;
DROP TYPE IF EXISTS risk_level CASCADE;
DROP TYPE IF EXISTS task_type CASCADE;
DROP TYPE IF EXISTS task_complexity CASCADE;
DROP TYPE IF EXISTS task_priority CASCADE;
DROP TYPE IF EXISTS task_status CASCADE;

-- Drop extensions if they were created by this migration
-- Note: Be careful with extensions as other schemas might use them
-- DROP EXTENSION IF EXISTS "pgcrypto";
-- DROP EXTENSION IF EXISTS "uuid-ossp";

-- Log rollback completion
-- INSERT INTO migration_records (version, name, checksum, applied_at, success, error_msg, duration_ms, backup_path, metadata)
-- VALUES ('001_rollback', 'Enhanced Tasks Table Rollback', 'rollback_001', NOW(), true, '', 0, '', '{"operation": "rollback", "target": "001"}');