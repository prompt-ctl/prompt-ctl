# Promptctl: Open Core Split — Strip Proprietary Code for Public Release

You are preparing the promptctl codebase for open-source release by removing all proprietary, cloud, and internal components. The public repo must compile, pass all tests, and contain zero references to private infrastructure.

## Context
- Repo: /Users/olegkoval/projects/personal/active/prompt-ctl.com/promptctl
- Goal: Remove all proprietary code so this branch can be pushed to a PUBLIC GitHub repo
- Strategy: Open Core — CLI is free, cloud/analytics/mac-app stay private
- Constraint: The CLI must still build and ALL tests must pass after removal
- Constraint: Do NOT delete files from disk — use git rm so they're removed from tracking only

## What to REMOVE (proprietary / cloud / internal)

### Cloud Workers (monetization layer)
- `worker/` — Enhance API, prompt engine, domain knowledge, D1 analytics
- `worker-try/` — OAuth social auth gateway, JWT, rate limiting, try-it-out
- `worker-alerts/` — Monitoring cron worker, alerting

### Analytics & Telemetry
- `internal/analytics/` — GA4 integration, consent management, client tracking
- Remove all imports of `internal/analytics` from other packages
- Remove analytics consent prompts from onboarding if referenced

### Mac App
- `promptctl-app/` — Swift macOS native app, DMG
- `.worktrees/` — Unreleased feature branches with app builds

### Enhance Client (connects to proprietary backend)
- `prompt/enhanceclient.go` — HTTP client calling private enhance worker
- `prompt/enhanceclient_test.go` — Tests for enhance client
- Remove all imports/references to EnhanceViaAPI from cmd/ and other packages
- If any command depends on enhance (e.g. `create` or `send`), stub it to print "This feature requires prompt-ctl cloud. Visit https://prompt-ctl.com for details."

### Internal Scripts
- `scripts/one-time-private-and-release.sh`
- `scripts/setup-public-releases-gh.sh`
- `scripts/setup-brew.sh`
- `scripts/deploy-enhance-and-alerts.sh`
- `scripts/cf-deploy-all.sh`
- `scripts/fix-release.sh`
- `scripts/release-0.2.0.sh`
- `scripts/PROMPT-FOR-LLM-PRICING.md`
- `scripts/PUBLISHING.md`
- `deploy-workers.sh`
- `apply_ci_mode.sh`
- Keep ONLY: `scripts/release.sh`, `scripts/coverage.sh`, `scripts/emulated_user_test.sh`, `scripts/validate_emulated.sh` (if they don't reference private infra)

### Internal Docs
- `docs/WRANGLER_ENV_VARS.md` — Secret names
- `docs/CLI-DEPLOY.md` — Internal deploy process
- `docs/RELEASE-SETUP.md` — Release infrastructure
- `docs/PRESS_RELEASE.md` — Marketing material
- `docs/TWEETS_0.7.1.md` — Social media drafts
- `docs/TWEETS_GEMINI_3.1.md` — Social media drafts
- `docs/TWEETS_SHIP_IT.md` — Social media drafts
- `docs/CHANGELOG_SHIP_IT.md` — Internal changelog
- `docs/chatgpt-ideas.md` — Internal brainstorming
- `docs/FEATURE_CONFIG.md` — Internal feature planning
- `docs/TAP-README.md` — Homebrew tap internal setup
- `docs/ANALYTICS_README.md` — Analytics internal setup
- Keep: README.md, CONTRIBUTING.md, INSTALL.md, ARCHITECTURE.md, ROADMAP.md, TESTING.md, EMULATED_USER_TEST_REPORT.md, SIMULATED_USER_TESTING_REPORT.md

### GitHub Workflows (private release pipeline)
- `.github/workflows/release.yml` — Private→public release pipeline
- `.github/workflows/release-trigger.yml` — Release trigger mechanism
- `.github/workflows/prompt-regression.yml` — Uses API keys, references private infra
- `.github/workflows/docs-index.yml` — Review if safe, keep if no private refs
- Keep: `.github/workflows/ci.yml` (safe, public CI)

### Misc
- `gs` file at repo root — check contents, likely remove
- `package.json` and `package-lock.json` at repo root — only needed for worker tooling, remove if not needed for Go CLI
- `node_modules/` at repo root — remove from tracking
- `.claude/` and `.cursor/` — already gitignored, verify

## Tasks

### 1. Remove Cloud Workers
- [x] `git rm -r worker/`
- [x] `git rm -r worker-try/`
- [x] `git rm -r worker-alerts/`
- [x] Verify no Go code imports anything from these directories
- [x] Remove `deploy-workers.sh` from repo root
- [x] Remove worker-related entries from Makefile (deploy-workers, deploy-enhance, deploy-try, setup-try, analytics-init)
- [x] Keep Makefile targets: build, test, release

### 2. Remove Mac App
- [x] `git rm -r promptctl-app/`
- [x] `git rm -r .worktrees/`
- [x] Add `.worktrees/` to .gitignore

### 3. Remove Analytics
- [x] `git rm -r internal/analytics/`
- [x] Search ALL Go files for `import` references to `internal/analytics`
- [x] For each reference found:
  - Remove the import line
  - Remove ALL code that calls analytics functions (SendEvent, Consent, etc.)
  - Replace with no-op or remove the block entirely
  - Ensure the file still compiles
- [x] Search for `PROMPTCTL_GA4_SECRET` references and remove them
- [x] Search for `G-DQBN89S2FZ` (GA4 measurement ID) and remove
- [x] Run `go build ./...` to verify compilation
- [x] Run `go test ./...` to verify tests pass

### 4. Remove Enhance Client
- [x] `git rm prompt/enhanceclient.go`
- [x] `git rm prompt/enhanceclient_test.go`
- [x] Search ALL Go files for references to `EnhanceViaAPI`, `EnhanceConfig`, `EnhanceResult`, `EnhanceViaAPIWithClient`
- [x] For each reference:
  - If it's a command that calls enhance API (like `create` or `send --enhance`):
    - Keep the command structure
    - Replace the enhance call with: `fmt.Println("The enhance feature requires prompt-ctl cloud. Visit https://prompt-ctl.com")`
    - Return appropriate exit code
  - If it's a test: remove the test or update it
  - If it's a type definition used elsewhere: keep a minimal stub
- [x] Check if `EnhanceConfig` or `EnhanceResult` types are used in other files
  - If yes, keep minimal type definitions in a new file `prompt/enhance_types.go`
  - If no, remove completely
- [x] Run `go build ./...` to verify
- [x] Run `go test ./...` to verify

### 5. Remove Internal Scripts
- [x] `git rm scripts/one-time-private-and-release.sh`
- [x] `git rm scripts/setup-public-releases-gh.sh`
- [x] `git rm scripts/setup-brew.sh`
- [x] `git rm scripts/deploy-enhance-and-alerts.sh`
- [x] `git rm scripts/cf-deploy-all.sh`
- [x] `git rm scripts/fix-release.sh`
- [x] `git rm scripts/release-0.2.0.sh`
- [x] `git rm scripts/PROMPT-FOR-LLM-PRICING.md`
- [x] `git rm scripts/PUBLISHING.md`
- [x] `git rm deploy-workers.sh`
- [x] `git rm apply_ci_mode.sh`
- [x] Review remaining scripts (release.sh, coverage.sh, emulated_user_test.sh, validate_emulated.sh) for private infra references
  - If they reference private repos, API keys, or internal URLs: remove or sanitize
  - If clean: keep

### 6. Remove Internal Docs
- [x] `git rm docs/WRANGLER_ENV_VARS.md`
- [x] `git rm docs/CLI-DEPLOY.md`
- [x] `git rm docs/RELEASE-SETUP.md`
- [x] `git rm docs/PRESS_RELEASE.md`
- [x] `git rm docs/TWEETS_0.7.1.md`
- [x] `git rm docs/TWEETS_GEMINI_3.1.md`
- [x] `git rm docs/TWEETS_SHIP_IT.md`
- [x] `git rm docs/CHANGELOG_SHIP_IT.md`
- [x] `git rm docs/chatgpt-ideas.md`
- [x] `git rm docs/FEATURE_CONFIG.md`
- [x] `git rm docs/TAP-README.md`
- [x] `git rm docs/ANALYTICS_README.md`
- [x] Update `docs/index.html` if it exists and references removed docs
- [x] Verify remaining docs don't reference removed files

### 7. Remove Private Workflows
- [x] `git rm .github/workflows/release.yml`
- [x] `git rm .github/workflows/release-trigger.yml`
- [x] `git rm .github/workflows/prompt-regression.yml`
- [x] Review `.github/workflows/docs-index.yml` — remove if it references private infra
- [x] Verify `.github/workflows/ci.yml` has no references to secrets beyond what's needed for public CI

### 8. Remove Misc Files
- [x] Check `gs` file contents — if internal, `git rm gs`
- [x] Check if `package.json` and `package-lock.json` are needed for Go CLI
  - If only for workers: `git rm package.json package-lock.json`
  - If needed for some Go tooling: keep
- [x] `git rm -r node_modules/` if tracked
- [x] Remove `coverage.out` if tracked
- [x] Remove `dist/` if tracked

### 9. Update .goreleaser.yaml
- [x] Review homebrew_casks section
  - Update repository owner if moving to new org
  - Or remove homebrew_casks entirely if handling separately
- [x] Remove any references to `promptctl-releases` private repo
- [x] Remove any references to `HOMEBREW_TAP_GITHUB_TOKEN` if not applicable
- [x] Verify builds section is correct for public release
- [x] Ensure ldflags reference correct module path

### 10. Update Makefile
- [x] Remove targets: deploy-workers, deploy-enhance, deploy-try, setup-try, analytics-init
- [x] Keep targets: build, test, release
- [x] Verify `make build` works
- [x] Verify `make test` works

### 11. Update .gitignore
- [x] Add entries for removed directories (so contributors don't recreate them):
  - `worker/`
  - `worker-try/`
  - `worker-alerts/`
  - `promptctl-app/`
  - `.worktrees/`
  - `internal/analytics/`
- [x] Verify existing entries are still relevant

### 12. Final Grep Audit
- [x] Search entire codebase for private URLs:
  - `prompt-ctl/promptctl-releases`
  - `prompt-ctl/homebrew-tap`
  - `prompt-ctl/promptctl` (should be updated to new org if applicable)
  - Any Cloudflare worker URLs
  - Any wrangler references
- [x] Search for API keys / secrets patterns:
  - `sk-ant-` (Anthropic key prefix)
  - `sk-` followed by long string
  - `PROMPTCTL_GA4_SECRET`
  - `CF_API_TOKEN`
  - `CF_ACCOUNT_ID`
  - `ALERT_WEBHOOK_URL`
  - `HOMEBREW_TAP_GITHUB_TOKEN`
- [x] Search for internal references:
  - `promptctl-enhance`
  - `promptctl-try`
  - `promptctl-alerts`
  - `wrangler`
  - `D1`
  - `Analytics Engine`
- [x] For each finding: remove or sanitize

### 13. Compile and Test
- [x] Run `go build -o promptctl .` — must succeed
- [x] Run `go vet ./...` — no issues
- [x] Run `go test ./...` — all tests pass
- [x] Run `./promptctl version` — prints correct version
- [x] Run `./promptctl init` — works in temp directory
- [x] Run `./promptctl list` — shows available templates
- [x] Run `./promptctl score` — works on a test template
- [x] Run `./promptctl --help` — all commands listed, no broken references

### 14. Commit
- [x] Stage all changes: `git add -A`
- [x] Commit: `git commit -m "chore: strip proprietary code for open-source release"`
- [x] Verify commit diff looks correct (only removals + stubs, no new proprietary code)

## Success Criteria
- ✓ Zero references to private infrastructure (workers, analytics, wrangler, private repos)
- ✓ Zero API keys, measurement IDs, or secrets in codebase
- ✓ `go build` succeeds
- ✓ `go vet` clean
- ✓ `go test ./...` all pass
- ✓ CLI commands work: version, init, list, score, review, debug, arch, commit
- ✓ Enhance-dependent features gracefully degrade with "requires cloud" message
- ✓ Makefile has no private targets
- ✓ .goreleaser.yaml has no private references
- ✓ No internal docs, scripts, or workflows remain

### 15. Preserve Rule-Based Enhance Mode
The CLI has an offline rule-based enhance mode (PROMPTCTL_ENHANCE=rule) that works WITHOUT the cloud API.
This is a KEY differentiator for the OSS version — free prompt enhancement without cloud dependency.
- [x] Check if rule-based enhance logic lives in prompt/enhance.go (NOT enhanceclient.go)
- [x] If rule-based logic is in enhance.go: KEEP it (this is open-source gold)
- [x] If rule-based logic is mixed with cloud client: extract it into its own file
- [x] Verify `promptctl create` or `promptctl fix` can still use rule-based mode offline
- [x] Ensure PROMPTCTL_ENHANCE=rule env var still works
- [x] Update README to highlight offline/rule-based enhance as a feature
- [x] The cloud enhance (LLM-powered via worker) should gracefully degrade to rule-based when no API

### 16. Bundle Pi Integration Package
A working pi extension already exists at ~/.pi/agent/extensions/promptctl-integration.ts
Create a pi package inside the public repo so users can `pi install` promptctl integration.
- [ ] Create `pi-package/` directory in repo root
- [ ] Create `pi-package/package.json` with pi-package keyword:
  ```json
  {
    "name": "@prompt-ctl/pi-promptctl",
    "version": "1.0.0",
    "keywords": ["pi-package"],
    "pi": {
      "extensions": ["./extensions"],
      "skills": ["./skills"]
    }
  }
  ```
- [ ] Create `pi-package/extensions/index.ts` based on existing promptctl-integration.ts
  - /promptctl <template> [--var=value] — render template to LLM
  - /quick-templates — list available templates
  - /cost-score — evaluate prompt efficiency
  - Register `promptctl_apply` tool for LLM to call
- [ ] Create `pi-package/skills/promptctl/SKILL.md` teaching the LLM:
  - What promptctl does (template-based prompt engineering)
  - When to suggest promptctl commands (review, debug, arch, score, fix)
  - How to recommend adding promptctl to projects
  - Example workflows
- [ ] Test pi extension loads: `pi -e ./pi-package/extensions/index.ts`
- [ ] Update README with "Use with pi" section:
  ```
  pi install npm:@prompt-ctl/pi-promptctl
  ```
- [ ] Add pi integration to docs/ROADMAP.md

### 17. Include Promotion Materials
- [ ] Review CASE_FOR_PROMPTCTL.md — keep as-is (excellent marketing)
- [ ] Check if docs/EMULATED_USER_TEST_REPORT.md is safe to include (shows testing rigor)
- [ ] Check if docs/SIMULATED_USER_TESTING_REPORT.md is safe to include
- [ ] Do NOT include any Bizcuit-specific templates or references (work project, private)
- [ ] Do NOT include PROMPTCTL_TEAM_PITCH.md (internal sales material)

## Notes
- Use `git rm` not `rm` — we want git to track the removal
- After removal, the enhance/create command should still exist but print a cloud upsell message
- The CLI must be fully functional offline without any cloud dependency
- CRITICAL: Keep rule-based enhance mode — it's the offline free version
- Keep CASE_FOR_PROMPTCTL.md — it's excellent marketing material for the OSS repo
- Keep docs/EMULATED_USER_TEST_REPORT.md and SIMULATED_USER_TESTING_REPORT.md — they show rigor
- Do NOT modify the private repo — only modify files on this branch
- Do NOT include any Bizcuit/work references (from ICM: Oleg's day job is private)
- The pi-package is a distribution multiplier — pi users discover promptctl through the marketplace
