import { Link } from "react-router-dom";
import SeoMeta from "@/components/SeoMeta";

export default function PrivacyPage() {
  return (
    <main className="galdr-shell min-h-screen px-6 py-12 sm:px-10 lg:px-14">
      <SeoMeta
        title="Privacy Policy | PulseScore"
        description="Read the PulseScore privacy policy and data handling practices."
        path="/privacy"
        noIndex
      />

      <div className="galdr-card galdr-noise mx-auto max-w-3xl p-8">
        <h1 className="text-3xl font-bold text-[var(--galdr-fg)]">
          Privacy Policy
        </h1>
        <p className="mt-2 text-sm text-[var(--galdr-fg-muted)]">
          Last updated: February 24, 2026
        </p>

        <div className="mt-6 space-y-5 text-sm leading-7 text-[var(--galdr-fg-muted)]">
          <p>
            This is placeholder policy content for MVP use. Replace with
            legal-reviewed copy before production launch.
          </p>
          <p>
            PulseScore collects account profile information and integration data
            only for product functionality, analytics, and support.
          </p>
          <p id="cookies">
            We use essential cookies for authentication and optional analytics
            cookies for product improvement.
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
