import { useEffect, useState } from "react";
import { Navigate, useLocation } from "react-router-dom";
import { useAuth } from "@/contexts/AuthContext";
import { onboardingApi, type OnboardingStepId } from "@/lib/api";

const BYPASS_ONBOARDING_PATH_PREFIXES = [
  "/onboarding",
  "/settings/stripe/callback",
  "/settings/hubspot/callback",
  "/settings/intercom/callback",
];

function shouldBypassOnboarding(pathname: string): boolean {
  return BYPASS_ONBOARDING_PATH_PREFIXES.some((prefix) =>
    pathname.startsWith(prefix),
  );
}

export default function ProtectedRoute({
  children,
}: {
  children: React.ReactNode;
}) {
  const location = useLocation();
  const { isAuthenticated, loading } = useAuth();
  const [onboardingLoading, setOnboardingLoading] = useState(true);
  const [onboardingCompleted, setOnboardingCompleted] = useState(true);
  const [onboardingStep, setOnboardingStep] =
    useState<OnboardingStepId>("welcome");

  useEffect(() => {
    let cancelled = false;

    async function checkOnboarding() {
      if (!isAuthenticated) {
        if (!cancelled) {
          setOnboardingCompleted(true);
          setOnboardingLoading(false);
        }
        return;
      }

      if (!cancelled) {
        setOnboardingLoading(true);
      }

      try {
        const { data } = await onboardingApi.getStatus();
        if (cancelled) return;

        setOnboardingCompleted(Boolean(data.completed_at));
        setOnboardingStep(data.current_step);
      } catch {
        // Fail open here so users are not blocked if onboarding endpoint is unavailable.
        if (!cancelled) {
          setOnboardingCompleted(true);
        }
      } finally {
        if (!cancelled) {
          setOnboardingLoading(false);
        }
      }
    }

    void checkOnboarding();

    return () => {
      cancelled = true;
    };
  }, [isAuthenticated, location.pathname]);

  if (loading || onboardingLoading) {
    return (
      <div className="flex min-h-screen items-center justify-center">
        <div className="h-8 w-8 animate-spin rounded-full border-4 border-[var(--galdr-accent)] border-t-transparent" />
      </div>
    );
  }

  if (!isAuthenticated) {
    return <Navigate to="/login" replace />;
  }

  if (!onboardingCompleted && !shouldBypassOnboarding(location.pathname)) {
    return <Navigate to={`/onboarding?step=${onboardingStep}`} replace />;
  }

  return <>{children}</>;
}
