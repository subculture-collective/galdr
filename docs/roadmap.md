# PulseScore Master Roadmap

GitHub issue: #240

This roadmap is the repo-tracked companion to the GitHub master roadmap. It keeps the implementation sequence, epic boundaries, and issue inventory visible in source control so agents can work one scoped task at a time.

## Scope

- Total phases: 3
- Total epics: 23
- Total implementation issues: 206
- Tech stack: Go + React/TypeScript + PostgreSQL
- Deployment target: VPS with Docker Compose, reverse proxy, and SSL
- Primary integrations: Stripe, HubSpot, Intercom, SendGrid

## Phase 1: MVP

Goal: ship the core product surface for customer health scoring with Stripe data, dashboard UI, billing, onboarding, and docs.

### 1. Foundation

Epic: #5

Issues: #32 #33 #34 #35 #36 #37 #38 #39 #40 #41 #42 #43 #44 #45

Deliverables: Go API, React/Vite web app, PostgreSQL local stack, CI, Docker, production compose, health checks, middleware, and VPS deploy path.

### 2. Database Schema

Epic: #1

Issues: #46 #47 #48 #49 #50 #51 #52 #53 #54 #55

Deliverables: organization, user, integration, customer, event, health score, Stripe data, alert, seed, and DB helper foundations.

### 3. Authentication & Multi-tenancy

Epic: #12

Issues: #56 #57 #58 #59 #60 #61 #62 #63 #64 #65 #66 #67 #68 #69 #70

Deliverables: registration, login, JWT, refresh, tenant isolation, org creation, invitations, SendGrid invite email, RBAC, password reset, profile API, auth UI, and protected routes.

### 4. Stripe Data Integration

Epic: #2

Issues: #71 #72 #73 #74 #75 #76 #77 #78 #79 #80 #81 #82

Deliverables: Stripe OAuth, connection UI, customer/subscription/payment sync, webhooks, MRR, failed payments, payment recency, sync orchestration, scheduler, monitoring, and error handling.

### 5. Health Scoring Engine

Epic: #4

Issues: #83 #84 #85 #86 #87 #88 #89 #90 #91 #92 #93

Deliverables: scoring config, factor services, weighted aggregation, scheduler, change detection, history, risk categorization, and admin API.

### 6. REST API Layer

Epic: #21

Issues: #94 #95 #96 #97 #98 #99 #100 #101 #102 #103

Deliverables: customer APIs, dashboard APIs, score distribution, integration management, organization settings, user management, alert rules, and OpenAPI docs.

### 7. Dashboard Core UI

Epic: #6

Issues: #106 #107 #108 #109 #110 #111 #112 #113 #114 #115 #116 #117 #118 #119 #120 #121 #122

Deliverables: app shell, responsive navigation, overview, charts, customer table, filters, detail page, timeline, status badges, settings, toasts, skeletons, error boundary, 404, and theme toggle.

### 8. HubSpot Integration

Epic: #7

Issues: #123 #124 #125 #126 #127 #128 #129 #130

Deliverables: HubSpot OAuth, contact/deal sync, company enrichment, webhooks, incremental sync, connection UI, mapping, dedupe, and initial sync orchestration.

### 9. Intercom Integration

Epic: #14

Issues: #131 #132 #133 #134 #135 #136 #137

Deliverables: Intercom OAuth, conversation/contact sync, ticket metrics, webhooks, incremental sync, connection UI, dedupe, merge, and initial sync orchestration.

### 10. Email Alerts

Epic: #11

Issues: #138 #139 #140 #141 #142 #143 #144 #145

Deliverables: SendGrid service, alert templates, evaluation engine, score-drop triggers, alert UI, delivery tracking, notification preferences, and in-app notifications.

### 11. Onboarding Wizard

Epic: #16

Issues: #146 #147 #148 #149 #150 #151 #152

Deliverables: wizard shell, welcome/org step, Stripe step, HubSpot/Intercom steps, score preview, resume tracking, completion, and analytics.

### 12. Billing & Subscription

Epic: #20

Issues: #153 #154 #155 #156 #157 #158 #159 #160 #161

Deliverables: Stripe billing products/prices, Checkout, billing webhooks, subscription tracking, feature gates, pricing UI, checkout success flow, subscription management, and Customer Portal redirect.

### 13. Landing Page

Epic: #17

Issues: #162 #163 #164 #165 #166 #167

Deliverables: hero, feature cards, pricing comparison, testimonials, footer/legal links, SEO metadata, sitemap, and robots.txt.

### 14. Documentation

Epic: #22

Issues: #168 #169 #170 #171 #172

Deliverables: API reference, quickstart, Stripe setup guide, HubSpot/Intercom guides, and scoring methodology.

### 15. Stripe App Marketplace

Epic: #24

Issues: #173 #174 #175 #176

Deliverables: Marketplace requirements research, Stripe App manifest/package, install flow, and listing content.

## Phase 2: Expansion

Goal: add workflow automation, team collaboration, more integrations, and tier-aware feature packaging.

### 16. Automated Playbooks

Epic: #23

Issues: #177 #178 #179 #180 #181 #182 #183 #184 #185 #186

Deliverables: playbook model, CRUD API, visual builder, score threshold trigger, customer event trigger, email customer action, internal alert/tag actions, execution engine, execution history UI, and webhook action.

### 17. Team Collaboration

Epic: #25

Issues: #187 #188 #189 #190 #191 #192 #193 #194

Deliverables: account assignment, customer notes, team activity feed, mentions, notifications, handoff workflow, team member management, saved views, and collaboration sidebar.

### 18. Additional Integrations

Epic: #26

Issues: #195 #196 #197 #198 #199 #200 #201 #202 #203 #204 #205

Deliverables: generic connector interface, Zendesk OAuth/sync/UI, Salesforce OAuth/sync/UI, PostHog sync/UI, generic webhook receiver/configuration/UI, integration health dashboard, and integration refactor.

### 19. Advanced Pricing & Feature Gating

Epic: #27

Issues: #206 #207 #208 #209 #210 #211

Deliverables: Growth/Scale feature specs, granular flags, API/UI tier gates, upgrade prompts, usage analytics, and plan upgrade/downgrade flow.

## Phase 3: Defensibility

Goal: create differentiated data products and AI-backed customer health insights.

### 20. Anonymized Benchmarking

Epic: #28

Issues: #212 #213 #214 #215 #216 #217 #218

Deliverables: benchmark model, anonymization pipeline, aggregation service, comparison dashboard, industry classification, privacy controls, insight notifications, and data quality validation.

### 21. AI-Powered Insights

Epic: #29

Issues: #219 #220 #221 #222 #223 #224 #225

Deliverables: LLM integration service, prompt templates, per-customer analysis pipeline, customer-detail AI insights UI, action recommendations, LLM cost tracking/rate limits, and dashboard AI summary.

### 22. Predictive Churn Models

Epic: #30

Issues: #226 #227 #228 #229 #230 #231

Deliverables: churn feature engineering, training data preparation, model implementation, prediction scoring service, churn indicators, and forecast dashboard.

### 23. Integration Marketplace

Epic: #31

Issues: #232 #233 #234 #235 #236 #237 #238 #239

Deliverables: connector SDK, developer docs, registration/discovery API, browse UI, install/configuration flow, community submission workflow, connector review/security pipeline, connector analytics, monitoring, search, and recommendations.

## Chronological Build Order

1. Foundation: #32 through #45
2. Schema: #46 through #55
3. Auth: #56 through #70
4. CI/CD and deploy: #38 through #43, parallelizable with auth
5. Stripe integration: #71 through #82
6. Health scoring: #83 through #93
7. API layer: #94 through #103
8. Dashboard UI: #106 through #122
9. HubSpot and Intercom: #123 through #137
10. Email alerts: #138 through #145
11. Onboarding: #146 through #152
12. Billing: #153 through #161
13. Landing page: #162 through #167
14. Documentation: #168 through #172
15. Stripe App Marketplace: #173 through #176
16. Expansion: #177 through #211
17. Defensibility: #212 through #239

## Key Decisions

- VPS deployment over Vercel: Docker Compose plus reverse proxy and SSL.
- Tests are expected for every implementation issue.
- Go, React/TypeScript, and PostgreSQL remain the core stack.
- Stripe OAuth reads customer data; Stripe billing is a separate domain.
- Sub-issues should stay small enough for focused coding agent work.
