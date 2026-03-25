import { useEffect, useMemo, useState } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import { intercomApi } from "@/lib/intercom";
import BaseLayout from "@/components/BaseLayout";
import { ONBOARDING_RESUME_STEP_STORAGE_KEY } from "@/contexts/onboarding/constants";

export default function IntercomCallbackPage() {
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();
  const [asyncError, setAsyncError] = useState("");

  const code = searchParams.get("code");
  const state = searchParams.get("state");
  const error = useMemo(() => {
    const errParam = searchParams.get("error");
    if (errParam) {
      const desc = searchParams.get("error_description") || errParam;
      return `Intercom connection failed: ${desc}`;
    }
    if (!code || !state) {
      return "Invalid callback parameters.";
    }
    return asyncError;
  }, [searchParams, code, state, asyncError]);

  useEffect(() => {
    if (error || !code || !state) {
      return;
    }

    intercomApi
      .callback(code, state)
      .then(() => {
        const resumeStep = localStorage.getItem(
          ONBOARDING_RESUME_STEP_STORAGE_KEY,
        );
        if (resumeStep) {
          localStorage.removeItem(ONBOARDING_RESUME_STEP_STORAGE_KEY);
          navigate(`/onboarding?step=${resumeStep}`, { replace: true });
          return;
        }
        navigate("/settings/integrations", { replace: true });
      })
      .catch(() => {
        setAsyncError("Failed to complete Intercom connection.");
      });
  }, [error, code, state, navigate]);

  if (error) {
    return (
      <BaseLayout>
        <div className="mx-auto max-w-md">
          <div className="galdr-card p-6 text-center">
            <div className="galdr-alert-danger p-6">
              <h2 className="text-lg font-semibold text-[var(--galdr-fg)]">
                Connection Failed
              </h2>
              <p className="mt-2 text-sm text-[var(--galdr-fg-muted)]">
                {error}
              </p>
              <button
                onClick={() => navigate("/settings/integrations")}
                className="galdr-button-primary mt-4 px-4 py-2 text-sm font-medium"
              >
                Back to Settings
              </button>
            </div>
          </div>
        </div>
      </BaseLayout>
    );
  }

  return (
    <BaseLayout>
      <div className="mx-auto max-w-md">
        <div className="galdr-card p-6 text-center">
          <p className="text-sm text-[var(--galdr-fg-muted)]">
            Connecting Intercom...
          </p>
        </div>
      </div>
    </BaseLayout>
  );
}
