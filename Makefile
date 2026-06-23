.PHONY: help install install-backend install-web backend frontend dev build build-backend build-web run-bin test env clean docker-up docker-down infra-up infra-down clickhouse-up clickhouse-down clickhouse-seed clickhouse-seed-287 clickhouse-reset clickhouse-client

BACKEND_PORT ?= 8080
WEB_PORT     ?= 5173
COMPOSE      ?= docker compose

help: ## Show available targets
	@echo "Mailtarget Sentinel — Makefile"
	@echo ""
	@grep -E '^[a-zA-Z0-9_-]+:.*##' $(firstword $(MAKEFILE_LIST)) | sort | awk 'BEGIN {FS = ":.*## "}; {printf "  \033[36m%-16s\033[0m %s\n", $$1, $$2}'

install: install-backend install-web ## Install Go + npm dependencies

install-backend: ## go mod download
	go mod download

install-web: ## npm install in web/
	cd web && npm install

backend: ## Run Go API + worker (port $(BACKEND_PORT))
	@bash -c 'set -a; [ -f .env ] && . ./.env; set +a; go run ./cmd/sentinel'

frontend: install-web ## Run React dashboard (port $(WEB_PORT))
	cd web && npm run dev -- --port $(WEB_PORT)

dev: ## Run backend + frontend in parallel
	@echo "Starting backend :$(BACKEND_PORT) and frontend :$(WEB_PORT) …"
	@$(MAKE) -j2 backend frontend

build: build-backend build-web ## Build backend binary + web dist

build-backend: ## Build sentinel binary to ./bin/sentinel
	@mkdir -p bin data
	go build -o bin/sentinel ./cmd/sentinel

build-web: install-web ## Production build for web/
	cd web && npm run build

run-bin: build-backend ## Run compiled binary (requires .env)
	@bash -c 'set -a; [ -f .env ] && . ./.env; set +a; ./bin/sentinel'

test: ## Run Go tests
	go test ./...

env: ## Copy .env.example to .env if missing
	@if [ ! -f .env ]; then cp .env.example .env && echo "Created .env from .env.example"; else echo ".env already exists"; fi

clean: ## Remove build artifacts
	rm -rf bin/ web/dist web/node_modules/.vite
	go clean -cache -testcache 2>/dev/null || true

docker-up: ## docker compose up --build
	docker compose up --build

docker-down: ## docker compose down
	$(COMPOSE) down

infra-up: ## Start ClickHouse + Redis for local dev (make backend / make dev)
	$(COMPOSE) up -d clickhouse redis
	@echo "ClickHouse native :9000 | HTTP :8123 | Redis :6379"
	@echo "Run 'make clickhouse-seed' to load sample events"

infra-down: ## Stop ClickHouse + Redis
	$(COMPOSE) stop clickhouse redis

clickhouse-up: infra-up ## Alias for infra-up

clickhouse-down: infra-down ## Alias for infra-down

clickhouse-seed: ## Insert sample events (anomalies for company 42 & 99)
	@chmod +x scripts/clickhouse/seed.sh
	@scripts/clickhouse/seed.sh scripts/clickhouse/seed-dev.sql
	@echo "Try: curl 'http://localhost:8080/api/v1/sentinel/companies/at-risk?window=5m'"

clickhouse-seed-287: ## Seed company_id=287 sub_account_id=4302 (anomaly ~8% bounce)
	@chmod +x scripts/clickhouse/seed.sh
	@scripts/clickhouse/seed.sh scripts/clickhouse/seed-company-287.sql
	@$(MAKE) --no-print-directory _trigger-worker

_trigger-worker:
	@echo "Triggering worker to record history …"
	@-curl -s -X POST http://localhost:$(BACKEND_PORT)/api/v1/sentinel/worker/run \
		-H "Authorization: Bearer $$(grep SENTINEL_ADMIN_TOKEN .env 2>/dev/null | cut -d= -f2-)" \
		| jq . 2>/dev/null || echo "(start backend with make backend to auto-detect)"

worker-run: ## Trigger detection worker manually (populate history)
	curl -s -X POST http://localhost:$(BACKEND_PORT)/api/v1/sentinel/worker/run \
		-H "Authorization: Bearer $$(grep SENTINEL_ADMIN_TOKEN .env 2>/dev/null | cut -d= -f2-)" | jq .

clickhouse-reset: ## Wipe ClickHouse volume and re-init schema
	$(COMPOSE) down clickhouse
	docker volume rm mailtarget-sentinel_clickhouse_data 2>/dev/null || true
	$(COMPOSE) up -d clickhouse
	@echo "Waiting for ClickHouse …"
	@until $(COMPOSE) exec clickhouse wget -q -O- http://localhost:8123/ping >/dev/null 2>&1; do sleep 1; done
	@echo "ClickHouse ready. Run 'make clickhouse-seed' for sample data."

clickhouse-client: ## Open clickhouse-client shell
	$(COMPOSE) exec clickhouse clickhouse-client
