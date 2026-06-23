# ClickHouse Local Development

Run ClickHouse locally with Docker for Sentinel development and testing.

## Quick Start

```bash
# 1. Start ClickHouse + Redis
make infra-up

# 2. Load sample events (anomalies for testing)
make clickhouse-seed

# 3. Start Sentinel (separate terminal)
make backend
# or full stack:
make dev
```

Verify:

```bash
# HTTP ping
curl http://localhost:8123/ping

# At-risk companies (after seed + backend running)
curl http://localhost:8080/api/v1/sentinel/companies/at-risk?window=5m | jq
```

## Ports

| Port | Protocol | Used by |
|------|----------|---------|
| `9000` | Native TCP | Go backend (`CLICKHOUSE_HOST=localhost:9000`) |
| `8123` | HTTP | curl, `clickhouse-client`, Play UI |

## Environment (`.env`)

```env
CLICKHOUSE_HOST=localhost:9000
CLICKHOUSE_HTTP=http://localhost:8123
CLICKHOUSE_DATABASE=default
CLICKHOUSE_USERNAME=default
CLICKHOUSE_PASSWORD=
COMPANY_ID=0          # monitor all companies (recommended for dev)
```

When Sentinel runs **inside Docker Compose**, `CLICKHOUSE_HOST` is set automatically to `clickhouse:9000`.

## Make Targets

| Command | Description |
|---------|-------------|
| `make infra-up` | Start ClickHouse + Redis |
| `make infra-down` | Stop ClickHouse + Redis |
| `make clickhouse-seed` | Insert dev sample data (company 42 & 99) |
| `make clickhouse-seed-287` | Seed company **287** / sub-account **4302** (~8% bounce anomaly) |
| `make clickhouse-reset` | Wipe data volume + re-run schema init |
| `make clickhouse-client` | Interactive SQL shell |

## Sample Seed Data

After `make clickhouse-seed`:

| Company | Sub-account | Sent | Bounce rate | Expected |
|---------|-------------|------|-------------|----------|
| 42 | 101 | 200 | ~7.5% | **Anomaly** |
| 99 | 201 | 150 | ~8% + spam | **Anomaly** |
| 42 | 102 | 500 | ~1% | Healthy |

Seed uses `now() - INTERVAL N MINUTE` so data stays inside the 5-minute detection window. Re-run `make clickhouse-seed` anytime data ages out.

## Schema

Init script: [`scripts/clickhouse/init.sql`](init.sql) — mirrors [`event-clickhouse-ddl.sql`](../../event-clickhouse-ddl.sql).

On first container start, Docker mounts init SQL to `/docker-entrypoint-initdb.d/`.

## Useful Queries

```bash
make clickhouse-client
```

```sql
-- Event counts by type (last 5 min)
SELECT type, count()
FROM default.event
WHERE injection_time >= now() - INTERVAL 5 MINUTE
GROUP BY type;

-- Bounce rate per sub-account
SELECT
    company_id,
    sub_account_id,
    countIf(type = 'injection') AS sent,
    countIf(type = 'bounce') AS bounced,
    round(bounced / sent * 100, 2) AS bounce_pct
FROM default.event
WHERE injection_time >= now() - INTERVAL 5 MINUTE
GROUP BY company_id, sub_account_id
ORDER BY bounce_pct DESC;
```

## Troubleshooting

**Backend fails with `clickhouse ping: connection refused`**
- Run `make infra-up` first
- Check `docker compose ps` — clickhouse should be `healthy`

**Dashboard shows no at-risk data**
- Run `make clickhouse-seed` (data expires after ~5 minutes)
- Confirm `COMPANY_ID=0` in `.env` to scan all companies

**Reset everything**
```bash
make clickhouse-reset
make clickhouse-seed
```

## Full Docker Stack

To run backend + dashboard + infra together:

```bash
make docker-up
```

This starts ClickHouse, Redis, Sentinel, and the Vite dashboard.
