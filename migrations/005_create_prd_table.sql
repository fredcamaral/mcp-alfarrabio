-- Migration 005: Create PRD Table
-- Description: Creates PRD (Project Requirements Document) table for document storage and metadata
-- Created: 2025-06-09
-- Version: 1.0.0

-- Enable UUID extension if not already enabled
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Create PRD table for storing Project Requirements Documents
CREATE TABLE prds (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    repository VARCHAR(255) NOT NULL,
    filename VARCHAR(255) NOT NULL,
    content TEXT NOT NULL,
    parsed_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    task_count INTEGER DEFAULT 0 NOT NULL,
    complexity_score NUMERIC(5,3) CHECK (complexity_score >= 0 AND complexity_score <= 1),
    metadata JSONB DEFAULT '{}' NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    deleted_at TIMESTAMP WITH TIME ZONE,
    
    -- Additional fields for enhanced functionality
    version INTEGER DEFAULT 1 NOT NULL,
    file_size_bytes BIGINT,
    file_hash VARCHAR(64), -- SHA-256 hash for content integrity
    content_type VARCHAR(100) DEFAULT 'text/markdown',
    author VARCHAR(255),
    last_parsed_version INTEGER DEFAULT 1,
    
    -- Parse quality and validation
    parse_status VARCHAR(20) DEFAULT 'pending' CHECK (parse_status IN ('pending', 'success', 'partial', 'failed')),
    parse_errors JSONB DEFAULT '[]',
    validation_score NUMERIC(5,3) CHECK (validation_score >= 0 AND validation_score <= 1),
    
    -- Document classification
    document_type VARCHAR(50) DEFAULT 'prd' CHECK (document_type IN ('prd', 'spec', 'requirements', 'design', 'other')),
    priority_level VARCHAR(20) DEFAULT 'medium' CHECK (priority_level IN ('low', 'medium', 'high', 'critical')),
    
    -- Status tracking
    status VARCHAR(20) DEFAULT 'active' CHECK (status IN ('active', 'archived', 'deprecated', 'draft')),
    
    -- Search optimization
    search_vector TSVECTOR GENERATED ALWAYS AS (
        setweight(to_tsvector('english', COALESCE(filename, '')), 'A') ||
        setweight(to_tsvector('english', COALESCE(content, '')), 'B')
    ) STORED
);

-- Create constraints
ALTER TABLE prds ADD CONSTRAINT chk_prds_task_count_positive 
    CHECK (task_count >= 0);

ALTER TABLE prds ADD CONSTRAINT chk_prds_file_size_positive 
    CHECK (file_size_bytes IS NULL OR file_size_bytes >= 0);

ALTER TABLE prds ADD CONSTRAINT chk_prds_version_positive 
    CHECK (version > 0);

-- Create unique constraint to prevent duplicate active PRDs
CREATE UNIQUE INDEX idx_prds_unique_active_repo_filename 
    ON prds(repository, filename) 
    WHERE deleted_at IS NULL;

-- Create performance indexes
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_prds_repository ON prds(repository) WHERE deleted_at IS NULL;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_prds_filename ON prds(filename) WHERE deleted_at IS NULL;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_prds_created_at ON prds(created_at DESC) WHERE deleted_at IS NULL;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_prds_updated_at ON prds(updated_at DESC) WHERE deleted_at IS NULL;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_prds_parsed_at ON prds(parsed_at DESC) WHERE deleted_at IS NULL;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_prds_task_count ON prds(task_count DESC) WHERE deleted_at IS NULL;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_prds_complexity_score ON prds(complexity_score DESC) WHERE deleted_at IS NULL;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_prds_status ON prds(status) WHERE deleted_at IS NULL;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_prds_document_type ON prds(document_type) WHERE deleted_at IS NULL;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_prds_priority_level ON prds(priority_level) WHERE deleted_at IS NULL;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_prds_parse_status ON prds(parse_status) WHERE deleted_at IS NULL;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_prds_search_vector ON prds USING GIN(search_vector);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_prds_metadata ON prds USING GIN(metadata) WHERE deleted_at IS NULL;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_prds_file_hash ON prds(file_hash) WHERE deleted_at IS NULL;

-- Composite indexes for common query patterns
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_prds_repo_status ON prds(repository, status) WHERE deleted_at IS NULL;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_prds_repo_type ON prds(repository, document_type) WHERE deleted_at IS NULL;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_prds_type_priority ON prds(document_type, priority_level) WHERE deleted_at IS NULL;

-- Create triggers

-- Update updated_at timestamp trigger
CREATE OR REPLACE FUNCTION update_prds_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_prds_updated_at_trigger
    BEFORE UPDATE ON prds
    FOR EACH ROW EXECUTE FUNCTION update_prds_updated_at();

-- Version increment trigger
CREATE OR REPLACE FUNCTION increment_prd_version()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'UPDATE' AND OLD.content IS DISTINCT FROM NEW.content THEN
        NEW.version = OLD.version + 1;
        NEW.parsed_at = CURRENT_TIMESTAMP;
        NEW.parse_status = 'pending';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER prd_version_increment_trigger
    BEFORE UPDATE ON prds
    FOR EACH ROW EXECUTE FUNCTION increment_prd_version();

-- Content hash calculation trigger
CREATE OR REPLACE FUNCTION calculate_prd_content_hash()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' OR (TG_OP = 'UPDATE' AND OLD.content IS DISTINCT FROM NEW.content) THEN
        NEW.file_hash = encode(digest(NEW.content, 'sha256'), 'hex');
        NEW.file_size_bytes = length(NEW.content);
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER prd_content_hash_trigger
    BEFORE INSERT OR UPDATE ON prds
    FOR EACH ROW EXECUTE FUNCTION calculate_prd_content_hash();

-- Soft delete cascade trigger for related tasks
CREATE OR REPLACE FUNCTION handle_prd_soft_delete()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'UPDATE' AND OLD.deleted_at IS NULL AND NEW.deleted_at IS NOT NULL THEN
        -- Mark related tasks as having an archived PRD
        UPDATE tasks 
        SET metadata = COALESCE(metadata, '{}') || '{"source_prd_archived": true}'
        WHERE (source_prd_id = NEW.id OR prd_id = NEW.id) 
        AND deleted_at IS NULL;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER prd_soft_delete_handler
    AFTER UPDATE ON prds
    FOR EACH ROW EXECUTE FUNCTION handle_prd_soft_delete();

-- Create helper functions

-- Function to get PRD statistics
CREATE OR REPLACE FUNCTION get_prd_statistics()
RETURNS TABLE (
    total_prds BIGINT,
    active_prds BIGINT,
    avg_complexity_score NUMERIC,
    avg_task_count NUMERIC,
    total_file_size_mb NUMERIC,
    repositories_count BIGINT,
    parse_success_rate NUMERIC
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        COUNT(*) as total_prds,
        COUNT(*) FILTER (WHERE deleted_at IS NULL AND status = 'active') as active_prds,
        AVG(complexity_score) FILTER (WHERE deleted_at IS NULL) as avg_complexity_score,
        AVG(task_count) FILTER (WHERE deleted_at IS NULL) as avg_task_count,
        ROUND((SUM(file_size_bytes) FILTER (WHERE deleted_at IS NULL) / 1024.0 / 1024.0)::NUMERIC, 2) as total_file_size_mb,
        COUNT(DISTINCT repository) FILTER (WHERE deleted_at IS NULL) as repositories_count,
        (COUNT(*) FILTER (WHERE parse_status = 'success' AND deleted_at IS NULL)::DECIMAL / 
         NULLIF(COUNT(*) FILTER (WHERE deleted_at IS NULL), 0)) as parse_success_rate
    FROM prds;
END;
$$ LANGUAGE plpgsql;

-- Function to search PRDs by content
CREATE OR REPLACE FUNCTION search_prds(
    search_query TEXT,
    repository_filter VARCHAR(255) DEFAULT NULL,
    document_type_filter VARCHAR(50) DEFAULT NULL,
    limit_count INTEGER DEFAULT 20
)
RETURNS TABLE (
    id UUID,
    repository VARCHAR(255),
    filename VARCHAR(255),
    document_type VARCHAR(50),
    priority_level VARCHAR(20),
    task_count INTEGER,
    complexity_score NUMERIC,
    created_at TIMESTAMP WITH TIME ZONE,
    rank REAL
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        p.id,
        p.repository,
        p.filename,
        p.document_type,
        p.priority_level,
        p.task_count,
        p.complexity_score,
        p.created_at,
        ts_rank(p.search_vector, plainto_tsquery('english', search_query)) as rank
    FROM prds p
    WHERE p.deleted_at IS NULL
    AND p.search_vector @@ plainto_tsquery('english', search_query)
    AND (repository_filter IS NULL OR p.repository = repository_filter)
    AND (document_type_filter IS NULL OR p.document_type = document_type_filter)
    ORDER BY rank DESC, p.created_at DESC
    LIMIT limit_count;
END;
$$ LANGUAGE plpgsql;

-- Function to get PRD parsing recommendations
CREATE OR REPLACE FUNCTION get_prd_parsing_recommendations()
RETURNS TABLE (
    recommendation_type TEXT,
    description TEXT,
    prd_count BIGINT,
    urgency TEXT
) AS $$
DECLARE
    failed_parse_count BIGINT;
    pending_parse_count BIGINT;
    outdated_parse_count BIGINT;
BEGIN
    -- Count different parsing issues
    SELECT COUNT(*) INTO failed_parse_count 
    FROM prds 
    WHERE parse_status = 'failed' AND deleted_at IS NULL;
    
    SELECT COUNT(*) INTO pending_parse_count 
    FROM prds 
    WHERE parse_status = 'pending' AND deleted_at IS NULL;
    
    SELECT COUNT(*) INTO outdated_parse_count 
    FROM prds 
    WHERE last_parsed_version < version AND deleted_at IS NULL;
    
    -- Generate recommendations
    IF failed_parse_count > 0 THEN
        RETURN QUERY SELECT 
            'Parse Failures'::TEXT,
            'PRDs with failed parsing need investigation'::TEXT,
            failed_parse_count,
            CASE WHEN failed_parse_count > 5 THEN 'High' ELSE 'Medium' END::TEXT;
    END IF;
    
    IF pending_parse_count > 0 THEN
        RETURN QUERY SELECT 
            'Pending Parsing'::TEXT,
            'PRDs waiting to be parsed'::TEXT,
            pending_parse_count,
            CASE WHEN pending_parse_count > 10 THEN 'High' ELSE 'Low' END::TEXT;
    END IF;
    
    IF outdated_parse_count > 0 THEN
        RETURN QUERY SELECT 
            'Outdated Parsing'::TEXT,
            'PRDs with content newer than last parse'::TEXT,
            outdated_parse_count,
            'Medium'::TEXT;
    END IF;
END;
$$ LANGUAGE plpgsql;

-- Function to archive old PRDs
CREATE OR REPLACE FUNCTION archive_old_prds(months_old INTEGER DEFAULT 12)
RETURNS INTEGER AS $$
DECLARE
    archived_count INTEGER;
    cutoff_date TIMESTAMP WITH TIME ZONE;
BEGIN
    cutoff_date := CURRENT_TIMESTAMP - (months_old || ' months')::INTERVAL;
    
    UPDATE prds 
    SET status = 'archived',
        updated_at = CURRENT_TIMESTAMP
    WHERE status = 'active' 
    AND updated_at < cutoff_date
    AND task_count = 0  -- Only archive PRDs with no associated tasks
    AND deleted_at IS NULL;
    
    GET DIAGNOSTICS archived_count = ROW_COUNT;
    
    RETURN archived_count;
END;
$$ LANGUAGE plpgsql;

-- Comments for documentation
COMMENT ON TABLE prds IS 'Stores Project Requirements Documents with parsing metadata and content analysis';
COMMENT ON COLUMN prds.search_vector IS 'Generated tsvector for full-text search of filename and content';
COMMENT ON COLUMN prds.file_hash IS 'SHA-256 hash of content for integrity verification';
COMMENT ON COLUMN prds.complexity_score IS 'AI-calculated complexity score from 0.0 to 1.0';
COMMENT ON COLUMN prds.validation_score IS 'Quality score of parsed content from 0.0 to 1.0';
COMMENT ON FUNCTION get_prd_statistics() IS 'Returns comprehensive statistics about PRD collection';
COMMENT ON FUNCTION search_prds(TEXT, VARCHAR, VARCHAR, INTEGER) IS 'Full-text search across PRD content with filtering';
COMMENT ON FUNCTION get_prd_parsing_recommendations() IS 'Provides recommendations for PRD parsing maintenance';
COMMENT ON FUNCTION archive_old_prds(INTEGER) IS 'Archives old PRDs that have no associated tasks';