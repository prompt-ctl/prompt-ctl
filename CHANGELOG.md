# Changelog

All notable changes to promptctl are documented here.

## [Unreleased]

## [0.9.3] – 2026-02-19

- **Loader:** Spinner clears line before each update (no leftover text). Model name shown on the right during prompt preparation.
- **Send:** Pre-call line shows model and estimated tokens (in/out). Cost report shows actual model and token stats.
- **Config:** API key prompt uses masked input (password-style). Browser opens automatically when pressing Enter on API key page.

## [0.9.0] – 2026-02-19

- **Gemini 3.1 Pro:** Google provider with model `gemini-3.1-pro`. `promptctl send` supports `--thinking-level=low|high` and `--media-resolution=low|medium|high` for Gemini; usage tracking uses `usageMetadata` (PDF tokens under IMAGE modality). Set `GOOGLE_API_KEY` or `promptctl config --provider=google --api-key=...`.
- **Score and fix:** New commands `promptctl score` and `promptctl fix` for prompt quality. `score` rates prompt files (0–100) by structure, clarity, constraints, and persona; CI-friendly exit codes and `--format=json`. `fix` applies fixes to low-scoring files. Optional config file `.promptctl/score.yaml` (dirs, include, ignore, min_score, rules). Flags override config: `--min-score`, `--format=json`, `--dry-run`, `--suggest`.
- **First-time onboarding:** On first run (interactive), show welcome wizard: 4 steps (init, default output format, LLM config, shell aliases). Overview of steps before starting; welcome box with aligned formatting. Mark completed with `~/.promptctl/first_run_done`.
- **Default output format:** Onboarding and create flow ask for output format (Markdown, XML, YAML, JSON, Plain text). "Remember my choice?" saves to `~/.promptctl/create_format`; next create uses it without asking. Hint: "To change later: promptctl create --format=... \"...\"."
- **Shell aliases:** `promptctl init` offers to add aliases (e.g. `prompt`, `p`) to `~/.zshrc`/`~/.bashrc`. If names are taken, ask for alternatives; if aliases already set, ask "Skip alias setup?" (default Yes). One-time tip after create or in usage when no aliases.
- **Usage and tips:** Usage text mentions init and aliases. One-time alias tip shown when running `promptctl` or `promptctl help` (and after create) if user hasn't added aliases.
- **Quality score:** Framed box (Unicode border) with bold score and hints; hints wrap to multiple lines (no truncation). Score shown on first run and on every retry.
- **Rating:** Horizontal prompt: "Rate this output: 1  2  3  4  5  [s]kip" then type 1–5 or s. Parser accepts "51" (default+typed) as 1; case-insensitive labels. Retry offered when rating 1–3.
- **Prompt name suggestion:** When saving to memory, suggest a name from intent (slug); if duplicate, append timestamp. User can accept (Enter) or type their own.
- **Analyze spinner:** Rotating funny messages while enhancing ("Consulting the prompt oracle...", "Polishing your words...", etc.) instead of a single spinner character.
- **Version check:** Once per 24h (when stderr is TTY), fetch latest release from GitHub; if newer, print hint at end of run: "Upgrade: brew upgrade --cask ...". Skip for `promptctl version`.
- **Scoring and retry:** If user rates output 1–3, offer retry (no daily limit); loop until 4/5, Skip, or 15 tries. Score is always saved to ratings and analytics.
- **Prompt display:** Colored section headers (markdown `#`/`##`/`###` and XML tags) in terminal; normalized newlines between sections and trailing newline.
- **Save to memory:** List existing folders when saving; after save, offer to open the folder in Finder (or system file manager).
- **Feedback session:** From time to time (every 10th create or every 7 days), prompt for anonymous freeform feedback; submit to enhance `/feedback` or append to `~/.promptctl/feedback.log`. Worker adds `POST /feedback` (no auth).
- **Atlas (hosted):** Config/onboarding skip API key step when Atlas is selected; no browser prompt or key paste.

## [0.8.0] – 2026-02-18

- **CLI UX:** Survey-based flows when TTY: model picker (`promptctl models`), rating and save-to-memory after create, onboarding config wizard; optional ANSI color for success/hint; onboarding skipped state with one-line reminder.
- **Analytics:** GA4 Measurement Protocol client; first-run consent prompt (Y/n); events: `onboarding_started`, `onboarding_completed`, `onboarding_skipped`, `model_selected`, `prompt_created`, `prompt_saved`, `prompt_rated`.
- **Docs:** README and design notes for CLI TUI and analytics; GA4 secret via `PROMPTCTL_GA4_SECRET` for CLI.

## [0.7.4] – 2026-02-17

- **Enhance Worker (prompt generation):** Rebuilt prompt engine: rule-based domain + task classification, domain knowledge (gaming, fintech, saas, e-commerce, mobile, ai_ml, devops, education, healthcare, social, gardening, general), intent enrichment (tech, MVP, monetize, scale), and structured prompt assembly. Optional LLM refinement with Llama 3.1 8B for polish. Placeholder intents still get minimal prompts (no hallucination).
- **Gardening domain:** Intents like "plant watering plan" now get horticulture-style prompts instead of software-architect.
- **Worker tests:** Added promptEngine.test.js (classifyIntent, enrichIntent, assemblePrompt); README Testing section.

## [0.7.1] – 2026-02-17

- **Models:** Default model selectable via `promptctl models --set`; added Atlas (Promptctl hosted LLM) as optional model; clearer models list (value-focused intro, column labels).
- **Config:** Remove or update API key per provider (`--api-key=` or `--remove-api-key`); README clarifies create uses hosted enhancer (no user API key).
- **Ratings:** If user rates output < 3, offer one free retry per day; ratings persisted to `~/.promptctl/ratings.json` and POST to enhance `/rating`; worker POST /rating for analytics; daily digest doc for hello@prompt-ctl.com.
- **Site analytics:** Events for Try button click, install/copy, Google OAuth login; auth redirect includes provider for `try_login_google`.
- **Worker:** worker-alerts cron schema fix (`crons` array); WRANGLER_ENV_VARS.md and secret-list loop.
- Try auth: error redirect no double-slash; fragment `reason` (missing, state, exchange, email, config) for debugging.
- GitHub OAuth troubleshooting in worker-try docs (callback URL, secrets).

## [0.7.0]

- worker-try: Try promptctl API (OAuth + JWT, 5 tries per account); GitHub and Google sign-in.
- deploy-workers.sh, docs/CLI-DEPLOY.md; .gitignore node_modules and worker-try/.wrangler.

## [0.6.0]

- Implement memory management features in promptctl.
- Added commands for managing saved prompts: `memory list`, `memory open`, and `memory set-dir`.
- Introduced functionality to list prompts in a specified directory, open the prompts directory in the file manager, and set a custom directory for saved prompts.
- Updated configuration to support a dedicated prompts directory, with persistence through environment variables and configuration files.
- Enhanced user interaction by prompting to save enhanced prompts to memory.
- Added tests for new memory management features and updated existing tests.

## [0.5.0]

- Current stable. See GitHub Releases for details.

## [0.4.6]

- Binaries published to public [promptctl-releases](https://github.com/oleg-koval/promptctl-releases); Homebrew cask points there (install with no token).
- `make release VERSION=x.y.z` bumps version in `cmd/root.go`, commits, tags, and triggers the release workflow.
- Docs page on site; install/upgrade use `brew install --cask` / `brew upgrade --cask`.

## [0.4.5] – [0.4.3]

- Release pipeline and Homebrew cask updates.

## [0.4.2]

- Stable release. Includes all improvements from 0.4.0 and 0.4.1.

## [0.4.1]

- Stability and packaging updates.

## [0.4.0]

### Security & usability

- Path safety checks to prevent path traversal in command execution.
- Template name validation – safe naming conventions enforced.
- README updates: HTTPS requirement for custom Worker URLs, security guidelines for API keys and webhooks.
- New tests for path safety and error handling.

## [0.3.5]

- Documentation and docs site updates.

## [0.3.4]

- **Tests:** Expanded test coverage (config, prompt enhance, enhanceclient, root command).
- **Worker:** Cloudflare Worker for prompt enhancement with rate limits and analytics.
- Deploy script and README improvements.
- Config package with tests.

## [0.3.3]

- Multi-model cost comparison (`promptctl cost --compare`).
- Structured prompt generation and template runs. Pipe to Claude CLI, OpenAI CLI, or send directly with cost tracking.
