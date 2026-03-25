import IntegrationStatusBadge from "@/components/IntegrationStatusBadge";
import { RefreshCw, Unplug } from "lucide-react";
import { relativeTime } from "@/lib/format";

interface IntegrationCardProps {
  provider: string;
  status: "connected" | "syncing" | "error" | "disconnected";
  lastSyncAt?: string;
  customerCount?: number;
  onSync?: () => void;
  onDisconnect?: () => void;
}

export default function IntegrationCard({
  provider,
  status,
  lastSyncAt,
  customerCount,
  onSync,
  onDisconnect,
}: IntegrationCardProps) {
  const isActive = status === "connected" || status === "syncing";

  return (
    <div className="galdr-card p-6">
      <div className="flex items-center justify-between">
        <div>
          <h3 className="text-sm font-semibold capitalize text-[var(--galdr-fg)]">
            {provider}
          </h3>
          <IntegrationStatusBadge status={status} />
        </div>
      </div>

      {isActive && (
        <div className="mt-4 space-y-1 text-xs text-[var(--galdr-fg-muted)]">
          <p>Last sync: {relativeTime(lastSyncAt)}</p>
          {customerCount !== undefined && (
            <p>
              {customerCount} customer{customerCount !== 1 ? "s" : ""}
            </p>
          )}
        </div>
      )}

      {isActive && (
        <div className="mt-4 flex gap-2">
          {onSync && (
            <button
              onClick={onSync}
              disabled={status === "syncing"}
              className="galdr-button-secondary inline-flex items-center gap-1 px-3 py-1.5 text-xs font-medium disabled:opacity-50"
            >
              <RefreshCw className="h-3 w-3" />
              Sync Now
            </button>
          )}
          {onDisconnect && (
            <button
              onClick={onDisconnect}
              className="galdr-button-danger-outline inline-flex items-center gap-1 px-3 py-1.5 text-xs font-medium"
            >
              <Unplug className="h-3 w-3" />
              Disconnect
            </button>
          )}
        </div>
      )}
    </div>
  );
}
