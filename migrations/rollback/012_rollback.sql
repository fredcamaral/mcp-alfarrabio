-- Rollback for migration 012: refactor_to_project_id
-- This rollback reverses the changes made in 012_refactor_to_project_id.sql

-- Restore original column names in chunks table
ALTER TABLE chunks 
RENAME COLUMN project_id TO repository;

-- Restore original column names in chunk_relationships table  
ALTER TABLE chunk_relationships
RENAME COLUMN project_id TO repository;

-- Restore original column names in sessions table
ALTER TABLE sessions
RENAME COLUMN project_id TO repository;

-- Restore original column names in tasks table (if exists)
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns 
               WHERE table_name = 'tasks' AND column_name = 'project_id') THEN
        ALTER TABLE tasks RENAME COLUMN project_id TO repository;
    END IF;
END $$;

-- Restore original column names in patterns table (if exists)
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns 
               WHERE table_name = 'patterns' AND column_name = 'project_id') THEN
        ALTER TABLE patterns RENAME COLUMN project_id TO repository;
    END IF;
END $$;

-- Restore original column names in templates table (if exists)
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns 
               WHERE table_name = 'templates' AND column_name = 'project_id') THEN
        ALTER TABLE templates RENAME COLUMN project_id TO repository;
    END IF;
END $$;

-- Drop new indexes on project_id columns
DROP INDEX IF EXISTS idx_chunks_project_id;
DROP INDEX IF EXISTS idx_chunk_relationships_project_id;
DROP INDEX IF EXISTS idx_sessions_project_id;
DROP INDEX IF EXISTS idx_tasks_project_id;
DROP INDEX IF EXISTS idx_patterns_project_id;
DROP INDEX IF EXISTS idx_templates_project_id;

-- Recreate original indexes on repository columns
CREATE INDEX IF NOT EXISTS idx_chunks_repository ON chunks(repository);
CREATE INDEX IF NOT EXISTS idx_chunk_relationships_repository ON chunk_relationships(repository);
CREATE INDEX IF NOT EXISTS idx_sessions_repository ON sessions(repository);

-- Recreate other original indexes if tables exist
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'tasks') THEN
        CREATE INDEX IF NOT EXISTS idx_tasks_repository ON tasks(repository);
    END IF;
    
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'patterns') THEN
        CREATE INDEX IF NOT EXISTS idx_patterns_repository ON patterns(repository);
    END IF;
    
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'templates') THEN
        CREATE INDEX IF NOT EXISTS idx_templates_repository ON templates(repository);
    END IF;
END $$;