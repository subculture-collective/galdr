CREATE TABLE IF NOT EXISTS zendesk_users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    customer_id UUID REFERENCES customers(id) ON DELETE SET NULL,
    zendesk_user_id TEXT NOT NULL,
    email TEXT NOT NULL,
    name TEXT,
    role TEXT,
    metadata JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (org_id, zendesk_user_id)
);

CREATE INDEX IF NOT EXISTS idx_zendesk_users_org_id ON zendesk_users(org_id);
CREATE INDEX IF NOT EXISTS idx_zendesk_users_customer_id ON zendesk_users(customer_id);
CREATE INDEX IF NOT EXISTS idx_zendesk_users_email ON zendesk_users(org_id, email);

CREATE TABLE IF NOT EXISTS zendesk_tickets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    customer_id UUID REFERENCES customers(id) ON DELETE SET NULL,
    zendesk_ticket_id TEXT NOT NULL,
    zendesk_user_id TEXT,
    subject TEXT,
    status TEXT NOT NULL,
    priority TEXT,
    type TEXT,
    created_at_remote TIMESTAMPTZ,
    updated_at_remote TIMESTAMPTZ,
    solved_at TIMESTAMPTZ,
    metadata JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (org_id, zendesk_ticket_id)
);

CREATE INDEX IF NOT EXISTS idx_zendesk_tickets_org_id ON zendesk_tickets(org_id);
CREATE INDEX IF NOT EXISTS idx_zendesk_tickets_customer_id ON zendesk_tickets(customer_id);
CREATE INDEX IF NOT EXISTS idx_zendesk_tickets_user_id ON zendesk_tickets(org_id, zendesk_user_id);
CREATE INDEX IF NOT EXISTS idx_zendesk_tickets_status ON zendesk_tickets(org_id, status);
CREATE INDEX IF NOT EXISTS idx_zendesk_tickets_updated_at ON zendesk_tickets(updated_at_remote);
