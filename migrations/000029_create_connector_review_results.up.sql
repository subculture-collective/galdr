CREATE TABLE connector_review_results (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    connector_id VARCHAR(100) NOT NULL,
    connector_version VARCHAR(50) NOT NULL,
    reviewer_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status VARCHAR(20) NOT NULL CHECK (status IN ('approved', 'blocked')),
    automated_checks JSONB NOT NULL DEFAULT '[]',
    security_checklist JSONB NOT NULL DEFAULT '{}',
    sandbox_checks JSONB NOT NULL DEFAULT '[]',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    FOREIGN KEY (connector_id, connector_version) REFERENCES marketplace_connectors(id, version)
);

CREATE INDEX idx_connector_review_results_connector ON connector_review_results(connector_id, connector_version);
CREATE INDEX idx_connector_review_results_status ON connector_review_results(status);
