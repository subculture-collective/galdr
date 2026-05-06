import { cleanup, render, screen, waitFor } from "@testing-library/react";
import "@testing-library/jest-dom/vitest";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";
import ConnectorInstaller from "./ConnectorInstaller";
import type { MarketplaceConnector } from "@/pages/MarketplacePage";

const connector: MarketplaceConnector = {
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
    auth: {
      type: "api_key",
      api_key: { header_name: "X-SupportDesk-Key", prefix: "Bearer" },
    },
    sync: {
      supported_modes: ["full"],
      default_mode: "full",
      options: {
        region: "us or eu",
        inbox_id: "Primary inbox ID",
      },
      resources: [{ name: "tickets", description: "Tickets", required: true }],
    },
  },
};

describe("ConnectorInstaller", () => {
  afterEach(() => {
    cleanup();
  });

  it("requires configuration and a connection test before activation", async () => {
    const user = userEvent.setup();
    const onInstall = vi.fn();

    render(
      <ConnectorInstaller
        connector={connector}
        installing={false}
        onCancel={vi.fn()}
        onInstall={onInstall}
      />,
    );

    const activateButton = screen.getByRole("button", {
      name: "Activate connector",
    });
    expect(screen.getByText("Connector settings")).toBeInTheDocument();
    expect(screen.getByLabelText("Region")).toBeInTheDocument();
    expect(activateButton).toBeDisabled();
    expect(screen.getByRole("button", { name: "Test connection" })).toBeDisabled();

    await user.type(screen.getByLabelText("API key"), "sk_test_supportdesk");
    await user.type(screen.getByLabelText("Region"), "eu");
    await user.type(screen.getByLabelText("Inbox Id"), "inbox_123");
    await user.click(screen.getByRole("button", { name: "Test connection" }));

    expect(screen.getByText("Connection test passed")).toBeInTheDocument();
    expect(activateButton).toBeEnabled();

    await user.click(activateButton);

    await waitFor(() => {
      expect(onInstall).toHaveBeenCalledWith({
        auth: { type: "api_key", api_key: "sk_test_supportdesk" },
        config: { region: "eu", inbox_id: "inbox_123" },
        test_connection: true,
      });
    });
  });
});
