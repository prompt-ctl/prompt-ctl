# promptctl

A CLI tool that turns your prompt engineering knowledge into reusable, composable templates. Stop writing the same prompt structures over and over - define them once, use them everywhere.

## Quick Start

```bash
# Build
go build -o promptctl .

# Install to your PATH
go install .
# or: cp promptctl /usr/local/bin/

# Initialize with starter templates
promptctl init

# List available templates
promptctl list

# Review a file
promptctl review --file=src/auth.ts

# Debug with error context
promptctl debug --file=src/api.ts --error="TypeError: Cannot read property 'id' of undefined"

# Architecture decision
promptctl arch --problem="Should we use event sourcing for the payments service?"

# Generate commit message
promptctl commit --changes="Added retry logic to API client with exponential backoff"
```

## How It Works

Templates live in `~/.promptctl/templates/` (global) or `.promptctl/templates/` (project-level). Each template is a YAML file that defines variables, metadata, and a prompt body with `{{.variable}}` placeholders.

Project-level templates override global ones with the same name, so you can customize per-repo.

## Template Format

```yaml
name: review
description: Code review with security focus
variables:
  - name: file
    description: Path to the file to review
    required: true
  - name: focus
    description: Review focus area
    default: general

body: |
  <context>
  You are an expert code reviewer.
  </context>

  <task>
  Review this file. Focus: {{.focus}}
  </task>

  <file name="{{.file_name}}" language="{{.file_ext}}">
  {{.file_content}}
  </file>
```

## Special Variables

When you pass `--file=path/to/file`, these are auto-populated:
- `{{.file_content}}` - full file contents
- `{{.file_name}}` - basename (e.g., `auth.ts`)
- `{{.file_ext}}` - extension without dot (e.g., `ts`)

When you pass `--dir=path/to/dir`:
- `{{.dir_content}}` - directory tree listing
- `{{.dir_name}}` - directory basename

## Commands

| Command | Description |
|---------|-------------|
| `run <name> [--var=val]` | Render a template (alias: `r`) |
| `list` | List all templates (alias: `ls`) |
| `add <name>` | Scaffold a new template |
| `edit <name>` | Open in `$EDITOR` |
| `show <name>` | Display template content and metadata |
| `copy <name> [--var=val]` | Copy rendered prompt to clipboard (alias: `cp`) |
| `vars <name>` | Show required/optional variables |
| `init` | Set up global config with starter templates |

Shorthand: `promptctl review --file=x.ts` is the same as `promptctl run review --file=x.ts`.

## Starter Templates

`promptctl init` creates these out of the box:

- **review** - Code review (security, performance, maintainability)
- **debug** - Systematic bug analysis and fix suggestions
- **arch** - Architecture decision records with trade-off analysis
- **commit** - Conventional commit message generation
- **explain** - Code explanation at configurable depth levels

## Piping to LLMs

The real power is piping output directly to an LLM CLI:

```bash
# With Anthropic's Claude CLI
promptctl review --file=src/auth.ts | claude

# With OpenAI
promptctl review --file=src/auth.ts | openai chat

# Copy to clipboard for pasting into any AI chat
promptctl cp review --file=src/auth.ts
```

## Enhance mode

By default, `promptctl create "your intent"` uses the **hosted LLM enhancer** (no env or keys needed). To use the offline rule-based enhancer instead:

```bash
export PROMPTCTL_ENHANCE=rule
promptctl create "review my auth code"
```

To point at your own Worker: `PROMPTCTL_ENHANCE_URL=https://your-worker.workers.dev` (see [worker/](worker/)). The URL must use HTTPS. The default hosted Worker applies request limits (e.g. 4000 chars intent, 32 KiB body) and optional analytics.

**Quality score and tuning**
When using the LLM enhancer, a **quality score (0–100)** is printed to stderr. It measures fidelity (your specific terms preserved in the output), absence of duplicate sections, and required structure. Use `promptctl create "intent" --score` to show the score for rule-based enhance too. If the score is low or the output is too generic, try: (1) `PROMPTCTL_ENHANCE=rule` for long or detailed intents (no LLM), or (2) shorten the intent and add specifics in a follow-up.

**Security & configuration**
- Do not commit API keys or webhook URLs; use environment variables or Cloudflare secrets.
- Prefer interactive `promptctl config` or setting the provider’s env var (e.g. `ANTHROPIC_API_KEY`) so the key is not stored in shell history. Using `promptctl config --api-key=sk-...` on the command line can expose the key in history.
- Paths for `--file` and `--dir` must be under the current working directory (path traversal is rejected).

## Project-Level Templates

Create project-specific templates that override or extend global ones:

```bash
cd my-project
promptctl init --local
promptctl add sprint-review --local
```

This creates `.promptctl/templates/sprint-review.yaml` in your project root. Commit it to your repo for team-wide consistency.

## Building Custom Templates

```bash
# Scaffold a new template
promptctl add my-template

# Edit it
promptctl edit my-template

# Test it
promptctl show my-template
promptctl run my-template --file=test.ts
```

Template tips:
- Use XML tags (`<context>`, `<task>`, `<constraints>`) for structure - Claude responds well to these
- Put the most important instructions early in the prompt
- Use `{{if .var}}...{{end}}` for optional sections
- Keep constraints specific and actionable

## Cross-Platform

Build for any platform:

```bash
# macOS (Apple Silicon)
GOOS=darwin GOARCH=arm64 go build -o promptctl-darwin-arm64 .

# macOS (Intel)
GOOS=darwin GOARCH=amd64 go build -o promptctl-darwin-amd64 .

# Windows
GOOS=windows GOARCH=amd64 go build -o promptctl.exe .

# Linux
GOOS=linux GOARCH=amd64 go build -o promptctl-linux-amd64 .
```
