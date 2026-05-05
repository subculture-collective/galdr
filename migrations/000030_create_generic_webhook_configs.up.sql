CREATE TABLE generic_webhook_configs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    secret TEXT NOT NULL DEFAULT '',
    field_mapping JSONB NOT NULL DEFAULT '{}',
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_generic_webhook_configs_org ON generic_webhook_configs(org_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_generic_webhook_configs_active ON generic_webhook_configs(org_id, id) WHERE is_active = TRUE AND deleted_at IS NULL;
