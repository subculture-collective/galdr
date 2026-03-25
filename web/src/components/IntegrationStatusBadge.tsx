interface IntegrationStatusBadgeProps {
  status: "connected" | "syncing" | "error" | "disconnected";
}

const statusStyles = {
  connected: {
    dot: "bg-[var(--galdr-success)]",
    text: "text-[var(--galdr-success)]",
    label: "Connected",
  },
  syncing: {
    dot: "bg-[var(--galdr-accent-2)] animate-pulse",
    text: "text-[var(--galdr-accent-2)]",
    label: "Syncing",
  },
  error: {
    dot: "bg-[var(--galdr-danger)]",
    text: "text-[var(--galdr-danger)]",
    label: "Error",
  },
  disconnected: {
    dot: "bg-[var(--galdr-fg-muted)]",
    text: "text-[var(--galdr-fg-muted)]",
    label: "Disconnected",
  },
};

export default function IntegrationStatusBadge({
  status,
}: IntegrationStatusBadgeProps) {
  const s = statusStyles[status] ?? statusStyles.disconnected;

  return (
    <span className={`inline-flex items-center gap-1.5 text-sm ${s.text}`}>
      <span className={`h-2 w-2 rounded-full ${s.dot}`} />
      {s.label}
    </span>
  );
}
