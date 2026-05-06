import { useCallback, useEffect, useState } from "react";
import { Link } from "react-router-dom";
import api from "@/lib/api";
import { useToast } from "@/contexts/ToastContext";
import {
  LIMIT_INTEGRATION,
  useFeatureFlag,
} from "@/contexts/FeatureFlagContext";
import StripeConnectionCard from "@/components/integrations/StripeConnectionCard";
import HubSpotConnectionCard from "@/components/integrations/HubSpotConnectionCard";
import IntercomConnectionCard from "@/components/integrations/IntercomConnectionCard";
import ZendeskConnectionCard from "@/components/integrations/ZendeskConnectionCard";
import SalesforceConnectionCard from "@/components/integrations/SalesforceConnectionCard";
import PostHogConnectionCard from "@/components/integrations/PostHogConnectionCard";
import WebhookConfig from "@/components/integrations/WebhookConfig";
import IntegrationCard from "@/components/IntegrationCard";
import UpgradePrompt from "@/components/UpgradePrompt";
import { Activity, Loader2 } from "lucide-react";

interface IntegrationConnection {
  id: string;
  provider: string;
  status: string;
  last_sync_at?: string;
  customer_count?: number;
}

const DEDICATED_INTEGRATION_PROVIDERS = new Set([
  "stripe",
  "hubspot",
  "intercom",
  "zendesk",
  "salesforce",
  "posthog",
]);

export default function IntegrationsTab() {
  const [integrations, setIntegrations] = useState<IntegrationConnection[]>([]);
  const [loading, setLoading] = useState(true);
  const toast = useToast();
  const integrationLimit = useFeatureFlag(LIMIT_INTEGRATION);

  const fetchIntegrations = useCallback(async () => {
    try {
      const { data } = await api.get<{ integrations: IntegrationConnection[] }>(
        "/integrations",
      );
      setIntegrations(data.integrations ?? []);
    } catch {
      toast.error("Failed to load integrations");
    } finally {
      setLoading(false);
    }
  }, [toast]);

  useEffect(() => {
    fetchIntegrations();
  }, [fetchIntegrations]);

  if (loading) {
    return (
      <div className="flex justify-center py-8">
        <Loader2 className="h-6 w-6 animate-spin text-[var(--galdr-fg-muted)]" />
      </div>
    );
  }

  const otherIntegrations = integrations.filter(
    (integration) => !DEDICATED_INTEGRATION_PROVIDERS.has(integration.provider),
  );

  return (
    <div className="space-y-6">
      <div className="galdr-card flex flex-col gap-3 p-4 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h3 className="flex items-center gap-2 text-sm font-medium text-[var(--galdr-fg)]">
            <Activity className="h-4 w-4 text-cyan-300" />
            Integration health monitoring
          </h3>
          <p className="mt-1 text-sm text-[var(--galdr-fg-muted)]">
            Review sync status, stale warnings, error rates, and sync history.
          </p>
        </div>
        <Link
          to="/integration-health"
          className="galdr-button galdr-button-secondary justify-center px-4 py-2 text-sm"
        >
          Open dashboard
        </Link>
      </div>

      {!integrationLimit.allowed && integrationLimit.limit !== null && (
        <UpgradePrompt
          featureName="More integrations"
          recommendedTier={integrationLimit.recommendedTier}
          description={`Your current plan includes ${integrationLimit.limit} integration${
            integrationLimit.limit === 1 ? "" : "s"
          }. Upgrade to connect additional data sources.`}
        />
      )}

      <div>
        <h3 className="mb-4 text-sm font-medium text-[var(--galdr-fg)]">
          Stripe
        </h3>
        <StripeConnectionCard />
      </div>

      <div>
        <h3 className="mb-4 text-sm font-medium text-[var(--galdr-fg)]">
          HubSpot
        </h3>
        <HubSpotConnectionCard />
      </div>

      <div>
        <h3 className="mb-4 text-sm font-medium text-[var(--galdr-fg)]">
          Intercom
        </h3>
        <IntercomConnectionCard />
      </div>

      <div>
        <h3 className="mb-4 text-sm font-medium text-[var(--galdr-fg)]">
          Zendesk
        </h3>
        <ZendeskConnectionCard />
      </div>

      <div>
        <h3 className="mb-4 text-sm font-medium text-[var(--galdr-fg)]">
          Salesforce
        </h3>
        <SalesforceConnectionCard />
      </div>

      <div>
        <h3 className="mb-4 text-sm font-medium text-[var(--galdr-fg)]">
          PostHog
        </h3>
        <PostHogConnectionCard />
      </div>

      <WebhookConfig />

      {otherIntegrations.length > 0 && (
        <div>
          <h3 className="mb-4 text-sm font-medium text-[var(--galdr-fg)]">
            Other Integrations
          </h3>
          <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
            {otherIntegrations.map((integration) => (
              <IntegrationCard
                key={integration.id}
                provider={integration.provider}
                status={
                  integration.status as
                    | "connected"
                    | "syncing"
                    | "error"
                    | "disconnected"
                }
                lastSyncAt={integration.last_sync_at}
                customerCount={integration.customer_count}
              />
            ))}
          </div>
        </div>
      )}
    </div>
  );
}
