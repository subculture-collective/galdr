.PHONY: build run test test-cover lint clean \
       migrate-up migrate-down migrate-down-all migrate-create seed \
       dev-db dev-db-down dev dev-stop \
       web-install web-dev web-build web-lint web-format web-format-check web-preview \
	docker-build docker-up deploy-prod docker-down docker-logs \
       check help

# ---------------------------------------------------------------------------
# Config
# ---------------------------------------------------------------------------
BINARY_NAME  = pulsescore-api
BUILD_DIR    = bin
DATABASE_URL ?= postgres://pulsescore:pulsescore@localhost:5434/pulsescore_dev?sslmode=disable

# ---------------------------------------------------------------------------
# Go API
# ---------------------------------------------------------------------------
build: ## Build the Go API binary
	go build -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/api

run: build ## Build and run the API
	./$(BUILD_DIR)/$(BINARY_NAME)

test: ## Run Go tests with race detector
	go test ./... -v -race

test-cover: ## Run Go tests with coverage report
	go test ./... -race -coverprofile=coverage.out
	go tool cover -func=coverage.out
	@rm -f coverage.out

lint: ## Run Go vet
	go vet ./...

clean: ## Remove build artifacts
	rm -rf $(BUILD_DIR) web/dist

# ---------------------------------------------------------------------------
# Database & Migrations
# ---------------------------------------------------------------------------
dev-db: ## Start dev Postgres container
	docker compose -f docker-compose.dev.yml up -d

dev-db-down: ## Stop dev Postgres container
	docker compose -f docker-compose.dev.yml down

migrate-up: ## Run all pending migrations
	migrate -path migrations -database "$(DATABASE_URL)" up

migrate-down: ## Roll back the last migration
	migrate -path migrations -database "$(DATABASE_URL)" down 1

migrate-down-all: ## Roll back all migrations
	migrate -path migrations -database "$(DATABASE_URL)" down -all

migrate-create: ## Create a new migration (NAME=my_migration)
	@if [ -z "$(NAME)" ]; then echo "Usage: make migrate-create NAME=my_migration"; exit 1; fi
	migrate create -ext sql -dir migrations -seq $(NAME)

seed: ## Seed the database with sample data
	psql "$(DATABASE_URL)" -f scripts/seed.sql

# ---------------------------------------------------------------------------
# Frontend (web/)
# ---------------------------------------------------------------------------
web-install: ## Install frontend dependencies
	cd web && npm install

web-dev: ## Start Vite dev server
	cd web && npm run dev

web-build: ## Build frontend for production
	cd web && npm run build

web-lint: ## Lint frontend code
	cd web && npm run lint

web-format: ## Format frontend code with Prettier
	cd web && npm run format

web-format-check: ## Check frontend formatting
	cd web && npm run format:check

web-preview: ## Preview production build locally
	cd web && npm run preview

# ---------------------------------------------------------------------------
# Full-stack dev
# ---------------------------------------------------------------------------
dev: dev-db ## Start DB, API, and frontend dev server
	@echo "Waiting for Postgres..."
	@until docker compose -f docker-compose.dev.yml exec -T postgres pg_isready -q 2>/dev/null; do sleep 1; done
	@echo "Postgres ready. Starting API and frontend..."
	$(MAKE) run &
	$(MAKE) web-dev

dev-stop: dev-db-down ## Stop all dev services
	@-pkill -f "$(BINARY_NAME)" 2>/dev/null || true

# ---------------------------------------------------------------------------
# Docker (production)
# ---------------------------------------------------------------------------
docker-build: ## Build production Docker images
	docker compose -f docker-compose.prod.yml build

docker-up: ## Start production stack
	docker compose -f docker-compose.prod.yml up -d

deploy-prod: ## Deploy production stack (build + remove orphan containers)
	docker compose -f docker-compose.prod.yml up -d --build --remove-orphans

docker-down: ## Stop production stack
	docker compose -f docker-compose.prod.yml down

docker-logs: ## Tail production logs
	docker compose -f docker-compose.prod.yml logs -f

# ---------------------------------------------------------------------------
# CI / Quality
# ---------------------------------------------------------------------------
check: lint test web-lint web-format-check ## Run all linters and tests

# ---------------------------------------------------------------------------
# Help
# ---------------------------------------------------------------------------
help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'
