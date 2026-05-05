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
  const trimmedApiKey = apiKey.trim();
  const trimmedProjectId = projectId.trim();

  if (!trimmedApiKey) {
    return { valid: false, message: "Enter a PostHog API key." };
  }
  if (!trimmedApiKey.startsWith("phx_")) {
    return {
      valid: false,
      message: "Enter a valid PostHog personal API key.",
    };
  }
  if (!trimmedProjectId) {
    return { valid: false, message: "Enter a PostHog project ID." };
  }
  return { valid: true, message: "" };
}

export function getPostHogConnectionView(
  status: PostHogConnectionStatus | null,
): PostHogConnectionView {
  const statusValue = status?.status ?? "disconnected";
  const isConnected = statusValue === "active" || statusValue === "syncing";

  return {
    badge: badgeLabel(statusValue),
    isConnected,
    canSync: statusValue === "active",
    metrics: isConnected && status ? connectionMetrics(status) : [],
  };
}

function badgeLabel(status: string): string {
  switch (status) {
    case "active":
      return "Connected";
    case "syncing":
      return "Syncing";
    case "error":
      return "Error";
    default:
      return "Not connected";
  }
}

function connectionMetrics(status: PostHogConnectionStatus): string[] {
  const metrics: string[] = [];

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

  return metrics;
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
