# promptctl

**A CLI for version control and testing of LLM prompt templates.**

Copy-paste prompts are unmaintainable. They drift across projects, lose context, and resist collaboration. **promptctl treats prompts like code** - version them, template them, test them, and share them across your team.

## Why promptctl?

| | promptctl | LangChain | PromptLayer | Manual copy-paste |
|---|---|---|---|---|
| **Language-agnostic** | Yes (CLI) | Python/JS only | Web UI | N/A |
| **Version control** | Git-native YAML | Framework-locked | Cloud-only | Manual |
| **No vendor lock-in** | Any LLM provider | LangChain ecosystem | PromptLayer API | N/A |
| **Offline capable** | Fully offline | Needs runtime | Needs cloud | Yes |
| **Zero dependencies** | Single binary | pip/npm install | SaaS signup | N/A |
| **CI/CD friendly** | Exit codes + JSON | Custom setup | API integration | Scripts |
| **Cost** | Free forever (Apache 2.0) | Free (OSS) | Freemium | Free |

## Installation

**macOS:**
```bash
brew tap oleg-koval/tap && brew install promptctl
```

**Linux / macOS / Windows (Go):**
```bash
go install github.com/oleg-koval/promptctl@latest
```

**From source:**
```bash
git clone https://github.com/oleg-koval/promptctl.git
cd promptctl
go build -o promptctl .
sudo mv promptctl /usr/local/bin/
```

**Direct binary download:**
Grab the latest release for your platform from [GitHub Releases](https://github.com/oleg-koval/promptctl/releases).

See [docs/INSTALL.md](docs/INSTALL.md) for detailed platform-specific instructions.

## Quick Start (5 minutes)

```bash
# 1. Initialize promptctl with starter templates
promptctl init

# 2. Review a file using the built-in review template
promptctl review --file=src/auth.ts
```

This loads `~/.promptctl/templates/review.yaml`, injects your file content, and outputs a structured prompt:

```
<context>
You are an expert code reviewer specializing in security, performance,
and maintainability.
</context>

<task>
Review this file with focus on: general
Identify issues by severity (critical, warning, suggestion).
</task>

<file name="auth.ts" language="ts">
import express from 'express';
...
</file>

<constraints>
- Be specific: reference line numbers
- Suggest fixes, not just problems
- Prioritize security issues
</constraints>
```

Pipe it to any LLM:

```bash
# Claude
promptctl review --file=src/auth.ts | claude

# OpenAI
promptctl review --file=src/auth.ts | openai chat

# Or send directly with built-in LLM support
promptctl send review --file=src/auth.ts

# Copy to clipboard for any AI chat
promptctl cp review --file=src/auth.ts
```

## Template Format

Templates are YAML files with `{{.variable}}` placeholders:

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

When you pass `--file=path/to/file`, these variables are auto-populated:
- `{{.file_content}}` - full file contents
- `{{.file_name}}` - basename (e.g., `auth.ts`)
- `{{.file_ext}}` - extension without dot (e.g., `ts`)

## Features

### Starter Templates

`promptctl init` ships with templates for common workflows:

- **review** - Code review (security, performance, maintainability)
- **debug** - Systematic bug analysis with error context
- **arch** - Architecture decision records with trade-off analysis
- **commit** - Conventional commit message generation
- **explain** - Code explanation at configurable depth levels

### Custom Templates

Build your own templates for any workflow:

```bash
# Create and edit
promptctl add api-review
promptctl edit api-review

# Use it
promptctl run api-review --file=routes.ts

# Project-local templates (committed to your repo)
promptctl init --local
promptctl add sprint-review --local
```

Project-level templates in `.promptctl/templates/` override global ones, so every repo can have its own prompt conventions.

### Prompt Engineering from Intent

Transform raw intent into structured prompts:

```bash
promptctl create "review my authentication code for security holes"
```

This uses an AI enhancer to expand your intent into a well-structured prompt with context, task, constraints, and output format. A quality score (0-100) is printed so you know how good the result is.

### Prompt Scoring and Fixing (CI-ready)

Score your prompt files for quality:

```bash
# Score all prompts in a directory
promptctl score prompts/
# Output: each file scored 0-100 on structure, clarity, constraints, persona

# CI gate: fail if any prompt scores below 80
promptctl score --min-score=80 prompts/

# Machine-readable output
promptctl score --format=json prompts/

# Auto-fix low-scoring prompts
promptctl fix prompts/
promptctl fix --dry-run prompts/  # preview changes first
```

### Experimentation

Benchmark templates across models and optimize:

```bash
# Compare a template across different models
promptctl experiment review --file=auth.ts

# Auto-generate variants and keep the best
promptctl experiment optimize review
```

### LLM Integration

Send prompts directly to LLMs without leaving the terminal:

```bash
# Configure your provider
promptctl config --provider=anthropic
# (prompts for API key interactively)

# Send and get a response
promptctl send review --file=src/auth.ts

# Compare costs across models
promptctl cost review --file=main.go --compare

# See potential annual savings from structured prompts
promptctl savings
```

Supported providers: Anthropic (Claude), OpenAI (GPT).

## Commands

| Command | Description |
|---------|-------------|
| `create "intent"` | Transform raw intent into a structured prompt (alias: `c`) |
| `run <name> [--var=val]` | Render a template (alias: `r`) |
| `send <name> [--var=val]` | Render and send to LLM (alias: `s`) |
| `cost <name> [--var=val]` | Estimate cost before sending |
| `experiment <name>` | Benchmark template across models (alias: `exp`) |
| `list` | List all available templates (alias: `ls`) |
| `add <name>` | Create a new prompt template |
| `edit <name>` | Open template in `$EDITOR` |
| `show <name>` | Display template content and metadata |
| `copy <name>` | Copy rendered prompt to clipboard (alias: `cp`) |
| `vars <name>` | Show variables required by a template |
| `score [dirs]` | Score prompt files (0-100), CI-friendly |
| `fix [dirs]` | Auto-fix low-scoring prompts |
| `config` | View or set LLM provider configuration |
| `models` | List supported models with pricing |
| `init` | Initialize config and starter templates |

Shorthand: `promptctl review --file=x.ts` is equivalent to `promptctl run review --file=x.ts`.

## Roadmap

### v1.0.0 (current)
- Template engine with YAML format and variable substitution
- Built-in starter templates (review, debug, arch, commit, explain)
- Project-level template overrides
- LLM integration (Anthropic, OpenAI) with cost estimation
- Prompt scoring and auto-fix
- Experimentation and optimization
- AI-powered prompt creation from intent

### v1.1.0: Prompt Testing Framework
- Test templates against expected outputs
- Regression testing for prompt changes
- Model comparison benchmarks

### v2.0.0: Cloud Platform
- Web dashboard for prompt management
- Prompt registry and versioning
- Team collaboration and sharing
- Usage analytics and insights

> Core CLI is forever free and open source. Cloud features coming soon.

## Contributing

We welcome contributions! See [docs/CONTRIBUTING.md](docs/CONTRIBUTING.md) for:
- Developer setup and build instructions
- Code style and PR process
- How to add new commands and templates
- Testing requirements

## License

[Apache License 2.0](LICENSE) - see [AUTHORS.md](AUTHORS.md) for contributors.

## Links

- **Website:** [prompt-ctl.com](https://prompt-ctl.com)
- **Issues:** [GitHub Issues](https://github.com/oleg-koval/promptctl/issues)
- **Discussions:** [GitHub Discussions](https://github.com/oleg-koval/promptctl/discussions)
