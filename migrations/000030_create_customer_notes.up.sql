CREATE TABLE customer_notes (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    customer_id UUID NOT NULL REFERENCES customers (id) ON DELETE CASCADE,
    user_id     UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    content     TEXT NOT NULL CHECK (btrim(content) <> ''),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_customer_notes_customer_created ON customer_notes (customer_id, created_at DESC);
CREATE INDEX idx_customer_notes_user ON customer_notes (user_id);

CREATE TRIGGER set_customer_notes_updated_at
    BEFORE UPDATE ON customer_notes
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_updated_at();
