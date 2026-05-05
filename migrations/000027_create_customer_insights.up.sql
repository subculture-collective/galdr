CREATE TABLE IF NOT EXISTS customer_insights (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    customer_id UUID NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    insight_type VARCHAR(100) NOT NULL,
    content JSONB NOT NULL DEFAULT '{}',
    generated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    model VARCHAR(100) NOT NULL DEFAULT '',
    token_cost NUMERIC(12, 6) NOT NULL DEFAULT 0 CHECK (token_cost >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_customer_insights_customer_recent
    ON customer_insights (org_id, customer_id, insight_type, generated_at DESC);

CREATE INDEX IF NOT EXISTS idx_customer_insights_org_recent
    ON customer_insights (org_id, generated_at DESC);
