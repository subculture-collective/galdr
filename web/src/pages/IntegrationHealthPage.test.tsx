import React from "react";
import { renderToStaticMarkup } from "react-dom/server";
import { IntegrationHealthView } from "./IntegrationHealthPage";
import type { IntegrationHealthResponse } from "@/lib/api";

const sampleHealth: IntegrationHealthResponse = {
  generated_at: "2026-05-05T12:00:00Z",
  stale_after_hours: 24,
  integrations: [
    {
      provider: "stripe",
      status: "active",
      health_status: "healthy",
      last_sync_at: "2026-05-05T11:30:00Z",
      connected_at: "2026-05-01T09:00:00Z",
      records_synced: 1480,
      error_count: 0,
      sync_duration_ms: 4200,
      error_rate: 0,
      customer_count: 1480,
      alerts: [],
      sync_history: [
        {
          date: "2026-05-05",
          status: "success",
          records_synced: 1480,
          duration_ms: 4200,
        },
      ],
    },
    {
      provider: "hubspot",
      status: "error",
      health_status: "down",
      last_sync_at: "2026-05-04T09:00:00Z",
      connected_at: "2026-05-01T09:00:00Z",
      records_synced: 640,
      error_count: 5,
      sync_duration_ms: 9000,
      error_rate: 0.5,
      customer_count: 640,
      last_sync_error: "rate limited",
      alerts: [
        { type: "integration_down", severity: "critical", message: "HubSpot sync is down." },
        { type: "sync_stale", severity: "warning", message: "Last successful sync is stale." },
      ],
      sync_history: [
        {
          date: "2026-05-04",
          status: "error",
          records_synced: 640,
          duration_ms: 9000,
        },
      ],
    },
  ],
};

function assertIncludes(markup: string, text: string) {
  if (!markup.includes(text)) {
    throw new Error(`Expected markup to include ${text}`);
  }
}

const markup = renderToStaticMarkup(
  <IntegrationHealthView health={sampleHealth} loading={false} />,
);

assertIncludes(markup, "Integration Health");
assertIncludes(markup, "Stripe");
assertIncludes(markup, "HubSpot");
assertIncludes(markup, "50.0%");
assertIncludes(markup, "rate limited");
assertIncludes(markup, "Last successful sync is stale.");
assertIncludes(markup, "Sync History");
assertIncludes(markup, "Sync Success Rate");
assertIncludes(markup, "Records Synced");
