import api from "./api";

export interface PostHogStatus {
  status: string;
  project_id?: string;
  last_sync_at?: string;
  last_sync_error?: string;
  connected_at?: string;
  event_count?: number;
  user_count?: number;
}

export interface PostHogConnectPayload {
  api_key: string;
  project_id: string;
}

export const postHogApi = {
  getStatus: () => api.get<PostHogStatus>("/integrations/posthog/status"),

  connect: (data: PostHogConnectPayload) =>
    api.post<PostHogStatus>("/integrations/posthog/connect", data),

  disconnect: () => api.delete("/integrations/posthog"),

  triggerSync: () =>
    api.post<{ message: string }>("/integrations/posthog/sync"),
};
