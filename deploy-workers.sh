#!/usr/bin/env bash
# Deploy Cloudflare Workers (enhance + try) from promptctl repo root.
# Prereqs: wrangler (npm install -g wrangler), wrangler login.
# Usage: ./deploy-workers.sh   or: make deploy-workers
set -e
ROOT="$(cd "$(dirname "$0")" && pwd)"
cd "$ROOT"

command -v wrangler >/dev/null 2>&1 || { echo "Install wrangler: npm install -g wrangler"; exit 1; }
wrangler whoami >/dev/null 2>&1 || { echo "Run: wrangler login"; exit 1; }

echo "Deploying promptctl-enhance..."
(cd worker && npm ci --omit=dev 2>/dev/null || true && wrangler d1 migrations apply promptctl-analytics --remote && wrangler deploy)
echo "Deploying promptctl-try..."
(cd worker-try && npm ci --omit=dev 2>/dev/null || true && wrangler deploy)

echo ""
echo "Workers deployed. Set secrets if needed:"
echo "  cd worker-try && wrangler secret put TRY_JWT_SECRET"
echo "  cd worker-try && wrangler secret put GOOGLE_CLIENT_ID"
echo "  cd worker-try && wrangler secret put GOOGLE_CLIENT_SECRET"
echo "  cd worker-try && wrangler secret put GITHUB_CLIENT_ID"
echo "  cd worker-try && wrangler secret put GITHUB_CLIENT_SECRET"
echo "  cd worker-try && wrangler secret put MARKETPLACE_WEBHOOK_SECRET   # optional, for GitHub Marketplace webhook"
echo ""
echo "Deploy site (from promptctl-site repo):"
echo "  wrangler pages deploy public --project-name=promptctl-site"
