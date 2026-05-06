import { useEffect, useState } from "react";
import api from "@/lib/api";
import type { MarketplaceConnector } from "@/pages/MarketplacePage";

type QueueAction = "review" | "reject" | "publish";

interface ReviewQueuePageViewProps {
  connectors: MarketplaceConnector[];
  loading: boolean;
  error: string;
  actingId: string | null;
  onReview: (connector: MarketplaceConnector) => void;
  onReject: (connector: MarketplaceConnector) => void;
  onPublish: (connector: MarketplaceConnector) => void;
  onRetry: () => void;
}

export function ReviewQueuePageView({
  connectors,
  loading,
  error,
  actingId,
  onReview,
  onReject,
  onPublish,
  onRetry,
}: ReviewQueuePageViewProps) {
  return (
    <div className="mx-auto flex max-w-6xl flex-col gap-8 px-4 py-8 sm:px-6 lg:px-8">
      <section className="rounded-[2rem] border border-[var(--galdr-border)] bg-[var(--galdr-surface)] p-8 shadow-sm">
        <p className="text-sm font-semibold uppercase tracking-[0.24em] text-[var(--galdr-accent)]">
          Admin review
        </p>
        <h1 className="mt-3 text-3xl font-semibold text-[var(--galdr-fg)]">
          Connector submission queue
        </h1>
        <p className="mt-3 max-w-2xl text-sm leading-6 text-[var(--galdr-fg-muted)]">
          Review pending community connectors, approve automated checks, reject
          unsafe submissions, or publish approved versions to the marketplace.
        </p>
      </section>

      {error ? (
        <div className="rounded-2xl border border-red-200 bg-red-50 p-4 text-sm text-red-700">
          {error}
          <button className="ml-3 font-semibold underline" onClick={onRetry}>
            Retry
          </button>
        </div>
      ) : null}

      <section className="overflow-hidden rounded-[2rem] border border-[var(--galdr-border)] bg-[var(--galdr-surface)] shadow-sm">
        {loading ? (
          <div className="p-8 text-sm text-[var(--galdr-fg-muted)]">
            Loading connector submissions...
          </div>
        ) : connectors.length === 0 ? (
          <div className="p-8 text-sm text-[var(--galdr-fg-muted)]">
            No pending connector submissions.
          </div>
        ) : (
          <div className="divide-y divide-[var(--galdr-border)]">
            {connectors.map((connector) => {
              const actionKey = `${connector.id}@${connector.version}`;
              const acting = actingId === actionKey;
              const canPublish = connector.status === "approved";
              return (
                <article
                  className="grid gap-4 p-6 lg:grid-cols-[1fr_auto] lg:items-center"
                  key={actionKey}
                >
                  <div>
                    <div className="flex flex-wrap items-center gap-3">
                      <h2 className="text-lg font-semibold text-[var(--galdr-fg)]">
                        {connector.name}
                      </h2>
                      <span className="rounded-full bg-[var(--galdr-bg)] px-3 py-1 text-xs font-medium text-[var(--galdr-fg-muted)]">
                        {connector.status.replace("_", " ")}
                      </span>
                      <span className="text-xs text-[var(--galdr-fg-muted)]">
                        v{connector.version}
                      </span>
                    </div>
                    <p className="mt-2 text-sm leading-6 text-[var(--galdr-fg-muted)]">
                      {connector.description}
                    </p>
                    <p className="mt-2 text-xs text-[var(--galdr-fg-muted)]">
                      Auth: {connector.manifest.auth.type} · Resources:{" "}
                      {connector.manifest.sync.resources
                        .map((resource) => resource.name)
                        .join(", ")}
                    </p>
                  </div>
                  <div className="flex flex-wrap gap-2">
                    <button
                      className="rounded-full border border-[var(--galdr-border)] px-4 py-2 text-sm font-semibold text-[var(--galdr-fg)] disabled:opacity-50"
                      disabled={acting}
                      onClick={() => onReview(connector)}
                    >
                      Run review
                    </button>
                    <button
                      className="rounded-full border border-red-200 px-4 py-2 text-sm font-semibold text-red-700 disabled:opacity-50"
                      disabled={acting}
                      onClick={() => onReject(connector)}
                    >
                      Reject
                    </button>
                    <button
                      className="rounded-full bg-[var(--galdr-accent)] px-4 py-2 text-sm font-semibold text-white disabled:opacity-50"
                      disabled={acting || !canPublish}
                      onClick={() => onPublish(connector)}
                    >
                      Publish
                    </button>
                  </div>
                </article>
              );
            })}
          </div>
        )}
      </section>
    </div>
  );
}

export default function ReviewQueuePage() {
  const [connectors, setConnectors] = useState<MarketplaceConnector[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [actingId, setActingId] = useState<string | null>(null);

  async function fetchQueue() {
    setLoading(true);
    setError("");
    try {
      const response = await api.get<{ connectors: MarketplaceConnector[] }>(
        "/marketplace/connectors/review-queue",
      );
      setConnectors(response.data.connectors ?? []);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load queue.");
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    void fetchQueue();
  }, []);

  async function act(connector: MarketplaceConnector, action: QueueAction) {
    const actionKey = `${connector.id}@${connector.version}`;
    setActingId(actionKey);
    setError("");
    try {
      if (action === "review") {
        await api.post(
          `/marketplace/connectors/${connector.id}/versions/${connector.version}/review`,
          {
            checklist: {
              data_access_justified: true,
              error_handling_ready: true,
              documentation_ready: true,
            },
          },
        );
      } else {
        await api.post(
          `/marketplace/connectors/${connector.id}/versions/${connector.version}/${action}`,
        );
      }
      await fetchQueue();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Queue action failed.");
    } finally {
      setActingId(null);
    }
  }

  return (
    <ReviewQueuePageView
      connectors={connectors}
      loading={loading}
      error={error}
      actingId={actingId}
      onReview={(connector) => void act(connector, "review")}
      onReject={(connector) => void act(connector, "reject")}
      onPublish={(connector) => void act(connector, "publish")}
      onRetry={() => void fetchQueue()}
    />
  );
}
