-- Rollback for migration 011: create_patterns_table
-- This rollback removes the patterns table and related structures

-- Drop any indexes on patterns table
DROP INDEX IF EXISTS idx_patterns_type;
DROP INDEX IF EXISTS idx_patterns_confidence;
DROP INDEX IF EXISTS idx_patterns_frequency;
DROP INDEX IF EXISTS idx_patterns_repository;
DROP INDEX IF EXISTS idx_patterns_created_at;

-- Drop any triggers on patterns table
DROP TRIGGER IF EXISTS update_patterns_updated_at ON patterns;

-- Drop the patterns table
DROP TABLE IF EXISTS patterns;

-- Drop the pattern_type enum if it exists
DROP TYPE IF EXISTS pattern_type;