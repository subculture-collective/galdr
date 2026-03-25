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
    <footer className="border-t border-gray-200 bg-white px-6 py-14 dark:border-gray-800 dark:bg-gray-950 sm:px-10 lg:px-14">
      <div className="mx-auto max-w-7xl">
        <div className="grid grid-cols-1 gap-10 lg:grid-cols-[1.1fr_1fr]">
          <div>
            <h3 className="text-2xl font-bold text-gray-900 dark:text-gray-100">
              PulseScore
            </h3>
            <p className="mt-3 max-w-md text-sm leading-6 text-gray-600 dark:text-gray-300">
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
                className="w-full rounded-xl border border-gray-300 bg-white px-4 py-2.5 text-sm text-gray-900 outline-none transition focus:border-indigo-500 dark:border-gray-700 dark:bg-gray-900 dark:text-gray-100"
              />
              <button
                type="submit"
                className="rounded-xl bg-indigo-600 px-4 py-2.5 text-sm font-semibold text-white hover:bg-indigo-700"
              >
                Join updates
              </button>
            </form>

            {subscribed && (
              <p className="mt-2 text-xs text-emerald-600 dark:text-emerald-400">
                Thanks — you’re on the list.
              </p>
            )}
          </div>

          <div className="grid grid-cols-2 gap-6 md:grid-cols-5">
            {footerGroups.map((group) => (
              <div key={group.title}>
                <h4 className="text-sm font-semibold text-gray-900 dark:text-gray-100">
                  {group.title}
                </h4>
                <ul className="mt-3 space-y-2 text-sm text-gray-600 dark:text-gray-300">
                  {group.links.map((link) => {
                    const isExternal =
                      link.href.startsWith("mailto:") ||
                      link.href.startsWith("http") ||
                      link.href.startsWith("/#");

                    return (
                      <li key={link.label}>
                        {isExternal ? (
                          <a
                            href={link.href}
                            className="hover:text-indigo-600 dark:hover:text-indigo-300"
                          >
                            {link.label}
                          </a>
                        ) : (
                          <Link
                            to={link.href}
                            className="hover:text-indigo-600 dark:hover:text-indigo-300"
                          >
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

        <div className="mt-12 flex flex-col gap-4 border-t border-gray-200 pt-5 text-sm text-gray-500 dark:border-gray-800 dark:text-gray-400 sm:flex-row sm:items-center sm:justify-between">
          <p>© {new Date().getFullYear()} PulseScore. All rights reserved.</p>
          <div className="flex items-center gap-4">
            <a
              href="https://x.com"
              target="_blank"
              rel="noreferrer"
              className="hover:text-indigo-600 dark:hover:text-indigo-300"
            >
              X
            </a>
            <a
              href="https://github.com"
              target="_blank"
              rel="noreferrer"
              className="hover:text-indigo-600 dark:hover:text-indigo-300"
            >
              GitHub
            </a>
            <a
              href="https://www.linkedin.com"
              target="_blank"
              rel="noreferrer"
              className="hover:text-indigo-600 dark:hover:text-indigo-300"
            >
              LinkedIn
            </a>
          </div>
        </div>
      </div>
    </footer>
  );
}
