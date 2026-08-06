-- ops-system schema fragment (SaaS monitoring + logs data plane + UModel)

CREATE TABLE IF NOT EXISTS ops_vm_clusters (
    id UUID PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    mode VARCHAR(50) NOT NULL,
    zone_id UUID,
    cluster_id UUID,
    release_name VARCHAR(100),
    namespace VARCHAR(100),
    select_url TEXT,
    insert_url TEXT,
    vmauth_url TEXT,
    target_url TEXT,
    status VARCHAR(20) DEFAULT 'active',
    created_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ,
    deleted_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS ops_log_clusters (
    id UUID PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    backend_type VARCHAR(50) NOT NULL DEFAULT 'victorialogs',
    zone_id UUID,
    cluster_id UUID,
    release_name VARCHAR(100),
    namespace VARCHAR(100),
    insert_url TEXT,
    select_url TEXT,
    kafka_brokers TEXT,
    kafka_topic VARCHAR(255),
    status VARCHAR(20) DEFAULT 'active',
    created_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ,
    deleted_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS ops_log_instances (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    zone_id UUID,
    backend_type VARCHAR(50) NOT NULL DEFAULT 'victorialogs',
    instance_name VARCHAR(255) NOT NULL,
    release_name VARCHAR(100),
    namespace VARCHAR(100),
    endpoint VARCHAR(255),
    token VARCHAR(255),
    retention_days INT,
    spec JSONB,
    status VARCHAR(20) DEFAULT 'creating',
    created_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ,
    deleted_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS ops_entities (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    entity_type VARCHAR(50) NOT NULL,
    name VARCHAR(255) NOT NULL,
    display_name VARCHAR(255),
    labels JSONB DEFAULT '{}',
    status VARCHAR(20) DEFAULT 'active',
    created_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ,
    deleted_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS ops_metric_sets (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    name VARCHAR(255) NOT NULL,
    display_name VARCHAR(255),
    component VARCHAR(100),
    description TEXT,
    labels JSONB DEFAULT '{}',
    status VARCHAR(20) DEFAULT 'active',
    created_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ,
    deleted_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS ops_log_sets (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    name VARCHAR(255) NOT NULL,
    display_name VARCHAR(255),
    component VARCHAR(100),
    description TEXT,
    labels JSONB DEFAULT '{}',
    status VARCHAR(20) DEFAULT 'active',
    created_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ,
    deleted_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS ops_data_links (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    entity_id UUID NOT NULL,
    target_type VARCHAR(50) NOT NULL,
    target_id UUID NOT NULL,
    relation_type VARCHAR(50) DEFAULT 'observes',
    status VARCHAR(20) DEFAULT 'active',
    created_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ,
    deleted_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS ops_business_clusters (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    instance_id UUID NOT NULL,
    name VARCHAR(255) NOT NULL,
    display_name VARCHAR(255),
    kubeconfig_path VARCHAR(500),
    agent_status VARCHAR(20) DEFAULT 'pending',
    log_agent_status VARCHAR(20) DEFAULT 'pending',
    log_instance_id UUID,
    labels JSONB DEFAULT '{}',
    metrics_collect_config JSONB DEFAULT '{}',
    logs_collect_config JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ,
    deleted_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_entities_tenant ON ops_entities(tenant_id);
CREATE INDEX IF NOT EXISTS idx_metric_sets_tenant ON ops_metric_sets(tenant_id);
CREATE INDEX IF NOT EXISTS idx_log_sets_tenant ON ops_log_sets(tenant_id);
CREATE INDEX IF NOT EXISTS idx_log_clusters_zone ON ops_log_clusters(zone_id);
CREATE INDEX IF NOT EXISTS idx_log_instances_tenant ON ops_log_instances(tenant_id);
CREATE INDEX IF NOT EXISTS idx_data_links_entity ON ops_data_links(entity_id);

-- Platform core refactor (IAM / audit)
CREATE TABLE IF NOT EXISTS ops_workspace_members (
    id UUID PRIMARY KEY,
    workspace_id UUID NOT NULL,
    user_id UUID NOT NULL,
    role VARCHAR(20) NOT NULL DEFAULT 'member',
    created_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ,
    deleted_at TIMESTAMPTZ
);
CREATE UNIQUE INDEX IF NOT EXISTS uk_ws_member ON ops_workspace_members(workspace_id, user_id) WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS ops_api_tokens (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL,
    name VARCHAR(255) NOT NULL,
    token_prefix VARCHAR(16) NOT NULL,
    token_hash VARCHAR(255) NOT NULL,
    scope VARCHAR(50) DEFAULT 'read_write',
    expires_at TIMESTAMPTZ,
    last_used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ,
    deleted_at TIMESTAMPTZ
);

-- ops_audit_logs extended columns: ip, user_agent, status (managed by AutoMigrate)

