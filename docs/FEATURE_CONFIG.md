# Feature and configuration reference

Where promptctl reads config and how to change behavior.

## Config locations

| Source | Path / env | Used for |
|--------|------------|----------|
| Global dir | `~/.promptctl/` | Templates, create format, prompts dir, onboarding state, analytics consent, LLM config |
| Local dir | `.promptctl/` (cwd or parent) | Local templates only |
| Env | `PROMPTCTL_*`, `ANTHROPIC_API_KEY`, etc. | Override enhance URL/mode, prompts dir, API keys |

## App config (promptctl behavior)

Loaded by `config.Load()`. No single file; options come from env and small files under `~/.promptctl/`.

| Option | How to set | Default | Purpose |
|--------|------------|---------|---------|
| **Default create format** | `~/.promptctl/create_format` (one line: `markdown`, `xml`, `yaml`, `json`, `text`) or onboarding / `promptctl create --format=...` then "Remember my choice?" | (ask each time) | Output format for `promptctl create` when `--format` not passed |
| **Prompts dir (memory)** | `~/.promptctl/prompts_dir` (one line: path) or `PROMPTCTL_PROMPTS_DIR` or `promptctl memory set-dir <path>` | `~/.promptctl/templates` | Where saved prompts (memory) are stored |
| **Enhance mode** | `PROMPTCTL_ENHANCE` | `llm` | `llm` = use hosted Worker; `rule` = offline rule-based only |
| **Enhance URL** | `PROMPTCTL_ENHANCE_URL` | Hosted Worker URL | API for prompt enhancement when mode is `llm` |
| **Templates (global)** | `~/.promptctl/templates/*.yaml` | (created by `promptctl init`) | Reusable prompt templates |
| **Templates (local)** | `.promptctl/templates/*.yaml` | — | Project-level overrides |
| **First run done** | `~/.promptctl/first_run_done` (exists = done) | — | Skip onboarding wizard if present |
| **Onboarding skipped** | `~/.promptctl/onboarding_skipped` (exists) | — | Show one-line "Run promptctl config" reminder |
| **Analytics consent** | `~/.promptctl/analytics.json` | — | `enabled`, `client_id` for GA4 (CLI) |
| **Alias tip shown** | `~/.promptctl/alias_tip_shown` (exists) | — | Don’t show alias tip again |
| **Last version check** | `~/.promptctl/last_version_check` (RFC3339 timestamp) | — | Throttle upgrade check to once per 24h |

## LLM config (send/create → API)

Stored in `~/.promptctl/llm.json`. Edited via `promptctl config` (interactive or flags) or by setting provider env vars. On macOS, API keys can be stored in Keychain instead of the file.

| Option | How to set | Purpose |
|--------|------------|---------|
| **Default provider** | `promptctl config` or `llm.json` `default_provider` | e.g. `anthropic`, `openai` |
| **Default model** | `promptctl config` or `llm.json` `default_model` | e.g. `claude-sonnet-4-5-20250929` |
| **API keys** | `promptctl config --provider=... --api-key=...` or env (e.g. `ANTHROPIC_API_KEY`) or macOS Keychain | Per-provider key; Keychain used on darwin when set via config flow |

## Score and fix config

`promptctl score` and `promptctl fix` read optional config from `.promptctl/score.yaml` (looked up from the current directory upward). If the file is missing, defaults are used.

| Option | Key in `score.yaml` | Default | Purpose |
|--------|---------------------|---------|---------|
| **Dirs** | `dirs` | (current dir when no positional args) | Directories to scan for prompt files |
| **Include** | `include` | `["*.txt", "*.md"]` | Glob patterns for files to include |
| **Ignore** | `ignore` | (none) | Glob patterns for files to skip |
| **Min score** | `min_score` | `80` | Minimum score (0–100) for pass / fix threshold |
| **Rules** | `rules` | (all) | Optional list of rule IDs to enable (empty = all) |

Example `.promptctl/score.yaml`:

```yaml
dirs: [prompts/, docs/prompts]
include: ["*.txt", "*.md"]
ignore: ["draft-*"]
min_score: 80
rules: []   # optional; empty = all rules
```

CLI flags override the config file: e.g. `--min-score=70` overrides `min_score`, and positional paths override `dirs`.

## CLI flags that override config

- `promptctl create --format=markdown|xml|yaml|json|text` → overrides default create format for that run
- `promptctl send ... --model=MODEL` → overrides default model for that run
- `promptctl score [dirs] --min-score=N --format=json` → score prompt files; exit 0 if all ≥ N, 1 if below or no files, 2 on error; JSON has `files`, `min_score`, `ok`
- `promptctl fix [dirs] --dry-run --suggest` → fix low-scoring prompts; `--dry-run` previews changes, `--suggest` prints suggestions only
- Env `PROMPTCTL_PROMPTS_DIR` → overrides prompts dir
- Env `PROMPTCTL_ENHANCE`, `PROMPTCTL_ENHANCE_URL` → override enhance behavior

## GA4 (CLI analytics)

- **CLI events** (e.g. onboarding, prompt_created, prompt_rated): only sent if `PROMPTCTL_GA4_SECRET` is set in the environment where the CLI runs. Create a Measurement Protocol API secret in GA4 and export it.
- **Consent**: `~/.promptctl/analytics.json` stores user choice; first time we ask (Y/n).

## Summary table: env vars

| Env var | Purpose |
|---------|---------|
| `PROMPTCTL_ENHANCE` | `llm` or `rule` |
| `PROMPTCTL_ENHANCE_URL` | Enhance API base URL |
| `PROMPTCTL_PROMPTS_DIR` | Override prompts (memory) directory |
| `PROMPTCTL_GA4_SECRET` | Enable CLI → GA4 events (optional) |
| `ANTHROPIC_API_KEY` | Fallback API key for Anthropic |
| `OPENAI_API_KEY` | Fallback for OpenAI |
| `GROQ_API_KEY` | Fallback for Groq |
| (etc. per provider) | See `promptctl config` / llm provider list |
