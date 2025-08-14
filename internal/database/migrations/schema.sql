CREATE TABLE tenants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    status VARCHAR(50) DEFAULT 'active', -- active, inactive, suspended
    max_workers INTEGER DEFAULT 3,
    current_workers INTEGER DEFAULT 3,
    queue_name VARCHAR(255) NOT NULL, -- tenant_{id}_queue
    consumer_tag VARCHAR(255), -- RabbitMQ consumer identifier
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    deleted_at TIMESTAMPTZ NULL -- Soft delete support
);

CREATE UNIQUE INDEX idx_tenants_queue_name ON tenants(queue_name);
CREATE INDEX idx_tenants_status ON tenants(status) WHERE deleted_at IS NULL;
CREATE TABLE messages (
    id UUID DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    payload JSONB NOT NULL,
    status VARCHAR(50) DEFAULT 'pending', -- pending, processing, completed, failed
    retry_count INTEGER DEFAULT 0,
    max_retries INTEGER DEFAULT 3,
    scheduled_at TIMESTAMPTZ DEFAULT NOW(),
    processed_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    -- Composite primary key MUST include partition key
    PRIMARY KEY (id, tenant_id),
    CONSTRAINT fk_messages_tenant_id FOREIGN KEY (tenant_id) REFERENCES tenants(id)
) PARTITION BY LIST (tenant_id);

-- Indexes for each partition will be created automatically
CREATE INDEX idx_messages_status ON messages(status);
CREATE INDEX idx_messages_created_at ON messages(created_at DESC);
CREATE INDEX idx_messages_scheduled_at ON messages(scheduled_at) WHERE status = 'pending';
CREATE INDEX idx_messages_tenant_id_id ON messages(tenant_id, id); -- For efficient lookups
CREATE TABLE tenant_configs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    config_key VARCHAR(100) NOT NULL,
    config_value JSONB NOT NULL,
    version INTEGER DEFAULT 1,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT fk_tenant_configs_tenant_id FOREIGN KEY (tenant_id) REFERENCES tenants(id),
    CONSTRAINT unique_tenant_config_key UNIQUE (tenant_id, config_key, is_active)
);

CREATE INDEX idx_tenant_configs_lookup ON tenant_configs(tenant_id, config_key) WHERE is_active = true;
CREATE TABLE message_processing_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    message_id UUID NOT NULL,
    tenant_id UUID NOT NULL,
    worker_id VARCHAR(100),
    status VARCHAR(50) NOT NULL, -- started, completed, failed, retrying
    error_message TEXT NULL,
    processing_duration_ms INTEGER NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    -- Foreign key must reference composite primary key
    CONSTRAINT fk_logs_message FOREIGN KEY (message_id, tenant_id) REFERENCES messages(id, tenant_id),
    CONSTRAINT fk_logs_tenant_id FOREIGN KEY (tenant_id) REFERENCES tenants(id)
);

CREATE INDEX idx_logs_message_id ON message_processing_logs(message_id);
CREATE INDEX idx_logs_tenant_created ON message_processing_logs(tenant_id, created_at DESC);
CREATE INDEX idx_logs_status ON message_processing_logs(status, created_at DESC);
CREATE TABLE dead_letter_messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    original_message_id UUID NOT NULL,
    tenant_id UUID NOT NULL,
    payload JSONB NOT NULL,
    failure_reason TEXT NOT NULL,
    retry_count INTEGER NOT NULL,
    last_error TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT fk_dlm_tenant_id FOREIGN KEY (tenant_id) REFERENCES tenants(id)
);

CREATE INDEX idx_dlm_tenant_id ON dead_letter_messages(tenant_id);
CREATE INDEX idx_dlm_created_at ON dead_letter_messages(created_at DESC);
