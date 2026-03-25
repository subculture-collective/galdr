import { useCallback, useEffect, useState } from "react";
import api from "@/lib/api";
import EmptyState from "@/components/EmptyState";
import {
  CheckCircle,
  XCircle,
  PlusCircle,
  Flag,
  Circle,
  Loader2,
} from "lucide-react";
import { relativeTime } from "@/lib/format";

interface CustomerEvent {
  id: string;
  type: string;
  title: string;
  source: string;
  timestamp: string;
  data?: Record<string, unknown>;
}

interface EventsResponse {
  events: CustomerEvent[];
  total: number;
  page: number;
  per_page: number;
}

const eventIcons: Record<string, { icon: typeof CheckCircle; color: string }> =
  {
    "payment.success": {
      icon: CheckCircle,
      color: "text-[var(--galdr-success)]",
    },
    "payment.succeeded": {
      icon: CheckCircle,
      color: "text-[var(--galdr-success)]",
    },
    "payment.failed": { icon: XCircle, color: "text-[var(--galdr-danger)]" },
    "subscription.created": {
      icon: PlusCircle,
      color: "text-[var(--galdr-accent-2)]",
    },
    "subscription.updated": {
      icon: PlusCircle,
      color: "text-[var(--galdr-accent-2)]",
    },
    "ticket.opened": { icon: Flag, color: "text-[var(--galdr-at-risk)]" },
  };

export default function EventTimeline({ customerId }: { customerId: string }) {
  const [events, setEvents] = useState<CustomerEvent[]>([]);
  const [loading, setLoading] = useState(true);
  const [loadingMore, setLoadingMore] = useState(false);
  const [page, setPage] = useState(1);
  const [hasMore, setHasMore] = useState(false);
  const [expandedId, setExpandedId] = useState<string | null>(null);

  const fetchEvents = useCallback(
    async (p: number, append: boolean) => {
      if (append) setLoadingMore(true);
      else setLoading(true);
      try {
        const { data } = await api.get<EventsResponse>(
          `/customers/${customerId}/events`,
          { params: { page: p, per_page: 20 } },
        );
        const evts = data.events ?? [];
        setEvents((prev) => (append ? [...prev, ...evts] : evts));
        setHasMore(p < Math.ceil(data.total / data.per_page));
      } catch {
        // silent
      } finally {
        setLoading(false);
        setLoadingMore(false);
      }
    },
    [customerId],
  );

  useEffect(() => {
    fetchEvents(1, false);
  }, [fetchEvents]);

  function loadMore() {
    const nextPage = page + 1;
    setPage(nextPage);
    fetchEvents(nextPage, true);
  }

  if (loading) {
    return (
      <div className="flex justify-center py-8">
        <Loader2 className="h-6 w-6 animate-spin text-[var(--galdr-fg-muted)]" />
      </div>
    );
  }

  if (events.length === 0) {
    return (
      <EmptyState
        title="No events recorded yet"
        description="Customer events will appear here once activity is tracked."
      />
    );
  }

  return (
    <div className="relative">
      {/* Timeline line */}
      <div className="absolute bottom-0 left-4 top-0 w-0.5 bg-[color:rgb(45_45_64_/_0.9)]" />

      <div className="space-y-4">
        {events.map((event) => {
          const config = eventIcons[event.type] ?? {
            icon: Circle,
            color: "text-[var(--galdr-fg-muted)]",
          };
          const Icon = config.icon;
          const expanded = expandedId === event.id;

          return (
            <div key={event.id} className="relative ml-10">
              {/* Dot */}
              <div
                className={`absolute -left-10 top-1 flex h-8 w-8 items-center justify-center rounded-full border border-[var(--galdr-border)] bg-[var(--galdr-bg-elevated)] ${config.color}`}
              >
                <Icon className="h-4 w-4" />
              </div>

              {/* Content */}
              <button
                onClick={() => setExpandedId(expanded ? null : event.id)}
                className="galdr-panel w-full p-4 text-left transition-colors hover:bg-[color:rgb(139_92_246_/_0.08)]"
              >
                <div className="flex items-center justify-between">
                  <span className="text-sm font-medium text-[var(--galdr-fg)]">
                    {event.title || event.type}
                  </span>
                  <span className="flex items-center gap-2 text-xs text-[var(--galdr-fg-muted)]">
                    <span className="galdr-pill px-2 py-0.5 text-[10px] uppercase tracking-[0.06em]">
                      {event.source}
                    </span>
                    {relativeTime(event.timestamp)}
                  </span>
                </div>
                {expanded && event.data && (
                  <pre className="mt-3 overflow-x-auto rounded-md border border-[var(--galdr-border)] bg-[var(--galdr-bg-elevated)] p-3 text-xs text-[var(--galdr-fg-muted)]">
                    {JSON.stringify(event.data, null, 2)}
                  </pre>
                )}
              </button>
            </div>
          );
        })}
      </div>

      {hasMore && (
        <div className="ml-10 mt-4">
          <button
            onClick={loadMore}
            disabled={loadingMore}
            className="galdr-link text-sm font-medium disabled:opacity-50"
          >
            {loadingMore ? "Loading..." : "Load more"}
          </button>
        </div>
      )}
    </div>
  );
}
