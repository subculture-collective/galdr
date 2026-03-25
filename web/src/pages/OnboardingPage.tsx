import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import api, { onboardingApi, type OnboardingStepId } from "@/lib/api";
import { stripeApi, type StripeStatus } from "@/lib/stripe";
import { hubspotApi, type HubSpotStatus } from "@/lib/hubspot";
import { intercomApi, type IntercomStatus } from "@/lib/intercom";
import WizardShell, {
  type WizardShellStep,
} from "@/components/wizard/WizardShell";
import WelcomeStep, {
  type WelcomeFormValue,
} from "@/components/wizard/steps/WelcomeStep";
import StripeConnectStep from "@/components/wizard/steps/StripeConnectStep";
import HubSpotConnectStep from "@/components/wizard/steps/HubSpotConnectStep";
import IntercomConnectStep from "@/components/wizard/steps/IntercomConnectStep";
import ScorePreviewStep, {
  type AtRiskCustomerPreview,
  type ScoreBucket,
} from "@/components/wizard/steps/ScorePreviewStep";
import {
  OnboardingProvider,
  useOnboarding,
} from "@/contexts/onboarding/OnboardingContext";
import {
  ONBOARDING_RESUME_STEP_STORAGE_KEY,
  stepIdToIndex,
  stepIndexToId,
} from "@/contexts/onboarding/constants";
import { useToast } from "@/contexts/ToastContext";
import { useAuth } from "@/contexts/AuthContext";

interface CustomerListResponse {
  customers: Array<{
    id: string;
    name: string;
    health_score: number;
  }>;
}

function isConnected(status?: string) {
  return status === "active" || status === "syncing";
}

function normalizeStatus(status?: string): string {
  if (!status) return "disconnected";
  if (status === "active") return "connected";
  return status;
}

function statusText(status?: string): string {
  const normalized = normalizeStatus(status);
  if (normalized === "connected") return "Connected";
  if (normalized === "syncing") return "Syncing";
  if (normalized === "error") return "Error";
  return "Not connected";
}

function OnboardingContent() {
  const navigate = useNavigate();
  const [searchParams, setSearchParams] = useSearchParams();
  const initialStepRef = useRef(searchParams.get("step"));
  const toast = useToast();
  const { organization } = useAuth();
  const {
    status,
    currentStepIndex,
    setCurrentStepIndex,
    hydrateFromStatus,
    markCompleted,
    markSkipped,
    setStepPayload,
    setCompletedAt,
  } = useOnboarding();

  const [initialLoading, setInitialLoading] = useState(true);
  const [initialError, setInitialError] = useState("");

  const [welcomeValue, setWelcomeValue] = useState<WelcomeFormValue>({
    name: "",
    industry: "",
    company_size: "",
  });

  const [stripeStatus, setStripeStatus] = useState<StripeStatus | null>(null);
  const [hubSpotStatus, setHubSpotStatus] = useState<HubSpotStatus | null>(
    null,
  );
  const [intercomStatus, setIntercomStatus] = useState<IntercomStatus | null>(
    null,
  );

  const [stripeBusy, setStripeBusy] = useState(false);
  const [hubSpotBusy, setHubSpotBusy] = useState(false);
  const [intercomBusy, setIntercomBusy] = useState(false);

  const [stripeError, setStripeError] = useState("");
  const [hubSpotError, setHubSpotError] = useState("");
  const [intercomError, setIntercomError] = useState("");

  const [previewLoading, setPreviewLoading] = useState(false);
  const [distribution, setDistribution] = useState<ScoreBucket[]>([]);
  const [atRiskCustomers, setAtRiskCustomers] = useState<
    AtRiskCustomerPreview[]
  >([]);

  const startedAtRef = useRef<Record<string, number>>({});
  const startedEventRef = useRef<Set<string>>(new Set());
  const previewSyncTriggeredRef = useRef(false);

  const fetchStripeStatus = useCallback(async () => {
    try {
      const { data } = await stripeApi.getStatus();
      setStripeStatus(data);
      setStripeError("");
    } catch {
      setStripeStatus(null);
    }
  }, []);

  const fetchHubSpotStatus = useCallback(async () => {
    try {
      const { data } = await hubspotApi.getStatus();
      setHubSpotStatus(data);
      setHubSpotError("");
    } catch {
      setHubSpotStatus(null);
    }
  }, []);

  const fetchIntercomStatus = useCallback(async () => {
    try {
      const { data } = await intercomApi.getStatus();
      setIntercomStatus(data);
      setIntercomError("");
    } catch {
      setIntercomStatus(null);
    }
  }, []);

  const fetchPreviewData = useCallback(async () => {
    setPreviewLoading(true);
    try {
      const [{ data: scoreRes }, { data: customerRes }] = await Promise.all([
        api.get("/dashboard/score-distribution"),
        api.get<CustomerListResponse>("/customers", {
          params: {
            page: 1,
            per_page: 5,
            sort: "health_score",
            order: "asc",
            risk: "red",
          },
        }),
      ]);

      const buckets =
        scoreRes?.buckets ?? scoreRes?.distribution ?? scoreRes ?? [];
      if (Array.isArray(buckets)) {
        setDistribution(
          buckets.map((bucket: Record<string, unknown>) => ({
            range: String(bucket.range ?? ""),
            count: Number(bucket.count ?? 0),
          })),
        );
      } else {
        setDistribution([]);
      }

      const risky = customerRes.customers ?? [];
      setAtRiskCustomers(
        risky.map((customer) => ({
          id: customer.id,
          name: customer.name,
          health_score: Math.round(customer.health_score),
        })),
      );
    } catch {
      setDistribution([]);
      setAtRiskCustomers([]);
    } finally {
      setPreviewLoading(false);
    }
  }, []);

  const loadOnboarding = useCallback(async () => {
    setInitialLoading(true);
    setInitialError("");
    try {
      const { data } = await onboardingApi.getStatus();
      if (data.completed_at) {
        navigate("/", { replace: true });
        return;
      }

      const initialStepParam = initialStepRef.current;
      const preferredStepIndex =
        initialStepParam &&
        ["welcome", "stripe", "hubspot", "intercom", "preview"].includes(
          initialStepParam,
        )
          ? stepIdToIndex(initialStepParam)
          : undefined;

      hydrateFromStatus(data, preferredStepIndex);

      const welcomePayload =
        data.step_payloads?.welcome &&
        typeof data.step_payloads.welcome === "object"
          ? (data.step_payloads.welcome as Record<string, unknown>)
          : null;

      setWelcomeValue({
        name:
          typeof welcomePayload?.name === "string"
            ? welcomePayload.name
            : (organization?.name ?? ""),
        industry:
          typeof welcomePayload?.industry === "string"
            ? welcomePayload.industry
            : "",
        company_size:
          typeof welcomePayload?.company_size === "string"
            ? welcomePayload.company_size
            : "",
      });

      await Promise.all([
        fetchStripeStatus(),
        fetchHubSpotStatus(),
        fetchIntercomStatus(),
      ]);
    } catch {
      setInitialError("Failed to load onboarding state.");
    } finally {
      setInitialLoading(false);
    }
  }, [
    navigate,
    hydrateFromStatus,
    organization?.name,
    fetchStripeStatus,
    fetchHubSpotStatus,
    fetchIntercomStatus,
  ]);

  useEffect(() => {
    void loadOnboarding();
  }, [loadOnboarding]);

  useEffect(() => {
    if (!status) return;

    const stepId = stepIndexToId(currentStepIndex);
    startedAtRef.current[stepId] = Date.now();

    if (startedEventRef.current.has(stepId)) return;
    startedEventRef.current.add(stepId);

    void onboardingApi.updateStatus({
      action: "step_started",
      step_id: stepId,
      current_step: stepId,
    });
  }, [status, currentStepIndex]);

  useEffect(() => {
    if (!status) return;
    const stepId = stepIndexToId(currentStepIndex);

    setSearchParams(
      (prev) => {
        if (prev.get("step") === stepId) return prev;
        const next = new URLSearchParams(prev);
        next.set("step", stepId);
        return next;
      },
      { replace: true },
    );
  }, [status, currentStepIndex, setSearchParams]);

  const connectedProviders = useMemo(() => {
    const providers: string[] = [];
    if (isConnected(stripeStatus?.status)) providers.push("stripe");
    if (isConnected(hubSpotStatus?.status)) providers.push("hubspot");
    if (isConnected(intercomStatus?.status)) providers.push("intercom");
    return providers;
  }, [stripeStatus?.status, hubSpotStatus?.status, intercomStatus?.status]);

  const syncStatus = useMemo(
    () => ({
      stripe: statusText(stripeStatus?.status),
      hubspot: statusText(hubSpotStatus?.status),
      intercom: statusText(intercomStatus?.status),
    }),
    [stripeStatus?.status, hubSpotStatus?.status, intercomStatus?.status],
  );

  useEffect(() => {
    const stepId = stepIndexToId(currentStepIndex);
    if (stepId !== "preview") {
      previewSyncTriggeredRef.current = false;
      return;
    }

    async function triggerSyncs() {
      if (previewSyncTriggeredRef.current) return;
      previewSyncTriggeredRef.current = true;

      const jobs: Promise<unknown>[] = [];
      if (isConnected(stripeStatus?.status)) jobs.push(stripeApi.triggerSync());
      if (isConnected(hubSpotStatus?.status))
        jobs.push(hubspotApi.triggerSync());
      if (isConnected(intercomStatus?.status))
        jobs.push(intercomApi.triggerSync());

      if (jobs.length > 0) {
        try {
          await Promise.allSettled(jobs);
          toast.info(
            "Initial sync started. Preview will update as data arrives.",
          );
        } catch {
          // Promise.allSettled shouldn't throw, but keep the wizard resilient.
        }
      }
    }

    void triggerSyncs();
    void fetchPreviewData();

    const poll = setInterval(() => {
      void Promise.all([
        fetchStripeStatus(),
        fetchHubSpotStatus(),
        fetchIntercomStatus(),
        fetchPreviewData(),
      ]);
    }, 10000);

    return () => clearInterval(poll);
  }, [
    currentStepIndex,
    stripeStatus?.status,
    hubSpotStatus?.status,
    intercomStatus?.status,
    fetchStripeStatus,
    fetchHubSpotStatus,
    fetchIntercomStatus,
    fetchPreviewData,
    toast,
  ]);

  async function recordStep(
    action: "step_completed" | "step_skipped",
    stepId: OnboardingStepId,
    nextStep: OnboardingStepId,
    payload?: Record<string, unknown>,
  ) {
    const startedAt = startedAtRef.current[stepId];
    const duration = startedAt
      ? Math.max(0, Date.now() - startedAt)
      : undefined;

    await onboardingApi.updateStatus({
      action,
      step_id: stepId,
      current_step: nextStep,
      payload,
      duration_ms: duration,
    });

    if (payload) {
      setStepPayload(stepId, payload);
    }

    if (action === "step_completed") {
      markCompleted(stepId);
    } else {
      markSkipped(stepId);
    }
  }

  async function handleWelcomeComplete() {
    const orgName = welcomeValue.name.trim();
    if (!orgName) {
      throw new Error("Organization name is required.");
    }

    try {
      await api.patch("/organizations/current", { name: orgName });
    } catch {
      throw new Error("Failed to save organization setup.");
    }

    const payload = {
      name: orgName,
      industry: welcomeValue.industry,
      company_size: welcomeValue.company_size,
    };

    try {
      await recordStep("step_completed", "welcome", "stripe", payload);
      toast.success("Organization setup saved.");
    } catch {
      throw new Error("Failed to save onboarding progress.");
    }
  }

  async function connectStripe() {
    setStripeBusy(true);
    setStripeError("");
    try {
      localStorage.setItem(ONBOARDING_RESUME_STEP_STORAGE_KEY, "stripe");
      const { data } = await stripeApi.getConnectUrl();
      window.location.href = data.url;
    } catch {
      setStripeError("Failed to start Stripe connection.");
    } finally {
      setStripeBusy(false);
    }
  }

  async function connectHubSpot() {
    setHubSpotBusy(true);
    setHubSpotError("");
    try {
      localStorage.setItem(ONBOARDING_RESUME_STEP_STORAGE_KEY, "hubspot");
      const { data } = await hubspotApi.getConnectUrl();
      window.location.href = data.url;
    } catch {
      setHubSpotError("Failed to start HubSpot connection.");
    } finally {
      setHubSpotBusy(false);
    }
  }

  async function connectIntercom() {
    setIntercomBusy(true);
    setIntercomError("");
    try {
      localStorage.setItem(ONBOARDING_RESUME_STEP_STORAGE_KEY, "intercom");
      const { data } = await intercomApi.getConnectUrl();
      window.location.href = data.url;
    } catch {
      setIntercomError("Failed to start Intercom connection.");
    } finally {
      setIntercomBusy(false);
    }
  }

  const steps: WizardShellStep[] = [
    {
      id: "welcome",
      label: "Welcome",
      canProceed: welcomeValue.name.trim().length > 0,
      content: (
        <WelcomeStep
          value={welcomeValue}
          organizationName={organization?.name ?? "Your organization"}
          onChange={setWelcomeValue}
        />
      ),
      onNext: handleWelcomeComplete,
    },
    {
      id: "stripe",
      label: "Stripe",
      canProceed: isConnected(stripeStatus?.status),
      canSkip: true,
      content: (
        <StripeConnectStep
          connected={isConnected(stripeStatus?.status)}
          loading={stripeBusy}
          statusText={statusText(stripeStatus?.status)}
          accountId={stripeStatus?.account_id}
          error={stripeError}
          onConnect={connectStripe}
          onRefresh={fetchStripeStatus}
        />
      ),
      onNext: async () => {
        await recordStep("step_completed", "stripe", "hubspot", {
          connected: true,
          status: normalizeStatus(stripeStatus?.status),
        });
      },
      onSkip: async () => {
        await recordStep("step_skipped", "stripe", "hubspot", {
          connected: false,
          reason: "skipped",
        });
      },
    },
    {
      id: "hubspot",
      label: "HubSpot",
      canProceed: isConnected(hubSpotStatus?.status),
      canSkip: true,
      content: (
        <HubSpotConnectStep
          connected={isConnected(hubSpotStatus?.status)}
          loading={hubSpotBusy}
          statusText={statusText(hubSpotStatus?.status)}
          accountId={hubSpotStatus?.external_account_id}
          error={hubSpotError}
          onConnect={connectHubSpot}
          onRefresh={fetchHubSpotStatus}
        />
      ),
      onNext: async () => {
        await recordStep("step_completed", "hubspot", "intercom", {
          connected: true,
          status: normalizeStatus(hubSpotStatus?.status),
        });
      },
      onSkip: async () => {
        await recordStep("step_skipped", "hubspot", "intercom", {
          connected: false,
          reason: "skipped",
        });
      },
    },
    {
      id: "intercom",
      label: "Intercom",
      canProceed: isConnected(intercomStatus?.status),
      canSkip: true,
      content: (
        <IntercomConnectStep
          connected={isConnected(intercomStatus?.status)}
          loading={intercomBusy}
          statusText={statusText(intercomStatus?.status)}
          accountId={intercomStatus?.external_account_id}
          error={intercomError}
          onConnect={connectIntercom}
          onRefresh={fetchIntercomStatus}
        />
      ),
      onNext: async () => {
        await recordStep("step_completed", "intercom", "preview", {
          connected: true,
          status: normalizeStatus(intercomStatus?.status),
        });
      },
      onSkip: async () => {
        await recordStep("step_skipped", "intercom", "preview", {
          connected: false,
          reason: "skipped",
        });
      },
    },
    {
      id: "preview",
      label: "Preview",
      content: (
        <ScorePreviewStep
          connectedProviders={connectedProviders}
          syncStatus={syncStatus}
          loading={previewLoading}
          distribution={distribution}
          atRiskCustomers={atRiskCustomers}
        />
      ),
      onNext: async () => {
        await recordStep("step_completed", "preview", "preview", {
          connected_providers: connectedProviders,
        });
      },
    },
  ];

  async function handleDone() {
    try {
      await onboardingApi.complete();
      setCompletedAt(new Date().toISOString());
      toast.success("Onboarding complete! Welcome to Galdr.");
      navigate("/", { replace: true });
    } catch {
      throw new Error("Failed to complete onboarding.");
    }
  }

  if (initialLoading) {
    return (
      <div className="flex min-h-[60vh] items-center justify-center">
        <div className="h-8 w-8 animate-spin rounded-full border-4 border-[var(--galdr-accent)] border-t-transparent" />
      </div>
    );
  }

  if (initialError || !status) {
    return (
      <div className="galdr-alert-danger mx-auto max-w-xl p-6 text-center">
        <h2 className="text-lg font-semibold">Unable to load onboarding</h2>
        <p className="mt-2 text-sm">
          {initialError || "Something went wrong while loading onboarding."}
        </p>
        <button
          onClick={() => void loadOnboarding()}
          className="galdr-button-danger-outline mt-4 px-4 py-2 text-sm font-medium"
        >
          Retry
        </button>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      <div>
        <h1 className="text-2xl font-bold text-[var(--galdr-fg)]">
          Onboarding Wizard
        </h1>
        <p className="mt-1 text-sm text-[var(--galdr-fg-muted)]">
          Complete setup to unlock customer health insights and alerts.
        </p>
      </div>

      <WizardShell
        steps={steps}
        currentStepIndex={currentStepIndex}
        completedSteps={status.completed_steps}
        skippedSteps={status.skipped_steps}
        onCurrentStepChange={setCurrentStepIndex}
        onDone={handleDone}
      />
    </div>
  );
}

export default function OnboardingPage() {
  return (
    <OnboardingProvider>
      <OnboardingContent />
    </OnboardingProvider>
  );
}
