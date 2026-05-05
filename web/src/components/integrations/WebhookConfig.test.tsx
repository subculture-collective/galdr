import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import "@testing-library/jest-dom/vitest";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { ToastProvider } from "@/contexts/ToastContext";
import WebhookConfig from "@/components/integrations/WebhookConfig";
import { webhooksApi } from "@/lib/webhooks";

vi.mock("@/lib/webhooks", () => ({
  webhooksApi: {
    list: vi.fn(),
    create: vi.fn(),
    testMapping: vi.fn(),
  },
}));

const mockedWebhooksApi = vi.mocked(webhooksApi);

describe("WebhookConfig", () => {
  beforeEach(() => {
    vi.resetAllMocks();
    mockedWebhooksApi.list.mockResolvedValue({
      data: {
        webhooks: [
          {
            id: "wh_existing",
            name: "Zapier lifecycle events",
            url: "https://api.pulsescore.test/api/v1/webhooks/generic/wh_existing",
            secret: "whsec_existing",
            mappings: [{ source_path: "company.name", target_field: "company_name" }],
            last_event_at: "2026-05-04T12:30:00Z",
            event_count: 42,
            status: "active",
          },
        ],
      },
    } as unknown as Awaited<ReturnType<typeof webhooksApi.list>>);
    mockedWebhooksApi.create.mockResolvedValue({
      data: {
        webhook: {
          id: "wh_created",
          name: "Customer.io product events",
          url: "https://api.pulsescore.test/api/v1/webhooks/generic/wh_created",
          secret: "whsec_created",
          mappings: [
            { source_path: "user.email", target_field: "email" },
            { source_path: "account.mrr", target_field: "mrr_cents" },
          ],
          last_event_at: null,
          event_count: 0,
          status: "active",
        },
      },
    } as unknown as Awaited<ReturnType<typeof webhooksApi.create>>);
    mockedWebhooksApi.testMapping.mockResolvedValue({
      data: {
        mapped_result: {
          email: "founder@acme.test",
          mrr_cents: 12900,
        },
      },
    } as unknown as Awaited<ReturnType<typeof webhooksApi.testMapping>>);
  });

  afterEach(() => {
    cleanup();
  });

  it("creates a webhook, builds mappings, tests a sample payload, and lists status", async () => {
    const user = userEvent.setup();
    render(
      <ToastProvider>
        <WebhookConfig />
      </ToastProvider>,
    );

    expect(await screen.findByText("Zapier lifecycle events")).toBeInTheDocument();
    expect(screen.getByText("42 events")).toBeInTheDocument();

    await user.type(screen.getByLabelText("Webhook name"), "Customer.io product events");
    await user.type(screen.getByLabelText("Source path 1"), "user.email");
    await user.type(screen.getByLabelText("Target field 1"), "email");
    await user.click(screen.getByRole("button", { name: "Add mapping" }));
    await user.type(screen.getByLabelText("Source path 2"), "account.mrr");
    await user.type(screen.getByLabelText("Target field 2"), "mrr_cents");
    await user.click(screen.getByRole("button", { name: "Create webhook" }));

    await waitFor(() => {
      expect(mockedWebhooksApi.create).toHaveBeenCalledWith({
        name: "Customer.io product events",
        mappings: [
          { source_path: "user.email", target_field: "email" },
          { source_path: "account.mrr", target_field: "mrr_cents" },
        ],
      });
    });

    expect(await screen.findByText("whsec_created")).toBeInTheDocument();

    fireEvent.change(screen.getByLabelText("Sample payload"), {
      target: {
        value: JSON.stringify({ user: { email: "founder@acme.test" }, account: { mrr: 12900 } }),
      },
    });
    await user.click(screen.getByRole("button", { name: "Test mapping" }));

    await waitFor(() => {
      expect(mockedWebhooksApi.testMapping).toHaveBeenCalledWith({
        mappings: [
          { source_path: "user.email", target_field: "email" },
          { source_path: "account.mrr", target_field: "mrr_cents" },
        ],
        sample_payload: {
          user: { email: "founder@acme.test" },
          account: { mrr: 12900 },
        },
      });
    });
    expect(await screen.findByText(/"email": "founder@acme.test"/)).toBeInTheDocument();
  });

  it("loads common webhook example payloads for testing", async () => {
    const user = userEvent.setup();
    render(
      <ToastProvider>
        <WebhookConfig />
      </ToastProvider>,
    );

    await screen.findByText("Zapier lifecycle events");

    await user.click(screen.getByRole("button", { name: "Use Zapier sample" }));
    expect(
      (screen.getByLabelText("Sample payload") as HTMLTextAreaElement).value,
    ).toContain('"zapier_hook_id"');

    await user.type(screen.getByLabelText("Source path 1"), "contact.email");
    await user.type(screen.getByLabelText("Target field 1"), "email");
    await user.click(screen.getByRole("button", { name: "Test mapping" }));

    await waitFor(() => {
      expect(mockedWebhooksApi.testMapping).toHaveBeenCalledWith({
        mappings: [{ source_path: "contact.email", target_field: "email" }],
        sample_payload: expect.objectContaining({
          zapier_hook_id: "hook_123",
          contact: expect.objectContaining({ email: "founder@acme.test" }),
        }),
      });
    });
  });
});
