import WizardStep from "@/components/wizard/WizardStep";

export interface ScoreBucket {
  range: string;
  count: number;
}

export interface AtRiskCustomerPreview {
  id: string;
  name: string;
  health_score: number;
}

interface ScorePreviewStepProps {
  connectedProviders: string[];
  syncStatus: Record<string, string>;
  loading: boolean;
  distribution: ScoreBucket[];
  atRiskCustomers: AtRiskCustomerPreview[];
}

export default function ScorePreviewStep({
  connectedProviders,
  syncStatus,
  loading,
  distribution,
  atRiskCustomers,
}: ScorePreviewStepProps) {
  return (
    <WizardStep
      title="Generate first score preview"
      description="We’ll run an initial sync and show a quick snapshot of customer health."
    >
      {connectedProviders.length === 0 ? (
        <div className="galdr-alert-warning p-4 text-sm">
          No integrations connected yet. You can finish onboarding now and
          connect data sources later from Settings.
        </div>
      ) : (
        <>
          <div className="galdr-panel p-4">
            <h3 className="text-sm font-semibold text-[var(--galdr-fg)]">
              Sync status
            </h3>
            <ul className="mt-2 space-y-1 text-sm text-[var(--galdr-fg-muted)]">
              {connectedProviders.map((provider) => (
                <li
                  key={provider}
                  className="flex items-center justify-between"
                >
                  <span className="capitalize">{provider}</span>
                  <span className="font-medium">
                    {syncStatus[provider] ?? "pending"}
                  </span>
                </li>
              ))}
            </ul>
          </div>

          {loading && (
            <p className="mt-3 text-sm text-[var(--galdr-fg-muted)]">
              Fetching preview data... this can take a minute.
            </p>
          )}

          {!loading && distribution.length > 0 && (
            <div className="galdr-panel mt-5 p-4">
              <h4 className="text-sm font-semibold text-[var(--galdr-fg)]">
                Score distribution preview
              </h4>
              <div className="mt-3 space-y-2">
                {distribution.map((bucket) => (
                  <div key={bucket.range} className="flex items-center gap-3">
                    <span className="w-16 text-xs text-[var(--galdr-fg-muted)]">
                      {bucket.range}
                    </span>
                    <div className="h-2 flex-1 rounded bg-[var(--galdr-surface-soft)]">
                      <div
                        className="h-2 rounded bg-[var(--chart-series-primary)]"
                        style={{
                          width: `${Math.min(100, bucket.count * 12)}%`,
                        }}
                      />
                    </div>
                    <span className="w-8 text-right text-xs text-[var(--galdr-fg-muted)]">
                      {bucket.count}
                    </span>
                  </div>
                ))}
              </div>
            </div>
          )}

          {!loading && atRiskCustomers.length > 0 && (
            <div className="galdr-alert-danger mt-5 p-4">
              <h4 className="text-sm font-semibold">Top at-risk customers</h4>
              <ul className="mt-2 space-y-1 text-sm">
                {atRiskCustomers.map((customer) => (
                  <li
                    key={customer.id}
                    className="flex items-center justify-between"
                  >
                    <span>{customer.name}</span>
                    <span className="font-semibold">
                      {customer.health_score}
                    </span>
                  </li>
                ))}
              </ul>
            </div>
          )}
        </>
      )}
    </WizardStep>
  );
}
