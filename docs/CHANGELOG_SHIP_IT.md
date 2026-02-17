# Changelog — Ship it release

## Summary

- Tiered savings (55% / 64% / 71%) and annual projections on site + CLI
- Landing page polish from reviewer feedback
- Mac DMG in release pipeline; Atlas (hosted LLM) in models list
- Analytics for Try, install/copy, Google login; config key remove/update; create rating + free retry
- launch.cab badge; WRANGLER_ENV_VARS and worker-alerts cron fix

---

## CLI & core

### Savings and cost

- **Tiered unstructured multiplier** by model price (llm/provider.go):
  - Input/MTok > 10 → 2.2× (~55% savings)
  - Input/MTok > 2 → 2.8× (~64%)
  - Input/MTok ≤ 2 → 3.5× (~71%)
- **`promptctl cost --compare`** prints annual projection (e.g. “At 30 calls/day, structured prompting saves ~$X–Y/year”).
- **`promptctl savings`** (new) shows annual savings range; optional `--calls-per-day=N` (default 30).
- **`promptctl send`** “Saved vs unstructured” uses tiered multiplier for selected model.
- **FormatCostEstimate** no longer says “avg. 3x”; uses model-specific multiplier.

### Models and config

- **Atlas** (codename for hosted LLM): provider `promptctl`, model `atlas`; uses `PROMPTCTL_LLM_URL` and optional `PROMPTCTL_API_KEY`.
- **`promptctl models`**: value-focused intro, clearer column headers (In/Out $ per 1M tok, Max context), “Change default: promptctl models --set”.
- **Config**: remove key with `--api-key=` or `--api-key=remove` or `--remove-api-key`; update with `--api-key=newkey`.

### Create (enhance)

- **Rating 1–5**: persisted to `~/.promptctl/ratings.json`, POST to worker `/rating`.
- **Rating < 3**: offer “try again for free” once per calendar day (UTC); state in `~/.promptctl/free_retry_used`.
- README clarifies: your API key is not used for `create`; only for `send` and `cost`.

---

## Landing (promptctl-site + docs)

### Copy and layout

- Hero sub shortened (e.g. “Structures every prompt for max signal, min waste. Claude, GPT-5, Groq, DeepSeek — save ~67%” → “55–71%” where applicable).
- Cookie bar: bottom-right corner, card style, no longer over hero.
- Try = primary (solid orange); Install = secondary (outline).
- Large-screen section/hero padding reduced (e.g. 1200px+).
- “Verified across 10 models” badge: dedicated `.verified-badge` styling.
- Terminal: first line (command) has stronger contrast (background + left border).
- Cost table: row hover with background + orange left border on first cell.
- Section labels: 12px, font-weight 600, letter-spacing 2.5px.

### Cost and annual

- Cost table: model-specific waste and savings (55%, 64%, 71%) and matching $.
- New section “What that looks like over a year”: Light (10 calls/day → $200–400), Regular (30 → $500–1,200), Heavy (100 → $1,500–3,000); line “90% of developers overspend $2,000+/year…”.
- Meta/OG/twitter and hero updated to “55–71%” where relevant.

### Other

- launch.cab badge in hero (link to launch.cab/product/promptctl).
- promptctl-site and docs index both updated for cost table and annual section.

---

## Mac app and releases

- **promptctl-app/** stub: SwiftUI Mac app (Package.swift, main.swift, Info.plist).
- **scripts/build-mac-app.sh**: builds app and `promptctl-macos.dmg` with hdiutil.
- **.github/workflows/release.yml**: job `build-mac-app` (macos-14) builds DMG and uploads to oleg-koval/promptctl-releases on tag.

---

## Workers and analytics

### worker-try

- Auth redirect fragment includes `provider` (e.g. `#try-auth=...&provider=google`).
- Site sends `try_signin_success` with provider; `try_login_google` when provider is Google.

### worker (enhance)

- **POST /rating**: body `{ rating, intent_len? }`; writes to Analytics Engine (promptctl_enhance).
- Worker README: daily ratings digest to hello@prompt-ctl.com (query + email integration documented, not implemented).

### worker-alerts

- wrangler.toml: `cron` → `crons` (array) for current Wrangler schema.

### Site analytics (main.js, try-promptctl.js)

- **try_button_click** (source: inline | cta | nav).
- **install_copy** (install_method: brew | npm | other, value: copied text).
- **try_login_google** on Google OAuth success.
- docs/ANALYTICS_EVENTS.md updated.

---

## Docs and ops

- **docs/WRANGLER_ENV_VARS.md**: all workers’ vars and secret names; how to list/set; loop that tolerates missing worker.
- **docs/MAC-APP.md** (if present): referenced for Mac app build and releases.
- **docs/RELEASE.md**: note that promptctl-macos.dmg must be on latest release for “Download for Mac.”

---

## Tests and build

- Go: `go test ./...` (cmd, config, llm, prompt, safepath).
- llm: TestUnstructuredMultiplier, TestEstimateCost (tiered), TestAnnualSavingsProjection.
- worker: `npm test` in worker/ (e.g. 10 tests).
- CLI: `go build -o promptctl .`; `./promptctl version` (e.g. v0.7.0).
