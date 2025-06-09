-- Migration 006: Create Template and Pattern Tables
-- Description: Creates task templates and task patterns tables for reusable structures and ML insights
-- Created: 2025-06-09
-- Version: 1.0.0

-- Enable UUID extension if not already enabled
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Create task templates table for reusable task structures
CREATE TABLE task_templates (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    category VARCHAR(100),
    
    -- Template data structure
    template_data JSONB NOT NULL DEFAULT '{}',
    applicability JSONB DEFAULT '{}', -- Conditions when this template applies
    variables JSONB DEFAULT '[]', -- Template variables that can be customized
    
    -- Template metadata
    project_type VARCHAR(100),
    complexity_level VARCHAR(20) DEFAULT 'medium' CHECK (complexity_level IN ('trivial', 'simple', 'moderate', 'complex', 'very_complex')),
    estimated_effort_hours NUMERIC(10,2),
    required_skills JSONB DEFAULT '[]',
    
    -- Usage and success metrics
    usage_count INTEGER DEFAULT 0 NOT NULL,
    success_rate NUMERIC(5,3) CHECK (success_rate >= 0 AND success_rate <= 1),
    avg_completion_time_hours NUMERIC(10,2),
    
    -- Ratings and feedback
    user_rating NUMERIC(3,2) CHECK (user_rating >= 0 AND user_rating <= 5),
    feedback_count INTEGER DEFAULT 0,
    
    -- Template versioning
    version INTEGER DEFAULT 1 NOT NULL,
    parent_template_id UUID REFERENCES task_templates(id),
    is_active BOOLEAN DEFAULT true,
    
    -- Metadata
    created_by VARCHAR(255),
    tags JSONB DEFAULT '[]',
    metadata JSONB DEFAULT '{}',
    
    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    deleted_at TIMESTAMP WITH TIME ZONE,
    
    -- Search optimization
    search_vector TSVECTOR GENERATED ALWAYS AS (
        setweight(to_tsvector('english', COALESCE(name, '')), 'A') ||
        setweight(to_tsvector('english', COALESCE(description, '')), 'B') ||
        setweight(to_tsvector('english', COALESCE(category, '')), 'C')
    ) STORED
);

-- Create task patterns table for ML insights and pattern recognition
CREATE TABLE task_patterns (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    pattern_type VARCHAR(50) NOT NULL CHECK (pattern_type IN ('sequence', 'dependency', 'parallel', 'conditional', 'iterative', 'workflow')),
    
    -- Pattern definition
    template JSONB NOT NULL DEFAULT '{}', -- The pattern structure
    conditions JSONB DEFAULT '{}', -- When this pattern applies
    task_sequence JSONB DEFAULT '[]', -- Ordered sequence of task types
    
    -- Pattern metrics
    occurrence_count INTEGER DEFAULT 0 NOT NULL,
    avg_completion_time_minutes INTEGER,
    success_rate NUMERIC(5,3) CHECK (success_rate >= 0 AND success_rate <= 1),
    efficiency_score NUMERIC(5,3) CHECK (efficiency_score >= 0 AND efficiency_score <= 1),
    
    -- Context and applicability
    repositories JSONB DEFAULT '[]', -- Where this pattern was found
    project_types JSONB DEFAULT '[]', -- Types of projects this applies to
    team_sizes JSONB DEFAULT '[]', -- Team sizes where this pattern works
    complexity_levels JSONB DEFAULT '[]', -- Complexity levels this handles
    
    -- Pattern relationships
    parent_pattern_id UUID REFERENCES task_patterns(id),
    related_patterns JSONB DEFAULT '[]', -- IDs of related patterns
    
    -- Machine learning features
    confidence_score NUMERIC(5,3) CHECK (confidence_score >= 0 AND confidence_score <= 1),
    feature_vector JSONB DEFAULT '{}', -- ML feature representation
    last_trained_at TIMESTAMP WITH TIME ZONE,
    
    -- Usage tracking
    last_used_at TIMESTAMP WITH TIME ZONE,
    auto_suggested_count INTEGER DEFAULT 0,
    user_accepted_count INTEGER DEFAULT 0,
    user_rejected_count INTEGER DEFAULT 0,
    
    -- Status and lifecycle
    status VARCHAR(20) DEFAULT 'active' CHECK (status IN ('active', 'experimental', 'deprecated', 'archived')),
    validation_status VARCHAR(20) DEFAULT 'pending' CHECK (validation_status IN ('pending', 'validated', 'rejected', 'needs_review')),
    
    -- Metadata
    discovered_by VARCHAR(20) DEFAULT 'system' CHECK (discovered_by IN ('system', 'user', 'ai', 'import')),
    created_by VARCHAR(255),
    tags JSONB DEFAULT '[]',
    metadata JSONB DEFAULT '{}',
    
    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    deleted_at TIMESTAMP WITH TIME ZONE,
    
    -- Search optimization
    search_vector TSVECTOR GENERATED ALWAYS AS (
        setweight(to_tsvector('english', COALESCE(name, '')), 'A') ||
        setweight(to_tsvector('english', COALESCE(description, '')), 'B') ||
        setweight(to_tsvector('english', COALESCE(pattern_type, '')), 'C')
    ) STORED
);

-- Create constraints
ALTER TABLE task_templates ADD CONSTRAINT chk_task_templates_usage_count_positive 
    CHECK (usage_count >= 0);

ALTER TABLE task_templates ADD CONSTRAINT chk_task_templates_version_positive 
    CHECK (version > 0);

ALTER TABLE task_patterns ADD CONSTRAINT chk_task_patterns_occurrence_count_positive 
    CHECK (occurrence_count >= 0);

ALTER TABLE task_patterns ADD CONSTRAINT chk_task_patterns_completion_time_positive 
    CHECK (avg_completion_time_minutes IS NULL OR avg_completion_time_minutes > 0);

-- Prevent circular references in templates
ALTER TABLE task_templates ADD CONSTRAINT chk_task_templates_no_self_reference 
    CHECK (id != parent_template_id);

-- Prevent circular references in patterns
ALTER TABLE task_patterns ADD CONSTRAINT chk_task_patterns_no_self_reference 
    CHECK (id != parent_pattern_id);

-- Create unique constraints
CREATE UNIQUE INDEX idx_task_templates_unique_name 
    ON task_templates(name) 
    WHERE deleted_at IS NULL;

CREATE UNIQUE INDEX idx_task_patterns_unique_name 
    ON task_patterns(name) 
    WHERE deleted_at IS NULL;

-- Create performance indexes for task_templates
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_task_templates_category ON task_templates(category) WHERE deleted_at IS NULL;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_task_templates_project_type ON task_templates(project_type) WHERE deleted_at IS NULL;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_task_templates_complexity ON task_templates(complexity_level) WHERE deleted_at IS NULL;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_task_templates_usage ON task_templates(usage_count DESC) WHERE deleted_at IS NULL;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_task_templates_success ON task_templates(success_rate DESC) WHERE deleted_at IS NULL;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_task_templates_rating ON task_templates(user_rating DESC) WHERE deleted_at IS NULL;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_task_templates_active ON task_templates(is_active) WHERE deleted_at IS NULL;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_task_templates_created_at ON task_templates(created_at DESC) WHERE deleted_at IS NULL;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_task_templates_search_vector ON task_templates USING GIN(search_vector);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_task_templates_tags ON task_templates USING GIN(tags) WHERE deleted_at IS NULL;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_task_templates_parent ON task_templates(parent_template_id) WHERE deleted_at IS NULL;

-- Create performance indexes for task_patterns
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_task_patterns_type ON task_patterns(pattern_type) WHERE deleted_at IS NULL;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_task_patterns_occurrence ON task_patterns(occurrence_count DESC) WHERE deleted_at IS NULL;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_task_patterns_success ON task_patterns(success_rate DESC) WHERE deleted_at IS NULL;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_task_patterns_efficiency ON task_patterns(efficiency_score DESC) WHERE deleted_at IS NULL;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_task_patterns_confidence ON task_patterns(confidence_score DESC) WHERE deleted_at IS NULL;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_task_patterns_status ON task_patterns(status) WHERE deleted_at IS NULL;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_task_patterns_validation ON task_patterns(validation_status) WHERE deleted_at IS NULL;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_task_patterns_last_used ON task_patterns(last_used_at DESC) WHERE deleted_at IS NULL;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_task_patterns_created_at ON task_patterns(created_at DESC) WHERE deleted_at IS NULL;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_task_patterns_search_vector ON task_patterns USING GIN(search_vector);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_task_patterns_tags ON task_patterns USING GIN(tags) WHERE deleted_at IS NULL;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_task_patterns_parent ON task_patterns(parent_pattern_id) WHERE deleted_at IS NULL;

-- Composite indexes for common query patterns
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_task_templates_category_active ON task_templates(category, is_active) WHERE deleted_at IS NULL;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_task_templates_type_complexity ON task_templates(project_type, complexity_level) WHERE deleted_at IS NULL;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_task_patterns_type_status ON task_patterns(pattern_type, status) WHERE deleted_at IS NULL;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_task_patterns_confidence_usage ON task_patterns(confidence_score DESC, occurrence_count DESC) WHERE deleted_at IS NULL;

-- Create triggers

-- Update updated_at timestamp triggers
CREATE OR REPLACE FUNCTION update_task_templates_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_task_templates_updated_at_trigger
    BEFORE UPDATE ON task_templates
    FOR EACH ROW EXECUTE FUNCTION update_task_templates_updated_at();

CREATE OR REPLACE FUNCTION update_task_patterns_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_task_patterns_updated_at_trigger
    BEFORE UPDATE ON task_patterns
    FOR EACH ROW EXECUTE FUNCTION update_task_patterns_updated_at();

-- Version increment trigger for templates
CREATE OR REPLACE FUNCTION increment_template_version()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'UPDATE' AND OLD.template_data IS DISTINCT FROM NEW.template_data THEN
        NEW.version = OLD.version + 1;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER template_version_increment_trigger
    BEFORE UPDATE ON task_templates
    FOR EACH ROW EXECUTE FUNCTION increment_template_version();

-- Pattern usage tracking trigger
CREATE OR REPLACE FUNCTION track_pattern_usage()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'UPDATE' AND OLD.occurrence_count IS DISTINCT FROM NEW.occurrence_count THEN
        NEW.last_used_at = CURRENT_TIMESTAMP;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER pattern_usage_tracking_trigger
    BEFORE UPDATE ON task_patterns
    FOR EACH ROW EXECUTE FUNCTION track_pattern_usage();

-- Create helper functions

-- Function to get template recommendations based on project context
CREATE OR REPLACE FUNCTION get_template_recommendations(
    project_type_param VARCHAR(100) DEFAULT NULL,
    complexity_param VARCHAR(20) DEFAULT NULL,
    required_skills_param JSONB DEFAULT NULL,
    limit_count INTEGER DEFAULT 10
)
RETURNS TABLE (
    id UUID,
    name VARCHAR(255),
    description TEXT,
    category VARCHAR(100),
    success_rate NUMERIC,
    usage_count INTEGER,
    user_rating NUMERIC,
    relevance_score NUMERIC
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        t.id,
        t.name,
        t.description,
        t.category,
        t.success_rate,
        t.usage_count,
        t.user_rating,
        -- Calculate relevance score based on multiple factors
        (
            CASE WHEN project_type_param IS NULL OR t.project_type = project_type_param THEN 0.3 ELSE 0.0 END +
            CASE WHEN complexity_param IS NULL OR t.complexity_level = complexity_param THEN 0.2 ELSE 0.0 END +
            CASE WHEN required_skills_param IS NULL OR t.required_skills ?| (SELECT array_agg(value::text) FROM jsonb_array_elements_text(required_skills_param)) THEN 0.2 ELSE 0.0 END +
            COALESCE(t.success_rate, 0.0) * 0.2 +
            LEAST(t.usage_count / 100.0, 1.0) * 0.1
        )::NUMERIC(5,3) as relevance_score
    FROM task_templates t
    WHERE t.deleted_at IS NULL 
    AND t.is_active = true
    ORDER BY relevance_score DESC, t.usage_count DESC, t.success_rate DESC
    LIMIT limit_count;
END;
$$ LANGUAGE plpgsql;

-- Function to get pattern suggestions for task sequences
CREATE OR REPLACE FUNCTION get_pattern_suggestions(
    task_types_param JSONB,
    repository_param VARCHAR(255) DEFAULT NULL,
    limit_count INTEGER DEFAULT 5
)
RETURNS TABLE (
    id UUID,
    name VARCHAR(255),
    pattern_type VARCHAR(50),
    confidence_score NUMERIC,
    success_rate NUMERIC,
    avg_completion_time_minutes INTEGER,
    match_score NUMERIC
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        p.id,
        p.name,
        p.pattern_type,
        p.confidence_score,
        p.success_rate,
        p.avg_completion_time_minutes,
        -- Calculate match score based on task sequence similarity
        (
            CASE WHEN p.task_sequence @> task_types_param THEN 0.5 ELSE 0.0 END +
            CASE WHEN repository_param IS NULL OR p.repositories @> to_jsonb(repository_param) THEN 0.2 ELSE 0.0 END +
            COALESCE(p.confidence_score, 0.0) * 0.2 +
            COALESCE(p.success_rate, 0.0) * 0.1
        )::NUMERIC(5,3) as match_score
    FROM task_patterns p
    WHERE p.deleted_at IS NULL 
    AND p.status = 'active'
    AND p.validation_status = 'validated'
    ORDER BY match_score DESC, p.confidence_score DESC, p.success_rate DESC
    LIMIT limit_count;
END;
$$ LANGUAGE plpgsql;

-- Function to update pattern metrics
CREATE OR REPLACE FUNCTION update_pattern_metrics(
    pattern_id_param UUID,
    completion_time_minutes INTEGER,
    was_successful BOOLEAN
)
RETURNS VOID AS $$
DECLARE
    current_avg_time INTEGER;
    current_count INTEGER;
    current_success_count INTEGER;
BEGIN
    -- Get current metrics
    SELECT avg_completion_time_minutes, occurrence_count, 
           COALESCE(ROUND(occurrence_count * COALESCE(success_rate, 0)), 0)
    INTO current_avg_time, current_count, current_success_count
    FROM task_patterns 
    WHERE id = pattern_id_param;
    
    -- Update metrics
    UPDATE task_patterns 
    SET 
        occurrence_count = occurrence_count + 1,
        avg_completion_time_minutes = CASE 
            WHEN current_avg_time IS NULL THEN completion_time_minutes
            ELSE ((current_avg_time * current_count) + completion_time_minutes) / (current_count + 1)
        END,
        success_rate = (current_success_count + CASE WHEN was_successful THEN 1 ELSE 0 END) / (current_count + 1.0),
        last_used_at = CURRENT_TIMESTAMP,
        updated_at = CURRENT_TIMESTAMP
    WHERE id = pattern_id_param;
END;
$$ LANGUAGE plpgsql;

-- Function to archive unused templates
CREATE OR REPLACE FUNCTION archive_unused_templates(months_unused INTEGER DEFAULT 6)
RETURNS INTEGER AS $$
DECLARE
    archived_count INTEGER;
    cutoff_date TIMESTAMP WITH TIME ZONE;
BEGIN
    cutoff_date := CURRENT_TIMESTAMP - (months_unused || ' months')::INTERVAL;
    
    UPDATE task_templates 
    SET is_active = false,
        updated_at = CURRENT_TIMESTAMP
    WHERE is_active = true 
    AND updated_at < cutoff_date
    AND usage_count = 0  -- Only archive unused templates
    AND deleted_at IS NULL;
    
    GET DIAGNOSTICS archived_count = ROW_COUNT;
    
    RETURN archived_count;
END;
$$ LANGUAGE plpgsql;

-- Function to get template and pattern statistics
CREATE OR REPLACE FUNCTION get_template_pattern_statistics()
RETURNS TABLE (
    metric_name TEXT,
    metric_value NUMERIC,
    description TEXT
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        'active_templates'::TEXT,
        COUNT(*)::NUMERIC,
        'Number of active task templates'::TEXT
    FROM task_templates 
    WHERE deleted_at IS NULL AND is_active = true
    
    UNION ALL
    
    SELECT 
        'validated_patterns'::TEXT,
        COUNT(*)::NUMERIC,
        'Number of validated task patterns'::TEXT
    FROM task_patterns 
    WHERE deleted_at IS NULL AND validation_status = 'validated'
    
    UNION ALL
    
    SELECT 
        'avg_template_success_rate'::TEXT,
        COALESCE(AVG(success_rate), 0),
        'Average success rate of templates'::TEXT
    FROM task_templates 
    WHERE deleted_at IS NULL AND success_rate IS NOT NULL
    
    UNION ALL
    
    SELECT 
        'avg_pattern_confidence'::TEXT,
        COALESCE(AVG(confidence_score), 0),
        'Average confidence score of patterns'::TEXT
    FROM task_patterns 
    WHERE deleted_at IS NULL AND confidence_score IS NOT NULL;
END;
$$ LANGUAGE plpgsql;

-- Comments for documentation
COMMENT ON TABLE task_templates IS 'Reusable task structures with metadata and success metrics';
COMMENT ON TABLE task_patterns IS 'Machine learning insights and pattern recognition for task sequences';
COMMENT ON COLUMN task_templates.template_data IS 'JSON structure defining the template with variables and structure';
COMMENT ON COLUMN task_templates.applicability IS 'Conditions and criteria when this template should be applied';
COMMENT ON COLUMN task_patterns.template IS 'Pattern definition with task relationships and dependencies';
COMMENT ON COLUMN task_patterns.feature_vector IS 'Machine learning feature representation for pattern matching';
COMMENT ON FUNCTION get_template_recommendations(VARCHAR, VARCHAR, JSONB, INTEGER) IS 'Returns template recommendations based on project context';
COMMENT ON FUNCTION get_pattern_suggestions(JSONB, VARCHAR, INTEGER) IS 'Suggests patterns based on task sequences and context';
COMMENT ON FUNCTION update_pattern_metrics(UUID, INTEGER, BOOLEAN) IS 'Updates pattern success metrics after task completion';
COMMENT ON FUNCTION archive_unused_templates(INTEGER) IS 'Archives templates that haven not been used recently';