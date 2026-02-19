# promptctl make targets

.PHONY: release build test deploy-workers deploy-enhance deploy-try setup-try analytics-init

# Release: bump version in cmd/root.go if VERSION given, commit, push main, tag, push tag.
# Triggers workflow -> promptctl-releases -> Homebrew cask.
# Usage: make release   (use version from cmd/root.go)   or: make release VERSION=0.4.7
release:
	@./scripts/release.sh $(VERSION)

build:
	go build -o promptctl .

test:
	go test ./...
	@cd worker && npm test 2>/dev/null || true

# One-time: create D1 promptctl-analytics and set database_id in worker/wrangler.toml. Prereq: wrangler login, jq.
analytics-init:
	@./worker/scripts/init-analytics-d1.sh

# Cloudflare: deploy both workers (enhance + try). Prereq: wrangler login. For enhance: run make analytics-init once first.
deploy-workers:
	@./deploy-workers.sh

deploy-enhance:
	@(cd worker && wrangler d1 migrations apply promptctl-analytics --remote && wrangler deploy)

deploy-try:
	@(cd worker-try && wrangler deploy)

# One-time: generate .dev.vars and print OAuth redirect URI + secret commands.
setup-try:
	@node worker-try/scripts/setup-try-env.cjs
