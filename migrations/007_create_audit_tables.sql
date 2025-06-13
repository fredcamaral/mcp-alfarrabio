-- Migration 007: Create Audit Tables
-- Description: Creates audit logging tables for tracking user actions and system changes
-- Created: 2025-06-12
-- Version: 1.0.0

-- Enable UUID extension if not already enabled
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Create audit action enum
CREATE TYPE audit_action AS ENUM (
    'create', 'read', 'update', 'delete', 'batch', 'search', 
    'transition', 'assign', 'comment'
);

-- Create task audit log table
CREATE TABLE task_audit_log (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    
    -- Event identification
    timestamp TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    user_id VARCHAR(255) NOT NULL,
    action audit_action NOT NULL,
    
    -- Resource information
    resource_type VARCHAR(50) NOT NULL DEFAULT 'task',
    resource_id VARCHAR(255) NOT NULL,
    resource_name VARCHAR(255),
    
    -- Request context
    ip_address INET,
    user_agent TEXT,
    session_id VARCHAR(255),
    
    -- Operation result
    success BOOLEAN NOT NULL DEFAULT true,
    error_message TEXT,
    
    -- Operation details
    details JSONB DEFAULT '{}',
    
    -- Performance tracking
    execution_time_ms INTEGER,
    
    -- Metadata
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL
);

-- Create indexes for performance
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_task_audit_log_timestamp ON task_audit_log(timestamp DESC);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_task_audit_log_user_id ON task_audit_log(user_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_task_audit_log_action ON task_audit_log(action);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_task_audit_log_resource ON task_audit_log(resource_type, resource_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_task_audit_log_success ON task_audit_log(success) WHERE success = false;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_task_audit_log_details ON task_audit_log USING GIN(details);

-- Composite indexes for common query patterns
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_task_audit_log_user_timestamp ON task_audit_log(user_id, timestamp DESC);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_task_audit_log_resource_timestamp ON task_audit_log(resource_type, resource_id, timestamp DESC);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_task_audit_log_action_timestamp ON task_audit_log(action, timestamp DESC);

-- Create partitioning function for large datasets (optional - can be enabled later)
-- This partitions by month to improve query performance on large audit logs
CREATE OR REPLACE FUNCTION create_audit_partition(start_date DATE, end_date DATE)
RETURNS VOID AS $$
DECLARE
    partition_name TEXT;
    start_str TEXT;
    end_str TEXT;
BEGIN
    start_str := to_char(start_date, 'YYYY_MM');
    end_str := to_char(end_date, 'YYYY_MM_DD');
    partition_name := 'task_audit_log_' || start_str;
    
    EXECUTE format('CREATE TABLE IF NOT EXISTS %I PARTITION OF task_audit_log FOR VALUES FROM (%L) TO (%L)',
                   partition_name, start_date, end_date);
                   
    -- Create indexes on partition
    EXECUTE format('CREATE INDEX CONCURRENTLY IF NOT EXISTS %I ON %I(timestamp DESC)',
                   partition_name || '_timestamp_idx', partition_name);
    EXECUTE format('CREATE INDEX CONCURRENTLY IF NOT EXISTS %I ON %I(user_id)',
                   partition_name || '_user_idx', partition_name);
END;
$$ LANGUAGE plpgsql;

-- Create audit summary view for reporting
CREATE OR REPLACE VIEW audit_summary AS
SELECT 
    date_trunc('day', timestamp) as audit_date,
    user_id,
    action,
    resource_type,
    COUNT(*) as operation_count,
    COUNT(CASE WHEN success THEN 1 END) as successful_operations,
    COUNT(CASE WHEN NOT success THEN 1 END) as failed_operations,
    ROUND(AVG(execution_time_ms), 2) as avg_execution_time_ms,
    COUNT(DISTINCT resource_id) as unique_resources_affected
FROM task_audit_log
WHERE timestamp >= CURRENT_DATE - INTERVAL '30 days'
GROUP BY date_trunc('day', timestamp), user_id, action, resource_type
ORDER BY audit_date DESC, operation_count DESC;

-- Create function to clean up old audit entries
CREATE OR REPLACE FUNCTION cleanup_old_audit_entries(retention_days INTEGER DEFAULT 90)
RETURNS INTEGER AS $$
DECLARE
    deleted_count INTEGER;
    cutoff_date TIMESTAMP WITH TIME ZONE;
BEGIN
    cutoff_date := CURRENT_TIMESTAMP - (retention_days || ' days')::INTERVAL;
    
    DELETE FROM task_audit_log 
    WHERE timestamp < cutoff_date;
    
    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    
    -- Log the cleanup operation
    INSERT INTO task_audit_log (
        user_id, action, resource_type, resource_id, 
        details, success
    ) VALUES (
        'system', 'delete', 'audit_log', 'cleanup',
        jsonb_build_object(
            'operation', 'audit_cleanup',
            'retention_days', retention_days,
            'deleted_count', deleted_count,
            'cutoff_date', cutoff_date
        ),
        true
    );
    
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

-- Create trigger function for automatic audit logging
CREATE OR REPLACE FUNCTION log_task_audit()
RETURNS TRIGGER AS $$
BEGIN
    -- Only log if the audit trigger is enabled
    IF current_setting('app.audit_enabled', true) = 'true' THEN
        INSERT INTO task_audit_log (
            user_id, action, resource_type, resource_id,
            details, success
        ) VALUES (
            COALESCE(current_setting('app.current_user_id', true), 'system'),
            CASE 
                WHEN TG_OP = 'INSERT' THEN 'create'::audit_action
                WHEN TG_OP = 'UPDATE' THEN 'update'::audit_action  
                WHEN TG_OP = 'DELETE' THEN 'delete'::audit_action
            END,
            'task',
            CASE 
                WHEN TG_OP = 'DELETE' THEN OLD.id::TEXT
                ELSE NEW.id::TEXT
            END,
            CASE 
                WHEN TG_OP = 'INSERT' THEN jsonb_build_object('operation', 'task_created', 'title', NEW.title)
                WHEN TG_OP = 'UPDATE' THEN jsonb_build_object('operation', 'task_updated', 'changes', 
                    jsonb_build_object(
                        'title', CASE WHEN OLD.title != NEW.title THEN jsonb_build_object('old', OLD.title, 'new', NEW.title) END,
                        'status', CASE WHEN OLD.status != NEW.status THEN jsonb_build_object('old', OLD.status, 'new', NEW.status) END,
                        'assignee', CASE WHEN OLD.assignee != NEW.assignee THEN jsonb_build_object('old', OLD.assignee, 'new', NEW.assignee) END
                    )
                )
                WHEN TG_OP = 'DELETE' THEN jsonb_build_object('operation', 'task_deleted', 'title', OLD.title)
            END,
            true
        );
    END IF;
    
    RETURN CASE WHEN TG_OP = 'DELETE' THEN OLD ELSE NEW END;
END;
$$ LANGUAGE plpgsql;

-- Create audit trigger on tasks table (optional - can be enabled/disabled)
-- This is disabled by default to avoid conflicts with application-level audit logging
-- To enable: SELECT set_config('app.audit_enabled', 'true', false);
CREATE TRIGGER task_audit_trigger
    AFTER INSERT OR UPDATE OR DELETE ON tasks
    FOR EACH ROW 
    EXECUTE FUNCTION log_task_audit();

-- Disable the trigger by default (application handles audit logging)
ALTER TABLE tasks DISABLE TRIGGER task_audit_trigger;

-- Create function to get audit statistics
CREATE OR REPLACE FUNCTION get_audit_statistics(
    start_date TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_DATE - INTERVAL '7 days',
    end_date TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
)
RETURNS TABLE (
    total_operations BIGINT,
    successful_operations BIGINT,
    failed_operations BIGINT,
    success_rate NUMERIC(5,2),
    unique_users BIGINT,
    unique_resources BIGINT,
    most_active_user TEXT,
    most_common_action TEXT,
    avg_execution_time_ms NUMERIC(10,2)
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        COUNT(*) as total_operations,
        COUNT(CASE WHEN tal.success THEN 1 END) as successful_operations,
        COUNT(CASE WHEN NOT tal.success THEN 1 END) as failed_operations,
        ROUND(
            COUNT(CASE WHEN tal.success THEN 1 END) * 100.0 / NULLIF(COUNT(*), 0), 
            2
        ) as success_rate,
        COUNT(DISTINCT tal.user_id) as unique_users,
        COUNT(DISTINCT tal.resource_id) as unique_resources,
        (
            SELECT tal2.user_id 
            FROM task_audit_log tal2 
            WHERE tal2.timestamp BETWEEN start_date AND end_date
            GROUP BY tal2.user_id 
            ORDER BY COUNT(*) DESC 
            LIMIT 1
        ) as most_active_user,
        (
            SELECT tal3.action::TEXT
            FROM task_audit_log tal3 
            WHERE tal3.timestamp BETWEEN start_date AND end_date
            GROUP BY tal3.action 
            ORDER BY COUNT(*) DESC 
            LIMIT 1
        ) as most_common_action,
        ROUND(AVG(tal.execution_time_ms), 2) as avg_execution_time_ms
    FROM task_audit_log tal
    WHERE tal.timestamp BETWEEN start_date AND end_date;
END;
$$ LANGUAGE plpgsql;

-- Add comments for documentation
COMMENT ON TABLE task_audit_log IS 'Audit trail for all task-related operations and system changes';
COMMENT ON COLUMN task_audit_log.details IS 'JSON object containing operation-specific details and metadata';
COMMENT ON COLUMN task_audit_log.execution_time_ms IS 'Time taken to execute the operation in milliseconds';
COMMENT ON FUNCTION cleanup_old_audit_entries(INTEGER) IS 'Removes audit entries older than specified retention period';
COMMENT ON FUNCTION get_audit_statistics(TIMESTAMP WITH TIME ZONE, TIMESTAMP WITH TIME ZONE) IS 'Returns comprehensive audit statistics for specified time period';
COMMENT ON VIEW audit_summary IS 'Daily summary of audit operations by user and action type';

-- Set default audit settings
SELECT set_config('app.audit_enabled', 'false', false); -- Disabled by default, app handles it
SELECT set_config('app.current_user_id', 'system', false);