CREATE TABLE usage_snapshots (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    metric VARCHAR(100) NOT NULL,
    value INTEGER NOT NULL CHECK (value >= 0),
    recorded_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    recorded_date DATE NOT NULL DEFAULT CURRENT_DATE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (org_id, metric, recorded_date)
);

CREATE INDEX idx_usage_snapshots_org_recorded_at ON usage_snapshots(org_id, recorded_at DESC);
CREATE INDEX idx_usage_snapshots_metric_recorded_at ON usage_snapshots(metric, recorded_at DESC);

CREATE TRIGGER update_usage_snapshots_updated_at
    BEFORE UPDATE ON usage_snapshots
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
