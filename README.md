# Mailtarget Sentinel

Real-time anomaly detection, circuit breaker, and super-user dashboard for Mailtarget sub-accounts.

## Features

- **5-minute anomaly worker** — scans `default.event` in ClickHouse (all companies or scoped via `COMPANY_ID`)
- **PostgreSQL enrichment** — company/sub-account/domain metadata from shared dashboard DB
- **Super-user mode** — browse all companies when `COMPANY_ID=0`; scoped testing with `COMPANY_ID=287`
- **Direct suspend/resume** — `UPDATE sub_account.status` in PostgreSQL (not API Config)
- **SQLite alert history** — persistent log for dashboard
- **AMP kill-switch email** — suspend sub-account via webhook
- **React dashboard** — overview, at-risk, companies, sub-accounts, history, settings

## Quick Start

```bash
cp .env.example .env
# Set POSTGRES_DSN to shared dashboard-api-service database
# Configure MAILTARGET_API_KEY (Transmission only) + ClickHouse + Redis

make install        # first time only
make infra-up       # ClickHouse :9000 + Redis :6379
make clickhouse-seed-287 # sample events for company 287
make dev            # backend :8080 + frontend :5173
```

Or run separately:

```bash
make backend   # Go API + worker
make frontend  # React dashboard
```

See [`scripts/clickhouse/README.md`](scripts/clickhouse/README.md) for ClickHouse local setup details.

### Docker

```bash
docker compose up --build
```

## API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/sentinel/metrics` | Bounce/delivery rates (`?window=5m&company_id=`) |
| GET | `/api/v1/sentinel/companies` | List companies from PostgreSQL (`?at_risk=true`) |
| GET | `/api/v1/sentinel/companies/at-risk` | Live at-risk sub-accounts + company summary (CH + PG enriched) |
| GET | `/api/v1/sentinel/companies/at-risk/summary` | Company-level risk summary only |
| GET | `/api/v1/sentinel/sub-accounts` | Sub-accounts from PG + ClickHouse metrics |
| GET | `/api/v1/sentinel/sub-accounts/:id` | Sub-account detail (PG + CH) |
| GET | `/api/v1/sentinel/alerts` | Alert history from SQLite (paginated, filterable) |
| GET | `/api/v1/sentinel/alerts/overview` | Dashboard overview stats |
| GET | `/api/v1/sentinel/alerts/:id` | Single alert detail |
| GET/POST | `/api/v1/sentinel/settings` | Threshold settings (Redis) |
| POST | `/api/v1/sentinel/kill-switch` | AMP webhook — suspend sub-account |
| POST | `/api/v1/sentinel/manual-override` | Manual suspend/resume (PostgreSQL) |

## Configuration

| Variable | Description |
|----------|-------------|
| `POSTGRES_DSN` | **Required.** Shared PostgreSQL from dashboard-api-service |
| `COMPANY_ID` | `0` = super-user (all companies); `287` = scoped testing |
| `SQLITE_PATH` | Path to alert history database (default `./data/sentinel.db`) |
| `CORS_ORIGINS` | Allowed dashboard origins (comma-separated) |
| `MAILTARGET_API_KEY` | Transmission API only (alert emails) |

### Company scope

| `COMPANY_ID` | Behavior |
|--------------|----------|
| `0` or empty | Full super-user — scan all companies in ClickHouse and PostgreSQL |
| `> 0` (e.g. `287`) | Scoped mode — worker + API limited to that company (recommended for dev) |

Query param `?company_id=` overrides env when present.

## Reference Files

- [`event-clickhouse-ddl.sql`](event-clickhouse-ddl.sql)
- [`transmission-openapi-spec.json`](transmission-openapi-spec.json)
- [`apiconfig-openapi-spec.json`](apiconfig-openapi-spec.json) — legacy reference; suspend/resume no longer uses API Config

## AI Agent Onboarding

Switching IDE (Antigravity, JetBrains, Copilot, etc.)? Read **[`AGENTS.md`](AGENTS.md)** first — architecture, API map, and coding conventions for AI assistants.

## One-Pager (Hackathon / Stakeholders)

See **[`ONEPAGER.md`](ONEPAGER.md)** — problem statement, Mailtarget APIs used, business case, and local setup (local Redis/ClickHouse + production PostgreSQL/API).
