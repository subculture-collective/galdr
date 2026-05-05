import { renderToStaticMarkup } from "react-dom/server";
import React from "react";
import {
  ConnectorDetailPageView,
  type ConnectorDetailPageViewProps,
} from "./ConnectorDetailPage";
import type { MarketplaceConnector } from "./MarketplacePage";

const noop = () => undefined;

const connector: MarketplaceConnector = {
  id: "mock-crm",
  version: "1.2.0",
  developer_id: "dev_1",
  name: "Mock CRM",
  description: "Sync account records and sales activity from Mock CRM.",
  status: "published",
  created_at: "2026-05-01T00:00:00Z",
  updated_at: "2026-05-02T00:00:00Z",
  rating: 4.7,
  install_count: 3210,
  manifest: {
    id: "mock-crm",
    name: "Mock CRM",
    version: "1.2.0",
    description:
      "Full connector detail for account records and sales activity.",
    icon_url: "https://example.com/mock-crm.svg",
    categories: ["crm", "sales"],
    auth: { type: "oauth2" },
    sync: {
      supported_modes: ["full", "incremental"],
      default_mode: "full",
      resources: [
        { name: "customers", description: "Account records", required: true },
        { name: "events", description: "Sales activity", required: false },
      ],
    },
    screenshots: ["https://example.com/mock-crm.png"],
    developer: {
      name: "PulseScore Labs",
      website: "https://example.com",
    },
  },
};

function render(props: Partial<ConnectorDetailPageViewProps> = {}) {
  return renderToStaticMarkup(
    React.createElement(ConnectorDetailPageView, {
      connector,
      loading: false,
      error: "",
      installing: false,
      showInstallConfirm: false,
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

const detail = render();
assertMatch(detail, /Mock CRM/);
assertMatch(detail, /Mock CRM icon/);
assertMatch(detail, /Full connector detail/);
assertMatch(detail, /PulseScore Labs/);
assertMatch(detail, /4\.7 rating/);
assertMatch(detail, /3,210 installs/);
assertMatch(detail, /Account records/);
assertMatch(detail, /Incremental/);
assertMatch(detail, /Install connector/);

const confirm = render({ showInstallConfirm: true, installing: true });
assertMatch(confirm, /Install Mock CRM/);
assertMatch(confirm, /Installing/);

console.log("ConnectorDetailPage states render correctly");
