CREATE TABLE IF NOT EXISTS tenants (
    id UUID PRIMARY KEY,
    name VARCHAR(128) NOT NULL,
    daily_quota INT NOT NULL DEFAULT 50000,
    monthly_quota INT NOT NULL DEFAULT 500000,
    created_at TIMESTAMP NOT NULL
);

CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    email VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    name VARCHAR(128) NOT NULL,
    role VARCHAR(32) NOT NULL,
    created_at TIMESTAMP NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_users_tenant ON users(tenant_id);

CREATE TABLE IF NOT EXISTS contacts (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    email VARCHAR(255) NOT NULL,
    name VARCHAR(128) NOT NULL DEFAULT '',
    attrs JSONB,
    status VARCHAR(32) NOT NULL DEFAULT 'subscribed',
    soft_bounce INT NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    UNIQUE (tenant_id, email)
);

CREATE TABLE IF NOT EXISTS contact_lists (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    name VARCHAR(128) NOT NULL,
    created_at TIMESTAMP NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_lists_tenant ON contact_lists(tenant_id);

CREATE TABLE IF NOT EXISTS list_memberships (
    list_id UUID NOT NULL REFERENCES contact_lists(id) ON DELETE CASCADE,
    contact_id UUID NOT NULL REFERENCES contacts(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL,
    created_at TIMESTAMP NOT NULL,
    PRIMARY KEY (list_id, contact_id)
);

CREATE TABLE IF NOT EXISTS templates (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    name VARCHAR(128) NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_tpl_tenant ON templates(tenant_id);

CREATE TABLE IF NOT EXISTS template_versions (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    template_id UUID NOT NULL REFERENCES templates(id) ON DELETE CASCADE,
    version INT NOT NULL,
    subject VARCHAR(255) NOT NULL,
    ast JSONB NOT NULL,
    created_at TIMESTAMP NOT NULL
);

CREATE TABLE IF NOT EXISTS campaigns (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    name VARCHAR(128) NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'draft',
    from_name VARCHAR(128) NOT NULL,
    from_email VARCHAR(255) NOT NULL,
    reply_to VARCHAR(255),
    subject VARCHAR(255) NOT NULL,
    list_id UUID,
    template_ver_id UUID,
    strategy VARCHAR(32) NOT NULL DEFAULT 'immediate',
    scheduled_at TIMESTAMP,
    batch_size INT NOT NULL DEFAULT 200,
    batch_interval_s INT NOT NULL DEFAULT 60,
    ramp_percent INT NOT NULL DEFAULT 20,
    channel_strategy VARCHAR(32) NOT NULL DEFAULT 'balanced',
    paused_reason VARCHAR(255),
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_camp_tenant ON campaigns(tenant_id, status);

CREATE TABLE IF NOT EXISTS campaign_recipients (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    campaign_id UUID NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    contact_id UUID NOT NULL,
    email VARCHAR(255) NOT NULL,
    domain VARCHAR(128) NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'pending',
    channel_key VARCHAR(64),
    message_id VARCHAR(128) UNIQUE,
    attempt INT NOT NULL DEFAULT 0,
    last_error VARCHAR(512),
    sent_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    UNIQUE (campaign_id, contact_id)
);
CREATE INDEX IF NOT EXISTS idx_rcpt_camp_status ON campaign_recipients(campaign_id, status);
CREATE INDEX IF NOT EXISTS idx_rcpt_domain ON campaign_recipients(domain);

CREATE TABLE IF NOT EXISTS send_channels (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    key VARCHAR(64) NOT NULL,
    name VARCHAR(128) NOT NULL,
    provider VARCHAR(32) NOT NULL,
    weight INT NOT NULL DEFAULT 1,
    health DOUBLE PRECISION NOT NULL DEFAULT 1,
    state VARCHAR(16) NOT NULL DEFAULT 'closed',
    fail_streak INT NOT NULL DEFAULT 0,
    host VARCHAR(255),
    port INT,
    username VARCHAR(255),
    password VARCHAR(255),
    created_at TIMESTAMP NOT NULL,
    UNIQUE (tenant_id, key)
);

CREATE TABLE IF NOT EXISTS email_events (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    campaign_id UUID NOT NULL,
    recipient_id UUID,
    kind VARCHAR(32) NOT NULL,
    unique_flag BOOLEAN NOT NULL DEFAULT FALSE,
    url VARCHAR(1024),
    meta JSONB,
    created_at TIMESTAMP NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_evt_camp ON email_events(tenant_id, campaign_id, kind);

CREATE TABLE IF NOT EXISTS suppressions (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    email VARCHAR(255) NOT NULL,
    reason VARCHAR(64) NOT NULL,
    source VARCHAR(64) NOT NULL,
    detail VARCHAR(512),
    created_at TIMESTAMP NOT NULL,
    UNIQUE (tenant_id, email)
);

CREATE TABLE IF NOT EXISTS bounce_records (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    email VARCHAR(255) NOT NULL,
    class VARCHAR(16) NOT NULL,
    code VARCHAR(16),
    enhanced VARCHAR(32),
    message VARCHAR(512),
    source VARCHAR(32) NOT NULL,
    created_at TIMESTAMP NOT NULL
);

CREATE TABLE IF NOT EXISTS import_jobs (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    list_id UUID,
    filename VARCHAR(255),
    status VARCHAR(32) NOT NULL,
    total INT NOT NULL DEFAULT 0,
    imported INT NOT NULL DEFAULT 0,
    updated INT NOT NULL DEFAULT 0,
    failed INT NOT NULL DEFAULT 0,
    error_csv TEXT,
    created_at TIMESTAMP NOT NULL
);

CREATE TABLE IF NOT EXISTS audit_logs (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    actor_id UUID,
    action VARCHAR(64) NOT NULL,
    target VARCHAR(255),
    detail VARCHAR(1024),
    created_at TIMESTAMP NOT NULL
);
