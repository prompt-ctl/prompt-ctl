# Wrangler env vars (all workers)

Secrets are **names only**; values are never shown. Set secrets with:
`npx wrangler secret put SECRET_NAME` (from the worker directory).

Local overrides: `.dev.vars` in each worker dir (gitignored).

---

## promptctl-try (`worker-try/`)

**From wrangler.toml [vars]:**
| Variable       | Example / note |
|----------------|-----------------|
| `ENHANCE_URL`  | `https://promptctl-enhance.kvl-olg.workers.dev` |

**Secrets** (list with `cd worker-try && npx wrangler secret list`):
| Secret                    | Purpose |
|---------------------------|---------|
| `TRY_JWT_SECRET`          | JWT signing (or use `SUPABASE_JWT_SECRET` for backward compat) |
| `GOOGLE_CLIENT_ID`        | Google OAuth |
| `GOOGLE_CLIENT_SECRET`    | Google OAuth |
| `GITHUB_CLIENT_ID`        | GitHub OAuth |
| `GITHUB_CLIENT_SECRET`    | GitHub OAuth |
| `MARKETPLACE_WEBHOOK_SECRET` | GitHub Marketplace webhook verification |

**Bindings (not env):** `ENHANCE_SERVICE` (service), `TRY_ACCOUNTS` (KV).

---

## promptctl-enhance (`worker/`)

**From wrangler.toml:** No `[vars]`. No secrets required. Uses bindings only: `AI`, `ENHANCE_ANALYTICS`, `ANALYTICS_DB` (D1).

**Bindings (not env):** `AI` (Workers AI), `ENHANCE_ANALYTICS` (Analytics Engine dataset `promptctl_enhance`), `ANALYTICS_DB` (D1 database `promptctl-analytics`, used for feedback storage). One-time setup: from repo root run **`make analytics-init`** to create the D1 database and set `database_id` in `worker/wrangler.toml`.

**Secrets:** None. If `PROMPTCTL_GA4_SECRET` appears in CF for this worker, it is unused (GA4 is used by the CLI only); you can remove it with `cd worker && npx wrangler secret delete PROMPTCTL_GA4_SECRET`.

---

## promptctl-enhance-alerts (`worker-alerts/`)

**From wrangler.toml:** No `[vars]` keys (set via secrets or .dev.vars). Worker not deployed by default; deploy with `make deploy-workers` from repo that includes worker-alerts.

**Secrets** (set with `npx wrangler secret put <NAME>` from `worker-alerts/`):
| Secret               | Purpose |
|----------------------|---------|
| `CF_ACCOUNT_ID`      | Account ID (dashboard URL); can be var or secret |
| `CF_API_TOKEN`       | API token for Analytics Engine SQL API |
| `ALERT_WEBHOOK_URL`  | Optional; e.g. Slack/Discord webhook for alert payloads |

---

## List all secret names (any worker)

```bash
# From repo root (promptctl repo). worker-alerts may error if not deployed yet.
for dir in worker-try worker worker-alerts; do
  echo "=== $dir ==="
  (cd "$dir" && npx wrangler secret list) 2>/dev/null || echo "(none or worker not deployed)"
done
```

To list **vars** from wrangler.toml only (no secrets):
```bash
grep -A 20 '^\[vars\]' worker-try/wrangler.toml worker/wrangler.toml worker-alerts/wrangler.toml
```
