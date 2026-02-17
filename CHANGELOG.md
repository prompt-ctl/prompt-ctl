# Changelog

All notable changes to promptctl are documented here.

## [Unreleased]

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
