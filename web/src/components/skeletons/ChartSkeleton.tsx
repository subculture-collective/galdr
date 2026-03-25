export default function ChartSkeleton() {
  return (
    <div className="galdr-panel p-6">
      <div className="mb-4 h-4 w-40 animate-pulse rounded bg-[color-mix(in_srgb,var(--galdr-fg-muted)_35%,transparent)]" />
      <div className="h-64 animate-pulse rounded bg-[color-mix(in_srgb,var(--galdr-fg-muted)_30%,transparent)]" />
    </div>
  );
}
