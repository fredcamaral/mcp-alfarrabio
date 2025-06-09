-- Migration 004: Create Table Partitioning Strategy
-- Description: Implements table partitioning for scalability on large datasets
-- Created: 2025-06-09
-- Version: 1.0.0

-- Note: This migration is optional and should be applied when dealing with large datasets
-- PostgreSQL 10+ partitioning by range (time-based) and list (repository-based)

-- Create partitioned tasks table (if starting fresh)
-- For existing installations, this would require data migration

-- First, let's create a partitioning strategy for audit logs (more likely to grow large)
-- Convert task_audit_log to partitioned table

-- Step 1: Create partitioned version of audit log table
CREATE TABLE task_audit_log_partitioned (
    LIKE task_audit_log INCLUDING ALL
) PARTITION BY RANGE (timestamp);

-- Step 2: Create monthly partitions for audit logs (example for 2025)
CREATE TABLE task_audit_log_2025_01 PARTITION OF task_audit_log_partitioned
    FOR VALUES FROM ('2025-01-01') TO ('2025-02-01');

CREATE TABLE task_audit_log_2025_02 PARTITION OF task_audit_log_partitioned
    FOR VALUES FROM ('2025-02-01') TO ('2025-03-01');

CREATE TABLE task_audit_log_2025_03 PARTITION OF task_audit_log_partitioned
    FOR VALUES FROM ('2025-03-01') TO ('2025-04-01');

CREATE TABLE task_audit_log_2025_04 PARTITION OF task_audit_log_partitioned
    FOR VALUES FROM ('2025-04-01') TO ('2025-05-01');

CREATE TABLE task_audit_log_2025_05 PARTITION OF task_audit_log_partitioned
    FOR VALUES FROM ('2025-05-01') TO ('2025-06-01');

CREATE TABLE task_audit_log_2025_06 PARTITION OF task_audit_log_partitioned
    FOR VALUES FROM ('2025-06-01') TO ('2025-07-01');

CREATE TABLE task_audit_log_2025_07 PARTITION OF task_audit_log_partitioned
    FOR VALUES FROM ('2025-07-01') TO ('2025-08-01');

CREATE TABLE task_audit_log_2025_08 PARTITION OF task_audit_log_partitioned
    FOR VALUES FROM ('2025-08-01') TO ('2025-09-01');

CREATE TABLE task_audit_log_2025_09 PARTITION OF task_audit_log_partitioned
    FOR VALUES FROM ('2025-09-01') TO ('2025-10-01');

CREATE TABLE task_audit_log_2025_10 PARTITION OF task_audit_log_partitioned
    FOR VALUES FROM ('2025-10-01') TO ('2025-11-01');

CREATE TABLE task_audit_log_2025_11 PARTITION OF task_audit_log_partitioned
    FOR VALUES FROM ('2025-11-01') TO ('2025-12-01');

CREATE TABLE task_audit_log_2025_12 PARTITION OF task_audit_log_partitioned
    FOR VALUES FROM ('2025-12-01') TO ('2026-01-01');

-- Create default partition for future dates
CREATE TABLE task_audit_log_default PARTITION OF task_audit_log_partitioned DEFAULT;

-- Step 3: Create function for automatic partition creation
CREATE OR REPLACE FUNCTION create_monthly_audit_partition(target_date DATE)
RETURNS VOID AS $$
DECLARE
    start_date DATE;
    end_date DATE;
    partition_name TEXT;
    sql_command TEXT;
BEGIN
    -- Calculate partition boundaries
    start_date := date_trunc('month', target_date)::DATE;
    end_date := (date_trunc('month', target_date) + INTERVAL '1 month')::DATE;
    
    -- Generate partition name
    partition_name := 'task_audit_log_' || to_char(start_date, 'YYYY_MM');
    
    -- Check if partition already exists
    IF NOT EXISTS (
        SELECT 1 FROM pg_class 
        WHERE relname = partition_name 
        AND relkind = 'r'
    ) THEN
        -- Create the partition
        sql_command := format(
            'CREATE TABLE %I PARTITION OF task_audit_log_partitioned FOR VALUES FROM (%L) TO (%L)',
            partition_name,
            start_date,
            end_date
        );
        
        EXECUTE sql_command;
        
        RAISE NOTICE 'Created partition: %', partition_name;
    ELSE
        RAISE NOTICE 'Partition % already exists', partition_name;
    END IF;
END;
$$ LANGUAGE plpgsql;

-- Step 4: Create function to automatically create partitions
CREATE OR REPLACE FUNCTION ensure_audit_partition_exists()
RETURNS TRIGGER AS $$
DECLARE
    target_date DATE;
BEGIN
    target_date := DATE(NEW.timestamp);
    
    -- Create partition for current month and next month
    PERFORM create_monthly_audit_partition(target_date);
    PERFORM create_monthly_audit_partition(target_date + INTERVAL '1 month');
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create trigger for automatic partition creation
CREATE TRIGGER ensure_audit_partition_trigger
    BEFORE INSERT ON task_audit_log_partitioned
    FOR EACH ROW EXECUTE FUNCTION ensure_audit_partition_exists();

-- Step 5: Create partitioning strategy for tasks table (for very large installations)
-- This is commented out by default as it requires careful data migration

/*
-- Only uncomment and use this if you have millions of tasks and need partitioning
-- This would require migrating existing data

-- Create tasks table partitioned by repository (list partitioning)
CREATE TABLE tasks_partitioned (
    LIKE tasks INCLUDING ALL
) PARTITION BY LIST (repository);

-- Example partitions for different repositories
CREATE TABLE tasks_repo_main PARTITION OF tasks_partitioned
    FOR VALUES IN ('main', 'master', 'primary');

CREATE TABLE tasks_repo_feature PARTITION OF tasks_partitioned
    FOR VALUES IN ('feature', 'feat', 'features');

CREATE TABLE tasks_repo_bugfix PARTITION OF tasks_partitioned
    FOR VALUES IN ('bugfix', 'fix', 'hotfix');

-- Default partition for other repositories
CREATE TABLE tasks_repo_others PARTITION OF tasks_partitioned DEFAULT;

-- Function to create repository-specific partitions
CREATE OR REPLACE FUNCTION create_repository_partition(repo_name TEXT)
RETURNS VOID AS $$
DECLARE
    partition_name TEXT;
    sql_command TEXT;
BEGIN
    -- Sanitize repository name for table name
    partition_name := 'tasks_repo_' || regexp_replace(lower(repo_name), '[^a-z0-9_]', '_', 'g');
    
    -- Check if partition already exists
    IF NOT EXISTS (
        SELECT 1 FROM pg_class 
        WHERE relname = partition_name 
        AND relkind = 'r'
    ) THEN
        -- Create the partition
        sql_command := format(
            'CREATE TABLE %I PARTITION OF tasks_partitioned FOR VALUES IN (%L)',
            partition_name,
            repo_name
        );
        
        EXECUTE sql_command;
        
        RAISE NOTICE 'Created repository partition: %', partition_name;
    END IF;
END;
$$ LANGUAGE plpgsql;
*/

-- Step 6: Create partitioning for large JSONB data (when metadata becomes very large)
-- Archive table for old task data

CREATE TABLE tasks_archive (
    LIKE tasks INCLUDING ALL
);

-- Function to archive old completed tasks
CREATE OR REPLACE FUNCTION archive_old_completed_tasks(months_old INTEGER DEFAULT 12)
RETURNS INTEGER AS $$
DECLARE
    archived_count INTEGER;
    cutoff_date TIMESTAMP WITH TIME ZONE;
BEGIN
    cutoff_date := CURRENT_TIMESTAMP - (months_old || ' months')::INTERVAL;
    
    -- Move old completed tasks to archive
    WITH archived_tasks AS (
        DELETE FROM tasks 
        WHERE status = 'completed' 
        AND completed_at < cutoff_date
        AND deleted_at IS NULL
        RETURNING *
    )
    INSERT INTO tasks_archive SELECT * FROM archived_tasks;
    
    GET DIAGNOSTICS archived_count = ROW_COUNT;
    
    RETURN archived_count;
END;
$$ LANGUAGE plpgsql;

-- Step 7: Create materialized views for performance on large datasets
CREATE MATERIALIZED VIEW task_summary_by_repository AS
SELECT 
    repository,
    COUNT(*) as total_tasks,
    COUNT(*) FILTER (WHERE status = 'pending') as pending_tasks,
    COUNT(*) FILTER (WHERE status = 'in_progress') as in_progress_tasks,
    COUNT(*) FILTER (WHERE status = 'completed') as completed_tasks,
    COUNT(*) FILTER (WHERE status = 'cancelled') as cancelled_tasks,
    COUNT(*) FILTER (WHERE status = 'blocked') as blocked_tasks,
    AVG(complexity_score) as avg_complexity_score,
    AVG(quality_score) as avg_quality_score,
    AVG(estimated_hours) as avg_estimated_hours,
    AVG(EXTRACT(EPOCH FROM (completed_at - created_at)) / 3600) as avg_completion_hours,
    MAX(created_at) as last_task_created,
    MAX(updated_at) as last_task_updated
FROM tasks 
WHERE deleted_at IS NULL
GROUP BY repository;

-- Create unique index for materialized view refresh
CREATE UNIQUE INDEX idx_task_summary_repository_unique ON task_summary_by_repository (repository);

-- Create materialized view for task metrics by date
CREATE MATERIALIZED VIEW task_metrics_by_date AS
SELECT 
    DATE(created_at) as task_date,
    repository,
    COUNT(*) as tasks_created,
    COUNT(*) FILTER (WHERE status = 'completed') as tasks_completed,
    AVG(complexity_score) as avg_complexity,
    AVG(quality_score) as avg_quality,
    SUM(estimated_hours) as total_estimated_hours,
    COUNT(DISTINCT assignee) as unique_assignees,
    COUNT(*) FILTER (WHERE ai_suggested = true) as ai_generated_tasks
FROM tasks 
WHERE deleted_at IS NULL
GROUP BY DATE(created_at), repository
ORDER BY task_date DESC, repository;

-- Create unique index for date metrics view
CREATE UNIQUE INDEX idx_task_metrics_date_repo_unique ON task_metrics_by_date (task_date, repository);

-- Function to refresh materialized views
CREATE OR REPLACE FUNCTION refresh_task_materialized_views()
RETURNS VOID AS $$
BEGIN
    REFRESH MATERIALIZED VIEW CONCURRENTLY task_summary_by_repository;
    REFRESH MATERIALIZED VIEW CONCURRENTLY task_metrics_by_date;
END;
$$ LANGUAGE plpgsql;

-- Step 8: Create partition maintenance functions

-- Function to drop old audit partitions
CREATE OR REPLACE FUNCTION drop_old_audit_partitions(months_to_keep INTEGER DEFAULT 12)
RETURNS INTEGER AS $$
DECLARE
    partition_record RECORD;
    dropped_count INTEGER := 0;
    cutoff_date DATE;
BEGIN
    cutoff_date := (CURRENT_DATE - (months_to_keep || ' months')::INTERVAL)::DATE;
    
    -- Find and drop old partitions
    FOR partition_record IN
        SELECT schemaname, tablename
        FROM pg_tables
        WHERE tablename LIKE 'task_audit_log_____'
        AND tablename < 'task_audit_log_' || to_char(cutoff_date, 'YYYY_MM')
    LOOP
        EXECUTE 'DROP TABLE IF EXISTS ' || quote_ident(partition_record.schemaname) || '.' || quote_ident(partition_record.tablename);
        dropped_count := dropped_count + 1;
        RAISE NOTICE 'Dropped old partition: %', partition_record.tablename;
    END LOOP;
    
    RETURN dropped_count;
END;
$$ LANGUAGE plpgsql;

-- Function to get partition information
CREATE OR REPLACE FUNCTION get_partition_info()
RETURNS TABLE (
    partition_name TEXT,
    partition_type TEXT,
    size_pretty TEXT,
    row_count BIGINT,
    last_update TIMESTAMP WITH TIME ZONE
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        t.tablename::TEXT,
        'audit_log'::TEXT as partition_type,
        pg_size_pretty(pg_total_relation_size(t.schemaname||'.'||t.tablename))::TEXT,
        (SELECT n_tup_ins + n_tup_upd FROM pg_stat_user_tables WHERE relname = t.tablename)::BIGINT,
        (SELECT last_autoanalyze FROM pg_stat_user_tables WHERE relname = t.tablename)
    FROM pg_tables t
    WHERE t.tablename LIKE 'task_audit_log_%'
    AND t.tablename != 'task_audit_log_partitioned'
    ORDER BY t.tablename;
END;
$$ LANGUAGE plpgsql;

-- Step 9: Create maintenance schedule recommendations

-- Function to suggest partitioning maintenance
CREATE OR REPLACE FUNCTION suggest_partition_maintenance()
RETURNS TABLE (
    maintenance_type TEXT,
    recommendation TEXT,
    urgency TEXT,
    estimated_benefit TEXT
) AS $$
DECLARE
    total_audit_size BIGINT;
    old_partition_count INTEGER;
    large_partition_count INTEGER;
BEGIN
    -- Check audit log size
    SELECT 
        COALESCE(SUM(pg_total_relation_size(schemaname||'.'||tablename)), 0)
    INTO total_audit_size
    FROM pg_tables 
    WHERE tablename LIKE 'task_audit_log_%';
    
    -- Count old partitions (> 6 months)
    SELECT COUNT(*)
    INTO old_partition_count
    FROM pg_tables
    WHERE tablename LIKE 'task_audit_log_____'
    AND tablename < 'task_audit_log_' || to_char(CURRENT_DATE - INTERVAL '6 months', 'YYYY_MM');
    
    -- Count large partitions (> 1GB)
    SELECT COUNT(*)
    INTO large_partition_count
    FROM pg_tables t
    WHERE t.tablename LIKE 'task_audit_log_%'
    AND pg_total_relation_size(t.schemaname||'.'||t.tablename) > 1024*1024*1024;
    
    -- Generate recommendations
    IF total_audit_size > 10*1024*1024*1024 THEN -- > 10GB
        RETURN QUERY SELECT 
            'Archive'::TEXT, 
            'Consider archiving audit logs older than 6 months'::TEXT,
            'High'::TEXT,
            'Significant storage reduction'::TEXT;
    END IF;
    
    IF old_partition_count > 12 THEN
        RETURN QUERY SELECT 
            'Cleanup'::TEXT,
            'Drop audit partitions older than 12 months'::TEXT,
            'Medium'::TEXT,
            'Storage cleanup and improved query performance'::TEXT;
    END IF;
    
    IF large_partition_count > 5 THEN
        RETURN QUERY SELECT 
            'Optimization'::TEXT,
            'Consider more frequent partitioning (weekly instead of monthly)'::TEXT,
            'Low'::TEXT,
            'Improved query performance for recent data'::TEXT;
    END IF;
    
    -- Check if main tasks table is getting large
    IF (SELECT pg_total_relation_size('tasks')) > 5*1024*1024*1024 THEN -- > 5GB
        RETURN QUERY SELECT 
            'Tasks Partitioning'::TEXT,
            'Consider implementing repository-based partitioning for tasks table'::TEXT,
            'Medium'::TEXT,
            'Improved query performance for repository-specific operations'::TEXT;
    END IF;
    
    -- Default recommendation
    IF NOT EXISTS (SELECT 1 FROM pg_tables WHERE tablename LIKE 'task_audit_log_____') THEN
        RETURN QUERY SELECT 
            'Setup'::TEXT,
            'Partitioning is ready but no partitions exist yet'::TEXT,
            'Info'::TEXT,
            'Automatic partition creation will happen on first audit log entry'::TEXT;
    END IF;
END;
$$ LANGUAGE plpgsql;

-- Comments for documentation
COMMENT ON TABLE task_audit_log_partitioned IS 'Partitioned version of audit log for better performance on large datasets';
COMMENT ON FUNCTION create_monthly_audit_partition(DATE) IS 'Creates monthly partitions for audit logs';
COMMENT ON FUNCTION ensure_audit_partition_exists() IS 'Automatically creates partitions when needed';
COMMENT ON FUNCTION archive_old_completed_tasks(INTEGER) IS 'Archives old completed tasks to separate table';
COMMENT ON MATERIALIZED VIEW task_summary_by_repository IS 'Repository-level task metrics summary';
COMMENT ON MATERIALIZED VIEW task_metrics_by_date IS 'Daily task creation and completion metrics';
COMMENT ON FUNCTION refresh_task_materialized_views() IS 'Refreshes all task-related materialized views';
COMMENT ON FUNCTION drop_old_audit_partitions(INTEGER) IS 'Maintenance function to drop old audit partitions';
COMMENT ON FUNCTION get_partition_info() IS 'Returns information about existing partitions';
COMMENT ON FUNCTION suggest_partition_maintenance() IS 'Provides maintenance recommendations for partitioning strategy';