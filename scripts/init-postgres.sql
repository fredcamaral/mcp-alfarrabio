-- Initialize PostgreSQL database for MCP Memory Server

-- Create extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Create user if not exists (for development/testing)
DO $$
BEGIN
    IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'mcpuser') THEN
        CREATE USER mcpuser WITH PASSWORD 'mcppassword';
    END IF;
END
$$;

-- Grant privileges
GRANT ALL PRIVILEGES ON DATABASE mcp_memory TO mcpuser;
GRANT ALL PRIVILEGES ON SCHEMA public TO mcpuser;

-- Create basic tables structure (these will be managed by migrations in production)
CREATE TABLE IF NOT EXISTS schema_migrations (
    version BIGINT PRIMARY KEY,
    dirty BOOLEAN NOT NULL DEFAULT FALSE
);

-- Insert initial migration version
INSERT INTO schema_migrations (version, dirty) VALUES (1, FALSE) ON CONFLICT DO NOTHING;