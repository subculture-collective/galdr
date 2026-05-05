CREATE TABLE churn_predictions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    customer_id UUID NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    probability NUMERIC(5,4) NOT NULL CHECK (probability >= 0 AND probability <= 1),
    confidence NUMERIC(5,4) NOT NULL CHECK (confidence >= 0 AND confidence <= 1),
    risk_factors JSONB NOT NULL DEFAULT '[]'::jsonb,
    model_version TEXT NOT NULL,
    predicted_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (customer_id)
);

CREATE INDEX idx_churn_predictions_customer_id ON churn_predictions(customer_id);
CREATE INDEX idx_churn_predictions_org_probability ON churn_predictions(org_id, probability DESC);
CREATE INDEX idx_churn_predictions_predicted_at ON churn_predictions(predicted_at DESC);

CREATE TRIGGER set_churn_predictions_updated_at
    BEFORE UPDATE ON churn_predictions
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_updated_at();
