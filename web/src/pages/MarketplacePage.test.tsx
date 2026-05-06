import { renderToStaticMarkup } from "react-dom/server";
import React from "react";
import {
  MarketplacePageView,
  type MarketplaceConnector,
} from "./MarketplacePage";

const noop = () => undefined;

const connectors: MarketplaceConnector[] = [
  {
    id: "mock-crm",
    version: "1.2.0",
    developer_id: "dev_1",
    name: "Mock CRM",
    description: "Sync account records and sales activity from Mock CRM.",
    status: "published",
    created_at: "2026-05-01T00:00:00Z",
    updated_at: "2026-05-02T00:00:00Z",
    manifest: {
      id: "mock-crm",
      name: "Mock CRM",
      version: "1.2.0",
      description: "Sync account records and sales activity from Mock CRM.",
      icon_url: "https://example.com/mock-crm.svg",
      categories: ["crm"],
      auth: { type: "oauth2" },
      sync: {
        supported_modes: ["full", "incremental"],
        default_mode: "full",
        resources: [
          { name: "customers", description: "Accounts", required: true },
        ],
      },
    },
  },
  {
    id: "supportdesk",
    version: "1.0.0",
    developer_id: "dev_2",
    name: "SupportDesk",
    description: "Bring support ticket health signals into PulseScore.",
    status: "published",
    created_at: "2026-05-01T00:00:00Z",
    updated_at: "2026-05-02T00:00:00Z",
    manifest: {
      id: "supportdesk",
      name: "SupportDesk",
      version: "1.0.0",
      description: "Bring support ticket health signals into PulseScore.",
      categories: ["support"],
      auth: { type: "api_key" },
      sync: {
        supported_modes: ["full"],
        default_mode: "full",
        resources: [
          { name: "tickets", description: "Tickets", required: true },
        ],
      },
    },
  },
  {
    id: "analytics-lab",
    version: "0.9.0",
    developer_id: "dev_3",
    name: "Analytics Lab",
    description: "Experimental product analytics connector.",
    status: "approved",
    created_at: "2026-05-01T00:00:00Z",
    updated_at: "2026-05-02T00:00:00Z",
    manifest: {
      id: "analytics-lab",
      name: "Analytics Lab",
      version: "0.9.0",
      description: "Experimental product analytics connector.",
      categories: ["analytics"],
      auth: { type: "none" },
      sync: {
        supported_modes: ["incremental"],
        default_mode: "incremental",
        resources: [
          { name: "events", description: "Product events", required: true },
        ],
      },
    },
  },
];

function render(
  props: Partial<React.ComponentProps<typeof MarketplacePageView>> = {},
) {
  return renderToStaticMarkup(
    React.createElement(MarketplacePageView, {
      connectors,
      loading: false,
      error: "",
      search: "",
      category: "all",
      status: "all",
      installingId: null,
      selectedInstall: null,
      onSearchChange: noop,
      onCategoryChange: noop,
      onStatusChange: noop,
      onOpenInstall: noop,
      onCloseInstall: noop,
      onConfirmInstall: noop,
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

function assertNoMatch(input: string, pattern: RegExp) {
  if (pattern.test(input)) {
    throw new Error(`Expected ${pattern} not to match ${input}`);
  }
}

const browse = render();
assertMatch(browse, /Integration Marketplace/);
assertMatch(browse, /Search connectors/);
assertMatch(browse, /Mock CRM/);
assertMatch(browse, /SupportDesk/);
assertMatch(browse, /Mock CRM icon/);
assertMatch(browse, /1\.2\.0/);
assertMatch(browse, /OAuth 2/);
assertMatch(browse, /Status: Published/);
assertMatch(browse, /Install/);

const filtered = render({ category: "support" });
assertMatch(filtered, /SupportDesk/);
assertNoMatch(filtered, /Mock CRM/);

const searched = render({ search: "crm" });
assertMatch(searched, /Mock CRM/);
assertNoMatch(searched, /SupportDesk/);

const statusFiltered = render({ status: "approved" });
assertMatch(statusFiltered, /Analytics Lab/);
assertNoMatch(statusFiltered, /Mock CRM/);

const empty = render({ search: "warehouse" });
assertMatch(empty, /No connectors found/);
assertMatch(empty, /Try a different search or category filter/);

const installing = render({
  selectedInstall: connectors[0],
  installingId: "mock-crm",
});
assertMatch(installing, /Install Mock CRM/);
assertMatch(installing, /redirect you to connector configuration/);
assertMatch(installing, /Installing/);

console.log("MarketplacePage browse states render correctly");
