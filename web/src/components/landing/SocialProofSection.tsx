import { Quote } from "lucide-react";

const logos = [
  "Acme Cloud",
  "Northwind",
  "Orbit Analytics",
  "Pixel Ops",
  "Summit AI",
  "Atlas CRM",
];

const testimonials = [
  {
    quote:
      "PulseScore helped us prioritize the exact 12 accounts that were quietly slipping. We recovered 4 before renewal.",
    name: "Jamie Patel",
    role: "Head of Customer Success",
    company: "Northwind SaaS",
  },
  {
    quote:
      "We moved from guessing risk in spreadsheets to seeing clear health movement every morning.",
    name: "Morgan Lee",
    role: "Founder",
    company: "Orbit Analytics",
  },
  {
    quote:
      "The Stripe + HubSpot sync made onboarding painless. Our team had actionable signals on day one.",
    name: "Riley Chen",
    role: "RevOps Lead",
    company: "Acme Cloud",
  },
];

const metrics = [
  { value: "500+", label: "customers monitored" },
  { value: "30%", label: "average churn reduction potential" },
  { value: "15 min", label: "time to first score" },
];

export default function SocialProofSection() {
  return (
    <section className="px-6 py-16 sm:px-10 lg:px-14 lg:py-24">
      <div className="mx-auto max-w-7xl">
        <p className="galdr-kicker px-3 py-1">
          Placeholder logos & testimonials for MVP (replace with production
          brand assets)
        </p>

        <div className="mt-5 grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-6">
          {logos.map((logo) => (
            <div
              key={logo}
              className="galdr-panel px-3 py-2 text-center text-sm font-semibold text-[var(--galdr-fg-muted)]"
            >
              {logo}
            </div>
          ))}
        </div>

        <div className="mt-10 grid grid-cols-1 gap-4 lg:grid-cols-3">
          {metrics.map((metric) => (
            <div key={metric.label} className="galdr-card p-6 text-center">
              <p className="text-3xl font-extrabold text-[var(--galdr-accent)]">
                {metric.value}
              </p>
              <p className="mt-1 text-sm text-[var(--galdr-fg-muted)]">
                {metric.label}
              </p>
            </div>
          ))}
        </div>

        <div className="mt-10 grid grid-cols-1 gap-4 lg:grid-cols-3">
          {testimonials.map((testimonial) => (
            <figure
              key={`${testimonial.name}-${testimonial.company}`}
              className="galdr-panel p-6"
            >
              <Quote className="h-5 w-5 text-[var(--galdr-accent)]" />
              <blockquote className="mt-3 text-sm leading-6 text-[var(--galdr-fg-muted)]">
                “{testimonial.quote}”
              </blockquote>
              <figcaption className="mt-4 text-xs text-[var(--galdr-fg-muted)]">
                <span className="font-semibold text-[var(--galdr-fg)]">
                  {testimonial.name}
                </span>
                <span>
                  {" "}
                  · {testimonial.role}, {testimonial.company}
                </span>
              </figcaption>
            </figure>
          ))}
        </div>
      </div>
    </section>
  );
}
