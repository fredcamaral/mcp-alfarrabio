-- Migration 001: Create Enhanced Tasks Table
-- Description: Creates the main tasks table with TRD-specified columns plus enhanced fields for AI-powered task management
-- Created: 2025-06-09
-- Version: 1.0.0

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Create ENUM types for constrained values
CREATE TYPE task_status AS ENUM ('pending', 'in_progress', 'completed', 'cancelled', 'blocked', 'todo');
CREATE TYPE task_priority AS ENUM ('low', 'medium', 'high', 'critical', 'blocking');
CREATE TYPE task_complexity AS ENUM ('trivial', 'simple', 'moderate', 'complex', 'very_complex');
CREATE TYPE task_type AS ENUM (
    'implementation', 'design', 'testing', 'documentation', 'research', 
    'review', 'deployment', 'architecture', 'bugfix', 'refactoring', 
    'integration', 'analysis'
);
CREATE TYPE risk_level AS ENUM ('low', 'medium', 'high', 'critical');
CREATE TYPE impact_level AS ENUM ('low', 'medium', 'high', 'critical');

-- Create the enhanced tasks table combining TRD requirements with extended AI features
CREATE TABLE tasks (
    -- Primary identification (TRD compliant)
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    title VARCHAR(255) NOT NULL,
    description TEXT,
    content TEXT NOT NULL, -- TRD required: task content/description
    
    -- Task classification and status (TRD compliant + enhanced)
    type task_type NOT NULL DEFAULT 'implementation',
    status task_status NOT NULL DEFAULT 'pending',
    priority task_priority NOT NULL DEFAULT 'medium',
    complexity task_complexity DEFAULT 'simple',
    
    -- Ownership and assignment
    assignee VARCHAR(255),
    
    -- Repository and session context (TRD compliant)
    repository VARCHAR(255) NOT NULL,
    session_id VARCHAR(255),
    
    -- Time tracking (TRD compliant + enhanced)
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    started_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    due_date TIMESTAMP WITH TIME ZONE,
    
    -- Effort estimation (TRD compliant + enhanced)
    estimated_minutes INTEGER CHECK (estimated_minutes >= 0),
    actual_minutes INTEGER CHECK (actual_minutes >= 0),
    estimated_hours DECIMAL(8,2) CHECK (estimated_hours >= 0),
    estimated_days DECIMAL(6,2) CHECK (estimated_days >= 0),
    story_points INTEGER CHECK (story_points > 0),
    
    -- Quality and complexity scoring (AI-enhanced)
    complexity_score DECIMAL(3,2) CHECK (complexity_score >= 0 AND complexity_score <= 1),
    quality_score DECIMAL(3,2) CHECK (quality_score >= 0 AND quality_score <= 1),
    confidence_score DECIMAL(3,2) CHECK (confidence_score >= 0 AND confidence_score <= 1),
    business_value_score DECIMAL(3,2) CHECK (business_value_score >= 0 AND business_value_score <= 1),
    technical_debt_score DECIMAL(3,2) CHECK (technical_debt_score >= 0 AND technical_debt_score <= 1),
    user_impact_score DECIMAL(3,2) CHECK (user_impact_score >= 0 AND user_impact_score <= 1),
    
    -- Risk assessment
    technical_risk risk_level DEFAULT 'low',
    business_impact impact_level DEFAULT 'medium',
    
    -- Hierarchical relationships (TRD compliant)
    parent_task_id UUID REFERENCES tasks(id) ON DELETE CASCADE,
    
    -- Complex JSON fields (TRD compliant + enhanced)
    tags JSONB DEFAULT '[]'::jsonb,
    dependencies JSONB DEFAULT '[]'::jsonb,
    blocks JSONB DEFAULT '[]'::jsonb,
    acceptance_criteria JSONB DEFAULT '[]'::jsonb,
    required_skills JSONB DEFAULT '[]'::jsonb,
    external_dependencies JSONB DEFAULT '[]'::jsonb,
    
    -- AI and generation metadata (TRD compliant + enhanced)
    ai_suggested BOOLEAN DEFAULT FALSE,
    ai_model VARCHAR(100),
    generation_source VARCHAR(50) DEFAULT 'user_created',
    generation_prompt TEXT,
    template_id UUID,
    
    -- PRD integration (TRD compliant)
    prd_id UUID,
    source_prd_id UUID,
    source_section VARCHAR(255),
    pattern_id UUID,
    
    -- Branch and version control context
    branch VARCHAR(255),
    commit_hash VARCHAR(64),
    
    -- Extended metadata for flexibility
    metadata JSONB DEFAULT '{}'::jsonb,
    extended_data JSONB DEFAULT '{}'::jsonb,
    
    -- Audit and soft delete
    deleted_at TIMESTAMP WITH TIME ZONE,
    deleted_by VARCHAR(255),
    version INTEGER DEFAULT 1,
    
    -- Performance optimization hints
    search_vector tsvector,
    
    -- Constraints
    CONSTRAINT tasks_valid_parent CHECK (parent_task_id != id),
    CONSTRAINT tasks_valid_dates CHECK (
        (started_at IS NULL OR started_at >= created_at) AND
        (completed_at IS NULL OR completed_at >= created_at) AND
        (due_date IS NULL OR due_date >= created_at) AND
        (completed_at IS NULL OR started_at IS NULL OR completed_at >= started_at)
    ),
    CONSTRAINT tasks_completion_status CHECK (
        (status = 'completed' AND completed_at IS NOT NULL) OR 
        (status != 'completed' AND completed_at IS NULL)
    ),
    CONSTRAINT tasks_valid_effort CHECK (
        (estimated_minutes IS NULL OR estimated_hours IS NULL OR estimated_minutes <= estimated_hours * 60) AND
        (actual_minutes IS NULL OR estimated_minutes IS NULL OR actual_minutes <= estimated_minutes * 3)
    )
);

-- Create supporting tables for normalized data

-- PRD table (TRD compliant)
CREATE TABLE IF NOT EXISTS prds (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    repository VARCHAR(255) NOT NULL,
    filename VARCHAR(255) NOT NULL,
    content TEXT NOT NULL,
    parsed_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    task_count INTEGER DEFAULT 0,
    complexity_score DECIMAL(3,2) CHECK (complexity_score >= 0 AND complexity_score <= 1),
    metadata JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    deleted_at TIMESTAMP WITH TIME ZONE
);

-- Task patterns for AI learning
CREATE TABLE task_patterns (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    pattern_type VARCHAR(50) NOT NULL,
    template JSONB NOT NULL,
    usage_count INTEGER DEFAULT 0,
    success_rate DECIMAL(3,2) DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL
);

-- Task templates for reusable task structures
CREATE TABLE task_templates (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    category VARCHAR(100),
    template_data JSONB NOT NULL,
    applicability JSONB DEFAULT '{}'::jsonb,
    variables JSONB DEFAULT '[]'::jsonb,
    usage_count INTEGER DEFAULT 0,
    success_rate DECIMAL(3,2) DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL
);

-- Task effort breakdown for detailed estimation
CREATE TABLE task_effort_breakdown (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    task_id UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    analysis_hours DECIMAL(6,2) DEFAULT 0,
    design_hours DECIMAL(6,2) DEFAULT 0,
    implementation_hours DECIMAL(6,2) DEFAULT 0,
    testing_hours DECIMAL(6,2) DEFAULT 0,
    documentation_hours DECIMAL(6,2) DEFAULT 0,
    review_hours DECIMAL(6,2) DEFAULT 0,
    integration_hours DECIMAL(6,2) DEFAULT 0,
    deployment_hours DECIMAL(6,2) DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL
);

-- Task quality issues for quality tracking
CREATE TABLE task_quality_issues (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    task_id UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    issue_type VARCHAR(50) NOT NULL,
    severity VARCHAR(20) NOT NULL,
    description TEXT NOT NULL,
    suggestion TEXT,
    resolved BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    resolved_at TIMESTAMP WITH TIME ZONE
);

-- Add foreign key constraints for PRD relationships
ALTER TABLE tasks ADD CONSTRAINT fk_tasks_prd FOREIGN KEY (prd_id) REFERENCES prds(id) ON DELETE SET NULL;
ALTER TABLE tasks ADD CONSTRAINT fk_tasks_source_prd FOREIGN KEY (source_prd_id) REFERENCES prds(id) ON DELETE SET NULL;
ALTER TABLE tasks ADD CONSTRAINT fk_tasks_pattern FOREIGN KEY (pattern_id) REFERENCES task_patterns(id) ON DELETE SET NULL;
ALTER TABLE tasks ADD CONSTRAINT fk_tasks_template FOREIGN KEY (template_id) REFERENCES task_templates(id) ON DELETE SET NULL;

-- Add updated_at trigger function
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Create triggers for automatic updated_at
CREATE TRIGGER update_tasks_updated_at BEFORE UPDATE ON tasks
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_prds_updated_at BEFORE UPDATE ON prds
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_task_patterns_updated_at BEFORE UPDATE ON task_patterns
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_task_templates_updated_at BEFORE UPDATE ON task_templates
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_task_effort_breakdown_updated_at BEFORE UPDATE ON task_effort_breakdown
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Create search vector trigger for full-text search
CREATE OR REPLACE FUNCTION update_task_search_vector()
RETURNS TRIGGER AS $$
BEGIN
    NEW.search_vector = 
        setweight(to_tsvector('english', COALESCE(NEW.title, '')), 'A') ||
        setweight(to_tsvector('english', COALESCE(NEW.description, '')), 'B') ||
        setweight(to_tsvector('english', COALESCE(NEW.content, '')), 'C') ||
        setweight(to_tsvector('english', COALESCE(NEW.assignee, '')), 'D') ||
        setweight(to_tsvector('english', COALESCE(NEW.source_section, '')), 'D');
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_task_search_vector_trigger BEFORE INSERT OR UPDATE ON tasks
    FOR EACH ROW EXECUTE FUNCTION update_task_search_vector();

-- Comments for documentation
COMMENT ON TABLE tasks IS 'Enhanced task table combining TRD requirements with AI-powered task management features';
COMMENT ON COLUMN tasks.content IS 'TRD required: main task content/description';
COMMENT ON COLUMN tasks.ai_suggested IS 'TRD required: indicates if task was AI-generated';
COMMENT ON COLUMN tasks.pattern_id IS 'TRD required: links to task generation patterns';
COMMENT ON COLUMN tasks.prd_id IS 'TRD required: links to source PRD document';
COMMENT ON COLUMN tasks.source_section IS 'TRD required: specific section in PRD that generated this task';
COMMENT ON COLUMN tasks.search_vector IS 'Full-text search vector for performance optimization';
COMMENT ON COLUMN tasks.deleted_at IS 'Soft delete timestamp - NULL means active';
COMMENT ON COLUMN tasks.version IS 'Optimistic locking version number';