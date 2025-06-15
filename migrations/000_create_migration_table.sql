-- Migration 000: Create Migration Tracking Table
-- Simple migration tracking for schema versioning

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Create simple migration tracking table
CREATE TABLE schema_migrations (
    version VARCHAR(255) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    executed_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL
);

-- Insert this migration
INSERT INTO schema_migrations (version, name) VALUES ('000', 'create_migration_table');