import { Link } from "react-router-dom";
import SeoMeta from "@/components/SeoMeta";

export default function TermsPage() {
  return (
    <main className="galdr-shell min-h-screen px-6 py-12 sm:px-10 lg:px-14">
      <SeoMeta
        title="Terms of Service | PulseScore"
        description="Read the PulseScore terms of service and acceptable use guidelines."
        path="/terms"
        noIndex
      />

      <div className="galdr-card galdr-noise mx-auto max-w-3xl p-8">
        <h1 className="text-3xl font-bold text-[var(--galdr-fg)]">
          Terms of Service
        </h1>
        <p className="mt-2 text-sm text-[var(--galdr-fg-muted)]">
          Last updated: February 24, 2026
        </p>

        <div className="mt-6 space-y-5 text-sm leading-7 text-[var(--galdr-fg-muted)]">
          <p>
            This is placeholder terms content for MVP use. Replace with
            legal-reviewed terms before production launch.
          </p>
          <p>
            By using PulseScore, you agree to provide accurate account
            information, comply with applicable laws, and avoid misuse of
            integrations.
          </p>
          <p>
            Service availability targets and support terms may vary by
            subscription tier.
          </p>
        </div>

        <Link
          to="/"
          className="galdr-button-primary mt-8 inline-flex px-4 py-2 text-sm font-semibold"
        >
          Back to home
        </Link>
      </div>
    </main>
  );
}
