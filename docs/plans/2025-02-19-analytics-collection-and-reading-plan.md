# Analytics Collection and Reading — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Persist user feedback in Cloudflare D1, wire worker to write feedback on POST `/feedback`, add IaC (migrations + bootstrap script + deploy integration), and add a single reading guide for GA4, Analytics Engine, and D1.

**Architecture:** Enhance worker gets a D1 binding; POST `/feedback` inserts into table `feedback`. Migrations live in `worker/migrations/`; bootstrap script creates D1 and updates wrangler.toml; deploy runs migrations then wrangler deploy. Reading guide lives in `docs/ANALYTICS_README.md`.

**Tech Stack:** Cloudflare Worker (existing), D1, Wrangler migrations, Bash (bootstrap script).

**Design reference:** `docs/plans/2025-02-19-analytics-collection-and-reading-design.md`

---

### Task 1: D1 migration file

**Files:**
- Create: `worker/migrations/0001_feedback.sql`

**Step 1: Create migration**

Add file with:

```sql
CREATE TABLE IF NOT EXISTS feedback (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  received_at TEXT NOT NULL,
  text TEXT NOT NULL,
  source TEXT NOT NULL DEFAULT 'cli'
);
```

**Step 2: Commit**

```bash
git add worker/migrations/0001_feedback.sql
git commit -m "chore(worker): add D1 migration for feedback table"
```

---

### Task 2: Bootstrap script and wrangler.toml placeholder

**Files:**
- Create: `scripts/init-analytics-d1.sh`
- Modify: `worker/wrangler.toml` (add `[[d1_databases]]` block with placeholder or comment that script will set database_id)

**Step 1: Add D1 block to wrangler.toml**

In `worker/wrangler.toml` add (use a placeholder id or leave database_id for script to inject):

```toml
[[d1_databases]]
binding = "ANALYTICS_DB"
database_name = "promptctl-analytics"
database_id = "REPLACE_BY_INIT_SCRIPT"
```

If Wrangler requires a valid UUID for deploy, use a dummy UUID and document that `scripts/init-analytics-d1.sh` replaces it; or have the script create the block entirely.

**Step 2: Create bootstrap script**

Create `scripts/init-analytics-d1.sh` (executable):

- Run `cd worker && wrangler d1 list` (or `wrangler d1 create promptctl-analytics`).
- If `promptctl-analytics` exists: parse its id from `wrangler d1 list` output and ensure `worker/wrangler.toml` has `[[d1_databases]]` with that `database_id`.
- If not: run `wrangler d1 create promptctl-analytics`, parse the created database id from output (e.g. "Created database promptctl-analytics (xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx)"), then update `worker/wrangler.toml` so the D1 block has that `database_id`.
- Script must be idempotent (re-run safe). Use `grep -q promptctl-analytics` on wrangler.toml or list output to decide create vs update.

**Step 3: Commit**

```bash
git add worker/wrangler.toml scripts/init-analytics-d1.sh
git commit -m "chore: add D1 binding and analytics bootstrap script"
```

---

### Task 3: Worker handleFeedback writes to D1

**Files:**
- Modify: `worker/src/index.js` (handleFeedback and request path that calls it)
- Test: `worker/src/index.test.js` (or existing test file that hits /feedback)

**Step 1: Add test for feedback persistence**

In worker tests: mock env with `ANALYTICS_DB` (D1 stub). POST `/feedback` with valid body; assert that the D1 binding's execute was called with an INSERT (or that in integration test the row appears). If current tests don't mock D1, add a test that when env.ANALYTICS_DB is present, handleFeedback calls env.ANALYTICS_DB.prepare(...).run(...) with expected params.

**Step 2: Run test to verify it fails**

Run: `cd worker && npm test`  
Expected: New test fails (e.g. no INSERT called or method not defined).

**Step 3: Implement handleFeedback D1 write**

In `worker/src/index.js`:

- In `handleFeedback`, after validating `text` (non-empty, length ≤ 8000), if `env.ANALYTICS_DB` exists: call `env.ANALYTICS_DB.prepare('INSERT INTO feedback (received_at, text, source) VALUES (?, ?, ?)').bind(new Date().toISOString(), text, 'cli').run()`. Catch errors: log and optionally write one Analytics Engine data point (e.g. blob 'feedback_error'). Then return `jsonResponse({ ok: true })`.
- Ensure the request router passes `env` into `handleFeedback` (if not already).

**Step 4: Run tests**

Run: `cd worker && npm test`  
Expected: All tests pass.

**Step 5: Commit**

```bash
git add worker/src/index.js worker/src/index.test.js
git commit -m "feat(worker): persist feedback to D1 on POST /feedback"
```

---

### Task 4: Deploy applies migrations and Makefile target

**Files:**
- Modify: `Makefile` (add `analytics-init`, ensure deploy-enhance or deploy-workers runs migrations)
- Modify: `deploy-workers.sh` or the script that deploys the enhance worker (if exists)

**Step 1: Add analytics-init target**

In `Makefile`: add target `analytics-init` that runs `./scripts/init-analytics-d1.sh` (and ensure script is executable in repo).

**Step 2: Run migrations before deploy**

Ensure when enhance worker is deployed, migrations run first. Options:
- In `Makefile`, change `deploy-enhance` to: `(cd worker && wrangler d1 migrations apply promptctl-analytics --remote && wrangler deploy)`, or
- In `worker/package.json`, change `"deploy": "wrangler d1 migrations apply promptctl-analytics --remote && wrangler deploy"`, and have `make deploy-enhance` run `cd worker && npm run deploy`.

Use the same pattern in `deploy-workers.sh` if it deploys the enhance worker: before deploying, run migrations for promptctl-analytics.

**Step 3: Commit**

```bash
git add Makefile deploy-workers.sh
git commit -m "chore: run D1 migrations before deploy, add analytics-init"
```

---

### Task 5: Document D1 binding in WRANGLER_ENV_VARS.md

**Files:**
- Modify: `docs/WRANGLER_ENV_VARS.md`

**Step 1: Add promptctl-enhance D1 binding**

Under the promptctl-enhance section, add a row for bindings: `ANALYTICS_DB` (D1) for database `promptctl-analytics`, used for feedback storage. Note that one-time setup: run `make analytics-init` from repo root.

**Step 2: Commit**

```bash
git add docs/WRANGLER_ENV_VARS.md
git commit -m "docs: document ANALYTICS_DB D1 binding for enhance worker"
```

---

### Task 6: Reading guide docs/ANALYTICS_README.md

**Files:**
- Create: `docs/ANALYTICS_README.md`

**Step 1: Write overview table**

Section "Overview": table with columns What | Where | How to read. Rows: Behavioral events (site + CLI) → GA4 → Google Analytics UI; Request metrics + ratings → Analytics Engine → SQL API / dashboard; Feedback → D1 → wrangler d1 execute / dashboard.

**Step 2: GA4 section**

Link to analytics.google.com, property G-DQBN89S2FZ. List CLI events: onboarding_started, onboarding_completed, onboarding_skipped, model_selected, prompt_created, prompt_saved, prompt_rated. Point to promptctl-site `docs/ANALYTICS_EVENTS.md` for site funnel. How: Reports → Engagement → Events; Explorations for funnels.

**Step 3: Analytics Engine section**

Link to Cloudflare Analytics Engine SQL API docs. Dataset `promptctl_enhance`. Schema: blobs (e.g. status, path or "rating", value), doubles (intent_len, count). Example queries (copy-paste): requests last 24h by status; error rate last 7 days; ratings by day (blob0 = 'rating'); ratings distribution 1–5. How to run: curl with API token or Cloudflare dashboard.

**Step 4: D1 section**

Table `feedback`. Command: `wrangler d1 execute promptctl-analytics --remote --command "SELECT id, received_at, source, substr(text, 1, 80) as preview FROM feedback ORDER BY received_at DESC LIMIT 10"`. Link to Cloudflare dashboard → D1 → promptctl-analytics → Execute SQL.

**Step 5: Future dashboard sentence**

One line: a future internal dashboard can read from D1, Analytics Engine SQL, and GA4 API.

**Step 6: Commit**

```bash
git add docs/ANALYTICS_README.md
git commit -m "docs: add analytics reading guide (GA4, Analytics Engine, D1)"
```

---

### Task 7: Optional — worker-alerts or error visibility

**Files:**
- Modify: `worker/src/index.js` (optional)

If design called for writing a "feedback_error" data point to Analytics Engine when D1 insert fails: in handleFeedback catch block, if env.ENHANCE_ANALYTICS exists, call writeDataPoint with blobs ['feedback_error', 'd1_insert_failed'] (or similar). Keeps satisfaction metrics visible. Commit as optional follow-up.

---

## Execution summary

- Tasks 1–2: Migration + bootstrap and wrangler config.
- Task 3: Worker persistence and tests.
- Task 4: Deploy pipeline and make analytics-init.
- Task 5: Wrangler env docs.
- Task 6: Reading guide.
- Task 7: Optional error metric.

After completing the plan, run once: `make analytics-init` (or `./scripts/init-analytics-d1.sh`) to create D1 and set database_id, then deploy as usual so migrations apply and worker has D1 binding.
