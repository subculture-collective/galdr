import { useCallback, useEffect, useState } from "react";
import { hubspotApi, type HubSpotStatus } from "@/lib/hubspot";

export default function HubSpotConnectionCard() {
  const [status, setStatus] = useState<HubSpotStatus | null>(null);
  const [loading, setLoading] = useState(true);
  const [actionLoading, setActionLoading] = useState(false);
  const [error, setError] = useState("");
  const [message, setMessage] = useState("");

  const fetchStatus = useCallback(async () => {
    try {
      const { data } = await hubspotApi.getStatus();
      setStatus(data);
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
    setActionLoading(true);
    setError("");
    try {
      const { data } = await hubspotApi.getConnectUrl();
      window.location.href = data.url;
    } catch {
      setError("Failed to start HubSpot connection.");
    } finally {
      setActionLoading(false);
    }
  }

  async function handleDisconnect() {
    if (!confirm("Are you sure you want to disconnect HubSpot?")) return;
    setActionLoading(true);
    setError("");
    try {
      await hubspotApi.disconnect();
      setStatus(null);
      setMessage("HubSpot disconnected.");
    } catch {
      setError("Failed to disconnect HubSpot.");
    } finally {
      setActionLoading(false);
    }
  }

  async function handleSync() {
    setActionLoading(true);
    setError("");
    setMessage("");
    try {
      await hubspotApi.triggerSync();
      setMessage("Sync started. This may take a few minutes.");
    } catch {
      setError("Failed to start sync.");
    } finally {
      setActionLoading(false);
    }
  }

  if (loading) {
    return (
      <div className="galdr-card p-6">
        <p className="text-sm text-[var(--galdr-fg-muted)]">
          Loading HubSpot status...
        </p>
      </div>
    );
  }

  const isConnected =
    status?.status === "active" || status?.status === "syncing";

  return (
    <div className="galdr-card p-6">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <div className="flex h-10 w-10 items-center justify-center rounded-lg border border-[color:rgb(245_158_11_/_0.35)] bg-[color:rgb(245_158_11_/_0.14)]">
            <svg
              className="h-6 w-6 text-[var(--galdr-at-risk)]"
              fill="none"
              viewBox="0 0 24 24"
              strokeWidth={1.5}
              stroke="currentColor"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                d="M18 18.72a9.094 9.094 0 003.741-.479 3 3 0 00-4.682-2.72m.94 3.198l.001.031c0 .225-.012.447-.037.666A11.944 11.944 0 0112 21c-2.17 0-4.207-.576-5.963-1.584A6.062 6.062 0 016 18.719m12 0a5.971 5.971 0 00-.941-3.197m0 0A5.995 5.995 0 0012 12.75a5.995 5.995 0 00-5.058 2.772m0 0a3 3 0 00-4.681 2.72 8.986 8.986 0 003.74.477m.94-3.197a5.971 5.971 0 00-.94 3.197M15 6.75a3 3 0 11-6 0 3 3 0 016 0zm6 3a2.25 2.25 0 11-4.5 0 2.25 2.25 0 014.5 0zm-13.5 0a2.25 2.25 0 11-4.5 0 2.25 2.25 0 014.5 0z"
              />
            </svg>
          </div>
          <div>
            <h3 className="text-sm font-semibold text-[var(--galdr-fg)]">
              HubSpot
            </h3>
            <p className="text-sm text-[var(--galdr-fg-muted)]">
              CRM contacts, deals, and companies
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

      {isConnected && status && (
        <div className="galdr-panel mt-4 space-y-2 p-3 text-sm text-[var(--galdr-fg-muted)]">
          {status.external_account_id && (
            <p>
              Portal ID:{" "}
              <span className="font-mono">{status.external_account_id}</span>
            </p>
          )}
          {status.last_sync_at && (
            <p>Last sync: {new Date(status.last_sync_at).toLocaleString()}</p>
          )}
          {status.contact_count !== undefined && status.contact_count > 0 && (
            <p>Contacts synced: {status.contact_count}</p>
          )}
          {status.deal_count !== undefined && status.deal_count > 0 && (
            <p>Deals synced: {status.deal_count}</p>
          )}
          {status.company_count !== undefined && status.company_count > 0 && (
            <p>Companies synced: {status.company_count}</p>
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
              onClick={handleSync}
              disabled={actionLoading}
              className="galdr-button-primary px-4 py-2 text-sm font-medium disabled:opacity-50"
            >
              {actionLoading ? "..." : "Sync Now"}
            </button>
            <button
              onClick={handleDisconnect}
              disabled={actionLoading}
              className="galdr-button-danger-outline px-4 py-2 text-sm font-medium disabled:opacity-50"
            >
              Disconnect
            </button>
          </>
        ) : (
          <button
            onClick={handleConnect}
            disabled={actionLoading}
            className="galdr-button-primary px-4 py-2 text-sm font-medium disabled:opacity-50"
          >
            {actionLoading ? "Connecting..." : "Connect HubSpot"}
          </button>
        )}
      </div>
    </div>
  );
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
