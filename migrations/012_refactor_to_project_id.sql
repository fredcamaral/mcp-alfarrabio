-- Migration 012: Refactor repository column to project_id
-- This migration supports the Phase 1 refactor by replacing the confusing 
-- "repository" parameter with the clear "project_id" parameter across all tables.

-- Update conversation_chunks table
ALTER TABLE conversation_chunks 
ADD COLUMN project_id VARCHAR(100);

-- Copy data from repository to project_id
UPDATE conversation_chunks 
SET project_id = repository 
WHERE repository IS NOT NULL;

-- Drop old repository column after data migration
ALTER TABLE conversation_chunks 
DROP COLUMN repository;

-- Add NOT NULL constraint and index for project_id
ALTER TABLE conversation_chunks 
ALTER COLUMN project_id SET NOT NULL;

CREATE INDEX idx_conversation_chunks_project_id 
ON conversation_chunks(project_id);

-- Update project_contexts table (if it exists)
DO $$ 
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'project_contexts') THEN
        -- Add project_id column
        ALTER TABLE project_contexts 
        ADD COLUMN project_id VARCHAR(100);
        
        -- Copy data from repository to project_id  
        UPDATE project_contexts 
        SET project_id = repository 
        WHERE repository IS NOT NULL;
        
        -- Drop old repository column
        ALTER TABLE project_contexts 
        DROP COLUMN repository;
        
        -- Add constraints and index
        ALTER TABLE project_contexts 
        ALTER COLUMN project_id SET NOT NULL;
        
        CREATE INDEX idx_project_contexts_project_id 
        ON project_contexts(project_id);
    END IF;
END $$;

-- Update tasks table (if it exists)
DO $$ 
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'tasks') THEN
        -- Add project_id column
        ALTER TABLE tasks 
        ADD COLUMN project_id VARCHAR(100);
        
        -- Copy data from repository to project_id
        UPDATE tasks 
        SET project_id = repository 
        WHERE repository IS NOT NULL;
        
        -- Drop old repository column
        ALTER TABLE tasks 
        DROP COLUMN repository;
        
        -- Add constraints and index
        ALTER TABLE tasks 
        ALTER COLUMN project_id SET NOT NULL;
        
        CREATE INDEX idx_tasks_project_id 
        ON tasks(project_id);
    END IF;
END $$;

-- Update sessions table (if it exists) 
DO $$ 
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'sessions') THEN
        -- Add project_id column
        ALTER TABLE sessions 
        ADD COLUMN project_id VARCHAR(100);
        
        -- Copy data from repository to project_id
        UPDATE sessions 
        SET project_id = repository 
        WHERE repository IS NOT NULL;
        
        -- Drop old repository column  
        ALTER TABLE sessions 
        DROP COLUMN repository;
        
        -- Add constraints and index
        ALTER TABLE sessions 
        ALTER COLUMN project_id SET NOT NULL;
        
        CREATE INDEX idx_sessions_project_id 
        ON sessions(project_id);
    END IF;
END $$;

-- Update patterns table (if it exists)
DO $$ 
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'patterns') THEN
        -- Add project_id column
        ALTER TABLE patterns 
        ADD COLUMN project_id VARCHAR(100);
        
        -- Copy data from repository to project_id
        UPDATE patterns 
        SET project_id = repository 
        WHERE repository IS NOT NULL;
        
        -- Drop old repository column
        ALTER TABLE patterns 
        DROP COLUMN repository;
        
        -- Add constraints and index
        ALTER TABLE patterns 
        ALTER COLUMN project_id SET NOT NULL;
        
        CREATE INDEX idx_patterns_project_id 
        ON patterns(project_id);
    END IF;
END $$;

-- Update any other tables that might have repository columns
-- This is a safety net for any additional tables created during development

-- Update insights table (if it exists)
DO $$ 
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'insights') THEN
        IF EXISTS (SELECT 1 FROM information_schema.columns 
                  WHERE table_name = 'insights' AND column_name = 'repository') THEN
            ALTER TABLE insights ADD COLUMN project_id VARCHAR(100);
            UPDATE insights SET project_id = repository WHERE repository IS NOT NULL;
            ALTER TABLE insights DROP COLUMN repository;
            ALTER TABLE insights ALTER COLUMN project_id SET NOT NULL;
            CREATE INDEX idx_insights_project_id ON insights(project_id);
        END IF;
    END IF;
END $$;

-- Update relationships table (if it exists)
DO $$ 
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'relationships') THEN
        IF EXISTS (SELECT 1 FROM information_schema.columns 
                  WHERE table_name = 'relationships' AND column_name = 'repository') THEN
            ALTER TABLE relationships ADD COLUMN project_id VARCHAR(100);
            UPDATE relationships SET project_id = repository WHERE repository IS NOT NULL;
            ALTER TABLE relationships DROP COLUMN repository;
            ALTER TABLE relationships ALTER COLUMN project_id SET NOT NULL;
            CREATE INDEX idx_relationships_project_id ON relationships(project_id);
        END IF;
    END IF;
END $$;

-- Create a function to validate project_id format
-- This enforces the validation rules defined in internal/types/core.go
CREATE OR REPLACE FUNCTION validate_project_id(project_id TEXT) 
RETURNS BOOLEAN AS $$
BEGIN
    -- Check length (1-100 characters)
    IF LENGTH(project_id) < 1 OR LENGTH(project_id) > 100 THEN
        RETURN FALSE;
    END IF;
    
    -- Check format (alphanumeric, hyphens, underscores, dots, colons, slashes)
    IF project_id !~ '^[a-zA-Z0-9\-_./:]+$' THEN
        RETURN FALSE;
    END IF;
    
    RETURN TRUE;
END;
$$ LANGUAGE plpgsql;

-- Add check constraint to ensure valid project_id format on main table
ALTER TABLE conversation_chunks 
ADD CONSTRAINT chk_project_id_valid 
CHECK (validate_project_id(project_id));

-- Update any existing foreign key constraints to reference project_id
-- This would need to be customized based on actual foreign key relationships

-- Add migration completion log
INSERT INTO schema_migrations (version, applied_at) 
VALUES (12, NOW())
ON CONFLICT (version) DO NOTHING;

-- Add comment for documentation
COMMENT ON COLUMN conversation_chunks.project_id IS 
'Project identifier for data isolation. Replaces the old repository column with cleaner semantics. Format: 1-100 characters, alphanumeric plus -_./:';

-- Migration rollback script (for reference)
-- To rollback this migration, run:
-- 
-- ALTER TABLE conversation_chunks ADD COLUMN repository VARCHAR(255);
-- UPDATE conversation_chunks SET repository = project_id;
-- ALTER TABLE conversation_chunks DROP COLUMN project_id;
-- DELETE FROM schema_migrations WHERE version = 12;