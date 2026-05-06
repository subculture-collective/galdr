CREATE TABLE connector_metrics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    connector_id VARCHAR(100) NOT NULL,
    metric_date DATE NOT NULL,
    install_count INTEGER NOT NULL DEFAULT 0,
    active_installs INTEGER NOT NULL DEFAULT 0,
    sync_success_count INTEGER NOT NULL DEFAULT 0,
    sync_failure_count INTEGER NOT NULL DEFAULT 0,
    total_sync_duration_ms BIGINT NOT NULL DEFAULT 0,
    uninstall_count INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (connector_id, metric_date)
);

CREATE INDEX idx_connector_metrics_connector_date ON connector_metrics(connector_id, metric_date DESC);
