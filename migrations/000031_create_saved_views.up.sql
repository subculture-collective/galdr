CREATE TABLE saved_views (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(120) NOT NULL,
    filters JSONB NOT NULL DEFAULT '{}'::jsonb,
    is_shared BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_saved_views_org_shared ON saved_views(org_id, is_shared);
CREATE INDEX idx_saved_views_org_user ON saved_views(org_id, user_id);
CREATE INDEX idx_saved_views_created_at ON saved_views(created_at DESC);

CREATE TRIGGER update_saved_views_updated_at
    BEFORE UPDATE ON saved_views
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_updated_at();
