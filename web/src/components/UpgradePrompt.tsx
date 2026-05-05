import React from "react";
import { LockKeyhole } from "lucide-react";

void React;

interface UpgradePromptProps {
  featureName: string;
  recommendedTier?: string | null;
  description?: string;
  ctaHref?: string;
  className?: string;
}

function formatTier(tier?: string | null) {
  if (!tier) return "Growth";
  return tier.charAt(0).toUpperCase() + tier.slice(1);
}

export default function UpgradePrompt({
  featureName,
  recommendedTier = "growth",
  description,
  ctaHref,
  className = "",
}: UpgradePromptProps) {
  const tierLabel = formatTier(recommendedTier);
  const href = ctaHref ?? `/pricing?tier=${recommendedTier ?? "growth"}`;

  return (
    <div
      role="region"
      aria-label={`${featureName} upgrade prompt`}
      className={`galdr-card overflow-hidden border border-[color:rgb(139_92_246_/_0.35)] bg-[radial-gradient(circle_at_top_left,rgb(139_92_246_/_0.18),transparent_36%),var(--galdr-surface)] p-6 ${className}`}
    >
      <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div className="flex gap-3">
          <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-full bg-[color:rgb(139_92_246_/_0.18)] text-[var(--galdr-accent)]">
            <LockKeyhole className="h-5 w-5" aria-hidden="true" />
          </div>
          <div>
            <h3 className="text-sm font-semibold text-[var(--galdr-fg)]">
              Upgrade to {tierLabel} to access {featureName}
            </h3>
            {description && (
              <p className="mt-1 max-w-2xl text-sm text-[var(--galdr-fg-muted)]">
                {description}
              </p>
            )}
          </div>
        </div>
        <a href={href} className="galdr-button-primary shrink-0 text-sm">
          View plans
        </a>
      </div>
    </div>
  );
}
