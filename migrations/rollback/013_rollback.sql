-- Rollback for Migration 013: Drop content_store table

-- Drop trigger function
DROP TRIGGER IF EXISTS trigger_content_store_updated_at ON content_store;
DROP FUNCTION IF EXISTS update_content_store_updated_at();

-- Drop indexes
DROP INDEX IF EXISTS idx_content_store_title_fts;
DROP INDEX IF EXISTS idx_content_store_content_fts;
DROP INDEX IF EXISTS idx_content_store_metadata_gin;
DROP INDEX IF EXISTS idx_content_store_tags_gin;
DROP INDEX IF EXISTS idx_content_store_thread_id;
DROP INDEX IF EXISTS idx_content_store_parent_id;
DROP INDEX IF EXISTS idx_content_store_updated_at;
DROP INDEX IF EXISTS idx_content_store_created_at;
DROP INDEX IF EXISTS idx_content_store_type;
DROP INDEX IF EXISTS idx_content_store_session_id;
DROP INDEX IF EXISTS idx_content_store_project_id;

-- Drop table
DROP TABLE IF EXISTS content_store;