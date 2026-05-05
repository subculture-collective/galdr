import React, { useCallback, useEffect, useState, type ChangeEvent } from "react";
import { zendeskApi, type ZendeskStatus } from "../../lib/zendesk";

void React;

export type { ZendeskStatus };

interface ZendeskConnectionCardViewProps {
  status: ZendeskStatus | null;
  subdomain: string;
  loading: boolean;
  actionLoading: boolean;
  error: string;
  message: string;
  onSubdomainChange: (subdomain: string) => void;
  onConnect: () => void;
  onDisconnect: () => void;
  onSync: () => void;
}

export default function ZendeskConnectionCard() {
  const [status, setStatus] = useState<ZendeskStatus | null>(null);
  const [subdomain, setSubdomain] = useState("");
  const [loading, setLoading] = useState(true);
  const [actionLoading, setActionLoading] = useState(false);
  const [error, setError] = useState("");
  const [message, setMessage] = useState("");

  const fetchStatus = useCallback(async () => {
    try {
      const { data } = await zendeskApi.getStatus();
      setStatus(data);
      setSubdomain(data.subdomain ?? data.external_account_id ?? "");
      setError("");
    } catch {
      setStatus(null);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchStatus();
  }, [fetchStatus]);

  async function handleConnect() {
    const normalizedSubdomain = normalizeSubdomain(subdomain);
    if (!normalizedSubdomain) {
      setError("Enter your Zendesk workspace subdomain before connecting.");
      return;
    }

    setActionLoading(true);
    setError("");
    try {
      const { data } = await zendeskApi.getConnectUrl(normalizedSubdomain);
      window.location.href = data.url;
    } catch {
      setError("Failed to start Zendesk connection.");
    } finally {
      setActionLoading(false);
    }
  }

  async function handleDisconnect() {
    if (!confirm("Are you sure you want to disconnect Zendesk?")) return;
    setActionLoading(true);
    setError("");
    try {
      await zendeskApi.disconnect();
      setStatus(null);
      setMessage("Zendesk disconnected.");
    } catch {
      setError("Failed to disconnect Zendesk.");
    } finally {
      setActionLoading(false);
    }
  }

  async function handleSync() {
    setActionLoading(true);
    setError("");
    setMessage("");
    try {
      await zendeskApi.triggerSync();
      setMessage("Sync started. This may take a few minutes.");
    } catch {
      setError("Failed to start sync.");
    } finally {
      setActionLoading(false);
    }
  }

  return (
    <ZendeskConnectionCardView
      status={status}
      subdomain={subdomain}
      loading={loading}
      actionLoading={actionLoading}
      error={error}
      message={message}
      onSubdomainChange={setSubdomain}
      onConnect={handleConnect}
      onDisconnect={handleDisconnect}
      onSync={handleSync}
    />
  );
}

export function ZendeskConnectionCardView({
  status,
  subdomain,
  loading,
  actionLoading,
  error,
  message,
  onSubdomainChange,
  onConnect,
  onDisconnect,
  onSync,
}: ZendeskConnectionCardViewProps) {
  if (loading) {
    return (
      <div className="galdr-card p-6">
        <p className="text-sm text-[var(--galdr-fg-muted)]">
          Loading Zendesk status...
        </p>
      </div>
    );
  }

  const isConnected =
    status?.status === "active" || status?.status === "syncing";
  const normalizedSubdomain = normalizeSubdomain(subdomain);
  const connectedSubdomain = normalizeSubdomain(
    status?.subdomain ?? status?.external_account_id ?? normalizedSubdomain,
  );

  return (
    <div className="galdr-card overflow-hidden p-6">
      <div className="flex items-center justify-between gap-4">
        <div className="flex items-center gap-3">
          <div className="relative flex h-10 w-10 items-center justify-center rounded-lg border border-[color:rgb(45_212_191_/_0.38)] bg-[radial-gradient(circle_at_30%_20%,rgb(45_212_191_/_0.25),rgb(15_23_42_/_0.02)_70%)]">
            <svg
              className="h-6 w-6 text-[color:rgb(45_212_191)]"
              fill="none"
              viewBox="0 0 24 24"
              strokeWidth={1.5}
              stroke="currentColor"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                d="M7.5 8.25h9m-9 3h5.25M21 12c0 4.556-4.03 8.25-9 8.25a9.77 9.77 0 01-3.208-.537L3 21l1.367-4.785A7.863 7.863 0 013 12c0-4.556 4.03-8.25 9-8.25s9 3.694 9 8.25z"
              />
            </svg>
          </div>
          <div>
            <h3 className="text-sm font-semibold text-[var(--galdr-fg)]">
              Zendesk
            </h3>
            <p className="text-sm text-[var(--galdr-fg-muted)]">
              Support tickets, users, and service health
            </p>
          </div>
        </div>

        <StatusBadge status={status?.status} />
      </div>

      {error && (
        <div className="galdr-alert-danger mt-4 p-3 text-sm">{error}</div>
      )}
      {message && (
        <div className="galdr-alert-success mt-4 p-3 text-sm">{message}</div>
      )}

      {!isConnected && (
        <label className="mt-5 block text-sm text-[var(--galdr-fg)]">
          <span className="font-medium">Workspace subdomain</span>
          <div className="mt-2 flex flex-col overflow-hidden rounded-lg border border-[var(--galdr-border)] bg-[var(--galdr-panel)] sm:flex-row sm:items-center">
            <input
              value={subdomain}
              onChange={(event: ChangeEvent<HTMLInputElement>) =>
                onSubdomainChange(event.target.value)
              }
              placeholder="acme"
              className="min-w-0 flex-1 bg-transparent px-3 py-2 text-sm text-[var(--galdr-fg)] outline-none placeholder:text-[var(--galdr-fg-muted)]"
              disabled={actionLoading}
            />
            <span className="border-t border-[var(--galdr-border)] px-3 py-2 font-mono text-xs text-[var(--galdr-fg-muted)] sm:border-l sm:border-t-0">
              .zendesk.com
            </span>
          </div>
          <span className="mt-2 block text-xs text-[var(--galdr-fg-muted)]">
            Use the prefix from your Zendesk URL, e.g. acme.zendesk.com.
          </span>
        </label>
      )}

      {(isConnected || status?.last_sync_error) && status && (
        <div className="galdr-panel mt-4 space-y-2 p-3 text-sm text-[var(--galdr-fg-muted)]">
          {connectedSubdomain && (
            <p>
              Subdomain: {" "}
              <span className="font-mono">{connectedSubdomain}.zendesk.com</span>
            </p>
          )}
          {status.last_sync_at && (
            <p>Last sync: {new Date(status.last_sync_at).toLocaleString()}</p>
          )}
          {status.ticket_count !== undefined && (
            <p>Tickets synced: {status.ticket_count}</p>
          )}
          {status.user_count !== undefined && (
            <p>Users synced: {status.user_count}</p>
          )}
          {status.last_sync_error && (
            <p className="text-[var(--galdr-danger)]">
              Last error: {status.last_sync_error}
            </p>
          )}
        </div>
      )}

      <div className="mt-6 flex gap-3">
        {isConnected ? (
          <>
            <button
              onClick={onSync}
              disabled={actionLoading}
              className="galdr-button-primary px-4 py-2 text-sm font-medium disabled:opacity-50"
            >
              {actionLoading ? "..." : "Sync Now"}
            </button>
            <button
              onClick={onDisconnect}
              disabled={actionLoading}
              className="galdr-button-danger-outline px-4 py-2 text-sm font-medium disabled:opacity-50"
            >
              Disconnect
            </button>
          </>
        ) : (
          <button
            onClick={onConnect}
            disabled={actionLoading || !normalizedSubdomain}
            className="galdr-button-primary px-4 py-2 text-sm font-medium disabled:opacity-50"
          >
            {actionLoading ? "Connecting..." : "Connect Zendesk"}
          </button>
        )}
      </div>
    </div>
  );
}

function normalizeSubdomain(value: string) {
  return value
    .trim()
    .toLowerCase()
    .replace(/^https?:\/\//, "")
    .replace(/\.zendesk\.com\/?$/, "")
    .replace(/\/$/, "");
}

function StatusBadge({ status }: { status?: string }) {
  if (!status || status === "disconnected") {
    return (
      <span className="galdr-pill inline-flex items-center px-2.5 py-0.5 text-xs font-medium">
        Not connected
      </span>
    );
  }
  if (status === "active") {
    return (
      <span className="inline-flex items-center rounded-full border border-[color:rgb(52_211_153_/_0.35)] bg-[color:rgb(52_211_153_/_0.14)] px-2.5 py-0.5 text-xs font-medium text-[var(--galdr-success)]">
        Connected
      </span>
    );
  }
  if (status === "syncing") {
    return (
      <span className="inline-flex items-center rounded-full border border-[color:rgb(34_211_238_/_0.35)] bg-[color:rgb(34_211_238_/_0.14)] px-2.5 py-0.5 text-xs font-medium text-[var(--galdr-accent-2)]">
        Syncing
      </span>
    );
  }
  if (status === "error") {
    return (
      <span className="inline-flex items-center rounded-full border border-[color:rgb(244_63_94_/_0.35)] bg-[color:rgb(244_63_94_/_0.14)] px-2.5 py-0.5 text-xs font-medium text-[var(--galdr-danger)]">
        Error
      </span>
    );
  }
  return (
    <span className="inline-flex items-center rounded-full border border-[color:rgb(245_158_11_/_0.35)] bg-[color:rgb(245_158_11_/_0.14)] px-2.5 py-0.5 text-xs font-medium text-[var(--galdr-at-risk)]">
      {status}
    </span>
  );
}
