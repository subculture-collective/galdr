export interface ValidationResult {
  valid: boolean;
  message: string;
}

export interface PostHogConnectionView {
  badge: string;
  isConnected: boolean;
  canSync: boolean;
  metrics: string[];
}

export interface PostHogConnectionStatus {
  status: string;
  project_id?: string;
  last_sync_at?: string;
  event_count?: number;
  user_count?: number;
}

export function validatePostHogCredentials(
  apiKey: string,
  projectId: string,
): ValidationResult {
  if (!apiKey.trim()) {
    return { valid: false, message: "Enter a PostHog API key." };
  }
  if (!apiKey.trim().startsWith("phx_")) {
    return {
      valid: false,
      message: "Enter a valid PostHog personal API key.",
    };
  }
  if (!projectId.trim()) {
    return { valid: false, message: "Enter a PostHog project ID." };
  }
  return { valid: true, message: "" };
}

export function getPostHogConnectionView(
  status: PostHogConnectionStatus | null,
): PostHogConnectionView {
  const statusValue = status?.status ?? "disconnected";
  const isConnected = statusValue === "active" || statusValue === "syncing";
  const metrics: string[] = [];

  if (isConnected && status) {
    if (status.project_id) metrics.push(`Project ID: ${status.project_id}`);
    if (status.event_count !== undefined) {
      metrics.push(`Events synced: ${formatCount(status.event_count)}`);
    }
    if (status.user_count !== undefined) {
      metrics.push(`Users synced: ${formatCount(status.user_count)}`);
    }
    if (status.last_sync_at) {
      metrics.push(`Last sync: ${formatSyncTime(status.last_sync_at)}`);
    }
  }

  return {
    badge: badgeLabel(statusValue),
    isConnected,
    canSync: statusValue === "active",
    metrics,
  };
}

function badgeLabel(status: string): string {
  if (status === "active") return "Connected";
  if (status === "syncing") return "Syncing";
  if (status === "error") return "Error";
  return "Not connected";
}

function formatCount(value: number): string {
  return new Intl.NumberFormat("en-US").format(value);
}

function formatSyncTime(value: string): string {
  return new Intl.DateTimeFormat("en-US", {
    dateStyle: "medium",
    timeStyle: "short",
    timeZone: "UTC",
  }).format(new Date(value));
}
