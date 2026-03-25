import { useCallback, useEffect, useState } from "react";
import { intercomApi, type IntercomStatus } from "@/lib/intercom";

export default function IntercomConnectionCard() {
  const [status, setStatus] = useState<IntercomStatus | null>(null);
  const [loading, setLoading] = useState(true);
  const [actionLoading, setActionLoading] = useState(false);
  const [error, setError] = useState("");
  const [message, setMessage] = useState("");

  const fetchStatus = useCallback(async () => {
    try {
      const { data } = await intercomApi.getStatus();
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
      const { data } = await intercomApi.getConnectUrl();
      window.location.href = data.url;
    } catch {
      setError("Failed to start Intercom connection.");
    } finally {
      setActionLoading(false);
    }
  }

  async function handleDisconnect() {
    if (!confirm("Are you sure you want to disconnect Intercom?")) return;
    setActionLoading(true);
    setError("");
    try {
      await intercomApi.disconnect();
      setStatus(null);
      setMessage("Intercom disconnected.");
    } catch {
      setError("Failed to disconnect Intercom.");
    } finally {
      setActionLoading(false);
    }
  }

  async function handleSync() {
    setActionLoading(true);
    setError("");
    setMessage("");
    try {
      await intercomApi.triggerSync();
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
          Loading Intercom status...
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
          <div className="flex h-10 w-10 items-center justify-center rounded-lg border border-[color:rgb(34_211_238_/_0.35)] bg-[color:rgb(34_211_238_/_0.14)]">
            <svg
              className="h-6 w-6 text-[var(--galdr-accent-2)]"
              fill="none"
              viewBox="0 0 24 24"
              strokeWidth={1.5}
              stroke="currentColor"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                d="M20.25 8.511c.884.284 1.5 1.128 1.5 2.097v4.286c0 1.136-.847 2.1-1.98 2.193-.34.027-.68.052-1.02.072v3.091l-3-3c-1.354 0-2.694-.055-4.02-.163a2.115 2.115 0 01-.825-.242m9.345-8.334a2.126 2.126 0 00-.476-.095 48.64 48.64 0 00-8.048 0c-1.131.094-1.976 1.057-1.976 2.192v4.286c0 .837.46 1.58 1.155 1.951m9.345-8.334V6.637c0-1.621-1.152-3.026-2.76-3.235A48.455 48.455 0 0011.25 3c-2.115 0-4.198.137-6.24.402-1.608.209-2.76 1.614-2.76 3.235v6.226c0 1.621 1.152 3.026 2.76 3.235.577.075 1.157.14 1.74.194V21l4.155-4.155"
              />
            </svg>
          </div>
          <div>
            <h3 className="text-sm font-semibold text-[var(--galdr-fg)]">
              Intercom
            </h3>
            <p className="text-sm text-[var(--galdr-fg-muted)]">
              Conversations, contacts, and support metrics
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
              App ID:{" "}
              <span className="font-mono">{status.external_account_id}</span>
            </p>
          )}
          {status.last_sync_at && (
            <p>Last sync: {new Date(status.last_sync_at).toLocaleString()}</p>
          )}
          {status.conversation_count !== undefined &&
            status.conversation_count > 0 && (
              <p>Conversations synced: {status.conversation_count}</p>
            )}
          {status.contact_count !== undefined && status.contact_count > 0 && (
            <p>Contacts synced: {status.contact_count}</p>
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
            {actionLoading ? "Connecting..." : "Connect Intercom"}
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
