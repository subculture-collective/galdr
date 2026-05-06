import assert from "node:assert/strict";
import { describe, it } from "node:test";
import {
  getPostHogConnectionView,
  validatePostHogCredentials,
} from "./posthogConnectionView.js";

describe("PostHog connection view", () => {
  it("requires API key and project ID before save", () => {
    assert.deepEqual(validatePostHogCredentials("", "project-1"), {
      valid: false,
      message: "Enter a PostHog API key.",
    });

    assert.deepEqual(validatePostHogCredentials("not-a-posthog-key", "12345"), {
      valid: false,
      message: "Enter a valid PostHog personal API key.",
    });

    assert.deepEqual(validatePostHogCredentials("phx_123", ""), {
      valid: false,
      message: "Enter a PostHog project ID.",
    });
  });

  it("shows connected metrics from status", () => {
    assert.deepEqual(
      getPostHogConnectionView({
        status: "active",
        project_id: "12345",
        event_count: 4200,
        user_count: 315,
        last_sync_at: "2026-05-05T12:30:00Z",
      }),
      {
        badge: "Connected",
        isConnected: true,
        canSync: true,
        metrics: [
          "Project ID: 12345",
          "Events synced: 4,200",
          "Users synced: 315",
          "Last sync: May 5, 2026, 12:30 PM",
        ],
      },
    );
  });

  it("uses generic integration status fields for PostHog metrics", () => {
    assert.deepEqual(
      getPostHogConnectionView({
        status: "active",
        external_account_id: "67890",
        customer_count: 52,
      }),
      {
        badge: "Connected",
        isConnected: true,
        canSync: true,
        metrics: ["Project ID: 67890", "Customers synced: 52"],
      },
    );
  });
});
