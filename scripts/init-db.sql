-- DriftMgr Database Initialization Script
-- This script sets up the initial database schema for all phases

-- Create extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Create schemas for different phases
CREATE SCHEMA IF NOT EXISTS drift;
CREATE SCHEMA IF NOT EXISTS remediation;
CREATE SCHEMA IF NOT EXISTS state;
CREATE SCHEMA IF NOT EXISTS discovery;
CREATE SCHEMA IF NOT EXISTS config;
CREATE SCHEMA IF NOT EXISTS monitoring;

-- Phase 1: Drift Results & History Management
CREATE TABLE IF NOT EXISTS drift.drift_results (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    provider VARCHAR(50) NOT NULL,
    status VARCHAR(20) NOT NULL CHECK (status IN ('completed', 'failed', 'running')),
    drift_count INTEGER NOT NULL DEFAULT 0,
    resources JSONB NOT NULL DEFAULT '[]',
    summary JSONB NOT NULL DEFAULT '{}',
    duration INTERVAL,
    error TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_drift_results_timestamp ON drift.drift_results(timestamp);
CREATE INDEX IF NOT EXISTS idx_drift_results_provider ON drift.drift_results(provider);
CREATE INDEX IF NOT EXISTS idx_drift_results_status ON drift.drift_results(status);

-- Phase 2: Remediation Engine
CREATE TABLE IF NOT EXISTS remediation.remediation_jobs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    status VARCHAR(20) NOT NULL CHECK (status IN ('pending', 'running', 'completed', 'failed', 'cancelled')),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    started_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    drift_result_id UUID REFERENCES drift.drift_results(id),
    strategy VARCHAR(50) NOT NULL,
    resources JSONB NOT NULL DEFAULT '[]',
    progress JSONB NOT NULL DEFAULT '{}',
    error TEXT,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_remediation_jobs_status ON remediation.remediation_jobs(status);
CREATE INDEX IF NOT EXISTS idx_remediation_jobs_created_at ON remediation.remediation_jobs(created_at);
CREATE INDEX IF NOT EXISTS idx_remediation_jobs_drift_result_id ON remediation.remediation_jobs(drift_result_id);

-- Phase 3: Enhanced State Management
CREATE TABLE IF NOT EXISTS state.state_operations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    operation_type VARCHAR(20) NOT NULL CHECK (operation_type IN ('import', 'remove', 'move', 'backup', 'restore')),
    resource_address TEXT NOT NULL,
    resource_type VARCHAR(100) NOT NULL,
    provider VARCHAR(50) NOT NULL,
    status VARCHAR(20) NOT NULL CHECK (status IN ('pending', 'running', 'completed', 'failed')),
    state_file_path TEXT NOT NULL,
    backup_path TEXT,
    error TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX IF NOT EXISTS idx_state_operations_type ON state.state_operations(operation_type);
CREATE INDEX IF NOT EXISTS idx_state_operations_status ON state.state_operations(status);
CREATE INDEX IF NOT EXISTS idx_state_operations_provider ON state.state_operations(provider);

CREATE TABLE IF NOT EXISTS state.state_backups (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    size BIGINT NOT NULL,
    checksum VARCHAR(64) NOT NULL,
    description TEXT,
    file_path TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_state_backups_created_at ON state.state_backups(created_at);

-- Phase 4: Advanced Discovery & Scanning
CREATE TABLE IF NOT EXISTS discovery.discovery_jobs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    status VARCHAR(20) NOT NULL CHECK (status IN ('pending', 'running', 'completed', 'failed')),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    started_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    providers TEXT[] NOT NULL,
    progress JSONB NOT NULL DEFAULT '{}',
    results JSONB,
    error TEXT,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_discovery_jobs_status ON discovery.discovery_jobs(status);
CREATE INDEX IF NOT EXISTS idx_discovery_jobs_created_at ON discovery.discovery_jobs(created_at);

CREATE TABLE IF NOT EXISTS discovery.discovered_resources (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    discovery_job_id UUID REFERENCES discovery.discovery_jobs(id),
    address TEXT NOT NULL,
    resource_type VARCHAR(100) NOT NULL,
    provider VARCHAR(50) NOT NULL,
    region VARCHAR(50),
    state VARCHAR(20) NOT NULL CHECK (state IN ('managed', 'unmanaged', 'unknown')),
    tags JSONB NOT NULL DEFAULT '{}',
    attributes JSONB NOT NULL DEFAULT '{}',
    discovered_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_discovered_resources_job_id ON discovery.discovered_resources(discovery_job_id);
CREATE INDEX IF NOT EXISTS idx_discovered_resources_provider ON discovery.discovered_resources(provider);
CREATE INDEX IF NOT EXISTS idx_discovered_resources_state ON discovery.discovered_resources(state);

-- Phase 5: Configuration & Provider Management
CREATE TABLE IF NOT EXISTS config.configurations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    version VARCHAR(20) NOT NULL,
    environment VARCHAR(50) NOT NULL,
    providers JSONB NOT NULL DEFAULT '{}',
    discovery JSONB NOT NULL DEFAULT '{}',
    remediation JSONB NOT NULL DEFAULT '{}',
    storage JSONB NOT NULL DEFAULT '{}',
    logging JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_configurations_environment ON config.configurations(environment);
CREATE INDEX IF NOT EXISTS idx_configurations_version ON config.configurations(version);

CREATE TABLE IF NOT EXISTS config.provider_credentials (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    provider VARCHAR(50) NOT NULL,
    credential_type VARCHAR(20) NOT NULL CHECK (credential_type IN ('env', 'file', 'vault', 'iam')),
    source TEXT NOT NULL,
    encrypted_data BYTEA NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_provider_credentials_provider ON config.provider_credentials(provider);

-- Phase 6: Monitoring & Observability
CREATE TABLE IF NOT EXISTS monitoring.system_metrics (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    cpu_usage DECIMAL(5,2),
    memory_usage DECIMAL(5,2),
    disk_usage DECIMAL(5,2),
    network_in BIGINT,
    network_out BIGINT,
    api_requests_per_second INTEGER,
    active_connections INTEGER,
    data JSONB NOT NULL DEFAULT '{}'
);

CREATE INDEX IF NOT EXISTS idx_system_metrics_timestamp ON monitoring.system_metrics(timestamp);

CREATE TABLE IF NOT EXISTS monitoring.health_checks (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    component VARCHAR(100) NOT NULL,
    status VARCHAR(20) NOT NULL CHECK (status IN ('healthy', 'degraded', 'unhealthy')),
    message TEXT,
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_health_checks_component ON monitoring.health_checks(component);
CREATE INDEX IF NOT EXISTS idx_health_checks_timestamp ON monitoring.health_checks(timestamp);

CREATE TABLE IF NOT EXISTS monitoring.alerts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(200) NOT NULL,
    description TEXT,
    severity VARCHAR(20) NOT NULL CHECK (severity IN ('info', 'warning', 'error', 'critical')),
    condition TEXT NOT NULL,
    threshold DECIMAL(10,2),
    enabled BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_alerts_severity ON monitoring.alerts(severity);
CREATE INDEX IF NOT EXISTS idx_alerts_enabled ON monitoring.alerts(enabled);

CREATE TABLE IF NOT EXISTS monitoring.system_events (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    event_type VARCHAR(100) NOT NULL,
    component VARCHAR(100) NOT NULL,
    message TEXT NOT NULL,
    severity VARCHAR(20) NOT NULL CHECK (severity IN ('info', 'warning', 'error', 'critical')),
    data JSONB NOT NULL DEFAULT '{}',
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_system_events_type ON monitoring.system_events(event_type);
CREATE INDEX IF NOT EXISTS idx_system_events_component ON monitoring.system_events(component);
CREATE INDEX IF NOT EXISTS idx_system_events_timestamp ON monitoring.system_events(timestamp);

-- Create functions for updating timestamps
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Create triggers for updated_at columns
CREATE TRIGGER update_drift_results_updated_at BEFORE UPDATE ON drift.drift_results FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_remediation_jobs_updated_at BEFORE UPDATE ON remediation.remediation_jobs FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_configurations_updated_at BEFORE UPDATE ON config.configurations FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_provider_credentials_updated_at BEFORE UPDATE ON config.provider_credentials FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_alerts_updated_at BEFORE UPDATE ON monitoring.alerts FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Insert initial configuration
INSERT INTO config.configurations (version, environment, providers, discovery, remediation, storage, logging) 
VALUES (
    '1.0.0',
    'development',
    '{"aws": {"enabled": false}, "azure": {"enabled": false}, "gcp": {"enabled": false}, "digitalocean": {"enabled": false}}',
    '{"recursive": true, "include_tags": true, "max_depth": 10, "timeout": "300s", "parallel": true}',
    '{"dry_run": true, "force_apply": false, "skip_backup": false, "max_concurrency": 5, "timeout": "1800s"}',
    '{"backup_retention_days": 30, "encryption_enabled": true}',
    '{"level": "debug", "format": "json", "output": "stdout"}'
) ON CONFLICT DO NOTHING;

-- Insert initial health check
INSERT INTO monitoring.health_checks (component, status, message) 
VALUES ('database', 'healthy', 'Database connection established') 
ON CONFLICT DO NOTHING;

-- Create views for common queries
CREATE OR REPLACE VIEW drift.drift_summary AS
SELECT 
    provider,
    COUNT(*) as total_detections,
    COUNT(*) FILTER (WHERE status = 'completed') as successful_detections,
    COUNT(*) FILTER (WHERE status = 'failed') as failed_detections,
    AVG(drift_count) as avg_drift_count,
    MAX(timestamp) as last_detection
FROM drift.drift_results
GROUP BY provider;

CREATE OR REPLACE VIEW remediation.remediation_summary AS
SELECT 
    strategy,
    COUNT(*) as total_jobs,
    COUNT(*) FILTER (WHERE status = 'completed') as successful_jobs,
    COUNT(*) FILTER (WHERE status = 'failed') as failed_jobs,
    AVG(EXTRACT(EPOCH FROM (completed_at - started_at))) as avg_duration_seconds
FROM remediation.remediation_jobs
WHERE started_at IS NOT NULL
GROUP BY strategy;

-- Grant permissions
GRANT USAGE ON SCHEMA drift TO driftmgr;
GRANT USAGE ON SCHEMA remediation TO driftmgr;
GRANT USAGE ON SCHEMA state TO driftmgr;
GRANT USAGE ON SCHEMA discovery TO driftmgr;
GRANT USAGE ON SCHEMA config TO driftmgr;
GRANT USAGE ON SCHEMA monitoring TO driftmgr;

GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA drift TO driftmgr;
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA remediation TO driftmgr;
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA state TO driftmgr;
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA discovery TO driftmgr;
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA config TO driftmgr;
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA monitoring TO driftmgr;

GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA drift TO driftmgr;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA remediation TO driftmgr;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA state TO driftmgr;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA discovery TO driftmgr;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA config TO driftmgr;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA monitoring TO driftmgr;

-- Create indexes for performance
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_drift_results_created_at ON drift.drift_results(created_at);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_remediation_jobs_updated_at ON remediation.remediation_jobs(updated_at);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_discovery_jobs_updated_at ON discovery.discovery_jobs(updated_at);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_system_metrics_timestamp_desc ON monitoring.system_metrics(timestamp DESC);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_system_events_timestamp_desc ON monitoring.system_events(timestamp DESC);

-- Create partial indexes for active records
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_drift_results_active ON drift.drift_results(id) WHERE status IN ('running', 'completed');
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_remediation_jobs_active ON remediation.remediation_jobs(id) WHERE status IN ('pending', 'running');
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_discovery_jobs_active ON discovery.discovery_jobs(id) WHERE status IN ('pending', 'running');

-- Create composite indexes for common query patterns
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_drift_results_provider_timestamp ON drift.drift_results(provider, timestamp DESC);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_remediation_jobs_strategy_status ON remediation.remediation_jobs(strategy, status);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_discovered_resources_provider_state ON discovery.discovered_resources(provider, state);

-- Analyze tables for query optimization
ANALYZE drift.drift_results;
ANALYZE remediation.remediation_jobs;
ANALYZE state.state_operations;
ANALYZE discovery.discovery_jobs;
ANALYZE discovery.discovered_resources;
ANALYZE config.configurations;
ANALYZE monitoring.system_metrics;
ANALYZE monitoring.health_checks;
ANALYZE monitoring.alerts;
ANALYZE monitoring.system_events;
