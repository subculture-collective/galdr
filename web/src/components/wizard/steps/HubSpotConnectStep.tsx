import WizardStep from "@/components/wizard/WizardStep";

interface HubSpotConnectStepProps {
  connected: boolean;
  loading: boolean;
  statusText: string;
  accountId?: string;
  error?: string;
  onConnect: () => void;
  onRefresh: () => void;
}

export default function HubSpotConnectStep({
  connected,
  loading,
  statusText,
  accountId,
  error,
  onConnect,
  onRefresh,
}: HubSpotConnectStepProps) {
  return (
    <WizardStep
      title="Connect HubSpot (optional)"
      description="HubSpot data enriches customer context with contacts, deals, and company attributes."
    >
      <div className="galdr-panel p-4">
        <div className="flex items-center justify-between gap-3">
          <div>
            <p className="text-sm font-medium text-[var(--galdr-fg)]">
              Status: {statusText}
            </p>
            {accountId && (
              <p className="mt-1 text-xs text-[var(--galdr-fg-muted)]">
                Portal ID: {accountId}
              </p>
            )}
          </div>
          <span
            className={`rounded-full px-2.5 py-0.5 text-xs font-medium ${
              connected
                ? "border border-[color:rgb(52_211_153_/_0.35)] bg-[color:rgb(52_211_153_/_0.14)] text-[var(--galdr-success)]"
                : "galdr-pill"
            }`}
          >
            {connected ? "Connected" : "Not connected"}
          </span>
        </div>
      </div>

      {error && (
        <div className="galdr-alert-danger mt-4 px-3 py-2 text-sm">{error}</div>
      )}

      <div className="mt-5 flex gap-2">
        {!connected && (
          <button
            onClick={onConnect}
            disabled={loading}
            className="galdr-button-primary px-4 py-2 text-sm font-medium disabled:opacity-50"
          >
            {loading ? "Connecting..." : "Connect HubSpot"}
          </button>
        )}
        <button
          onClick={onRefresh}
          disabled={loading}
          className="galdr-button-secondary px-4 py-2 text-sm font-medium disabled:opacity-50"
        >
          Refresh status
        </button>
      </div>
    </WizardStep>
  );
}
