import { renderToStaticMarkup } from "react-dom/server";
import React from "react";
import {
  PostHogConnectionCardView,
  type PostHogStatus,
} from "./PostHogConnectionCard";

const noop = () => undefined;

function render(status: PostHogStatus | null, apiKey = "", projectId = "") {
  return renderToStaticMarkup(
    React.createElement(PostHogConnectionCardView, {
      status,
      apiKey,
      projectId,
      loading: false,
      actionLoading: false,
      error: "",
      message: "",
      onApiKeyChange: noop,
      onProjectIdChange: noop,
      onSave: noop,
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

const disconnected = render(null);
assertMatch(disconnected, /PostHog/);
assertMatch(disconnected, /API key/);
assertMatch(disconnected, /Project ID/);
assertMatch(disconnected, /Save Connection/);
assertMatch(disconnected, /phx_/);

const readyToSave = render(null, "phx_test", "12345");
assertMatch(readyToSave, /value="phx_test"/);
assertMatch(readyToSave, /value="12345"/);
assertNoMatch(readyToSave, /Sync Now/);

const connected = render({
  status: "active",
  project_id: "12345",
  event_count: 4200,
  user_count: 315,
  last_sync_at: "2026-05-05T12:30:00Z",
});
assertMatch(connected, /Connected/);
assertMatch(connected, /Project ID: 12345/);
assertMatch(connected, /Events synced: 4,200/);
assertMatch(connected, /Users synced: 315/);
assertMatch(connected, /Last sync: May 5, 2026, 12:30 PM/);
assertMatch(connected, /Sync Now/);
assertMatch(connected, /Disconnect/);

const errored = render({
  status: "error",
  last_sync_error: "invalid project",
});
assertMatch(errored, /Error/);
assertMatch(errored, /Last error: invalid project/);
