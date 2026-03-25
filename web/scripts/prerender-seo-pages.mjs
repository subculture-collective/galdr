import { mkdir, readFile, writeFile } from "node:fs/promises";

const SITE_ORIGIN = "https://pulsescore.app";

const FAMILY_CONFIG = {
  templates: {
    path: "/templates",
    label: "Templates",
    hubTitle: "Customer health templates for lean SaaS teams",
    hubDescription:
      "Action-ready customer health templates you can use immediately, then operationalize in PulseScore.",
  },
  integrations: {
    path: "/integrations",
    label: "Integrations",
    hubTitle: "Integration playbooks for customer health scoring",
    hubDescription:
      "Connect billing, CRM, and support signals to unify churn-risk visibility in one workflow.",
  },
  personas: {
    path: "/for",
    label: "Personas",
    hubTitle: "PulseScore by persona and growth stage",
    hubDescription:
      "See how founders, CS, and RevOps teams tailor health scoring to their operating model.",
  },
  comparisons: {
    path: "/compare",
    label: "Comparisons",
    hubTitle: "PulseScore comparison guides",
    hubDescription:
      "Balanced comparisons focused on setup speed, pricing, and fit for lean teams.",
  },
  glossary: {
    path: "/glossary",
    label: "Glossary",
    hubTitle: "Customer success and churn glossary",
    hubDescription:
      "Clear, practical definitions with examples and implementation context for SaaS operators.",
  },
  examples: {
    path: "/examples",
    label: "Examples",
    hubTitle: "Real customer health examples",
    hubDescription:
      "Concrete examples of health scores, thresholds, alerts, and intervention patterns.",
  },
  curation: {
    path: "/best",
    label: "Best-of Guides",
    hubTitle: "Curated best-of customer success guides",
    hubDescription:
      "Research-backed shortlists of tools and approaches by team size, stack, and outcomes.",
  },
};

const FAMILY_ORDER = [
  "templates",
  "integrations",
  "personas",
  "comparisons",
  "examples",
  "curation",
  "glossary",
];

const CORE_PAGES = [
  {
    path: "/",
    title: "PulseScore | Know Customer Health Before They Churn",
    description:
      "Connect Stripe, HubSpot, and Intercom to monitor customer health and reduce churn with proactive alerts.",
    keywords: [
      "customer health score",
      "churn prevention",
      "b2b saas",
      "customer success software",
    ],
    h1: "Know customer risk before churn hits your MRR.",
    robots: "index, follow",
  },
  {
    path: "/pricing",
    title: "PulseScore Pricing",
    description:
      "Compare PulseScore Free, Growth, and Scale plans with monthly and annual billing options.",
    keywords: ["pulsescore pricing", "customer success pricing", "b2b saas pricing"],
    h1: "Simple pricing for lean customer success teams",
    robots: "index, follow",
  },
  {
    path: "/privacy",
    title: "Privacy Policy | PulseScore",
    description: "Read the PulseScore privacy policy and data handling practices.",
    keywords: ["privacy policy", "data handling", "pulsescore"],
    h1: "Privacy Policy",
    robots: "noindex, follow",
  },
  {
    path: "/terms",
    title: "Terms of Service | PulseScore",
    description:
      "Read the PulseScore terms of service and acceptable use guidelines.",
    keywords: ["terms of service", "acceptable use", "pulsescore"],
    h1: "Terms of Service",
    robots: "noindex, follow",
  },
];

function escapeHtml(value) {
  return String(value)
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
    .replaceAll("'", "&#39;");
}

function withBrandTitle(base) {
  const suffix = " | PulseScore";
  const maxLength = 60;
  const full = `${base}${suffix}`;

  if (full.length <= maxLength) {
    return full;
  }

  const available = maxLength - suffix.length;
  const truncatedBase =
    available > 1
      ? `${base.slice(0, available - 1).trimEnd()}…`
      : base.slice(0, available);

  return `${truncatedBase}${suffix}`;
}

function buildTitle(page) {
  switch (page.family) {
    case "templates":
      return withBrandTitle(`${page.entity} Template`);
    case "integrations":
      return withBrandTitle(`PulseScore + ${page.entity} Integration`);
    case "personas":
      return withBrandTitle(`Health Scoring for ${page.entity}`);
    case "comparisons":
      return withBrandTitle(`${page.entity} Comparison`);
    case "glossary":
      return withBrandTitle(`${page.entity} Definition`);
    case "examples":
      return withBrandTitle(`${page.entity} Examples`);
    case "curation":
      return withBrandTitle(page.entity);
    default:
      return withBrandTitle(page.entity);
  }
}

function buildDescription(page) {
  switch (page.family) {
    case "templates":
      return `Use this ${page.entity.toLowerCase()} template to prioritize risk, define thresholds, and run a repeatable retention workflow for B2B SaaS.`;
    case "integrations":
      return `Learn how ${page.entity} can feed customer health signals into PulseScore so your team can identify churn risk earlier and act faster.`;
    case "personas":
      return `See how ${page.entity.toLowerCase()} use PulseScore to monitor account health, reduce churn, and focus effort where it drives revenue retention.`;
    case "comparisons":
      return `Compare ${page.entity} across pricing, setup speed, integrations, and fit so lean customer success teams can choose with confidence.`;
    case "glossary":
      return `Understand ${page.entity.toLowerCase()} with clear definitions, formulas, examples, and practical guidance for SaaS customer success workflows.`;
    case "examples":
      return `Explore ${page.entity.toLowerCase()} with practical patterns, implementation notes, and proven workflows for retention-focused teams.`;
    case "curation":
      return `Review ${page.entity.toLowerCase()} using transparent criteria: setup complexity, integration depth, pricing, and operational fit for SMB SaaS.`;
    default:
      return `${page.entity} resources and implementation guidance from PulseScore.`;
  }
}

function buildPath(page) {
  return `${FAMILY_CONFIG[page.family].path}/${page.slug}`;
}

function buildHeading(page) {
  switch (page.family) {
    case "templates":
      return `Free ${page.entity} Template for B2B SaaS`;
    case "integrations":
      return `PulseScore + ${page.entity}`;
    case "personas":
      return `PulseScore for ${page.entity}`;
    case "comparisons":
      return page.entity;
    case "glossary":
      return `What is ${page.entity}?`;
    default:
      return page.entity;
  }
}

function toAbsolute(path) {
  return `${SITE_ORIGIN}${path}`;
}

function getAssetTags(indexHtml) {
  const cssTags = [];
  const scriptTags = [];

  const cssRegex = /<link[^>]+rel="stylesheet"[^>]+href="([^"]+)"[^>]*>/g;
  const scriptRegex = /<script[^>]+type="module"[^>]+src="([^"]+)"[^>]*><\/script>/g;

  for (const match of indexHtml.matchAll(cssRegex)) {
    cssTags.push(`<link rel="stylesheet" href="${match[1]}">`);
  }

  for (const match of indexHtml.matchAll(scriptRegex)) {
    scriptTags.push(`<script type="module" crossorigin src="${match[1]}"></script>`);
  }

  return { cssTags, scriptTags };
}

function layoutHtml({
  title,
  description,
  path,
  body,
  keywords,
  robots = "index, follow",
  structuredData,
  cssTags,
  scriptTags,
}) {
  const canonical = toAbsolute(path);

  return `<!doctype html>
<html lang="en">
  <head>
    <meta charset="UTF-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <title>${escapeHtml(title)}</title>
    <meta name="description" content="${escapeHtml(description)}" />
    <meta name="keywords" content="${escapeHtml(keywords.join(", "))}" />
    <meta name="robots" content="${escapeHtml(robots)}" />
    <link rel="canonical" href="${canonical}" />
    <meta property="og:type" content="website" />
    <meta property="og:title" content="${escapeHtml(title)}" />
    <meta property="og:description" content="${escapeHtml(description)}" />
    <meta property="og:url" content="${canonical}" />
    <meta property="og:image" content="${SITE_ORIGIN}/og-card.svg" />
    <meta name="twitter:card" content="summary_large_image" />
    <meta name="twitter:title" content="${escapeHtml(title)}" />
    <meta name="twitter:description" content="${escapeHtml(description)}" />
    <meta name="twitter:image" content="${SITE_ORIGIN}/og-card.svg" />
    <meta name="prerender-source" content="seo-prerender" />
    <script type="application/ld+json">${JSON.stringify(structuredData)}</script>
    ${cssTags.join("\n    ")}
    <style>
      :root { color-scheme: light dark; }
      body { margin: 0; font-family: Inter, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif; background: #fff; color: #0f172a; }
      .page { max-width: 1120px; margin: 0 auto; padding: 40px 24px; }
      .muted { color: #475569; }
      .chip { display: inline-block; margin-right: 8px; margin-bottom: 8px; border: 1px solid #cbd5e1; border-radius: 9999px; padding: 4px 12px; font-size: 12px; color: #334155; }
      .hero { border: 1px solid #c7d2fe; border-radius: 16px; padding: 24px; background: #eef2ff; }
      .grid { display: grid; grid-template-columns: repeat(auto-fill,minmax(240px,1fr)); gap: 12px; margin-top: 20px; }
      .card { border: 1px solid #e2e8f0; border-radius: 14px; padding: 16px; background: #fff; }
      .card a { color: #1d4ed8; text-decoration: none; }
      .card a:hover { text-decoration: underline; }
      .cta { display: inline-block; margin-right: 10px; margin-top: 12px; background: #4f46e5; color: #fff; text-decoration: none; padding: 10px 16px; border-radius: 10px; font-weight: 600; }
      .cta.secondary { background: transparent; color: #1e293b; border: 1px solid #cbd5e1; }
      @media (prefers-color-scheme: dark) {
        body { background: #020617; color: #e2e8f0; }
        .hero { background: #1e1b4b; border-color: #4338ca; }
        .card { background: #0f172a; border-color: #1e293b; }
        .muted, .chip { color: #cbd5e1; border-color: #334155; }
        .card a, .cta.secondary { color: #c7d2fe; }
      }
    </style>
  </head>
  <body>
    <div id="root">${body}</div>
    ${scriptTags.join("\n    ")}
  </body>
</html>
`;
}

function buildCoreBody(corePage) {
  const ctaSection =
    corePage.path === "/privacy" || corePage.path === "/terms"
      ? `<a class="cta secondary" href="/">Back to home</a>`
      : `<a class="cta" href="/register">Start free</a>
    <a class="cta secondary" href="/pricing">View pricing</a>`;

  return `
<main class="page">
  <section class="hero">
    <p class="muted">PulseScore</p>
    <h1>${escapeHtml(corePage.h1)}</h1>
    <p class="muted">${escapeHtml(corePage.description)}</p>
    <span class="chip">B2B SaaS customer health scoring</span>
    <span class="chip">Stripe · HubSpot · Intercom</span>
  </section>

  <section class="card" style="margin-top:20px;">
    <h2>Next step</h2>
    <p class="muted">Choose the route that best matches your workflow: evaluate plans, start a trial, or review product policies.</p>
    ${ctaSection}
  </section>
</main>
`;
}

function buildHubBody(family, familyPages) {
  const config = FAMILY_CONFIG[family];
  const relatedHubs = FAMILY_ORDER.filter((item) => item !== family)
    .slice(0, 4)
    .map((key) => FAMILY_CONFIG[key]);

  return `
<main class="page">
  <section class="hero">
    <p class="muted">${escapeHtml(config.label)}</p>
    <h1>${escapeHtml(config.hubTitle)}</h1>
    <p class="muted">${escapeHtml(config.hubDescription)}</p>
    <span class="chip">${familyPages.length} pages in this hub</span>
    <span class="chip">Intent-aligned content</span>
    <span class="chip">Built for B2B SaaS teams</span>
  </section>

  <section class="grid">
    ${familyPages
      .map((page) => {
        const path = buildPath(page);
        const heading = buildHeading(page);
        const description = buildDescription(page);
        return `<article class="card">
          <p class="muted">${escapeHtml(page.intent)}</p>
          <h2><a href="${path}">${escapeHtml(heading)}</a></h2>
          <p class="muted">${escapeHtml(description)}</p>
          <p class="muted">Keyword: ${escapeHtml(page.keyword)}</p>
        </article>`;
      })
      .join("\n    ")}
  </section>

  <section style="margin-top: 32px" class="card">
    <h2>Explore related hubs</h2>
    <div class="grid">
      ${relatedHubs
        .map(
          (hub) => `<a class="card" href="${hub.path}">
            <strong>${escapeHtml(hub.label)}</strong>
            <p class="muted">${escapeHtml(hub.hubDescription)}</p>
          </a>`,
        )
        .join("\n      ")}
    </div>
    <a class="cta" href="/register">Start free</a>
    <a class="cta secondary" href="/pricing">View pricing</a>
  </section>
</main>
`;
}

function hashString(value) {
  let hash = 0;
  for (let index = 0; index < value.length; index += 1) {
    hash = (hash * 31 + value.charCodeAt(index)) >>> 0;
  }
  return hash;
}

function pickBySeed(values, seed, offset = 0) {
  return values[(seed + offset) % values.length];
}

function buildDetailBody(page, byFamily) {
  const config = FAMILY_CONFIG[page.family];
  const heading = buildHeading(page);
  const seed = hashString(`${page.family}:${page.slug}:${page.keyword}`);

  const cadence = pickBySeed(["weekly", "bi-weekly", "monthly"], seed, 1);
  const owner = pickBySeed(
    [
      "Customer Success leadership",
      "RevOps owners",
      "account managers",
      "cross-functional retention squad",
    ],
    seed,
    2,
  );
  const dataSource = pickBySeed(
    [
      "billing and product events",
      "CRM lifecycle fields",
      "support and onboarding interactions",
      "renewal and expansion account notes",
    ],
    seed,
    3,
  );
  const successMetric = pickBySeed(
    [
      "risk-to-resolution time",
      "renewal retention lift",
      "intervention completion rate",
      "expansion conversion quality",
    ],
    seed,
    4,
  );

  const sameFamilyRelated = byFamily[page.family]
    .filter((candidate) => candidate.slug !== page.slug)
    .slice(0, 4);

  const crossFamilyRelated = FAMILY_ORDER.map((key) => byFamily[key]?.[0])
    .filter(Boolean)
    .filter((candidate) => candidate.family !== page.family)
    .slice(0, 2);

  const related = [...sameFamilyRelated, ...crossFamilyRelated];

  return `
<main class="page">
  <nav class="muted" style="font-size:12px; margin-bottom:12px;">
    <a href="/">Home</a> / <a href="${config.path}">${escapeHtml(config.label)}</a> / ${escapeHtml(page.entity)}
  </nav>

  <section class="hero">
    <p class="muted">${escapeHtml(config.label)} · ${escapeHtml(page.intent)}</p>
    <h1>${escapeHtml(heading)}</h1>
    <p class="muted">Teams searching &quot;${escapeHtml(page.keyword)}&quot; typically need a repeatable way to convert signals into action. This page turns ${escapeHtml(page.entity)} into a ${escapeHtml(cadence)} workflow owned by ${escapeHtml(owner)}.</p>
    <span class="chip">Primary keyword: ${escapeHtml(page.keyword)}</span>
    <span class="chip">Last updated: ${new Date().toISOString().slice(0, 10)}</span>
  </section>

  <section class="grid" style="margin-top:20px;">
    <article class="card">
      <h2>How to use this page</h2>
      <p class="muted">Use this as an implementation blueprint, not just a definition. Map ${escapeHtml(dataSource)} into one action matrix tied to ${escapeHtml(successMetric)}.</p>
    </article>
    <article class="card">
      <h2>Operational playbook</h2>
      <p class="muted">Run ${escapeHtml(cadence)} reviews, calibrate thresholds on a fixed rhythm, and keep a named owner (${escapeHtml(owner)}) accountable for follow-through.</p>
    </article>
    <article class="card">
      <h2>Quality safeguards</h2>
      <p class="muted">Avoid thin workflows: require fresh ${escapeHtml(dataSource)}, explicit intervention thresholds, and outcome checks against ${escapeHtml(successMetric)}.</p>
    </article>
  </section>

  <section class="card" style="margin-top:20px;">
    <h2>Next step in PulseScore</h2>
    <p class="muted">Start with this framework, then connect Stripe, HubSpot, and Intercom signals in PulseScore to automate prioritization around ${escapeHtml(successMetric)}.</p>
    <a class="cta" href="/register">Start free</a>
    <a class="cta secondary" href="/pricing">Compare plans</a>
  </section>

  <section style="margin-top:20px;" class="card">
    <h2>Related resources</h2>
    <div class="grid">
      ${related
        .map((item) => {
          const path = buildPath(item);
          const title = buildHeading(item);
          return `<a class="card" href="${path}">
            <strong>${escapeHtml(title)}</strong>
            <p class="muted">${escapeHtml(item.keyword)}</p>
          </a>`;
        })
        .join("\n      ")}
    </div>
  </section>

  <section style="margin-top:20px;" class="card">
    <h2>Explore all ${escapeHtml(config.label.toLowerCase())}</h2>
    <p class="muted">Browse the full hub for this playbook family.</p>
    <a class="cta secondary" href="${config.path}">Open ${escapeHtml(config.label)} hub</a>
  </section>
</main>
`;
}

async function writeRouteHtml(distDir, routePath, html) {
  const normalized = routePath.replace(/^\/+/, "");
  const routeDir = normalized ? new URL(`./${normalized}/`, distDir) : distDir;
  await mkdir(routeDir, { recursive: true });
  await writeFile(new URL("./index.html", routeDir), html, "utf8");
}

async function main() {
  const pagesRaw = await readFile(
    new URL("../src/content/seo-pages.json", import.meta.url),
    "utf8",
  );
  const pages = JSON.parse(pagesRaw);

  const distDir = new URL("../dist/", import.meta.url);
  const distIndexHtml = await readFile(new URL("./index.html", distDir), "utf8");
  const { cssTags, scriptTags } = getAssetTags(distIndexHtml);

  const byFamily = {
    templates: [],
    integrations: [],
    personas: [],
    comparisons: [],
    glossary: [],
    examples: [],
    curation: [],
  };

  for (const page of pages) {
    if (!FAMILY_CONFIG[page.family]) {
      continue;
    }
    byFamily[page.family].push(page);
  }

  const renderedPaths = [];

  for (const corePage of CORE_PAGES) {
    const coreStructuredData = {
      "@context": "https://schema.org",
      "@type": "WebPage",
      name: corePage.h1,
      description: corePage.description,
      url: toAbsolute(corePage.path),
    };

    const coreHtml = layoutHtml({
      title: corePage.title,
      description: corePage.description,
      path: corePage.path,
      keywords: corePage.keywords,
      robots: corePage.robots,
      body: buildCoreBody(corePage),
      structuredData: coreStructuredData,
      cssTags,
      scriptTags,
    });

    await writeRouteHtml(distDir, corePage.path, coreHtml);
    renderedPaths.push(corePage.path);
  }

  for (const family of FAMILY_ORDER) {
    const familyPages = byFamily[family] ?? [];
    const hubConfig = FAMILY_CONFIG[family];
    const hubPath = hubConfig.path;

    const hubStructuredData = {
      "@context": "https://schema.org",
      "@type": "CollectionPage",
      name: hubConfig.hubTitle,
      description: hubConfig.hubDescription,
      url: toAbsolute(hubPath),
      hasPart: familyPages.map((page) => ({
        "@type": "WebPage",
        name: buildHeading(page),
        url: toAbsolute(buildPath(page)),
      })),
    };

    const hubHtml = layoutHtml({
      title: withBrandTitle(hubConfig.hubTitle),
      description: hubConfig.hubDescription,
      path: hubPath,
      keywords: [
        `${hubConfig.label.toLowerCase()} pulsescore`,
        "customer health scoring",
        "b2b saas retention",
      ],
      body: buildHubBody(family, familyPages),
      structuredData: hubStructuredData,
      cssTags,
      scriptTags,
    });

    await writeRouteHtml(distDir, hubPath, hubHtml);
    renderedPaths.push(hubPath);

    for (const page of familyPages) {
      const path = buildPath(page);
      const title = buildTitle(page);
      const description = buildDescription(page);

      const detailStructuredData = [
        {
          "@context": "https://schema.org",
          "@type": "BreadcrumbList",
          itemListElement: [
            {
              "@type": "ListItem",
              position: 1,
              name: "Home",
              item: toAbsolute("/"),
            },
            {
              "@type": "ListItem",
              position: 2,
              name: hubConfig.label,
              item: toAbsolute(hubPath),
            },
            {
              "@type": "ListItem",
              position: 3,
              name: buildHeading(page),
              item: toAbsolute(path),
            },
          ],
        },
        {
          "@context": "https://schema.org",
          "@type": page.family === "glossary" ? "DefinedTerm" : "WebPage",
          name: buildHeading(page),
          description,
          url: toAbsolute(path),
          keywords: [page.keyword, "customer health score", "saas retention"],
        },
      ];

      const detailHtml = layoutHtml({
        title,
        description,
        path,
        keywords: [page.keyword, page.entity, "customer health scoring"],
        body: buildDetailBody(page, byFamily),
        structuredData: detailStructuredData,
        cssTags,
        scriptTags,
      });

      await writeRouteHtml(distDir, path, detailHtml);
      renderedPaths.push(path);
    }
  }

  await writeFile(
    new URL("./seo-prerender-manifest.json", distDir),
    JSON.stringify(
      {
        generatedAt: new Date().toISOString(),
        routeCount: renderedPaths.length,
        routes: renderedPaths,
      },
      null,
      2,
    ),
    "utf8",
  );

  console.log(`Prerendered ${renderedPaths.length} SEO routes into dist.`);
}

main().catch((error) => {
  console.error(`Failed SEO prerender: ${error.message}`);
  process.exitCode = 1;
});
