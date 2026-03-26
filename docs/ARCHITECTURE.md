# Architecture

A deep dive into how promptctl is built, for contributors and curious users.

## Overview

```
promptctl CLI (main.go)
  │
  ├── cmd/           Command routing & user interaction (Cobra-style)
  │     ├── root.go          Main command dispatcher
  │     ├── execution.go     Template execution pipeline
  │     ├── experiment.go    Model benchmarking & optimization
  │     ├── score.go         Prompt quality scoring
  │     ├── fix.go           Auto-fix low-scoring prompts
  │     └── ...
  │
  ├── config/        Configuration discovery & merging
  │     └── config.go        Global + local config loading
  │
  ├── prompt/        Template engine & enhancement
  │     ├── template.go      YAML parsing, variable substitution, rendering
  │     ├── enhance.go       Rule-based prompt enhancement
  │     ├── quality.go       Prompt quality scoring (0-100)
  │     ├── domain_knowledge.go  Domain-specific expertise
  │     └── ...
  │
  ├── llm/           Provider abstraction & execution
  │     ├── provider.go      Multi-provider LLM client (Anthropic, OpenAI, Groq, etc.)
  │     ├── client.go        Client interface for mocking
  │     └── keychain_*.go    Platform-specific API key storage
  │
  ├── internal/      Shared utilities
  │     ├── analytics/       Optional GA4 telemetry (opt-in)
  │     ├── discover/        File & directory discovery
  │     ├── onboarding/      First-run setup tracking
  │     ├── safepath/        Path traversal protection
  │     ├── scoreconfig/     Per-project score configuration
  │     ├── shell/           Shell alias management
  │     └── ui/              Terminal colors, prompts, spinners
  │
  └── worker/        Cloud components (Cloudflare Worker, optional)
        └── src/index.js     /enhance endpoint for LLM-powered enhancement
```

## Core Packages

### `cmd/` - CLI Commands

All commands live in `cmd/root.go` as functions registered with a simple argument-based dispatcher. Key commands:

| Command | Function | Purpose |
|---------|----------|---------|
| `create` | `createPrompt()` | Transform raw intent into a structured prompt |
| `run` | `runPrompt()` | Render a template with variables |
| `send` | `sendPrompt()` | Render and send to an LLM provider |
| `cost` | `showCost()` | Estimate LLM cost before sending |
| `experiment` | `runExperiment()` | Benchmark a prompt across models |
| `score` | `runScore()` | Evaluate prompt quality (0-100) |
| `fix` | `runFix()` | Auto-fix low-scoring prompts |
| `list` | `listPrompts()` | List available templates |
| `add` | `addPrompt()` | Create a new template interactively |
| `edit` | `editPrompt()` | Open a template in `$EDITOR` |
| `show` | `showPrompt()` | Display template content |
| `copy` | `copyPrompt()` | Copy rendered prompt to clipboard |
| `vars` | `showVars()` | Show a template's required variables |
| `memory` | `memoryList()` | Manage saved prompts |
| `config` | `configLLM()` | Set LLM provider and API key |
| `models` | `listModels()` | List supported models and pricing |
| `init` | `initConfig()` | First-time setup wizard |

### `prompt/template.go` - Template Engine

The template engine parses YAML files into `Template` structs and renders them with variable substitution.

```go
type Template struct {
    Name        string
    Description string
    Variables   []Variable
    Body        string
    Path        string
    IsLocal     bool
}

type Variable struct {
    Name        string
    Description string
    Required    bool
    Default     string
}
```

Key operations:
- **`LoadTemplate(name, cfg)`** - Finds and parses a template YAML file. Supports versioned folders (with `meta.json`) and flat `.yaml` files.
- **`Render(vars)`** - Substitutes `{{.variable}}` placeholders, handles `{{if .var}}...{{end}}` conditionals, validates required variables, and applies defaults.

### `config/` - Configuration

Configuration is loaded from multiple sources with a cascading priority:

1. **Environment variables** - `PROMPTCTL_ENHANCE_URL`, `PROMPTCTL_ENHANCE`, `PROMPTCTL_PROMPTS_DIR`
2. **Local** `.promptctl/` directory (project-specific overrides)
3. **Global** `~/.promptctl/` directory (user defaults)

```go
type Config struct {
    GlobalTemplateDir string   // ~/.promptctl/templates
    LocalTemplateDir  string   // .promptctl/templates
    PromptsDir        string   // saved prompts location
    EnhanceURL        string   // optional hosted enhancement endpoint
    EnhanceMode       string   // "llm" (default) or "rule" (offline)
    DefaultVars       map[string]string
}
```

LLM provider settings are stored in `~/.promptctl/llm.json`:

```json
{
  "default_provider": "anthropic",
  "default_model": "claude-sonnet-4-5-20250929",
  "api_keys": { "anthropic": "sk-ant-..." }
}
```

### `llm/` - Provider Abstraction

All LLM interaction goes through an interface-based abstraction:

```go
type Client interface {
    CompleteWithOptions(prompt, model string, opts *CompleteOptions) (*CompletionResult, error)
}
```

Supported providers:
- **Anthropic** - Claude Sonnet 4.5, Claude Haiku 4.5, Claude Opus 4.6
- **OpenAI** - GPT-5, GPT-5.1, GPT-5.2
- **Groq** - Llama 4 Maverick, Llama 3.3 70B, Mixtral 8x7B
- **DeepSeek** - DeepSeek Chat V3.2, DeepSeek R1
- **Google** - Gemini 3.1 Pro

Each provider has a dedicated API caller (`callAnthropic()`, `callOpenAICompatible()`, `callGemini()`). Cost estimation uses per-model pricing and a token heuristic (3.8 chars/token).

### `internal/` - Shared Utilities

| Package | Purpose |
|---------|---------|
| `analytics/` | Optional Google Analytics 4 telemetry (opt-in) |
| `discover/` | File and directory discovery for templates |
| `onboarding/` | First-run setup state tracking |
| `safepath/` | Path traversal protection (prevents `../` escapes) |
| `scoreconfig/` | Per-project score thresholds from `.promptctl/score.yaml` |
| `shell/` | Shell alias management |
| `ui/` | Terminal colors, interactive prompts (survey), spinners |

## Template Format

Templates are YAML files stored in `~/.promptctl/templates/` (global) or `.promptctl/templates/` (project-local). Local templates override global ones with the same name.

```yaml
name: review
description: Code review template
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

  <file name="{{.file_name}}" ext="{{.file_ext}}">
  {{.file_content}}
  </file>

  Please review this code with a focus on {{.focus}}.
```

**Variable substitution:**
- `{{.variable_name}}` - Replaced from the variables map
- `{{if .optional}}...{{end}}` - Conditional sections, included only when the variable is set
- Missing required variables produce an error
- Missing optional variables use their default value

**Versioned templates** use a folder structure with `meta.json`:

```
~/.promptctl/templates/
  review/
    meta.json    # {"current": "v2", "versions": ["v1", "v2"]}
    v1.yaml
    v2.yaml
```

## How a Command Works

Example: `promptctl send review --file=src/auth.ts`

```
1. Entry point
   main.go → cmd.Execute()

2. Command routing
   root.go parses os.Args → dispatches to sendPrompt()

3. Configuration
   config.Load() reads ~/.promptctl/ and .promptctl/
   Loads LLM settings from ~/.promptctl/llm.json

4. Template loading
   prompt.LoadTemplate("review", cfg)
   Finds review.yaml (or review/v2.yaml), parses YAML → Template struct

5. Variable enrichment
   execution.go: enrichFileVars() reads src/auth.ts
   Populates: file_content, file_name, file_ext

6. Rendering
   template.Render(vars) substitutes {{.file_content}}, {{.focus}}, etc.
   Validates required variables, applies defaults

7. LLM execution
   llm.CompleteWithOptions(renderedPrompt, model, opts)
   Calls the configured provider's API (e.g., Anthropic Messages API)

8. Output
   Streams response to terminal
   Optionally tracks analytics and shows cost
```

## Extending promptctl

### Add a New Command

1. Add a new function in `cmd/` (e.g., in `root.go` or a new file)
2. Implement the command logic following existing patterns
3. Register it in the command dispatcher in `root.go`
4. Add tests in `cmd/*_test.go`
5. Update README quick reference

### Add a New LLM Provider

1. Define the provider and its models in `llm/provider.go` (add to the providers list)
2. Implement an API caller function (e.g., `callMyProvider()`)
3. Add the caller to the dispatch logic in `Complete()`/`CompleteWithOptions()`
4. Add pricing data for cost estimation

### Add Templates

- **Global**: Drop YAML files in `~/.promptctl/templates/`
- **Project-local**: Create `.promptctl/templates/` in your project directory
- Project-local templates override global templates with the same name

## Testing Architecture

### Test Files

| File | What It Tests |
|------|---------------|
| `cmd/root_test.go` | Command routing and execution |
| `cmd/helpers_test.go` | Flag parsing and utility functions |
| `cmd/score_test.go` | Score command integration |
| `cmd/send_test.go` | Send command with mock LLM |
| `cmd/experiment_baseline_test.go` | Experiment baseline comparison |
| `prompt/template_test.go` | Template rendering and variable substitution |
| `prompt/enhance_test.go` | Rule-based prompt enhancement |
| `prompt/quality_test.go` | Prompt quality scoring |
| `prompt/domain_knowledge_test.go` | Domain expertise mapping |
| `llm/provider_test.go` | Model lookup and cost estimation |
| `internal/safepath/path_test.go` | Path traversal prevention |
| `internal/ui/ui_test.go` | Terminal UI rendering |

### Testing Patterns

- **Isolation**: `t.TempDir()` for filesystem tests, `t.Setenv()` for environment
- **Mock LLM**: `llm.MockClient` implements the `Client` interface for testing without API calls
- **Stdout capture**: `os.Pipe()` + `io.Copy()` to verify command output
- **Args patching**: Modify `os.Args` with `defer` restore for command tests

### Running Tests

```bash
go test ./...              # All tests
go test -cover ./...       # With coverage
go test ./prompt/...       # Template package only
go test ./cmd/ -run Score  # Specific test
```

## Cloud Components

The `worker/` directory contains an optional Cloudflare Worker that provides LLM-powered prompt enhancement.

- **Endpoint**: `POST /enhance` - Takes a user's raw intent and returns a structured prompt
- **Stack**: Cloudflare Workers with AI (Llama 3.1), Analytics Engine, D1 Database
- **Fallback**: When the worker is unavailable, promptctl falls back to rule-based enhancement offline

The cloud component is completely separate from the core CLI. The CLI works fully offline; the worker adds optional AI-powered enhancement.

## Dependencies

promptctl keeps dependencies minimal:

| Dependency | Purpose |
|------------|---------|
| [survey/v2](https://github.com/AlecAivazis/survey) | Interactive terminal prompts |
| [yaml.v3](https://github.com/go-yaml/yaml) | YAML template parsing |
| [google/uuid](https://github.com/google/uuid) | Unique identifiers |
| [go-shellquote](https://github.com/kballard/go-shellquote) | Shell argument quoting |
| [mgutz/ansi](https://github.com/mgutz/ansi) | Terminal color output |

No runtime LLM library is required - provider communication uses Go's standard `net/http` package with direct API calls.
