-- Migration 010: Create Comprehensive Templates Table
-- Description: Creates a unified templates table for all template types (task, prd, trd, workflow) with version control and usage tracking
-- Created: 2025-10-06
-- Version: 1.0.0

-- Enable UUID extension if not already enabled
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Create template_types enum
CREATE TYPE template_type_enum AS ENUM ('task', 'prd', 'trd', 'workflow', 'document', 'code', 'config', 'test', 'other');

-- Create template_status enum
CREATE TYPE template_status_enum AS ENUM ('draft', 'active', 'deprecated', 'archived', 'experimental');

-- Create templates table
CREATE TABLE templates (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    
    -- Basic Information
    name VARCHAR(255) NOT NULL,
    description TEXT,
    template_type template_type_enum NOT NULL,
    
    -- Template Content
    content TEXT NOT NULL, -- The actual template content
    content_format VARCHAR(50) DEFAULT 'markdown' CHECK (content_format IN ('markdown', 'json', 'yaml', 'xml', 'text', 'html')),
    
    -- Variables and Parameters
    variables JSONB DEFAULT '[]', -- Array of variable definitions [{name, type, default, required, description}]
    parameters JSONB DEFAULT '{}', -- Default parameter values
    placeholders JSONB DEFAULT '[]', -- Template placeholders with descriptions
    
    -- Categorization and Organization
    category VARCHAR(100),
    subcategory VARCHAR(100),
    tags JSONB DEFAULT '[]',
    keywords JSONB DEFAULT '[]', -- For better search
    
    -- Version Control
    version INTEGER DEFAULT 1 NOT NULL,
    parent_template_id UUID REFERENCES templates(id),
    version_notes TEXT,
    is_latest BOOLEAN DEFAULT true,
    published_at TIMESTAMP WITH TIME ZONE,
    
    -- Usage Tracking
    usage_count INTEGER DEFAULT 0 NOT NULL,
    last_used_at TIMESTAMP WITH TIME ZONE,
    last_used_by VARCHAR(255),
    
    -- Performance Metrics
    avg_time_saved_minutes NUMERIC(10,2),
    success_rate NUMERIC(5,3) CHECK (success_rate >= 0 AND success_rate <= 1),
    error_rate NUMERIC(5,3) CHECK (error_rate >= 0 AND error_rate <= 1),
    completion_rate NUMERIC(5,3) CHECK (completion_rate >= 0 AND completion_rate <= 1),
    
    -- User Feedback
    rating_score NUMERIC(3,2) CHECK (rating_score >= 0 AND rating_score <= 5),
    rating_count INTEGER DEFAULT 0,
    feedback_positive INTEGER DEFAULT 0,
    feedback_negative INTEGER DEFAULT 0,
    feedback_comments JSONB DEFAULT '[]', -- Array of user feedback
    
    -- Access Control
    visibility VARCHAR(20) DEFAULT 'public' CHECK (visibility IN ('public', 'private', 'team', 'organization')),
    owner_id VARCHAR(255),
    team_id VARCHAR(255),
    organization_id VARCHAR(255),
    shared_with JSONB DEFAULT '[]', -- Array of user/team IDs
    
    -- Template Configuration
    config JSONB DEFAULT '{}', -- Template-specific configuration
    validation_rules JSONB DEFAULT '[]', -- Rules for validating template usage
    dependencies JSONB DEFAULT '[]', -- Other templates this depends on
    
    -- Metadata
    metadata JSONB DEFAULT '{}',
    custom_fields JSONB DEFAULT '{}',
    
    -- Status and Lifecycle
    status template_status_enum DEFAULT 'draft',
    approved_by VARCHAR(255),
    approved_at TIMESTAMP WITH TIME ZONE,
    review_notes TEXT,
    
    -- Audit Fields
    created_by VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_by VARCHAR(255),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    deleted_by VARCHAR(255),
    deleted_at TIMESTAMP WITH TIME ZONE,
    
    -- Search Optimization
    search_vector TSVECTOR GENERATED ALWAYS AS (
        setweight(to_tsvector('english', COALESCE(name, '')), 'A') ||
        setweight(to_tsvector('english', COALESCE(description, '')), 'B') ||
        setweight(to_tsvector('english', COALESCE(category || ' ' || subcategory, '')), 'C') ||
        setweight(to_tsvector('english', COALESCE(content, '')), 'D')
    ) STORED
);

-- Create template usage history table
CREATE TABLE template_usage_history (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    template_id UUID NOT NULL REFERENCES templates(id),
    template_version INTEGER NOT NULL,
    
    -- Usage Information
    used_by VARCHAR(255) NOT NULL,
    used_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    usage_context JSONB DEFAULT '{}', -- Context of where/how it was used
    
    -- Parameters Used
    parameters_used JSONB DEFAULT '{}',
    variables_filled JSONB DEFAULT '{}',
    
    -- Results
    result_status VARCHAR(20) CHECK (result_status IN ('success', 'partial', 'failed', 'abandoned')),
    time_saved_minutes NUMERIC(10,2),
    error_details TEXT,
    
    -- Feedback
    user_rating INTEGER CHECK (user_rating >= 1 AND user_rating <= 5),
    user_feedback TEXT,
    
    -- Metadata
    session_id VARCHAR(255),
    project_id VARCHAR(255),
    repository VARCHAR(255),
    metadata JSONB DEFAULT '{}'
);

-- Create template categories table
CREATE TABLE template_categories (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(100) UNIQUE NOT NULL,
    parent_category_id UUID REFERENCES template_categories(id),
    description TEXT,
    icon VARCHAR(50),
    color VARCHAR(7), -- Hex color code
    display_order INTEGER DEFAULT 0,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL
);

-- Create template relationships table
CREATE TABLE template_relationships (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    source_template_id UUID NOT NULL REFERENCES templates(id),
    target_template_id UUID NOT NULL REFERENCES templates(id),
    relationship_type VARCHAR(50) NOT NULL CHECK (relationship_type IN ('extends', 'includes', 'depends_on', 'related_to', 'alternative_to', 'replaces')),
    relationship_data JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    
    CONSTRAINT unique_template_relationship UNIQUE (source_template_id, target_template_id, relationship_type)
);

-- Create constraints
ALTER TABLE templates ADD CONSTRAINT chk_templates_usage_count_positive 
    CHECK (usage_count >= 0);

ALTER TABLE templates ADD CONSTRAINT chk_templates_version_positive 
    CHECK (version > 0);

ALTER TABLE templates ADD CONSTRAINT chk_templates_no_self_reference 
    CHECK (id != parent_template_id);

-- Create unique constraint for template names within same type and organization
CREATE UNIQUE INDEX idx_templates_unique_name 
    ON templates(name, template_type, COALESCE(organization_id, 'global')) 
    WHERE deleted_at IS NULL AND is_latest = true;

-- Create performance indexes
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_templates_type ON templates(template_type) WHERE deleted_at IS NULL;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_templates_status ON templates(status) WHERE deleted_at IS NULL;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_templates_category ON templates(category, subcategory) WHERE deleted_at IS NULL;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_templates_visibility ON templates(visibility) WHERE deleted_at IS NULL;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_templates_owner ON templates(owner_id) WHERE deleted_at IS NULL;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_templates_organization ON templates(organization_id) WHERE deleted_at IS NULL;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_templates_usage ON templates(usage_count DESC) WHERE deleted_at IS NULL;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_templates_rating ON templates(rating_score DESC) WHERE deleted_at IS NULL;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_templates_created_at ON templates(created_at DESC) WHERE deleted_at IS NULL;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_templates_last_used ON templates(last_used_at DESC) WHERE deleted_at IS NULL;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_templates_parent ON templates(parent_template_id) WHERE deleted_at IS NULL;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_templates_latest ON templates(is_latest) WHERE deleted_at IS NULL AND is_latest = true;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_templates_search_vector ON templates USING GIN(search_vector);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_templates_tags ON templates USING GIN(tags) WHERE deleted_at IS NULL;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_templates_keywords ON templates USING GIN(keywords) WHERE deleted_at IS NULL;

-- Indexes for template usage history
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_usage_history_template ON template_usage_history(template_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_usage_history_user ON template_usage_history(used_by);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_usage_history_date ON template_usage_history(used_at DESC);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_usage_history_status ON template_usage_history(result_status);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_usage_history_project ON template_usage_history(project_id);

-- Indexes for template categories
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_categories_parent ON template_categories(parent_category_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_categories_active ON template_categories(is_active);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_categories_order ON template_categories(display_order);

-- Indexes for template relationships
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_relationships_source ON template_relationships(source_template_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_relationships_target ON template_relationships(target_template_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_relationships_type ON template_relationships(relationship_type);

-- Create triggers

-- Update updated_at timestamp
CREATE OR REPLACE FUNCTION update_templates_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_templates_updated_at_trigger
    BEFORE UPDATE ON templates
    FOR EACH ROW EXECUTE FUNCTION update_templates_updated_at();

-- Version management trigger
CREATE OR REPLACE FUNCTION manage_template_versions()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' AND NEW.parent_template_id IS NOT NULL THEN
        -- This is a new version, set is_latest to false for previous versions
        UPDATE templates 
        SET is_latest = false, updated_at = CURRENT_TIMESTAMP
        WHERE id = NEW.parent_template_id OR parent_template_id = NEW.parent_template_id
        AND id != NEW.id;
        
        -- Set the version number
        SELECT COALESCE(MAX(version), 0) + 1 INTO NEW.version
        FROM templates 
        WHERE id = NEW.parent_template_id OR parent_template_id = NEW.parent_template_id;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER manage_template_versions_trigger
    BEFORE INSERT ON templates
    FOR EACH ROW EXECUTE FUNCTION manage_template_versions();

-- Usage tracking trigger
CREATE OR REPLACE FUNCTION track_template_usage()
RETURNS TRIGGER AS $$
BEGIN
    -- Update template usage count and last used timestamp
    UPDATE templates 
    SET 
        usage_count = usage_count + 1,
        last_used_at = NEW.used_at,
        last_used_by = NEW.used_by
    WHERE id = NEW.template_id;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER track_template_usage_trigger
    AFTER INSERT ON template_usage_history
    FOR EACH ROW EXECUTE FUNCTION track_template_usage();

-- Create helper functions

-- Function to search templates
CREATE OR REPLACE FUNCTION search_templates(
    search_query TEXT,
    template_type_filter template_type_enum DEFAULT NULL,
    category_filter VARCHAR(100) DEFAULT NULL,
    visibility_filter VARCHAR(20) DEFAULT NULL,
    min_rating NUMERIC DEFAULT NULL,
    limit_count INTEGER DEFAULT 20,
    offset_count INTEGER DEFAULT 0
)
RETURNS TABLE (
    id UUID,
    name VARCHAR(255),
    description TEXT,
    template_type template_type_enum,
    category VARCHAR(100),
    rating_score NUMERIC,
    usage_count INTEGER,
    relevance_score REAL
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        t.id,
        t.name,
        t.description,
        t.template_type,
        t.category,
        t.rating_score,
        t.usage_count,
        ts_rank(t.search_vector, plainto_tsquery('english', search_query)) as relevance_score
    FROM templates t
    WHERE t.deleted_at IS NULL 
    AND t.is_latest = true
    AND t.status = 'active'
    AND (template_type_filter IS NULL OR t.template_type = template_type_filter)
    AND (category_filter IS NULL OR t.category = category_filter)
    AND (visibility_filter IS NULL OR t.visibility = visibility_filter)
    AND (min_rating IS NULL OR t.rating_score >= min_rating)
    AND (search_query IS NULL OR t.search_vector @@ plainto_tsquery('english', search_query))
    ORDER BY relevance_score DESC, t.usage_count DESC, t.rating_score DESC
    LIMIT limit_count
    OFFSET offset_count;
END;
$$ LANGUAGE plpgsql;

-- Function to get template with all versions
CREATE OR REPLACE FUNCTION get_template_versions(template_id_param UUID)
RETURNS TABLE (
    id UUID,
    version INTEGER,
    created_at TIMESTAMP WITH TIME ZONE,
    created_by VARCHAR(255),
    version_notes TEXT,
    is_latest BOOLEAN
) AS $$
BEGIN
    RETURN QUERY
    WITH RECURSIVE template_tree AS (
        -- Start with the given template
        SELECT id, parent_template_id FROM templates WHERE id = template_id_param
        UNION ALL
        -- Find all related versions
        SELECT t.id, t.parent_template_id 
        FROM templates t
        INNER JOIN template_tree tt ON (t.parent_template_id = tt.id OR t.id = tt.parent_template_id)
    )
    SELECT DISTINCT
        t.id,
        t.version,
        t.created_at,
        t.created_by,
        t.version_notes,
        t.is_latest
    FROM templates t
    WHERE t.id IN (SELECT id FROM template_tree)
    ORDER BY t.version DESC;
END;
$$ LANGUAGE plpgsql;

-- Function to calculate template statistics
CREATE OR REPLACE FUNCTION calculate_template_statistics(
    start_date TIMESTAMP WITH TIME ZONE DEFAULT NULL,
    end_date TIMESTAMP WITH TIME ZONE DEFAULT NULL
)
RETURNS TABLE (
    total_templates BIGINT,
    active_templates BIGINT,
    total_usage BIGINT,
    avg_rating NUMERIC,
    most_used_type template_type_enum,
    most_used_category VARCHAR(100)
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        COUNT(DISTINCT t.id) as total_templates,
        COUNT(DISTINCT t.id) FILTER (WHERE t.status = 'active') as active_templates,
        COALESCE(SUM(t.usage_count), 0) as total_usage,
        ROUND(AVG(t.rating_score) FILTER (WHERE t.rating_score IS NOT NULL), 2) as avg_rating,
        MODE() WITHIN GROUP (ORDER BY t.template_type) as most_used_type,
        MODE() WITHIN GROUP (ORDER BY t.category) as most_used_category
    FROM templates t
    WHERE t.deleted_at IS NULL
    AND (start_date IS NULL OR t.created_at >= start_date)
    AND (end_date IS NULL OR t.created_at <= end_date);
END;
$$ LANGUAGE plpgsql;

-- Function to recommend related templates
CREATE OR REPLACE FUNCTION recommend_related_templates(
    template_id_param UUID,
    limit_count INTEGER DEFAULT 5
)
RETURNS TABLE (
    id UUID,
    name VARCHAR(255),
    template_type template_type_enum,
    similarity_score NUMERIC
) AS $$
BEGIN
    RETURN QUERY
    WITH template_info AS (
        SELECT template_type, category, tags, keywords
        FROM templates
        WHERE id = template_id_param
    )
    SELECT 
        t.id,
        t.name,
        t.template_type,
        (
            -- Type match
            CASE WHEN t.template_type = ti.template_type THEN 0.3 ELSE 0.0 END +
            -- Category match
            CASE WHEN t.category = ti.category THEN 0.3 ELSE 0.0 END +
            -- Tag overlap
            COALESCE(
                jsonb_array_length(t.tags) * 1.0 / NULLIF(jsonb_array_length(t.tags || ti.tags), 0) * 0.2,
                0.0
            ) +
            -- Keyword overlap
            COALESCE(
                jsonb_array_length(t.keywords) * 1.0 / NULLIF(jsonb_array_length(t.keywords || ti.keywords), 0) * 0.2,
                0.0
            )
        )::NUMERIC(5,3) as similarity_score
    FROM templates t, template_info ti
    WHERE t.id != template_id_param
    AND t.deleted_at IS NULL
    AND t.is_latest = true
    AND t.status = 'active'
    ORDER BY similarity_score DESC, t.usage_count DESC
    LIMIT limit_count;
END;
$$ LANGUAGE plpgsql;

-- Comments for documentation
COMMENT ON TABLE templates IS 'Comprehensive template system for all template types with version control and usage tracking';
COMMENT ON TABLE template_usage_history IS 'Historical record of template usage for analytics and improvement';
COMMENT ON TABLE template_categories IS 'Hierarchical categorization system for templates';
COMMENT ON TABLE template_relationships IS 'Relationships between templates for dependency and extension management';

COMMENT ON COLUMN templates.variables IS 'Array of variable definitions with name, type, default value, and validation rules';
COMMENT ON COLUMN templates.parameters IS 'Default parameter values that can be overridden during template usage';
COMMENT ON COLUMN templates.config IS 'Template-specific configuration such as rendering options, validation settings, etc.';
COMMENT ON COLUMN templates.validation_rules IS 'Rules for validating template usage and parameter values';

COMMENT ON FUNCTION search_templates IS 'Full-text search for templates with filtering and ranking';
COMMENT ON FUNCTION get_template_versions IS 'Retrieve all versions of a template for version history';
COMMENT ON FUNCTION calculate_template_statistics IS 'Calculate usage statistics for templates within a date range';
COMMENT ON FUNCTION recommend_related_templates IS 'Recommend similar templates based on type, category, and tags';