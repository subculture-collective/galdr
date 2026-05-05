import api from "@/lib/api";

export interface WebhookFieldMapping {
  source_path: string;
  target_field: string;
}

export interface WebhookConfiguration {
  id: string;
  name: string;
  url: string;
  secret: string;
  mappings: WebhookFieldMapping[];
  last_event_at: string | null;
  event_count: number;
  status: "active" | "paused" | "error";
}

export interface CreateWebhookPayload {
  name: string;
  mappings: WebhookFieldMapping[];
}

export interface TestWebhookMappingPayload {
  mappings: WebhookFieldMapping[];
  sample_payload: Record<string, unknown>;
}

export const webhooksApi = {
  list: () =>
    api.get<{ webhooks: WebhookConfiguration[] }>("/integrations/webhooks"),

  create: (payload: CreateWebhookPayload) =>
    api.post<{ webhook: WebhookConfiguration }>(
      "/integrations/webhooks",
      payload,
    ),

  testMapping: (payload: TestWebhookMappingPayload) =>
    api.post<{ mapped_result: Record<string, unknown> }>(
      "/integrations/webhooks/test",
      payload,
    ),
};
