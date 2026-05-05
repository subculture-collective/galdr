import axios from "axios";

const api = axios.create({
  baseURL: import.meta.env.VITE_API_URL || "http://localhost:8080/api/v1",
  headers: {
    "Content-Type": "application/json",
  },
});

export interface AuthUser {
  id: string;
  email: string;
  first_name: string;
  last_name: string;
}

export interface AuthOrg {
  id: string;
  name: string;
  slug: string;
  industry: string;
  role: string;
  plan: string;
}

export interface TokenPair {
  access_token: string;
  refresh_token: string;
}

export interface AuthResponse {
  user: AuthUser;
  organization: AuthOrg;
  tokens: TokenPair;
}

export interface RegisterPayload {
  email: string;
  password: string;
  first_name: string;
  last_name: string;
  org_name: string;
  industry: string;
}

export interface LoginPayload {
  email: string;
  password: string;
}

export const authApi = {
  register: (data: RegisterPayload) =>
    api.post<AuthResponse>("/auth/register", data),

  login: (data: LoginPayload) => api.post<AuthResponse>("/auth/login", data),

  refresh: (refreshToken: string) =>
    api.post<AuthResponse>("/auth/refresh", { refresh_token: refreshToken }),

  requestPasswordReset: (email: string) =>
    api.post("/auth/password-reset/request", { email }),

  completePasswordReset: (token: string, newPassword: string) =>
    api.post("/auth/password-reset/complete", {
      token,
      new_password: newPassword,
    }),
};

export interface BillingUsageMetric {
  used: number;
  limit: number;
}

export interface BillingSubscriptionResponse {
  tier: string;
  status: string;
  billing_cycle: "monthly" | "annual";
  renewal_date: string | null;
  cancel_at_period_end: boolean;
  usage: {
    customers: BillingUsageMetric;
    integrations: BillingUsageMetric;
  };
  features: Record<string, unknown>;
}

export interface CheckoutPayload {
  priceId?: string;
  tier?: string;
  cycle?: "monthly" | "annual";
  annual?: boolean;
}

export interface PlanChangePayload {
  tier: string;
  cycle: "monthly" | "annual";
}

export interface PlanChangeResponse {
  action: "upgrade" | "downgrade";
  status: "checkout_required" | "scheduled" | "active";
  current_tier: string;
  current_cycle: "monthly" | "annual";
  target_tier: string;
  target_cycle: "monthly" | "annual";
  checkout_url?: string;
  effective_at?: string;
  effective_at_period_end: boolean;
  proration_cents: number;
  limits: {
    current: { customer_limit: number; integration_limit: number };
    target: { customer_limit: number; integration_limit: number };
  };
  features: {
    current: Record<string, boolean>;
    target: Record<string, boolean>;
  };
}

export const billingApi = {
  getSubscription: () =>
    api.get<BillingSubscriptionResponse>("/billing/subscription"),

  createCheckout: (payload: CheckoutPayload) =>
    api.post<{ url: string }>("/billing/checkout", payload),

  changePlan: (payload: PlanChangePayload) =>
    api.post<PlanChangeResponse>("/billing/change-plan", payload),

  createPortalSession: () =>
    api.post<{ url: string }>("/billing/portal-session"),

  cancelAtPeriodEnd: () => api.post<{ status: string }>("/billing/cancel"),
};

// Alert types and API
export interface AlertRule {
  id: string;
  org_id: string;
  name: string;
  description: string;
  trigger_type: string;
  conditions: Record<string, unknown>;
  channel: string;
  recipients: string[];
  is_active: boolean;
  created_by?: string;
  created_at: string;
  updated_at: string;
}

export interface AlertHistory {
  id: string;
  org_id: string;
  alert_rule_id: string;
  customer_id?: string;
  trigger_data: Record<string, unknown>;
  channel: string;
  status: string;
  sent_at?: string;
  error_message?: string;
  sendgrid_message_id?: string;
  delivered_at?: string;
  opened_at?: string;
  clicked_at?: string;
  bounced_at?: string;
  created_at: string;
}

export interface CreateAlertRulePayload {
  name: string;
  description: string;
  trigger_type: string;
  conditions: Record<string, unknown>;
  channel: string;
  recipients: string[];
}

export interface UpdateAlertRulePayload {
  name?: string;
  description?: string;
  trigger_type?: string;
  conditions?: Record<string, unknown>;
  channel?: string;
  recipients?: string[];
  is_active?: boolean;
}

export const alertsApi = {
  listRules: () => api.get<{ rules: AlertRule[] }>("/alerts/rules"),

  getRule: (id: string) =>
    api.get<{ rule: AlertRule }>(`/alerts/rules/${encodeURIComponent(id)}`),

  createRule: (data: CreateAlertRulePayload) =>
    api.post<{ rule: AlertRule }>("/alerts/rules", data),

  updateRule: (id: string, data: UpdateAlertRulePayload) =>
    api.patch<{ rule: AlertRule }>(
      `/alerts/rules/${encodeURIComponent(id)}`,
      data,
    ),

  deleteRule: (id: string) =>
    api.delete(`/alerts/rules/${encodeURIComponent(id)}`),

  listHistory: (params?: {
    status?: string;
    limit?: number;
    offset?: number;
  }) =>
    api.get<{
      history: AlertHistory[];
      total: number;
      limit: number;
      offset: number;
    }>("/alerts/history", { params }),

  getStats: () => api.get<Record<string, number>>("/alerts/stats"),

  listRuleHistory: (
    ruleId: string,
    params?: { limit?: number; offset?: number },
  ) =>
    api.get<{ history: AlertHistory[] }>(
      `/alerts/rules/${encodeURIComponent(ruleId)}/history`,
      { params },
    ),
};

export interface NotificationPreference {
  id: string;
  user_id: string;
  org_id: string;
  email_enabled: boolean;
  in_app_enabled: boolean;
  digest_enabled: boolean;
  digest_frequency: string;
  muted_rule_ids: string[];
  created_at: string;
  updated_at: string;
}

export interface UpdateNotificationPreferencePayload {
  email_enabled?: boolean;
  in_app_enabled?: boolean;
  digest_enabled?: boolean;
  digest_frequency?: string;
  muted_rule_ids?: string[];
}

export const notificationPreferencesApi = {
  get: () => api.get<NotificationPreference>("/notifications/preferences"),

  update: (data: UpdateNotificationPreferencePayload) =>
    api.patch<NotificationPreference>("/notifications/preferences", data),
};

export interface AppNotification {
  id: string;
  user_id: string;
  org_id: string;
  type: string;
  title: string;
  message: string;
  data: Record<string, unknown>;
  read_at: string | null;
  created_at: string;
}

export const notificationsApi = {
  list: (params?: { limit?: number; offset?: number }) =>
    api.get<{
      notifications: AppNotification[];
      total: number;
      limit: number;
      offset: number;
    }>("/notifications", { params }),

  unreadCount: () => api.get<{ count: number }>("/notifications/unread-count"),

  markRead: (id: string) =>
    api.post(`/notifications/${encodeURIComponent(id)}/read`),

  markAllRead: () => api.post("/notifications/read-all"),
};

export type OnboardingStepId =
  | "welcome"
  | "stripe"
  | "hubspot"
  | "intercom"
  | "preview";

export interface OnboardingStatus {
  current_step: OnboardingStepId;
  completed_steps: OnboardingStepId[];
  skipped_steps: OnboardingStepId[];
  step_payloads: Record<string, unknown>;
  completed_at: string | null;
  updated_at: string;
}

export interface UpdateOnboardingStatusPayload {
  step_id?: OnboardingStepId;
  action:
    | "step_started"
    | "step_completed"
    | "step_skipped"
    | "onboarding_completed"
    | "onboarding_abandoned";
  current_step?: OnboardingStepId;
  payload?: Record<string, unknown>;
  metadata?: Record<string, unknown>;
  duration_ms?: number;
}

export interface OnboardingStepMetric {
  step_id: OnboardingStepId;
  started_count: number;
  completed_count: number;
  skipped_count: number;
  completion_rate: number;
  skip_rate: number;
  average_duration_ms: number;
}

export interface OnboardingAnalytics {
  overall_completion_rate: number;
  average_step_duration_ms: number;
  step_metrics: OnboardingStepMetric[];
}

export const onboardingApi = {
  getStatus: () => api.get<OnboardingStatus>("/onboarding/status"),

  updateStatus: (data: UpdateOnboardingStatusPayload) =>
    api.patch<OnboardingStatus>("/onboarding/status", data),

  complete: () => api.post<OnboardingStatus>("/onboarding/complete"),

  reset: () => api.post<OnboardingStatus>("/onboarding/reset"),

  analytics: () => api.get<OnboardingAnalytics>("/onboarding/analytics"),
};

export default api;
