import type { OnboardingStepId } from "@/lib/api";

interface WizardProgressProps {
  steps: Array<{ id: OnboardingStepId; label: string }>;
  currentStepIndex: number;
  completedSteps: OnboardingStepId[];
  skippedSteps: OnboardingStepId[];
}

function statusFor(
  index: number,
  stepId: OnboardingStepId,
  currentStepIndex: number,
  completedSteps: Set<OnboardingStepId>,
  skippedSteps: Set<OnboardingStepId>,
) {
  if (completedSteps.has(stepId)) return "completed";
  if (skippedSteps.has(stepId)) return "skipped";
  if (index === currentStepIndex) return "active";
  if (index < currentStepIndex) return "past";
  return "upcoming";
}

export default function WizardProgress({
  steps,
  currentStepIndex,
  completedSteps,
  skippedSteps,
}: WizardProgressProps) {
  const completedStepSet = new Set(completedSteps);
  const skippedStepSet = new Set(skippedSteps);

  return (
    <ol className="mb-6 grid grid-cols-1 gap-3 sm:grid-cols-5">
      {steps.map((step, idx) => {
        const status = statusFor(
          idx,
          step.id,
          currentStepIndex,
          completedStepSet,
          skippedStepSet,
        );

        const tone =
          status === "completed"
            ? "border-[color:rgb(52_211_153_/_0.45)] bg-[color:rgb(52_211_153_/_0.14)] text-[var(--galdr-success)]"
            : status === "skipped"
              ? "border-[color:rgb(245_158_11_/_0.45)] bg-[color:rgb(245_158_11_/_0.14)] text-[var(--galdr-at-risk)]"
              : status === "active"
                ? "border-[color:rgb(139_92_246_/_0.45)] bg-[color:rgb(139_92_246_/_0.16)] text-[var(--galdr-accent)]"
                : status === "past"
                  ? "border-[var(--galdr-border)] bg-[color-mix(in_srgb,var(--galdr-surface-soft)_82%,black_18%)] text-[var(--galdr-fg-muted)]"
                  : "border-[var(--galdr-border)] bg-[color-mix(in_srgb,var(--galdr-bg-elevated)_82%,black_18%)] text-[var(--galdr-fg-muted)]";

        return (
          <li
            key={step.id}
            className={`rounded-lg border px-3 py-2 text-xs font-medium sm:text-sm ${tone}`}
          >
            <span className="mr-2 inline-flex h-5 w-5 items-center justify-center rounded-full border border-current text-[11px]">
              {idx + 1}
            </span>
            {step.label}
          </li>
        );
      })}
    </ol>
  );
}
