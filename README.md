# PulseScore

Customer health scoring platform for B2B SaaS companies. Connect Stripe, HubSpot, and Intercom to monitor customer health with automated scoring.

## Project Structure

```
pulse-score/
├── cmd/api/              # Application entry point
│   └── main.go
├── internal/             # Private application code
│   ├── config/           # Configuration management
│   ├── handler/          # HTTP handlers
│   ├── middleware/        # HTTP middleware
│   ├── model/            # Domain models
│   ├── repository/       # Database access layer
│   └── service/          # Business logic
├── pkg/                  # Reusable packages
├── migrations/           # Database migrations
├── scripts/
│   └── init-db/          # PostgreSQL initialization scripts
├── web/                  # React/TypeScript frontend (Vite + TailwindCSS v4)
│   └── src/
│       └── components/
├── docker-compose.dev.yml
├── Makefile
└── .env.example
```

## Tech Stack

- **Backend:** Go (net/http)
- **Frontend:** React 19, TypeScript, Vite, TailwindCSS v4
- **Database:** PostgreSQL 16
- **Deployment:** Docker, Nginx, VPS

## Prerequisites

- Go 1.24+
- Node.js 20+
- Docker & Docker Compose

## Getting Started

### 1. Clone and configure

```bash
cp .env.example .env
```

### 2. Start the database

```bash
docker compose -f docker-compose.dev.yml up -d
```

### 3. Run the API

```bash
make run
```

The API starts on http://localhost:8080. Health check: `GET /healthz`

### 4. Run the frontend

```bash
cd web
npm install
npm run dev
```

The frontend starts on http://localhost:5173.

## Shipping to Production (VPS)

PulseScore ships to production through a manual GitHub Actions workflow:

- Workflow: `.github/workflows/deploy-prod.yml`
- Trigger: **Actions → Deploy Production → Run workflow**
- Inputs:
    - `ref` (branch/tag/SHA to deploy)
    - `run_migrations` (reserved; currently no-op)

### Required GitHub repository secrets

- `VPS_HOST` — production server hostname/IP
- `VPS_USER` — SSH user on the VPS
- `VPS_SSH_KEY` — private SSH key for deploy user
- `VPS_APP_DIR` — absolute path to repo on VPS (example: `/opt/pulse-score`)

Optional:

- `VPS_PORT` — SSH port (defaults to `22` if omitted by your SSH client/server config)

### What deployment does

The deploy workflow SSHes into your VPS and:

1. Checks out the requested `ref`
2. Ensures the external Docker network `web` exists
3. Pulls latest DB image and rebuilds API/Web images with `--pull`
4. Runs `docker compose -f docker-compose.prod.yml up -d --remove-orphans`
5. Verifies DB readiness (`pg_isready`) and API health (`/healthz`)

### Manual deploy fallback (on VPS)

If needed, you can run the same deploy logic directly on the server:

`./scripts/deploy/vps-deploy.sh`

## Development Commands

### Backend (Makefile)

| Command             | Description                   |
| ------------------- | ----------------------------- |
| `make build`        | Build the Go binary           |
| `make run`          | Build and run the API         |
| `make test`         | Run tests with race detection |
| `make lint`         | Run `go vet`                  |
| `make dev-db`       | Start development PostgreSQL  |
| `make dev-db-down`  | Stop development PostgreSQL   |
| `make migrate-up`   | Run database migrations up    |
| `make migrate-down` | Roll back database migrations |

### Frontend (web/)

| Command           | Description              |
| ----------------- | ------------------------ |
| `npm run dev`     | Start Vite dev server    |
| `npm run seo:validate` | Validate SEO catalog integrity (families/slugs/keywords) |
| `npm run seo:artifacts` | Generate sitemap artifacts in `web/public/sitemaps/` |
| `npm run seo:prerender` | Pre-render SEO routes into `web/dist/` |
| `npm run build`   | Production build + SEO validate/artifacts/prerender |
| `npm run lint`    | ESLint check             |
| `npm run format`  | Format with Prettier     |
| `npm run preview` | Preview production build |

## Billing & Subscription (Epic 12)

PulseScore now includes a dedicated Stripe billing domain (separate from Stripe customer-data integration):

- Plan catalog: `free`, `growth`, `scale` (`internal/billing/plans.go`)
- Protected billing APIs:
    - `GET /api/v1/billing/subscription`
    - `POST /api/v1/billing/checkout` (admin)
    - `POST /api/v1/billing/portal-session` (admin)
    - `POST /api/v1/billing/cancel` (admin)
- Public billing webhook:
    - `POST /api/v1/webhooks/stripe-billing`

### Required production billing env vars

- `STRIPE_BILLING_SECRET_KEY`
- `STRIPE_BILLING_PUBLISHABLE_KEY`
- `STRIPE_BILLING_WEBHOOK_SECRET`
- `STRIPE_BILLING_PRICE_GROWTH_MONTHLY`
- `STRIPE_BILLING_PRICE_GROWTH_ANNUAL`
- `STRIPE_BILLING_PRICE_SCALE_MONTHLY`
- `STRIPE_BILLING_PRICE_SCALE_ANNUAL`

Optional:

- `STRIPE_BILLING_PORTAL_RETURN_URL` (defaults to `http://localhost:5173/settings/billing`)
