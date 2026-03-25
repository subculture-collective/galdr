export default function TableSkeleton({ rows = 5 }: { rows?: number }) {
  return (
    <div className="overflow-hidden rounded-lg border border-[var(--galdr-border)]">
      {/* Header */}
      <div className="flex gap-4 border-b border-[var(--galdr-border)] bg-[color-mix(in_srgb,var(--galdr-surface-soft)_82%,black_18%)] px-6 py-3">
        {[...Array(5)].map((_, i) => (
          <div
            key={i}
            className="h-4 flex-1 animate-pulse rounded bg-[color-mix(in_srgb,var(--galdr-fg-muted)_35%,transparent)]"
          />
        ))}
      </div>
      {/* Rows */}
      {[...Array(rows)].map((_, row) => (
        <div
          key={row}
          className="flex gap-4 border-b border-[var(--galdr-border)]/60 px-6 py-4 last:border-0"
        >
          {[...Array(5)].map((_, col) => (
            <div
              key={col}
              className="h-4 flex-1 animate-pulse rounded bg-[color-mix(in_srgb,var(--galdr-fg-muted)_30%,transparent)]"
            />
          ))}
        </div>
      ))}
    </div>
  );
}
