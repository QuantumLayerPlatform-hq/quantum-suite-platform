-- Quantum Suite Core Database Schema
-- Migration: 001_core_schema.sql
-- Description: Creates the foundational tables for the platform

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Enable vector extension for embeddings
CREATE EXTENSION IF NOT EXISTS vector;

-- Create custom types
CREATE TYPE job_status AS ENUM ('pending', 'running', 'completed', 'failed', 'cancelled');
CREATE TYPE priority_level AS ENUM ('low', 'medium', 'high', 'critical');
CREATE TYPE tenant_status AS ENUM ('active', 'inactive', 'suspended', 'trial');

-- =============================================================================
-- CORE DOMAIN TABLES
-- =============================================================================

-- Tenants (Organizations/Companies)
CREATE TABLE tenants (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    plan VARCHAR(50) NOT NULL DEFAULT 'free',
    status tenant_status NOT NULL DEFAULT 'active',
    settings JSONB DEFAULT '{}',
    limits JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    
    -- Indexes
    CONSTRAINT tenants_name_unique UNIQUE (name)
);

CREATE INDEX idx_tenants_status ON tenants(status);
CREATE INDEX idx_tenants_created_at ON tenants(created_at);

-- Users
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    email VARCHAR(255) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    role VARCHAR(50) NOT NULL DEFAULT 'user',
    password_hash VARCHAR(255),
    preferences JSONB DEFAULT '{}',
    is_active BOOLEAN DEFAULT true,
    last_login TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    
    -- Constraints
    CONSTRAINT users_email_check CHECK (email ~* '^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}$')
);

CREATE INDEX idx_users_tenant_id ON users(tenant_id);
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_role ON users(role);

-- Projects
CREATE TABLE projects (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    type VARCHAR(50) NOT NULL DEFAULT 'general',
    settings JSONB DEFAULT '{}',
    status VARCHAR(50) DEFAULT 'active',
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    
    -- Constraints
    UNIQUE(tenant_id, name)
);

CREATE INDEX idx_projects_tenant_id ON projects(tenant_id);
CREATE INDEX idx_projects_status ON projects(status);
CREATE INDEX idx_projects_type ON projects(type);
CREATE INDEX idx_projects_created_by ON projects(created_by);

-- Workspaces
CREATE TABLE workspaces (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    type VARCHAR(50) NOT NULL DEFAULT 'development',
    configuration JSONB DEFAULT '{}',
    state JSONB DEFAULT '{}',
    is_active BOOLEAN DEFAULT true,
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    
    -- Constraints
    UNIQUE(project_id, name)
);

CREATE INDEX idx_workspaces_project_id ON workspaces(project_id);
CREATE INDEX idx_workspaces_type ON workspaces(type);
CREATE INDEX idx_workspaces_created_by ON workspaces(created_by);

-- =============================================================================
-- EVENT STORE TABLES
-- =============================================================================

-- Event Store
CREATE TABLE events (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    stream_id VARCHAR(255) NOT NULL,
    stream_version BIGINT NOT NULL,
    event_type VARCHAR(255) NOT NULL,
    event_data JSONB NOT NULL,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    created_by UUID REFERENCES users(id),
    
    -- Constraints
    CONSTRAINT events_stream_version_unique UNIQUE(stream_id, stream_version)
);

CREATE INDEX idx_events_stream_id ON events(stream_id);
CREATE INDEX idx_events_event_type ON events(event_type);
CREATE INDEX idx_events_created_at ON events(created_at);
CREATE INDEX idx_events_stream_version ON events(stream_id, stream_version);

-- Event Snapshots
CREATE TABLE snapshots (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    stream_id VARCHAR(255) NOT NULL,
    stream_version BIGINT NOT NULL,
    snapshot_data JSONB NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    
    -- Constraints
    UNIQUE(stream_id, stream_version)
);

CREATE INDEX idx_snapshots_stream ON snapshots(stream_id, stream_version DESC);

-- Projections
CREATE TABLE projections (
    id VARCHAR(255) PRIMARY KEY,
    projection_name VARCHAR(255) NOT NULL,
    state JSONB NOT NULL,
    last_event_id UUID,
    last_updated TIMESTAMPTZ DEFAULT NOW(),
    version BIGINT DEFAULT 1,
    is_active BOOLEAN DEFAULT true
);

CREATE INDEX idx_projections_name ON projections(projection_name);
CREATE INDEX idx_projections_updated ON projections(last_updated);

-- =============================================================================
-- AUDIT AND COMPLIANCE TABLES
-- =============================================================================

-- Audit Logs
CREATE TABLE audit_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    user_id UUID REFERENCES users(id),
    action VARCHAR(255) NOT NULL,
    resource VARCHAR(255) NOT NULL,
    resource_id VARCHAR(255),
    changes JSONB DEFAULT '{}',
    ip_address INET,
    user_agent TEXT,
    status VARCHAR(50) DEFAULT 'success',
    error_message TEXT,
    session_id VARCHAR(255),
    request_id VARCHAR(255),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    
    -- Retention policy: Keep for 7 years for compliance
    expires_at TIMESTAMPTZ DEFAULT (NOW() + INTERVAL '7 years')
);

CREATE INDEX idx_audit_logs_tenant_id ON audit_logs(tenant_id);
CREATE INDEX idx_audit_logs_user_id ON audit_logs(user_id);
CREATE INDEX idx_audit_logs_action ON audit_logs(action);
CREATE INDEX idx_audit_logs_resource ON audit_logs(resource, resource_id);
CREATE INDEX idx_audit_logs_created_at ON audit_logs(created_at);
CREATE INDEX idx_audit_logs_expires_at ON audit_logs(expires_at);

-- API Keys for authentication
CREATE TABLE api_keys (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    key_hash VARCHAR(255) NOT NULL UNIQUE,
    key_prefix VARCHAR(20) NOT NULL,
    scopes TEXT[] DEFAULT '{}',
    is_active BOOLEAN DEFAULT true,
    expires_at TIMESTAMPTZ,
    last_used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_api_keys_tenant_id ON api_keys(tenant_id);
CREATE INDEX idx_api_keys_user_id ON api_keys(user_id);
CREATE INDEX idx_api_keys_key_hash ON api_keys(key_hash);
CREATE INDEX idx_api_keys_active ON api_keys(is_active) WHERE is_active = true;

-- =============================================================================
-- METRICS AND MONITORING TABLES
-- =============================================================================

-- System Metrics
CREATE TABLE metrics (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID REFERENCES tenants(id),
    metric_name VARCHAR(255) NOT NULL,
    value DOUBLE PRECISION NOT NULL,
    unit VARCHAR(50),
    tags JSONB DEFAULT '{}',
    timestamp TIMESTAMPTZ DEFAULT NOW(),
    metadata JSONB DEFAULT '{}'
);

-- Partition by time for performance
CREATE INDEX idx_metrics_tenant_name_time ON metrics(tenant_id, metric_name, timestamp DESC);
CREATE INDEX idx_metrics_timestamp ON metrics(timestamp);

-- Jobs Queue
CREATE TABLE jobs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    queue_name VARCHAR(255) NOT NULL DEFAULT 'default',
    job_type VARCHAR(255) NOT NULL,
    payload JSONB NOT NULL,
    status job_status DEFAULT 'pending',
    priority priority_level DEFAULT 'medium',
    attempts INTEGER DEFAULT 0,
    max_attempts INTEGER DEFAULT 3,
    scheduled_at TIMESTAMPTZ DEFAULT NOW(),
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    error_message TEXT,
    result JSONB,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_jobs_tenant_queue ON jobs(tenant_id, queue_name);
CREATE INDEX idx_jobs_status_priority ON jobs(status, priority, scheduled_at) WHERE status = 'pending';
CREATE INDEX idx_jobs_type ON jobs(job_type);
CREATE INDEX idx_jobs_scheduled_at ON jobs(scheduled_at);

-- =============================================================================
-- VECTOR DATABASE TABLES (using pgvector)
-- =============================================================================

-- Code Embeddings
CREATE TABLE code_embeddings (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    project_id UUID REFERENCES projects(id) ON DELETE CASCADE,
    code_hash VARCHAR(64) UNIQUE NOT NULL,
    code_snippet TEXT,
    language VARCHAR(50),
    embedding vector(1536), -- OpenAI ada-002 dimensions
    metadata JSONB DEFAULT '{}',
    quality_score REAL,
    usage_count INTEGER DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Vector similarity search indexes
CREATE INDEX code_embeddings_vector_idx ON code_embeddings 
USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100);

CREATE INDEX idx_code_embeddings_tenant ON code_embeddings(tenant_id);
CREATE INDEX idx_code_embeddings_project ON code_embeddings(project_id);
CREATE INDEX idx_code_embeddings_language ON code_embeddings(language);
CREATE INDEX idx_code_embeddings_hash ON code_embeddings(code_hash);

-- Prompt Embeddings
CREATE TABLE prompt_embeddings (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    prompt_id UUID,
    embedding vector(1536),
    prompt_text TEXT NOT NULL,
    category VARCHAR(100),
    domain VARCHAR(100),
    success_rate REAL DEFAULT 0.0,
    usage_count INTEGER DEFAULT 0,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX prompt_embeddings_vector_idx ON prompt_embeddings 
USING ivfflat (embedding vector_cosine_ops) WITH (lists = 50);

CREATE INDEX idx_prompt_embeddings_tenant ON prompt_embeddings(tenant_id);
CREATE INDEX idx_prompt_embeddings_category ON prompt_embeddings(category);
CREATE INDEX idx_prompt_embeddings_domain ON prompt_embeddings(domain);

-- Knowledge Base Embeddings
CREATE TABLE knowledge_embeddings (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    document_id VARCHAR(255),
    chunk_index INTEGER,
    content TEXT NOT NULL,
    embedding vector(1536),
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    
    -- Constraints
    UNIQUE(document_id, chunk_index)
);

CREATE INDEX knowledge_embeddings_vector_idx ON knowledge_embeddings 
USING ivfflat (embedding vector_cosine_ops) WITH (lists = 200);

CREATE INDEX idx_knowledge_embeddings_tenant ON knowledge_embeddings(tenant_id);
CREATE INDEX idx_knowledge_embeddings_document ON knowledge_embeddings(document_id);

-- =============================================================================
-- CONFIGURATION AND SETTINGS TABLES
-- =============================================================================

-- Feature Flags
CREATE TABLE feature_flags (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) UNIQUE NOT NULL,
    description TEXT,
    is_enabled BOOLEAN DEFAULT false,
    conditions JSONB DEFAULT '{}',
    rollout_percentage INTEGER DEFAULT 0 CHECK (rollout_percentage >= 0 AND rollout_percentage <= 100),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Tenant Feature Overrides
CREATE TABLE tenant_feature_flags (
    tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE,
    feature_flag_name VARCHAR(255) NOT NULL,
    is_enabled BOOLEAN NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    
    PRIMARY KEY (tenant_id, feature_flag_name),
    FOREIGN KEY (feature_flag_name) REFERENCES feature_flags(name) ON DELETE CASCADE
);

-- System Configuration
CREATE TABLE configurations (
    key VARCHAR(255) PRIMARY KEY,
    value JSONB NOT NULL,
    description TEXT,
    is_encrypted BOOLEAN DEFAULT false,
    updated_by UUID REFERENCES users(id),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- =============================================================================
-- TRIGGERS FOR UPDATED_AT TIMESTAMPS
-- =============================================================================

CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Apply triggers to all tables with updated_at columns
CREATE TRIGGER update_tenants_updated_at BEFORE UPDATE ON tenants 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_projects_updated_at BEFORE UPDATE ON projects 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_workspaces_updated_at BEFORE UPDATE ON workspaces 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_api_keys_updated_at BEFORE UPDATE ON api_keys 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_jobs_updated_at BEFORE UPDATE ON jobs 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_code_embeddings_updated_at BEFORE UPDATE ON code_embeddings 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_prompt_embeddings_updated_at BEFORE UPDATE ON prompt_embeddings 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_configurations_updated_at BEFORE UPDATE ON configurations 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- =============================================================================
-- INITIAL DATA
-- =============================================================================

-- Default system tenant
INSERT INTO tenants (id, name, plan, status) VALUES 
('00000000-0000-0000-0000-000000000000', 'System', 'unlimited', 'active');

-- Default system configurations
INSERT INTO configurations (key, value, description) VALUES
('llm.default_model', '"gpt-4-turbo"', 'Default LLM model for code generation'),
('llm.max_tokens', '4000', 'Maximum tokens per LLM request'),
('vector_db.similarity_threshold', '0.8', 'Minimum similarity score for vector search'),
('rate_limits.requests_per_minute', '1000', 'Default API rate limit per minute'),
('feature.experimental_agents', 'false', 'Enable experimental AI agents');

-- Default feature flags
INSERT INTO feature_flags (name, description, is_enabled) VALUES
('vector_search', 'Enable vector similarity search', true),
('advanced_security_scanning', 'Enable advanced security features', false),
('chaos_engineering', 'Enable chaos engineering tools', false),
('multi_cloud_deployment', 'Enable multi-cloud infrastructure deployment', true),
('ai_code_review', 'Enable AI-powered code review', false);

-- Grant permissions (adjust as needed for your setup)
-- GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO quantum_app;
-- GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO quantum_app;