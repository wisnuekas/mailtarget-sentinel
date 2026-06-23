# Mailtarget Sentinel — One-Pager

## What It Is

**Mailtarget Sentinel** is a real-time reputation guardrail for Mailtarget customers. It watches sending behavior across sub-accounts and sending domains, detects bounce/spam anomalies within minutes, alerts the right people, and can **suspend a sub-account immediately** before reputation damage spreads.

Think of it as a **circuit breaker** on top of existing Mailtarget sending data — not a replacement for the platform, but an operational safety layer for Mailtarget ops and enterprise customers.

---

## The Problem Today

Most of the time, the team only learns about issues **after an accident has already happened**:

| Incident | What happens today |
|----------|-------------------|
| **IP blacklisted** | Deliverability drops; customers complain; ops investigates logs retroactively |
| **Spamhaus / blocklist listing** | External alert or customer report triggers firefighting |
| **Leaked client API key** | Abusive sending continues until volume spikes or complaints pile up |
| **Runaway bounce/spam rate** | Reputation damage on shared pools before anyone can isolate the sub-account |

Reaction is **manual, slow, and expensive**. By the time someone connects the dots, the blast radius may already include shared IPs, domains, and neighboring customers.

**Sentinel shifts the model from reactive to proactive** — detect abnormal sending in rolling 5-minute windows, notify the company owner (with ops CC’d), and offer one-click suspend **before** a small problem becomes a platform-wide incident.

---

## What It Does

| Capability | How |
|------------|-----|
| **Anomaly detection** | Cron worker scans ClickHouse `default.event` every 5 minutes for bounce/spam spikes above configurable thresholds |
| **At-risk visibility** | Super-user dashboard: at-risk companies, sub-accounts, and sending domains — enriched with names from PostgreSQL |
| **Automated alerting** | AMP email with one-click **Kill Switch** to suspend a sub-account |
| **Human-in-the-loop** | Ops can suspend/resume from dashboard, resend alerts, or send warning emails |
| **Post-suspend recovery** | After kill-switch via email, customer receives a follow-up email with **Resume** button |
| **Audit trail** | Alert history (SQLite): `detected` → `alert_sent` → `suspended` → `resolved` |
| **Scoped or global** | Monitor one company (e.g. 287) or all companies — configurable in dashboard Settings |

**Data flow:** ClickHouse first (metrics & detection) → PostgreSQL enrich (company / sub-account / domain / owner) → suspend/resume via `UPDATE sub_account.status` in the shared dashboard database.

---

## Mailtarget APIs & Platform Integration

### Mailtarget APIs Used

| API | Endpoint | Purpose |
|-----|----------|---------|
| **Transmission API** | `POST /v1/layang/transmissions` | Anomaly alerts (AMP kill-switch), warning emails, resume confirmation |

**Auth:** `Authorization: Bearer {MAILTARGET_API_KEY}`

**Email routing (anomaly alerts):**

- **To:** company owner (`company.owner_id` → `user.email` from PostgreSQL)
- **CC:** ops team (`ALERT_TO_EMAIL` / `ALERT_TO_NAME` from env)
- **AMP:** kill-switch suspend + resume actions embedded in the email body

### Platform Data (Mailtarget stack, not HTTP APIs)

| Source | Usage |
|--------|--------|
| **ClickHouse** `default.event` | Read-only: sent, bounced, spam bounces, delivery rates |
| **PostgreSQL** (dashboard DB) | Read: `company`, `sub_account`, `domain`, `user` — Write: `sub_account.status` |

> **API Config** (`apiconfig.mailtarget.co`) is **not used**. Suspend/resume writes directly to PostgreSQL for reliability and super-user access.

---

## Business Case

### Value for Mailtarget

| Stakeholder | Benefit |
|-------------|---------|
| **Platform ops** | Early warning instead of post-mortem after blacklist / Spamhaus |
| **Reputation / deliverability** | Circuit breaker limits damage from leaked keys or bad campaigns |
| **Enterprise customers** | Owner gets the alert; can suspend in one click from email |
| **Product** | Deep integration: Transmission API + ClickHouse events + dashboard PostgreSQL |

### Expected outcome

- **Before:** incident → discovery (hours/days) → manual isolation → reputation cleanup  
- **After:** anomaly (minutes) → owner + ops notified → suspend in one click → resume when fixed  

---

## Architecture at a Glance

```
ClickHouse (events) ──► Sentinel Worker ──► Transmission API (alert email)
        │                      │
        ▼                      ▼
PostgreSQL (metadata)    Kill-switch / Resume webhook
        │                      │
        ▼                      ▼
React Dashboard ◄──── API (Fiber) ──► Redis (settings) + SQLite (history)
```

---

## Local Setup Instructions

This is the recommended **hackathon / dev** layout:

| Component | Where it runs | Notes |
|-----------|---------------|--------|
| **Redis** | Local (Docker) | Settings, alert dedup locks, kill-switch tokens |
| **ClickHouse** | Local (Docker) | Sample `default.event` data via seed scripts |
| **PostgreSQL** | **Production** (shared dashboard DB) | Real company / sub-account / domain / owner data |
| **Transmission API** | **Production** | Real email delivery (`MAILTARGET_API_KEY`) |

PostgreSQL and Mailtarget API are **not** bundled in `docker-compose.yml` — point your `.env` at production (or staging) endpoints you are allowed to use.

### Prerequisites

- Go 1.23+
- Node.js 18+ and npm
- Docker + Docker Compose
- VPN / network access to production PostgreSQL (if required)
- Valid `MAILTARGET_API_KEY` with Transmission send permission
- Verified sender domain for `ALERT_FROM_EMAIL`

### 1. Clone and configure environment

```bash
git clone <repo-url> mailtarget-sentinel
cd mailtarget-sentinel

cp .env.example .env
cp web/.env.example web/.env   # optional; login session handles admin token
```

Edit **`.env`** — minimum required:

```env
# --- Production PostgreSQL (dashboard-api-service DB) ---
POSTGRES_DSN=postgres://USER:PASS@<prod-pg-host>:5432/<dbname>?sslmode=require

# --- Production Mailtarget Transmission ---
MAILTARGET_API_KEY=<your-production-api-key>
MAILTARGET_TRANSMISSION_URL=https://transmission.mailtarget.co/v1
ALERT_FROM_EMAIL=alerts@your-verified-domain.com
ALERT_FROM_NAME=Mailtarget Sentinel
ALERT_TO_EMAIL=ops@yourdomain.com
ALERT_TO_NAME=Ops Team

# --- Local ClickHouse & Redis (defaults match docker compose) ---
CLICKHOUSE_HOST=localhost:9000
REDIS_ADDR=localhost:6379

# --- Scope (optional; dashboard Settings can override) ---
COMPANY_ID=287

# --- Kill-switch & admin ---
PUBLIC_BASE_URL=http://localhost:8080
KILL_SWITCH_HMAC_SECRET=change-me-in-production
SENTINEL_ADMIN_TOKEN=dev-sentinel-secret-2026
DASHBOARD_USERNAME=admin
DASHBOARD_PASSWORD=s3ntinelg0d
```

> **Kill-switch from Gmail:** `PUBLIC_BASE_URL` must be a **public URL** (e.g. ngrok) for email links to work outside your machine. `localhost` is fine for dashboard-only testing.

### 2. Install dependencies

```bash
make install
```

### 3. Start local infrastructure (Redis + ClickHouse only)

```bash
make infra-up
```

This starts:

- ClickHouse native `:9000`, HTTP `:8123`
- Redis `:6379`

### 4. Seed sample ClickHouse events (optional demo data)

For company **287**, sub-account **4302** (~8% bounce):

```bash
make clickhouse-seed-287
```

Data uses `now()` — **expires in ~5 minutes**. Re-run before demos.

### 5. Run the application

**Backend + dashboard together:**

```bash
make dev
```

| Service | URL |
|---------|-----|
| Dashboard | http://localhost:5173 |
| API | http://localhost:8080 |
| Health | http://localhost:8080/health |

**Login:** `admin` / `s3ntinelg0d`

**Or run separately:**

```bash
make backend    # Go API + worker on :8080
make frontend   # React on :5173
```

### 6. Verify it works

```bash
# Health
curl http://localhost:8080/health

# At-risk (after seed)
curl 'http://localhost:8080/api/v1/sentinel/companies/at-risk?window=5m'

# Manual worker run (populate alert history)
make worker-run
```

In the dashboard: **Overview** → **At Risk** → **Sub-accounts** → **Settings** (company ID default **287**).

### 7. Docker all-in-one (alternative)

Runs ClickHouse, Redis, Sentinel, and Vite dev dashboard:

```bash
docker compose up --build
```

Override `POSTGRES_DSN` and `MAILTARGET_API_KEY` in `.env` before starting. The `sentinel` container still expects production PostgreSQL reachable from the container network (may need host gateway or VPN sidecar).

---

## Demo Quick Reference

| Item | Value |
|------|-------|
| Dashboard login | `admin` / `s3ntinelg0d` |
| Default company scope | `287` (Settings page) |
| Demo seed | `make clickhouse-seed-287` |
| Suspend/resume | Dashboard buttons or email kill-switch / resume |

---

## Safety Notes

- Sentinel **reads** ClickHouse and PostgreSQL; **writes** only `sub_account.status` (suspend/resume) plus local SQLite alert history.
- Use **scoped company ID** (`287`) when testing against production PostgreSQL to avoid scanning all tenants.
- Do not commit `.env` with production credentials.

---

*Mailtarget Sentinel — be proactive before blacklist, Spamhaus, and leaked keys become your wake-up call.*
