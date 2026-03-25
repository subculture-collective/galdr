import { Link, useParams } from "react-router-dom";
import SeoMeta from "@/components/SeoMeta";
import {
  type SeoFamily,
  type SeoPage,
  getCrossFamilyHubs,
  getHubByFamily,
  getRelatedPages,
  getSeoPageByFamilyAndSlug,
} from "@/lib/seoCatalog";

interface SeoProgrammaticPageProps {
  family: SeoFamily;
}

function getSchemaType(family: SeoFamily): "article" | "website" {
  if (family === "glossary" || family === "examples") {
    return "article";
  }
  return "website";
}

function hashString(value: string): number {
  let hash = 0;
  for (let index = 0; index < value.length; index += 1) {
    hash = (hash * 31 + value.charCodeAt(index)) >>> 0;
  }
  return hash;
}

function pickBySeed<T>(values: T[], seed: number, offset = 0): T {
  return values[(seed + offset) % values.length];
}

function getFamilyContext(family: SeoFamily): {
  owner: string;
  northStar: string;
  dataSource: string;
  hazard: string;
} {
  switch (family) {
    case "templates":
      return {
        owner: "Customer Success leadership",
        northStar: "retention lift",
        dataSource: "health score model inputs",
        hazard: "score inflation",
      };
    case "integrations":
      return {
        owner: "RevOps and data owners",
        northStar: "signal freshness",
        dataSource: "billing, CRM, and support events",
        hazard: "duplicate or stale events",
      };
    case "personas":
      return {
        owner: "cross-functional account owners",
        northStar: "time-to-intervention",
        dataSource: "role-specific dashboards",
        hazard: "one-size-fits-all workflows",
      };
    case "comparisons":
      return {
        owner: "operator stakeholders",
        northStar: "decision clarity",
        dataSource: "evaluation criteria and implementation constraints",
        hazard: "feature checklist bias",
      };
    case "glossary":
      return {
        owner: "CS and RevOps enablement",
        northStar: "terminology consistency",
        dataSource: "shared operating definitions",
        hazard: "definition drift",
      };
    case "examples":
      return {
        owner: "playbook operators",
        northStar: "execution velocity",
        dataSource: "historical intervention outcomes",
        hazard: "copy-paste execution",
      };
    case "curation":
      return {
        owner: "tooling decision makers",
        northStar: "fit-to-stage score",
        dataSource: "pricing, onboarding effort, and integration depth",
        hazard: "tool sprawl",
      };
    default:
      return {
        owner: "operations owners",
        northStar: "retention outcomes",
        dataSource: "health signals",
        hazard: "unclear ownership",
      };
  }
}

function buildSections(page: SeoPage): {
  summary: string;
  outcomes: string[];
  implementation: string[];
  pitfalls: string[];
} {
  const seed = hashString(`${page.family}:${page.slug}:${page.keyword}`);
  const context = getFamilyContext(page.family);

  const cadence = pickBySeed(["weekly", "bi-weekly", "monthly"], seed, 1);
  const reviewWindow = pickBySeed(["14-day", "30-day", "quarterly"], seed, 2);
  const interventionChannel = pickBySeed(
    ["in-app tasks", "Slack escalation", "CSM action queue", "QBR follow-up"],
    seed,
    3,
  );
  const evidenceArtifact = pickBySeed(
    [
      "renewal cohort outcomes",
      "risk-resolution lag",
      "expansion conversion",
      "retention trend deltas",
    ],
    seed,
    4,
  );

  const keywordAnchor = page.keyword.split(" ").slice(0, 4).join(" ");

  const summary = `Teams searching "${page.keyword}" usually need a repeatable way to move from signal to action. This guide turns ${page.entity} into a ${cadence} workflow owned by ${context.owner}.`;

  const baseOutcomes = [
    `Tie ${page.entity.toLowerCase()} to one decision metric (${context.northStar}) so prioritization stays explicit.`,
    `Anchor this page to intent around "${keywordAnchor}" and connect it to ${context.dataSource}.`,
  ];

  const baseImplementation = [
    `Audit ${context.dataSource} freshness first, then set a ${reviewWindow} calibration rhythm for ${page.entity.toLowerCase()}.`,
    `Define trigger → owner (${context.owner}) → response paths through ${interventionChannel}.`,
  ];

  const basePitfalls = [
    `Launching ${page.entity.toLowerCase()} without explicit ownership from ${context.owner}.`,
    `Skipping recalibration after each ${reviewWindow} cycle, which weakens ${context.northStar}.`,
  ];

  let familyOutcome = "";
  let familyImplementation = "";
  let familyPitfall = "";

  switch (page.family) {
    case "templates":
      familyOutcome =
        "Standardize scorecards across segments so account reviews focus on intervention priority, not score debate.";
      familyImplementation =
        "Publish one template changelog so teams can trace threshold edits to retention impact.";
      familyPitfall =
        "Treating templates as static docs instead of operational controls that evolve with customer behavior.";
      break;
    case "integrations":
      familyOutcome =
        "Unify Stripe, CRM, and support events into a single health timeline for faster triage.";
      familyImplementation =
        "Document field mappings and sync ownership before automating downstream alerts.";
      familyPitfall =
        "Shipping integrations without deduplication rules, which inflates churn-risk alerts.";
      break;
    case "personas":
      familyOutcome =
        "Align dashboards to role-specific outcomes so each team sees only the actions they control.";
      familyImplementation =
        "Add role-based intervention SLAs for onboarding, adoption, renewal, and expansion phases.";
      familyPitfall =
        "Using one global dashboard for all roles, causing weak accountability and slower interventions.";
      break;
    case "comparisons":
      familyOutcome =
        "Compare options on operator effort and data fit, not just feature breadth.";
      familyImplementation =
        "Run a short pilot with one success metric before committing to a platform migration.";
      familyPitfall =
        "Optimizing for enterprise feature depth your current team cannot operationalize.";
      break;
    case "glossary":
      familyOutcome =
        "Create shared definitions so CS, RevOps, and leadership interpret risk signals consistently.";
      familyImplementation =
        "Pair each term with one practical example and one operating decision it should influence.";
      familyPitfall =
        "Maintaining definitions separately across teams, which causes interpretation drift.";
      break;
    case "examples":
      familyOutcome =
        "Use concrete examples to shorten implementation time and improve intervention quality.";
      familyImplementation =
        "Tag examples by lifecycle stage so teams can reuse patterns with minimal rework.";
      familyPitfall =
        "Copying examples verbatim without adjusting thresholds for segment, ACV, and product usage.";
      break;
    case "curation":
      familyOutcome =
        "Evaluate tooling with transparent criteria so stakeholders can reproduce selection decisions.";
      familyImplementation =
        "Re-score candidates quarterly against pricing, onboarding effort, and integration depth.";
      familyPitfall =
        "Ranking tools on popularity alone while ignoring migration and change-management overhead.";
      break;
    default:
      familyOutcome =
        "Align teams on a single operating model for health scoring execution.";
      familyImplementation =
        "Track one leading and one lagging indicator to validate intervention quality over time.";
      familyPitfall =
        "Treating this page as reference-only content instead of a decision-support workflow.";
      break;
  }

  return {
    summary,
    outcomes: [...baseOutcomes, familyOutcome],
    implementation: [
      ...baseImplementation,
      familyImplementation,
      `Keep ${evidenceArtifact} visible in your monthly review so the workflow stays outcome-driven.`,
    ],
    pitfalls: [
      ...basePitfalls,
      familyPitfall,
      `Ignoring ${context.hazard} controls turns scoring into noise and slows response quality.`,
    ],
  };
}

export default function SeoProgrammaticPage({
  family,
}: SeoProgrammaticPageProps) {
  const params = useParams<{ slug: string }>();
  const slug = params.slug ?? "";

  const page = getSeoPageByFamilyAndSlug(family, slug);
  const hub = getHubByFamily(family);

  if (!page) {
    return (
      <div className="galdr-shell mx-auto flex min-h-[60vh] max-w-3xl flex-col items-center justify-center px-6 text-center">
        <SeoMeta
          title="Resource not found | PulseScore"
          description="The requested SEO resource could not be found."
          path={`${hub.path}/${slug}`}
          noIndex
        />
        <h1 className="text-2xl font-bold">Resource not found</h1>
        <p className="mt-2 text-sm text-[var(--galdr-fg-muted)]">
          This page may have moved. Explore the full {hub.label.toLowerCase()}{" "}
          library instead.
        </p>
        <Link
          to={hub.path}
          className="galdr-button-primary mt-6 px-4 py-2 text-sm font-semibold"
        >
          Go to {hub.label}
        </Link>
      </div>
    );
  }

  const sections = buildSections(page);
  const relatedPages = getRelatedPages(page, 6);
  const crossFamilyHubs = getCrossFamilyHubs(family);

  const structuredData = [
    {
      "@context": "https://schema.org",
      "@type": "BreadcrumbList",
      itemListElement: [
        {
          "@type": "ListItem",
          position: 1,
          name: "Home",
          item: "https://pulsescore.app/",
        },
        {
          "@type": "ListItem",
          position: 2,
          name: hub.label,
          item: `https://pulsescore.app${hub.path}`,
        },
        {
          "@type": "ListItem",
          position: 3,
          name: page.h1,
          item: `https://pulsescore.app${page.path}`,
        },
      ],
    },
    {
      "@context": "https://schema.org",
      "@type": page.family === "glossary" ? "DefinedTerm" : "WebPage",
      name: page.h1,
      url: `https://pulsescore.app${page.path}`,
      description: page.description,
      keywords: [page.keyword, "customer health score", "saas retention"],
    },
  ];

  return (
    <div className="galdr-shell min-h-screen px-6 py-12 text-[var(--galdr-fg)] sm:px-10 lg:px-14">
      <SeoMeta
        title={page.title}
        description={page.description}
        path={page.path}
        type={getSchemaType(page.family)}
        keywords={[page.keyword, page.entity, "customer health scoring"]}
        structuredData={structuredData}
      />

      <main className="mx-auto max-w-5xl">
        <nav className="mb-6 text-xs text-[var(--galdr-fg-muted)]">
          <Link to="/" className="galdr-link">
            Home
          </Link>
          <span className="mx-2">/</span>
          <Link to={hub.path} className="galdr-link">
            {hub.label}
          </Link>
          <span className="mx-2">/</span>
          <span>{page.entity}</span>
        </nav>

        <header className="galdr-card galdr-noise p-8">
          <p className="galdr-kicker px-3 py-1">
            {hub.label} · {page.intent}
          </p>
          <h1 className="mt-2 text-3xl font-extrabold tracking-tight sm:text-4xl">
            {page.h1}
          </h1>
          <p className="mt-3 text-sm leading-7 text-[var(--galdr-fg-muted)]">
            {sections.summary}
          </p>
          <div className="mt-4 flex flex-wrap gap-2 text-xs">
            <span className="galdr-pill px-3 py-1">
              Primary keyword: {page.keyword}
            </span>
            <span className="galdr-pill px-3 py-1">
              Last updated: 2026-02-24
            </span>
          </div>
        </header>

        <section className="mt-8 grid gap-4 md:grid-cols-3">
          {sections.outcomes.map((outcome) => (
            <article key={outcome} className="galdr-panel p-4 text-sm">
              {outcome}
            </article>
          ))}
        </section>

        <section className="mt-10 grid gap-8 lg:grid-cols-2">
          <div className="galdr-panel p-6">
            <h2 className="text-xl font-semibold">Implementation blueprint</h2>
            <ol className="mt-4 list-decimal space-y-2 pl-5 text-sm leading-6 text-[var(--galdr-fg-muted)]">
              {sections.implementation.map((step) => (
                <li key={step}>{step}</li>
              ))}
            </ol>
          </div>

          <div className="galdr-panel p-6">
            <h2 className="text-xl font-semibold">Common pitfalls to avoid</h2>
            <ul className="mt-4 list-disc space-y-2 pl-5 text-sm leading-6 text-[var(--galdr-fg-muted)]">
              {sections.pitfalls.map((pitfall) => (
                <li key={pitfall}>{pitfall}</li>
              ))}
            </ul>
          </div>
        </section>

        <section className="galdr-card mt-10 p-6">
          <h2 className="text-xl font-semibold">Operational next step</h2>
          <p className="mt-2 text-sm text-[var(--galdr-fg-muted)]">
            Start with this workflow in a lightweight template, then connect
            live billing, CRM, and support signals in PulseScore to automate
            account prioritization.
          </p>
          <div className="mt-5 flex flex-wrap gap-3">
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
              Compare plans
            </Link>
          </div>
        </section>

        <section className="mt-12">
          <h2 className="text-xl font-semibold">Related pages</h2>
          <div className="mt-4 grid grid-cols-1 gap-3 md:grid-cols-2">
            {relatedPages.map((related) => (
              <Link
                key={related.path}
                to={related.path}
                className="galdr-panel p-4 text-sm transition hover:-translate-y-0.5"
              >
                <p className="font-semibold">{related.h1}</p>
                <p className="mt-1 text-xs text-[var(--galdr-fg-muted)]">
                  {related.keyword}
                </p>
              </Link>
            ))}
          </div>
        </section>

        <section className="mt-12">
          <h2 className="text-xl font-semibold">Explore other playbooks</h2>
          <div className="mt-4 grid grid-cols-1 gap-3 md:grid-cols-2">
            {crossFamilyHubs.map((relatedHub) => (
              <Link
                key={relatedHub.path}
                to={relatedHub.path}
                className="galdr-panel p-4 text-sm transition hover:-translate-y-0.5"
              >
                <p className="font-semibold">{relatedHub.label}</p>
                <p className="mt-1 text-xs text-[var(--galdr-fg-muted)]">
                  {relatedHub.description}
                </p>
              </Link>
            ))}
          </div>
        </section>
      </main>
    </div>
  );
}
