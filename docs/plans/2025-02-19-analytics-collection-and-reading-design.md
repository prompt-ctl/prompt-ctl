# Analytics Collection and Reading — Design

**Status:** Approved  
**Date:** 2025-02-19

## Goal

1. Collect user activity and satisfaction properly (fix feedback persistence).
2. Provide a single “where to read what” guide for the founder.
3. Use Cloudflare D1 for feedback; keep GA4 and Analytics Engine as-is. All setup and deploy via IaC (no manual steps after one-time bootstrap).

## Decisions

| Topic | Choice |
|-------|--------|
| Feedback storage | D1 database `promptctl-analytics`, table `feedback`. Worker persists on POST `/feedback`. |
| Ratings / request metrics | Unchanged: Analytics Engine dataset `promptctl_enhance`. |
| Behavioral events | Unchanged: GA4 (site + CLI). |
| IaC | Migrations in repo; bootstrap script creates D1 and updates wrangler.toml; deploy applies migrations then deploys. |

---

## Section 1: Architecture and data flow

**Unchanged:**
- **GA4** (G-DQBN89S2FZ): site (cookie consent) + CLI (Measurement Protocol when `PROMPTCTL_GA4_SECRET` set).
- **Analytics Engine** `promptctl_enhance`: request metrics (status, path, intent length) and ratings (1–5 + intent_len).
- CLI continues to POST `/feedback` and `/rating` to enhance worker. No CLI code change.
- Site: no new events; existing GA4 events stay.

**New:**
- **D1** `promptctl-analytics`: one table for feedback.
- **Worker:** On POST `/feedback`, after validation, insert one row into D1 and return 200.
- **Single reading guide:** `docs/ANALYTICS_README.md` (GA4, Analytics Engine, D1 + example queries).

**Data flow:**
- Feedback: CLI (or future site) → Worker POST `/feedback` → D1. Read via D1 SQL/dashboard.
- Ratings: CLI → Worker POST `/rating` → Analytics Engine.
- Activity: CLI → GA4; site → GA4; Worker request/error counts → Analytics Engine.

---

## Section 2: D1 schema and worker changes

**D1:**
- Database: `promptctl-analytics` (created by bootstrap script).
- Table `feedback`:

| Column       | Type    | Notes |
|-------------|---------|--------|
| id          | INTEGER | PRIMARY KEY AUTOINCREMENT |
| received_at | TEXT    | ISO8601 |
| text        | TEXT    | Max 8000 chars |
| source      | TEXT    | `'cli'` or `'site'`; default `'cli'` |

- Migration: `worker/migrations/0001_feedback.sql` with `CREATE TABLE IF NOT EXISTS feedback (...)`.

**Wrangler:**
- `worker/wrangler.toml`: add `[[d1_databases]]` with `binding = "ANALYTICS_DB"`, `database_name = "promptctl-analytics"`, `database_id` (set by bootstrap script). Document in `docs/WRANGLER_ENV_VARS.md`.

**Worker code:**
- `handleFeedback`: after validation (non-empty, length ≤ 8000), `INSERT INTO feedback (received_at, text, source) VALUES (?, ?, 'cli')` via D1 binding; then return 200. On D1 error: log and still return 200; optional: write one “feedback_error” data point to Analytics Engine.

---

## Section 2 addendum: IaC (no manual actions)

**In repo:**
- `worker/migrations/0001_feedback.sql`.
- `worker/wrangler.toml` includes D1 binding (database_id set by bootstrap).

**Bootstrap (one-time):**
- Script `scripts/init-analytics-d1.sh` (or `make analytics-init`): runs `wrangler d1 create promptctl-analytics` (or idempotent check), parses `database_id`, ensures `worker/wrangler.toml` has the `[[d1_databases]]` block with that id. Single command; no copy-paste.

**Every deploy:**
- Deploy path (e.g. `deploy-workers.sh` or `make deploy-enhance`) runs: (1) `wrangler d1 migrations apply promptctl-analytics --remote`, (2) `wrangler deploy`. Optional: `worker/package.json` deploy script runs migrations then deploy.

---

## Section 3: Reading guide

**Doc:** `docs/ANALYTICS_README.md` in promptctl repo.

**Contents:**
1. **Overview** — Table: What | Where | How (GA4, Analytics Engine, D1).
2. **GA4** — Link to analytics.google.com, property; list CLI events; pointer to promptctl-site `docs/ANALYTICS_EVENTS.md` for site funnel; how to open Reports/Explorations.
3. **Analytics Engine** — Link to SQL API; dataset `promptctl_enhance`; schema (blobs, doubles); copy-paste example queries: requests last 24h, error rate, ratings by day, ratings distribution; how to run (curl or dashboard).
4. **D1** — Table `feedback`; `wrangler d1 execute promptctl-analytics --remote --command "SELECT ..."` and dashboard; example “last 10 feedback” query.
5. **Future dashboard** — One sentence: D1 + Analytics Engine SQL + GA4 API can power a future internal dashboard.

Length: ~1–2 pages.

---

## Out of scope (this design)

- Changing GA4 or Analytics Engine schema.
- New site events or satisfaction signals from the website.
- Building the future dashboard (only documented as next step).
