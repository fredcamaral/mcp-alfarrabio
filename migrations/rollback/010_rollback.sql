-- Rollback for migration 010: create_templates_table
-- This rollback removes the templates table and related structures

-- Drop any indexes on templates table
DROP INDEX IF EXISTS idx_templates_type;
DROP INDEX IF EXISTS idx_templates_status;
DROP INDEX IF EXISTS idx_templates_repository;
DROP INDEX IF EXISTS idx_templates_created_at;

-- Drop any triggers on templates table
DROP TRIGGER IF EXISTS update_templates_updated_at ON templates;

-- Drop the templates table
DROP TABLE IF EXISTS templates;

-- Drop the template_status enum if it exists
DROP TYPE IF EXISTS template_status;