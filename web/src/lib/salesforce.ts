import api from "./api";

export interface SalesforceStatus {
  status: string;
  external_account_id?: string;
  instance_url?: string;
  last_sync_at?: string;
  last_sync_error?: string;
  connected_at?: string;
  account_count?: number;
  contact_count?: number;
  opportunity_count?: number;
}

export const salesforceApi = {
  getConnectUrl: () =>
    api.get<{ url: string }>("/integrations/salesforce/connect"),

  getStatus: () => api.get<SalesforceStatus>("/integrations/salesforce/status"),

  disconnect: () => api.delete("/integrations/salesforce"),

  triggerSync: () =>
    api.post<{ message: string }>("/integrations/salesforce/sync"),

  callback: (code: string, state: string) =>
    api.get("/integrations/salesforce/callback", { params: { code, state } }),
};
