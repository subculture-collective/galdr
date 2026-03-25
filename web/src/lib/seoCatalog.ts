import rawSeoPages from "@/content/seo-pages.json";

export type SeoFamily =
  | "templates"
  | "integrations"
  | "personas"
  | "comparisons"
  | "glossary"
  | "examples"
  | "curation";

export type SeoIntent = "informational" | "commercial" | "transactional";

export interface SeoPageSeed {
  family: SeoFamily;
  slug: string;
  entity: string;
  keyword: string;
  intent: SeoIntent;
}

export interface SeoPage extends SeoPageSeed {
  path: string;
  title: string;
  description: string;
  h1: string;
}

export interface SeoHub {
  family: SeoFamily;
  path: string;
  label: string;
  title: string;
  description: string;
  hero: string;
}

interface FamilyConfig {
  path: string;
  label: string;
  titleSuffix: string;
  hubTitle: string;
  hubDescription: string;
  hero: string;
}

const familyConfig: Record<SeoFamily, FamilyConfig> = {
  templates: {
    path: "/templates",
    label: "Templates",
    titleSuffix: "Template",
    hubTitle: "Customer health templates for lean SaaS teams",
    hubDescription:
      "Action-ready customer health templates you can use immediately, then operationalize in PulseScore.",
    hero: "Turn retention strategy into execution with practical templates.",
  },
  integrations: {
    path: "/integrations",
    label: "Integrations",
    titleSuffix: "Integration",
    hubTitle: "Integration playbooks for customer health scoring",
    hubDescription:
      "Connect billing, CRM, and support signals to unify churn-risk visibility in one workflow.",
    hero: "Build a single source of truth for customer health signals.",
  },
  personas: {
    path: "/for",
    label: "Personas",
    titleSuffix: "for Teams",
    hubTitle: "PulseScore by persona and growth stage",
    hubDescription:
      "See how founders, CS, and RevOps teams tailor health scoring to their operating model.",
    hero: "Find the exact customer health workflow for your team shape.",
  },
  comparisons: {
    path: "/compare",
    label: "Comparisons",
    titleSuffix: "Comparison",
    hubTitle: "PulseScore comparison guides",
    hubDescription:
      "Balanced comparisons focused on setup speed, pricing, and fit for lean teams.",
    hero: "Choose the right customer success stack with clarity, not guesswork.",
  },
  glossary: {
    path: "/glossary",
    label: "Glossary",
    titleSuffix: "Definition",
    hubTitle: "Customer success and churn glossary",
    hubDescription:
      "Clear, practical definitions with examples and implementation context for SaaS operators.",
    hero: "Learn the language of retention, then apply it in your workflow.",
  },
  examples: {
    path: "/examples",
    label: "Examples",
    titleSuffix: "Examples",
    hubTitle: "Real customer health examples",
    hubDescription:
      "Concrete examples of health scores, thresholds, alerts, and intervention patterns.",
    hero: "Skip theory—see what good customer health execution looks like.",
  },
  curation: {
    path: "/best",
    label: "Best-of Guides",
    titleSuffix: "Best Guide",
    hubTitle: "Curated best-of customer success guides",
    hubDescription:
      "Research-backed shortlists of tools and approaches by team size, stack, and outcomes.",
    hero: "Use transparent evaluation criteria to pick the right tools faster.",
  },
};

const seoPageSeeds = rawSeoPages as SeoPageSeed[];

function buildPath(seed: SeoPageSeed): string {
  return `${familyConfig[seed.family].path}/${seed.slug}`;
}

function withBrandTitle(base: string): string {
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

export function toSeoTitle(base: string): string {
  return withBrandTitle(base);
}

function buildTitle(seed: SeoPageSeed): string {
  switch (seed.family) {
    case "templates":
      return withBrandTitle(`${seed.entity} Template`);
    case "integrations":
      return withBrandTitle(`PulseScore + ${seed.entity} Integration`);
    case "personas":
      return withBrandTitle(`Health Scoring for ${seed.entity}`);
    case "comparisons":
      return withBrandTitle(`${seed.entity} Comparison`);
    case "glossary":
      return withBrandTitle(`${seed.entity} Definition`);
    case "examples":
      return withBrandTitle(`${seed.entity} Examples`);
    case "curation":
      return withBrandTitle(seed.entity);
    default:
      return withBrandTitle(seed.entity);
  }
}

function buildDescription(seed: SeoPageSeed): string {
  switch (seed.family) {
    case "templates":
      return `Use this ${seed.entity.toLowerCase()} template to prioritize risk, define thresholds, and run a repeatable retention workflow for B2B SaaS.`;
    case "integrations":
      return `Learn how ${seed.entity} can feed customer health signals into PulseScore so your team can identify churn risk earlier and act faster.`;
    case "personas":
      return `See how ${seed.entity.toLowerCase()} use PulseScore to monitor account health, reduce churn, and focus effort where it drives revenue retention.`;
    case "comparisons":
      return `Compare ${seed.entity} across pricing, setup speed, integrations, and fit so lean customer success teams can choose with confidence.`;
    case "glossary":
      return `Understand ${seed.entity.toLowerCase()} with clear definitions, formulas, examples, and practical guidance for SaaS customer success workflows.`;
    case "examples":
      return `Explore ${seed.entity.toLowerCase()} with practical patterns, implementation notes, and proven workflows for retention-focused teams.`;
    case "curation":
      return `Review ${seed.entity.toLowerCase()} using transparent criteria: setup complexity, integration depth, pricing, and operational fit for SMB SaaS.`;
    default:
      return `${seed.entity} resources and implementation guidance from PulseScore.`;
  }
}

function buildH1(seed: SeoPageSeed): string {
  switch (seed.family) {
    case "templates":
      return `Free ${seed.entity} Template for B2B SaaS`;
    case "integrations":
      return `PulseScore + ${seed.entity}`;
    case "personas":
      return `PulseScore for ${seed.entity}`;
    case "comparisons":
      return seed.entity;
    case "glossary":
      return `What is ${seed.entity}?`;
    case "examples":
      return seed.entity;
    case "curation":
      return seed.entity;
    default:
      return seed.entity;
  }
}

export const seoHubs: SeoHub[] = (
  Object.keys(familyConfig) as SeoFamily[]
).map((family) => ({
  family,
  path: familyConfig[family].path,
  label: familyConfig[family].label,
  title: familyConfig[family].hubTitle,
  description: familyConfig[family].hubDescription,
  hero: familyConfig[family].hero,
}));

export const seoPages: SeoPage[] = seoPageSeeds.map((seed) => ({
  ...seed,
  path: buildPath(seed),
  title: buildTitle(seed),
  description: buildDescription(seed),
  h1: buildH1(seed),
}));

export function getHubByFamily(family: SeoFamily): SeoHub {
  const hub = seoHubs.find((item) => item.family === family);
  if (!hub) {
    throw new Error(`Unknown SEO family: ${family}`);
  }
  return hub;
}

export function getSeoPagesByFamily(family: SeoFamily): SeoPage[] {
  return seoPages.filter((page) => page.family === family);
}

export function getSeoPageByFamilyAndSlug(
  family: SeoFamily,
  slug: string,
): SeoPage | null {
  return (
    seoPages.find((page) => page.family === family && page.slug === slug) ??
    null
  );
}

export function getRelatedPages(page: SeoPage, limit = 6): SeoPage[] {
  const sameFamily = seoPages.filter(
    (item) => item.family === page.family && item.slug !== page.slug,
  );

  const crossFamily = seoPages.filter(
    (item) => item.family !== page.family && item.intent === page.intent,
  );

  return [...sameFamily.slice(0, limit - 2), ...crossFamily.slice(0, 2)].slice(
    0,
    limit,
  );
}

export function getCrossFamilyHubs(activeFamily: SeoFamily): SeoHub[] {
  return seoHubs.filter((hub) => hub.family !== activeFamily).slice(0, 4);
}

export const seoFamilyOrder: SeoFamily[] = [
  "templates",
  "integrations",
  "personas",
  "comparisons",
  "examples",
  "curation",
  "glossary",
];
