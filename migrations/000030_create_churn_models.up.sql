CREATE TABLE churn_models (
    id            UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    org_id        UUID NOT NULL REFERENCES organizations (id) ON DELETE CASCADE,
    version       TEXT NOT NULL,
    feature_names JSONB NOT NULL,
    weights       JSONB NOT NULL,
    bias          DOUBLE PRECISION NOT NULL,
    cutoff        DOUBLE PRECISION NOT NULL,
    metrics       JSONB NOT NULL,
    trained_at    TIMESTAMPTZ NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE (org_id, version)
);

CREATE INDEX idx_churn_models_org_trained ON churn_models (org_id, trained_at DESC);

CREATE TRIGGER set_churn_models_updated_at
    BEFORE UPDATE ON churn_models
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_updated_at();
