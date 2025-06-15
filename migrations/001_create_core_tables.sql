-- Migration 001: Create Core Tables
-- Tasks and PRDs tables for the task management system

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Create ENUM types
CREATE TYPE task_status AS ENUM ('pending', 'in_progress', 'completed', 'cancelled', 'blocked', 'todo');
CREATE TYPE task_priority AS ENUM ('low', 'medium', 'high', 'critical', 'blocking');
CREATE TYPE task_type AS ENUM (
    'implementation', 'design', 'testing', 'documentation', 'research', 
    'review', 'deployment', 'architecture', 'bugfix', 'refactoring', 
    'integration', 'analysis'
);

-- Create update timestamp function (shared by all tables)
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Create PRDs table
CREATE TABLE prds (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    repository VARCHAR(255) NOT NULL,
    filename VARCHAR(255) NOT NULL,
    content TEXT NOT NULL,
    parsed_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    metadata JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    
    UNIQUE(repository, filename)
);

-- Create tasks table with all required columns
CREATE TABLE tasks (
    -- Primary identification
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    title VARCHAR(255) NOT NULL,
    description TEXT,
    
    -- Task classification
    type task_type NOT NULL DEFAULT 'implementation',
    status task_status NOT NULL DEFAULT 'pending',
    priority task_priority NOT NULL DEFAULT 'medium',
    
    -- Assignment and context
    assignee VARCHAR(255),
    repository VARCHAR(255) NOT NULL,
    session_id VARCHAR(255),
    prd_id UUID REFERENCES prds(id) ON DELETE SET NULL,
    
    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    started_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    deleted_at TIMESTAMP WITH TIME ZONE,
    due_date TIMESTAMP WITH TIME ZONE,
    
    -- Task details
    acceptance_criteria JSONB DEFAULT '[]'::jsonb,
    dependencies JSONB DEFAULT '[]'::jsonb,
    tags JSONB DEFAULT '[]'::jsonb,
    
    -- Effort and complexity
    estimated_hours DECIMAL(8,2),
    complexity VARCHAR(50) DEFAULT 'simple',
    complexity_score DECIMAL(3,2) DEFAULT 0.5,
    quality_score DECIMAL(3,2) DEFAULT 0.8,
    
    -- Extended metadata
    metadata JSONB DEFAULT '{}'::jsonb
);

-- Create indexes
CREATE INDEX idx_tasks_repository ON tasks(repository);
CREATE INDEX idx_tasks_status ON tasks(status);
CREATE INDEX idx_tasks_assignee ON tasks(assignee) WHERE assignee IS NOT NULL;
CREATE INDEX idx_tasks_created_at ON tasks(created_at DESC);
CREATE INDEX idx_tasks_deleted_at ON tasks(deleted_at) WHERE deleted_at IS NULL;
CREATE INDEX idx_prds_repository ON prds(repository);

-- Create triggers
CREATE TRIGGER update_tasks_updated_at 
    BEFORE UPDATE ON tasks
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_prds_updated_at 
    BEFORE UPDATE ON prds
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();

-- Insert migration record
INSERT INTO schema_migrations (version, name) VALUES ('001', 'create_core_tables');