import { useCallback, useEffect, useState } from "react";
import { postHogApi, type PostHogStatus } from "@/lib/posthog";
import {
  getPostHogConnectionView,
  validatePostHogCredentials,
} from "@/lib/posthogConnectionView";

export type { PostHogStatus };

interface PostHogConnectionCardViewProps {
  status: PostHogStatus | null;
  apiKey: string;
  projectId: string;
  loading: boolean;
  actionLoading: boolean;
  error: string;
  message: string;
  onApiKeyChange: (apiKey: string) => void;
  onProjectIdChange: (projectId: string) => void;
  onSave: () => void;
  onDisconnect: () => void;
  onSync: () => void;
}

export default function PostHogConnectionCard() {
  const [status, setStatus] = useState<PostHogStatus | null>(null);
  const [apiKey, setApiKey] = useState("");
  const [projectId, setProjectId] = useState("");
  const [loading, setLoading] = useState(true);
  const [actionLoading, setActionLoading] = useState(false);
  const [error, setError] = useState("");
  const [message, setMessage] = useState("");

  const fetchStatus = useCallback(async () => {
    try {
      const { data } = await postHogApi.getStatus();
      setStatus(data);
      setProjectId(data.project_id ?? "");
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

  async function handleSave() {
    const validation = validatePostHogCredentials(apiKey, projectId);
    if (!validation.valid) {
      setError(validation.message);
      return;
    }

    setActionLoading(true);
    setError("");
    setMessage("");
    try {
      const { data } = await postHogApi.connect({
        api_key: apiKey.trim(),
        project_id: projectId.trim(),
      });
      setStatus(data);
      setApiKey("");
      setMessage("PostHog connected and API key validated.");
    } catch {
      setError("Failed to validate PostHog API key.");
    } finally {
      setActionLoading(false);
    }
  }

  async function handleDisconnect() {
    if (!confirm("Are you sure you want to disconnect PostHog?")) return;
    setActionLoading(true);
    setError("");
    setMessage("");
    try {
      await postHogApi.disconnect();
      setStatus(null);
      setProjectId("");
      setMessage("PostHog disconnected.");
    } catch {
      setError("Failed to disconnect PostHog.");
    } finally {
      setActionLoading(false);
    }
  }

  async function handleSync() {
    setActionLoading(true);
    setError("");
    setMessage("");
    try {
      await postHogApi.triggerSync();
      setStatus((current) =>
        current ? { ...current, status: "syncing" } : current,
      );
      setMessage("Sync started. This may take a few minutes.");
    } catch {
      setError("Failed to start sync.");
    } finally {
      setActionLoading(false);
    }
  }

  return (
    <PostHogConnectionCardView
      status={status}
      apiKey={apiKey}
      projectId={projectId}
      loading={loading}
      actionLoading={actionLoading}
      error={error}
      message={message}
      onApiKeyChange={setApiKey}
      onProjectIdChange={setProjectId}
      onSave={handleSave}
      onDisconnect={handleDisconnect}
      onSync={handleSync}
    />
  );
}

export function PostHogConnectionCardView({
  status,
  apiKey,
  projectId,
  loading,
  actionLoading,
  error,
  message,
  onApiKeyChange,
  onProjectIdChange,
  onSave,
  onDisconnect,
  onSync,
}: PostHogConnectionCardViewProps) {
  if (loading) {
    return (
      <div className="galdr-card p-6">
        <p className="text-sm text-[var(--galdr-fg-muted)]">
          Loading PostHog status...
        </p>
      </div>
    );
  }

  const view = getPostHogConnectionView(status);
  const showStatusDetails =
    view.isConnected || Boolean(status?.last_sync_error);

  return (
    <div className="galdr-card overflow-hidden p-6">
      <div className="flex items-start justify-between gap-4">
        <div className="flex items-center gap-3">
          <div className="flex h-10 w-10 items-center justify-center rounded-lg border border-[color:rgb(251_146_60_/_0.4)] bg-[color:rgb(251_146_60_/_0.14)]">
            <svg
              className="h-6 w-6 text-[color:rgb(251_146_60)]"
              fill="none"
              viewBox="0 0 24 24"
              strokeWidth={1.5}
              stroke="currentColor"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                d="M3.75 6.75h16.5m-16.5 5.25h16.5m-16.5 5.25h16.5M7.5 4.5v15m9-15v15"
              />
            </svg>
          </div>
          <div>
            <h3 className="text-sm font-semibold text-[var(--galdr-fg)]">
              PostHog
            </h3>
            <p className="text-sm text-[var(--galdr-fg-muted)]">
              Product events, users, and behavioral signals
            </p>
          </div>
        </div>

        <StatusBadge status={status?.status} label={view.badge} />
      </div>

      {error && (
        <div className="galdr-alert-danger mt-4 p-3 text-sm">{error}</div>
      )}
      {message && (
        <div className="galdr-alert-success mt-4 p-3 text-sm">{message}</div>
      )}

      {showStatusDetails && (
        <div className="galdr-panel mt-4 grid gap-2 p-3 text-sm text-[var(--galdr-fg-muted)] sm:grid-cols-2">
          {view.metrics.map((metric) => (
            <p key={metric}>{metric}</p>
          ))}
          {status?.last_sync_error && (
            <p className="text-[var(--galdr-danger)]">
              Last error: {status.last_sync_error}
            </p>
          )}
        </div>
      )}

      {!view.isConnected && (
        <div className="mt-5 grid gap-4 md:grid-cols-2">
          <label className="block text-sm font-medium text-[var(--galdr-fg-muted)]">
            API key
            <input
              type="password"
              value={apiKey}
              onChange={(event) => onApiKeyChange(event.target.value)}
              placeholder="phx_..."
              className="galdr-input mt-1 w-full px-3 py-2 text-sm"
              disabled={actionLoading}
            />
          </label>
          <label className="block text-sm font-medium text-[var(--galdr-fg-muted)]">
            Project ID
            <input
              type="text"
              value={projectId}
              onChange={(event) => onProjectIdChange(event.target.value)}
              placeholder="12345"
              className="galdr-input mt-1 w-full px-3 py-2 text-sm"
              disabled={actionLoading}
            />
          </label>
        </div>
      )}

      {!view.isConnected && (
        <p className="mt-3 text-xs text-[var(--galdr-fg-muted)]">
          Use a PostHog personal API key with project read access. PulseScore
          validates the key before saving it.
        </p>
      )}

      <div className="mt-6 flex flex-wrap gap-3">
        {view.isConnected ? (
          <>
            <button
              onClick={onSync}
              disabled={actionLoading || !view.canSync}
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
            onClick={onSave}
            disabled={actionLoading}
            className="galdr-button-primary px-4 py-2 text-sm font-medium disabled:opacity-50"
          >
            {actionLoading ? "Validating..." : "Save Connection"}
          </button>
        )}
      </div>
    </div>
  );
}

function StatusBadge({ status, label }: { status?: string; label: string }) {
  const className = postHogStatusBadgeClassName(status);

  return <span className={className}>{label}</span>;
}

function postHogStatusBadgeClassName(status?: string): string {
  const baseClassName =
    "inline-flex items-center rounded-full border px-2.5 py-0.5 text-xs font-medium";

  switch (status) {
    case "active":
      return `${baseClassName} border-[color:rgb(52_211_153_/_0.35)] bg-[color:rgb(52_211_153_/_0.14)] text-[var(--galdr-success)]`;
    case "syncing":
      return `${baseClassName} border-[color:rgb(34_211_238_/_0.35)] bg-[color:rgb(34_211_238_/_0.14)] text-[var(--galdr-accent-2)]`;
    case "error":
      return `${baseClassName} border-[color:rgb(244_63_94_/_0.35)] bg-[color:rgb(244_63_94_/_0.14)] text-[var(--galdr-danger)]`;
    case undefined:
    case "disconnected":
      return "galdr-pill inline-flex items-center px-2.5 py-0.5 text-xs font-medium";
    default:
      return `${baseClassName} border-[color:rgb(245_158_11_/_0.35)] bg-[color:rgb(245_158_11_/_0.14)] text-[var(--galdr-at-risk)]`;
  }
}
