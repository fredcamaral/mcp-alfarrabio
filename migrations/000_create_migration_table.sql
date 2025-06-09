-- Migration 000: Create Migration Tracking Table
-- Description: Creates the migration tracking system for schema versioning and rollback capabilities
-- Created: 2025-06-09
-- Version: 1.0.0

-- Enable UUID extension if not already enabled
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Create migration tracking table
CREATE TABLE IF NOT EXISTS schema_migrations (
    id SERIAL PRIMARY KEY,
    migration_id VARCHAR(255) NOT NULL UNIQUE,
    filename VARCHAR(255) NOT NULL,
    version INTEGER NOT NULL,
    description TEXT,
    
    -- Execution tracking
    executed_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    execution_time_ms INTEGER,
    rollback_sql TEXT,
    
    -- Validation and safety
    checksum VARCHAR(64), -- SHA-256 hash of migration content
    is_rolled_back BOOLEAN DEFAULT false NOT NULL,
    rolled_back_at TIMESTAMP WITH TIME ZONE,
    rollback_reason TEXT,
    
    -- Migration metadata
    batch_id INTEGER,
    dependencies TEXT[], -- Array of migration IDs this depends on
    tags TEXT[],
    environment VARCHAR(50), -- dev, staging, production
    
    -- Execution context
    executed_by VARCHAR(255),
    execution_host VARCHAR(255),
    database_version VARCHAR(100),
    application_version VARCHAR(100),
    
    -- Safety and validation flags
    is_destructive BOOLEAN DEFAULT false NOT NULL,
    has_rollback BOOLEAN DEFAULT false NOT NULL,
    validation_status VARCHAR(20) DEFAULT 'pending' CHECK (validation_status IN ('pending', 'passed', 'failed', 'skipped')),
    validation_errors JSONB DEFAULT '[]',
    
    -- Performance tracking
    affected_rows INTEGER,
    table_changes JSONB DEFAULT '{}', -- Tables created, modified, or dropped
    index_changes JSONB DEFAULT '{}', -- Indexes created or dropped
    
    -- Additional metadata
    migration_type VARCHAR(50) DEFAULT 'schema' CHECK (migration_type IN ('schema', 'data', 'rollback', 'hotfix')),
    priority VARCHAR(20) DEFAULT 'normal' CHECK (priority IN ('low', 'normal', 'high', 'critical')),
    notes TEXT,
    external_refs JSONB DEFAULT '{}', -- Links to tickets, PRs, etc.
    
    -- Audit trail
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL
);

-- Create indexes for performance
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_schema_migrations_migration_id ON schema_migrations(migration_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_schema_migrations_version ON schema_migrations(version);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_schema_migrations_executed_at ON schema_migrations(executed_at DESC);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_schema_migrations_batch_id ON schema_migrations(batch_id) WHERE batch_id IS NOT NULL;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_schema_migrations_rollback ON schema_migrations(is_rolled_back, rolled_back_at) WHERE is_rolled_back = true;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_schema_migrations_validation ON schema_migrations(validation_status) WHERE validation_status != 'passed';
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_schema_migrations_environment ON schema_migrations(environment);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_schema_migrations_type ON schema_migrations(migration_type);

-- Create composite indexes for common queries
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_schema_migrations_version_rollback ON schema_migrations(version DESC, is_rolled_back);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_schema_migrations_env_version ON schema_migrations(environment, version DESC);

-- Create migration state tracking table
CREATE TABLE IF NOT EXISTS migration_state (
    id SERIAL PRIMARY KEY,
    state_key VARCHAR(100) UNIQUE NOT NULL,
    state_value TEXT,
    environment VARCHAR(50) NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_by VARCHAR(255)
);

-- Create indexes for migration state
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_migration_state_key ON migration_state(state_key);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_migration_state_environment ON migration_state(environment);

-- Insert initial state values
INSERT INTO migration_state (state_key, state_value, environment, updated_by) VALUES
    ('current_version', '0', 'dev', 'system'),
    ('last_migration', NULL, 'dev', 'system'),
    ('migration_lock', 'false', 'dev', 'system'),
    ('pending_migrations', '[]', 'dev', 'system')
ON CONFLICT (state_key) DO NOTHING;

-- Create migration dependencies table for complex dependency tracking
CREATE TABLE IF NOT EXISTS migration_dependencies (
    id SERIAL PRIMARY KEY,
    migration_id VARCHAR(255) NOT NULL REFERENCES schema_migrations(migration_id) ON DELETE CASCADE,
    depends_on_migration_id VARCHAR(255) NOT NULL,
    dependency_type VARCHAR(50) DEFAULT 'required' CHECK (dependency_type IN ('required', 'optional', 'conflicting')),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL
);

-- Create indexes for dependencies
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_migration_dependencies_migration_id ON migration_dependencies(migration_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_migration_dependencies_depends_on ON migration_dependencies(depends_on_migration_id);
CREATE UNIQUE INDEX CONCURRENTLY IF NOT EXISTS idx_migration_dependencies_unique ON migration_dependencies(migration_id, depends_on_migration_id);

-- Create migration batches table for grouped operations
CREATE TABLE IF NOT EXISTS migration_batches (
    id SERIAL PRIMARY KEY,
    batch_name VARCHAR(255),
    description TEXT,
    environment VARCHAR(50) NOT NULL,
    started_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    completed_at TIMESTAMP WITH TIME ZONE,
    status VARCHAR(20) DEFAULT 'running' CHECK (status IN ('running', 'completed', 'failed', 'cancelled')),
    total_migrations INTEGER DEFAULT 0,
    successful_migrations INTEGER DEFAULT 0,
    failed_migrations INTEGER DEFAULT 0,
    executed_by VARCHAR(255),
    execution_host VARCHAR(255),
    error_message TEXT,
    rollback_batch_id INTEGER REFERENCES migration_batches(id),
    notes TEXT
);

-- Create indexes for batches
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_migration_batches_environment ON migration_batches(environment);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_migration_batches_status ON migration_batches(status);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_migration_batches_started_at ON migration_batches(started_at DESC);

-- Create triggers for automatic updates

-- Update updated_at timestamp on schema_migrations
CREATE OR REPLACE FUNCTION update_schema_migrations_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER schema_migrations_updated_at_trigger
    BEFORE UPDATE ON schema_migrations
    FOR EACH ROW EXECUTE FUNCTION update_schema_migrations_updated_at();

-- Update migration state timestamp
CREATE OR REPLACE FUNCTION update_migration_state_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER migration_state_updated_at_trigger
    BEFORE UPDATE ON migration_state
    FOR EACH ROW EXECUTE FUNCTION update_migration_state_updated_at();

-- Create validation functions

-- Function to validate migration dependencies
CREATE OR REPLACE FUNCTION validate_migration_dependencies(migration_id_param VARCHAR(255))
RETURNS BOOLEAN AS $$
DECLARE
    dep_record RECORD;
    is_valid BOOLEAN := true;
BEGIN
    -- Check if all required dependencies are satisfied
    FOR dep_record IN
        SELECT depends_on_migration_id 
        FROM migration_dependencies 
        WHERE migration_id = migration_id_param 
        AND dependency_type = 'required'
    LOOP
        -- Check if the dependency migration has been executed
        IF NOT EXISTS (
            SELECT 1 FROM schema_migrations 
            WHERE migration_id = dep_record.depends_on_migration_id 
            AND is_rolled_back = false
        ) THEN
            is_valid := false;
            EXIT;
        END IF;
    END LOOP;
    
    -- Check for conflicting migrations
    FOR dep_record IN
        SELECT depends_on_migration_id 
        FROM migration_dependencies 
        WHERE migration_id = migration_id_param 
        AND dependency_type = 'conflicting'
    LOOP
        -- Check if the conflicting migration has been executed
        IF EXISTS (
            SELECT 1 FROM schema_migrations 
            WHERE migration_id = dep_record.depends_on_migration_id 
            AND is_rolled_back = false
        ) THEN
            is_valid := false;
            EXIT;
        END IF;
    END LOOP;
    
    RETURN is_valid;
END;
$$ LANGUAGE plpgsql;

-- Function to get current migration version
CREATE OR REPLACE FUNCTION get_current_migration_version(env VARCHAR(50) DEFAULT 'dev')
RETURNS INTEGER AS $$
DECLARE
    current_version INTEGER;
BEGIN
    SELECT COALESCE(MAX(version), 0) INTO current_version
    FROM schema_migrations 
    WHERE environment = env 
    AND is_rolled_back = false;
    
    RETURN current_version;
END;
$$ LANGUAGE plpgsql;

-- Function to get pending migrations
CREATE OR REPLACE FUNCTION get_pending_migrations(env VARCHAR(50) DEFAULT 'dev')
RETURNS TABLE (
    migration_id VARCHAR(255),
    filename VARCHAR(255),
    version INTEGER,
    description TEXT,
    dependencies_satisfied BOOLEAN
) AS $$
BEGIN
    RETURN QUERY
    WITH executed_migrations AS (
        SELECT sm.migration_id
        FROM schema_migrations sm
        WHERE sm.environment = env 
        AND sm.is_rolled_back = false
    ),
    all_possible_migrations AS (
        -- This would be populated by scanning the migrations directory
        -- For now, it's a placeholder that would be populated by the migration tool
        SELECT 
            'placeholder'::VARCHAR(255) as migration_id,
            'placeholder.sql'::VARCHAR(255) as filename,
            0 as version,
            'Placeholder migration'::TEXT as description
        WHERE false  -- This ensures no rows are returned from this placeholder
    )
    SELECT 
        apm.migration_id,
        apm.filename,
        apm.version,
        apm.description,
        COALESCE(validate_migration_dependencies(apm.migration_id), true) as dependencies_satisfied
    FROM all_possible_migrations apm
    LEFT JOIN executed_migrations em ON apm.migration_id = em.migration_id
    WHERE em.migration_id IS NULL
    ORDER BY apm.version;
END;
$$ LANGUAGE plpgsql;

-- Function to lock migrations for atomic execution
CREATE OR REPLACE FUNCTION acquire_migration_lock(env VARCHAR(50) DEFAULT 'dev')
RETURNS BOOLEAN AS $$
DECLARE
    lock_acquired BOOLEAN := false;
BEGIN
    -- Try to acquire the lock atomically
    UPDATE migration_state 
    SET state_value = 'true', updated_at = CURRENT_TIMESTAMP
    WHERE state_key = 'migration_lock' 
    AND environment = env 
    AND state_value = 'false';
    
    GET DIAGNOSTICS lock_acquired = FOUND;
    
    RETURN lock_acquired;
END;
$$ LANGUAGE plpgsql;

-- Function to release migration lock
CREATE OR REPLACE FUNCTION release_migration_lock(env VARCHAR(50) DEFAULT 'dev')
RETURNS VOID AS $$
BEGIN
    UPDATE migration_state 
    SET state_value = 'false', updated_at = CURRENT_TIMESTAMP
    WHERE state_key = 'migration_lock' 
    AND environment = env;
END;
$$ LANGUAGE plpgsql;

-- Function to get migration statistics
CREATE OR REPLACE FUNCTION get_migration_statistics(env VARCHAR(50) DEFAULT 'dev')
RETURNS TABLE (
    total_migrations INTEGER,
    successful_migrations INTEGER,
    rolled_back_migrations INTEGER,
    current_version INTEGER,
    last_migration_date TIMESTAMP WITH TIME ZONE,
    avg_execution_time_ms NUMERIC,
    destructive_migrations INTEGER,
    migrations_without_rollback INTEGER
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        COUNT(*)::INTEGER as total_migrations,
        COUNT(*) FILTER (WHERE is_rolled_back = false)::INTEGER as successful_migrations,
        COUNT(*) FILTER (WHERE is_rolled_back = true)::INTEGER as rolled_back_migrations,
        get_current_migration_version(env) as current_version,
        MAX(executed_at) as last_migration_date,
        ROUND(AVG(execution_time_ms), 2) as avg_execution_time_ms,
        COUNT(*) FILTER (WHERE is_destructive = true)::INTEGER as destructive_migrations,
        COUNT(*) FILTER (WHERE has_rollback = false)::INTEGER as migrations_without_rollback
    FROM schema_migrations
    WHERE environment = env;
END;
$$ LANGUAGE plpgsql;

-- Comments for documentation
COMMENT ON TABLE schema_migrations IS 'Tracks all database schema migrations with rollback support and validation';
COMMENT ON TABLE migration_state IS 'Stores migration system state including locks and current version';
COMMENT ON TABLE migration_dependencies IS 'Defines dependencies between migrations for proper execution order';
COMMENT ON TABLE migration_batches IS 'Groups related migrations for atomic execution and rollback';
COMMENT ON FUNCTION validate_migration_dependencies(VARCHAR) IS 'Validates that all migration dependencies are satisfied';
COMMENT ON FUNCTION get_current_migration_version(VARCHAR) IS 'Returns the current migration version for an environment';
COMMENT ON FUNCTION get_pending_migrations(VARCHAR) IS 'Returns all pending migrations with dependency status';
COMMENT ON FUNCTION acquire_migration_lock(VARCHAR) IS 'Atomically acquires migration lock to prevent concurrent execution';
COMMENT ON FUNCTION release_migration_lock(VARCHAR) IS 'Releases migration lock after execution';
COMMENT ON FUNCTION get_migration_statistics(VARCHAR) IS 'Returns comprehensive migration statistics and health metrics';