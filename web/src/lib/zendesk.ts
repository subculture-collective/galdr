import api from "./api";

export interface ZendeskStatus {
  status: string;
  external_account_id?: string;
  subdomain?: string;
  last_sync_at?: string;
  last_sync_error?: string;
  connected_at?: string;
  ticket_count?: number;
  user_count?: number;
}

export const zendeskApi = {
  getConnectUrl: (subdomain: string) =>
    api.get<{ url: string }>("/integrations/zendesk/connect", {
      params: { subdomain },
    }),

  getStatus: () => api.get<ZendeskStatus>("/integrations/zendesk/status"),

  disconnect: () => api.delete("/integrations/zendesk"),

  triggerSync: () =>
    api.post<{ message: string }>("/integrations/zendesk/sync"),

  callback: (code: string, state: string) =>
    api.get("/integrations/zendesk/callback", { params: { code, state } }),
};
