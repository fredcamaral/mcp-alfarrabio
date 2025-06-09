-- Migration 007: Create Work Sessions Table
-- Description: Creates work sessions table for productivity tracking and analytics
-- Created: 2025-06-09
-- Version: 1.0.0

-- Enable UUID extension if not already enabled
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Create work sessions table for productivity tracking
CREATE TABLE work_sessions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    
    -- Session identification
    session_id VARCHAR(255) NOT NULL, -- External session ID from client
    repository VARCHAR(255) NOT NULL,
    branch VARCHAR(255),
    
    -- Time tracking
    start_time TIMESTAMP WITH TIME ZONE NOT NULL,
    end_time TIMESTAMP WITH TIME ZONE,
    duration_minutes INTEGER GENERATED ALWAYS AS (
        CASE 
            WHEN end_time IS NOT NULL THEN 
                EXTRACT(EPOCH FROM (end_time - start_time)) / 60
            ELSE NULL 
        END
    ) STORED,
    
    -- Task activity
    tasks_completed JSONB DEFAULT '[]' NOT NULL,
    tasks_started JSONB DEFAULT '[]' NOT NULL,
    tasks_in_progress JSONB DEFAULT '[]' NOT NULL,
    tasks_blocked JSONB DEFAULT '[]' NOT NULL,
    
    -- Development activity
    tools_used JSONB DEFAULT '[]' NOT NULL,
    files_changed JSONB DEFAULT '[]' NOT NULL,
    commits_made JSONB DEFAULT '[]' NOT NULL,
    commands_executed JSONB DEFAULT '[]' NOT NULL,
    
    -- Session context
    session_type VARCHAR(50) DEFAULT 'development' CHECK (session_type IN ('development', 'planning', 'review', 'debugging', 'testing', 'documentation', 'research')),
    work_mode VARCHAR(20) DEFAULT 'focused' CHECK (work_mode IN ('focused', 'exploratory', 'collaborative', 'maintenance', 'learning')),
    session_summary TEXT,
    session_notes TEXT,
    
    -- Productivity metrics
    productivity_score NUMERIC(5,3) CHECK (productivity_score >= 0 AND productivity_score <= 1),
    focus_score NUMERIC(5,3) CHECK (focus_score >= 0 AND focus_score <= 1),
    efficiency_score NUMERIC(5,3) CHECK (efficiency_score >= 0 AND focus_score <= 1),
    quality_score NUMERIC(5,3) CHECK (quality_score >= 0 AND quality_score <= 1),
    
    -- Activity counts
    total_tasks_touched INTEGER DEFAULT 0,
    tasks_completed_count INTEGER DEFAULT 0,
    files_modified_count INTEGER DEFAULT 0,
    lines_added INTEGER DEFAULT 0,
    lines_deleted INTEGER DEFAULT 0,
    
    -- Tool and environment
    cli_mode BOOLEAN DEFAULT false,
    ide_used VARCHAR(100),
    operating_system VARCHAR(50),
    environment VARCHAR(50), -- dev, staging, production
    
    -- Interruptions and breaks
    interruption_count INTEGER DEFAULT 0,
    break_duration_minutes INTEGER DEFAULT 0,
    context_switches INTEGER DEFAULT 0,
    
    -- AI assistance
    ai_interactions_count INTEGER DEFAULT 0,
    ai_suggestions_accepted INTEGER DEFAULT 0,
    ai_suggestions_rejected INTEGER DEFAULT 0,
    ai_generated_tasks_count INTEGER DEFAULT 0,
    
    -- Session status
    status VARCHAR(20) DEFAULT 'active' CHECK (status IN ('active', 'paused', 'completed', 'abandoned', 'interrupted')),
    completion_reason VARCHAR(100),
    
    -- Goals and outcomes
    session_goals JSONB DEFAULT '[]',
    goals_achieved JSONB DEFAULT '[]',
    blockers_encountered JSONB DEFAULT '[]',
    learnings JSONB DEFAULT '[]',
    
    -- Team collaboration
    collaborators JSONB DEFAULT '[]',
    pair_programming BOOLEAN DEFAULT false,
    code_reviews_given INTEGER DEFAULT 0,
    code_reviews_received INTEGER DEFAULT 0,
    
    -- Metadata and tags
    tags JSONB DEFAULT '[]',
    metadata JSONB DEFAULT '{}',
    
    -- User identification
    user_id VARCHAR(255),
    user_email VARCHAR(255),
    
    -- External integrations
    github_session_id VARCHAR(255),
    jira_session_id VARCHAR(255),
    external_tool_data JSONB DEFAULT '{}',
    
    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    deleted_at TIMESTAMP WITH TIME ZONE,
    
    -- Search optimization
    search_vector TSVECTOR GENERATED ALWAYS AS (
        setweight(to_tsvector('english', COALESCE(session_summary, '')), 'A') ||
        setweight(to_tsvector('english', COALESCE(session_notes, '')), 'B') ||
        setweight(to_tsvector('english', COALESCE(repository, '')), 'C')
    ) STORED
);

-- Create constraints
ALTER TABLE work_sessions ADD CONSTRAINT chk_work_sessions_valid_duration 
    CHECK (end_time IS NULL OR end_time > start_time);

ALTER TABLE work_sessions ADD CONSTRAINT chk_work_sessions_positive_counts 
    CHECK (
        total_tasks_touched >= 0 AND
        tasks_completed_count >= 0 AND
        files_modified_count >= 0 AND
        interruption_count >= 0 AND
        break_duration_minutes >= 0 AND
        context_switches >= 0 AND
        ai_interactions_count >= 0
    );

ALTER TABLE work_sessions ADD CONSTRAINT chk_work_sessions_completed_count_logic
    CHECK (tasks_completed_count <= total_tasks_touched);

-- Create unique constraint to prevent duplicate active sessions
CREATE UNIQUE INDEX idx_work_sessions_unique_active_session 
    ON work_sessions(session_id, repository) 
    WHERE deleted_at IS NULL AND status = 'active';

-- Create performance indexes
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_work_sessions_session_id ON work_sessions(session_id) WHERE deleted_at IS NULL;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_work_sessions_repository ON work_sessions(repository) WHERE deleted_at IS NULL;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_work_sessions_user_id ON work_sessions(user_id) WHERE deleted_at IS NULL;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_work_sessions_start_time ON work_sessions(start_time DESC) WHERE deleted_at IS NULL;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_work_sessions_end_time ON work_sessions(end_time DESC) WHERE deleted_at IS NULL;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_work_sessions_duration ON work_sessions(duration_minutes DESC) WHERE deleted_at IS NULL;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_work_sessions_status ON work_sessions(status) WHERE deleted_at IS NULL;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_work_sessions_session_type ON work_sessions(session_type) WHERE deleted_at IS NULL;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_work_sessions_productivity ON work_sessions(productivity_score DESC) WHERE deleted_at IS NULL;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_work_sessions_tasks_completed ON work_sessions(tasks_completed_count DESC) WHERE deleted_at IS NULL;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_work_sessions_search_vector ON work_sessions USING GIN(search_vector);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_work_sessions_tags ON work_sessions USING GIN(tags) WHERE deleted_at IS NULL;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_work_sessions_created_at ON work_sessions(created_at DESC) WHERE deleted_at IS NULL;

-- Composite indexes for common query patterns
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_work_sessions_user_repo ON work_sessions(user_id, repository) WHERE deleted_at IS NULL;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_work_sessions_repo_status ON work_sessions(repository, status) WHERE deleted_at IS NULL;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_work_sessions_user_timerange ON work_sessions(user_id, start_time DESC) WHERE deleted_at IS NULL;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_work_sessions_type_productivity ON work_sessions(session_type, productivity_score DESC) WHERE deleted_at IS NULL;

-- Date-based indexes for analytics
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_work_sessions_start_date ON work_sessions(DATE(start_time)) WHERE deleted_at IS NULL;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_work_sessions_month ON work_sessions(date_trunc('month', start_time)) WHERE deleted_at IS NULL;

-- Create triggers

-- Update updated_at timestamp trigger
CREATE OR REPLACE FUNCTION update_work_sessions_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_work_sessions_updated_at_trigger
    BEFORE UPDATE ON work_sessions
    FOR EACH ROW EXECUTE FUNCTION update_work_sessions_updated_at();

-- Session completion trigger
CREATE OR REPLACE FUNCTION handle_session_completion()
RETURNS TRIGGER AS $$
BEGIN
    -- Auto-complete session if end_time is set but status is still active
    IF TG_OP = 'UPDATE' AND NEW.end_time IS NOT NULL AND NEW.status = 'active' THEN
        NEW.status = 'completed';
        
        -- Calculate productivity metrics if not set
        IF NEW.productivity_score IS NULL THEN
            NEW.productivity_score = LEAST(
                (NEW.tasks_completed_count::DECIMAL / NULLIF(NEW.total_tasks_touched, 0)) * 0.5 +
                (CASE WHEN NEW.duration_minutes > 0 THEN LEAST(NEW.tasks_completed_count::DECIMAL / (NEW.duration_minutes / 60.0), 1.0) ELSE 0 END) * 0.3 +
                (CASE WHEN NEW.ai_interactions_count > 0 THEN (NEW.ai_suggestions_accepted::DECIMAL / NEW.ai_interactions_count) ELSE 0.5 END) * 0.2,
                1.0
            );
        END IF;
        
        -- Set completion reason if not provided
        IF NEW.completion_reason IS NULL THEN
            NEW.completion_reason = 'normal_completion';
        END IF;
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER session_completion_handler
    BEFORE UPDATE ON work_sessions
    FOR EACH ROW EXECUTE FUNCTION handle_session_completion();

-- Task activity counter trigger
CREATE OR REPLACE FUNCTION update_session_task_counts()
RETURNS TRIGGER AS $$
BEGIN
    -- Update task counts based on JSON arrays
    NEW.total_tasks_touched = (
        jsonb_array_length(COALESCE(NEW.tasks_completed, '[]'::jsonb)) +
        jsonb_array_length(COALESCE(NEW.tasks_started, '[]'::jsonb)) +
        jsonb_array_length(COALESCE(NEW.tasks_in_progress, '[]'::jsonb)) +
        jsonb_array_length(COALESCE(NEW.tasks_blocked, '[]'::jsonb))
    );
    
    NEW.tasks_completed_count = jsonb_array_length(COALESCE(NEW.tasks_completed, '[]'::jsonb));
    NEW.files_modified_count = jsonb_array_length(COALESCE(NEW.files_changed, '[]'::jsonb));
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER session_task_counts_updater
    BEFORE INSERT OR UPDATE ON work_sessions
    FOR EACH ROW EXECUTE FUNCTION update_session_task_counts();

-- Create helper functions

-- Function to get productivity insights for a user
CREATE OR REPLACE FUNCTION get_user_productivity_insights(
    user_id_param VARCHAR(255),
    days_back INTEGER DEFAULT 30
)
RETURNS TABLE (
    metric_name TEXT,
    metric_value NUMERIC,
    trend_direction TEXT,
    description TEXT
) AS $$
DECLARE
    cutoff_date TIMESTAMP WITH TIME ZONE;
    avg_productivity NUMERIC;
    avg_session_duration NUMERIC;
    total_sessions INTEGER;
    completed_sessions INTEGER;
BEGIN
    cutoff_date := CURRENT_TIMESTAMP - (days_back || ' days')::INTERVAL;
    
    -- Calculate key metrics
    SELECT 
        AVG(productivity_score),
        AVG(duration_minutes),
        COUNT(*),
        COUNT(*) FILTER (WHERE status = 'completed')
    INTO avg_productivity, avg_session_duration, total_sessions, completed_sessions
    FROM work_sessions 
    WHERE user_id = user_id_param 
    AND start_time >= cutoff_date 
    AND deleted_at IS NULL;
    
    RETURN QUERY
    SELECT 
        'avg_productivity_score'::TEXT,
        COALESCE(avg_productivity, 0),
        'stable'::TEXT, -- TODO: Calculate actual trend
        'Average productivity score over the period'::TEXT
    
    UNION ALL
    
    SELECT 
        'avg_session_duration_hours'::TEXT,
        COALESCE(avg_session_duration / 60.0, 0),
        'stable'::TEXT,
        'Average session duration in hours'::TEXT
    
    UNION ALL
    
    SELECT 
        'session_completion_rate'::TEXT,
        CASE WHEN total_sessions > 0 THEN completed_sessions::DECIMAL / total_sessions ELSE 0 END,
        'stable'::TEXT,
        'Percentage of sessions completed normally'::TEXT
    
    UNION ALL
    
    SELECT 
        'total_sessions'::TEXT,
        total_sessions::NUMERIC,
        'stable'::TEXT,
        'Total number of work sessions in period'::TEXT;
END;
$$ LANGUAGE plpgsql;

-- Function to get repository activity summary
CREATE OR REPLACE FUNCTION get_repository_activity_summary(
    repository_param VARCHAR(255),
    days_back INTEGER DEFAULT 7
)
RETURNS TABLE (
    metric_name TEXT,
    metric_value NUMERIC,
    unit TEXT
) AS $$
DECLARE
    cutoff_date TIMESTAMP WITH TIME ZONE;
BEGIN
    cutoff_date := CURRENT_TIMESTAMP - (days_back || ' days')::INTERVAL;
    
    RETURN QUERY
    SELECT 
        'active_sessions'::TEXT,
        COUNT(*)::NUMERIC,
        'sessions'::TEXT
    FROM work_sessions 
    WHERE repository = repository_param 
    AND start_time >= cutoff_date 
    AND deleted_at IS NULL
    
    UNION ALL
    
    SELECT 
        'unique_contributors'::TEXT,
        COUNT(DISTINCT user_id)::NUMERIC,
        'users'::TEXT
    FROM work_sessions 
    WHERE repository = repository_param 
    AND start_time >= cutoff_date 
    AND deleted_at IS NULL
    
    UNION ALL
    
    SELECT 
        'total_tasks_completed'::TEXT,
        SUM(tasks_completed_count)::NUMERIC,
        'tasks'::TEXT
    FROM work_sessions 
    WHERE repository = repository_param 
    AND start_time >= cutoff_date 
    AND deleted_at IS NULL
    
    UNION ALL
    
    SELECT 
        'avg_productivity_score'::TEXT,
        AVG(productivity_score)::NUMERIC,
        'score'::TEXT
    FROM work_sessions 
    WHERE repository = repository_param 
    AND start_time >= cutoff_date 
    AND productivity_score IS NOT NULL
    AND deleted_at IS NULL
    
    UNION ALL
    
    SELECT 
        'total_development_hours'::TEXT,
        ROUND((SUM(duration_minutes) / 60.0)::NUMERIC, 2),
        'hours'::TEXT
    FROM work_sessions 
    WHERE repository = repository_param 
    AND start_time >= cutoff_date 
    AND duration_minutes IS NOT NULL
    AND deleted_at IS NULL;
END;
$$ LANGUAGE plpgsql;

-- Function to start a new work session
CREATE OR REPLACE FUNCTION start_work_session(
    session_id_param VARCHAR(255),
    repository_param VARCHAR(255),
    user_id_param VARCHAR(255),
    session_type_param VARCHAR(50) DEFAULT 'development',
    branch_param VARCHAR(255) DEFAULT NULL,
    session_goals_param JSONB DEFAULT '[]'
)
RETURNS UUID AS $$
DECLARE
    new_session_uuid UUID;
BEGIN
    INSERT INTO work_sessions (
        session_id,
        repository,
        branch,
        user_id,
        session_type,
        start_time,
        session_goals,
        status
    ) VALUES (
        session_id_param,
        repository_param,
        branch_param,
        user_id_param,
        session_type_param,
        CURRENT_TIMESTAMP,
        session_goals_param,
        'active'
    ) RETURNING id INTO new_session_uuid;
    
    RETURN new_session_uuid;
END;
$$ LANGUAGE plpgsql;

-- Function to end a work session
CREATE OR REPLACE FUNCTION end_work_session(
    session_id_param VARCHAR(255),
    completion_reason_param VARCHAR(100) DEFAULT 'normal_completion',
    session_summary_param TEXT DEFAULT NULL
)
RETURNS BOOLEAN AS $$
DECLARE
    session_found BOOLEAN := false;
BEGIN
    UPDATE work_sessions 
    SET 
        end_time = CURRENT_TIMESTAMP,
        status = 'completed',
        completion_reason = completion_reason_param,
        session_summary = COALESCE(session_summary_param, session_summary)
    WHERE session_id = session_id_param 
    AND status = 'active'
    AND deleted_at IS NULL;
    
    GET DIAGNOSTICS session_found = FOUND;
    
    RETURN session_found;
END;
$$ LANGUAGE plpgsql;

-- Function to clean up old sessions
CREATE OR REPLACE FUNCTION cleanup_old_sessions(days_old INTEGER DEFAULT 90)
RETURNS INTEGER AS $$
DECLARE
    deleted_count INTEGER;
    cutoff_date TIMESTAMP WITH TIME ZONE;
BEGIN
    cutoff_date := CURRENT_TIMESTAMP - (days_old || ' days')::INTERVAL;
    
    UPDATE work_sessions 
    SET deleted_at = CURRENT_TIMESTAMP
    WHERE start_time < cutoff_date
    AND status IN ('abandoned', 'interrupted')
    AND deleted_at IS NULL;
    
    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

-- Comments for documentation
COMMENT ON TABLE work_sessions IS 'Tracks work sessions for productivity analytics and development insights';
COMMENT ON COLUMN work_sessions.duration_minutes IS 'Automatically calculated duration when session ends';
COMMENT ON COLUMN work_sessions.productivity_score IS 'Calculated productivity score based on tasks completed and time spent';
COMMENT ON COLUMN work_sessions.search_vector IS 'Generated tsvector for full-text search of session content';
COMMENT ON FUNCTION get_user_productivity_insights(VARCHAR, INTEGER) IS 'Returns productivity metrics and trends for a user';
COMMENT ON FUNCTION get_repository_activity_summary(VARCHAR, INTEGER) IS 'Provides activity summary for a repository over time period';
COMMENT ON FUNCTION start_work_session(VARCHAR, VARCHAR, VARCHAR, VARCHAR, VARCHAR, JSONB) IS 'Creates and starts a new work session';
COMMENT ON FUNCTION end_work_session(VARCHAR, VARCHAR, TEXT) IS 'Ends an active work session with completion details';
COMMENT ON FUNCTION cleanup_old_sessions(INTEGER) IS 'Soft deletes old abandoned or interrupted sessions';