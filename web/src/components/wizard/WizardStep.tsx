import type { ReactNode } from "react";

interface WizardStepProps {
  title: string;
  description?: string;
  children: ReactNode;
}

export default function WizardStep({
  title,
  description,
  children,
}: WizardStepProps) {
  return (
    <section className="galdr-card p-6">
      <header className="mb-6">
        <h2 className="text-xl font-semibold text-[var(--galdr-fg)]">
          {title}
        </h2>
        {description && (
          <p className="mt-2 text-sm text-[var(--galdr-fg-muted)]">
            {description}
          </p>
        )}
      </header>

      <div>{children}</div>
    </section>
  );
}
