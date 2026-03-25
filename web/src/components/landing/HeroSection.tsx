import { ArrowRight, BarChart3, ShieldCheck, Zap } from "lucide-react";
import { Link } from "react-router-dom";

export default function HeroSection() {
  return (
    <section className="relative overflow-hidden px-6 pb-16 pt-14 sm:px-10 lg:px-14 lg:pb-24 lg:pt-20">
      <div className="pointer-events-none absolute -left-24 top-0 h-80 w-80 rounded-full bg-[color:rgb(139_92_246_/_0.22)] blur-3xl" />
      <div className="pointer-events-none absolute -right-28 bottom-0 h-80 w-80 rounded-full bg-[color:rgb(34_211_238_/_0.18)] blur-3xl" />

      <div className="relative mx-auto grid w-full max-w-7xl gap-12 lg:grid-cols-[1.1fr_1fr] lg:items-center">
        <div>
          <span className="galdr-kicker px-3 py-1">
            Norse-grade customer health intelligence
          </span>
          <h1 className="mt-5 text-4xl font-bold tracking-tight text-[var(--galdr-fg)] sm:text-5xl lg:text-6xl">
            Cast retention signals before churn drains your ARR.
          </h1>
          <p className="mt-5 max-w-2xl text-lg text-[var(--galdr-fg-muted)]">
            Galdr connects Stripe, HubSpot, and Intercom to surface at-risk
            accounts in minutes, giving your team a precise ritual for saving
            revenue before renewal risk spikes.
          </p>

          <div className="mt-8 flex flex-wrap gap-3">
            <Link
              to="/register"
              className="galdr-button-primary inline-flex items-center gap-2 px-5 py-3 text-sm font-semibold"
            >
              Start free
              <ArrowRight className="h-4 w-4" />
            </Link>
            <Link
              to="/login"
              className="galdr-button-secondary inline-flex items-center px-5 py-3 text-sm font-semibold"
            >
              Sign in
            </Link>
          </div>

          <div className="mt-8 grid max-w-xl grid-cols-1 gap-3 text-sm text-[var(--galdr-fg-muted)] sm:grid-cols-3">
            <div className="galdr-panel px-3 py-2">
              <Zap className="mb-1 h-4 w-4 text-[var(--galdr-accent)]" />
              15-minute setup
            </div>
            <div className="galdr-panel px-3 py-2">
              <BarChart3 className="mb-1 h-4 w-4 text-[var(--galdr-accent)]" />
              Real-time scoring
            </div>
            <div className="galdr-panel px-3 py-2">
              <ShieldCheck className="mb-1 h-4 w-4 text-[var(--galdr-accent)]" />
              Privacy-first
            </div>
          </div>
        </div>

        <div className="relative">
          <div className="galdr-card p-4">
            <div className="galdr-panel p-4">
              <div className="mb-4 flex items-center justify-between">
                <p className="text-sm font-semibold text-[var(--galdr-fg)]">
                  Health Score Snapshot
                </p>
                <span className="rounded-full border border-[color:rgb(52_211_153_/_0.45)] bg-[color:rgb(52_211_153_/_0.12)] px-2 py-1 text-xs font-semibold text-[var(--galdr-success)]">
                  Live
                </span>
              </div>

              <div className="space-y-3">
                {[
                  {
                    name: "Northwind SaaS",
                    score: 89,
                    tone: "from-emerald-400 to-emerald-600 dark:from-emerald-500 dark:to-emerald-400",
                  },
                  {
                    name: "Orbit Analytics",
                    score: 64,
                    tone: "from-amber-400 to-amber-600 dark:from-amber-500 dark:to-amber-400",
                  },
                  {
                    name: "Acme Ops",
                    score: 37,
                    tone: "from-rose-400 to-rose-600 dark:from-rose-500 dark:to-rose-400",
                  },
                ].map((customer) => (
                  <div
                    key={customer.name}
                    className="rounded-lg border border-[var(--galdr-border)] bg-[var(--galdr-bg-elevated)] p-3"
                  >
                    <div className="mb-2 flex items-center justify-between text-xs text-[var(--galdr-fg-muted)]">
                      <span>{customer.name}</span>
                      <span className="font-semibold">
                        {customer.score}/100
                      </span>
                    </div>
                    <div className="h-2 rounded-full bg-[var(--galdr-surface-soft)]">
                      <div
                        className={`h-2 rounded-full bg-gradient-to-r ${customer.tone}`}
                        style={{ width: `${customer.score}%` }}
                      />
                    </div>
                  </div>
                ))}
              </div>
            </div>
          </div>
        </div>
      </div>
    </section>
  );
}
