import { renderToStaticMarkup } from "react-dom/server";
import React from "react";
import {
  ZendeskConnectionCardView,
  type ZendeskStatus,
} from "./ZendeskConnectionCard";

const noop = () => undefined;

function render(status: ZendeskStatus | null, subdomain = "") {
  const props = {
    status,
    subdomain,
    loading: false,
    actionLoading: false,
    error: "",
    message: "",
    onSubdomainChange: noop,
    onConnect: noop,
    onDisconnect: noop,
    onSync: noop,
  };

  return renderToStaticMarkup(
    React.createElement(ZendeskConnectionCardView, props),
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

const disconnected = render({ status: "disconnected" });
assertMatch(disconnected, /Zendesk/);
assertMatch(disconnected, /workspace subdomain/i);
assertMatch(disconnected, /acme\.zendesk\.com/);
assertMatch(disconnected, /Connect Zendesk/);
assertMatch(disconnected, /disabled=""/);

const readyToConnect = render({ status: "disconnected" }, "acme");
assertMatch(readyToConnect, /value="acme"/);
assertNoMatch(readyToConnect, /disabled=""/);

const connected = render({
  status: "active",
  external_account_id: "acme",
  ticket_count: 42,
  user_count: 17,
  last_sync_at: "2026-05-05T12:00:00Z",
});
assertMatch(connected, /Connected/);
assertMatch(connected, /Subdomain:/);
assertMatch(connected, /acme\.zendesk\.com/);
assertMatch(connected, /Tickets synced: 42/);
assertMatch(connected, /Users synced: 17/);
assertMatch(connected, /Sync Now/);
assertMatch(connected, /Disconnect/);

const connectedWithFullDomain = render({
  status: "active",
  subdomain: "acme.zendesk.com",
});
assertMatch(connectedWithFullDomain, /acme\.zendesk\.com/);
assertNoMatch(connectedWithFullDomain, /acme\.zendesk\.com\.zendesk\.com/);

const syncing = render({ status: "syncing", ticket_count: 2, user_count: 3 });
assertMatch(syncing, /Syncing/);
assertMatch(syncing, /Tickets synced: 2/);
assertMatch(syncing, /Users synced: 3/);

const errored = render({ status: "error", last_sync_error: "rate limited" });
assertMatch(errored, /Error/);
assertMatch(errored, /Last error: rate limited/);
