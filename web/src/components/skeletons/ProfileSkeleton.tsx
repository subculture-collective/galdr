export default function ProfileSkeleton() {
  return (
    <div className="flex items-start gap-4">
      <div className="h-16 w-16 animate-pulse rounded-full bg-[color-mix(in_srgb,var(--galdr-fg-muted)_32%,transparent)]" />
      <div className="flex-1 space-y-3">
        <div className="h-5 w-48 animate-pulse rounded bg-[color-mix(in_srgb,var(--galdr-fg-muted)_32%,transparent)]" />
        <div className="h-4 w-32 animate-pulse rounded bg-[color-mix(in_srgb,var(--galdr-fg-muted)_32%,transparent)]" />
        <div className="h-4 w-64 animate-pulse rounded bg-[color-mix(in_srgb,var(--galdr-fg-muted)_32%,transparent)]" />
      </div>
    </div>
  );
}
