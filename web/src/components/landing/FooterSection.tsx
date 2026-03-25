import { type FormEvent, useState } from "react";
import { Link } from "react-router-dom";

const footerGroups = [
  {
    title: "Product",
    links: [
      { label: "Features", href: "/#features" },
      { label: "Pricing", href: "/#pricing" },
      { label: "Dashboard", href: "/dashboard" },
    ],
  },
  {
    title: "Company",
    links: [
      { label: "About", href: "/#" },
      { label: "Blog", href: "/#" },
      { label: "Careers", href: "/#" },
    ],
  },
  {
    title: "Resources",
    links: [
      { label: "Templates", href: "/templates" },
      { label: "Integrations", href: "/integrations" },
      { label: "Comparisons", href: "/compare" },
      { label: "Glossary", href: "/glossary" },
    ],
  },
  {
    title: "Legal",
    links: [
      { label: "Privacy", href: "/privacy" },
      { label: "Terms", href: "/terms" },
      { label: "Cookies", href: "/privacy#cookies" },
    ],
  },
  {
    title: "Support",
    links: [
      { label: "Help", href: "/#" },
      { label: "Contact", href: "mailto:support@pulsescore.app" },
      { label: "Status", href: "/#" },
    ],
  },
];

export default function FooterSection() {
  const [email, setEmail] = useState("");
  const [subscribed, setSubscribed] = useState(false);

  function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();

    if (!email.trim()) {
      return;
    }

    setSubscribed(true);
    setEmail("");
  }

  return (
    <footer className="border-t border-[var(--galdr-border)] bg-[color:rgb(18_18_26_/_0.72)] px-6 py-14 sm:px-10 lg:px-14">
      <div className="mx-auto max-w-7xl">
        <div className="grid grid-cols-1 gap-10 lg:grid-cols-[1.1fr_1fr]">
          <div>
            <h3 className="text-2xl font-bold text-[var(--galdr-fg)]">
              PulseScore
            </h3>
            <p className="mt-3 max-w-md text-sm leading-6 text-[var(--galdr-fg-muted)]">
              Customer health scoring for B2B SaaS teams that need to move
              faster than churn.
            </p>

            <form
              onSubmit={handleSubmit}
              className="mt-5 flex max-w-md flex-col gap-2 sm:flex-row"
            >
              <input
                type="email"
                placeholder="you@company.com"
                value={email}
                onChange={(event) => setEmail(event.target.value)}
                className="galdr-input w-full px-4 py-2.5 text-sm outline-none"
              />
              <button
                type="submit"
                className="galdr-button-primary px-4 py-2.5 text-sm font-semibold"
              >
                Join updates
              </button>
            </form>

            {subscribed && (
              <p className="mt-2 text-xs text-[var(--galdr-success)]">
                Thanks — you’re on the list.
              </p>
            )}
          </div>

          <div className="grid grid-cols-2 gap-6 md:grid-cols-5">
            {footerGroups.map((group) => (
              <div key={group.title}>
                <h4 className="text-sm font-semibold text-[var(--galdr-fg)]">
                  {group.title}
                </h4>
                <ul className="mt-3 space-y-2 text-sm text-[var(--galdr-fg-muted)]">
                  {group.links.map((link) => {
                    const isExternal =
                      link.href.startsWith("mailto:") ||
                      link.href.startsWith("http") ||
                      link.href.startsWith("/#");

                    return (
                      <li key={link.label}>
                        {isExternal ? (
                          <a href={link.href} className="galdr-link">
                            {link.label}
                          </a>
                        ) : (
                          <Link to={link.href} className="galdr-link">
                            {link.label}
                          </Link>
                        )}
                      </li>
                    );
                  })}
                </ul>
              </div>
            ))}
          </div>
        </div>

        <div className="mt-12 flex flex-col gap-4 border-t border-[var(--galdr-border)] pt-5 text-sm text-[var(--galdr-fg-muted)] sm:flex-row sm:items-center sm:justify-between">
          <p>© {new Date().getFullYear()} PulseScore. All rights reserved.</p>
          <div className="flex items-center gap-4">
            <a
              href="https://x.com"
              target="_blank"
              rel="noreferrer"
              className="galdr-link"
            >
              X
            </a>
            <a
              href="https://github.com"
              target="_blank"
              rel="noreferrer"
              className="galdr-link"
            >
              GitHub
            </a>
            <a
              href="https://www.linkedin.com"
              target="_blank"
              rel="noreferrer"
              className="galdr-link"
            >
              LinkedIn
            </a>
          </div>
        </div>
      </div>
    </footer>
  );
}
