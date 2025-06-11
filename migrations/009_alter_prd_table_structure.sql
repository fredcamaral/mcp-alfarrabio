-- Migration 009: Alter PRD Table Structure
-- Description: Updates PRD table to match new requirements with proper fields for Product Requirements Documents
-- Created: 2025-01-10
-- Version: 1.0.0

-- First, drop existing constraints and indexes that might conflict
DROP TRIGGER IF EXISTS prd_version_increment_trigger ON prds;
DROP TRIGGER IF EXISTS prd_content_hash_trigger ON prds;
DROP TRIGGER IF EXISTS prd_soft_delete_handler ON prds;
DROP FUNCTION IF EXISTS increment_prd_version() CASCADE;
DROP FUNCTION IF EXISTS calculate_prd_content_hash() CASCADE;
DROP FUNCTION IF EXISTS handle_prd_soft_delete() CASCADE;

-- Add new columns if they don't exist
ALTER TABLE prds 
ADD COLUMN IF NOT EXISTS title TEXT,
ADD COLUMN IF NOT EXISTS summary TEXT,
ADD COLUMN IF NOT EXISTS goals JSONB DEFAULT '[]'::jsonb,
ADD COLUMN IF NOT EXISTS requirements JSONB DEFAULT '[]'::jsonb,
ADD COLUMN IF NOT EXISTS success_criteria JSONB DEFAULT '[]'::jsonb,
ADD COLUMN IF NOT EXISTS created_by TEXT;

-- Update existing columns
ALTER TABLE prds
ALTER COLUMN content SET NOT NULL;

-- Add status column with proper constraint if it doesn't match
DO $$
BEGIN
    -- Check if status column exists with correct constraint
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'prds' 
        AND column_name = 'status'
    ) THEN
        ALTER TABLE prds ADD COLUMN status TEXT DEFAULT 'draft';
    END IF;
    
    -- Drop existing status constraint if it exists
    ALTER TABLE prds DROP CONSTRAINT IF EXISTS prds_status_check;
    
    -- Add new status constraint
    ALTER TABLE prds ADD CONSTRAINT prds_status_check 
        CHECK (status IN ('draft', 'review', 'approved', 'deprecated'));
END $$;

-- Make title NOT NULL (handle existing rows first)
UPDATE prds SET title = filename WHERE title IS NULL;
ALTER TABLE prds ALTER COLUMN title SET NOT NULL;

-- Update status values to match new constraint
UPDATE prds 
SET status = CASE 
    WHEN status = 'active' THEN 'approved'
    WHEN status = 'archived' THEN 'deprecated'
    ELSE 'draft'
END
WHERE status NOT IN ('draft', 'review', 'approved', 'deprecated');

-- Create new indexes for the updated structure
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_prds_title ON prds(title) WHERE deleted_at IS NULL;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_prds_goals ON prds USING GIN(goals) WHERE deleted_at IS NULL;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_prds_requirements ON prds USING GIN(requirements) WHERE deleted_at IS NULL;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_prds_success_criteria ON prds USING GIN(success_criteria) WHERE deleted_at IS NULL;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_prds_created_by ON prds(created_by) WHERE deleted_at IS NULL;

-- Recreate the version increment trigger with simpler logic
CREATE OR REPLACE FUNCTION increment_prd_version()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'UPDATE' AND (
        OLD.content IS DISTINCT FROM NEW.content OR
        OLD.title IS DISTINCT FROM NEW.title OR
        OLD.goals IS DISTINCT FROM NEW.goals OR
        OLD.requirements IS DISTINCT FROM NEW.requirements OR
        OLD.success_criteria IS DISTINCT FROM NEW.success_criteria
    ) THEN
        NEW.version = OLD.version + 1;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER prd_version_increment_trigger
    BEFORE UPDATE ON prds
    FOR EACH ROW EXECUTE FUNCTION increment_prd_version();

-- Create new search vector that includes title
ALTER TABLE prds DROP COLUMN IF EXISTS search_vector;
ALTER TABLE prds ADD COLUMN search_vector TSVECTOR GENERATED ALWAYS AS (
    setweight(to_tsvector('english', COALESCE(title, '')), 'A') ||
    setweight(to_tsvector('english', COALESCE(summary, '')), 'B') ||
    setweight(to_tsvector('english', COALESCE(content, '')), 'C')
) STORED;

-- Recreate search index
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_prds_search_vector ON prds USING GIN(search_vector);

-- Update the search function to use title instead of filename
CREATE OR REPLACE FUNCTION search_prds(
    search_query TEXT,
    repository_filter VARCHAR(255) DEFAULT NULL,
    status_filter TEXT DEFAULT NULL,
    limit_count INTEGER DEFAULT 20
)
RETURNS TABLE (
    id UUID,
    repository VARCHAR(255),
    title TEXT,
    summary TEXT,
    status TEXT,
    created_by TEXT,
    version INTEGER,
    created_at TIMESTAMP WITH TIME ZONE,
    updated_at TIMESTAMP WITH TIME ZONE,
    rank REAL
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        p.id,
        p.repository,
        p.title,
        p.summary,
        p.status,
        p.created_by,
        p.version,
        p.created_at,
        p.updated_at,
        ts_rank(p.search_vector, plainto_tsquery('english', search_query)) as rank
    FROM prds p
    WHERE p.deleted_at IS NULL
    AND p.search_vector @@ plainto_tsquery('english', search_query)
    AND (repository_filter IS NULL OR p.repository = repository_filter)
    AND (status_filter IS NULL OR p.status = status_filter)
    ORDER BY rank DESC, p.updated_at DESC
    LIMIT limit_count;
END;
$$ LANGUAGE plpgsql;

-- Add helpful view for PRD summary
CREATE OR REPLACE VIEW prd_summary AS
SELECT 
    id,
    title,
    summary,
    status,
    created_by,
    version,
    jsonb_array_length(goals) as goals_count,
    jsonb_array_length(requirements) as requirements_count,
    jsonb_array_length(success_criteria) as success_criteria_count,
    created_at,
    updated_at
FROM prds
WHERE deleted_at IS NULL;

-- Comments for documentation
COMMENT ON COLUMN prds.title IS 'Title of the Product Requirements Document';
COMMENT ON COLUMN prds.summary IS 'Executive summary of the PRD';
COMMENT ON COLUMN prds.goals IS 'Array of project goals in JSONB format';
COMMENT ON COLUMN prds.requirements IS 'Array of functional and non-functional requirements in JSONB format';
COMMENT ON COLUMN prds.success_criteria IS 'Array of measurable success criteria in JSONB format';
COMMENT ON COLUMN prds.created_by IS 'User or system that created this PRD';
COMMENT ON COLUMN prds.status IS 'Current status of the PRD: draft, review, approved, or deprecated';
COMMENT ON VIEW prd_summary IS 'Simplified view of PRDs with counts of goals, requirements, and success criteria';