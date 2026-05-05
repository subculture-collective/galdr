CREATE TABLE customer_features (
    id            UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    org_id        UUID NOT NULL REFERENCES organizations (id) ON DELETE CASCADE,
    customer_id   UUID NOT NULL REFERENCES customers (id) ON DELETE CASCADE,
    features      JSONB NOT NULL,
    calculated_at TIMESTAMPTZ NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE (customer_id)
);

CREATE INDEX idx_customer_features_customer ON customer_features (customer_id);
CREATE INDEX idx_customer_features_org_calculated ON customer_features (org_id, calculated_at DESC);

CREATE TRIGGER set_customer_features_updated_at
    BEFORE UPDATE ON customer_features
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_updated_at();
