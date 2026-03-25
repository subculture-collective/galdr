export default function CardSkeleton() {
  return (
    <div className="galdr-panel p-6">
      <div className="mb-4 h-4 w-24 animate-pulse rounded bg-[color-mix(in_srgb,var(--galdr-fg-muted)_35%,transparent)]" />
      <div className="mb-2 h-8 w-32 animate-pulse rounded bg-[color-mix(in_srgb,var(--galdr-fg-muted)_32%,transparent)]" />
      <div className="h-4 w-20 animate-pulse rounded bg-[color-mix(in_srgb,var(--galdr-fg-muted)_35%,transparent)]" />
    </div>
  );
}
