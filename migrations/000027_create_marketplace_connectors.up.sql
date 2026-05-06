CREATE TABLE marketplace_connectors (
    id VARCHAR(100) NOT NULL,
    version VARCHAR(50) NOT NULL,
    developer_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT NOT NULL,
    manifest JSONB NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'submitted', 'approved', 'published', 'deprecated')),
    published_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (id, version)
);

CREATE INDEX idx_marketplace_connectors_status_name ON marketplace_connectors(status, name);
CREATE INDEX idx_marketplace_connectors_developer_id ON marketplace_connectors(developer_id);
CREATE INDEX idx_marketplace_connectors_manifest_categories ON marketplace_connectors USING GIN ((manifest->'categories'));
CREATE INDEX idx_marketplace_connectors_manifest_tags ON marketplace_connectors USING GIN ((manifest->'tags'));

CREATE TABLE connector_installations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    connector_id VARCHAR(100) NOT NULL,
    connector_version VARCHAR(50) NOT NULL,
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    config JSONB NOT NULL DEFAULT '{}',
    status VARCHAR(20) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'disabled', 'error', 'uninstalled')),
    installed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (org_id, connector_id),
    FOREIGN KEY (connector_id, connector_version) REFERENCES marketplace_connectors(id, version)
);

CREATE INDEX idx_connector_installations_org_id ON connector_installations(org_id);
CREATE INDEX idx_connector_installations_connector ON connector_installations(connector_id, connector_version);
