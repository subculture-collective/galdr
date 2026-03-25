# PulseScore Programmatic SEO Master Plan (2026)

## 0) Scope and objective

This document is a full programmatic SEO execution plan for PulseScore, aligned to the product and codebase in this repository.

**Primary business goal:** generate qualified trials from search by ranking for high-intent customer-health and customer-success queries.

**Primary conversion event:** `Start free` (registration).

**Secondary conversion events:**
- `Pricing` page visits
- `Book demo` (to be added)
- Marketplace intent clicks (Stripe/HubSpot app ecosystem pages)

---

## 1) Initial assessment

## 1.1 Business context

PulseScore is a B2B SaaS customer health scoring platform for lean CS/revops teams, with Stripe, HubSpot, and Intercom integrations.

Current public surface includes:
- `/`
- `/pricing`
- `/login`
- `/register`
- `/privacy`
- `/terms`

Protected app routes (`/dashboard`, `/customers`, `/settings/*`) are product UX routes and should not be part of pSEO indexing.

## 1.2 Target audience

Primary ICP:
- B2B SaaS teams (roughly 10–100 employees)
- Customer Success, RevOps, and founders/operators
- Currently managing retention/churn in spreadsheets

Common jobs-to-be-done:
- Build customer health score model quickly
- Detect churn risk earlier
- Centralize customer health signals from billing + CRM + support

## 1.3 Conversion goals for programmatic pages

Each pSEO page should drive one of:
1. Free signup
2. Template/tool usage (lead magnet)
3. Integration setup intent
4. Comparison-page assisted conversion (commercial research phase)

---

## 2) Opportunity assessment

> Volumes below are **directional ranges** to prioritize execution. Validate exact monthly search volume in Ahrefs/SEMrush/GSC before production roll-out.

## 2.1 Pattern inventory (priority)

| Priority | Playbook | Pattern | Intent | Estimated demand shape | Feasibility |
|---|---|---|---|---|---|
| P1 | Templates | `customer health score template`, `saas health score template`, `churn score template` | High, immediate utility | Mid-head + large long tail | High |
| P1 | Integrations | `[tool A] [tool B] integration` where PulseScore is bridge | High, solution-seeking | Long-tail scalable | High |
| P1 | Personas | `customer success software for [audience]` | Commercial investigation | Medium long tail | Medium-High |
| P2 | Comparisons | `PulseScore vs [competitor]`, `[competitor] alternative` | Bottom-funnel | Medium long tail | Medium |
| P2 | Glossary | `what is [term]` for CS/churn/health terms | TOFU authority | High long tail | High |
| P3 | Curation | `best customer success software for [segment]` | Commercial research | High competition | Medium |

## 2.2 Opportunity by page family (phase 1-3)

### Family A — Templates (P1)
- Pattern examples:
  - `customer health score template`
  - `customer churn risk template`
  - `saas renewal risk score template`
  - `hubspot customer health dashboard template`
- Initial target count: **20–40 pages**
- Why this family first:
  - Strong intent match with PulseScore’s value proposition
  - Natural CTA to migrate from spreadsheet template to automated health scoring

### Family B — Integrations (P1)
- Pattern examples:
  - `stripe hubspot customer health`
  - `intercom churn risk tracking`
  - `[platform] + PulseScore` setup pages
- Initial target count: **25–60 pages** (start with real supported integrations + connector workflows)
- Why this family first:
  - Product truth is strong (real integrations already exist)
  - Commercial and implementation intent is high

### Family C — Personas (P1)
- Pattern examples:
  - `customer health scoring for b2b saas`
  - `churn prevention software for startups`
  - `customer success dashboard for revops`
- Initial target count: **15–30 pages**
- Why:
  - Captures segmented use cases with higher conversion potential than generic pages

### Family D — Comparisons (P2)
- Pattern examples:
  - `PulseScore vs ChurnZero`
  - `Vitally alternative for startups`
  - `Totango alternative for small teams`
- Initial target count: **10–25 pages**
- Why:
  - BOFU intent, but requires high editorial quality and fair comparisons

### Family E — Glossary (P2)
- Pattern examples:
  - `what is customer health score`
  - `what is churn risk`
  - `gross revenue retention definition`
- Initial target count: **50–120 pages**
- Why:
  - Topical authority + strong internal linking source to money pages

## 2.3 Search volume distribution hypothesis

- **Head terms (10–20% pages):** broad terms like `customer health score template` and `customer success software`.
- **Body terms (30–40% pages):** role/segment/integration variants.
- **Long-tail (40–60% pages):** precise combinations (`for [persona] + [integration] + [goal]`).

This is ideal for pSEO because long-tail pages can convert better despite lower individual volume.

---

## 3) Competitive landscape

## 3.1 Current SERP archetypes observed

From spot checks, SERPs are dominated by:
- Vendor educational content (Vitally, Sweep, CS blogs)
- Generic listicles (`best customer success software`) with weak practical depth
- Community content (Reddit/YouTube) around templates

## 3.2 What PulseScore must do to beat incumbents

1. **Utility > opinion**
   - Provide downloadable + interactive templates, not just theory.
2. **Explainability**
   - Show score drivers, thresholds, intervention playbooks.
3. **Evidence and specificity**
   - Use concrete examples per segment/integration.
4. **Freshness mechanics**
   - Show `last updated` dates and update cadence.
5. **Commercial honesty**
   - Balanced comparison pages with clear fit-by-scenario recommendations.

## 3.3 Feasibility rating

- Overall pSEO feasibility for PulseScore: **8/10**
- Why high:
  - Strong intent-match product
  - Natural template/integration use cases
  - Distinct wedge (lean teams, low-cost, fast setup)
- Main risk:
  - Thin/duplicative page generation if content quality gates are not strict

---

## 4) Playbook selection across all 12 frameworks

| Playbook | Fit for PulseScore | Decision |
|---|---|---|
| Templates | Excellent | Execute P1 |
| Curation | Good, competitive | Execute P2 |
| Conversions | Weak direct fit | Skip for now |
| Comparisons | Strong BOFU | Execute P2 |
| Examples | Good (health score examples) | Execute P2 |
| Locations | Weak (not local-first product) | Skip |
| Personas | Excellent | Execute P1 |
| Integrations | Excellent | Execute P1 |
| Glossary | Strong authority engine | Execute P2 |
| Translations | Strong future expansion | Execute P3 |
| Directory | Medium (tools/category pages) | Pilot P3 |
| Profiles | Medium (company/software profiles) | Pilot P3 |

---

## 5) Data strategy (defensibility)

## 5.1 Data hierarchy and what PulseScore should use

1. **Product-derived data (highest practical near-term value)**
   - Aggregated, anonymized benchmark metrics from health scoring outcomes.
2. **Proprietary methodologies**
   - PulseScore scoring frameworks and intervention maps.
3. **User-generated/field examples**
   - Case studies, anonymized score-change stories.
4. **Public data support**
   - Industry definitions/benchmarks as context, not core differentiation.

## 5.2 Data safety rule

Only publish aggregate statistics where k-anonymity threshold is met (e.g., `k >= 20 orgs`), with no customer-identifiable details.

## 5.3 Content data model (minimum)

```yaml
page:
  slug:
  playbook_type: templates|integrations|personas|comparisons|glossary
  keyword_primary:
  keyword_secondary: []
  intent: informational|commercial|transactional
  cta_variant:
  last_updated_at:

entity:
  integration_name:
  persona_name:
  competitor_name:
  template_type:

unique_data:
  benchmark_metrics: []
  score_driver_examples: []
  implementation_steps: []
  common_pitfalls: []

seo:
  title:
  meta_description:
  canonical_url:
  schema_type:
```

---

## 6) URL architecture and information architecture

## 6.1 Core URL rules

- Use **subfolders only** (no subdomains).
- Keep URLs readable and consistent.
- No query parameters for canonical page versions.

## 6.2 Recommended pSEO route structure

- `/templates/[template-type]/`
- `/integrations/[product]/`
- `/for/[persona]/`
- `/compare/[x]-vs-[y]/`
- `/glossary/[term]/`
- `/examples/[type]/` (phase 2)
- `/best/[category]/` (phase 2)

## 6.3 Hub-and-spoke linking model

Hubs:
- `/templates/`
- `/integrations/`
- `/for/`
- `/compare/`
- `/glossary/`

Spokes:
- Each programmatic leaf page

Mandatory links per leaf page:
1. Parent hub page
2. 3–8 related sibling pages
3. 1 downstream BOFU page (`/pricing` or relevant comparison)
4. 1 contextual signup CTA

---

## 7) Implementation framework

## 7.1 Phase plan

### Phase 0 (foundation, 1–2 weeks)
- Set up content source of truth (JSON/MDX/content DB).
- Add programmatic route generation.
- Split sitemap by page type.
- Add SEO QA lints (missing metadata, thin content flags).

### Phase 1 (high-intent launch, 3–6 weeks)
- Publish:
  - 25 template pages
  - 30 integration pages
  - 20 persona pages
- Build hub pages for each family.

### Phase 2 (authority + BOFU capture, 4–8 weeks)
- Publish:
  - 15 comparison pages
  - 60 glossary pages
  - 20 examples pages

### Phase 3 (expansion)
- Translation rollout (`/es/`, `/de/` etc.)
- Directory/profile pilots
- International keyword localization

## 7.2 Technical requirements for current stack

Current marketing site is a React SPA with runtime meta tags. For large-scale SEO pages, switch to one of:

1. **Preferred:** SSR/SSG framework for marketing pages (Next.js/Astro)
2. **Acceptable short-term:** Vite prerender pipeline generating static HTML per pSEO route

Minimum technical requirements regardless of framework:
- Static-rendered HTML for pSEO pages
- Unique canonical per page
- JSON-LD schema support
- Sitemap index + segmented sitemaps
- robots directives by page class
- Fast page load (Core Web Vitals targets)

## 7.3 Existing repo SEO fixes required before pSEO scale

1. Remove `/login` and `/register` from sitemap (or set `noindex` and exclude from sitemap).
2. Keep `/privacy` and `/terms` low priority (already fine).
3. Add sitemap index strategy (`/sitemap-index.xml` + per-family sitemap files).
4. Ensure new pSEO pages are server/prerender rendered.
5. Add Search Console and analytics segmentation by page family.

---

## 8) Production page templates (implementation-ready)

## 8.1 Template Playbook page

**URL pattern**: `/templates/[template-type]/`  
**Example**: `/templates/customer-health-score/`

**Title template**: `[Template Type] Template (Free) | PulseScore`  
**Meta description template**: `Download a free [template type] template with scoring logic, thresholds, and action playbooks. Built for B2B SaaS teams.`  
**H1 template**: `Free [Template Type] Template for B2B SaaS`

**Content outline**:
1. What this template solves (intent match)
2. Download + interactive preview
3. Included fields and scoring logic
4. How to customize by segment
5. Common mistakes and fixes
6. “When to move from spreadsheet to automation” CTA
7. Related templates

**Schema**:
- `SoftwareApplication` (if interactive)
- `HowTo`
- `FAQPage`
- `BreadcrumbList`

## 8.2 Integrations page

**URL pattern**: `/integrations/[product]/`  
**Example**: `/integrations/stripe/`

**Title template**: `PulseScore + [Product] Integration | Setup in Minutes`  
**Meta description template**: `Connect [Product] to PulseScore to unify customer health signals, detect churn risk early, and trigger proactive actions.`  
**H1 template**: `PulseScore + [Product] Integration`

**Content outline**:
1. Integration outcome summary
2. What data syncs and why it matters
3. Step-by-step setup
4. Example health insights unlocked
5. Troubleshooting and limits
6. Related integrations and persona pages

**Schema**:
- `Product`
- `HowTo`
- `FAQPage`
- `BreadcrumbList`

## 8.3 Persona page

**URL pattern**: `/for/[persona]/`  
**Example**: `/for/revops-teams/`

**Title template**: `Customer Health Scoring for [Persona] | PulseScore`  
**Meta description template**: `See how [persona] teams use PulseScore to identify at-risk accounts, reduce churn, and prioritize actions.`  
**H1 template**: `PulseScore for [Persona]`

**Content outline**:
1. Persona pain points
2. Score model for this persona
3. Dashboard/use-case walkthrough
4. Recommended integration stack
5. Proof points / testimonials
6. CTA (start free or demo)

**Schema**:
- `WebPage`
- `FAQPage`
- `BreadcrumbList`

## 8.4 Comparison page

**URL pattern**: `/compare/[x]-vs-[y]/`  
**Example**: `/compare/pulsescore-vs-churnzero/`

**Title template**: `PulseScore vs [Competitor]: Which Fits Lean CS Teams?`  
**Meta description template**: `Compare PulseScore vs [competitor] on pricing, integrations, setup time, and best-fit use cases.`  
**H1 template**: `PulseScore vs [Competitor]`

**Content outline**:
1. Quick verdict by team size/use case
2. Feature comparison table
3. Pricing and implementation effort
4. Integration depth comparison
5. Pros/cons and who should choose each
6. CTA to try PulseScore

**Schema**:
- `Product`
- `FAQPage`
- `BreadcrumbList`

## 8.5 Glossary page

**URL pattern**: `/glossary/[term]/`  
**Example**: `/glossary/customer-health-score/`

**Title template**: `[Term] Definition for SaaS Teams | PulseScore Glossary`  
**Meta description template**: `Learn what [term] means, why it matters, and how to apply it in customer success workflows.`  
**H1 template**: `What is [Term]?`

**Content outline**:
1. 1-sentence definition
2. Why it matters
3. Formula/framework (if applicable)
4. Example in B2B SaaS context
5. Related terms + next-step resource

**Schema**:
- `DefinedTerm`
- `FAQPage` (optional)
- `BreadcrumbList`

---

## 9) Quality control system (anti-thin-content safeguards)

## 9.1 Minimum uniqueness requirements per page

Each page must contain at least:
- 1 unique insight block specific to the target entity
- 1 example/use-case specific to that entity
- 1 contextual internal-link cluster
- 1 non-generic CTA variant tied to intent

## 9.2 Do-not-publish rules

Block publish if any are true:
- Word-level duplication > 60% vs sibling pages
- Missing entity-specific examples
- Placeholder or empty data sections
- No unique metadata/title
- No indexability/canonical validation

## 9.3 Editorial QA checklist

- Intent match passes human review
- Claims are factual and current
- Comparison pages remain balanced and fair
- Last-updated timestamp is visible

---

## 10) Indexation and sitemap strategy

## 10.1 Sitemap structure

- `/sitemap-index.xml`
- `/sitemaps/templates.xml`
- `/sitemaps/integrations.xml`
- `/sitemaps/personas.xml`
- `/sitemaps/comparisons.xml`
- `/sitemaps/glossary.xml`

## 10.2 Indexation policy

- Index pages with validated demand and quality.
- `noindex` pages below quality threshold or unresolved data quality issues.
- Keep faceted/filter pages out of index unless they have standalone demand.

## 10.3 Crawl budget safeguards

- Avoid infinite parameterized URLs.
- Consistent canonical URLs.
- Keep orphan-page count at zero.

---

## 11) Launch checklist (pre-launch)

## 11.1 Content quality
- [ ] Every page has unique value beyond variable swaps
- [ ] Intent match verified for each keyword family
- [ ] Entity-specific examples and implementation details included
- [ ] CTA alignment to page intent is clear

## 11.2 Technical SEO
- [ ] Unique title + meta per page
- [ ] Correct heading hierarchy
- [ ] Canonical tags validated
- [ ] Appropriate schema markup emitted
- [ ] Core Web Vitals pass thresholds
- [ ] Static-rendered HTML is crawlable

## 11.3 Information architecture
- [ ] Hub pages published for each family
- [ ] Breadcrumbs on all pages
- [ ] Related links block on all leaf pages
- [ ] No orphan pages

## 11.4 Indexation readiness
- [ ] Included in correct sitemap file
- [ ] Not blocked by robots.txt
- [ ] No accidental `noindex`
- [ ] Search Console property + sitemap submissions complete

## 11.5 Measurement
- [ ] Analytics events for CTA clicks by page family
- [ ] GSC queries and page groups dashboard ready
- [ ] Weekly index coverage report configured

---

## 12) Post-launch monitoring and targets

## 12.1 Weekly checks (first 8 weeks)
- Indexation rate by page family
- Click-through rate from SERP by template
- Ranking movement for primary keywords
- Conversion rate by landing page family
- Bounce/engagement diagnostics for low performers

## 12.2 Suggested 90-day targets
- Indexation of published pSEO pages: **>70%**
- First-page rankings: **10–20 priority terms**
- Trial signups from pSEO pages: **meaningful upward trend week-over-week**
- No thin-content/manual-action signals in GSC

---

## 13) Initial build backlog (first 50 pages)

## 13.1 Templates (20)
- customer health score template
- b2b saas health score template
- churn risk score template
- renewal risk template
- expansion opportunity score template
- onboarding health template
- hubspot health score template
- stripe health score template
- intercom health score template
- cs weekly risk review template
- + 10 segmented variants

## 13.2 Integrations (15)
- stripe
- hubspot
- intercom
- zapier
- salesforce (future/roadmap if truthful and clearly marked)
- zendesk (future/roadmap if truthful and clearly marked)
- + role/use-case integration pages

## 13.3 Personas (10)
- for founders
- for customer success teams
- for revops teams
- for startup saas
- for growth-stage saas
- for plg saas
- + 4 additional role/segment pages

## 13.4 Comparisons (5)
- pulsescore vs churnzero
- pulsescore vs vitally
- pulsescore vs totango
- pulsescore vs gainsight (small-team angle)
- pulsescore vs spreadsheets

---

## 14) Inputs still needed to finalize page-by-page production queue

1. Preferred keyword tool export (Ahrefs/SEMrush/GSC) for exact scoring
2. Confirmed integration roadmap (to avoid vaporware pages)
3. Public-proof assets (customer stories, anonymized benchmark snippets)
4. Decision on rendering strategy (SSG/SSR path)
5. Editorial owner and refresh SLA per page family

---

## 15) Recommended immediate next steps (this sprint)

1. Approve this plan and page-family priorities.
2. Implement technical SEO foundation (rendering + sitemap split + auth route index policy).
3. Produce first 10 high-intent template pages + 10 integration pages.
4. Ship QA gate checks before bulk generation.
5. Submit new sitemaps and begin weekly measurement loop.

---

## Appendix A — Quick win fixes in current repo

- `web/public/sitemap.xml` currently includes `/login` and `/register`; move these out of index target set.
- Keep pSEO pages under dedicated subfolders (`/templates`, `/integrations`, `/for`, `/compare`, `/glossary`).
- Existing `SeoMeta` component can remain for metadata orchestration, but pSEO pages should be prerendered/SSR for stronger crawl reliability at scale.

---

## Appendix B — Canonical CTA mapping by intent

- **Informational (glossary/examples):** CTA = template download or educational lead magnet
- **Commercial (comparisons/curation):** CTA = pricing and trial
- **Transactional (templates/integrations):** CTA = start free + setup walkthrough
