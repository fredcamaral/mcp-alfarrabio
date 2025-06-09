-- Migration 003: Create Task Audit Triggers
-- Description: Creates comprehensive audit trail triggers for all task-related operations
-- Created: 2025-06-09
-- Version: 1.0.0

-- Create audit log table for task changes
CREATE TABLE task_audit_log (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    task_id UUID NOT NULL,
    operation VARCHAR(10) NOT NULL CHECK (operation IN ('INSERT', 'UPDATE', 'DELETE')),
    old_values JSONB,
    new_values JSONB,
    changed_fields TEXT[],
    user_id VARCHAR(255),
    session_id VARCHAR(255),
    ip_address INET,
    user_agent TEXT,
    timestamp TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    transaction_id BIGINT DEFAULT txid_current(),
    
    -- Additional context fields
    repository VARCHAR(255),
    branch VARCHAR(255),
    operation_context VARCHAR(100), -- 'api', 'cli', 'ai_generation', 'migration', etc.
    source_system VARCHAR(50),
    
    -- Performance optimization
    created_date DATE GENERATED ALWAYS AS (DATE(timestamp)) STORED
);

-- Index for audit log queries
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_task_audit_log_task_id ON task_audit_log(task_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_task_audit_log_timestamp ON task_audit_log(timestamp DESC);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_task_audit_log_operation ON task_audit_log(operation);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_task_audit_log_user_id ON task_audit_log(user_id) WHERE user_id IS NOT NULL;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_task_audit_log_date ON task_audit_log(created_date);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_task_audit_log_repository ON task_audit_log(repository) WHERE repository IS NOT NULL;

-- Composite index for common audit queries
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_task_audit_log_task_time ON task_audit_log(task_id, timestamp DESC);

-- Create audit trigger function
CREATE OR REPLACE FUNCTION audit_task_changes()
RETURNS TRIGGER AS $$
DECLARE
    old_values_json JSONB := NULL;
    new_values_json JSONB := NULL;
    changed_fields_array TEXT[] := ARRAY[]::TEXT[];
    current_user_id VARCHAR(255);
    current_session_id VARCHAR(255);
    current_ip_address INET;
    current_user_agent TEXT;
    current_operation_context VARCHAR(100);
    current_source_system VARCHAR(50);
BEGIN
    -- Get current session context (these would be set by the application)
    current_user_id := COALESCE(current_setting('app.current_user_id', true), 'system');
    current_session_id := COALESCE(current_setting('app.current_session_id', true), 'unknown');
    current_ip_address := COALESCE(current_setting('app.current_ip_address', true)::INET, '127.0.0.1'::INET);
    current_user_agent := current_setting('app.current_user_agent', true);
    current_operation_context := COALESCE(current_setting('app.operation_context', true), 'unknown');
    current_source_system := COALESCE(current_setting('app.source_system', true), 'mcp-memory');

    -- Handle different operation types
    IF TG_OP = 'DELETE' THEN
        old_values_json := to_jsonb(OLD);
        
        INSERT INTO task_audit_log (
            task_id, operation, old_values, new_values, changed_fields,
            user_id, session_id, ip_address, user_agent, operation_context, source_system,
            repository, branch
        ) VALUES (
            OLD.id, TG_OP, old_values_json, NULL, ARRAY['*'],
            current_user_id, current_session_id, current_ip_address, current_user_agent,
            current_operation_context, current_source_system,
            OLD.repository, OLD.branch
        );
        
        RETURN OLD;
        
    ELSIF TG_OP = 'INSERT' THEN
        new_values_json := to_jsonb(NEW);
        
        INSERT INTO task_audit_log (
            task_id, operation, old_values, new_values, changed_fields,
            user_id, session_id, ip_address, user_agent, operation_context, source_system,
            repository, branch
        ) VALUES (
            NEW.id, TG_OP, NULL, new_values_json, ARRAY['*'],
            current_user_id, current_session_id, current_ip_address, current_user_agent,
            current_operation_context, current_source_system,
            NEW.repository, NEW.branch
        );
        
        RETURN NEW;
        
    ELSIF TG_OP = 'UPDATE' THEN
        old_values_json := to_jsonb(OLD);
        new_values_json := to_jsonb(NEW);
        
        -- Identify changed fields
        IF OLD.title IS DISTINCT FROM NEW.title THEN
            changed_fields_array := array_append(changed_fields_array, 'title');
        END IF;
        IF OLD.description IS DISTINCT FROM NEW.description THEN
            changed_fields_array := array_append(changed_fields_array, 'description');
        END IF;
        IF OLD.content IS DISTINCT FROM NEW.content THEN
            changed_fields_array := array_append(changed_fields_array, 'content');
        END IF;
        IF OLD.type IS DISTINCT FROM NEW.type THEN
            changed_fields_array := array_append(changed_fields_array, 'type');
        END IF;
        IF OLD.status IS DISTINCT FROM NEW.status THEN
            changed_fields_array := array_append(changed_fields_array, 'status');
        END IF;
        IF OLD.priority IS DISTINCT FROM NEW.priority THEN
            changed_fields_array := array_append(changed_fields_array, 'priority');
        END IF;
        IF OLD.complexity IS DISTINCT FROM NEW.complexity THEN
            changed_fields_array := array_append(changed_fields_array, 'complexity');
        END IF;
        IF OLD.assignee IS DISTINCT FROM NEW.assignee THEN
            changed_fields_array := array_append(changed_fields_array, 'assignee');
        END IF;
        IF OLD.started_at IS DISTINCT FROM NEW.started_at THEN
            changed_fields_array := array_append(changed_fields_array, 'started_at');
        END IF;
        IF OLD.completed_at IS DISTINCT FROM NEW.completed_at THEN
            changed_fields_array := array_append(changed_fields_array, 'completed_at');
        END IF;
        IF OLD.due_date IS DISTINCT FROM NEW.due_date THEN
            changed_fields_array := array_append(changed_fields_array, 'due_date');
        END IF;
        IF OLD.estimated_hours IS DISTINCT FROM NEW.estimated_hours THEN
            changed_fields_array := array_append(changed_fields_array, 'estimated_hours');
        END IF;
        IF OLD.quality_score IS DISTINCT FROM NEW.quality_score THEN
            changed_fields_array := array_append(changed_fields_array, 'quality_score');
        END IF;
        IF OLD.complexity_score IS DISTINCT FROM NEW.complexity_score THEN
            changed_fields_array := array_append(changed_fields_array, 'complexity_score');
        END IF;
        IF OLD.parent_task_id IS DISTINCT FROM NEW.parent_task_id THEN
            changed_fields_array := array_append(changed_fields_array, 'parent_task_id');
        END IF;
        IF OLD.tags IS DISTINCT FROM NEW.tags THEN
            changed_fields_array := array_append(changed_fields_array, 'tags');
        END IF;
        IF OLD.dependencies IS DISTINCT FROM NEW.dependencies THEN
            changed_fields_array := array_append(changed_fields_array, 'dependencies');
        END IF;
        IF OLD.blocks IS DISTINCT FROM NEW.blocks THEN
            changed_fields_array := array_append(changed_fields_array, 'blocks');
        END IF;
        IF OLD.acceptance_criteria IS DISTINCT FROM NEW.acceptance_criteria THEN
            changed_fields_array := array_append(changed_fields_array, 'acceptance_criteria');
        END IF;
        IF OLD.metadata IS DISTINCT FROM NEW.metadata THEN
            changed_fields_array := array_append(changed_fields_array, 'metadata');
        END IF;
        IF OLD.deleted_at IS DISTINCT FROM NEW.deleted_at THEN
            changed_fields_array := array_append(changed_fields_array, 'deleted_at');
        END IF;
        
        -- Only log if there are actual changes
        IF array_length(changed_fields_array, 1) > 0 THEN
            INSERT INTO task_audit_log (
                task_id, operation, old_values, new_values, changed_fields,
                user_id, session_id, ip_address, user_agent, operation_context, source_system,
                repository, branch
            ) VALUES (
                NEW.id, TG_OP, old_values_json, new_values_json, changed_fields_array,
                current_user_id, current_session_id, current_ip_address, current_user_agent,
                current_operation_context, current_source_system,
                NEW.repository, NEW.branch
            );
        END IF;
        
        RETURN NEW;
    END IF;
    
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

-- Create the main audit trigger
CREATE TRIGGER task_audit_trigger
    AFTER INSERT OR UPDATE OR DELETE ON tasks
    FOR EACH ROW EXECUTE FUNCTION audit_task_changes();

-- Create task status change notification trigger
CREATE OR REPLACE FUNCTION notify_task_status_change()
RETURNS TRIGGER AS $$
BEGIN
    -- Only notify on status changes
    IF TG_OP = 'UPDATE' AND OLD.status IS DISTINCT FROM NEW.status THEN
        -- Send notification (this would integrate with your notification system)
        PERFORM pg_notify(
            'task_status_change',
            json_build_object(
                'task_id', NEW.id,
                'old_status', OLD.status,
                'new_status', NEW.status,
                'assignee', NEW.assignee,
                'repository', NEW.repository,
                'title', NEW.title,
                'timestamp', NEW.updated_at
            )::text
        );
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER task_status_change_notification
    AFTER UPDATE ON tasks
    FOR EACH ROW EXECUTE FUNCTION notify_task_status_change();

-- Create task completion trigger for metrics
CREATE OR REPLACE FUNCTION handle_task_completion()
RETURNS TRIGGER AS $$
DECLARE
    total_time_minutes INTEGER;
BEGIN
    -- Handle completion tracking
    IF TG_OP = 'UPDATE' AND OLD.status != 'completed' AND NEW.status = 'completed' THEN
        -- Set completed_at if not already set
        IF NEW.completed_at IS NULL THEN
            NEW.completed_at := CURRENT_TIMESTAMP;
        END IF;
        
        -- Calculate actual time if started_at is available
        IF NEW.started_at IS NOT NULL THEN
            total_time_minutes := EXTRACT(EPOCH FROM (NEW.completed_at - NEW.started_at)) / 60;
            NEW.actual_minutes := total_time_minutes;
        END IF;
        
        -- Increment task counter in PRD if linked
        IF NEW.source_prd_id IS NOT NULL THEN
            UPDATE prds 
            SET task_count = task_count + 1 
            WHERE id = NEW.source_prd_id;
        END IF;
        
        -- Update pattern success metrics if pattern is linked
        IF NEW.pattern_id IS NOT NULL THEN
            UPDATE task_patterns 
            SET usage_count = usage_count + 1,
                success_rate = (
                    SELECT (COUNT(*) FILTER (WHERE status = 'completed')::DECIMAL / COUNT(*))
                    FROM tasks 
                    WHERE pattern_id = NEW.pattern_id 
                    AND deleted_at IS NULL
                )
            WHERE id = NEW.pattern_id;
        END IF;
        
        -- Update template success metrics if template is used
        IF NEW.template_id IS NOT NULL THEN
            UPDATE task_templates 
            SET usage_count = usage_count + 1,
                success_rate = (
                    SELECT (COUNT(*) FILTER (WHERE status = 'completed')::DECIMAL / COUNT(*))
                    FROM tasks 
                    WHERE template_id = NEW.template_id 
                    AND deleted_at IS NULL
                )
            WHERE id = NEW.template_id;
        END IF;
    END IF;
    
    -- Handle task start tracking
    IF TG_OP = 'UPDATE' AND OLD.status != 'in_progress' AND NEW.status = 'in_progress' AND NEW.started_at IS NULL THEN
        NEW.started_at := CURRENT_TIMESTAMP;
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER task_completion_handler
    BEFORE UPDATE ON tasks
    FOR EACH ROW EXECUTE FUNCTION handle_task_completion();

-- Create soft delete trigger
CREATE OR REPLACE FUNCTION handle_task_soft_delete()
RETURNS TRIGGER AS $$
BEGIN
    -- If deleted_at is being set (soft delete), update child tasks
    IF TG_OP = 'UPDATE' AND OLD.deleted_at IS NULL AND NEW.deleted_at IS NOT NULL THEN
        -- Handle child tasks - either cascade delete or reassign
        UPDATE tasks 
        SET parent_task_id = NULL,
            updated_at = CURRENT_TIMESTAMP
        WHERE parent_task_id = NEW.id 
        AND deleted_at IS NULL;
        
        -- Set deleted_by if not already set
        IF NEW.deleted_by IS NULL THEN
            NEW.deleted_by := COALESCE(current_setting('app.current_user_id', true), 'system');
        END IF;
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER task_soft_delete_handler
    BEFORE UPDATE ON tasks
    FOR EACH ROW EXECUTE FUNCTION handle_task_soft_delete();

-- Create version increment trigger
CREATE OR REPLACE FUNCTION increment_task_version()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'UPDATE' THEN
        NEW.version := OLD.version + 1;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER task_version_increment
    BEFORE UPDATE ON tasks
    FOR EACH ROW EXECUTE FUNCTION increment_task_version();

-- Create supporting table audit triggers

-- PRD audit trigger
CREATE OR REPLACE FUNCTION audit_prd_changes()
RETURNS TRIGGER AS $$
DECLARE
    current_user_id VARCHAR(255);
BEGIN
    current_user_id := COALESCE(current_setting('app.current_user_id', true), 'system');
    
    IF TG_OP = 'UPDATE' AND OLD.task_count IS DISTINCT FROM NEW.task_count THEN
        -- Log significant PRD changes
        INSERT INTO task_audit_log (
            task_id, operation, old_values, new_values, changed_fields,
            user_id, operation_context, source_system, repository
        ) VALUES (
            NEW.id, 'PRD_UPDATE', 
            json_build_object('task_count', OLD.task_count),
            json_build_object('task_count', NEW.task_count),
            ARRAY['task_count'],
            current_user_id, 'prd_update', 'mcp-memory', NEW.repository
        );
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER prd_audit_trigger
    AFTER UPDATE ON prds
    FOR EACH ROW EXECUTE FUNCTION audit_prd_changes();

-- Create audit log cleanup function (for maintenance)
CREATE OR REPLACE FUNCTION cleanup_old_audit_logs(retention_days INTEGER DEFAULT 90)
RETURNS INTEGER AS $$
DECLARE
    deleted_count INTEGER;
BEGIN
    DELETE FROM task_audit_log 
    WHERE timestamp < CURRENT_TIMESTAMP - (retention_days || ' days')::INTERVAL;
    
    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

-- Create function to get task change history
CREATE OR REPLACE FUNCTION get_task_change_history(task_uuid UUID, limit_count INTEGER DEFAULT 50)
RETURNS TABLE (
    timestamp TIMESTAMP WITH TIME ZONE,
    operation VARCHAR(10),
    changed_fields TEXT[],
    user_id VARCHAR(255),
    old_values JSONB,
    new_values JSONB,
    operation_context VARCHAR(100)
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        tal.timestamp,
        tal.operation,
        tal.changed_fields,
        tal.user_id,
        tal.old_values,
        tal.new_values,
        tal.operation_context
    FROM task_audit_log tal
    WHERE tal.task_id = task_uuid
    ORDER BY tal.timestamp DESC
    LIMIT limit_count;
END;
$$ LANGUAGE plpgsql;

-- Create function to get task activity summary
CREATE OR REPLACE FUNCTION get_task_activity_summary(task_uuid UUID)
RETURNS JSON AS $$
DECLARE
    result JSON;
BEGIN
    SELECT json_build_object(
        'total_changes', COUNT(*),
        'first_change', MIN(timestamp),
        'last_change', MAX(timestamp),
        'unique_users', COUNT(DISTINCT user_id),
        'operations', json_object_agg(operation, op_count)
    ) INTO result
    FROM (
        SELECT 
            operation,
            timestamp,
            user_id,
            COUNT(*) as op_count
        FROM task_audit_log 
        WHERE task_id = task_uuid
        GROUP BY operation, timestamp, user_id
    ) activity
    GROUP BY ();
    
    RETURN COALESCE(result, '{}'::JSON);
END;
$$ LANGUAGE plpgsql;

-- Comments for documentation
COMMENT ON TABLE task_audit_log IS 'Comprehensive audit trail for all task-related operations';
COMMENT ON FUNCTION audit_task_changes() IS 'Main audit trigger function that captures all task changes';
COMMENT ON FUNCTION notify_task_status_change() IS 'Sends notifications when task status changes';
COMMENT ON FUNCTION handle_task_completion() IS 'Handles task completion logic and metrics updates';
COMMENT ON FUNCTION handle_task_soft_delete() IS 'Manages soft delete operations and child task handling';
COMMENT ON FUNCTION cleanup_old_audit_logs(INTEGER) IS 'Maintenance function to clean up old audit logs';
COMMENT ON FUNCTION get_task_change_history(UUID, INTEGER) IS 'Retrieves change history for a specific task';
COMMENT ON FUNCTION get_task_activity_summary(UUID) IS 'Provides activity summary for a specific task';