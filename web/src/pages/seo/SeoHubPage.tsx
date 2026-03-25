import { Link } from "react-router-dom";
import SeoMeta from "@/components/SeoMeta";
import {
  type SeoFamily,
  getHubByFamily,
  getSeoPagesByFamily,
  seoFamilyOrder,
  seoHubs,
  toSeoTitle,
} from "@/lib/seoCatalog";

interface SeoHubPageProps {
  family: SeoFamily;
}

export default function SeoHubPage({ family }: SeoHubPageProps) {
  const hub = getHubByFamily(family);
  const pages = getSeoPagesByFamily(family);

  const structuredData = {
    "@context": "https://schema.org",
    "@type": "CollectionPage",
    name: hub.title,
    description: hub.description,
    url: `https://pulsescore.app${hub.path}`,
    hasPart: pages.map((page) => ({
      "@type": "WebPage",
      name: page.h1,
      url: `https://pulsescore.app${page.path}`,
    })),
  };

  const relatedHubs = seoFamilyOrder
    .map((key) => seoHubs.find((hubEntry) => hubEntry.family === key))
    .filter((hubEntry): hubEntry is NonNullable<typeof hubEntry> => {
      if (!hubEntry) {
        return false;
      }
      return hubEntry.family !== family;
    })
    .slice(0, 4);

  return (
    <div className="galdr-shell min-h-screen px-6 py-12 text-[var(--galdr-fg)] sm:px-10 lg:px-14">
      <SeoMeta
        title={toSeoTitle(hub.title)}
        description={hub.description}
        path={hub.path}
        keywords={[
          `pulsescore ${hub.label.toLowerCase()}`,
          "customer health scoring",
          "b2b saas retention",
        ]}
        structuredData={structuredData}
      />

      <main className="mx-auto max-w-7xl">
        <header className="galdr-card galdr-noise p-8">
          <p className="galdr-kicker px-3 py-1">{hub.label}</p>
          <h1 className="mt-2 text-3xl font-extrabold tracking-tight sm:text-4xl">
            {hub.title}
          </h1>
          <p className="mt-3 max-w-3xl text-sm leading-6 text-[var(--galdr-fg-muted)]">
            {hub.hero} {hub.description}
          </p>
          <div className="mt-5 flex flex-wrap gap-2 text-xs">
            <span className="galdr-pill px-3 py-1">
              {pages.length} pages in this cluster
            </span>
            <span className="galdr-pill px-3 py-1">
              Intent-driven templates
            </span>
            <span className="galdr-pill px-3 py-1">Internal-link ready</span>
          </div>
        </header>

        <section className="mt-10 grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3">
          {pages.map((page) => (
            <article
              key={page.path}
              className="galdr-panel p-5 transition hover:-translate-y-0.5"
            >
              <p className="text-xs font-semibold text-[var(--galdr-accent)]">
                {page.intent}
              </p>
              <h2 className="mt-1 text-lg font-semibold text-[var(--galdr-fg)]">
                <Link to={page.path} className="galdr-link">
                  {page.h1}
                </Link>
              </h2>
              <p className="mt-2 text-sm text-[var(--galdr-fg-muted)]">
                {page.description}
              </p>
              <p className="mt-3 text-xs text-[var(--galdr-fg-muted)]">
                Keyword pattern: {page.keyword}
              </p>
            </article>
          ))}
        </section>

        <section className="galdr-panel mt-14 p-6">
          <h3 className="text-lg font-semibold">Explore related playbooks</h3>
          <div className="mt-4 grid grid-cols-1 gap-3 md:grid-cols-2">
            {relatedHubs.map((relatedHub) => (
              <Link
                key={relatedHub.path}
                to={relatedHub.path}
                className="galdr-card px-4 py-3 text-sm transition hover:-translate-y-0.5"
              >
                <p className="font-semibold">{relatedHub.label}</p>
                <p className="mt-1 text-xs text-[var(--galdr-fg-muted)]">
                  {relatedHub.description}
                </p>
              </Link>
            ))}
          </div>
          <div className="mt-6 flex flex-wrap gap-3">
            <Link
              to="/register"
              className="galdr-button-primary px-4 py-2 text-sm font-semibold"
            >
              Start free
            </Link>
            <Link
              to="/pricing"
              className="galdr-button-secondary px-4 py-2 text-sm font-semibold"
            >
              View pricing
            </Link>
          </div>
        </section>
      </main>
    </div>
  );
}
