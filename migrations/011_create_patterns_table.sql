-- Migration: Create patterns detection system tables
-- Description: Support for pattern detection, learning, and evolution tracking
-- Version: 011
-- Date: 2025-01-10

-- Pattern types enum
CREATE TYPE pattern_type AS ENUM (
    'code',           -- Code patterns (design patterns, idioms, anti-patterns)
    'workflow',       -- Workflow patterns (development processes, task sequences)
    'architectural',  -- Architectural patterns (system design, component relationships)
    'behavioral',     -- Behavioral patterns (user interactions, decision making)
    'error',         -- Error patterns (common mistakes, debugging sequences)
    'optimization',  -- Optimization patterns (performance improvements)
    'refactoring'    -- Refactoring patterns (code improvement strategies)
);

-- Pattern confidence levels
CREATE TYPE confidence_level AS ENUM (
    'very_low',      -- 0-20% confidence
    'low',           -- 20-40% confidence
    'medium',        -- 40-60% confidence
    'high',          -- 60-80% confidence
    'very_high'      -- 80-100% confidence
);

-- Pattern validation status
CREATE TYPE validation_status AS ENUM (
    'unvalidated',   -- Pattern detected but not validated
    'pending',       -- Awaiting validation
    'validated',     -- Pattern confirmed as valid
    'invalidated',   -- Pattern marked as invalid
    'evolved'        -- Pattern has evolved into a new pattern
);

-- Main patterns table
CREATE TABLE IF NOT EXISTS patterns (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- Pattern identification
    name VARCHAR(255) NOT NULL,
    description TEXT,
    pattern_type pattern_type NOT NULL,
    category VARCHAR(100),  -- Sub-category within type
    
    -- Pattern signature and matching
    signature JSONB NOT NULL,  -- Pattern matching rules/criteria
    keywords TEXT[],           -- Associated keywords for search
    
    -- Repository and context
    repository_url VARCHAR(500),  -- Repository where pattern was detected
    file_patterns TEXT[],         -- File types/paths where pattern applies
    language VARCHAR(50),         -- Programming language (if applicable)
    
    -- Confidence and validation
    confidence_score FLOAT CHECK (confidence_score >= 0 AND confidence_score <= 1),
    confidence_level confidence_level GENERATED ALWAYS AS (
        CASE 
            WHEN confidence_score < 0.2 THEN 'very_low'::confidence_level
            WHEN confidence_score < 0.4 THEN 'low'::confidence_level
            WHEN confidence_score < 0.6 THEN 'medium'::confidence_level
            WHEN confidence_score < 0.8 THEN 'high'::confidence_level
            ELSE 'very_high'::confidence_level
        END
    ) STORED,
    validation_status validation_status DEFAULT 'unvalidated',
    
    -- Learning metrics
    occurrence_count INTEGER DEFAULT 0,
    positive_feedback_count INTEGER DEFAULT 0,
    negative_feedback_count INTEGER DEFAULT 0,
    last_seen_at TIMESTAMPTZ,
    
    -- Evolution tracking
    parent_pattern_id UUID REFERENCES patterns(id),
    evolution_reason TEXT,
    version INTEGER DEFAULT 1,
    
    -- Metadata
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

-- Pattern occurrences table
CREATE TABLE IF NOT EXISTS pattern_occurrences (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    pattern_id UUID NOT NULL REFERENCES patterns(id) ON DELETE CASCADE,
    
    -- Occurrence context
    repository_url VARCHAR(500) NOT NULL,
    file_path TEXT,
    line_start INTEGER,
    line_end INTEGER,
    
    -- Code context
    code_snippet TEXT,
    surrounding_context TEXT,
    
    -- Detection details
    detection_score FLOAT CHECK (detection_score >= 0 AND detection_score <= 1),
    detection_method VARCHAR(100),  -- Method used to detect (ML, rule-based, etc.)
    
    -- Session and chunk references
    session_id UUID,
    chunk_id UUID,
    
    -- Metadata
    metadata JSONB DEFAULT '{}',
    detected_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

-- Pattern relationships table
CREATE TABLE IF NOT EXISTS pattern_relationships (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- Relationship definition
    source_pattern_id UUID NOT NULL REFERENCES patterns(id) ON DELETE CASCADE,
    target_pattern_id UUID NOT NULL REFERENCES patterns(id) ON DELETE CASCADE,
    relationship_type VARCHAR(50) NOT NULL,  -- 'extends', 'conflicts_with', 'complements', 'alternative_to', etc.
    
    -- Relationship strength
    strength FLOAT CHECK (strength >= 0 AND strength <= 1),
    confidence FLOAT CHECK (confidence >= 0 AND confidence <= 1),
    
    -- Context
    context TEXT,
    examples JSONB DEFAULT '[]',
    
    -- Metadata
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    
    -- Ensure no duplicate relationships
    UNIQUE(source_pattern_id, target_pattern_id, relationship_type)
);

-- Pattern learning history table
CREATE TABLE IF NOT EXISTS pattern_learning_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    pattern_id UUID NOT NULL REFERENCES patterns(id) ON DELETE CASCADE,
    
    -- Learning event
    event_type VARCHAR(50) NOT NULL,  -- 'detection', 'validation', 'feedback', 'evolution', 'merge'
    event_data JSONB NOT NULL,
    
    -- Metrics before and after
    confidence_before FLOAT,
    confidence_after FLOAT,
    occurrence_count_before INTEGER,
    occurrence_count_after INTEGER,
    
    -- User feedback
    user_feedback TEXT,
    feedback_sentiment VARCHAR(20),  -- 'positive', 'negative', 'neutral'
    
    -- Context
    session_id UUID,
    triggered_by VARCHAR(100),  -- 'user', 'system', 'scheduled'
    
    -- Metadata
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

-- Pattern templates table (for common patterns)
CREATE TABLE IF NOT EXISTS pattern_templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- Template definition
    name VARCHAR(255) NOT NULL UNIQUE,
    description TEXT,
    pattern_type pattern_type NOT NULL,
    
    -- Template matching rules
    template_signature JSONB NOT NULL,
    example_code TEXT,
    
    -- Usage statistics
    instantiation_count INTEGER DEFAULT 0,
    success_rate FLOAT,
    
    -- Metadata
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

-- Pattern validation queue
CREATE TABLE IF NOT EXISTS pattern_validation_queue (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    pattern_id UUID NOT NULL REFERENCES patterns(id) ON DELETE CASCADE,
    
    -- Validation request
    priority INTEGER DEFAULT 0,
    reason TEXT,
    requested_by VARCHAR(100),
    
    -- Validation status
    status VARCHAR(50) DEFAULT 'pending',  -- 'pending', 'in_progress', 'completed', 'failed'
    validator_type VARCHAR(50),  -- 'manual', 'automated', 'ml_model'
    
    -- Results
    validation_result JSONB,
    validated_at TIMESTAMPTZ,
    
    -- Metadata
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for performance
CREATE INDEX idx_patterns_type ON patterns(pattern_type);
CREATE INDEX idx_patterns_repository ON patterns(repository_url);
CREATE INDEX idx_patterns_confidence ON patterns(confidence_score);
CREATE INDEX idx_patterns_validation_status ON patterns(validation_status);
CREATE INDEX idx_patterns_keywords ON patterns USING GIN(keywords);
CREATE INDEX idx_patterns_signature ON patterns USING GIN(signature);
CREATE INDEX idx_patterns_metadata ON patterns USING GIN(metadata);
CREATE INDEX idx_patterns_created_at ON patterns(created_at);

CREATE INDEX idx_occurrences_pattern ON pattern_occurrences(pattern_id);
CREATE INDEX idx_occurrences_repository ON pattern_occurrences(repository_url);
CREATE INDEX idx_occurrences_detected_at ON pattern_occurrences(detected_at);
CREATE INDEX idx_occurrences_session ON pattern_occurrences(session_id);
CREATE INDEX idx_occurrences_chunk ON pattern_occurrences(chunk_id);

CREATE INDEX idx_relationships_source ON pattern_relationships(source_pattern_id);
CREATE INDEX idx_relationships_target ON pattern_relationships(target_pattern_id);
CREATE INDEX idx_relationships_type ON pattern_relationships(relationship_type);

CREATE INDEX idx_learning_pattern ON pattern_learning_history(pattern_id);
CREATE INDEX idx_learning_event_type ON pattern_learning_history(event_type);
CREATE INDEX idx_learning_created_at ON pattern_learning_history(created_at);

CREATE INDEX idx_validation_queue_pattern ON pattern_validation_queue(pattern_id);
CREATE INDEX idx_validation_queue_status ON pattern_validation_queue(status);
CREATE INDEX idx_validation_queue_priority ON pattern_validation_queue(priority DESC);

-- Full-text search index
CREATE INDEX idx_patterns_fulltext ON patterns USING GIN(
    to_tsvector('english', 
        COALESCE(name, '') || ' ' || 
        COALESCE(description, '') || ' ' || 
        COALESCE(category, '')
    )
);

-- Create triggers for updated_at
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_patterns_updated_at BEFORE UPDATE ON patterns
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_pattern_templates_updated_at BEFORE UPDATE ON pattern_templates
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_pattern_validation_queue_updated_at BEFORE UPDATE ON pattern_validation_queue
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Function to update pattern confidence based on feedback
CREATE OR REPLACE FUNCTION update_pattern_confidence(
    p_pattern_id UUID,
    p_is_positive BOOLEAN
) RETURNS VOID AS $$
DECLARE
    v_current_confidence FLOAT;
    v_positive_count INTEGER;
    v_negative_count INTEGER;
    v_new_confidence FLOAT;
BEGIN
    -- Get current values
    SELECT confidence_score, positive_feedback_count, negative_feedback_count
    INTO v_current_confidence, v_positive_count, v_negative_count
    FROM patterns
    WHERE id = p_pattern_id;
    
    -- Update feedback counts
    IF p_is_positive THEN
        v_positive_count := v_positive_count + 1;
    ELSE
        v_negative_count := v_negative_count + 1;
    END IF;
    
    -- Calculate new confidence using Bayesian approach
    v_new_confidence := (v_positive_count + 1.0) / (v_positive_count + v_negative_count + 2.0);
    
    -- Update pattern
    UPDATE patterns
    SET confidence_score = v_new_confidence,
        positive_feedback_count = v_positive_count,
        negative_feedback_count = v_negative_count,
        updated_at = CURRENT_TIMESTAMP
    WHERE id = p_pattern_id;
    
    -- Log learning event
    INSERT INTO pattern_learning_history (
        pattern_id, event_type, event_data,
        confidence_before, confidence_after,
        feedback_sentiment
    ) VALUES (
        p_pattern_id, 
        'feedback',
        jsonb_build_object('is_positive', p_is_positive),
        v_current_confidence,
        v_new_confidence,
        CASE WHEN p_is_positive THEN 'positive' ELSE 'negative' END
    );
END;
$$ LANGUAGE plpgsql;

-- Function to detect pattern hierarchies
CREATE OR REPLACE FUNCTION detect_pattern_hierarchy(
    p_pattern_id UUID
) RETURNS TABLE (
    level INTEGER,
    pattern_id UUID,
    pattern_name VARCHAR(255),
    relationship_type VARCHAR(50)
) AS $$
WITH RECURSIVE pattern_hierarchy AS (
    -- Base case: the pattern itself
    SELECT 0 as level, 
           p.id as pattern_id, 
           p.name as pattern_name,
           NULL::VARCHAR(50) as relationship_type
    FROM patterns p
    WHERE p.id = p_pattern_id
    
    UNION ALL
    
    -- Recursive case: find related patterns
    SELECT ph.level + 1,
           p.id,
           p.name,
           pr.relationship_type
    FROM pattern_hierarchy ph
    JOIN pattern_relationships pr ON ph.pattern_id = pr.source_pattern_id
    JOIN patterns p ON pr.target_pattern_id = p.id
    WHERE ph.level < 5  -- Limit recursion depth
)
SELECT * FROM pattern_hierarchy
ORDER BY level, pattern_name;
$$ LANGUAGE sql;

-- Create materialized view for pattern statistics
CREATE MATERIALIZED VIEW pattern_statistics AS
SELECT 
    p.id,
    p.name,
    p.pattern_type,
    p.confidence_score,
    p.occurrence_count,
    COUNT(DISTINCT po.repository_url) as repository_count,
    COUNT(DISTINCT po.id) as total_occurrences,
    AVG(po.detection_score) as avg_detection_score,
    MAX(po.detected_at) as last_detected,
    (p.positive_feedback_count::FLOAT / NULLIF(p.positive_feedback_count + p.negative_feedback_count, 0)) as feedback_ratio
FROM patterns p
LEFT JOIN pattern_occurrences po ON p.id = po.pattern_id
GROUP BY p.id, p.name, p.pattern_type, p.confidence_score, p.occurrence_count, 
         p.positive_feedback_count, p.negative_feedback_count;

-- Create index on materialized view
CREATE INDEX idx_pattern_statistics_id ON pattern_statistics(id);

-- Function to refresh pattern statistics
CREATE OR REPLACE FUNCTION refresh_pattern_statistics() RETURNS VOID AS $$
BEGIN
    REFRESH MATERIALIZED VIEW CONCURRENTLY pattern_statistics;
END;
$$ LANGUAGE plpgsql;

-- Add comments for documentation
COMMENT ON TABLE patterns IS 'Core table for storing detected patterns across repositories';
COMMENT ON TABLE pattern_occurrences IS 'Individual instances where patterns were detected';
COMMENT ON TABLE pattern_relationships IS 'Relationships between different patterns';
COMMENT ON TABLE pattern_learning_history IS 'Historical tracking of pattern learning and evolution';
COMMENT ON TABLE pattern_templates IS 'Pre-defined pattern templates for common patterns';
COMMENT ON TABLE pattern_validation_queue IS 'Queue for patterns awaiting validation';

COMMENT ON COLUMN patterns.signature IS 'JSON structure defining the pattern matching rules';
COMMENT ON COLUMN patterns.confidence_score IS 'Confidence score from 0 to 1 based on occurrences and feedback';
COMMENT ON COLUMN pattern_occurrences.detection_score IS 'Score indicating how well this occurrence matches the pattern';
COMMENT ON COLUMN pattern_relationships.strength IS 'Strength of the relationship from 0 to 1';