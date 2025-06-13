-- Migration 013: Create content_store table for template content persistence
-- This supports the PostgreSQL ContentStore implementation

-- Create content_store table for storing template and general content
CREATE TABLE IF NOT EXISTS content_store (
    id VARCHAR(255) NOT NULL,
    project_id VARCHAR(255) NOT NULL,
    session_id VARCHAR(255),
    type VARCHAR(100) NOT NULL, -- 'memory', 'task', 'decision', 'insight', 'template'
    title TEXT,
    content TEXT NOT NULL,
    summary TEXT,
    tags JSONB,
    metadata JSONB,
    embeddings JSONB, -- Store vector embeddings as JSON
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    accessed_at TIMESTAMP WITH TIME ZONE,
    quality DECIMAL(3,2) CHECK (quality >= 0 AND quality <= 1), -- 0.0-1.0
    confidence DECIMAL(3,2) CHECK (confidence >= 0 AND confidence <= 1), -- 0.0-1.0
    parent_id VARCHAR(255),
    thread_id VARCHAR(255),
    source VARCHAR(100), -- 'conversation', 'file', 'api'
    source_path TEXT,
    version INTEGER DEFAULT 1,
    
    -- Primary key combines id and project_id for multi-tenancy
    PRIMARY KEY (id, project_id)
);

-- Create indexes for efficient queries
CREATE INDEX IF NOT EXISTS idx_content_store_project_id ON content_store(project_id);
CREATE INDEX IF NOT EXISTS idx_content_store_session_id ON content_store(session_id) WHERE session_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_content_store_type ON content_store(type);
CREATE INDEX IF NOT EXISTS idx_content_store_created_at ON content_store(created_at);
CREATE INDEX IF NOT EXISTS idx_content_store_updated_at ON content_store(updated_at);
CREATE INDEX IF NOT EXISTS idx_content_store_parent_id ON content_store(parent_id) WHERE parent_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_content_store_thread_id ON content_store(thread_id) WHERE thread_id IS NOT NULL;

-- Create GIN indexes for JSONB fields for efficient querying
CREATE INDEX IF NOT EXISTS idx_content_store_tags_gin ON content_store USING GIN(tags);
CREATE INDEX IF NOT EXISTS idx_content_store_metadata_gin ON content_store USING GIN(metadata);

-- Create index for full-text search on content
CREATE INDEX IF NOT EXISTS idx_content_store_content_fts ON content_store USING GIN(to_tsvector('english', content));
CREATE INDEX IF NOT EXISTS idx_content_store_title_fts ON content_store USING GIN(to_tsvector('english', title)) WHERE title IS NOT NULL;

-- Create trigger to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_content_store_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER trigger_content_store_updated_at
    BEFORE UPDATE ON content_store
    FOR EACH ROW
    EXECUTE FUNCTION update_content_store_updated_at();

-- Add comment explaining the table purpose
COMMENT ON TABLE content_store IS 'Stores content items with project isolation for template system and general memory storage';
COMMENT ON COLUMN content_store.id IS 'Unique identifier for content within a project';
COMMENT ON COLUMN content_store.project_id IS 'Project identifier for multi-tenant isolation';
COMMENT ON COLUMN content_store.type IS 'Content type: memory, task, decision, insight, template';
COMMENT ON COLUMN content_store.embeddings IS 'Vector embeddings stored as JSON array';
COMMENT ON COLUMN content_store.quality IS 'Content quality score from 0.0 to 1.0';
COMMENT ON COLUMN content_store.confidence IS 'Confidence score from 0.0 to 1.0';