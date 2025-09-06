-- QAgent Module Database Schema
-- Migration: 002_qagent_schema.sql
-- Description: Creates tables for AI-powered code generation and agent management

-- Custom types for QAgent
CREATE TYPE agent_type AS ENUM ('code_generator', 'code_reviewer', 'refactor_agent', 'documentation_agent', 'test_agent');
CREATE TYPE agent_status AS ENUM ('idle', 'busy', 'training', 'error', 'offline');
CREATE TYPE generation_status AS ENUM ('pending', 'processing', 'completed', 'failed', 'cancelled');
CREATE TYPE validation_status AS ENUM ('valid', 'invalid', 'warning', 'needs_review');

-- =============================================================================
-- AGENT MANAGEMENT TABLES
-- =============================================================================

-- Agent Definitions
CREATE TABLE agents (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    type agent_type NOT NULL,
    version VARCHAR(50) DEFAULT '1.0.0',
    description TEXT,
    capabilities TEXT[] DEFAULT '{}',
    configuration JSONB DEFAULT '{}',
    status agent_status DEFAULT 'idle',
    model_config JSONB DEFAULT '{}',
    performance_metrics JSONB DEFAULT '{}',
    is_active BOOLEAN DEFAULT true,
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    
    -- Constraints
    UNIQUE(tenant_id, name, version)
);

CREATE INDEX idx_agents_tenant_id ON agents(tenant_id);
CREATE INDEX idx_agents_type ON agents(type);
CREATE INDEX idx_agents_status ON agents(status);
CREATE INDEX idx_agents_active ON agents(is_active) WHERE is_active = true;

-- Agent Sessions (tracks agent conversations/contexts)
CREATE TABLE agent_sessions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    agent_id UUID NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id),
    context JSONB DEFAULT '{}',
    memory JSONB DEFAULT '{}',
    conversation_history JSONB DEFAULT '[]',
    is_active BOOLEAN DEFAULT true,
    expires_at TIMESTAMPTZ DEFAULT (NOW() + INTERVAL '24 hours'),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_agent_sessions_agent_id ON agent_sessions(agent_id);
CREATE INDEX idx_agent_sessions_workspace_id ON agent_sessions(workspace_id);
CREATE INDEX idx_agent_sessions_user_id ON agent_sessions(user_id);
CREATE INDEX idx_agent_sessions_active ON agent_sessions(is_active) WHERE is_active = true;
CREATE INDEX idx_agent_sessions_expires_at ON agent_sessions(expires_at);

-- =============================================================================
-- PROMPT MANAGEMENT TABLES
-- =============================================================================

-- Prompt Templates
CREATE TABLE prompt_templates (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    category VARCHAR(100),
    subcategory VARCHAR(100),
    template TEXT NOT NULL,
    variables JSONB DEFAULT '[]',
    examples JSONB DEFAULT '[]',
    version INTEGER DEFAULT 1,
    is_active BOOLEAN DEFAULT true,
    performance_metrics JSONB DEFAULT '{}',
    tags TEXT[] DEFAULT '{}',
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    
    -- Constraints
    UNIQUE(tenant_id, name, version)
);

CREATE INDEX idx_prompt_templates_tenant_id ON prompt_templates(tenant_id);
CREATE INDEX idx_prompt_templates_category ON prompt_templates(category, subcategory);
CREATE INDEX idx_prompt_templates_active ON prompt_templates(is_active) WHERE is_active = true;
CREATE INDEX idx_prompt_templates_tags ON prompt_templates USING GIN(tags);

-- Meta-Prompt Configurations
CREATE TABLE meta_prompts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    base_template_id UUID NOT NULL REFERENCES prompt_templates(id),
    name VARCHAR(255) NOT NULL,
    strategy VARCHAR(100), -- 'chain_of_thought', 'few_shot', 'zero_shot', 'self_criticism'
    layers JSONB NOT NULL, -- Array of prompt layers
    optimization_config JSONB DEFAULT '{}',
    performance_metrics JSONB DEFAULT '{}',
    is_active BOOLEAN DEFAULT true,
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    
    UNIQUE(tenant_id, name)
);

CREATE INDEX idx_meta_prompts_tenant_id ON meta_prompts(tenant_id);
CREATE INDEX idx_meta_prompts_strategy ON meta_prompts(strategy);
CREATE INDEX idx_meta_prompts_base_template ON meta_prompts(base_template_id);

-- =============================================================================
-- CODE GENERATION TABLES
-- =============================================================================

-- Code Generation Requests
CREATE TABLE code_generations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    agent_id UUID NOT NULL REFERENCES agents(id),
    session_id UUID REFERENCES agent_sessions(id),
    workspace_id UUID NOT NULL REFERENCES workspaces(id),
    user_id UUID NOT NULL REFERENCES users(id),
    
    -- Request details
    input_prompt TEXT NOT NULL,
    processed_prompt TEXT,
    language VARCHAR(50),
    framework VARCHAR(100),
    context JSONB DEFAULT '{}',
    requirements JSONB DEFAULT '{}',
    
    -- Generation results
    generated_code TEXT,
    explanation TEXT,
    suggestions JSONB DEFAULT '[]',
    
    -- Status and metadata
    status generation_status DEFAULT 'pending',
    priority priority_level DEFAULT 'medium',
    
    -- LLM usage tracking
    model_used VARCHAR(100),
    tokens_input INTEGER DEFAULT 0,
    tokens_output INTEGER DEFAULT 0,
    tokens_total INTEGER DEFAULT 0,
    cost_usd DECIMAL(10,6) DEFAULT 0,
    
    -- Timing
    processing_time_ms INTEGER,
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    
    -- Error handling
    error_message TEXT,
    retry_count INTEGER DEFAULT 0,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_code_generations_tenant_id ON code_generations(tenant_id);
CREATE INDEX idx_code_generations_agent_id ON code_generations(agent_id);
CREATE INDEX idx_code_generations_workspace_id ON code_generations(workspace_id);
CREATE INDEX idx_code_generations_user_id ON code_generations(user_id);
CREATE INDEX idx_code_generations_status ON code_generations(status);
CREATE INDEX idx_code_generations_language ON code_generations(language);
CREATE INDEX idx_code_generations_created_at ON code_generations(created_at);

-- Code Validation Results
CREATE TABLE code_validations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    generation_id UUID NOT NULL REFERENCES code_generations(id) ON DELETE CASCADE,
    
    -- Validation results
    status validation_status DEFAULT 'valid',
    syntax_valid BOOLEAN DEFAULT true,
    semantic_valid BOOLEAN DEFAULT true,
    security_valid BOOLEAN DEFAULT true,
    style_valid BOOLEAN DEFAULT true,
    
    -- Detailed results
    syntax_errors JSONB DEFAULT '[]',
    semantic_errors JSONB DEFAULT '[]',
    security_issues JSONB DEFAULT '[]',
    style_issues JSONB DEFAULT '[]',
    suggestions JSONB DEFAULT '[]',
    
    -- Metrics
    complexity_score REAL,
    maintainability_score REAL,
    quality_score REAL,
    
    -- Tree-sitter analysis
    ast_data JSONB,
    dependencies JSONB DEFAULT '[]',
    
    -- Validation metadata
    validator_version VARCHAR(50),
    validation_time_ms INTEGER,
    validated_at TIMESTAMPTZ DEFAULT NOW(),
    
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_code_validations_generation_id ON code_validations(generation_id);
CREATE INDEX idx_code_validations_status ON code_validations(status);
CREATE INDEX idx_code_validations_quality ON code_validations(quality_score DESC NULLS LAST);

-- Self-Criticism and Feedback Loop
CREATE TABLE code_feedback (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    generation_id UUID NOT NULL REFERENCES code_generations(id) ON DELETE CASCADE,
    validation_id UUID REFERENCES code_validations(id),
    
    -- Feedback source
    feedback_type VARCHAR(50), -- 'user', 'automated', 'peer_review', 'execution'
    feedback_source UUID, -- user_id or system identifier
    
    -- Feedback content
    rating INTEGER CHECK (rating >= 1 AND rating <= 5),
    issues JSONB DEFAULT '[]',
    improvements JSONB DEFAULT '[]',
    comments TEXT,
    
    -- Impact tracking
    was_helpful BOOLEAN,
    action_taken VARCHAR(100), -- 'regenerated', 'modified', 'accepted', 'rejected'
    
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_code_feedback_generation_id ON code_feedback(generation_id);
CREATE INDEX idx_code_feedback_type ON code_feedback(feedback_type);
CREATE INDEX idx_code_feedback_rating ON code_feedback(rating);

-- Code Generation Analytics
CREATE TABLE generation_analytics (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    agent_id UUID NOT NULL REFERENCES agents(id),
    
    -- Time period
    period_start TIMESTAMPTZ NOT NULL,
    period_end TIMESTAMPTZ NOT NULL,
    
    -- Metrics
    total_generations INTEGER DEFAULT 0,
    successful_generations INTEGER DEFAULT 0,
    failed_generations INTEGER DEFAULT 0,
    average_processing_time_ms INTEGER DEFAULT 0,
    total_tokens_used INTEGER DEFAULT 0,
    total_cost_usd DECIMAL(10,4) DEFAULT 0,
    
    -- Quality metrics
    average_quality_score REAL DEFAULT 0,
    user_satisfaction_score REAL DEFAULT 0,
    code_acceptance_rate REAL DEFAULT 0,
    
    -- Language breakdown
    language_stats JSONB DEFAULT '{}',
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    
    -- Ensure no overlapping periods for same agent
    EXCLUDE USING gist (
        agent_id WITH =,
        tstzrange(period_start, period_end, '[]') WITH &&
    )
);

CREATE INDEX idx_generation_analytics_tenant_id ON generation_analytics(tenant_id);
CREATE INDEX idx_generation_analytics_agent_id ON generation_analytics(agent_id);
CREATE INDEX idx_generation_analytics_period ON generation_analytics(period_start, period_end);

-- =============================================================================
-- AGENT LEARNING AND OPTIMIZATION TABLES
-- =============================================================================

-- Agent Training Data
CREATE TABLE agent_training_data (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    agent_id UUID NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    
    -- Training sample
    input_prompt TEXT NOT NULL,
    expected_output TEXT,
    actual_output TEXT,
    
    -- Quality metrics
    quality_score REAL,
    user_rating INTEGER CHECK (user_rating >= 1 AND user_rating <= 5),
    
    -- Context
    language VARCHAR(50),
    framework VARCHAR(100),
    complexity_level VARCHAR(20), -- 'simple', 'medium', 'complex', 'expert'
    context_tags TEXT[] DEFAULT '{}',
    
    -- Usage tracking
    used_for_training BOOLEAN DEFAULT false,
    training_weight REAL DEFAULT 1.0,
    
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_agent_training_data_agent_id ON agent_training_data(agent_id);
CREATE INDEX idx_agent_training_data_language ON agent_training_data(language);
CREATE INDEX idx_agent_training_data_quality ON agent_training_data(quality_score DESC NULLS LAST);
CREATE INDEX idx_agent_training_data_tags ON agent_training_data USING GIN(context_tags);

-- Agent Performance Tracking
CREATE TABLE agent_performance (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    agent_id UUID NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    
    -- Performance period
    measurement_date DATE NOT NULL DEFAULT CURRENT_DATE,
    
    -- Core metrics
    success_rate REAL DEFAULT 0, -- Percentage of successful generations
    average_response_time_ms INTEGER DEFAULT 0,
    code_quality_score REAL DEFAULT 0, -- Average quality of generated code
    user_satisfaction REAL DEFAULT 0, -- Average user rating
    
    -- Efficiency metrics
    tokens_per_generation REAL DEFAULT 0,
    cost_per_generation DECIMAL(8,4) DEFAULT 0,
    regeneration_rate REAL DEFAULT 0, -- How often code needs to be regenerated
    
    -- Learning metrics
    improvement_rate REAL DEFAULT 0, -- How much the agent is improving
    accuracy_trend REAL DEFAULT 0, -- Trend in accuracy over time
    
    -- Comparative metrics
    relative_performance REAL DEFAULT 0, -- Performance vs other agents
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    
    UNIQUE(agent_id, measurement_date)
);

CREATE INDEX idx_agent_performance_agent_date ON agent_performance(agent_id, measurement_date DESC);
CREATE INDEX idx_agent_performance_success_rate ON agent_performance(success_rate DESC);

-- =============================================================================
-- CODE LIBRARY AND REUSABILITY TABLES
-- =============================================================================

-- Code Patterns Library
CREATE TABLE code_patterns (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    
    -- Pattern identification
    name VARCHAR(255) NOT NULL,
    category VARCHAR(100),
    pattern_type VARCHAR(100), -- 'design_pattern', 'algorithm', 'snippet', 'template'
    
    -- Pattern content
    code_template TEXT NOT NULL,
    description TEXT,
    use_cases TEXT[],
    parameters JSONB DEFAULT '{}',
    
    -- Metadata
    language VARCHAR(50) NOT NULL,
    framework VARCHAR(100),
    complexity_level VARCHAR(20),
    tags TEXT[] DEFAULT '{}',
    
    -- Usage statistics
    usage_count INTEGER DEFAULT 0,
    success_rate REAL DEFAULT 0,
    user_rating REAL DEFAULT 0,
    
    -- Embeddings for similarity search
    embedding vector(1536),
    
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    
    UNIQUE(tenant_id, name, language)
);

CREATE INDEX code_patterns_vector_idx ON code_patterns 
USING ivfflat (embedding vector_cosine_ops) WITH (lists = 50);

CREATE INDEX idx_code_patterns_tenant_id ON code_patterns(tenant_id);
CREATE INDEX idx_code_patterns_category ON code_patterns(category, pattern_type);
CREATE INDEX idx_code_patterns_language ON code_patterns(language);
CREATE INDEX idx_code_patterns_tags ON code_patterns USING GIN(tags);
CREATE INDEX idx_code_patterns_usage ON code_patterns(usage_count DESC);

-- Code Generation History (for learning from successful patterns)
CREATE TABLE generation_history (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    generation_id UUID NOT NULL REFERENCES code_generations(id) ON DELETE CASCADE,
    
    -- Success indicators
    was_accepted BOOLEAN DEFAULT false,
    user_rating INTEGER,
    code_executed BOOLEAN DEFAULT false,
    tests_passed BOOLEAN DEFAULT false,
    
    -- Learning data
    similar_patterns UUID[], -- References to code_patterns.id
    reused_components JSONB DEFAULT '[]',
    novel_patterns JSONB DEFAULT '[]',
    
    -- Outcome tracking
    final_code TEXT, -- The code after user modifications (if any)
    modifications_made JSONB DEFAULT '[]',
    
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_generation_history_generation_id ON generation_history(generation_id);
CREATE INDEX idx_generation_history_accepted ON generation_history(was_accepted);
CREATE INDEX idx_generation_history_rating ON generation_history(user_rating DESC NULLS LAST);

-- =============================================================================
-- TRIGGERS FOR QAGENT TABLES
-- =============================================================================

-- Add updated_at triggers for QAgent tables
CREATE TRIGGER update_agents_updated_at BEFORE UPDATE ON agents 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_agent_sessions_updated_at BEFORE UPDATE ON agent_sessions 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_prompt_templates_updated_at BEFORE UPDATE ON prompt_templates 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_meta_prompts_updated_at BEFORE UPDATE ON meta_prompts 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_code_generations_updated_at BEFORE UPDATE ON code_generations 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_code_patterns_updated_at BEFORE UPDATE ON code_patterns 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Function to automatically update agent status based on activity
CREATE OR REPLACE FUNCTION update_agent_status()
RETURNS TRIGGER AS $$
BEGIN
    -- Update agent status to busy when a new generation starts
    IF NEW.status = 'processing' AND OLD.status = 'pending' THEN
        UPDATE agents SET status = 'busy' WHERE id = NEW.agent_id;
    END IF;
    
    -- Update agent status back to idle when generation completes
    IF NEW.status IN ('completed', 'failed', 'cancelled') AND OLD.status = 'processing' THEN
        -- Check if there are other pending/processing requests for this agent
        IF NOT EXISTS (
            SELECT 1 FROM code_generations 
            WHERE agent_id = NEW.agent_id 
            AND status IN ('pending', 'processing')
            AND id != NEW.id
        ) THEN
            UPDATE agents SET status = 'idle' WHERE id = NEW.agent_id;
        END IF;
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_agent_status 
    AFTER UPDATE OF status ON code_generations
    FOR EACH ROW 
    EXECUTE FUNCTION update_agent_status();

-- =============================================================================
-- INITIAL DATA FOR QAGENT MODULE
-- =============================================================================

-- Default prompt templates
INSERT INTO prompt_templates (id, tenant_id, name, category, template, variables, created_by) VALUES
(
    uuid_generate_v4(),
    '00000000-0000-0000-0000-000000000000',
    'basic_code_generation',
    'code_generation',
    'Generate {{language}} code that {{requirement}}. Follow best practices and include error handling.',
    '["language", "requirement"]',
    (SELECT id FROM users WHERE email = 'system@quantum-suite.io' LIMIT 1)
),
(
    uuid_generate_v4(),
    '00000000-0000-0000-0000-000000000000',
    'api_endpoint_generation',
    'web_development',
    'Create a {{framework}} REST API endpoint for {{resource}} with the following operations: {{operations}}. Include proper validation, error handling, and documentation.',
    '["framework", "resource", "operations"]',
    (SELECT id FROM users WHERE email = 'system@quantum-suite.io' LIMIT 1)
),
(
    uuid_generate_v4(),
    '00000000-0000-0000-0000-000000000000',
    'database_query_generation',
    'database',
    'Write a {{database_type}} query to {{operation}} in the {{table_name}} table. {{additional_requirements}}',
    '["database_type", "operation", "table_name", "additional_requirements"]',
    (SELECT id FROM users WHERE email = 'system@quantum-suite.io' LIMIT 1)
);

-- Default system agent
INSERT INTO agents (id, tenant_id, name, type, description, capabilities, created_by) VALUES
(
    uuid_generate_v4(),
    '00000000-0000-0000-0000-000000000000',
    'default_code_generator',
    'code_generator',
    'Default AI code generation agent with multi-language support',
    ARRAY['python', 'javascript', 'typescript', 'go', 'java', 'rust'],
    (SELECT id FROM users WHERE email = 'system@quantum-suite.io' LIMIT 1)
);

-- Grant permissions
-- GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO quantum_app;
-- GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO quantum_app;