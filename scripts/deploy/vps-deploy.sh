#!/usr/bin/env bash

set -Eeuo pipefail

log() {
  printf "\n[%s] %s\n" "$(date -u +"%Y-%m-%dT%H:%M:%SZ")" "$*"
}

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

DEPLOY_REF="${DEPLOY_REF:-}"
COMPOSE_FILE="${COMPOSE_FILE:-docker-compose.prod.yml}"
RUN_MIGRATIONS="${RUN_MIGRATIONS:-false}"

if [[ -n "$DEPLOY_REF" ]]; then
  log "Syncing repository to ref: $DEPLOY_REF"
  git fetch --prune origin

  if git show-ref --verify --quiet "refs/remotes/origin/${DEPLOY_REF}"; then
    git checkout "$DEPLOY_REF"
    git reset --hard "origin/${DEPLOY_REF}"
  else
    # Allows deploying tags/commit SHAs as well.
    git checkout "$DEPLOY_REF"
  fi
fi

if [[ -f ".env" ]]; then
  log "Loading environment from .env"
  set -a
  # shellcheck disable=SC1091
  source .env
  set +a
fi

if ! docker network inspect web >/dev/null 2>&1; then
  log "Creating external docker network: web"
  docker network create web
fi

log "Pulling latest database image"
docker compose -f "$COMPOSE_FILE" pull db || true

log "Building application images"
docker compose -f "$COMPOSE_FILE" build --pull api web

log "Starting production stack"
docker compose -f "$COMPOSE_FILE" up -d --remove-orphans

if [[ "$RUN_MIGRATIONS" == "true" ]]; then
  log "RUN_MIGRATIONS=true requested. No automated migration command is configured; skipping."
fi

log "Container status"
docker compose -f "$COMPOSE_FILE" ps

if [[ -n "${POSTGRES_USER:-}" && -n "${POSTGRES_DB:-}" ]]; then
  log "Checking database readiness"
  docker compose -f "$COMPOSE_FILE" exec -T db \
    pg_isready -U "$POSTGRES_USER" -d "$POSTGRES_DB"
fi

log "Checking API health endpoint"
docker compose -f "$COMPOSE_FILE" exec -T api \
  wget --no-verbose --tries=1 --spider http://localhost:8080/healthz

log "Production deployment completed successfully."
