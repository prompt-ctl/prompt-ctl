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
| `TRY_JWT_SECRET`          | JWT signing (or use `SUPABASE_JWT_SECRET`) |
| `GOOGLE_CLIENT_ID`        | Google OAuth |
| `GOOGLE_CLIENT_SECRET`    | Google OAuth |
| `GITHUB_CLIENT_ID`        | GitHub OAuth |
| `GITHUB_CLIENT_SECRET`    | GitHub OAuth |
| `MARKETPLACE_WEBHOOK_SECRET` | GitHub Marketplace webhook verification |

**Bindings (not env):** `ENHANCE_SERVICE` (service), `TRY_ACCOUNTS` (KV).

---

## promptctl-enhance (`worker/`)

**From wrangler.toml:** No `[vars]`. Uses bindings: `AI`, `ENHANCE_ANALYTICS`, `ANALYTICS_DB` (D1).

**Bindings (not env):** `AI` (Workers AI), `ENHANCE_ANALYTICS` (Analytics Engine dataset `promptctl_enhance`), `ANALYTICS_DB` (D1 database `promptctl-analytics`, used for feedback storage). One-time setup: from repo root run **`make analytics-init`** to create the D1 database and set `database_id` in `worker/wrangler.toml`.

**Secrets:** Run `cd worker && npx wrangler secret list` (requires `CLOUDFLARE_API_TOKEN` if not logged in).

---

## promptctl-enhance-alerts (`worker-alerts/`)

**From wrangler.toml [vars]:**
| Variable           | Set in .dev.vars or production |
|--------------------|--------------------------------|
| `CF_ACCOUNT_ID`    | Account ID (dashboard URL)     |
| `ALERT_WEBHOOK_URL`| Optional; e.g. Slack webhook    |

**Secrets:** `CF_API_TOKEN` (and optionally others). Run:
`cd worker-alerts && npx wrangler secret list`

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
