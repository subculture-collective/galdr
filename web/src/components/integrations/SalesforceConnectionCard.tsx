import { useCallback, useEffect, useState } from "react";
import { salesforceApi, type SalesforceStatus } from "@/lib/salesforce";

type SyncedResource = {
  label: string;
  count?: number;
};

export type { SalesforceStatus };

interface SalesforceConnectionCardViewProps {
  status: SalesforceStatus | null;
  loading: boolean;
  actionLoading: boolean;
  error: string;
  message: string;
  onConnect: () => void;
  onDisconnect: () => void;
  onSync: () => void;
}

function isConnectedStatus(status?: string) {
  return status === "active" || status === "syncing";
}

export default function SalesforceConnectionCard() {
  const [status, setStatus] = useState<SalesforceStatus | null>(null);
  const [loading, setLoading] = useState(true);
  const [actionLoading, setActionLoading] = useState(false);
  const [error, setError] = useState("");
  const [message, setMessage] = useState("");

  const fetchStatus = useCallback(async () => {
    try {
      const { data } = await salesforceApi.getStatus();
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
      const { data } = await salesforceApi.getConnectUrl();
      window.location.href = data.url;
    } catch {
      setError("Failed to start Salesforce connection.");
    } finally {
      setActionLoading(false);
    }
  }

  async function handleDisconnect() {
    if (!confirm("Are you sure you want to disconnect Salesforce?")) return;
    setActionLoading(true);
    setError("");
    try {
      await salesforceApi.disconnect();
      setStatus(null);
      setMessage("Salesforce disconnected.");
    } catch {
      setError("Failed to disconnect Salesforce.");
    } finally {
      setActionLoading(false);
    }
  }

  async function handleSync() {
    setActionLoading(true);
    setError("");
    setMessage("");
    try {
      await salesforceApi.triggerSync();
      setMessage("Sync started. This may take a few minutes.");
    } catch {
      setError("Failed to start sync.");
    } finally {
      setActionLoading(false);
    }
  }

  return (
    <SalesforceConnectionCardView
      status={status}
      loading={loading}
      actionLoading={actionLoading}
      error={error}
      message={message}
      onConnect={handleConnect}
      onDisconnect={handleDisconnect}
      onSync={handleSync}
    />
  );
}

export function SalesforceConnectionCardView({
  status,
  loading,
  actionLoading,
  error,
  message,
  onConnect,
  onDisconnect,
  onSync,
}: SalesforceConnectionCardViewProps) {
  if (loading) {
    return (
      <div className="galdr-card p-6">
        <p className="text-sm text-[var(--galdr-fg-muted)]">
          Loading Salesforce status...
        </p>
      </div>
    );
  }

  const isConnected = isConnectedStatus(status?.status);
  const syncedResources: SyncedResource[] = [
    { label: "Accounts synced", count: status?.account_count },
    { label: "Contacts synced", count: status?.contact_count },
    { label: "Opportunities synced", count: status?.opportunity_count },
  ];
  const showSyncDetails = status && (isConnected || status.last_sync_error);

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
                d="M2.25 12.76c0-1.6 1.34-2.9 2.99-2.9.34-1.94 2.08-3.42 4.18-3.42a4.3 4.3 0 013.65 2.03 3.5 3.5 0 015.06 3.13 3.1 3.1 0 01.37 6.17H5.33a3.1 3.1 0 01-3.08-3.1v-1.91z"
              />
            </svg>
          </div>
          <div>
            <h3 className="text-sm font-semibold text-[var(--galdr-fg)]">
              Salesforce
            </h3>
            <p className="text-sm text-[var(--galdr-fg-muted)]">
              CRM accounts, contacts, and opportunities
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

      {showSyncDetails && (
        <div className="galdr-panel mt-4 space-y-2 p-3 text-sm text-[var(--galdr-fg-muted)]">
          {status.external_account_id && (
            <p>
              Org ID:{" "}
              <span className="font-mono">{status.external_account_id}</span>
            </p>
          )}
          {status.last_sync_at && (
            <p>Last sync: {new Date(status.last_sync_at).toLocaleString()}</p>
          )}
          {syncedResources.map((resource) => (
            <SyncedResourceCount key={resource.label} {...resource} />
          ))}
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
            disabled={actionLoading}
            className="galdr-button-primary px-4 py-2 text-sm font-medium disabled:opacity-50"
          >
            {actionLoading ? "Connecting..." : "Connect Salesforce"}
          </button>
        )}
      </div>
    </div>
  );
}

function SyncedResourceCount({ label, count }: SyncedResource) {
  if (count === undefined) {
    return null;
  }

  return (
    <p>
      {label}: {count}
    </p>
  );
}

function StatusBadge({ status }: { status?: string }) {
  switch (status) {
    case undefined:
    case "disconnected":
      return (
        <span className="galdr-pill inline-flex items-center px-2.5 py-0.5 text-xs font-medium">
          Not connected
        </span>
      );
    case "active":
      return (
        <span className="inline-flex items-center rounded-full border border-[color:rgb(52_211_153_/_0.35)] bg-[color:rgb(52_211_153_/_0.14)] px-2.5 py-0.5 text-xs font-medium text-[var(--galdr-success)]">
          Connected
        </span>
      );
    case "syncing":
      return (
        <span className="inline-flex items-center rounded-full border border-[color:rgb(34_211_238_/_0.35)] bg-[color:rgb(34_211_238_/_0.14)] px-2.5 py-0.5 text-xs font-medium text-[var(--galdr-accent-2)]">
          Syncing
        </span>
      );
    case "error":
      return (
        <span className="inline-flex items-center rounded-full border border-[color:rgb(244_63_94_/_0.35)] bg-[color:rgb(244_63_94_/_0.14)] px-2.5 py-0.5 text-xs font-medium text-[var(--galdr-danger)]">
          Error
        </span>
      );
    default:
      return (
        <span className="inline-flex items-center rounded-full border border-[color:rgb(245_158_11_/_0.35)] bg-[color:rgb(245_158_11_/_0.14)] px-2.5 py-0.5 text-xs font-medium text-[var(--galdr-at-risk)]">
          {status}
        </span>
      );
  }
}
