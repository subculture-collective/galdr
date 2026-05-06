import { useEffect, useState } from "react";
import {
  Activity,
  AlertTriangle,
  CheckCircle2,
  Clock3,
  PlugZap,
  RefreshCcw,
  XCircle,
} from "lucide-react";
import {
  integrationsApi,
  type IntegrationHealthResponse,
  type IntegrationHealthStatus,
  type IntegrationHealthSummary,
} from "@/lib/api";

interface IntegrationHealthViewProps {
  health: IntegrationHealthResponse | null;
  loading: boolean;
  error?: string | null;
}

const healthStyles: Record<IntegrationHealthStatus, string> = {
  healthy: "border-emerald-500/30 bg-emerald-500/10 text-emerald-300",
  warning: "border-amber-500/30 bg-amber-500/10 text-amber-300",
  down: "border-red-500/30 bg-red-500/10 text-red-300",
  disconnected:
    "border-[var(--galdr-border)] bg-[var(--galdr-surface-subtle)] text-[var(--galdr-fg-muted)]",
};

const providerDisplayNames: Record<string, string> = {
  hubspot: "HubSpot",
  intercom: "Intercom",
  posthog: "PostHog",
  salesforce: "Salesforce",
  stripe: "Stripe",
  zendesk: "Zendesk",
};

function providerName(provider: string) {
  return (
    providerDisplayNames[provider] ??
    provider.charAt(0).toUpperCase() + provider.slice(1)
  );
}

function formatDate(value?: string | null) {
  if (!value) {
    return "Never";
  }
  return new Intl.DateTimeFormat("en", {
    month: "short",
    day: "numeric",
    hour: "numeric",
    minute: "2-digit",
  }).format(new Date(value));
}

function formatDuration(ms: number) {
  if (ms <= 0) {
    return "n/a";
  }
  if (ms < 1000) {
    return `${ms}ms`;
  }
  return `${(ms / 1000).toFixed(1)}s`;
}

function formatPercent(value: number) {
  return `${(value * 100).toFixed(1)}%`;
}

function formatSyncSuccessRate(integration: IntegrationHealthSummary) {
  const history = integration.sync_history;
  if (history.length === 0) {
    return "n/a";
  }
  const successes = history.filter(
    (point) => point.status === "success",
  ).length;
  return formatPercent(successes / history.length);
}

function statusIcon(status: IntegrationHealthStatus) {
  switch (status) {
    case "healthy":
      return <CheckCircle2 className="h-4 w-4" />;
    case "down":
      return <XCircle className="h-4 w-4" />;
    case "warning":
      return <AlertTriangle className="h-4 w-4" />;
    case "disconnected":
      return <PlugZap className="h-4 w-4" />;
  }

  return <PlugZap className="h-4 w-4" />;
}

function historyBarColor(status: string) {
  return status === "error" ? "bg-red-400" : "bg-emerald-400";
}

function historyBarWidth(recordsSynced: number, maxRecords: number) {
  return `${Math.max(8, (recordsSynced / maxRecords) * 100)}%`;
}

function getSummary(health: IntegrationHealthResponse | null) {
  const integrations = health?.integrations ?? [];
  return {
    total: integrations.length,
    healthy: integrations.filter((item) => item.health_status === "healthy")
      .length,
    warnings: integrations.filter((item) => item.health_status === "warning")
      .length,
    down: integrations.filter((item) => item.health_status === "down").length,
  };
}

function RecordsBars({
  integration,
}: {
  integration: IntegrationHealthSummary;
}) {
  const maxRecords = Math.max(
    1,
    ...integration.sync_history.map((point) => point.records_synced),
  );
  const label = `${providerName(integration.provider)} records synced per day`;

  return (
    <div className="space-y-2" aria-label={label}>
      {integration.sync_history.map((point) => (
        <div
          key={`${point.date}-${point.status}`}
          className="grid grid-cols-[6rem_1fr_4rem] items-center gap-3 text-xs"
        >
          <span className="text-[var(--galdr-fg-muted)]">{point.date}</span>
          <div className="h-2 overflow-hidden rounded-full bg-[var(--galdr-surface-subtle)]">
            <div
              className={`h-full rounded-full ${historyBarColor(point.status)}`}
              style={{
                width: historyBarWidth(point.records_synced, maxRecords),
              }}
            />
          </div>
          <span className="text-right text-[var(--galdr-fg)]">
            {point.records_synced.toLocaleString()}
          </span>
        </div>
      ))}
    </div>
  );
}

function IntegrationCard({
  integration,
}: {
  integration: IntegrationHealthSummary;
}) {
  const alerts = integration.alerts ?? [];
  return (
    <article className="galdr-card space-y-5 p-5">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <h2 className="text-lg font-semibold text-[var(--galdr-fg)]">
            {providerName(integration.provider)}
          </h2>
          <p className="text-sm text-[var(--galdr-fg-muted)]">
            Last sync {formatDate(integration.last_sync_at)}
          </p>
        </div>
        <span
          className={`inline-flex items-center gap-2 rounded-full border px-3 py-1 text-xs font-semibold ${healthStyles[integration.health_status]}`}
        >
          {statusIcon(integration.health_status)}
          {integration.health_status}
        </span>
      </div>

      <div className="grid grid-cols-2 gap-3 md:grid-cols-5">
        <Metric
          label="Records Synced"
          value={integration.records_synced.toLocaleString()}
        />
        <Metric
          label="Error Rate"
          value={formatPercent(integration.error_rate)}
        />
        <Metric
          label="Sync Success Rate"
          value={formatSyncSuccessRate(integration)}
        />
        <Metric label="Errors" value={integration.error_count.toString()} />
        <Metric
          label="Sync Duration"
          value={formatDuration(integration.sync_duration_ms)}
        />
      </div>

      {integration.last_sync_error && (
        <div className="rounded-lg border border-red-500/20 bg-red-500/10 px-3 py-2 text-sm text-red-200">
          {integration.last_sync_error}
        </div>
      )}

      {alerts.length > 0 && (
        <div className="space-y-2">
          <h3 className="text-xs font-semibold uppercase tracking-[0.18em] text-[var(--galdr-fg-muted)]">
            Alerts
          </h3>
          {alerts.map((alert) => (
            <div
              key={`${alert.type}-${alert.message}`}
              className="flex items-start gap-2 rounded-lg border border-amber-500/20 bg-amber-500/10 px-3 py-2 text-sm text-amber-100"
            >
              <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0" />
              <span>{alert.message}</span>
            </div>
          ))}
        </div>
      )}

      <details
        className="rounded-xl border border-[var(--galdr-border)] bg-[var(--galdr-surface-subtle)] p-4"
        open
      >
        <summary className="cursor-pointer text-sm font-semibold text-[var(--galdr-fg)]">
          Sync History
        </summary>
        <div className="mt-4 space-y-4">
          {integration.sync_history.length > 0 ? (
            <RecordsBars integration={integration} />
          ) : (
            <p className="text-sm text-[var(--galdr-fg-muted)]">
              No sync history recorded yet.
            </p>
          )}
          <div className="grid gap-2 text-xs text-[var(--galdr-fg-muted)] md:grid-cols-3">
            <span>Status: {integration.status}</span>
            <span>
              Customers: {integration.customer_count.toLocaleString()}
            </span>
            <span>Connected: {formatDate(integration.connected_at)}</span>
          </div>
        </div>
      </details>
    </article>
  );
}

function Metric({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-xl border border-[var(--galdr-border)] bg-[var(--galdr-surface-subtle)] p-3">
      <p className="text-xs text-[var(--galdr-fg-muted)]">{label}</p>
      <p className="mt-1 text-lg font-semibold text-[var(--galdr-fg)]">
        {value}
      </p>
    </div>
  );
}

export function IntegrationHealthView({
  health,
  loading,
  error,
}: IntegrationHealthViewProps) {
  const summary = getSummary(health);

  if (loading) {
    return (
      <div className="space-y-6">
        <div className="galdr-card h-32 animate-pulse" />
        <div className="grid gap-4 lg:grid-cols-2">
          <div className="galdr-card h-72 animate-pulse" />
          <div className="galdr-card h-72 animate-pulse" />
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <header className="galdr-card overflow-hidden p-6">
        <div className="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
          <div>
            <div className="mb-3 inline-flex items-center gap-2 rounded-full border border-cyan-400/20 bg-cyan-400/10 px-3 py-1 text-xs font-semibold uppercase tracking-[0.18em] text-cyan-200">
              <Activity className="h-3.5 w-3.5" />
              Integration Health
            </div>
            <h1 className="text-3xl font-bold tracking-tight text-[var(--galdr-fg)]">
              Integration Health Dashboard
            </h1>
            <p className="mt-2 max-w-2xl text-sm text-[var(--galdr-fg-muted)]">
              Monitor sync status, error rates, stale connections, and
              per-provider sync history in one place.
            </p>
          </div>
          <div className="flex items-center gap-2 rounded-xl border border-[var(--galdr-border)] bg-[var(--galdr-surface-subtle)] px-4 py-3 text-sm text-[var(--galdr-fg-muted)]">
            <Clock3 className="h-4 w-4" />
            Stale after {health?.stale_after_hours ?? 24}h
          </div>
        </div>
      </header>

      {error && (
        <div className="rounded-xl border border-red-500/30 bg-red-500/10 px-4 py-3 text-sm text-red-200">
          {error}
        </div>
      )}

      <section className="grid gap-4 md:grid-cols-4">
        <Metric label="All Integrations" value={summary.total.toString()} />
        <Metric label="Healthy" value={summary.healthy.toString()} />
        <Metric label="Warnings" value={summary.warnings.toString()} />
        <Metric label="Down" value={summary.down.toString()} />
      </section>

      <section className="grid gap-4 lg:grid-cols-2">
        {(health?.integrations ?? []).map((integration) => (
          <IntegrationCard
            key={integration.provider}
            integration={integration}
          />
        ))}
      </section>

      {health && health.integrations.length === 0 && (
        <div className="galdr-card flex flex-col items-center justify-center gap-3 p-10 text-center">
          <RefreshCcw className="h-8 w-8 text-[var(--galdr-fg-muted)]" />
          <p className="text-sm text-[var(--galdr-fg-muted)]">
            No integrations found yet. Connect a provider to start monitoring
            sync health.
          </p>
        </div>
      )}
    </div>
  );
}

export default function IntegrationHealthPage() {
  const [health, setHealth] = useState<IntegrationHealthResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    async function loadHealth() {
      try {
        const { data } = await integrationsApi.getHealth();
        if (!cancelled) setHealth(data);
      } catch {
        if (!cancelled) setError("Failed to load integration health.");
      } finally {
        if (!cancelled) setLoading(false);
      }
    }

    void loadHealth();
    return () => {
      cancelled = true;
    };
  }, []);

  return (
    <IntegrationHealthView health={health} loading={loading} error={error} />
  );
}
