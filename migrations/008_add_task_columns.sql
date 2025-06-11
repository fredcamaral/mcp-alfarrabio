-- Migration: Add missing columns to enhanced_tasks table
-- Purpose: Enhance task tracking with session context, complexity, dependencies, and time tracking

-- Add session_id column for multi-session task continuity
ALTER TABLE enhanced_tasks 
ADD COLUMN session_id TEXT;

COMMENT ON COLUMN enhanced_tasks.session_id IS 'Session identifier for tracking tasks across LLM sessions - enables cross-session task continuity';

-- Add complexity column with constraint
ALTER TABLE enhanced_tasks 
ADD COLUMN complexity TEXT CHECK (complexity IN ('simple', 'medium', 'complex', 'very_complex'));

COMMENT ON COLUMN enhanced_tasks.complexity IS 'Task complexity level affecting estimation and prioritization';

-- Add dependencies column for task relationships
ALTER TABLE enhanced_tasks 
ADD COLUMN dependencies JSONB DEFAULT '{}';

COMMENT ON COLUMN enhanced_tasks.dependencies IS 'JSON structure storing task dependencies and relationships';

-- Add time tracking columns
ALTER TABLE enhanced_tasks 
ADD COLUMN estimated_hours NUMERIC(10,2);

COMMENT ON COLUMN enhanced_tasks.estimated_hours IS 'Estimated hours for task completion';

ALTER TABLE enhanced_tasks 
ADD COLUMN actual_hours NUMERIC(10,2);

COMMENT ON COLUMN enhanced_tasks.actual_hours IS 'Actual hours spent on task completion';

-- Add completion percentage
ALTER TABLE enhanced_tasks 
ADD COLUMN completion_percentage INTEGER DEFAULT 0 CHECK (completion_percentage >= 0 AND completion_percentage <= 100);

COMMENT ON COLUMN enhanced_tasks.completion_percentage IS 'Task completion progress as percentage (0-100)';

-- Add blocking relationship columns
ALTER TABLE enhanced_tasks 
ADD COLUMN blocked_by TEXT[] DEFAULT '{}';

COMMENT ON COLUMN enhanced_tasks.blocked_by IS 'Array of task IDs that block this task';

ALTER TABLE enhanced_tasks 
ADD COLUMN blocking TEXT[] DEFAULT '{}';

COMMENT ON COLUMN enhanced_tasks.blocking IS 'Array of task IDs that this task blocks';

-- Create indexes for performance

-- Index for session-based queries (cross-session task retrieval)
CREATE INDEX IF NOT EXISTS idx_enhanced_tasks_session_id ON enhanced_tasks(session_id);

-- Index for complexity-based filtering and analytics
CREATE INDEX IF NOT EXISTS idx_enhanced_tasks_complexity ON enhanced_tasks(complexity);

-- Index for completion tracking and progress monitoring
CREATE INDEX IF NOT EXISTS idx_enhanced_tasks_completion ON enhanced_tasks(completion_percentage);

-- Composite index for session and status queries (common query pattern)
CREATE INDEX IF NOT EXISTS idx_enhanced_tasks_session_status ON enhanced_tasks(session_id, status);

-- Composite index for time tracking queries
CREATE INDEX IF NOT EXISTS idx_enhanced_tasks_time_tracking ON enhanced_tasks(estimated_hours, actual_hours) 
WHERE estimated_hours IS NOT NULL OR actual_hours IS NOT NULL;

-- GIN index for JSONB dependencies column for efficient querying
CREATE INDEX IF NOT EXISTS idx_enhanced_tasks_dependencies ON enhanced_tasks USING GIN (dependencies);

-- GIN indexes for array columns (blocking relationships)
CREATE INDEX IF NOT EXISTS idx_enhanced_tasks_blocked_by ON enhanced_tasks USING GIN (blocked_by);
CREATE INDEX IF NOT EXISTS idx_enhanced_tasks_blocking ON enhanced_tasks USING GIN (blocking);

-- Add triggers for data integrity

-- Function to update blocking relationships bidirectionally
CREATE OR REPLACE FUNCTION update_blocking_relationships()
RETURNS TRIGGER AS $$
BEGIN
    -- When blocked_by is updated, update the blocking array in referenced tasks
    IF TG_OP = 'UPDATE' AND OLD.blocked_by IS DISTINCT FROM NEW.blocked_by THEN
        -- Remove this task from old blockers
        UPDATE enhanced_tasks 
        SET blocking = array_remove(blocking, NEW.id::TEXT)
        WHERE id::TEXT = ANY(OLD.blocked_by) AND id != NEW.id;
        
        -- Add this task to new blockers
        UPDATE enhanced_tasks 
        SET blocking = array_append(blocking, NEW.id::TEXT)
        WHERE id::TEXT = ANY(NEW.blocked_by) AND id != NEW.id
        AND NOT (NEW.id::TEXT = ANY(blocking));
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create trigger for blocking relationship consistency
CREATE TRIGGER trigger_update_blocking_relationships
AFTER UPDATE OF blocked_by ON enhanced_tasks
FOR EACH ROW
EXECUTE FUNCTION update_blocking_relationships();

-- Function to validate completion percentage based on status
CREATE OR REPLACE FUNCTION validate_completion_percentage()
RETURNS TRIGGER AS $$
BEGIN
    -- Ensure completed tasks have 100% completion
    IF NEW.status = 'completed' AND NEW.completion_percentage != 100 THEN
        NEW.completion_percentage = 100;
    END IF;
    
    -- Ensure cancelled tasks maintain their completion percentage
    -- Ensure pending tasks start at 0%
    IF NEW.status = 'pending' AND OLD.status IS DISTINCT FROM 'pending' THEN
        NEW.completion_percentage = 0;
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create trigger for completion percentage validation
CREATE TRIGGER trigger_validate_completion_percentage
BEFORE INSERT OR UPDATE ON enhanced_tasks
FOR EACH ROW
EXECUTE FUNCTION validate_completion_percentage();

-- Add helpful comments on table usage
COMMENT ON TABLE enhanced_tasks IS 'Enhanced task tracking with session continuity, complexity analysis, dependency management, and time tracking for AI-assisted development workflows';