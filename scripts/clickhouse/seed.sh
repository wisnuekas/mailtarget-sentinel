#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
COMPOSE="docker compose -f ${ROOT}/docker-compose.yml"
CLIENT="${COMPOSE} exec -T clickhouse clickhouse-client --multiquery"
SEED_FILE="${1:-${ROOT}/scripts/clickhouse/seed-dev.sql}"

echo "==> Seeding ClickHouse from $(basename "${SEED_FILE}") …"
${CLIENT} < "${SEED_FILE}"
echo "==> Done."
