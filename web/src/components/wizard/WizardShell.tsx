import { useState, type ReactNode } from "react";
import WizardProgress from "@/components/wizard/WizardProgress";
import type { OnboardingStepId } from "@/lib/api";

export interface WizardShellStep {
  id: OnboardingStepId;
  label: string;
  content: ReactNode;
  canProceed?: boolean;
  canSkip?: boolean;
  onNext?: () => Promise<boolean | void> | boolean | void;
  onSkip?: () => Promise<boolean | void> | boolean | void;
}

interface WizardShellProps {
  steps: WizardShellStep[];
  currentStepIndex: number;
  completedSteps: OnboardingStepId[];
  skippedSteps: OnboardingStepId[];
  onCurrentStepChange: (index: number) => void;
  onDone?: () => Promise<boolean | void> | boolean | void;
}

export default function WizardShell({
  steps,
  currentStepIndex,
  completedSteps,
  skippedSteps,
  onCurrentStepChange,
  onDone,
}: WizardShellProps) {
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  const isFirst = currentStepIndex <= 0;
  const isLast = currentStepIndex >= steps.length - 1;
  const currentStep = steps[currentStepIndex];

  async function handleNext() {
    if (!currentStep) return;
    if (currentStep.canProceed === false) return;

    setBusy(true);
    setError("");
    try {
      if (currentStep.onNext) {
        const ok = await currentStep.onNext();
        if (ok === false) {
          return;
        }
      }

      if (isLast) {
        if (onDone) {
          const ok = await onDone();
          if (ok === false) {
            return;
          }
        }
        return;
      }

      onCurrentStepChange(currentStepIndex + 1);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Something went wrong.");
    } finally {
      setBusy(false);
    }
  }

  async function handleSkip() {
    if (!currentStep || !currentStep.canSkip || isLast) return;

    setBusy(true);
    setError("");
    try {
      if (currentStep.onSkip) {
        const ok = await currentStep.onSkip();
        if (ok === false) {
          return;
        }
      }
      onCurrentStepChange(currentStepIndex + 1);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Unable to skip step.");
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="mx-auto w-full max-w-5xl">
      <WizardProgress
        steps={steps.map((step) => ({ id: step.id, label: step.label }))}
        currentStepIndex={currentStepIndex}
        completedSteps={completedSteps}
        skippedSteps={skippedSteps}
      />

      {currentStep?.content}

      {error && (
        <div className="galdr-alert-danger mt-4 px-4 py-3 text-sm">{error}</div>
      )}

      <div className="mt-6 flex items-center justify-between">
        <button
          onClick={() => onCurrentStepChange(currentStepIndex - 1)}
          disabled={isFirst || busy}
          className="galdr-button-secondary px-4 py-2 text-sm font-medium disabled:cursor-not-allowed disabled:opacity-50"
        >
          Back
        </button>

        <div className="flex items-center gap-2">
          {currentStep?.canSkip && !isLast && (
            <button
              onClick={handleSkip}
              disabled={busy}
              className="galdr-alert-warning px-4 py-2 text-sm font-medium disabled:cursor-not-allowed disabled:opacity-50"
            >
              Skip for now
            </button>
          )}
          <button
            onClick={handleNext}
            disabled={busy || currentStep?.canProceed === false}
            className="galdr-button-primary px-4 py-2 text-sm font-medium disabled:cursor-not-allowed disabled:opacity-50"
          >
            {busy ? "Working..." : isLast ? "Done" : "Next"}
          </button>
        </div>
      </div>
    </div>
  );
}
