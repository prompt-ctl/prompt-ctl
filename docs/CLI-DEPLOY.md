# Deploy promptctl via CLI (Cloudflare)

All steps use the Cloudflare CLI (`wrangler`). No dashboard required after one-time setup.

## Prerequisites

```bash
npm install -g wrangler
wrangler login
```

## One-time setup

### 1. KV namespace for promptctl-try (if new)

```bash
cd worker-try
wrangler kv namespace create TRY_ACCOUNTS
wrangler kv namespace create TRY_ACCOUNTS --preview
```

Put the returned `id` and `preview_id` into `worker-try/wrangler.toml` under `[[kv_namespaces]]`.

### 2. Try worker env and secrets

```bash
node worker-try/scripts/setup-try-env.cjs
```

Then fill `.dev.vars` with Google/GitHub OAuth client IDs and secrets.

**OAuth credentials** (Google/GitHub client ID and secret) are created only in the web UIs; `gh` and `gcloud` cannot create or read them. To open the setup pages and get the callback URL:

```bash
cd worker-try && ./scripts/open-oauth-setup.sh
```

Create the OAuth apps, add the callback URL `https://promptctl-try.kvl-olg.workers.dev/auth/callback`, then put the four values into `.dev.vars`. Push to production in one go:

```bash
./scripts/set-production-secrets.sh
```

## Deploy workers (promptctl repo)

```bash
# Deploy both enhance + try
make deploy-workers
# or
./deploy-workers.sh
```

Single worker:

```bash
cd worker && wrangler deploy
cd worker-try && wrangler deploy
```

## Deploy site (promptctl-site repo)

```bash
cd promptctl-site
make deploy
# or
wrangler pages deploy public --project-name=promptctl-site
```

First time (create project + deploy):

```bash
cd promptctl-site
./deploy-site.sh
```

## Summary

| What            | Where          | Command |
|-----------------|----------------|---------|
| promptctl-enhance | promptctl/worker   | `cd worker && wrangler deploy` |
| promptctl-try    | promptctl/worker-try | `cd worker-try && wrangler deploy` |
| promptctl-site   | promptctl-site     | `wrangler pages deploy public --project-name=promptctl-site` |
| Both workers     | promptctl          | `make deploy-workers` or `./deploy-workers.sh` |
