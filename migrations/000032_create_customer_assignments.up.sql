CREATE TABLE customer_assignments (
    customer_id UUID NOT NULL REFERENCES customers (id) ON DELETE CASCADE,
    user_id     UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    assigned_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    assigned_by UUID NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
    PRIMARY KEY (customer_id, user_id)
);

CREATE INDEX idx_customer_assignments_user ON customer_assignments (user_id, assigned_at DESC);
CREATE INDEX idx_customer_assignments_assigned_by ON customer_assignments (assigned_by);
