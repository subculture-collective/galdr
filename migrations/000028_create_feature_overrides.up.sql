CREATE TABLE feature_overrides (
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    feature_name VARCHAR(100) NOT NULL,
    enabled BOOLEAN,
    limit_override INTEGER CHECK (limit_override = -1 OR limit_override >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (org_id, feature_name),
    CHECK (enabled IS NOT NULL OR limit_override IS NOT NULL)
);

CREATE INDEX idx_feature_overrides_org_id ON feature_overrides(org_id);
