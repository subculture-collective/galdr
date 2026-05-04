-- Create playbooks table
CREATE TABLE playbooks (
    id             UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    org_id         UUID NOT NULL REFERENCES organizations (id) ON DELETE CASCADE,
    name           VARCHAR(255) NOT NULL,
    description    TEXT,
    enabled        BOOLEAN NOT NULL DEFAULT true,
    trigger_type   VARCHAR(50) NOT NULL,
    trigger_config JSONB NOT NULL DEFAULT '{}',
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_playbooks_org_id ON playbooks (org_id);

CREATE TRIGGER set_playbooks_updated_at
    BEFORE UPDATE ON playbooks
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_updated_at();

-- Create playbook_actions table
CREATE TABLE playbook_actions (
    id            UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    playbook_id   UUID NOT NULL REFERENCES playbooks (id) ON DELETE CASCADE,
    action_type   VARCHAR(50) NOT NULL,
    action_config JSONB NOT NULL DEFAULT '{}',
    order_index   INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX idx_playbook_actions_playbook_id ON playbook_actions (playbook_id);

-- Create playbook_executions table
CREATE TABLE playbook_executions (
    id           UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    playbook_id  UUID NOT NULL REFERENCES playbooks (id) ON DELETE CASCADE,
    customer_id  UUID REFERENCES customers (id) ON DELETE SET NULL,
    triggered_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    -- status values must match PlaybookExecution* constants in internal/repository/playbook.go
    status       VARCHAR(20) NOT NULL CHECK (status IN ('pending', 'running', 'success', 'failed')),
    result       JSONB NOT NULL DEFAULT '{}'
);

CREATE INDEX idx_playbook_executions_playbook_id ON playbook_executions (playbook_id);
CREATE INDEX idx_playbook_executions_triggered_at ON playbook_executions (triggered_at DESC);
