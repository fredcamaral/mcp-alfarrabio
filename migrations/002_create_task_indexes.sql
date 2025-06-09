-- Migration 002: Create Task Table Indexes
-- Description: Creates comprehensive indexes for optimal query performance on tasks table
-- Created: 2025-06-09
-- Version: 1.0.0

-- Primary performance indexes based on common query patterns

-- TRD Required Indexes (from TRD specification)
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_tasks_repository ON tasks(repository);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_tasks_status ON tasks(status);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_tasks_created_at ON tasks(created_at);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_tasks_parent ON tasks(parent_task_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_tasks_prd ON tasks(prd_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_tasks_tags ON tasks USING GIN(tags);

-- Enhanced Performance Indexes for Common Query Patterns

-- Repository and session filtering (most common queries)
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_tasks_repo_session ON tasks(repository, session_id) WHERE deleted_at IS NULL;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_tasks_repo_status ON tasks(repository, status) WHERE deleted_at IS NULL;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_tasks_session_status ON tasks(session_id, status) WHERE deleted_at IS NULL;

-- Priority and complexity filtering
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_tasks_priority ON tasks(priority) WHERE deleted_at IS NULL;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_tasks_complexity ON tasks(complexity) WHERE deleted_at IS NULL;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_tasks_priority_status ON tasks(priority, status) WHERE deleted_at IS NULL;

-- Assignment and ownership
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_tasks_assignee ON tasks(assignee) WHERE deleted_at IS NULL AND assignee IS NOT NULL;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_tasks_assignee_status ON tasks(assignee, status) WHERE deleted_at IS NULL AND assignee IS NOT NULL;

-- Time-based queries (dashboard, reporting)
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_tasks_updated_at ON tasks(updated_at DESC) WHERE deleted_at IS NULL;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_tasks_due_date ON tasks(due_date) WHERE deleted_at IS NULL AND due_date IS NOT NULL;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_tasks_completed_at ON tasks(completed_at DESC) WHERE deleted_at IS NULL AND completed_at IS NOT NULL;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_tasks_started_at ON tasks(started_at) WHERE deleted_at IS NULL AND started_at IS NOT NULL;

-- AI and quality scoring queries
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_tasks_ai_suggested ON tasks(ai_suggested) WHERE deleted_at IS NULL;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_tasks_quality_score ON tasks(quality_score DESC) WHERE deleted_at IS NULL AND quality_score IS NOT NULL;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_tasks_complexity_score ON tasks(complexity_score DESC) WHERE deleted_at IS NULL AND complexity_score IS NOT NULL;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_tasks_business_value ON tasks(business_value_score DESC) WHERE deleted_at IS NULL AND business_value_score IS NOT NULL;

-- PRD and source tracking
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_tasks_source_prd ON tasks(source_prd_id) WHERE deleted_at IS NULL;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_tasks_source_section ON tasks(source_section) WHERE deleted_at IS NULL AND source_section IS NOT NULL;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_tasks_pattern_id ON tasks(pattern_id) WHERE deleted_at IS NULL AND pattern_id IS NOT NULL;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_tasks_template_id ON tasks(template_id) WHERE deleted_at IS NULL AND template_id IS NOT NULL;

-- Branch and version control
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_tasks_branch ON tasks(branch) WHERE deleted_at IS NULL AND branch IS NOT NULL;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_tasks_repo_branch ON tasks(repository, branch) WHERE deleted_at IS NULL;

-- Hierarchical queries (parent-child relationships)
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_tasks_parent_status ON tasks(parent_task_id, status) WHERE deleted_at IS NULL;

-- Full-text search optimization
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_tasks_search_vector ON tasks USING GIN(search_vector);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_tasks_title_search ON tasks USING GIN(to_tsvector('english', title)) WHERE deleted_at IS NULL;

-- JSON field indexes for complex queries
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_tasks_dependencies ON tasks USING GIN(dependencies) WHERE deleted_at IS NULL;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_tasks_blocks ON tasks USING GIN(blocks) WHERE deleted_at IS NULL;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_tasks_acceptance_criteria ON tasks USING GIN(acceptance_criteria) WHERE deleted_at IS NULL;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_tasks_required_skills ON tasks USING GIN(required_skills) WHERE deleted_at IS NULL;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_tasks_metadata ON tasks USING GIN(metadata) WHERE deleted_at IS NULL;

-- Soft delete optimization
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_tasks_deleted_at ON tasks(deleted_at);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_tasks_active ON tasks(id) WHERE deleted_at IS NULL;

-- Type-specific indexes
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_tasks_type ON tasks(type) WHERE deleted_at IS NULL;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_tasks_type_status ON tasks(type, status) WHERE deleted_at IS NULL;

-- Composite indexes for complex dashboard queries
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_tasks_dashboard_main ON tasks(repository, status, priority, created_at DESC) WHERE deleted_at IS NULL;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_tasks_assignee_dashboard ON tasks(assignee, status, due_date) WHERE deleted_at IS NULL AND assignee IS NOT NULL;

-- Effort and estimation indexes
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_tasks_estimated_hours ON tasks(estimated_hours) WHERE deleted_at IS NULL AND estimated_hours IS NOT NULL;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_tasks_story_points ON tasks(story_points) WHERE deleted_at IS NULL AND story_points IS NOT NULL;

-- Risk assessment indexes
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_tasks_technical_risk ON tasks(technical_risk) WHERE deleted_at IS NULL;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_tasks_business_impact ON tasks(business_impact) WHERE deleted_at IS NULL;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_tasks_risk_impact ON tasks(technical_risk, business_impact) WHERE deleted_at IS NULL;

-- Analytics and reporting indexes
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_tasks_created_month ON tasks(date_trunc('month', created_at)) WHERE deleted_at IS NULL;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_tasks_completed_month ON tasks(date_trunc('month', completed_at)) WHERE deleted_at IS NULL AND completed_at IS NOT NULL;

-- Supporting table indexes

-- PRD table indexes
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_prds_repository ON prds(repository) WHERE deleted_at IS NULL;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_prds_filename ON prds(filename) WHERE deleted_at IS NULL;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_prds_created_at ON prds(created_at DESC) WHERE deleted_at IS NULL;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_prds_complexity_score ON prds(complexity_score DESC) WHERE deleted_at IS NULL;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_prds_task_count ON prds(task_count DESC) WHERE deleted_at IS NULL;

-- Task patterns indexes
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_task_patterns_type ON task_patterns(pattern_type);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_task_patterns_name ON task_patterns(name);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_task_patterns_usage ON task_patterns(usage_count DESC);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_task_patterns_success ON task_patterns(success_rate DESC);

-- Task templates indexes
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_task_templates_category ON task_templates(category);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_task_templates_name ON task_templates(name);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_task_templates_usage ON task_templates(usage_count DESC);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_task_templates_success ON task_templates(success_rate DESC);

-- Task effort breakdown indexes
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_task_effort_task_id ON task_effort_breakdown(task_id);

-- Task quality issues indexes
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_task_quality_task_id ON task_quality_issues(task_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_task_quality_type ON task_quality_issues(issue_type);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_task_quality_severity ON task_quality_issues(severity);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_task_quality_resolved ON task_quality_issues(resolved);

-- Unique constraints for data integrity
CREATE UNIQUE INDEX CONCURRENTLY IF NOT EXISTS idx_tasks_unique_active_title_repo 
    ON tasks(title, repository) 
    WHERE deleted_at IS NULL;

CREATE UNIQUE INDEX CONCURRENTLY IF NOT EXISTS idx_prds_unique_active_repo_filename 
    ON prds(repository, filename) 
    WHERE deleted_at IS NULL;

CREATE UNIQUE INDEX CONCURRENTLY IF NOT EXISTS idx_task_patterns_unique_name 
    ON task_patterns(name);

CREATE UNIQUE INDEX CONCURRENTLY IF NOT EXISTS idx_task_templates_unique_name 
    ON task_templates(name);

-- Partial indexes for specific query optimization

-- Recently active tasks (last 7 days)
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_tasks_recent_activity 
    ON tasks(updated_at DESC) 
    WHERE deleted_at IS NULL 
    AND updated_at > CURRENT_TIMESTAMP - INTERVAL '7 days';

-- High priority incomplete tasks
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_tasks_high_priority_incomplete 
    ON tasks(priority, created_at DESC) 
    WHERE deleted_at IS NULL 
    AND status IN ('pending', 'in_progress', 'blocked') 
    AND priority IN ('high', 'critical', 'blocking');

-- Overdue tasks
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_tasks_overdue 
    ON tasks(due_date, priority) 
    WHERE deleted_at IS NULL 
    AND due_date < CURRENT_TIMESTAMP 
    AND status NOT IN ('completed', 'cancelled');

-- Tasks without assignee
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_tasks_unassigned 
    ON tasks(priority, created_at DESC) 
    WHERE deleted_at IS NULL 
    AND assignee IS NULL 
    AND status IN ('pending', 'blocked');

-- AI-generated tasks for analysis
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_tasks_ai_analysis 
    ON tasks(ai_model, quality_score DESC) 
    WHERE deleted_at IS NULL 
    AND ai_suggested = true;

-- Comments for documentation
COMMENT ON INDEX idx_tasks_repository IS 'TRD required: repository filtering';
COMMENT ON INDEX idx_tasks_status IS 'TRD required: status filtering';
COMMENT ON INDEX idx_tasks_created_at IS 'TRD required: time-based queries';
COMMENT ON INDEX idx_tasks_parent IS 'TRD required: hierarchical queries';
COMMENT ON INDEX idx_tasks_prd IS 'TRD required: PRD relationship queries';
COMMENT ON INDEX idx_tasks_tags IS 'TRD required: tag-based filtering using GIN';
COMMENT ON INDEX idx_tasks_search_vector IS 'Full-text search optimization';
COMMENT ON INDEX idx_tasks_dashboard_main IS 'Optimized for main dashboard queries';
COMMENT ON INDEX idx_tasks_recent_activity IS 'Partial index for recently active tasks performance';