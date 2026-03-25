import { useCallback, useEffect, useState } from "react";
import api from "@/lib/api";
import { useToast } from "@/contexts/ToastContext";
import StripeConnectionCard from "@/components/integrations/StripeConnectionCard";
import HubSpotConnectionCard from "@/components/integrations/HubSpotConnectionCard";
import IntercomConnectionCard from "@/components/integrations/IntercomConnectionCard";
import IntegrationCard from "@/components/IntegrationCard";
import { Loader2 } from "lucide-react";

interface IntegrationConnection {
  id: string;
  provider: string;
  status: string;
  last_sync_at?: string;
  customer_count?: number;
}

export default function IntegrationsTab() {
  const [integrations, setIntegrations] = useState<IntegrationConnection[]>([]);
  const [loading, setLoading] = useState(true);
  const toast = useToast();

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

  // Filter out Stripe and HubSpot from generic list since they have dedicated cards
  const otherIntegrations = integrations.filter(
    (i) =>
      i.provider !== "stripe" &&
      i.provider !== "hubspot" &&
      i.provider !== "intercom",
  );

  return (
    <div className="space-y-6">
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
