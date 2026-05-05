import React from "react";
import { renderToStaticMarkup } from "react-dom/server";
import {
  SalesforceConnectionCardView,
  type SalesforceStatus,
} from "./SalesforceConnectionCard";

const noop = () => undefined;

function render(status: SalesforceStatus | null, loading = false) {
  return renderToStaticMarkup(
    React.createElement(SalesforceConnectionCardView, {
      status,
      loading,
      actionLoading: false,
      error: "",
      message: "",
      onConnect: noop,
      onDisconnect: noop,
      onSync: noop,
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

const loading = render(null, true);
assertMatch(loading, /Loading Salesforce status/);

const disconnected = render({ status: "disconnected" });
assertMatch(disconnected, /Salesforce/);
assertMatch(disconnected, /CRM accounts, contacts, and opportunities/);
assertMatch(disconnected, /Not connected/);
assertMatch(disconnected, /Connect Salesforce/);
assertNoMatch(disconnected, /Sync Now/);

const connected = render({
  status: "active",
  external_account_id: "00Dxx0000001gPFEAY",
  account_count: 12,
  contact_count: 340,
  opportunity_count: 28,
  last_sync_at: "2026-05-05T12:00:00Z",
});
assertMatch(connected, /Connected/);
assertMatch(connected, /Org ID:/);
assertMatch(connected, /00Dxx0000001gPFEAY/);
assertMatch(connected, /Accounts synced: 12/);
assertMatch(connected, /Contacts synced: 340/);
assertMatch(connected, /Opportunities synced: 28/);
assertMatch(connected, /Sync Now/);
assertMatch(connected, /Disconnect/);

const connectedWithZeroCounts = render({
  status: "active",
  account_count: 0,
  contact_count: 0,
  opportunity_count: 0,
});
assertMatch(connectedWithZeroCounts, /Accounts synced: 0/);
assertMatch(connectedWithZeroCounts, /Contacts synced: 0/);
assertMatch(connectedWithZeroCounts, /Opportunities synced: 0/);

const syncing = render({ status: "syncing", account_count: 2 });
assertMatch(syncing, /Syncing/);
assertMatch(syncing, /Accounts synced: 2/);

const errored = render({ status: "error", last_sync_error: "token expired" });
assertMatch(errored, /Error/);
assertMatch(errored, /Last error: token expired/);
