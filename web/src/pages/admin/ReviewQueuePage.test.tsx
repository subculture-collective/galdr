import { renderToStaticMarkup } from "react-dom/server";
import React from "react";
import { ReviewQueuePageView } from "./ReviewQueuePage";
import type { MarketplaceConnector } from "../MarketplacePage";

const noop = () => undefined;

const connectors: MarketplaceConnector[] = [
  {
    id: "mock-crm",
    version: "1.0.0",
    developer_id: "dev_1",
    name: "Mock CRM",
    description: "Sync account records from Mock CRM.",
    status: "submitted",
    created_at: "2026-05-01T00:00:00Z",
    updated_at: "2026-05-02T00:00:00Z",
    manifest: {
      id: "mock-crm",
      name: "Mock CRM",
      version: "1.0.0",
      description: "Sync account records from Mock CRM.",
      categories: ["crm"],
      auth: { type: "oauth2" },
      sync: {
        supported_modes: ["full"],
        default_mode: "full",
        resources: [{ name: "customers", required: true }],
      },
    },
  },
];

function render(
  props: Partial<React.ComponentProps<typeof ReviewQueuePageView>> = {},
) {
  return renderToStaticMarkup(
    React.createElement(ReviewQueuePageView, {
      connectors,
      loading: false,
      error: "",
      actingId: null,
      onReview: noop,
      onReject: noop,
      onPublish: noop,
      onRetry: noop,
      ...props,
    }),
  );
}

function assertMatch(input: string, pattern: RegExp) {
  if (!pattern.test(input)) {
    throw new Error(`Expected ${pattern} to match ${input}`);
  }
}

const queue = render();
assertMatch(queue, /Connector submission queue/);
assertMatch(queue, /Mock CRM/);
assertMatch(queue, /submitted/);
assertMatch(queue, /Run review/);
assertMatch(queue, /Reject/);
assertMatch(queue, /Publish/);

const empty = render({ connectors: [] });
assertMatch(empty, /No pending connector submissions/);

const loading = render({ loading: true });
assertMatch(loading, /Loading connector submissions/);

const error = render({ error: "Failed to load queue." });
assertMatch(error, /Failed to load queue/);
assertMatch(error, /Retry/);

console.log("ReviewQueuePage states render correctly");
