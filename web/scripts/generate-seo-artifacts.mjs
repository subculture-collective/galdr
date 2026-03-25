import { mkdir, readFile, writeFile } from "node:fs/promises";

const SITE_ORIGIN = "https://pulsescore.app";

const FAMILY_PATH_PREFIX = {
  templates: "/templates",
  integrations: "/integrations",
  personas: "/for",
  comparisons: "/compare",
  glossary: "/glossary",
  examples: "/examples",
  curation: "/best",
};

const CHANGEFREQ = {
  core: "weekly",
  templates: "weekly",
  integrations: "weekly",
  personas: "weekly",
  comparisons: "weekly",
  glossary: "monthly",
  examples: "monthly",
  curation: "monthly",
};

const PRIORITY = {
  core: "0.9",
  templates: "0.8",
  integrations: "0.8",
  personas: "0.75",
  comparisons: "0.75",
  glossary: "0.65",
  examples: "0.65",
  curation: "0.7",
};

function xmlEscape(value) {
  return value
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
    .replaceAll("'", "&apos;");
}

function buildUrlset(urlEntries) {
  const rows = urlEntries
    .map(
      (entry) => `  <url>\n    <loc>${xmlEscape(entry.loc)}</loc>\n    <lastmod>${entry.lastmod}</lastmod>\n    <changefreq>${entry.changefreq}</changefreq>\n    <priority>${entry.priority}</priority>\n  </url>`,
    )
    .join("\n");

  return `<?xml version="1.0" encoding="UTF-8"?>\n<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">\n${rows}\n</urlset>\n`;
}

function buildSitemapIndex(sitemapPaths, lastmod) {
  const rows = sitemapPaths
    .map(
      (path) =>
        `  <sitemap>\n    <loc>${SITE_ORIGIN}${path}</loc>\n    <lastmod>${lastmod}</lastmod>\n  </sitemap>`,
    )
    .join("\n");

  return `<?xml version="1.0" encoding="UTF-8"?>\n<sitemapindex xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">\n${rows}\n</sitemapindex>\n`;
}

function toAbsolute(path) {
  return `${SITE_ORIGIN}${path}`;
}

async function main() {
  const raw = await readFile(new URL("../src/content/seo-pages.json", import.meta.url), "utf8");
  const pages = JSON.parse(raw);

  const today = new Date().toISOString().slice(0, 10);

  const corePaths = [
    "/",
    "/pricing",
    "/templates",
    "/integrations",
    "/for",
    "/compare",
    "/glossary",
    "/examples",
    "/best",
  ];

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
    const prefix = FAMILY_PATH_PREFIX[page.family];
    if (!prefix) {
      continue;
    }

    byFamily[page.family].push(`${prefix}/${page.slug}`);
  }

  const distDir = new URL("../dist/", import.meta.url);
  const sitemapDir = new URL("../dist/sitemaps/", import.meta.url);
  await mkdir(sitemapDir, { recursive: true });

  const sitemapFiles = [
    { key: "core", filename: "marketing.xml", paths: corePaths },
    { key: "templates", filename: "templates.xml", paths: byFamily.templates },
    { key: "integrations", filename: "integrations.xml", paths: byFamily.integrations },
    { key: "personas", filename: "personas.xml", paths: byFamily.personas },
    { key: "comparisons", filename: "comparisons.xml", paths: byFamily.comparisons },
    { key: "glossary", filename: "glossary.xml", paths: byFamily.glossary },
    { key: "examples", filename: "examples.xml", paths: byFamily.examples },
    { key: "curation", filename: "curation.xml", paths: byFamily.curation },
  ];

  const sitemapPaths = [];

  for (const sitemap of sitemapFiles) {
    const urlset = buildUrlset(
      sitemap.paths.map((path) => ({
        loc: toAbsolute(path),
        lastmod: today,
        changefreq: CHANGEFREQ[sitemap.key] ?? "monthly",
        priority: PRIORITY[sitemap.key] ?? "0.6",
      })),
    );

    await writeFile(new URL(`./${sitemap.filename}`, sitemapDir), urlset, "utf8");
    sitemapPaths.push(`/sitemaps/${sitemap.filename}`);
  }

  const sitemapIndexXml = buildSitemapIndex(sitemapPaths, today);

  await writeFile(new URL("./sitemap-index.xml", distDir), sitemapIndexXml, "utf8");
  await writeFile(new URL("./sitemap.xml", distDir), sitemapIndexXml, "utf8");

  console.log("Generated sitemap artifacts:");
  for (const path of sitemapPaths) {
    console.log(`- ${path}`);
  }
}

main().catch((error) => {
  console.error(`Failed to generate SEO artifacts: ${error.message}`);
  process.exitCode = 1;
});
